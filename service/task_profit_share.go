/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

package service

import (
	"context"
	"math"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"
)

// profitShareExtraTotalTokensKey 由 RecalculateTaskQuotaByTokens 传入，用于异步 token 补扣的加价比例推算；不入库日志。
const profitShareExtraTotalTokensKey = "_profit_share_total_tokens"

// TryPostWalletProfitShareForTaskBilledQuota 任务完成侧已知「最终对用户计费额度」后入账利润分成（含预扣与实际一致、操练场等无补扣 delta 的场景）。
// 提交阶段预扣结算不调用本函数，由轮询/fetch 完成路径统一触发。
func TryPostWalletProfitShareForTaskBilledQuota(ctx context.Context, task *model.Task, billedQuota int, hintTotalTokens int) {
	_ = ctx
	if task == nil || billedQuota <= 0 {
		return
	}
	if !common.IsDistributorProfitShareMode() {
		return
	}
	bs := strings.TrimSpace(task.PrivateData.BillingSource)
	if bs != "" && bs != BillingSourceWallet {
		return
	}
	ratio, ok := taskProfitShareMarkupSliceRatio(task, hintTotalTokens)
	if !ok || ratio <= 0 {
		return
	}
	slice := int(math.Round(float64(billedQuota) * ratio))
	if slice <= 0 {
		return
	}
	invitee, err := model.GetUserById(task.UserId, false)
	if err != nil || invitee == nil || invitee.InviterId <= 0 {
		return
	}
	inviter, err2 := model.GetUserById(invitee.InviterId, false)
	if err2 != nil || inviter == nil || !model.UserIsDistributor(inviter) {
		return
	}
	modelName := strings.TrimSpace(taskModelName(task))
	bps := model.EffectiveAffiliateCommissionBps(inviter, task.UserId)
	if bps <= 0 {
		return
	}
	maxBps := 10000
	if bps > maxBps {
		bps = maxBps
	}
	reward := int(int64(slice) * int64(bps) / int64(maxBps))
	if reward <= 0 {
		return
	}
	if err := model.CreditDistributorProfitShare(invitee.InviterId, task.UserId, task.ChannelId, modelName, billedQuota, slice, reward, bps, taskProfitShareTotalTokens(task, hintTotalTokens), taskProfitShareBillingMode(task)); err != nil {
		common.SysError("TryPostWalletProfitShareForTaskBilledQuota: " + err.Error())
	}
}

// taskProfitShareTotalTokens returns the token consumption for a task, preferring the hint.
func taskProfitShareTotalTokens(task *model.Task, hintTotalTokens int) int {
	if hintTotalTokens > 0 {
		return hintTotalTokens
	}
	return 0
}

// taskProfitShareBillingMode classifies the profit-share billing mode for display/filtering.
func taskProfitShareBillingMode(task *model.Task) string {
	if task == nil {
		return ""
	}
	bc := task.PrivateData.BillingContext
	if bc != nil && strings.EqualFold(strings.TrimSpace(bc.VideoRuleUnit), VideoRuleUnitPerToken) {
		return "video_token"
	}
	ch, err := model.CacheGetChannel(task.ChannelId)
	if err == nil && ch != nil && constant.UsesRelayVideoPricing(ch.Type) {
		return "video"
	}
	return "text"
}

func taskProfitShareMarkupSliceRatio(task *model.Task, hintTotalTokens int) (float64, bool) {
	if task == nil {
		return 0, false
	}
	ch, err := model.CacheGetChannel(task.ChannelId)
	if err != nil || ch == nil {
		return 0, false
	}
	if constant.UsesRelayVideoPricing(ch.Type) {
		// 视频按 Token 规则计费：BillingContext 快照单价与 per_video 相同的两档公式，不依赖 ModelRatio。
		if r, ok := taskProfitShareMarkupRatioVideoPerTokenSubmitInput(task); ok {
			return r, true
		}
		if r, ok := taskProfitShareMarkupRatioVideoComplete(task); ok {
			return r, true
		}
		if r, ok := taskProfitShareMarkupRatioVideoSubmitInput(task); ok {
			return r, true
		}
		if r, ok := taskProfitShareMarkupRatioVideoPerVideoSubmitInput(task); ok {
			return r, true
		}
	}
	if hintTotalTokens > 0 {
		if r, ok := taskProfitShareMarkupRatioFromUpstreamTokens(task, hintTotalTokens); ok {
			return r, true
		}
	}
	if r, ok := taskProfitShareMarkupRatioPerCallModelPrice(task); ok {
		return r, true
	}
	return 0, false
}

