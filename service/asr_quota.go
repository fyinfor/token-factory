package service

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	perfmetrics "github.com/QuantumNous/new-api/pkg/perf_metrics"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/gin-gonic/gin"
)

// ComputeASRQuota 按秒计算 ASR 应扣额度（无副作用，仅供调用方预写任务记录）。
func ComputeASRQuota(ctx *gin.Context, relayInfo *relaycommon.RelayInfo, seconds float64) int {
	if seconds <= 0 {
		seconds = 1
	}
	relayInfo.PriceData.AddOtherRatio("seconds", seconds)
	usage := &dto.Usage{
		PromptTokens: int(math.Ceil(seconds)),
		TotalTokens:  int(math.Ceil(seconds)),
	}
	return calculateTextQuotaSummary(ctx, relayInfo, usage).Quota
}

// LogASRAsyncPreConsume 异步任务提交成功后写入预扣消费日志（billing_phase=pre_charge）。
// 实际扣费已由 PreConsumeBilling 完成；此处仅记日志并累计 used_quota / 渠道用量。
func LogASRAsyncPreConsume(ctx *gin.Context, relayInfo *relaycommon.RelayInfo, taskID string, preConsumed int, preSeconds float64) {
	if relayInfo == nil || preConsumed <= 0 {
		return
	}
	if preSeconds <= 0 {
		preSeconds = 60
	}
	chID := 0
	if relayInfo.ChannelMeta != nil {
		chID = relayInfo.ChannelId
	}
	tokenName := ""
	if ctx != nil {
		tokenName = ctx.GetString("token_name")
	}

	effUnitUSD := model.EffectiveModelPrice(
		relayInfo.PriceData.ModelPrice,
		relayInfo.PriceData.GlobalModelPrice,
		relayInfo.PriceData.CostDiscountPercent,
		relayInfo.PriceData.MarkupDiscountPercent,
	)
	groupRatio := relayInfo.PriceData.GroupRatioInfo.GroupRatio
	if groupRatio <= 0 {
		groupRatio = 1
	}
	userUnitUSD := effUnitUSD * groupRatio
	logContent := fmt.Sprintf("ASR 异步转写预扣 %s 秒，每秒价格 %s，预扣 %s",
		formatASRSecondsDisplay(preSeconds), formatASRUnitPriceDisplay(userUnitUSD), logger.FormatQuota(preConsumed))

	other := GenerateTextOtherInfo(ctx, relayInfo, 0, groupRatio, 0,
		0, 0, relayInfo.PriceData.ModelPrice, relayInfo.PriceData.GroupRatioInfo.GroupSpecialRatio)
	other["asr"] = true
	other["audio_seconds"] = preSeconds
	other["asr_unit_price"] = userUnitUSD
	other["use_price"] = true
	other["pre_consumed_quota"] = preConsumed
	if taskID != "" {
		other["task_id"] = taskID
	}
	other = model.SetBillingLogMetadata(other, model.BillingPhasePreCharge, true, preConsumed, -int64(preConsumed))

	model.RecordConsumeLog(ctx, relayInfo.UserId, model.RecordConsumeLogParams{
		ChannelId:      chID,
		PromptTokens:   int(math.Ceil(preSeconds)),
		ModelName:      relayInfo.OriginModelName,
		TokenName:      tokenName,
		Quota:          preConsumed,
		TokenUsed:      preConsumed,
		Content:        logContent,
		TokenId:        relayInfo.TokenId,
		UseTimeSeconds: 0,
		IsStream:       false,
		Group:          relayInfo.UsingGroup,
		Other:          other,
	})
	recordWalletUsedQuota(relayInfo, relayInfo.UserId, preConsumed)
	model.UpdateChannelUsedQuota(chID, preConsumed)
}

