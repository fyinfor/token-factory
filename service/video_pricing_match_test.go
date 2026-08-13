package service

import (
	"testing"

	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/stretchr/testify/require"
)

func TestMatchPerSecondPriceRowByLabel_Prefers720pOver1080pPixels(t *testing.T) {
	rows := []ratio_setting.VideoResolutionAudioPriceRule{
		{Resolution: "720p", HasAudio: false, Price: 0.01},
		{Resolution: "1080p", HasAudio: false, Price: 0.02},
	}
	// 像素尺寸像 1080p，但 resolution 标识为 720p 时应匹配 720p 单价。
	match, ok := matchPerSecondPriceRowByLabel(rows, "720p", 1920, 1080, false, true)
	require.True(t, ok)
	require.NotNil(t, match)
	require.InDelta(t, 0.01, match.PricePerSecond, 1e-9)
	require.Equal(t, "720p", match.Resolution)
}

func TestMatchPerSecondPriceDetail_LabelBeforePixelBucket(t *testing.T) {
	rules := ratio_setting.VideoPricingRules{
		TextToVideoPerSecond: []ratio_setting.VideoResolutionAudioPriceRule{
			{Resolution: "720p", HasAudio: false, Price: 0.03},
			{Resolution: "1080p", HasAudio: false, Price: 0.06},
		},
	}
	match, ok := matchPerSecondPriceDetail(rules, "text_to_video", 1920, 1080, false, "720p")
	require.True(t, ok)
	require.InDelta(t, 0.03, match.PricePerSecond, 1e-9)
}

func TestMatchPerSecondPriceRowByLabel_MiniMaxH3768P(t *testing.T) {
	rows := []ratio_setting.VideoResolutionAudioPriceRule{
		{Resolution: "720p", HasAudio: false, Price: 0.01},
		{Resolution: "1366x768", HasAudio: false, Price: 0.015},
		{Resolution: "2K", HasAudio: false, Price: 0.04},
	}
	match, ok := matchPerSecondPriceRowByLabel(rows, "768P", 1366, 768, false, true)
	require.True(t, ok)
	require.NotNil(t, match)
	require.InDelta(t, 0.015, match.PricePerSecond, 1e-9)
	require.Equal(t, "1366x768", match.Resolution)
}