func taskProfitShareMarkupRatioVideoComplete(task *model.Task) (float64, bool) {
	meta, ok := extractVideoMetadataFromTaskData(task)
	if !ok {
		return 0, false
	}
	modelName := strings.TrimSpace(taskModelName(task))
	if modelName == "" {
		return 0, false
	}
	mode := detectTaskVideoBillingMode(task)
	channelPerSec := channelVideoPerSecondUSD(task.ChannelId, modelName, mode, meta.Width, meta.Height, meta.HasAudio)
	if channelPerSec <= 0 {
		return 0, false
	}
	seconds := int(math.Ceil(meta.DurationSec))
	if seconds <= 0 {
		return 0, false
	}
	groupRatio := 1.0
	if task.PrivateData.BillingContext != nil && task.PrivateData.BillingContext.GroupRatio > 0 {
		groupRatio = task.PrivateData.BillingContext.GroupRatio
	}
	costDisc := channelCostDiscountPercentFromTask(task)
	markup := markupDiscountPercentFromTask(task, modelName)
	globalPerSec := globalVideoPerSecondUSD(modelName, mode, meta.Width, meta.Height, meta.HasAudio)
	effW := effectiveVideoPerSecondUSD(channelPerSec, globalPerSec, costDisc, markup)
	eff0 := effectiveVideoPerSecondUSD(channelPerSec, globalPerSec, costDisc, 0)
	qW := int(math.Round(float64(seconds) * effW * common.QuotaPerUnit * groupRatio))
	q0 := int(math.Round(float64(seconds) * eff0 * common.QuotaPerUnit * groupRatio))
	if qW <= 0 || qW <= q0 {
		return 0, false
	}
	return float64(qW-q0) / float64(qW), true
}

func taskProfitShareMarkupRatioVideoSubmitInput(task *model.Task) (float64, bool) {
	if task == nil || strings.TrimSpace(task.Properties.Input) == "" {
		return 0, false
	}
	var req relaycommon.TaskSubmitReq
	if err := common.UnmarshalJsonStr(task.Properties.Input, &req); err != nil {
		return 0, false
	}
	ri, ok := relayInfoSnapshotForProfitShare(task)
	if !ok {
		return 0, false
	}
	qW := calcVideoPerSecondQuotaFromTaskReq(ri, &req, ri.PriceData.MarkupDiscountPercent)
	q0 := calcVideoPerSecondQuotaFromTaskReq(ri, &req, 0)
	if qW <= 0 || qW <= q0 {
		return 0, false
	}
	return float64(qW-q0) / float64(qW), true
}

func taskProfitShareMarkupRatioVideoPerTokenSubmitInput(task *model.Task) (float64, bool) {
	if r, ok := taskProfitShareMarkupRatioVideoPerTokenSnapshot(task); ok {
		return r, true
	}
	if task == nil || strings.TrimSpace(task.Properties.Input) == "" {
		return 0, false
	}
	var req relaycommon.TaskSubmitReq
	if err := common.UnmarshalJsonStr(task.Properties.Input, &req); err != nil {
		return 0, false
	}
	modelName := strings.TrimSpace(taskModelName(task))
	if modelName == "" {
		return 0, false
	}
	mode := detectVideoBillingModeFromTaskReq(&req)
	width, height := videoDimensionsFromTaskRequest(req)
	hasAudio := taskRequestHasAudio(req)
	channelUSD := channelVideoPerTokenUSD(task.ChannelId, modelName, mode, width, height, hasAudio)
	if channelUSD <= 0 {
		return 0, false
	}
	groupRatio := 1.0
	if task.PrivateData.BillingContext != nil && task.PrivateData.BillingContext.GroupRatio > 0 {
		groupRatio = task.PrivateData.BillingContext.GroupRatio
	}
	costDisc := channelCostDiscountPercentFromTask(task)
	markup := markupDiscountPercentFromTask(task, modelName)
	globalUSD := globalVideoPerTokenUSD(modelName, mode, width, height, hasAudio)
	return taskProfitShareMarkupRatioFromRulePrices(channelUSD, globalUSD, costDisc, markup, groupRatio)
}

func taskProfitShareMarkupRatioVideoPerTokenSnapshot(task *model.Task) (float64, bool) {
	if task == nil || task.PrivateData.BillingContext == nil {
		return 0, false
	}
	bc := task.PrivateData.BillingContext
	if !strings.EqualFold(strings.TrimSpace(bc.VideoRuleUnit), VideoRuleUnitPerToken) || bc.VideoChannelRulePrice <= 0 {
		return 0, false
	}
	modelName := strings.TrimSpace(taskModelName(task))
	if modelName == "" {
		return 0, false
	}
	groupRatio := bc.GroupRatio
	if groupRatio <= 0 {
		groupRatio = 1
	}
	costDisc := channelCostDiscountPercentFromTask(task)
	markup := markupDiscountPercentFromTask(task, modelName)
	return taskProfitShareMarkupRatioFromRulePrices(bc.VideoChannelRulePrice, bc.VideoGlobalRulePrice, costDisc, markup, groupRatio)
}

