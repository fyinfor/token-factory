package service

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

// 视频超分分辨率枚举（渠道规则配置的可选值）。
var (
	// VideoUpscaleSourceResolutions 生成分辨率可选值（实际生成档位）。
	VideoUpscaleSourceResolutions = []string{"480p", "540p", "720p", "768p", "1080p", "2K", "4K"}
	// VideoUpscaleTargetResolutions 超分分辨率可选值（对外输出档位）。
	VideoUpscaleTargetResolutions = []string{"720p", "1080p", "2K", "4K"}
)

// context key：提交阶段命中渠道超分规则时写入，供预扣日志与任务落库读取。
const contextKeyVideoUpscaleRule = "video_upscale_rule"

func isVideoUpscaleSourceResolution(label string) bool {
	for _, v := range VideoUpscaleSourceResolutions {
		if strings.EqualFold(v, label) {
			return true
		}
	}
	return false
}

func isVideoUpscaleTargetResolution(label string) bool {
	for _, v := range VideoUpscaleTargetResolutions {
		if strings.EqualFold(v, label) {
			return true
		}
	}
	return false
}

// SanitizeChannelVideoUpscaleRules 清洗渠道超分规则：
// 归一化分辨率标识、过滤非法档位/空模版、超分分辨率去重（保留第一条）。
// 与 SanitizeChannelModelRateLimits 同风格：非法行静默丢弃，保存即收敛。
func SanitizeChannelVideoUpscaleRules(rules []dto.ChannelVideoUpscaleRule) []dto.ChannelVideoUpscaleRule {
	if len(rules) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(rules))
	out := make([]dto.ChannelVideoUpscaleRule, 0, len(rules))
	for _, r := range rules {
		source := common.FormatVideoResolutionLabel(r.SourceResolution)
		target := common.FormatVideoResolutionLabel(r.TargetResolution)
		if !isVideoUpscaleSourceResolution(source) || !isVideoUpscaleTargetResolution(target) {
			continue
		}
		if r.TemplateId == 0 {
			continue
		}
		key := strings.ToLower(target)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, dto.ChannelVideoUpscaleRule{
			SourceResolution: source,
			TargetResolution: target,
			TemplateId:       r.TemplateId,
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// MatchChannelVideoUpscaleRule 按用户请求分辨率匹配渠道超分规则：
// 请求分辨率等于规则的「超分分辨率」（target）时命中，返回规则指针。
func MatchChannelVideoUpscaleRule(rules []dto.ChannelVideoUpscaleRule, requestResolutionLabel string) *dto.ChannelVideoUpscaleRule {
	want := normalizeResolutionLabelForMatch(requestResolutionLabel)
	if want == "" {
		return nil
	}
	for i := range rules {
		if normalizeResolutionLabelForMatch(rules[i].TargetResolution) == want {
			return &rules[i]
		}
	}
	return nil
}

// GetVideoUpscaleRuleFromContext 读取提交阶段命中的渠道超分规则。
func GetVideoUpscaleRuleFromContext(c *gin.Context) *dto.ChannelVideoUpscaleRule {
	if c == nil {
		return nil
	}
	v, exists := c.Get(contextKeyVideoUpscaleRule)
	if !exists {
		return nil
	}
	rule, ok := v.(dto.ChannelVideoUpscaleRule)
	if !ok {
		return nil
	}
	return &rule
}

// ApplyChannelVideoUpscaleRule 视频任务提交阶段的超分钩子：
// 命中渠道规则时只把规则写入 context，不改写请求分辨率。
// 预扣按用户选择的超分分辨率计价；上游生成前再改写为生成分辨率。
// 全局配置不完整、非视频计费渠道、透传 body 渠道、未命中规则时均为无操作。
func ApplyChannelVideoUpscaleRule(c *gin.Context, info *relaycommon.RelayInfo) {
	if c == nil || info == nil {
		return
	}
	if !constant.UsesRelayVideoPricing(info.ChannelType) {
		return
	}
	if !operation_setting.IsVideoUpscaleReady() {
		return
	}
	// 透传 body 渠道直接转发客户端原始请求体，改写 task_request 不会生效，暂不支持超分。
	if info.ChannelSetting.PassThroughBodyEnabled {
		return
	}
	rules := SanitizeChannelVideoUpscaleRules(info.ChannelOtherSettings.VideoUpscaleRules)
	if len(rules) == 0 {
		return
	}
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return
	}
	// 请求分辨率标识：与计费匹配同口径（metadata.resolution > 顶层 resolution > size）。
	requestLabel := requestVideoResolutionLabel(&req)
	if requestLabel == "" {
		return
	}
	rule := MatchChannelVideoUpscaleRule(rules, requestLabel)
	if rule == nil {
		return
	}
	c.Set(contextKeyVideoUpscaleRule, *rule)
}

// RewriteTaskRequestForVideoUpscale 预扣完成后、构建上游请求前：
// 将 task_request 分辨率改写为渠道规则的生成分辨率。
func RewriteTaskRequestForVideoUpscale(c *gin.Context) {
	rule := GetVideoUpscaleRuleFromContext(c)
	if c == nil || rule == nil {
		return
	}
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return
	}
	rewriteTaskRequestResolution(&req, rule.SourceResolution)
	c.Set("task_request", req)
}

