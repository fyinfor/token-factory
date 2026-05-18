package tencent

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	tasktencentvod "github.com/QuantumNous/new-api/relay/channel/task/tencentvod"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

func buildTencentVODImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (map[string]any, error) {
	cred, err := tasktencentvod.ParseCredentials(common.GetContextKeyString(c, constant.ContextKeyChannelKey))
	if err != nil {
		return nil, err
	}
	modelID := strings.TrimSpace(info.UpstreamModelName)
	if modelID == "" {
		modelID = strings.TrimSpace(request.Model)
	}
	modelName, modelVersion := tasktencentvod.SplitCombinedModel(modelID)
	if modelName == "" || modelVersion == "" {
		return nil, fmt.Errorf("invalid model %q, expected ModelName-ModelVersion", modelID)
	}
	prompt := strings.TrimSpace(request.Prompt)
	if prompt == "" {
		return nil, errors.New("prompt is required")
	}
	body := map[string]any{
		"SubAppId":     cred.SubAppID,
		"ModelName":    modelName,
		"ModelVersion": modelVersion,
		"Prompt":       prompt,
	}
	enrichTencentVODImageBody(body, modelName, request)
	return body, nil
}

func enrichTencentVODImageBody(body map[string]any, modelName string, request dto.ImageRequest) {
	outputConfig := map[string]any{
		"StorageMode": "Temporary",
	}
	if request.N != nil && *request.N > 0 {
		outputConfig["OutputImageCount"] = capTencentOutputImageCount(modelName, int(*request.N))
	}
	sizeForUpstream := tencentSizeForUpstream(strings.TrimSpace(request.Size))
	applyTencentImageSizeToOutput(modelName, sizeForUpstream, outputConfig)

	for k, raw := range request.Extra {
		if len(raw) == 0 {
			continue
		}
		if strings.EqualFold(k, "OutputConfig") {
			var userOutput map[string]any
			if err := common.Unmarshal(raw, &userOutput); err == nil {
				outputConfig = mergeTencentOutputConfig(outputConfig, userOutput)
			}
			continue
		}
		if strings.EqualFold(k, "ExtInfo") {
			if ext := mergeTencentExtInfoSize(sizeForUpstream, raw); ext != "" {
				body["ExtInfo"] = ext
			}
			continue
		}
		var v any
		if err := common.Unmarshal(raw, &v); err == nil {
			body[k] = v
		}
	}
	if len(outputConfig) > 0 {
		body["OutputConfig"] = outputConfig
	}
	if _, ok := body["ExtInfo"]; !ok {
		if ext := buildTencentExtInfoSize(sizeForUpstream); ext != "" {
			body["ExtInfo"] = ext
		}
	}
}

const tencentImageSizeAlign = 16

// tencentSizeForUpstream normalizes WxH so both dimensions are divisible by 16 (Tencent GPT image API requirement).
func tencentSizeForUpstream(size string) string {
	if normalized, ok := normalizeTencentImageSizeString(size); ok {
		return normalized
	}
	return strings.TrimSpace(size)
}

func alignTencentDimension(n int) int {
	if n <= 0 {
		return n
	}
	rem := n % tencentImageSizeAlign
	if rem == 0 {
		return n
	}
	down := n - rem
	up := down + tencentImageSizeAlign
	if rem >= tencentImageSizeAlign/2 {
		return up
	}
	if down < tencentImageSizeAlign {
		return tencentImageSizeAlign
	}
	return down
}

func normalizeTencentImageSizeString(size string) (string, bool) {
	w, h, ok := parseTencentImageSize(size)
	if !ok {
		return "", false
	}
	return fmt.Sprintf("%dx%d", alignTencentDimension(w), alignTencentDimension(h)), true
}

func capTencentOutputImageCount(modelName string, n int) int {
	if n < 1 {
		return 1
	}
	max := 10
	switch strings.ToUpper(strings.TrimSpace(modelName)) {
	case "OG":
		max = 8
	case "KLING":
		max = 9
	}
	if n > max {
		return max
	}
	return n
}

func mergeTencentOutputConfig(base, override map[string]any) map[string]any {
	out := make(map[string]any, len(base)+len(override))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range override {
		out[k] = v
	}
	return out
}

func applyTencentImageSizeToOutput(modelName, size string, outputConfig map[string]any) {
	w, h, ok := parseTencentImageSize(size)
	if !ok {
		return
	}
	if ar := tencentAspectRatioFromWH(w, h); ar != "" {
		outputConfig["AspectRatio"] = ar
	}
	if res := tencentResolutionFromWH(modelName, w, h); res != "" {
		outputConfig["Resolution"] = res
	}
}

