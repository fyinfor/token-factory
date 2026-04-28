// Package service: routing_policy_cache 给「用户默认路由策略」加一层进程内 TTL 缓存。
//
// 背景：每条聊天/补全请求都会在 distributor 里调一次 ResolveRoutingPolicy，进而
// model.GetDefaultRoutingPolicyByUser → DB 查 1 主表行 + 1 候选池行（LIMIT 数十）。
// 高 QPS 下这就是一次稳定的 N+M 次 DB hit；而「默认策略」实际上几乎不变，更新频率
// 远低于读取频率，是经典的「读多写少」场景，正适合 TTL 缓存。
//
// 设计要点：
//
//   - 缓存粒度 userID：每个用户一条；命中是 *model.RoutingPolicy（含 Targets 已展开）。
//
//   - TTL 由 ROUTING_POLICY_CACHE_TTL_SECONDS 环境变量控制，默认 30s。
//     设为 0 时整个缓存层降级为透传，便于运维一键关闭（排错 / 灰度时使用）。
//
//   - 写操作触发 invalidate：controller 在 Create/Update/Delete/SetDefault/ClearDefault
//     之后立即 InvalidateRoutingPolicyCache(userID)，使 30s TTL 不会让用户改完策略
//     还要等半分钟才生效——配合 TTL 兜底「极端情况下也最多 30s 不一致」。
//
//   - 缓存 nil（即「该用户无默认策略」）也写入：避免没有策略的用户每个请求都打 DB
//     做无意义查询；这是缓存穿透防御中的「空结果保护」。
//
//   - 同 user 并发 miss 用 singleflight 合并：在「默认策略 invalidate 后被几百个并发
//     请求同时打到」的场景，singleflight 让其中只有一个 goroutine 真正打 DB，其余等
//     这一次结果。否则缓存击穿会瞬间放大到 DB（典型缓存雪崩问题）。
//
// 失效语义：
//
//   - 任何一条策略写改都直接清掉「该 user 的整条缓存」（哪怕改的不是默认策略），
//     因为 IsDefault 互斥规则下「改了非默认策略」也可能重新触发默认（例如调用方先
//     把另一条设为默认）；保险起见全清最简单且无副作用。
package service

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/model"

	"golang.org/x/sync/singleflight"
)

// 默认 TTL 30s——比 OAuth/SSO token 的常见短缓存稍长，避免「写后立刻读」时哪怕 invalidate
// 漏调用也最多 30s 不一致；同时短到运营调整策略后用户感知足够及时。
const (
	defaultRoutingPolicyCacheTTLSeconds = 30
	routingPolicyCacheTTLEnv            = "ROUTING_POLICY_CACHE_TTL_SECONDS"
)

// cachedRoutingPolicy 是缓存条目；policy=nil 表示「已查过，该用户没有默认策略」。
//
// expireUnix 用 unix 秒，比 time.Time 更轻；Now().Unix() 一次比较即可，省去时区/位置。
type cachedRoutingPolicy struct {
	policy     *model.RoutingPolicy
	expireUnix int64
}

var (
	// routingPolicyCache: userID -> *cachedRoutingPolicy。sync.Map 适合「写少读多 + key
	// 集合稳定」场景，这里 user 数有限且大多数 hit；不需要分片锁。
	routingPolicyCache sync.Map
	// routingPolicyGroup: 同 userID 的并发 miss 合并打 DB，缓解雪崩。
	routingPolicyGroup singleflight.Group
	// 命中 / miss 计数；运维可通过 GetRoutingPolicyCacheStats 查命中率。
	routingPolicyHits   atomic.Uint64
	routingPolicyMisses atomic.Uint64

	// routingPolicyLoader 是回源函数。包级变量便于单测注入 mock loader——避免缓存层
	// 单测耦合 DB；生产环境不应改写它。
	routingPolicyLoader = func(userID int) (*model.RoutingPolicy, error) {
		return model.GetDefaultRoutingPolicyByUser(userID)
	}
)

// routingPolicyCacheTTL 读取环境变量，0 表示禁用缓存（直接透传到 DB）。负值或非法
// 值时使用默认值，避免运维误配把生产打挂。
func routingPolicyCacheTTL() time.Duration {
	v := strings.TrimSpace(os.Getenv(routingPolicyCacheTTLEnv))
	if v == "" {
		return defaultRoutingPolicyCacheTTLSeconds * time.Second
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return defaultRoutingPolicyCacheTTLSeconds * time.Second
	}
	return time.Duration(n) * time.Second
}

