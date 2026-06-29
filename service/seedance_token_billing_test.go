package service

import (
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
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

	preQuota, _, ok := CalcVideoTokenQuota(0, 0, modelName, mode, width, height, false, SeedanceTokenPreConsumeTokens, groupRatio)
	require.True(t, ok)
	require.Greater(t, preQuota, 0)

	settleQuota, _, ok := CalcVideoTokenQuota(0, 0, modelName, mode, width, height, false, 50638, groupRatio)
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
		0, 0, modelName, relaycommon.VideoBillingModeImageToVideo, 1280, 720, false,
	)
	require.True(t, ok)
	require.Equal(t, relaycommon.VideoBillingModeImageToVideo, match.Mode)
	require.InDelta(t, 0.31, match.ChannelPricePerToken, 1e-9)
}
