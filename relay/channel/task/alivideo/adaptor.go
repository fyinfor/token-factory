package alivideo

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

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

const contextKeyNativeBody = "alivideo_native_body"

// AliVideoMediaItem matches DashScope video-synthesis input.media entries.
type AliVideoMediaItem struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

// AliVideoInput is the upstream input object (media array + prompt).
type AliVideoInput struct {
	Prompt         string              `json:"prompt,omitempty"`
	Media          []AliVideoMediaItem `json:"media,omitempty"`
	ImgURL         string              `json:"img_url,omitempty"`
	FirstFrameURL  string              `json:"first_frame_url,omitempty"`
	LastFrameURL   string              `json:"last_frame_url,omitempty"`
	NegativePrompt string              `json:"negative_prompt,omitempty"`
}

// AliVideoParameters is the upstream parameters object.
type AliVideoParameters struct {
	Resolution   string `json:"resolution,omitempty"`
	Ratio        string `json:"ratio,omitempty"`
	Size         string `json:"size,omitempty"`
	Duration     int    `json:"duration,omitempty"`
	PromptExtend *bool  `json:"prompt_extend,omitempty"`
	Watermark    *bool  `json:"watermark,omitempty"`
}

// AliVideoRequest is the DashScope video-synthesis request body.
type AliVideoRequest struct {
	Model      string              `json:"model"`
	Input      AliVideoInput       `json:"input"`
	Parameters *AliVideoParameters `json:"parameters,omitempty"`
}

// AliVideoResponse matches DashScope async task submit / poll responses.
type AliVideoResponse struct {
	Output    AliVideoOutput `json:"output"`
	RequestID string         `json:"request_id"`
	Code      string         `json:"code,omitempty"`
	Message   string         `json:"message,omitempty"`
}

// AliVideoOutput is the task output section.
type AliVideoOutput struct {
	TaskID     string `json:"task_id"`
	TaskStatus string `json:"task_status"`
	VideoURL   string `json:"video_url,omitempty"`
	Code       string `json:"code,omitempty"`
	Message    string `json:"message,omitempty"`
}

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

func normalizeBaseURL(raw string) string {
	u := strings.TrimRight(strings.TrimSpace(raw), "/")
	if u == "" {
		u = "https://dashscope.aliyuncs.com/api"
	}
	return u
}

func joinAPIPath(baseURL, suffix string) string {
	base := normalizeBaseURL(baseURL)
	if strings.HasSuffix(strings.ToLower(base), "/api") {
		return base + suffix
	}
	return base + "/api" + suffix
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	var probe struct {
		Model string                 `json:"model"`
		Input map[string]interface{} `json:"input"`
	}
	if err := common.UnmarshalBodyReusable(c, &probe); err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	if probe.Input != nil {
		if strings.TrimSpace(probe.Model) == "" {
			return service.TaskErrorWrapperLocal(fmt.Errorf("model field is required"), "missing_model", http.StatusBadRequest)
		}
		prompt, _ := probe.Input["prompt"].(string)
		if strings.TrimSpace(prompt) == "" {
			return service.TaskErrorWrapperLocal(fmt.Errorf("input.prompt is required"), "missing_prompt", http.StatusBadRequest)
		}
		storage, err := common.GetBodyStorage(c)
		if err != nil {
			return service.TaskErrorWrapperLocal(err, "read_body_failed", http.StatusBadRequest)
		}
		body, err := storage.Bytes()
		if err != nil {
			return service.TaskErrorWrapperLocal(err, "read_body_failed", http.StatusBadRequest)
		}
		c.Set(contextKeyNativeBody, body)
		info.Action = constant.TaskActionTextGenerate
		if hasNativeInputMedia(probe.Input) {
			info.Action = constant.TaskActionGenerate
		}
		return nil
	}
	return relaycommon.ValidateBasicTaskRequest(c, info, constant.TaskActionGenerate)
}

func hasNativeInputMedia(input map[string]interface{}) bool {
	if hasMediaURL(input) {
		return true
	}
	for _, key := range []string{"img_url", "first_frame_url", "last_frame_url"} {
		if u, _ := input[key].(string); strings.TrimSpace(u) != "" {
			return true
		}
	}
	return false
}

