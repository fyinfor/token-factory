package helper

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

// capabilityPricingConfig 描述某一计费能力下的已配置价格信息。
type capabilityPricingConfig struct {
	hasFlatPricing     bool
	hasResolutionTiers bool
	resolutions        map[string]struct{}
	displayResolutions []string
}

func (c *capabilityPricingConfig) configured() bool {
	if c == nil {
		return false
	}
	return c.hasFlatPricing || c.hasResolutionTiers
}

type capabilityPricingIndex map[string]*capabilityPricingConfig

const (
	capabilityTextToImage  = "text_to_image"
	capabilityImageToImage = "image_to_image"
	capabilityTextToVideo  = "text_to_video"
	capabilityImageToVideo = "image_to_video"
	capabilityVideoToVideo = "video_to_video"
)

// normalizePricingResolutionLabel 将视频分辨率配置/入参统一为可比对标识（如 720p、1080p、4K）。
func normalizePricingResolutionLabel(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if label := common.FormatVideoResolutionLabel(s); label != "" {
		return strings.ToLower(label)
	}
	return strings.ToLower(s)
}

// normalizeImagePricingResolutionLabel 将图片分辨率统一为 Ai 绘图档位标识（1080p≡1k / 2k / 4k）。
func normalizeImagePricingResolutionLabel(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if label := common.FormatImageResolutionLabel(s); label != "" {
		return strings.ToLower(label)
	}
	return strings.ToLower(s)
}

func videoCapabilityLabelCN(mode string) string {
	switch strings.TrimSpace(mode) {
	case capabilityImageToVideo:
		return "图生视频"
	case capabilityVideoToVideo:
		return "视频生视频"
	default:
		return "文生视频"
	}
}

func imageCapabilityLabelCN(mode string) string {
	if strings.TrimSpace(mode) == capabilityImageToImage {
		return "图生图"
	}
	return "文生图"
}

func ensureCapabilityIndex(idx capabilityPricingIndex, cap string) *capabilityPricingConfig {
	if idx[cap] == nil {
		idx[cap] = &capabilityPricingConfig{resolutions: make(map[string]struct{})}
	}
	return idx[cap]
}

func addCapabilityResolution(idx capabilityPricingIndex, cap, raw string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return
	}
	cfg := ensureCapabilityIndex(idx, cap)
	norm := normalizePricingResolutionLabel(raw)
	if norm == "" {
		return
	}
	cfg.hasResolutionTiers = true
	if _, exists := cfg.resolutions[norm]; exists {
		return
	}
	cfg.resolutions[norm] = struct{}{}
	display := common.FormatVideoResolutionLabel(raw)
	if display == "" {
		display = raw
	}
	cfg.displayResolutions = append(cfg.displayResolutions, display)
}

func markCapabilityFlat(idx capabilityPricingIndex, cap string) {
	ensureCapabilityIndex(idx, cap).hasFlatPricing = true
}

func addVideoAudioPriceRows(idx capabilityPricingIndex, cap string, rows ...[]ratio_setting.VideoResolutionAudioPriceRule) {
	for _, list := range rows {
		for _, row := range list {
			if row.Price <= 0 {
				continue
			}
			addCapabilityResolution(idx, cap, row.Resolution)
		}
	}
}

func addVideoPerVideoRows(idx capabilityPricingIndex, cap string, rows ...[]ratio_setting.VideoResolutionPerVideoRule) {
	for _, list := range rows {
		for _, row := range list {
			if row.VideoPrice <= 0 {
				continue
			}
			addCapabilityResolution(idx, cap, row.Resolution)
		}
	}
}

func addVideoTokenRows(idx capabilityPricingIndex, cap string, rows []ratio_setting.VideoResolutionPriceRule) {
	for _, row := range rows {
		if row.TokenPrice <= 0 {
			continue
		}
		addCapabilityResolution(idx, cap, row.Resolution)
	}
}

