package helper

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

// https://docs.claude.com/en/docs/build-with-claude/prompt-caching#1-hour-cache-duration
const claudeCacheCreation1hMultiplier = 6 / 3.75

func resolveSupplierIDByChannel(info *relaycommon.RelayInfo) int {
	if info == nil || info.ChannelMeta == nil || info.ChannelId <= 0 {
		return 0
	}
	channel, err := model.CacheGetChannel(info.ChannelId)
	if err != nil || channel == nil {
		return 0
	}
	return channel.SupplierApplicationID
}

// HandleGroupRatio checks for "auto_group" in the context and updates the group ratio and relayInfo.UsingGroup if present
func HandleGroupRatio(ctx *gin.Context, relayInfo *relaycommon.RelayInfo) types.GroupRatioInfo {
	groupRatioInfo := types.GroupRatioInfo{
		GroupRatio:        1.0, // default ratio
		GroupSpecialRatio: -1,
	}

	// check auto group
	autoGroup, exists := ctx.Get("auto_group")
	if exists {
		logger.LogDebug(ctx, fmt.Sprintf("final group: %s", autoGroup))
		relayInfo.UsingGroup = autoGroup.(string)
	}

	// check user group special ratio
	userGroupRatio, ok := ratio_setting.GetGroupGroupRatio(relayInfo.UserGroup, relayInfo.UsingGroup)
	if ok {
		// user group special ratio
		groupRatioInfo.GroupSpecialRatio = userGroupRatio
		groupRatioInfo.GroupRatio = userGroupRatio
		groupRatioInfo.HasSpecialRatio = true
	} else {
		// normal group ratio
		groupRatioInfo.GroupRatio = ratio_setting.GetGroupRatio(relayInfo.UsingGroup)
	}

	return groupRatioInfo
}

func ModelPriceHelper(c *gin.Context, info *relaycommon.RelayInfo, promptTokens int, meta *types.TokenCountMeta) (types.PriceData, error) {
	if info == nil {
		return types.PriceData{}, fmt.Errorf("relay info is nil")
	}
	channelID := 0
	if info.ChannelMeta != nil {
		channelID = info.ChannelId
	}
	groupRatioInfo := HandleGroupRatio(c, info)
	supplierID := resolveSupplierIDByChannel(info)
	modelPrice, usePrice := model.ResolveSupplierScopedFixedModelPrice(channelID, supplierID, info.OriginModelName)
	// 归属供应商的渠道：固定价以 supplier_* 独立表优先于用户分组价；非供应商渠道保留分组覆盖。
	if supplierID <= 0 {
		if groupPrice, ok := ratio_setting.GetGroupModelPrice(info.UsingGroup, info.OriginModelName); ok {
			modelPrice = groupPrice
			usePrice = true
		}
	}
	channelVideoRatio, hasChannelVideoRatio := ratio_setting.GetChannelVideoRatio(channelID, info.OriginModelName)
	channelVideoCompletionRatio, hasChannelVideoCompletionRatio := ratio_setting.GetChannelVideoCompletionRatio(channelID, info.OriginModelName)

	var preConsumedQuota int
	var modelRatio float64
	var completionRatio float64
	var cacheRatio float64
	var imageRatio float64
	var cacheCreationRatio float64
	var cacheCreationRatio5m float64
	var cacheCreationRatio1h float64
	var audioRatio float64
	var audioCompletionRatio float64
	var videoRatio float64
	var videoCompletionRatio float64
	var freeModel bool
	if !usePrice {
		preConsumedTokens := common.Max(promptTokens, common.PreConsumedQuota)
		if meta.MaxTokens != 0 {
			preConsumedTokens += meta.MaxTokens
		}
		var success bool
		var matchName string
		modelRatio, success, matchName = model.ResolveSupplierScopedModelRatio(channelID, supplierID, info.OriginModelName)
		// 供应商自有渠道：输入倍率以独立表（及 Resolve 内平台渠道 Option 回退）为准，不被用户分组倍率覆盖。
		if supplierID <= 0 {
			if groupModelRatio, ok := ratio_setting.GetGroupModelRatio(info.UsingGroup, info.OriginModelName); ok {
				modelRatio = groupModelRatio
				success = true
			}
		}
		if !success {
			acceptUnsetRatio := false
			if info.UserSetting.AcceptUnsetRatioModel {
				acceptUnsetRatio = true
			}
			if !acceptUnsetRatio {
				return types.PriceData{}, fmt.Errorf("模型 %s 倍率或价格未配置，请联系管理员设置或开始自用模式；Model %s ratio or price not set, please set or start self-use mode", matchName, matchName)
			}
		}
		// 输出/缓存/图音倍率：ResolveSupplierScoped* 内已为「供应商渠道表 > 供应商全局表 > 平台渠道 Option > 全局」；
		// 此处禁止再次用 channel_* Option 覆盖，否则会同模型下压过供应商独立表。
		completionRatio = model.ResolveSupplierScopedCompletionRatio(channelID, supplierID, info.OriginModelName)
		cacheRatio, cacheCreationRatio = model.ResolveSupplierScopedCacheRatios(channelID, supplierID, info.OriginModelName)
		cacheCreationRatio5m = cacheCreationRatio
		// 固定1h和5min缓存写入价格的比例
		cacheCreationRatio1h = cacheCreationRatio * claudeCacheCreation1hMultiplier

		imageRatio, _ = model.ResolveSupplierScopedImageRatio(channelID, supplierID, info.OriginModelName)
		audioRatio = model.ResolveSupplierScopedAudioRatio(channelID, supplierID, info.OriginModelName)
		audioCompletionRatio = model.ResolveSupplierScopedAudioCompletionRatio(channelID, supplierID, info.OriginModelName)
		// 供应商表暂无 Video 字段：仍采用全局 + 平台渠道 Option（与旧逻辑一致）。
		videoRatio = ratio_setting.GetVideoRatio(info.OriginModelName)
		videoCompletionRatio = ratio_setting.GetVideoCompletionRatio(info.OriginModelName)
		if hasChannelVideoRatio {
			videoRatio = channelVideoRatio
		}
		if hasChannelVideoCompletionRatio {
			videoCompletionRatio = channelVideoCompletionRatio
		}
		ratio := modelRatio * groupRatioInfo.GroupRatio
		dPreConsumedTokens := decimal.NewFromInt(int64(preConsumedTokens))
		if tier, ok := ratio_setting.ResolveModelTierRatio(channelID, info.OriginModelName); ok {
			dPreConsumedTokens = ratio_setting.ApplyTierSegmentsForType(dPreConsumedTokens, tier)
		}
		preConsumedQuota = int(dPreConsumedTokens.Mul(decimal.NewFromFloat(ratio)).Round(0).IntPart())
	} else {
		if meta.ImagePriceRatio != 0 {
			modelPrice = modelPrice * meta.ImagePriceRatio
		}
		preConsumedQuota = int(modelPrice * common.QuotaPerUnit * groupRatioInfo.GroupRatio)
	}

	// check if free model pre-consume is disabled
	if !operation_setting.GetQuotaSetting().EnableFreeModelPreConsume {
		// if model price or ratio is 0, do not pre-consume quota
		if groupRatioInfo.GroupRatio == 0 {
			preConsumedQuota = 0
			freeModel = true
		} else if usePrice {
			if modelPrice == 0 {
				preConsumedQuota = 0
				freeModel = true
			}
		} else {
			if modelRatio == 0 {
				preConsumedQuota = 0
				freeModel = true
			}
		}
	}

	chDisc := model.ResolveChannelPriceDiscountPercent(channelID)
	chDiscCopy := chDisc
	preConsumedQuota = model.ApplyChannelPriceDiscountToQuota(preConsumedQuota, chDisc)

	priceData := types.PriceData{
		FreeModel:            freeModel,
		ModelPrice:           modelPrice,
		ModelRatio:           modelRatio,
		CompletionRatio:      completionRatio,
		GroupRatioInfo:       groupRatioInfo,
		UsePrice:             usePrice,
		CacheRatio:           cacheRatio,
		ImageRatio:           imageRatio,
		AudioRatio:           audioRatio,
		AudioCompletionRatio: audioCompletionRatio,
		VideoRatio:           videoRatio,
		VideoCompletionRatio: videoCompletionRatio,
		CacheCreationRatio:   cacheCreationRatio,
		CacheCreation5mRatio: cacheCreationRatio5m,
		CacheCreation1hRatio: cacheCreationRatio1h,
		ChannelPriceDiscount: &chDiscCopy,
		QuotaToPreConsume:    preConsumedQuota,
	}

	if common.DebugEnabled {
		println(fmt.Sprintf("model_price_helper result: %s", priceData.ToSetting()))
	}
	info.PriceData = priceData
	return priceData, nil
}