func hasMediaURL(input map[string]interface{}) bool {
	raw, ok := input["media"]
	if !ok {
		return false
	}
	items, ok := raw.([]interface{})
	if !ok {
		return false
	}
	for _, it := range items {
		m, ok := it.(map[string]interface{})
		if !ok {
			continue
		}
		if u, _ := m["url"].(string); strings.TrimSpace(u) != "" {
			return true
		}
	}
	return false
}

func ensureMediaFromLegacyFields(input *AliVideoInput, _ string) {
	if input == nil || len(input.Media) > 0 {
		return
	}
	first := strings.TrimSpace(input.FirstFrameURL)
	if first == "" {
		first = strings.TrimSpace(input.ImgURL)
	}
	last := strings.TrimSpace(input.LastFrameURL)
	if first != "" {
		input.Media = []AliVideoMediaItem{{Type: "first_frame", URL: first}}
		if last != "" && last != first {
			input.Media = append(input.Media, AliVideoMediaItem{Type: "last_frame", URL: last})
		}
	}
}

func enrichNativeAliVideoBody(body []byte) ([]byte, error) {
	var aliReq AliVideoRequest
	if err := common.Unmarshal(body, &aliReq); err != nil {
		return nil, err
	}
	ensureMediaFromLegacyFields(&aliReq.Input, aliReq.Model)
	if len(aliReq.Input.Media) > 0 {
		profile := aliVideoMediaProfile(aliReq.Model)
		aliReq.Input.Media = finalizeAliVideoMedia(profile, normalizeAliVideoMedia(profile, aliReq.Input.Media))
	}
	if len(aliReq.Input.Media) == 0 {
		return body, nil
	}
	return common.Marshal(aliReq)
}

func (a *TaskAdaptor) BuildRequestURL(_ *relaycommon.RelayInfo) (string, error) {
	return joinAPIPath(a.baseURL, submitPath), nil
}

func (a *TaskAdaptor) BuildRequestHeader(_ *gin.Context, req *http.Request, _ *relaycommon.RelayInfo) error {
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-DashScope-Async", "enable")
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	if raw, ok := c.Get(contextKeyNativeBody); ok {
		if body, ok := raw.([]byte); ok && len(body) > 0 {
			if enriched, err := enrichNativeAliVideoBody(body); err == nil && len(enriched) > 0 && !bytes.Equal(enriched, body) {
				return bytes.NewReader(enriched), nil
			}
			return bytes.NewReader(body), nil
		}
	}

	taskReq, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, errors.Wrap(err, "get_task_request_failed")
	}

	aliReq, err := a.convertToAliRequest(info, taskReq)
	if err != nil {
		return nil, errors.Wrap(err, "convert_to_ali_video_request_failed")
	}
	logger.LogJson(c, "alivideo request body", aliReq)

	bodyBytes, err := common.Marshal(aliReq)
	if err != nil {
		return nil, errors.Wrap(err, "marshal_ali_video_request_failed")
	}
	return bytes.NewReader(bodyBytes), nil
}

func (a *TaskAdaptor) convertToAliRequest(info *relaycommon.RelayInfo, req relaycommon.TaskSubmitReq) (*AliVideoRequest, error) {
	upstreamModel := req.Model
	if info.UseRelayTaskUpstreamModel() {
		upstreamModel = info.UpstreamModelName
	}

	aliReq := &AliVideoRequest{
		Model: upstreamModel,
		Input: AliVideoInput{
			Prompt: req.Prompt,
		},
		Parameters: &AliVideoParameters{},
	}

	media := buildMediaFromTaskReq(upstreamModel, req)
	if len(media) > 0 {
		aliReq.Input.Media = media
	} else if ref := strings.TrimSpace(req.InputReference); ref != "" {
		aliReq.Input.Media = mediaItemsForReferenceURL(upstreamModel, ref)
	}

	if req.Metadata != nil {
		if err := mergeMetadataIntoAliRequest(aliReq, req.Metadata, upstreamModel); err != nil {
			return nil, err
		}
	}

	applySizeAndDuration(aliReq, req)
	ensureMediaFromLegacyFields(&aliReq.Input, upstreamModel)
	if len(aliReq.Input.Media) > 0 {
		profile := aliVideoMediaProfile(upstreamModel)
		aliReq.Input.Media = finalizeAliVideoMedia(profile, normalizeAliVideoMedia(profile, aliReq.Input.Media))
	}

	if aliReq.Model != upstreamModel {
		return nil, errors.New("can't change model with metadata")
	}

	return aliReq, nil
}

