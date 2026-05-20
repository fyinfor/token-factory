package model

import "math"

// ============================================================================
// 成本折扣率（price_discount_percent）相关函数
// ============================================================================

// ApplyChannelPriceDiscountToQuota 将「原价算出的额度」乘以渠道折扣（百分数转小数，如 60% -> ×0.6）。
// percent 为渠道存储值：100=不打折，0=全免；<0 或越上限时与 ResolvedPriceDiscountPercent 对齐。
func ApplyChannelPriceDiscountToQuota(quota int, percent float64) int {
	percent = clampChannelPriceDiscountPercent(percent)
	if quota == 0 {
		return 0
	}
	return int(math.Round(float64(quota) * (percent / 100.0)))
}

// clampChannelPriceDiscountPercent 将渠道折扣限制在合理范围；负值或 nil 读出的 0 在 Resolved 中已处理。
func clampChannelPriceDiscountPercent(percent float64) float64 {
	if percent < 0 {
		return 0
	}
	if percent > 1000 {
		return 1000
	}
	return percent
}

// ResolvedPriceDiscountPercent 返回用于计费的成本折扣百分数，缺省/NULL 视为 100（无折扣）。
func (c *Channel) ResolvedPriceDiscountPercent() float64 {
	if c == nil {
		return 100
	}
	if c.PriceDiscountPercent == nil {
		return 100
	}
	return clampChannelPriceDiscountPercent(*c.PriceDiscountPercent)
}

// ResolveChannelPriceDiscountPercent 按渠道 ID 从缓存取渠道并解析成本折扣，失败或无渠道时 100%。
func ResolveChannelPriceDiscountPercent(channelId int) float64 {
	if channelId <= 0 {
		return 100
	}
	ch, err := CacheGetChannel(channelId)
	if err != nil || ch == nil {
		return 100
	}
	return ch.ResolvedPriceDiscountPercent()
}

// ChannelPriceDiscountMultiplierForPricing 用于展示：把 60% 转为 0.6。
func ChannelPriceDiscountMultiplierForPricing(percent float64) float64 {
	percent = clampChannelPriceDiscountPercent(percent)
	return percent / 100.0
}

// ============================================================================
// 加价折扣率（markup_discount_rate）相关函数
// ============================================================================

// clampChannelMarkupDiscountRate 将加价折扣率限制在合理范围（0-1000%）。
func clampChannelMarkupDiscountRate(percent float64) float64 {
	if percent < 0 {
		return 0
	}
	if percent > 1000 {
		return 1000
	}
	return percent
}

// ResolvedMarkupDiscountRate 返回用于计费的加价折扣百分数，缺省/NULL 视为 0（无加价）。
func (c *Channel) ResolvedMarkupDiscountRate() float64 {
	if c == nil {
		return 0
	}
	if c.MarkupDiscountRate == nil {
		return 0
	}
	return clampChannelMarkupDiscountRate(*c.MarkupDiscountRate)
}

// ResolveChannelMarkupDiscountRate 按渠道 ID 从缓存取渠道并解析加价折扣，失败或无渠道时 0%。
func ResolveChannelMarkupDiscountRate(channelId int) float64 {
	if channelId <= 0 {
		return 0
	}
	ch, err := CacheGetChannel(channelId)
	if err != nil || ch == nil {
		return 0
	}
	return ch.ResolvedMarkupDiscountRate()
}

// ============================================================================
// 新计费公式辅助函数
// ============================================================================
// 新公式（各类型独立有效倍率，所有百分比字段按 % 计算，如 90 代表 90%）：
//   输入       = (ch.model_ratio × costDisc% + globalMr × markupRate%) × groupRatio
//   输出       = (ch.model_ratio × completionRatio × costDisc% + globalMr × 全局输出倍率 × markupRate%) × groupRatio
//   缓存读取   = (ch.model_ratio × ch.cache_ratio × costDisc% + globalMr × 全局读取缓存倍率 × markupRate%) × groupRatio
//   缓存创建   = (ch.model_ratio × ch.create_cache_ratio × costDisc% + globalMr × 全局创建缓存倍率 × markupRate%) × groupRatio
//   固定价格   = ch.model_price × costDisc% + 全局固定价 × markupRate%
// ============================================================================

