package service

import (
	"fmt"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/pkg/cachex"
	"github.com/gin-gonic/gin"
	"github.com/samber/hot"
)

// ── TokenFactory 路由黏性 + 报错熔断 ─────────────────────────────
//
// 设计目标（对应用户需求「指定渠道后考虑缓存，多次报错才切换」）：
//   - 黏性：某个 (归类 groupKey + group) 维度一旦选定渠道，后续相同维度复用同一渠道，
//     避免每次请求抖动（仍按 TokenFactory 返回的有序候选作为候选集）。
//   - 熔断：对 (维度, 渠道) 维护「连续报错计数」，达到阈值（默认 3）才摘除该黏性渠道、
//     顺延到下一个候选；任意一次成功即清零，体现「连续」。
//
// 计数与绑定走 HybridCache（内存 + Redis），多实例下也能近似一致；TTL 控制窗口。

const (
	tfRouteStickyNamespace = "new-api:tf_route_sticky:v1"
	tfRouteErrorNamespace  = "new-api:tf_route_error:v1"

	ginKeyTFRouteStickyKey = "tf_route_sticky_key"
	ginKeyTFRouteChannelID = "tf_route_channel_id"
)

var (
	tfRouteStickyOnce  sync.Once
	tfRouteStickyCache *cachex.HybridCache[int]

	tfRouteErrorOnce  sync.Once
	tfRouteErrorCache *cachex.HybridCache[int]

	tfRouteErrorLocks [64]sync.Mutex
)

func getTFRouteStickyCache() *cachex.HybridCache[int] {
	tfRouteStickyOnce.Do(func() {
		ttl := time.Duration(common.TokenFactoryRouteStickyTTLSeconds()) * time.Second
		tfRouteStickyCache = cachex.NewHybridCache[int](cachex.HybridCacheConfig[int]{
			Namespace: cachex.Namespace(tfRouteStickyNamespace),
			Redis:     common.RDB,
			RedisEnabled: func() bool {
				return common.RedisEnabled && common.RDB != nil
			},
			RedisCodec: cachex.IntCodec{},
			Memory: func() *hot.HotCache[string, int] {
				return hot.NewHotCache[string, int](hot.LRU, 100_000).
					WithTTL(ttl).
					WithJanitor().
					Build()
			},
		})
	})
	return tfRouteStickyCache
}

func getTFRouteErrorCache() *cachex.HybridCache[int] {
	tfRouteErrorOnce.Do(func() {
		ttl := time.Duration(common.TokenFactoryRouteErrorTTLSeconds()) * time.Second
		tfRouteErrorCache = cachex.NewHybridCache[int](cachex.HybridCacheConfig[int]{
			Namespace: cachex.Namespace(tfRouteErrorNamespace),
			Redis:     common.RDB,
			RedisEnabled: func() bool {
				return common.RedisEnabled && common.RDB != nil
			},
			RedisCodec: cachex.IntCodec{},
			Memory: func() *hot.HotCache[string, int] {
				return hot.NewHotCache[string, int](hot.LRU, 100_000).
					WithTTL(ttl).
					WithJanitor().
					Build()
			},
		})
	})
	return tfRouteErrorCache
}

func tfStickyKey(groupKey, group string) string {
	return groupKey + "#" + group
}

func tfErrorKey(stickyKey string, channelID int) string {
	return fmt.Sprintf("%s#%d", stickyKey, channelID)
}

func tfStickyGet(stickyKey string) (int, bool) {
	v, found, err := getTFRouteStickyCache().Get(stickyKey)
	if err != nil || !found || v <= 0 {
		return 0, false
	}
	return v, true
}

func tfStickySet(stickyKey string, channelID int) {
	ttl := time.Duration(common.TokenFactoryRouteStickyTTLSeconds()) * time.Second
	_ = getTFRouteStickyCache().SetWithTTL(stickyKey, channelID, ttl)
}

func tfStickyDel(stickyKey string) {
	_, _ = getTFRouteStickyCache().DeleteMany([]string{stickyKey})
}

func tfErrorCount(stickyKey string, channelID int) int {
	v, found, err := getTFRouteErrorCache().Get(tfErrorKey(stickyKey, channelID))
	if err != nil || !found {
		return 0
	}
	return v
}

func tfErrorReset(stickyKey string, channelID int) {
	_, _ = getTFRouteErrorCache().DeleteMany([]string{tfErrorKey(stickyKey, channelID)})
}