// applyUpscaleTargetToBillingRequest 有超分规则时，把本地请求副本改回用户选择的超分分辨率。
// 仅用于日志匹配/展示；不改写 context 里发给上游的请求。
func applyUpscaleTargetToBillingRequest(c *gin.Context, req *relaycommon.TaskSubmitReq) {
	if c == nil || req == nil {
		return
	}
	rule := GetVideoUpscaleRuleFromContext(c)
	if rule == nil || strings.TrimSpace(rule.TargetResolution) == "" {
		return
	}
	rewriteTaskRequestResolution(req, rule.TargetResolution)
}

// shouldKeepUpscaleTargetResolution 仅当超分成功且配置了对应超分单价时，
// 结算/日志保留用户选择的超分分辨率，不做上游成片分辨率矫正。
func shouldKeepUpscaleTargetResolution(task *model.Task) bool {
	if task == nil || task.PrivateData.VideoUpscale == nil {
		return false
	}
	ups := task.PrivateData.VideoUpscale
	if ups.Status != model.TaskVideoUpscaleStatusSuccess {
		return false
	}
	target := strings.TrimSpace(ups.TargetResolution)
	source := strings.TrimSpace(ups.SourceResolution)
	if target == "" {
		return false
	}
	_, ok := VideoUpscalePricePerSecond(task.ChannelId, taskModelName(task), target, source)
	return ok
}

// applyTaskUpscaleTargetToBillingRequest 结算侧：超分成功且配置了超分价时，
// 按用户选择的目标分辨率改写本地请求副本。
func applyTaskUpscaleTargetToBillingRequest(task *model.Task, req *relaycommon.TaskSubmitReq) bool {
	if task == nil || req == nil || !shouldKeepUpscaleTargetResolution(task) {
		return false
	}
	target := strings.TrimSpace(task.PrivateData.VideoUpscale.TargetResolution)
	if target == "" {
		return false
	}
	rewriteTaskRequestResolution(req, target)
	return true
}

// requestVideoResolutionLabel 提取请求的分辨率标识并归一化（如 720p）。
func requestVideoResolutionLabel(req *relaycommon.TaskSubmitReq) string {
	if req == nil {
		return ""
	}
	if req.Metadata != nil {
		if v, ok := req.Metadata["resolution"].(string); ok {
			if label := common.FormatVideoResolutionLabel(v); label != "" {
				return label
			}
		}
	}
	if label := common.FormatVideoResolutionLabel(req.Resolution); label != "" {
		return label
	}
	if w, h, ok := common.ResolveVideoDimensionsFromRequest(req.Size, "", req.Ratio, req.Metadata); ok && w > 0 && h > 0 {
		return common.FormatVideoResolutionLabel(fmt.Sprintf("%dx%d", w, h))
	}
	return ""
}

