package service

// ============================================================================
// Seedance 视频「按 Token 计费」模块
//
// 触发条件：视频价格配置中计费模式为「按 token 收费」，且存在与当前请求
// （文生/图生/视频生视频 + 分辨率）匹配的分辨率 token 单价规则。
// 不再依赖模型基础价格里的输出倍率/输出价格（completionRatio）。
//
// 业务流程：
//   · 任务发起：预扣固定 50000 tokens，按匹配分辨率单价（美元/1M tokens）折算额度；
//   · 任务完成：按上游 usage.total_tokens ÷ 1M × 同一单价补差，并更新日志。
// ============================================================================

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

// SeedanceTokenPreConsumeTokens 任务发起阶段预扣的固定 Token 数（不估算真实 token）。
const SeedanceTokenPreConsumeTokens = 50000

// SeedanceVideoTokenBillingMode 日志 billing_mode，前端按 token 单价展示。
const SeedanceVideoTokenBillingMode = "video_token_output"

// VideoRuleUnitPerToken 视频规则计费单位：按 token。
const VideoRuleUnitPerToken = "per_token"

// VideoTokenPricePerMillion 视频按 token 规则中 price 字段的计量单位（1M tokens）。
const VideoTokenPricePerMillion = 1_000_000.0

// seedanceVideoSpec 日志展示用视频规格（分辨率标识 + 画面比例，禁止像素尺寸）。
type seedanceVideoSpec struct {
	Resolution string
	Ratio      string
	Duration   int
}

// videoTokenRuleMatch 一次请求匹配到的 token 单价规则快照。
type videoTokenRuleMatch struct {
	Mode                   string
	Resolution             string
	RuleWidth              int
	RuleHeight             int
	HasAudio               bool
	UnifiedAudio           bool
	ChannelPricePerToken   float64
	GlobalPricePerToken    float64
	EffectivePricePerToken float64
}

// resolveVideoPerTokenPricingRules 读取渠道/全局视频按 token 规则表。
func resolveVideoPerTokenPricingRules(channelID int, modelName string) (ratio_setting.VideoPricingRules, bool) {
	if rules, ok := ratio_setting.GetChannelVideoPricingRules(channelID, modelName); ok && ratio_setting.HasUsableVideoPerTokenRules(rules) {
		return rules, true
	}
	if rules, ok := ratio_setting.GetVideoPricingRules(modelName); ok && ratio_setting.HasUsableVideoPerTokenRules(rules) {
		return rules, true
	}
	return ratio_setting.VideoPricingRules{}, false
}

// perTokenRowsForMode 按文生/图生/视频生视频取对应 token 规则行。
func perTokenRowsForMode(rules ratio_setting.VideoPricingRules, mode string) []ratio_setting.VideoResolutionAudioPriceRule {
	switch strings.TrimSpace(mode) {
	case relaycommon.VideoBillingModeImageToVideo:
		return rules.ImageToVideoPerToken
	case relaycommon.VideoBillingModeVideoToVideo:
		return rules.VideoToVideoPerToken
	default:
		return rules.TextToVideoPerToken
	}
}

// matchPerTokenPriceDetail 按分辨率匹配 token 单价（resolution 标识优先，再回退像素档位）。
func matchPerTokenPriceDetail(rules ratio_setting.VideoPricingRules, mode string, width, height int, hasAudio bool, resolutionLabel string) (*videoPerSecondPriceMatch, bool) {
	rows := perTokenRowsForMode(rules, mode)
	if len(rows) == 0 {
		return nil, false
	}
	wrapped := ratio_setting.VideoPricingRules{TextToVideoPerSecond: rows}
	return matchPerSecondPriceDetail(wrapped, relaycommon.VideoBillingModeTextToVideo, width, height, hasAudio, resolutionLabel)
}

func matchPerTokenPrice(rules ratio_setting.VideoPricingRules, mode string, width, height int, hasAudio bool) (float64, bool) {
	match, ok := matchPerTokenPriceDetail(rules, mode, width, height, hasAudio, "")
	if !ok {
		return 0, false
	}
	return match.PricePerSecond, true
}

