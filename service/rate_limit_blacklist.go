package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
)

const rateLimitUserBlacklistKeyPrefix = "rateLimit:blacklist:user:"

type RateLimitBlacklistItem struct {
	UserID     int    `json:"user_id"`
	TTLSeconds int64  `json:"ttl_seconds"`
	Reason     string `json:"reason"`
}

func AddUserRateLimitBlacklist(userID int, ttlSeconds int64, reason string) error {
	if !common.RedisEnabled || common.RDB == nil || userID <= 0 || ttlSeconds <= 0 {
		return nil
	}
	ctx := context.Background()
	key := fmt.Sprintf("%s%d", rateLimitUserBlacklistKeyPrefix, userID)
	return common.RDB.Set(ctx, key, strings.TrimSpace(reason), time.Duration(ttlSeconds)*time.Second).Err()
}

func IsUserRateLimitBlacklisted(userID int) (bool, error) {
	if !common.RedisEnabled || common.RDB == nil || userID <= 0 {
		return false, nil
	}
	ctx := context.Background()
	key := fmt.Sprintf("%s%d", rateLimitUserBlacklistKeyPrefix, userID)
	exists, err := common.RDB.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return exists > 0, nil
}

func RemoveUserRateLimitBlacklist(userID int) error {
	if !common.RedisEnabled || common.RDB == nil || userID <= 0 {
		return nil
	}
	ctx := context.Background()
	keys := []string{
		fmt.Sprintf("%s%d", rateLimitUserBlacklistKeyPrefix, userID),
		// user-based counters in middleware/rate-limit.go
		fmt.Sprintf("rateLimit:GA:user:%d", userID),
		fmt.Sprintf("rateLimit:GW:user:%d", userID),
		fmt.Sprintf("rateLimit:CT:user:%d", userID),
		fmt.Sprintf("rateLimit:SR:user:%d", userID),
		// model request counters in middleware/model-rate-limit.go
		fmt.Sprintf("rateLimit:MRRLS:%d", userID),
		fmt.Sprintf("rateLimit:%d", userID),
	}
	if err := common.RDB.Del(ctx, keys...).Err(); err != nil {
		return err
	}

	// Backward compatibility for historical key styles.
	patterns := []string{
		fmt.Sprintf("rateLimit:MRRL%d*", userID),
		fmt.Sprintf("rateLimit:MRRLS%d*", userID),
	}
	for _, pattern := range patterns {
		iter := common.RDB.Scan(ctx, 0, pattern, 100).Iterator()
		extraKeys := make([]string, 0)
		for iter.Next(ctx) {
			extraKeys = append(extraKeys, iter.Val())
		}
		if err := iter.Err(); err != nil {
			return err
		}
		if len(extraKeys) > 0 {
			if err := common.RDB.Del(ctx, extraKeys...).Err(); err != nil {
				return err
			}
		}
	}
	return nil
}

func ListUserRateLimitBlacklist(limit int64) ([]RateLimitBlacklistItem, error) {
	if !common.RedisEnabled || common.RDB == nil {
		return []RateLimitBlacklistItem{}, nil
	}
	if limit <= 0 {
		limit = 200
	}
	ctx := context.Background()
	pattern := rateLimitUserBlacklistKeyPrefix + "*"
	iter := common.RDB.Scan(ctx, 0, pattern, limit).Iterator()

	items := make([]RateLimitBlacklistItem, 0)
	for iter.Next(ctx) {
		key := iter.Val()
		idStr := strings.TrimPrefix(key, rateLimitUserBlacklistKeyPrefix)
		userID, err := strconv.Atoi(idStr)
		if err != nil || userID <= 0 {
			continue
		}
		ttl, err := common.RDB.TTL(ctx, key).Result()
		if err != nil {
			continue
		}
		reason, _ := common.RDB.Get(ctx, key).Result()
		ttlSeconds := int64(ttl.Seconds())
		if ttlSeconds < 0 {
			ttlSeconds = 0
		}
		items = append(items, RateLimitBlacklistItem{
			UserID:     userID,
			TTLSeconds: ttlSeconds,
			Reason:     reason,
		})
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
