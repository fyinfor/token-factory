package model

import (
	"testing"

	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

// 模拟库内 GV-3.1-fast：渠道 854x480 文生视频 $1/s，全局同档 $2/s，成本 100%、加价 100%。
func TestBuildVideoFlatClipHint_AppliesMarkupDiscount(t *testing.T) {
	channelRules := ratio_setting.VideoPricingRules{
		TextToVideoPerSecond: []ratio_setting.VideoResolutionAudioPriceRule{
			{Resolution: "854x480", HasAudio: false, Price: 1},
		},
	}
	globalRules := ratio_setting.VideoPricingRules{
		TextToVideoPerSecond: []ratio_setting.VideoResolutionAudioPriceRule{
			{Resolution: "854x480", HasAudio: false, Price: 2},
		},
	}
	// 直接测档位展示价：1*100% + 2*100% = 3
	tiers := collectVideoPerSecondTiers(channelRules)
	rows := buildSortedTierRows(tiers, globalRules, 100, 100)
	if len(rows) != 1 {
		t.Fatalf("rows=%d", len(rows))
	}
	if rows[0].UsdAfterChannelDiscount != 3 {
		t.Fatalf("usd=%v want 3", rows[0].UsdAfterChannelDiscount)
	}
	if rows[0].UsdChannelRaw != 1 {
		t.Fatalf("channel raw=%v want 1", rows[0].UsdChannelRaw)
	}
	if rows[0].UsdOfficial != 2 {
		t.Fatalf("official=%v want 2", rows[0].UsdOfficial)
	}
}

func TestBuildVideoFlatClipHint_PerTokenPriority(t *testing.T) {
	channelRules := ratio_setting.VideoPricingRules{
		TextToVideoPerToken: []ratio_setting.VideoResolutionAudioPriceRule{
			{Resolution: "1280x720", HasAudio: false, Price: 0.00002},
		},
		TextToVideoPerSecond: []ratio_setting.VideoResolutionAudioPriceRule{
			{Resolution: "1280x720", HasAudio: false, Price: 1},
		},
	}
	globalRules := ratio_setting.VideoPricingRules{
		TextToVideoPerToken: []ratio_setting.VideoResolutionAudioPriceRule{
			{Resolution: "1280x720", HasAudio: false, Price: 0.00003},
		},
	}
	_ = globalRules
	tiers := collectVideoPerTokenTiers(channelRules)
	if len(tiers) != 1 {
		t.Fatalf("per_token tiers=%d want 1", len(tiers))
	}
	if tiers[0].Lane != "text_to_video_per_token" {
		t.Fatalf("lane=%s", tiers[0].Lane)
	}
	rows := buildSortedTierRows(tiers, ratio_setting.VideoPricingRules{}, 100, 0)
	if len(rows) != 1 || rows[0].UsdAfterChannelDiscount <= 0 {
		t.Fatalf("rows=%+v", rows)
	}
}
