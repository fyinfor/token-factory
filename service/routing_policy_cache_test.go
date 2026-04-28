// 路由策略缓存层单测：覆盖
//
//   - 命中：第二次调用走缓存、不再触发 loader；
//   - 过期：TTL 走完后重新打 loader；
//   - invalidate：写后再读必定 miss；
//   - 空结果（policy=nil）也缓存——防穿透；
//   - TTL=0 的 bypass 路径：每次都打 loader；
//   - 错误传播 + 不污染缓存；
//   - 并发场景下 singleflight 把同一 user 的并发 miss 合并到「只有一次」loader 调用。
package service

import (
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
)

// withLoader 把 routingPolicyLoader 临时替换成 mock，并在 t.Cleanup 时还原；同时清空
// 当前缓存与 stats，让每条 case 之间独立。返回 *atomic.Int64 让测试用例统计 loader 次数。
func withLoader(t *testing.T, loader func(userID int) (*model.RoutingPolicy, error)) *atomic.Int64 {
	t.Helper()
	calls := &atomic.Int64{}
	prev := routingPolicyLoader
	routingPolicyLoader = func(userID int) (*model.RoutingPolicy, error) {
		calls.Add(1)
		return loader(userID)
	}
	ClearAllRoutingPolicyCache()
	resetCacheStats()
	t.Cleanup(func() {
		routingPolicyLoader = prev
		ClearAllRoutingPolicyCache()
		resetCacheStats()
	})
	return calls
}

// resetCacheStats 不通过暴露 API 重置（生产 API 不应允许重置），直接 atomic.Store。
func resetCacheStats() {
	routingPolicyHits.Store(0)
	routingPolicyMisses.Store(0)
}

// withTTL 临时覆盖 TTL 环境变量；t.Cleanup 还原原值。
func withTTL(t *testing.T, ttlSeconds string) {
	t.Helper()
	old, hadOld := os.LookupEnv(routingPolicyCacheTTLEnv)
	if err := os.Setenv(routingPolicyCacheTTLEnv, ttlSeconds); err != nil {
		t.Fatalf("setenv: %v", err)
	}
	t.Cleanup(func() {
		if hadOld {
			_ = os.Setenv(routingPolicyCacheTTLEnv, old)
		} else {
			_ = os.Unsetenv(routingPolicyCacheTTLEnv)
		}
	})
}

func TestLoadDefaultRoutingPolicyCached_HitAndMiss(t *testing.T) {
	withTTL(t, "30")
	calls := withLoader(t, func(userID int) (*model.RoutingPolicy, error) {
		return &model.RoutingPolicy{ID: 99, UserID: userID, Strategy: "price"}, nil
	})

	p1, err := LoadDefaultRoutingPolicyCached(7)
	if err != nil {
		t.Fatalf("first load: %v", err)
	}
	if p1 == nil || p1.ID != 99 {
		t.Fatalf("unexpected first policy: %+v", p1)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("loader calls after first load: want 1 got %d", got)
	}

	p2, err := LoadDefaultRoutingPolicyCached(7)
	if err != nil {
		t.Fatalf("second load: %v", err)
	}
	if p2 != p1 {
		t.Fatalf("second load should return same cached pointer, got %p vs %p", p2, p1)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("loader calls after cached load: want 1 got %d", got)
	}
	hits, misses := RoutingPolicyCacheStats()
	if hits != 1 || misses != 1 {
		t.Fatalf("stats want hits=1 misses=1, got hits=%d misses=%d", hits, misses)
	}
}

func TestLoadDefaultRoutingPolicyCached_NilPolicyAlsoCached(t *testing.T) {
	withTTL(t, "30")
	calls := withLoader(t, func(userID int) (*model.RoutingPolicy, error) {
		return nil, nil
	})

	for i := 0; i < 5; i++ {
		p, err := LoadDefaultRoutingPolicyCached(11)
		if err != nil {
			t.Fatalf("load #%d: %v", i, err)
		}
		if p != nil {
			t.Fatalf("expected nil policy on iteration %d, got %+v", i, p)
		}
	}
	// 缓存命中 4 次，loader 只该被打 1 次：典型「防穿透」校验。
	if got := calls.Load(); got != 1 {
		t.Fatalf("loader calls for nil-cached user: want 1 got %d", got)
	}
}

