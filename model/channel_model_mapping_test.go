package model

import (
	"testing"

	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/stretchr/testify/require"
)

func TestApplyChannelModelMapping(t *testing.T) {
	require.Equal(t, "kk", ApplyChannelModelMapping("", "kk"))
	require.Equal(t, "kimi-k2.5", ApplyChannelModelMapping(`{"kk":"kimi-k2.5"}`, "kk"))
	require.Equal(t, "kimi-k2.5", ApplyChannelModelMapping(`{"kk":"mid","mid":"kimi-k2.5"}`, "kk"))
}

func TestResolvePricingModelNameAliasWinsAndFallback(t *testing.T) {
	prevPrice := ratio_setting.ModelPrice2JSONString()
	prevRatio := ratio_setting.ModelRatio2JSONString()
	t.Cleanup(func() {
		_ = ratio_setting.UpdateModelPriceByJSONString(prevPrice)
		_ = ratio_setting.UpdateModelRatioByJSONString(prevRatio)
	})

	require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{"kimi-k2.5":0.01}`))
	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{}`))

	mapping := `{"kk":"kimi-k2.5"}`
	require.Equal(t, "kimi-k2.5", ResolvePricingModelName(mapping, "kk"))

	require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{"kk":0.02,"kimi-k2.5":0.01}`))
	require.Equal(t, "kk", ResolvePricingModelName(mapping, "kk"))
}

func TestBuildAliasCanonicalAndDisplayPricing(t *testing.T) {
	prevPrice := ratio_setting.ModelPrice2JSONString()
	t.Cleanup(func() {
		_ = ratio_setting.UpdateModelPriceByJSONString(prevPrice)
		setEnabledAliasCanonicalIndex(nil)
	})
	require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{"kimi-k2.5":0.01}`))

	metas := []ChannelPricingMeta{
		{
			ChannelID:    1016,
			Models:       "kk,kimi-k2.5",
			ModelMapping: `{"kk":"kimi-k2.5"}`,
		},
	}
	index := buildAliasCanonicalIndexFromMeta(metas)
	require.Equal(t, "kimi-k2.5", index["kk"])
	setEnabledAliasCanonicalIndex(index)

	require.True(t, ModelHasDisplayConfiguredPricing("kk"))
	require.Equal(t, "kimi-k2.5", ResolveDisplayPricingModelName("kk"))
	require.Equal(t, "kimi-k2.5", LookupCachedAliasCanonical("kk"))
}
