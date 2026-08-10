package service

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// CollectSameModelRouteCandidates 收集分组下同模型的启用渠道候选（含单价）。
func CollectSameModelRouteCandidates(group, modelName string) []RouteChannelCandidate {
	return CollectSameModelRouteCandidatesWithFilter(group, modelName, nil)
}

// CollectSameModelRouteCandidatesWithFilter 同 CollectSameModelRouteCandidates，并支持渠道过滤
//（如视频 submit 仅保留具备视频 endpoint 的渠道）。
func CollectSameModelRouteCandidatesWithFilter(group, modelName string, filter func(*model.Channel) bool) []RouteChannelCandidate {
	channelIDs := model.GetGroupEnabledChannelIDs(group, modelName)
	if len(channelIDs) == 0 {
		return nil
	}
	var candidates []RouteChannelCandidate
	seen := make(map[int]bool, len(channelIDs))
	for _, cid := range channelIDs {
		if seen[cid] {
			continue
		}
		seen[cid] = true
		ch, err := model.CacheGetChannel(cid)
		if err != nil || ch == nil || ch.Status != common.ChannelStatusEnabled {
			continue
		}
		if filter != nil && !filter(ch) {
			continue
		}
		candidates = append(candidates, RouteChannelCandidate{
			ChannelID:    ch.Id,
			ChannelName:  ch.Name,
			Priority:     int(ch.GetPriority()),
			Weight:       ch.GetWeight(),
			Status:       ch.Status,
			ProviderSlug: strings.ToLower(strings.TrimSpace(ch.SupplierType)),
			Price:        ResolveChannelModelUnitPrice(ch, modelName),
			Healthy:      true,
		})
	}
	return candidates
}

// PreferChannelFirst 将 preferredID 置于有序候选首位（若已存在则去重后提前；若不在列表中则插入首位）。
func PreferChannelFirst(order []int, preferredID int) []int {
	if preferredID <= 0 {
		return order
	}
	out := make([]int, 0, len(order)+1)
	out = append(out, preferredID)
	for _, id := range order {
		if id == preferredID {
			continue
		}
		out = append(out, id)
	}
	return out
}

// resolveConcreteGroupForModel 在 usingGroup=auto 时挑选对该模型有候选的实际分组。
func resolveConcreteGroupForModel(c *gin.Context, usingGroup, modelName string) string {
	if usingGroup != "auto" {
		return usingGroup
	}
	userGroup := ""
	if c != nil {
		userGroup = common.GetContextKeyString(c, constant.ContextKeyUserGroup)
	}
	for _, g := range GetUserAutoGroup(userGroup) {
		if len(model.GetGroupEnabledChannelIDs(g, modelName)) > 0 {
			return g
		}
	}
	return usingGroup
}

// BuildPreferredChannelFailoverOrder 为 {model}/{route_slug} 构建保底有序候选：
// 首跳为偏好渠道，其后按当前智能路由策略（weight/price；未实现则价格升序）排列同模型其余渠道。
func BuildPreferredChannelFailoverOrder(c *gin.Context, modelName, group string, preferredID int) []int {
	return BuildPreferredChannelFailoverOrderWithFilter(c, modelName, group, preferredID, nil)
}

// BuildPreferredChannelFailoverOrderWithFilter 同 BuildPreferredChannelFailoverOrder，
// 并对候选池应用 filter（视频创建保底时排除无视频 endpoint 的渠道）。
func BuildPreferredChannelFailoverOrderWithFilter(c *gin.Context, modelName, group string, preferredID int, filter func(*model.Channel) bool) []int {
	group = resolveConcreteGroupForModel(c, group, modelName)
	candidates := CollectSameModelRouteCandidatesWithFilter(group, modelName, filter)

	userID := 0
	if c != nil {
		userID = c.GetInt("id")
	}
	// 用户指定价：保底候选与偏好首跳都不允许超出价格上限。
	candidates = FilterRouteCandidatesByUserPriceCap(userID, modelName, candidates)
	if preferredID > 0 {
		if ch, err := model.CacheGetChannel(preferredID); err == nil && ch != nil {
			if !ChannelWithinUserPriceCap(userID, modelName, ch) {
				preferredID = 0
			} else if filter != nil && !filter(ch) {
				// 偏好渠道本身不支持当前 relay：清除偏好，只在合法池内排序。
				preferredID = 0
			}
		}
	}

	if len(candidates) == 0 {
		if preferredID > 0 {
			return []int{preferredID}
		}
		return nil
	}

	res := SelectChannelLocal(modelName, userID, candidates)
	var ordered []int
	if !res.Fallback && len(res.OrderedChannelIDs) > 0 {
		ordered = res.OrderedChannelIDs
	} else {
		// 未启用 weight/price 时仍按价格升序保底，与系统默认价格优先对齐。
		sorted := sortRouteCandidatesByPrice(candidates)
		ordered = make([]int, 0, len(sorted))
		for _, cand := range sorted {
			ordered = append(ordered, cand.ChannelID)
		}
	}
	return PreferChannelFirst(ordered, preferredID)
}
