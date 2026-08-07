package relay

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel/aliyunasr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

// resolveASRAsyncAudioURL 解析异步转写音频地址：
// 优先使用请求中的 audio_url/file_url；若缺失且为 multipart，则将 file 上传到操练场附件库后取公网 URL。
func resolveASRAsyncAudioURL(c *gin.Context, request *dto.ASRTaskSubmitRequest) (string, error) {
	fileURL := ""
	if request != nil {
		fileURL = strings.TrimSpace(request.GetAudioURL())
	}
	if fileURL == "" && strings.HasPrefix(c.ContentType(), "multipart/form-data") {
		form, err := common.ParseMultipartFormReusable(c)
		if err != nil {
			return "", fmt.Errorf("解析 multipart 表单失败: %w", err)
		}
		fileURL = firstASRMultipartFormValue(form, "audio_url", "file_url", "url")
		if fileURL == "" {
			fileHeaders := form.File["file"]
			if len(fileHeaders) == 0 {
				return "", errors.New("异步转写需提供 audio_url，或通过 multipart file 上传音频文件")
			}
			uploadedURL, uploadErr := aliyunasr.UploadPlaygroundAudioFile(c, fileHeaders[0])
			if uploadErr != nil {
				return "", uploadErr
			}
			fileURL = uploadedURL
		}
	}
	if fileURL == "" {
		return "", errors.New("异步转写需提供 audio_url（公网可访问的音频地址），或通过 multipart file 上传音频文件")
	}
	if !strings.HasPrefix(fileURL, "http://") && !strings.HasPrefix(fileURL, "https://") {
		return "", errors.New("audio_url 必须是 http/https 可公开访问的音频地址")
	}
	return fileURL, nil
}

func firstASRMultipartFormValue(form *multipart.Form, keys ...string) string {
	if form == nil {
		return ""
	}
	for _, key := range keys {
		if values, ok := form.Value[key]; ok && len(values) > 0 {
			if v := strings.TrimSpace(values[0]); v != "" {
				return v
			}
		}
	}
	return ""
}

