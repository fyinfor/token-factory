package helper

import (
	"fmt"
	"math"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

type imageBillingMode string

const (
	imageBillingModeTextToImage  imageBillingMode = "text_to_image"
	imageBillingModeImageToImage imageBillingMode = "image_to_image"
)

type imageEstimateContext struct {
	Mode   imageBillingMode
	Width  int
	Height int
	Count  int
}

type imagePerImagePriceMatch struct {
	Price           float64
	Resolution      string
	RuleWidth       int
	RuleHeight      int
	CappedToMaxTier bool
}

// HasImagePerImageTablePricing reports whether resolution-tier per-image rules exist.
func HasImagePerImageTablePricing(channelID int, modelName string) bool {
	_, ok := resolveImagePricingRules(channelID, modelName)
	return ok
}

// HasImageGenerationPricing reports whether per-image generation pricing is configured.
func HasImageGenerationPricing(channelID int, modelName string) bool {
	if HasImagePerImageTablePricing(channelID, modelName) {
		return true
	}
	for _, name := range imageModelNameCandidates(modelName) {
		if _, ok := ratio_setting.GetImagePrice(name); ok {
			return true
		}
		if _, ok := ratio_setting.GetChannelImagePrice(channelID, name); ok {
			return true
		}
	}
	return false
}

func imageModelNameCandidates(modelName string) []string {
	name := ratio_setting.FormatMatchingModelName(strings.TrimSpace(modelName))
	if name == "" {
		return nil
	}
	return []string{name}
}

func imageModelNameCandidatesFromInfo(info *relaycommon.RelayInfo) []string {
	if info == nil {
		return nil
	}
	seen := make(map[string]struct{}, 6)
	out := make([]string, 0, 4)
	add := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		formatted := ratio_setting.FormatMatchingModelName(name)
		if formatted == "" {
			return
		}
		if _, ok := seen[formatted]; ok {
			return
		}
		seen[formatted] = struct{}{}
		out = append(out, formatted)
	}
	add(info.OriginModelName)
	if info.ChannelMeta != nil {
		add(info.UpstreamModelName)
	}
	return out
}

func resolveImagePricingRules(channelID int, modelName string) (ratio_setting.ImagePricingRules, bool) {
	return resolveImagePricingRulesForNames(channelID, imageModelNameCandidates(modelName))
}

func resolveImagePricingRulesForInfo(channelID int, info *relaycommon.RelayInfo) (ratio_setting.ImagePricingRules, bool) {
	return resolveImagePricingRulesForNames(channelID, imageModelNameCandidatesFromInfo(info))
}

func resolveImagePricingRulesForNames(channelID int, names []string) (ratio_setting.ImagePricingRules, bool) {
	var merged ratio_setting.ImagePricingRules
	hasMerged := false
	for _, name := range names {
		if name == "" {
			continue
		}
		if rules, ok := ratio_setting.GetChannelImagePricingRules(channelID, name); ok {
			merged = mergeImagePricingRules(merged, rules)
			hasMerged = hasMerged || ratio_setting.HasUsableImagePerImageRules(rules)
		}
		if rules, ok := ratio_setting.GetImagePricingRules(name); ok {
			merged = mergeImagePricingRules(merged, rules)
			hasMerged = hasMerged || ratio_setting.HasUsableImagePerImageRules(rules)
		}
	}
	if !hasMerged || !ratio_setting.HasUsableImagePerImageRules(merged) {
		return ratio_setting.ImagePricingRules{}, false
	}
	return normalizeMergedImageRules(merged), true
}

func mergeImagePricingRules(dst, src ratio_setting.ImagePricingRules) ratio_setting.ImagePricingRules {
	if dst.SimilarityThreshold <= 0 && src.SimilarityThreshold > 0 {
		dst.SimilarityThreshold = src.SimilarityThreshold
	}
	if dst.PriceUnit == "" && src.PriceUnit != "" {
		dst.PriceUnit = src.PriceUnit
	}
	dst.TextToImagePerImage = mergeImagePerImageRows(dst.TextToImagePerImage, src.TextToImagePerImage)
	dst.ImageToImagePerImage = mergeImagePerImageRows(dst.ImageToImagePerImage, src.ImageToImagePerImage)
	return dst
}

func mergeImagePerImageRows(dst, src []ratio_setting.ImageResolutionPerImageRule) []ratio_setting.ImageResolutionPerImageRule {
	if len(src) == 0 {
		return dst
	}
	index := make(map[string]int, len(dst))
	for i, row := range dst {
		index[strings.ToLower(strings.TrimSpace(row.Resolution))] = i
	}
	for _, row := range src {
		key := strings.ToLower(strings.TrimSpace(row.Resolution))
		if key == "" || row.ImagePrice <= 0 {
			continue
		}
		if i, ok := index[key]; ok {
			dst[i] = row
			continue
		}
		dst = append(dst, row)
		index[key] = len(dst) - 1
	}
	return dst
}

