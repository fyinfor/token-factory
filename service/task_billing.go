package service

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
)

// LogTaskConsumption 记录任务消费日志和统计信息（仅记录，不涉及实际扣费）。
// 实际扣费已由 BillingSession（PreConsumeBilling + SettleBilling）完成。
func LogTaskConsumption(c *gin.Context, info *relaycommon.RelayInfo) {
	tokenName := c.GetString("token_name")
	logContent := fmt.Sprintf("操作 %s", info.Action)

	// 视频按 token 计费分支：任务型视频渠道 + UsePrice + ModelPrice=0 + VideoOutputTokens>0。
	// 该分支下 quota 已由 outputVideoTokens × ratios × group 直接算出，
	// OtherRatios 的 seconds/size 不参与计费（已在 relay_task.go 步骤 5/6 跳过），
	// 因此 logContent 应展示真实公式而非 "计算参数：seconds, size"。
	isVideoTokenBilling := constant.IsVideoTaskChannel(info.ChannelType) &&
		info.PriceData.UsePrice &&
		info.PriceData.ModelPrice == 0 &&
		info.PriceData.VideoOutputTokens > 0

	// 视频按分辨率/条一口价（*_per_video）：ModelPriceHelperVideo 将 ModelRatio 置 0、
	// VideoOutputTokens 为 0，预扣已在 relay 中按条合并，不应再展示为「按次 $0」或 seconds 倍率文案。
	isVideoPerVideoFlatBilling := constant.IsVideoTaskChannel(info.ChannelType) &&
		info.PriceData.UsePrice &&
		info.PriceData.ModelPrice == 0 &&
		info.PriceData.VideoOutputTokens == 0 &&
		info.PriceData.ModelRatio == 0
	isVideoPerSecondBilling := isVideoPerVideoFlatBilling &&
		info.PriceData.OtherRatios != nil &&
		info.PriceData.OtherRatios["seconds"] > 0
	var videoPerSecondDetail *videoPerSecondBillingDetail
	if isVideoPerSecondBilling {
		videoPerSecondDetail = videoPerSecondBillingDetailFromSubmit(c, info)
	}

	switch {
	case common.StringsContains(constant.TaskPricePatches, info.OriginModelName):
		logContent = fmt.Sprintf("%s，按次计费", logContent)
	case isVideoTokenBilling:
		// 例：操作 generate, 视频 tokens：86400 (输入文本 13), 模型倍率 15.00, 视频倍率 1.00 × 1.00
		logContent = fmt.Sprintf(
			"%s, 视频 tokens：%d (输入文本 %d), 模型倍率 %.2f, 视频倍率 %.2f × %.2f",
			logContent,
			info.PriceData.VideoOutputTokens,
			info.PriceData.VideoInputTextTokens,
			info.PriceData.ModelRatio,
			info.PriceData.VideoRatio,
			info.PriceData.VideoCompletionRatio,
		)
	case isVideoPerSecondBilling:
		logContent = formatVideoPerSecondBillingDetail(logContent+"，视频按秒计费", videoPerSecondDetail, info.PriceData.Quota)
	case isVideoPerVideoFlatBilling:
		logContent = fmt.Sprintf("%s，按视频数量计费", logContent)
	default:
		if len(info.PriceData.OtherRatios) > 0 {
			var contents []string
			for key, ra := range info.PriceData.OtherRatios {
				if 1.0 != ra {
					contents = append(contents, fmt.Sprintf("%s: %.2f", key, ra))
				}
			}
			if len(contents) > 0 {
				logContent = fmt.Sprintf("%s, 计算参数：%s", logContent, strings.Join(contents, ", "))
			}
		}
	}

	other := make(map[string]interface{})
	other["request_path"] = c.Request.URL.Path
	if strings.TrimSpace(info.PublicTaskID) != "" {
		other["task_id"] = strings.TrimSpace(info.PublicTaskID)
	}
	other["model_price"] = info.PriceData.ModelPrice
	other["group_ratio"] = info.PriceData.GroupRatioInfo.GroupRatio
	if info.PriceData.GroupRatioInfo.HasSpecialRatio {
		other["user_group_ratio"] = info.PriceData.GroupRatioInfo.GroupSpecialRatio
	}
	if info.IsModelMapped {
		other["is_model_mapped"] = true
		other["upstream_model_name"] = info.UpstreamModelName
	}
	// 视频按 token 计费：写入完整计费元数据，供前端日志详情按 token 公式展示。
	if isVideoTokenBilling {
		other["billing_mode"] = "video_token"
		other["model_ratio"] = info.PriceData.ModelRatio
		other["video_ratio"] = info.PriceData.VideoRatio
		other["video_completion_ratio"] = info.PriceData.VideoCompletionRatio
		other["video_output_tokens"] = info.PriceData.VideoOutputTokens
		other["video_input_text_tokens"] = info.PriceData.VideoInputTextTokens
	}
	if isVideoPerSecondBilling {
		other["billing_mode"] = "video_per_second"
		other["model_ratio"] = info.PriceData.ModelRatio
		appendVideoPerSecondBillingDetailOther(other, videoPerSecondDetail, info.PriceData.Quota)
	} else if isVideoPerVideoFlatBilling {
		other["billing_mode"] = "video_per_video"
		other["model_ratio"] = info.PriceData.ModelRatio
		appendVideoPerVideoBillingDetailOther(c, other, info)
	}
	discPct := float64(100)
	if info.PriceData.ChannelPriceDiscount != nil {
		discPct = *info.PriceData.ChannelPriceDiscount
	} else {
		discPct = model.ResolveChannelPriceDiscountPercent(info.ChannelId)
	}
	other["channel_price_discount_percent"] = discPct
	model.RecordConsumeLog(c, info.UserId, model.RecordConsumeLogParams{
		ChannelId: info.ChannelId,
		ModelName: info.OriginModelName,
		TokenName: tokenName,
		Quota:     info.PriceData.Quota,
		Content:   logContent,
		TokenId:   info.TokenId,
		Group:     info.UsingGroup,
		Other:     other,
	})
	model.UpdateUserUsedQuotaAndRequestCount(info.UserId, info.PriceData.Quota)
	model.UpdateChannelUsedQuota(info.ChannelId, info.PriceData.Quota)
}