func ingestVideoPricingRules(idx capabilityPricingIndex, rules ratio_setting.VideoPricingRules) {
	addVideoAudioPriceRows(idx, capabilityTextToVideo,
		rules.TextToVideoPerSecond, rules.TextToVideoPerItem, rules.TextToVideoPerToken)
	addVideoPerVideoRows(idx, capabilityTextToVideo, rules.TextToVideoPerVideo)
	addVideoTokenRows(idx, capabilityTextToVideo, rules.TextToVideo)

	addVideoAudioPriceRows(idx, capabilityImageToVideo,
		rules.ImageToVideoPerSecond, rules.ImageToVideoPerItem, rules.ImageToVideoPerToken)
	addVideoPerVideoRows(idx, capabilityImageToVideo, rules.ImageToVideoPerVideo)
	addVideoTokenRows(idx, capabilityImageToVideo, rules.ImageToVideoRules)
	if rules.ImageToVideo != nil && rules.ImageToVideo.TokenPrice > 0 {
		markCapabilityFlat(idx, capabilityImageToVideo)
	}

	addVideoAudioPriceRows(idx, capabilityVideoToVideo,
		rules.VideoToVideoPerSecond, rules.VideoToVideoPerItem, rules.VideoToVideoPerToken)
	addVideoPerVideoRows(idx, capabilityVideoToVideo,
		rules.VideoToVideoInputPerVideo, rules.VideoToVideoOutputPerVideo)
	addVideoTokenRows(idx, capabilityVideoToVideo, rules.VideoToVideoInput)
	addVideoTokenRows(idx, capabilityVideoToVideo, rules.VideoToVideoOutput)
	addVideoTokenRows(idx, capabilityVideoToVideo, rules.VideoToVideo)
}

func addImageCapabilityResolution(idx capabilityPricingIndex, cap, raw string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return
	}
	cfg := ensureCapabilityIndex(idx, cap)
	norm := normalizeImagePricingResolutionLabel(raw)
	if norm == "" {
		return
	}
	cfg.hasResolutionTiers = true
	if _, exists := cfg.resolutions[norm]; exists {
		return
	}
	cfg.resolutions[norm] = struct{}{}
	display := common.FormatImageResolutionLabel(raw)
	if display == "" {
		display = raw
	}
	cfg.displayResolutions = append(cfg.displayResolutions, display)
}

func ingestImagePricingRules(idx capabilityPricingIndex, rules ratio_setting.ImagePricingRules) {
	for _, row := range rules.TextToImagePerImage {
		if row.ImagePrice <= 0 {
			continue
		}
		addImageCapabilityResolution(idx, capabilityTextToImage, row.Resolution)
	}
	for _, row := range rules.ImageToImagePerImage {
		if row.ImagePrice <= 0 {
			continue
		}
		addImageCapabilityResolution(idx, capabilityImageToImage, row.Resolution)
	}
}

func collectVideoCapabilityPricing(channelID int, modelName string) capabilityPricingIndex {
	idx := make(capabilityPricingIndex)
	if rules, ok := ratio_setting.GetChannelVideoPricingRules(channelID, modelName); ok {
		ingestVideoPricingRules(idx, rules)
	}
	if rules, ok := ratio_setting.GetVideoPricingRules(modelName); ok {
		ingestVideoPricingRules(idx, rules)
	}
	return idx
}

func collectImageCapabilityPricing(channelID int, names []string) capabilityPricingIndex {
	idx := make(capabilityPricingIndex)
	for _, name := range names {
		if name == "" {
			continue
		}
		if rules, ok := ratio_setting.GetChannelImagePricingRules(channelID, name); ok {
			ingestImagePricingRules(idx, rules)
		}
		if rules, ok := ratio_setting.GetImagePricingRules(name); ok {
			ingestImagePricingRules(idx, rules)
		}
	}
	return idx
}

func sortedCapabilityLabelsCN(order []string, idx capabilityPricingIndex, labelFn func(string) string) []string {
	out := make([]string, 0, len(idx))
	for _, cap := range order {
		if cfg := idx[cap]; cfg != nil && cfg.configured() {
			out = append(out, labelFn(cap))
		}
	}
	return out
}

func sortedDisplayResolutions(cfg *capabilityPricingConfig) []string {
	if cfg == nil || len(cfg.displayResolutions) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(cfg.displayResolutions))
	out := make([]string, 0, len(cfg.displayResolutions))
	for _, res := range cfg.displayResolutions {
		key := normalizePricingResolutionLabel(res)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, res)
	}
	sort.Slice(out, func(i, j int) bool {
		return normalizePricingResolutionLabel(out[i]) < normalizePricingResolutionLabel(out[j])
	})
	return out
}

