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

const TaskBillingOtherHeader = "X-New-Api-Task-Billing-Other"

// LogTaskConsumption 记录任务消费日志和统计信息（仅记录，不涉及实际扣费）。
// 实际扣费已由 BillingSession（PreConsumeBilling + SettleBilling）完成。
func LogTaskConsumption(c *gin.Context, info *relaycommon.RelayInfo) {
	tokenName := c.GetString("token_name")
	logContent := fmt.Sprintf("操作 %s", info.Action)

	// 视频按 token 规则计费：per_token 表 + 预扣固定 token。
	isVideoPerTokenRuleBilling := info.PriceData.UsePrice &&
		info.PriceData.ModelPrice == 0 &&
		info.PriceData.VideoRuleUnit == VideoRuleUnitPerToken &&
		info.PriceData.VideoOutputTokens > 0 &&
		constant.UsesRelayVideoPricing(info.ChannelType)

	// 视频按 token 计费分支（legacy token ratio 路径）：任务型视频渠道 + UsePrice + ModelPrice=0 + VideoOutputTokens>0。
	// 该分支下 quota 已由 outputVideoTokens × ratios × group 直接算出，
	// OtherRatios 的 seconds/size 不参与计费（已在 relay_task.go 步骤 5/6 跳过），
	// 因此 logContent 应展示真实公式而非 "计算参数：seconds, size"。
	isVideoTokenBilling := info.PriceData.UsePrice &&
		info.PriceData.ModelPrice == 0 &&
		info.PriceData.VideoOutputTokens > 0 &&
		info.PriceData.VideoRuleUnit != VideoRuleUnitPerToken &&
		constant.UsesRelayVideoPricing(info.ChannelType)

	// 视频按分辨率/条一口价（*_per_video）：ModelPriceHelperVideo 将 ModelRatio 置 0、
	// VideoOutputTokens 为 0，预扣已在 relay 中按条合并，不应再展示为「按次 $0」或 seconds 倍率文案。
	isVideoPerVideoFlatBilling := info.PriceData.UsePrice &&
		info.PriceData.ModelPrice == 0 &&
		info.PriceData.VideoOutputTokens == 0 &&
		info.PriceData.ModelRatio == 0 &&
		constant.UsesRelayVideoPricing(info.ChannelType)
	isVideoPerSecondBilling := isVideoPerVideoFlatBilling &&
		info.PriceData.OtherRatios != nil &&
		info.PriceData.OtherRatios["seconds"] > 0
	var videoPerSecondDetail *videoPerSecondBillingDetail
	if isVideoPerSecondBilling {
		videoPerSecondDetail = videoPerSecondBillingDetailFromSubmit(c, info)
	}
	var videoPerTokenDetail *videoPerTokenBillingDetail
	if isVideoPerTokenRuleBilling {
		if d := videoPerTokenBillingDetailFromSubmit(c, info); d != nil {
			d.TotalTokens = info.PriceData.VideoOutputTokens
			d.IsPreCharge = true
			videoPerTokenDetail = d
		}
	}

	switch {
	case common.StringsContains(constant.TaskPricePatches, info.OriginModelName):
		logContent = fmt.Sprintf("%s，按次计费", logContent)
	case isVideoPerTokenRuleBilling:
		logContent = formatVideoPerTokenBillingDetail(logContent+"，视频按 token 计费", videoPerTokenDetail, info.PriceData.Quota)
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
	// 视频按 token 规则计费：写入完整计费元数据，供前端日志详情展示（结构与按秒计费对齐）。
	if isVideoPerTokenRuleBilling {
		other["billing_mode"] = SeedanceVideoTokenBillingMode
		appendVideoPerTokenBillingDetailOther(other, videoPerTokenDetail, info.PriceData.Quota)
	} else if isVideoTokenBilling {
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
	if upsOther := videoUpscalePreChargeOther(c, info); upsOther != nil {
		for k, v := range upsOther {
			other[k] = v
		}
	}
	appendSettlementDiscountSnapshotsFromPriceData(info.ChannelId, info.PriceData, other)
	appendTimePricingInfo(info.PriceData, other)
	if len(info.UpstreamTaskBillingOther) > 0 {
		for k, v := range info.UpstreamTaskBillingOther {
			if !isUpstreamVideoMetadataLogKey(k) {
				continue
			}
			if _, exists := other[k]; !exists {
				other[k] = v
			}
		}
	}
	other = model.SetBillingLogMetadata(other, model.BillingPhasePreCharge, true, info.PriceData.Quota, -int64(info.PriceData.Quota))
	if c != nil && !c.Writer.Written() {
		if otherJSON, err := common.Marshal(other); err == nil {
			c.Header(TaskBillingOtherHeader, string(otherJSON))
		}
	}
	// 异步任务没有真实 prompt/completion tokens，将预扣额度作为 token_used 上报，
	// 使 /rankings 排行（聚合 quota_data.token_used）能看到 Seedance/Kling/Sora 等视频模型。
	preChargeTokens := info.PriceData.Quota
	if preChargeTokens <= 0 && info.PriceData.VideoOutputTokens > 0 {
		// 兜底：使用按 token 规则计费时的预扣 token 数（按 token 视频任务的价格以这个为基准）。
		preChargeTokens = info.PriceData.VideoOutputTokens
	}
	model.RecordConsumeLog(c, info.UserId, model.RecordConsumeLogParams{
		ChannelId: info.ChannelId,
		ModelName: info.OriginModelName,
		TokenName: tokenName,
		Quota:     info.PriceData.Quota,
		TokenUsed: preChargeTokens,
		Content:   logContent,
		TokenId:   info.TokenId,
		Group:     info.UsingGroup,
		Other:     other,
	})
	recordWalletUsedQuota(info, info.UserId, info.PriceData.Quota)
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

func releaseTaskInvoiceAttribution(task *model.Task, refundQuota int) {
	if task == nil || refundQuota <= 0 || taskIsSubscription(task) || task.PrivateData.WalletPaidQuota <= 0 {
		return
	}
	paidRefund := refundQuota
	if paidRefund > task.PrivateData.WalletPaidQuota {
		paidRefund = task.PrivateData.WalletPaidQuota
	}
	if err := model.ReleaseConsumeQuotaFromTopUps(task.UserId, paidRefund); err != nil {
		common.SysLog(fmt.Sprintf("failed to release invoice attribution for task %s: %s", task.TaskID, err.Error()))
		return
	}
	task.PrivateData.WalletPaidQuota -= paidRefund
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
		for k, v := range bc.UpstreamBillingOther {
			if !isUpstreamVideoMetadataLogKey(k) {
				continue
			}
			if _, exists := other[k]; !exists {
				other[k] = v
			}
		}
		if bc.TimePricingPlanID > 0 {
			other["time_pricing_schedule_id"] = bc.TimePricingScheduleID
			other["time_pricing_plan_id"] = bc.TimePricingPlanID
			other["time_pricing_plan_version"] = bc.TimePricingPlanVersion
			other["time_pricing_schedule_name"] = bc.TimePricingScheduleName
			other["time_pricing_plan_name"] = bc.TimePricingPlanName
			other["time_pricing_timezone"] = bc.TimePricingTimezone
			if bc.TimePricingWeekdays > 0 && bc.TimePricingStartMinute != bc.TimePricingEndMinute {
				other["time_pricing_weekdays"] = bc.TimePricingWeekdays
				other["time_pricing_start_minute"] = bc.TimePricingStartMinute
				other["time_pricing_end_minute"] = bc.TimePricingEndMinute
				if bc.TimePricingEffectiveFrom != "" {
					other["time_pricing_effective_from"] = bc.TimePricingEffectiveFrom
				}
				if bc.TimePricingEffectiveTo != "" {
					other["time_pricing_effective_to"] = bc.TimePricingEffectiveTo
				}
			}
			other["time_pricing_matched_at"] = bc.TimePricingMatchedAt
		}
	}
	props := task.Properties
	if props.UpstreamModelName != "" && props.UpstreamModelName != props.OriginModelName {
		other["is_model_mapped"] = true
		other["upstream_model_name"] = props.UpstreamModelName
	}
	appendTaskBillingDiscountSnapshots(task.PrivateData.BillingContext, task.ChannelId, other)
	return other
}

func appendTaskBillingDiscountSnapshots(bc *model.TaskBillingContext, channelID int, other map[string]interface{}) {
	rawDisc := model.ResolveChannelPriceDiscountPercent(channelID)
	operatingCost := model.ResolveChannelOperatingCostPercent(channelID)
	operatingDiscount := model.EffectiveCostPercent(rawDisc, operatingCost)
	markupDisc := model.ResolveChannelMarkupDiscountRate(channelID)
	if bc != nil {
		if bc.PriceDiscountPercent != nil {
			rawDisc = *bc.PriceDiscountPercent
		}
		if bc.OperatingCostPercent != nil {
			operatingCost = *bc.OperatingCostPercent
		}
		if bc.EffectiveCostPercent != nil {
			operatingDiscount = *bc.EffectiveCostPercent
		} else if bc.ChannelPriceDiscountPercent > 0 {
			operatingDiscount = bc.ChannelPriceDiscountPercent
		} else {
			operatingDiscount = model.EffectiveCostPercent(rawDisc, operatingCost)
		}
		if bc.MarkupDiscountPercent != nil {
			markupDisc = *bc.MarkupDiscountPercent
		}
	}
	other["price_discount_percent"] = rawDisc
	other["operating_cost_percent"] = operatingCost
	other["channel_price_discount_percent"] = operatingDiscount
	other["markup_discount_rate"] = markupDisc
	other["sales_discount_percent"] = model.SalesDiscountPercent(rawDisc, operatingCost, markupDisc)
	if bc != nil {
		if strings.TrimSpace(bc.VideoBillingMode) != "" {
			other["video_billing_lane"] = strings.TrimSpace(bc.VideoBillingMode)
		}
		if bc.VideoRuleWidth > 0 {
			other["video_rule_width"] = bc.VideoRuleWidth
		}
		if bc.VideoRuleHeight > 0 {
			other["video_rule_height"] = bc.VideoRuleHeight
		}
		other["video_has_audio"] = bc.VideoRuleHasAudio
		switch strings.ToLower(strings.TrimSpace(bc.VideoRuleUnit)) {
		case "per_second":
			if bc.VideoChannelRulePrice > 0 {
				other["video_price_per_second"] = bc.VideoChannelRulePrice
			}
			if bc.VideoGlobalRulePrice > 0 {
				other["global_video_price_per_second"] = bc.VideoGlobalRulePrice
			}
		case "per_video":
			if bc.VideoChannelRulePrice > 0 {
				other["video_price_per_video"] = bc.VideoChannelRulePrice
			}
			if bc.VideoGlobalRulePrice > 0 {
				other["global_video_price_per_video"] = bc.VideoGlobalRulePrice
			}
		}
	}
}

func isUpstreamVideoMetadataLogKey(key string) bool {
	switch key {
	case "billing_mode",
		"video_total_tokens",
		"video_output_tokens",
		"video_input_text_tokens",
		"video_seconds",
		"video_duration",
		"video_width",
		"video_height",
		"video_rule_width",
		"video_rule_height",
		"video_resolution",
		"video_resolution_from_input",
		"video_ratio_label",
		"video_aspect_ratio",
		"video_ratio",
		"video_has_audio",
		"video_unified_audio_price",
		"video_capped_to_max_tier",
		"video_count",
		"video_billing_lane",
		"video_rule_unit":
		return true
	default:
		return false
	}
}

func videoPerSecondBillingDetailFromSubmit(c *gin.Context, info *relaycommon.RelayInfo) *videoPerSecondBillingDetail {
	if c == nil || info == nil {
		return nil
	}
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil
	}
	applyUpscaleTargetToBillingRequest(c, &req)
	modelName := strings.TrimSpace(info.OriginModelName)
	if modelName == "" {
		return nil
	}
	var rules ratio_setting.VideoPricingRules
	var ok bool
	hasIndependentTimePricing := false
	if info.PriceData.TimePricingPlanID > 0 && strings.TrimSpace(info.PriceData.TimePricingPayload) != "" {
		if payload, err := model.ParseChannelModelPricePlanPayload(info.PriceData.TimePricingPayload); err == nil && payload.ResolvedMode() == model.ChannelModelPricePlanModePrice {
			hasIndependentTimePricing = true
			if payload.VideoPricingRules != nil {
				rules = *payload.VideoPricingRules
				ok = ratio_setting.HasUsableVideoPerSecondRules(rules)
			}
		}
	}
	if !hasIndependentTimePricing {
		rules, ok = ratio_setting.GetChannelVideoPricingRules(info.ChannelId, modelName)
	}
	if !hasIndependentTimePricing && (!ok || !ratio_setting.HasUsableVideoPerSecondRules(rules)) {
		var globalOK bool
		rules, globalOK = ratio_setting.GetVideoPricingRules(modelName)
		if !globalOK || !ratio_setting.HasUsableVideoPerSecondRules(rules) {
			return nil
		}
	}
	width, height := videoDimensionsFromTaskRequest(req)
	hasAudio := taskRequestHasAudio(req)
	mode := detectVideoBillingModeFromSubmitRequest(c)
	resolutionLabel := VideoBillingResolutionLabelFromRequest(req)
	match, ok := matchPerSecondPriceDetail(rules, mode, width, height, hasAudio, resolutionLabel)
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
	detail := &videoPerSecondBillingDetail{
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
		CappedToMaxTier:        match.CappedToMaxTier,
	}
	applyPreChargeVideoResolution(req, &detail.Resolution, &detail.ResolutionFromRequest, match.Resolution)
	if req.Metadata != nil {
		if r, ok := req.Metadata["ratio"].(string); ok {
			detail.Ratio = strings.TrimSpace(r)
		}
	}
	if strings.TrimSpace(detail.Ratio) == "" && strings.TrimSpace(req.Ratio) != "" {
		detail.Ratio = strings.TrimSpace(req.Ratio)
	}
	fillVideoPerSecondEffectiveRates(detail, info.ChannelId, info.UserId, modelName, mode)
	return detail
}

type videoPerTokenBillingDetail struct {
	Mode                     string
	TotalTokens              int
	Width                    int
	Height                   int
	HasAudio                 bool
	Resolution               string
	RuleWidth                int
	RuleHeight               int
	Ratio                    string
	Duration                 int
	PricePerMillionTokens    float64
	GlobalPricePerMillion    float64
	EffectivePricePerMillion float64
	MarkupDiscountPercent    float64
	GroupRatio               float64
	QuotaPerUnit             float64
	ChannelDiscountPercent   float64
	UnifiedAudio             bool
	CappedToMaxTier          bool
	IsPreCharge              bool
	ResolutionFromRequest    bool
}

func videoPerTokenBillingDetailFromSubmit(c *gin.Context, info *relaycommon.RelayInfo) *videoPerTokenBillingDetail {
	if c == nil || info == nil {
		return nil
	}
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil
	}
	applyUpscaleTargetToBillingRequest(c, &req)
	modelName := strings.TrimSpace(info.OriginModelName)
	if modelName == "" {
		return nil
	}
	var rules ratio_setting.VideoPricingRules
	var ok bool
	hasIndependentTimePricing := false
	if info.PriceData.TimePricingPlanID > 0 && strings.TrimSpace(info.PriceData.TimePricingPayload) != "" {
		if payload, err := model.ParseChannelModelPricePlanPayload(info.PriceData.TimePricingPayload); err == nil && payload.ResolvedMode() == model.ChannelModelPricePlanModePrice {
			hasIndependentTimePricing = true
			if payload.VideoPricingRules != nil {
				rules = *payload.VideoPricingRules
				ok = ratio_setting.HasUsableVideoPerTokenRules(rules)
			}
		}
	}
	if !hasIndependentTimePricing {
		rules, ok = ratio_setting.GetChannelVideoPricingRules(info.ChannelId, modelName)
	}
	if !hasIndependentTimePricing && (!ok || !ratio_setting.HasUsableVideoPerTokenRules(rules)) {
		var globalOK bool
		rules, globalOK = ratio_setting.GetVideoPricingRules(modelName)
		if !globalOK || !ratio_setting.HasUsableVideoPerTokenRules(rules) {
			return nil
		}
	}
	width, height := videoDimensionsFromTaskRequest(req)
	hasAudio := taskRequestHasAudio(req)
	mode := detectVideoBillingModeFromSubmitRequest(c)
	resolutionLabel := VideoBillingResolutionLabelFromRequest(req)
	match, ok := matchPerTokenPriceDetail(rules, mode, width, height, hasAudio, resolutionLabel)
	if !ok || match.PricePerSecond <= 0 {
		return nil
	}
	groupRatio := info.PriceData.GroupRatioInfo.GroupRatio
	if groupRatio <= 0 {
		groupRatio = 1
	}
	detail := &videoPerTokenBillingDetail{
		Mode:                   mode,
		Width:                  width,
		Height:                 height,
		HasAudio:               hasAudio,
		Resolution:             match.Resolution,
		RuleWidth:              match.RuleWidth,
		RuleHeight:             match.RuleHeight,
		PricePerMillionTokens:  match.PricePerSecond,
		GroupRatio:             groupRatio,
		QuotaPerUnit:           common.QuotaPerUnit,
		ChannelDiscountPercent: resolveVideoLogChannelDiscountPercent(info),
		UnifiedAudio:           match.UnifiedAudio,
		CappedToMaxTier:        match.CappedToMaxTier,
		Duration:               videoDurationFromTaskRequest(req),
	}
	applyPreChargeVideoResolution(req, &detail.Resolution, &detail.ResolutionFromRequest, match.Resolution)
	if req.Metadata != nil {
		if r, ok := req.Metadata["ratio"].(string); ok {
			detail.Ratio = strings.TrimSpace(r)
		}
	}
	fillVideoPerTokenEffectiveRates(detail, info.ChannelId, info.UserId, modelName, mode)
	return detail
}