// ---------------------------------------------------------------------------
// 异步任务计费辅助函数
// ---------------------------------------------------------------------------

// resolveTokenKey 通过 TokenId 运行时获取令牌 Key（用于 Redis 缓存操作）。
// 如果令牌已被删除或查询失败，返回空字符串。
func resolveTokenKey(ctx context.Context, tokenId int, taskID string) string {
	token, err := model.GetTokenById(tokenId)
	if err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("获取令牌 key 失败 (tokenId=%d, task=%s): %s", tokenId, taskID, err.Error()))
		return ""
	}
	return token.Key
}

// taskIsSubscription 判断任务是否通过订阅计费。
func taskIsSubscription(task *model.Task) bool {
	return task.PrivateData.BillingSource == BillingSourceSubscription && task.PrivateData.SubscriptionId > 0
}

// taskAdjustFunding 调整任务的资金来源（钱包或订阅），delta > 0 表示扣费，delta < 0 表示退还。
func taskAdjustFunding(task *model.Task, delta int) error {
	if taskIsSubscription(task) {
		return model.PostConsumeUserSubscriptionDelta(task.PrivateData.SubscriptionId, int64(delta))
	}
	if delta > 0 {
		return model.DecreaseUserQuota(task.UserId, delta)
	}
	return model.IncreaseUserQuota(task.UserId, -delta, false)
}