func collectAllDisplayResolutions(idx capabilityPricingIndex, capabilityOrder []string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, 8)
	for _, cap := range capabilityOrder {
		for _, res := range sortedDisplayResolutions(idx[cap]) {
			key := normalizePricingResolutionLabel(res)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, res)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return normalizePricingResolutionLabel(out[i]) < normalizePricingResolutionLabel(out[j])
	})
	return out
}

func formatCapabilityList(items []string) string {
	if len(items) == 0 {
		return "（无）"
	}
	return strings.Join(items, "、")
}

func formatResolutionList(items []string) string {
	if len(items) == 0 {
		return "（无）"
	}
	return strings.Join(items, "、")
}

// newModelPriceFriendlyError 统一 model_price_error 友好提示。
// 模板：{模型类型} {模型名称}不支持{当前调用}，仅支持{能力列表}，可用分辨率：{分辨率列表}
// 当「当前调用」仅是已支持的能力名（如文生图）且已配置分辨率时，避免出现
// 「不支持文生图，仅支持文生图」这类自相矛盾文案，改为强调分辨率档位未匹配。
func newModelPriceFriendlyError(modelKind, modelName, currentInvocation string, supportedCaps, resolutions []string) error {
	currentInvocation = strings.TrimSpace(currentInvocation)
	if currentInvocation == "" {
		currentInvocation = "当前请求"
	}
	matchName := ratio_setting.FormatMatchingModelName(modelName)
	if matchName == "" {
		matchName = strings.TrimSpace(modelName)
	}
	var msg string
	if len(resolutions) > 0 && invocationIsSupportedCapabilityOnly(currentInvocation, supportedCaps) {
		msg = fmt.Sprintf("%s %s无法匹配当前分辨率计费档位，仅支持%s，可用分辨率：%s",
			modelKind, matchName,
			formatCapabilityList(supportedCaps),
			formatResolutionList(resolutions),
		)
	} else {
		msg = fmt.Sprintf("%s %s不支持%s，仅支持%s，可用分辨率：%s",
			modelKind, matchName, currentInvocation,
			formatCapabilityList(supportedCaps),
			formatResolutionList(resolutions),
		)
	}
	return types.NewError(errors.New(msg), types.ErrorCodeModelPriceError, types.ErrOptionWithSkipRetry())
}

func invocationIsSupportedCapabilityOnly(invocation string, supportedCaps []string) bool {
	invocation = strings.TrimSpace(invocation)
	if invocation == "" {
		return false
	}
	for _, cap := range supportedCaps {
		if strings.TrimSpace(cap) == invocation {
			return true
		}
	}
	return false
}

func videoRequestResolutionLabel(c *gin.Context, ctx videoEstimateContext) string {
	if req, err := relaycommon.GetTaskRequest(c); err == nil {
		if label := service.VideoBillingResolutionLabelFromRequest(req); label != "" {
			return label
		}
	}
	if ctx.Width > 0 && ctx.Height > 0 {
		return common.FormatVideoResolutionLabel(fmt.Sprintf("%dx%d", ctx.Width, ctx.Height))
	}
	return ""
}

func imageRequestResolutionLabel(ctx imageEstimateContext) string {
	if ctx.Width > 0 && ctx.Height > 0 {
		return common.FormatImageResolutionLabel(fmt.Sprintf("%dx%d", ctx.Width, ctx.Height))
	}
	return ""
}

func formatInvocationWithResolution(resLabel, capabilityCN string) string {
	capabilityCN = strings.TrimSpace(capabilityCN)
	resLabel = strings.TrimSpace(resLabel)
	if resLabel == "" {
		return capabilityCN
	}
	return resLabel + capabilityCN
}

