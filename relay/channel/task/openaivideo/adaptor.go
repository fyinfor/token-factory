package openaivideo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

// ============================================================================
// OpenAI Video gateway dual-protocol adaptor.
//
// This channel exposes a unified OpenAI-compatible /v1/videos surface to the
// caller, but the upstream may speak one of two concrete protocols. The
// protocol is auto-detected from the channel base URL:
//
//   1. MaaS (Hidream official gateway, https://hiharness.hidreamai.com/docs/)
//      - base URL must include the "/api/maas/gw" prefix (or the official
//        maas domain), e.g. "https://maas.hidreamai.com/api/maas/gw".
//      - Submit:  POST {baseURL}/v1/videos/generations
//      - Result:  GET  {baseURL}/v1/videos/generations/results?task_id=xxx
//      - Body:    schema is a discriminated union keyed by "model_id".
//                 - Avatar / TTS family (e.g. Video-t2ze92dg) uses flat fields:
//                     {"model_id":"...","prompt":"...","image":"...",
//                      "sound_file":"...","mode":"std|pro",...}
//                 - Seedance / Doubao family (e.g. Video-a4lzrja7) uses the
//                   ByteDance-Ark-compatible content array:
//                     {"model_id":"...","content":[
//                       {"type":"text","text":"..."},
//                       {"type":"image_url","image_url":{"url":"..."}}]}
//      - Response shape:
//          {"code":0,"message":"","result":{"task_id":"..."}}
//          status integers in result.status / sub_task_results[].task_status.
//
//   2. ARK (ByteDance Volcano Ark compatible proxy)
//      - any non-MaaS base URL, e.g. third-party reseller domains.
//      - Submit:  POST {baseURL}/v1/videos/generations
//      - Result:  GET  {baseURL}/v1/videos/generations/{id}
//      - Body:    {"model":"...","content":[{"type":"text","text":"..."},
//                  {"type":"image_url","image_url":{"url":"..."}}]}
//      - Response shape:
//          {"id":"cgt-...","status":"queued|running|succeeded|failed|...",
//           "content":{"video_url":"..."},"error":{...}}
//
// The submit path is identical for both protocols (/v1/videos/generations);
// whether to prepend "/api/maas/gw" is decided by the user via base URL.
// ============================================================================

const (
	ProtocolMaaS    = "maas"
	ProtocolArk     = "ark"
	ProtocolSophnet = "sophnet"

	// Submit path is shared by both protocols. The "/api/maas/gw" prefix (if
	// any) lives on the user-configured base URL, not in this constant.
	SubmitPath = "/v1/videos/generations"

	// MaaS result endpoint: <submitPath>/results, task_id passed via query.
	maasResultPath = "/v1/videos/generations/results"

	// ARK result endpoint: <submitPath>/{id}, task_id baked into the path.
	arkResultFmt      = "/v1/videos/generations/%s"
	sophnetSubmitPath = "/videogenerator/generate"
	sophnetResultFmt  = "/videogenerator/generate/%s"

	defaultRatio      = "adaptive"
	defaultResolution = "480p"
	defaultDuration   = 5
)

// DetectProtocol infers the protocol from the channel base URL.
// The Hidream MaaS gateway domain or any URL whose path contains "/api/maas"
// is treated as MaaS; everything else falls back to ARK.
func DetectProtocol(baseURL string) string {
	base := strings.ToLower(strings.TrimSpace(baseURL))
	if strings.Contains(base, "/videogenerator") || strings.Contains(base, "sophnet.com/api/open-apis/projects/easyllms") {
		return ProtocolSophnet
	}
	if strings.Contains(base, "maas.hidreamai.com") || strings.Contains(base, "/api/maas") {
		return ProtocolMaaS
	}
	return ProtocolArk
}

// ============================================================================
// Response structs
// ============================================================================

// --- ARK protocol responses ---

type apiError struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	Type    string `json:"type,omitempty"`
}

type arkSubmitResponse struct {
	ID        string    `json:"id"`
	Model     string    `json:"model,omitempty"`
	Status    string    `json:"status,omitempty"`
	CreatedAt int64     `json:"created_at,omitempty"`
	Error     *apiError `json:"error,omitempty"`
}

type arkTaskContent struct {
	VideoURL string `json:"video_url,omitempty"`
}

type arkVideoOutput struct {
	VideoURL string `json:"video_url,omitempty"`
}