// taskAdjustTokenQuota 调整任务的令牌额度，delta > 0 表示扣费，delta < 0 表示退还。
// 需要通过 resolveTokenKey 运行时获取 key（不从 PrivateData 中读取）。
func taskAdjustTokenQuota(ctx context.Context, task *model.Task, delta int) {
	if task.PrivateData.TokenId <= 0 || delta == 0 {
		return
	}
	tokenKey := resolveTokenKey(ctx, task.PrivateData.TokenId, task.TaskID)
	if tokenKey == "" {
		return
	}
	var err error
	if delta > 0 {
		err = model.DecreaseTokenQuota(task.PrivateData.TokenId, tokenKey, delta)
	} else {
		err = model.IncreaseTokenQuota(task.PrivateData.TokenId, tokenKey, -delta)
	}
	if err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("调整令牌额度失败 (delta=%d, task=%s): %s", delta, task.TaskID, err.Error()))
	}
}

// taskBillingOther 从 task 的 BillingContext 构建日志 Other 字段。
func taskBillingOther(task *model.Task) map[string]interface{} {
	other := make(map[string]interface{})
	if bc := task.PrivateData.BillingContext; bc != nil {
		other["model_price"] = bc.ModelPrice
		other["model_ratio"] = bc.ModelRatio
		other["group_ratio"] = bc.GroupRatio
		if len(bc.OtherRatios) > 0 {
			for k, v := range bc.OtherRatios {
				other[k] = v
			}
		}
		// 任务差额日志补全视频计费模式，避免前端误判为“上游返回”并渲染 NaN。
		if bc.ModelPrice == 0 && bc.ModelRatio == 0 {
			if secs, ok := bc.OtherRatios["seconds"]; ok && secs > 0 {
				other["billing_mode"] = "video_per_second"
			}
		}
	}
	props := task.Properties
	if props.UpstreamModelName != "" && props.UpstreamModelName != props.OriginModelName {
		other["is_model_mapped"] = true
		other["upstream_model_name"] = props.UpstreamModelName
	}
	discPct := float64(0)
	if bc := task.PrivateData.BillingContext; bc != nil && bc.ChannelPriceDiscountPercent > 0 {
		discPct = bc.ChannelPriceDiscountPercent
	}
	if discPct <= 0 && task.ChannelId > 0 {
		discPct = model.ResolveChannelPriceDiscountPercent(task.ChannelId)
	}
	if discPct <= 0 {
		discPct = 100
	}
	other["channel_price_discount_percent"] = discPct
	return other
}

func videoPerSecondBillingDetailFromSubmit(c *gin.Context, info *relaycommon.RelayInfo) *videoPerSecondBillingDetail {
	if c == nil || info == nil {
		return nil
	}
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil
	}
	modelName := strings.TrimSpace(info.OriginModelName)
	if modelName == "" {
		return nil
	}
	rules, ok := ratio_setting.GetChannelVideoPricingRules(info.ChannelId, modelName)
	if !ok || !ratio_setting.HasUsableVideoPerSecondRules(rules) {
		var globalOK bool
		rules, globalOK = ratio_setting.GetVideoPricingRules(modelName)
		if !globalOK || !ratio_setting.HasUsableVideoPerSecondRules(rules) {
			return nil
		}
	}
	width, height := videoDimensionsFromTaskRequest(req)
	hasAudio := taskRequestHasAudio(req)
	mode := detectVideoBillingModeFromSubmitRequest(c)
	match, ok := matchPerSecondPriceDetail(rules, mode, width, height, hasAudio)
	if !ok || match.PricePerSecond <= 0 {
		return nil
	}
	seconds := videoDurationFromTaskRequest(req)
	if seconds <= 0 {
		seconds = int(info.PriceData.OtherRatios["seconds"])
	}
	if seconds <= 0 {
		return nil
	}
	groupRatio := info.PriceData.GroupRatioInfo.GroupRatio
	if groupRatio <= 0 {
		groupRatio = 1
	}
	return &videoPerSecondBillingDetail{
		Mode:                   mode,
		Seconds:                seconds,
		Width:                  width,
		Height:                 height,
		HasAudio:               hasAudio,
		Resolution:             match.Resolution,
		RuleWidth:              match.RuleWidth,
		RuleHeight:             match.RuleHeight,
		PricePerSecond:         match.PricePerSecond,
		GroupRatio:             groupRatio,
		QuotaPerUnit:           common.QuotaPerUnit,
		ChannelDiscountPercent: resolveVideoLogChannelDiscountPercent(info),
		UnifiedAudio:           match.UnifiedAudio,
	}
}

