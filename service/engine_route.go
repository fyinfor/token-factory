package service

import (
	"fmt"
	"sort"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/fyinfor/router-engine/pkg/router"
	"github.com/gin-gonic/gin"
)

// ResolveEffectiveRouteMode 解析用户最终生效的路由模式（用户配置 → 全局配置）。
func ResolveEffectiveRouteMode(userID int) string {
	if userID > 0 {
		if userCfg := model.GetUserRouteConfig(userID); userCfg != nil && userCfg.Mode != "" {
			return userCfg.Mode
		}
	}
	return model.GetRouteConfig().Mode
}

// IsRouteStrategyEnabled 返回是否启用智能路由策略（weight / price）；default 表示路由关闭。
func IsRouteStrategyEnabled(mode string) bool {
	return mode == model.RouteModeWeight || mode == model.RouteModePrice
}

// ApplyGroupRouteDisable 若用户对该归类（或原始模型名）关闭了智能路由，则降级为关闭路由模式。
func ApplyGroupRouteDisable(userID int, groupKey, mode string) string {
	return ApplyGroupRouteDisableForModel(userID, "", groupKey, mode)
}

// ApplyGroupRouteDisableForModel 同时按归类 key 与原始模型名匹配关闭开关。
func ApplyGroupRouteDisableForModel(userID int, modelName, groupKey, mode string) string {
	if !IsRouteStrategyEnabled(mode) {
		return mode
	}
	if model.IsUserModelRouteDisabled(userID, modelName, groupKey) {
		return model.RouteModeDefault
	}
	return mode
}

// EffectiveRouteStrategyEnabled 用户对该模型是否启用了智能路由（weight/price，且未单独关闭）。
func EffectiveRouteStrategyEnabled(userID int, modelName string) bool {
	mode := ResolveEffectiveRouteMode(userID)
	globalOverrides, _ := model.LoadModelGroupOverrides()
	var userOverrides map[string]string
	if userID > 0 {
		userOverrides, _ = model.LoadUserModelGroupOverrides(userID)
	}
	groupKey := ResolveModelGroupKeyWithUser(modelName, userOverrides, globalOverrides)
	mode = ApplyGroupRouteDisableForModel(userID, modelName, groupKey, mode)
	return IsRouteStrategyEnabled(mode)
}

// OrderRouteChannelIDs 按路由模式对候选渠道排序；default 模式按渠道 ID 升序（路由关闭）。
func OrderRouteChannelIDs(c *gin.Context, userID int, modelName, groupKey, mode string, candidates []RouteChannelCandidate, providerJSON string) ([]int, string) {
	if len(candidates) == 0 {
		return nil, mode
	}
	if UserPricingUsesManualChannelPriority(userID, modelName) {
		sorted := SortRouteCandidatesByUserPricingPriority(userID, modelName, candidates)
		ids := make([]int, 0, len(sorted))
		for _, cand := range sorted {
			ids = append(ids, cand.ChannelID)
		}
		return ids, model.UserPricingModeChannelList
	}
	if !IsRouteStrategyEnabled(mode) {
		ids := orderRouteCandidatesByChannelID(candidates)
		if len(ids) > 1 {
			ids = ids[:1]
		}
		return ids, mode
	}

	endpointCands := buildEndpointCandidatesFromRoute(candidates, userID, groupKey, modelName, mode)
	if len(endpointCands) == 0 {
		return orderRouteCandidatesByChannelID(candidates), mode
	}

	models := []string{modelName}
	if c != nil {
		if raw, ok := common.GetContextKey(c, constant.ContextKeyRequestModelsList); ok {
			if sl, ok := raw.([]string); ok && len(sl) > 0 {
				models = sl
			}
		}
	}
	req := router.SelectRequest{
		Models:                  models,
		ProviderPreferencesJSON: providerJSON,
		Candidates:              endpointCands,
	}
	if c != nil {
		if v, ok := common.GetContextKey(c, constant.ContextKeyRequestHasTools); ok {
			if b, ok := v.(bool); ok {
				req.RequestHasTools = b
			}
		}
	}
	if res, err := router.SelectProviders(req); err == nil && len(res.OrderedChannelIDs) > 0 {
		return res.OrderedChannelIDs, res.Strategy
	}
	if mode == model.RouteModePrice {
		return orderEndpointCandidateIDsByPrice(endpointCands), "price_fallback"
	}
	return orderRouteCandidatesByGroupWeight(candidates), "weight_fallback"
}

