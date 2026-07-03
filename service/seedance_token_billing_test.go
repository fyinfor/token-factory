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

func TestSettleSeedanceVideoTokenDeltaWritesSettlementMarkerWithZeroQuota(t *testing.T) {
	truncate(t)
	seedUser(t, 1, 1000000)

	task := &model.Task{
		TaskID:    "task_token_marker",
		UserId:    1,
		Group:     "default",
		ChannelId: 1,
		Quota:     750000,
		Status:    model.TaskStatusSuccess,
		PrivateData: model.TaskPrivateData{
			BillingContext: &model.TaskBillingContext{
				GroupRatio:    1,
				VideoRuleUnit: VideoRuleUnitPerToken,
			},
		},
	}

	settleSeedanceVideoTokenDelta(context.Background(), task, 550000, map[string]interface{}{
		"billing_mode":         SeedanceVideoTokenBillingMode,
		"video_total_tokens":   50000,
		"video_billed_quota":   550000,
		"video_quota_per_unit": common.QuotaPerUnit,
	})

	var logs []model.Log
	require.NoError(t, model.LOG_DB.Order("id asc").Find(&logs).Error)
	require.Len(t, logs, 1)
	require.Equal(t, model.LogTypeConsume, logs[0].Type)
	require.Equal(t, 0, logs[0].Quota)

	other, err := common.StrToMap(logs[0].Other)
	require.NoError(t, err)
	require.Equal(t, model.BillingPhaseSettlementMarker, other["billing_phase"])
	require.Equal(t, false, other["affects_balance"])
	require.EqualValues(t, 550000, other["actual_quota"])
	require.EqualValues(t, 750000, other["pre_consumed_quota"])
	require.EqualValues(t, 550000, other["display_quota"])
	require.EqualValues(t, 0, other["balance_delta"])
}
