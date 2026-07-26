package service

import (
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// RouteOrderedChannelIDs 返回本次请求写入的有序渠道路由候选（智能路由 / TokenFactory 路由）。
func RouteOrderedChannelIDs(c *gin.Context) ([]int, bool) {
	if c == nil {
		return nil, false
	}
	orderAny, ok := common.GetContextKey(c, constant.ContextKeySmartRouteChannelOrder)
	if !ok {
		return nil, false
	}
	order, ok := orderAny.([]int)
	if !ok || len(order) == 0 {
		return nil, false
	}
	return order, true
}

// RouteOrderedMaxRetries 有序路由下允许的最大重试次数（含首跳后的 failover 次数）。
// 当存在多个有序候选时，至少允许遍历完整列表；否则回退全局 RetryTimes。
func RouteOrderedMaxRetries(c *gin.Context) int {
	order, ok := RouteOrderedChannelIDs(c)
	if !ok || len(order) <= 1 {
		return common.RetryTimes
	}
	routeRetries := len(order) - 1
	if routeRetries > common.RetryTimes {
		return routeRetries
	}
	return common.RetryTimes
}

// UsedChannelIDSet 返回本请求已尝试过的渠道 ID 集合（来自 relay 的 use_channel 日志键）。
func UsedChannelIDSet(c *gin.Context) map[int]bool {
	used := make(map[int]bool)
	if c == nil {
		return used
	}
	for _, raw := range c.GetStringSlice("use_channel") {
		id, err := strconv.Atoi(raw)
		if err != nil || id <= 0 {
			continue
		}
		used[id] = true
	}
	return used
}

// HasUnusedOrderedRouteChannel 判断有序智能路由是否仍有未尝试的启用渠道（可用于 4xx/5xx 保底切换）。
func HasUnusedOrderedRouteChannel(c *gin.Context) bool {
	order, ok := RouteOrderedChannelIDs(c)
	if !ok || len(order) == 0 {
		return false
	}
	used := UsedChannelIDSet(c)
	for _, id := range order {
		if used[id] {
			continue
		}
		ch, err := model.CacheGetChannel(id)
		if err != nil || ch == nil || ch.Status != common.ChannelStatusEnabled {
			continue
		}
		return true
	}
	return false
}

// PickNextOrderedRouteChannel 在有序候选中选取下一个「未尝试过且仍启用」的渠道。
func PickNextOrderedRouteChannel(c *gin.Context, order []int) (*model.Channel, bool) {
	if len(order) == 0 {
		return nil, false
	}
	used := UsedChannelIDSet(c)
	for _, id := range order {
		if used[id] {
			continue
		}
		ch, err := model.CacheGetChannel(id)
		if err != nil || ch == nil || ch.Status != common.ChannelStatusEnabled {
			continue
		}
		return ch, true
	}
	return nil, false
}

// RefreshTFRouteStickyChannel 在有序 failover 切换渠道后刷新 TF 黏性上下文，
// 使 RecordTFRouteResult 能正确反馈到当前实际使用的渠道。
func RefreshTFRouteStickyChannel(c *gin.Context, channelID int) {
	if c == nil || channelID <= 0 {
		return
	}
	stickyKey, _, ok := tfGetStickyContext(c)
	if !ok {
		return
	}
	tfSetStickyContext(c, stickyKey, channelID)
}