// ModelPriceHelperPerCall 按次计费的 PriceHelper (MJ、Task)
func ModelPriceHelperPerCall(c *gin.Context, info *relaycommon.RelayInfo) (types.PriceData, error) {
	if info == nil {
		return types.PriceData{}, fmt.Errorf("relay info is nil")
	}
	channelID := 0
	if info.ChannelMeta != nil {
		channelID = info.ChannelId
	}
	groupRatioInfo := HandleGroupRatio(c, info)

	supplierID := resolveSupplierIDByChannel(info)
	modelPrice, success := model.ResolveSupplierScopedFixedModelPrice(channelID, supplierID, info.OriginModelName)
	if supplierID <= 0 {
		if groupPrice, ok := ratio_setting.GetGroupModelPrice(info.UsingGroup, info.OriginModelName); ok {
			modelPrice = groupPrice
			success = true
		}
	}
	// 如果没有配置价格，检查模型倍率配置
	if !success {

		// 没有配置费用，也要使用默认费用,否则按费率计费模型无法使用
		defaultPrice, ok := ratio_setting.GetDefaultModelPriceMap()[info.OriginModelName]
		if ok {
			modelPrice = defaultPrice
		} else {
			// 没有配置倍率也不接受没配置,那就返回错误
			_, ratioSuccess, matchName := model.ResolveSupplierScopedModelRatio(channelID, supplierID, info.OriginModelName)
			acceptUnsetRatio := false
			if info.UserSetting.AcceptUnsetRatioModel {
				acceptUnsetRatio = true
			}
			if !ratioSuccess && !acceptUnsetRatio {
				return types.PriceData{}, fmt.Errorf("模型 %s 倍率或价格未配置，请联系管理员设置或开始自用模式；Model %s ratio or price not set, please set or start self-use mode", matchName, matchName)
			}
			// 未配置价格但配置了倍率，使用默认预扣价格
			modelPrice = float64(common.PreConsumedQuota) / common.QuotaPerUnit
		}

	}
	quota := int(modelPrice * common.QuotaPerUnit * groupRatioInfo.GroupRatio)

	// 免费模型检测（与 ModelPriceHelper 对齐）
	freeModel := false
	if !operation_setting.GetQuotaSetting().EnableFreeModelPreConsume {
		if groupRatioInfo.GroupRatio == 0 || modelPrice == 0 {
			quota = 0
			freeModel = true
		}
	}
	chDisc := model.ResolveChannelPriceDiscountPercent(channelID)
	chDiscCopy := chDisc
	quota = model.ApplyChannelPriceDiscountToQuota(quota, chDisc)

	priceData := types.PriceData{
		FreeModel:            freeModel,
		ModelPrice:           modelPrice,
		Quota:                quota,
		GroupRatioInfo:       groupRatioInfo,
		ChannelPriceDiscount: &chDiscCopy,
	}
	return priceData, nil
}

