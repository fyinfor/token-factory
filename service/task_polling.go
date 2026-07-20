package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/samber/lo"
)

// TaskPollingAdaptor 定义轮询所需的最小适配器接口，避免 service -> relay 的循环依赖
type TaskPollingAdaptor interface {
	Init(info *relaycommon.RelayInfo)
	FetchTask(baseURL string, key string, body map[string]any, proxy string) (*http.Response, error)
	ParseTaskResult(body []byte) (*relaycommon.TaskInfo, error)
	// AdjustBillingOnComplete 在任务到达终态（成功/失败）时由轮询循环调用。
	// 返回正数触发差额结算（补扣/退还），返回 0 保持预扣费金额不变。
	AdjustBillingOnComplete(task *model.Task, taskResult *relaycommon.TaskInfo) int
}

// GetTaskAdaptorFunc 由 main 包注入，用于获取指定平台的任务适配器。
// 打破 service -> relay -> relay/channel -> service 的循环依赖。
var GetTaskAdaptorFunc func(platform constant.TaskPlatform) TaskPollingAdaptor

// sweepTimedOutTasks 在主轮询之前独立清理超时任务。
// 每次最多处理 100 条，剩余的下个周期继续处理。
// 使用 per-task CAS (UpdateWithStatus) 防止覆盖被正常轮询已推进的任务。
func sweepTimedOutTasks(ctx context.Context) {
	if constant.TaskTimeoutMinutes <= 0 {
		return
	}
	cutoff := time.Now().Unix() - int64(constant.TaskTimeoutMinutes)*60
	tasks := model.GetTimedOutUnfinishedTasks(cutoff, 100)
	if len(tasks) == 0 {
		return
	}

	const legacyTaskCutoff int64 = 1740182400 // 2026-02-22 00:00:00 UTC
	reason := fmt.Sprintf("任务超时（%d分钟）", constant.TaskTimeoutMinutes)
	legacyReason := "任务超时（旧系统遗留任务，不进行退款，请联系管理员）"
	now := time.Now().Unix()
	timedOutCount := 0

	for _, task := range tasks {
		isLegacy := task.SubmitTime > 0 && task.SubmitTime < legacyTaskCutoff

		oldStatus := task.Status
		task.Status = model.TaskStatusFailure
		task.Progress = "100%"
		task.FinishTime = now
		if isLegacy {
			task.FailReason = legacyReason
		} else {
			task.FailReason = reason
		}

		won, err := task.UpdateWithStatus(oldStatus)
		if err != nil {
			logger.LogError(ctx, fmt.Sprintf("sweepTimedOutTasks CAS update error for task %s: %v", task.TaskID, err))
			continue
		}
		if !won {
			logger.LogInfo(ctx, fmt.Sprintf("sweepTimedOutTasks: task %s already transitioned, skip", task.TaskID))
			continue
		}
		timedOutCount++
		if !isLegacy && task.Quota != 0 {
			RefundTaskQuota(ctx, task, reason)
		}
	}

	if timedOutCount > 0 {
		logger.LogInfo(ctx, fmt.Sprintf("sweepTimedOutTasks: timed out %d tasks", timedOutCount))
	}
}

// TaskPollingLoop 主轮询循环，每 15 秒检查一次未完成的任务
func TaskPollingLoop() {
	for {
		time.Sleep(time.Duration(15) * time.Second)
		common.SysLog("任务进度轮询开始")
		ctx := context.TODO()
		sweepTimedOutTasks(ctx)
		allTasks := model.GetAllUnFinishSyncTasks(constant.TaskQueryLimit)
		platformTask := make(map[constant.TaskPlatform][]*model.Task)
		for _, t := range allTasks {
			platformTask[t.Platform] = append(platformTask[t.Platform], t)
		}
		for platform, tasks := range platformTask {
			if len(tasks) == 0 {
				continue
			}
			taskChannelM := make(map[int][]string)
			taskM := make(map[string]*model.Task)
			nullTaskIds := make([]int64, 0)
			for _, task := range tasks {
				upstreamID := task.GetUpstreamTaskID()
				if upstreamID == "" {
					// 统计失败的未完成任务
					nullTaskIds = append(nullTaskIds, task.ID)
					continue
				}
				taskM[upstreamID] = task
				taskChannelM[task.ChannelId] = append(taskChannelM[task.ChannelId], upstreamID)
			}
			if len(nullTaskIds) > 0 {
				reason := "empty upstream task id"
				for _, task := range tasks {
					if task.GetUpstreamTaskID() == "" {
						failTaskAndRefund(ctx, task, reason)
					}
				}
				logger.LogInfo(ctx, fmt.Sprintf("Fix null task_id task success: %v", nullTaskIds))
			}
			if len(taskChannelM) == 0 {
				continue
			}

			DispatchPlatformUpdate(platform, taskChannelM, taskM)
		}
		common.SysLog("任务进度轮询完成")
	}
}

// DispatchPlatformUpdate 按平台分发轮询更新
func DispatchPlatformUpdate(platform constant.TaskPlatform, taskChannelM map[int][]string, taskM map[string]*model.Task) {
	switch platform {
	case constant.TaskPlatformMidjourney:
		// MJ 轮询由其自身处理，这里预留入口
	case constant.TaskPlatformSuno:
		_ = UpdateSunoTasks(context.Background(), taskChannelM, taskM)
	default:
		if err := UpdateVideoTasks(context.Background(), platform, taskChannelM, taskM); err != nil {
			common.SysLog(fmt.Sprintf("UpdateVideoTasks fail: %s", err))
		}
	}
}