// rewriteTaskRequestResolution 将请求中的分辨率改写为生成分辨率：
// 顶层 resolution、metadata.resolution 直接改写；size（像素尺寸）按原比例换算改写。
func rewriteTaskRequestResolution(req *relaycommon.TaskSubmitReq, sourceLabel string) {
	if req == nil || sourceLabel == "" {
		return
	}
	if strings.TrimSpace(req.Resolution) != "" {
		req.Resolution = sourceLabel
	}
	if req.Metadata != nil {
		if _, ok := req.Metadata["resolution"]; ok {
			req.Metadata["resolution"] = sourceLabel
		}
		if _, ok := req.Metadata["size"]; ok {
			ratio := strings.TrimSpace(req.Ratio)
			if ratio == "" {
				if v, ok := req.Metadata["ratio"].(string); ok {
					ratio = strings.TrimSpace(v)
				}
			}
			if w, h, ok := common.ParseVideoResolutionAndRatio(sourceLabel, ratio); ok && w > 0 && h > 0 {
				req.Metadata["size"] = fmt.Sprintf("%dx%d", w, h)
			}
		}
	}
	if strings.TrimSpace(req.Size) != "" {
		ratio := strings.TrimSpace(req.Ratio)
		if ratio == "" && req.Metadata != nil {
			if v, ok := req.Metadata["ratio"].(string); ok {
				ratio = strings.TrimSpace(v)
			}
		}
		if w, h, ok := common.ParseVideoResolutionAndRatio(sourceLabel, ratio); ok && w > 0 && h > 0 {
			req.Size = fmt.Sprintf("%dx%d", w, h)
		}
	}
}

// ---------------------------------------------------------------------------
// 超分计费
// ---------------------------------------------------------------------------

// VideoUpscalePricePerSecond 解析模型在指定「原分辨率 → 超分分辨率」下的每秒单价（USD）。
// 优先精确匹配 source+target；旧数据无 source 时按超分分辨率兜底。渠道价优先、全局价兜底。
func VideoUpscalePricePerSecond(channelID int, modelName, targetResolutionLabel, sourceResolutionLabel string) (float64, bool) {
	target := normalizeResolutionLabelForMatch(targetResolutionLabel)
	if target == "" {
		return 0, false
	}
	source := normalizeResolutionLabelForMatch(sourceResolutionLabel)
	matchRows := func(rows []ratio_setting.VideoUpscalePriceRule) (float64, bool) {
		var fallback float64
		var hasFallback bool
		for i := range rows {
			if rows[i].Price <= 0 {
				continue
			}
			if normalizeResolutionLabelForMatch(rows[i].Resolution) != target {
				continue
			}
			rowSource := normalizeResolutionLabelForMatch(rows[i].SourceResolution)
			if source != "" && rowSource != "" && rowSource == source {
				return rows[i].Price, true
			}
			if rowSource == "" && !hasFallback {
				fallback = rows[i].Price
				hasFallback = true
			}
		}
		if source == "" && hasFallback {
			return fallback, true
		}
		if hasFallback {
			return fallback, true
		}
		return 0, false
	}
	if rules, ok := ratio_setting.GetChannelVideoPricingRules(channelID, modelName); ok {
		if p, ok := matchRows(rules.VideoUpscalePerSecond); ok {
			return p, true
		}
	}
	if rules, ok := ratio_setting.GetVideoPricingRules(modelName); ok {
		if p, ok := matchRows(rules.VideoUpscalePerSecond); ok {
			return p, true
		}
	}
	return 0, false
}