func videoPerTokenBillingDetailFromTask(task *model.Task, match *videoTokenRuleMatch, spec seedanceVideoSpec, totalTokens int) *videoPerTokenBillingDetail {
	if task == nil {
		return nil
	}
	var req relaycommon.TaskSubmitReq
	_ = common.UnmarshalJsonStr(task.Properties.Input, &req)
	usedUpscaleTarget := applyTaskUpscaleTargetToBillingRequest(task, &req)
	modelName := strings.TrimSpace(taskModelName(task))
	if modelName == "" {
		return nil
	}
	width, height := videoDimensionsFromTaskRequest(req)
	if !usedUpscaleTarget {
		if w, h, ok := videoDimensionsFromTaskCompletion(task, nil); ok {
			width, height = w, h
		}
	}
	hasAudio := taskRequestHasAudio(req)
	mode := relaycommon.DetectVideoBillingMode(&req)
	if bc := task.PrivateData.BillingContext; bc != nil && strings.TrimSpace(bc.VideoBillingMode) != "" {
		mode = bc.VideoBillingMode
	}
	groupRatio := 1.0
	if bc := task.PrivateData.BillingContext; bc != nil && bc.GroupRatio > 0 {
		groupRatio = bc.GroupRatio
	}
	detail := &videoPerTokenBillingDetail{
		Mode:         mode,
		TotalTokens:  totalTokens,
		Width:        width,
		Height:       height,
		HasAudio:     hasAudio,
		Ratio:        spec.Ratio,
		Duration:     spec.Duration,
		GroupRatio:   groupRatio,
		QuotaPerUnit: common.QuotaPerUnit,
	}
	if match != nil {
		detail.Resolution = match.Resolution
		detail.RuleWidth = match.RuleWidth
		detail.RuleHeight = match.RuleHeight
		detail.PricePerMillionTokens = match.ChannelPricePerToken
		detail.UnifiedAudio = match.UnifiedAudio
	} else if strings.TrimSpace(spec.Resolution) != "" {
		detail.Resolution = spec.Resolution
	}
	if display := common.FormatVideoResolutionLabel(strings.TrimSpace(spec.Resolution)); display != "" {
		detail.Resolution = display
	}
	if usedUpscaleTarget {
		if label := common.FormatVideoResolutionLabel(task.PrivateData.VideoUpscale.TargetResolution); label != "" {
			detail.Resolution = label
		}
		detail.ResolutionFromRequest = true
	}
	if bc := task.PrivateData.BillingContext; bc != nil {
		detail.ChannelDiscountPercent = taskBillingContextEffectiveCostPercent(bc, task.ChannelId)
		if detail.PricePerMillionTokens <= 0 {
			detail.PricePerMillionTokens = bc.VideoChannelRulePrice
		}
	}
	if detail.ChannelDiscountPercent < 0 {
		detail.ChannelDiscountPercent = model.ResolveChannelEffectiveCostPercent(task.ChannelId)
	}
	fillVideoPerTokenEffectiveRates(detail, task.ChannelId, task.UserId, modelName, mode)
	if bc := task.PrivateData.BillingContext; bc != nil && bc.TimePricingPlanID > 0 {
		detail.GlobalPricePerMillion = bc.VideoGlobalRulePrice
		if bc.MarkupDiscountPercent != nil {
			detail.MarkupDiscountPercent = *bc.MarkupDiscountPercent
		}
		detail.EffectivePricePerMillion = effectiveVideoPerSecondUSD(
			detail.PricePerMillionTokens,
			detail.GlobalPricePerMillion,
			detail.ChannelDiscountPercent,
			detail.MarkupDiscountPercent,
		)
	}
	return detail
}

