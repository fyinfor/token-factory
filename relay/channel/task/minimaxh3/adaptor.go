package minimaxh3

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

const contextKeyNativeBody = "minimax_h3_native_body"

type TaskAdaptor struct {
	taskcommon.BaseBilling
	ChannelType int
	apiKey      string
	baseURL     string
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.ChannelType = info.ChannelType
	a.baseURL = info.ChannelBaseUrl
	a.apiKey = info.ApiKey
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	var probe struct {
		Model   string `json:"model"`
		Content []any  `json:"content"`
	}
	if err := common.UnmarshalBodyReusable(c, &probe); err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	if len(probe.Content) > 0 {
		return a.validateNativeRequest(c, info)
	}
	if taskErr := relaycommon.ValidateBasicTaskRequest(c, info, constant.TaskActionGenerate); taskErr != nil {
		return taskErr
	}
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	if _, err := convertToRequestPayload(&req, info); err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	return nil
}

func (a *TaskAdaptor) validateNativeRequest(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return service.TaskErrorWrapperLocal(err, "read_body_failed", http.StatusBadRequest)
	}
	body, err := storage.Bytes()
	if err != nil {
		return service.TaskErrorWrapperLocal(err, "read_body_failed", http.StatusBadRequest)
	}

	var native VideoGenerationV2Req
	if err := common.Unmarshal(body, &native); err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	if strings.TrimSpace(native.Model) == "" {
		native.Model = strings.TrimSpace(info.OriginModelName)
	}
	if info != nil && info.UseRelayTaskUpstreamModel() && strings.TrimSpace(info.UpstreamModelName) != "" {
		native.Model = info.UpstreamModelName
	}
	if err := ValidateVideoGenerationV2Req(&native); err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}

	c.Set(contextKeyNativeBody, body)
	relaycommon.SetTaskPersistedInput(c, string(body))

	taskReq := taskSubmitFromNative(&native)
	action := relaycommon.ResolveTaskActionToStore(info, constant.TaskActionGenerate, &taskReq)
	relaycommon.StoreTaskRequest(c, info, action, taskReq)
	return nil
}

func (a *TaskAdaptor) BuildRequestURL(_ *relaycommon.RelayInfo) (string, error) {
	return SubmitURL(a.baseURL), nil
}

func (a *TaskAdaptor) BuildRequestHeader(_ *gin.Context, req *http.Request, _ *relaycommon.RelayInfo) error {
	ApplyAuthHeaders(req.Header, a.apiKey)
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	if raw, ok := c.Get(contextKeyNativeBody); ok {
		body, ok := raw.([]byte)
		if !ok || len(body) == 0 {
			return nil, fmt.Errorf("native request body is empty")
		}
		payload, err := a.normalizeNativeBody(body, info)
		if err != nil {
			return nil, err
		}
		data, err := common.Marshal(payload)
		if err != nil {
			return nil, errors.Wrap(err, "marshal minimax h3 native request failed")
		}
		logger.LogJson(c, "minimax h3 request body", payload)
		return bytes.NewReader(data), nil
	}

	taskReq, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, errors.Wrap(err, "get_task_request_failed")
	}
	payload, err := convertToRequestPayload(&taskReq, info)
	if err != nil {
		return nil, errors.Wrap(err, "convert_to_minimax_h3_request_failed")
	}
	logger.LogJson(c, "minimax h3 request body", payload)
	data, err := common.Marshal(payload)
	if err != nil {
		return nil, errors.Wrap(err, "marshal minimax h3 request failed")
	}
	return bytes.NewReader(data), nil
}

func (a *TaskAdaptor) normalizeNativeBody(body []byte, info *relaycommon.RelayInfo) (*VideoGenerationV2Req, error) {
	var payload VideoGenerationV2Req
	if err := common.Unmarshal(body, &payload); err != nil {
		return nil, errors.Wrap(err, "unmarshal native minimax h3 request failed")
	}
	if info != nil && info.UseRelayTaskUpstreamModel() && strings.TrimSpace(info.UpstreamModelName) != "" {
		payload.Model = info.UpstreamModelName
	}
	if err := ValidateVideoGenerationV2Req(&payload); err != nil {
		return nil, err
	}
	return &payload, nil
}

func (a *TaskAdaptor) EstimateBilling(c *gin.Context, _ *relaycommon.RelayInfo) map[string]float64 {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil
	}
	duration := req.Duration
	if duration <= 0 {
		duration = DefaultDuration
	}
	return map[string]float64{"seconds": float64(duration)}
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
		return
	}
	_ = resp.Body.Close()

	decodedBody := taskcommon.DecodeBase64Response(responseBody)

	var createResp VideoGenerationV2Resp
	if err := common.Unmarshal(decodedBody, &createResp); err != nil {
		taskErr = service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", responseBody), "unmarshal_response_body_failed", http.StatusInternalServerError)
		return
	}
	if createResp.HasError() {
		status := resp.StatusCode
		if status == http.StatusOK {
			status = http.StatusBadRequest
		}
		taskErr = service.TaskErrorWrapper(fmt.Errorf("minimax h3 api error: %s", createResp.ErrorMessage()), createResp.ErrorCode(), status)
		return
	}
	if strings.TrimSpace(createResp.TaskID) == "" {
		taskErr = service.TaskErrorWrapper(fmt.Errorf("task_id is empty, body: %s", responseBody), "invalid_response", http.StatusInternalServerError)
		return
	}

	ov := dto.NewOpenAIVideo()
	ov.ID = info.PublicTaskID
	ov.CreatedAt = dto.FormatTimeUnixRFC3339(time.Now().Unix())
	ov.Model = info.OriginModelName
	if ov.Model == "" {
		ov.Model = c.GetString("model")
	}
	ov.Status = dto.VideoStatusQueued
	taskcommon.WriteOpenAIVideoResponse(c, ov)
	return createResp.TaskID, responseBody, nil
}

