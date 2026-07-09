package model

import (
	"fmt"
	"math"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

// SettlementDiscountSnapshot 从日志 other 解析的折扣快照。
type SettlementDiscountSnapshot struct {
	PriceDiscountPercent     float64
	OperatingCostPercent     float64
	OperatingDiscountPercent float64
	MarkupDiscountPercent    float64
	SalesDiscountPercent     float64
}

// SettlementPriceBreakdown 结算单价格拆解（基于官方价 × 折扣率，金额为内部 USD）。
type SettlementPriceBreakdown struct {
	OfficialInputPrice  float64
	OfficialOutputPrice float64
	OfficialCachePrice  float64
	OfficialTotal       float64
	CostPrice           float64
	OperatingPrice      float64
	SalesPrice          float64
	Discounts           SettlementDiscountSnapshot
}

// SalesDiscountPercent 销售折扣 = 成本折扣 + 经营成本 + 加价折扣（百分数相加）。
func SalesDiscountPercent(priceDiscount, operatingCost, markupDiscount float64) float64 {
	return clampChannelPriceDiscountPercent(
		EffectiveCostPercent(priceDiscount, operatingCost) + clampChannelMarkupDiscountRate(markupDiscount),
	)
}

func otherFloat(m map[string]interface{}, key string) (float64, bool) {
	if m == nil {
		return 0, false
	}
	v, ok := m[key]
	if !ok || v == nil {
		return 0, false
	}
	switch x := v.(type) {
	case float64:
		return x, true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	default:
		return 0, false
	}
}

func otherInt(m map[string]interface{}, key string) int {
	if m == nil {
		return 0
	}
	v, ok := m[key]
	if !ok || v == nil {
		return 0
	}
	switch x := v.(type) {
	case float64:
		return int(x)
	case int:
		return x
	case int64:
		return int(x)
	default:
		return 0
	}
}

func resolveSettlementGroupRatio(otherMap map[string]interface{}) float64 {
	groupRatio := 1.0
	if v, ok := otherFloat(otherMap, "group_ratio"); ok && v > 0 {
		groupRatio = v
	}
	if v, ok := otherFloat(otherMap, "user_group_ratio"); ok && v > 0 {
		return v
	}
	return groupRatio
}

func isExplicitPerCallModelPrice(modelPrice float64) bool {
	return modelPrice > 0
}

func resolveSettlementGlobalRatio(otherMap map[string]interface{}) float64 {
	if v, ok := otherFloat(otherMap, "global_model_ratio"); ok && v > 0 {
		return v
	}
	if v, ok := otherFloat(otherMap, "model_ratio"); ok && v > 0 {
		return v
	}
	return 0
}

func resolveSettlementGlobalPrice(otherMap map[string]interface{}) float64 {
	if v, ok := otherFloat(otherMap, "global_model_price"); ok && v > 0 {
		return v
	}
	modelPrice, _ := otherFloat(otherMap, "model_price")
	if isExplicitPerCallModelPrice(modelPrice) {
		return modelPrice
	}
	return 0
}

func resolveSettlementGlobalCompletionRatio(otherMap map[string]interface{}) float64 {
	if v, ok := otherFloat(otherMap, "global_completion_ratio"); ok && v > 0 {
		return v
	}
	if v, ok := otherFloat(otherMap, "completion_ratio"); ok && v > 0 {
		return v
	}
	return 1
}

func resolveSettlementGlobalCacheRatio(otherMap map[string]interface{}) float64 {
	if v, ok := otherFloat(otherMap, "global_cache_ratio"); ok && v > 0 {
		return v
	}
	if v, ok := otherFloat(otherMap, "cache_ratio"); ok && v > 0 {
		return v
	}
	return 1
}

func applySettlementDiscountBreakdown(breakdown *SettlementPriceBreakdown, discounts SettlementDiscountSnapshot) {
	if breakdown == nil || breakdown.OfficialTotal <= 0 {
		return
	}
	breakdown.CostPrice = breakdown.OfficialTotal * discounts.PriceDiscountPercent / 100
	breakdown.OperatingPrice = breakdown.OfficialTotal * discounts.OperatingDiscountPercent / 100
	breakdown.SalesPrice = breakdown.OfficialTotal * discounts.SalesDiscountPercent / 100
}

func backfillOfficialFromQuota(breakdown *SettlementPriceBreakdown, quota int, discounts SettlementDiscountSnapshot) {
	if breakdown == nil || quota <= 0 || common.QuotaPerUnit <= 0 {
		return
	}
	salesUSD := float64(quota) / common.QuotaPerUnit
	breakdown.SalesPrice = salesUSD
	if breakdown.OfficialTotal > 0 {
		return
	}
	if discounts.SalesDiscountPercent > 0 {
		breakdown.OfficialTotal = salesUSD * 100 / discounts.SalesDiscountPercent
	} else {
		breakdown.OfficialTotal = salesUSD
	}
	applySettlementDiscountBreakdown(breakdown, discounts)
}

// ParseSettlementDiscountSnapshot 从日志 other 解析折扣快照，兼容旧日志。
func ParseSettlementDiscountSnapshot(otherMap map[string]interface{}) SettlementDiscountSnapshot {
	out := SettlementDiscountSnapshot{
		PriceDiscountPercent:     100,
		OperatingCostPercent:     0,
		OperatingDiscountPercent: 100,
		MarkupDiscountPercent:    0,
		SalesDiscountPercent:     100,
	}
	if otherMap == nil {
		return out
	}
	if v, ok := otherFloat(otherMap, "price_discount_percent"); ok {
		out.PriceDiscountPercent = v
	}
	if v, ok := otherFloat(otherMap, "operating_cost_percent"); ok {
		out.OperatingCostPercent = v
	}
	if v, ok := otherFloat(otherMap, "channel_price_discount_percent"); ok {
		out.OperatingDiscountPercent = v
	} else {
		out.OperatingDiscountPercent = EffectiveCostPercent(out.PriceDiscountPercent, out.OperatingCostPercent)
	}
	if v, ok := otherFloat(otherMap, "markup_discount_rate"); ok {
		out.MarkupDiscountPercent = v
	}
	if v, ok := otherFloat(otherMap, "sales_discount_percent"); ok {
		out.SalesDiscountPercent = v
	}
	if _, ok := otherFloat(otherMap, "price_discount_percent"); !ok {
		out.PriceDiscountPercent = out.OperatingDiscountPercent - out.OperatingCostPercent
		if out.PriceDiscountPercent < 0 {
			out.PriceDiscountPercent = out.OperatingDiscountPercent
		}
	}
	if _, ok := otherFloat(otherMap, "sales_discount_percent"); !ok {
		out.SalesDiscountPercent = SalesDiscountPercent(out.PriceDiscountPercent, out.OperatingCostPercent, out.MarkupDiscountPercent)
	}
	return out
}

// ComputeSettlementPriceBreakdown 基于日志用量、实扣 quota 与 other 快照计算结算价格拆解（内部 USD）。
func ComputeSettlementPriceBreakdown(promptTokens, completionTokens, cacheTokens, quota int, otherMap map[string]interface{}) SettlementPriceBreakdown {
	discounts := ParseSettlementDiscountSnapshot(otherMap)
	breakdown := SettlementPriceBreakdown{Discounts: discounts}
	groupRatio := resolveSettlementGroupRatio(otherMap)

	if cacheTokens == 0 {
		cacheTokens = otherInt(otherMap, "cache_tokens")
	}

	globalPrice := resolveSettlementGlobalPrice(otherMap)
	if globalPrice > 0 {
		breakdown.OfficialTotal = globalPrice * groupRatio
		applySettlementDiscountBreakdown(&breakdown, discounts)
		backfillOfficialFromQuota(&breakdown, quota, discounts)
		return breakdown
	}

	globalRatio := resolveSettlementGlobalRatio(otherMap)
	if globalRatio > 0 {
		completionRatio := resolveSettlementGlobalCompletionRatio(otherMap)
		cacheRatio := resolveSettlementGlobalCacheRatio(otherMap)
		textInputTokens := promptTokens - cacheTokens
		if textInputTokens < 0 {
			textInputTokens = promptTokens
		}

		const million = 1_000_000.0
		const ratioToUSD = 2.0
		inputUSD := float64(textInputTokens) * globalRatio * ratioToUSD / million * groupRatio
		outputUSD := float64(completionTokens) * globalRatio * completionRatio * ratioToUSD / million * groupRatio
		cacheUSD := float64(cacheTokens) * globalRatio * cacheRatio * ratioToUSD / million * groupRatio

		if textInputTokens > 0 {
			breakdown.OfficialInputPrice = globalRatio * ratioToUSD * groupRatio
		}
		if completionTokens > 0 {
			breakdown.OfficialOutputPrice = globalRatio * completionRatio * ratioToUSD * groupRatio
		}
		if cacheTokens > 0 {
			breakdown.OfficialCachePrice = globalRatio * cacheRatio * ratioToUSD * groupRatio
		}
		breakdown.OfficialTotal = inputUSD + outputUSD + cacheUSD
	}

	if breakdown.OfficialTotal > 0 {
		applySettlementDiscountBreakdown(&breakdown, discounts)
	}
	backfillOfficialFromQuota(&breakdown, quota, discounts)
	return breakdown
}

// SettlementCurrencyLabel 返回结算导出表头用的货币标识。
func SettlementCurrencyLabel() string {
	switch operation_setting.GetQuotaDisplayType() {
	case operation_setting.QuotaDisplayTypeCNY:
		return "CNY"
	case operation_setting.QuotaDisplayTypeCustom:
		symbol := operation_setting.GetGeneralSetting().CustomCurrencySymbol
		if symbol != "" {
			return symbol
		}
		return "自定义"
	case operation_setting.QuotaDisplayTypeTokens:
		return "额度"
	default:
		return "USD"
	}
}

// UsdToSettlementDisplayAmount 将内部 USD 金额转为平台结算展示货币数值（与额度展示舍入一致）。
func UsdToSettlementDisplayAmount(usd float64) float64 {
	if !isFiniteFloat(usd) || usd == 0 || common.QuotaPerUnit <= 0 {
		return 0
	}
	quota := int(math.Round(usd * common.QuotaPerUnit))
	if quota <= 0 && usd > 0 {
		quota = 1
	}
	return logger.QuotaToRoundedDisplayAmount(quota, 2)
}

// FormatSettlementMoney 将内部 USD 金额格式化为带货币符号的结算展示字符串。
func FormatSettlementMoney(usdAmount float64) string {
	if !isFiniteFloat(usdAmount) {
		usdAmount = 0
	}
	dispType := operation_setting.GetQuotaDisplayType()
	if dispType == operation_setting.QuotaDisplayTypeTokens {
		if common.QuotaPerUnit <= 0 {
			return "0"
		}
		return fmt.Sprintf("%d", int(math.Round(usdAmount*common.QuotaPerUnit)))
	}
	display := UsdToSettlementDisplayAmount(usdAmount)
	sym := operation_setting.GetCurrencySymbol()
	return fmt.Sprintf("%s%.2f", sym, display)
}

// FormatSettlementPercent 格式化百分数展示（如 85.0%）。
func FormatSettlementPercent(percent float64) string {
	if !isFiniteFloat(percent) {
		return "0.0%"
	}
	return fmt.Sprintf("%.1f%%", percent)
}

// FormatSettlementUSD 兼容旧调用：按平台结算货币格式化金额。
func FormatSettlementUSD(amount float64) string {
	return FormatSettlementMoney(amount)
}

// QuotaToMoneyAmount 将 quota 换算为与充值订单一致的金额（美元）。
func QuotaToMoneyAmount(quota int) float64 {
	if quota <= 0 || common.QuotaPerUnit <= 0 {
		return 0
	}
	return float64(quota) / common.QuotaPerUnit
}