func resolveVideoLogChannelDiscountPercent(info *relaycommon.RelayInfo) float64 {
	if info != nil && info.PriceData.ChannelPriceDiscount != nil {
		return *info.PriceData.ChannelPriceDiscount
	}
	if info != nil {
		return model.ResolveChannelPriceDiscountPercent(info.ChannelId)
	}
	return 100
}

func videoDurationFromTaskRequest(req relaycommon.TaskSubmitReq) int {
	if req.Metadata != nil {
		if d := toInt(req.Metadata["duration"]); d > 0 {
			return d
		}
	}
	if strings.TrimSpace(req.Seconds) != "" {
		if f := toFloat64(req.Seconds); f > 0 {
			return int(math.Ceil(f))
		}
	}
	if req.Duration > 0 {
		return req.Duration
	}
	return 5
}

func videoDimensionsFromTaskRequest(req relaycommon.TaskSubmitReq) (int, int) {
	if size := strings.TrimSpace(req.Size); size != "" {
		parts := strings.Split(strings.ToLower(size), "x")
		if len(parts) == 2 {
			w := toInt(parts[0])
			h := toInt(parts[1])
			if w > 0 && h > 0 {
				return w, h
			}
		}
	}
	if req.Metadata != nil {
		w := toInt(req.Metadata["width"])
		h := toInt(req.Metadata["height"])
		if w > 0 && h > 0 {
			return w, h
		}
	}
	return 720, 1280
}

func taskRequestHasAudio(req relaycommon.TaskSubmitReq) bool {
	if req.Metadata == nil {
		return false
	}
	for _, key := range []string{"has_audio", "generate_audio"} {
		if v, ok := req.Metadata[key]; ok {
			switch x := v.(type) {
			case bool:
				return x
			case string:
				return strings.EqualFold(strings.TrimSpace(x), "true")
			}
		}
	}
	return false
}

func formatVideoPerSecondBillingDetail(prefix string, detail *videoPerSecondBillingDetail, quota int) string {
	if detail == nil {
		return fmt.Sprintf("%s（按最终成片时长向上取整 × 对应分辨率/音轨单价）", prefix)
	}
	priceLabel := "每秒价"
	if !detail.UnifiedAudio {
		if detail.HasAudio {
			priceLabel = "有音轨价"
		} else {
			priceLabel = "无音轨价"
		}
	}
	resolution := strings.TrimSpace(detail.Resolution)
	if resolution == "" {
		resolution = fmt.Sprintf("%dx%d", detail.RuleWidth, detail.RuleHeight)
	}
	return fmt.Sprintf(
		"%s：%d秒 × %s(%dx%d，实际 %dx%d，%s) %s $%g/秒 × QuotaPerUnit %.0f × 分组倍率 %.4g × 渠道折扣 %.4g%% = %d tokens",
		prefix,
		detail.Seconds,
		resolution,
		detail.RuleWidth,
		detail.RuleHeight,
		detail.Width,
		detail.Height,
		audioLabel(detail.HasAudio),
		priceLabel,
		detail.PricePerSecond,
		detail.QuotaPerUnit,
		detail.GroupRatio,
		videoChannelDiscountPercent(detail),
		quota,
	)
}

func videoChannelDiscountPercent(detail *videoPerSecondBillingDetail) float64 {
	if detail == nil || detail.ChannelDiscountPercent <= 0 {
		return 100
	}
	return detail.ChannelDiscountPercent
}

