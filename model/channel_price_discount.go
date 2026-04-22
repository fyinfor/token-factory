package model

import "math"

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

// ResolvedPriceDiscountPercent 返回用于计费的折扣百分数，缺省/NULL 视为 100（无折扣）。
func (c *Channel) ResolvedPriceDiscountPercent() float64 {
	if c == nil {
		return 100
	}
	if c.PriceDiscountPercent == nil {
		return 100
	}
	return clampChannelPriceDiscountPercent(*c.PriceDiscountPercent)
}

// ResolveChannelPriceDiscountPercent 按渠道 ID 从缓存取渠道并解析折扣，失败或无渠道时 100%。
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