func normalizeMergedImageRules(v ratio_setting.ImagePricingRules) ratio_setting.ImagePricingRules {
	if v.SimilarityThreshold <= 0 {
		v.SimilarityThreshold = 0.35
	}
	return v
}

func resolveImageFlatUSD(channelID int, modelName string) (float64, bool) {
	for _, name := range imageModelNameCandidates(modelName) {
		if price, ok := ratio_setting.GetChannelImagePrice(channelID, name); ok && price > 0 {
			return price, true
		}
		if price, ok := ratio_setting.GetImagePrice(name); ok && price > 0 {
			return price, true
		}
	}
	return 0, false
}

func HasImageGenerationPricingForInfo(channelID int, info *relaycommon.RelayInfo) bool {
	if info == nil {
		return false
	}
	if HasImagePerImageTablePricingForInfo(channelID, info) {
		return true
	}
	for _, name := range imageModelNameCandidatesFromInfo(info) {
		if _, ok := ratio_setting.GetImagePrice(name); ok {
			return true
		}
		if _, ok := ratio_setting.GetChannelImagePrice(channelID, name); ok {
			return true
		}
	}
	return false
}

func HasImagePerImageTablePricingForInfo(channelID int, info *relaycommon.RelayInfo) bool {
	_, ok := resolveImagePricingRulesForInfo(channelID, info)
	return ok
}

func resolveChannelOnlyImagePricingRules(channelID int, names []string) (ratio_setting.ImagePricingRules, bool) {
	for _, name := range names {
		if name == "" {
			continue
		}
		if rules, ok := ratio_setting.GetChannelImagePricingRules(channelID, name); ok && ratio_setting.HasUsableImagePerImageRules(rules) {
			return rules, true
		}
	}
	return ratio_setting.ImagePricingRules{}, false
}

func resolveGlobalOnlyImagePricingRules(names []string) (ratio_setting.ImagePricingRules, bool) {
	for _, name := range names {
		if name == "" {
			continue
		}
		if rules, ok := ratio_setting.GetImagePricingRules(name); ok && ratio_setting.HasUsableImagePerImageRules(rules) {
			return rules, true
		}
	}
	return ratio_setting.ImagePricingRules{}, false
}

func resolveChannelImageFlatUSD(channelID int, names []string) (float64, bool) {
	for _, name := range names {
		if price, ok := ratio_setting.GetChannelImagePrice(channelID, name); ok && price > 0 {
			return price, true
		}
	}
	return 0, false
}

func resolveGlobalImageFlatUSD(names []string) (float64, bool) {
	for _, name := range names {
		if price, ok := ratio_setting.GetImagePrice(name); ok && price > 0 {
			return price, true
		}
	}
	return 0, false
}

func resolveImageFlatUSDForInfo(channelID int, info *relaycommon.RelayInfo) (float64, bool) {
	for _, name := range imageModelNameCandidatesFromInfo(info) {
		if price, ok := ratio_setting.GetChannelImagePrice(channelID, name); ok && price > 0 {
			return price, true
		}
		if price, ok := ratio_setting.GetImagePrice(name); ok && price > 0 {
			return price, true
		}
	}
	return 0, false
}

