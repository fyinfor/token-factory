package tencentvod

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	taskcommon "github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
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
	return a.validateNativeOrLegacyRequest(c, info)
}

func (a *TaskAdaptor) AdjustBillingOnSubmit(_ *relaycommon.RelayInfo, _ []byte) map[string]float64 {
	return nil
}

func (a *TaskAdaptor) BuildRequestURL(_ *relaycommon.RelayInfo) (string, error) {
	u := normalizeVodEndpoint(a.baseURL)
	return u + "/", nil
}

func (a *TaskAdaptor) BuildRequestHeader(_ *gin.Context, req *http.Request, _ *relaycommon.RelayInfo) error {
	req.Header.Set("Content-Type", "application/json")
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	if raw, ok := c.Get(contextKeyNativeBody); ok {
		if body, ok := raw.([]byte); ok && len(body) > 0 {
			generateAudio := true
			if req, err := relaycommon.GetTaskRequest(c); err == nil {
				generateAudio = parseGenerateAudioFromMetadata(req.Metadata)
			}
			var native CreateAigcVideoTaskRequest
			if err := common.Unmarshal(body, &native); err == nil {
				enrichNativeCreateRequestAudio(&native, generateAudio)
				if enriched, err := common.Marshal(native); err == nil {
					return bytes.NewReader(enriched), nil
				}
			}
			return bytes.NewReader(body), nil
		}
	}

	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, err
	}
	cred, err := ParseCredentials(a.apiKey)
	if err != nil {
		return nil, err
	}
	modelName, modelVersion := SplitCombinedModel(taskcommon.RelayTaskUpstreamModel(info, req.Model))
	body := map[string]any{
		"SubAppId":     cred.SubAppID,
		"ModelName":    modelName,
		"ModelVersion": modelVersion,
	}
	if prompt := strings.TrimSpace(req.GetPrompt()); prompt != "" {
		body["Prompt"] = prompt
	}
	fileInfos := make([]map[string]any, 0, 2)
	appendImageURL := func(url string) {
		u := strings.TrimSpace(url)
		if u == "" {
			return
		}
		if strings.HasPrefix(strings.ToLower(u), "data:") {
			b64 := extractBase64Payload(u)
			if b64 == "" {
				return
			}
			fileInfos = append(fileInfos, map[string]any{
				"Type":     "Base64",
				"Category": "Image",
				"Base64":   b64,
				"Usage":    "Reference",
			})
			return
		}
		fileInfos = append(fileInfos, map[string]any{
			"Type":     "Url",
			"Category": "Image",
			"Url":      u,
			"Usage":    "Reference",
		})
	}
	appendVideoURL := func(url string) {
		u := strings.TrimSpace(url)
		if u == "" {
			return
		}
		fileInfos = append(fileInfos, map[string]any{
			"Type":          "Url",
			"Category":      "Video",
			"Url":           u,
			"ReferenceType": "base",
		})
	}
	if img := strings.TrimSpace(req.Image); img != "" {
		appendImageURL(img)
	}
	for _, img := range req.Images {
		appendImageURL(img)
	}
	if ref := strings.TrimSpace(req.InputReference); ref != "" {
		appendVideoURL(ref)
	}
	if len(fileInfos) > 0 {
		body["FileInfos"] = fileInfos
	}

	oc, err := outputConfigFromTaskSubmitReq(req)
	if err != nil {
		return nil, err
	}
	body["OutputConfig"] = oc

	data, err := common.Marshal(body)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