// UpdateSunoTasks 按渠道更新所有 Suno 任务
func UpdateSunoTasks(ctx context.Context, taskChannelM map[int][]string, taskM map[string]*model.Task) error {
	for channelId, taskIds := range taskChannelM {
		err := updateSunoTasks(ctx, channelId, taskIds, taskM)
		if err != nil {
			logger.LogError(ctx, fmt.Sprintf("渠道 #%d 更新异步任务失败: %s", channelId, err.Error()))
		}
	}
	return nil
}

func updateSunoTasks(ctx context.Context, channelId int, taskIds []string, taskM map[string]*model.Task) error {
	logger.LogInfo(ctx, fmt.Sprintf("渠道 #%d 未完成的任务有: %d", channelId, len(taskIds)))
	if len(taskIds) == 0 {
		return nil
	}
	ch, err := model.CacheGetChannel(channelId)
	if err != nil {
		common.SysLog(fmt.Sprintf("CacheGetChannel: %v", err))
		reason := fmt.Sprintf("channel info unavailable, channel ID: %d", channelId)
		for _, upstreamID := range taskIds {
			if t, ok := taskM[upstreamID]; ok {
				failTaskAndRefund(ctx, t, reason)
			}
		}
		return err
	}
	adaptor := GetTaskAdaptorFunc(constant.TaskPlatformSuno)
	if adaptor == nil {
		return errors.New("adaptor not found")
	}
	proxy := ch.GetSetting().Proxy
	resp, err := adaptor.FetchTask(*ch.BaseURL, ch.Key, map[string]any{
		"ids": taskIds,
	}, proxy)
	if err != nil {
		common.SysLog(fmt.Sprintf("Get Task Do req error: %v", err))
		return err
	}
	if resp.StatusCode != http.StatusOK {
		logger.LogError(ctx, fmt.Sprintf("Get Task status code: %d", resp.StatusCode))
		return fmt.Errorf("Get Task status code: %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		common.SysLog(fmt.Sprintf("Get Suno Task parse body error: %v", err))
		return err
	}
	var responseItems dto.TaskResponse[[]dto.SunoDataResponse]
	err = common.Unmarshal(responseBody, &responseItems)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("Get Suno Task parse body error2: %v, body: %s", err, string(responseBody)))
		return err
	}
	if !responseItems.IsSuccess() {
		common.SysLog(fmt.Sprintf("渠道 #%d 未完成的任务有: %d, 成功获取到任务数: %s", channelId, len(taskIds), string(responseBody)))
		return err
	}

	for _, responseItem := range responseItems.Data {
		task := taskM[responseItem.TaskID]
		if !taskNeedsUpdate(task, responseItem) {
			continue
		}

		task.Status = lo.If(model.TaskStatus(responseItem.Status) != "", model.TaskStatus(responseItem.Status)).Else(task.Status)
		task.FailReason = lo.If(responseItem.FailReason != "", responseItem.FailReason).Else(task.FailReason)
		task.SubmitTime = lo.If(responseItem.SubmitTime != 0, responseItem.SubmitTime).Else(task.SubmitTime)
		task.StartTime = lo.If(responseItem.StartTime != 0, responseItem.StartTime).Else(task.StartTime)
		task.FinishTime = lo.If(responseItem.FinishTime != 0, responseItem.FinishTime).Else(task.FinishTime)
		if responseItem.FailReason != "" || task.Status == model.TaskStatusFailure {
			logger.LogInfo(ctx, task.TaskID+" 构建失败，"+task.FailReason)
			task.Progress = "100%"
			RefundTaskQuota(ctx, task, task.FailReason)
		}
		if responseItem.Status == model.TaskStatusSuccess {
			task.Progress = "100%"
		}
		task.Data = responseItem.Data

		err = task.Update()
		if err != nil {
			common.SysLog("UpdateSunoTask task error: " + err.Error())
		}
	}
	return nil
}

// taskNeedsUpdate 检查 Suno 任务是否需要更新
func taskNeedsUpdate(oldTask *model.Task, newTask dto.SunoDataResponse) bool {
	if oldTask.SubmitTime != newTask.SubmitTime {
		return true
	}
	if oldTask.StartTime != newTask.StartTime {
		return true
	}
	if oldTask.FinishTime != newTask.FinishTime {
		return true
	}
	if string(oldTask.Status) != newTask.Status {
		return true
	}
	if oldTask.FailReason != newTask.FailReason {
		return true
	}

	if (oldTask.Status == model.TaskStatusFailure || oldTask.Status == model.TaskStatusSuccess) && oldTask.Progress != "100%" {
		return true
	}

	oldData, _ := common.Marshal(oldTask.Data)
	newData, _ := common.Marshal(newTask.Data)

	sort.Slice(oldData, func(i, j int) bool {
		return oldData[i] < oldData[j]
	})
	sort.Slice(newData, func(i, j int) bool {
		return newData[i] < newData[j]
	})

	if string(oldData) != string(newData) {
		return true
	}
	return false
}

// UpdateVideoTasks 按渠道更新所有视频任务
func UpdateVideoTasks(ctx context.Context, platform constant.TaskPlatform, taskChannelM map[int][]string, taskM map[string]*model.Task) error {
	for channelId, taskIds := range taskChannelM {
		if err := updateVideoTasks(ctx, platform, channelId, taskIds, taskM); err != nil {
			logger.LogError(ctx, fmt.Sprintf("Channel #%d failed to update video async tasks: %s", channelId, err.Error()))
		}
	}
	return nil
}

