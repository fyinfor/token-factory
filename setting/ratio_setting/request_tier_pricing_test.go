package ratio_setting

import (
	"testing"

	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/require"
)

func sampleThreeTier() []RequestTierBand {
	return []RequestTierBand{
		{UpTo: 16000, Prices: RequestTierPrices{Input: 1, Output: 4, CacheRead: 0.1, CacheWrite: 0.2}},
		{UpTo: 32000, Prices: RequestTierPrices{Input: 2, Output: 5, CacheRead: 0.1, CacheWrite: 0.2}},
		{UpTo: 0, Prices: RequestTierPrices{Input: 3, Output: 6, CacheRead: 0.1, CacheWrite: 0.2}},
	}
}

func TestFindRequestTierBandIndexBoundary(t *testing.T) {
	tiers := sampleThreeTier()

	cases := []struct {
		tokens   int64
		boundary string
		wantUpTo int64
	}{
		{0, RequestTierBoundaryLt, 16000},
		{15999, RequestTierBoundaryLt, 16000},
		{16000, RequestTierBoundaryLt, 32000},
		{16001, RequestTierBoundaryLt, 32000},
		{32000, RequestTierBoundaryLt, 0},
		{32001, RequestTierBoundaryLt, 0},
		{0, RequestTierBoundaryLte, 16000},
		{15999, RequestTierBoundaryLte, 16000},
		{16000, RequestTierBoundaryLte, 16000},
		{16001, RequestTierBoundaryLte, 32000},
		{32000, RequestTierBoundaryLte, 32000},
		{32001, RequestTierBoundaryLte, 0},
	}
	for _, tc := range cases {
		idx := FindRequestTierBandIndex(tc.tokens, tiers, tc.boundary)
		require.GreaterOrEqual(t, idx, 0, "tokens=%d boundary=%s", tc.tokens, tc.boundary)
		require.Equal(t, tc.wantUpTo, tiers[idx].UpTo, "tokens=%d boundary=%s", tc.tokens, tc.boundary)
	}
}

func TestBuildRequestTierLabelBoundary(t *testing.T) {
	tiers := sampleThreeTier()
	require.Equal(t, "输入token<16000", BuildRequestTierLabel("输入", tiers, 100, RequestTierBoundaryLt))
	require.Equal(t, "输入token≤16000", BuildRequestTierLabel("输入", tiers, 16000, RequestTierBoundaryLte))
	require.Equal(t, "输入token≥32000", BuildRequestTierLabel("输入", tiers, 50000, RequestTierBoundaryLt))
}

func TestMergeLegacyTierSegmentsToPricing(t *testing.T) {
	rule := MergeLegacyTierSegmentsToPricing(
		[]legacySegment{{UpTo: 16000, Ratio: 0.5}, {UpTo: 0, Ratio: 1.5}},
		[]legacySegment{{UpTo: 16000, Ratio: 2}, {UpTo: 0, Ratio: 3}},
		[]legacySegment{{UpTo: 16000, Ratio: 0.05}, {UpTo: 0, Ratio: 0.05}},
		[]legacySegment{{UpTo: 16000, Ratio: 0.1}, {UpTo: 0, Ratio: 0.1}},
	)
	require.Equal(t, RequestTierBoundaryLt, rule.Boundary)
	require.Equal(t, RequestTierDimensionInputTokens, rule.Dimension)
	require.Len(t, rule.Tiers, 2)
	require.InDelta(t, 1.0, rule.Tiers[0].Prices.Input, 1e-9)
	require.InDelta(t, 4.0, rule.Tiers[0].Prices.Output, 1e-9)
	require.InDelta(t, 3.0, rule.Tiers[1].Prices.Input, 1e-9)
}