type arkResultResponse struct {
	ID          string          `json:"id"`
	Model       string          `json:"model,omitempty"`
	Status      string          `json:"status,omitempty"`
	Content     *arkTaskContent `json:"content,omitempty"`
	Output      *arkVideoOutput `json:"output,omitempty"`
	CreatedAt   int64           `json:"created_at,omitempty"`
	UpdatedAt   int64           `json:"updated_at,omitempty"`
	CompletedAt int64           `json:"completed_at,omitempty"`
	Error       *apiError       `json:"error,omitempty"`
}

// --- MaaS protocol responses (Hidream official gateway) ---

type maasSubmitResponse struct {
	Code     int    `json:"code"`
	Message  string `json:"message,omitempty"`
	Messasge string `json:"messasge,omitempty"` // tolerate the typo seen in upstream docs
	Result   struct {
		TaskID string `json:"task_id"`
	} `json:"result"`
}

type maasSubTaskResult struct {
	URL        string `json:"url,omitempty"`
	TaskStatus int    `json:"task_status"`
	ErrorMsg   string `json:"error_msg,omitempty"`
}

type maasResultResponse struct {
	Code     int    `json:"code"`
	Message  string `json:"message,omitempty"`
	Messasge string `json:"messasge,omitempty"`
	Result   struct {
		Status         int                 `json:"status"`
		SubTaskResults []maasSubTaskResult `json:"sub_task_results"`
	} `json:"result"`
}

type sophnetSubmitResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message,omitempty"`
	Result  struct {
		TaskID string `json:"task_id"`
	} `json:"result"`
}

type sophnetResultResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message,omitempty"`
	Result  struct {
		ID      string           `json:"id"`
		Model   string           `json:"model,omitempty"`
		Status  string           `json:"status,omitempty"`
		Content *arkTaskContent  `json:"content,omitempty"`
		Output  *arkVideoOutput  `json:"output,omitempty"`
		Error   *apiError        `json:"error,omitempty"`
	} `json:"result"`
}

// ============================================================================
// TaskAdaptor implementation
// ============================================================================

type TaskAdaptor struct {
	taskcommon.BaseBilling
	ChannelType int
	apiKey      string
	baseURL     string
	protocol    string // "maas" | "ark", inferred from base URL.
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.ChannelType = info.ChannelType
	a.baseURL = info.ChannelBaseUrl
	a.apiKey = info.ApiKey
	a.protocol = DetectProtocol(info.ChannelBaseUrl)
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	return relaycommon.ValidateBasicTaskRequest(c, info, constant.TaskActionGenerate)
}

// EstimateBilling extracts duration / resolution from the request and exposes
// them as OtherRatios for the billing layer, so per-second / per-resolution
// pricing rules can pre-deduct quota correctly.
func (a *TaskAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
	if info.Action == constant.TaskActionRemix {
		return nil
	}
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil
	}

	duration, resolution := extractDurationAndResolution(req)

	ratios := map[string]float64{
		"seconds": float64(duration),
		"size":    resolutionToSizeRatio(resolution),
	}
	return ratios
}

func (a *TaskAdaptor) BuildRequestURL(_ *relaycommon.RelayInfo) (string, error) {
	if a.protocol == ProtocolSophnet {
		return fmt.Sprintf("%s%s", strings.TrimRight(a.baseURL, "/"), sophnetSubmitPath), nil
	}
	return fmt.Sprintf("%s%s", strings.TrimRight(a.baseURL, "/"), SubmitPath), nil
}

