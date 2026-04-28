package service

import (
	"encoding/json"
	"os"
	"strings"
	"time"

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

func buildRouterCandidates(group, modelName string) ([]*router.EndpointCandidate, error) {
	return buildRouterCandidatesFiltered(group, modelName, nil)
}

// buildRouterCandidatesFiltered 在 buildRouterCandidates 基础上额外支持按渠道过滤。
// filter 为 nil 时行为与 buildRouterCandidates 相同；filter 返回 false 的渠道将被剔除。
//
// PR4：候选构建时把 (channel × modelName) 维度的真实测试结果（来自 model_test_results）
// 塞进 EndpointCandidate，让 router-engine 的 sort=latency / sort=throughput / 熔断
// 行为对应到 PR1 采集的 EWMA 时延 / 实测 TPS / CircuitState，而不是渠道粗粒度的
// ch.ResponseTime + 伪造的 1/latSec 吞吐。
//
// 字段优先级：
//   - LatencyP50Seconds：LatencyEwmaMs > LastResponseTime > ch.ResponseTime；
//     选 EWMA 是因为单次抖动会被指数加权抹平，比单次更适合做排序信号。
//   - ThroughputTps：LastThroughputTps（>0 时）> 1/latSec 兜底；
//     兜底是为了让没采到吞吐的旧渠道仍然能参与 sort=throughput（隐含「越快越好」）。
//   - Healthy：CircuitState=open 且 now < UnhealthyUntil 时 false；router-engine
//     会把 unhealthy 候选降级到二级桶，不直接踢出，保留「全部熔断时不至于 503」的退化能力。
func buildRouterCandidatesFiltered(group, modelName string, filter func(*model.Channel) bool) ([]*router.EndpointCandidate, error) {
	ids := model.ListChannelIDsForGroupModel(group, modelName)
	if len(ids) == 0 {
		return nil, nil
	}
	// 一次拉本批所有渠道的 (channel, modelName) 健康行，避免循环里 N+1。
	// 失败不致命：metrics 缺失就退化到老逻辑（ch.ResponseTime），仍能选出渠道。
	metricsByChannel := make(map[int]*model.ModelTestResult, len(ids))
	if rows, err := model.GetModelTestResultsByModelNameAndChannelIDs(modelName, ids); err == nil {
		for i := range rows {
			row := rows[i]
			metricsByChannel[row.ChannelId] = &row
		}
	}
	nowUnix := time.Now().Unix()
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
		unitPrice := 1.0
		sid := ch.SupplierApplicationID
		if p, ok := model.ResolveSupplierScopedFixedModelPrice(ch.Id, sid, modelName); ok {
			unitPrice = p
		} else if r, ok, _ := model.ResolveSupplierScopedModelRatio(ch.Id, sid, modelName); ok {
			unitPrice = r
		}
		if unitPrice <= 0 {
			ratio, _, _ := ratio_setting.GetModelRatio(modelName)
			if ratio > 0 {
				unitPrice = ratio
			}
		}
		if unitPrice <= 0 {
			unitPrice = 1
		}
		latSec, tps, healthy := candidatePerfFromMetrics(ch, metricsByChannel[ch.Id], nowUnix)
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
			Healthy:           healthy,
			LatencyP50Seconds: latSec,
			ThroughputTps:     tps,
			Priority:          prio,
			Weight:            w,
		})
	}
	return out, nil
}

// candidatePerfFromMetrics 把 (channel × model) 测试结果合并成 router-engine 需要的
// (LatencyP50Seconds, ThroughputTps, Healthy) 三元组。抽出独立函数便于单测覆盖三种分支：
//   - mtr=nil：完全退化到 ch.ResponseTime，与 PR4 之前行为一致；
//   - mtr 有 EWMA / TPS：用真实测试值；
//   - mtr.CircuitState=open 且未到期：Healthy=false。
//
// 该函数不读 DB、不依赖 ctx，纯函数；nowUnix 由调用方注入便于测试穿越时间。
func candidatePerfFromMetrics(ch *model.Channel, mtr *model.ModelTestResult, nowUnix int64) (latSec float64, tps float64, healthy bool) {
	healthy = true
	latSec = float64(ch.ResponseTime) / 1000.0
	if mtr != nil {
		switch {
		case mtr.LatencyEwmaMs > 0:
			latSec = mtr.LatencyEwmaMs / 1000.0
		case mtr.LastResponseTime > 0:
			latSec = float64(mtr.LastResponseTime) / 1000.0
		}
	}
	if latSec <= 0 {
		latSec = 0.001
	}
	if mtr != nil && mtr.LastThroughputTps > 0 {
		tps = mtr.LastThroughputTps
	} else {
		tps = 1.0 / latSec
	}
	if mtr != nil && mtr.CircuitState == model.CircuitStateOpen && mtr.UnhealthyUntil > nowUnix {
		healthy = false
	}
	return latSec, tps, healthy
}

