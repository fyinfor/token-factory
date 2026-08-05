package helper

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestModelPriceHelperASR_SetsGlobalModelPriceForMarkupSlice(t *testing.T) {
	gin.SetMode(gin.TestMode)
	require.NoError(t, ratio_setting.UpdateASRPriceByJSONString(`{"test-asr-model":0.0001}`))
	t.Cleanup(func() {
		_ = ratio_setting.UpdateASRPriceByJSONString(`{}`)
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/audio/transcriptions", nil)

	info := &relaycommon.RelayInfo{
		OriginModelName: "test-asr-model",
		ChannelMeta:     &relaycommon.ChannelMeta{ChannelId: 0},
	}

	priceData, err := ModelPriceHelperASR(c, info, 10)
	require.NoError(t, err)
	require.True(t, priceData.UsePrice)
	require.Equal(t, 0.0001, priceData.ModelPrice)
	require.Equal(t, priceData.ModelPrice, priceData.GlobalModelPrice,
		"ASR 须写入 GlobalModelPrice，否则加价折扣与利润分成切片恒为 0")

	// 成本 100% + 加价 50%：有效价 = channel*1 + global*0.5 = 1.5 * unit
	effWithMarkup := model.EffectiveRuleUnitPrice(priceData.ModelPrice, priceData.GlobalModelPrice, 100, 50)
	effCostOnly := model.EffectiveRuleUnitPrice(priceData.ModelPrice, priceData.GlobalModelPrice, 100, 0)
	require.Greater(t, effWithMarkup, effCostOnly)

	seconds := 10.0
	quotaWithMarkup := int(effWithMarkup * common.QuotaPerUnit * seconds)
	quotaCostOnly := int(effCostOnly * common.QuotaPerUnit * seconds)
	slice := quotaWithMarkup - quotaCostOnly
	require.Greater(t, slice, 0, "有加价时应产生分润利润切片")
}
