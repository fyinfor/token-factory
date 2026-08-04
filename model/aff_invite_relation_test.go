package model

import (
	"testing"

	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

func TestMaxInviteeMarkupDiscountRateUsesChannelCostDiscount(t *testing.T) {
	tests := []struct {
		name        string
		costPercent float64
		want        float64
	}{
		{name: "discounted cost leaves room", costPercent: 40, want: 160},
		{name: "full cost still leaves markup room", costPercent: 100, want: 100},
		{name: "negative cost is clamped", costPercent: -10, want: 200},
		{name: "over official cost leaves reduced room", costPercent: 130, want: 70},
		{name: "over max sale rate is clamped", costPercent: 230, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MaxInviteeMarkupDiscountRate(InviteeModelMarkupDiscountRateItem{
				ChannelPriceDiscountPercent: tt.costPercent,
			})
			if got != tt.want {
				t.Fatalf("MaxInviteeMarkupDiscountRate(%v) = %v, want %v", tt.costPercent, got, tt.want)
			}
		})
	}
}

func TestOfficialVideoPricingRulesForExportKeepsExactModelNames(t *testing.T) {
	previous := ratio_setting.VideoPricingRules2JSONString()
	t.Cleanup(func() {
		if err := ratio_setting.UpdateVideoPricingRulesByJSONString(previous); err != nil {
			t.Fatalf("restore video pricing rules: %v", err)
		}
	})

	const rulesJSON = `{
		"Seedance 2.0": {
			"text_to_video_per_token": [
				{"resolution":"720p","has_audio":false,"price":6.745}
			]
		},
		"Seedance2.0": {
			"text_to_video_per_item": [
				{"resolution":"1080p","has_audio":true,"price":9.5}
			]
		}
	}`
	if err := ratio_setting.UpdateVideoPricingRulesByJSONString(rulesJSON); err != nil {
		t.Fatalf("set video pricing rules: %v", err)
	}

	spaced := officialVideoPricingRulesForExport("Seedance 2.0")
	if spaced == nil || len(spaced.TextToVideoPerToken) != 1 || spaced.TextToVideoPerToken[0].Price != 6.745 {
		t.Fatalf("Seedance 2.0 export rules = %#v", spaced)
	}
	if ratio_setting.HasUsableVideoPerVideoRules(*spaced) {
		t.Fatalf("Seedance 2.0 unexpectedly merged per-item rules: %#v", spaced)
	}

	compact := officialVideoPricingRulesForExport("Seedance2.0")
	if compact == nil || len(compact.TextToVideoPerItem) != 1 || compact.TextToVideoPerItem[0].Price != 9.5 {
		t.Fatalf("Seedance2.0 export rules = %#v", compact)
	}
	if ratio_setting.HasUsableVideoPerTokenRules(*compact) {
		t.Fatalf("Seedance2.0 unexpectedly merged per-token rules: %#v", compact)
	}
}