// ============================================================================
// Video task pricing
// ============================================================================
//
// Video generation channels (OpenAI Sora /v1/videos, OpenAI-compatible video
// gateway /v1/videos/generations, etc.) are submitted via the task framework
// and historically only supported per-call pricing through ModelPriceHelperPerCall.
//
// ModelPriceHelperVideo extends that with optional token-based pricing using
// VideoRatio / VideoCompletionRatio, so admins who prefer "$/1M token" semantics
// can configure the per-second/per-resolution cost via ratios instead of a
// flat per-video price.
//
// Selection rules (highest priority first), aligned with the ratio UI
// 「按 token 计费 / 按视频计费」:
//
//  1. Token-based pricing when tryVideoTokenPriceData succeeds (VideoRatio /
//     resolution token_price rules + ModelRatio). Takes precedence over
//     per-call ModelPrice / VideoPrice so admins who choose per-token in the
//     console are not overridden by legacy fixed prices.
//
//  2. Per-resolution flat price per video (*_per_video tables), then
//
//  3. Any other per-call tier (supplier ModelPrice, group ModelPrice, VideoPrice,
//     ChannelVideoPrice) -> ModelPriceHelperPerCall (OtherRatios from adaptor).
//
//  4. Nothing matched -> ModelPriceHelperPerCall fallback (error or default).
const (
	// defaultVideoFPS is used when the request body does not pin an explicit
	// fps; 24 matches Seedance / Doubao / most consumer video generators.
	defaultVideoFPS = 24

	// defaultVideoWidth / defaultVideoHeight are the fallback dimensions when
	// the request did not provide a "size" field; 720p portrait keeps quota
	// estimates conservative for the most common Seedance preset.
	defaultVideoWidth  = 720
	defaultVideoHeight = 1280

	// defaultVideoDuration is used when neither metadata.duration, req.Seconds
	// nor req.Duration carry a positive value.
	defaultVideoDuration = 5
)

type videoBillingMode string

const (
	videoBillingModeTextToVideo  videoBillingMode = "text_to_video"
	videoBillingModeImageToVideo videoBillingMode = "image_to_video"
	videoBillingModeVideoToVideo videoBillingMode = "video_to_video"
)

type videoEstimateContext struct {
	Mode            videoBillingMode
	InputTextTokens int
	Width           int
	Height          int
	FPS             int
	DurationSec     int
}

// ModelPriceHelperVideo computes the quota for video generation tasks.
// Video billing only supports the new rule tables:
// 1) per-second rules (ceil(seconds) * unit price)
// 2) per-item rules (fixed price per generated video)
// Legacy token-based / per-call fallback is intentionally disabled.
func ModelPriceHelperVideo(c *gin.Context, info *relaycommon.RelayInfo) (types.PriceData, error) {
	if info == nil {
		return types.PriceData{}, fmt.Errorf("relay info is nil")
	}

	// 1) Per-second rules first (new mode): ceil(seconds) × unit price.
	if priceData, ok, err := tryVideoPerSecondRulesPriceData(c, info); err != nil {
		return types.PriceData{}, err
	} else if ok {
		return priceData, nil
	}

	// 2) Per-item rules.
	if priceData, ok, err := tryVideoPerVideoRulesPriceData(c, info); err != nil {
		return types.PriceData{}, err
	} else if ok {
		return priceData, nil
	}

	// 3) No video rules configured -> explicit "price not set" error.
	matchName := ratio_setting.FormatMatchingModelName(info.OriginModelName)
	if matchName == "" {
		matchName = info.OriginModelName
	}
	return types.PriceData{}, fmt.Errorf("视频模型 %s 未设置价格，请配置按视频秒收费或按视频条数收费规则；Video model %s price not set, please configure per-second or per-item video pricing rules", matchName, matchName)
}

// hasAnyPerCallVideoPrice reports whether any per-call price tier is set for
// this model。与 ModelPriceHelperPerCall 对齐：优先 supplier_* 独立表再回退 Option。
func hasAnyPerCallVideoPrice(channelID, supplierID int, group, modelName string) bool {
	if _, ok := model.ResolveSupplierScopedFixedModelPrice(channelID, supplierID, modelName); ok {
		return true
	}
	if supplierID <= 0 {
		if _, ok := ratio_setting.GetGroupModelPrice(group, modelName); ok {
			return true
		}
	}
	// VideoPrice：专用按次价（与通用 ModelPrice 字段不同）。
	if _, ok := ratio_setting.GetVideoPrice(modelName); ok {
		return true
	}
	if _, ok := ratio_setting.GetChannelVideoPrice(channelID, modelName); ok {
		return true
	}
	return false
}