// TryModelPriceHelperImage prices image generation when per-image rules or flat ImagePrice exist.
// Returns (priceData, true, nil) on success; (zero, false, nil) when not configured.
func TryModelPriceHelperImage(c *gin.Context, info *relaycommon.RelayInfo) (types.PriceData, bool, error) {
	if info == nil {
		return types.PriceData{}, false, nil
	}
	channelID := 0
	if info.ChannelMeta != nil {
		channelID = info.ChannelId
	}
	modelName := info.OriginModelName

	if !HasImageGenerationPricing(channelID, modelName) &&
		!HasImageGenerationPricingForInfo(channelID, info) {
		return types.PriceData{}, false, nil
	}

	names := imageModelNameCandidatesFromInfo(info)
	if len(names) == 0 {
		names = imageModelNameCandidates(modelName)
	}
	estimateCtx := estimateImageRequestContext(c, info)
	channelUSD, globalUSD, chOK, glOK, channelMatch, globalMatch := resolveImagePerImageUnitUSD(channelID, names, estimateCtx)
	usdPerImage := channelUSD
	okPrice := chOK
	if !okPrice || usdPerImage <= 0 {
		usdPerImage = globalUSD
		okPrice = glOK
	}
	if !okPrice || usdPerImage <= 0 {
		matchName := ratio_setting.FormatMatchingModelName(modelName)
		if matchName == "" {
			matchName = modelName
		}
		return types.PriceData{}, false, fmt.Errorf(
			"图片模型 %s 未设置按张价格，请配置文生图/图生图分辨率价格或兜底每张价格；Image model %s per-image price not set",
			matchName, matchName,
		)
	}

	count := estimateCtx.Count
	if count <= 0 {
		count = 1
	}
	estimateCtx.Count = count

	priceData, ok := buildImagePerImagePriceData(c, info, channelID, channelUSD, globalUSD, chOK, glOK, channelMatch, globalMatch, usdPerImage, estimateCtx)
	if !ok {
		matchName := ratio_setting.FormatMatchingModelName(modelName)
		if matchName == "" {
			matchName = modelName
		}
		return types.PriceData{}, false, fmt.Errorf(
			"图片模型 %s 未设置按张价格，请配置文生图/图生图分辨率价格或兜底每张价格；Image model %s per-image price not set",
			matchName, matchName,
		)
	}
	info.PriceData = priceData
	return priceData, true, nil
}

// resolveImagePerImageUnitUSD 分别解析渠道规则价与全局规则价（不合并规则表）。
func resolveImagePerImageUnitUSD(channelID int, names []string, estimateCtx imageEstimateContext) (channelUSD, globalUSD float64, chOK, glOK bool, channelMatch, globalMatch *imagePerImagePriceMatch) {
	channelRules, chHasRules := resolveChannelOnlyImagePricingRules(channelID, names)
	globalRules, glHasRules := resolveGlobalOnlyImagePricingRules(names)
	chFallback, chHasFallback := resolveChannelImageFlatUSD(channelID, names)
	glFallback, glHasFallback := resolveGlobalImageFlatUSD(names)
	channelUSD, chOK, channelMatch = matchFlatPerImageUSDRules(estimateCtx, channelRules, chHasRules, chFallback, chHasFallback)
	globalUSD, glOK, globalMatch = matchFlatPerImageUSDRules(estimateCtx, globalRules, glHasRules, glFallback, glHasFallback)
	return channelUSD, globalUSD, chOK, glOK, channelMatch, globalMatch
}

func buildImagePerImagePriceData(
	c *gin.Context,
	info *relaycommon.RelayInfo,
	channelID int,
	channelUSD, globalUSD float64,
	chOK, glOK bool,
	channelMatch, globalMatch *imagePerImagePriceMatch,
	fallbackUSD float64,
	estimateCtx imageEstimateContext,
) (types.PriceData, bool) {
	if info == nil {
		return types.PriceData{}, false
	}
	usdPerImage := channelUSD
	okPrice := chOK
	selectedMatch := channelMatch
	if !okPrice || usdPerImage <= 0 {
		usdPerImage = globalUSD
		okPrice = glOK
		selectedMatch = globalMatch
	}
	if !okPrice || usdPerImage <= 0 {
		return types.PriceData{}, false
	}

	count := estimateCtx.Count
	if count <= 0 {
		count = 1
	}

	groupRatioInfo := HandleGroupRatio(c, info)
	freeModel := false
	if !operation_setting.GetQuotaSetting().EnableFreeModelPreConsume {
		if groupRatioInfo.GroupRatio == 0 {
			freeModel = true
		}
	}

	rawDiscImg, operatingCostImg, chDiscImg := resolveChannelCostPercents(channelID)
	markupDiscImg := effectiveMarkupDiscountPercent(c, info, channelID, info.OriginModelName)
	channelRuleUSD := channelUSD
	if !chOK || channelRuleUSD <= 0 {
		channelRuleUSD = usdPerImage
	}
	globalRuleUSD := globalUSD
	if !glOK || globalRuleUSD <= 0 {
		globalRuleUSD = 0
	}
	effUsdPerImage := model.EffectiveRuleUnitPrice(channelRuleUSD, globalRuleUSD, chDiscImg, markupDiscImg)
	rawQuota := effUsdPerImage * float64(count) * common.QuotaPerUnit * groupRatioInfo.GroupRatio
	chDiscCopyImg := chDiscImg
	quota := int(math.Round(rawQuota))
	if !freeModel && quota <= 0 && rawQuota > 0 && groupRatioInfo.GroupRatio > 0 {
		quota = 1
	}
	if freeModel {
		quota = 0
		rawQuota = 0
	}

	priceData := types.PriceData{
		FreeModel:               freeModel,
		ModelPrice:              channelRuleUSD,
		GroupRatioInfo:          groupRatioInfo,
		UsePrice:                true,
		Quota:                   quota,
		QuotaToPreConsume:       quota,
		ChannelPriceDiscount:    &chDiscCopyImg,
		CostDiscountPercent:     chDiscImg,
		RawPriceDiscountPercent: rawDiscImg,
		OperatingCostPercent:    operatingCostImg,
		MarkupDiscountPercent:   markupDiscImg,
		GlobalModelPrice:        globalRuleUSD,
	}
	priceData.AddOtherRatio("n", float64(count))
	info.ImageBilling = &relaycommon.ImageBillingSnapshot{
		UsdPerImage:     effUsdPerImage,
		Width:           estimateCtx.Width,
		Height:          estimateCtx.Height,
		Count:           count,
		Mode:            string(estimateCtx.Mode),
		CappedToMaxTier: selectedMatch != nil && selectedMatch.CappedToMaxTier,
	}
	if selectedMatch != nil {
		info.ImageBilling.RuleWidth = selectedMatch.RuleWidth
		info.ImageBilling.RuleHeight = selectedMatch.RuleHeight
		info.ImageBilling.RuleRes = selectedMatch.Resolution
	}
	if common.DebugEnabled {
		logger.LogDebug(c, fmt.Sprintf(
			"[image][per-image] model=%s mode=%s w=%d h=%d count=%d channelUSD=%.6f globalUSD=%.6f effUSD=%.6f quota=%d",
			info.OriginModelName, estimateCtx.Mode, estimateCtx.Width, estimateCtx.Height, count,
			channelRuleUSD, globalRuleUSD, effUsdPerImage, quota,
		))
	}
	return priceData, true
}

