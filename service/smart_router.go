package service

import (
	"encoding/json"
	"os"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/fyinfor/router-engine/pkg/router"
	"github.com/gin-gonic/gin"
)

// SmartRouterEnabled 默认开启。仅当 SMART_ROUTER_ENABLED 为 0 / false / no / off（不区分大小写）时关闭。
func SmartRouterEnabled() bool {
	v := strings.TrimSpace(os.Getenv("SMART_ROUTER_ENABLED"))
	if v == "" {
		return true
	}
	if v == "0" || strings.EqualFold(v, "false") || strings.EqualFold(v, "no") || strings.EqualFold(v, "off") {
		return false
	}
	return true
}

func channelProviderSlug(ch *model.Channel) string {
	switch ch.Type {
	case constant.ChannelTypeOpenAI:
		return "openai"
	case constant.ChannelTypeAzure:
		return "azure"
	case constant.ChannelTypeAnthropic:
		return "anthropic"
	case constant.ChannelTypeOpenRouter:
		return "openrouter"
	case constant.ChannelTypeGemini:
		return "google"
	case constant.ChannelTypeVertexAi:
		return "google-vertex"
	case constant.ChannelTypeDeepSeek:
		return "deepseek"
	case constant.ChannelTypeSiliconFlow:
		return "siliconflow"
	case constant.ChannelTypeVolcEngine:
		return "volcengine"
	case constant.ChannelTypeMoonshot:
		return "moonshot"
	case constant.ChannelTypeXai:
		return "xai"
	case constant.ChannelTypeMistral:
		return "mistral"
	case constant.ChannelTypePerplexity:
		return "perplexity"
	case constant.ChannelTypeTencent:
		return "tencent"
	case constant.ChannelTypeZhipu, constant.ChannelTypeZhipu_v4:
		return "zhipu"
	case constant.ChannelTypeBaidu, constant.ChannelTypeBaiduV2:
		return "baidu"
	case constant.ChannelTypeAli:
		return "dashscope"
	case constant.ChannelTypeAws:
		return "aws"
	case constant.ChannelTypeCohere:
		return "cohere"
	default:
		if n, ok := constant.ChannelTypeNames[ch.Type]; ok {
			return strings.ToLower(strings.ReplaceAll(n, " ", ""))
		}
		return "unknown"
	}
}

// ResolveChannelModelUnitPrice 解析某渠道下某模型的「最终单价信号」，用于路由排序
// （价格优 / SmartRouter）。计算口径与 relay 计费一致，已计入供应商成本折扣
// （price_discount_percent）与加价折扣（markup_discount_rate）：
//
//	固定价模型：channelPrice × 成本折扣% + globalPrice × 加价折扣%
//	倍率模型：  channelRatio × 成本折扣% + globalRatio × 加价折扣%
//
// 解析优先级与计费一致：供应商作用域固定价 → 供应商作用域倍率 → 全局模型倍率 → 兜底 1。
// 返回值为相对排序信号（非精确计费单价），但渠道间相对高低与用户最终支付价一致。
func ResolveChannelModelUnitPrice(ch *model.Channel, modelName string) float64 {
	if price, ok := ResolveChannelModelConfiguredUnitPrice(ch, modelName); ok {
		return price
	}
	// 未配置任何定价：兜底为倍率 1，保证选路时仍有可比价（不会被当成 0 即「免费」）。
	return 1
}

// ResolveChannelModelConfiguredUnitPrice 与 ResolveChannelModelUnitPrice 同口径，
// 但仅在「确有已配置定价」时返回 (price, true)；无任何配置时返回 (0, false)，不兜底为 1。
// 用于价格优模式的展示 / 排序：未配置定价的渠道应显示「—」并排在最后，
// 与 /api/pricing（仅展示已配置定价的模型）口径一致；而真实选路仍用带兜底的
// ResolveChannelModelUnitPrice，避免未配置渠道因 0 价被误判为最便宜。
func ResolveChannelModelConfiguredUnitPrice(ch *model.Channel, modelName string) (float64, bool) {
	channelID := ch.Id
	sid := ch.SupplierApplicationID
	costDisc := model.ResolveChannelEffectiveCostPercent(channelID) // 最终成本率%（成本折扣 + 经营成本，默认 100）
	markupDisc := model.ResolveChannelMarkupDiscountRate(channelID) // 加价折扣%（默认 0）

	// 固定价优先（对应计费 usePrice 分支）。
	if channelPrice, ok := model.ResolveSupplierScopedFixedModelPrice(channelID, sid, modelName); ok {
		globalPrice, _ := ratio_setting.GetModelPrice(modelName, false)
		if eff := model.EffectiveModelPrice(channelPrice, globalPrice, costDisc, markupDisc); eff > 0 {
			return eff, true
		}
	}

	// 倍率分支。
	if channelRatio, ok, _ := model.ResolveSupplierScopedModelRatio(channelID, sid, modelName); ok {
		globalRatio, _, _ := ratio_setting.GetModelRatio(modelName)
		if eff := model.EffectiveInputRate(channelRatio, globalRatio, costDisc, markupDisc); eff > 0 {
			return eff, true
		}
	}

	// 全局倍率兜底（平台级定价，视为已配置）。
	if ratio, _, _ := ratio_setting.GetModelRatio(modelName); ratio > 0 {
		return ratio, true
	}
	return 0, false
}

