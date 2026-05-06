package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

var timeFormat = "2006-01-02T15:04:05.000Z"

var inMemoryRateLimiter common.InMemoryRateLimiter

var defNext = func(c *gin.Context) {
	c.Next()
}

func redisRateLimiter(c *gin.Context, maxRequestNum int, duration int64, mark string) {
	if !shouldApplyGeneralRateLimit(c) {
		return
	}
	if shouldBypassRateLimit(c) {
		return
	}
	ctx := context.Background()
	rdb := common.RDB
	userID := getRateLimitUserID(c)
	key := "rateLimit:" + mark + ":" + buildRateLimitSubject(c)
	listLength, err := rdb.LLen(ctx, key).Result()
	if err != nil {
		fmt.Println(err.Error())
		c.Status(http.StatusInternalServerError)
		c.Abort()
		return
	}
	if listLength < int64(maxRequestNum) {
		rdb.LPush(ctx, key, time.Now().Format(timeFormat))
		rdb.Expire(ctx, key, common.RateLimitKeyExpirationDuration)
	} else {
		oldTimeStr, _ := rdb.LIndex(ctx, key, -1).Result()
		oldTime, err := time.Parse(timeFormat, oldTimeStr)
		if err != nil {
			fmt.Println(err)
			c.Status(http.StatusInternalServerError)
			c.Abort()
			return
		}
		nowTimeStr := time.Now().Format(timeFormat)
		nowTime, err := time.Parse(timeFormat, nowTimeStr)
		if err != nil {
			fmt.Println(err)
			c.Status(http.StatusInternalServerError)
			c.Abort()
			return
		}
		// time.Since will return negative number!
		// See: https://stackoverflow.com/questions/50970900/why-is-time-since-returning-negative-durations-on-windows
		if int64(nowTime.Sub(oldTime).Seconds()) < duration {
			rdb.Expire(ctx, key, common.RateLimitKeyExpirationDuration)
			if userID > 0 {
				_ = service.AddUserRateLimitBlacklist(userID, duration, "api-rate-limit:"+mark)
			}
			c.Status(http.StatusTooManyRequests)
			c.Abort()
			return
		} else {
			rdb.LPush(ctx, key, time.Now().Format(timeFormat))
			rdb.LTrim(ctx, key, 0, int64(maxRequestNum-1))
			rdb.Expire(ctx, key, common.RateLimitKeyExpirationDuration)
		}
	}
}

func memoryRateLimiter(c *gin.Context, maxRequestNum int, duration int64, mark string) {
	if !shouldApplyGeneralRateLimit(c) {
		return
	}
	if shouldBypassRateLimit(c) {
		return
	}
	userID := getRateLimitUserID(c)
	key := mark + ":" + buildRateLimitSubject(c)
	if !inMemoryRateLimiter.Request(key, maxRequestNum, duration) {
		if userID > 0 {
			_ = service.AddUserRateLimitBlacklist(userID, duration, "api-rate-limit:"+mark)
		}
		c.Status(http.StatusTooManyRequests)
		c.Abort()
		return
	}
}

// buildRateLimitSubject returns a stable per-user identifier when available.
// Priority: authenticated context id -> session id -> New-Api-User header -> client IP.
func buildRateLimitSubject(c *gin.Context) string {
	if userID := getRateLimitUserID(c); userID > 0 {
		return fmt.Sprintf("user:%d", userID)
	}
	return "ip:" + c.ClientIP()
}

func getRateLimitUserID(c *gin.Context) int {
	if userID := c.GetInt("id"); userID > 0 {
		return userID
	}
	session := sessions.Default(c)
	if sessionID := session.Get("id"); sessionID != nil {
		if id, ok := sessionID.(int); ok && id > 0 {
			return id
		}
		if id, ok := sessionID.(int64); ok && id > 0 {
			return int(id)
		}
		if id, ok := sessionID.(float64); ok && id > 0 {
			return int(id)
		}
		if idStr, ok := sessionID.(string); ok {
			if parsed, err := strconv.Atoi(idStr); err == nil && parsed > 0 {
				return parsed
			}
		}
	}
	if apiUserIDStr := c.GetHeader("New-Api-User"); apiUserIDStr != "" {
		if id, err := strconv.Atoi(apiUserIDStr); err == nil && id > 0 {
			return id
		}
	}
	return 0
}

func shouldBypassRateLimit(c *gin.Context) bool {
	userID := getRateLimitUserID(c)
	if userID <= 0 {
		return false
	}
	return model.IsAdmin(userID) || setting.IsUserInRateLimitWhitelist(userID)
}

// shouldApplyGeneralRateLimit ensures GA/GW/CT only apply to identified users.
// Public unauthenticated routes (e.g. homepage) are excluded.
func shouldApplyGeneralRateLimit(c *gin.Context) bool {
	return getRateLimitUserID(c) > 0
}

func rateLimitFactory(maxRequestNum int, duration int64, mark string) func(c *gin.Context) {
	if common.RedisEnabled {
		return func(c *gin.Context) {
			redisRateLimiter(c, maxRequestNum, duration, mark)
		}
	} else {
		// It's safe to call multi times.
		inMemoryRateLimiter.Init(common.RateLimitKeyExpirationDuration)
		return func(c *gin.Context) {
			memoryRateLimiter(c, maxRequestNum, duration, mark)
		}
	}
}