// SyncImagePerImagePriceData 按渠道/全局规则价刷新 PriceData（供 finalize 与结算对齐）。
func SyncImagePerImagePriceData(c *gin.Context, info *relaycommon.RelayInfo, estimateCtx imageEstimateContext) bool {
	if info == nil || !info.PriceData.UsePrice {
		return false
	}
	channelID := 0
	if info.ChannelMeta != nil {
		channelID = info.ChannelId
	}
	names := imageModelNameCandidatesFromInfo(info)
	if len(names) == 0 {
		names = imageModelNameCandidates(info.OriginModelName)
	}
	channelUSD, globalUSD, chOK, glOK, channelMatch, globalMatch := resolveImagePerImageUnitUSD(channelID, names, estimateCtx)
	usdPerImage := channelUSD
	okPrice := chOK
	if !okPrice || usdPerImage <= 0 {
		usdPerImage = globalUSD
		okPrice = glOK
	}
	if !okPrice || usdPerImage <= 0 {
		return false
	}
	pd, ok := buildImagePerImagePriceData(c, info, channelID, channelUSD, globalUSD, chOK, glOK, channelMatch, globalMatch, usdPerImage, estimateCtx)
	if !ok {
		return false
	}
	info.PriceData = pd
	return true
}

func estimateImageRequestContext(c *gin.Context, info *relaycommon.RelayInfo) imageEstimateContext {
	ctx := imageEstimateContext{
		Mode:   imageBillingModeTextToImage,
		Width:  0,
		Height: 0,
		Count:  1,
	}
	if info != nil && info.RelayMode == relayconstant.RelayModeImagesEdits {
		ctx.Mode = imageBillingModeImageToImage
	}
	if info != nil {
		if req, ok := info.Request.(*dto.ImageRequest); ok && req != nil {
			if w, h, ok := parseResolutionFlexible(req.Size); ok {
				ctx.Width = w
				ctx.Height = h
			}
			if req.N != nil && *req.N > 0 {
				ctx.Count = int(*req.N)
			}
			if ctx.Mode == imageBillingModeTextToImage && hasImageInputInRequest(req) {
				ctx.Mode = imageBillingModeImageToImage
			}
		}
	}
	_ = c
	return ctx
}

func hasImageInputInRequest(req *dto.ImageRequest) bool {
	if req == nil {
		return false
	}
	raw := strings.TrimSpace(string(req.Image))
	return raw != "" && raw != "null"
}

