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
	"github.com/QuantumNous/new-api/relay/helper"
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
	explicitRatio := extractTencentImageRatio(request.Extra)
	applyTencentImageSizeToOutput(modelName, sizeForUpstream, outputConfig, explicitRatio)

	for k, raw := range request.Extra {
		if len(raw) == 0 {
			continue
		}
		// 腾讯云 CreateAigcImageTask 不识别顶层 ratio；比例写入 OutputConfig.AspectRatio。
		if strings.EqualFold(k, "ratio") || strings.EqualFold(k, "aspect_ratio") {
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

func extractTencentImageRatio(extra map[string]json.RawMessage) string {
	if len(extra) == 0 {
		return ""
	}
	for _, key := range []string{"ratio", "aspect_ratio"} {
		raw, ok := extra[key]
		if !ok || len(raw) == 0 {
			continue
		}
		var s string
		if err := common.Unmarshal(raw, &s); err != nil {
			continue
		}
		s = strings.TrimSpace(s)
		if s == "" || strings.EqualFold(s, "auto") {
			continue
		}
		return s
	}
	return ""
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

func applyTencentImageSizeToOutput(modelName, size string, outputConfig map[string]any, explicitRatio string) {
	if ar := strings.TrimSpace(explicitRatio); ar != "" {
		outputConfig["AspectRatio"] = ar
	}
	w, h, ok := parseTencentImageSize(size)
	if !ok {
		return
	}
	if _, hasAR := outputConfig["AspectRatio"]; !hasAR {
		if ar := tencentAspectRatioFromWH(w, h); ar != "" {
			outputConfig["AspectRatio"] = ar
		}
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
	if w <= 0 || h <= 0 {
		return ""
	}
	ratio := float64(w) / float64(h)
	// 尺寸对齐到 16 倍数后（如 854x480→848x480）纯 GCD 会得到非法比例（53:30），
	// 优先匹配标准画幅，避免上游 UnknownParameter / 非法 AspectRatio。
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

	pollResult, pollErr := pollTencentImageTask(info, taskID, 120, 3*time.Second)
	if pollErr != nil {
		return nil, pollErr
	}
	if pollResult == nil || len(pollResult.URLs) == 0 {
		return nil, types.NewError(errors.New("tencent image task timed out after polling"), types.ErrorCodeBadResponseBody)
	}

	applyTencentImageActualBilling(info, pollResult)
	if pollResult.Width > 0 && pollResult.Height > 0 {
		count := pollResult.OutputImageCount
		if count <= 0 {
			count = len(pollResult.URLs)
		}
		helper.ApplyActualImageDimensionsForBilling(c, info, pollResult.Width, pollResult.Height, count)
	}

	out := dto.ImageResponse{Created: common.GetTimestamp(), Data: make([]dto.ImageData, 0, len(pollResult.URLs))}
	for _, u := range pollResult.URLs {
		out.Data = append(out.Data, dto.ImageData{Url: u})
	}
	if meta, metaErr := buildTencentImageCommonMetadata(pollResult, info); metaErr == nil && len(meta) > 0 {
		out.Metadata = meta
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

// tencentImagePollResult holds DescribeTaskDetail fields mapped for common image response + billing.
type tencentImagePollResult struct {
	URLs             []string
	Status           string
	Progress         int
	CreateTime       string
	FinishTime       string
	StorageMode      string
	Resolution       string
	AspectRatio      string
	OutputImageCount int
	Width            int
	Height           int
	RequestId        string
}

// tencentImageCommonMetadata is project-common image metadata returned via ImageResponse.Metadata.
type tencentImageCommonMetadata struct {
	Status           string `json:"status,omitempty"`
	Progress         int    `json:"progress,omitempty"`
	CreateTime       string `json:"create_time,omitempty"`
	FinishTime       string `json:"finish_time,omitempty"`
	StorageMode      string `json:"storage_mode,omitempty"`
	Resolution       string `json:"resolution,omitempty"` // 计费档位：1080p（≡1K）/ 2K / 4K
	Size             string `json:"size,omitempty"`       // 实际像素尺寸：1360x768
	AspectRatio      string `json:"aspect_ratio,omitempty"`
	OutputImageCount int    `json:"output_image_count,omitempty"`
	Width            int    `json:"width,omitempty"`
	Height           int    `json:"height,omitempty"`
	RequestId        string `json:"request_id,omitempty"`
}

func buildTencentImageCommonMetadata(r *tencentImagePollResult, info *relaycommon.RelayInfo) (json.RawMessage, error) {
	if r == nil {
		return nil, nil
	}
	size := ""
	if r.Width > 0 && r.Height > 0 {
		size = fmt.Sprintf("%dx%d", r.Width, r.Height)
	}
	meta := tencentImageCommonMetadata{
		Status:           strings.TrimSpace(r.Status),
		Progress:         r.Progress,
		CreateTime:       strings.TrimSpace(r.CreateTime),
		FinishTime:       strings.TrimSpace(r.FinishTime),
		StorageMode:      strings.TrimSpace(r.StorageMode),
		Resolution:       resolveTencentImageBillingTierLabel(r, info),
		Size:             size,
		AspectRatio:      strings.TrimSpace(r.AspectRatio),
		OutputImageCount: r.OutputImageCount,
		Width:            r.Width,
		Height:           r.Height,
		RequestId:        strings.TrimSpace(r.RequestId),
	}
	return common.Marshal(meta)
}

// resolveTencentImageBillingTierLabel 优先用已匹配的按张计费档位，其次用实际像素推算，最后回退上游 OutputConfig.Resolution。
func resolveTencentImageBillingTierLabel(r *tencentImagePollResult, info *relaycommon.RelayInfo) string {
	if info != nil && info.ImageBilling != nil {
		if res := strings.TrimSpace(info.ImageBilling.RuleRes); res != "" {
			if label := common.FormatImageResolutionLabel(res); label != "" {
				return label
			}
			return res
		}
		if info.ImageBilling.RuleWidth > 0 && info.ImageBilling.RuleHeight > 0 {
			if label := common.FormatImageResolutionLabel(fmt.Sprintf("%dx%d", info.ImageBilling.RuleWidth, info.ImageBilling.RuleHeight)); label != "" {
				return label
			}
		}
	}
	if r != nil && r.Width > 0 && r.Height > 0 {
		if label := common.FormatImageResolutionLabel(fmt.Sprintf("%dx%d", r.Width, r.Height)); label != "" {
			return label
		}
	}
	if r != nil {
		return common.FormatImageResolutionLabel(strings.TrimSpace(r.Resolution))
	}
	return ""
}

func applyTencentImageActualBilling(info *relaycommon.RelayInfo, r *tencentImagePollResult) {
	if info == nil || r == nil {
		return
	}
	if rid := strings.TrimSpace(r.RequestId); rid != "" {
		info.UpstreamRequestId = rid
	}
	if info.ImageBilling == nil {
		return
	}
	if r.Width > 0 && r.Height > 0 {
		info.ImageBilling.Width = r.Width
		info.ImageBilling.Height = r.Height
		info.ImageBilling.DimensionsFromUpstream = true
	}
	if r.OutputImageCount > 0 {
		info.ImageBilling.Count = r.OutputImageCount
	} else if n := len(r.URLs); n > 0 {
		info.ImageBilling.Count = n
	}
}

func pollTencentImageTask(info *relaycommon.RelayInfo, taskID string, maxRetry int, interval time.Duration) (*tencentImagePollResult, *types.TokenFactoryError) {
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

		parsed, parseErr := parseTencentDescribeImageTask(body)
		if parseErr != nil || parsed == nil {
			time.Sleep(interval)
			continue
		}

		if parsed.taskErr != "" {
			return nil, types.NewError(errors.New(parsed.taskErr), types.ErrorCodeBadResponseBody)
		}
		if len(parsed.result.URLs) > 0 {
			return &parsed.result, nil
		}

		upperStatus := strings.ToUpper(strings.TrimSpace(parsed.result.Status))
		if upperStatus == "ABORTED" {
			return nil, types.NewError(errors.New("tencent image task was aborted"), types.ErrorCodeBadResponseBody)
		}
		if upperStatus == "FINISH" {
			return nil, types.NewError(errors.New("tencent image task finished but no image url returned"), types.ErrorCodeBadResponseBody)
		}

		time.Sleep(interval)
	}
	return nil, nil
}

type parsedTencentDescribeImage struct {
	result  tencentImagePollResult
	taskErr string
}

func parseTencentDescribeImageTask(body []byte) (*parsedTencentDescribeImage, error) {
	var describe struct {
		Response *struct {
			Status           *string `json:"Status,omitempty"`
			CreateTime       *string `json:"CreateTime,omitempty"`
			FinishTime       *string `json:"FinishTime,omitempty"`
			RequestId        *string `json:"RequestId,omitempty"`
			AigcImageTask    *struct {
				Status   *string `json:"Status,omitempty"`
				ErrCode  int     `json:"ErrCode"`
				ErrCodeExt string `json:"ErrCodeExt"`
				Message  *string `json:"Message,omitempty"`
				Progress *int    `json:"Progress,omitempty"`
				Input    *struct {
					OutputConfig *struct {
						StorageMode      *string `json:"StorageMode,omitempty"`
						Resolution       *string `json:"Resolution,omitempty"`
						AspectRatio      *string `json:"AspectRatio,omitempty"`
						OutputImageCount *int    `json:"OutputImageCount,omitempty"`
					} `json:"OutputConfig,omitempty"`
				} `json:"Input,omitempty"`
				Output *struct {
					FileInfos []struct {
						FileUrl     *string `json:"FileUrl,omitempty"`
						StorageMode *string `json:"StorageMode,omitempty"`
						MetaData    *struct {
							Height *int `json:"Height,omitempty"`
							Width  *int `json:"Width,omitempty"`
							VideoStreamSet []struct {
								Height int `json:"Height"`
								Width  int `json:"Width"`
							} `json:"VideoStreamSet,omitempty"`
						} `json:"MetaData,omitempty"`
					} `json:"FileInfos,omitempty"`
				} `json:"Output,omitempty"`
			} `json:"AigcImageTask,omitempty"`
		} `json:"Response,omitempty"`
	}
	if err := common.Unmarshal(body, &describe); err != nil {
		return nil, err
	}
	if describe.Response == nil {
		return nil, errors.New("empty describe response")
	}

	out := &parsedTencentDescribeImage{}
	out.result.Status = strings.TrimSpace(ptrString(describe.Response.Status))
	out.result.CreateTime = strings.TrimSpace(ptrString(describe.Response.CreateTime))
	out.result.FinishTime = strings.TrimSpace(ptrString(describe.Response.FinishTime))
	out.result.RequestId = strings.TrimSpace(ptrString(describe.Response.RequestId))

	task := describe.Response.AigcImageTask
	if task == nil {
		return out, nil
	}

	if task.ErrCode != 0 {
		errMsg := fmt.Sprintf("tencent image task failed (ErrCode=%d, ErrCodeExt=%s)", task.ErrCode, task.ErrCodeExt)
		if task.Message != nil && strings.TrimSpace(*task.Message) != "" {
			errMsg = fmt.Sprintf("tencent image task failed: %s (ErrCode=%d, ErrCodeExt=%s)", strings.TrimSpace(*task.Message), task.ErrCode, task.ErrCodeExt)
		}
		out.taskErr = errMsg
		return out, nil
	}

	if st := strings.TrimSpace(ptrString(task.Status)); st != "" {
		out.result.Status = st
	}
	if task.Progress != nil {
		out.result.Progress = *task.Progress
	}

	if task.Input != nil && task.Input.OutputConfig != nil {
		oc := task.Input.OutputConfig
		out.result.StorageMode = strings.TrimSpace(ptrString(oc.StorageMode))
		out.result.Resolution = strings.TrimSpace(ptrString(oc.Resolution))
		out.result.AspectRatio = strings.TrimSpace(ptrString(oc.AspectRatio))
		if oc.OutputImageCount != nil && *oc.OutputImageCount > 0 {
			out.result.OutputImageCount = *oc.OutputImageCount
		}
	}

	if task.Output != nil {
		urls := make([]string, 0, len(task.Output.FileInfos))
		for _, fi := range task.Output.FileInfos {
			u := strings.TrimSpace(ptrString(fi.FileUrl))
			if u != "" {
				urls = append(urls, u)
			}
			if out.result.StorageMode == "" {
				out.result.StorageMode = strings.TrimSpace(ptrString(fi.StorageMode))
			}
			if out.result.Width > 0 && out.result.Height > 0 {
				continue
			}
			w, h := 0, 0
			if fi.MetaData != nil {
				if fi.MetaData.Width != nil {
					w = *fi.MetaData.Width
				}
				if fi.MetaData.Height != nil {
					h = *fi.MetaData.Height
				}
				if (w <= 0 || h <= 0) && len(fi.MetaData.VideoStreamSet) > 0 {
					for _, vs := range fi.MetaData.VideoStreamSet {
						if vs.Width > 0 && vs.Height > 0 {
							w, h = vs.Width, vs.Height
							break
						}
					}
				}
			}
			if w > 0 && h > 0 {
				out.result.Width = w
				out.result.Height = h
			}
		}
		out.result.URLs = urls
	}
	return out, nil
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
