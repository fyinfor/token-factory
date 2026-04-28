// Package service: routing_policy_metrics 给「路由策略」生命周期建一组进程内 atomic 计数器，
// 让运维 / 后续接入的 Prometheus exporter 可以拿到「策略解析的总量与分布、fallback 命中率、
// 缓存命中率」这些核心 KPI。
//
// 设计取舍：
//
//   - 不依赖 Prometheus 客户端（项目当前没引）：先用 atomic 自管，给一个 Snapshot 接口做
//     pull-style 输出；外部接 Prometheus 时再写一层 collector 即可（解耦）。
//
//   - 维度按「source」分桶，与 ResolvedRoutingPolicy.Source 保持一致；新增 ResolveSourceError
//     专门记录解析失败次数。这样 source 维度 + fallback 维度 + cache 维度三合一就够看主场景。
//
//   - 不收集 (user_id, model) 这种高基数标签——单进程内放成 map 会内存爆掉；这种细粒度
//     由请求级日志 /  consume_log 表去回放，不进 metrics。
//
//   - Reset 不动 startTime：Snapshot.UptimeSeconds 是从进程启动算起，「重置 metrics 不等于
//     重启进程」语义更直观，避免运营把 reset 后的速率算成 24h 内但实际上是 5min 内。
//
// 与 cache 模块的关系：
//
//   - Cache 自身有命中/miss 计数（routing_policy_cache.go），这里通过 RoutingPolicyCacheStats()
//     被合并进同一个 Snapshot。Snapshot 是 metrics 唯一的「对外」聚合点。
package service

import (
	"sync"
	"sync/atomic"
	"time"
)

// ResolveSourceError 仅在 metrics 维度使用，标识 ResolveRoutingPolicy 抛错的次数。
// 不放在 routing_policy_resolver.go 是为了避免把「监控用枚举」混进解析合约里。
const ResolveSourceError = "error"

// routingPolicyMetrics 集中维护所有计数器。
//
// 字段为指针/map：用 sync.Map 处理 source 维度的字符串 -> *atomic.Uint64 映射，避免
// 在不知道 source 字符串集合时漏字段（外部工具未来可能塞新 source）。预初始化常用 source
// 让 Snapshot 输出稳定不抖动。
type routingPolicyMetrics struct {
	resolveCountsMu sync.RWMutex
	// resolveCounts 按 source 字符串分桶，*atomic.Uint64 用指针是因 sync.Map.Load 返回
	// any 接口，无法直接 atomic 操作；用指针 + Add 才线程安全。
	resolveCounts map[string]*atomic.Uint64

	fallbackUsed atomic.Uint64
	startUnix    int64
}

// newRoutingPolicyMetrics 预创建固定 source 桶；这样即使未发生过的 source 也会出现在
// Snapshot 里值为 0，前端展示时不会因「字段缺失」而报错。
func newRoutingPolicyMetrics() *routingPolicyMetrics {
	m := &routingPolicyMetrics{
		resolveCounts: make(map[string]*atomic.Uint64, 8),
		startUnix:     time.Now().Unix(),
	}
	for _, src := range []string{
		ResolveSourceNone,
		ResolveSourceRequestOnly,
		ResolveSourceUserDefault,
		ResolveSourceMerged,
		"external", // SimulateResolveRoutingPolicy / resolveWithoutPolicy 走这条路径
		ResolveSourceError,
	} {
		v := &atomic.Uint64{}
		m.resolveCounts[src] = v
	}
	return m
}

var routingPolicyMetricsState = newRoutingPolicyMetrics()

// RecordResolveOutcome 在 distributor 解析完一次 ResolvedRoutingPolicy 后调一次。
//
// 接受任意 source 字符串：未预分配的桶会按需创建，保证「未来在 resolver 里加新 source 时
// metrics 自动跟上」。空字符串视为 ResolveSourceNone（防御兜底）。
func RecordResolveOutcome(source string) {
	if source == "" {
		source = ResolveSourceNone
	}
	routingPolicyMetricsState.resolveCountsMu.RLock()
	bucket, ok := routingPolicyMetricsState.resolveCounts[source]
	routingPolicyMetricsState.resolveCountsMu.RUnlock()
	if ok {
		bucket.Add(1)
		return
	}
	// 写锁路径：双检，避免并发首次写入 source 时漏计或重复创建。
	routingPolicyMetricsState.resolveCountsMu.Lock()
	defer routingPolicyMetricsState.resolveCountsMu.Unlock()
	if bucket, ok = routingPolicyMetricsState.resolveCounts[source]; !ok {
		bucket = &atomic.Uint64{}
		routingPolicyMetricsState.resolveCounts[source] = bucket
	}
	bucket.Add(1)
}

