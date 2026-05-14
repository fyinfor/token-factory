package openaivideo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
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
	// ProtocolTokenFactory 本仓库网关统一任务视频入口（router/video-router.go）：
	// POST/GET /v1/video/generations，与第三方 Ark 网关使用的 /v1/videos/generations 不同。
	ProtocolTokenFactory = "tokenfactory"

	tfStyleVideoGenerations = "video_generations"
	tfStyleOpenAIVideos     = "openai_videos"
	tfStyleOpenAIRemix     = "openai_remix"

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
	if strings.Contains(base, "maas.hidreamai.com") ||
		strings.Contains(base, "hiharness.hidreamai.com") ||
		strings.Contains(base, "/api/maas") {
		return ProtocolMaaS
	}
	return ProtocolArk
}

func normalizeMaaSBaseURL(baseURL string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	lower := strings.ToLower(base)
	if strings.Contains(lower, "hiharness.hidreamai.com") && !strings.Contains(lower, "/api/maas") {
		return base + "/api/maas/gw"
	}
	return base
}

// classifyTfOpenVideoClientPath maps the downstream HTTP path (after playground normalization)
// to an upstream TokenFactory route family.
func classifyTfOpenVideoClientPath(requestURLPath string) string {
	path := strings.TrimSpace(requestURLPath)
	if path == "" {
		return tfStyleVideoGenerations
	}
	if u, err := url.Parse(path); err == nil && strings.TrimSpace(u.Path) != "" {
		path = u.Path
	} else {
		path = strings.Split(path, "?")[0]
	}
	path = strings.TrimRight(path, "/")
	if strings.Contains(path, "/v1/videos/") && strings.HasSuffix(path, "/remix") {
		return tfStyleOpenAIRemix
	}
	if strings.HasSuffix(path, "/v1/videos") {
		return tfStyleOpenAIVideos
	}
	return tfStyleVideoGenerations
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
	ID        string          `json:"id"`
	Model     string          `json:"model,omitempty"`
	Status    string          `json:"status,omitempty"`
	CreatedAt json.RawMessage `json:"created_at,omitempty"` // upstream may send unix (number) or RFC3339 (string)
	Error     *apiError       `json:"error,omitempty"`
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
	CreatedAt   json.RawMessage `json:"created_at,omitempty"`
	UpdatedAt   json.RawMessage `json:"updated_at,omitempty"`
	CompletedAt json.RawMessage `json:"completed_at,omitempty"`
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
		ID      string          `json:"id"`
		Model   string          `json:"model,omitempty"`
		Status  string          `json:"status,omitempty"`
		Content *arkTaskContent `json:"content,omitempty"`
		Output  *arkVideoOutput `json:"output,omitempty"`
		Error   *apiError       `json:"error,omitempty"`
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
	if info.ChannelType == constant.ChannelTypeTokenFactoryOpen {
		a.protocol = ProtocolTokenFactory
		info.TfOpenVideoUpstreamStyle = classifyTfOpenVideoClientPath(info.RequestURLPath)
	} else {
		a.protocol = DetectProtocol(info.ChannelBaseUrl)
	}
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

func asStringAny(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

// parseTfUpstreamSubmitTaskID extracts upstream task/video id from TokenFactory OpenAI-style or generic JSON bodies.
func parseTfUpstreamSubmitTaskID(respBody []byte) (string, *dto.TaskError) {
	var probe map[string]any
	if err := common.Unmarshal(respBody, &probe); err != nil {
		return "", service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", respBody), "unmarshal_response_body_failed", http.StatusInternalServerError)
	}
	if errObj, ok := probe["error"]; ok && errObj != nil {
		if em, ok := errObj.(map[string]any); ok {
			msg := firstNonEmpty(asStringAny(em["message"]), asStringAny(em["code"]), "video upstream returned error")
			return "", service.TaskErrorWrapper(errors.New(msg), "video_submit_failed", http.StatusBadRequest)
		}
	}
	if id := asStringAny(probe["id"]); id != "" {
		return id, nil
	}
	if data, ok := probe["data"].(map[string]any); ok {
		if id := asStringAny(data["id"]); id != "" {
			return id, nil
		}
	}
	return "", service.TaskErrorWrapper(fmt.Errorf("task id is empty, body: %s", string(respBody)), "invalid_response", http.StatusInternalServerError)
}

func buildTokenFactoryOpenAIVideoPassthroughBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return nil, errors.Wrap(err, "get_request_body_failed")
	}
	cachedBody, err := storage.Bytes()
	if err != nil {
		return nil, errors.Wrap(err, "read_request_body_failed")
	}
	contentType := c.Request.Header.Get("Content-Type")

	if strings.HasPrefix(contentType, "application/json") {
		var bodyMap map[string]any
		if err := common.Unmarshal(cachedBody, &bodyMap); err == nil {
			if um := strings.TrimSpace(info.UpstreamModelName); um != "" {
				bodyMap["model"] = um
			}
			if newBody, err := common.Marshal(bodyMap); err == nil {
				return bytes.NewReader(newBody), nil
			}
		}
		return bytes.NewReader(cachedBody), nil
	}

	if strings.Contains(contentType, "multipart/form-data") {
		formData, err := common.ParseMultipartFormReusable(c)
		if err != nil {
			return bytes.NewReader(cachedBody), nil
		}
		var buf bytes.Buffer
		writer := multipart.NewWriter(&buf)
		writer.WriteField("model", strings.TrimSpace(info.UpstreamModelName))
		for key, values := range formData.Value {
			if key == "model" {
				continue
			}
			for _, v := range values {
				writer.WriteField(key, v)
			}
		}
		for fieldName, fileHeaders := range formData.File {
			for _, fh := range fileHeaders {
				f, err := fh.Open()
				if err != nil {
					continue
				}
				fileCT := fh.Header.Get("Content-Type")
				if fileCT == "" || fileCT == "application/octet-stream" {
					buf512 := make([]byte, 512)
					n, _ := io.ReadFull(f, buf512)
					fileCT = http.DetectContentType(buf512[:n])
					f.Close()
					f, err = fh.Open()
					if err != nil {
						continue
					}
				}
				h := make(textproto.MIMEHeader)
				h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fieldName, fh.Filename))
				h.Set("Content-Type", fileCT)
				part, err := writer.CreatePart(h)
				if err != nil {
					f.Close()
					continue
				}
				_, _ = io.Copy(part, f)
				f.Close()
			}
		}
		if err := writer.Close(); err != nil {
			return nil, err
		}
		c.Request.Header.Set("Content-Type", writer.FormDataContentType())
		return &buf, nil
	}

	return bytes.NewReader(cachedBody), nil
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	if a.protocol == ProtocolSophnet {
		return fmt.Sprintf("%s%s", strings.TrimRight(a.baseURL, "/"), sophnetSubmitPath), nil
	}
	baseURL := strings.TrimRight(a.baseURL, "/")
	if a.protocol == ProtocolMaaS {
		baseURL = normalizeMaaSBaseURL(a.baseURL)
		return fmt.Sprintf("%s%s", baseURL, SubmitPath), nil
	}
	if a.protocol == ProtocolTokenFactory {
		if info != nil && info.Action == constant.TaskActionRemix {
			vid := strings.TrimSpace(info.OriginTaskID)
			if vid == "" {
				return "", fmt.Errorf("remix requires origin video id")
			}
			return fmt.Sprintf("%s/v1/videos/%s/remix", baseURL, vid), nil
		}
		if info != nil && info.TfOpenVideoUpstreamStyle == tfStyleOpenAIVideos {
			return fmt.Sprintf("%s/v1/videos", baseURL), nil
		}
		return fmt.Sprintf("%s/v1/video/generations", baseURL), nil
	}
	return fmt.Sprintf("%s%s", baseURL, SubmitPath), nil
}