func orderRouteCandidatesByChannelID(candidates []RouteChannelCandidate) []int {
	sorted := append([]RouteChannelCandidate(nil), candidates...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].ChannelID < sorted[j].ChannelID
	})
	ids := make([]int, 0, len(sorted))
	for _, c := range sorted {
		ids = append(ids, c.ChannelID)
	}
	return ids
}

func orderRouteCandidatesByGroupWeight(candidates []RouteChannelCandidate) []int {
	sorted := sortRouteCandidatesByWeightDesc(candidates)
	ids := make([]int, 0, len(sorted))
	for _, c := range sorted {
		ids = append(ids, c.ChannelID)
	}
	return ids
}

func buildEndpointCandidatesFromRoute(candidates []RouteChannelCandidate, userID int, groupKey, modelName, mode string) []*router.EndpointCandidate {
	weightMap := map[int]int{}
	if mode == model.RouteModeWeight && groupKey != "" {
		if userWeights, _ := model.LoadUserModelGroupWeights(userID, groupKey); len(userWeights) > 0 {
			for _, w := range userWeights {
				if w.Enabled && w.Weight > 0 {
					weightMap[w.ChannelID] = w.Weight
				}
			}
		} else if globalWeights, _ := model.LoadModelGroupWeights(groupKey); len(globalWeights) > 0 {
			for _, w := range globalWeights {
				if w.Enabled && w.Weight > 0 {
					weightMap[w.ChannelID] = w.Weight
				}
			}
		}
	}

	out := make([]*router.EndpointCandidate, 0, len(candidates))
	for _, cand := range candidates {
		ch, err := model.CacheGetChannel(cand.ChannelID)
		if err != nil || ch == nil {
			continue
		}
		w := 0
		if mode == model.RouteModeWeight {
			if gw, ok := weightMap[cand.ChannelID]; ok {
				w = gw
			} else {
				continue
			}
		}
		unitPrice := cand.Price
		if unitPrice <= 0 {
			unitPrice = ResolveChannelModelUnitPrice(ch, modelName)
		}
		latSec := float64(ch.ResponseTime) / 1000.0
		if latSec <= 0 {
			latSec = 0.001
		}
		out = append(out, &router.EndpointCandidate{
			ChannelID:         ch.Id,
			Model:             modelName,
			ProviderSlug:      channelProviderSlug(ch),
			UnitPrice:         unitPrice,
			Healthy:           cand.Healthy,
			LatencyP50Seconds: latSec,
			ThroughputTps:     1.0 / latSec,
			Weight:            w,
		})
	}
	return out
}

// TryEngineRoute 统一选路入口：default=路由关闭（按渠道 ID）；weight/price=router-engine。
func TryEngineRoute(c *gin.Context, usingGroup, userGroup, modelName, providerJSON string, filter func(*model.Channel) bool) (*model.Channel, string, bool) {
	selectGroup := resolveSmartRouteGroupFiltered(usingGroup, userGroup, modelName, filter)
	if selectGroup == "" {
		return nil, "", false
	}

	userID := 0
	if c != nil {
		userID = c.GetInt("id")
	}
	candidates := CollectSameModelRouteCandidatesWithFilter(selectGroup, modelName, filter)
	if len(candidates) == 0 {
		return nil, "", false
	}
	candidates = FilterRouteCandidatesByUserPriceCap(userID, modelName, candidates)
	if len(candidates) == 0 {
		return nil, "", false
	}

	mode := ResolveEffectiveRouteMode(userID)
	globalOverrides, _ := model.LoadModelGroupOverrides()
	var userOverrides map[string]string
	if userID > 0 {
		userOverrides, _ = model.LoadUserModelGroupOverrides(userID)
	}
	groupKey := ResolveModelGroupKeyWithUser(modelName, userOverrides, globalOverrides)
	mode = ApplyGroupRouteDisableForModel(userID, modelName, groupKey, mode)

	ordered, strategy := OrderRouteChannelIDs(c, userID, modelName, groupKey, mode, candidates, providerJSON)
	if len(ordered) == 0 {
		return nil, "", false
	}

	isEnabled := func(id int) bool {
		ch, err := model.CacheGetChannel(id)
		return err == nil && ch != nil && ch.Status == common.ChannelStatusEnabled
	}
	picked, ok := TFRoutePickChannel(c, groupKey, selectGroup, strategy, ordered, isEnabled)
	if !ok {
		return nil, "", false
	}
	ch, err := model.CacheGetChannel(picked)
	if err != nil || ch == nil || ch.Status != common.ChannelStatusEnabled {
		return nil, "", false
	}
	if filter != nil && !filter(ch) {
		return nil, "", false
	}

	common.SetContextKey(c, constant.ContextKeySmartRouteChannelOrder, ordered)
	common.SetContextKey(c, constant.ContextKeySmartRouteSelectGroup, selectGroup)
	if usingGroup == "auto" {
		common.SetContextKey(c, constant.ContextKeyAutoGroup, selectGroup)
	}
	if c != nil {
		logger.LogInfo(c, fmt.Sprintf("engine_route selected: channel=%s(id=%d) model=%s group=%s mode=%s strategy=%s ordered=%v",
			ch.Name, ch.Id, modelName, selectGroup, mode, strategy, ordered))
	}
	return ch, selectGroup, true
}