func parseTencentImageSize(size string) (int, int, bool) {
	size = strings.ToLower(strings.TrimSpace(size))
	size = strings.ReplaceAll(size, " ", "")
	if size == "" {
		return 0, 0, false
	}
	parts := strings.Split(size, "x")
	if len(parts) != 2 {
		return 0, 0, false
	}
	w, errW := strconv.Atoi(parts[0])
	h, errH := strconv.Atoi(parts[1])
	if errW != nil || errH != nil || w <= 0 || h <= 0 {
		return 0, 0, false
	}
	return w, h, true
}

func tencentAspectRatioFromWH(w, h int) string {
	g := gcdInt(w, h)
	if g <= 0 {
		return ""
	}
	return fmt.Sprintf("%d:%d", w/g, h/g)
}

func tencentResolutionFromWH(modelName string, w, h int) string {
	maxEdge := w
	if h > maxEdge {
		maxEdge = h
	}
	switch strings.ToUpper(strings.TrimSpace(modelName)) {
	case "OG", "GG", "SI", "VIDU":
		switch {
		case maxEdge >= 3500:
			return "4K"
		case maxEdge >= 1900:
			return "2K"
		default:
			return "1080P"
		}
	case "KLING":
		switch {
		case maxEdge >= 3500:
			return "4k"
		case maxEdge >= 1900:
			return "2k"
		default:
			return "1k"
		}
	default:
		return ""
	}
}

func gcdInt(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	if a < 0 {
		return -a
	}
	return a
}

func buildTencentExtInfoSize(size string) string {
	size = strings.TrimSpace(size)
	if size == "" {
		return ""
	}
	additional, err := common.Marshal(map[string]string{"size": size})
	if err != nil {
		return ""
	}
	ext, err := common.Marshal(map[string]string{"AdditionalParameters": string(additional)})
	if err != nil {
		return ""
	}
	return string(ext)
}

func mergeTencentExtInfoSize(size string, raw json.RawMessage) string {
	size = strings.TrimSpace(size)
	var ext map[string]any
	if err := common.Unmarshal(raw, &ext); err != nil || ext == nil {
		if size == "" {
			return ""
		}
		return buildTencentExtInfoSize(size)
	}
	if size != "" {
		ap := map[string]string{"size": size}
		if existing, ok := ext["AdditionalParameters"].(string); ok && strings.TrimSpace(existing) != "" {
			var parsed map[string]string
			if err := common.Unmarshal([]byte(existing), &parsed); err == nil && parsed != nil {
				parsed["size"] = size
				ap = parsed
			}
		}
		additional, err := common.Marshal(ap)
		if err == nil {
			ext["AdditionalParameters"] = string(additional)
		}
	}
	out, err := common.Marshal(ext)
	if err != nil {
		return ""
	}
	return string(out)
}

func doTencentVODImageRequest(info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	payload, err := io.ReadAll(requestBody)
	if err != nil {
		return nil, err
	}
	cred, err := tasktencentvod.ParseCredentials(info.ApiKey)
	if err != nil {
		return nil, err
	}
	endpoint := normalizeVodEndpoint(info.ChannelBaseUrl)
	return tasktencentvod.SignedPOSTJSON(strings.TrimSpace(info.ChannelSetting.Proxy), endpoint, cred.Region, cred, "CreateAigcImageTask", payload)
}

func handleTencentVODImageResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (any, *types.TokenFactoryError) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}
	service.CloseResponseBodyGracefully(resp)
	var create struct {
		Response *struct {
			TaskID *string `json:"TaskId,omitempty"`
			Error  *struct {
				Code    string `json:"Code,omitempty"`
				Message string `json:"Message,omitempty"`
			} `json:"Error,omitempty"`
		} `json:"Response,omitempty"`
	}
	if err = common.Unmarshal(body, &create); err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	if create.Response == nil {
		return nil, types.NewError(errors.New("empty create image response"), types.ErrorCodeBadResponseBody)
	}
	if create.Response.Error != nil && strings.TrimSpace(create.Response.Error.Message) != "" {
		return nil, types.WithOpenAIError(types.OpenAIError{Message: create.Response.Error.Message, Code: create.Response.Error.Code, Type: "tencent_vod_error"}, http.StatusBadRequest)
	}
	taskID := strings.TrimSpace(ptrString(create.Response.TaskID))
	if taskID == "" {
		return nil, types.NewError(errors.New("missing task id in create image response"), types.ErrorCodeBadResponseBody)
	}

	urls, pollErr := pollTencentImageURLs(info, taskID, 120, 3*time.Second)
	if pollErr != nil {
		return nil, pollErr
	}
	if len(urls) == 0 {
		return nil, types.NewError(errors.New("tencent image task timed out after polling"), types.ErrorCodeBadResponseBody)
	}

	out := dto.ImageResponse{Created: common.GetTimestamp(), Data: make([]dto.ImageData, 0, len(urls))}
	for _, u := range urls {
		out.Data = append(out.Data, dto.ImageData{Url: u})
	}
	data, err := common.Marshal(out)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeBadResponseBody)
	}
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(http.StatusOK)
	service.IOCopyBytesGracefully(c, resp, data)
	return &dto.Usage{}, nil
}

