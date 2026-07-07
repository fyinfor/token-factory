package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

func enforceChannelModelRateLimit(c *gin.Context, channel *model.Channel, modelName string) bool {
	if tfErr := channelModelRateLimitError(channel, modelName); tfErr != nil {
		if retryAfter := rateLimitRetryAfterSeconds(tfErr); retryAfter > 0 {
			c.Header("Retry-After", strconv.Itoa(retryAfter))
			c.Header("X-RateLimit-Reset-Requests", strconv.Itoa(retryAfter))
		}
		c.Header("X-RateLimit-Remaining-Requests", "0")
		abortWithOpenAiMessage(c, tfErr.StatusCode, tfErr.Error(), tfErr.GetErrorCode())
		return false
	}
	if rule := service.MatchChannelModelRateLimit(channel, modelName); rule != nil {
		c.Header("X-RateLimit-Limit-Requests", strconv.Itoa(rule.RPM))
	}
	return true
}

func channelModelRateLimitError(channel *model.Channel, modelName string) *types.TokenFactoryError {
	rule := service.MatchChannelModelRateLimit(channel, modelName)
	if rule == nil {
		return nil
	}
	allowed, retryAfter, err := service.TryAcquireChannelModelRateLimit(channel.Id, modelName, *rule)
	if err != nil {
		return types.NewError(err, types.ErrorCodePlatformModelRateLimit)
	}
	if allowed {
		return nil
	}
	message := fmt.Sprintf(
		"平台模型限流：渠道「%s」的模型 %s 已达到 %d RPM 上限",
		channel.Name,
		modelName,
		rule.RPM,
	)
	tfErr := types.NewErrorWithStatusCode(
		errors.New(message),
		types.ErrorCodePlatformModelRateLimit,
		http.StatusTooManyRequests,
		types.ErrOptionWithSkipRetry(),
	)
	tfErr.Metadata = []byte(fmt.Sprintf(`{"retry_after":%d,"rpm":%d}`, retryAfter, rule.RPM))
	return tfErr
}

func ChannelModelRateLimitError(channel *model.Channel, modelName string) *types.TokenFactoryError {
	return channelModelRateLimitError(channel, modelName)
}

func rateLimitRetryAfterSeconds(tfErr *types.TokenFactoryError) int {
	if tfErr == nil || len(tfErr.Metadata) == 0 {
		return 0
	}
	var meta struct {
		RetryAfter int `json:"retry_after"`
	}
	if err := common.Unmarshal(tfErr.Metadata, &meta); err != nil {
		return 0
	}
	return meta.RetryAfter
}

func proceedRelayWithChannel(c *gin.Context, channel *model.Channel, modelName string) {
	if !enforceChannelModelRateLimit(c, channel, modelName) {
		return
	}
	c.Next()
	if channel != nil && c.Writer != nil && c.Writer.Status() < http.StatusBadRequest {
		service.RecordChannelAffinity(c, channel.Id)
	}
}
