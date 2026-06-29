package hidreamimage

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

type maasSubmitResponse struct {
	Code     int    `json:"code"`
	Message  string `json:"message,omitempty"`
	Messasge string `json:"messasge,omitempty"`
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

func oaiImage2HiDreamRequest(info *relaycommon.RelayInfo, request dto.ImageRequest) (map[string]any, error) {
	modelName := strings.TrimSpace(info.UpstreamModelName)
	if modelName == "" {
		modelName = strings.TrimSpace(request.Model)
	}
	modelID := resolveModelID(modelName)
	if modelID == "" {
		return nil, fmt.Errorf("unknown hidream image model %q, configure model_mapping to upstream model_id", modelName)
	}

	body := map[string]any{
		"model_id": modelID,
	}

	for k, raw := range request.Extra {
		if len(raw) == 0 || strings.EqualFold(k, "model_id") || strings.EqualFold(k, "model") {
			continue
		}
		var v any
		if err := common.Unmarshal(raw, &v); err != nil {
			return nil, fmt.Errorf("invalid extra field %q: %w", k, err)
		}
		body[k] = v
	}

	if _, ok := body["prompt"]; !ok && strings.TrimSpace(request.Prompt) != "" {
		body["prompt"] = strings.TrimSpace(request.Prompt)
	}

	if request.N != nil {
		if _, ok := body["n"]; !ok {
			body["n"] = int(*request.N)
		}
		info.PriceData.AddOtherRatio("n", float64(*request.N))
	}

	applyStandardFieldMappings(body, request, modelName)

	return body, nil
}

func applyStandardFieldMappings(body map[string]any, request dto.ImageRequest, modelName string) {
	size := strings.TrimSpace(request.Size)
	if size == "" {
		return
	}

	series := detectSeries(modelName)
	switch series {
	case seriesH:
		if _, ok := body["size"]; !ok {
			body["size"] = strings.NewReplacer("x", "*", "X", "*").Replace(size)
		}
	case seriesO:
		if _, ok := body["wh_ratio"]; !ok {
			if ratio := sizeToAspectRatio(size); ratio != "" {
				body["wh_ratio"] = ratio
			}
		}
	case seriesQ:
		if _, ok := body["aspect_ratio"]; !ok {
			if ratio := sizeToAspectRatio(size); ratio != "" {
				body["aspect_ratio"] = ratio
			}
		}
	default:
		if _, ok := body["size"]; !ok {
			body["size"] = strings.NewReplacer("x", "*", "X", "*").Replace(size)
		}
	}

	if len(request.Image) > 0 && !isToolModel(modelName) {
		if _, hasImage := body["image"]; hasImage {
			return
		}
		if _, hasImageList := body["image_list"]; hasImageList {
			return
		}
		var imageVal any
		if err := common.Unmarshal(request.Image, &imageVal); err == nil && imageVal != nil {
			body["image"] = imageVal
		}
	}
}

func sizeToAspectRatio(size string) string {
	parts := strings.Split(strings.ToLower(strings.TrimSpace(size)), "x")
	if len(parts) != 2 {
		return ""
	}
	w, errW := strconv.Atoi(strings.TrimSpace(parts[0]))
	h, errH := strconv.Atoi(strings.TrimSpace(parts[1]))
	if errW != nil || errH != nil || w <= 0 || h <= 0 {
		return ""
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
		{"3:2", 3.0 / 2.0},
		{"2:3", 2.0 / 3.0},
		{"21:9", 21.0 / 9.0},
	}
	for _, candidate := range candidates {
		if diff := ratio - candidate.ratio; diff > -0.03 && diff < 0.03 {
			return candidate.value
		}
	}
	return ""
}

func fetchTaskResult(info *relaycommon.RelayInfo, taskID string) ([]byte, *maasResultResponse, error) {
	baseURL := normalizeBaseURL(info.ChannelBaseUrl)
	query := url.Values{}
	query.Set("task_id", taskID)
	fullURL := fmt.Sprintf("%s/v1/images/generations/results?%s", baseURL, query.Encode())

	req, err := http.NewRequest(http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+info.ApiKey)

	client := service.GetHttpClient()
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	var result maasResultResponse
	if err := common.Unmarshal(responseBody, &result); err != nil {
		return nil, nil, err
	}
	return responseBody, &result, nil
}

func asyncTaskWait(c *gin.Context, info *relaycommon.RelayInfo, taskID string) (*maasResultResponse, []byte, error) {
	waitSeconds := 5
	maxStep := 24

	time.Sleep(2 * time.Second)

	for step := 1; step <= maxStep; step++ {
		logger.LogDebug(c, fmt.Sprintf("hidream image async wait step %d/%d", step, maxStep))
		responseBody, result, err := fetchTaskResult(info, taskID)
		if err != nil {
			logger.LogWarn(c, "hidream image fetch task err: "+err.Error())
			time.Sleep(time.Duration(waitSeconds) * time.Second)
			continue
		}

		if result.Code != 0 {
			msg := firstNonEmpty(result.Message, result.Messasge, fmt.Sprintf("code=%d", result.Code))
			return result, responseBody, errors.New(msg)
		}

		done, failed, reason := classifyMaasSubTasks(result)
		switch {
		case failed:
			return result, responseBody, errors.New(firstNonEmpty(reason, "hidream image task failed"))
		case done:
			return result, responseBody, nil
		}

		time.Sleep(time.Duration(waitSeconds) * time.Second)
	}

	return nil, nil, fmt.Errorf("hidream image task timeout")
}

func classifyMaasSubTasks(result *maasResultResponse) (done bool, failed bool, reason string) {
	if result == nil {
		return false, false, ""
	}
	if len(result.Result.SubTaskResults) == 0 {
		return false, false, ""
	}

	successCount, failureCount := 0, 0
	var firstErr string
	for _, sub := range result.Result.SubTaskResults {
		switch sub.TaskStatus {
		case 1:
			successCount++
		case 3, 4:
			failureCount++
			if firstErr == "" {
				firstErr = sub.ErrorMsg
			}
		}
	}

	total := len(result.Result.SubTaskResults)
	switch {
	case failureCount > 0:
		return false, true, firstNonEmpty(firstErr, "hidream image sub-task failed")
	case successCount == total:
		return true, false, ""
	default:
		return false, false, ""
	}
}

func responseHiDream2OpenAIImage(result *maasResultResponse, originBody []byte, info *relaycommon.RelayInfo) *dto.ImageResponse {
	imageResponse := dto.ImageResponse{
		Created:  info.StartTime.Unix(),
		Metadata: originBody,
	}
	for _, sub := range result.Result.SubTaskResults {
		if sub.TaskStatus == 1 && strings.TrimSpace(sub.URL) != "" {
			imageResponse.Data = append(imageResponse.Data, dto.ImageData{
				Url: sub.URL,
			})
		}
	}
	return &imageResponse
}

func hidreamImageHandler(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (*types.TokenFactoryError, *dto.Usage) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError), nil
	}
	service.CloseResponseBodyGracefully(resp)

	var submitResp maasSubmitResponse
	if err := common.Unmarshal(responseBody, &submitResp); err != nil {
		return types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError), nil
	}

	if submitResp.Code != 0 {
		msg := firstNonEmpty(submitResp.Message, submitResp.Messasge, fmt.Sprintf("code=%d", submitResp.Code))
		return types.NewError(errors.New(msg), types.ErrorCodeBadResponse), nil
	}

	taskID := strings.TrimSpace(submitResp.Result.TaskID)
	if taskID == "" {
		return types.NewError(errors.New("hidream image submit returned empty task_id"), types.ErrorCodeBadResponse), nil
	}

	result, originRespBody, err := asyncTaskWait(c, info, taskID)
	if err != nil {
		return types.NewError(err, types.ErrorCodeBadResponse), nil
	}

	imageResponses := responseHiDream2OpenAIImage(result, originRespBody, info)
	if len(imageResponses.Data) == 0 {
		return types.NewError(errors.New("hidream image task succeeded but returned no image url"), types.ErrorCodeBadResponse), nil
	}

	if n := len(imageResponses.Data); n > 0 {
		info.PriceData.AddOtherRatio("n", float64(n))
	}

	jsonResponse, err := common.Marshal(imageResponses)
	if err != nil {
		return types.NewError(err, types.ErrorCodeBadResponseBody), nil
	}
	service.IOCopyBytesGracefully(c, resp, jsonResponse)
	return nil, &dto.Usage{}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
