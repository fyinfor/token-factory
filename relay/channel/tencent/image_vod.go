package tencent

import (
	"errors"
	"fmt"
	"io"
	"net/http"
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
	for k, raw := range request.Extra {
		if len(raw) == 0 {
			continue
		}
		var v any
		if err := common.Unmarshal(raw, &v); err == nil {
			body[k] = v
		}
	}
	return body, nil
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

	urls := pollTencentImageURLs(info, taskID, 40, 2*time.Second)
	if len(urls) == 0 {
		return nil, types.NewError(errors.New("tencent image task completed but no image url returned"), types.ErrorCodeBadResponseBody)
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

func pollTencentImageURLs(info *relaycommon.RelayInfo, taskID string, maxRetry int, interval time.Duration) []string {
	cred, err := tasktencentvod.ParseCredentials(info.ApiKey)
	if err != nil {
		return nil
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
					Output *struct {
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
		if describe.Response.AigcImageTask != nil && describe.Response.AigcImageTask.Output != nil {
			urls := make([]string, 0)
			for _, fi := range describe.Response.AigcImageTask.Output.FileInfos {
				u := strings.TrimSpace(ptrString(fi.FileUrl))
				if u != "" {
					urls = append(urls, u)
				}
			}
			if len(urls) > 0 {
				return urls
			}
		}
		if describe.Response.Status != nil && strings.ToUpper(strings.TrimSpace(*describe.Response.Status)) == "ABORTED" {
			return nil
		}
		time.Sleep(interval)
	}
	return nil
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