// tryVideoTokenPriceData attempts to price the request using token ratios.
// Returns (priceData, true, nil) on success; (zero, false, nil) when ratios
// are not configured (caller should fall through); or (zero, false, err) on
// hard failures.
func tryVideoTokenPriceData(c *gin.Context, info *relaycommon.RelayInfo) (types.PriceData, bool, error) {
	channelID := 0
	if info.ChannelMeta != nil {
		channelID = info.ChannelId
	}
	supplierID := resolveSupplierIDByChannel(info)
	modelName := info.OriginModelName

	// 输入倍率：与 ModelPriceHelper 一致，供应商渠道走 ResolveSupplierScoped（独立表优先于渠道 Option）。
	modelRatio, modelRatioOK, _ := model.ResolveSupplierScopedModelRatio(channelID, supplierID, modelName)
	if supplierID <= 0 {
		if r, ok := ratio_setting.GetGroupModelRatio(info.UsingGroup, modelName); ok {
			modelRatio = r
			modelRatioOK = true
		}
	}

	// Resolve videoRatio: channel > global. For legacy pricing without resolution
	// rules, an explicit map entry is required. When VideoPricingRules exist
	// (per-resolution token_price from the UI), that alone is enough signal even
	// if VideoRatio / ChannelVideoRatio were never set.
	videoRatio := ratio_setting.GetVideoRatio(modelName)
	hasVideoRatio := ratio_setting.ContainsVideoRatio(modelName)
	if r, ok := ratio_setting.GetChannelVideoRatio(channelID, modelName); ok {
		videoRatio = r
		hasVideoRatio = true
	}
	if !hasVideoRatio {
		if _, ok := resolveVideoPricingRules(channelID, modelName); ok {
			hasVideoRatio = true
		}
	}

	if !hasVideoRatio {
		return types.PriceData{}, false, nil
	}
	// Without a ModelRatio the entire formula collapses to 0; refuse.
	if !modelRatioOK || modelRatio <= 0 {
		return types.PriceData{}, false, nil
	}

	videoCompletionRatio := ratio_setting.GetVideoCompletionRatio(modelName)
	if r, ok := ratio_setting.GetChannelVideoCompletionRatio(channelID, modelName); ok {
		videoCompletionRatio = r
	}
	if videoCompletionRatio <= 0 {
		videoCompletionRatio = 1.0
	}

	groupRatioInfo := HandleGroupRatio(c, info)

	estimateCtx := estimateVideoRequestContext(c)
	inputTextTokens := estimateCtx.InputTextTokens
	outputVideoTokens := 0
	appliedVideoRatio := videoRatio
	appliedVideoCompletionRatio := videoCompletionRatio
	usedRulePricing := false

	if rules, ok := resolveVideoPricingRules(channelID, modelName); ok {
		if tokens, tokenPrice, ok := estimateVideoTokensWithRules(estimateCtx, rules); ok {
			outputVideoTokens = tokens
			appliedVideoRatio = tokenPrice
			appliedVideoCompletionRatio = 1.0
			usedRulePricing = true
		}
	}
	if outputVideoTokens <= 0 {
		outputVideoTokens = estimateVideoOutputTokens(estimateCtx)
	}

	// Token-weighted quota (mirrors the audio formula in service.calculateAudioQuota).
	weightedTokens := float64(inputTextTokens) +
		float64(outputVideoTokens)*appliedVideoRatio*appliedVideoCompletionRatio
	rawQuota := weightedTokens * modelRatio * groupRatioInfo.GroupRatio

	// Free-model handling: align with ModelPriceHelper.
	freeModel := false
	if !operation_setting.GetQuotaSetting().EnableFreeModelPreConsume {
		if groupRatioInfo.GroupRatio == 0 || modelRatio == 0 {
			rawQuota = 0
			freeModel = true
		}
	}

	chDisc := model.ResolveChannelPriceDiscountPercent(channelID)
	chDiscCopy := chDisc
	quota := model.ApplyChannelPriceDiscountToQuota(int(math.Round(rawQuota)), chDisc)

	// Floor non-zero results at 1 quota unit, matching calculateAudioQuota.
	if !freeModel && weightedTokens > 0 && quota <= 0 && modelRatio > 0 && groupRatioInfo.GroupRatio > 0 {
		quota = 1
	}

	priceData := types.PriceData{
		FreeModel:            freeModel,
		ModelRatio:           modelRatio,
		VideoRatio:           appliedVideoRatio,
		VideoCompletionRatio: appliedVideoCompletionRatio,
		VideoOutputTokens:    outputVideoTokens,
		VideoInputTextTokens: inputTextTokens,
		GroupRatioInfo:       groupRatioInfo,
		Quota:                quota,
		QuotaToPreConsume:    quota,
		ChannelPriceDiscount: &chDiscCopy,
		// UsePrice = true tells relay_task to skip the OtherRatios multiplication
		// loop, since outputVideoTokens already encodes duration and resolution.
		UsePrice: true,
	}
	if common.DebugEnabled {
		branch := "legacy_ratio"
		if usedRulePricing {
			branch = "rule_based"
		}
		logger.LogDebug(c, fmt.Sprintf(
			"[video][token-pricing][%s] model=%s inputTextTokens=%d outputVideoTokens=%d modelRatio=%.4f videoRatio=%.4f videoCompletionRatio=%.4f groupRatio=%.4f -> quota=%d",
			branch,
			modelName, inputTextTokens, outputVideoTokens,
			modelRatio, appliedVideoRatio, appliedVideoCompletionRatio,
			groupRatioInfo.GroupRatio, quota,
		))
	}
	return priceData, true, nil
}

func resolveVideoPricingRules(channelID int, modelName string) (ratio_setting.VideoPricingRules, bool) {
	if rules, ok := ratio_setting.GetChannelVideoPricingRules(channelID, modelName); ok {
		if hasUsableVideoPricingRules(rules) {
			return rules, true
		}
	}
	if rules, ok := ratio_setting.GetVideoPricingRules(modelName); ok {
		if hasUsableVideoPricingRules(rules) {
			return rules, true
		}
	}
	return ratio_setting.VideoPricingRules{}, false
}