func extractBase64Payload(dataURL string) string {
	s := strings.TrimSpace(dataURL)
	if idx := strings.Index(s, ","); idx >= 0 {
		return strings.TrimSpace(s[idx+1:])
	}
	return s
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

	decodedBody := taskcommon.DecodeBase64Response(respBody)

	var env struct {
		Response *struct {
			TaskId *string `json:"TaskId,omitempty"`
			Error  *struct {
				Code    string `json:"Code"`
				Message string `json:"Message"`
			} `json:"Error,omitempty"`
		} `json:"Response"`
	}
	if err = common.Unmarshal(decodedBody, &env); err != nil {
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
	ov.CreatedAt = dto.FormatTimeUnixRFC3339(time.Now().Unix())
	ov.Model = info.OriginModelName

	taskcommon.WriteOpenAIVideoResponse(c, ov)
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
				Status  *string `json:"Status,omitempty"`
				ErrCode *int64  `json:"ErrCode,omitempty"`
				Message *string `json:"Message,omitempty"`
				Input   *struct {
					OutputConfig *AigcVideoOutputConfig `json:"OutputConfig,omitempty"`
				} `json:"Input,omitempty"`
				Output *struct {
					FileInfos []struct {
						FileUrl *string `json:"FileUrl,omitempty"`
					} `json:"FileInfos,omitempty"`
				} `json:"Output,omitempty"`
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

	// 无论终态与否，先把回包 Input.OutputConfig 落到 TaskInfo，供计费对账使用。
	if env.Response.AigcVideoTask != nil && env.Response.AigcVideoTask.Input != nil {
		ApplyActualOutputConfigToTaskInfo(ti, env.Response.AigcVideoTask.Input.OutputConfig)
	}

	switch strings.ToUpper(strings.TrimSpace(*env.Response.Status)) {
	case "FINISH":
		if env.Response.AigcVideoTask != nil && env.Response.AigcVideoTask.Output != nil {
			for _, fi := range env.Response.AigcVideoTask.Output.FileInfos {
				if fi.FileUrl != nil && strings.TrimSpace(*fi.FileUrl) != "" {
					ti.Status = string(model.TaskStatusSuccess)
					ti.Progress = "100%"
					ti.Url = strings.TrimSpace(*fi.FileUrl)
					if ti.Resolution != "" || ti.Duration > 0 || ti.Ratio != "" {
						logger.LogInfo(context.Background(), fmt.Sprintf(
							"[tencentvod] ParseTaskResult 回填计费规格 resolution=%s duration=%d aspect_ratio=%s",
							ti.Resolution, ti.Duration, ti.Ratio,
						))
					}
					return ti, nil
				}
			}
		}
		ti.Status = string(model.TaskStatusFailure)
		ti.Progress = "100%"
		if env.Response.AigcVideoTask != nil && env.Response.AigcVideoTask.Message != nil {
			ti.Reason = strings.TrimSpace(*env.Response.AigcVideoTask.Message)
		}
	case "ABORTED":
		ti.Status = string(model.TaskStatusFailure)
		ti.Progress = "100%"
		if env.Response.AigcVideoTask != nil && env.Response.AigcVideoTask.Message != nil {
			ti.Reason = strings.TrimSpace(*env.Response.AigcVideoTask.Message)
		}
	}
	return ti, nil
}

func (a *TaskAdaptor) GetModelList() []string { return ModelList }
func (a *TaskAdaptor) GetChannelName() string { return ChannelName }

func (a *TaskAdaptor) ConvertToOpenAIVideo(originTask *model.Task) ([]byte, error) {
	ov := originTask.ToOpenAIVideo()
	var env struct {
		Response *struct {
			Error *struct {
				Code    string `json:"Code,omitempty"`
				Message string `json:"Message,omitempty"`
			} `json:"Error,omitempty"`
			AigcVideoTask *struct {
				Message *string `json:"Message,omitempty"`
				Input   *struct {
					OutputConfig *AigcVideoOutputConfig `json:"OutputConfig,omitempty"`
				} `json:"Input,omitempty"`
				Output *struct {
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
		if env.Response.AigcVideoTask != nil && env.Response.AigcVideoTask.Input != nil && env.Response.AigcVideoTask.Input.OutputConfig != nil {
			oc := env.Response.AigcVideoTask.Input.OutputConfig
			spec := oc.ToBillingSpec()
			if spec.Resolution != "" {
				ov.SetMetadata("resolution", spec.Resolution)
			}
			if spec.Duration > 0 {
				ov.SetMetadata("duration", strconv.Itoa(spec.Duration))
			}
			if spec.AspectRatio != "" {
				ov.SetMetadata("ratio", spec.AspectRatio)
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