func videoUpscalePricePerSecondFromRules(rules ratio_setting.VideoPricingRules, targetResolutionLabel, sourceResolutionLabel string) (float64, bool) {
	target := normalizeResolutionLabelForMatch(targetResolutionLabel)
	source := normalizeResolutionLabelForMatch(sourceResolutionLabel)
	if target == "" {
		return 0, false
	}
	var fallback float64
	hasFallback := false
	for _, row := range rules.VideoUpscalePerSecond {
		if row.Price <= 0 || normalizeResolutionLabelForMatch(row.Resolution) != target {
			continue
		}
		rowSource := normalizeResolutionLabelForMatch(row.SourceResolution)
		if source != "" && rowSource != "" && rowSource == source {
			return row.Price, true
		}
		if rowSource == "" && !hasFallback {
			fallback = row.Price
			hasFallback = true
		}
	}
	if hasFallback {
		return fallback, true
	}
	return 0, false
}

// videoUpscaleEffectivePricePerSecondUSD 计算超分每秒有效价（USD）。
// 超分价来自模型价格配置（无独立渠道档），作为基准价传入；全局价传 0 时
// EffectiveRuleUnitPrice 会回退为基准价，避免加价折扣为 0% 时有效价被算成 0。
func videoUpscaleEffectivePricePerSecondUSD(task *model.Task, targetResolutionLabel, sourceResolutionLabel string) (float64, bool) {
	modelName := taskModelName(task)
	var price float64
	var ok bool
	bc := task.PrivateData.BillingContext
	hasIndependentTimePricing := false
	if bc != nil && bc.TimePricingPlanID > 0 && strings.TrimSpace(bc.TimePricingPayload) != "" {
		if payload, err := model.ParseChannelModelPricePlanPayload(bc.TimePricingPayload); err == nil && payload.ResolvedMode() == model.ChannelModelPricePlanModePrice {
			hasIndependentTimePricing = true
			if payload.VideoPricingRules != nil {
				price, ok = videoUpscalePricePerSecondFromRules(*payload.VideoPricingRules, targetResolutionLabel, sourceResolutionLabel)
			}
		}
	}
	if !hasIndependentTimePricing {
		price, ok = VideoUpscalePricePerSecond(task.ChannelId, modelName, targetResolutionLabel, sourceResolutionLabel)
	}
	if !ok || price <= 0 {
		return 0, false
	}
	costDisc := taskBillingContextEffectiveCostPercent(bc, task.ChannelId)
	markupDisc := model.ResolveEffectiveMarkupDiscountPercentForInviteeBilling(task.UserId, task.ChannelId, modelName)
	if bc != nil && bc.TimePricingPlanID > 0 && bc.MarkupDiscountPercent != nil {
		markupDisc = *bc.MarkupDiscountPercent
	}
	eff := effectiveVideoPerSecondUSD(price, 0, costDisc, markupDisc)
	if eff <= 0 {
		return 0, false
	}
	return eff, true
}

// videoUpscaleBillingOnComplete 任务完成时计算超分附加费用（quota）与日志明细。
// 超分费用 = ceil(超分后视频时长) × 有效每秒价 × QuotaPerUnit × groupRatio（decimal 高精度）。
// 返回 0 表示无超分费用（未超分/超分失败/未配置价格）。
func videoUpscaleBillingOnComplete(task *model.Task) (int, map[string]interface{}) {
	ups := task.PrivateData.VideoUpscale
	if ups == nil || ups.Status != model.TaskVideoUpscaleStatusSuccess {
		return 0, nil
	}
	durationSec := ups.DurationSec
	if durationSec <= 0 {
		if d := extractDurationFromTaskData(task.Data); d > 0 {
			durationSec = float64(d)
		}
	}
	if durationSec <= 0 {
		if sec := taskBillingSecondsEstimate(task); sec > 0 {
			durationSec = float64(sec)
		}
	}
	if durationSec <= 0 {
		return 0, nil
	}
	effPrice, ok := videoUpscaleEffectivePricePerSecondUSD(task, ups.TargetResolution, ups.SourceResolution)
	if !ok {
		return 0, nil
	}
	groupRatio := 1.0
	if bc := task.PrivateData.BillingContext; bc != nil && bc.GroupRatio > 0 {
		groupRatio = bc.GroupRatio
	}
	seconds := math.Ceil(durationSec)
	if seconds <= 0 {
		return 0, nil
	}
	// 高精度：seconds × price × QuotaPerUnit × groupRatio，最后四舍五入到整数 quota。
	rawQuota := decimal.NewFromInt(int64(seconds)).
		Mul(decimal.NewFromFloat(effPrice)).
		Mul(decimal.NewFromInt(int64(common.QuotaPerUnit))).
		Mul(decimal.NewFromFloat(groupRatio))
	quota := int(rawQuota.Round(0).IntPart())
	if quota <= 0 {
		quota = 1
	}
	other := map[string]interface{}{
		"video_upscale":                   true,
		"video_upscale_resolution":        common.FormatVideoResolutionLabel(ups.TargetResolution),
		"video_upscale_source_resolution": common.FormatVideoResolutionLabel(ups.SourceResolution),
		"video_upscale_seconds":           seconds,
		"video_upscale_price_per_second":  effPrice,
		"video_upscale_quota":             quota,
	}
	return quota, other
}

