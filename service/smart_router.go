package service

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/fyinfor/router-engine/pkg/router"
	"github.com/gin-gonic/gin"
)

// SmartRouterEnabled 已废弃：统一由 router-engine 处理智能路由，恒为 true。
func SmartRouterEnabled() bool {
	return true
}

// TrySmartRouteChannel 兼容入口，委托 TryEngineRoute。
func TrySmartRouteChannel(c *gin.Context, usingGroup, userGroup, modelName, providerJSON string) (*model.Channel, string, bool) {
	return TryEngineRoute(c, usingGroup, userGroup, modelName, providerJSON, nil)
}

// TrySmartRouteChannelWithFilter 兼容入口，委托 TryEngineRoute。
func TrySmartRouteChannelWithFilter(c *gin.Context, usingGroup, userGroup, modelName, providerJSON string, filter func(*model.Channel) bool) (*model.Channel, string, bool) {
	return TryEngineRoute(c, usingGroup, userGroup, modelName, providerJSON, filter)
}

// TrySupplierRouteChannel 兼容入口，委托 TrySupplierEngineRoute。
func TrySupplierRouteChannel(c *gin.Context, usingGroup, userGroup, modelName, providerJSON string, supplierApplicationID int) (*model.Channel, string, bool) {
	return TrySupplierEngineRoute(c, usingGroup, userGroup, modelName, providerJSON, supplierApplicationID, nil)
}

// TrySupplierRouteChannelWithFilter 兼容入口，委托 TrySupplierEngineRoute。
func TrySupplierRouteChannelWithFilter(c *gin.Context, usingGroup, userGroup, modelName, providerJSON string, supplierApplicationID int, channelFilter func(*model.Channel) bool) (*model.Channel, string, bool) {
	return TrySupplierEngineRoute(c, usingGroup, userGroup, modelName, providerJSON, supplierApplicationID, channelFilter)
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
		out = append(out, &router.EndpointCandidate{
			ChannelID:         ch.Id,
			Model:             modelName,
			ProviderSlug:      channelProviderSlug(ch),
			UnitPrice:         unitPrice,
			Healthy:           true,
			LatencyP50Seconds: latSec,
			ThroughputTps:     tps,
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
