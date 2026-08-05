package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// TestASRProfitShareAmounts_WithGlobalModelPrice 验证 ASR 在写入 GlobalModelPrice 后，
// 与文本模型相同：利润金额 = 用户消耗 − 成本价，收益 = 利润 × 分润比例。
func TestASRProfitShareAmounts_WithGlobalModelPrice(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(nil)

	const (
		unitPrice = 0.0001 // USD/秒
		seconds   = 10.0
		costDisc  = 100.0
		markup    = 50.0
		bps       = 2500 // 25%
	)

	relayInfo := &relaycommon.RelayInfo{
		OriginModelName: "test-asr",
		PriceData: types.PriceData{
			UsePrice:              true,
			ModelPrice:            unitPrice,
			GlobalModelPrice:      unitPrice,
			CostDiscountPercent:   costDisc,
			MarkupDiscountPercent: markup,
			GroupRatioInfo:        types.GroupRatioInfo{GroupRatio: 1},
			OtherRatios:           map[string]float64{"seconds": seconds},
		},
	}
	usage := &dto.Usage{
		PromptTokens: int(seconds),
		TotalTokens:  int(seconds),
	}

	summary := calculateTextQuotaSummary(ctx, relayInfo, usage)
	base := textQuotaSummaryWithMarkupOverride(ctx, relayInfo, usage, 0)

	require.Greater(t, summary.Quota, base.Quota, "加价后用户消耗应高于成本价")

	slice, reward, shouldRecord := textProfitShareAmounts(summary.Quota, base.Quota, bps, markup)
	require.True(t, shouldRecord)
	require.Equal(t, summary.Quota-base.Quota, slice)
	require.Equal(t, int(int64(slice)*int64(bps)/10000), reward)
	require.Greater(t, reward, 0)

	// 与 EffectiveRuleUnitPrice 手算对齐
	effW := model.EffectiveRuleUnitPrice(unitPrice, unitPrice, costDisc, markup)
	eff0 := model.EffectiveRuleUnitPrice(unitPrice, unitPrice, costDisc, 0)
	wantW := int(effW * common.QuotaPerUnit * seconds)
	want0 := int(eff0 * common.QuotaPerUnit * seconds)
	require.Equal(t, wantW, summary.Quota)
	require.Equal(t, want0, base.Quota)
}
