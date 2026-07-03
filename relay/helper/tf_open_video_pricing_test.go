package helper

import (
	"math"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestModelPriceHelperVideo_TokenFactoryOpenUsesPerSecondRules(t *testing.T) {
	gin.SetMode(gin.TestMode)

	prevRules := ratio_setting.VideoPricingRules2JSONString()
	t.Cleanup(func() { _ = ratio_setting.UpdateVideoPricingRulesByJSONString(prevRules) })

	modelName := "tf-open-video-model"
	require.NoError(t, ratio_setting.UpdateVideoPricingRulesByJSONString(
		`{"`+modelName+`":{"text_to_video_per_second":[{"resolution":"720p","has_audio":false,"price":2.5}]}}`,
	))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := relaycommon.TaskSubmitReq{
		Model:    modelName,
		Prompt:   "test",
		Seconds:  "4",
		Metadata: map[string]interface{}{"resolution": "720p"},
	}
	req.Metadata = common.NormalizeTaskVideoMetadata(req.Metadata, req.Size, nil, nil)
	c.Set("task_request", req)

	relayInfo := &relaycommon.RelayInfo{
		OriginModelName: modelName,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType: constant.ChannelTypeTokenFactoryOpen,
		},
	}
	pd, err := ModelPriceHelperVideo(c, relayInfo)
	require.NoError(t, err)
	require.True(t, pd.UsePrice)
	require.Equal(t, float64(0), pd.ModelPrice)
	require.NotNil(t, pd.OtherRatios)
	require.Equal(t, float64(4), pd.OtherRatios["seconds"])
	require.Greater(t, pd.Quota, 0)
}

func TestModelPriceHelperVideo_TokenFactoryOpenUsesChannelPerTokenRules(t *testing.T) {
	gin.SetMode(gin.TestMode)

	prevRules := ratio_setting.VideoPricingRules2JSONString()
	prevChannelRules := ratio_setting.ChannelVideoPricingRules2JSONString()
	prevMemoryCacheEnabled := common.MemoryCacheEnabled
	common.MemoryCacheEnabled = true
	t.Cleanup(func() {
		_ = ratio_setting.UpdateVideoPricingRulesByJSONString(prevRules)
		_ = ratio_setting.UpdateChannelVideoPricingRulesByJSONString(prevChannelRules)
		common.MemoryCacheEnabled = prevMemoryCacheEnabled
	})

	modelName := "tf-open-channel-video-token-model"
	channelID := 60
	const pricePerMillionTokens = 47.3334
	require.NoError(t, ratio_setting.UpdateVideoPricingRulesByJSONString(
		`{"`+modelName+`":{"text_to_video_per_second":[{"resolution":"540p","has_audio":false,"price":0.01}]}}`,
	))
	require.NoError(t, ratio_setting.UpdateChannelVideoPricingRulesByJSONString(
		`{"`+strconv.Itoa(channelID)+`":{"`+modelName+`":{"text_to_video_per_token":[{"resolution":"540p","has_audio":false,"price":`+
			strconv.FormatFloat(pricePerMillionTokens, 'g', -1, 64)+`}]}}}`,
	))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := relaycommon.TaskSubmitReq{
		Model:    modelName,
		Prompt:   "test",
		Duration: 5,
		Size:     "960x540",
	}
	req.Metadata = common.NormalizeTaskVideoMetadata(req.Metadata, req.Size, nil, nil)
	c.Set("task_request", req)

	relayInfo := &relaycommon.RelayInfo{
		OriginModelName: modelName,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId:   channelID,
			ChannelType: constant.ChannelTypeTokenFactoryOpen,
		},
	}
	pd, err := ModelPriceHelperVideo(c, relayInfo)
	require.NoError(t, err)
	require.True(t, pd.UsePrice)
	require.Equal(t, service.VideoRuleUnitPerToken, pd.VideoRuleUnit)
	require.Equal(t, float64(0), pd.ModelPrice)
	require.Equal(t, service.SeedanceTokenPreConsumeTokens, pd.VideoOutputTokens)
	require.InDelta(t, pricePerMillionTokens, pd.VideoChannelRulePrice, 1e-9)
	require.Empty(t, pd.OtherRatios)

	want := int(math.Round(
		(float64(service.SeedanceTokenPreConsumeTokens) / service.VideoTokenPricePerMillion) *
			pricePerMillionTokens * common.QuotaPerUnit * pd.GroupRatioInfo.GroupRatio,
	))
	require.Equal(t, want, pd.Quota)
}