func fillVideoPerTokenEffectiveRates(detail *videoPerTokenBillingDetail, channelId, userId int, modelName, mode string) {
	if detail == nil {
		return
	}
	costDisc := detail.ChannelDiscountPercent
	if costDisc < 0 {
		costDisc = model.ResolveChannelEffectiveCostPercent(channelId)
		detail.ChannelDiscountPercent = costDisc
	}
	markupDisc := model.ResolveEffectiveMarkupDiscountPercentForInviteeBilling(userId, channelId, modelName)
	globalPerM := globalVideoPerTokenUSDForChannelTier(
		modelName, mode, detail.Resolution, detail.RuleWidth, detail.RuleHeight, detail.HasAudio, detail.UnifiedAudio,
	)
	detail.GlobalPricePerMillion = globalPerM
	detail.MarkupDiscountPercent = markupDisc
	detail.EffectivePricePerMillion = effectiveVideoPerSecondUSD(detail.PricePerMillionTokens, globalPerM, costDisc, markupDisc)
}

func formatVideoPerTokenBillingDetail(prefix string, detail *videoPerTokenBillingDetail, quota int) string {
	if detail == nil {
		return fmt.Sprintf("%s（按上游 total_tokens ÷ 1M × 对应分辨率 /1M tokens 单价）", prefix)
	}
	priceLabel := "Token价"
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
	pricePerM := detail.EffectivePricePerMillion
	if pricePerM <= 0 {
		pricePerM = detail.PricePerMillionTokens
	}
	tokenPart := fmt.Sprintf("%d tokens", detail.TotalTokens)
	if detail.IsPreCharge && detail.TotalTokens > 0 {
		tokenPart = fmt.Sprintf("预扣 %d tokens", detail.TotalTokens)
	}
	return fmt.Sprintf(
		"%s：%s / 1M × %s(%dx%d，实际 %dx%d，%s) %s $%g/1M tokens(渠道$%g+全局$%g×加价%.0f%%) × QuotaPerUnit %.0f × 分组倍率 %.4g × 渠道折扣 %.4g%% = %d tokens",
		prefix,
		tokenPart,
		resolution,
		detail.RuleWidth,
		detail.RuleHeight,
		detail.Width,
		detail.Height,
		audioLabel(detail.HasAudio),
		priceLabel,
		pricePerM,
		detail.PricePerMillionTokens,
		detail.GlobalPricePerMillion,
		detail.MarkupDiscountPercent,
		detail.QuotaPerUnit,
		detail.GroupRatio,
		videoTokenChannelDiscountPercent(detail),
		quota,
	)
}