func failTaskAndRefund(ctx context.Context, task *model.Task, reason string) bool {
	if task == nil {
		return false
	}
	snap := task.Snapshot()
	if task.Status == model.TaskStatusFailure || task.Status == model.TaskStatusSuccess {
		return false
	}
	now := time.Now().Unix()
	task.Status = model.TaskStatusFailure
	task.Progress = taskcommon.ProgressComplete
	task.FailReason = reason
	if task.FinishTime == 0 {
		task.FinishTime = now
	}
	won, err := task.UpdateWithStatus(snap.Status)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("failTaskAndRefund update error for task %s: %v", task.TaskID, err))
		return false
	}
	if !won {
		logger.LogWarn(ctx, fmt.Sprintf("failTaskAndRefund: task %s already transitioned, skip refund", task.TaskID))
		return false
	}
	if task.Quota != 0 {
		RefundTaskQuota(ctx, task, reason)
	}
	return true
}

func updateVideoTasks(ctx context.Context, platform constant.TaskPlatform, channelId int, taskIds []string, taskM map[string]*model.Task) error {
	logger.LogInfo(ctx, fmt.Sprintf("Channel #%d pending video tasks: %d", channelId, len(taskIds)))
	if len(taskIds) == 0 {
		return nil
	}
	cacheGetChannel, err := model.CacheGetChannel(channelId)
	if err != nil {
		reason := fmt.Sprintf("Failed to get channel info, channel ID: %d", channelId)
		for _, upstreamID := range taskIds {
			if t, ok := taskM[upstreamID]; ok {
				failTaskAndRefund(ctx, t, reason)
			}
		}
		return fmt.Errorf("CacheGetChannel failed: %w", err)
	}
	adaptor := GetTaskAdaptorFunc(platform)
	if adaptor == nil {
		return fmt.Errorf("video adaptor not found")
	}
	info := &relaycommon.RelayInfo{}
	info.ChannelMeta = &relaycommon.ChannelMeta{
		ChannelBaseUrl: cacheGetChannel.GetBaseURL(),
	}
	info.ApiKey = cacheGetChannel.Key
	adaptor.Init(info)
	for _, taskId := range taskIds {
		if err := updateVideoSingleTask(ctx, adaptor, cacheGetChannel, taskId, taskM); err != nil {
			logger.LogError(ctx, fmt.Sprintf("Failed to update video task %s: %s", taskId, err.Error()))
		}
		// sleep 1 second between each task to avoid hitting rate limits of upstream platforms
		time.Sleep(1 * time.Second)
	}
	return nil
}