// appendVideoUpscaleBilling 将超分附加费并入结算额度与日志明细。
// actualQuota 为原业务计费额度；返回合并后的额度与需并入 other 的明细（无超分费时为 nil）。
func appendVideoUpscaleBilling(task *model.Task, actualQuota int) (int, map[string]interface{}) {
	upsQuota, upsOther := videoUpscaleBillingOnComplete(task)
	if upsQuota <= 0 {
		return actualQuota, nil
	}
	return actualQuota + upsQuota, upsOther
}

// videoUpscalePreChargeOther 预扣日志的超分标签（仅单价，时长未知）：
// 命中渠道规则且模型配置了对应超分价格时返回标签字段，否则返回 nil。
func videoUpscalePreChargeOther(c *gin.Context, info *relaycommon.RelayInfo) map[string]interface{} {
	rule := GetVideoUpscaleRuleFromContext(c)
	if rule == nil || info == nil {
		return nil
	}
	price, ok := VideoUpscalePricePerSecond(info.ChannelId, info.OriginModelName, rule.TargetResolution, rule.SourceResolution)
	if info.PriceData.TimePricingPlanID > 0 && strings.TrimSpace(info.PriceData.TimePricingPayload) != "" {
		if payload, err := model.ParseChannelModelPricePlanPayload(info.PriceData.TimePricingPayload); err == nil && payload.ResolvedMode() == model.ChannelModelPricePlanModePrice && payload.VideoPricingRules != nil {
			price, ok = videoUpscalePricePerSecondFromRules(*payload.VideoPricingRules, rule.TargetResolution, rule.SourceResolution)
		}
	}
	if !ok || price <= 0 {
		return nil
	}
	return map[string]interface{}{
		"video_upscale":                   true,
		"video_upscale_resolution":        common.FormatVideoResolutionLabel(rule.TargetResolution),
		"video_upscale_source_resolution": common.FormatVideoResolutionLabel(rule.SourceResolution),
		"video_upscale_price_per_second":  price,
	}
}

// AttachVideoUpscaleToTask 任务落库时写入超分上下文（pending）。
func AttachVideoUpscaleToTask(c *gin.Context, task *model.Task) {
	if c == nil || task == nil {
		return
	}
	rule := GetVideoUpscaleRuleFromContext(c)
	if rule == nil {
		return
	}
	task.PrivateData.VideoUpscale = &model.TaskVideoUpscaleInfo{
		SourceResolution: common.FormatVideoResolutionLabel(rule.SourceResolution),
		TargetResolution: common.FormatVideoResolutionLabel(rule.TargetResolution),
		TemplateId:       rule.TemplateId,
		Status:           model.TaskVideoUpscaleStatusPending,
	}
}

// IsVideoUpscaleInProgress 任务是否正在等待/处理超分（对外仍视为进行中）。
func IsVideoUpscaleInProgress(task *model.Task) bool {
	if task == nil || task.PrivateData.VideoUpscale == nil {
		return false
	}
	st := task.PrivateData.VideoUpscale.Status
	return st == model.TaskVideoUpscaleStatusPending || st == model.TaskVideoUpscaleStatusProcessing
}

