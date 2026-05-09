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

// 与用户提供的 playground JSON 等价（字段语义一致），用于演练 ModelPriceHelperVideo 的估值链路。
func seedancePlaygroundTaskReq(t *testing.T) relaycommon.TaskSubmitReq {
	t.Helper()
	const payload = `{
  "model": "Seedance2.0",
  "prompt": "生成一个小猫嗷嗷叫的视频",
  "n": 1,
  "size": "960x540",
  "fps": 24,
  "duration": 5,
  "motion": 0.6,
  "negative_prompt": "",
  "seed": null,
  "images": []
}`
	var req relaycommon.TaskSubmitReq
	require.NoError(t, common.UnmarshalJsonStr(payload, &req))
	return req
}

// TestSeedancePlaygroundJSON_PerSecondQuota 演示：若全局为 Seedance2.0 配置了「文生视频按秒」规则，
// 则预扣额度 = ceil(duration)×每秒单价(USD)×QuotaPerUnit×分组倍率×渠道折扣。
// 单价需替换为你们控制台真实配置；此处用 0.01 USD/秒便于断言公式。
func TestSeedancePlaygroundJSON_PerSecondQuota(t *testing.T) {
	gin.SetMode(gin.TestMode)

	prevRules := ratio_setting.VideoPricingRules2JSONString()
	defer func() { _ = ratio_setting.UpdateVideoPricingRulesByJSONString(prevRules) }()

	const pricePerSecUSD = 0.01
	cfg := `{"Seedance2.0":{"text_to_video_per_second":[{"resolution":"540p","has_audio":false,"price":` +
		strconv.FormatFloat(pricePerSecUSD, 'g', -1, 64) + `}]}}`
	require.NoError(t, ratio_setting.UpdateVideoPricingRulesByJSONString(cfg))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req := seedancePlaygroundTaskReq(t)
	c.Set("task_request", req)

	relayInfo := &relaycommon.RelayInfo{
		UserGroup:       "test-group",
		UsingGroup:      "default",
		OriginModelName: req.Model,
	}

	pd, err := ModelPriceHelperVideo(c, relayInfo)
	require.NoError(t, err)

	ctx := estimateVideoRequestContext(c)
	require.Equal(t, videoBillingModeTextToVideo, ctx.Mode, "images 为空应为文生视频")
	require.Equal(t, 960, ctx.Width)
	require.Equal(t, 540, ctx.Height)
	require.Equal(t, 5, ctx.DurationSec)

	sec := 5
	if ctx.DurationSec > 0 {
		sec = ctx.DurationSec
	}
	want := int(common.QuotaPerUnit * float64(sec) * pricePerSecUSD * pd.GroupRatioInfo.GroupRatio)
	require.Equal(t, want, pd.Quota,
		"应与 ceil(秒)×单价×QuotaPerUnit×group 一致（渠道 id=0 无折扣）")

	t.Logf("estimate ctx: mode=%s WxH=%dx%d durationSec=%d fps(estimate)=%d inputTextTokens≈%d",
		ctx.Mode, ctx.Width, ctx.Height, ctx.DurationSec, ctx.FPS, ctx.InputTextTokens)
	t.Logf("preconsume quota=%d (QuotaPerUnit=%.0f groupRatio=%.4f)",
		pd.Quota, common.QuotaPerUnit, pd.GroupRatioInfo.GroupRatio)
}