func buildRouterCandidates(group, modelName string) ([]*router.EndpointCandidate, error) {
	return buildRouterCandidatesFiltered(group, modelName, nil)
}

// buildRouterCandidatesFiltered 在 buildRouterCandidates 基础上额外支持按渠道过滤。
// filter 为 nil 时行为与 buildRouterCandidates 相同；filter 返回 false 的渠道将被剔除。
func buildRouterCandidatesFiltered(group, modelName string, filter func(*model.Channel) bool) ([]*router.EndpointCandidate, error) {
	ids := model.ListChannelIDsForGroupModel(group, modelName)
	if len(ids) == 0 {
		return nil, nil
	}
	var out []*router.EndpointCandidate
	for _, id := range ids {
		ch, err := model.CacheGetChannel(id)
		if err != nil || ch == nil || ch.Status != common.ChannelStatusEnabled {
			continue
		}
		if !model.IsChannelEnabledForGroupModel(group, modelName, ch.Id) {
			continue
		}
		if filter != nil && !filter(ch) {
			continue
		}
		// UnitPrice is the primary sorting signal for smart routing（与 relay 定价优先级对齐）。
		unitPrice := ResolveChannelModelUnitPrice(ch, modelName)
		latSec := float64(ch.ResponseTime) / 1000.0
		if latSec <= 0 {
			latSec = 0.001
		}
		tps := 1.0 / latSec
		w := 0
		if ch.Weight != nil {
			w = int(*ch.Weight)
		}
		prio := int64(0)
		if ch.Priority != nil {
			prio = *ch.Priority
		}
		out = append(out, &router.EndpointCandidate{
			ChannelID:         ch.Id,
			Model:             modelName,
			ProviderSlug:      channelProviderSlug(ch),
			UnitPrice:         unitPrice,
			Healthy:           true,
			LatencyP50Seconds: latSec,
			ThroughputTps:     tps,
			Priority:          prio,
			Weight:            w,
		})
	}
	return out, nil
}

func resolveSmartRouteGroup(usingGroup, userGroup, modelName string) string {
	return resolveSmartRouteGroupFiltered(usingGroup, userGroup, modelName, nil)
}

// resolveSmartRouteGroupFiltered 在 usingGroup=auto 时挑选对该模型有可用候选的实际分组。
// filter 非空时仅统计通过过滤的渠道（例如视频 endpoint 能力）。
func resolveSmartRouteGroupFiltered(usingGroup, userGroup, modelName string, filter func(*model.Channel) bool) string {
	if usingGroup != "auto" {
		return usingGroup
	}
	for _, g := range GetUserAutoGroup(userGroup) {
		if filter == nil {
			if len(model.ListChannelIDsForGroupModel(g, modelName)) > 0 {
				return g
			}
			continue
		}
		cands, _ := buildRouterCandidatesFiltered(g, modelName, filter)
		if len(cands) > 0 {
			return g
		}
	}
	return ""
}

func orderEndpointCandidateIDsByPrice(cands []*router.EndpointCandidate) []int {
	if len(cands) == 0 {
		return nil
	}
	sorted := make([]*router.EndpointCandidate, 0, len(cands))
	for _, c := range cands {
		if c != nil {
			sorted = append(sorted, c)
		}
	}
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].UnitPrice < sorted[j].UnitPrice
	})
	out := make([]int, 0, len(sorted))
	for _, c := range sorted {
		out = append(out, c.ChannelID)
	}
	return out
}

func pickFirstEnabledChannel(orderedIDs []int) *model.Channel {
	for _, id := range orderedIDs {
		ch, err := model.CacheGetChannel(id)
		if err != nil || ch == nil || ch.Status != common.ChannelStatusEnabled {
			continue
		}
		return ch
	}
	return nil
}

