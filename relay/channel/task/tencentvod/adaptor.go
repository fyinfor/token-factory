package tencentvod

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
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

var ChannelName = "tencentcloud-vod-video"
var ModelList = []string{"GV-3.1-fast"}

type TaskAdaptor struct {
	ChannelType int
	apiKey      string
	baseURL     string
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.ChannelType = info.ChannelType
	a.baseURL = strings.TrimSpace(info.ChannelBaseUrl)
	a.apiKey = info.ApiKey
}
func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	return relaycommon.ValidateBasicTaskRequest(c, info, constant.TaskActionGenerate)
}
func (a *TaskAdaptor) EstimateBilling(_ *gin.Context, _ *relaycommon.RelayInfo) map[string]float64 {
	return nil
}
func (a *TaskAdaptor) AdjustBillingOnSubmit(_ *relaycommon.RelayInfo, _ []byte) map[string]float64 { return nil }
func (a *TaskAdaptor) AdjustBillingOnComplete(_ *model.Task, _ *relaycommon.TaskInfo) int          { return 0 }

func (a *TaskAdaptor) BuildRequestURL(_ *relaycommon.RelayInfo) (string, error) {
	u := normalizeVodEndpoint(a.baseURL)
	return u + "/", nil
}
func (a *TaskAdaptor) BuildRequestHeader(_ *gin.Context, req *http.Request, _ *relaycommon.RelayInfo) error {
	req.Header.Set("Content-Type", "application/json")
	return nil
}
func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, err
	}
	cred, err := ParseCredentials(a.apiKey)
	if err != nil {
		return nil, err
	}
	modelName, modelVersion := SplitCombinedModel(info.UpstreamModelName)
	body := map[string]any{
		"SubAppId":     cred.SubAppID,
		"ModelName":    modelName,
		"ModelVersion": modelVersion,
	}
	if prompt := strings.TrimSpace(req.GetPrompt()); prompt != "" {
		body["Prompt"] = prompt
	}
	data, err := common.Marshal(body)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

func normalizeVodEndpoint(raw string) string {
	u := strings.TrimRight(strings.TrimSpace(raw), "/")
	if u == "" {
		u = "https://vod.tencentcloudapi.com"
	}
	if !strings.HasPrefix(strings.ToLower(u), "http://") && !strings.HasPrefix(strings.ToLower(u), "https://") {
		u = "https://" + u
	}
	return u
}

func (a *TaskAdaptor) DoRequest(_ *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	payload, err := io.ReadAll(requestBody)
	if err != nil {
		return nil, err
	}
	cred, err := ParseCredentials(info.ApiKey)
	if err != nil {
		return nil, err
	}
	return SignedPOSTJSON(strings.TrimSpace(info.ChannelSetting.Proxy), normalizeVodEndpoint(info.ChannelBaseUrl), cred.Region, cred, "CreateAigcVideoTask", payload)
}

