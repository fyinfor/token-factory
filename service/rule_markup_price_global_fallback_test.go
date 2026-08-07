package service

import (
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/stretchr/testify/require"
)

// 仅配置全局按秒档位、无渠道档位时，有效单价须套用成本折扣（与日志展示一致）。
func TestEffectiveVideoPerSecondUSD_GlobalFallbackAppliesCostDiscount(t *testing.T) {
	prevGlobal := ratio_setting.VideoPricingRules2JSONString()
	prevChannel := ratio_setting.ChannelVideoPricingRules2JSONString()
	t.Cleanup(func() {
		_ = ratio_setting.UpdateVideoPricingRulesByJSONString(prevGlobal)
		_ = ratio_setting.UpdateChannelVideoPricingRulesByJSONString(prevChannel)
	})
	require.NoError(t, ratio_setting.UpdateChannelVideoPricingRulesByJSONString(`{}`))

	const (
		modelName  = "happyhorse-1.1-t2v-global-fb"
		rawPerSec  = 0.129
		costDisc   = 59.0
		channelID  = 42
		resolution = "720p"
	)
	require.NoError(t, ratio_setting.UpdateVideoPricingRulesByJSONString(
		`{"`+modelName+`":{"text_to_video_per_second":[{"resolution":"`+resolution+`","has_audio":true,"price":`+
			strconv.FormatFloat(rawPerSec, 'g', -1, 64)+`}]}}`,
	))

	eff, ch, glob, ok := EffectiveVideoPerSecondUSDForDimensions(
		channelID, modelName, "text_to_video", 405, 720, true, costDisc, 0, resolution,
	)
	require.True(t, ok)
	require.InDelta(t, rawPerSec, ch, 1e-9)
	require.InDelta(t, rawPerSec, glob, 1e-9)
	wantEff := model.EffectiveRuleUnitPrice(rawPerSec, rawPerSec, costDisc, 0)
	require.InDelta(t, wantEff, eff, 1e-9)
	require.InDelta(t, rawPerSec*(costDisc/100), eff, 1e-9)

	// 4 秒结算额度须按折扣后单价，而非档位原价。
	seconds := 4
	gotQuota := int(float64(seconds) * eff * common.QuotaPerUnit)
	wantQuota := int(float64(seconds) * wantEff * common.QuotaPerUnit)
	require.Equal(t, wantQuota, gotQuota)
	rawQuota := int(float64(seconds) * rawPerSec * common.QuotaPerUnit)
	require.NotEqual(t, rawQuota, gotQuota, "不得按全局原价实扣")
}

// 成片结算在无渠道档位时须用全局按秒价×折扣重算，不能因缺渠道规则直接放弃。
func TestRecalcVideoPerSecondQuota_GlobalFallbackAppliesCostDiscount(t *testing.T) {
	prevGlobal := ratio_setting.VideoPricingRules2JSONString()
	prevChannel := ratio_setting.ChannelVideoPricingRules2JSONString()
	t.Cleanup(func() {
		_ = ratio_setting.UpdateVideoPricingRulesByJSONString(prevGlobal)
		_ = ratio_setting.UpdateChannelVideoPricingRulesByJSONString(prevChannel)
	})
	require.NoError(t, ratio_setting.UpdateChannelVideoPricingRulesByJSONString(`{}`))

	const (
		modelName  = "happyhorse-recalc-global-fb"
		rawPerSec  = 0.129
		costDisc   = 59.0
		resolution = "720p"
		seconds    = 4
	)
	require.NoError(t, ratio_setting.UpdateVideoPricingRulesByJSONString(
		`{"`+modelName+`":{"text_to_video_per_second":[{"resolution":"`+resolution+`","has_audio":true,"price":`+
			strconv.FormatFloat(rawPerSec, 'g', -1, 64)+`}]}}`,
	))

	costDiscCopy := costDisc
	req := relaycommon.TaskSubmitReq{
		Model:      modelName,
		Prompt:     "test",
		Resolution: resolution,
		Seconds:    strconv.Itoa(seconds),
		Metadata: map[string]any{
			"resolution": resolution,
			"has_audio":  true,
			"ratio":      "9:16",
		},
	}
	reqBytes, err := common.Marshal(req)
	require.NoError(t, err)

	task := &model.Task{
		TaskID:    "task_global_fb_recalc",
		UserId:    1,
		ChannelId: 42,
		Quota:     int(float64(seconds) * rawPerSec * common.QuotaPerUnit), // 模拟错误的原价预扣
		Status:    model.TaskStatusSuccess,
		Properties: model.Properties{
			Input:           string(reqBytes),
			OriginModelName: modelName,
		},
		PrivateData: model.TaskPrivateData{
			BillingContext: &model.TaskBillingContext{
				ModelPrice:                  0,
				ModelRatio:                  0,
				GroupRatio:                  1,
				OriginModelName:             modelName,
				OtherRatios:                 map[string]float64{"seconds": float64(seconds)},
				ChannelPriceDiscountPercent: costDisc,
				EffectiveCostPercent:        &costDiscCopy,
			},
		},
	}
	taskResult := &relaycommon.TaskInfo{
		Status:     string(model.TaskStatusSuccess),
		Duration:   seconds,
		Resolution: resolution,
		Ratio:      "9:16",
	}

	quota, detail := recalcVideoPerSecondQuotaDetailOnComplete(task, taskResult)
	require.NotNil(t, detail)
	wantEff := model.EffectiveRuleUnitPrice(rawPerSec, rawPerSec, costDisc, 0)
	wantQuota := int(float64(seconds) * wantEff * common.QuotaPerUnit)
	require.Equal(t, wantQuota, quota)
	require.InDelta(t, wantEff, detail.EffectivePricePerSecond, 1e-9)
	require.NotEqual(t, task.Quota, quota, "须相对原价预扣发生差额纠正")
}