// PostASRConsumeQuota 阿里云 ASR 语音转写结算、日志与分润。
//
// 三条链路：
//  1. 同步转写：Relay 主流程已建立 Billing 会话（可能预扣），此处按真实秒数 SettleBilling 差额；
//  2. 异步转写（已写预扣日志）：写结算标记（合并进预扣日志展示实际扣费总额），余额仅按差额调整；
//  3. 异步转写（兼容旧任务无预扣日志）：按实际秒数差额结算并写一条全额消费日志。
//
// 分润沿用钱包分润链路（tryPostWalletProfitShareCredit），按按秒消耗金额核算代理收益。
// taskID 非空时写入日志 other（异步任务日志详情展示）。
// prechargeLogged=true 表示提交阶段已调用 LogASRAsyncPreConsume（used_quota 已计入预扣）。
func PostASRConsumeQuota(ctx *gin.Context, relayInfo *relaycommon.RelayInfo, seconds float64, taskID string, extraContent string) *types.TokenFactoryError {
	return postASRConsumeQuota(ctx, relayInfo, seconds, taskID, extraContent, false)
}

// PostASRConsumeQuotaAsync 异步任务成功结算：在已写预扣日志的前提下按真实时长差额结算并写结算日志。
func PostASRConsumeQuotaAsync(ctx *gin.Context, relayInfo *relaycommon.RelayInfo, seconds float64, taskID string, extraContent string, prechargeLogged bool) *types.TokenFactoryError {
	return postASRConsumeQuota(ctx, relayInfo, seconds, taskID, extraContent, prechargeLogged)
}

func postASRConsumeQuota(ctx *gin.Context, relayInfo *relaycommon.RelayInfo, seconds float64, taskID string, extraContent string, prechargeLogged bool) *types.TokenFactoryError {
	if seconds <= 0 {
		seconds = 1
	}
	relayInfo.PriceData.AddOtherRatio("seconds", seconds)
	usage := &dto.Usage{
		PromptTokens: int(math.Ceil(seconds)),
		TotalTokens:  int(math.Ceil(seconds)),
	}
	summary := calculateTextQuotaSummary(ctx, relayInfo, usage)

	chID := 0
	if relayInfo.ChannelMeta != nil {
		chID = relayInfo.ChannelId
	}

	preConsumed := relayInfo.FinalPreConsumedQuota
	if relayInfo.Billing != nil {
		preConsumed = relayInfo.Billing.GetPreConsumedQuota()
	}

	switch {
	case relayInfo.Billing != nil:
		// 同步链路：有计费会话，按实际额度结算差额
		if err := SettleBilling(ctx, relayInfo, summary.Quota); err != nil {
			logger.LogError(ctx, "error settling billing: "+err.Error())
		} else {
			tryPostWalletProfitShareCredit(ctx, relayInfo, usage, &summary, "asr")
		}
	case preConsumed > 0:
		// 异步链路：提交时已预扣，此处仅补差价 / 退差额
		delta := summary.Quota - preConsumed
		if delta != 0 {
			if err := PostConsumeQuota(relayInfo, delta, preConsumed, true); err != nil {
				logger.LogError(ctx, "error settling asr async delta: "+err.Error())
				return types.NewError(err, types.ErrorCodeUpdateDataError)
			}
		}
		tryPostWalletProfitShareCredit(ctx, relayInfo, usage, &summary, "asr")
	default:
		// 兼容旧异步任务（提交未预扣）：全额新建会话扣费
		if summary.Quota > 0 {
			if tfErr := PreConsumeBilling(ctx, summary.Quota, relayInfo); tfErr != nil {
				return tfErr
			}
		}
		if err := SettleBilling(ctx, relayInfo, summary.Quota); err != nil {
			logger.LogError(ctx, "error settling billing: "+err.Error())
		} else {
			tryPostWalletProfitShareCredit(ctx, relayInfo, usage, &summary, "asr")
		}
	}

	useTimeSeconds := 0
	if !relayInfo.StartTime.IsZero() {
		useTimeSeconds = int(time.Now().Unix() - relayInfo.StartTime.Unix())
	}
	effUnitUSD := model.EffectiveModelPrice(
		relayInfo.PriceData.ModelPrice,
		relayInfo.PriceData.GlobalModelPrice,
		relayInfo.PriceData.CostDiscountPercent,
		relayInfo.PriceData.MarkupDiscountPercent,
	)
	userUnitUSD := effUnitUSD * summary.GroupRatio

	// 已写预扣日志的异步任务：只记差额/结算标记，避免 UI 出现双倍消费
	if prechargeLogged && taskID != "" && relayInfo.Billing == nil && preConsumed > 0 {
		recordASRAsyncSettlementLog(ctx, relayInfo, chID, summary, seconds, userUnitUSD, preConsumed, taskID, useTimeSeconds, extraContent)
		if !relayInfo.IsChannelTest {
			gopool.Go(func() {
				perfmetrics.RecordRelaySample(relayInfo, true, 0, int64(summary.PromptTokens), 0)
			})
		}
		return nil
	}

	if summary.TotalTokens == 0 {
		logger.LogError(ctx, fmt.Sprintf("asr total tokens is 0, cannot consume quota, userId %d, channelId %d, tokenId %d, model %s",
			relayInfo.UserId, chID, relayInfo.TokenId, summary.ModelName))
	} else {
		model.UpdateUserUsedQuotaAndRequestCountWithGiftOffset(relayInfo.UserId, summary.Quota, walletGiftOffset(relayInfo))
		model.UpdateChannelUsedQuota(chID, summary.Quota)
	}

	logContent := fmt.Sprintf("ASR 语音转写时长 %s 秒，每秒价格 %s",
		formatASRSecondsDisplay(seconds), formatASRUnitPriceDisplay(userUnitUSD))
	if preConsumed > 0 && relayInfo.Billing == nil {
		logContent += fmt.Sprintf("，预扣 %s，实际 %s", logger.FormatQuota(preConsumed), logger.FormatQuota(summary.Quota))
	}
	if extraContent != "" {
		logContent += ", " + extraContent
	}

	other := GenerateTextOtherInfo(ctx, relayInfo, summary.ModelRatio, summary.GroupRatio, summary.CompletionRatio,
		0, 0, summary.ModelPrice, relayInfo.PriceData.GroupRatioInfo.GroupSpecialRatio)
	other["asr"] = true
	other["audio_seconds"] = seconds
	other["asr_unit_price"] = userUnitUSD
	if taskID != "" {
		other["task_id"] = taskID
	}
	other["use_price"] = relayInfo.PriceData.UsePrice
	if preConsumed > 0 {
		other["pre_consumed_quota"] = preConsumed
		other["actual_quota"] = summary.Quota
	}

	model.RecordConsumeLog(ctx, relayInfo.UserId, model.RecordConsumeLogParams{
		ChannelId:        chID,
		PromptTokens:     summary.PromptTokens,
		CompletionTokens: 0,
		ModelName:        summary.ModelName,
		TokenName:        summary.TokenName,
		Quota:            summary.Quota,
		Content:          logContent,
		TokenId:          relayInfo.TokenId,
		UseTimeSeconds:   useTimeSeconds,
		IsStream:         false,
		Group:            relayInfo.UsingGroup,
		Other:            other,
	})
	if !relayInfo.IsChannelTest {
		gopool.Go(func() {
			perfmetrics.RecordRelaySample(relayInfo, true, 0, int64(summary.PromptTokens), 0)
		})
	}
	return nil
}

