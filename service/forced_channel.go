package service

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// ForcedChannelRoute 表示一次「指定渠道直连」路由的解析结果。
type ForcedChannelRoute struct {
	SupplierAlias string // 例如 "P0"、"P5"、自定义别名
	ModelName     string // 去掉前后缀后的真实模型名
	ChannelNo     string // 例如 "c2"
	ChannelID     int    // 匹配到的渠道 ID
}

// ForcedSupplierRoute 表示一次「指定供应商 + 任意渠道」路由的解析结果。
//
// 对应模型名形式 {alias}/{model}，例如 "P0/claude-haiku-4-5-20251001"。
// 命中后会把候选渠道限制在 supplier_applications.id = SupplierApplicationID 的范围内，
// 再交由 SmartRouter（或兜底随机）从中挑选。
type ForcedSupplierRoute struct {
	SupplierAlias         string
	ModelName             string
	SupplierApplicationID int
}

// aliasPattern 匹配供应商别名：P 后跟数字，或由字母数字/下划线/连字符组成的自定义别名。
//
// 为避免与实际模型名（可能含斜杠，例如 "openai/gpt-4o"）混淆，此处对别名收敛为
// 较严格的集合；自定义别名仅允许 ASCII 字母数字与常见分隔符，这也和
// SupplierApplicationAutoAlias 产出的 "P<id>" 保持兼容。
var (
	aliasPattern     = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_\-]*$`)
	channelNoPattern = regexp.MustCompile(`^c\d+$`)
)

// ParseForcedChannelModelName 尝试把 {alias}/{model}/{channel_no} 形式的模型名
// 解析为 ForcedChannelRoute。不符合该形式时返回 nil, false, nil。
//
// 解析规则：
//   - 至少包含两个 "/"；
//   - 最后一段必须匹配 `c\d+`（渠道编号）；
//   - 第一段必须匹配 aliasPattern（供应商别名）；
//   - 中间任意多段拼回来作为真实模型名（兼容如 "openai/gpt-4o" 这种带斜杠的模型）。
//
// 当格式匹配但渠道查不到时，ok 为 true，err 非空，调用方可据此拒绝请求。
func ParseForcedChannelModelName(raw string) (*ForcedChannelRoute, bool, error) {
	name := strings.TrimSpace(raw)
	if name == "" || !strings.Contains(name, "/") {
		return nil, false, nil
	}
	parts := strings.Split(name, "/")
	if len(parts) < 3 {
		return nil, false, nil
	}
	alias := parts[0]
	channelNo := parts[len(parts)-1]
	if !aliasPattern.MatchString(alias) || !channelNoPattern.MatchString(channelNo) {
		return nil, false, nil
	}
	modelName := strings.Join(parts[1:len(parts)-1], "/")
	if modelName == "" {
		return nil, false, nil
	}

	channelID, err := model.FindChannelIDBySupplierAliasAndNo(alias, channelNo)
	if err != nil {
		return nil, true, err
	}
	return &ForcedChannelRoute{
		SupplierAlias: alias,
		ModelName:     modelName,
		ChannelNo:     channelNo,
		ChannelID:     channelID,
	}, true, nil
}

// ParseForcedSupplierModelName 尝试把 {alias}/{model} 形式的模型名解析为 ForcedSupplierRoute。
// 不符合该形式时返回 nil, false, nil；匹配别名但别名查不到时 matched=true 且 err 非空。
//
// 解析规则（区别于 ParseForcedChannelModelName）：
//   - 必须恰好只有一个 "/"（两段）；
//   - 第一段必须匹配 aliasPattern；
//   - 最后一段必须「不是」 channelNoPattern，以避免误吞 "alias/c3" 这种无模型名的形式；
//   - alias 必须能够解析为已存在的供应商（或 P0），否则按「未命中」处理，将模型串原样交由
//     后续正常路由（便于兼容 "openai/gpt-4o" 这种真实模型名）。
func ParseForcedSupplierModelName(raw string) (*ForcedSupplierRoute, bool, error) {
	name := strings.TrimSpace(raw)
	if name == "" || !strings.Contains(name, "/") {
		return nil, false, nil
	}
	parts := strings.Split(name, "/")
	if len(parts) != 2 {
		return nil, false, nil
	}
	alias := parts[0]
	modelName := strings.TrimSpace(parts[1])
	if !aliasPattern.MatchString(alias) || modelName == "" {
		return nil, false, nil
	}
	// 若第二段形如 cN（渠道编号），不应走供应商路由（由三段形式处理）。
	if channelNoPattern.MatchString(modelName) {
		return nil, false, nil
	}

	supplierApplicationID, found, err := model.ResolveSupplierApplicationIDByAlias(alias)
	if err != nil || !found {
		// 别名查不到时不 matched=true，让模型串继续按普通模型名走常规路由，
		// 避免与 "openai/gpt-4o" 这种合法模型名冲突。
		return nil, false, nil
	}
	return &ForcedSupplierRoute{
		SupplierAlias:         alias,
		ModelName:             modelName,
		SupplierApplicationID: supplierApplicationID,
	}, true, nil
}

// ApplyForcedChannelOnRequestBody 把解析出的真实模型名写回请求体（仅处理 JSON 请求），
// 并在上下文中记录「强制渠道 ID」与原始模型串，供后续中间件 / 日志引用。
//
// 非 JSON 请求（如 multipart 语音上传）目前不改写请求体，仅更新上下文；这类场景下
// 具体模型名一般由路径或其他字段给出，不会因为模型串里带斜杠而被上游拒绝。
func ApplyForcedChannelOnRequestBody(c *gin.Context, route *ForcedChannelRoute, originalModel string) error {
	common.SetContextKey(c, constant.ContextKeyForcedChannelID, route.ChannelID)
	common.SetContextKey(c, constant.ContextKeyForcedChannelModelKey, originalModel)
	return rewriteRequestModelField(c, route.ModelName)
}

// ApplyForcedSupplierOnRequestBody 写入强制供应商上下文并改写请求体 model 字段。
// 语义同 ApplyForcedChannelOnRequestBody，差异在于只限制候选池而不绑定到单一渠道。
func ApplyForcedSupplierOnRequestBody(c *gin.Context, route *ForcedSupplierRoute, originalModel string) error {
	common.SetContextKey(c, constant.ContextKeyForcedSupplierApplicationID, route.SupplierApplicationID)
	common.SetContextKey(c, constant.ContextKeyForcedSupplierApplicationIDSet, true)
	common.SetContextKey(c, constant.ContextKeyForcedChannelModelKey, originalModel)
	return rewriteRequestModelField(c, route.ModelName)
}

func rewriteRequestModelField(c *gin.Context, modelName string) error {
	contentType := c.Request.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		return nil
	}

	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return err
	}
	body, err := storage.Bytes()
	if err != nil {
		return err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return nil
	}

	var obj map[string]json.RawMessage
	if err := common.Unmarshal(body, &obj); err != nil {
		return nil
	}
	if _, ok := obj["model"]; !ok {
		return nil
	}
	newModel, err := json.Marshal(modelName)
	if err != nil {
		return err
	}
	obj["model"] = newModel
	newBody, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	return common.ReplaceRequestBody(c, newBody)
}

// ModelRouteResult 表示一次「模型 + 全局 route_slug」解析结果（{model}/{route_slug}）。
type ModelRouteResult struct {
	ModelName string // 去掉后缀后的真实模型名
	RouteSlug string // 渠道全局路由后缀（channels.route_slug）
	ChannelID int    // 解析得到的渠道 ID
}

// ParseModelRouteIndex 尝试把 {model}/{route_slug} 形式的模型名解析为 ModelRouteResult。
//
// 解析规则：
//   - 字符串中至少包含一个 "/"；
//   - 最后一段须为合法 route_slug（见 model.IsValidRouteSlug），且不能为旧 channel_no 形态 c\d+；
//   - 去掉最后一段后的部分作为模型名，按 route_slug 查启用渠道并校验 models 列表包含该模型；
//   - 未命中或渠道禁用或模型不在列表：返回 nil, false, nil（静默降级为普通路由）。
func ParseModelRouteIndex(raw string) (*ModelRouteResult, bool, error) {
	name := strings.TrimSpace(raw)
	if name == "" || !strings.Contains(name, "/") {
		return nil, false, nil
	}
	lastSlash := strings.LastIndex(name, "/")
	potentialSlug := name[lastSlash+1:]
	potentialModel := name[:lastSlash]
	if potentialSlug == "" || potentialModel == "" {
		return nil, false, nil
	}
	if !model.IsValidRouteSlug(potentialSlug) {
		return nil, false, nil
	}

	channelID := model.ResolveChannelIDByRouteSlugAndModel(potentialSlug, potentialModel)
	if channelID <= 0 {
		return nil, false, nil
	}
	return &ModelRouteResult{
		ModelName:  potentialModel,
		RouteSlug:  potentialSlug,
		ChannelID:  channelID,
	}, true, nil
}

// ApplyModelRouteOnRequestBody 写入强制渠道 ID 上下文并把真实模型名写回请求体。
// 语义同 ApplyForcedChannelOnRequestBody，用于 {model}/{route_slug} 路由格式。
func ApplyModelRouteOnRequestBody(c *gin.Context, result *ModelRouteResult, originalModel string) error {
	common.SetContextKey(c, constant.ContextKeyForcedChannelID, result.ChannelID)
	common.SetContextKey(c, constant.ContextKeyForcedChannelModelKey, originalModel)
	return rewriteRequestModelField(c, result.ModelName)
}

// ForcedSupplierFromContext 返回当前请求是否绑定了「强制供应商」路由，以及对应的
// supplier_applications.id（P0 时为 0）。
func ForcedSupplierFromContext(c *gin.Context) (int, bool) {
	if _, ok := common.GetContextKey(c, constant.ContextKeyForcedSupplierApplicationIDSet); !ok {
		return 0, false
	}
	raw, ok := common.GetContextKey(c, constant.ContextKeyForcedSupplierApplicationID)
	if !ok {
		return 0, false
	}
	id, ok := raw.(int)
	if !ok {
		return 0, false
	}
	return id, true
}
