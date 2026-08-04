package service

import (
	"fmt"
	"math"
	"strings"
	"time"

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

// PostASRConsumeQuota 阿里云 ASR 语音转写结算、日志与分润。
//
// 三条链路：
//  1. 同步转写：Relay 主流程已建立 Billing 会话（可能预扣），此处按真实秒数 SettleBilling 差额；
//  2. 异步转写（提交已预扣）：relayInfo.FinalPreConsumedQuota > 0 且 Billing == nil，按实际秒数与预扣差额补扣/退还；
//  3. 异步转写（兼容旧任务无预扣）：Billing == nil 且 FinalPreConsumedQuota == 0，全额新建计费会话扣费。
//
// 分润沿用钱包分润链路（tryPostWalletProfitShareCredit），按按秒消耗金额核算代理收益。
// taskID 非空时写入日志 other（异步任务日志详情展示）。
func PostASRConsumeQuota(ctx *gin.Context, relayInfo *relaycommon.RelayInfo, seconds float64, taskID string, extraContent string) *types.TokenFactoryError {
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
			tryPostWalletProfitShareCredit(ctx, relayInfo, usage, &summary)
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
		tryPostWalletProfitShareCredit(ctx, relayInfo, usage, &summary)
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
			tryPostWalletProfitShareCredit(ctx, relayInfo, usage, &summary)
		}
	}

	if summary.TotalTokens == 0 {
		logger.LogError(ctx, fmt.Sprintf("asr total tokens is 0, cannot consume quota, userId %d, channelId %d, tokenId %d, model %s",
			relayInfo.UserId, chID, relayInfo.TokenId, summary.ModelName))
	} else {
		model.UpdateUserUsedQuotaAndRequestCountWithGiftOffset(relayInfo.UserId, summary.Quota, walletGiftOffset(relayInfo))
		model.UpdateChannelUsedQuota(chID, summary.Quota)
	}

	useTimeSeconds := time.Now().Unix() - relayInfo.StartTime.Unix()
	logContent := fmt.Sprintf("ASR 语音转写时长 %s 秒，每秒价格 %s",
		formatASRSecondsDisplay(seconds), formatASRUnitPriceDisplay(relayInfo.PriceData.ModelPrice))
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
		UseTimeSeconds:   int(useTimeSeconds),
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

// RefundASRPreConsumedQuota 异步任务失败时退还提交阶段预扣额度。
func RefundASRPreConsumedQuota(ctx *gin.Context, relayInfo *relaycommon.RelayInfo, preConsumed int, taskID, reason string) {
	if relayInfo == nil || preConsumed <= 0 {
		return
	}
	if err := PostConsumeQuota(relayInfo, -preConsumed, preConsumed, true); err != nil {
		logger.LogError(ctx, fmt.Sprintf("asr async refund failed task=%s: %s", taskID, err.Error()))
		return
	}
	logger.LogInfo(ctx, fmt.Sprintf("ASR 异步任务失败已退还预扣 %s（task_id=%s）: %s",
		logger.FormatQuota(preConsumed), taskID, reason))
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
