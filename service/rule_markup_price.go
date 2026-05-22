package service

import (
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