func updateVideoSingleTask(ctx context.Context, adaptor TaskPollingAdaptor, ch *model.Channel, taskId string, taskM map[string]*model.Task) error {
	baseURL := constant.ChannelBaseURLs[ch.Type]
	if ch.GetBaseURL() != "" {
		baseURL = ch.GetBaseURL()
	}
	proxy := ch.GetSetting().Proxy

	task := taskM[taskId]
	if task == nil {
		logger.LogError(ctx, fmt.Sprintf("Task %s not found in taskM", taskId))
		return fmt.Errorf("task %s not found", taskId)
	}
	key := ch.Key

	privateData := task.PrivateData
	if privateData.Key != "" {
		key = privateData.Key
	}
	resp, err := adaptor.FetchTask(baseURL, key, map[string]any{
		"task_id":                      task.GetUpstreamTaskID(),
		"action":                       task.Action,
		"channel_type":                 ch.Type,
		"tf_open_video_upstream_style": task.PrivateData.TfOpenVideoUpstreamStyle,
	}, proxy)
	if err != nil {
		return fmt.Errorf("fetchTask failed for task %s: %w", taskId, err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("readAll failed for task %s: %w", taskId, err)
	}

	logger.LogDebug(ctx, fmt.Sprintf("updateVideoSingleTask response: %s", string(responseBody)))

	snap := task.Snapshot()

	taskResult := &relaycommon.TaskInfo{}
	// try parse as TokenFactory response format
	var responseItems dto.TaskResponse[model.Task]
	if err = common.Unmarshal(responseBody, &responseItems); err == nil && responseItems.IsSuccess() {
		logger.LogDebug(ctx, fmt.Sprintf("updateVideoSingleTask parsed as TokenFactory response format: %+v", responseItems))
		t := responseItems.Data
		taskResult.TaskID = t.TaskID
		taskResult.Status = string(t.Status)
		taskResult.Url = t.ResultURLForResponse()
		taskResult.Progress = t.Progress
		taskResult.Reason = t.FailReason
		task.Data = t.Data
	} else if taskResult, err = adaptor.ParseTaskResult(responseBody); err != nil {
		return fmt.Errorf("parseTaskResult failed for task %s: %w", taskId, err)
	}

	task.Data = redactVideoResponseBody(responseBody)

	logger.LogDebug(ctx, fmt.Sprintf("updateVideoSingleTask taskResult: %+v", taskResult))

	now := time.Now().Unix()
	if taskResult.Status == "" {
		//taskResult = relaycommon.FailTaskInfo("upstream returned empty status")
		errorResult := &dto.GeneralErrorResponse{}
		if err = common.Unmarshal(responseBody, &errorResult); err == nil {
			openaiError := errorResult.TryToOpenAIError()
			if openaiError != nil {
				// 返回规范的 OpenAI 错误格式，提取错误信息，判断错误是否为任务失败
				if openaiError.Code == "429" {
					// 429 错误通常表示请求过多或速率限制，暂时不认为是任务失败，保持原状态等待下一轮轮询
					return nil
				}

				// 其他错误认为是任务失败，记录错误信息并更新任务状态
				taskResult = relaycommon.FailTaskInfo("upstream returned error")
			} else {
				// unknown error format, log original response
				logger.LogError(ctx, fmt.Sprintf("Task %s returned empty status with unrecognized error format, response: %s", taskId, string(responseBody)))
				taskResult = relaycommon.FailTaskInfo("upstream returned unrecognized message")
			}
		}
	}

	shouldRefund := false
	shouldSettle := false
	quota := task.Quota

	task.Status = model.TaskStatus(taskResult.Status)
	switch taskResult.Status {
	case model.TaskStatusSubmitted:
		task.Progress = taskcommon.ProgressSubmitted
	case model.TaskStatusQueued:
		task.Progress = taskcommon.ProgressQueued
	case model.TaskStatusInProgress:
		task.Progress = taskcommon.ProgressInProgress
		if task.StartTime == 0 {
			task.StartTime = now
		}
	case model.TaskStatusSuccess:
		if setting.ShouldCheckAliyunGuardrailVideo() && taskResult.Url != "" && !strings.HasPrefix(taskResult.Url, "data:") {
			moderationCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			complete, blocked, guardrailErr := CheckAliyunVideoGuardrail(moderationCtx, task, taskResult.Url)
			cancel()
			if guardrailErr != nil {
				logger.LogWarn(ctx, fmt.Sprintf("Aliyun video guardrail failed for task %s, allowing result: %s", task.TaskID, guardrailErr.Error()))
			} else if !complete {
				task.Status = model.TaskStatusInProgress
				task.Progress = "95%"
				break
			} else if blocked {
				task.Status = model.TaskStatusFailure
				task.Progress = taskcommon.ProgressComplete
				task.FailReason = setting.AliyunGuardrailBlockedReply
				task.PrivateData.ResultURL = ""
				taskResult.Status = string(model.TaskStatusFailure)
				taskResult.Reason = task.FailReason
				if task.FinishTime == 0 {
					task.FinishTime = now
				}
				if quota != 0 {
					shouldRefund = true
				}
				break
			}
		}
		task.Progress = taskcommon.ProgressComplete
		if task.FinishTime == 0 {
			task.FinishTime = now
		}
		if strings.HasPrefix(taskResult.Url, "data:") {
			// data: URI (e.g. Vertex base64 encoded video) — keep in Data, not in ResultURL
			task.PrivateData.ResultURL = taskcommon.BuildProxyURL(task.TaskID)
		} else if taskResult.Url != "" {
			// Direct upstream URL (e.g. Kling, Ali, Doubao, etc.)
			task.PrivateData.ResultURL = taskResult.Url
		} else {
			// No URL from adaptor — construct proxy URL using public task ID
			task.PrivateData.ResultURL = taskcommon.BuildProxyURL(task.TaskID)
		}
	case model.TaskStatusFailure:
		logger.LogJson(ctx, fmt.Sprintf("Task %s failed", taskId), task)
		task.Status = model.TaskStatusFailure
		task.Progress = taskcommon.ProgressComplete
		if task.FinishTime == 0 {
			task.FinishTime = now
		}
		task.FailReason = taskResult.Reason
		logger.LogInfo(ctx, fmt.Sprintf("Task %s failed: %s", task.TaskID, task.FailReason))
		taskResult.Progress = taskcommon.ProgressComplete
		if quota != 0 {
			shouldRefund = true
		}
	default:
		return fmt.Errorf("unknown task status %s for task %s", taskResult.Status, task.TaskID)
	}
	if taskResult.Progress != "" {
		task.Progress = taskResult.Progress
	}

	isDone := task.Status == model.TaskStatusSuccess || task.Status == model.TaskStatusFailure
	if isDone && snap.Status != task.Status {
		won, err := task.UpdateWithStatus(snap.Status)
		if err != nil {
			logger.LogError(ctx, fmt.Sprintf("UpdateWithStatus failed for task %s: %s", task.TaskID, err.Error()))
			shouldRefund = false
			shouldSettle = false
		} else if !won {
			logger.LogWarn(ctx, fmt.Sprintf("Task %s already transitioned by another process, skip billing", task.TaskID))
			shouldRefund = false
			shouldSettle = false
		} else if task.Status == model.TaskStatusSuccess {
			// 仅在本轮成功抢到「进入 SUCCESS」的迁移时做完成结算，避免轮询重复 settle/分润
			shouldSettle = true
		}
	} else if !snap.Equal(task.Snapshot()) {
		if _, err := task.UpdateWithStatus(snap.Status); err != nil {
			logger.LogError(ctx, fmt.Sprintf("Failed to update task %s: %s", task.TaskID, err.Error()))
		}
	} else {
		// No changes, skip update
		logger.LogDebug(ctx, fmt.Sprintf("No update needed for task %s", task.TaskID))
	}

	if shouldSettle {
		settleTaskBillingOnComplete(ctx, adaptor, task, taskResult)
	}
	if shouldRefund {
		RefundTaskQuota(ctx, task, task.FailReason)
	}

	return nil
}

func redactVideoResponseBody(body []byte) []byte {
	var m map[string]any
	if err := common.Unmarshal(body, &m); err != nil {
		return body
	}
	resp, _ := m["response"].(map[string]any)
	if resp != nil {
		delete(resp, "bytesBase64Encoded")
		if v, ok := resp["video"].(string); ok {
			resp["video"] = truncateBase64(v)
		}
		if vs, ok := resp["videos"].([]any); ok {
			for i := range vs {
				if vm, ok := vs[i].(map[string]any); ok {
					delete(vm, "bytesBase64Encoded")
				}
			}
		}
	}
	b, err := common.Marshal(m)
	if err != nil {
		return body
	}
	return b
}

func truncateBase64(s string) string {
	const maxKeep = 256
	if len(s) <= maxKeep {
		return s
	}
	return s[:maxKeep] + "..."
}

// settleTaskBillingOnComplete 任务完成时的统一计费调整。
// 优先级：1. adaptor.AdjustBillingOnComplete 返回正数 → 使用 adaptor 计算的额度
//
//  2. taskResult.TotalTokens > 0 → 按 token 重算
//  3. 都不满足 → 保持预扣额度不变
func settleTaskBillingOnComplete(ctx context.Context, adaptor TaskPollingAdaptor, task *model.Task, taskResult *relaycommon.TaskInfo) {
	if task == nil {
		return
	}
	hintTokens := 0
	if taskResult != nil {
		hintTokens = taskResult.TotalTokens
	}
	defer func() {
		TryPostWalletProfitShareForTaskBilledQuota(ctx, task, task.Quota, hintTokens)
	}()
	// 0. 按次计费的任务不做差额结算
	if bc := task.PrivateData.BillingContext; bc != nil && bc.PerCallBilling {
		logger.LogInfo(ctx, fmt.Sprintf("任务 %s 按次计费，跳过差额结算", task.TaskID))
		return
	}
	// 0.3 视频「按 Token 计费」模式（按量计费基础价格配置了输出价格）：
	//     以 total_tokens × 输出单价补差，并覆盖视频规格日志字段。优先级高于通用 token / 视频按秒结算。
	if SettleVideoTokenBillingOnComplete(ctx, task, taskResult) {
		return
	}
	// 0.5 上游返回 total_tokens 时按 token 结算；视频按秒任务跳过，避免覆盖视频规则价。
	if taskResult.TotalTokens > 0 && !taskPreferVideoPerSecondSettlement(task) {
		if settled := RecalculateTaskQuotaByTokens(ctx, task, taskResult.TotalTokens); settled {
			return
		}
	}

	// 1. 视频按秒规则优先按真实成片重算。
	if actualQuota, detail := recalcVideoPerSecondQuotaDetailOnComplete(task, taskResult); actualQuota > 0 {
		// 侧写：渠道 adaptor 比对提交/回包计费字段并打点（如腾讯云 Input.OutputConfig），返回值此处忽略。
		_ = adaptor.AdjustBillingOnComplete(task, taskResult)
		RecalculateTaskQuota(ctx, task, actualQuota, detail)
		return
	}

	// 2. 让 adaptor 决定最终额度
	if actualQuota := adaptor.AdjustBillingOnComplete(task, taskResult); actualQuota > 0 {
		RecalculateTaskQuota(ctx, task, actualQuota, nil)
		return
	}
	// 3. 无调整，保持预扣额度（估算值）；仍写入结算标记供前端展示「实际扣费」。
	_, markerDetail := recalcVideoPerSecondQuotaDetailOnComplete(task, taskResult)
	recordVideoTaskSettlementMarker(ctx, task, task.Quota, markerDetail)
}

// SettleTaskBillingOnFetch 用于 /v1/videos/{task_id} 查询链路下的成功结算。
// 该路径不会走后台轮询适配器，因此在状态首次进入 SUCCESS 时主动触发与轮询一致的结算优先级：
// 1) 上游 total_tokens
// 2) 视频真实元数据重算
// 3) 保持预扣（估算值）
func SettleTaskBillingOnFetch(ctx context.Context, task *model.Task, taskResult *relaycommon.TaskInfo) {
	if task == nil || taskResult == nil {
		return
	}
	hintTokens := taskResult.TotalTokens
	defer func() {
		TryPostWalletProfitShareForTaskBilledQuota(ctx, task, task.Quota, hintTokens)
	}()
	// 按次模型不做差额结算
	if bc := task.PrivateData.BillingContext; bc != nil && bc.PerCallBilling {
		return
	}
	// 视频「按 Token 计费」模式优先：以 total_tokens × 输出单价补差并覆盖视频规格日志字段。
	if SettleVideoTokenBillingOnComplete(ctx, task, taskResult) {
		return
	}
	if taskResult.TotalTokens > 0 && !taskPreferVideoPerSecondSettlement(task) {
		if settled := RecalculateTaskQuotaByTokens(ctx, task, taskResult.TotalTokens); settled {
			return
		}
	}
	if actualQuota, detail := recalcVideoPerSecondQuotaDetailOnComplete(task, taskResult); actualQuota > 0 {
		logTencentVodBillingMismatch(ctx, task, taskResult)
		RecalculateTaskQuota(ctx, task, actualQuota, detail)
		return
	}
	_, markerDetail := recalcVideoPerSecondQuotaDetailOnComplete(task, taskResult)
	recordVideoTaskSettlementMarker(ctx, task, task.Quota, markerDetail)
}

func recalcVideoPerSecondQuotaOnComplete(task *model.Task, taskResult *relaycommon.TaskInfo) int {
	quota, _ := recalcVideoPerSecondQuotaDetailOnComplete(task, taskResult)
	return quota
}

func recalcVideoPerSecondQuotaDetailOnComplete(task *model.Task, taskResult *relaycommon.TaskInfo) (int, *videoPerSecondBillingDetail) {
	if task == nil || taskResult == nil || task.Status != model.TaskStatusSuccess {
		return 0, nil
	}
	modelName := taskModelName(task)
	if strings.TrimSpace(modelName) == "" {
		return 0, nil
	}
	channelRules, chRulesOK := ratio_setting.GetChannelVideoPricingRules(task.ChannelId, modelName)
	if !chRulesOK || !ratio_setting.HasUsableVideoPerSecondRules(channelRules) {
		return 0, nil
	}
	videoURL := strings.TrimSpace(taskResult.Url)
	if videoURL == "" {
		videoURL = strings.TrimSpace(task.GetResultURL())
	}
	// 优先使用上游任务查询回包中的 resolution / duration / ratio；缺失时再 URL 探测或回退估算。
	var meta *VideoMetadata
	if m, ok := videoMetadataFromTaskCompletion(task, taskResult); ok {
		meta = m
	}
	if meta == nil && videoURL != "" {
		if probed, err := ProbeVideoMetadataFromURL(videoURL); err == nil {
			meta = probed
		}
	}
	if meta == nil {
		if sec := taskBillingSecondsEstimate(task); sec > 0 {
			meta, _ = videoMetadataFromBillingEstimate(task, sec)
		}
	}
	if meta == nil || meta.DurationSec <= 0 {
		return 0, nil
	}
	mode := detectTaskVideoBillingMode(task)
	resolutionLabel := VideoBillingResolutionLabelForTask(task, taskResult)
	match, ok := matchPerSecondPriceDetail(channelRules, mode, meta.Width, meta.Height, meta.HasAudio, resolutionLabel)
	if !ok || match.PricePerSecond <= 0 {
		return 0, nil
	}
	groupRatio := 1.0
	if task.PrivateData.BillingContext != nil && task.PrivateData.BillingContext.GroupRatio > 0 {
		groupRatio = task.PrivateData.BillingContext.GroupRatio
	}
	seconds := int(math.Ceil(meta.DurationSec))
	if seconds <= 0 {
		return 0, nil
	}
	costDisc := taskBillingContextEffectiveCostPercent(task.PrivateData.BillingContext, task.ChannelId)
	markupDisc := model.ResolveEffectiveMarkupDiscountPercentForInviteeBilling(task.UserId, task.ChannelId, modelName)
	globalPerSec := globalVideoPerSecondUSDForChannelTier(
		modelName, mode, match.Resolution, match.RuleWidth, match.RuleHeight, meta.HasAudio, match.UnifiedAudio,
	)
	effPerSec := effectiveVideoPerSecondUSD(match.PricePerSecond, globalPerSec, costDisc, markupDisc)
	rawQuota := float64(seconds) * effPerSec * common.QuotaPerUnit * groupRatio
	quota := int(math.Round(rawQuota))
	if quota <= 0 && rawQuota > 0 {
		quota = 1
	}
	channelDiscountPercent := costDisc
	upstreamSpec := resolveVideoOutputSpecFromUpstream(task, taskResult)
	detail := &videoPerSecondBillingDetail{
		Mode:                    mode,
		Seconds:                 seconds,
		Width:                   meta.Width,
		Height:                  meta.Height,
		HasAudio:                meta.HasAudio,
		Resolution:              match.Resolution,
		RuleWidth:               match.RuleWidth,
		RuleHeight:              match.RuleHeight,
		PricePerSecond:          match.PricePerSecond,
		GlobalPricePerSecond:    globalPerSec,
		EffectivePricePerSecond: effPerSec,
		MarkupDiscountPercent:   markupDisc,
		GroupRatio:              groupRatio,
		QuotaPerUnit:            common.QuotaPerUnit,
		ChannelDiscountPercent:  channelDiscountPercent,
		UnifiedAudio:            match.UnifiedAudio,
		CappedToMaxTier:         match.CappedToMaxTier,
		Ratio:                   upstreamSpec.Ratio,
	}
	if display := common.FormatVideoResolutionLabel(resolutionLabel); display != "" {
		detail.Resolution = display
	} else if label := strings.TrimSpace(upstreamSpec.Resolution); label != "" {
		detail.Resolution = common.FormatVideoResolutionLabel(label)
		if detail.Resolution == "" {
			detail.Resolution = label
		}
	}
	return quota, detail
}

func taskBillingSecondsEstimate(task *model.Task) int {
	if task == nil {
		return 0
	}
	if bc := task.PrivateData.BillingContext; bc != nil && bc.OtherRatios != nil {
		if sec := int(math.Ceil(bc.OtherRatios["seconds"])); sec > 0 {
			return sec
		}
	}
	if input := strings.TrimSpace(task.Properties.Input); input != "" {
		var req map[string]any
		if err := common.Unmarshal([]byte(input), &req); err == nil {
			if params, _ := req["parameters"].(map[string]any); params != nil {
				if d := metadataToFloat64(params["duration"]); d > 0 {
					return int(math.Ceil(d))
				}
			}
		}
	}
	return 0
}

func videoMetadataFromBillingEstimate(task *model.Task, seconds int) (*VideoMetadata, bool) {
	if task == nil || seconds <= 0 {
		return nil, false
	}
	width, height := 1280, 720
	if m, ok := extractVideoMetadataFromTaskData(task); ok {
		if m.Width > 0 {
			width = m.Width
		}
		if m.Height > 0 {
			height = m.Height
		}
	}
	return &VideoMetadata{
		DurationSec: float64(seconds),
		Width:       width,
		Height:      height,
		HasAudio:    false,
	}, true
}

func extractVideoMetadataFromTaskData(task *model.Task) (*VideoMetadata, bool) {
	if task == nil || len(task.Data) == 0 {
		return nil, false
	}
	var payload map[string]any
	if err := common.Unmarshal(task.Data, &payload); err != nil {
		return nil, false
	}
	return extractVideoMetadataFromMap(payload)
}

func toFloat64(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case int:
		return float64(x)
	case int64:
		return float64(x)
	case int32:
		return float64(x)
	case uint:
		return float64(x)
	case uint64:
		return float64(x)
	case uint32:
		return float64(x)
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
		if err == nil {
			return f
		}
	}
	return 0
}