func isVideoURL(u string) bool {
	u = strings.TrimSpace(u)
	if u == "" {
		return false
	}
	lower := strings.ToLower(u)
	for _, ext := range []string{".mp4", ".mov", ".avi", ".mkv", ".webm"} {
		if strings.Contains(lower, ext) {
			return true
		}
	}
	return false
}

// aliVideoMediaProfile 根据上游模型 ID 决定 media.type 映射规则（DashScope 各子能力类型不同）。
func aliVideoMediaProfile(model string) string {
	m := strings.ToLower(strings.TrimSpace(model))
	if strings.Contains(m, "video-edit") {
		return "video-edit"
	}
	if strings.Contains(m, "-r2v") || (strings.Contains(m, "r2v") && !strings.Contains(m, "i2v")) {
		return "r2v"
	}
	if strings.Contains(m, "-i2v") || strings.Contains(m, "i2v") {
		return "i2v"
	}
	return "default"
}

// normalizeAliVideoMedia 将客户端/操练场传入的 media 规范为当前模型支持的 type。
func normalizeAliVideoMedia(profile string, items []AliVideoMediaItem) []AliVideoMediaItem {
	if len(items) == 0 {
		return items
	}
	out := make([]AliVideoMediaItem, 0, len(items))
	switch profile {
	case "r2v":
		for _, it := range items {
			if u := strings.TrimSpace(it.URL); u != "" {
				out = append(out, AliVideoMediaItem{Type: "reference_image", URL: u})
			}
		}
	case "i2v":
		for _, it := range items {
			u := strings.TrimSpace(it.URL)
			if u == "" {
				continue
			}
			switch strings.ToLower(strings.TrimSpace(it.Type)) {
			case "first_frame", "last_frame", "reference_image":
				out = append(out, AliVideoMediaItem{Type: strings.ToLower(strings.TrimSpace(it.Type)), URL: u})
			case "video":
				out = append(out, AliVideoMediaItem{Type: "reference_image", URL: u})
			default:
				if isVideoURL(u) {
					out = append(out, AliVideoMediaItem{Type: "reference_image", URL: u})
				} else {
					out = append(out, AliVideoMediaItem{Type: "first_frame", URL: u})
				}
			}
		}
	case "video-edit":
		for _, it := range items {
			u := strings.TrimSpace(it.URL)
			if u == "" {
				continue
			}
			switch strings.ToLower(strings.TrimSpace(it.Type)) {
			case "video":
				out = append(out, AliVideoMediaItem{Type: "video", URL: u})
			case "first_frame", "last_frame", "reference_image":
				out = append(out, AliVideoMediaItem{Type: "reference_image", URL: u})
			default:
				if isVideoURL(u) {
					out = append(out, AliVideoMediaItem{Type: "video", URL: u})
				} else {
					out = append(out, AliVideoMediaItem{Type: "reference_image", URL: u})
				}
			}
		}
	default:
		return finalizeAliVideoMedia(profile, items)
	}
	return finalizeAliVideoMedia(profile, out)
}

// finalizeAliVideoMedia 去重 URL，video-edit 仅保留一个 video 条目。
func finalizeAliVideoMedia(profile string, items []AliVideoMediaItem) []AliVideoMediaItem {
	if len(items) == 0 {
		return items
	}
	seenURL := make(map[string]struct{})
	out := make([]AliVideoMediaItem, 0, len(items))
	videoCount := 0
	for _, it := range items {
		u := strings.TrimSpace(it.URL)
		if u == "" {
			continue
		}
		key := strings.ToLower(u)
		if _, ok := seenURL[key]; ok {
			continue
		}
		seenURL[key] = struct{}{}
		typ := strings.ToLower(strings.TrimSpace(it.Type))
		if profile == "video-edit" && typ == "video" {
			if videoCount >= 1 {
				continue
			}
			videoCount++
		}
		out = append(out, AliVideoMediaItem{Type: typ, URL: u})
	}
	return out
}