// HasSuccessfulVideoUpscale 超分是否已成功（应用超分结果 URL / 分辨率覆盖）。
func HasSuccessfulVideoUpscale(task *model.Task) bool {
	return task != nil &&
		task.PrivateData.VideoUpscale != nil &&
		task.PrivateData.VideoUpscale.Status == model.TaskVideoUpscaleStatusSuccess
}

// ApplyVideoUpscaleToClientVideoResponse 用户查询视频结果时：
// 1) 超分成功则用超分后的 ResultURL 替换响应中的视频链接；
// 2) 配置了超分计费时，将 resolution 改写为用户选择的超分目标分辨率。
func ApplyVideoUpscaleToClientVideoResponse(respJSON []byte, task *model.Task) []byte {
	if len(respJSON) == 0 || !HasSuccessfulVideoUpscale(task) {
		return respJSON
	}
	var root map[string]interface{}
	if err := common.Unmarshal(respJSON, &root); err != nil || root == nil {
		return respJSON
	}
	changed := false
	if url := strings.TrimSpace(task.GetResultURL()); url != "" {
		if applyVideoURLToResponseMap(root, url) {
			changed = true
		}
	}
	if shouldKeepUpscaleTargetResolution(task) {
		if label := common.FormatVideoResolutionLabel(task.PrivateData.VideoUpscale.TargetResolution); label != "" {
			if cur, _ := root["resolution"].(string); strings.TrimSpace(cur) != label {
				root["resolution"] = label
				changed = true
			}
		}
	}
	if !changed {
		return respJSON
	}
	out, err := common.Marshal(root)
	if err != nil {
		return respJSON
	}
	return out
}

func applyVideoURLToResponseMap(root map[string]interface{}, url string) bool {
	if root == nil || url == "" {
		return false
	}
	changed := false
	// 与非超分查询一致：只保留 content.video_url / output.video_url，去掉顶层 url/result_url 及嵌套 url。
	for _, key := range []string{"url", "result_url"} {
		if _, ok := root[key]; ok {
			delete(root, key)
			changed = true
		}
	}
	ensureNestedVideoURL := func(key string) {
		nested, _ := root[key].(map[string]interface{})
		if nested == nil {
			root[key] = map[string]interface{}{"video_url": url}
			changed = true
			return
		}
		if _, ok := nested["url"]; ok {
			delete(nested, "url")
			changed = true
		}
		if cur, _ := nested["video_url"].(string); strings.TrimSpace(cur) != url {
			nested["video_url"] = url
			changed = true
		}
	}
	ensureNestedVideoURL("content")
	ensureNestedVideoURL("output")
	return changed
}

// SyncVideoUpscaleOnFetch 用户查询任务时同步推进超分：更新结束时间/状态/结果地址，并在完成时结算。
// 返回 true 表示任务已被推进到新状态（调用方应基于最新 task 构建响应）。
func SyncVideoUpscaleOnFetch(ctx context.Context, task *model.Task) bool {
	if task == nil || !IsVideoUpscaleInProgress(task) {
		return false
	}
	snap := task.Snapshot()
	handled, shouldSettle := HandleVideoUpscalePoll(ctx, task)
	if !handled {
		return false
	}
	if !shouldSettle && snap.Equal(task.Snapshot()) {
		return false
	}
	won, err := task.UpdateWithStatus(snap.Status)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("UpdateWithStatus failed for upscale fetch task %s: %s", task.TaskID, err.Error()))
		return false
	}
	if !won {
		logger.LogWarn(ctx, fmt.Sprintf("Task %s already transitioned by another process during upscale fetch", task.TaskID))
		return false
	}
	if shouldSettle && task.Status == model.TaskStatusSuccess {
		SettleTaskBillingOnFetch(ctx, task, &relaycommon.TaskInfo{
			Status: string(model.TaskStatusSuccess),
			Url:    task.GetResultURL(),
		})
	}
	return true
}