func toInt(v any) int {
	switch x := v.(type) {
	case int:
		return x
	case int64:
		return int(x)
	case int32:
		return int(x)
	case uint:
		return int(x)
	case uint64:
		return int(x)
	case uint32:
		return int(x)
	case float64:
		return int(x)
	case float32:
		return int(x)
	case string:
		i, err := strconv.Atoi(strings.TrimSpace(x))
		if err == nil {
			return i
		}
	}
	return 0
}

func detectTaskVideoBillingMode(task *model.Task) string {
	var req relaycommon.TaskSubmitReq
	if err := common.UnmarshalJsonStr(task.Properties.Input, &req); err != nil {
		return "text_to_video"
	}
	return relaycommon.DetectVideoBillingMode(&req)
}

type videoPerSecondPriceMatch struct {
	Resolution      string
	RuleWidth       int
	RuleHeight      int
	PricePerSecond  float64
	UnifiedAudio    bool
	CappedToMaxTier bool
}

type videoPerSecondBillingDetail struct {
	Mode                    string
	Seconds                 int
	Width                   int
	Height                  int
	HasAudio                bool
	Resolution              string
	Ratio                   string
	ResolutionFromRequest   bool
	RuleWidth               int
	RuleHeight              int
	PricePerSecond          float64
	GlobalPricePerSecond    float64
	EffectivePricePerSecond float64
	MarkupDiscountPercent   float64
	GroupRatio              float64
	QuotaPerUnit            float64
	ChannelDiscountPercent  float64
	UnifiedAudio            bool
	CappedToMaxTier         bool
}