// TrySupplierEngineRoute 在强制供应商约束下使用 router-engine 选路。
func TrySupplierEngineRoute(c *gin.Context, usingGroup, userGroup, modelName, providerJSON string, supplierApplicationID int, channelFilter func(*model.Channel) bool) (*model.Channel, string, bool) {
	combinedFilter := func(ch *model.Channel) bool {
		if ch == nil || ch.SupplierApplicationID != supplierApplicationID {
			return false
		}
		if channelFilter != nil && !channelFilter(ch) {
			return false
		}
		return true
	}
	selectGroup := usingGroup
	if usingGroup == "auto" {
		selectGroup = ""
		for _, g := range GetUserAutoGroup(userGroup) {
			if len(CollectSameModelRouteCandidatesWithFilter(g, modelName, combinedFilter)) > 0 {
				selectGroup = g
				break
			}
		}
		if selectGroup == "" {
			return nil, "", false
		}
	}

	userID := 0
	if c != nil {
		userID = c.GetInt("id")
	}
	candidates := CollectSameModelRouteCandidatesWithFilter(selectGroup, modelName, combinedFilter)
	if len(candidates) == 0 {
		return nil, "", false
	}
	candidates = FilterRouteCandidatesByUserPriceCap(userID, modelName, candidates)
	if len(candidates) == 0 {
		return nil, "", false
	}

	mode := ResolveEffectiveRouteMode(userID)
	globalOverrides, _ := model.LoadModelGroupOverrides()
	var userOverrides map[string]string
	if userID > 0 {
		userOverrides, _ = model.LoadUserModelGroupOverrides(userID)
	}
	groupKey := ResolveModelGroupKeyWithUser(modelName, userOverrides, globalOverrides)
	mode = ApplyGroupRouteDisableForModel(userID, modelName, groupKey, mode)
	ordered, strategy := OrderRouteChannelIDs(c, userID, modelName, groupKey, mode, candidates, providerJSON)
	if len(ordered) == 0 {
		return nil, "", false
	}

	isEnabled := func(id int) bool {
		ch, err := model.CacheGetChannel(id)
		return err == nil && ch != nil && ch.Status == common.ChannelStatusEnabled
	}
	picked, ok := TFRoutePickChannel(c, groupKey, selectGroup, strategy, ordered, isEnabled)
	if !ok {
		return nil, "", false
	}
	ch, err := model.CacheGetChannel(picked)
	if err != nil || ch == nil || ch.Status != common.ChannelStatusEnabled {
		return nil, "", false
	}
	common.SetContextKey(c, constant.ContextKeySmartRouteChannelOrder, ordered)
	common.SetContextKey(c, constant.ContextKeySmartRouteSelectGroup, selectGroup)
	if usingGroup == "auto" {
		common.SetContextKey(c, constant.ContextKeyAutoGroup, selectGroup)
	}
	return ch, selectGroup, true
}
