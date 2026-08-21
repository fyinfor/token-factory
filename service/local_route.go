package service

import (
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

// ── 进程内归类路由（从 TokenFactory gRPC 剥离）──────────────────
//
// 使用本地 channels 表 + 路由配置表，不再依赖外部 TokenFactory。
// 排序算法：weight / price 走 router-engine；default 模式按渠道 ID 顺序（路由关闭）。

// RouteChannelCandidate 选路候选（由本地渠道构建）。
type RouteChannelCandidate struct {
	ChannelID     int
	ChannelName   string
	Priority      int
	Weight        int
	Price         float64
	Healthy       bool
	LatencyP50    float64
	ThroughputTps float64
	ProviderSlug  string
	Status        int
}

// SelectChannelLocalResult 本地选路结果。
type SelectChannelLocalResult struct {
	OrderedChannelIDs []int
	Strategy          string
	GroupKey          string
	Fallback          bool
}

// SelectChannelLocal 根据用户/全局路由模式对候选渠道排序。
func SelectChannelLocal(modelName string, userID int, candidates []RouteChannelCandidate) SelectChannelLocalResult {
	mode := ""
	if userID > 0 {
		if userCfg := model.GetUserRouteConfig(userID); userCfg != nil && userCfg.Mode != "" {
			mode = userCfg.Mode
		}
	}
	if mode == "" {
		mode = model.GetRouteConfig().Mode
	}

	// Status=0 视为调用方已过滤；否则仅保留启用渠道。
	filtered := make([]RouteChannelCandidate, 0, len(candidates))
	for _, c := range candidates {
		if c.Status != 0 && c.Status != common.ChannelStatusEnabled {
			continue
		}
		filtered = append(filtered, c)
	}

	globalOverrides, _ := model.LoadModelGroupOverrides()
	var userOverrides map[string]string
	if userID > 0 {
		userOverrides, _ = model.LoadUserModelGroupOverrides(userID)
	}
	groupKey := ResolveModelGroupKeyWithUser(modelName, userOverrides, globalOverrides)
	mode = ApplyGroupRouteDisableForModel(userID, modelName, groupKey, mode)
	if !model.IsImplementedRouteMode(mode) {
		return SelectChannelLocalResult{Strategy: mode, GroupKey: groupKey, Fallback: true}
	}

	var ordered []RouteChannelCandidate
	switch mode {
	case model.RouteModeWeight:
		ordered = ApplyGroupWeightsAndSortWithUser(userID, groupKey, filtered)
	case model.RouteModePrice:
		ordered = sortRouteCandidatesByPrice(filtered)
	default:
		return SelectChannelLocalResult{Strategy: mode, GroupKey: groupKey, Fallback: true}
	}

	if len(ordered) == 0 {
		return SelectChannelLocalResult{Strategy: mode, GroupKey: groupKey, Fallback: true}
	}

	ids := make([]int, 0, len(ordered))
	for _, c := range ordered {
		ids = append(ids, c.ChannelID)
	}
	return SelectChannelLocalResult{
		OrderedChannelIDs: ids,
		Strategy:          mode,
		GroupKey:          groupKey,
		Fallback:          false,
	}
}

// ResolveModelGroupKeyWithUser 优先用户级覆盖，再全局覆盖，最后自动归一化。
func ResolveModelGroupKeyWithUser(raw string, userOverrides map[string]string, globalOverrides map[string]string) string {
	key := strings.ToLower(strings.TrimSpace(raw))
	if key == "" {
		return ""
	}
	if ov, ok := userOverrides[key]; ok && ov != "" {
		return ov
	}
	if ov, ok := globalOverrides[key]; ok && ov != "" {
		return ov
	}
	return model.NormalizeModelName(raw)
}

// ApplyGroupWeightsAndSortWithUser 按归类权重过滤并排序候选（权重模式）。
//
//	查找顺序：用户级权重 → 全局权重 → 候选自身权重
func ApplyGroupWeightsAndSortWithUser(userID int, groupKey string, candidates []RouteChannelCandidate) []RouteChannelCandidate {
	userWeights, _ := model.LoadUserModelGroupWeights(userID, groupKey)
	userWeightMap := make(map[int]model.UserModelGroupWeight, len(userWeights))
	for _, w := range userWeights {
		userWeightMap[w.ChannelID] = w
	}

	globalWeights, _ := model.LoadModelGroupWeights(groupKey)
	globalWeightMap := make(map[int]model.ModelGroupWeight, len(globalWeights))
	for _, w := range globalWeights {
		globalWeightMap[w.ChannelID] = w
	}

	out := make([]RouteChannelCandidate, 0, len(candidates))
	for _, c := range candidates {
		if uw, ok := userWeightMap[c.ChannelID]; ok {
			if !uw.Enabled || uw.Weight <= 0 {
				continue
			}
			c.Weight = uw.Weight
		} else if gw, ok := globalWeightMap[c.ChannelID]; ok {
			if !gw.Enabled || gw.Weight <= 0 {
				continue
			}
			c.Weight = gw.Weight
		}
		out = append(out, c)
	}
	if len(out) == 0 {
		return nil
	}
	return sortRouteCandidatesByWeightDesc(out)
}

func sortRouteCandidatesByPrice(candidates []RouteChannelCandidate) []RouteChannelCandidate {
	result := make([]RouteChannelCandidate, len(candidates))
	copy(result, candidates)
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Price != result[j].Price {
			return result[i].Price < result[j].Price
		}
		return result[i].ChannelID < result[j].ChannelID
	})
	return result
}

func sortRouteCandidatesByWeightDesc(candidates []RouteChannelCandidate) []RouteChannelCandidate {
	result := make([]RouteChannelCandidate, len(candidates))
	copy(result, candidates)
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Weight != result[j].Weight {
			return result[i].Weight > result[j].Weight
		}
		return result[i].ChannelID < result[j].ChannelID
	})
	return result
}

func sanitizeUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	return strings.ToValidUTF8(s, "")
}