// MatchVideoTokenRuleForRequest 根据请求参数（分辨率 + 素材类型）匹配 token 单价规则。
func MatchVideoTokenRuleForRequest(channelID, userID int, modelName, mode string, width, height int, hasAudio bool, resolutionLabel string) (*videoTokenRuleMatch, bool) {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" || width <= 0 || height <= 0 {
		return nil, false
	}
	rules, ok := resolveVideoPerTokenPricingRules(channelID, modelName)
	if !ok {
		return nil, false
	}
	match, ok := matchPerTokenPriceDetail(rules, mode, width, height, hasAudio, resolutionLabel)
	if !ok || match.PricePerSecond <= 0 {
		return nil, false
	}
	costDisc := model.ResolveChannelEffectiveCostPercent(channelID)
	markupDisc := model.ResolveEffectiveMarkupDiscountPercentForInviteeBilling(userID, channelID, modelName)
	globalPerToken := globalVideoPerTokenUSD(modelName, mode, match.RuleWidth, match.RuleHeight, hasAudio)
	effPerToken := effectiveVideoPerSecondUSD(match.PricePerSecond, globalPerToken, costDisc, markupDisc)
	if effPerToken <= 0 {
		return nil, false
	}
	return &videoTokenRuleMatch{
		Mode:                   mode,
		Resolution:             match.Resolution,
		RuleWidth:              match.RuleWidth,
		RuleHeight:             match.RuleHeight,
		HasAudio:               hasAudio,
		UnifiedAudio:           match.UnifiedAudio,
		ChannelPricePerToken:   match.PricePerSecond,
		GlobalPricePerToken:    globalPerToken,
		EffectivePricePerToken: effPerToken,
	}, true
}

// ShouldUseVideoTokenBilling 是否对该模型启用视频按 token 规则计费（存在可用 per_token 表）。
func ShouldUseVideoTokenBilling(channelID int, modelName string) bool {
	_, ok := resolveVideoPerTokenPricingRules(channelID, modelName)
	return ok
}

// CalcVideoTokenQuota 按 token 单价折算 quota：tokens / 1M × 美元/1M tokens × QuotaPerUnit × 分组倍率。
func CalcVideoTokenQuota(channelID, userID int, modelName, mode string, width, height int, hasAudio bool, totalTokens int, groupRatio float64, resolutionLabel string) (int, *videoTokenRuleMatch, bool) {
	if totalTokens <= 0 {
		return 0, nil, false
	}
	match, ok := MatchVideoTokenRuleForRequest(channelID, userID, modelName, mode, width, height, hasAudio, resolutionLabel)
	if !ok {
		return 0, nil, false
	}
	if groupRatio <= 0 {
		groupRatio = 1
	}
	rawQuota := (float64(totalTokens) / VideoTokenPricePerMillion) * match.EffectivePricePerToken * common.QuotaPerUnit * groupRatio
	quota := int(math.Round(rawQuota))
	if quota <= 0 && rawQuota > 0 {
		quota = 1
	}
	return quota, match, true
}

// SettleSeedanceVideoTokenBillingOnComplete 任务成功后的 token 差额结算。
func SettleSeedanceVideoTokenBillingOnComplete(ctx context.Context, task *model.Task, taskResult *relaycommon.TaskInfo) bool {
	if task == nil || taskResult == nil || task.Status != model.TaskStatusSuccess {
		return false
	}
	totalTokens := taskResult.TotalTokens
	if totalTokens <= 0 {
		return false
	}
	// 仅处理按 token 规则提交的任务（避免误伤其它 token 结算分支）。
	if bc := task.PrivateData.BillingContext; bc == nil || bc.VideoRuleUnit != VideoRuleUnitPerToken {
		return false
	}

	modelName := taskModelName(task)
	mode, width, height, hasAudio := videoTokenBillingParamsFromTask(task)
	if w, h, ok := videoDimensionsFromTaskCompletion(task, taskResult); ok {
		width, height = w, h
	}
	resolutionLabel := VideoBillingResolutionLabelForTask(task, taskResult)
	groupRatio := 1.0
	if bc := task.PrivateData.BillingContext; bc != nil && bc.GroupRatio > 0 {
		groupRatio = bc.GroupRatio
	}

	actualQuota, match, ok := CalcVideoTokenQuota(task.ChannelId, task.UserId, modelName, mode, width, height, hasAudio, totalTokens, groupRatio, resolutionLabel)
	if !ok || actualQuota <= 0 {
		// 回退：用提交时快照单价结算，避免规则变更导致无法结算。
		actualQuota = calcVideoTokenQuotaFromBillingContext(task, totalTokens)
		if actualQuota <= 0 {
			return false
		}
	}

	spec := resolveSeedanceVideoSpec(task, taskResult)
	extra := buildSeedanceVideoTokenLogOther(totalTokens, match, task, spec, actualQuota)
	settleSeedanceVideoTokenDelta(ctx, task, actualQuota, extra)
	return true
}