// taskPreferVideoPerSecondSettlement 该任务是否应按视频按秒规则结算（避免误走文本 token 重算）。
func taskPreferVideoPerSecondSettlement(task *model.Task) bool {
	if task == nil {
		return false
	}
	modelName := strings.TrimSpace(taskModelName(task))
	if modelName == "" {
		return false
	}
	if rules, ok := ratio_setting.GetChannelVideoPricingRules(task.ChannelId, modelName); ok && ratio_setting.HasUsableVideoPerSecondRules(rules) {
		return true
	}
	if rules, ok := ratio_setting.GetVideoPricingRules(modelName); ok && ratio_setting.HasUsableVideoPerSecondRules(rules) {
		return true
	}
	return false
}

func matchPerSecondPrice(r ratio_setting.VideoPricingRules, mode string, width, height int, hasAudio bool) (float64, bool) {
	match, ok := matchPerSecondPriceDetail(r, mode, width, height, hasAudio, "")
	if !ok {
		return 0, false
	}
	return match.PricePerSecond, true
}

// MatchVideoPerSecondUnitPrice 按 resolution 标识优先匹配按秒单价，再回退像素档位匹配。
func MatchVideoPerSecondUnitPrice(r ratio_setting.VideoPricingRules, mode string, width, height int, hasAudio bool, resolutionLabel string) (*videoPerSecondPriceMatch, bool) {
	return matchPerSecondPriceDetail(r, mode, width, height, hasAudio, resolutionLabel)
}