// TrySmartRouteChannel runs in-process router-engine when SmartRouterEnabled(). On success it stores
// ContextKeySmartRouteChannelOrder for relay retries and returns the first channel.
func TrySmartRouteChannel(c *gin.Context, usingGroup, userGroup, modelName, providerJSON string) (*model.Channel, string, bool) {
	if !SmartRouterEnabled() {
		return nil, "", false
	}
	return TrySmartRouteChannelWithFilter(c, usingGroup, userGroup, modelName, providerJSON, nil)
}

// TrySmartRouteChannelWithFilter 在可选渠道过滤后做选路，并把完整有序候选写入
// ContextKeySmartRouteChannelOrder，供创建任务失败时按序保底切换。
//
// filter 非空时（如视频 submit）：即使 SmartRouter 关闭，也会按单价排序产出有序候选，
// 保证「创建保底」可用。filter 为空时行为与 TrySmartRouteChannel 一致（需 SmartRouter 开启）。
func TrySmartRouteChannelWithFilter(c *gin.Context, usingGroup, userGroup, modelName, providerJSON string, filter func(*model.Channel) bool) (*model.Channel, string, bool) {
	if filter == nil && !SmartRouterEnabled() {
		return nil, "", false
	}
	selectGroup := resolveSmartRouteGroupFiltered(usingGroup, userGroup, modelName, filter)
	if selectGroup == "" {
		return nil, "", false
	}
	cands, err := buildRouterCandidatesFiltered(selectGroup, modelName, filter)
	if err != nil || len(cands) == 0 {
		return nil, "", false
	}
	// 用户指定价：排除单价超出用户价格上限的渠道。
	cands = filterEndpointCandidatesByUserPriceCap(c.GetInt("id"), modelName, cands)
	if len(cands) == 0 {
		return nil, "", false
	}

	candidateIDs := make([]int, 0, len(cands))
	for _, cand := range cands {
		if cand != nil {
			candidateIDs = append(candidateIDs, cand.ChannelID)
		}
	}

	if SmartRouterEnabled() {
		models := []string{modelName}
		if raw, ok := common.GetContextKey(c, constant.ContextKeyRequestModelsList); ok {
			if sl, ok := raw.([]string); ok && len(sl) > 0 {
				models = sl
			}
		}
		req := router.SelectRequest{
			Models:                  models,
			ProviderPreferencesJSON: providerJSON,
			Candidates:              cands,
		}
		if v, ok := common.GetContextKey(c, constant.ContextKeyRequestHasTools); ok {
			if b, ok := v.(bool); ok {
				req.RequestHasTools = b
			}
		}
		if res, err := router.SelectProviders(req); err == nil && len(res.OrderedChannelIDs) > 0 {
			candidateIDs = res.OrderedChannelIDs
		} else if filter != nil {
			// 视频等过滤路径：router-engine 失败时仍按价格序保底，不能丢候选池。
			candidateIDs = orderEndpointCandidateIDsByPrice(cands)
		} else {
			return nil, "", false
		}
	} else {
		// SmartRouter 关闭：过滤路径仍给出价格序，保证创建失败可切换。
		candidateIDs = orderEndpointCandidateIDsByPrice(cands)
	}

	ch := pickFirstEnabledChannel(candidateIDs)
	if ch == nil {
		return nil, "", false
	}
	common.SetContextKey(c, constant.ContextKeySmartRouteChannelOrder, candidateIDs)
	common.SetContextKey(c, constant.ContextKeySmartRouteSelectGroup, selectGroup)
	if usingGroup == "auto" {
		common.SetContextKey(c, constant.ContextKeyAutoGroup, selectGroup)
	}
	return ch, selectGroup, true
}

// TrySupplierRouteChannel 在「强制供应商」语义下选择渠道：候选池限制为该供应商下满足
// (group, model) 条件的启用渠道。SmartRouter 开启时走 router-engine 排序；关闭或 router-engine
// 无可用候选时，回退到按优先级 + 权重的随机选择（与 GetRandomSatisfiedChannel 一致），并把最终
// 候选顺序写入 ContextKeySmartRouteChannelOrder，保证控制器侧重试也严格落在同一供应商内。
//
// 返回 (channel, selectGroup, true) 表示已完成选择；返回 false 时表示候选为空，调用方应按
// 正常"无可用渠道"错误处理，而不是再去兜底 SmartRouter / 随机，因为那会绕过供应商约束。
func TrySupplierRouteChannel(c *gin.Context, usingGroup, userGroup, modelName, providerJSON string, supplierApplicationID int) (*model.Channel, string, bool) {
	return TrySupplierRouteChannelWithFilter(c, usingGroup, userGroup, modelName, providerJSON, supplierApplicationID, nil)
}