func matchFlatPerImageUSDRules(
	ctx imageEstimateContext,
	rules ratio_setting.ImagePricingRules,
	hasRules bool,
	fallbackUSD float64,
	hasFallback bool,
) (float64, bool, *imagePerImagePriceMatch) {
	if hasRules {
		threshold := rules.SimilarityThreshold
		if threshold <= 0 {
			threshold = 0.35
		}
		var rows []ratio_setting.ImageResolutionPerImageRule
		if ctx.Mode == imageBillingModeImageToImage {
			rows = rules.ImageToImagePerImage
		} else {
			rows = rules.TextToImagePerImage
		}
		if match, ok := matchPerImageRulesByPixels(ctx, rows, threshold, fallbackUSD, hasFallback); ok {
			if match != nil {
				return match.Price, true, match
			}
			return fallbackUSD, true, nil
		}
	}
	if hasFallback && fallbackUSD > 0 {
		return fallbackUSD, true, nil
	}
	return 0, false, nil
}

// matchPerImageRulesByPixels picks the closest resolution row. If the image is
// larger than every configured tier in the current lane, it caps to that lane's
// highest tier instead of falling back to a generic fixed price.
func matchPerImageRulesByPixels(
	ctx imageEstimateContext,
	rules []ratio_setting.ImageResolutionPerImageRule,
	threshold float64,
	fallbackUSD float64,
	hasFallback bool,
) (*imagePerImagePriceMatch, bool) {
	if len(rules) == 0 {
		if hasFallback && fallbackUSD > 0 {
			return nil, true
		}
		return nil, false
	}
	if ctx.Width <= 0 || ctx.Height <= 0 {
		if hasFallback && fallbackUSD > 0 {
			return nil, true
		}
		return nil, false
	}
	bestIdx := -1
	targetPixels := ctx.Width * ctx.Height
	minDiffRatio := math.MaxFloat64
	maxTierIdx := -1
	maxTierPixels := 0
	for i, rule := range rules {
		if rule.ImagePrice <= 0 {
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
		if rulePixels > maxTierPixels {
			maxTierPixels = rulePixels
			maxTierIdx = i
		}
		diffRatio := math.Abs(float64(targetPixels-rulePixels)) / float64(rulePixels)
		if diffRatio < minDiffRatio {
			minDiffRatio = diffRatio
			bestIdx = i
		}
	}
	if bestIdx < 0 {
		if hasFallback && fallbackUSD > 0 {
			return nil, true
		}
		return nil, false
	}
	if targetPixels > maxTierPixels && maxTierIdx >= 0 {
		match := imagePerImageRuleMatch(rules[maxTierIdx], true)
		if match != nil {
			return match, true
		}
	}
	if threshold <= 0 {
		threshold = 0.35
	}
	if minDiffRatio > threshold {
		if hasFallback && fallbackUSD > 0 {
			return nil, true
		}
		return nil, false
	}
	match := imagePerImageRuleMatch(rules[bestIdx], false)
	return match, match != nil
}

func imagePerImageRuleMatch(rule ratio_setting.ImageResolutionPerImageRule, capped bool) *imagePerImagePriceMatch {
	if rule.ImagePrice <= 0 {
		return nil
	}
	w, h, ok := parseResolution(rule.Resolution)
	if !ok {
		return nil
	}
	return &imagePerImagePriceMatch{
		Price:           rule.ImagePrice,
		Resolution:      strings.TrimSpace(rule.Resolution),
		RuleWidth:       w,
		RuleHeight:      h,
		CappedToMaxTier: capped,
	}
}

// ModelPriceHelperForImageFallback is used only when per-image rules are not configured.
// If rules exist in Option but were not applied, return an error instead of silent supplier ModelPrice.
func ModelPriceHelperForImageFallback(c *gin.Context, info *relaycommon.RelayInfo, promptTokens int, meta *types.TokenCountMeta) (types.PriceData, error) {
	if info == nil {
		return ModelPriceHelper(c, info, promptTokens, meta)
	}
	channelID := 0
	if info.ChannelMeta != nil {
		channelID = info.ChannelId
	}
	if HasImagePerImageTablePricingForInfo(channelID, info) ||
		HasImagePerImageTablePricing(channelID, info.OriginModelName) {
		matchName := info.OriginModelName
		return types.PriceData{}, fmt.Errorf(
			"图片模型 %s 已保存按张分辨率价格但未生效，请确认已保存 ImagePricingRules/ChannelImagePricingRules 并重启服务；模型名须与请求一致。Image per-image rules exist for %s but billing did not apply",
			matchName, matchName,
		)
	}
	priceData, err := ModelPriceHelper(c, info, promptTokens, meta)
	if err != nil {
		return priceData, err
	}
	if priceData.UsePrice && priceData.ModelPrice > 0 {
		logger.LogInfo(c, fmt.Sprintf(
			"[image][fallback] model=%s channel=%d using fixed price $%.4f/request (no ImagePricingRules for this model). Set per-image rules in ratio settings.",
			info.OriginModelName, channelID, priceData.ModelPrice,
		))
	}
	return priceData, err
}