func GlobalWebRateLimit() func(c *gin.Context) {
	// Web-side global limiter is intentionally disabled.
	// Use GlobalAPIRateLimit + CriticalRateLimit for authenticated API traffic.
	return defNext
}

func GlobalAPIRateLimit() func(c *gin.Context) {
	return func(c *gin.Context) {
		if !common.GlobalApiRateLimitEnable {
			c.Next()
			return
		}
		rateLimitFactory(common.GlobalApiRateLimitNum, common.GlobalApiRateLimitDuration, "GA")(c)
	}
}

func CriticalRateLimit() func(c *gin.Context) {
	return func(c *gin.Context) {
		if !common.CriticalRateLimitEnable {
			c.Next()
			return
		}
		rateLimitFactory(common.CriticalRateLimitNum, common.CriticalRateLimitDuration, "CT")(c)
	}
}

func DownloadRateLimit() func(c *gin.Context) {
	return rateLimitFactory(common.DownloadRateLimitNum, common.DownloadRateLimitDuration, "DW")
}

func UploadRateLimit() func(c *gin.Context) {
	return rateLimitFactory(common.UploadRateLimitNum, common.UploadRateLimitDuration, "UP")
}

// userRateLimitFactory creates a rate limiter keyed by authenticated user ID
// instead of client IP, making it resistant to proxy rotation attacks.
// Must be used AFTER authentication middleware (UserAuth).
func userRateLimitFactory(maxRequestNum int, duration int64, mark string) func(c *gin.Context) {
	if common.RedisEnabled {
		return func(c *gin.Context) {
			userId := c.GetInt("id")
			if userId == 0 {
				c.Status(http.StatusUnauthorized)
				c.Abort()
				return
			}
			// Admin users are always whitelisted by default.
			if model.IsAdmin(userId) || setting.IsUserInRateLimitWhitelist(userId) {
				return
			}
			blacklisted, err := service.IsUserRateLimitBlacklisted(userId)
			if err == nil && blacklisted {
				c.Status(http.StatusTooManyRequests)
				c.Abort()
				return
			}
			key := fmt.Sprintf("rateLimit:%s:user:%d", mark, userId)
			userRedisRateLimiter(c, maxRequestNum, duration, key)
		}
	}
	// It's safe to call multi times.
	inMemoryRateLimiter.Init(common.RateLimitKeyExpirationDuration)
	return func(c *gin.Context) {
		userId := c.GetInt("id")
		if userId == 0 {
			c.Status(http.StatusUnauthorized)
			c.Abort()
			return
		}
		if model.IsAdmin(userId) || setting.IsUserInRateLimitWhitelist(userId) {
			return
		}
		key := fmt.Sprintf("%s:user:%d", mark, userId)
		if !inMemoryRateLimiter.Request(key, maxRequestNum, duration) {
			c.Status(http.StatusTooManyRequests)
			c.Abort()
			return
		}
	}
}

// userRedisRateLimiter is like redisRateLimiter but accepts a pre-built key
// (to support user-ID-based keys).
func userRedisRateLimiter(c *gin.Context, maxRequestNum int, duration int64, key string) {
	ctx := context.Background()
	rdb := common.RDB
	listLength, err := rdb.LLen(ctx, key).Result()
	if err != nil {
		fmt.Println(err.Error())
		c.Status(http.StatusInternalServerError)
		c.Abort()
		return
	}
	if listLength < int64(maxRequestNum) {
		rdb.LPush(ctx, key, time.Now().Format(timeFormat))
		rdb.Expire(ctx, key, common.RateLimitKeyExpirationDuration)
	} else {
		oldTimeStr, _ := rdb.LIndex(ctx, key, -1).Result()
		oldTime, err := time.Parse(timeFormat, oldTimeStr)
		if err != nil {
			fmt.Println(err)
			c.Status(http.StatusInternalServerError)
			c.Abort()
			return
		}
		nowTimeStr := time.Now().Format(timeFormat)
		nowTime, err := time.Parse(timeFormat, nowTimeStr)
		if err != nil {
			fmt.Println(err)
			c.Status(http.StatusInternalServerError)
			c.Abort()
			return
		}
		if int64(nowTime.Sub(oldTime).Seconds()) < duration {
			rdb.Expire(ctx, key, common.RateLimitKeyExpirationDuration)
			_ = service.AddUserRateLimitBlacklist(c.GetInt("id"), duration, "user-rate-limit")
			c.Status(http.StatusTooManyRequests)
			c.Abort()
			return
		} else {
			rdb.LPush(ctx, key, time.Now().Format(timeFormat))
			rdb.LTrim(ctx, key, 0, int64(maxRequestNum-1))
			rdb.Expire(ctx, key, common.RateLimitKeyExpirationDuration)
		}
	}
}

// SearchRateLimit returns a per-user rate limiter for search endpoints.
// Configurable via SEARCH_RATE_LIMIT_ENABLE / SEARCH_RATE_LIMIT / SEARCH_RATE_LIMIT_DURATION.
func SearchRateLimit() func(c *gin.Context) {
	if !common.SearchRateLimitEnable {
		return defNext
	}
	return userRateLimitFactory(common.SearchRateLimitNum, common.SearchRateLimitDuration, "SR")
}
