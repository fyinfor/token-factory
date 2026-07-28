package helper

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
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

func TestMatchPerImageRulesByPixels_Square1080MapsTo1080pTier(t *testing.T) {
	ctx := imageEstimateContext{
		Mode:   imageBillingModeTextToImage,
		Width:  1080,
		Height: 1080,
		Count:  1,
	}
	// Landscape 720p has closer total pixels to 1080x1080 than landscape 1080p,
	// but short-side tier label must win so square 1080 maps to 1080p.
	rules := []ratio_setting.ImageResolutionPerImageRule{
		{Resolution: "1280x720", ImagePrice: 0.01},
		{Resolution: "1920x1080", ImagePrice: 0.02},
		{Resolution: "2560x1440", ImagePrice: 0.03},
		{Resolution: "3840x2160", ImagePrice: 0.04},
	}

	match, ok := matchPerImageRulesByPixels(ctx, rules, 0.35, 0, false)

	require.True(t, ok)
	require.NotNil(t, match)
	assert.Equal(t, "1920x1080", match.Resolution)
	assert.Equal(t, 0.02, match.Price)
	assert.False(t, match.CappedToMaxTier)
}

func TestMatchPerImageRulesByPixels_Square1080MatchesLabelOnlyTiers(t *testing.T) {
	ctx := imageEstimateContext{
		Mode:   imageBillingModeTextToImage,
		Width:  1080,
		Height: 1080,
		Count:  1,
	}
	rules := []ratio_setting.ImageResolutionPerImageRule{
		{Resolution: "1920x1080", ImagePrice: 0.02},
		{Resolution: "2560x1440", ImagePrice: 0.03},
		{Resolution: "3840x2160", ImagePrice: 0.04},
	}

	match, ok := matchPerImageRulesByPixels(ctx, rules, 0.35, 0, false)

	require.True(t, ok)
	require.NotNil(t, match)
	assert.Equal(t, "1920x1080", match.Resolution)
	assert.Equal(t, 0.02, match.Price)
}

func TestTryModelPriceHelperImage_Square1080Uses1080pPrice(t *testing.T) {
	gin.SetMode(gin.TestMode)

	prevRules := ratio_setting.ImagePricingRules2JSONString()
	t.Cleanup(func() { _ = ratio_setting.UpdateImagePricingRulesByJSONString(prevRules) })

	modelName := "Kling-3.0-image"
	require.NoError(t, ratio_setting.UpdateImagePricingRulesByJSONString(
		`{"`+modelName+`":{"text_to_image_per_image":[`+
			`{"resolution":"1920x1080","image_price":0.02},`+
			`{"resolution":"2560x1440","image_price":0.03},`+
			`{"resolution":"3840x2160","image_price":0.04}`+
			`]}}`,
	))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	size := "1080x1080"
	info := &relaycommon.RelayInfo{
		OriginModelName: modelName,
		Request: &dto.ImageRequest{
			Model: modelName,
			Size:  size,
		},
	}

	pd, ok, err := TryModelPriceHelperImage(c, info)
	require.NoError(t, err)
	require.True(t, ok)
	require.NotNil(t, info.ImageBilling)
	assert.Equal(t, "1920x1080", info.ImageBilling.RuleRes)
	assert.InDelta(t, 0.02, info.ImageBilling.UsdPerImage, 1e-9)
	assert.True(t, pd.UsePrice)
}

func TestNewModelPriceFriendlyError_AvoidsContradictoryCapabilityMessage(t *testing.T) {
	err := newModelPriceFriendlyError(
		"图片模型",
		"Kling-3.0-image",
		"文生图",
		[]string{"文生图"},
		[]string{"1080p", "2K", "4K"},
	)
	require.Error(t, err)
	require.NotContains(t, err.Error(), "不支持文生图，仅支持文生图")
	require.Contains(t, err.Error(), "无法匹配当前分辨率计费档位")
	require.Contains(t, err.Error(), "可用分辨率：1080p、2K、4K")

	var apiErr *types.TokenFactoryError
	require.True(t, errors.As(err, &apiErr))
	require.Equal(t, types.ErrorCodeModelPriceError, apiErr.GetErrorCode())
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