func videoTokenChannelDiscountPercent(detail *videoPerTokenBillingDetail) float64 {
	if detail == nil {
		return 100
	}
	if detail.ChannelDiscountPercent < 0 {
		return 0
	}
	return detail.ChannelDiscountPercent
}

func appendVideoPerTokenBillingDetailOther(other map[string]interface{}, detail *videoPerTokenBillingDetail, quota int) {
	if other == nil || detail == nil {
		return
	}
	if detail.TotalTokens > 0 {
		other["video_total_tokens"] = detail.TotalTokens
		other["video_output_tokens"] = detail.TotalTokens
	}
	other["video_width"] = detail.Width
	other["video_height"] = detail.Height
	other["video_has_audio"] = detail.HasAudio
	writeVideoResolutionLogOther(other, detail.Resolution, detail.ResolutionFromRequest, detail.RuleWidth, detail.RuleHeight)
	other["video_rule_width"] = detail.RuleWidth
	other["video_rule_height"] = detail.RuleHeight
	pricePerM := detail.EffectivePricePerMillion
	if pricePerM <= 0 {
		pricePerM = detail.PricePerMillionTokens
	}
	other["video_token_unit_price"] = pricePerM
	other["video_channel_token_price"] = detail.PricePerMillionTokens
	if detail.GlobalPricePerMillion > 0 {
		other["video_global_token_price"] = detail.GlobalPricePerMillion
	}
	if detail.EffectivePricePerMillion > 0 {
		other["effective_video_token_unit_price"] = detail.EffectivePricePerMillion
	}
	if detail.MarkupDiscountPercent > 0 {
		other["markup_discount_rate"] = detail.MarkupDiscountPercent
	}
	other["video_quota_per_unit"] = detail.QuotaPerUnit
	other["channel_price_discount"] = videoTokenChannelDiscountPercent(detail)
	other["video_billed_quota"] = quota
	other["video_unified_audio_price"] = detail.UnifiedAudio
	if detail.CappedToMaxTier {
		other["video_capped_to_max_tier"] = true
	}
	if detail.Ratio != "" {
		other["video_ratio_label"] = detail.Ratio
	}
	if detail.Duration > 0 {
		other["video_duration"] = detail.Duration
	}
	if detail.Mode != "" {
		other["video_billing_lane"] = detail.Mode
	}
}

