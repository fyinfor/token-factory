package types

import "fmt"

type GroupRatioInfo struct {
	GroupRatio        float64
	GroupSpecialRatio float64
	HasSpecialRatio   bool
}

type PriceData struct {
	FreeModel            bool
	ModelPrice           float64
	ModelRatio           float64
	CompletionRatio      float64
	CacheRatio           float64
	CacheCreationRatio   float64
	CacheCreation5mRatio float64
	CacheCreation1hRatio float64
	ImageRatio           float64
	AudioRatio           float64
	AudioCompletionRatio float64
	VideoRatio           float64
	VideoCompletionRatio float64
	// VideoOutputTokens is the estimated token count for the generated video,
	// computed by the video task pricing path as duration*W*H*fps/1024.
	// 0 means the request is not a video token-billed call.
	VideoOutputTokens int
	// VideoInputTextTokens is the rough token count of the prompt accompanying
	// the video request (used by the video token-billing branch only).
	VideoInputTextTokens int
	OtherRatios          map[string]float64
	UsePrice             bool
	// ChannelPriceDiscount 非 nil 时，表示最终成本率（成本折扣率 + 经营成本率，百分数），用于日志展示
	ChannelPriceDiscount *float64
	Quota                int // 按次计费的最终额度（MJ / Task）
	QuotaToPreConsume    int // 按量计费的预消耗额度
	GroupRatioInfo       GroupRatioInfo

	// 新计费公式所需字段
	// CostDiscountPercent 最终成本率百分数（price_discount_percent + operating_cost_percent），如 90 表示 90%，默认 100
	CostDiscountPercent float64
	// RawPriceDiscountPercent 是渠道原始 price_discount_percent，未叠加经营成本。
	RawPriceDiscountPercent float64
	// OperatingCostPercent 是渠道经营成本率，会叠加到 RawPriceDiscountPercent。
	OperatingCostPercent float64
	// MarkupDiscountPercent 加价折扣率百分数（markup_discount_rate），如 5 表示 5%，默认 0
	MarkupDiscountPercent float64
	// UserPricingOverride 为 true 时表示本次计费命中「用户指定价」改写：三折扣为用户级覆盖值、
	// 基价已替换为全局官方价（渠道无关）、分组倍率强制为 1，且不再套用阶梯计费。
	// 代理身份不会置 true（指定价仅约束选路，自用仍按渠道成本价）。
	UserPricingOverride bool
	// GlobalModelRatio 全局模型输入倍率（不含渠道/分组覆盖），用于新计费公式加价部分
	GlobalModelRatio float64
	// GlobalModelPrice 全局模型固定价格（USD，不含渠道/分组覆盖），用于固定价新计费公式
	GlobalModelPrice float64
	// GlobalCompletionRatio 全局模型输出倍率，用于输出侧加价计算
	// 新公式输出加价部分 = globalMr × GlobalCompletionRatio × markupRate%
	GlobalCompletionRatio float64
	// GlobalCacheRatio 全局缓存读取倍率，用于缓存读取侧加价计算
	// 新公式缓存读取加价部分 = globalMr × GlobalCacheRatio × markupRate%
	GlobalCacheRatio float64
	// GlobalCreateCacheRatio 全局缓存创建倍率，用于缓存写入侧加价计算
	// 新公式缓存创建加价部分 = globalMr × GlobalCreateCacheRatio × markupRate%
	GlobalCreateCacheRatio float64

	// Video rule billing snapshot. Used by async task settlement/profit share to
	// keep per-task billing stable when pricing settings change after submit.
	VideoRuleUnit         string
	VideoBillingMode      string
	VideoChannelRulePrice float64
	VideoGlobalRulePrice  float64
	VideoRuleWidth        int
	VideoRuleHeight       int
	VideoRuleHasAudio     bool
}

func (p *PriceData) AddOtherRatio(key string, ratio float64) {
	if p.OtherRatios == nil {
		p.OtherRatios = make(map[string]float64)
	}
	if ratio <= 0 {
		return
	}
	p.OtherRatios[key] = ratio
}

func (p *PriceData) ToSetting() string {
	if p == nil {
		return "PriceData: <nil>"
	}
	chdStr := "unset"
	if p.ChannelPriceDiscount != nil {
		chdStr = fmt.Sprintf("%f", *p.ChannelPriceDiscount)
	}
	return fmt.Sprintf("ModelPrice: %f, ModelRatio: %f, CompletionRatio: %f, CacheRatio: %f, GroupRatio: %f, UsePrice: %t, CacheCreationRatio: %f, CacheCreation5mRatio: %f, CacheCreation1hRatio: %f, QuotaToPreConsume: %d, ImageRatio: %f, AudioRatio: %f, AudioCompletionRatio: %f, VideoRatio: %f, VideoCompletionRatio: %f, ChannelPriceDiscount(%%): %s", p.ModelPrice, p.ModelRatio, p.CompletionRatio, p.CacheRatio, p.GroupRatioInfo.GroupRatio, p.UsePrice, p.CacheCreationRatio, p.CacheCreation5mRatio, p.CacheCreation1hRatio, p.QuotaToPreConsume, p.ImageRatio, p.AudioRatio, p.AudioCompletionRatio, p.VideoRatio, p.VideoCompletionRatio, chdStr)
}