func appendVideoPerSecondBillingDetailOther(other map[string]interface{}, detail *videoPerSecondBillingDetail, quota int) {
	if other == nil || detail == nil {
		return
	}
	other["video_seconds"] = detail.Seconds
	other["video_width"] = detail.Width
	other["video_height"] = detail.Height
	other["video_has_audio"] = detail.HasAudio
	other["video_resolution"] = detail.Resolution
	other["video_rule_width"] = detail.RuleWidth
	other["video_rule_height"] = detail.RuleHeight
	other["video_price_per_second"] = detail.PricePerSecond
	other["video_quota_per_unit"] = detail.QuotaPerUnit
	other["channel_price_discount"] = videoChannelDiscountPercent(detail)
	other["video_billed_quota"] = quota
	other["video_unified_audio_price"] = detail.UnifiedAudio
}

type videoPerVideoBillingDetail struct {
	Mode                   string
	Count                  int
	Width                  int
	Height                 int
	HasAudio               bool
	Resolution             string
	RuleWidth              int
	RuleHeight             int
	PricePerVideo          float64
	GroupRatio             float64
	QuotaPerUnit           float64
	ChannelDiscountPercent float64
	UnifiedAudio           bool
}

type videoPerVideoPriceMatch struct {
	Resolution    string
	RuleWidth     int
	RuleHeight    int
	PricePerVideo float64
	UnifiedAudio  bool
}

func videoPerVideoBillingDetailFromSubmit(c *gin.Context, info *relaycommon.RelayInfo, quota int) *videoPerVideoBillingDetail {
	if c == nil || info == nil {
		return nil
	}
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil
	}
	modelName := strings.TrimSpace(info.OriginModelName)
	if modelName == "" {
		return nil
	}
	rules, ok := ratio_setting.GetChannelVideoPricingRules(info.ChannelId, modelName)
	if !ok || !ratio_setting.HasUsableVideoPerVideoRules(rules) {
		var globalOK bool
		rules, globalOK = ratio_setting.GetVideoPricingRules(modelName)
		if !globalOK || !ratio_setting.HasUsableVideoPerVideoRules(rules) {
			return nil
		}
	}
	width, height := videoDimensionsFromTaskRequest(req)
	hasAudio := taskRequestHasAudio(req)
	mode := detectVideoBillingModeFromSubmitRequest(c)
	match, ok := matchPerVideoPriceDetail(rules, mode, width, height, hasAudio)
	if !ok || match.PricePerVideo <= 0 {
		return nil
	}
	groupRatio := info.PriceData.GroupRatioInfo.GroupRatio
	if groupRatio <= 0 {
		groupRatio = 1
	}
	count := 1
	finalPricePerVideo := match.PricePerVideo * groupRatio * (resolveVideoLogChannelDiscountPercent(info) / 100)
	if common.QuotaPerUnit > 0 && quota > 0 {
		finalPricePerVideo = float64(quota) / common.QuotaPerUnit / float64(count)
	}
	return &videoPerVideoBillingDetail{
		Mode:                   mode,
		Count:                  count,
		Width:                  width,
		Height:                 height,
		HasAudio:               hasAudio,
		Resolution:             match.Resolution,
		RuleWidth:              match.RuleWidth,
		RuleHeight:             match.RuleHeight,
		PricePerVideo:          finalPricePerVideo,
		GroupRatio:             groupRatio,
		QuotaPerUnit:           common.QuotaPerUnit,
		ChannelDiscountPercent: resolveVideoLogChannelDiscountPercent(info),
		UnifiedAudio:           match.UnifiedAudio,
	}
}

