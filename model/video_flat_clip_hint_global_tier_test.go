package model

import (
	"testing"

	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

// 渠道档 854x480 $3/s + 全局同档 $2/s × 加价 25% => 3.5；不得用成片 1280x720 的全局 $4/s（会得到 4）。
func TestLookupVideoTierRawUSD_AlignedTierEffectivePrice(t *testing.T) {
	globalRules := ratio_setting.VideoPricingRules{
		TextToVideoPerSecond: []ratio_setting.VideoResolutionAudioPriceRule{
			{Resolution: "854x480", HasAudio: false, Price: 2},
			{Resolution: "1280x720", HasAudio: false, Price: 4},
		},
	}
	falseAudio := false
	tier := videoFlatTier{
		Res:      "854x480",
		Lane:     "text_to_video_per_second",
		HasAudio: &falseAudio,
	}
	global := lookupVideoTierRawUSD(globalRules, tier)
	if global != 2 {
		t.Fatalf("global 854x480 tier = %v want 2", global)
	}
	eff := EffectiveRuleUnitPrice(3, global, 100, 25)
	if eff != 3.5 {
		t.Fatalf("effective = %v want 3.5", eff)
	}
	wrongGlobal := lookupVideoTierRawUSD(globalRules, videoFlatTier{
		Res:      "1280x720",
		Lane:     "text_to_video_per_second",
		HasAudio: &falseAudio,
	})
	if wrongGlobal != 4 {
		t.Fatalf("global 1280x720 = %v want 4", wrongGlobal)
	}
	wrongEff := EffectiveRuleUnitPrice(3, wrongGlobal, 100, 25)
	if wrongEff != 4 {
		t.Fatalf("wrong effective = %v want 4 (bug case)", wrongEff)
	}
}