// RecordRoutingPolicyResolveError 是 RecordResolveOutcome(ResolveSourceError) 的语法糖；
// 让调用方意图一目了然。
func RecordRoutingPolicyResolveError() {
	RecordResolveOutcome(ResolveSourceError)
}

// RecordRoutingPolicyFallback 在 distributor 命中 fallback 后调一次（含两种 fallback：
// 候选池失败时按 FallbackStrategy 重新选 + 候选池失败时直接 CacheGetRandomSatisfiedChannel 兜底）。
//
// 不细分 fallback 类型——上层日志里有 source 字段，运营在做事故复盘时按时间窗去捞日志即可；
// 这里只关心「整体 fallback 命中率」这个 KPI。
func RecordRoutingPolicyFallback() {
	routingPolicyMetricsState.fallbackUsed.Add(1)
}

// RoutingPolicyMetricsSnapshot 是「pull-style」输出结构；JSON 字段名固定，便于运维 / 前端
// 看板直接消费，不会因为内部字段重构而破坏对外契约。
//
// 字段含义：
//   - ResolveCounts: 各 source 累计次数；包括预初始化的 6 个 + 任何运行期新出现的 source。
//   - FallbackUsed: fallback 命中累计；除以 ResolveCounts 总和约等于「兜底命中率」。
//   - CacheHits / CacheMisses: 来自 routing_policy_cache.go 的累计统计。
//   - UptimeSeconds: 进程启动到 Snapshot 时刻的秒数；不受 ResetRoutingPolicyMetrics 影响。
//   - SinceResetSeconds: 距上次 Reset 的秒数；Reset 后立即变成 0。运维做 A/B 时用得上。
type RoutingPolicyMetricsSnapshot struct {
	ResolveCounts     map[string]uint64 `json:"resolve_counts"`
	FallbackUsed      uint64            `json:"fallback_used"`
	CacheHits         uint64            `json:"cache_hits"`
	CacheMisses       uint64            `json:"cache_misses"`
	UptimeSeconds     int64             `json:"uptime_seconds"`
	SinceResetSeconds int64             `json:"since_reset_seconds"`
}

// 进程启动时间：Reset 不动它，让 UptimeSeconds 始终对齐进程生命周期。
var routingPolicyMetricsProcessStartUnix = time.Now().Unix()

// GetRoutingPolicyMetricsSnapshot 抓一份当前指标的快照，原子复制；后续 mutation 不会污染返回值。
//
// O(N) 拷贝 source map（N 通常 < 10），可接受。若未来 source 数量爆炸再考虑分片。
func GetRoutingPolicyMetricsSnapshot() RoutingPolicyMetricsSnapshot {
	now := time.Now().Unix()
	hits, misses := RoutingPolicyCacheStats()

	routingPolicyMetricsState.resolveCountsMu.RLock()
	out := RoutingPolicyMetricsSnapshot{
		ResolveCounts:     make(map[string]uint64, len(routingPolicyMetricsState.resolveCounts)),
		FallbackUsed:      routingPolicyMetricsState.fallbackUsed.Load(),
		CacheHits:         hits,
		CacheMisses:       misses,
		UptimeSeconds:     now - routingPolicyMetricsProcessStartUnix,
		SinceResetSeconds: now - routingPolicyMetricsState.startUnix,
	}
	for src, v := range routingPolicyMetricsState.resolveCounts {
		out.ResolveCounts[src] = v.Load()
	}
	routingPolicyMetricsState.resolveCountsMu.RUnlock()
	return out
}

// ResetRoutingPolicyMetrics 把所有运行期累计计数清零；通常运维在做策略 A/B 或事故后清盘
// 时调用。startUnix 重置成 now，后续 SinceResetSeconds 重新从 0 开始；UptimeSeconds 不动。
//
// 不重置 cache 的 hits / misses（这是 cache 层的 KPI，需要时单独清；混在一起重置易误用）。
func ResetRoutingPolicyMetrics() {
	routingPolicyMetricsState.resolveCountsMu.Lock()
	defer routingPolicyMetricsState.resolveCountsMu.Unlock()
	for _, v := range routingPolicyMetricsState.resolveCounts {
		v.Store(0)
	}
	routingPolicyMetricsState.fallbackUsed.Store(0)
	routingPolicyMetricsState.startUnix = time.Now().Unix()
}
