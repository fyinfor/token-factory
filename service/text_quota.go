package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	perfmetrics "github.com/QuantumNous/new-api/pkg/perf_metrics"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

type textQuotaSummary struct {
	PromptTokens             int
	CompletionTokens         int
	TotalTokens              int
	CacheTokens              int
	CacheCreationTokens      int
	CacheCreationTokens5m    int
	CacheCreationTokens1h    int
	ImageTokens              int
	AudioTokens              int
	ModelName                string
	TokenName                string
	UseTimeSeconds           int64
	CompletionRatio          float64
	CacheRatio               float64
	ImageRatio               float64
	ModelRatio               float64
	GroupRatio               float64
	ModelPrice               float64
	CacheCreationRatio       float64
	CacheCreationRatio5m     float64
	CacheCreationRatio1h     float64
	Quota                    int
	IsClaudeUsageSemantic    bool
	UsageSemantic            string
	WebSearchPrice           float64
	WebSearchCallCount       int
	ClaudeWebSearchPrice     float64
	ClaudeWebSearchCallCount int
	FileSearchPrice          float64
	FileSearchCallCount      int
	AudioInputPrice          float64
	ImageGenerationCallPrice float64
	RequestTierPricing       bool
	RequestTierBreakdown     ratio_setting.RequestTierPricingBreakdown
	// 新计费公式字段
	CostDiscountPercent    float64 // 最终成本率%（price_discount_percent + operating_cost_percent），默认 100
	MarkupDiscountPercent  float64 // 加价折扣率%（markup_discount_rate），默认 0
	GlobalModelRatio       float64 // 全局模型输入倍率
	GlobalModelPrice       float64 // 全局模型固定价格
	GlobalCompletionRatio  float64 // 全局模型输出倍率（用于输出侧加价计算）
	GlobalCacheRatio       float64 // 全局缓存读取倍率（用于缓存读取侧加价计算）
	GlobalCreateCacheRatio float64 // 全局缓存创建倍率（用于缓存写入侧加价计算）

	// 阶梯计费展示字段
	TierInputLabel          string  // 输入档位标签，如 "输入token<32k"
	TierOutputLabel         string  // 输出档位标签
	TierCacheReadLabel      string  // 缓存读取档位标签
	TierCacheWriteLabel     string  // 缓存写入档位标签
	TierInputUnitPrice      float64 // 输入单价（$/1M tokens，仅文本输入，已含阶梯折抵+有效倍率+分组倍率）
	TierOutputUnitPrice     float64 // 输出单价（$/1M tokens）
	TierCacheReadUnitPrice  float64 // 缓存读取单价（$/1M tokens）
	TierCacheWriteUnitPrice float64 // 缓存写入单价（$/1M tokens）
}

// formatTierUsdPrice 格式化为最多 6 位小数，并去除尾随零（$7.500000 → 7.5，0.06666666 → 0.066667）
func formatTierUsdPrice(price float64) string {
	s := fmt.Sprintf("%.6f", price)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
	}
	return s
}

func cacheWriteTokensTotal(summary textQuotaSummary) int {
	if summary.CacheCreationTokens5m > 0 || summary.CacheCreationTokens1h > 0 {
		splitCacheWriteTokens := summary.CacheCreationTokens5m + summary.CacheCreationTokens1h
		if summary.CacheCreationTokens > splitCacheWriteTokens {
			return summary.CacheCreationTokens
		}
		return splitCacheWriteTokens
	}
	return summary.CacheCreationTokens
}

func resolveTextQuotaChannelDiscountPercent(relayInfo *relaycommon.RelayInfo) float64 {
	if relayInfo == nil {
		return 100
	}
	if relayInfo.PriceData.ChannelPriceDiscount != nil {
		return *relayInfo.PriceData.ChannelPriceDiscount
	}
	chID := 0
	if relayInfo.ChannelMeta != nil {
		chID = relayInfo.ChannelId
	}
	return model.ResolveChannelEffectiveCostPercent(chID)
}

func shouldRecordInputTokensTotal(relayInfo *relaycommon.RelayInfo, usage *dto.Usage) bool {
	if relayInfo == nil || usage == nil {
		return false
	}
	if relayInfo.GetFinalRequestRelayFormat() == types.RelayFormatClaude {
		return false
	}
	if usage.UsageSource != "" || usage.UsageSemantic != "" {
		return false
	}
	return usage.ClaudeCacheCreation5mTokens > 0 || usage.ClaudeCacheCreation1hTokens > 0
}

func isLegacyClaudeDerivedOpenAIUsage(relayInfo *relaycommon.RelayInfo, usage *dto.Usage) bool {
	if relayInfo == nil || usage == nil {
		return false
	}
	if relayInfo.GetFinalRequestRelayFormat() == types.RelayFormatClaude {
		return false
	}
	return !summaryUsageSemanticIsClaude(usage) &&
		(strings.Contains(strings.ToLower(relayInfo.OriginModelName), "claude") ||
			usage.ClaudeCacheCreation5mTokens > 0 ||
			usage.ClaudeCacheCreation1hTokens > 0)
}

func summaryUsageSemanticIsClaude(usage *dto.Usage) bool {
	return usage != nil && strings.EqualFold(usage.UsageSemantic, "anthropic")
}

func relayChannelID(relayInfo *relaycommon.RelayInfo) int {
	if relayInfo == nil || relayInfo.ChannelMeta == nil {
		return 0
	}
	return relayInfo.ChannelId
}