func recordASRAsyncSettlementLog(ctx *gin.Context, relayInfo *relaycommon.RelayInfo, chID int, summary textQuotaSummary, seconds float64, userUnitUSD float64, preConsumed int, taskID string, useTimeSeconds int, extraContent string) {
	actualQuota := summary.Quota
	delta := actualQuota - preConsumed
	tokenName := summary.TokenName
	if tokenName == "" && ctx != nil {
		tokenName = ctx.GetString("token_name")
	}

	// 余额差额已在 postASRConsumeQuota 中通过 PostConsumeQuota 调整；
	// 此处仅同步 used_quota / 渠道用量，并写入结算标记供列表合并进预扣日志。
	if delta > 0 {
		model.UpdateUserUsedQuotaAndRequestCountWithGiftOffset(relayInfo.UserId, delta, walletGiftOffset(relayInfo))
		model.UpdateChannelUsedQuota(chID, delta)
	} else if delta < 0 {
		refundQuota := -delta
		model.DecreaseUserUsedQuota(relayInfo.UserId, refundQuota)
		model.UpdateChannelUsedQuota(chID, -refundQuota)
	}

	other := GenerateTextOtherInfo(ctx, relayInfo, summary.ModelRatio, summary.GroupRatio, summary.CompletionRatio,
		0, 0, summary.ModelPrice, relayInfo.PriceData.GroupRatioInfo.GroupSpecialRatio)
	other["asr"] = true
	other["audio_seconds"] = seconds
	other["asr_unit_price"] = userUnitUSD
	other["use_price"] = true
	other["task_id"] = taskID
	other["pre_consumed_quota"] = preConsumed
	other["actual_quota"] = actualQuota

	baseContent := fmt.Sprintf("ASR 语音转写时长 %s 秒，每秒价格 %s，实际扣费 %s（预扣 %s）",
		formatASRSecondsDisplay(seconds), formatASRUnitPriceDisplay(userUnitUSD),
		logger.FormatQuota(actualQuota), logger.FormatQuota(preConsumed))
	if delta > 0 {
		baseContent += fmt.Sprintf("，补扣 %s", logger.FormatQuota(delta))
	} else if delta < 0 {
		baseContent += fmt.Sprintf("，退还差额 %s", logger.FormatQuota(-delta))
	}
	if extraContent != "" {
		baseContent += ", " + extraContent
	}

	// 仅写结算标记（Quota=0, affects_balance=false）：列表默认隐藏，
	// 由 mergeSettlementMarkersIntoPreChargeLogs 合并进预扣日志并展示实际扣费总额。
	other = model.SetBillingLogMetadata(other, model.BillingPhaseSettlementMarker, false, actualQuota, 0)
	model.RecordTaskBillingLog(model.RecordTaskBillingLogParams{
		UserId:         relayInfo.UserId,
		LogType:        model.LogTypeConsume,
		Content:        baseContent,
		ChannelId:      chID,
		ModelName:      summary.ModelName,
		TokenName:      tokenName,
		Quota:          0,
		TokenId:        relayInfo.TokenId,
		UseTimeSeconds: useTimeSeconds,
		Group:          relayInfo.UsingGroup,
		Other:          other,
	})
}