func matchPerVideoPriceDetail(r ratio_setting.VideoPricingRules, mode string, width, height int, hasAudio bool) (*videoPerVideoPriceMatch, bool) {
	var rows []ratio_setting.VideoResolutionAudioPriceRule
	switch mode {
	case "image_to_video":
		rows = r.ImageToVideoPerItem
	case "video_to_video":
		rows = r.VideoToVideoPerItem
	default:
		rows = r.TextToVideoPerItem
	}
	if match, ok := matchPerSecondPriceDetail(ratio_setting.VideoPricingRules{
		TextToVideoPerSecond: rows,
	}, "text_to_video", width, height, hasAudio); ok {
		return &videoPerVideoPriceMatch{
			Resolution:    match.Resolution,
			RuleWidth:     match.RuleWidth,
			RuleHeight:    match.RuleHeight,
			PricePerVideo: match.PricePerSecond,
			UnifiedAudio:  match.UnifiedAudio,
		}, true
	}

	switch mode {
	case "image_to_video":
		return matchLegacyPerVideoRulesByPixelsDetail(width, height, r.ImageToVideoPerVideo)
	case "video_to_video":
		return matchLegacyVideoToVideoRulesByPixelsDetail(width, height, r.VideoToVideoInputPerVideo, r.VideoToVideoOutputPerVideo)
	default:
		return matchLegacyPerVideoRulesByPixelsDetail(width, height, r.TextToVideoPerVideo)
	}
}

func matchLegacyVideoToVideoRulesByPixelsDetail(width, height int, inputRows, outputRows []ratio_setting.VideoResolutionPerVideoRule) (*videoPerVideoPriceMatch, bool) {
	input, inputOK := matchLegacyPerVideoRulesByPixelsDetail(width, height, inputRows)
	output, outputOK := matchLegacyPerVideoRulesByPixelsDetail(width, height, outputRows)
	if !inputOK && !outputOK {
		return nil, false
	}
	if inputOK && outputOK {
		output.PricePerVideo += input.PricePerVideo
		return output, true
	}
	if outputOK {
		return output, true
	}
	return input, true
}

func matchLegacyPerVideoRulesByPixelsDetail(width, height int, rows []ratio_setting.VideoResolutionPerVideoRule) (*videoPerVideoPriceMatch, bool) {
	if len(rows) == 0 || width <= 0 || height <= 0 {
		return nil, false
	}
	targetPixels := width * height
	targetRatio := targetVideoResolutionRatio(width, height)
	best := -1
	minDiffRatio := math.MaxFloat64
	bestW, bestH := 0, 0
	for i, row := range rows {
		if row.VideoPrice <= 0 {
			continue
		}
		ruleW, ruleH, ok := parseVideoResolutionFlexibleForRatio(row.Resolution, targetRatio)
		if !ok || ruleW <= 0 || ruleH <= 0 {
			continue
		}
		rulePixels := ruleW * ruleH
		diffRatio := math.Abs(float64(targetPixels-rulePixels)) / float64(rulePixels)
		if diffRatio < minDiffRatio {
			minDiffRatio = diffRatio
			best = i
			bestW = ruleW
			bestH = ruleH
		}
	}
	if best < 0 {
		return nil, false
	}
	row := rows[best]
	return &videoPerVideoPriceMatch{
		Resolution:    row.Resolution,
		RuleWidth:     bestW,
		RuleHeight:    bestH,
		PricePerVideo: row.VideoPrice,
		UnifiedAudio:  true,
	}, true
}

