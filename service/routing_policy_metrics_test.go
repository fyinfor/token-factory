// 路由策略 metrics 单测：
//
//   - source 分桶累计：每个 source 独立、互不影响；
//   - 未预分配 source 也能动态创建并计数（防回归：未来 resolver 加新 source 时不需要改 metrics）；
//   - fallback 累计；
//   - Snapshot 是值快照（mutation 不会污染已返回的 map）；
//   - Reset 把 resolve / fallback 计数清零，但 cache 命中数与 UptimeSeconds 不动；
//   - 并发场景下 atomic 计数无竞态。
package service

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// resetMetricsForTest 让每条 case 独立。直接调 ResetRoutingPolicyMetrics 即可——它已经
// 把 resolveCounts 与 fallbackUsed 清零；同时为了避免互相干扰，把 cache stats 也清零。
func resetMetricsForTest(t *testing.T) {
	t.Helper()
	ResetRoutingPolicyMetrics()
	resetCacheStats()
}

func TestRecordResolveOutcome_BumpsBucket(t *testing.T) {
	resetMetricsForTest(t)

	RecordResolveOutcome(ResolveSourceUserDefault)
	RecordResolveOutcome(ResolveSourceUserDefault)
	RecordResolveOutcome(ResolveSourceMerged)

	snap := GetRoutingPolicyMetricsSnapshot()
	if got := snap.ResolveCounts[ResolveSourceUserDefault]; got != 2 {
		t.Fatalf("user_default count: want 2 got %d", got)
	}
	if got := snap.ResolveCounts[ResolveSourceMerged]; got != 1 {
		t.Fatalf("merged count: want 1 got %d", got)
	}
	if got := snap.ResolveCounts[ResolveSourceNone]; got != 0 {
		t.Fatalf("none count must remain 0, got %d", got)
	}
}

func TestRecordResolveOutcome_EmptySourceFallsBackToNone(t *testing.T) {
	resetMetricsForTest(t)

	RecordResolveOutcome("")

	if got := GetRoutingPolicyMetricsSnapshot().ResolveCounts[ResolveSourceNone]; got != 1 {
		t.Fatalf("empty source should bucket as 'none', got %d", got)
	}
}

func TestRecordResolveOutcome_DynamicSourceBucket(t *testing.T) {
	resetMetricsForTest(t)
	const exotic = "future_source_X"

	for i := 0; i < 5; i++ {
		RecordResolveOutcome(exotic)
	}

	snap := GetRoutingPolicyMetricsSnapshot()
	if got := snap.ResolveCounts[exotic]; got != 5 {
		t.Fatalf("exotic source count: want 5 got %d", got)
	}
	// 现有预分配 source 不应被污染。
	if got := snap.ResolveCounts[ResolveSourceUserDefault]; got != 0 {
		t.Fatalf("user_default polluted: want 0 got %d", got)
	}
}

func TestRecordRoutingPolicyResolveError_RoutesToErrorBucket(t *testing.T) {
	resetMetricsForTest(t)

	RecordRoutingPolicyResolveError()
	RecordRoutingPolicyResolveError()

	snap := GetRoutingPolicyMetricsSnapshot()
	if got := snap.ResolveCounts[ResolveSourceError]; got != 2 {
		t.Fatalf("error bucket: want 2 got %d", got)
	}
}

func TestRecordRoutingPolicyFallback(t *testing.T) {
	resetMetricsForTest(t)

	RecordRoutingPolicyFallback()
	RecordRoutingPolicyFallback()
	RecordRoutingPolicyFallback()

	if got := GetRoutingPolicyMetricsSnapshot().FallbackUsed; got != 3 {
		t.Fatalf("fallback_used count: want 3 got %d", got)
	}
}

func TestSnapshot_IsCopy(t *testing.T) {
	resetMetricsForTest(t)

	RecordResolveOutcome(ResolveSourceUserDefault)
	snap := GetRoutingPolicyMetricsSnapshot()
	// 拿到 snapshot 之后再产生计数：snapshot 应保持原值（typed by-value semantics）。
	RecordResolveOutcome(ResolveSourceUserDefault)
	RecordResolveOutcome(ResolveSourceUserDefault)

	if got := snap.ResolveCounts[ResolveSourceUserDefault]; got != 1 {
		t.Fatalf("snapshot should be a copy, got mutated value %d", got)
	}
	// 二次 snapshot 才能看到累计后的值。
	if got := GetRoutingPolicyMetricsSnapshot().ResolveCounts[ResolveSourceUserDefault]; got != 3 {
		t.Fatalf("fresh snapshot count: want 3 got %d", got)
	}
}