func resolveVideoPerVideoPricingRules(channelID int, modelName string) (ratio_setting.VideoPricingRules, bool) {
	if rules, ok := ratio_setting.GetChannelVideoPricingRules(channelID, modelName); ok {
		if ratio_setting.HasUsableVideoPerVideoRules(rules) {
			return rules, true
		}
	}
	if rules, ok := ratio_setting.GetVideoPricingRules(modelName); ok {
		if ratio_setting.HasUsableVideoPerVideoRules(rules) {
			return rules, true
		}
	}
	return ratio_setting.VideoPricingRules{}, false
}

func resolveVideoPerSecondPricingRules(channelID int, modelName string) (ratio_setting.VideoPricingRules, bool) {
	if rules, ok := ratio_setting.GetChannelVideoPricingRules(channelID, modelName); ok {
		if ratio_setting.HasUsableVideoPerSecondRules(rules) {
			return rules, true
		}
	}
	if rules, ok := ratio_setting.GetVideoPricingRules(modelName); ok {
		if ratio_setting.HasUsableVideoPerSecondRules(rules) {
			return rules, true
		}
	}
	return ratio_setting.VideoPricingRules{}, false
}

func pickAudioPriceByResolution(ctx videoEstimateContext, hasAudio bool, rows []ratio_setting.VideoResolutionAudioPriceRule) (float64, bool) {
	if len(rows) == 0 {
		return 0, false
	}
	targetPixels := ctx.Width * ctx.Height
	bestIdx := -1
	minDiffRatio := math.MaxFloat64
	for i := range rows {
		r := rows[i]
		if r.Price <= 0 || r.HasAudio != hasAudio {
			continue
		}
		ruleW, ruleH, ok := parseResolutionFlexible(r.Resolution)
		if !ok {
			continue
		}
		rulePixels := ruleW * ruleH
		if rulePixels <= 0 {
			continue
		}
		diffRatio := math.Abs(float64(targetPixels-rulePixels)) / float64(rulePixels)
		if diffRatio < minDiffRatio {
			minDiffRatio = diffRatio
			bestIdx = i
		}
	}
	if bestIdx < 0 {
		return 0, false
	}
	return rows[bestIdx].Price, true
}

func parseResolutionFlexible(s string) (int, int, bool) {
	raw := strings.ToLower(strings.TrimSpace(s))
	if raw == "" {
		return 0, 0, false
	}
	if w, h, ok := parseResolution(raw); ok {
		return w, h, true
	}
	switch raw {
	case "480p":
		return 854, 480, true
	case "540p":
		return 960, 540, true
	case "720p":
		return 1280, 720, true
	case "1080p":
		return 1920, 1080, true
	case "2k":
		return 2560, 1440, true
	case "4k":
		return 3840, 2160, true
	default:
		return 0, 0, false
	}
}

func requestHasAudio(c *gin.Context) bool {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil || req.Metadata == nil {
		return false
	}
	if v, ok := req.Metadata["has_audio"]; ok {
		switch x := v.(type) {
		case bool:
			return x
		case string:
			return strings.EqualFold(strings.TrimSpace(x), "true")
		}
	}
	if v, ok := req.Metadata["generate_audio"]; ok {
		switch x := v.(type) {
		case bool:
			return x
		case string:
			return strings.EqualFold(strings.TrimSpace(x), "true")
		}
	}
	return false
}

func tryVideoPerSecondRulesPriceData(c *gin.Context, info *relaycommon.RelayInfo) (types.PriceData, bool, error) {
	channelID := 0
	if info.ChannelMeta != nil {
		channelID = info.ChannelId
	}
	rules, ok := resolveVideoPerSecondPricingRules(channelID, info.OriginModelName)
	if !ok {
		return types.PriceData{}, false, nil
	}
	estimateCtx := estimateVideoRequestContext(c)
	hasAudio := requestHasAudio(c)
	seconds := estimateCtx.DurationSec
	if seconds <= 0 {
		seconds = defaultVideoDuration
	}
	seconds = int(math.Ceil(float64(seconds)))

	var pricePerSecond float64
	switch estimateCtx.Mode {
	case videoBillingModeImageToVideo:
		pricePerSecond, ok = pickAudioPriceByResolution(estimateCtx, hasAudio, rules.ImageToVideoPerSecond)
	case videoBillingModeVideoToVideo:
		pricePerSecond, ok = pickAudioPriceByResolution(estimateCtx, hasAudio, rules.VideoToVideoPerSecond)
	default:
		pricePerSecond, ok = pickAudioPriceByResolution(estimateCtx, hasAudio, rules.TextToVideoPerSecond)
	}
	if !ok || pricePerSecond <= 0 {
		return types.PriceData{}, false, nil
	}
	groupRatioInfo := HandleGroupRatio(c, info)
	rawQuota := float64(seconds) * pricePerSecond * common.QuotaPerUnit * groupRatioInfo.GroupRatio
	chDisc := model.ResolveChannelPriceDiscountPercent(channelID)
	chDiscCopy := chDisc
	quota := model.ApplyChannelPriceDiscountToQuota(int(math.Round(rawQuota)), chDisc)
	if quota <= 0 && rawQuota > 0 {
		quota = 1
	}
	pd := types.PriceData{
		ModelPrice:           0,
		ModelRatio:           0,
		GroupRatioInfo:       groupRatioInfo,
		UsePrice:             true,
		Quota:                quota,
		QuotaToPreConsume:    quota,
		ChannelPriceDiscount: &chDiscCopy,
	}
	pd.AddOtherRatio("seconds", float64(seconds))
	if hasAudio {
		pd.AddOtherRatio("has_audio", 1)
	}
	return pd, true, nil
}

