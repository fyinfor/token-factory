package model

import (
	"testing"

	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

func TestCollectVideoFlatTiers_MinAcrossLanes(t *testing.T) {
	rules := ratio_setting.VideoPricingRules{
		TextToVideoPerItem: []ratio_setting.VideoResolutionAudioPriceRule{
			{Resolution: "1080p", HasAudio: false, Price: 3},
			{Resolution: "720p", HasAudio: false, Price: 2.05},
		},
		ImageToVideoPerItem: []ratio_setting.VideoResolutionAudioPriceRule{
			{Resolution: "1080p", HasAudio: true, Price: 1},
		},
	}
	tiers := collectVideoFlatTiers(rules)
	if len(tiers) != 3 {
		t.Fatalf("tiers len=%d", len(tiers))
	}
	best, ok := pickMinVideoFlatTier(tiers)
	if !ok {
		t.Fatal("no best")
	}
	if best.RawUSD != 1 || best.Lane != "image_to_video" {
		t.Fatalf("best=%+v", best)
	}
	rows := buildSortedTierRows(tiers, 1)
	if len(rows) != 3 {
		t.Fatalf("rows len=%d", len(rows))
	}
	if rows[0].UsdAfterChannelDiscount != 1 || rows[2].UsdAfterChannelDiscount != 3 {
		t.Fatalf("sort order wrong: %#v", rows)
	}
}

func TestCollectVideoPerSecondTiers_BuildHint(t *testing.T) {
	rules := ratio_setting.VideoPricingRules{
		TextToVideoPerSecond: []ratio_setting.VideoResolutionAudioPriceRule{
			{Resolution: "540p", HasAudio: false, Price: 0.02},
			{Resolution: "720p", HasAudio: false, Price: 0.05},
		},
	}
	sec := collectVideoPerSecondTiers(rules)
	sec = collapsePairedUnifiedAudioTiers(sec)
	sec = normalizeLegacyAllFalseToUnifiedHintTiers(sec)
	if len(sec) != 2 {
		t.Fatalf("per-second tiers len=%d", len(sec))
	}
	for _, ti := range sec {
		if ti.HasAudio != nil {
			t.Fatalf("legacy unified should clear HasAudio, got %+v", ti)
		}
	}
	best, ok := pickMinVideoFlatTier(sec)
	if !ok || best.RawUSD != 0.02 || best.Lane != "text_to_video_per_second" {
		t.Fatalf("best=%+v", best)
	}
}

func TestCollapsePairedUnifiedAudioTiers(t *testing.T) {
	tiers := []videoFlatTier{
		{RawUSD: 0.1, Res: "720p", Lane: "text_to_video_per_second", HasAudio: ptrBool(false)},
		{RawUSD: 0.1, Res: "720p", Lane: "text_to_video_per_second", HasAudio: ptrBool(true)},
	}
	out := collapsePairedUnifiedAudioTiers(tiers)
	if len(out) != 1 {
		t.Fatalf("len=%d %#v", len(out), out)
	}
	if out[0].HasAudio != nil {
		t.Fatal("expected merged unified nil HasAudio")
	}
}

func ptrBool(b bool) *bool { return &b }