func taskProfitShareMarkupRatioVideoPerVideoSubmitInput(task *model.Task) (float64, bool) {
	if r, ok := taskProfitShareMarkupRatioVideoPerVideoSnapshot(task); ok {
		return r, true
	}
	if task == nil || strings.TrimSpace(task.Properties.Input) == "" {
		return 0, false
	}
	var req relaycommon.TaskSubmitReq
	if err := common.UnmarshalJsonStr(task.Properties.Input, &req); err != nil {
		return 0, false
	}
	modelName := strings.TrimSpace(taskModelName(task))
	if modelName == "" {
		return 0, false
	}
	mode := detectVideoBillingModeFromTaskReq(&req)
	width, height := videoDimensionsFromTaskRequest(req)
	hasAudio := taskRequestHasAudio(req)
	channelUSD, ok := channelVideoPerVideoUSDForProfitShare(task.ChannelId, modelName, mode, width, height, hasAudio)
	if !ok || channelUSD <= 0 {
		return 0, false
	}
	groupRatio := 1.0
	if task.PrivateData.BillingContext != nil && task.PrivateData.BillingContext.GroupRatio > 0 {
		groupRatio = task.PrivateData.BillingContext.GroupRatio
	}
	costDisc := channelCostDiscountPercentFromTask(task)
	markup := markupDiscountPercentFromTask(task, modelName)
	globalUSD := globalVideoPerVideoUSDForProfitShare(modelName, mode, width, height, hasAudio)
	return taskProfitShareMarkupRatioFromRulePrices(channelUSD, globalUSD, costDisc, markup, groupRatio)
}

func taskProfitShareMarkupRatioVideoPerVideoSnapshot(task *model.Task) (float64, bool) {
	if task == nil || task.PrivateData.BillingContext == nil {
		return 0, false
	}
	bc := task.PrivateData.BillingContext
	if !strings.EqualFold(strings.TrimSpace(bc.VideoRuleUnit), "per_video") || bc.VideoChannelRulePrice <= 0 {
		return 0, false
	}
	modelName := strings.TrimSpace(taskModelName(task))
	if modelName == "" {
		return 0, false
	}
	groupRatio := bc.GroupRatio
	if groupRatio <= 0 {
		groupRatio = 1
	}
	costDisc := channelCostDiscountPercentFromTask(task)
	markup := markupDiscountPercentFromTask(task, modelName)
	return taskProfitShareMarkupRatioFromRulePrices(bc.VideoChannelRulePrice, bc.VideoGlobalRulePrice, costDisc, markup, groupRatio)
}

func taskProfitShareMarkupRatioFromRulePrices(channelUSD, globalUSD, costDisc, markup, groupRatio float64) (float64, bool) {
	effW := effectiveVideoPerVideoUSD(channelUSD, globalUSD, costDisc, markup)
	eff0 := effectiveVideoPerVideoUSD(channelUSD, globalUSD, costDisc, 0)
	qW := int(math.Round(effW * common.QuotaPerUnit * groupRatio))
	q0 := int(math.Round(eff0 * common.QuotaPerUnit * groupRatio))
	if qW <= 0 || qW <= q0 {
		return 0, false
	}
	return float64(qW-q0) / float64(qW), true
}

func markupDiscountPercentFromTask(task *model.Task, modelName string) float64 {
	if task == nil {
		return 0
	}
	if bc := task.PrivateData.BillingContext; bc != nil && bc.MarkupDiscountPercent != nil {
		return *bc.MarkupDiscountPercent
	}
	return model.ResolveEffectiveMarkupDiscountPercentForInviteeBilling(task.UserId, task.ChannelId, modelName)
}

func channelVideoPerVideoUSDForProfitShare(channelId int, modelName, mode string, width, height int, hasAudio bool) (float64, bool) {
	rules, ok := ratio_setting.GetChannelVideoPricingRules(channelId, modelName)
	if !ok || !ratio_setting.HasUsableVideoPerVideoRules(rules) {
		var globalOK bool
		rules, globalOK = ratio_setting.GetVideoPricingRules(modelName)
		if !globalOK || !ratio_setting.HasUsableVideoPerVideoRules(rules) {
			return 0, false
		}
	}
	match, ok := matchPerVideoPriceDetail(rules, mode, width, height, hasAudio)
	if !ok || match == nil || match.PricePerVideo <= 0 {
		return 0, false
	}
	return match.PricePerVideo, true
}

