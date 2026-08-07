package service

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

// effectiveVideoPerSecondUSD 视频按秒：渠道/秒 × 成本折扣% + 全局/秒 × 加价折扣%（全局未配时用渠道价作加价基准）。
func effectiveVideoPerSecondUSD(channelPerSec, globalPerSec, costDiscPercent, markupDiscPercent float64) float64 {
	if costDiscPercent <= 0 {
		costDiscPercent = 100
	}
	return model.EffectiveRuleUnitPrice(channelPerSec, globalPerSec, costDiscPercent, markupDiscPercent)
}

// effectiveVideoPerVideoUSD 视频按条：与按秒相同的两档公式（美元/条）。
func effectiveVideoPerVideoUSD(channelUSD, globalUSD, costDiscPercent, markupDiscPercent float64) float64 {
	if costDiscPercent <= 0 {
		costDiscPercent = 100
	}
	return model.EffectiveRuleUnitPrice(channelUSD, globalUSD, costDiscPercent, markupDiscPercent)
}

// effectiveImagePerImageUSD 图片按张：渠道/张 × 成本折扣% + 全局/张 × 加价折扣%。
func effectiveImagePerImageUSD(channelUSD, globalUSD, costDiscPercent, markupDiscPercent float64) float64 {
	if costDiscPercent <= 0 {
		costDiscPercent = 100
	}
	return model.EffectiveRuleUnitPrice(channelUSD, globalUSD, costDiscPercent, markupDiscPercent)
}

func channelVideoPerSecondUSD(channelID int, modelName, mode string, width, height int, hasAudio bool) float64 {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" || channelID <= 0 {
		return 0
	}
	rules, ok := ratio_setting.GetChannelVideoPricingRules(channelID, modelName)
	if !ok {
		return 0
	}
	p, ok := matchPerSecondPrice(rules, mode, width, height, hasAudio)
	if !ok {
		return 0
	}
	return p
}

// GlobalVideoPerSecondUSD 全局视频按秒规则价（美元/秒），供 relay 等包调用。
func GlobalVideoPerSecondUSD(modelName, mode string, width, height int, hasAudio bool) float64 {
	return globalVideoPerSecondUSD(modelName, mode, width, height, hasAudio)
}

func globalVideoPerSecondUSD(modelName, mode string, width, height int, hasAudio bool) float64 {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return 0
	}
	rules, ok := ratio_setting.GetVideoPricingRules(modelName)
	if !ok {
		return 0
	}
	p, ok := matchPerSecondPrice(rules, mode, width, height, hasAudio)
	if !ok {
		return 0
	}
	return p
}

// globalVideoPerSecondUSDForChannelTier 全局价与定价卡片一致：按渠道已匹配档位的分辨率查全局规则，而非按成片像素重匹。
func globalVideoPerSecondUSDForChannelTier(modelName, mode, resolution string, ruleWidth, ruleHeight int, hasAudio, unifiedAudio bool) float64 {
	res := strings.TrimSpace(resolution)
	if res == "" && ruleWidth > 0 && ruleHeight > 0 {
		res = fmt.Sprintf("%dx%d", ruleWidth, ruleHeight)
	}
	if p := model.LookupGlobalVideoPerSecondUSD(modelName, mode, res, hasAudio, unifiedAudio); p > 0 {
		return p
	}
	// 全局无同档时回退按像素匹配（EffectiveRuleUnitPrice 内 global<=0 会用渠道价）
	return globalVideoPerSecondUSD(modelName, mode, ruleWidth, ruleHeight, hasAudio)
}

func channelVideoPerTokenUSD(channelID int, modelName, mode string, width, height int, hasAudio bool) float64 {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" || channelID <= 0 {
		return 0
	}
	rules, ok := ratio_setting.GetChannelVideoPricingRules(channelID, modelName)
	if !ok {
		return 0
	}
	p, ok := matchPerTokenPrice(rules, mode, width, height, hasAudio)
	if !ok {
		return 0
	}
	return p
}

func globalVideoPerTokenUSD(modelName, mode string, width, height int, hasAudio bool) float64 {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return 0
	}
	rules, ok := ratio_setting.GetVideoPricingRules(modelName)
	if !ok {
		return 0
	}
	p, ok := matchPerTokenPrice(rules, mode, width, height, hasAudio)
	if !ok {
		return 0
	}
	return p
}

