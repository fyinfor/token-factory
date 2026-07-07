package helper

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestValidateVideoModelPrice_CapabilityMismatch(t *testing.T) {
	gin.SetMode(gin.TestMode)

	prevRules := ratio_setting.VideoPricingRules2JSONString()
	t.Cleanup(func() { _ = ratio_setting.UpdateVideoPricingRulesByJSONString(prevRules) })

	modelName := "happyhorse-1.0-i2v"
	require.NoError(t, ratio_setting.UpdateVideoPricingRulesByJSONString(
		`{"`+modelName+`":{"image_to_video_per_second":[{"resolution":"720p","has_audio":false,"price":4}]}}`,
	))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := relaycommon.TaskSubmitReq{
		Model:    modelName,
		Prompt:   "test",
		Seconds:  "5",
		Metadata: map[string]interface{}{"resolution": "720p"},
	}
	req.Metadata = common.NormalizeTaskVideoMetadata(req.Metadata, req.Size, nil, nil)
	c.Set("task_request", req)

	err := validateVideoModelPrice(c, 0, modelName)
	require.Error(t, err)
	require.Contains(t, err.Error(), "不支持文生视频")
	require.Contains(t, err.Error(), "仅支持图生视频")
	require.Contains(t, err.Error(), "可用分辨率：720p")

	var apiErr *types.TokenFactoryError
	require.True(t, errors.As(err, &apiErr))
	require.Equal(t, types.ErrorCodeModelPriceError, apiErr.GetErrorCode())
}

func TestValidateVideoModelPrice_ResolutionMismatch(t *testing.T) {
	gin.SetMode(gin.TestMode)

	prevRules := ratio_setting.VideoPricingRules2JSONString()
	t.Cleanup(func() { _ = ratio_setting.UpdateVideoPricingRulesByJSONString(prevRules) })

	modelName := "happyhorse-1.0-i2v"
	require.NoError(t, ratio_setting.UpdateVideoPricingRulesByJSONString(
		`{"`+modelName+`":{"image_to_video_per_second":[{"resolution":"720p","has_audio":false,"price":4}]}}`,
	))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := relaycommon.TaskSubmitReq{
		Model:    modelName,
		Prompt:   "test",
		Images:   []string{"https://example.com/a.png"},
		Seconds:  "5",
		Metadata: map[string]interface{}{"resolution": "480p"},
	}
	req.Metadata = common.NormalizeTaskVideoMetadata(req.Metadata, req.Size, nil, nil)
	c.Set("task_request", req)

	err := validateVideoModelPrice(c, 0, modelName)
	require.Error(t, err)
	require.Contains(t, err.Error(), "不支持480p图生视频")
	require.Contains(t, err.Error(), "仅支持图生视频")
	require.Contains(t, err.Error(), "可用分辨率：720p")
}

func TestModelPriceHelperVideo_RejectsTextToVideoWhenOnlyImageToVideoConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)

	prevRules := ratio_setting.VideoPricingRules2JSONString()
	t.Cleanup(func() { _ = ratio_setting.UpdateVideoPricingRulesByJSONString(prevRules) })

	modelName := "happyhorse-1.0-i2v"
	require.NoError(t, ratio_setting.UpdateVideoPricingRulesByJSONString(
		`{"`+modelName+`":{"image_to_video_per_second":[{"resolution":"720p","has_audio":false,"price":4}]}}`,
	))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := relaycommon.TaskSubmitReq{
		Model:    modelName,
		Prompt:   "test",
		Seconds:  "5",
		Metadata: map[string]interface{}{"resolution": "720p"},
	}
	req.Metadata = common.NormalizeTaskVideoMetadata(req.Metadata, req.Size, nil, nil)
	c.Set("task_request", req)

	_, err := ModelPriceHelperVideo(c, &relaycommon.RelayInfo{OriginModelName: modelName})
	require.Error(t, err)
	require.Contains(t, err.Error(), "不支持文生视频")
}