func calculateTextQuotaSummary(ctx *gin.Context, relayInfo *relaycommon.RelayInfo, usage *dto.Usage) textQuotaSummary {
	// 从 PriceData 中读取成本折扣率和加价折扣率（默认值：100% 成本折扣，0% 加价）
	costDisc := relayInfo.PriceData.CostDiscountPercent
	if relayInfo.PriceData.ChannelPriceDiscount != nil {
		costDisc = *relayInfo.PriceData.ChannelPriceDiscount
	} else if costDisc == 0 {
		costDisc = 100 // 兼容老数据，未设置时默认无折扣
	}
	markupDisc := relayInfo.PriceData.MarkupDiscountPercent

	summary := textQuotaSummary{
		ModelName:              relayInfo.OriginModelName,
		TokenName:              ctx.GetString("token_name"),
		UseTimeSeconds:         time.Now().Unix() - relayInfo.StartTime.Unix(),
		CompletionRatio:        relayInfo.PriceData.CompletionRatio,
		CacheRatio:             relayInfo.PriceData.CacheRatio,
		ImageRatio:             relayInfo.PriceData.ImageRatio,
		ModelRatio:             relayInfo.PriceData.ModelRatio,
		GroupRatio:             relayInfo.PriceData.GroupRatioInfo.GroupRatio,
		ModelPrice:             relayInfo.PriceData.ModelPrice,
		CacheCreationRatio:     relayInfo.PriceData.CacheCreationRatio,
		CacheCreationRatio5m:   relayInfo.PriceData.CacheCreation5mRatio,
		CacheCreationRatio1h:   relayInfo.PriceData.CacheCreation1hRatio,
		UsageSemantic:          usageSemanticFromUsage(relayInfo, usage),
		CostDiscountPercent:    costDisc,
		MarkupDiscountPercent:  markupDisc,
		GlobalModelRatio:       relayInfo.PriceData.GlobalModelRatio,
		GlobalModelPrice:       relayInfo.PriceData.GlobalModelPrice,
		GlobalCompletionRatio:  relayInfo.PriceData.GlobalCompletionRatio,
		GlobalCacheRatio:       relayInfo.PriceData.GlobalCacheRatio,
		GlobalCreateCacheRatio: relayInfo.PriceData.GlobalCreateCacheRatio,
	}
	summary.IsClaudeUsageSemantic = summary.UsageSemantic == "anthropic"

	if usage == nil {
		usage = &dto.Usage{
			PromptTokens:     relayInfo.GetEstimatePromptTokens(),
			CompletionTokens: 0,
			TotalTokens:      relayInfo.GetEstimatePromptTokens(),
		}
	}

	summary.PromptTokens = usage.PromptTokens
	summary.CompletionTokens = usage.CompletionTokens
	summary.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	summary.CacheTokens = usage.PromptTokensDetails.CachedTokens
	summary.CacheCreationTokens = usage.PromptTokensDetails.CachedCreationTokens
	summary.CacheCreationTokens5m = usage.ClaudeCacheCreation5mTokens
	summary.CacheCreationTokens1h = usage.ClaudeCacheCreation1hTokens
	summary.ImageTokens = usage.PromptTokensDetails.ImageTokens
	summary.AudioTokens = usage.PromptTokensDetails.AudioTokens
	legacyClaudeDerived := isLegacyClaudeDerivedOpenAIUsage(relayInfo, usage)
	isOpenRouterClaudeBilling := relayInfo.ChannelMeta != nil &&
		relayInfo.ChannelType == constant.ChannelTypeOpenRouter &&
		summary.IsClaudeUsageSemantic

	if isOpenRouterClaudeBilling {
		summary.PromptTokens -= summary.CacheTokens
		isUsingCustomSettings := relayInfo.PriceData.UsePrice || hasCustomModelRatio(summary.ModelName, relayInfo.PriceData.ModelRatio)
		if summary.CacheCreationTokens == 0 && relayInfo.PriceData.CacheCreationRatio != 1 && usage.Cost != 0 && !isUsingCustomSettings {
			maybeCacheCreationTokens := CalcOpenRouterCacheCreateTokens(*usage, relayInfo.PriceData)
			if maybeCacheCreationTokens >= 0 && summary.PromptTokens >= maybeCacheCreationTokens {
				summary.CacheCreationTokens = maybeCacheCreationTokens
			}
		}
		summary.PromptTokens -= summary.CacheCreationTokens
	}

	dPromptTokens := decimal.NewFromInt(int64(summary.PromptTokens))
	dCacheTokens := decimal.NewFromInt(int64(summary.CacheTokens))
	dImageTokens := decimal.NewFromInt(int64(summary.ImageTokens))
	dAudioTokens := decimal.NewFromInt(int64(summary.AudioTokens))
	dCompletionTokens := decimal.NewFromInt(int64(summary.CompletionTokens))
	dCachedCreationTokens := decimal.NewFromInt(int64(summary.CacheCreationTokens))
	dImageRatio := decimal.NewFromFloat(summary.ImageRatio)
	dGroupRatio := decimal.NewFromFloat(summary.GroupRatio)
	dQuotaPerUnit := decimal.NewFromFloat(common.QuotaPerUnit)

	var dWebSearchQuota decimal.Decimal
	if relayInfo.ResponsesUsageInfo != nil {
		if webSearchTool, exists := relayInfo.ResponsesUsageInfo.BuiltInTools[dto.BuildInToolWebSearchPreview]; exists && webSearchTool.CallCount > 0 {
			summary.WebSearchCallCount = webSearchTool.CallCount
			summary.WebSearchPrice = operation_setting.GetWebSearchPricePerThousand(summary.ModelName, webSearchTool.SearchContextSize)
			dWebSearchQuota = decimal.NewFromFloat(summary.WebSearchPrice).
				Mul(decimal.NewFromInt(int64(webSearchTool.CallCount))).
				Div(decimal.NewFromInt(1000)).Mul(dGroupRatio).Mul(dQuotaPerUnit)
		}
	} else if strings.HasSuffix(summary.ModelName, "search-preview") {
		searchContextSize := ctx.GetString("chat_completion_web_search_context_size")
		if searchContextSize == "" {
			searchContextSize = "medium"
		}
		summary.WebSearchCallCount = 1
		summary.WebSearchPrice = operation_setting.GetWebSearchPricePerThousand(summary.ModelName, searchContextSize)
		dWebSearchQuota = decimal.NewFromFloat(summary.WebSearchPrice).
			Div(decimal.NewFromInt(1000)).Mul(dGroupRatio).Mul(dQuotaPerUnit)
	}

	var dClaudeWebSearchQuota decimal.Decimal
	summary.ClaudeWebSearchCallCount = ctx.GetInt("claude_web_search_requests")
	if summary.ClaudeWebSearchCallCount > 0 {
		summary.ClaudeWebSearchPrice = operation_setting.GetClaudeWebSearchPricePerThousand()
		dClaudeWebSearchQuota = decimal.NewFromFloat(summary.ClaudeWebSearchPrice).
			Div(decimal.NewFromInt(1000)).Mul(dGroupRatio).Mul(dQuotaPerUnit).
			Mul(decimal.NewFromInt(int64(summary.ClaudeWebSearchCallCount)))
	}

	var dFileSearchQuota decimal.Decimal
	if relayInfo.ResponsesUsageInfo != nil {
		if fileSearchTool, exists := relayInfo.ResponsesUsageInfo.BuiltInTools[dto.BuildInToolFileSearch]; exists && fileSearchTool.CallCount > 0 {
			summary.FileSearchCallCount = fileSearchTool.CallCount
			summary.FileSearchPrice = operation_setting.GetFileSearchPricePerThousand()
			dFileSearchQuota = decimal.NewFromFloat(summary.FileSearchPrice).
				Mul(decimal.NewFromInt(int64(fileSearchTool.CallCount))).
				Div(decimal.NewFromInt(1000)).Mul(dGroupRatio).Mul(dQuotaPerUnit)
		}
	}

	var dImageGenerationCallQuota decimal.Decimal
	if ctx.GetBool("image_generation_call") {
		summary.ImageGenerationCallPrice = operation_setting.GetGPTImage1PriceOnceCall(ctx.GetString("image_generation_call_quality"), ctx.GetString("image_generation_call_size"))
		dImageGenerationCallQuota = decimal.NewFromFloat(summary.ImageGenerationCallPrice).Mul(dGroupRatio).Mul(dQuotaPerUnit)
	}

	var audioInputQuota decimal.Decimal
	if !relayInfo.PriceData.UsePrice {
		baseTokens := dPromptTokens

		// 对于非 Claude 语义计费：各类 token 从 baseTokens 中剔除，单独按各自有效倍率计费。
		// 对于 Claude 语义计费：upstream 已将缓存 token 排除在 PromptTokens 之外，
		// 故 baseTokens 不再减去缓存 token；缓存创建 token 仍需剔除以避免重复计费。
		if !dCacheTokens.IsZero() {
			if !summary.IsClaudeUsageSemantic && !legacyClaudeDerived {
				baseTokens = baseTokens.Sub(dCacheTokens)
			}
		}

		hasSplitCacheCreationTokens := summary.CacheCreationTokens5m > 0 || summary.CacheCreationTokens1h > 0
		if !dCachedCreationTokens.IsZero() || hasSplitCacheCreationTokens {
			if !summary.IsClaudeUsageSemantic && !legacyClaudeDerived {
				baseTokens = baseTokens.Sub(dCachedCreationTokens)
			}
		}

		var imageTokensWithRatio decimal.Decimal
		if !dImageTokens.IsZero() {
			baseTokens = baseTokens.Sub(dImageTokens)
			imageTokensWithRatio = dImageTokens.Mul(dImageRatio)
		}

		if !dAudioTokens.IsZero() {
			summary.AudioInputPrice = operation_setting.GetGeminiInputAudioPricePerMillionTokens(summary.ModelName)
			if summary.AudioInputPrice > 0 {
				baseTokens = baseTokens.Sub(dAudioTokens)
				audioInputQuota = decimal.NewFromFloat(summary.AudioInputPrice).
					Div(decimal.NewFromInt(1000000)).Mul(dAudioTokens).Mul(dGroupRatio).Mul(dQuotaPerUnit)
			}
		}

		// 阶梯倍率应用于 token 数量层
		channelID := relayChannelID(relayInfo)
		inputQuota := baseTokens
		if modelTier, ok := ratio_setting.ResolveModelTierRatio(channelID, summary.ModelName); ok {
			inputQuota = ratio_setting.ApplyTierSegmentsForType(inputQuota, modelTier)
			summary.RequestTierPricing = true
		}

		completionTokensAdj := dCompletionTokens
		if completionTier, ok := ratio_setting.ResolveCompletionTierRatio(channelID, summary.ModelName); ok {
			completionTokensAdj = ratio_setting.ApplyTierSegmentsForType(completionTokensAdj, completionTier)
			summary.RequestTierPricing = true
		}

		// 缓存读取 token（含阶梯）
		cacheReadTokensAdj := dCacheTokens
		if !dCacheTokens.IsZero() {
			if cacheTier, ok := ratio_setting.ResolveCacheTierRatio(channelID, summary.ModelName); ok {
				cacheReadTokensAdj = ratio_setting.ApplyTierSegmentsForType(cacheReadTokensAdj, cacheTier)
				summary.RequestTierPricing = true
			}
		}

		// 缓存创建 token（非拆分，含阶梯）
		cacheWriteTokensAdj := dCachedCreationTokens
		if !dCachedCreationTokens.IsZero() && !hasSplitCacheCreationTokens {
			if createCacheTier, ok := ratio_setting.ResolveCreateCacheTierRatio(channelID, summary.ModelName); ok {
				cacheWriteTokensAdj = ratio_setting.ApplyTierSegmentsForType(cacheWriteTokensAdj, createCacheTier)
				summary.RequestTierPricing = true
			}
		}

		// ============================================================
		// 新计费公式（token-based）：各类型使用独立有效倍率
		//   输入     = (ch.model_ratio × costDisc% + globalMr × markupDisc%) × groupRatio（扣费：tokens×有效倍率×groupRatio）
		//   输出     = (ch.model_ratio × completionRatio × costDisc% + globalMr × globalCR × markupDisc%) × groupRatio
		//   缓存读取  = (ch.model_ratio × cacheRatio × costDisc% + globalMr × globalCacheR × markupDisc%) × groupRatio
		//   缓存创建  = (ch.model_ratio × createCacheRatio × costDisc% + globalMr × globalCreateCacheR × markupDisc%) × groupRatio
		// ============================================================
		effInputRate := model.EffectiveInputRate(summary.ModelRatio, summary.GlobalModelRatio, summary.CostDiscountPercent, summary.MarkupDiscountPercent)
		effOutputRate := model.EffectiveOutputRate(summary.ModelRatio, summary.CompletionRatio, summary.GlobalModelRatio, summary.GlobalCompletionRatio, summary.CostDiscountPercent, summary.MarkupDiscountPercent)
		effCacheReadRate := model.EffectiveCacheReadRate(summary.ModelRatio, summary.CacheRatio, summary.GlobalModelRatio, summary.GlobalCacheRatio, summary.CostDiscountPercent, summary.MarkupDiscountPercent)
		effCacheCreateRate := model.EffectiveCacheCreationRate(summary.ModelRatio, summary.CacheCreationRatio, summary.GlobalModelRatio, summary.GlobalCreateCacheRatio, summary.CostDiscountPercent, summary.MarkupDiscountPercent)
		effCacheCreate5mRate := model.EffectiveCacheCreationRate(summary.ModelRatio, summary.CacheCreationRatio5m, summary.GlobalModelRatio, summary.GlobalCreateCacheRatio, summary.CostDiscountPercent, summary.MarkupDiscountPercent)
		// 1h 缓存写入全局倍率 = 5m 全局倍率 × claudeCacheCreation1hMultiplier
		const claudeCacheCreate1hMult = 6.0 / 3.75
		effCacheCreate1hRate := model.EffectiveCacheCreationRate(summary.ModelRatio, summary.CacheCreationRatio1h, summary.GlobalModelRatio, summary.GlobalCreateCacheRatio*claudeCacheCreate1hMult, summary.CostDiscountPercent, summary.MarkupDiscountPercent)

		dEffInputRate := decimal.NewFromFloat(effInputRate)
		dEffOutputRate := decimal.NewFromFloat(effOutputRate)
		dEffCacheReadRate := decimal.NewFromFloat(effCacheReadRate)
		dEffCacheCreateRate := decimal.NewFromFloat(effCacheCreateRate)
		dEffCacheCreate5mRate := decimal.NewFromFloat(effCacheCreate5mRate)
		dEffCacheCreate1hRate := decimal.NewFromFloat(effCacheCreate1hRate)

		// 输入侧：纯输入 token + 图片 token（图片已乘 imageRatio，再乘有效输入倍率）
		inputSideTotal := inputQuota.Add(imageTokensWithRatio).Mul(dEffInputRate).Mul(dGroupRatio)

		// 缓存读取侧：独立有效缓存读取倍率
		var cacheReadSideTotal decimal.Decimal
		if !dCacheTokens.IsZero() {
			cacheReadSideTotal = cacheReadTokensAdj.Mul(dEffCacheReadRate).Mul(dGroupRatio)
		}

		// 缓存创建侧：区分 Claude 5m/1h 拆分和非拆分场景
		var cacheWriteSideTotal decimal.Decimal
		if hasSplitCacheCreationTokens {
			// Claude 语义拆分计费（5m/1h）
			remaining := summary.CacheCreationTokens - summary.CacheCreationTokens5m - summary.CacheCreationTokens1h
			if remaining < 0 {
				remaining = 0
			}
			if summary.IsClaudeUsageSemantic || legacyClaudeDerived {
				// Claude 语义：缓存创建 token 已计入 baseTokens（按输入倍率计费），
				// 此处仅补充与输入倍率的差价（premium）。
				premiumRate5m := effCacheCreate5mRate - effInputRate
				premiumRate1h := effCacheCreate1hRate - effInputRate
				cacheWriteSideTotal = decimal.NewFromInt(int64(remaining)).Mul(decimal.NewFromFloat(premiumRate5m)).
					Add(decimal.NewFromInt(int64(summary.CacheCreationTokens5m)).Mul(decimal.NewFromFloat(premiumRate5m))).
					Add(decimal.NewFromInt(int64(summary.CacheCreationTokens1h)).Mul(decimal.NewFromFloat(premiumRate1h))).
					Mul(dGroupRatio)
			} else {
				// 非 Claude 语义：缓存创建 token 已从 baseTokens 中剔除，按完整有效倍率计费。
				cacheWriteSideTotal = decimal.NewFromInt(int64(remaining)).Mul(dEffCacheCreate5mRate).
					Add(decimal.NewFromInt(int64(summary.CacheCreationTokens5m)).Mul(dEffCacheCreate5mRate)).
					Add(decimal.NewFromInt(int64(summary.CacheCreationTokens1h)).Mul(dEffCacheCreate1hRate)).
					Mul(dGroupRatio)
			}
		} else if !dCachedCreationTokens.IsZero() {
			if summary.IsClaudeUsageSemantic || legacyClaudeDerived {
				premiumRate5m := effCacheCreate5mRate - effInputRate
				cacheWriteSideTotal = cacheWriteTokensAdj.Mul(decimal.NewFromFloat(premiumRate5m)).Mul(dGroupRatio)
			} else {
				cacheWriteSideTotal = cacheWriteTokensAdj.Mul(dEffCacheCreate5mRate).Mul(dGroupRatio)
			}
		}

		if hasSplitCacheCreationTokens {
			remaining := summary.CacheCreationTokens - summary.CacheCreationTokens5m - summary.CacheCreationTokens1h
			if remaining < 0 {
				remaining = 0
			}
			cacheWriteSideTotal = decimal.NewFromInt(int64(remaining)).Mul(dEffCacheCreateRate).
				Add(decimal.NewFromInt(int64(summary.CacheCreationTokens5m)).Mul(dEffCacheCreate5mRate)).
				Add(decimal.NewFromInt(int64(summary.CacheCreationTokens1h)).Mul(dEffCacheCreate1hRate)).
				Mul(dGroupRatio)
		} else if !dCachedCreationTokens.IsZero() {
			cacheWriteSideTotal = cacheWriteTokensAdj.Mul(dEffCacheCreateRate).Mul(dGroupRatio)
		}

		// 输出侧
		outputSideTotal := completionTokensAdj.Mul(dEffOutputRate).Mul(dGroupRatio)

		quotaCalculateDecimal := inputSideTotal.Add(cacheReadSideTotal).Add(cacheWriteSideTotal).Add(outputSideTotal)
		quotaCalculateDecimal = quotaCalculateDecimal.Add(dWebSearchQuota)
		quotaCalculateDecimal = quotaCalculateDecimal.Add(dClaudeWebSearchQuota)
		quotaCalculateDecimal = quotaCalculateDecimal.Add(dFileSearchQuota)
		quotaCalculateDecimal = quotaCalculateDecimal.Add(audioInputQuota)
		quotaCalculateDecimal = quotaCalculateDecimal.Add(dImageGenerationCallQuota)

		if len(relayInfo.PriceData.OtherRatios) > 0 {
			for _, otherRatio := range relayInfo.PriceData.OtherRatios {
				quotaCalculateDecimal = quotaCalculateDecimal.Mul(decimal.NewFromFloat(otherRatio))
			}
		}

		if effInputRate > 0 && quotaCalculateDecimal.LessThanOrEqual(decimal.Zero) {
			quotaCalculateDecimal = decimal.NewFromInt(1)
		}
		summary.Quota = int(quotaCalculateDecimal.Round(0).IntPart())

		// 阶梯计费展示字段：按命中档位区间倍率计算平台单价（$/1M tokens），与定价页阶梯表一致
		if summary.RequestTierPricing {
			inputTierPair := ratio_setting.LoadTierSegmentPair(
				channelID, summary.ModelName,
				ratio_setting.GetChannelModelTierRatio,
				ratio_setting.GetModelTierRatio,
			)
			bandSegments := ratio_setting.SelectBandSegments(inputTierPair)
			bandFrom := ratio_setting.FindTierBandFromTokens(int64(summary.PromptTokens), bandSegments)
			costDisc := summary.CostDiscountPercent
			markupDisc := summary.MarkupDiscountPercent
			groupRatio := summary.GroupRatio

			if len(inputTierPair.Channel) > 0 || len(inputTierPair.Global) > 0 {
				summary.TierInputUnitPrice = ratio_setting.TierPlatformUsdPerM(
					inputTierPair.Channel, inputTierPair.Global, bandFrom, costDisc, markupDisc, groupRatio,
				)
			}

			outputTierPair := ratio_setting.LoadTierSegmentPair(
				channelID, summary.ModelName,
				ratio_setting.GetChannelCompletionTierRatio,
				ratio_setting.GetCompletionTierRatio,
			)
			if len(outputTierPair.Channel) > 0 || len(outputTierPair.Global) > 0 {
				summary.TierOutputUnitPrice = ratio_setting.TierPlatformUsdPerM(
					outputTierPair.Channel, outputTierPair.Global, bandFrom, costDisc, markupDisc, groupRatio,
				)
			}

			cacheReadTierPair := ratio_setting.LoadTierSegmentPair(
				channelID, summary.ModelName,
				ratio_setting.GetChannelCacheTierRatio,
				ratio_setting.GetCacheTierRatio,
			)
			if len(cacheReadTierPair.Channel) > 0 || len(cacheReadTierPair.Global) > 0 {
				summary.TierCacheReadUnitPrice = ratio_setting.TierPlatformUsdPerM(
					cacheReadTierPair.Channel, cacheReadTierPair.Global, bandFrom, costDisc, markupDisc, groupRatio,
				)
			}

			cacheWriteTierPair := ratio_setting.LoadTierSegmentPair(
				channelID, summary.ModelName,
				ratio_setting.GetChannelCreateCacheTierRatio,
				ratio_setting.GetCreateCacheTierRatio,
			)
			if len(cacheWriteTierPair.Channel) > 0 || len(cacheWriteTierPair.Global) > 0 {
				summary.TierCacheWriteUnitPrice = ratio_setting.TierPlatformUsdPerM(
					cacheWriteTierPair.Channel, cacheWriteTierPair.Global, bandFrom, costDisc, markupDisc, groupRatio,
				)
			}

			// 档位判定只依据输入 Token，所有类型共用同一档位标签
			modelTier, modelTierOk := ratio_setting.ResolveModelTierRatio(channelID, summary.ModelName)
			completionTier, compTierOk := ratio_setting.ResolveCompletionTierRatio(channelID, summary.ModelName)
			cacheTier, cacheTierOk := ratio_setting.ResolveCacheTierRatio(channelID, summary.ModelName)
			createCacheTier, createCacheTierOk := ratio_setting.ResolveCreateCacheTierRatio(channelID, summary.ModelName)

			if modelTierOk && len(modelTier.Segments) > 0 {
				tierLabel := ratio_setting.BuildTierLabel("输入", modelTier.Segments, int64(summary.PromptTokens))
				summary.TierInputLabel = tierLabel
				if compTierOk && len(completionTier.Segments) > 0 {
					summary.TierOutputLabel = tierLabel
				}
				if cacheTierOk && len(cacheTier.Segments) > 0 {
					summary.TierCacheReadLabel = tierLabel
				}
				if createCacheTierOk && len(createCacheTier.Segments) > 0 {
					summary.TierCacheWriteLabel = tierLabel
				}
			}
		}
	} else {
		// ============================================================
		// 新计费公式（固定价格）：
		//   模型固定价格 = 渠道固定价 * 成本折扣率% + 全局固定价 * 加价折扣率%
		// ============================================================
		effModelPrice := model.EffectiveModelPrice(summary.ModelPrice, summary.GlobalModelPrice, summary.CostDiscountPercent, summary.MarkupDiscountPercent)
		quotaCalculateDecimal := decimal.NewFromFloat(effModelPrice).Mul(dQuotaPerUnit).Mul(dGroupRatio)
		quotaCalculateDecimal = quotaCalculateDecimal.Add(dWebSearchQuota)
		quotaCalculateDecimal = quotaCalculateDecimal.Add(dClaudeWebSearchQuota)
		quotaCalculateDecimal = quotaCalculateDecimal.Add(dFileSearchQuota)
		quotaCalculateDecimal = quotaCalculateDecimal.Add(audioInputQuota)
		quotaCalculateDecimal = quotaCalculateDecimal.Add(dImageGenerationCallQuota)
		if len(relayInfo.PriceData.OtherRatios) > 0 {
			for _, otherRatio := range relayInfo.PriceData.OtherRatios {
				quotaCalculateDecimal = quotaCalculateDecimal.Mul(decimal.NewFromFloat(otherRatio))
			}
		}
		summary.Quota = int(quotaCalculateDecimal.Round(0).IntPart())
	}

	// 新公式中折扣已内嵌，不再单独调用 ApplyChannelPriceDiscountToQuota
	if summary.TotalTokens == 0 {
		summary.Quota = 0
	} else if summary.Quota == 0 && (summary.ModelRatio > 0 || summary.ModelPrice > 0) {
		summary.Quota = 1
	}

	return summary
}

