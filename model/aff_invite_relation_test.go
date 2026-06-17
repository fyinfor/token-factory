package model

import "testing"

func TestMaxInviteeMarkupDiscountRateUsesChannelCostDiscount(t *testing.T) {
	tests := []struct {
		name        string
		costPercent float64
		want        float64
	}{
		{name: "discounted cost leaves room", costPercent: 40, want: 60},
		{name: "full cost leaves no room", costPercent: 100, want: 0},
		{name: "negative cost is clamped", costPercent: -10, want: 100},
		{name: "over official cost is clamped", costPercent: 130, want: 0},
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