func resolveSmartRouteGroup(usingGroup, userGroup, modelName string) string {
	if usingGroup != "auto" {
		return usingGroup
	}
	for _, g := range GetUserAutoGroup(userGroup) {
		if len(model.ListChannelIDsForGroupModel(g, modelName)) > 0 {
			return g
		}
	}
	return ""
}

// TrySmartRouteChannel runs in-process router-engine when SmartRouterEnabled(). On success it stores
// ContextKeySmartRouteChannelOrder for relay retries and returns the first channel.
func TrySmartRouteChannel(c *gin.Context, usingGroup, userGroup, modelName, providerJSON string) (*model.Channel, string, bool) {
	return TrySmartRouteChannelWithPolicy(c, usingGroup, userGroup, modelName, providerJSON, nil)
}

// TrySmartRouteChannelWithPolicy 在 TrySmartRouteChannel 的基础上叠加「用户路由策略」语义：
//
//   - 当 resolved 非 nil 且 policy.EffectiveProviderJSON 非空时，优先使用其作为 router-engine
//     的 provider 偏好；只有当 effective 为空时才回退用入参 providerJSON。
//   - 当 resolved 非 nil 且存在候选池约束（HasCandidatePoolConstraint），构建候选时按
//     IsCandidateAllowed 过滤；候选为空（说明 user 候选池命中不到当前 group/model 的活跃渠道），
//     交由调用方决定是否走 fallback；此函数本身不做 fallback 扩展，保持单一职责。
//   - 不设上下文（fallback_used 等监控指标）由 distributor 决定写入。
func TrySmartRouteChannelWithPolicy(c *gin.Context, usingGroup, userGroup, modelName, providerJSON string, resolved *ResolvedRoutingPolicy) (*model.Channel, string, bool) {
	if !SmartRouterEnabled() {
		return nil, "", false
	}

	var filter func(ch *model.Channel) bool
	if resolved != nil && resolved.HasCandidatePoolConstraint() {
		filter = func(ch *model.Channel) bool {
			return resolved.IsCandidateAllowed(ch.Id, modelName)
		}
	}

	// auto 分组：挨个找一个「过滤后仍有候选」的子分组；非 auto 直接用 usingGroup。
	// 加 filter 后单纯靠 resolveSmartRouteGroup 可能选到空组，所以在这里就近重选。
	selectGroup := usingGroup
	if usingGroup == "auto" {
		selectGroup = ""
		for _, g := range GetUserAutoGroup(userGroup) {
			c2, _ := buildRouterCandidatesFiltered(g, modelName, filter)
			if len(c2) > 0 {
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
	if resolved != nil && strings.TrimSpace(resolved.EffectiveProviderJSON) != "" {
		providerJSON = resolved.EffectiveProviderJSON
	}
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
	res, err := router.SelectProviders(req)
	if err != nil || len(res.OrderedChannelIDs) == 0 {
		return nil, "", false
	}
	common.SetContextKey(c, constant.ContextKeySmartRouteChannelOrder, res.OrderedChannelIDs)
	common.SetContextKey(c, constant.ContextKeySmartRouteSelectGroup, selectGroup)
	firstID := res.OrderedChannelIDs[0]
	ch, err := model.CacheGetChannel(firstID)
	if err != nil || ch == nil || ch.Status != common.ChannelStatusEnabled {
		return nil, "", false
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
	filter := func(ch *model.Channel) bool { return ch.SupplierApplicationID == supplierApplicationID }

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

	candidateIDs := make([]int, 0, len(cands))
	for _, c := range cands {
		candidateIDs = append(candidateIDs, c.ChannelID)
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
	var chosen *model.Channel
	for _, id := range candidateIDs {
		ch, err := model.CacheGetChannel(id)
		if err != nil || ch == nil || ch.Status != common.ChannelStatusEnabled {
			continue
		}
		chosen = ch
		break
	}
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