func textQuotaSummaryWithMarkupOverride(ctx *gin.Context, relayInfo *relaycommon.RelayInfo, usage *dto.Usage, markupPercent float64) textQuotaSummary {
	if relayInfo == nil {
		return textQuotaSummary{}
	}
	pd := relayInfo.PriceData
	pd.MarkupDiscountPercent = markupPercent
	ri := *relayInfo
	ri.PriceData = pd
	return calculateTextQuotaSummary(ctx, &ri, usage)
}

func tryPostWalletProfitShareCredit(ctx *gin.Context, relayInfo *relaycommon.RelayInfo, usage *dto.Usage, summary *textQuotaSummary) {
	if relayInfo == nil || summary == nil {
		return
	}
	if !common.IsDistributorProfitShareMode() {
		return
	}
	bs := strings.TrimSpace(relayInfo.BillingSource)
	if bs != "" && bs != BillingSourceWallet {
		return
	}
	if summary.Quota <= 0 {
		return
	}
	if summary.TotalTokens == 0 && !relayInfo.PriceData.UsePrice {
		return
	}
	invitee, err := model.GetUserById(relayInfo.UserId, false)
	if err != nil || invitee == nil || invitee.InviterId <= 0 {
		return
	}
	inviter, err2 := model.GetUserById(invitee.InviterId, false)
	if err2 != nil || inviter == nil || !model.UserIsDistributor(inviter) {
		return
	}
	s0 := textQuotaSummaryWithMarkupOverride(ctx, relayInfo, usage, 0)
	slice := summary.Quota - s0.Quota
	if slice <= 0 {
		return
	}
	bps := model.EffectiveAffiliateCommissionBps(inviter, relayInfo.UserId)
	if bps <= 0 {
		return
	}
	const maxAffBps = 10000
	if bps > maxAffBps {
		bps = maxAffBps
	}
	reward := int(int64(slice) * int64(bps) / int64(maxAffBps))
	if reward <= 0 {
		return
	}
	chID := 0
	if relayInfo.ChannelMeta != nil {
		chID = relayInfo.ChannelId
	}
	modelName := strings.TrimSpace(summary.ModelName)
	if err := model.CreditDistributorProfitShare(invitee.InviterId, relayInfo.UserId, chID, modelName, summary.Quota, slice, reward, bps, summary.TotalTokens, "text"); err != nil {
		common.SysError("tryPostWalletProfitShareCredit: " + err.Error())
	}
}