func collectVideoURLs(req relaycommon.TaskSubmitReq) []string {
	seen := make(map[string]struct{})
	var urls []string
	add := func(raw string) {
		u := strings.TrimSpace(raw)
		if u == "" || !isVideoURL(u) {
			return
		}
		key := strings.ToLower(u)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		urls = append(urls, u)
	}
	if req.Metadata != nil {
		raw, ok := req.Metadata["video_urls"]
		if ok {
			switch arr := raw.(type) {
			case []interface{}:
				for _, it := range arr {
					if u, ok := it.(string); ok {
						add(u)
					}
				}
			case []string:
				for _, u := range arr {
					add(u)
				}
			}
		}
	}
	add(req.InputReference)
	for _, img := range req.Images {
		add(img)
	}
	return urls
}

func mediaItemsForReferenceURL(model, ref string) []AliVideoMediaItem {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil
	}
	typ := "first_frame"
	if isVideoURL(ref) {
		typ = "video"
	}
	return normalizeAliVideoMedia(aliVideoMediaProfile(model), []AliVideoMediaItem{{Type: typ, URL: ref}})
}

// buildMediaFromTaskReq 从 images / video_urls / input_reference 构建 media，并按模型规范化 type。
func buildMediaFromTaskReq(model string, req relaycommon.TaskSubmitReq) []AliVideoMediaItem {
	profile := aliVideoMediaProfile(model)
	var items []AliVideoMediaItem

	for _, u := range collectVideoURLs(req) {
		items = append(items, AliVideoMediaItem{Type: "video", URL: u})
	}

	imgs := make([]string, 0, len(req.Images))
	for _, img := range req.Images {
		if u := strings.TrimSpace(img); u != "" && !isVideoURL(u) {
			imgs = append(imgs, u)
		}
	}
	if len(imgs) > 0 {
		items = append(items, AliVideoMediaItem{Type: "first_frame", URL: imgs[0]})
		if len(imgs) == 2 {
			items = append(items, AliVideoMediaItem{Type: "last_frame", URL: imgs[1]})
		} else if len(imgs) > 2 {
			for i := 1; i < len(imgs)-1; i++ {
				items = append(items, AliVideoMediaItem{Type: "reference_image", URL: imgs[i]})
			}
			items = append(items, AliVideoMediaItem{Type: "last_frame", URL: imgs[len(imgs)-1]})
		}
	} else if ref := strings.TrimSpace(req.InputReference); ref != "" && !isVideoURL(ref) {
		items = append(items, AliVideoMediaItem{Type: "first_frame", URL: ref})
	}
	return normalizeAliVideoMedia(profile, items)
}

func mergeMetadataIntoAliRequest(aliReq *AliVideoRequest, metadata map[string]interface{}, upstreamModel string) error {
	if rawInput, ok := metadata["input"]; ok {
		b, err := common.Marshal(rawInput)
		if err != nil {
			return errors.Wrap(err, "marshal metadata.input failed")
		}
		var in AliVideoInput
		if err := common.Unmarshal(b, &in); err != nil {
			return errors.Wrap(err, "unmarshal metadata.input failed")
		}
		if strings.TrimSpace(in.Prompt) != "" {
			aliReq.Input.Prompt = in.Prompt
		}
		if len(in.Media) > 0 {
			aliReq.Input.Media = in.Media
		}
		if in.ImgURL != "" {
			aliReq.Input.ImgURL = in.ImgURL
		}
		if in.FirstFrameURL != "" {
			aliReq.Input.FirstFrameURL = in.FirstFrameURL
		}
		if in.LastFrameURL != "" {
			aliReq.Input.LastFrameURL = in.LastFrameURL
		}
		if in.NegativePrompt != "" {
			aliReq.Input.NegativePrompt = in.NegativePrompt
		}
		ensureMediaFromLegacyFields(&aliReq.Input, upstreamModel)
	}
	if rawParams, ok := metadata["parameters"]; ok {
		b, err := common.Marshal(rawParams)
		if err != nil {
			return errors.Wrap(err, "marshal metadata.parameters failed")
		}
		var params AliVideoParameters
		if err := common.Unmarshal(b, &params); err != nil {
			return errors.Wrap(err, "unmarshal metadata.parameters failed")
		}
		mergeParameters(aliReq.Parameters, &params)
	}
	// Legacy: metadata may still carry flat fields merged into the whole request.
	metaBytes, err := common.Marshal(metadata)
	if err != nil {
		return errors.Wrap(err, "marshal metadata failed")
	}
	var overlay AliVideoRequest
	if err := common.Unmarshal(metaBytes, &overlay); err != nil {
		return errors.Wrap(err, "unmarshal metadata overlay failed")
	}
	if overlay.Model != "" && overlay.Model != upstreamModel {
		return errors.New("can't change model with metadata")
	}
	if strings.TrimSpace(overlay.Input.Prompt) != "" {
		aliReq.Input.Prompt = overlay.Input.Prompt
	}
	if len(overlay.Input.Media) > 0 {
		aliReq.Input.Media = overlay.Input.Media
	}
	if overlay.Parameters != nil {
		mergeParameters(aliReq.Parameters, overlay.Parameters)
	}
	return nil
}

