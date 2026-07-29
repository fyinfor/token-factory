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

func TestMatchPerImageRulesByPixels_AiDrawingShortSideTiers(t *testing.T) {
	rules := []ratio_setting.ImageResolutionPerImageRule{
		{Resolution: "1K", ImagePrice: 0.01},
		{Resolution: "2K", ImagePrice: 0.02},
		{Resolution: "4K", ImagePrice: 0.04},
	}

	cases := []struct {
		name   string
		w, h   int
		wantRes string
		wantPrice float64
	}{
		{"1024x1536 is 1080p", 1024, 1536, "1K", 0.01},
		{"1536x2048 is 2K", 1536, 2048, "2K", 0.02},
		{"2160x3840 is 4K", 2160, 3840, "4K", 0.04},
		{"1024x1024 is 1080p", 1024, 1024, "1K", 0.01},
		{"1080x1080 is 2K", 1080, 1080, "2K", 0.02},
		{"1K alias matches 1080p tier", 800, 800, "1K", 0.01},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := imageEstimateContext{
				Mode:   imageBillingModeTextToImage,
				Width:  tc.w,
				Height: tc.h,
				Count:  1,
			}
			match, ok := matchPerImageRulesByPixels(ctx, rules, 0.35, 0, false)
			require.True(t, ok)
			require.NotNil(t, match)
			assert.Equal(t, tc.wantRes, match.Resolution)
			assert.Equal(t, tc.wantPrice, match.Price)
		})
	}
}

func TestMatchPerImageRulesByPixels_OneKAliasMatches1080pRule(t *testing.T) {
	ctx := imageEstimateContext{
		Mode:   imageBillingModeTextToImage,
		Width:  1024,
		Height: 1024,
		Count:  1,
	}
	rules := []ratio_setting.ImageResolutionPerImageRule{
		{Resolution: "1080p", ImagePrice: 0.01},
		{Resolution: "2K", ImagePrice: 0.02},
		{Resolution: "4K", ImagePrice: 0.04},
	}

	match, ok := matchPerImageRulesByPixels(ctx, rules, 0.35, 0, false)

	require.True(t, ok)
	require.NotNil(t, match)
	assert.Equal(t, "1080p", match.Resolution)
	assert.Equal(t, 0.01, match.Price)
}

func TestMatchPerImageRulesByPixels_Classic1080pPixelsMatchLowTier(t *testing.T) {
	ctx := imageEstimateContext{
		Mode:   imageBillingModeTextToImage,
		Width:  1024,
		Height: 1024,
		Count:  1,
	}
	// 定价 UI 将 1080p 存为 1920x1080，应与短边≤1024 输出同属 1080p 档。
	rules := []ratio_setting.ImageResolutionPerImageRule{
		{Resolution: "1920x1080", ImagePrice: 0.01},
		{Resolution: "2560x1440", ImagePrice: 0.02},
		{Resolution: "3840x2160", ImagePrice: 0.04},
	}

	match, ok := matchPerImageRulesByPixels(ctx, rules, 0.35, 0, false)

	require.True(t, ok)
	require.NotNil(t, match)
	assert.Equal(t, "1920x1080", match.Resolution)
	assert.Equal(t, 0.01, match.Price)
}

func TestMatchPerImageRulesByPixels_Square1080MapsTo2KTier(t *testing.T) {
	ctx := imageEstimateContext{
		Mode:   imageBillingModeTextToImage,
		Width:  1080,
		Height: 1080,
		Count:  1,
	}
	// 1080 短边属于 2K；1920x1080 现归一为 1080p 档，故应命中 2560x1440。
	rules := []ratio_setting.ImageResolutionPerImageRule{
		{Resolution: "1280x720", ImagePrice: 0.01},
		{Resolution: "1920x1080", ImagePrice: 0.02},
		{Resolution: "2560x1440", ImagePrice: 0.03},
		{Resolution: "3840x2160", ImagePrice: 0.04},
	}

	match, ok := matchPerImageRulesByPixels(ctx, rules, 0.35, 0, false)

	require.True(t, ok)
	require.NotNil(t, match)
	assert.Equal(t, "2560x1440", match.Resolution)
	assert.Equal(t, 0.03, match.Price)
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
	assert.Equal(t, "2560x1440", match.Resolution)
	assert.Equal(t, 0.03, match.Price)
}

func TestTryModelPriceHelperImage_Square1080Uses2KPrice(t *testing.T) {
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
	assert.Equal(t, "2560x1440", info.ImageBilling.RuleRes)
	assert.InDelta(t, 0.03, info.ImageBilling.UsdPerImage, 1e-9)
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
