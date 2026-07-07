package service

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/common/limiter"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/go-redis/redis/v8"
)

const channelModelRateLimitWindowSec int64 = 60

var (
	channelModelRateLimitTimeFormat = "2006-01-02T15:04:05.000Z"
	channelModelMemoryLimiter       common.InMemoryRateLimiter
	channelModelMemoryLimiterOnce   sync.Once
	channelModelTokenBuckets        sync.Map // key string -> *channelModelTokenBucket
)

type channelModelTokenBucket struct {
	tokens   float64
	capacity float64
	rate     float64 // 每分钟 RPM 折算为「每秒补充 token 数」
	last     time.Time
	mu       sync.Mutex
}

func MatchChannelModelRateLimit(channel *model.Channel, modelName string) *dto.ChannelModelRateLimitRule {
	if channel == nil {
		return nil
	}
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return nil
	}
	settings := channel.GetOtherSettings()
	for i := range settings.ModelRateLimits {
		rule := settings.ModelRateLimits[i]
		if !rule.IsEnabled() {
			continue
		}
		if strings.TrimSpace(rule.Model) == modelName {
			return &settings.ModelRateLimits[i]
		}
	}
	return nil
}

func SanitizeChannelModelRateLimits(rules []dto.ChannelModelRateLimitRule) []dto.ChannelModelRateLimitRule {
	if len(rules) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(rules))
	out := make([]dto.ChannelModelRateLimitRule, 0, len(rules))
	for _, rule := range rules {
		modelName := strings.TrimSpace(rule.Model)
		if modelName == "" || rule.RPM <= 0 {
			continue
		}
		enabled := true
		if rule.Enabled != nil {
			enabled = *rule.Enabled
		}
		if !enabled {
			continue
		}
		if _, ok := seen[modelName]; ok {
			continue
		}
		seen[modelName] = struct{}{}
		burst := rule.Burst
		if burst < 0 {
			burst = 0
		}
		out = append(out, dto.ChannelModelRateLimitRule{
			Model:   modelName,
			RPM:     rule.RPM,
			Burst:   burst,
			Enabled: &enabled,
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func TryAcquireChannelModelRateLimit(channelID int, modelName string, rule dto.ChannelModelRateLimitRule) (allowed bool, retryAfterSec int, err error) {
	if !rule.IsEnabled() {
		return true, 0, nil
	}
	key := fmt.Sprintf("rl:cm:%d:%s", channelID, strings.TrimSpace(modelName))
	if rule.Burst > 0 {
		if common.RedisEnabled && common.RDB != nil {
			return tryAcquireChannelModelRateLimitRedisTokenBucket(context.Background(), common.RDB, key, rule)
		}
		return tryAcquireChannelModelRateLimitMemoryTokenBucket(key, rule)
	}
	if common.RedisEnabled && common.RDB != nil {
		return tryAcquireChannelModelRateLimitRedisSlidingWindow(context.Background(), common.RDB, key, rule)
	}
	return tryAcquireChannelModelRateLimitMemorySlidingWindow(key, rule)
}

// 分钟级令牌桶：单次请求消耗 60 token，桶容量 (RPM+Burst)*60，按 RPM/分钟 匀速补充。
func channelModelTokenBucketConfig(rule dto.ChannelModelRateLimitRule) (capacity int64, rate int64, requested int64) {
	rpm := int64(rule.RPM)
	if rpm < 1 {
		rpm = 1
	}
	burst := int64(rule.Burst)
	if burst < 0 {
		burst = 0
	}
	return (rpm + burst) * channelModelRateLimitWindowSec, rpm, channelModelRateLimitWindowSec
}

func retryAfterForTokenBucket(rule dto.ChannelModelRateLimitRule) int {
	rpm := rule.RPM
	if rpm < 1 {
		rpm = 1
	}
	// 补满一次请求所需 token（60）大约需要 60/rpm 秒。
	secs := int(math.Ceil(float64(channelModelRateLimitWindowSec) / float64(rpm)))
	if secs < 1 {
		return 1
	}
	return secs
}

func tryAcquireChannelModelRateLimitRedisTokenBucket(ctx context.Context, rdb *redis.Client, key string, rule dto.ChannelModelRateLimitRule) (bool, int, error) {
	capacity, rate, requested := channelModelTokenBucketConfig(rule)
	tb := limiter.New(ctx, rdb)
	allowed, err := tb.Allow(
		ctx,
		key+":tb",
		limiter.WithCapacity(capacity),
		limiter.WithRate(rate),
		limiter.WithRequested(requested),
	)
	if err != nil {
		return false, 0, err
	}
	if !allowed {
		return false, retryAfterForTokenBucket(rule), nil
	}
	return true, 0, nil
}

func tryAcquireChannelModelRateLimitRedisSlidingWindow(ctx context.Context, rdb *redis.Client, key string, rule dto.ChannelModelRateLimitRule) (bool, int, error) {
	length, err := rdb.LLen(ctx, key).Result()
	if err != nil {
		return false, 0, err
	}
	if length < int64(rule.RPM) {
		now := time.Now().Format(channelModelRateLimitTimeFormat)
		pipe := rdb.TxPipeline()
		pipe.LPush(ctx, key, now)
		pipe.LTrim(ctx, key, 0, int64(rule.RPM-1))
		pipe.Expire(ctx, key, time.Duration(channelModelRateLimitWindowSec)*time.Second)
		if _, err := pipe.Exec(ctx); err != nil {
			return false, 0, err
		}
		return true, 0, nil
	}

	oldTimeStr, _ := rdb.LIndex(ctx, key, -1).Result()
	oldTime, err := time.Parse(channelModelRateLimitTimeFormat, oldTimeStr)
	if err != nil {
		return false, 0, err
	}
	elapsed := time.Since(oldTime).Seconds()
	if int64(elapsed) >= channelModelRateLimitWindowSec {
		now := time.Now().Format(channelModelRateLimitTimeFormat)
		pipe := rdb.TxPipeline()
		pipe.LPush(ctx, key, now)
		pipe.LTrim(ctx, key, 0, int64(rule.RPM-1))
		pipe.Expire(ctx, key, time.Duration(channelModelRateLimitWindowSec)*time.Second)
		if _, err := pipe.Exec(ctx); err != nil {
			return false, 0, err
		}
		return true, 0, nil
	}

	retryAfter := int(channelModelRateLimitWindowSec - int64(elapsed))
	if retryAfter < 1 {
		retryAfter = 1
	}
	rdb.Expire(ctx, key, time.Duration(channelModelRateLimitWindowSec)*time.Second)
	return false, retryAfter, nil
}

func tryAcquireChannelModelRateLimitMemorySlidingWindow(key string, rule dto.ChannelModelRateLimitRule) (bool, int, error) {
	channelModelMemoryLimiterOnce.Do(func() {
		channelModelMemoryLimiter.Init(time.Duration(channelModelRateLimitWindowSec) * time.Second)
	})
	if channelModelMemoryLimiter.Request(key, rule.RPM, channelModelRateLimitWindowSec) {
		return true, 0, nil
	}
	return false, 1, nil
}

func tryAcquireChannelModelRateLimitMemoryTokenBucket(key string, rule dto.ChannelModelRateLimitRule) (bool, int, error) {
	capacity, rate, requested := channelModelTokenBucketConfig(rule)
	bucket := getChannelModelTokenBucket(key, float64(capacity), float64(rate))
	ok, retryAfter := bucket.allow(float64(requested))
	if !ok {
		if retryAfter <= 0 {
			retryAfter = float64(retryAfterForTokenBucket(rule))
		}
		return false, int(math.Ceil(retryAfter)), nil
	}
	return true, 0, nil
}

func getChannelModelTokenBucket(key string, capacity, rate float64) *channelModelTokenBucket {
	if v, ok := channelModelTokenBuckets.Load(key); ok {
		return v.(*channelModelTokenBucket)
	}
	b := &channelModelTokenBucket{
		tokens:   capacity,
		capacity: capacity,
		rate:     rate,
		last:     time.Now(),
	}
	if actual, loaded := channelModelTokenBuckets.LoadOrStore(key, b); loaded {
		return actual.(*channelModelTokenBucket)
	}
	return b
}

func (b *channelModelTokenBucket) allow(cost float64) (bool, float64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.last).Seconds()
	b.last = now
	b.tokens += elapsed * b.rate
	if b.tokens > b.capacity {
		b.tokens = b.capacity
	}
	if b.tokens < cost {
		retryAfter := 0.0
		if b.rate > 0 {
			retryAfter = (cost - b.tokens) / b.rate
		}
		return false, retryAfter
	}
	b.tokens -= cost
	return true, 0
}
