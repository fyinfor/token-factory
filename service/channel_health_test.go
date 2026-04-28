package service

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/types"

	"github.com/stretchr/testify/require"
)

func TestClassifyError_HTTPStatus(t *testing.T) {
	cases := []struct {
		name   string
		status int
		want   string
	}{
		{"timeout-408", http.StatusRequestTimeout, model.ErrorCategoryTimeout},
		{"timeout-504", http.StatusGatewayTimeout, model.ErrorCategoryTimeout},
		{"rate-429", http.StatusTooManyRequests, model.ErrorCategoryRateLimit},
		{"auth-401", http.StatusUnauthorized, model.ErrorCategoryAuth},
		{"auth-403", http.StatusForbidden, model.ErrorCategoryAuth},
		{"bad-400", http.StatusBadRequest, model.ErrorCategoryBadRequest},
		{"upstream-500", http.StatusInternalServerError, model.ErrorCategoryUpstream5xx},
		{"upstream-502", http.StatusBadGateway, model.ErrorCategoryUpstream5xx},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyError(HealthSignalInput{Success: false, HTTPStatus: tc.status})
			require.Equal(t, tc.want, got)
		})
	}
}

func TestClassifyError_TokenFactoryErrorOverridesHTTPStatus(t *testing.T) {
	// TokenFactoryError 内部 StatusCode 优先于入参 HTTPStatus（通常 relay 层只拿到一个 wrapper）。
	tfe := types.NewOpenAIError(errors.New("upstream 500"), "upstream", http.StatusInternalServerError)
	got := classifyError(HealthSignalInput{
		Success:           false,
		HTTPStatus:        http.StatusOK,
		TokenFactoryError: tfe,
	})
	require.Equal(t, model.ErrorCategoryUpstream5xx, got)
}

func TestClassifyError_RawErrFallback(t *testing.T) {
	got := classifyError(HealthSignalInput{Success: false, RawErr: context.DeadlineExceeded})
	require.Equal(t, model.ErrorCategoryTimeout, got)

	got = classifyError(HealthSignalInput{Success: false, RawErr: errors.New("json unmarshal failed")})
	require.Equal(t, model.ErrorCategoryParse, got)

	got = classifyError(HealthSignalInput{Success: false, RawErr: errors.New("nothing matches")})
	require.Equal(t, model.ErrorCategoryOther, got)
}

func TestClassifyError_SuccessYieldsNone(t *testing.T) {
	require.Equal(t, model.ErrorCategoryNone, classifyError(HealthSignalInput{Success: true}))
}

func TestTransitionCircuit_ClosedNotEnoughSamples(t *testing.T) {
	state, until := transitionCircuit(circuitInputs{
		now:         1000,
		prevState:   model.CircuitStateClosed,
		success:     false,
		sampleCount: 5,
		successRate: 0.0,
		category:    model.ErrorCategoryUpstream5xx,
	})
	require.Equal(t, model.CircuitStateClosed, state)
	require.EqualValues(t, 0, until)
}

func TestTransitionCircuit_ClosedTripsOnFailureRate(t *testing.T) {
	state, until := transitionCircuit(circuitInputs{
		now:         1000,
		prevState:   model.CircuitStateClosed,
		success:     false,
		sampleCount: 30,
		successRate: 0.4,
		category:    model.ErrorCategoryUpstream5xx,
	})
	require.Equal(t, model.CircuitStateOpen, state)
	require.EqualValues(t, 1000+channelHealthCooldownSeconds, until)
}

func TestTransitionCircuit_AuthCategoryTripsImmediatelyWithLongCooldown(t *testing.T) {
	state, until := transitionCircuit(circuitInputs{
		now:         1000,
		prevState:   model.CircuitStateClosed,
		success:     false,
		sampleCount: 1,
		successRate: 0.0,
		category:    model.ErrorCategoryAuth,
	})
	require.Equal(t, model.CircuitStateOpen, state)
	require.EqualValues(t, 1000+channelHealthCooldownAuthSeconds, until)
}

func TestTransitionCircuit_OpenStaysWithinCooldown(t *testing.T) {
	state, until := transitionCircuit(circuitInputs{
		now:               1000,
		prevState:         model.CircuitStateOpen,
		prevUnhealthyTill: 1100,
		success:           true,
		sampleCount:       30,
		successRate:       1.0,
	})
	require.Equal(t, model.CircuitStateOpen, state)
	require.EqualValues(t, 1100, until)
}