// tfErrorIncr 自增错误计数并返回新值（best-effort，按 key 分片加锁降低竞态）。
func tfErrorIncr(stickyKey string, channelID int) int {
	key := tfErrorKey(stickyKey, channelID)
	lock := tfRouteErrorLock(key)
	lock.Lock()
	defer lock.Unlock()

	cur := tfErrorCount(stickyKey, channelID)
	cur++
	ttl := time.Duration(common.TokenFactoryRouteErrorTTLSeconds()) * time.Second
	_ = getTFRouteErrorCache().SetWithTTL(key, cur, ttl)
	return cur
}

func tfRouteErrorLock(key string) *sync.Mutex {
	var h uint32 = 2166136261
	for i := 0; i < len(key); i++ {
		h ^= uint32(key[i])
		h *= 16777619
	}
	return &tfRouteErrorLocks[h%uint32(len(tfRouteErrorLocks))]
}

// TFRoutePickChannel 在 TokenFactory 返回的有序候选上应用「黏性 + 熔断」选择：
//  1. 若该维度已有黏性渠道且仍在候选内、启用、未达报错阈值，则复用。
//  2. 否则取首个「启用且未达阈值」的候选，写入黏性。
//  3. 若全部达阈值，则取首个启用候选并清零其计数（保证可用，给予新机会）。
//
// isEnabled 由调用方提供（校验渠道是否仍启用）。同时把选择写入 gin 上下文供成功/失败反馈使用。
func TFRoutePickChannel(c *gin.Context, groupKey, group string, orderedIDs []int, isEnabled func(int) bool) (int, bool) {
	if len(orderedIDs) == 0 {
		return 0, false
	}
	stickyKey := tfStickyKey(groupKey, group)
	threshold := common.TokenFactoryRouteErrorThreshold()

	inOrder := func(id int) bool {
		for _, x := range orderedIDs {
			if x == id {
				return true
			}
		}
		return false
	}

	// 1) 黏性命中
	if sticky, ok := tfStickyGet(stickyKey); ok {
		if inOrder(sticky) && isEnabled(sticky) && tfErrorCount(stickyKey, sticky) < threshold {
			tfSetStickyContext(c, stickyKey, sticky)
			return sticky, true
		}
		tfStickyDel(stickyKey)
	}

	// 2) 首个「启用且未达阈值」候选
	for _, id := range orderedIDs {
		if !isEnabled(id) {
			continue
		}
		if tfErrorCount(stickyKey, id) >= threshold {
			continue
		}
		tfStickySet(stickyKey, id)
		tfSetStickyContext(c, stickyKey, id)
		return id, true
	}

	// 3) 全部达阈值：取首个启用候选并清零（避免无渠道可用）
	for _, id := range orderedIDs {
		if !isEnabled(id) {
			continue
		}
		tfErrorReset(stickyKey, id)
		tfStickySet(stickyKey, id)
		tfSetStickyContext(c, stickyKey, id)
		return id, true
	}
	return 0, false
}

func tfSetStickyContext(c *gin.Context, stickyKey string, channelID int) {
	if c == nil {
		return
	}
	c.Set(ginKeyTFRouteStickyKey, stickyKey)
	c.Set(ginKeyTFRouteChannelID, channelID)
}

func tfGetStickyContext(c *gin.Context) (string, int, bool) {
	if c == nil {
		return "", 0, false
	}
	keyAny, ok := c.Get(ginKeyTFRouteStickyKey)
	if !ok {
		return "", 0, false
	}
	key, _ := keyAny.(string)
	if key == "" {
		return "", 0, false
	}
	idAny, ok := c.Get(ginKeyTFRouteChannelID)
	if !ok {
		return "", 0, false
	}
	id, _ := idAny.(int)
	if id <= 0 {
		return "", 0, false
	}
	return key, id, true
}

// RecordTFRouteResult 反馈某次中继结果到黏性熔断器（仅对 TF 路由选中的渠道生效）：
//   - 成功：清零计数并刷新黏性 TTL。
//   - 失败：自增计数；达阈值则摘除黏性绑定，下次请求顺延到下一候选。
//
// 只统计 TF 选中的那个渠道（重试时换的其它渠道不计入），从而实现「连续」语义。
func RecordTFRouteResult(c *gin.Context, channelID int, success bool) {
	stickyKey, chosenID, ok := tfGetStickyContext(c)
	if !ok || channelID <= 0 || chosenID != channelID {
		return
	}
	if success {
		tfErrorReset(stickyKey, channelID)
		tfStickySet(stickyKey, channelID)
		return
	}
	if cnt := tfErrorIncr(stickyKey, channelID); cnt >= common.TokenFactoryRouteErrorThreshold() {
		tfStickyDel(stickyKey)
	}
}