// videoPerSecondBillingDetailFromTask rebuilds log detail from the saved request
// when the completed video response does not expose probeable metadata.
func videoPerSecondBillingDetailFromTask(task *model.Task) *videoPerSecondBillingDetail {
	if task == nil {
		return nil
	}
	var req relaycommon.TaskSubmitReq
	if err := common.UnmarshalJsonStr(task.Properties.Input, &req); err != nil {
		return nil
	}
	usedUpscaleTarget := applyTaskUpscaleTargetToBillingRequest(task, &req)
	modelName := strings.TrimSpace(taskModelName(task))
	if modelName == "" {
		return nil
	}
	rules, ok := ratio_setting.GetChannelVideoPricingRules(task.ChannelId, modelName)
	if !ok || !ratio_setting.HasUsableVideoPerSecondRules(rules) {
		var globalOK bool
		rules, globalOK = ratio_setting.GetVideoPricingRules(modelName)
		if !globalOK || !ratio_setting.HasUsableVideoPerSecondRules(rules) {
			return nil
		}
	}
	width, height := videoDimensionsFromTaskRequest(req)
	if !usedUpscaleTarget {
		if w, h, ok := videoDimensionsFromTaskCompletion(task, nil); ok {
			width, height = w, h
		}
	}
	hasAudio := taskRequestHasAudio(req)
	if bc := task.PrivateData.BillingContext; bc != nil && bc.OtherRatios != nil {
		if bc.OtherRatios["has_audio"] > 0 {
			hasAudio = true
		}
	}
	mode := detectVideoBillingModeFromTaskReq(&req)
	resolutionLabel := VideoBillingResolutionLabelForTask(task, nil)
	match, ok := matchPerSecondPriceDetail(rules, mode, width, height, hasAudio, resolutionLabel)
	if !ok || match.PricePerSecond <= 0 {
		return nil
	}
	seconds := 0
	if meta, ok := extractVideoMetadataFromTaskData(task); ok && meta.DurationSec > 0 {
		seconds = int(math.Ceil(meta.DurationSec))
	}
	if seconds <= 0 {
		if spec := resolveVideoOutputSpecFromUpstream(task, nil); spec.Duration > 0 {
			seconds = spec.Duration
		}
	}
	if seconds <= 0 {
		seconds = taskBillingSecondsEstimate(task)
	}
	if seconds <= 0 {
		seconds = videoDurationFromTaskRequest(req)
	}
	if seconds <= 0 {
		return nil
	}
	upstreamSpec := resolveVideoOutputSpecFromUpstream(task, nil)
	groupRatio := 1.0
	if bc := task.PrivateData.BillingContext; bc != nil && bc.GroupRatio > 0 {
		groupRatio = bc.GroupRatio
	}
	channelDiscount := taskBillingContextEffectiveCostPercent(task.PrivateData.BillingContext, task.ChannelId)
	detail := &videoPerSecondBillingDetail{
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
		ChannelDiscountPercent: channelDiscount,
		UnifiedAudio:           match.UnifiedAudio,
		CappedToMaxTier:        match.CappedToMaxTier,
		Ratio:                  upstreamSpec.Ratio,
	}
	if display := common.FormatVideoResolutionLabel(resolutionLabel); display != "" {
		detail.Resolution = display
	} else if label := strings.TrimSpace(upstreamSpec.Resolution); label != "" {
		detail.Resolution = common.FormatVideoResolutionLabel(label)
		if detail.Resolution == "" {
			detail.Resolution = label
		}
	}
	if usedUpscaleTarget {
		detail.ResolutionFromRequest = true
	}
	fillVideoPerSecondEffectiveRates(detail, task.ChannelId, task.UserId, modelName, mode)
	return detail
}

