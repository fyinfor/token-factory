package model

import (
	"testing"

	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

func TestBuildSortedImagePerImageTierRows_IncludesOfficialPrice(t *testing.T) {
	channelRules := ratio_setting.ImagePricingRules{
		TextToImagePerImage: []ratio_setting.ImageResolutionPerImageRule{
			{Resolution: "1024x1024", ImagePrice: 0.01},
		},
	}
	globalRules := ratio_setting.ImagePricingRules{
		TextToImagePerImage: []ratio_setting.ImageResolutionPerImageRule{
			{Resolution: "1024x1024", ImagePrice: 0.02},
		},
	}

	tiers := collectImagePerImageTiers(channelRules)
	rows := buildSortedImagePerImageTierRows(tiers, globalRules, 100, 100)
	if len(rows) != 1 {
		t.Fatalf("rows=%d", len(rows))
	}
	if rows[0].UsdAfterChannelDiscount != 0.03 {
		t.Fatalf("usd=%v want 0.03", rows[0].UsdAfterChannelDiscount)
	}
	if rows[0].UsdOfficial != 0.02 {
		t.Fatalf("official=%v want 0.02", rows[0].UsdOfficial)
	}
}