// RefundASRPreConsumedQuota 异步任务失败时退还提交阶段预扣额度，并写入退款使用日志。
// prechargeLogged=true 时同步回退 used_quota（与预扣日志配套）；旧任务无预扣日志则只退余额。
func RefundASRPreConsumedQuota(ctx *gin.Context, relayInfo *relaycommon.RelayInfo, preConsumed int, taskID, reason string, prechargeLogged bool) {
	if relayInfo == nil || preConsumed <= 0 {
		return
	}
	if err := PostConsumeQuota(relayInfo, -preConsumed, preConsumed, true); err != nil {
		logger.LogError(ctx, fmt.Sprintf("asr async refund failed task=%s: %s", taskID, err.Error()))
		return
	}
	logger.LogInfo(ctx, fmt.Sprintf("ASR 异步任务失败已退还预扣 %s（task_id=%s）: %s",
		logger.FormatQuota(preConsumed), taskID, reason))

	chID := 0
	if relayInfo.ChannelMeta != nil {
		chID = relayInfo.ChannelId
	}
	tokenName := ""
	if ctx != nil {
		tokenName = ctx.GetString("token_name")
	}
	useTimeSeconds := 0
	if !relayInfo.StartTime.IsZero() {
		useTimeSeconds = int(time.Now().Unix() - relayInfo.StartTime.Unix())
	}
	other := map[string]interface{}{
		"asr":     true,
		"task_id": taskID,
		"reason":  reason,
	}
	other = model.SetBillingLogMetadata(other, model.BillingPhaseRefund, prechargeLogged, preConsumed, int64(preConsumed))
	model.RecordTaskBillingLog(model.RecordTaskBillingLogParams{
		UserId:         relayInfo.UserId,
		LogType:        model.LogTypeRefund,
		Content:        fmt.Sprintf("ASR 异步任务失败退还预扣 %s：%s", logger.FormatQuota(preConsumed), reason),
		ChannelId:      chID,
		ModelName:      relayInfo.OriginModelName,
		TokenName:      tokenName,
		Quota:          preConsumed,
		TokenId:        relayInfo.TokenId,
		UseTimeSeconds: useTimeSeconds,
		Group:          relayInfo.UsingGroup,
		Other:          other,
	})
}