func calcVideoTokenQuotaFromBillingContext(task *model.Task, totalTokens int) int {
	bc := task.PrivateData.BillingContext
	if bc == nil || bc.VideoRuleUnit != VideoRuleUnitPerToken {
		return 0
	}
	groupRatio := bc.GroupRatio
	if groupRatio <= 0 {
		groupRatio = 1
	}
	markupDisc := float64(0)
	if bc.MarkupDiscountPercent != nil {
		markupDisc = *bc.MarkupDiscountPercent
	}
	effPerToken := effectiveVideoPerSecondUSD(
		bc.VideoChannelRulePrice,
		bc.VideoGlobalRulePrice,
		taskBillingContextEffectiveCostPercent(bc, task.ChannelId),
		markupDisc,
	)
	if effPerToken <= 0 {
		return 0
	}
	raw := (float64(totalTokens) / VideoTokenPricePerMillion) * effPerToken * common.QuotaPerUnit * groupRatio
	quota := int(math.Round(raw))
	if quota <= 0 && raw > 0 {
		return 1
	}
	return quota
}

func videoTokenBillingParamsFromTask(task *model.Task) (mode string, width, height int, hasAudio bool) {
	mode = relaycommon.VideoBillingModeTextToVideo
	var req relaycommon.TaskSubmitReq
	if err := common.UnmarshalJsonStr(task.Properties.Input, &req); err == nil {
		mode = relaycommon.DetectVideoBillingMode(&req)
		width, height = videoDimensionsFromTaskRequest(req)
		hasAudio = taskRequestHasAudio(req)
	}
	if bc := task.PrivateData.BillingContext; bc != nil && strings.TrimSpace(bc.VideoBillingMode) != "" {
		mode = bc.VideoBillingMode
	}
	if width <= 0 || height <= 0 {
		width, height = 720, 1280
	}
	return mode, width, height, hasAudio
}

