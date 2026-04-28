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
	modelPrice, usePrice := ratio_setting.GetModelPrice(info.OriginModelName, false)
	if channelPrice, ok := ratio_setting.GetChannelModelPrice(channelID, info.OriginModelName); ok {
		modelPrice = channelPrice
		usePrice = true
	}
	supplierID := resolveSupplierIDByChannel(info)
	if supplierPrice, ok := ratio_setting.GetSupplierModelPrice(supplierID, info.OriginModelName); ok {
		modelPrice = supplierPrice
		usePrice = true
	}
	if groupPrice, ok := ratio_setting.GetGroupModelPrice(info.UsingGroup, info.OriginModelName); ok {
		modelPrice = groupPrice
		usePrice = true
	}
	channelCompletionRatio, hasChannelCompletionRatio := ratio_setting.GetChannelCompletionRatio(channelID, info.OriginModelName)
	channelCacheRatio, hasChannelCacheRatio := ratio_setting.GetChannelCacheRatio(channelID, info.OriginModelName)
	channelCreateCacheRatio, hasChannelCreateCacheRatio := ratio_setting.GetChannelCreateCacheRatio(channelID, info.OriginModelName)
	channelImageRatio, hasChannelImageRatio := ratio_setting.GetChannelImageRatio(channelID, info.OriginModelName)
	channelAudioRatio, hasChannelAudioRatio := ratio_setting.GetChannelAudioRatio(channelID, info.OriginModelName)
	channelAudioCompletionRatio, hasChannelAudioCompletionRatio := ratio_setting.GetChannelAudioCompletionRatio(channelID, info.OriginModelName)
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
		modelRatio, success, matchName = ratio_setting.GetModelRatio(info.OriginModelName)
		if channelRatio, ok := ratio_setting.GetChannelModelRatio(channelID, info.OriginModelName); ok {
			modelRatio = channelRatio
			success = true
		}
		if supplierRatio, ok := ratio_setting.GetSupplierModelRatio(supplierID, info.OriginModelName); ok {
			modelRatio = supplierRatio
			success = true
		}
		if groupModelRatio, ok := ratio_setting.GetGroupModelRatio(info.UsingGroup, info.OriginModelName); ok {
			modelRatio = groupModelRatio
			success = true
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
		completionRatio = ratio_setting.GetCompletionRatio(info.OriginModelName)
		cacheRatio, _ = ratio_setting.GetCacheRatio(info.OriginModelName)
		cacheCreationRatio, _ = ratio_setting.GetCreateCacheRatio(info.OriginModelName)
		cacheCreationRatio5m = cacheCreationRatio
		// 固定1h和5min缓存写入价格的比例
		cacheCreationRatio1h = cacheCreationRatio * claudeCacheCreation1hMultiplier
		imageRatio, _ = ratio_setting.GetImageRatio(info.OriginModelName)
		audioRatio = ratio_setting.GetAudioRatio(info.OriginModelName)
		audioCompletionRatio = ratio_setting.GetAudioCompletionRatio(info.OriginModelName)
		videoRatio = ratio_setting.GetVideoRatio(info.OriginModelName)
		videoCompletionRatio = ratio_setting.GetVideoCompletionRatio(info.OriginModelName)
		if hasChannelCompletionRatio {
			completionRatio = channelCompletionRatio
		}
		if hasChannelCacheRatio {
			cacheRatio = channelCacheRatio
		}
		if hasChannelCreateCacheRatio {
			cacheCreationRatio = channelCreateCacheRatio
		}
		if hasChannelImageRatio {
			imageRatio = channelImageRatio
		}
		if hasChannelAudioRatio {
			audioRatio = channelAudioRatio
		}
		if hasChannelAudioCompletionRatio {
			audioCompletionRatio = channelAudioCompletionRatio
		}
		if hasChannelVideoRatio {
			videoRatio = channelVideoRatio
		}
		if hasChannelVideoCompletionRatio {
			videoCompletionRatio = channelVideoCompletionRatio
		}
		ratio := modelRatio * groupRatioInfo.GroupRatio
		preConsumedQuota = int(float64(preConsumedTokens) * ratio)
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

	modelPrice, success := ratio_setting.GetModelPrice(info.OriginModelName, true)
	if channelPrice, ok := ratio_setting.GetChannelModelPrice(channelID, info.OriginModelName); ok {
		modelPrice = channelPrice
		success = true
	}
	supplierID := resolveSupplierIDByChannel(info)
	if supplierPrice, ok := ratio_setting.GetSupplierModelPrice(supplierID, info.OriginModelName); ok {
		modelPrice = supplierPrice
		success = true
	}
	if groupPrice, ok := ratio_setting.GetGroupModelPrice(info.UsingGroup, info.OriginModelName); ok {
		modelPrice = groupPrice
		success = true
	}
	// 如果没有配置价格，检查模型倍率配置
	if !success {

		// 没有配置费用，也要使用默认费用,否则按费率计费模型无法使用
		defaultPrice, ok := ratio_setting.GetDefaultModelPriceMap()[info.OriginModelName]
		if ok {
			modelPrice = defaultPrice
		} else {
			// 没有配置倍率也不接受没配置,那就返回错误
			_, ratioSuccess, matchName := ratio_setting.GetModelRatio(info.OriginModelName)
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
// Selection rules (highest priority first):
//
//  1. Per-call price configured for this (channel, supplier, group, model)
//     -> identical to ModelPriceHelperPerCall: quota = price * QuotaPerUnit *
//     groupRatio * channelDiscount, then later multiplied by adaptor
//     OtherRatios (seconds, size).
//
//  2. Token-based ratios configured (VideoRatio for the model AND a
//     ModelRatio resolvable for the model)
//     -> quota = (inputTextTokens
//     + outputVideoTokens * videoRatio * videoCompletionRatio
//     ) * modelRatio * groupRatio
//     outputVideoTokens are estimated from the request:
//     duration * width * height * fps / 1024
//     OtherRatios are intentionally NOT applied on top, because
//     outputVideoTokens already accounts for duration and resolution; we
//     signal this by setting PriceData.UsePrice = true (treated by
//     relay_task as "quota is final, do not multiply OtherRatios again").
//
//  3. Neither configured
//     -> defer to ModelPriceHelperPerCall, which produces either the
//     canonical "ratio or price not set" error or the default-price
//     fallback, preserving existing behaviour for legacy users.
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
// See the file-level comment above for the full selection rules.
func ModelPriceHelperVideo(c *gin.Context, info *relaycommon.RelayInfo) (types.PriceData, error) {
	if info == nil {
		return types.PriceData{}, fmt.Errorf("relay info is nil")
	}

	channelID := 0
	if info.ChannelMeta != nil {
		channelID = info.ChannelId
	}
	supplierID := resolveSupplierIDByChannel(info)
	modelName := info.OriginModelName

	// Branch 1: any per-call price configured -> keep PerCall behaviour exactly.
	if hasAnyPerCallVideoPrice(channelID, supplierID, info.UsingGroup, modelName) {
		return ModelPriceHelperPerCall(c, info)
	}

	// Branch 2: try the token-based path. Requires both a usable ModelRatio AND
	// a VideoRatio configured for the model; otherwise we cannot meaningfully
	// price video tokens and have to fall through.
	if priceData, ok, err := tryVideoTokenPriceData(c, info); err != nil {
		return types.PriceData{}, err
	} else if ok {
		return priceData, nil
	}

	// Branch 3: nothing applicable -> defer to PerCall (error or default fallback).
	return ModelPriceHelperPerCall(c, info)
}

// hasAnyPerCallVideoPrice reports whether any per-call price tier is set for
// this model. Mirrors the lookup priority used by ModelPriceHelperPerCall.
func hasAnyPerCallVideoPrice(channelID, supplierID int, group, modelName string) bool {
	if _, ok := ratio_setting.GetModelPrice(modelName, false); ok {
		return true
	}
	if _, ok := ratio_setting.GetChannelModelPrice(channelID, modelName); ok {
		return true
	}
	if _, ok := ratio_setting.GetSupplierModelPrice(supplierID, modelName); ok {
		return true
	}
	if _, ok := ratio_setting.GetGroupModelPrice(group, modelName); ok {
		return true
	}
	// VideoPrice is the dedicated per-video price field (distinct from the
	// generic ModelPrice). Currently only configurable at the global scope.
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

	// Resolve modelRatio with the same precedence as ModelPriceHelper:
	// channel > supplier > group > global.
	modelRatio, modelRatioOK, _ := ratio_setting.GetModelRatio(modelName)
	if r, ok := ratio_setting.GetChannelModelRatio(channelID, modelName); ok {
		modelRatio = r
		modelRatioOK = true
	}
	if r, ok := ratio_setting.GetSupplierModelRatio(supplierID, modelName); ok {
		modelRatio = r
		modelRatioOK = true
	}
	if r, ok := ratio_setting.GetGroupModelRatio(info.UsingGroup, modelName); ok {
		modelRatio = r
		modelRatioOK = true
	}

	// Resolve videoRatio: channel > global. Required for token mode.
	videoRatio := ratio_setting.GetVideoRatio(modelName)
	hasVideoRatio := ratio_setting.ContainsVideoRatio(modelName)
	if r, ok := ratio_setting.GetChannelVideoRatio(channelID, modelName); ok {
		videoRatio = r
		hasVideoRatio = true
	}

	// Without a VideoRatio (input multiplier) the token formula has no signal
	// distinguishing "video" from "text"; refuse and let the caller fall back.
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