// VideoPerSecondChannelUnitPrice 返回渠道按秒单价（美元/秒）。
func VideoPerSecondChannelUnitPrice(r ratio_setting.VideoPricingRules, mode string, width, height int, hasAudio bool, resolutionLabel string) (float64, bool) {
	match, ok := matchPerSecondPriceDetail(r, mode, width, height, hasAudio, resolutionLabel)
	if !ok || match == nil {
		return 0, false
	}
	return match.PricePerSecond, true
}

func matchPerSecondPriceDetail(r ratio_setting.VideoPricingRules, mode string, width, height int, hasAudio bool, resolutionLabel string) (*videoPerSecondPriceMatch, bool) {
	var rows []ratio_setting.VideoResolutionAudioPriceRule
	switch mode {
	case "image_to_video":
		rows = r.ImageToVideoPerSecond
	case "video_to_video":
		rows = r.VideoToVideoPerSecond
	default:
		rows = r.TextToVideoPerSecond
	}
	if len(rows) == 0 {
		return nil, false
	}
	if label := strings.TrimSpace(resolutionLabel); label != "" {
		if match, ok := matchPerSecondPriceRowByLabel(rows, label, width, height, hasAudio, true); ok {
			return match, true
		}
		if match, ok := matchPerSecondPriceRowByLabel(rows, label, width, height, hasAudio, false); ok {
			return match, true
		}
	}
	targetLong, targetShort := normalizeVideoResolutionSides(width, height)
	targetRatio := targetVideoResolutionRatio(width, height)
	best, capped := matchPerSecondPriceRow(rows, targetLong, targetShort, targetRatio, hasAudio, true)
	if best < 0 && !hasPerSecondPriceRowsForAudio(rows, hasAudio) {
		best, capped = matchPerSecondPriceRow(rows, targetLong, targetShort, targetRatio, hasAudio, false)
	}
	if best < 0 {
		return nil, false
	}
	row := rows[best]
	rw, rh, _ := parseVideoResolutionFlexibleForRatio(row.Resolution, targetRatio)
	return &videoPerSecondPriceMatch{
		Resolution:      row.Resolution,
		RuleWidth:       rw,
		RuleHeight:      rh,
		PricePerSecond:  row.Price,
		UnifiedAudio:    hasSamePerSecondPriceForAudio(rows, row),
		CappedToMaxTier: capped,
	}, true
}

