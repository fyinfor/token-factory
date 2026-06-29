package helper

import (
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestModelPriceHelperVideo_PerTokenPreConsumeDividesBy1M(t *testing.T) {
	gin.SetMode(gin.TestMode)

	prevRules := ratio_setting.VideoPricingRules2JSONString()
	t.Cleanup(func() { _ = ratio_setting.UpdateVideoPricingRulesByJSONString(prevRules) })

	const price480PerM = 0.15
	const price720PerM = 0.31
	modelName := "seedance-per-token-precharge"
	cfg := `{"` + modelName + `":{"text_to_video_per_token":[` +
		`{"resolution":"480p","has_audio":false,"price":` + strconv.FormatFloat(price480PerM, 'g', -1, 64) + `},` +
		`{"resolution":"720p","has_audio":false,"price":` + strconv.FormatFloat(price720PerM, 'g', -1, 64) + `}` +
		`]}}`
	require.NoError(t, ratio_setting.UpdateVideoPricingRulesByJSONString(cfg))

	run := func(t *testing.T, resolution string, pricePerM float64) int {
		t.Helper()
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		req := relaycommon.TaskSubmitReq{
			Model:  modelName,
			Prompt: "test",
			Metadata: map[string]interface{}{
				"resolution": resolution,
				"ratio":      "16:9",
			},
		}
		req.Metadata = common.NormalizeTaskVideoMetadata(req.Metadata, req.Size, nil, nil)
		c.Set("task_request", req)

		relayInfo := &relaycommon.RelayInfo{OriginModelName: modelName}
		pd, err := ModelPriceHelperVideo(c, relayInfo)
		require.NoError(t, err)
		require.Equal(t, "per_token", pd.VideoRuleUnit)

		want := int(
			(float64(50000) / 1_000_000) * pricePerM * common.QuotaPerUnit * pd.GroupRatioInfo.GroupRatio,
		)
		require.InDelta(t, want, pd.Quota, 1, "precharge quota should be tokens÷1M×$/1M×QuotaPerUnit")
		return pd.Quota
	}

	q480 := run(t, "480p", price480PerM)
	q720 := run(t, "720p", price720PerM)
	require.Less(t, q480, q720, "720p /1M token price should preconsume more quota than 480p")
}
