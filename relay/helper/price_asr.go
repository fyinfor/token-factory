package helper

import (
	"fmt"
	"math"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

// ModelPriceHelperASR 阿里云 ASR 语音识别按秒计费价格助手。
//
// 计费模型：UsePrice=true，ModelPrice=每秒单价（美元/秒），OtherRatios["seconds"]=音频时长（秒）。
// seconds 为预估时长：
//   - 同步文件上传：本地解析的音频时长；
//   - 同步 URL 模式：0（无法预估，不预扣）；
//   - 异步提交：固定 60 秒预扣（aliyunasr.AsyncPreConsumeSeconds），成功后按 usage.duration 补差价。
//
// 实际结算前由适配器/任务流程把真实秒数写入 PriceData.OtherRatios["seconds"]，
// calculateTextQuotaSummary 的 UsePrice 分支完成金额计算，分润沿用钱包分润链路。
func ModelPriceHelperASR(c *gin.Context, info *relaycommon.RelayInfo, seconds float64) (types.PriceData, error) {
	if info == nil {
		return types.PriceData{}, fmt.Errorf("relay info is nil")
	}
	channelID := 0
	if info.ChannelMeta != nil {
		channelID = info.ChannelId
	}
	groupRatioInfo := HandleGroupRatio(c, info)

	timePricing := resolveChannelModelTimePricing(c, channelID, info.OriginModelName)
	modelPrice, ok := ratio_setting.GetASRPrice(info.OriginModelName)
	if usesIndependentTimePricing(timePricing) && timePricing.Payload.ASRPrice != nil {
		modelPrice = *timePricing.Payload.ASRPrice
		ok = true
	}
	if !ok || modelPrice <= 0 {
		return types.PriceData{}, fmt.Errorf("模型 %s 未配置 ASR 按秒价格（ASRPrice），请联系管理员在模型定价页面设置；Model %s ASR per-second price not set, please contact the administrator", info.OriginModelName, info.OriginModelName)
	}

	rawDisc, operatingCost, chDisc, markupDisc := resolveChannelBillingPercents(c, info, channelID, info.OriginModelName, timePricing)

	// 预扣额度 = 每秒单价 × 成本/加价折扣 × QuotaPerUnit × 分组倍率 × 预估秒数。
	// ASR 仅有全局 ASRPrice，无独立渠道价表：渠道价与全局价均取该单价，
	// 与图片/视频规则价一致走 EffectiveRuleUnitPrice，使加价折扣进入用户扣费与利润分成切片。
	var preConsumedQuota int
	estSeconds := math.Ceil(seconds)
	if estSeconds > 0 {
		effModelPrice := model.EffectiveRuleUnitPrice(modelPrice, modelPrice, chDisc, markupDisc)
		preConsumedQuota = int(decimal.NewFromFloat(effModelPrice).
			Mul(decimal.NewFromFloat(common.QuotaPerUnit)).
			Mul(decimal.NewFromFloat(groupRatioInfo.GroupRatio)).
			Mul(decimal.NewFromFloat(estSeconds)).Round(0).IntPart())
	}

	// 与 ModelPriceHelper 一致：免费模型（分组倍率 0）未开启预扣时直接放行
	freeModel := false
	if !operation_setting.GetQuotaSetting().EnableFreeModelPreConsume {
		if groupRatioInfo.GroupRatio == 0 {
			preConsumedQuota = 0
			freeModel = true
		}
	}

	chDiscCopy := chDisc
	priceData := types.PriceData{
		FreeModel:               freeModel,
		ModelPrice:              modelPrice, // 美元/秒（渠道侧，当前等同全局 ASRPrice）
		GlobalModelPrice:        modelPrice, // 全局每秒单价，供 calculateTextQuotaSummary / 分润加价切片使用
		GroupRatioInfo:          groupRatioInfo,
		UsePrice:                true,
		ChannelPriceDiscount:    &chDiscCopy,
		QuotaToPreConsume:       preConsumedQuota,
		CostDiscountPercent:     chDisc,
		RawPriceDiscountPercent: rawDisc,
		OperatingCostPercent:    operatingCost,
		MarkupDiscountPercent:   markupDisc,
	}
	if estSeconds > 0 {
		priceData.AddOtherRatio("seconds", estSeconds)
	}
	attachChannelModelTimePricing(&priceData, timePricing)
	info.PriceData = priceData
	return priceData, nil
}