func TestLoadDefaultRoutingPolicyCached_Expiration(t *testing.T) {
	// TTL=0 在我们的实现里是「禁用缓存」，不能用来测过期。
	// 这里 TTL=1 秒，等 time.Sleep 让条目过期后再触发回源。
	withTTL(t, "1")
	calls := withLoader(t, func(userID int) (*model.RoutingPolicy, error) {
		return &model.RoutingPolicy{ID: int64(userID * 10), UserID: userID}, nil
	})

	if _, err := LoadDefaultRoutingPolicyCached(3); err != nil {
		t.Fatalf("first load: %v", err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("after first load: want 1 got %d", got)
	}
	// 略大于 TTL 让 expireUnix 落在 now 之前；TTL 单位是秒，sleep 1100ms 已足够。
	time.Sleep(1100 * time.Millisecond)
	if _, err := LoadDefaultRoutingPolicyCached(3); err != nil {
		t.Fatalf("after expire: %v", err)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("after expire reload: want 2 got %d", got)
	}
}

func TestInvalidateRoutingPolicyCache(t *testing.T) {
	withTTL(t, "30")
	calls := withLoader(t, func(userID int) (*model.RoutingPolicy, error) {
		return &model.RoutingPolicy{ID: 1, UserID: userID}, nil
	})

	if _, err := LoadDefaultRoutingPolicyCached(8); err != nil {
		t.Fatalf("load: %v", err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("after first load: want 1 got %d", got)
	}
	InvalidateRoutingPolicyCache(8)
	// invalidate 之后再读必定走 loader——这是写后立即可见的关键保证。
	if _, err := LoadDefaultRoutingPolicyCached(8); err != nil {
		t.Fatalf("load after invalidate: %v", err)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("after invalidate reload: want 2 got %d", got)
	}
}

func TestLoadDefaultRoutingPolicyCached_TTLZeroBypass(t *testing.T) {
	withTTL(t, "0")
	calls := withLoader(t, func(userID int) (*model.RoutingPolicy, error) {
		return &model.RoutingPolicy{ID: 42, UserID: userID}, nil
	})

	for i := 0; i < 3; i++ {
		if _, err := LoadDefaultRoutingPolicyCached(5); err != nil {
			t.Fatalf("load #%d: %v", i, err)
		}
	}
	// TTL=0 等价于禁用缓存：每次都该打 loader；命中数应为 0。
	if got := calls.Load(); got != 3 {
		t.Fatalf("ttl=0 bypass: want 3 loader calls got %d", got)
	}
	hits, misses := RoutingPolicyCacheStats()
	if hits != 0 || misses != 3 {
		t.Fatalf("ttl=0 stats want hits=0 misses=3, got hits=%d misses=%d", hits, misses)
	}
}

func TestLoadDefaultRoutingPolicyCached_ErrorNotCached(t *testing.T) {
	withTTL(t, "30")
	dbErr := errors.New("transient db error")
	var attempt atomic.Int64
	withLoader(t, func(userID int) (*model.RoutingPolicy, error) {
		// 第一次返回错误；第二次返回成功——验证错误期间没有把任何东西塞进缓存。
		if attempt.Add(1) == 1 {
			return nil, dbErr
		}
		return &model.RoutingPolicy{ID: 7}, nil
	})

	if _, err := LoadDefaultRoutingPolicyCached(20); !errors.Is(err, dbErr) {
		t.Fatalf("first call should propagate db error, got %v", err)
	}
	// 错误时不写缓存：第二次仍是 miss，会再次打 loader（这里依赖 mock 第二次成功）。
	p, err := LoadDefaultRoutingPolicyCached(20)
	if err != nil {
		t.Fatalf("second call after recovery: %v", err)
	}
	if p == nil || p.ID != 7 {
		t.Fatalf("unexpected policy after recovery: %+v", p)
	}
}

func TestLoadDefaultRoutingPolicyCached_NonPositiveUser(t *testing.T) {
	withTTL(t, "30")
	calls := withLoader(t, func(userID int) (*model.RoutingPolicy, error) {
		return &model.RoutingPolicy{ID: 1}, nil
	})

	for _, uid := range []int{0, -1, -42} {
		p, err := LoadDefaultRoutingPolicyCached(uid)
		if err != nil {
			t.Fatalf("uid=%d unexpected err: %v", uid, err)
		}
		if p != nil {
			t.Fatalf("uid=%d should return nil policy without consulting loader", uid)
		}
	}
	if got := calls.Load(); got != 0 {
		t.Fatalf("non-positive user shouldn't touch loader, but got %d calls", got)
	}
}

func TestLoadDefaultRoutingPolicyCached_SingleflightDedupesConcurrentMiss(t *testing.T) {
	withTTL(t, "30")
	// loader 故意慢一点，让并发的 N-1 个 goroutine 都进 singleflight wait。
	calls := withLoader(t, func(userID int) (*model.RoutingPolicy, error) {
		time.Sleep(50 * time.Millisecond)
		return &model.RoutingPolicy{ID: int64(userID), UserID: userID}, nil
	})

	const concurrency = 50
	var wg sync.WaitGroup
	wg.Add(concurrency)
	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			if _, err := LoadDefaultRoutingPolicyCached(101); err != nil {
				t.Errorf("concurrent load: %v", err)
			}
		}()
	}
	wg.Wait()
	// singleflight 把 50 个并发 miss 合并：loader 至多被打 1 次，而绝不可能 50 次。
	if got := calls.Load(); got != 1 {
		t.Fatalf("singleflight should dedupe concurrent misses to 1, got %d", got)
	}
}

func TestRoutingPolicyCacheTTL_FallbackOnInvalidEnv(t *testing.T) {
	cases := []struct {
		name string
		val  string
		want time.Duration
	}{
		{"empty falls back to default", "", defaultRoutingPolicyCacheTTLSeconds * time.Second},
		{"non-numeric falls back to default", "abc", defaultRoutingPolicyCacheTTLSeconds * time.Second},
		{"negative falls back to default", "-5", defaultRoutingPolicyCacheTTLSeconds * time.Second},
		{"zero disables", "0", 0},
		{"explicit value honored", "12", 12 * time.Second},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			withTTL(t, tc.val)
			if got := routingPolicyCacheTTL(); got != tc.want {
				t.Fatalf("ttl env=%q: want %v got %v", tc.val, tc.want, got)
			}
		})
	}
}