func pollTencentImageURLs(info *relaycommon.RelayInfo, taskID string, maxRetry int, interval time.Duration) ([]string, *types.TokenFactoryError) {
	cred, err := tasktencentvod.ParseCredentials(info.ApiKey)
	if err != nil {
		return nil, types.NewError(err, types.ErrorCodeBadResponseBody)
	}
	payload, _ := common.Marshal(map[string]any{"TaskId": taskID, "SubAppId": cred.SubAppID})
	endpoint := normalizeVodEndpoint(info.ChannelBaseUrl)
	for i := 0; i < maxRetry; i++ {
		resp, reqErr := tasktencentvod.SignedPOSTJSON(strings.TrimSpace(info.ChannelSetting.Proxy), endpoint, cred.Region, cred, "DescribeTaskDetail", payload)
		if reqErr != nil || resp == nil {
			time.Sleep(interval)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		var describe struct {
			Response *struct {
				Status        *string `json:"Status,omitempty"`
				AigcImageTask *struct {
					ErrCode    int     `json:"ErrCode"`
					ErrCodeExt string  `json:"ErrCodeExt"`
					Message    *string `json:"Message,omitempty"`
					Output     *struct {
						FileInfos []struct {
							FileUrl *string `json:"FileUrl,omitempty"`
						} `json:"FileInfos,omitempty"`
					} `json:"Output,omitempty"`
				} `json:"AigcImageTask,omitempty"`
			} `json:"Response,omitempty"`
		}
		if err = common.Unmarshal(body, &describe); err != nil || describe.Response == nil {
			time.Sleep(interval)
			continue
		}

		// Check for task-level error first
		if describe.Response.AigcImageTask != nil && describe.Response.AigcImageTask.ErrCode != 0 {
			errMsg := fmt.Sprintf("tencent image task failed (ErrCode=%d, ErrCodeExt=%s)", describe.Response.AigcImageTask.ErrCode, describe.Response.AigcImageTask.ErrCodeExt)
			if describe.Response.AigcImageTask.Message != nil && strings.TrimSpace(*describe.Response.AigcImageTask.Message) != "" {
				errMsg = fmt.Sprintf("tencent image task failed: %s (ErrCode=%d, ErrCodeExt=%s)", strings.TrimSpace(*describe.Response.AigcImageTask.Message), describe.Response.AigcImageTask.ErrCode, describe.Response.AigcImageTask.ErrCodeExt)
			}
			return nil, types.NewError(errors.New(errMsg), types.ErrorCodeBadResponseBody)
		}

		// Check for completed image URLs
		if describe.Response.AigcImageTask != nil && describe.Response.AigcImageTask.Output != nil {
			urls := make([]string, 0)
			for _, fi := range describe.Response.AigcImageTask.Output.FileInfos {
				u := strings.TrimSpace(ptrString(fi.FileUrl))
				if u != "" {
					urls = append(urls, u)
				}
			}
			if len(urls) > 0 {
				return urls, nil
			}
		}

		// Check terminal statuses
		if describe.Response.Status != nil {
			upperStatus := strings.ToUpper(strings.TrimSpace(*describe.Response.Status))
			if upperStatus == "ABORTED" {
				return nil, types.NewError(errors.New("tencent image task was aborted"), types.ErrorCodeBadResponseBody)
			}
			if upperStatus == "FINISH" {
				return nil, types.NewError(errors.New("tencent image task finished but no image url returned"), types.ErrorCodeBadResponseBody)
			}
		}

		time.Sleep(interval)
	}
	return nil, nil
}

func ptrString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
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