func matchPerSecondPriceRow(rows []ratio_setting.VideoResolutionAudioPriceRule, targetLong, targetShort int, targetRatio float64, hasAudio bool, requireAudioMatch bool) (int, bool) {
	best := -1
	bestPixels := int(^uint(0) >> 1)
	maxTier := -1
	maxTierPixels := 0
	for i := range rows {
		row := rows[i]
		if row.Price <= 0 {
			continue
		}
		if requireAudioMatch && row.HasAudio != hasAudio {
			continue
		}
		rw, rh, ok := parseVideoResolutionFlexibleForRatio(row.Resolution, targetRatio)
		if !ok {
			continue
		}
		p := rw * rh
		if p <= 0 {
			continue
		}
		if p > maxTierPixels {
			maxTierPixels = p
			maxTier = i
		}
		ruleLong, ruleShort := normalizeVideoResolutionSides(rw, rh)
		if ruleLong >= targetLong && ruleShort >= targetShort {
			if p < bestPixels {
				bestPixels = p
				best = i
			}
		}
	}
	if best < 0 {
		return maxTier, maxTier >= 0
	}
	return best, false
}

func normalizeResolutionLabelForMatch(raw string) string {
	label := common.FormatVideoResolutionLabel(strings.TrimSpace(raw))
	if label != "" {
		return strings.ToLower(label)
	}
	return strings.ToLower(strings.TrimSpace(raw))
}

func matchPerSecondPriceRowByLabel(rows []ratio_setting.VideoResolutionAudioPriceRule, label string, width, height int, hasAudio, requireAudioMatch bool) (*videoPerSecondPriceMatch, bool) {
	want := normalizeResolutionLabelForMatch(label)
	if want == "" {
		return nil, false
	}
	targetRatio := targetVideoResolutionRatio(width, height)
	var best *videoPerSecondPriceMatch
	bestPixels := int(^uint(0) >> 1)
	for i := range rows {
		row := rows[i]
		if row.Price <= 0 {
			continue
		}
		if requireAudioMatch && row.HasAudio != hasAudio {
			continue
		}
		if normalizeResolutionLabelForMatch(row.Resolution) != want {
			continue
		}
		rw, rh, ok := parseVideoResolutionFlexibleForRatio(row.Resolution, targetRatio)
		if !ok {
			rw, rh, ok = parseVideoResolutionFlexible(row.Resolution)
		}
		if !ok {
			continue
		}
		pixels := rw * rh
		if best == nil || pixels < bestPixels {
			bestPixels = pixels
			best = &videoPerSecondPriceMatch{
				Resolution:     row.Resolution,
				RuleWidth:      rw,
				RuleHeight:     rh,
				PricePerSecond: row.Price,
				UnifiedAudio:   hasSamePerSecondPriceForAudio(rows, row),
			}
		}
	}
	if best != nil {
		return best, true
	}
	return nil, false
}

func hasPerSecondPriceRowsForAudio(rows []ratio_setting.VideoResolutionAudioPriceRule, hasAudio bool) bool {
	for _, row := range rows {
		if row.Price > 0 && row.HasAudio == hasAudio {
			return true
		}
	}
	return false
}

func hasSamePerSecondPriceForAudio(rows []ratio_setting.VideoResolutionAudioPriceRule, row ratio_setting.VideoResolutionAudioPriceRule) bool {
	for _, other := range rows {
		if other.Resolution == row.Resolution &&
			other.HasAudio != row.HasAudio &&
			other.Price == row.Price {
			return true
		}
	}
	return false
}

func normalizeVideoResolutionSides(width, height int) (longSide, shortSide int) {
	if width >= height {
		return width, height
	}
	return height, width
}

func targetVideoResolutionRatio(width, height int) float64 {
	longSide, shortSide := normalizeVideoResolutionSides(width, height)
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

func parseVideoResolutionFlexibleForRatio(v string, ratio float64) (int, int, bool) {
	s := strings.ToLower(strings.TrimSpace(v))
	parts := strings.Split(s, "x")
	if len(parts) == 2 {
		w, ew := strconv.Atoi(strings.TrimSpace(parts[0]))
		h, eh := strconv.Atoi(strings.TrimSpace(parts[1]))
		if ew == nil && eh == nil && w > 0 && h > 0 {
			return w, h, true
		}
	}
	shortSide := 0
	switch s {
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

func parseVideoResolutionFlexible(v string) (int, int, bool) {
	s := strings.ToLower(strings.TrimSpace(v))
	parts := strings.Split(s, "x")
	if len(parts) == 2 {
		w, ew := strconv.Atoi(strings.TrimSpace(parts[0]))
		h, eh := strconv.Atoi(strings.TrimSpace(parts[1]))
		if ew == nil && eh == nil && w > 0 && h > 0 {
			return w, h, true
		}
	}
	switch s {
	case "480p":
		return 854, 480, true
	case "540p":
		return 960, 540, true
	case "720p":
		return 1280, 720, true
	case "1080p":
		return 1920, 1080, true
	case "2k":
		return 2560, 1440, true
	case "4k":
		return 3840, 2160, true
	default:
		return 0, 0, false
	}
}
