package model

import (
	"strings"

	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

// ApplyUserPricingOverrideToPricingAPI 将登录用户的「用户指定价」应用到定价接口展示数据：
//   - 普通用户命中覆盖：渠道展示价改写为「全局官方价 × (成本折扣 + 经营成本 + 加价折扣)」，
//     与计费口径一致（渠道无关）；
//   - 代理身份命中覆盖：不改写为指定售价，保留渠道成本展示（加价置 0，与自用计费一致），
//     仍按 Mode 过滤渠道；
//   - price_cap：渠道有效单价超出用户指定价上限的条目整条隐藏；
//   - channel_list：仅保留勾选渠道，未勾选整条隐藏；
//   - 未命中覆盖的模型原样保留。
//
// 返回过滤后的新切片。data 为「模型 × 单渠道」打平结构，每条仅一个 ChannelList 元素。
func ApplyUserPricingOverrideToPricingAPI(userId int, pricingData []PricingAPIItem) []PricingAPIItem {
	if userId <= 0 || len(pricingData) == 0 {
		return pricingData
	}
	rewriteBillingDisplay := UserPricingBillingApplies(userId)
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
		mode := ov.NormalizedMode()
		allowSet, allowOK := GetEnabledUserModelPricingChannelAllowSet(userId, modelName)

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

			if mode == UserPricingModeChannelList {
				if !allowOK || allowSet == nil {
					hide = true
					break
				}
				if _, ok := allowSet[ch.ChannelID]; !ok {
					hide = true
					break
				}
			} else {
				// 渠道现价与上限（与 service.ResolveChannelModelUnitPrice / UserModelUnitPriceCap 同口径）。
				// 选路上限对代理/普通用户一致，仍按渠道默认加价计算「渠道有效单价」。
				var chEff, capBasis float64
				if ch.ModelPrice > 0 && hasGlobalPrice && globalPrice > 0 {
					chEff = EffectiveModelPrice(ch.ModelPrice, globalPrice, ch.EffectiveCostPercent, ch.MarkupDiscountRate)
					capBasis = globalPrice
				} else if globalRatio > 0 {
					chEff = EffectiveInputRate(ch.ModelRatio, globalRatio, ch.EffectiveCostPercent, ch.MarkupDiscountRate)
					capBasis = globalRatio
				} else {
					// 全局官方价未配置：上限无法定义，保留原展示（计费同样回退渠道基价）。
					applyUserPricingChannelDisplay(ch, rewriteBillingDisplay, globalRatio, globalPrice, hasGlobalPrice,
						globalCompletionRatio, globalCacheRatio, globalCreateCacheRatio, effCostOv, markupOv)
					item.VideoFlatClipHint = BuildVideoFlatClipHint(ch.ChannelID, modelName, displayCostPercent(ch, rewriteBillingDisplay, effCostOv), displayMarkupPercent(ch, rewriteBillingDisplay, markupOv))
					item.ImagePerImageHint = BuildImagePerImageHint(ch.ChannelID, modelName, displayCostPercent(ch, rewriteBillingDisplay, effCostOv), displayMarkupPercent(ch, rewriteBillingDisplay, markupOv))
					continue
				}
				if chEff > capBasis*(totalPercent/100.0)*(1+1e-9) {
					hide = true
					break
				}
			}

			applyUserPricingChannelDisplay(ch, rewriteBillingDisplay, globalRatio, globalPrice, hasGlobalPrice,
				globalCompletionRatio, globalCacheRatio, globalCreateCacheRatio, effCostOv, markupOv)
			item.VideoFlatClipHint = BuildVideoFlatClipHint(ch.ChannelID, modelName, displayCostPercent(ch, rewriteBillingDisplay, effCostOv), displayMarkupPercent(ch, rewriteBillingDisplay, markupOv))
			item.ImagePerImageHint = BuildImagePerImageHint(ch.ChannelID, modelName, displayCostPercent(ch, rewriteBillingDisplay, effCostOv), displayMarkupPercent(ch, rewriteBillingDisplay, markupOv))
		}
		if !hide {
			out = append(out, item)
		}
	}
	return out
}

func displayCostPercent(ch *PricingChannelItem, rewriteBilling bool, effCostOv float64) float64 {
	if rewriteBilling {
		return effCostOv
	}
	if ch == nil {
		return 100
	}
	return ch.EffectiveCostPercent
}

func displayMarkupPercent(ch *PricingChannelItem, rewriteBilling bool, markupOv float64) float64 {
	if rewriteBilling {
		return markupOv
	}
	if ch == nil {
		return 0
	}
	return ch.MarkupDiscountRate
}

// applyUserPricingChannelDisplay 普通用户改写为指定售价；代理仅将加价置 0 以对齐自用成本价。
func applyUserPricingChannelDisplay(
	ch *PricingChannelItem,
	rewriteBilling bool,
	globalRatio, globalPrice float64,
	hasGlobalPrice bool,
	globalCompletionRatio, globalCacheRatio, globalCreateCacheRatio, effCostOv, markupOv float64,
) {
	if ch == nil {
		return
	}
	if rewriteBilling {
		rewriteUserPricingChannelDisplay(ch, globalRatio, globalPrice, hasGlobalPrice,
			globalCompletionRatio, globalCacheRatio, globalCreateCacheRatio, effCostOv, markupOv)
		return
	}
	// 代理：保留渠道成本侧字段，加价强制为 0（与 ResolveEffectiveMarkup… 自用计费一致）。
	ch.MarkupDiscountRate = 0
}

func rewriteUserPricingChannelDisplay(
	ch *PricingChannelItem,
	globalRatio, globalPrice float64,
	hasGlobalPrice bool,
	globalCompletionRatio, globalCacheRatio, globalCreateCacheRatio, effCostOv, markupOv float64,
) {
	if ch == nil {
		return
	}
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
}
