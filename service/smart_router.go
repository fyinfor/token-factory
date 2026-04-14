package service

import (
	"encoding/json"
	"os"
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

func buildRouterCandidates(group, modelName string) ([]*router.EndpointCandidate, error) {
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
		ratio, _, _ := ratio_setting.GetModelRatio(modelName)
		if ratio <= 0 {
			ratio = 1
		}
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
			UnitPrice:         ratio,
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
	if !SmartRouterEnabled() {
		return nil, "", false
	}
	selectGroup := resolveSmartRouteGroup(usingGroup, userGroup, modelName)
	if selectGroup == "" {
		return nil, "", false
	}
	cands, err := buildRouterCandidates(selectGroup, modelName)
	if err != nil || len(cands) == 0 {
		return nil, "", false
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