// TrySupplierRouteChannelWithFilter 同 TrySupplierRouteChannel，并额外叠加 channelFilter
//（如视频 endpoint 能力），避免强制供应商池内落到不支持当前 relay 的渠道。
func TrySupplierRouteChannelWithFilter(c *gin.Context, usingGroup, userGroup, modelName, providerJSON string, supplierApplicationID int, channelFilter func(*model.Channel) bool) (*model.Channel, string, bool) {
	filter := func(ch *model.Channel) bool {
		if ch == nil || ch.SupplierApplicationID != supplierApplicationID {
			return false
		}
		if channelFilter != nil && !channelFilter(ch) {
			return false
		}
		return true
	}

	// 自动分组下挑选一个"对该供应商下的该模型有候选"的子分组。
	selectGroup := usingGroup
	if usingGroup == "auto" {
		selectGroup = ""
		for _, g := range GetUserAutoGroup(userGroup) {
			cands, _ := buildRouterCandidatesFiltered(g, modelName, filter)
			if len(cands) > 0 {
				selectGroup = g
				break
			}
		}
		if selectGroup == "" {
			return nil, "", false
		}
	}

	cands, err := buildRouterCandidatesFiltered(selectGroup, modelName, filter)
	if err != nil || len(cands) == 0 {
		return nil, "", false
	}
	// 用户指定价：供应商内候选同样受价格上限约束。
	cands = filterEndpointCandidatesByUserPriceCap(c.GetInt("id"), modelName, cands)
	if len(cands) == 0 {
		return nil, "", false
	}

	candidateIDs := make([]int, 0, len(cands))
	for _, cand := range cands {
		candidateIDs = append(candidateIDs, cand.ChannelID)
	}

	if SmartRouterEnabled() {
		models := []string{modelName}
		if raw, ok := common.GetContextKey(c, constant.ContextKeyRequestModelsList); ok {
			if sl, ok := raw.([]string); ok && len(sl) > 0 {
				models = sl
			}
		}
		req := router.SelectRequest{
			Models:                  models,
			ProviderPreferencesJSON: providerJSON,
			Candidates:              cands,
		}
		if v, ok := common.GetContextKey(c, constant.ContextKeyRequestHasTools); ok {
			if b, ok := v.(bool); ok {
				req.RequestHasTools = b
			}
		}
		if res, err := router.SelectProviders(req); err == nil && len(res.OrderedChannelIDs) > 0 {
			candidateIDs = res.OrderedChannelIDs
		}
	}

	// 按 candidateIDs 顺序取第一个启用渠道作为本次命中；其余供重试回退。
	chosen := pickFirstEnabledChannel(candidateIDs)
	if chosen == nil {
		return nil, "", false
	}
	common.SetContextKey(c, constant.ContextKeySmartRouteChannelOrder, candidateIDs)
	common.SetContextKey(c, constant.ContextKeySmartRouteSelectGroup, selectGroup)
	if usingGroup == "auto" {
		common.SetContextKey(c, constant.ContextKeyAutoGroup, selectGroup)
	}
	return chosen, selectGroup, true
}

// IngestChatCompletionRoutingHints parses provider / models / tools from JSON body (OpenRouter-compatible).
func IngestChatCompletionRoutingHints(c *gin.Context, modelName string) {
	if c == nil || !strings.Contains(c.Request.URL.Path, "chat/completions") {
		return
	}
	var pick struct {
		Provider json.RawMessage   `json:"provider"`
		Models   []string          `json:"models"`
		Tools    []json.RawMessage `json:"tools"`
	}
	if err := common.UnmarshalBodyReusable(c, &pick); err != nil {
		return
	}
	if len(pick.Provider) > 0 {
		common.SetContextKey(c, constant.ContextKeyOpenRouterProviderJSON, string(pick.Provider))
	}
	if len(pick.Models) > 0 {
		common.SetContextKey(c, constant.ContextKeyRequestModelsList, pick.Models)
	} else if modelName != "" {
		common.SetContextKey(c, constant.ContextKeyRequestModelsList, []string{modelName})
	}
	common.SetContextKey(c, constant.ContextKeyRequestHasTools, len(pick.Tools) > 0)
}