func globalVideoPerVideoUSDForProfitShare(modelName, mode string, width, height int, hasAudio bool) float64 {
	rules, ok := ratio_setting.GetVideoPricingRules(modelName)
	if !ok || !ratio_setting.HasUsableVideoPerVideoRules(rules) {
		return 0
	}
	match, ok := matchPerVideoPriceDetail(rules, mode, width, height, hasAudio)
	if !ok || match == nil {
		return 0
	}
	return match.PricePerVideo
}

func taskProfitShareMarkupRatioFromUpstreamTokens(task *model.Task, totalTokens int) (float64, bool) {
	if task == nil || totalTokens <= 0 {
		return 0, false
	}
	ri, ok := relayInfoSnapshotForProfitShare(task)
	if !ok {
		return 0, false
	}
	globalMr, globalOK, _ := ratio_setting.GetModelRatio(ri.OriginModelName)
	if !globalOK {
		globalMr = ri.PriceData.GlobalModelRatio
	}
	if globalMr <= 0 {
		globalMr = ri.PriceData.GlobalModelRatio
	}
	pd := ri.PriceData
	pd.GlobalModelRatio = globalMr
	ri.PriceData = pd
	qW := calcQuotaByUpstreamTokensWithMarkup(ri, totalTokens, pd.MarkupDiscountPercent)
	q0 := calcQuotaByUpstreamTokensWithMarkup(ri, totalTokens, 0)
	if qW <= 0 || qW <= q0 {
		return 0, false
	}
	return float64(qW-q0) / float64(qW), true
}

func taskProfitShareMarkupRatioPerCallModelPrice(task *model.Task) (float64, bool) {
	bc := task.PrivateData.BillingContext
	if bc == nil || bc.ModelPrice <= 0 {
		return 0, false
	}
	modelName := strings.TrimSpace(taskModelName(task))
	if modelName == "" {
		return 0, false
	}
	costDisc := channelCostDiscountPercentFromTask(task)
	markup := markupDiscountPercentFromTask(task, modelName)
	globalPrice, _ := ratio_setting.GetModelPrice(modelName, false)
	effW := model.EffectiveModelPrice(bc.ModelPrice, globalPrice, costDisc, markup)
	eff0 := model.EffectiveModelPrice(bc.ModelPrice, globalPrice, costDisc, 0)
	if effW <= 0 || effW <= eff0 {
		return 0, false
	}
	return (effW - eff0) / effW, true
}

func channelCostDiscountPercentFromTask(task *model.Task) float64 {
	if task == nil {
		return 100
	}
	if bc := task.PrivateData.BillingContext; bc != nil && bc.ChannelPriceDiscountPercent > 0 {
		return bc.ChannelPriceDiscountPercent
	}
	return model.ResolveChannelPriceDiscountPercent(task.ChannelId)
}

func relayInfoSnapshotForProfitShare(task *model.Task) (*relaycommon.RelayInfo, bool) {
	if task == nil {
		return nil, false
	}
	ch, err := model.CacheGetChannel(task.ChannelId)
	if err != nil || ch == nil {
		return nil, false
	}
	bc := task.PrivateData.BillingContext
	if bc == nil {
		return nil, false
	}
	modelName := strings.TrimSpace(taskModelName(task))
	if modelName == "" {
		return nil, false
	}
	markup := markupDiscountPercentFromTask(task, modelName)
	cost := bc.ChannelPriceDiscountPercent
	if cost <= 0 {
		cost = model.ResolveChannelPriceDiscountPercent(task.ChannelId)
	}
	gr := bc.GroupRatio
	if gr <= 0 {
		gr = 1
	}
	globalMr, globalOK, _ := ratio_setting.GetModelRatio(modelName)
	if !globalOK {
		globalMr = 0
	}
	ri := &relaycommon.RelayInfo{
		UserId:          task.UserId,
		OriginModelName: modelName,
		BillingSource:   task.PrivateData.BillingSource,
		ChannelMeta:     &relaycommon.ChannelMeta{ChannelType: ch.Type, ChannelId: task.ChannelId},
	}
	other := bc.OtherRatios
	if other != nil {
		cp := make(map[string]float64, len(other))
		for k, v := range other {
			cp[k] = v
		}
		other = cp
	}
	ri.PriceData = types.PriceData{
		ModelPrice:            bc.ModelPrice,
		ModelRatio:            bc.ModelRatio,
		GlobalModelRatio:      globalMr,
		GroupRatioInfo:        types.GroupRatioInfo{GroupRatio: gr},
		UsePrice:              true,
		CostDiscountPercent:   cost,
		MarkupDiscountPercent: markup,
		OtherRatios:           other,
		VideoOutputTokens:     0,
	}
	return ri, true
}
