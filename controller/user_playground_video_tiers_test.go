package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

func TestAppendPlaygroundVideoPricingRuleTiersIncludesConfiguredPerItemResolutions(t *testing.T) {
	rules := ratio_setting.VideoPricingRules{
		ImageToVideoPerItem: []ratio_setting.VideoResolutionAudioPriceRule{
			{Resolution: "854x480", HasAudio: false, Price: 2},
			{Resolution: "854x480", HasAudio: true, Price: 2},
			{Resolution: "960x540", HasAudio: false, Price: 3},
			{Resolution: "960x540", HasAudio: true, Price: 3},
			{Resolution: "1280x720", HasAudio: false, Price: 4},
			{Resolution: "1280x720", HasAudio: true, Price: 4},
			{Resolution: "1920x1080", HasAudio: false, Price: 5},
			{Resolution: "1920x1080", HasAudio: true, Price: 5},
			{Resolution: "2560x1440", HasAudio: false, Price: 6},
			{Resolution: "2560x1440", HasAudio: true, Price: 6},
			{Resolution: "3840x2160", HasAudio: false, Price: 7},
			{Resolution: "3840x2160", HasAudio: true, Price: 7},
		},
	}
	seen := make(map[string]struct{})
	out := make([]playgroundVideoPricingTier, 0)
	appendPlaygroundVideoPricingRuleTiers(&out, seen, rules)

	want := []string{
		"854x480",
		"960x540",
		"1280x720",
		"1920x1080",
		"2560x1440",
		"3840x2160",
	}
	if len(out) != len(want) {
		t.Fatalf("tier count = %d, want %d: %#v", len(out), len(want), out)
	}
	for i := range want {
		if out[i].Resolution != want[i] {
			t.Fatalf("tier[%d].Resolution = %q, want %q", i, out[i].Resolution, want[i])
		}
		if out[i].Lane != "image_to_video" {
			t.Fatalf("tier[%d].Lane = %q, want image_to_video", i, out[i].Lane)
		}
		if out[i].BillingMode != "per_item" {
			t.Fatalf("tier[%d].BillingMode = %q, want per_item", i, out[i].BillingMode)
		}
	}
}

func TestAppendPlaygroundImagePricingRuleTiersIncludesConfiguredResolutions(t *testing.T) {
	rules := ratio_setting.ImagePricingRules{
		TextToImagePerImage: []ratio_setting.ImageResolutionPerImageRule{
			{Resolution: "854x480", ImagePrice: 0.01},
			{Resolution: "1280x720", ImagePrice: 0.02},
			{Resolution: "1920x1080", ImagePrice: 0},
		},
		ImageToImagePerImage: []ratio_setting.ImageResolutionPerImageRule{
			{Resolution: "1280x720", ImagePrice: 0.03},
			{Resolution: "2560x1440", ImagePrice: 0.04},
		},
	}
	seen := make(map[string]struct{})
	out := make([]playgroundImagePricingTier, 0)
	appendPlaygroundImagePricingRuleTiers(&out, seen, rules)

	want := []playgroundImagePricingTier{
		{Resolution: "854x480", Lane: "text_to_image"},
		{Resolution: "1280x720", Lane: "text_to_image"},
		{Resolution: "1280x720", Lane: "image_to_image"},
		{Resolution: "2560x1440", Lane: "image_to_image"},
	}
	if len(out) != len(want) {
		t.Fatalf("tier count = %d, want %d: %#v", len(out), len(want), out)
	}
	for i := range want {
		if out[i] != want[i] {
			t.Fatalf("tier[%d] = %#v, want %#v", i, out[i], want[i])
		}
	}
}