func usageSemanticFromUsage(relayInfo *relaycommon.RelayInfo, usage *dto.Usage) string {
	if usage != nil && usage.UsageSemantic != "" {
		return usage.UsageSemantic
	}
	if relayInfo != nil && relayInfo.GetFinalRequestRelayFormat() == types.RelayFormatClaude {
		return "anthropic"
	}
	return "openai"
}

func PostTextConsumeQuota(ctx *gin.Context, relayInfo *relaycommon.RelayInfo, usage *dto.Usage, extraContent []string) {
	originUsage := usage
	if usage == nil {
		extraContent = append(extraContent, "上游无计费信息")
	}
	if originUsage != nil {
		ObserveChannelAffinityUsageCacheByRelayFormat(ctx, usage, relayInfo.GetFinalRequestRelayFormat())
	}

	adminRejectReason := common.GetContextKeyString(ctx, constant.ContextKeyAdminRejectReason)
	summary := calculateTextQuotaSummary(ctx, relayInfo, usage)

	if summary.WebSearchCallCount > 0 {
		extraContent = append(extraContent, fmt.Sprintf("Web Search 调用 %d 次，调用花费 %s", summary.WebSearchCallCount, decimal.NewFromFloat(summary.WebSearchPrice).Mul(decimal.NewFromInt(int64(summary.WebSearchCallCount))).Div(decimal.NewFromInt(1000)).Mul(decimal.NewFromFloat(summary.GroupRatio)).Mul(decimal.NewFromFloat(common.QuotaPerUnit)).String()))
	}
	if summary.ClaudeWebSearchCallCount > 0 {
		extraContent = append(extraContent, fmt.Sprintf("Claude Web Search 调用 %d 次，调用花费 %s", summary.ClaudeWebSearchCallCount, decimal.NewFromFloat(summary.ClaudeWebSearchPrice).Div(decimal.NewFromInt(1000)).Mul(decimal.NewFromFloat(summary.GroupRatio)).Mul(decimal.NewFromFloat(common.QuotaPerUnit)).Mul(decimal.NewFromInt(int64(summary.ClaudeWebSearchCallCount))).String()))
	}
	if summary.FileSearchCallCount > 0 {
		extraContent = append(extraContent, fmt.Sprintf("File Search 调用 %d 次，调用花费 %s", summary.FileSearchCallCount, decimal.NewFromFloat(summary.FileSearchPrice).Mul(decimal.NewFromInt(int64(summary.FileSearchCallCount))).Div(decimal.NewFromInt(1000)).Mul(decimal.NewFromFloat(summary.GroupRatio)).Mul(decimal.NewFromFloat(common.QuotaPerUnit)).String()))
	}
	if summary.AudioInputPrice > 0 && summary.AudioTokens > 0 {
		extraContent = append(extraContent, fmt.Sprintf("Audio Input 花费 %s", decimal.NewFromFloat(summary.AudioInputPrice).Div(decimal.NewFromInt(1000000)).Mul(decimal.NewFromInt(int64(summary.AudioTokens))).Mul(decimal.NewFromFloat(summary.GroupRatio)).Mul(decimal.NewFromFloat(common.QuotaPerUnit)).String()))
	}
	if summary.ImageGenerationCallPrice > 0 {
		extraContent = append(extraContent, fmt.Sprintf("Image Generation Call 花费 %s", decimal.NewFromFloat(summary.ImageGenerationCallPrice).Mul(decimal.NewFromFloat(summary.GroupRatio)).Mul(decimal.NewFromFloat(common.QuotaPerUnit)).String()))
	}

	if summary.TotalTokens == 0 {
		extraContent = append(extraContent, "上游没有返回计费信息，无法扣费（可能是上游超时）")
		logger.LogError(ctx, fmt.Sprintf("total tokens is 0, cannot consume quota, userId %d, channelId %d, tokenId %d, model %s， pre-consumed quota %d", relayInfo.UserId, relayInfo.ChannelId, relayInfo.TokenId, summary.ModelName, relayInfo.FinalPreConsumedQuota))
	} else {
		model.UpdateUserUsedQuotaAndRequestCount(relayInfo.UserId, summary.Quota)
		model.UpdateChannelUsedQuota(relayInfo.ChannelId, summary.Quota)
	}

	if err := SettleBilling(ctx, relayInfo, summary.Quota); err != nil {
		logger.LogError(ctx, "error settling billing: "+err.Error())
	} else {
		tryPostWalletProfitShareCredit(ctx, relayInfo, usage, &summary)
	}

	logModel := summary.ModelName
	if strings.HasPrefix(logModel, "gpt-4-gizmo") {
		logModel = "gpt-4-gizmo-*"
		extraContent = append(extraContent, fmt.Sprintf("模型 %s", summary.ModelName))
	}
	if strings.HasPrefix(logModel, "gpt-4o-gizmo") {
		logModel = "gpt-4o-gizmo-*"
		extraContent = append(extraContent, fmt.Sprintf("模型 %s", summary.ModelName))
	}

	logContent := strings.Join(extraContent, ", ")
	var other map[string]interface{}
	if summary.IsClaudeUsageSemantic {
		other = GenerateClaudeOtherInfo(ctx, relayInfo,
			summary.ModelRatio, summary.GroupRatio, summary.CompletionRatio,
			summary.CacheTokens, summary.CacheRatio,
			summary.CacheCreationTokens, summary.CacheCreationRatio,
			summary.CacheCreationTokens5m, summary.CacheCreationRatio5m,
			summary.CacheCreationTokens1h, summary.CacheCreationRatio1h,
			summary.ModelPrice, relayInfo.PriceData.GroupRatioInfo.GroupSpecialRatio)
		other["usage_semantic"] = "anthropic"
	} else {
		other = GenerateTextOtherInfo(ctx, relayInfo, summary.ModelRatio, summary.GroupRatio, summary.CompletionRatio, summary.CacheTokens, summary.CacheRatio, summary.ModelPrice, relayInfo.PriceData.GroupRatioInfo.GroupSpecialRatio)
	}
	if adminRejectReason != "" {
		other["reject_reason"] = adminRejectReason
	}
	if summary.ImageTokens != 0 {
		other["image"] = true
		other["image_ratio"] = summary.ImageRatio
		other["image_output"] = summary.ImageTokens
	}
	if summary.WebSearchCallCount > 0 {
		other["web_search"] = true
		other["web_search_call_count"] = summary.WebSearchCallCount
		other["web_search_price"] = summary.WebSearchPrice
	} else if summary.ClaudeWebSearchCallCount > 0 {
		other["web_search"] = true
		other["web_search_call_count"] = summary.ClaudeWebSearchCallCount
		other["web_search_price"] = summary.ClaudeWebSearchPrice
	}
	if summary.FileSearchCallCount > 0 {
		other["file_search"] = true
		other["file_search_call_count"] = summary.FileSearchCallCount
		other["file_search_price"] = summary.FileSearchPrice
	}
	if summary.AudioInputPrice > 0 && summary.AudioTokens > 0 {
		other["audio_input_seperate_price"] = true
		other["audio_input_token_count"] = summary.AudioTokens
		other["audio_input_price"] = summary.AudioInputPrice
	}
	if summary.ImageGenerationCallPrice > 0 {
		other["image_generation_call"] = true
		other["image_generation_call_price"] = summary.ImageGenerationCallPrice
	}
	if summary.CacheCreationTokens > 0 {
		other["cache_creation_tokens"] = summary.CacheCreationTokens
		other["cache_creation_ratio"] = summary.CacheCreationRatio
	}
	if summary.CacheCreationTokens5m > 0 {
		other["cache_creation_tokens_5m"] = summary.CacheCreationTokens5m
		other["cache_creation_ratio_5m"] = summary.CacheCreationRatio5m
	}
	if summary.CacheCreationTokens1h > 0 {
		other["cache_creation_tokens_1h"] = summary.CacheCreationTokens1h
		other["cache_creation_ratio_1h"] = summary.CacheCreationRatio1h
	}
	cacheWriteTokens := cacheWriteTokensTotal(summary)
	if cacheWriteTokens > 0 {
		// cache_write_tokens: normalized cache creation total for UI display.
		// If split 5m/1h values are present, this is their sum; otherwise it falls back
		// to cache_creation_tokens.
		other["cache_write_tokens"] = cacheWriteTokens
	}
	if summary.RequestTierPricing {
		other["request_tier_pricing"] = true
		other["request_tier_breakdown"] = summary.RequestTierBreakdown
		other["tier_input_label"] = summary.TierInputLabel
		other["tier_output_label"] = summary.TierOutputLabel
		other["tier_cache_read_label"] = summary.TierCacheReadLabel
		other["tier_cache_write_label"] = summary.TierCacheWriteLabel
		other["tier_input_unit_price"] = summary.TierInputUnitPrice
		other["tier_output_unit_price"] = summary.TierOutputUnitPrice
		other["tier_cache_read_unit_price"] = summary.TierCacheReadUnitPrice
		other["tier_cache_write_unit_price"] = summary.TierCacheWriteUnitPrice

		// 构建日志展示文本（固定 6 行格式，小数点保留 6 位）
		// 第一行：阶梯计费
		// 第二行：命中档位（仅依据输入 Token）
		// 第三行：输入价格
		// 第四行：输出价格
		// 第五行：计算公式
		// 第六行：仅供参考
		tierLines := make([]string, 0, 6)

		// 第一行：阶梯计费
		// tierLines = append(tierLines, "阶梯计费")

		// 第二行：命中档位（档位判定只依据输入 Token）
		if summary.TierInputLabel != "" {
			tierLines = append(tierLines, fmt.Sprintf("命中档位：%s", summary.TierInputLabel))
		}

		// 第三行：输入价格
		tierLines = append(tierLines, fmt.Sprintf("输入价格：$%s / 1M tokens", formatTierUsdPrice(summary.TierInputUnitPrice)))

		// 第四行：输出价格
		tierLines = append(tierLines, fmt.Sprintf("输出价格：$%s / 1M tokens", formatTierUsdPrice(summary.TierOutputUnitPrice)))

		// 缓存读取/写入价格（仅在实际使用对应 token 时展示）
		if summary.CacheTokens > 0 && summary.TierCacheReadUnitPrice > 0 {
			tierLines = append(tierLines, fmt.Sprintf("缓存读取价格：$%s / 1M tokens", formatTierUsdPrice(summary.TierCacheReadUnitPrice)))
		}
		cacheWriteTokens := cacheWriteTokensTotal(summary)
		if cacheWriteTokens > 0 && summary.TierCacheWriteUnitPrice > 0 {
			tierLines = append(tierLines, fmt.Sprintf("缓存写入价格：$%s / 1M tokens", formatTierUsdPrice(summary.TierCacheWriteUnitPrice)))
		}

		// 第五行：计算展示公式
		inputUSD := decimal.NewFromFloat(summary.TierInputUnitPrice).Mul(decimal.NewFromInt(int64(summary.PromptTokens))).Div(decimal.NewFromInt(1_000_000))
		outputUSD := decimal.NewFromFloat(summary.TierOutputUnitPrice).Mul(decimal.NewFromInt(int64(summary.CompletionTokens))).Div(decimal.NewFromInt(1_000_000))
		cacheReadUSD := decimal.NewFromFloat(summary.TierCacheReadUnitPrice).Mul(decimal.NewFromInt(int64(summary.CacheTokens))).Div(decimal.NewFromInt(1_000_000))
		cacheWriteUSD := decimal.NewFromFloat(summary.TierCacheWriteUnitPrice).Mul(decimal.NewFromInt(int64(cacheWriteTokens))).Div(decimal.NewFromInt(1_000_000))
		totalUSD := inputUSD.Add(outputUSD).Add(cacheReadUSD).Add(cacheWriteUSD)

		formulaParts := make([]string, 0, 4)
		if summary.PromptTokens > 0 {
			formulaParts = append(formulaParts, fmt.Sprintf("输入 %d tokens / 1M tokens * $%s", summary.PromptTokens, formatTierUsdPrice(summary.TierInputUnitPrice)))
		}
		if summary.CompletionTokens > 0 {
			formulaParts = append(formulaParts, fmt.Sprintf("输出 %d tokens / 1M tokens * $%s", summary.CompletionTokens, formatTierUsdPrice(summary.TierOutputUnitPrice)))
		}
		if summary.CacheTokens > 0 && summary.TierCacheReadUnitPrice > 0 {
			formulaParts = append(formulaParts, fmt.Sprintf("缓存读取 %d tokens / 1M tokens * $%s", summary.CacheTokens, formatTierUsdPrice(summary.TierCacheReadUnitPrice)))
		}
		if cacheWriteTokens > 0 && summary.TierCacheWriteUnitPrice > 0 {
			formulaParts = append(formulaParts, fmt.Sprintf("缓存写入 %d tokens / 1M tokens * $%s", cacheWriteTokens, formatTierUsdPrice(summary.TierCacheWriteUnitPrice)))
		}
		if len(formulaParts) > 0 {
			tierLines = append(tierLines, fmt.Sprintf("(%s) = $%s", strings.Join(formulaParts, " + "), formatTierUsdPrice(totalUSD.InexactFloat64())))
		} else {
			tierLines = append(tierLines, fmt.Sprintf("(=) = $%s", formatTierUsdPrice(totalUSD.InexactFloat64())))
		}

		// 第六行：仅供参考
		tierLines = append(tierLines, "仅供参考，以实际扣费为准")
		other["request_tier_display"] = strings.Join(tierLines, "\n")
	}
	// use_price：与 PriceData.UsePrice 一致，供前端区分按量（token 单价）与按次等计费形态（旧日志无此字段时前端自行推断）
	other["use_price"] = relayInfo.PriceData.UsePrice
	if relayInfo.GetFinalRequestRelayFormat() != types.RelayFormatClaude && usage != nil && usage.UsageSource != "" && usage.InputTokens > 0 {
		// input_tokens_total: explicit normalized total input used by the usage log UI.
		// Only write this field when upstream/current conversion has already provided a
		// reliable total input value and tagged the usage source. Do not infer it from
		// prompt/cache fields here, otherwise old upstream payloads may be double-counted.
		other["input_tokens_total"] = usage.InputTokens
	}

	model.RecordConsumeLog(ctx, relayInfo.UserId, model.RecordConsumeLogParams{
		ChannelId:        relayInfo.ChannelId,
		PromptTokens:     summary.PromptTokens,
		CompletionTokens: summary.CompletionTokens,
		ModelName:        logModel,
		TokenName:        summary.TokenName,
		Quota:            summary.Quota,
		Content:          logContent,
		TokenId:          relayInfo.TokenId,
		UseTimeSeconds:   int(summary.UseTimeSeconds),
		IsStream:         relayInfo.IsStream,
		Group:            relayInfo.UsingGroup,
		Other:            other,
	})
	if !relayInfo.IsChannelTest {
		gopool.Go(func() {
			perfmetrics.RecordRelaySample(relayInfo, true, int64(summary.CompletionTokens), int64(summary.PromptTokens), int64(summary.CacheTokens))
		})
	}
}