// matchPerVideoRulesByPixels picks the resolution row whose WxH is closest to
// the request (same relative pixel error heuristic as token resolution rules).
func matchPerVideoRulesByPixels(ctx videoEstimateContext, rules []ratio_setting.VideoResolutionPerVideoRule) (float64, bool) {
	if len(rules) == 0 || ctx.Width <= 0 || ctx.Height <= 0 {
		return 0, false
	}
	bestIdx := -1
	targetPixels := ctx.Width * ctx.Height
	minDiffRatio := math.MaxFloat64
	for i, rule := range rules {
		if rule.VideoPrice <= 0 {
			continue
		}
		ruleW, ruleH, ok := parseResolution(rule.Resolution)
		if !ok {
			continue
		}
		rulePixels := ruleW * ruleH
		if rulePixels <= 0 {
			continue
		}
		diffRatio := math.Abs(float64(targetPixels-rulePixels)) / float64(rulePixels)
		if diffRatio < minDiffRatio {
			minDiffRatio = diffRatio
			bestIdx = i
		}
	}
	if bestIdx < 0 {
		return 0, false
	}
	return rules[bestIdx].VideoPrice, true
}

func matchFlatPerVideoUSDRules(ctx videoEstimateContext, rules ratio_setting.VideoPricingRules) (float64, bool) {
	switch ctx.Mode {
	case videoBillingModeImageToVideo:
		return matchPerVideoRulesByPixels(ctx, rules.ImageToVideoPerVideo)
	case videoBillingModeVideoToVideo:
		var sum float64
		n := 0
		if u, ok := matchPerVideoRulesByPixels(ctx, rules.VideoToVideoInputPerVideo); ok {
			sum += u
			n++
		}
		if u, ok := matchPerVideoRulesByPixels(ctx, rules.VideoToVideoOutputPerVideo); ok {
			sum += u
			n++
		}
		if n > 0 {
			return sum, true
		}
		return 0, false
	default:
		return matchPerVideoRulesByPixels(ctx, rules.TextToVideoPerVideo)
	}
}

func tryVideoPerVideoRulesPriceData(c *gin.Context, info *relaycommon.RelayInfo) (types.PriceData, bool, error) {
	channelID := 0
	if info.ChannelMeta != nil {
		channelID = info.ChannelId
	}
	modelName := info.OriginModelName

	rules, ok := resolveVideoPerVideoPricingRules(channelID, modelName)
	if !ok {
		return types.PriceData{}, false, nil
	}

	estimateCtx := estimateVideoRequestContext(c)
	hasAudio := requestHasAudio(c)
	usd, okPrice := matchFlatPerVideoUSDRules(estimateCtx, rules)
	if !okPrice || usd <= 0 {
		// New per-item table first; fallback to legacy *_per_video.
		switch estimateCtx.Mode {
		case videoBillingModeImageToVideo:
			usd, okPrice = pickAudioPriceByResolution(estimateCtx, hasAudio, rules.ImageToVideoPerItem)
		case videoBillingModeVideoToVideo:
			usd, okPrice = pickAudioPriceByResolution(estimateCtx, hasAudio, rules.VideoToVideoPerItem)
		default:
			usd, okPrice = pickAudioPriceByResolution(estimateCtx, hasAudio, rules.TextToVideoPerItem)
		}
	}
	if !okPrice || usd <= 0 {
		return types.PriceData{}, false, nil
	}

	groupRatioInfo := HandleGroupRatio(c, info)
	rawQuota := usd * common.QuotaPerUnit * groupRatioInfo.GroupRatio

	freeModel := false
	if !operation_setting.GetQuotaSetting().EnableFreeModelPreConsume {
		if groupRatioInfo.GroupRatio == 0 {
			rawQuota = 0
			freeModel = true
		}
	}

	chDisc := model.ResolveChannelPriceDiscountPercent(channelID)
	chDiscCopy := chDisc
	quota := model.ApplyChannelPriceDiscountToQuota(int(math.Round(rawQuota)), chDisc)

	if !freeModel && quota <= 0 && rawQuota > 0 && groupRatioInfo.GroupRatio > 0 {
		quota = 1
	}

	priceData := types.PriceData{
		FreeModel:            freeModel,
		ModelPrice:           0,
		ModelRatio:           0,
		GroupRatioInfo:       groupRatioInfo,
		UsePrice:             true,
		Quota:                quota,
		QuotaToPreConsume:    quota,
		ChannelPriceDiscount: &chDiscCopy,
	}
	if common.DebugEnabled {
		logger.LogDebug(c, fmt.Sprintf(
			"[video][per-video-rules] model=%s mode=%s w=%d h=%d usd=%.6f groupRatio=%.4f -> quota=%d",
			modelName, estimateCtx.Mode, estimateCtx.Width, estimateCtx.Height, usd, groupRatioInfo.GroupRatio, quota,
		))
	}
	return priceData, true, nil
}

func hasUsableVideoPricingRules(rules ratio_setting.VideoPricingRules) bool {
	if len(rules.TextToVideo) > 0 {
		return true
	}
	if len(rules.ImageToVideoRules) > 0 {
		return true
	}
	if len(rules.VideoToVideoInput) > 0 {
		return true
	}
	if len(rules.VideoToVideoOutput) > 0 {
		return true
	}
	if len(rules.VideoToVideo) > 0 {
		return true
	}
	return rules.ImageToVideo != nil && rules.ImageToVideo.TokenPrice > 0
}

