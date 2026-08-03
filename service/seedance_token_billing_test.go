package service

import (
	"context"
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/stretchr/testify/require"
)

func withVideoPricingRules(t *testing.T, cfg string) {
	t.Helper()
	prev := ratio_setting.VideoPricingRules2JSONString()
	t.Cleanup(func() { _ = ratio_setting.UpdateVideoPricingRulesByJSONString(prev) })
	require.NoError(t, ratio_setting.UpdateVideoPricingRulesByJSONString(cfg))
}

func TestShouldUseVideoTokenBilling_PerTokenRulesConfigured(t *testing.T) {
	modelName := "seedance-test-per-token"
	withVideoPricingRules(t, `{"`+modelName+`":{"text_to_video_per_token":[{"resolution":"1280x720","has_audio":false,"price":0.15}]}}`)
	require.True(t, ShouldUseVideoTokenBilling(0, modelName))
}

func TestShouldUseVideoTokenBilling_NoPerTokenRules(t *testing.T) {
	modelName := "seedance-test-no-per-token"
	withVideoPricingRules(t, `{"`+modelName+`":{"text_to_video_per_second":[{"resolution":"1280x720","has_audio":false,"price":0.01}]}}`)
	require.False(t, ShouldUseVideoTokenBilling(0, modelName))
}

func TestCalcVideoTokenQuota_PreConsumeAndSettle(t *testing.T) {
	modelName := "seedance-test-quota"
	const pricePerMillionTokens = 0.15
	withVideoPricingRules(t, `{"`+modelName+`":{"text_to_video_per_token":[{"resolution":"1280x720","has_audio":false,"price":`+
		strconv.FormatFloat(pricePerMillionTokens, 'g', -1, 64)+`}]}}`)

	groupRatio := 1.0
	mode := relaycommon.VideoBillingModeTextToVideo
	width, height := 1280, 720

	preQuota, _, ok := CalcVideoTokenQuota(0, 0, modelName, mode, width, height, false, SeedanceTokenPreConsumeTokens, groupRatio, "")
	require.True(t, ok)
	require.Greater(t, preQuota, 0)

	settleQuota, _, ok := CalcVideoTokenQuota(0, 0, modelName, mode, width, height, false, 50638, groupRatio, "")
	require.True(t, ok)
	require.Greater(t, settleQuota, preQuota)

	expectedPre := int(
		(float64(SeedanceTokenPreConsumeTokens) / VideoTokenPricePerMillion) *
			pricePerMillionTokens * common.QuotaPerUnit * groupRatio,
	)
	require.InDelta(t, expectedPre, preQuota, 1)
}

func TestMatchVideoTokenRule_DetectsImageToVideoLane(t *testing.T) {
	modelName := "seedance-test-lane"
	withVideoPricingRules(t, `{"`+modelName+`":{"image_to_video_per_token":[{"resolution":"1280x720","has_audio":false,"price":0.31}]}}`)
	match, ok := MatchVideoTokenRuleForRequest(
		0, 0, modelName, relaycommon.VideoBillingModeImageToVideo, 1280, 720, false, "720p",
	)
	require.True(t, ok)
	require.Equal(t, relaycommon.VideoBillingModeImageToVideo, match.Mode)
	require.InDelta(t, 0.31, match.ChannelPricePerToken, 1e-9)
}

func newSeedanceTokenSettleTask(preConsumed int) *model.Task {
	return &model.Task{
		TaskID:    "task_token_settle",
		UserId:    1,
		Group:     "default",
		ChannelId: 1,
		Quota:     preConsumed,
		Status:    model.TaskStatusSuccess,
		PrivateData: model.TaskPrivateData{
			BillingContext: &model.TaskBillingContext{
				GroupRatio:    1,
				VideoRuleUnit: VideoRuleUnitPerToken,
			},
		},
	}
}

func settleSeedanceTokenExtra(actualQuota int) map[string]interface{} {
	return map[string]interface{}{
		"billing_mode":         SeedanceVideoTokenBillingMode,
		"video_total_tokens":   50000,
		"video_billed_quota":   actualQuota,
		"video_quota_per_unit": common.QuotaPerUnit,
	}
}