// fillVideoPerSecondEffectiveRates 补全日志展示用的全局价、有效单价（含成本折扣与加价切片）。
func fillVideoPerSecondEffectiveRates(detail *videoPerSecondBillingDetail, channelId, userId int, modelName, mode string) {
	if detail == nil {
		return
	}
	costDisc := detail.ChannelDiscountPercent
	if costDisc < 0 {
		costDisc = model.ResolveChannelEffectiveCostPercent(channelId)
		detail.ChannelDiscountPercent = costDisc
	}
	markupDisc := model.ResolveEffectiveMarkupDiscountPercentForInviteeBilling(userId, channelId, modelName)
	globalPerSec := globalVideoPerSecondUSDForChannelTier(
		modelName, mode, detail.Resolution, detail.RuleWidth, detail.RuleHeight, detail.HasAudio, detail.UnifiedAudio,
	)
	detail.GlobalPricePerSecond = globalPerSec
	detail.MarkupDiscountPercent = markupDisc
	detail.EffectivePricePerSecond = effectiveVideoPerSecondUSD(detail.PricePerSecond, globalPerSec, costDisc, markupDisc)
}

func resolveVideoLogChannelDiscountPercent(info *relaycommon.RelayInfo) float64 {
	if info != nil && info.PriceData.ChannelPriceDiscount != nil {
		return *info.PriceData.ChannelPriceDiscount
	}
	if info != nil {
		return model.ResolveChannelEffectiveCostPercent(info.ChannelId)
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
	if w, h, ok := common.ResolveVideoDimensionsFromRequest(
		req.Size, req.Resolution, req.Ratio, req.Metadata,
	); ok {
		return w, h
	}
	return 720, 1280
}

// videoResolutionParamFromRequest 读取用户显式提交的 resolution（顶层或 metadata），不含 size 推断。
func videoResolutionParamFromRequest(req relaycommon.TaskSubmitReq) string {
	if res := strings.TrimSpace(req.Resolution); res != "" {
		return res
	}
	if req.Metadata != nil {
		if v, ok := req.Metadata["resolution"].(string); ok {
			if res := strings.TrimSpace(v); res != "" {
				return res
			}
		}
	}
	return ""
}

// applyPreChargeVideoResolution 预扣日志：有 resolution 参数则原样展示，否则回退定价档位/推断。
func applyPreChargeVideoResolution(req relaycommon.TaskSubmitReq, resolution *string, fromRequest *bool, ruleResolution string) {
	if resolution == nil || fromRequest == nil {
		return
	}
	if userRes := videoResolutionParamFromRequest(req); userRes != "" {
		*resolution = userRes
		*fromRequest = true
		return
	}
	*fromRequest = false
	if label, ok := inferredVideoResolutionLabel(ruleResolution, 0, 0); ok {
		*resolution = label
	}
}

func inferredVideoResolutionLabel(primary string, ruleWidth, ruleHeight int) (string, bool) {
	if label := common.FormatVideoResolutionLabel(primary); label != "" {
		return label, true
	}
	if ruleWidth > 0 && ruleHeight > 0 {
		if label := common.FormatVideoResolutionLabel(fmt.Sprintf("%dx%d", ruleWidth, ruleHeight)); label != "" {
			return label, true
		}
	}
	return "", false
}

func writeVideoResolutionLogOther(other map[string]interface{}, resolution string, fromRequest bool, ruleWidth, ruleHeight int) {
	if other == nil {
		return
	}
	if fromRequest && strings.TrimSpace(resolution) != "" {
		other["video_resolution"] = strings.TrimSpace(resolution)
		other["video_resolution_from_input"] = true
		return
	}
	if label, ok := inferredVideoResolutionLabel(resolution, ruleWidth, ruleHeight); ok {
		other["video_resolution"] = label
	}
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
	pricePerSec := detail.PricePerSecond
	if detail.EffectivePricePerSecond > 0 {
		pricePerSec = detail.EffectivePricePerSecond
	}
	return fmt.Sprintf(
		"%s：%d秒 × %s(%dx%d，实际 %dx%d，%s) %s $%g/秒(渠道$%g+全局$%g×加价%.0f%%) × QuotaPerUnit %.0f × 分组倍率 %.4g × 渠道折扣 %.4g%% = %d tokens",
		prefix,
		detail.Seconds,
		resolution,
		detail.RuleWidth,
		detail.RuleHeight,
		detail.Width,
		detail.Height,
		audioLabel(detail.HasAudio),
		priceLabel,
		pricePerSec,
		detail.PricePerSecond,
		detail.GlobalPricePerSecond,
		detail.MarkupDiscountPercent,
		detail.QuotaPerUnit,
		detail.GroupRatio,
		videoChannelDiscountPercent(detail),
		quota,
	)
}

func videoChannelDiscountPercent(detail *videoPerSecondBillingDetail) float64 {
	if detail == nil {
		return 100
	}
	if detail.ChannelDiscountPercent < 0 {
		return 0
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
	writeVideoResolutionLogOther(other, detail.Resolution, detail.ResolutionFromRequest, detail.RuleWidth, detail.RuleHeight)
	other["video_rule_width"] = detail.RuleWidth
	other["video_rule_height"] = detail.RuleHeight
	other["video_price_per_second"] = detail.PricePerSecond
	if detail.GlobalPricePerSecond > 0 {
		other["global_video_price_per_second"] = detail.GlobalPricePerSecond
	}
	if detail.EffectivePricePerSecond > 0 {
		other["effective_video_price_per_second"] = detail.EffectivePricePerSecond
	}
	if detail.MarkupDiscountPercent > 0 {
		other["markup_discount_rate"] = detail.MarkupDiscountPercent
	}
	other["video_quota_per_unit"] = detail.QuotaPerUnit
	other["channel_price_discount"] = videoChannelDiscountPercent(detail)
	other["video_billed_quota"] = quota
	other["video_unified_audio_price"] = detail.UnifiedAudio
	if detail.CappedToMaxTier {
		other["video_capped_to_max_tier"] = true
	}
	if detail.Ratio != "" {
		other["video_ratio_label"] = detail.Ratio
	}
}

type videoPerVideoBillingDetail struct {
	Mode                   string
	Count                  int
	Width                  int
	Height                 int
	HasAudio               bool
	Resolution             string
	ResolutionFromRequest  bool
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
	applyUpscaleTargetToBillingRequest(c, &req)
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
	detail := &videoPerVideoBillingDetail{
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
	applyPreChargeVideoResolution(req, &detail.Resolution, &detail.ResolutionFromRequest, match.Resolution)
	return detail
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
	}, "text_to_video", width, height, hasAudio, ""); ok {
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
		writeVideoResolutionLogOther(other, detail.Resolution, detail.ResolutionFromRequest, detail.RuleWidth, detail.RuleHeight)
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

func taskUseTimeSeconds(task *model.Task) int {
	if task == nil || task.FinishTime <= 0 {
		return 0
	}
	start := task.SubmitTime
	if start <= 0 {
		start = task.StartTime
	}
	if start <= 0 || task.FinishTime <= start {
		return 0
	}
	return int(task.FinishTime - start)
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
	releaseTaskInvoiceAttribution(task, quota)

	// 2. 退还令牌额度
	taskAdjustTokenQuota(ctx, task, -quota)

	// 3. 记录日志
	other := taskBillingOther(task)
	other["task_id"] = task.TaskID
	other["reason"] = reason
	other = model.SetBillingLogMetadata(other, model.BillingPhaseRefund, true, quota, int64(quota))
	model.RecordTaskBillingLog(model.RecordTaskBillingLogParams{
		UserId:         task.UserId,
		LogType:        model.LogTypeRefund,
		Content:        "",
		ChannelId:      task.ChannelId,
		ModelName:      taskModelName(task),
		TokenName:      task.PrivateData.TokenName,
		Quota:          quota,
		TokenId:        task.PrivateData.TokenId,
		UseTimeSeconds: taskUseTimeSeconds(task),
		Group:          task.Group,
		Other:          other,
	})
}

// recordVideoTaskSettlementMarker 在任务成功且预扣与实扣一致（或无法差额结算）时，
// 写入带 actual_quota 与视频计费 other 的结算日志；content 留空，由前端按 other 渲染（与预扣日志一致）。
func recordVideoTaskSettlementMarker(ctx context.Context, task *model.Task, actualQuota int, detail *videoPerSecondBillingDetail, extraOther ...map[string]interface{}) {
	if task == nil || actualQuota <= 0 || task.Status != model.TaskStatusSuccess {
		return
	}
	if !taskNeedsVideoSettlementMarker(task) {
		return
	}
	preConsumed := task.Quota
	if preConsumed <= 0 {
		preConsumed = actualQuota
	}
	other := taskBillingOther(task)
	other["task_id"] = task.TaskID
	other["actual_quota"] = actualQuota
	other["pre_consumed_quota"] = preConsumed
	other["video_final_quota"] = actualQuota
	if other["billing_mode"] == nil || other["billing_mode"] == "" {
		other["billing_mode"] = "video_per_second"
	}
	for _, extra := range extraOther {
		if extra == nil {
			continue
		}
		for k, v := range extra {
			if k == profitShareExtraTotalTokensKey {
				continue
			}
			other[k] = v
		}
	}
	other = model.SetBillingLogMetadata(other, model.BillingPhaseSettlementMarker, false, actualQuota, 0)
	if detail == nil {
		detail = videoPerSecondBillingDetailFromTask(task)
	}
	if detail != nil {
		other["billing_mode"] = "video_per_second"
		appendVideoPerSecondBillingDetailOther(other, detail, actualQuota)
	}
	model.RecordTaskBillingLog(model.RecordTaskBillingLogParams{
		UserId:         task.UserId,
		LogType:        model.LogTypeConsume,
		Content:        "",
		ChannelId:      task.ChannelId,
		ModelName:      taskModelName(task),
		TokenName:      task.PrivateData.TokenName,
		Quota:          0,
		TokenId:        task.PrivateData.TokenId,
		UseTimeSeconds: taskUseTimeSeconds(task),
		Group:          task.Group,
		Other:          other,
	})
}

func taskNeedsVideoSettlementMarker(task *model.Task) bool {
	bc := task.PrivateData.BillingContext
	if bc == nil {
		return false
	}
	if bc.PerCallBilling {
		return false
	}
	if bc.VideoRuleUnit == VideoRuleUnitPerToken {
		return true
	}
	if bc.ModelPrice == 0 && bc.ModelRatio == 0 {
		return true
	}
	if bc.OtherRatios != nil {
		if _, ok := bc.OtherRatios["seconds"]; ok {
			return true
		}
	}
	if task.PrivateData.VideoUpscale != nil {
		return true
	}
	return false
}

// RecalculateTaskQuota 通用的异步差额结算。
// actualQuota 是任务完成后的实际应扣额度，与预扣额度 (task.Quota) 做差额结算。
// detail 为视频按秒计费明细（写入 other，供前端展示）；结算日志 content 恒为空。
func RecalculateTaskQuota(ctx context.Context, task *model.Task, actualQuota int, detail *videoPerSecondBillingDetail, extraOther ...map[string]interface{}) {
	if task != nil {
		var upsOther map[string]interface{}
		actualQuota, upsOther = appendVideoUpscaleBilling(task, actualQuota)
		if upsOther != nil {
			extraOther = append(extraOther, upsOther)
		}
	}
	if actualQuota <= 0 {
		return
	}
	preConsumedQuota := task.Quota
	quotaDelta := actualQuota - preConsumedQuota
	logReason := "视频按秒结算"
	if detail != nil {
		logReason = formatVideoPerSecondBillingDetail("视频按秒重算", detail, actualQuota)
	}

	if quotaDelta == 0 {
		logger.LogInfo(ctx, fmt.Sprintf("任务 %s 预扣费准确（%s，%s）",
			task.TaskID, logger.LogQuota(actualQuota), logReason))
		recordVideoTaskSettlementMarker(ctx, task, actualQuota, detail, extraOther...)
		return
	}

	logger.LogInfo(ctx, fmt.Sprintf("任务 %s 差额结算：delta=%s（实际：%s，预扣：%s，%s）",
		task.TaskID,
		logger.LogQuota(quotaDelta),
		logger.LogQuota(actualQuota),
		logger.LogQuota(preConsumedQuota),
		logReason,
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
		releaseTaskInvoiceAttribution(task, logQuota)
	}
	other := taskBillingOther(task)
	other["task_id"] = task.TaskID
	//other["reason"] = reason
	other["pre_consumed_quota"] = preConsumedQuota
	other["actual_quota"] = actualQuota
	other["video_final_quota"] = actualQuota
	for _, extra := range extraOther {
		if extra == nil {
			continue
		}
		for k, v := range extra {
			if k == profitShareExtraTotalTokensKey {
				continue
			}
			other[k] = v
		}
	}
	if detail == nil && taskNeedsVideoSettlementMarker(task) {
		detail = videoPerSecondBillingDetailFromTask(task)
	}
	if detail != nil {
		other["billing_mode"] = "video_per_second"
		appendVideoPerSecondBillingDetailOther(other, detail, actualQuota)
	}
	// 明细写入后再次对齐：保证实际扣费含超分附加费。
	other["actual_quota"] = actualQuota
	other["video_final_quota"] = actualQuota
	other["video_billed_quota"] = actualQuota
	phase := model.BillingPhaseDeltaCharge
	balanceDelta := -int64(logQuota)
	if logType == model.LogTypeRefund {
		phase = model.BillingPhaseDeltaRefund
		balanceDelta = int64(logQuota)
	}
	other = model.SetBillingLogMetadata(other, phase, true, logQuota, balanceDelta)
	model.RecordTaskBillingLog(model.RecordTaskBillingLogParams{
		UserId:         task.UserId,
		LogType:        logType,
		Content:        "",
		ChannelId:      task.ChannelId,
		ModelName:      taskModelName(task),
		TokenName:      task.PrivateData.TokenName,
		Quota:          logQuota,
		TokenId:        task.PrivateData.TokenId,
		UseTimeSeconds: taskUseTimeSeconds(task),
		Group:          task.Group,
		Other:          other,
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

	costDisc := taskBillingContextEffectiveCostPercent(task.PrivateData.BillingContext, task.ChannelId)
	markupDisc := model.ResolveEffectiveMarkupDiscountPercentForInviteeBilling(task.UserId, task.ChannelId, modelName)
	globalMr, globalOK, _ := ratio_setting.GetModelRatio(modelName)
	if !globalOK {
		globalMr = 0
	}
	effRate := model.EffectiveInputRate(modelRatio, globalMr, costDisc, markupDisc)
	actualQuota := int(math.Round(float64(totalTokens) * effRate * finalGroupRatio))

	RecalculateTaskQuota(ctx, task, actualQuota, nil, map[string]interface{}{
		profitShareExtraTotalTokensKey: totalTokens,
	})
	return true
}