// LoadDefaultRoutingPolicyCached 是 ResolveRoutingPolicy 唯一应该调用的「读」入口。
//
// 行为：
//   - userID<=0 直接返回 (nil, nil)，不进缓存（与 model 层零值语义一致）。
//   - TTL=0 时跳过缓存，直接透传到 model.GetDefaultRoutingPolicyByUser。
//   - 命中且未过期 → 直接返回缓存；不复制 policy 内部字段，调用方应视为只读。
//   - miss / 过期 → singleflight 合并对该 user 的并发请求，只让一个 goroutine 打 DB。
//
// 错误传播：
//   - DB 错误向上抛，不写缓存（避免把抖动期间的错误结果钉进缓存）；调用方应当继续走
//     现有「降级为无策略」的逻辑（distributor 已实现）。
func LoadDefaultRoutingPolicyCached(userID int) (*model.RoutingPolicy, error) {
	if userID <= 0 {
		return nil, nil
	}
	ttl := routingPolicyCacheTTL()
	if ttl <= 0 {
		// 显式禁用缓存：透传，但仍走 atomic 计数便于运维观测「禁用后读 DB 量」。
		routingPolicyMisses.Add(1)
		return routingPolicyLoader(userID)
	}
	now := time.Now().Unix()
	if v, ok := routingPolicyCache.Load(userID); ok {
		if entry, ok := v.(*cachedRoutingPolicy); ok && entry != nil && entry.expireUnix > now {
			routingPolicyHits.Add(1)
			return entry.policy, nil
		}
	}
	// miss / 过期：合并并发回源。singleflight key 用 user id 字符串，shared=true 时多个
	// 调用拿到同一 result，符合预期。
	routingPolicyMisses.Add(1)
	key := strconv.Itoa(userID)
	v, err, _ := routingPolicyGroup.Do(key, func() (any, error) {
		// 双检：拿到 lock 之后再看一次缓存；前一个 goroutine 可能已经写好了。
		if v, ok := routingPolicyCache.Load(userID); ok {
			if entry, ok := v.(*cachedRoutingPolicy); ok && entry != nil && entry.expireUnix > time.Now().Unix() {
				return entry.policy, nil
			}
		}
		policy, err := routingPolicyLoader(userID)
		if err != nil {
			// 不写缓存：让下一次请求重试 DB；此处不区分错误类型——任何错都不该污染缓存。
			return nil, err
		}
		routingPolicyCache.Store(userID, &cachedRoutingPolicy{
			policy:     policy,
			expireUnix: time.Now().Unix() + int64(ttl/time.Second),
		})
		return policy, nil
	})
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, nil
	}
	policy, ok := v.(*model.RoutingPolicy)
	if !ok {
		return nil, errors.New("routing_policy_cache: unexpected value type from singleflight")
	}
	return policy, nil
}

// InvalidateRoutingPolicyCache 在「该 user 的任何路由策略写操作」之后调一次。
//
// 不区分写的是哪条策略：因为 SetDefaultRoutingPolicy 互斥更新会把同 user 其它策略的
// is_default 改成 false（参考 model.SetDefaultRoutingPolicy 实现）；为了简单稳妥直接
// 全清该用户的整条缓存。
//
// 即便 TTL=0 也保留无害调用：让 controller 不必关心当前缓存状态。
func InvalidateRoutingPolicyCache(userID int) {
	if userID <= 0 {
		return
	}
	routingPolicyCache.Delete(userID)
}

// ClearAllRoutingPolicyCache 清空整张缓存；测试与运维（如修复 DB 数据后）使用。
//
// 不重置 hits/misses 计数；监控指标累计语义更直观。
func ClearAllRoutingPolicyCache() {
	routingPolicyCache.Range(func(k, _ any) bool {
		routingPolicyCache.Delete(k)
		return true
	})
}

// RoutingPolicyCacheStats 返回 (hits, misses)；调用方可计算命中率，无须并发保护，
// atomic.Uint64 自带读隔离。
func RoutingPolicyCacheStats() (hits uint64, misses uint64) {
	return routingPolicyHits.Load(), routingPolicyMisses.Load()
}