func mergeParameters(dst *AliVideoParameters, src *AliVideoParameters) {
	if dst == nil || src == nil {
		return
	}
	if src.Resolution != "" {
		dst.Resolution = src.Resolution
	}
	if src.Ratio != "" {
		dst.Ratio = src.Ratio
	}
	if src.Size != "" {
		dst.Size = src.Size
	}
	if src.Duration > 0 {
		dst.Duration = src.Duration
	}
	if src.PromptExtend != nil {
		dst.PromptExtend = src.PromptExtend
	}
	if src.Watermark != nil {
		dst.Watermark = src.Watermark
	}
}

func applySizeAndDuration(aliReq *AliVideoRequest, req relaycommon.TaskSubmitReq) {
	if aliReq.Parameters == nil {
		aliReq.Parameters = &AliVideoParameters{}
	}
	if ratio, res := parseSizeField(req.Size); ratio != "" {
		aliReq.Parameters.Ratio = ratio
	} else if res != "" {
		aliReq.Parameters.Resolution = res
	}
	if req.Duration > 0 {
		aliReq.Parameters.Duration = req.Duration
	} else if req.Seconds != "" {
		if seconds, err := strconv.Atoi(req.Seconds); err == nil && seconds > 0 {
			aliReq.Parameters.Duration = seconds
		}
	}
	if aliReq.Parameters.Duration <= 0 {
		aliReq.Parameters.Duration = 5
	}
	if req.Metadata != nil {
		if metaRatio, ok := req.Metadata["ratio"].(string); ok && strings.TrimSpace(metaRatio) != "" {
			aliReq.Parameters.Ratio = strings.TrimSpace(metaRatio)
		}
		if metaRes, ok := req.Metadata["resolution"].(string); ok && strings.TrimSpace(metaRes) != "" {
			aliReq.Parameters.Resolution = normalizeResolution(metaRes)
		}
	}
}

func parseSizeField(size string) (ratio string, resolution string) {
	s := strings.TrimSpace(size)
	if s == "" {
		return "", ""
	}
	if strings.Contains(s, ":") {
		return s, ""
	}
	return "", normalizeResolution(s)
}

func normalizeResolution(s string) string {
	r := strings.ToUpper(strings.TrimSpace(s))
	if r == "" {
		return ""
	}
	if !strings.HasSuffix(r, "P") && len(r) <= 5 && strings.ContainsAny(r, "0123456789") {
		return r + "P"
	}
	return r
}

