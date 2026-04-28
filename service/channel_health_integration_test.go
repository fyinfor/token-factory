package service

import (
	"context"
	"net/http"
	"sync"
	"testing"

	"github.com/QuantumNous/new-api/model"

	"github.com/stretchr/testify/require"
)

// truncateMTR 给 PR1 集成测试单独清表，避免依赖 truncate(t) 引入的 user/token 等表。
func truncateMTR(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		model.DB.Exec("DELETE FROM model_test_results")
		channelHealthLocks = sync.Map{}
	})
}

// TestPR1_Schema_HasNewColumns 直接查 SQLite 元数据，确认 AutoMigrate 把新列都建出来了。
// 这是「字段补齐」改动的最直观证据：列名 + 默认值都按 GORM tag 兑现。
func TestPR1_Schema_HasNewColumns(t *testing.T) {
	truncateMTR(t)

	type colInfo struct {
		Name string `gorm:"column:name"`
	}
	var cols []colInfo
	require.NoError(t, model.DB.Raw("PRAGMA table_info(model_test_results)").Scan(&cols).Error)

	have := map[string]bool{}
	for _, c := range cols {
		have[c.Name] = true
	}

	expected := []string{
		"channel_id", "model_name",
		"last_test_success", "last_test_time", "last_response_time", "last_test_message",
		"test_count_success", "test_count_fail",
		"manual_display_response_time", "manual_stability_grade",
		"last_ttft_ms", "last_throughput_tps",
		"latency_ewma_ms", "latency_p50_ms", "latency_p95_ms",
		"recent_success_rate", "recent_sample_count",
		"consecutive_fail_count", "unhealthy_until", "circuit_state",
		"last_error_category", "last_http_status",
		"last_test_endpoint_type", "last_signal_source",
	}
	for _, name := range expected {
		require.Truef(t, have[name], "AutoMigrate 应建出列 %q，实际列：%v", name, have)
	}
}

// TestPR1_LegacyUpsertWrapper_PreservesBehavior 验证旧签名 UpsertModelTestResult 仍按原样工作：
//   - 基础字段写入；
//   - 累计计数 +1；
//   - 新增的健康度列保持默认值 0/空串（不被旧 wrapper 误覆盖）。
// 这是 controller/channel-test.go 等四处调用方零改动的安全网。
func TestPR1_LegacyUpsertWrapper_PreservesBehavior(t *testing.T) {
	truncateMTR(t)

	const channelID = 9001
	const modelName = "gpt-4o-mini"

	require.NoError(t, model.UpsertModelTestResult(channelID, modelName, true, 1234, ""))

	row, err := model.GetModelTestResult(channelID, modelName)
	require.NoError(t, err)
	require.NotNil(t, row)

	require.Equal(t, channelID, row.ChannelId)
	require.Equal(t, modelName, row.ModelName)
	require.True(t, row.LastTestSuccess)
	require.Equal(t, 1234, row.LastResponseTime)
	require.Equal(t, 1, row.TestCountSuccess)
	require.Equal(t, 0, row.TestCountFail)

	require.Equal(t, 0, row.LastTtftMs, "新增列在旧 wrapper 路径下应保持默认 0")
	require.InDelta(t, 0.0, row.LatencyEwmaMs, 1e-9)
	require.Equal(t, 0, row.RecentSampleCount)
	require.Equal(t, "", row.CircuitState)
	require.Equal(t, "", row.LastErrorCategory)
	require.Equal(t, "", row.LastSignalSource)

	// 第二次失败：累计计数 +1，最近一次字段切到失败语义。
	require.NoError(t, model.UpsertModelTestResult(channelID, modelName, false, 0, "boom"))
	row2, err := model.GetModelTestResult(channelID, modelName)
	require.NoError(t, err)
	require.False(t, row2.LastTestSuccess)
	require.Equal(t, "boom", row2.LastTestMessage)
	require.Equal(t, 1, row2.TestCountSuccess)
	require.Equal(t, 1, row2.TestCountFail)
}

