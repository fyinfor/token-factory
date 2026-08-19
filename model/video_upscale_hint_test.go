package model

import (
	"math"
	"testing"

	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

func almostEqualUSD(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

func TestMergeVideoUpscaleTiers_DedupeByTargetAndChannelOverride(t *testing.T) {
	global := []ratio_setting.VideoUpscalePriceRule{
		{Resolution: "720p", SourceResolution: "480p", Price: 0.02},
		{Resolution: "1080p", SourceResolution: "720p", Price: 0.08},
		{Resolution: "2560x1440", SourceResolution: "1080p", Price: 0.15},
		{Resolution: "720p", SourceResolution: "360p", Price: 0.03},
	}
	channel := []ratio_setting.VideoUpscalePriceRule{
		{Resolution: "1280x720", SourceResolution: "480p", Price: 0.025},
	}
	rows := mergeVideoUpscaleTiers(channel, global, 100, 0)
	if len(rows) != 3 {
		t.Fatalf("rows=%d want 3: %+v", len(rows), rows)
	}
	if rows[0].Resolution != "720p" || !almostEqualUSD(rows[0].UsdAfterChannelDiscount, 0.025) {
		t.Fatalf("720p row=%+v want channel 0.025", rows[0])
	}
	if rows[1].Resolution != "1080p" || !almostEqualUSD(rows[1].UsdAfterChannelDiscount, 0.08) {
		t.Fatalf("1080p row=%+v", rows[1])
	}
	if rows[2].Resolution != "2K" || !almostEqualUSD(rows[2].UsdAfterChannelDiscount, 0.15) {
		t.Fatalf("2K row=%+v", rows[2])
	}
}

func TestMergeVideoUpscaleTiers_AppliesCostAndMarkup(t *testing.T) {
	rows := mergeVideoUpscaleTiers(nil, []ratio_setting.VideoUpscalePriceRule{
		{Resolution: "720p", Price: 0.02},
	}, 80, 20)
	if len(rows) != 1 {
		t.Fatalf("rows=%d", len(rows))
	}
	// 0.02*80% + 0.02*20% = 0.02
	if !almostEqualUSD(rows[0].UsdAfterChannelDiscount, 0.02) {
		t.Fatalf("usd=%v want 0.02", rows[0].UsdAfterChannelDiscount)
	}
	rows = mergeVideoUpscaleTiers(nil, []ratio_setting.VideoUpscalePriceRule{
		{Resolution: "720p", Price: 0.02},
	}, 100, 50)
	if len(rows) != 1 || !almostEqualUSD(rows[0].UsdAfterChannelDiscount, 0.03) {
		t.Fatalf("usd=%v want 0.03", rows)
	}
}

func TestBuildVideoFlatClipHint_AttachesUpscaleOnly(t *testing.T) {
	previous := ratio_setting.VideoPricingRules2JSONString()
	t.Cleanup(func() {
		if err := ratio_setting.UpdateVideoPricingRulesByJSONString(previous); err != nil {
			t.Errorf("restore VideoPricingRules: %v", err)
		}
	})
	modelName := "upscale-hint-only-model"
	cfg := `{"` + modelName + `":{"video_upscale_per_second":[{"resolution":"720p","source_resolution":"480p","price":0.02},{"resolution":"1080p","source_resolution":"720p","price":0.08}]}}`
	if err := ratio_setting.UpdateVideoPricingRulesByJSONString(cfg); err != nil {
		t.Fatalf("update rules: %v", err)
	}
	hint := BuildVideoFlatClipHint(0, modelName, 100, 0)
	if hint == nil || len(hint.UpscaleTiers) != 2 {
		t.Fatalf("hint=%+v", hint)
	}
	if hint.TierCount != 0 || len(hint.Tiers) != 0 {
		t.Fatalf("video tiers should be empty: %+v", hint)
	}
	if hint.UpscaleTiers[0].Resolution != "720p" || !almostEqualUSD(hint.UpscaleTiers[0].UsdAfterChannelDiscount, 0.02) {
		t.Fatalf("first=%+v", hint.UpscaleTiers[0])
	}
}