func appendVideoPerVideoBillingDetailOther(c *gin.Context, other map[string]interface{}, info *relaycommon.RelayInfo) {
	if other == nil || info == nil {
		return
	}
	quota := info.PriceData.Quota
	if quota < 0 {
		quota = 0
	}
	videoCount := 1
	quotaPerUnit := common.QuotaPerUnit
	finalPricePerVideo := 0.0
	if quotaPerUnit > 0 && videoCount > 0 {
		finalPricePerVideo = float64(quota) / quotaPerUnit / float64(videoCount)
	}
	other["video_count"] = videoCount
	other["video_price_per_video"] = finalPricePerVideo
	other["video_quota_per_unit"] = quotaPerUnit
	other["channel_price_discount"] = resolveVideoLogChannelDiscountPercent(info)
	other["video_billed_quota"] = quota

	if detail := videoPerVideoBillingDetailFromSubmit(c, info, quota); detail != nil {
		other["video_count"] = detail.Count
		other["video_width"] = detail.Width
		other["video_height"] = detail.Height
		other["video_has_audio"] = detail.HasAudio
		other["video_resolution"] = detail.Resolution
		other["video_rule_width"] = detail.RuleWidth
		other["video_rule_height"] = detail.RuleHeight
		other["video_price_per_video"] = detail.PricePerVideo
		other["video_quota_per_unit"] = detail.QuotaPerUnit
		other["channel_price_discount"] = detail.ChannelDiscountPercent
		other["video_unified_audio_price"] = detail.UnifiedAudio
		return
	}

	if c == nil {
		return
	}
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return
	}
	width, height := videoDimensionsFromTaskRequest(req)
	if width > 0 {
		other["video_width"] = width
	}
	if height > 0 {
		other["video_height"] = height
	}
	if duration := videoDurationFromTaskRequest(req); duration > 0 {
		other["video_seconds"] = duration
	}
	other["video_has_audio"] = taskRequestHasAudio(req)
}

func videoPerSecondBillingDetailOther(detail *videoPerSecondBillingDetail, quota int) map[string]interface{} {
	other := make(map[string]interface{})
	appendVideoPerSecondBillingDetailOther(other, detail, quota)
	return other
}

func audioLabel(hasAudio bool) string {
	if hasAudio {
		return "有音轨"
	}
	return "无音轨"
}

// taskModelName 从 BillingContext 或 Properties 中获取模型名称。
func taskModelName(task *model.Task) string {
	if bc := task.PrivateData.BillingContext; bc != nil && bc.OriginModelName != "" {
		return bc.OriginModelName
	}
	return task.Properties.OriginModelName
}

// RefundTaskQuota 统一的任务失败退款逻辑。
// 当异步任务失败时，将预扣的 quota 退还给用户（支持钱包和订阅），并退还令牌额度。
func RefundTaskQuota(ctx context.Context, task *model.Task, reason string) {
	quota := task.Quota
	if quota == 0 {
		return
	}

	// 1. 退还资金来源（钱包或订阅）
	if err := taskAdjustFunding(task, -quota); err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("退还资金来源失败 task %s: %s", task.TaskID, err.Error()))
		return
	}

	// 2. 退还令牌额度
	taskAdjustTokenQuota(ctx, task, -quota)

	// 3. 记录日志
	other := taskBillingOther(task)
	other["task_id"] = task.TaskID
	other["reason"] = reason
	model.RecordTaskBillingLog(model.RecordTaskBillingLogParams{
		UserId:    task.UserId,
		LogType:   model.LogTypeRefund,
		Content:   "",
		ChannelId: task.ChannelId,
		ModelName: taskModelName(task),
		TokenName: task.PrivateData.TokenName,
		Quota:     quota,
		TokenId:   task.PrivateData.TokenId,
		Group:     task.Group,
		Other:     other,
	})
}

