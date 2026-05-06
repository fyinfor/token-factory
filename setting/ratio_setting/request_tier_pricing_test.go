package ratio_setting

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestApplyRequestTierPricingDecimalProgressive(t *testing.T) {
	rule := RequestTierPricingRule{
		Mode: RequestTierModeProgressive,
		Input: []RequestTierSegment{
			{UpTo: 1000, Ratio: 1},
			{UpTo: 2000, Ratio: 0.8},
			{UpTo: 0, Ratio: 0.5},
		},
		Output: []RequestTierSegment{
			{UpTo: 500, Ratio: 2},
			{UpTo: 0, Ratio: 1.5},
		},
		CacheRead: []RequestTierSegment{
			{UpTo: 0, Ratio: 0.1},
		},
		CacheWrite: []RequestTierSegment{
			{UpTo: 0, Ratio: 1.25},
		},
	}

	input, output, cacheRead, cacheWrite, breakdown := ApplyRequestTierPricingDecimal(
		rule,
		decimal.NewFromInt(2500),
		decimal.NewFromInt(800),
		decimal.NewFromInt(300),
		decimal.NewFromInt(400),
	)

	require.True(t, decimal.NewFromInt(2050).Equal(input))
	require.True(t, decimal.NewFromInt(1450).Equal(output))
	require.True(t, decimal.NewFromInt(30).Equal(cacheRead))
	require.True(t, decimal.NewFromInt(500).Equal(cacheWrite))
	require.Len(t, breakdown.Details["input"], 3)
	require.Len(t, breakdown.Details["output"], 2)
}