func (a *TaskAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
	taskReq, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil
	}
	aliReq, err := a.convertToAliRequest(info, taskReq)
	if err != nil || aliReq.Parameters == nil {
		return nil
	}
	return map[string]float64{
		"seconds": float64(aliReq.Parameters.Duration),
	}
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

	var aliResp AliVideoResponse
	if err := common.Unmarshal(responseBody, &aliResp); err != nil {
		taskErr = service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", responseBody), "unmarshal_response_body_failed", http.StatusInternalServerError)
		return
	}

	if aliResp.Code != "" {
		taskErr = service.TaskErrorWrapper(fmt.Errorf("%s: %s", aliResp.Code, aliResp.Message), "ali_video_api_error", resp.StatusCode)
		return
	}

	if aliResp.Output.TaskID == "" {
		taskErr = service.TaskErrorWrapper(fmt.Errorf("task_id is empty"), "invalid_response", http.StatusInternalServerError)
		return
	}

	openAIResp := dto.NewOpenAIVideo()
	openAIResp.ID = info.PublicTaskID
	openAIResp.Model = c.GetString("model")
	if openAIResp.Model == "" && info != nil {
		openAIResp.Model = info.OriginModelName
	}
	openAIResp.Status = convertAliStatus(aliResp.Output.TaskStatus)
	openAIResp.CreatedAt = dto.FormatTimeUnixRFC3339(common.GetTimestamp())
	c.JSON(http.StatusOK, openAIResp)

	return aliResp.Output.TaskID, responseBody, nil
}

func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok || strings.TrimSpace(taskID) == "" {
		return nil, fmt.Errorf("invalid task_id")
	}

	uri := joinAPIPath(baseUrl, tasksPath+"/"+strings.TrimSpace(taskID))
	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)

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
	var aliResp AliVideoResponse
	if err := common.Unmarshal(respBody, &aliResp); err != nil {
		return nil, errors.Wrap(err, "unmarshal task result failed")
	}

	taskResult := relaycommon.TaskInfo{Code: 0}
	switch aliResp.Output.TaskStatus {
	case "PENDING":
		taskResult.Status = model.TaskStatusQueued
	case "RUNNING":
		taskResult.Status = model.TaskStatusInProgress
	case "SUCCEEDED":
		taskResult.Status = model.TaskStatusSuccess
		taskResult.Url = aliResp.Output.VideoURL
	case "FAILED", "CANCELED", "UNKNOWN":
		taskResult.Status = model.TaskStatusFailure
		if aliResp.Message != "" {
			taskResult.Reason = aliResp.Message
		} else if aliResp.Output.Message != "" {
			taskResult.Reason = fmt.Sprintf("task failed, code: %s, message: %s", aliResp.Output.Code, aliResp.Output.Message)
		} else {
			taskResult.Reason = "task failed"
		}
	default:
		taskResult.Status = model.TaskStatusQueued
	}
	return &taskResult, nil
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(task *model.Task) ([]byte, error) {
	var aliResp AliVideoResponse
	if err := common.Unmarshal(task.Data, &aliResp); err != nil {
		return nil, errors.Wrap(err, "unmarshal ali video response failed")
	}

	openAIResp := dto.NewOpenAIVideo()
	openAIResp.ID = task.TaskID
	openAIResp.Status = convertAliStatus(aliResp.Output.TaskStatus)
	openAIResp.Model = task.Properties.OriginModelName
	openAIResp.SetProgressStr(task.Progress)
	openAIResp.CreatedAt = dto.FormatTimeUnixRFC3339(task.CreatedAt)
	if task.FinishTime > 0 {
		openAIResp.CompletedAt = dto.FormatTimeUnixRFC3339(task.FinishTime)
	}
	openAIResp.SetMetadata("url", aliResp.Output.VideoURL)

	if aliResp.Code != "" {
		openAIResp.Error = &dto.OpenAIVideoError{Code: aliResp.Code, Message: aliResp.Message}
	} else if aliResp.Output.Code != "" {
		openAIResp.Error = &dto.OpenAIVideoError{Code: aliResp.Output.Code, Message: aliResp.Output.Message}
	}

	return common.Marshal(openAIResp)
}

func convertAliStatus(aliStatus string) string {
	switch aliStatus {
	case "PENDING":
		return dto.VideoStatusQueued
	case "RUNNING":
		return dto.VideoStatusInProgress
	case "SUCCEEDED":
		return dto.VideoStatusCompleted
	case "FAILED", "CANCELED", "UNKNOWN":
		return dto.VideoStatusFailed
	default:
		return dto.VideoStatusUnknown
	}
}