func (a *TaskAdaptor) BuildRequestHeader(_ *gin.Context, req *http.Request, _ *relaycommon.RelayInfo) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, err
	}

	var bodyMap map[string]any
	if a.protocol == ProtocolMaaS {
		bodyMap, err = a.buildMaasPayloadMap(&req)
	} else {
		bodyMap, err = a.buildArkPayloadMap(&req)
	}
	if err != nil {
		return nil, errors.Wrap(err, "convert request payload failed")
	}

	// Model field name differs by protocol: MaaS = "model_id", ARK = "model".
	modelKey := "model"
	if a.protocol == ProtocolMaaS {
		modelKey = "model_id"
	}
	if info.IsModelMapped {
		bodyMap[modelKey] = info.UpstreamModelName
	} else if v, ok := bodyMap[modelKey].(string); ok {
		info.UpstreamModelName = v
	}
	data, err := common.Marshal(bodyMap)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (string, []byte, *dto.TaskError) {
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
	}
	_ = resp.Body.Close()

	var taskID string
	if a.protocol == ProtocolMaaS {
		var sub maasSubmitResponse
		if err := common.Unmarshal(respBody, &sub); err != nil {
			return "", nil, service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", respBody), "unmarshal_response_body_failed", http.StatusInternalServerError)
		}
		if sub.Code != 0 {
			msg := firstNonEmpty(sub.Message, sub.Messasge, fmt.Sprintf("video upstream returned code=%d", sub.Code))
			return "", nil, service.TaskErrorWrapper(errors.New(msg), "video_submit_failed", http.StatusBadRequest)
		}
		taskID = strings.TrimSpace(sub.Result.TaskID)
	} else if a.protocol == ProtocolSophnet {
		var sub sophnetSubmitResponse
		if err := common.Unmarshal(respBody, &sub); err != nil {
			return "", nil, service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", respBody), "unmarshal_response_body_failed", http.StatusInternalServerError)
		}
		if sub.Status != 0 {
			msg := firstNonEmpty(sub.Message, fmt.Sprintf("video upstream returned status=%d", sub.Status))
			return "", nil, service.TaskErrorWrapper(errors.New(msg), "video_submit_failed", http.StatusBadRequest)
		}
		taskID = strings.TrimSpace(sub.Result.TaskID)
	} else {
		var sub arkSubmitResponse
		if err := common.Unmarshal(respBody, &sub); err != nil {
			return "", nil, service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", respBody), "unmarshal_response_body_failed", http.StatusInternalServerError)
		}
		if sub.Error != nil && (sub.Error.Message != "" || sub.Error.Code != "") {
			msg := firstNonEmpty(sub.Error.Message, sub.Error.Code, "video upstream returned error")
			return "", nil, service.TaskErrorWrapper(errors.New(msg), "video_submit_failed", http.StatusBadRequest)
		}
		taskID = strings.TrimSpace(sub.ID)
	}

	if taskID == "" {
		return "", nil, service.TaskErrorWrapper(fmt.Errorf("task id is empty, body: %s", string(respBody)), "invalid_response", http.StatusInternalServerError)
	}

	ov := dto.NewOpenAIVideo()
	ov.ID = info.PublicTaskID
	ov.CreatedAt = dto.FormatTimeUnixRFC3339(time.Now().Unix())
	ov.Model = info.OriginModelName
	c.JSON(http.StatusOK, ov)

	return taskID, respBody, nil
}

func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok || strings.TrimSpace(taskID) == "" {
		return nil, fmt.Errorf("invalid task_id")
	}
	taskID = strings.TrimSpace(taskID)
	// FetchTask is invoked by the framework outside of Init, so we have to
	// re-detect the protocol from baseUrl every time.
	protocol := DetectProtocol(baseUrl)
	var uri string
	if protocol == ProtocolMaaS {
		uri = fmt.Sprintf("%s%s?task_id=%s", strings.TrimRight(baseUrl, "/"), maasResultPath, taskID)
	} else if protocol == ProtocolSophnet {
		uri = fmt.Sprintf("%s%s", strings.TrimRight(baseUrl, "/"), fmt.Sprintf(sophnetResultFmt, taskID))
	} else {
		uri = fmt.Sprintf("%s%s", strings.TrimRight(baseUrl, "/"), fmt.Sprintf(arkResultFmt, taskID))
	}

	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)

	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	return client.Do(req)
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	normalized := normalizeOpenAIVideoPollJSON(respBody)
	// Pick the parser via shape detection: MaaS responses carry top-level
	// "code"/"result"; ARK responses carry top-level "id"/"status".
	switch detectResponseProtocol(normalized) {
	case ProtocolMaaS:
		return parseMaasResult(normalized)
	case ProtocolSophnet:
		return parseSophnetResult(normalized)
	default:
		return parseArkResult(normalized)
	}
}

// normalizeOpenAIVideoPollJSON unwraps common reseller / Volc-style envelopes so that
// Ark-shaped bodies are visible to detectResponseProtocol / parseArkResult.
// Example: {"code":0,"data":{"id":"...","status":"succeeded","content":{...}}}
func normalizeOpenAIVideoPollJSON(respBody []byte) []byte {
	var probe map[string]any
	if err := common.Unmarshal(respBody, &probe); err != nil {
		return respBody
	}
	data, ok := probe["data"].(map[string]any)
	if !ok {
		return respBody
	}
	if _, ok := data["status"]; !ok {
		return respBody
	}
	// Ark task object: has id, or has content/output typical of video poll
	_, hasID := data["id"]
	_, hasContent := data["content"]
	_, hasOutput := data["output"]
	if !hasID && !hasContent && !hasOutput {
		return respBody
	}
	nested, err := common.Marshal(data)
	if err != nil {
		return respBody
	}
	return nested
}

