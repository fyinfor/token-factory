package model

import "testing"

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
