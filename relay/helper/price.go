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
	"github.com/QuantumNous/new-api/service"
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
	// 计价名可回退到渠道 model_mapping 规范模型；日志仍用 OriginModelName。
	pricingModelName := model.ResolvePricingModelName(c.GetString("model_mapping"), info.OriginModelName)
	modelPrice, usePrice := model.ResolveSupplierScopedFixedModelPrice(channelID, supplierID, pricingModelName)
	// 归属供应商的渠道：固定价以 supplier_* 独立表优先于用户分组价；非供应商渠道保留分组覆盖。
	if supplierID <= 0 {
		if groupPrice, ok := ratio_setting.GetGroupModelPrice(info.UsingGroup, pricingModelName); ok {
			modelPrice = groupPrice
			usePrice = true
		}
	}
	channelVideoRatio, hasChannelVideoRatio := ratio_setting.GetChannelVideoRatio(channelID, pricingModelName)
	channelVideoCompletionRatio, hasChannelVideoCompletionRatio := ratio_setting.GetChannelVideoCompletionRatio(channelID, pricingModelName)

	// 提前获取成本折扣率、加价折扣率及全局倍率/固定价（新计费公式所需）
	rawDisc, operatingCost, chDisc := resolveChannelCostPercents(channelID)
	markupDisc := effectiveMarkupDiscountPercent(c, info, channelID, pricingModelName)
	globalRatio, _, _ := ratio_setting.GetModelRatio(pricingModelName)
	globalPrice, _ := ratio_setting.GetModelPrice(pricingModelName, false)
	// 全局子倍率（用于各类型加价部分的独立计算）
	globalCompletionRatio := ratio_setting.GetCompletionRatio(pricingModelName)
	globalCacheRatio, globalCacheRatioOK := ratio_setting.GetCacheRatio(pricingModelName)
	if !globalCacheRatioOK {
		globalCacheRatio = 1.0
	}
	globalCreateCacheRatio, globalCreateCacheRatioOK := ratio_setting.GetCreateCacheRatio(pricingModelName)
	if !globalCreateCacheRatioOK {
		globalCreateCacheRatio = 1.25
	}

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
		modelRatio, success, matchName = model.ResolveSupplierScopedModelRatio(channelID, supplierID, pricingModelName)
		// 供应商自有渠道：输入倍率以独立表（及 Resolve 内平台渠道 Option 回退）为准，不被用户分组倍率覆盖。
		if supplierID <= 0 {
			if groupModelRatio, ok := ratio_setting.GetGroupModelRatio(info.UsingGroup, pricingModelName); ok {
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
		completionRatio = model.ResolveSupplierScopedCompletionRatio(channelID, supplierID, pricingModelName)
		cacheRatio, cacheCreationRatio = model.ResolveSupplierScopedCacheRatios(channelID, supplierID, pricingModelName)
		cacheCreationRatio5m = cacheCreationRatio
		// 固定1h和5min缓存写入价格的比例
		cacheCreationRatio1h = cacheCreationRatio * claudeCacheCreation1hMultiplier

		imageRatio, _ = model.ResolveSupplierScopedImageRatio(channelID, supplierID, pricingModelName)
		audioRatio = model.ResolveSupplierScopedAudioRatio(channelID, supplierID, pricingModelName)
		audioCompletionRatio = model.ResolveSupplierScopedAudioCompletionRatio(channelID, supplierID, pricingModelName)
		// 供应商表暂无 Video 字段：仍采用全局 + 平台渠道 Option（与旧逻辑一致）。
		videoRatio = ratio_setting.GetVideoRatio(pricingModelName)
		videoCompletionRatio = ratio_setting.GetVideoCompletionRatio(pricingModelName)
		if hasChannelVideoRatio {
			videoRatio = channelVideoRatio
		}
		if hasChannelVideoCompletionRatio {
			videoCompletionRatio = channelVideoCompletionRatio
		}

		// 新公式：有效输入倍率 = 渠道倍率 * 成本折扣率% + 全局倍率 * 加价折扣率%
		effInputRate := model.EffectiveInputRate(modelRatio, globalRatio, chDisc, markupDisc)
		effInputRateWithGroup := effInputRate * groupRatioInfo.GroupRatio

		dPreConsumedTokens := decimal.NewFromInt(int64(preConsumedTokens))
		if tierHit, ok := ratio_setting.ResolveRequestTierHit(channelID, pricingModelName, int64(preConsumedTokens), chDisc, markupDisc, groupRatioInfo.GroupRatio); ok {
			preConsumedQuota = int(dPreConsumedTokens.Mul(decimal.NewFromFloat(tierHit.EffectiveInput * groupRatioInfo.GroupRatio)).Round(0).IntPart())
		} else {
			preConsumedQuota = int(dPreConsumedTokens.Mul(decimal.NewFromFloat(effInputRateWithGroup)).Round(0).IntPart())
		}
	} else {
		// 固定价格：渠道固定价 * 成本折扣率% + 全局固定价 * 加价折扣率%
		effModelPrice := model.EffectiveModelPrice(modelPrice, globalPrice, chDisc, markupDisc)
		if meta.ImagePriceRatio != 0 {
			effModelPrice = effModelPrice * meta.ImagePriceRatio
			modelPrice = modelPrice * meta.ImagePriceRatio // 保持 ModelPrice 字段与 imagePriceRatio 一致（供日志展示）
		}
		preConsumedQuota = int(effModelPrice * common.QuotaPerUnit * groupRatioInfo.GroupRatio)
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

	chDiscCopy := chDisc
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
		// 新计费公式字段
		CostDiscountPercent:     chDisc,
		RawPriceDiscountPercent: rawDisc,
		OperatingCostPercent:    operatingCost,
		MarkupDiscountPercent:   markupDisc,
		GlobalModelRatio:        globalRatio,
		GlobalModelPrice:        globalPrice,
		GlobalCompletionRatio:   globalCompletionRatio,
		GlobalCacheRatio:        globalCacheRatio,
		GlobalCreateCacheRatio:  globalCreateCacheRatio,
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
	pricingModelName := model.ResolvePricingModelName(c.GetString("model_mapping"), info.OriginModelName)
	modelPrice, success := model.ResolveSupplierScopedFixedModelPrice(channelID, supplierID, pricingModelName)
	if supplierID <= 0 {
		if groupPrice, ok := ratio_setting.GetGroupModelPrice(info.UsingGroup, pricingModelName); ok {
			modelPrice = groupPrice
			success = true
		}
	}
	// 如果没有配置价格，检查模型倍率配置
	if !success {

		// 没有配置费用，也要使用默认费用,否则按费率计费模型无法使用
		defaultPrice, ok := ratio_setting.GetDefaultModelPriceMap()[pricingModelName]
		if ok {
			modelPrice = defaultPrice
		} else {
			// 没有配置倍率也不接受没配置,那就返回错误
			_, ratioSuccess, matchName := model.ResolveSupplierScopedModelRatio(channelID, supplierID, pricingModelName)
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
	// 新公式：固定价格 = 渠道固定价 * 成本折扣率% + 全局固定价 * 加价折扣率%
	rawDisc, operatingCost, chDisc := resolveChannelCostPercents(channelID)
	markupDisc := effectiveMarkupDiscountPercent(c, info, channelID, pricingModelName)
	globalPrice, _ := ratio_setting.GetModelPrice(pricingModelName, false)
	effModelPrice := model.EffectiveModelPrice(modelPrice, globalPrice, chDisc, markupDisc)
	quota := int(effModelPrice * common.QuotaPerUnit * groupRatioInfo.GroupRatio)

	// 免费模型检测（与 ModelPriceHelper 对齐）
	freeModel := false
	if !operation_setting.GetQuotaSetting().EnableFreeModelPreConsume {
		if groupRatioInfo.GroupRatio == 0 || modelPrice == 0 {
			quota = 0
			freeModel = true
		}
	}
	chDiscCopy := chDisc

	priceData := types.PriceData{
		FreeModel:               freeModel,
		ModelPrice:              modelPrice,
		Quota:                   quota,
		GroupRatioInfo:          groupRatioInfo,
		ChannelPriceDiscount:    &chDiscCopy,
		CostDiscountPercent:     chDisc,
		RawPriceDiscountPercent: rawDisc,
		OperatingCostPercent:    operatingCost,
		MarkupDiscountPercent:   markupDisc,
		GlobalModelRatio:        0,
		GlobalModelPrice:        globalPrice,
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

	channelID := 0
	if info.ChannelMeta != nil {
		channelID = info.ChannelId
	}
	// 前置双重校验：能力匹配 + 分辨率档位匹配（不通过则统一友好提示）。
	if err := validateVideoModelPrice(c, channelID, info.OriginModelName); err != nil {
		return types.PriceData{}, err
	}

	// 0) 视频「按 Token 计费」（最高优先级）：视频价格配置为 per_token 规则且与当前请求
	//    （文生/图生/视频生 + 分辨率）匹配时生效；预扣固定 50000 Token，完成后按 total_tokens 补差。
	if priceData, ok := tryVideoPerTokenRulesPriceData(c, info); ok {
		return priceData, nil
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

	// 3) 无任何可用视频计费规则 -> 统一友好提示（替换原中英混合报错）。
	estimateCtx := estimateVideoRequestContext(c)
	currentInvocation := videoCapabilityLabelCN(string(estimateCtx.Mode))
	idx := collectVideoCapabilityPricing(channelID, info.OriginModelName)
	supportedCaps := sortedCapabilityLabelsCN(
		[]string{capabilityTextToVideo, capabilityImageToVideo, capabilityVideoToVideo},
		idx, videoCapabilityLabelCN,
	)
	allRes := collectAllDisplayResolutions(idx, []string{capabilityTextToVideo, capabilityImageToVideo, capabilityVideoToVideo})
	return types.PriceData{}, newModelPriceFriendlyError("视频模型", info.OriginModelName, currentInvocation, supportedCaps, allRes)
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

// tryVideoPerTokenRulesPriceData 视频「按 token 规则」预扣价格计算。
//
// 触发条件：VideoPricingRules 配置了 *_per_token 且与当前请求（素材类型 + 分辨率）匹配。
// 预扣额度 = 50000 ÷ 1M × 美元/1M tokens × QuotaPerUnit × 分组倍率。
func tryVideoPerTokenRulesPriceData(c *gin.Context, info *relaycommon.RelayInfo) (types.PriceData, bool) {
	channelID := 0
	if info.ChannelMeta != nil {
		channelID = info.ChannelId
	}
	estimateCtx := estimateVideoRequestContext(c)
	hasAudio := requestHasAudio(c)
	mode := string(estimateCtx.Mode)

	groupRatioInfo := HandleGroupRatio(c, info)
	preConsumeTokens := service.SeedanceTokenPreConsumeTokens
	resolutionLabel := ""
	if req, err := relaycommon.GetTaskRequest(c); err == nil {
		resolutionLabel = service.VideoBillingResolutionLabelFromRequest(req)
	}
	quota, match, ok := service.CalcVideoTokenQuota(
		channelID, info.UserId, info.OriginModelName, mode,
		estimateCtx.Width, estimateCtx.Height, hasAudio,
		preConsumeTokens, groupRatioInfo.GroupRatio,
		resolutionLabel,
	)
	if !ok || quota <= 0 || match == nil {
		return types.PriceData{}, false
	}

	rawDisc, operatingCost, chDisc := resolveChannelCostPercents(channelID)
	markupDisc := effectiveMarkupDiscountPercent(c, info, channelID, info.OriginModelName)
	chDiscCopy := chDisc

	priceData := types.PriceData{
		ModelPrice:              0,
		ModelRatio:              0,
		VideoOutputTokens:       preConsumeTokens,
		VideoInputTextTokens:    estimateCtx.InputTextTokens,
		GroupRatioInfo:          groupRatioInfo,
		UsePrice:                true,
		Quota:                   quota,
		QuotaToPreConsume:       quota,
		ChannelPriceDiscount:    &chDiscCopy,
		CostDiscountPercent:     chDisc,
		RawPriceDiscountPercent: rawDisc,
		OperatingCostPercent:    operatingCost,
		MarkupDiscountPercent:   markupDisc,
		VideoRuleUnit:           service.VideoRuleUnitPerToken,
		VideoBillingMode:        mode,
		VideoChannelRulePrice:   match.ChannelPricePerToken,
		VideoGlobalRulePrice:    match.GlobalPricePerToken,
		VideoRuleWidth:          match.RuleWidth,
		VideoRuleHeight:         match.RuleHeight,
		VideoRuleHasAudio:       hasAudio,
	}
	if common.DebugEnabled {
		logger.LogDebug(c, fmt.Sprintf(
			"[video][per-token-rules] model=%s mode=%s w=%d h=%d pricePerToken=%.8f preConsumeTokens=%d groupRatio=%.4f -> quota=%d",
			info.OriginModelName, mode, estimateCtx.Width, estimateCtx.Height,
			match.EffectivePricePerToken, preConsumeTokens, groupRatioInfo.GroupRatio, quota,
		))
	}
	info.PriceData = priceData
	return priceData, true
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

	// 新公式（视频 token 计费）：有效倍率 = 渠道倍率 * 成本折扣率% + 全局倍率 * 加价折扣率%
	rawDisc, operatingCost, chDisc := resolveChannelCostPercents(channelID)
	markupDisc := effectiveMarkupDiscountPercent(c, info, channelID, info.OriginModelName)
	globalRatioVideo, _, _ := ratio_setting.GetModelRatio(modelName)
	effRateVideo := model.EffectiveInputRate(modelRatio, globalRatioVideo, chDisc, markupDisc)
	// rawQuota 已按 modelRatio * groupRatio 计算，用 effRateVideo/modelRatio 修正（modelRatio>0 时）
	if modelRatio > 0 {
		rawQuota = rawQuota * effRateVideo / modelRatio
	}
	chDiscCopy := chDisc
	quota := int(math.Round(rawQuota))

	// Floor non-zero results at 1 quota unit, matching calculateAudioQuota.
	if !freeModel && weightedTokens > 0 && quota <= 0 && modelRatio > 0 && groupRatioInfo.GroupRatio > 0 {
		quota = 1
	}

	priceData := types.PriceData{
		FreeModel:               freeModel,
		ModelRatio:              modelRatio,
		VideoRatio:              appliedVideoRatio,
		VideoCompletionRatio:    appliedVideoCompletionRatio,
		VideoOutputTokens:       outputVideoTokens,
		VideoInputTextTokens:    inputTextTokens,
		GroupRatioInfo:          groupRatioInfo,
		Quota:                   quota,
		QuotaToPreConsume:       quota,
		ChannelPriceDiscount:    &chDiscCopy,
		CostDiscountPercent:     chDisc,
		RawPriceDiscountPercent: rawDisc,
		OperatingCostPercent:    operatingCost,
		MarkupDiscountPercent:   markupDisc,
		GlobalModelRatio:        globalRatioVideo,
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

func hasChannelVideoPerSecondPricingRules(channelID int, modelName string) bool {
	if rules, ok := ratio_setting.GetChannelVideoPricingRules(channelID, modelName); ok {
		return ratio_setting.HasUsableVideoPerSecondRules(rules)
	}
	return false
}

func pickAudioPriceByResolution(ctx videoEstimateContext, hasAudio bool, rows []ratio_setting.VideoResolutionAudioPriceRule) (float64, bool) {
	if len(rows) == 0 {
		return 0, false
	}
	targetLong, targetShort := normalizeResolutionSides(ctx.Width, ctx.Height)
	targetRatio := targetResolutionRatio(ctx.Width, ctx.Height)
	bestIdx := -1
	bestPixels := int(^uint(0) >> 1)
	fallbackIdx := -1
	fallbackPixels := 0
	for i := range rows {
		r := rows[i]
		if r.Price <= 0 || r.HasAudio != hasAudio {
			continue
		}
		ruleW, ruleH, ok := parseResolutionFlexibleForRatio(r.Resolution, targetRatio)
		if !ok {
			continue
		}
		rulePixels := ruleW * ruleH
		if rulePixels <= 0 {
			continue
		}
		ruleLong, ruleShort := normalizeResolutionSides(ruleW, ruleH)
		if ruleLong >= targetLong && ruleShort >= targetShort {
			if rulePixels < bestPixels {
				bestPixels = rulePixels
				bestIdx = i
			}
			continue
		}
		if rulePixels > fallbackPixels {
			fallbackPixels = rulePixels
			fallbackIdx = i
		}
	}
	if bestIdx < 0 {
		bestIdx = fallbackIdx
	}
	if bestIdx < 0 {
		return 0, false
	}
	return rows[bestIdx].Price, true
}

func normalizeResolutionSides(width, height int) (longSide, shortSide int) {
	if width >= height {
		return width, height
	}
	return height, width
}

func targetResolutionRatio(width, height int) float64 {
	longSide, shortSide := normalizeResolutionSides(width, height)
	if longSide <= 0 || shortSide <= 0 {
		return 16.0 / 9.0
	}
	ratio := float64(longSide) / float64(shortSide)
	candidates := []float64{
		1.0,
		4.0 / 3.0,
		16.0 / 9.0,
		21.0 / 9.0,
	}
	best := candidates[0]
	bestDiff := math.Abs(ratio - best)
	for _, candidate := range candidates[1:] {
		if diff := math.Abs(ratio - candidate); diff < bestDiff {
			best = candidate
			bestDiff = diff
		}
	}
	return best
}

func parseResolutionFlexibleForRatio(s string, ratio float64) (int, int, bool) {
	raw := strings.ToLower(strings.TrimSpace(s))
	if raw == "" {
		return 0, 0, false
	}
	if w, h, ok := parseResolution(raw); ok {
		return w, h, true
	}
	shortSide := 0
	switch raw {
	case "480p":
		shortSide = 480
	case "540p":
		shortSide = 540
	case "720p":
		shortSide = 720
	case "1080p":
		shortSide = 1080
	case "2k":
		shortSide = 1440
	case "4k":
		shortSide = 2160
	default:
		return 0, 0, false
	}
	longSide := int(math.Ceil(float64(shortSide) * ratio))
	return longSide, shortSide, true
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
	case "1k":
		return 1024, 1024, true
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

	resolutionLabel := ""
	if req, err := relaycommon.GetTaskRequest(c); err == nil {
		resolutionLabel = service.VideoBillingResolutionLabelFromRequest(req)
	}

	var pricePerSecond float64
	pricePerSecond, ok = service.VideoPerSecondChannelUnitPrice(
		rules,
		string(estimateCtx.Mode),
		estimateCtx.Width,
		estimateCtx.Height,
		hasAudio,
		resolutionLabel,
	)
	if !ok || pricePerSecond <= 0 {
		return types.PriceData{}, false, nil
	}
	groupRatioInfo := HandleGroupRatio(c, info)
	rawDiscVPS, operatingCostVPS, chDiscVPS := resolveChannelCostPercents(channelID)
	markupDiscVPS := effectiveMarkupDiscountPercent(c, info, channelID, info.OriginModelName)
	effPricePerSecond, _, _, effOK := service.EffectiveVideoPerSecondUSDForDimensions(
		channelID,
		info.OriginModelName,
		string(estimateCtx.Mode),
		estimateCtx.Width,
		estimateCtx.Height,
		hasAudio,
		chDiscVPS,
		markupDiscVPS,
		resolutionLabel,
	)
	if !effOK || effPricePerSecond <= 0 {
		// 渠道未配置按秒规则而仅配置全局规则时，按全局匹配单价计费。
		// service.EffectiveVideoPerSecondUSDForDimensions 需要渠道档位，故这里回退。
		if !hasChannelVideoPerSecondPricingRules(channelID, info.OriginModelName) {
			effPricePerSecond = pricePerSecond
		} else {
			return types.PriceData{}, false, nil
		}
	}
	rawQuota := float64(seconds) * effPricePerSecond * common.QuotaPerUnit * groupRatioInfo.GroupRatio
	chDiscCopyVPS := chDiscVPS
	quota := int(math.Round(rawQuota))
	if quota <= 0 && rawQuota > 0 {
		quota = 1
	}
	pd := types.PriceData{
		ModelPrice:              0,
		ModelRatio:              0,
		GroupRatioInfo:          groupRatioInfo,
		UsePrice:                true,
		Quota:                   quota,
		QuotaToPreConsume:       quota,
		ChannelPriceDiscount:    &chDiscCopyVPS,
		CostDiscountPercent:     chDiscVPS,
		RawPriceDiscountPercent: rawDiscVPS,
		OperatingCostPercent:    operatingCostVPS,
		MarkupDiscountPercent:   markupDiscVPS,
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

	rawDiscVPV, operatingCostVPV, chDiscVPV := resolveChannelCostPercents(channelID)
	markupDiscVPV := effectiveMarkupDiscountPercent(c, info, channelID, info.OriginModelName)
	globalUsd := globalVideoPerVideoUSDForRelay(modelName, string(estimateCtx.Mode), estimateCtx.Width, estimateCtx.Height, hasAudio)
	effUsd := model.EffectiveRuleUnitPrice(usd, globalUsd, chDiscVPV, markupDiscVPV)
	rawQuota = effUsd * common.QuotaPerUnit * groupRatioInfo.GroupRatio
	chDiscCopyVPV := chDiscVPV
	quota := int(math.Round(rawQuota))

	if !freeModel && quota <= 0 && rawQuota > 0 && groupRatioInfo.GroupRatio > 0 {
		quota = 1
	}

	priceData := types.PriceData{
		FreeModel:               freeModel,
		ModelPrice:              0,
		ModelRatio:              0,
		GroupRatioInfo:          groupRatioInfo,
		UsePrice:                true,
		Quota:                   quota,
		QuotaToPreConsume:       quota,
		ChannelPriceDiscount:    &chDiscCopyVPV,
		CostDiscountPercent:     chDiscVPV,
		RawPriceDiscountPercent: rawDiscVPV,
		OperatingCostPercent:    operatingCostVPV,
		MarkupDiscountPercent:   markupDiscVPV,
		GlobalModelPrice:        globalUsd,
		VideoRuleUnit:           "per_video",
		VideoBillingMode:        string(estimateCtx.Mode),
		VideoChannelRulePrice:   usd,
		VideoGlobalRulePrice:    globalUsd,
		VideoRuleWidth:          estimateCtx.Width,
		VideoRuleHeight:         estimateCtx.Height,
		VideoRuleHasAudio:       hasAudio,
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
	switch relaycommon.DetectVideoBillingMode(req) {
	case relaycommon.VideoBillingModeImageToVideo:
		return videoBillingModeImageToVideo
	case relaycommon.VideoBillingModeVideoToVideo:
		return videoBillingModeVideoToVideo
	default:
		return videoBillingModeTextToVideo
	}
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

// resolveVideoDimensions 解析请求分辨率像素（metadata / 顶层 resolution / size），
// 用于按分辨率档位匹配 per_token 单价；无法识别时回退默认值以保证预扣非零。
func resolveVideoDimensions(req *relaycommon.TaskSubmitReq) (width, height int) {
	if w, h, ok := common.ResolveVideoDimensionsFromRequest(
		req.Size, req.Resolution, req.Ratio, req.Metadata,
	); ok {
		return w, h
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