// RecalculateTaskQuota 通用的异步差额结算。
// actualQuota 是任务完成后的实际应扣额度，与预扣额度 (task.Quota) 做差额结算。
// reason 用于日志记录（例如 "token重算" 或 "adaptor调整"）。
func RecalculateTaskQuota(ctx context.Context, task *model.Task, actualQuota int, reason string, extraOther ...map[string]interface{}) {
	if actualQuota <= 0 {
		return
	}
	preConsumedQuota := task.Quota
	quotaDelta := actualQuota - preConsumedQuota

	if quotaDelta == 0 {
		logger.LogInfo(ctx, fmt.Sprintf("任务 %s 预扣费准确（%s，%s）",
			task.TaskID, logger.LogQuota(actualQuota), reason))
		return
	}

	logger.LogInfo(ctx, fmt.Sprintf("任务 %s 差额结算：delta=%s（实际：%s，预扣：%s，%s）",
		task.TaskID,
		logger.LogQuota(quotaDelta),
		logger.LogQuota(actualQuota),
		logger.LogQuota(preConsumedQuota),
		reason,
	))

	// 调整资金来源
	if err := taskAdjustFunding(task, quotaDelta); err != nil {
		logger.LogError(ctx, fmt.Sprintf("差额结算资金调整失败 task %s: %s", task.TaskID, err.Error()))
		return
	}

	// 调整令牌额度
	taskAdjustTokenQuota(ctx, task, quotaDelta)

	task.Quota = actualQuota
	if task.ID > 0 {
		if err := model.DB.Model(&model.Task{}).Where("id = ?", task.ID).Update("quota", actualQuota).Error; err != nil {
			logger.LogWarn(ctx, fmt.Sprintf("更新任务实际计费额度失败 task %s: %s", task.TaskID, err.Error()))
		}
	}

	var logType int
	var logQuota int
	if quotaDelta > 0 {
		logType = model.LogTypeConsume
		logQuota = quotaDelta
		model.UpdateUserUsedQuotaAndRequestCount(task.UserId, quotaDelta)
		model.UpdateChannelUsedQuota(task.ChannelId, quotaDelta)
	} else {
		logType = model.LogTypeRefund
		logQuota = -quotaDelta
	}
	other := taskBillingOther(task)
	other["task_id"] = task.TaskID
	//other["reason"] = reason
	other["pre_consumed_quota"] = preConsumedQuota
	other["actual_quota"] = actualQuota
	for _, extra := range extraOther {
		for k, v := range extra {
			other[k] = v
		}
	}
	model.RecordTaskBillingLog(model.RecordTaskBillingLogParams{
		UserId:    task.UserId,
		LogType:   logType,
		Content:   reason,
		ChannelId: task.ChannelId,
		ModelName: taskModelName(task),
		TokenName: task.PrivateData.TokenName,
		Quota:     logQuota,
		TokenId:   task.PrivateData.TokenId,
		Group:     task.Group,
		Other:     other,
	})
}

// RecalculateTaskQuotaByTokens 根据实际 token 消耗重新计费（异步差额结算）。
// 当任务成功且返回了 totalTokens 时，根据模型倍率和分组倍率重新计算实际扣费额度，
// 与预扣费的差额进行补扣或退还。支持钱包和订阅计费来源。
func RecalculateTaskQuotaByTokens(ctx context.Context, task *model.Task, totalTokens int) bool {
	if totalTokens <= 0 {
		return false
	}

	modelName := taskModelName(task)

	// 获取模型价格和倍率
	modelRatio, hasRatioSetting, _ := ratio_setting.GetModelRatio(modelName)
	// 只有配置了倍率(非固定价格)时才按 token 重新计费
	if !hasRatioSetting || modelRatio <= 0 {
		return false
	}

	// 获取用户和组的倍率信息
	group := task.Group
	if group == "" {
		user, err := model.GetUserById(task.UserId, false)
		if err == nil {
			group = user.Group
		}
	}
	if group == "" {
		return false
	}

	groupRatio := ratio_setting.GetGroupRatio(group)
	userGroupRatio, hasUserGroupRatio := ratio_setting.GetGroupGroupRatio(group, group)

	var finalGroupRatio float64
	if hasUserGroupRatio {
		finalGroupRatio = userGroupRatio
	} else {
		finalGroupRatio = groupRatio
	}

	// 计算实际应扣费额度: totalTokens * modelRatio * groupRatio * 渠道折扣
	actualQuota := int(float64(totalTokens) * modelRatio * finalGroupRatio)
	actualQuota = model.ApplyChannelPriceDiscountToQuota(actualQuota, model.ResolveChannelPriceDiscountPercent(task.ChannelId))

	reason := fmt.Sprintf("token重算：tokens=%d, modelRatio=%.2f, groupRatio=%.2f, channelId=%d", totalTokens, modelRatio, finalGroupRatio, task.ChannelId)
	RecalculateTaskQuota(ctx, task, actualQuota, reason)
	return true
}