// estimateVideoTokens derives (inputTextTokens, outputVideoTokens) from the
// parsed TaskSubmitReq currently stored in the gin context.
//
// outputVideoTokens follows the formula widely used by Volcano Engine /
// Doubao docs:
//
//	tokens = duration * width * height * fps / 1024
//
// inputTextTokens use a conservative "1 token per 4 prompt characters" heuristic
// that does not require pulling in the heavy tokenizer dependency for every
// task submission. This is intentionally a coarse estimate; real-world video
// pricing is dominated by the output term (it scales with W*H*fps), so prompt
// inaccuracy is negligible.
func estimateVideoRequestContext(c *gin.Context) videoEstimateContext {
	ctx := videoEstimateContext{
		Mode:        videoBillingModeTextToVideo,
		Width:       defaultVideoWidth,
		Height:      defaultVideoHeight,
		FPS:         defaultVideoFPS,
		DurationSec: defaultVideoDuration,
	}
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return ctx
	}

	ctx.Mode = detectVideoBillingMode(&req)
	ctx.DurationSec = resolveVideoDuration(&req)
	ctx.Width, ctx.Height = resolveVideoDimensions(&req)
	ctx.FPS = resolveVideoFPS(&req)

	if prompt := strings.TrimSpace(req.GetPrompt()); prompt != "" {
		// 1 token per ~4 characters is the well-known OpenAI rule of thumb
		// for English; for CJK-heavy prompts it overestimates slightly, which
		// is acceptable here (we err on charging more rather than less).
		ctx.InputTextTokens = int(math.Ceil(float64(len([]rune(prompt))) / 4.0))
	}
	return ctx
}

func estimateVideoOutputTokens(ctx videoEstimateContext) int {
	return videoOutputTokens(ctx.DurationSec, ctx.Width, ctx.Height, ctx.FPS)
}

func detectVideoBillingMode(req *relaycommon.TaskSubmitReq) videoBillingMode {
	if req == nil {
		return videoBillingModeTextToVideo
	}
	if strings.TrimSpace(req.InputReference) != "" {
		return videoBillingModeVideoToVideo
	}
	if strings.TrimSpace(req.Image) != "" {
		return videoBillingModeImageToVideo
	}
	for _, img := range req.Images {
		if strings.TrimSpace(img) != "" {
			return videoBillingModeImageToVideo
		}
	}
	return videoBillingModeTextToVideo
}

func estimateVideoTokensWithRules(ctx videoEstimateContext, rules ratio_setting.VideoPricingRules) (tokens int, tokenPrice float64, ok bool) {
	switch ctx.Mode {
	case videoBillingModeImageToVideo:
		if len(rules.ImageToVideoRules) > 0 {
			return estimateImageByResolutionRules(ctx, rules.ImageToVideoRules, rules.SimilarityThreshold)
		}
		if rules.ImageToVideo != nil && rules.ImageToVideo.TokenPrice > 0 {
			compression := rules.ImageToVideo.PixelCompression
			if compression <= 0 {
				compression = 1024
			}
			raw := float64(ctx.Width*ctx.Height) / compression
			if raw < 1 {
				raw = 1
			}
			return int(math.Ceil(raw)), rules.ImageToVideo.TokenPrice, true
		}
		return estimateByResolutionRules(ctx, rules.TextToVideo, rules.SimilarityThreshold)
	case videoBillingModeVideoToVideo:
		if len(rules.VideoToVideoInput) > 0 || len(rules.VideoToVideoOutput) > 0 {
			inTokens, inPrice, okIn := estimateByResolutionRules(ctx, rules.VideoToVideoInput, rules.SimilarityThreshold)
			outTokens, outPrice, okOut := estimateByResolutionRules(ctx, rules.VideoToVideoOutput, rules.SimilarityThreshold)
			if okIn || okOut {
				weighted := 0.0
				if okIn {
					weighted += float64(inTokens) * inPrice
				}
				if okOut {
					weighted += float64(outTokens) * outPrice
				}
				if weighted > 0 {
					return int(math.Ceil(weighted)), 1.0, true
				}
			}
		}
		if tokens, tokenPrice, ok := estimateByResolutionRules(ctx, rules.VideoToVideo, rules.SimilarityThreshold); ok {
			return tokens, tokenPrice, ok
		}
		return estimateByResolutionRules(ctx, rules.TextToVideo, rules.SimilarityThreshold)
	default:
		return estimateByResolutionRules(ctx, rules.TextToVideo, rules.SimilarityThreshold)
	}
}

func estimateImageByResolutionRules(ctx videoEstimateContext, rules []ratio_setting.VideoResolutionPriceRule, threshold float64) (tokens int, tokenPrice float64, ok bool) {
	if len(rules) == 0 {
		return 0, 0, false
	}
	bestRule := rules[0]
	targetPixels := ctx.Width * ctx.Height
	minDiffRatio := math.MaxFloat64
	for _, rule := range rules {
		if rule.TokenPrice <= 0 {
			continue
		}
		ruleW, ruleH, ok := parseResolution(rule.Resolution)
		if !ok {
			continue
		}
		rulePixels := ruleW * ruleH
		if rulePixels <= 0 {
			continue
		}
		diffRatio := math.Abs(float64(targetPixels-rulePixels)) / float64(rulePixels)
		if diffRatio < minDiffRatio {
			minDiffRatio = diffRatio
			bestRule = rule
		}
	}
	if bestRule.TokenPrice <= 0 {
		return 0, 0, false
	}
	if threshold <= 0 {
		threshold = 0.35
	}
	compression := bestRule.PixelCompression
	if compression <= 0 {
		compression = 1024
	}
	raw := float64(ctx.Width*ctx.Height) / compression
	if raw < 1 {
		raw = 1
	}
	return int(math.Ceil(raw)), bestRule.TokenPrice, true
}