func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (string, []byte, *dto.TaskError) {
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
	}
	_ = resp.Body.Close()
	var env struct {
		Response *struct {
			TaskId *string `json:"TaskId,omitempty"`
			Error  *struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"Error,omitempty"`
		} `json:"Response"`
	}
	if err = common.Unmarshal(respBody, &env); err != nil {
		return "", nil, service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", respBody), "unmarshal_response_body_failed", http.StatusInternalServerError)
	}
	if env.Response != nil && env.Response.Error != nil && strings.TrimSpace(env.Response.Error.Message) != "" {
		return "", nil, service.TaskErrorWrapper(errors.New(env.Response.Error.Message), "video_submit_failed", http.StatusBadRequest)
	}
	taskID := ""
	if env.Response != nil && env.Response.TaskId != nil {
		taskID = strings.TrimSpace(*env.Response.TaskId)
	}
	if taskID == "" {
		return "", nil, service.TaskErrorWrapper(fmt.Errorf("task id is empty, body: %s", string(respBody)), "invalid_response", http.StatusInternalServerError)
	}
	ov := dto.NewOpenAIVideo()
	ov.ID = info.PublicTaskID
	ov.TaskID = info.PublicTaskID
	ov.CreatedAt = time.Now().Unix()
	ov.Model = info.OriginModelName
	c.JSON(http.StatusOK, ov)
	return taskID, respBody, nil
}

func (a *TaskAdaptor) FetchTask(baseURL, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, _ := body["task_id"].(string)
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, fmt.Errorf("invalid task_id")
	}
	cred, err := ParseCredentials(key)
	if err != nil {
		return nil, err
	}
	payload, err := common.Marshal(map[string]any{"TaskId": taskID, "SubAppId": cred.SubAppID})
	if err != nil {
		return nil, err
	}
	return SignedPOSTJSON(strings.TrimSpace(proxy), normalizeVodEndpoint(baseURL), cred.Region, cred, "DescribeTaskDetail", payload)
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	var env struct {
		Response *struct {
			Status        *string `json:"Status,omitempty"`
			AigcVideoTask *struct {
				Output *struct {
					FileInfos []struct {
						FileUrl *string `json:"FileUrl,omitempty"`
					} `json:"FileInfos,omitempty"`
				} `json:"Output,omitempty"`
				Message *string `json:"Message,omitempty"`
			} `json:"AigcVideoTask,omitempty"`
		} `json:"Response"`
	}
	if err := common.Unmarshal(respBody, &env); err != nil {
		return nil, err
	}
	ti := &relaycommon.TaskInfo{Code: 0, Status: string(model.TaskStatusInProgress), Progress: "0%"}
	if env.Response == nil || env.Response.Status == nil {
		return ti, nil
	}
	switch strings.ToUpper(strings.TrimSpace(*env.Response.Status)) {
	case "FINISH":
		if env.Response.AigcVideoTask != nil && env.Response.AigcVideoTask.Output != nil {
			for _, fi := range env.Response.AigcVideoTask.Output.FileInfos {
				if fi.FileUrl != nil && strings.TrimSpace(*fi.FileUrl) != "" {
					ti.Status = string(model.TaskStatusSuccess)
					ti.Progress = "100%"
					ti.Url = strings.TrimSpace(*fi.FileUrl)
					return ti, nil
				}
			}
		}
		ti.Status = string(model.TaskStatusFailure)
		ti.Progress = "100%"
	case "ABORTED":
		ti.Status = string(model.TaskStatusFailure)
		ti.Progress = "100%"
	}
	return ti, nil
}

func (a *TaskAdaptor) GetModelList() []string { return ModelList }
func (a *TaskAdaptor) GetChannelName() string { return ChannelName }

func (a *TaskAdaptor) ConvertToOpenAIVideo(originTask *model.Task) ([]byte, error) {
	ov := originTask.ToOpenAIVideo()
	ov.TaskID = originTask.TaskID
	var env struct {
		Response *struct {
			Error *struct {
				Code    string `json:"Code,omitempty"`
				Message string `json:"Message,omitempty"`
			} `json:"Error,omitempty"`
			AigcVideoTask *struct {
				Message *string `json:"Message,omitempty"`
				Output  *struct {
					FileInfos []struct {
						FileUrl *string `json:"FileUrl,omitempty"`
					} `json:"FileInfos,omitempty"`
				} `json:"Output,omitempty"`
			} `json:"AigcVideoTask,omitempty"`
		} `json:"Response,omitempty"`
	}
	if err := common.Unmarshal(originTask.Data, &env); err == nil && env.Response != nil {
		if env.Response.Error != nil && strings.TrimSpace(env.Response.Error.Message) != "" {
			ov.Error = &dto.OpenAIVideoError{Message: strings.TrimSpace(env.Response.Error.Message), Code: strings.TrimSpace(env.Response.Error.Code)}
		}
		if env.Response.AigcVideoTask != nil && env.Response.AigcVideoTask.Output != nil {
			for _, fi := range env.Response.AigcVideoTask.Output.FileInfos {
				if fi.FileUrl != nil && strings.TrimSpace(*fi.FileUrl) != "" {
					ov.SetMetadata("url", strings.TrimSpace(*fi.FileUrl))
					break
				}
			}
		}
		if ov.Error == nil && originTask.Status == model.TaskStatusFailure {
			msg := strings.TrimSpace(originTask.FailReason)
			if env.Response.AigcVideoTask != nil && env.Response.AigcVideoTask.Message != nil && strings.TrimSpace(*env.Response.AigcVideoTask.Message) != "" {
				msg = strings.TrimSpace(*env.Response.AigcVideoTask.Message)
			}
			if msg != "" {
				ov.Error = &dto.OpenAIVideoError{Message: msg, Code: "tencent_vod_task_failed"}
			}
		}
	}
	return common.Marshal(ov)
}