// SubmitASRTask 提交阿里云 ASR 异步转写任务（POST /v1/audio/transcriptions/async）。
//
// 计费：提交时预扣 60 秒费用；成功取结果后按 usage.duration 补差价；失败退还预扣。
// 提交后由后台 AsrTaskPollingLoop 定时轮询上游并结算写日志，无需用户主动查询。
func SubmitASRTask(c *gin.Context, info *relaycommon.RelayInfo, request *dto.ASRTaskSubmitRequest, proxy string) *types.TokenFactoryError {
	info.InitChannelMeta(c)

	if info.ChannelType != constant.ChannelTypeAliASRAsync {
		return types.NewErrorWithStatusCode(
			fmt.Errorf("模型 %s 未配置阿里云 ASR 异步渠道（当前渠道类型 %d），请在渠道管理中为异步转写模型配置对应渠道", info.OriginModelName, info.ChannelType),
			types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}

	fileURL, resolveErr := resolveASRAsyncAudioURL(c, request)
	if resolveErr != nil {
		return types.NewErrorWithStatusCode(
			resolveErr,
			types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}

	if err := helper.ModelMappedHelper(c, info, request); err != nil {
		return types.NewError(err, types.ErrorCodeChannelModelMappedError, types.ErrOptionWithSkipRetry())
	}

	priceData, err := helper.ModelPriceHelperASR(c, info, aliyunasr.AsyncPreConsumeSeconds)
	if err != nil {
		return types.NewError(err, types.ErrorCodeModelPriceError, types.ErrOptionWithSkipRetry())
	}

	if strings.TrimSpace(info.ChannelBaseUrl) == "" {
		return types.NewErrorWithStatusCode(
			errors.New("阿里云 ASR 渠道未配置上游基础地址（Base URL），请联系管理员"),
			types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}

	// 预扣 60 秒费用（强制全额预扣，禁用信任旁路）
	if info.Billing == nil && !priceData.FreeModel && priceData.QuotaToPreConsume > 0 {
		info.ForcePreConsume = true
		if tfErr := service.PreConsumeBilling(c, priceData.QuotaToPreConsume, info); tfErr != nil {
			return tfErr
		}
	}
	preConsumed := 0
	if info.Billing != nil {
		preConsumed = info.Billing.GetPreConsumedQuota()
	}
	// 提交失败时退还预扣
	defer func() {
		if info.Billing != nil {
			info.Billing.Refund(c)
		}
	}()

	taskResp, rawBody, err := aliyunasr.SubmitAsyncTask(info.ChannelBaseUrl, info.ApiKey, proxy, request.Model, fileURL)
	if err != nil {
		return types.NewErrorWithStatusCode(
			fmt.Errorf("异步任务提交失败: %w", err),
			types.ErrorCodeDoRequestFailed, http.StatusBadGateway)
	}

	task := &model.AsrTask{
		TaskID:         model.NewAsrTaskID(),
		UpstreamTaskID: taskResp.Output.TaskID,
		UserID:         info.UserId,
		TokenID:        info.TokenId,
		ChannelID:      info.ChannelId,
		Model:          info.OriginModelName,
		AudioURL:       fileURL,
		Status:         dto.ASRTaskStatusPending,
		Quota:          preConsumed,
	}
	if taskResp.Output.TaskStatus == aliyunasr.AliASRTaskStatusRunning {
		task.Status = dto.ASRTaskStatusRunning
	}
	if err := task.Insert(); err != nil {
		return types.NewError(fmt.Errorf("保存异步任务失败: %w, 上游响应: %s", err, string(rawBody)), types.ErrorCodeInvalidRequest)
	}

	// 提交成功：取消 defer 退款（预扣保留至结果结算）
	if info.Billing != nil {
		// 通过 Settle(预扣额度) 把会话标记为已结算，阻止 defer Refund
		_ = info.Billing.Settle(preConsumed)
		info.Billing = nil
	}

	c.JSON(http.StatusOK, dto.ASRTaskSubmitResponse{
		TaskID:    task.TaskID,
		Status:    task.Status,
		Model:     task.Model,
		CreatedAt: task.CreatedAt,
	})
	return nil
}

// FetchASRTaskResult 查询异步转写任务结果（GET /v1/audio/transcriptions/async/{task_id}）。
// 终态直接读库；未完成时再查一次上游（与后台轮询互补）。
func FetchASRTaskResult(c *gin.Context, taskID string) *types.TokenFactoryError {
	userID := common.GetContextKeyInt(c, constant.ContextKeyUserId)
	task, err := model.GetAsrTaskByTaskID(taskID, userID)
	if err != nil {
		return types.NewErrorWithStatusCode(
			fmt.Errorf("任务不存在: %s", taskID),
			types.ErrorCodeInvalidRequest, http.StatusNotFound, types.ErrOptionWithSkipRetry())
	}

	switch task.Status {
	case dto.ASRTaskStatusSucceeded:
		writeASRFetchResponse(c, task, "")
		return nil
	case dto.ASRTaskStatusFailed:
		writeASRFetchResponse(c, task, task.FailReason)
		return nil
	}

	if err := pollAndSettleASRTask(c.Request.Context(), task); err != nil {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("asr fetch poll task %s: %v", task.TaskID, err))
	}
	// 重新读库，避免并发后台轮询已推进状态
	if refreshed, err := model.GetAsrTaskByTaskID(taskID, userID); err == nil {
		task = refreshed
	}
	writeASRFetchResponse(c, task, task.FailReason)
	return nil
}

// AsrTaskPollingLoop 后台轮询未完成的阿里云 ASR 异步任务（对齐视频 TaskPollingLoop）。
// 成功后直接结算并写消费日志，无需等待用户主动查询。
func AsrTaskPollingLoop() {
	for {
		time.Sleep(15 * time.Second)
		common.SysLog("ASR 异步任务进度轮询开始")
		ctx := context.TODO()
		sweepTimedOutAsrTasks(ctx)
		tasks := model.GetUnfinishedAsrTasks(constant.TaskQueryLimit)
		if len(tasks) == 0 {
			common.SysLog("ASR 异步任务进度轮询完成（无待处理任务）")
			continue
		}
		for _, task := range tasks {
			if task == nil {
				continue
			}
			if err := pollAndSettleASRTask(ctx, task); err != nil {
				logger.LogError(ctx, fmt.Sprintf("ASR 轮询任务 %s 失败: %v", task.TaskID, err))
			}
			time.Sleep(1 * time.Second)
		}
		common.SysLog(fmt.Sprintf("ASR 异步任务进度轮询完成（处理 %d 条）", len(tasks)))
	}
}

func sweepTimedOutAsrTasks(ctx context.Context) {
	if constant.TaskTimeoutMinutes <= 0 {
		return
	}
	cutoff := time.Now().Unix() - int64(constant.TaskTimeoutMinutes)*60
	tasks := model.GetTimedOutUnfinishedAsrTasks(cutoff, 100)
	if len(tasks) == 0 {
		return
	}
	reason := fmt.Sprintf("任务超时（%d分钟）", constant.TaskTimeoutMinutes)
	timedOut := 0
	for _, task := range tasks {
		if task == nil {
			continue
		}
		if err := failASRTaskAndRefund(ctx, task, reason); err != nil {
			logger.LogError(ctx, fmt.Sprintf("ASR 超时清理任务 %s 失败: %v", task.TaskID, err))
			continue
		}
		timedOut++
	}
	if timedOut > 0 {
		logger.LogInfo(ctx, fmt.Sprintf("sweepTimedOutAsrTasks: timed out %d tasks", timedOut))
	}
}

// pollAndSettleASRTask 查询上游任务状态；成功则下载结果并结算写日志，失败则退款。
func pollAndSettleASRTask(ctx context.Context, task *model.AsrTask) error {
	if task == nil {
		return nil
	}
	if task.Status == dto.ASRTaskStatusSucceeded || task.Status == dto.ASRTaskStatusFailed {
		return nil
	}
	if strings.TrimSpace(task.UpstreamTaskID) == "" {
		return failASRTaskAndRefund(ctx, task, "empty upstream task id")
	}

	channelModel, err := model.GetChannelById(task.ChannelID, true)
	if err != nil {
		return fmt.Errorf("获取任务渠道失败: %w", err)
	}
	baseURL := strings.TrimSuffix(channelModel.GetBaseURL(), "/")
	if baseURL == "" {
		return failASRTaskAndRefund(ctx, task, "任务渠道未配置上游基础地址（Base URL）")
	}
	proxy := channelModel.GetSetting().Proxy
	apiKey := strings.Split(channelModel.Key, "\n")[0]

	taskResp, rawBody, err := aliyunasr.FetchAsyncTask(baseURL, apiKey, proxy, task.UpstreamTaskID)
	if err != nil {
		return fmt.Errorf("查询上游任务失败: %w", err)
	}

	switch taskResp.Output.TaskStatus {
	case aliyunasr.AliASRTaskStatusPending:
		return nil
	case aliyunasr.AliASRTaskStatusRunning:
		_ = task.MarkRunning()
		task.Status = dto.ASRTaskStatusRunning
		return nil
	case aliyunasr.AliASRTaskStatusFailed, aliyunasr.AliASRTaskStatusCanceled:
		reason := taskResp.Output.FailReason()
		if reason == "" {
			reason = fmt.Sprintf("上游任务失败: %s", string(rawBody))
		}
		return failASRTaskAndRefund(ctx, task, reason)
	case aliyunasr.AliASRTaskStatusSucceeded:
		return settleASRTaskSuccess(ctx, task, taskResp, channelModel, proxy)
	default:
		return fmt.Errorf("上游返回未知任务状态 %s: %s", taskResp.Output.TaskStatus, string(rawBody))
	}
}

func failASRTaskAndRefund(ctx context.Context, task *model.AsrTask, reason string) error {
	if task == nil {
		return nil
	}
	if task.Status == dto.ASRTaskStatusSucceeded || task.Status == dto.ASRTaskStatusFailed {
		return nil
	}
	won, err := task.MarkFailed(reason)
	if err != nil {
		return fmt.Errorf("标记失败状态失败: %w", err)
	}
	if !won {
		return nil
	}

	c, billingInfo, prepErr := prepareASRBillingContext(task, nil)
	if prepErr != nil {
		logger.LogError(ctx, fmt.Sprintf("ASR 失败日志/退款准备计费上下文失败 task=%s: %v", task.TaskID, prepErr))
	} else {
		// 上游失败原因写入使用日志（错误类型），便于在日志页排查
		apiErr := types.NewOpenAIError(
			fmt.Errorf("ASR 异步任务失败: %s", reason),
			types.ErrorCodeBadResponse,
			http.StatusBadGateway,
		)
		service.RecordASRErrorLog(c, billingInfo, apiErr, task.TaskID)
	}

	if task.Quota <= 0 {
		return nil
	}
	preConsumed := task.Quota
	if prepErr != nil {
		return nil
	}
	service.RefundASRPreConsumedQuota(c, billingInfo, preConsumed, task.TaskID, reason)
	task.Quota = 0
	_ = model.DB.Model(&model.AsrTask{}).Where("id = ?", task.ID).Update("quota", 0).Error
	return nil
}

// settleASRTaskSuccess 任务成功：下载识别结果 → 按 usage.duration 结算差额 → 写日志。
func settleASRTaskSuccess(ctx context.Context, task *model.AsrTask, taskResp *aliyunasr.ASRTaskResponse, channelModel *model.Channel, proxy string) error {
	transcriptionURL := taskResp.Output.ResolveTranscriptionURL()
	if transcriptionURL == "" {
		return fmt.Errorf("上游任务成功但未返回 transcription_url (request_id: %s)", taskResp.RequestID)
	}

	result, err := aliyunasr.DownloadTranscriptionResult(transcriptionURL, proxy)
	if err != nil {
		return fmt.Errorf("下载识别结果失败: %w", err)
	}
	text, fileSeconds := aliyunasr.MergeTranscriptsText(result)
	if strings.TrimSpace(text) == "" {
		return errors.New("识别结果文件无有效文本内容")
	}

	// 优先使用查询响应 usage.duration，其次结果文件时长，兜底 1 秒
	seconds := taskResp.Usage.AudioSeconds()
	if seconds <= 0 {
		seconds = fileSeconds
	}
	if seconds <= 0 {
		seconds = 1
	}

	c, billingInfo, tfErr := prepareASRBillingContext(task, channelModel)
	if tfErr != nil {
		return tfErr
	}
	if _, err := helper.ModelPriceHelperASR(c, billingInfo, seconds); err != nil {
		return fmt.Errorf("计算 ASR 价格失败: %w", err)
	}
	actualQuota := service.ComputeASRQuota(c, billingInfo, seconds)
	preConsumed := task.Quota // 提交预扣额度，必须在占位更新前保存
	billingInfo.FinalPreConsumedQuota = preConsumed

	won, err := task.TryMarkSucceededAndBilled(text, seconds, actualQuota)
	if err != nil {
		return fmt.Errorf("更新任务状态失败: %w", err)
	}
	if !won {
		logger.LogInfo(ctx, fmt.Sprintf("ASR 任务 %s 已由其他协程结算，跳过", task.TaskID))
		return nil
	}

	if settleErr := service.PostASRConsumeQuota(c, billingInfo, seconds, task.TaskID, fmt.Sprintf("异步任务 %s", task.TaskID)); settleErr != nil {
		_ = task.ResetBilledAt()
		return fmt.Errorf("ASR 结算失败: %w", settleErr)
	}
	_ = task.UpdateSettledQuota(actualQuota)
	logger.LogInfo(ctx, fmt.Sprintf("ASR 异步任务结算成功 task=%s seconds=%.2f quota=%d", task.TaskID, seconds, actualQuota))
	return nil
}

// prepareASRBillingContext 为后台轮询/用户查询构造计费用 gin.Context 与 RelayInfo。
func prepareASRBillingContext(task *model.AsrTask, channelModel *model.Channel) (*gin.Context, *relaycommon.RelayInfo, error) {
	if task == nil {
		return nil, nil, errors.New("asr task is nil")
	}
	var err error
	if channelModel == nil {
		channelModel, err = model.GetChannelById(task.ChannelID, true)
		if err != nil {
			return nil, nil, fmt.Errorf("获取任务渠道失败: %w", err)
		}
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest(http.MethodGet, "/v1/audio/transcriptions/async/"+task.TaskID, nil)
	c.Request = req

	common.SetContextKey(c, constant.ContextKeyUserId, task.UserID)
	common.SetContextKey(c, constant.ContextKeyTokenId, task.TokenID)
	common.SetContextKey(c, constant.ContextKeyOriginalModel, task.Model)
	common.SetContextKey(c, constant.ContextKeyChannelId, task.ChannelID)
	common.SetContextKey(c, constant.ContextKeyRequestStartTime, time.Unix(task.CreatedAt, 0))

	if userCache, userErr := model.GetUserCache(task.UserID); userErr == nil && userCache != nil {
		userCache.WriteContext(c)
		common.SetContextKey(c, constant.ContextKeyUsingGroup, userCache.Group)
	}
	if task.TokenID > 0 {
		if tok, tokErr := model.GetTokenById(task.TokenID); tokErr == nil && tok != nil {
			common.SetContextKey(c, constant.ContextKeyTokenId, tok.Id)
			common.SetContextKey(c, constant.ContextKeyTokenKey, tok.Key)
			c.Set("token_name", tok.Name)
			if tok.Group != "" {
				common.SetContextKey(c, constant.ContextKeyTokenGroup, tok.Group)
				common.SetContextKey(c, constant.ContextKeyUsingGroup, tok.Group)
			}
		}
	}

	if setupErr := middleware.SetupContextForSelectedChannel(c, channelModel, task.Model); setupErr != nil {
		return nil, nil, setupErr
	}

	billingInfo, err := relaycommon.GenRelayInfo(c, types.RelayFormatOpenAIAudio, nil, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("构造 RelayInfo 失败: %w", err)
	}
	billingInfo.InitChannelMeta(c)
	billingInfo.OriginModelName = task.Model
	billingInfo.UserId = task.UserID
	billingInfo.TokenId = task.TokenID
	billingInfo.RelayMode = relayconstant.RelayModeAudioTranscriptionAsyncFetch
	billingInfo.FinalPreConsumedQuota = task.Quota
	billingInfo.StartTime = time.Unix(task.CreatedAt, 0)
	if task.TokenID > 0 {
		if tok, tokErr := model.GetTokenById(task.TokenID); tokErr == nil && tok != nil {
			billingInfo.TokenId = tok.Id
			billingInfo.TokenKey = tok.Key
		}
	}
	return c, billingInfo, nil
}

func writeASRFetchResponse(c *gin.Context, task *model.AsrTask, errMsg string) {
	c.JSON(http.StatusOK, dto.ASRTaskFetchResponse{
		TaskID:     task.TaskID,
		Status:     task.Status,
		Model:      task.Model,
		Text:       task.ResultText,
		Duration:   task.AudioSeconds,
		Error:      errMsg,
		CreatedAt:  task.CreatedAt,
		FinishedAt: task.FinishedAt,
	})
}