func TestReset_ClearsCountersButNotUptime(t *testing.T) {
	resetMetricsForTest(t)

	RecordResolveOutcome(ResolveSourceUserDefault)
	RecordRoutingPolicyFallback()
	// 故意伪造一些 cache stats，验证 Reset 不动它们。
	routingPolicyHits.Store(42)
	routingPolicyMisses.Store(7)

	preReset := GetRoutingPolicyMetricsSnapshot()
	if preReset.ResolveCounts[ResolveSourceUserDefault] != 1 || preReset.FallbackUsed != 1 {
		t.Fatalf("pre-reset baseline wrong: %+v", preReset)
	}

	ResetRoutingPolicyMetrics()
	post := GetRoutingPolicyMetricsSnapshot()

	if post.ResolveCounts[ResolveSourceUserDefault] != 0 {
		t.Fatalf("resolve count not reset: %d", post.ResolveCounts[ResolveSourceUserDefault])
	}
	if post.FallbackUsed != 0 {
		t.Fatalf("fallback_used not reset: %d", post.FallbackUsed)
	}
	if post.CacheHits != 42 || post.CacheMisses != 7 {
		t.Fatalf("cache stats should not be reset by metrics reset: hits=%d misses=%d", post.CacheHits, post.CacheMisses)
	}
	// UptimeSeconds 单调递增；跨 Reset 不归零。这里只断言 >= 0（不依赖 wall-clock 精度）。
	if post.UptimeSeconds < 0 {
		t.Fatalf("uptime negative: %d", post.UptimeSeconds)
	}
	// SinceResetSeconds 在 Reset 后 <= 1（取决于调度抖动）。
	if post.SinceResetSeconds > 2 {
		t.Fatalf("since_reset_seconds should be near 0 right after reset, got %d", post.SinceResetSeconds)
	}
}

func TestSnapshot_IncludesCacheStats(t *testing.T) {
	resetMetricsForTest(t)
	routingPolicyHits.Store(13)
	routingPolicyMisses.Store(4)

	snap := GetRoutingPolicyMetricsSnapshot()
	if snap.CacheHits != 13 || snap.CacheMisses != 4 {
		t.Fatalf("cache stats not surfaced: hits=%d misses=%d", snap.CacheHits, snap.CacheMisses)
	}
}

func TestRecordResolveOutcome_ConcurrentAtomic(t *testing.T) {
	resetMetricsForTest(t)

	const goroutines = 50
	const perGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < perGoroutine; i++ {
				RecordResolveOutcome(ResolveSourceMerged)
				RecordResolveOutcome("dynamic_concurrent_source")
			}
		}()
	}
	wg.Wait()

	want := uint64(goroutines * perGoroutine)
	snap := GetRoutingPolicyMetricsSnapshot()
	if got := snap.ResolveCounts[ResolveSourceMerged]; got != want {
		t.Fatalf("merged count race: want %d got %d", want, got)
	}
	if got := snap.ResolveCounts["dynamic_concurrent_source"]; got != want {
		t.Fatalf("dynamic source race: want %d got %d", want, got)
	}
}

func TestRecordRoutingPolicyFallback_ConcurrentAtomic(t *testing.T) {
	resetMetricsForTest(t)

	const goroutines = 64
	var counter atomic.Uint64
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			RecordRoutingPolicyFallback()
			counter.Add(1)
		}()
	}
	wg.Wait()

	if counter.Load() != goroutines {
		t.Fatalf("test sentinel race: %d", counter.Load())
	}
	if got := GetRoutingPolicyMetricsSnapshot().FallbackUsed; got != uint64(goroutines) {
		t.Fatalf("fallback race: want %d got %d", goroutines, got)
	}
}

// 防止冷门字段（UptimeSeconds 单调递增）回归：让小 sleep 后 snap2.Uptime > snap1.Uptime。
func TestSnapshot_UptimeMonotonic(t *testing.T) {
	resetMetricsForTest(t)
	snap1 := GetRoutingPolicyMetricsSnapshot()
	time.Sleep(1100 * time.Millisecond)
	snap2 := GetRoutingPolicyMetricsSnapshot()
	if snap2.UptimeSeconds < snap1.UptimeSeconds {
		t.Fatalf("uptime regressed: %d -> %d", snap1.UptimeSeconds, snap2.UptimeSeconds)
	}
}
