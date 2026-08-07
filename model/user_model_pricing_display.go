package model

import (
	"strings"

	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

// ApplyUserPricingOverrideToPricingAPI 将登录用户的「用户指定价」应用到定价接口展示数据：
//   - 命中覆盖的模型：渠道展示价改写为「全局官方价 × (成本折扣 + 经营成本 + 加价折扣)」，
//     与计费口径一致（渠道无关）；
//   - 渠道有效单价超出用户指定价上限的条目整条隐藏（该用户无法调用这些渠道）；
//   - 未命中覆盖的模型原样保留。
//
// 返回过滤后的新切片。data 为「模型 × 单渠道」打平结构，每条仅一个 ChannelList 元素。
func ApplyUserPricingOverrideToPricingAPI(userId int, pricingData []PricingAPIItem) []PricingAPIItem {
	if userId <= 0 || len(pricingData) == 0 {
		return pricingData
	}
	out := make([]PricingAPIItem, 0, len(pricingData))
	for i := range pricingData {
		item := pricingData[i]
		modelName := strings.TrimSpace(item.ModelName)
		ov, ok := GetEnabledUserModelPricingOverride(userId, modelName)
		if !ok || len(item.ChannelList) == 0 {
			out = append(out, item)
			continue
		}

		totalPercent := ov.TotalPercent()
		effCostOv := EffectiveCostPercent(ov.PriceDiscountPercent, ov.OperatingCostPercent)
		markupOv := ov.MarkupDiscountRate
		if markupOv < 0 {
			markupOv = 0
		}

		globalRatio, _, _ := ratio_setting.GetModelRatio(modelName)
		globalPrice, hasGlobalPrice := ratio_setting.GetModelPrice(modelName, false)
		globalCompletionRatio := ratio_setting.GetCompletionRatio(modelName)
		globalCacheRatio, okCache := ratio_setting.GetCacheRatio(modelName)
		if !okCache {
			globalCacheRatio = 1.0
		}
		globalCreateCacheRatio, okCreate := ratio_setting.GetCreateCacheRatio(modelName)
		if !okCreate {
			globalCreateCacheRatio = 1.25
		}

		hide := false
		for j := range item.ChannelList {
			ch := &item.ChannelList[j]

			// 渠道现价与上限（与 service.ResolveChannelModelUnitPrice / UserModelUnitPriceCap 同口径）。
			var chEff, capBasis float64
			if ch.ModelPrice > 0 && hasGlobalPrice && globalPrice > 0 {
				chEff = EffectiveModelPrice(ch.ModelPrice, globalPrice, ch.EffectiveCostPercent, ch.MarkupDiscountRate)
				capBasis = globalPrice
			} else if globalRatio > 0 {
				chEff = EffectiveInputRate(ch.ModelRatio, globalRatio, ch.EffectiveCostPercent, ch.MarkupDiscountRate)
				capBasis = globalRatio
			} else {
				// 全局官方价未配置：上限无法定义，保留原展示（计费同样回退渠道基价）。
				continue
			}
			if chEff > capBasis*(totalPercent/100.0)*(1+1e-9) {
				hide = true
				break
			}

			// 展示改写：基价替换为全局官方价，三折扣替换为用户覆盖值，
			// 前端公式 chMr×cost% + globalMr×markup% 即得 全局价 × 总折扣。
			if globalRatio > 0 {
				ch.ModelRatio = globalRatio
			}
			if ch.ModelPrice > 0 && hasGlobalPrice && globalPrice > 0 {
				ch.ModelPrice = globalPrice
			}
			ch.CompletionRatio = globalCompletionRatio
			ch.CacheRatio = globalCacheRatio
			ch.CreateCacheRatio = globalCreateCacheRatio
			ch.PriceDiscountPercent = effCostOv
			ch.EffectiveCostPercent = effCostOv
			ch.MarkupDiscountRate = markupOv
			// 指定价不套阶梯计费；Option 渠道成本价字段一并清除避免误导。
			ch.RequestTierPricing = nil
			if ch.QuotaType == 3 {
				ch.QuotaType = 0
			}
			ch.OptionModelRatio = nil
			ch.OptionCompletionRatio = nil
			ch.OptionCacheRatio = nil
			ch.OptionCreateCacheRatio = nil
			ch.OptionModelPrice = nil

			// 分档展示价（视频/图片）按用户折扣重算，与被邀请人加价覆盖处理一致。
			item.VideoFlatClipHint = BuildVideoFlatClipHint(ch.ChannelID, modelName, effCostOv, markupOv)
			item.ImagePerImageHint = BuildImagePerImageHint(ch.ChannelID, modelName, effCostOv, markupOv)
		}
		if !hide {
			out = append(out, item)
		}
	}
	return out
}