// TestPR1_RecordChannelHealthSignal_HappyPath 验证「成功路径」: EWMA 推进、连续失败清零、信号来源写入。
func TestPR1_RecordChannelHealthSignal_HappyPath(t *testing.T) {
	truncateMTR(t)

	const channelID = 9002
	const modelName = "claude-3-5-sonnet"

	for i := 0; i < 3; i++ {
		RecordChannelHealthSignal(context.Background(), HealthSignalInput{
			ChannelID:    channelID,
			ModelName:    modelName,
			Success:      true,
			LatencyMs:    1000,
			EndpointType: "chat",
			Source:       model.HealthSignalSourceRelay,
		})
	}

	row, err := model.GetModelTestResult(channelID, modelName)
	require.NoError(t, err)
	require.NotNil(t, row)

	require.True(t, row.LastTestSuccess)
	require.Equal(t, 3, row.TestCountSuccess)
	require.Equal(t, 0, row.TestCountFail)
	require.Equal(t, 0, row.ConsecutiveFailCount)
	require.Equal(t, model.CircuitStateClosed, row.CircuitState)
	require.EqualValues(t, 0, row.UnhealthyUntil)
	require.Equal(t, "chat", row.LastTestEndpointType)
	require.Equal(t, model.HealthSignalSourceRelay, row.LastSignalSource)
	// 三次同样的 1000ms：EWMA 应快速收敛到 1000 附近。
	require.InDelta(t, 1000.0, row.LatencyEwmaMs, 1.0)
	require.Greater(t, row.RecentSampleCount, 0)
	require.InDelta(t, 1.0, row.RecentSuccessRate, 0.001)
}

// TestPR1_RecordChannelHealthSignal_TripsCircuitOnHighFailureRate
// 写入足够样本数（≥20）的失败，触发熔断 closed → open，并校验 unhealthy_until 与状态。
func TestPR1_RecordChannelHealthSignal_TripsCircuitOnHighFailureRate(t *testing.T) {
	truncateMTR(t)

	const channelID = 9003
	const modelName = "deepseek-chat"

	// 25 次连续失败，足以触发熔断（min sample=20, threshold=0.5）。
	for i := 0; i < 25; i++ {
		RecordChannelHealthSignal(context.Background(), HealthSignalInput{
			ChannelID:  channelID,
			ModelName:  modelName,
			Success:    false,
			LatencyMs:  100,
			HTTPStatus: http.StatusInternalServerError,
			Source:     model.HealthSignalSourceRelay,
		})
	}

	row, err := model.GetModelTestResult(channelID, modelName)
	require.NoError(t, err)
	require.NotNil(t, row)

	require.False(t, row.LastTestSuccess)
	require.Equal(t, 25, row.TestCountFail)
	require.GreaterOrEqual(t, row.ConsecutiveFailCount, 20)
	require.Equal(t, model.CircuitStateOpen, row.CircuitState)
	require.Greater(t, row.UnhealthyUntil, int64(0))
	require.Equal(t, model.ErrorCategoryUpstream5xx, row.LastErrorCategory)
	require.Equal(t, http.StatusInternalServerError, row.LastHttpStatus)
	require.True(t, IsChannelHealthCircuitOpen(row), "IsChannelHealthCircuitOpen 应判定为 open")
}

// TestPR1_RecordChannelHealthSignal_AuthErrorTripsImmediately 验证 auth 类失败立刻熔断（不等样本数）。
func TestPR1_RecordChannelHealthSignal_AuthErrorTripsImmediately(t *testing.T) {
	truncateMTR(t)

	const channelID = 9004
	const modelName = "gpt-4o"

	RecordChannelHealthSignal(context.Background(), HealthSignalInput{
		ChannelID:  channelID,
		ModelName:  modelName,
		Success:    false,
		LatencyMs:  50,
		HTTPStatus: http.StatusUnauthorized,
		Source:     model.HealthSignalSourceRelay,
	})

	row, err := model.GetModelTestResult(channelID, modelName)
	require.NoError(t, err)
	require.NotNil(t, row)

	require.Equal(t, model.CircuitStateOpen, row.CircuitState)
	require.Equal(t, model.ErrorCategoryAuth, row.LastErrorCategory)
	require.Equal(t, http.StatusUnauthorized, row.LastHttpStatus)
	require.Greater(t, row.UnhealthyUntil, int64(0))
}