func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	if a.protocol == ProtocolTokenFactory && info != nil &&
		(info.TfOpenVideoUpstreamStyle == tfStyleOpenAIVideos || info.TfOpenVideoUpstreamStyle == tfStyleOpenAIRemix) {
		if ct := c.Request.Header.Get("Content-Type"); ct != "" {
			req.Header.Set("Content-Type", ct)
		}
		return nil
	}
	req.Header.Set("Content-Type", "application/json")
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, err
	}

	if a.protocol == ProtocolTokenFactory {
		if info != nil && (info.TfOpenVideoUpstreamStyle == tfStyleOpenAIVideos || info.TfOpenVideoUpstreamStyle == tfStyleOpenAIRemix) {
			return buildTokenFactoryOpenAIVideoPassthroughBody(c, info)
		}
		raw, mErr := common.Marshal(req)
		if mErr != nil {
			return nil, mErr
		}
		var bodyMap map[string]any
		if err := common.Unmarshal(raw, &bodyMap); err != nil {
			return nil, err
		}
		um := strings.TrimSpace(info.UpstreamModelName)
		if um == "" {
			um = strings.TrimSpace(req.Model)
		}
		if um != "" {
			bodyMap["model"] = um
		}
		data, mErr := common.Marshal(bodyMap)
		if mErr != nil {
			return nil, mErr
		}
		return bytes.NewReader(data), nil
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
	if info.UseRelayTaskUpstreamModel() {
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
	} else if a.protocol == ProtocolTokenFactory && info != nil &&
		(info.TfOpenVideoUpstreamStyle == tfStyleOpenAIVideos || info.TfOpenVideoUpstreamStyle == tfStyleOpenAIRemix) {
		var terr *dto.TaskError
		taskID, terr = parseTfUpstreamSubmitTaskID(respBody)
		if terr != nil {
			return "", nil, terr
		}
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

func channelTypeFromFetchBody(body map[string]any) int {
	if body == nil {
		return 0
	}
	switch v := body["channel_type"].(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float32:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

func tfOpenVideoUpstreamStyleFromBody(body map[string]any) string {
	if body == nil {
		return ""
	}
	if s, ok := body["tf_open_video_upstream_style"].(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
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
	if channelTypeFromFetchBody(body) == constant.ChannelTypeTokenFactoryOpen {
		protocol = ProtocolTokenFactory
	}
	var uri string
	if protocol == ProtocolMaaS {
		uri = fmt.Sprintf("%s%s?task_id=%s", normalizeMaaSBaseURL(baseUrl), maasResultPath, taskID)
	} else if protocol == ProtocolSophnet {
		uri = fmt.Sprintf("%s%s", strings.TrimRight(baseUrl, "/"), fmt.Sprintf(sophnetResultFmt, taskID))
	} else if protocol == ProtocolTokenFactory {
		base := strings.TrimRight(baseUrl, "/")
		switch tfOpenVideoUpstreamStyleFromBody(body) {
		case tfStyleOpenAIVideos, tfStyleOpenAIRemix:
			uri = fmt.Sprintf("%s/v1/videos/%s", base, taskID)
		default:
			uri = fmt.Sprintf("%s/v1/video/generations/%s", base, taskID)
		}
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
	ov := originTask.ToOpenAIVideo()

	normalizedData := normalizeOpenAIVideoPollJSON(originTask.Data)
	// Pick the parser via response shape detection, then surface URL/error.
	switch detectResponseProtocol(normalizedData) {
	case ProtocolMaaS:
		var resp maasResultResponse
		if err := common.Unmarshal(normalizedData, &resp); err != nil {
			return common.Marshal(ov)
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
			return common.Marshal(ov)
		}
		if resp.Result.Content != nil && strings.TrimSpace(resp.Result.Content.VideoURL) != "" {
			ov.SetMetadata("url", resp.Result.Content.VideoURL)
		} else if resp.Result.Output != nil && strings.TrimSpace(resp.Result.Output.VideoURL) != "" {
			ov.SetMetadata("url", resp.Result.Output.VideoURL)
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
			return common.Marshal(ov)
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
			content = append(content, maasMediaContentPart(url))
		}
		appendMaasMetadataMediaContent(&content, req.Metadata, "video_urls", "video_url")
		appendMaasMetadataMediaContent(&content, req.Metadata, "audio_urls", "audio_url")
		if len(content) > 0 {
			body["content"] = content
		}
	}

	applyMaasDefaultVideoFields(body, req)

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
		if k == "video_urls" || k == "audio_urls" {
			continue
		}
		body[k] = v
	}
	return body, nil
}

func applyMaasDefaultVideoFields(body map[string]any, req *relaycommon.TaskSubmitReq) {
	duration, resolution := extractDurationAndResolution(*req)
	if duration > 0 {
		body["duration"] = duration
	}
	if strings.TrimSpace(resolution) != "" {
		body["resolution"] = resolution
	}
	body["ratio"] = sizeToRatio(req.Size)
	body["generate_audio"] = false
}

func appendMaasMetadataMediaContent(content *[]map[string]any, metadata map[string]any, key string, contentType string) {
	if len(metadata) == 0 {
		return
	}
	appendURL := func(url string) {
		url = strings.TrimSpace(url)
		if url == "" {
			return
		}
		*content = append(*content, map[string]any{
			"type":      contentType,
			contentType: map[string]any{"url": url},
		})
	}
	switch v := metadata[key].(type) {
	case string:
		appendURL(v)
	case []string:
		for _, url := range v {
			appendURL(url)
		}
	case []any:
		for _, item := range v {
			if url, ok := item.(string); ok {
				appendURL(url)
			}
		}
	}
}

func maasMediaContentPart(url string) map[string]any {
	contentType := "image_url"
	lower := strings.ToLower(strings.TrimSpace(url))
	if i := strings.IndexAny(lower, "?#"); i >= 0 {
		lower = lower[:i]
	}
	switch {
	case strings.HasSuffix(lower, ".mp4") ||
		strings.HasSuffix(lower, ".mov") ||
		strings.HasSuffix(lower, ".avi") ||
		strings.HasSuffix(lower, ".mkv") ||
		strings.HasSuffix(lower, ".webm"):
		contentType = "video_url"
	case strings.HasSuffix(lower, ".mp3") ||
		strings.HasSuffix(lower, ".wav") ||
		strings.HasSuffix(lower, ".m4a") ||
		strings.HasSuffix(lower, ".aac") ||
		strings.HasSuffix(lower, ".ogg") ||
		strings.HasSuffix(lower, ".flac"):
		contentType = "audio_url"
	}
	return map[string]any{
		"type":      contentType,
		contentType: map[string]any{"url": url},
	}
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

func sizeToRatio(size string) string {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(size)), "x")
	if len(parts) != 2 {
		return defaultRatio
	}
	w, errW := strconv.Atoi(strings.TrimSpace(parts[0]))
	h, errH := strconv.Atoi(strings.TrimSpace(parts[1]))
	if errW != nil || errH != nil || w <= 0 || h <= 0 {
		return defaultRatio
	}
	ratio := float64(w) / float64(h)
	candidates := []struct {
		value string
		ratio float64
	}{
		{"16:9", 16.0 / 9.0},
		{"9:16", 9.0 / 16.0},
		{"1:1", 1.0},
		{"4:3", 4.0 / 3.0},
		{"3:4", 3.0 / 4.0},
		{"21:9", 21.0 / 9.0},
	}
	for _, candidate := range candidates {
		if diff := ratio - candidate.ratio; diff > -0.03 && diff < 0.03 {
			return candidate.value
		}
	}
	return defaultRatio
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