func estimateByResolutionRules(ctx videoEstimateContext, rules []ratio_setting.VideoResolutionPriceRule, threshold float64) (tokens int, tokenPrice float64, ok bool) {
	if len(rules) == 0 {
		return 0, 0, false
	}
	bestRule := rules[0]
	targetPixels := ctx.Width * ctx.Height
	minDiffRatio := math.MaxFloat64
	for _, rule := range rules {
		if rule.TokenPrice <= 0 {
			continue
		}
		ruleW, ruleH, ok := parseResolution(rule.Resolution)
		if !ok {
			continue
		}
		rulePixels := ruleW * ruleH
		if rulePixels <= 0 {
			continue
		}
		diffRatio := math.Abs(float64(targetPixels-rulePixels)) / float64(rulePixels)
		if diffRatio < minDiffRatio {
			minDiffRatio = diffRatio
			bestRule = rule
		}
	}
	if bestRule.TokenPrice <= 0 {
		return 0, 0, false
	}
	if threshold <= 0 {
		threshold = 0.35
	}
	compression := bestRule.PixelCompression
	if compression <= 0 {
		compression = 1024
	}
	_ = minDiffRatio > threshold // 超出阈值时仍按实际分辨率计费，不再强行套固定档位像素。
	raw := float64(ctx.Width*ctx.Height*ctx.FPS*ctx.DurationSec) / compression
	if raw < 1 {
		raw = 1
	}
	return int(math.Ceil(raw)), bestRule.TokenPrice, true
}

func parseResolution(size string) (width, height int, ok bool) {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(size)), "x")
	if len(parts) != 2 {
		return 0, 0, false
	}
	w, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || w <= 0 {
		return 0, 0, false
	}
	h, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil || h <= 0 {
		return 0, 0, false
	}
	return w, h, true
}

// videoOutputTokens is the canonical Volcano-style output-token formula.
// All inputs are positive integers; the result is rounded up to the nearest
// token to avoid silently zeroing out very small clips.
func videoOutputTokens(durationSec, width, height, fps int) int {
	if durationSec <= 0 || width <= 0 || height <= 0 || fps <= 0 {
		return 0
	}
	raw := float64(durationSec) * float64(width) * float64(height) * float64(fps) / 1024.0
	if raw < 1 {
		return 1
	}
	return int(math.Ceil(raw))
}

// resolveVideoDuration prefers metadata.duration (most authoritative, allows
// caller to override) > Seconds > Duration > defaultVideoDuration.
func resolveVideoDuration(req *relaycommon.TaskSubmitReq) int {
	if req.Metadata != nil {
		if v, ok := req.Metadata["duration"]; ok {
			if d := coerceToPositiveInt(v); d > 0 {
				return d
			}
		}
	}
	if s := strings.TrimSpace(req.Seconds); s != "" {
		if d, err := strconv.Atoi(s); err == nil && d > 0 {
			return d
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil && f > 0 {
			return int(math.Ceil(f))
		}
	}
	if req.Duration > 0 {
		return req.Duration
	}
	return defaultVideoDuration
}

// resolveVideoDimensions parses req.Size ("WIDTHxHEIGHT") or falls back to
// metadata.width/metadata.height; defaults at the end keep quota non-zero.
func resolveVideoDimensions(req *relaycommon.TaskSubmitReq) (width, height int) {
	if size := strings.TrimSpace(req.Size); size != "" {
		parts := strings.Split(strings.ToLower(size), "x")
		if len(parts) == 2 {
			if w, err := strconv.Atoi(strings.TrimSpace(parts[0])); err == nil && w > 0 {
				if h, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil && h > 0 {
					return w, h
				}
			}
		}
	}
	if req.Metadata != nil {
		w := coerceToPositiveInt(req.Metadata["width"])
		h := coerceToPositiveInt(req.Metadata["height"])
		if w > 0 && h > 0 {
			return w, h
		}
	}
	return defaultVideoWidth, defaultVideoHeight
}

// resolveVideoFPS uses metadata.fps when present; otherwise the safe default.
func resolveVideoFPS(req *relaycommon.TaskSubmitReq) int {
	if req.Metadata != nil {
		if v := coerceToPositiveInt(req.Metadata["fps"]); v > 0 {
			return v
		}
	}
	return defaultVideoFPS
}

// coerceToPositiveInt turns common JSON-decoded numeric forms (float64, int,
// int64, json.Number, numeric strings) into a positive int, or 0 if absent
// / non-positive / unparseable.
func coerceToPositiveInt(v any) int {
	switch x := v.(type) {
	case nil:
		return 0
	case int:
		if x > 0 {
			return x
		}
	case int64:
		if x > 0 {
			return int(x)
		}
	case float64:
		if x > 0 {
			return int(math.Ceil(x))
		}
	case string:
		if d, err := strconv.Atoi(strings.TrimSpace(x)); err == nil && d > 0 {
			return d
		}
		if f, err := strconv.ParseFloat(strings.TrimSpace(x), 64); err == nil && f > 0 {
			return int(math.Ceil(f))
		}
	}
	return 0
}

func ContainPriceOrRatio(modelName string) bool {
	_, ok := ratio_setting.GetModelPrice(modelName, false)
	if ok {
		return true
	}
	_, ok, _ = ratio_setting.GetModelRatio(modelName)
	if ok {
		return true
	}
	return false
}