// detectResponseProtocol probes characteristic top-level fields to figure out
// which protocol shape the response uses.
func detectResponseProtocol(respBody []byte) string {
	var probe map[string]any
	if err := common.Unmarshal(respBody, &probe); err != nil {
		return ProtocolArk
	}
	if status, hasStatus := probe["status"]; hasStatus {
		if _, hasResult := probe["result"]; hasResult {
			switch status.(type) {
			case float64, int, int32, int64, json.Number:
				return ProtocolSophnet
			}
		}
	}
	if _, hasResult := probe["result"]; hasResult {
		return ProtocolMaaS
	}
	// Many gateways wrap Ark in {"code":0,"data":{...}} — that must NOT be treated as MaaS
	// just because of top-level "code". Only treat as MaaS when it looks like a MaaS / business
	// error envelope (numeric code, no Ark "id" at top level, and data is absent or not an Ark task).
	if _, hasCode := probe["code"]; hasCode {
		switch probe["code"].(type) {
		case float64, int, int32, int64, json.Number:
			if _, hasArkID := probe["id"]; hasArkID {
				return ProtocolArk
			}
			if dm, ok := probe["data"].(map[string]any); ok {
				if _, ok := dm["id"]; ok {
					return ProtocolArk
				}
				if _, ok := dm["status"]; ok && (dm["content"] != nil || dm["output"] != nil) {
					return ProtocolArk
				}
			}
			return ProtocolMaaS
		default:
			// non-numeric code — fall through to Ark
		}
	}
	return ProtocolArk
}

func parseArkResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	var resp arkResultResponse
	if err := common.Unmarshal(respBody, &resp); err != nil {
		return nil, errors.Wrap(err, "unmarshal ark task result failed")
	}

	taskResult := &relaycommon.TaskInfo{Code: 0}

	// ARK statuses: queued / running / succeeded / failed / expired / cancelled.
	switch strings.ToLower(strings.TrimSpace(resp.Status)) {
	case "queued":
		taskResult.Status = model.TaskStatusQueued
		taskResult.Progress = taskcommon.ProgressQueued
	case "running", "in_progress", "processing":
		taskResult.Status = model.TaskStatusInProgress
		taskResult.Progress = taskcommon.ProgressInProgress
	case "succeeded", "completed", "success", "done":
		taskResult.Status = model.TaskStatusSuccess
		taskResult.Progress = "100%"
		if resp.Content != nil && strings.TrimSpace(resp.Content.VideoURL) != "" {
			taskResult.Url = resp.Content.VideoURL
		} else if resp.Output != nil && strings.TrimSpace(resp.Output.VideoURL) != "" {
			taskResult.Url = resp.Output.VideoURL
		}
	case "failed", "expired", "cancelled", "canceled":
		taskResult.Status = model.TaskStatusFailure
		taskResult.Progress = "100%"
		if resp.Error != nil {
			taskResult.Reason = firstNonEmpty(resp.Error.Message, resp.Error.Code, fmt.Sprintf("video task %s", resp.Status))
		} else {
			taskResult.Reason = fmt.Sprintf("video task %s", resp.Status)
		}
	default:
		if resp.Error != nil && (resp.Error.Message != "" || resp.Error.Code != "") {
			taskResult.Status = model.TaskStatusFailure
			taskResult.Progress = "100%"
			taskResult.Reason = firstNonEmpty(resp.Error.Message, resp.Error.Code)
		} else {
			taskResult.Status = model.TaskStatusInProgress
			taskResult.Progress = taskcommon.ProgressInProgress
		}
	}

	return taskResult, nil
}

func parseSophnetResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	var resp sophnetResultResponse
	if err := common.Unmarshal(respBody, &resp); err != nil {
		return nil, errors.Wrap(err, "unmarshal sophnet task result failed")
	}

	taskResult := &relaycommon.TaskInfo{Code: 0}
	if resp.Status != 0 {
		taskResult.Status = model.TaskStatusFailure
		taskResult.Progress = "100%"
		taskResult.Reason = firstNonEmpty(resp.Message, fmt.Sprintf("status=%d", resp.Status))
		return taskResult, nil
	}

	arkView := arkResultResponse{
		ID:      resp.Result.ID,
		Model:   resp.Result.Model,
		Status:  resp.Result.Status,
		Content: resp.Result.Content,
		Output:  resp.Result.Output,
		Error:   resp.Result.Error,
	}
	normalized, err := common.Marshal(arkView)
	if err != nil {
		return nil, err
	}
	return parseArkResult(normalized)
}

func parseMaasResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	var resp maasResultResponse
	if err := common.Unmarshal(respBody, &resp); err != nil {
		return nil, errors.Wrap(err, "unmarshal maas task result failed")
	}

	taskResult := &relaycommon.TaskInfo{Code: 0}

	if resp.Code != 0 {
		taskResult.Status = model.TaskStatusFailure
		taskResult.Progress = "100%"
		taskResult.Reason = firstNonEmpty(resp.Message, resp.Messasge, fmt.Sprintf("code=%d", resp.Code))
		return taskResult, nil
	}

	// Sub-tasks: usually a single one. Any failure -> failed; all success ->
	// success; otherwise -> in progress.
	successCount, failureCount, queuedCount, processingCount := 0, 0, 0, 0
	var firstURL, firstErr string
	for _, sub := range resp.Result.SubTaskResults {
		switch sub.TaskStatus {
		case 1:
			successCount++
			if firstURL == "" {
				firstURL = sub.URL
			}
		case 3, 4:
			failureCount++
			if firstErr == "" {
				firstErr = sub.ErrorMsg
			}
		case 0:
			queuedCount++
		case 2:
			processingCount++
		}
	}

	totalSubTasks := len(resp.Result.SubTaskResults)
	switch {
	case failureCount > 0:
		taskResult.Status = model.TaskStatusFailure
		taskResult.Progress = "100%"
		taskResult.Reason = firstNonEmpty(firstErr, "video sub-task failed")
	case totalSubTasks > 0 && successCount == totalSubTasks:
		taskResult.Status = model.TaskStatusSuccess
		taskResult.Progress = "100%"
		taskResult.Url = firstURL
	case resp.Result.Status == 1 && totalSubTasks == 0:
		taskResult.Status = model.TaskStatusFailure
		taskResult.Progress = "100%"
		taskResult.Reason = "video task returned status=1 but no sub_task_results"
	case processingCount > 0:
		taskResult.Status = model.TaskStatusInProgress
		taskResult.Progress = taskcommon.ProgressInProgress
	case queuedCount > 0:
		taskResult.Status = model.TaskStatusQueued
		taskResult.Progress = taskcommon.ProgressQueued
	default:
		taskResult.Status = model.TaskStatusInProgress
		taskResult.Progress = taskcommon.ProgressInProgress
	}

	return taskResult, nil
}

func (a *TaskAdaptor) GetModelList() []string {
	return ModelList
}