// validateVideoModelPrice 视频计费前置双重校验：能力 -> 分辨率。
func validateVideoModelPrice(c *gin.Context, channelID int, modelName string) error {
	idx := collectVideoCapabilityPricing(channelID, modelName)
	estimateCtx := estimateVideoRequestContext(c)
	mode := string(estimateCtx.Mode)
	capabilityCN := videoCapabilityLabelCN(mode)
	supportedCaps := sortedCapabilityLabelsCN(
		[]string{capabilityTextToVideo, capabilityImageToVideo, capabilityVideoToVideo},
		idx, videoCapabilityLabelCN,
	)

	// ① 能力校验：当前调用能力必须在已配置计费能力内。
	if len(idx) == 0 {
		return newModelPriceFriendlyError("视频模型", modelName, capabilityCN, supportedCaps, nil)
	}
	cfg := idx[mode]
	if cfg == nil || !cfg.configured() {
		return newModelPriceFriendlyError("视频模型", modelName, capabilityCN, supportedCaps,
			collectAllDisplayResolutions(idx, []string{capabilityTextToVideo, capabilityImageToVideo, capabilityVideoToVideo}))
	}

	// ② 分辨率校验：能力匹配后，请求分辨率必须存在于该能力已配置档位中（禁止静默回退到其它档位）。
	if !cfg.hasResolutionTiers {
		return nil
	}
	reqLabel := videoRequestResolutionLabel(c, estimateCtx)
	if reqLabel == "" {
		return nil
	}
	if _, ok := cfg.resolutions[normalizePricingResolutionLabel(reqLabel)]; ok {
		return nil
	}
	currentInvocation := formatInvocationWithResolution(reqLabel, capabilityCN)
	return newModelPriceFriendlyError("视频模型", modelName, currentInvocation, supportedCaps, sortedDisplayResolutions(cfg))
}

// ImageModelPriceMatchError 图片按张规则已配置但未能匹配价格时的统一友好提示。
func ImageModelPriceMatchError(c *gin.Context, channelID int, info *relaycommon.RelayInfo) error {
	if info == nil {
		return newModelPriceFriendlyError("图片模型", "", "当前请求", nil, nil)
	}
	names := imageModelNameCandidatesFromInfo(info)
	if len(names) == 0 {
		names = imageModelNameCandidates(info.OriginModelName)
	}
	estimateCtx := estimateImageRequestContext(c, info)
	currentInvocation := imageCapabilityLabelCN(string(estimateCtx.Mode))
	if label := imageRequestResolutionLabel(estimateCtx); label != "" {
		currentInvocation = formatInvocationWithResolution(label, imageCapabilityLabelCN(string(estimateCtx.Mode)))
	}
	idx := collectImageCapabilityPricing(channelID, names)
	supportedCaps := sortedCapabilityLabelsCN(
		[]string{capabilityTextToImage, capabilityImageToImage},
		idx, imageCapabilityLabelCN,
	)
	allRes := collectAllDisplayResolutions(idx, []string{capabilityTextToImage, capabilityImageToImage})
	return newModelPriceFriendlyError("图片模型", info.OriginModelName, currentInvocation, supportedCaps, allRes)
}

// validateImageModelPrice 图片计费前置双重校验：能力 -> 分辨率。
func validateImageModelPrice(c *gin.Context, channelID int, info *relaycommon.RelayInfo) error {
	names := imageModelNameCandidatesFromInfo(info)
	if len(names) == 0 && info != nil {
		names = imageModelNameCandidates(info.OriginModelName)
	}
	idx := collectImageCapabilityPricing(channelID, names)
	estimateCtx := estimateImageRequestContext(c, info)
	mode := string(estimateCtx.Mode)
	capabilityCN := imageCapabilityLabelCN(mode)
	supportedCaps := sortedCapabilityLabelsCN(
		[]string{capabilityTextToImage, capabilityImageToImage},
		idx, imageCapabilityLabelCN,
	)

	if len(idx) == 0 {
		return nil
	}

	// ① 能力校验
	cfg := idx[mode]
	if cfg == nil || !cfg.configured() {
		return newModelPriceFriendlyError("图片模型", info.OriginModelName, capabilityCN, supportedCaps,
			collectAllDisplayResolutions(idx, []string{capabilityTextToImage, capabilityImageToImage}))
	}

	// ② 分辨率校验
	if !cfg.hasResolutionTiers {
		return nil
	}
	reqLabel := imageRequestResolutionLabel(estimateCtx)
	if reqLabel == "" {
		return nil
	}
	if _, ok := cfg.resolutions[normalizeImagePricingResolutionLabel(reqLabel)]; ok {
		return nil
	}
	currentInvocation := formatInvocationWithResolution(reqLabel, capabilityCN)
	return newModelPriceFriendlyError("图片模型", info.OriginModelName, currentInvocation, supportedCaps, sortedDisplayResolutions(cfg))
}
