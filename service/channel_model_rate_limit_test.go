package service

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func TestSanitizeChannelModelRateLimits(t *testing.T) {
	enabled := true
	disabled := false
	rules := []dto.ChannelModelRateLimitRule{
		{Model: " glm-5.2 ", RPM: 500, Enabled: &enabled},
		{Model: "glm-5.2", RPM: 300},
		{Model: "", RPM: 100},
		{Model: "gpt-4.1", RPM: 0},
		{Model: "claude-3", RPM: 100, Enabled: &disabled},
	}
	out := SanitizeChannelModelRateLimits(rules)
	require.Len(t, out, 1)
	require.Equal(t, "glm-5.2", out[0].Model)
	require.Equal(t, 500, out[0].RPM)
	require.NotNil(t, out[0].Enabled)
	require.True(t, *out[0].Enabled)
}

func TestMatchChannelModelRateLimit(t *testing.T) {
	ch := &model.Channel{
		OtherSettings: `{"model_rate_limits":[{"model":"glm-5.2","rpm":500,"enabled":true}]}`,
	}
	rule := MatchChannelModelRateLimit(ch, "glm-5.2")
	require.NotNil(t, rule)
	require.Equal(t, 500, rule.RPM)
	require.Nil(t, MatchChannelModelRateLimit(ch, "gpt-4.1"))
}

func TestTryAcquireChannelModelRateLimitMemory(t *testing.T) {
	enabled := true
	rule := dto.ChannelModelRateLimitRule{Model: "glm-5.2", RPM: 2, Enabled: &enabled}
	allowed, _, err := TryAcquireChannelModelRateLimit(1, "glm-5.2", rule)
	require.NoError(t, err)
	require.True(t, allowed)
	allowed, _, err = TryAcquireChannelModelRateLimit(1, "glm-5.2", rule)
	require.NoError(t, err)
	require.True(t, allowed)
	allowed, retryAfter, err := TryAcquireChannelModelRateLimit(1, "glm-5.2", rule)
	require.NoError(t, err)
	require.False(t, allowed)
	require.GreaterOrEqual(t, retryAfter, 1)
}

func TestTryAcquireChannelModelRateLimitMemoryBurst(t *testing.T) {
	enabled := true
	rule := dto.ChannelModelRateLimitRule{Model: "glm-5.2", RPM: 2, Burst: 3, Enabled: &enabled}
	key := "rl:cm:99:glm-5.2"
	channelModelTokenBuckets.Delete(key + ":tb")
	channelModelTokenBuckets.Delete(key)

	for i := 0; i < 5; i++ {
		allowed, _, err := TryAcquireChannelModelRateLimit(99, "glm-5.2", rule)
		require.NoError(t, err)
		require.Truef(t, allowed, "request %d should be allowed", i+1)
	}
	allowed, retryAfter, err := TryAcquireChannelModelRateLimit(99, "glm-5.2", rule)
	require.NoError(t, err)
	require.False(t, allowed)
	require.GreaterOrEqual(t, retryAfter, 1)
}

func TestChannelModelTokenBucketConfig(t *testing.T) {
	capacity, rate, requested := channelModelTokenBucketConfig(dto.ChannelModelRateLimitRule{RPM: 2, Burst: 3})
	require.Equal(t, int64(300), capacity)
	require.Equal(t, int64(2), rate)
	require.Equal(t, int64(60), requested)
}