func TestTransitionCircuit_OpenAfterCooldown_SuccessClosesCircuit(t *testing.T) {
	state, until := transitionCircuit(circuitInputs{
		now:               2000,
		prevState:         model.CircuitStateOpen,
		prevUnhealthyTill: 1500,
		success:           true,
		sampleCount:       30,
		successRate:       1.0,
	})
	require.Equal(t, model.CircuitStateClosed, state)
	require.EqualValues(t, 0, until)
}

func TestTransitionCircuit_OpenAfterCooldown_FailureExtendsCooldownExponentially(t *testing.T) {
	state, until := transitionCircuit(circuitInputs{
		now:               2000,
		prevState:         model.CircuitStateOpen,
		prevUnhealthyTill: 1500,
		success:           false,
		sampleCount:       30,
		successRate:       0.0,
		category:          model.ErrorCategoryUpstream5xx,
	})
	require.Equal(t, model.CircuitStateOpen, state)
	require.EqualValues(t, 2000+int64(channelHealthCooldownSeconds*2), until)
}

func TestTransitionCircuit_HalfOpen_SuccessClosesCircuit(t *testing.T) {
	state, until := transitionCircuit(circuitInputs{
		now:         3000,
		prevState:   model.CircuitStateHalfOpen,
		success:     true,
		sampleCount: 31,
		successRate: 0.97,
	})
	require.Equal(t, model.CircuitStateClosed, state)
	require.EqualValues(t, 0, until)
}

func TestTransitionCircuit_HalfOpen_FailureReopensWithBackoff(t *testing.T) {
	state, until := transitionCircuit(circuitInputs{
		now:         3000,
		prevState:   model.CircuitStateHalfOpen,
		success:     false,
		sampleCount: 31,
		successRate: 0.5,
		category:    model.ErrorCategoryUpstream5xx,
	})
	require.Equal(t, model.CircuitStateOpen, state)
	require.EqualValues(t, 3000+int64(channelHealthCooldownSeconds*2), until)
}

// TestComposeHealthSignal_FreshRow 验证冷启动（prev=nil）时 EWMA 直接采纳本次值，连续失败计数=1。
func TestComposeHealthSignal_FreshRow_FailureBootstrap(t *testing.T) {
	out := composeHealthSignal(nil, HealthSignalInput{
		ChannelID:  101,
		ModelName:  "gpt-4o-mini",
		Success:    false,
		LatencyMs:  1200,
		HTTPStatus: http.StatusInternalServerError,
		Source:     model.HealthSignalSourceRelay,
	})
	require.NotNil(t, out.LatencyEwmaMs)
	require.EqualValues(t, 0, *out.LatencyEwmaMs, "失败请求不应污染 EWMA")
	require.NotNil(t, out.ConsecutiveFailCount)
	require.Equal(t, 1, *out.ConsecutiveFailCount)
	require.NotNil(t, out.LastErrorCategory)
	require.Equal(t, model.ErrorCategoryUpstream5xx, *out.LastErrorCategory)
	require.NotNil(t, out.RecentSampleCount)
	require.Equal(t, 1, *out.RecentSampleCount)
}

// TestComposeHealthSignal_SuccessAfterFailureResetsConsecAndUpdatesEWMA 验证成功路径会清零连续失败、推进 EWMA。
func TestComposeHealthSignal_SuccessAfterFailureResetsConsec(t *testing.T) {
	prev := &model.ModelTestResult{
		LatencyEwmaMs:        500,
		RecentSampleCount:    20,
		RecentSuccessRate:    0.5,
		ConsecutiveFailCount: 3,
		LastTestTime:         900,
	}
	out := composeHealthSignal(prev, HealthSignalInput{
		ChannelID: 101,
		ModelName: "gpt-4o-mini",
		Success:   true,
		LatencyMs: 1000,
		Source:    model.HealthSignalSourceRelay,
	})
	require.NotNil(t, out.ConsecutiveFailCount)
	require.Equal(t, 0, *out.ConsecutiveFailCount)
	require.NotNil(t, out.LatencyEwmaMs)
	// EWMA = 0.2*1000 + 0.8*500 = 600
	require.InDelta(t, 600.0, *out.LatencyEwmaMs, 0.001)
	require.NotNil(t, out.LastErrorCategory)
	require.Equal(t, model.ErrorCategoryNone, *out.LastErrorCategory)
}
