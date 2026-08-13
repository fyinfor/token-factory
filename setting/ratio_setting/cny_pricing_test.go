package ratio_setting

import (
	"testing"

	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/require"
)

func TestCNYUnitPricesToInternalRatios(t *testing.T) {
	prev := operation_setting.USDExchangeRate
	operation_setting.USDExchangeRate = 6.82
	t.Cleanup(func() {
		operation_setting.USDExchangeRate = prev
	})

	ratios := CNYUnitPricesToInternalRatios(CNYUnitPrices{
		Input:     1,
		Output:    2,
		CacheRead: 0.02,
	})
	require.InDelta(t, (1.0/6.82)/2.0, ratios.ModelRatio, 1e-12)
	require.InDelta(t, 2.0, ratios.CompletionRatio, 1e-12)
	require.InDelta(t, 0.02, ratios.CacheRatio, 1e-12)
	require.Equal(t, 0.0, ratios.CreateCacheRatio)
}

func TestResolveCNYPricingChannelOverridesGlobal(t *testing.T) {
	modelCNYPricingMap.Clear()
	channelModelCNYPricingMap.Clear()
	t.Cleanup(func() {
		modelCNYPricingMap.Clear()
		channelModelCNYPricingMap.Clear()
	})

	modelCNYPricingMap.AddAll(map[string]CNYUnitPrices{
		"deepseek-v4-flash": {Input: 1, Output: 2, CacheRead: 0.02},
	})
	channelModelCNYPricingMap.AddAll(map[string]map[string]CNYUnitPrices{
		"82": {
			"deepseek-v4-flash": {Input: 1.2, Output: 2.4, CacheRead: 0.024},
		},
	})

	global, ok := GetModelCNYPricing("deepseek-v4-flash")
	require.True(t, ok)
	require.Equal(t, 1.0, global.Input)

	resolved, ok := ResolveCNYPricing(82, "deepseek-v4-flash")
	require.True(t, ok)
	require.Equal(t, 1.2, resolved.Input)

	fallback, ok := ResolveCNYPricing(99, "deepseek-v4-flash")
	require.True(t, ok)
	require.Equal(t, 1.0, fallback.Input)
}
