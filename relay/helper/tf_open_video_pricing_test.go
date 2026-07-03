package helper

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
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
		`{"` + modelName + `":{"text_to_video_per_second":[{"resolution":"720p","has_audio":false,"price":2.5}]}}`,
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