// EffectiveInputRate 计算有效输入倍率（不含分组倍率）。
// channelRatio: 渠道解析后的模型输入倍率（渠道无设置时已回退至全局）
// globalRatio: 全局模型输入倍率（ratio_setting.GetModelRatio 返回值）
// costDiscPercent: 成本折扣率百分数（price_discount_percent，如 90 表示 90%）
// markupDiscPercent: 加价折扣率百分数（markup_discount_rate，如 5 表示 5%）
func EffectiveInputRate(channelRatio, globalRatio, costDiscPercent, markupDiscPercent float64) float64 {
	return channelRatio*(costDiscPercent/100.0) + globalRatio*(markupDiscPercent/100.0)
}

// EffectiveOutputRate 计算有效输出倍率（每输出 token 的有效价格，不含分组倍率）。
// channelRatio: 渠道解析后的模型输入倍率
// completionRatio: 渠道解析后的输出倍率（相对于输入价格的倍数）
// globalRatio: 全局模型输入倍率
// globalCompletionRatio: 全局模型输出倍率（用于加价部分）
// costDiscPercent/markupDiscPercent: 成本/加价折扣率百分数
//
// 新公式：输出 = channelRatio × completionRatio × costDisc% + globalRatio × globalCompletionRatio × markupDisc%
func EffectiveOutputRate(channelRatio, completionRatio, globalRatio, globalCompletionRatio, costDiscPercent, markupDiscPercent float64) float64 {
	return channelRatio*completionRatio*(costDiscPercent/100.0) + globalRatio*globalCompletionRatio*(markupDiscPercent/100.0)
}

// EffectiveCacheReadRate 计算有效缓存读取倍率（每个缓存读取 token 的有效价格，不含分组倍率）。
// channelRatio: 渠道解析后的模型输入倍率
// channelCacheRatio: 渠道解析后的缓存读取倍率
// globalRatio: 全局模型输入倍率
// globalCacheRatio: 全局缓存读取倍率（用于加价部分）
// costDiscPercent/markupDiscPercent: 成本/加价折扣率百分数
//
// 新公式：缓存读取 = channelRatio × channelCacheRatio × costDisc% + globalRatio × globalCacheRatio × markupDisc%
func EffectiveCacheReadRate(channelRatio, channelCacheRatio, globalRatio, globalCacheRatio, costDiscPercent, markupDiscPercent float64) float64 {
	return channelRatio*channelCacheRatio*(costDiscPercent/100.0) + globalRatio*globalCacheRatio*(markupDiscPercent/100.0)
}

// EffectiveCacheCreationRate 计算有效缓存创建倍率（每个缓存写入 token 的有效价格，不含分组倍率）。
// channelRatio: 渠道解析后的模型输入倍率
// channelCreateCacheRatio: 渠道解析后的缓存创建倍率
// globalRatio: 全局模型输入倍率
// globalCreateCacheRatio: 全局缓存创建倍率（用于加价部分）
// costDiscPercent/markupDiscPercent: 成本/加价折扣率百分数
//
// 新公式：缓存创建 = channelRatio × channelCreateCacheRatio × costDisc% + globalRatio × globalCreateCacheRatio × markupDisc%
func EffectiveCacheCreationRate(channelRatio, channelCreateCacheRatio, globalRatio, globalCreateCacheRatio, costDiscPercent, markupDiscPercent float64) float64 {
	return channelRatio*channelCreateCacheRatio*(costDiscPercent/100.0) + globalRatio*globalCreateCacheRatio*(markupDiscPercent/100.0)
}

// EffectiveModelPrice 计算有效固定价格（USD/次，不含分组倍率）。
// channelPrice: 渠道解析后的固定价（渠道无设置时已回退至全局）
// globalPrice: 全局模型固定价（ratio_setting.GetModelPrice 返回值）
// costDiscPercent/markupDiscPercent: 成本/加价折扣率百分数
func EffectiveModelPrice(channelPrice, globalPrice, costDiscPercent, markupDiscPercent float64) float64 {
	return channelPrice*(costDiscPercent/100.0) + globalPrice*(markupDiscPercent/100.0)
}