func TestResolveRequestTierHitUsesSameBandPrices(t *testing.T) {
	modelRequestTierPricingMap.Clear()
	channelModelRequestTierPricingMap.Clear()
	t.Cleanup(func() {
		modelRequestTierPricingMap.Clear()
		channelModelRequestTierPricingMap.Clear()
	})

	rule := normalizeRequestTierPricing(RequestTierPricing{
		Mode:      RequestTierModeProgressive,
		Dimension: RequestTierDimensionInputTokens,
		Boundary:  RequestTierBoundaryLt,
		Tiers:     sampleThreeTier(),
	})
	modelRequestTierPricingMap.AddAll(map[string]RequestTierPricing{"demo-model": rule})

	hit, ok := ResolveRequestTierHit(0, "demo-model", 20000, 100, 0, 1)
	require.True(t, ok)
	require.Equal(t, int64(16000), hit.FromToken)
	require.Equal(t, int64(32000), hit.ToToken)
	require.InDelta(t, 2.0, hit.InputUnitPrice, 1e-9)  // price 2 * 100% * 1
	require.InDelta(t, 5.0, hit.OutputUnitPrice, 1e-9)
	require.InDelta(t, 1.0, hit.EffectiveInput, 1e-9) // 2/2
}

func TestValidateRequestTierPricing(t *testing.T) {
	err := ValidateRequestTierPricing(RequestTierPricing{
		Mode:      RequestTierModeProgressive,
		Dimension: "output_tokens",
		Tiers:     sampleThreeTier(),
	})
	require.Error(t, err)

	err = ValidateRequestTierPricing(normalizeRequestTierPricing(RequestTierPricing{
		Tiers: sampleThreeTier(),
	}))
	require.NoError(t, err)
}

func TestConvertRequestTierPriceToUSD(t *testing.T) {
	require.InDelta(t, 10.0, ConvertRequestTierPriceToUSD(10, RequestTierCurrencyUSD), 1e-9)

	prev := operation_setting.USDExchangeRate
	operation_setting.USDExchangeRate = 7.0
	t.Cleanup(func() { operation_setting.USDExchangeRate = prev })

	require.InDelta(t, 2.0, ConvertRequestTierPriceToUSD(14, RequestTierCurrencyCNY), 1e-9)
	// 与基准货币一致（按 USD 存）时不换算
	require.InDelta(t, 14.0, ConvertRequestTierPriceToUSD(14, RequestTierCurrencyUSD), 1e-9)
}

func TestResolveRequestTierHitConvertsCurrency(t *testing.T) {
	modelRequestTierPricingMap.Clear()
	channelModelRequestTierPricingMap.Clear()
	t.Cleanup(func() {
		modelRequestTierPricingMap.Clear()
		channelModelRequestTierPricingMap.Clear()
	})

	prev := operation_setting.USDExchangeRate
	operation_setting.USDExchangeRate = 7.0
	t.Cleanup(func() { operation_setting.USDExchangeRate = prev })

	rule := normalizeRequestTierPricing(RequestTierPricing{
		Mode:      RequestTierModeProgressive,
		Dimension: RequestTierDimensionInputTokens,
		Boundary:  RequestTierBoundaryLt,
		Currency:  RequestTierCurrencyCNY,
		Tiers: []RequestTierBand{
			{UpTo: 0, Prices: RequestTierPrices{Input: 14, Output: 28}},
		},
	})
	modelRequestTierPricingMap.AddAll(map[string]RequestTierPricing{"cny-model": rule})

	hit, ok := ResolveRequestTierHit(0, "cny-model", 100, 100, 0, 1)
	require.True(t, ok)
	require.InDelta(t, 2.0, hit.InputUnitPrice, 1e-9)  // 14 CNY / 7 = 2 USD
	require.InDelta(t, 4.0, hit.OutputUnitPrice, 1e-9) // 28 / 7 = 4
	require.InDelta(t, 1.0, hit.EffectiveInput, 1e-9)  // 2/2
}

func TestPlatformUsdPerMChannelMarkup(t *testing.T) {
	// channel price=10, global=12, cost=90%, markup=5%, group=1.2
	// ratios: 5 and 6; eff=(5*0.9+6*0.05)*2*1.2 = (4.5+0.3)*2.4 = 11.52
	price := platformUsdPerMFromPrices(10, 12, true, 90, 5, 1.2)
	require.InDelta(t, 11.52, price, 1e-9)
}