// RecordASRErrorLog 将 ASR 上游/请求失败写入使用日志（LogTypeError）。
// ASR 错误日志不受 ERROR_LOG_ENABLED 开关限制，便于在使用日志中排查上游返回。
func RecordASRErrorLog(c *gin.Context, relayInfo *relaycommon.RelayInfo, apiErr *types.TokenFactoryError, taskID string) {
	if c == nil || apiErr == nil || !types.IsRecordErrorLog(apiErr) {
		return
	}
	userId := 0
	tokenId := 0
	channelId := 0
	channelType := 0
	modelName := ""
	tokenName := ""
	userGroup := ""
	isStream := false
	if relayInfo != nil {
		userId = relayInfo.UserId
		tokenId = relayInfo.TokenId
		modelName = relayInfo.OriginModelName
		userGroup = relayInfo.UsingGroup
		isStream = relayInfo.IsStream
		if relayInfo.ChannelMeta != nil {
			channelId = relayInfo.ChannelId
			channelType = relayInfo.ChannelType
		}
	}
	if userId == 0 {
		userId = c.GetInt("id")
	}
	if tokenId == 0 {
		tokenId = c.GetInt("token_id")
	}
	if channelId == 0 {
		channelId = c.GetInt("channel_id")
	}
	if channelType == 0 {
		channelType = c.GetInt("channel_type")
	}
	if modelName == "" {
		modelName = c.GetString("original_model")
	}
	if tokenName == "" {
		tokenName = c.GetString("token_name")
	}
	if userGroup == "" {
		userGroup = c.GetString("group")
	}

	other := make(map[string]interface{})
	if c.Request != nil && c.Request.URL != nil {
		other["request_path"] = c.Request.URL.Path
	}
	other["asr"] = true
	other["error_type"] = apiErr.GetErrorType()
	other["error_code"] = apiErr.GetErrorCode()
	other["status_code"] = apiErr.StatusCode
	other["channel_id"] = channelId
	other["channel_name"] = c.GetString("channel_name")
	other["channel_type"] = channelType
	if taskID != "" {
		other["task_id"] = taskID
	}
	adminInfo := make(map[string]interface{})
	adminInfo["use_channel"] = c.GetStringSlice("use_channel")
	isMultiKey := common.GetContextKeyBool(c, constant.ContextKeyChannelIsMultiKey)
	if isMultiKey {
		adminInfo["is_multi_key"] = true
		adminInfo["multi_key_index"] = common.GetContextKeyInt(c, constant.ContextKeyChannelMultiKeyIndex)
	}
	AppendChannelAffinityAdminInfo(c, adminInfo)
	other["admin_info"] = adminInfo

	useTimeSeconds := 0
	if relayInfo != nil && !relayInfo.StartTime.IsZero() {
		useTimeSeconds = int(time.Now().Unix() - relayInfo.StartTime.Unix())
	} else {
		startTime := common.GetContextKeyTime(c, constant.ContextKeyRequestStartTime)
		if !startTime.IsZero() {
			useTimeSeconds = int(time.Since(startTime).Seconds())
		}
	}

	model.RecordErrorLog(
		c,
		userId,
		channelId,
		modelName,
		tokenName,
		apiErr.MaskSensitiveErrorWithStatusCode(),
		tokenId,
		useTimeSeconds,
		isStream,
		userGroup,
		other,
		apiErr.LogErrorOriginHint(),
	)
}

// formatASRSecondsDisplay 音频时长展示：最多 2 位小数并去掉末尾 0（10.00 → 10，10.50 → 10.5）。
func formatASRSecondsDisplay(seconds float64) string {
	s := fmt.Sprintf("%.2f", seconds)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	if s == "" || s == "-" {
		return "0"
	}
	return s
}

// formatASRUnitPriceDisplay 将每秒美元单价转为系统默认展示货币（含符号）。
func formatASRUnitPriceDisplay(usdPerSecond float64) string {
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		return "$" + formatTierUsdPrice(usdPerSecond)
	}
	symbol := operation_setting.GetCurrencySymbol()
	if symbol == "" {
		symbol = "$"
	}
	rate := operation_setting.GetUsdToCurrencyRate(operation_setting.USDExchangeRate)
	if rate <= 0 {
		rate = 1
	}
	return symbol + formatTierUsdPrice(usdPerSecond*rate)
}