// HandleVideoUpscalePoll 轮询入口：若任务处于 MPS 超分处理中，查询 MPS 并更新任务。
// handled=true 表示本轮已接管，调用方不应再请求上游生成任务。
func HandleVideoUpscalePoll(ctx context.Context, task *model.Task) (handled bool, shouldSettle bool) {
	if task == nil || task.PrivateData.VideoUpscale == nil {
		return false, false
	}
	ups := task.PrivateData.VideoUpscale
	if ups.Status != model.TaskVideoUpscaleStatusProcessing || strings.TrimSpace(ups.MpsTaskId) == "" {
		return false, false
	}
	result, err := DescribeVideoUpscaleTask(ups.MpsTaskId)
	if err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("查询 MPS 超分任务失败 task=%s: %s", task.TaskID, err.Error()))
		return true, false
	}
	if result == nil || !result.Finished {
		task.Status = model.TaskStatusInProgress
		task.Progress = "97%"
		return true, false
	}
	now := time.Now().Unix()
	if result.Success && strings.TrimSpace(result.OutputURL) != "" {
		ups.Status = model.TaskVideoUpscaleStatusSuccess
		ups.OutputUrl = result.OutputURL
		ups.DurationSec = result.DurationSec
		ups.FailReason = ""
		task.PrivateData.ResultURL = result.OutputURL
		task.Status = model.TaskStatusSuccess
		task.Progress = "100%"
		// 超分完成时刻作为任务结束时间（生成阶段已清空 FinishTime）。
		task.FinishTime = now
		return true, true
	}
	// 超分失败：主流程仍按生成成功结算，不附加超分费用。
	reason := strings.TrimSpace(result.FailReason)
	if reason == "" {
		reason = "MPS 超分失败"
	}
	logger.LogWarn(ctx, fmt.Sprintf("视频超分失败，回退原始视频 task=%s: %s", task.TaskID, reason))
	ups.Status = model.TaskVideoUpscaleStatusFailed
	ups.FailReason = reason
	task.Status = model.TaskStatusSuccess
	task.Progress = "100%"
	task.FinishTime = now
	return true, true
}

// BeginVideoUpscaleAfterGenerate 上游视频生成成功后启动超分。
// 返回 true 表示已提交 MPS、任务应保持 IN_PROGRESS，调用方不得结算。
// 提交失败则标记 failed 并返回 false，主流程按生成成功继续。
func BeginVideoUpscaleAfterGenerate(ctx context.Context, task *model.Task, sourceURL string) bool {
	if task == nil || task.PrivateData.VideoUpscale == nil {
		return false
	}
	ups := task.PrivateData.VideoUpscale
	if ups.Status != model.TaskVideoUpscaleStatusPending {
		return false
	}
	if !operation_setting.IsVideoUpscaleReady() {
		ups.Status = model.TaskVideoUpscaleStatusFailed
		ups.FailReason = "视频超分全局配置不完整"
		return false
	}
	sourceURL = strings.TrimSpace(sourceURL)
	if sourceURL == "" || strings.HasPrefix(sourceURL, "data:") {
		ups.Status = model.TaskVideoUpscaleStatusFailed
		ups.FailReason = "超分输入视频 URL 无效"
		return false
	}
	mpsTaskId, err := SubmitVideoUpscaleTask(sourceURL, ups.TemplateId)
	if err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("提交 MPS 超分失败，回退原始视频 task=%s: %s", task.TaskID, err.Error()))
		ups.Status = model.TaskVideoUpscaleStatusFailed
		ups.FailReason = err.Error()
		return false
	}
	ups.MpsTaskId = mpsTaskId
	ups.Status = model.TaskVideoUpscaleStatusProcessing
	task.Status = model.TaskStatusInProgress
	task.Progress = "97%"
	task.FinishTime = 0
	logger.LogInfo(ctx, fmt.Sprintf("已提交 MPS 超分任务 task=%s mps=%s", task.TaskID, mpsTaskId))
	return true
}