func (a *TaskAdaptor) GetChannelName() string {
	return ChannelName
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(originTask *model.Task) ([]byte, error) {
	ov := dto.NewOpenAIVideo()
	ov.ID = originTask.TaskID
	ov.Status = originTask.Status.ToVideoStatus()
	ov.SetProgressStr(originTask.Progress)
	ov.CreatedAt = dto.FormatTimeUnixRFC3339(originTask.CreatedAt)
	if originTask.FinishTime > 0 {
		ov.CompletedAt = dto.FormatTimeUnixRFC3339(originTask.FinishTime)
	}
	ov.Model = originTask.Properties.OriginModelName

	normalizedData := normalizeOpenAIVideoPollJSON(originTask.Data)
	// Pick the parser via response shape detection, then surface URL/error.
	switch detectResponseProtocol(normalizedData) {
	case ProtocolMaaS:
		var resp maasResultResponse
		if err := common.Unmarshal(normalizedData, &resp); err != nil {
			return nil, errors.Wrap(err, "unmarshal video task data (maas) failed")
		}
		var firstURL, firstErr string
		for _, sub := range resp.Result.SubTaskResults {
			if firstURL == "" && sub.URL != "" {
				firstURL = sub.URL
			}
			if firstErr == "" && sub.ErrorMsg != "" {
				firstErr = sub.ErrorMsg
			}
		}
		if firstURL != "" {
			ov.SetMetadata("url", firstURL)
		}
		if firstErr != "" {
			ov.Error = &dto.OpenAIVideoError{
				Message: firstErr,
				Code:    "video_subtask_failed",
			}
		}
	case ProtocolSophnet:
		var resp sophnetResultResponse
		if err := common.Unmarshal(normalizedData, &resp); err != nil {
			return nil, errors.Wrap(err, "unmarshal video task data (sophnet) failed")
		}
		if resp.Result.Content != nil && strings.TrimSpace(resp.Result.Content.VideoURL) != "" {
			ov.SetMetadata("url", resp.Result.Content.VideoURL)
		}
		if resp.Status != 0 {
			ov.Error = &dto.OpenAIVideoError{
				Message: firstNonEmpty(resp.Message, fmt.Sprintf("status=%d", resp.Status)),
				Code:    "video_task_failed",
			}
		} else if resp.Result.Error != nil && (resp.Result.Error.Message != "" || resp.Result.Error.Code != "") {
			ov.Error = &dto.OpenAIVideoError{
				Message: firstNonEmpty(resp.Result.Error.Message, resp.Result.Error.Code),
				Code:    firstNonEmpty(resp.Result.Error.Code, "video_task_failed"),
			}
		}
	default:
		var resp arkResultResponse
		if err := common.Unmarshal(normalizedData, &resp); err != nil {
			return nil, errors.Wrap(err, "unmarshal video task data (ark) failed")
		}
		if resp.Content != nil && strings.TrimSpace(resp.Content.VideoURL) != "" {
			ov.SetMetadata("url", resp.Content.VideoURL)
		} else if resp.Output != nil && strings.TrimSpace(resp.Output.VideoURL) != "" {
			ov.SetMetadata("url", resp.Output.VideoURL)
		}
		if resp.Error != nil && (resp.Error.Message != "" || resp.Error.Code != "") {
			ov.Error = &dto.OpenAIVideoError{
				Message: firstNonEmpty(resp.Error.Message, resp.Error.Code),
				Code:    firstNonEmpty(resp.Error.Code, "video_task_failed"),
			}
		}
	}

	return common.Marshal(ov)
}

// ============================================================================
// helpers
// ============================================================================

// metadataPayload is only used by EstimateBilling to extract duration /
// resolution pricing fields. The actual upstream request body no longer
// depends on this struct; everything else is forwarded through metadata.
type metadataPayload struct {
	Ratio         string         `json:"ratio,omitempty"`
	Resolution    string         `json:"resolution,omitempty"`
	GenerateAudio *dto.BoolValue `json:"generate_audio,omitempty"`
	Duration      *dto.IntValue  `json:"duration,omitempty"`
	RequestID     string         `json:"request_id,omitempty"`
	ImageURL      string         `json:"image_url,omitempty"`
}

// buildArkPayloadMap builds the ByteDance Volcano Ark compatible request body:
//
//	{
//	  "model": "<model_id>",
//	  "content": [
//	    {"type": "text", "text": "..."},
//	    {"type": "image_url", "image_url": {"url": "..."}}
//	  ]
//	}
func (a *TaskAdaptor) buildArkPayloadMap(req *relaycommon.TaskSubmitReq) (map[string]any, error) {
	body := make(map[string]any, 4)
	body["model"] = req.Model

	content := make([]map[string]any, 0, len(req.Images)+1)
	if prompt := strings.TrimSpace(req.GetPrompt()); prompt != "" {
		content = append(content, map[string]any{
			"type": "text",
			"text": prompt,
		})
	}
	for _, url := range req.Images {
		url = strings.TrimSpace(url)
		if url == "" {
			continue
		}
		content = append(content, map[string]any{
			"type":      "image_url",
			"image_url": map[string]any{"url": url},
		})
	}
	if len(content) > 0 {
		body["content"] = content
	}

	// Forward metadata to top-level body, but never overwrite model/content.
	for k, v := range req.Metadata {
		if v == nil {
			continue
		}
		if k == "model" || k == "content" {
			continue
		}
		body[k] = v
	}
	return body, nil
}

// buildMaasPayloadMap builds the Hidream MaaS gateway request body.
//
// The MaaS gateway uses a discriminated-union body schema keyed by model_id:
//
//  1. Avatar / TTS family (e.g. Video-t2ze92dg avatar_image2video) uses flat
//     fields:
//
//     {"model_id":"...","prompt":"...","image":"...",
//     "sound_file":"...","mode":"std|pro",...}
//
//  2. Seedance / Doubao family (e.g. Video-a4lzrja7) uses the Ark-compatible
//     content array, only with the model field renamed to model_id:
//
//     {"model_id":"...","content":[
//     {"type":"text","text":"..."},
//     {"type":"image_url","image_url":{"url":"..."}}]}
//
// We pick the mode by sniffing whether the user explicitly supplied any of the
// flat MaaS-only fields (image / sound_file) in metadata:
//
//   - flat fields present  -> flat mode (avatar etc.)
//   - flat fields missing  -> content[] mode (Seedance etc., default)
//
// All other fields (mode, ratio, resolution, duration, generate_audio,
// request_id, ...) are forwarded from metadata as-is.
func (a *TaskAdaptor) buildMaasPayloadMap(req *relaycommon.TaskSubmitReq) (map[string]any, error) {
	body := make(map[string]any, 8)
	body["model_id"] = req.Model

	_, hasFlatImage := req.Metadata["image"]
	_, hasFlatSound := req.Metadata["sound_file"]
	useFlat := hasFlatImage || hasFlatSound

	if useFlat {
		// Flat mode: prompt + a single image URL (avatar models accept exactly
		// one input image).
		if prompt := strings.TrimSpace(req.GetPrompt()); prompt != "" {
			body["prompt"] = prompt
		}
		for _, url := range req.Images {
			if url = strings.TrimSpace(url); url != "" {
				body["image"] = url
				break
			}
		}
	} else {
		// Content-array mode: structurally identical to ARK, only the model
		// field name differs.
		content := make([]map[string]any, 0, len(req.Images)+1)
		if prompt := strings.TrimSpace(req.GetPrompt()); prompt != "" {
			content = append(content, map[string]any{
				"type": "text",
				"text": prompt,
			})
		}
		for _, url := range req.Images {
			url = strings.TrimSpace(url)
			if url == "" {
				continue
			}
			content = append(content, map[string]any{
				"type":      "image_url",
				"image_url": map[string]any{"url": url},
			})
		}
		if len(content) > 0 {
			body["content"] = content
		}
	}

	// Forward metadata; never overwrite model_id. Other keys (including
	// "content" and the flat fields) may legitimately be supplied/overridden
	// by the caller.
	for k, v := range req.Metadata {
		if v == nil {
			continue
		}
		if k == "model_id" {
			continue
		}
		body[k] = v
	}
	return body, nil
}

func sizeToResolution(size string) string {
	parts := strings.Split(strings.ToLower(size), "x")
	if len(parts) != 2 {
		return ""
	}
	w, errW := strconv.Atoi(strings.TrimSpace(parts[0]))
	h, errH := strconv.Atoi(strings.TrimSpace(parts[1]))
	if errW != nil || errH != nil {
		return ""
	}
	short := w
	if h < short {
		short = h
	}
	switch {
	case short >= 1080:
		return "1080p"
	case short >= 720:
		return "720p"
	case short >= 480:
		return "480p"
	default:
		return "480p"
	}
}

// resolutionToSizeRatio maps a resolution token to a default size ratio used
// for billing. 480p == 1.0; higher resolutions scale roughly by area. These
// are only fallbacks - per-model pricing should override them.
func resolutionToSizeRatio(resolution string) float64 {
	switch strings.ToLower(strings.TrimSpace(resolution)) {
	case "1080p":
		return 4.5
	case "720p":
		return 2.25
	case "480p":
		return 1.0
	default:
		return 1.0
	}
}

func extractDurationAndResolution(req relaycommon.TaskSubmitReq) (int, string) {
	duration := defaultDuration
	resolution := defaultResolution

	meta := metadataPayload{}
	_ = taskcommon.UnmarshalMetadata(req.Metadata, &meta)

	switch {
	case meta.Duration != nil && *meta.Duration > 0:
		duration = int(*meta.Duration)
	case strings.TrimSpace(req.Seconds) != "":
		if d, err := strconv.Atoi(strings.TrimSpace(req.Seconds)); err == nil && d > 0 {
			duration = d
		}
	case req.Duration > 0:
		duration = req.Duration
	}

	if v := strings.TrimSpace(meta.Resolution); v != "" {
		resolution = v
	} else if size := strings.TrimSpace(req.Size); size != "" {
		if r := sizeToResolution(size); r != "" {
			resolution = r
		}
	}
	return duration, resolution
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}