func globalVideoPerTokenUSDForChannelTier(modelName, mode, resolution string, ruleWidth, ruleHeight int, hasAudio, unifiedAudio bool) float64 {
	res := strings.TrimSpace(resolution)
	if res == "" && ruleWidth > 0 && ruleHeight > 0 {
		res = fmt.Sprintf("%dx%d", ruleWidth, ruleHeight)
	}
	if p := model.LookupGlobalVideoPerTokenUSD(modelName, mode, res, hasAudio, unifiedAudio); p > 0 {
		return p
	}
	return globalVideoPerTokenUSD(modelName, mode, ruleWidth, ruleHeight, hasAudio)
}

// EffectiveVideoPerTokenUSDForDimensions 按请求像素匹配渠道档位后，计算有效 token 单价（美元/token）。
func EffectiveVideoPerTokenUSDForDimensions(
	channelID int,
	modelName, mode string,
	width, height int,
	hasAudio bool,
	costDiscPercent, markupDiscPercent float64,
) (effUSD, channelUSD, globalUSD float64, ok bool) {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return 0, 0, 0, false
	}
	var channelRules ratio_setting.VideoPricingRules
	var okRules bool
	if channelID > 0 {
		channelRules, okRules = ratio_setting.GetChannelVideoPricingRules(channelID, modelName)
	}
	if !okRules || !ratio_setting.HasUsableVideoPerTokenRules(channelRules) {
		channelRules, okRules = ratio_setting.GetVideoPricingRules(modelName)
		if !okRules || !ratio_setting.HasUsableVideoPerTokenRules(channelRules) {
			return 0, 0, 0, false
		}
		channelID = 0
	}
	match, okMatch := matchPerTokenPriceDetail(channelRules, mode, width, height, hasAudio, "")
	if !okMatch || match.PricePerSecond <= 0 {
		return 0, 0, 0, false
	}
	channelUSD = match.PricePerSecond
	globalUSD = globalVideoPerTokenUSDForChannelTier(
		modelName, mode, match.Resolution, match.RuleWidth, match.RuleHeight, hasAudio, match.UnifiedAudio,
	)
	effUSD = effectiveVideoPerSecondUSD(channelUSD, globalUSD, costDiscPercent, markupDiscPercent)
	return effUSD, channelUSD, globalUSD, true
}

// EffectiveVideoPerSecondUSDForDimensions 按 resolution 标识（优先）或像素匹配档位后计算有效单价。
// 优先渠道 ChannelVideoPricingRules；无可用按秒档位时回退全局 VideoPricingRules（与按 token 路径一致），
// 仍套用成本折扣% + 加价折扣%，避免全局垫底按原价实扣、日志却展示折扣单价。
func EffectiveVideoPerSecondUSDForDimensions(
	channelID int,
	modelName, mode string,
	width, height int,
	hasAudio bool,
	costDiscPercent, markupDiscPercent float64,
	resolutionLabel string,
) (effUSD, channelUSD, globalUSD float64, ok bool) {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return 0, 0, 0, false
	}
	var rules ratio_setting.VideoPricingRules
	var okRules bool
	if channelID > 0 {
		rules, okRules = ratio_setting.GetChannelVideoPricingRules(channelID, modelName)
	}
	if !okRules || !ratio_setting.HasUsableVideoPerSecondRules(rules) {
		rules, okRules = ratio_setting.GetVideoPricingRules(modelName)
		if !okRules || !ratio_setting.HasUsableVideoPerSecondRules(rules) {
			return 0, 0, 0, false
		}
	}
	match, okMatch := MatchVideoPerSecondUnitPrice(rules, mode, width, height, hasAudio, resolutionLabel)
	if !okMatch || match.PricePerSecond <= 0 {
		return 0, 0, 0, false
	}
	channelUSD = match.PricePerSecond
	globalUSD = globalVideoPerSecondUSDForChannelTier(
		modelName, mode, match.Resolution, match.RuleWidth, match.RuleHeight, hasAudio, match.UnifiedAudio,
	)
	effUSD = effectiveVideoPerSecondUSD(channelUSD, globalUSD, costDiscPercent, markupDiscPercent)
	return effUSD, channelUSD, globalUSD, true
}