func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, _ := body["task_id"].(string)
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, fmt.Errorf("invalid task_id")
	}

	req, err := http.NewRequest(http.MethodGet, QueryURL(baseUrl, taskID), nil)
	if err != nil {
		return nil, err
	}
	ApplyAuthHeaders(req.Header, key)

	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	return client.Do(req)
}

func (a *TaskAdaptor) GetModelList() []string {
	return ModelList
}

func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	var queryResp QueryTaskResponse
	if err := common.Unmarshal(respBody, &queryResp); err != nil {
		return nil, errors.Wrap(err, "unmarshal minimax h3 task result failed")
	}

	taskResult := &relaycommon.TaskInfo{Code: 0}
	if queryResp.HasError() {
		taskResult.Code = 1
		taskResult.Reason = queryResp.ErrorMessage()
		taskResult.Status = model.TaskStatusFailure
		taskResult.Progress = taskcommon.ProgressComplete
		return taskResult, nil
	}

	task := queryResp.Task
	taskResult.TaskID = task.ID
	taskResult.Resolution = strings.TrimSpace(task.Resolution)
	taskResult.Ratio = strings.TrimSpace(task.Ratio)
	if task.Duration > 0 {
		taskResult.Duration = task.Duration
	} else if task.Usage != nil && task.Usage.OutputSeconds > 0 {
		taskResult.Duration = task.Usage.OutputSeconds
	} else if task.Usage != nil && task.Usage.TotalSeconds > 0 {
		taskResult.Duration = task.Usage.TotalSeconds
	}

	switch strings.ToLower(strings.TrimSpace(task.Status)) {
	case TaskStatusQueued:
		taskResult.Status = model.TaskStatusQueued
		taskResult.Progress = taskcommon.ProgressQueued
	case TaskStatusRunning:
		taskResult.Status = model.TaskStatusInProgress
		taskResult.Progress = taskcommon.ProgressInProgress
	case TaskStatusSucceeded:
		taskResult.Status = model.TaskStatusSuccess
		taskResult.Progress = taskcommon.ProgressComplete
		if task.Content != nil {
			taskResult.Url = strings.TrimSpace(task.Content.URL)
		}
	case TaskStatusFailed, TaskStatusCancelled:
		taskResult.Status = model.TaskStatusFailure
		taskResult.Progress = taskcommon.ProgressComplete
		if task.Error != nil {
			taskResult.Code = 1
			if task.Error.Message != "" {
				taskResult.Reason = task.Error.Message
			} else {
				taskResult.Reason = task.Error.Code
			}
		}
		if taskResult.Reason == "" {
			if task.Status == TaskStatusCancelled {
				taskResult.Reason = "task cancelled"
			} else {
				taskResult.Reason = "task failed"
			}
		}
	default:
		if strings.TrimSpace(task.Status) == "" && strings.TrimSpace(task.ID) == "" {
			taskResult.Status = ""
			return taskResult, nil
		}
		taskResult.Status = model.TaskStatusInProgress
		taskResult.Progress = taskcommon.ProgressInProgress
	}
	return taskResult, nil
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(originTask *model.Task) ([]byte, error) {
	openAIVideo := dto.NewOpenAIVideo()
	openAIVideo.ID = originTask.TaskID
	openAIVideo.Status = originTask.Status.ToVideoStatus()
	openAIVideo.Model = originTask.Properties.OriginModelName
	openAIVideo.SetProgressStr(originTask.Progress)
	openAIVideo.CreatedAt = dto.FormatTimeUnixRFC3339(originTask.CreatedAt)
	if originTask.FinishTime > 0 {
		openAIVideo.CompletedAt = dto.FormatTimeUnixRFC3339(originTask.FinishTime)
	}

	var queryResp QueryTaskResponse
	if err := common.Unmarshal(originTask.Data, &queryResp); err == nil {
		if queryResp.Task.Content != nil {
			openAIVideo.SetMetadata("url", queryResp.Task.Content.URL)
		}
		if queryResp.HasError() {
			openAIVideo.Error = &dto.OpenAIVideoError{
				Code:    queryResp.ErrorCode(),
				Message: queryResp.ErrorMessage(),
			}
		} else if queryResp.Task.Error != nil {
			openAIVideo.Error = &dto.OpenAIVideoError{
				Code:    queryResp.Task.Error.Code,
				Message: queryResp.Task.Error.Message,
			}
		}
		if queryResp.Task.Duration > 0 {
			openAIVideo.Seconds = fmt.Sprintf("%d", queryResp.Task.Duration)
		}
		if queryResp.Task.Resolution != "" {
			openAIVideo.Size = queryResp.Task.Resolution
		}
	}

	if openAIVideo.Status == dto.VideoStatusFailed && openAIVideo.Error == nil && originTask.FailReason != "" {
		openAIVideo.Error = &dto.OpenAIVideoError{
			Code:    "video_task_failed",
			Message: originTask.FailReason,
		}
	}

	return common.Marshal(openAIVideo)
}
