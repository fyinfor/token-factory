package helper

import (
	"strings"

	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

func globalVideoPerSecondUSDForRelay(modelName, mode string, width, height int, hasAudio bool) float64 {
	return service.GlobalVideoPerSecondUSD(modelName, mode, width, height, hasAudio)
}

func globalVideoPerVideoUSDForRelay(modelName, mode string, width, height int, hasAudio bool) float64 {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return 0
	}
	rules, ok := ratio_setting.GetVideoPricingRules(modelName)
	if !ok {
		return 0
	}
	ctx := estimateVideoRequestContextForMode(mode, width, height)
	usd, ok := matchFlatPerVideoUSDRules(ctx, rules)
	if ok && usd > 0 {
		return usd
	}
	switch mode {
	case string(videoBillingModeImageToVideo):
		if p, ok := pickAudioPriceByResolution(ctx, hasAudio, rules.ImageToVideoPerItem); ok {
			return p
		}
	case string(videoBillingModeVideoToVideo):
		if p, ok := pickAudioPriceByResolution(ctx, hasAudio, rules.VideoToVideoPerItem); ok {
			return p
		}
	default:
		if p, ok := pickAudioPriceByResolution(ctx, hasAudio, rules.TextToVideoPerItem); ok {
			return p
		}
	}
	return 0
}

func estimateVideoRequestContextForMode(mode string, width, height int) videoEstimateContext {
	ctx := videoEstimateContext{Width: width, Height: height}
	switch mode {
	case string(videoBillingModeImageToVideo):
		ctx.Mode = videoBillingModeImageToVideo
	case string(videoBillingModeVideoToVideo):
		ctx.Mode = videoBillingModeVideoToVideo
	default:
		ctx.Mode = videoBillingModeTextToVideo
	}
	return ctx
}