// 预扣与实扣一致：没有资金变动，结算日志只作展示标记，quota 必须为 0，
// 否则会和预扣日志一起被重复计入消费统计。
func TestSettleSeedanceVideoTokenDelta_ZeroDeltaWritesSettlementMarker(t *testing.T) {
	truncate(t)
	seedUser(t, 1, 1000000)

	task := newSeedanceTokenSettleTask(550000)
	settleSeedanceVideoTokenDelta(context.Background(), task, 550000, settleSeedanceTokenExtra(550000))

	var logs []model.Log
	require.NoError(t, model.LOG_DB.Order("id asc").Find(&logs).Error)
	require.Len(t, logs, 1)
	require.Equal(t, model.LogTypeConsume, logs[0].Type)
	require.Equal(t, 0, logs[0].Quota)
	require.Equal(t, 1000000, getUserQuota(t, 1))

	other, err := common.StrToMap(logs[0].Other)
	require.NoError(t, err)
	require.Equal(t, model.BillingPhaseSettlementMarker, other["billing_phase"])
	require.Equal(t, false, other["affects_balance"])
	require.EqualValues(t, 550000, other["actual_quota"])
	require.EqualValues(t, 550000, other["pre_consumed_quota"])
	require.EqualValues(t, 550000, other["display_quota"])
	require.EqualValues(t, 0, other["balance_delta"])
}

// 实扣大于预扣：差额已经从余额扣走，日志 quota 必须记这笔差额。
// 这样「预扣日志 + 差额日志」按 quota 求和才等于实扣总额（/api/log/stat、对账单同一口径）。
func TestSettleSeedanceVideoTokenDelta_PositiveDeltaWritesDeltaChargeLog(t *testing.T) {
	truncate(t)
	seedUserWithUsed(t, 1, 1000000, 154039)

	const preConsumed, actualQuota = 154039, 335497
	const delta = actualQuota - preConsumed

	task := newSeedanceTokenSettleTask(preConsumed)
	settleSeedanceVideoTokenDelta(context.Background(), task, actualQuota, settleSeedanceTokenExtra(actualQuota))

	require.Equal(t, actualQuota, task.Quota)
	require.Equal(t, 1000000-delta, getUserQuota(t, 1))
	require.Equal(t, preConsumed+delta, getUserUsedQuota(t, 1))

	var logs []model.Log
	require.NoError(t, model.LOG_DB.Order("id asc").Find(&logs).Error)
	require.Len(t, logs, 1)
	require.Equal(t, model.LogTypeConsume, logs[0].Type)
	require.Equal(t, delta, logs[0].Quota)

	other, err := common.StrToMap(logs[0].Other)
	require.NoError(t, err)
	require.Equal(t, model.BillingPhaseDeltaCharge, other["billing_phase"])
	require.Equal(t, true, other["affects_balance"])
	require.EqualValues(t, actualQuota, other["actual_quota"])
	require.EqualValues(t, preConsumed, other["pre_consumed_quota"])
	require.EqualValues(t, actualQuota, other["video_final_quota"])
	require.EqualValues(t, delta, other["display_quota"])
	require.EqualValues(t, -delta, other["balance_delta"])
}

// 预扣大于实扣：差额退回余额，写 refund 日志。
// 「累积已用」只应回退一次（由 RecordTaskBillingLog 统一处理）。
func TestSettleSeedanceVideoTokenDelta_NegativeDeltaWritesDeltaRefundLog(t *testing.T) {
	truncate(t)
	seedUserWithUsed(t, 1, 1000000, 750000)

	const preConsumed, actualQuota = 750000, 550000
	const refund = preConsumed - actualQuota

	task := newSeedanceTokenSettleTask(preConsumed)
	settleSeedanceVideoTokenDelta(context.Background(), task, actualQuota, settleSeedanceTokenExtra(actualQuota))

	require.Equal(t, actualQuota, task.Quota)
	require.Equal(t, 1000000+refund, getUserQuota(t, 1))
	require.Equal(t, actualQuota, getUserUsedQuota(t, 1))

	var logs []model.Log
	require.NoError(t, model.LOG_DB.Order("id asc").Find(&logs).Error)
	require.Len(t, logs, 1)
	require.Equal(t, model.LogTypeRefund, logs[0].Type)
	require.Equal(t, refund, logs[0].Quota)

	other, err := common.StrToMap(logs[0].Other)
	require.NoError(t, err)
	require.Equal(t, model.BillingPhaseDeltaRefund, other["billing_phase"])
	require.Equal(t, true, other["affects_balance"])
	require.EqualValues(t, actualQuota, other["actual_quota"])
	require.EqualValues(t, preConsumed, other["pre_consumed_quota"])
	require.EqualValues(t, refund, other["display_quota"])
	require.EqualValues(t, refund, other["balance_delta"])
}
