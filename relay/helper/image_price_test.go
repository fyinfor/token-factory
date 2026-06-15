package helper

import (
	"testing"

	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchPerImageRulesByPixels_CapsAboveHighestTier(t *testing.T) {
	ctx := imageEstimateContext{
		Mode:   imageBillingModeTextToImage,
		Width:  4096,
		Height: 4096,
		Count:  1,
	}
	rules := []ratio_setting.ImageResolutionPerImageRule{
		{Resolution: "1024x1024", ImagePrice: 1},
		{Resolution: "2048x2048", ImagePrice: 2},
	}

	match, ok := matchPerImageRulesByPixels(ctx, rules, 0.35, 0.1, true)

	require.True(t, ok)
	require.NotNil(t, match)
	assert.Equal(t, 2.0, match.Price)
	assert.Equal(t, "2048x2048", match.Resolution)
	assert.Equal(t, 2048, match.RuleWidth)
	assert.Equal(t, 2048, match.RuleHeight)
	assert.True(t, match.CappedToMaxTier)
}

func TestMatchFlatPerImageUSDRules_CapsWithinCurrentMode(t *testing.T) {
	ctx := imageEstimateContext{
		Mode:   imageBillingModeImageToImage,
		Width:  4096,
		Height: 4096,
		Count:  1,
	}
	rules := ratio_setting.ImagePricingRules{
		TextToImagePerImage: []ratio_setting.ImageResolutionPerImageRule{
			{Resolution: "8192x8192", ImagePrice: 99},
		},
		ImageToImagePerImage: []ratio_setting.ImageResolutionPerImageRule{
			{Resolution: "1024x1024", ImagePrice: 1},
			{Resolution: "2048x2048", ImagePrice: 2},
		},
	}

	price, ok, match := matchFlatPerImageUSDRules(ctx, rules, true, 0.1, true)

	require.True(t, ok)
	require.NotNil(t, match)
	assert.Equal(t, 2.0, price)
	assert.Equal(t, "2048x2048", match.Resolution)
	assert.True(t, match.CappedToMaxTier)
}