func settleSeedanceVideoTokenDelta(ctx context.Context, task *model.Task, actualQuota int, extra map[string]interface{}) {
	preConsumed := task.Quota
	delta := actualQuota - preConsumed

	// 结算日志的记账口径与按秒计费 RecalculateTaskQuota 一致：
	//   · delta == 0：资金未变动，只写 quota=0 的展示标记（settlement_marker）；
	//   · delta != 0：资金已变动，必须把差额写进日志 quota，否则按 quota 求和的
	//     消费统计（/api/log/stat）与对账单会漏掉这笔差额。
	logType := model.LogTypeConsume
	logQuota := 0
	logPhase := model.BillingPhaseSettlementMarker
	affectsBalance := false
	displayQuota := actualQuota
	balanceDelta := int64(0)

	if delta != 0 {
		if err := taskAdjustFunding(task, delta); err != nil {
			logger.LogError(ctx, fmt.Sprintf("Seedance Token 计费差额调整资金失败 task %s: %s", task.TaskID, err.Error()))
			return
		}
		taskAdjustTokenQuota(ctx, task, delta)
		task.Quota = actualQuota
		if task.ID > 0 {
			if err := model.DB.Model(&model.Task{}).Where("id = ?", task.ID).Update("quota", actualQuota).Error; err != nil {
				logger.LogWarn(ctx, fmt.Sprintf("更新任务实际计费额度失败 task %s: %s", task.TaskID, err.Error()))
			}
		}
		affectsBalance = true
		if delta > 0 {
			logQuota = delta
			logPhase = model.BillingPhaseDeltaCharge
			balanceDelta = -int64(delta)
			model.UpdateUserUsedQuotaAndRequestCount(task.UserId, delta)
			model.UpdateChannelUsedQuota(task.ChannelId, delta)
		} else {
			// 预扣大于实扣：差额退回钱包。用户与渠道「累积已用」的回退由
			// RecordTaskBillingLog 按 refund 日志统一处理，此处不可重复扣减。
			logType = model.LogTypeRefund
			logQuota = -delta
			logPhase = model.BillingPhaseDeltaRefund
			balanceDelta = int64(logQuota)
		}
		displayQuota = logQuota
		logger.LogInfo(ctx, fmt.Sprintf("任务 %s Token 差额结算：delta=%s（实际：%s，预扣：%s）",
			task.TaskID, logger.LogQuota(delta), logger.LogQuota(actualQuota), logger.LogQuota(preConsumed)))
	} else {
		logger.LogInfo(ctx, fmt.Sprintf("任务 %s Token 预扣准确（%s）", task.TaskID, logger.LogQuota(actualQuota)))
	}

	other := taskBillingOther(task)
	other["task_id"] = task.TaskID
	other["pre_consumed_quota"] = preConsumed
	other["actual_quota"] = actualQuota
	other["video_final_quota"] = actualQuota
	for k, v := range extra {
		other[k] = v
	}

	other = model.SetBillingLogMetadata(other, logPhase, affectsBalance, displayQuota, balanceDelta)

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

func buildSeedanceVideoTokenLogOther(totalTokens int, match *videoTokenRuleMatch, task *model.Task, spec seedanceVideoSpec, billedQuota int) map[string]interface{} {
	other := map[string]interface{}{
		"billing_mode": SeedanceVideoTokenBillingMode,
	}
	detail := videoPerTokenBillingDetailFromTask(task, match, spec, totalTokens)
	if detail != nil {
		appendVideoPerTokenBillingDetailOther(other, detail, billedQuota)
		return other
	}
	// 回退：规则匹配失败时仍写入基础 token 单价字段。
	effPerToken := 0.0
	channelPerToken := 0.0
	if match != nil {
		effPerToken = match.EffectivePricePerToken
		channelPerToken = match.ChannelPricePerToken
	} else if bc := task.PrivateData.BillingContext; bc != nil {
		channelPerToken = bc.VideoChannelRulePrice
		markupDisc := float64(0)
		if bc.MarkupDiscountPercent != nil {
			markupDisc = *bc.MarkupDiscountPercent
		}
		effPerToken = effectiveVideoPerSecondUSD(channelPerToken, bc.VideoGlobalRulePrice, taskBillingContextEffectiveCostPercent(bc, task.ChannelId), markupDisc)
	}
	other["video_total_tokens"] = totalTokens
	other["video_token_unit_price"] = effPerToken
	other["video_channel_token_price"] = channelPerToken
	other["video_quota_per_unit"] = common.QuotaPerUnit
	other["video_billed_quota"] = billedQuota
	if bc := task.PrivateData.BillingContext; bc != nil && bc.GroupRatio > 0 {
		other["group_ratio"] = bc.GroupRatio
	}
	if strings.TrimSpace(spec.Resolution) != "" {
		other["video_resolution"] = spec.Resolution
	}
	if strings.TrimSpace(spec.Ratio) != "" {
		other["video_ratio_label"] = spec.Ratio
	}
	if spec.Duration > 0 {
		other["video_duration"] = spec.Duration
	}
	return other
}

func resolveSeedanceVideoSpec(task *model.Task, taskResult *relaycommon.TaskInfo) seedanceVideoSpec {
	spec := resolveVideoOutputSpecFromUpstream(task, taskResult)
	if spec.Resolution != "" && spec.Ratio != "" && spec.Duration > 0 {
		return spec
	}
	if spec.Resolution == "" || spec.Ratio == "" || spec.Duration <= 0 {
		var req relaycommon.TaskSubmitReq
		_ = common.UnmarshalJsonStr(task.Properties.Input, &req)
		width, height := videoDimensionsFromTaskRequest(req)
		if spec.Resolution == "" {
			if label := common.FormatVideoResolutionLabel(strings.TrimSpace(req.Size)); label != "" {
				spec.Resolution = label
			} else if width > 0 && height > 0 {
				spec.Resolution = common.FormatVideoResolutionLabel(fmt.Sprintf("%dx%d", width, height))
			}
		}
		if spec.Ratio == "" {
			if r := submitMetadataString(req.Metadata, "ratio"); r != "" {
				spec.Ratio = r
			} else if width > 0 && height > 0 {
				spec.Ratio = deriveVideoAspectRatioLabel(width, height)
			}
		}
		if spec.Duration <= 0 {
			spec.Duration = videoDurationFromTaskRequest(req)
		}
	}
	return spec
}

func submitMetadataString(metadata map[string]interface{}, key string) string {
	if metadata == nil {
		return ""
	}
	if v, ok := metadata[key].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func deriveVideoAspectRatioLabel(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	ratio := float64(width) / float64(height)
	candidates := []struct{ w, h int }{
		{16, 9}, {9, 16}, {4, 3}, {3, 4}, {1, 1}, {21, 9},
	}
	best := ""
	bestDiff := math.MaxFloat64
	for _, c := range candidates {
		diff := math.Abs(ratio - float64(c.w)/float64(c.h))
		if diff < bestDiff {
			bestDiff = diff
			best = fmt.Sprintf("%d:%d", c.w, c.h)
		}
	}
	if bestDiff <= 0.02 {
		return best
	}
	g := gcdInt(width, height)
	if g <= 0 {
		return ""
	}
	return fmt.Sprintf("%d:%d", width/g, height/g)
}

func gcdInt(a, b int) int {
	if a < 0 {
		a = -a
	}
	if b < 0 {
		b = -b
	}
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// SettleVideoTokenBillingOnComplete task_polling 调用入口。
func SettleVideoTokenBillingOnComplete(ctx context.Context, task *model.Task, taskResult *relaycommon.TaskInfo) bool {
	return SettleSeedanceVideoTokenBillingOnComplete(ctx, task, taskResult)
}
