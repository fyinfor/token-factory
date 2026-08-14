package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/types"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	green "github.com/alibabacloud-go/green-20220302/v3/client"
	"github.com/alibabacloud-go/tea/dara"
	"github.com/gin-gonic/gin"
)

const aliyunGuardrailMaxRunes = 2000

type AliyunGuardrailResult struct {
	Blocked   bool
	RiskLevel string
	Detail    string
}

func CheckAliyunGuardrailInput(c *gin.Context, info *relaycommon.RelayInfo, request dto.Request) (*AliyunGuardrailResult, error) {
	if !setting.ShouldCheckAliyunGuardrailInputForUser(aliyunGuardrailUserID(c)) || request == nil {
		return nil, nil
	}
	meta := request.GetTokenCountMeta()
	serviceName := `query_security_check`
	content := ``
	if meta != nil {
		content = truncateAliyunGuardrailContent(meta.CombineText)
	}
	params := map[string]any{`content`: content}
	if openAIRequest, ok := request.(*dto.GeneralOpenAIRequest); ok {
		var imageURL string
		var imageBase64 string
		var fileURL string
		fileCount := 0
		for _, message := range openAIRequest.Messages {
			for _, item := range message.ParseContent() {
				if image := item.GetImageMedia(); image != nil && imageURL == `` && imageBase64 == `` {
					if strings.HasPrefix(image.Url, `http`) {
						imageURL = image.Url
					}
					if strings.HasPrefix(image.Url, `data:image/`) {
						imageBase64 = image.Url
					}
				}
				if file := item.GetFile(); file != nil {
					fileCount++
					if fileCount > 1 {
						return nil, fmt.Errorf(`aliyun guardrail supports only one file per request`)
					}
					if strings.HasPrefix(file.FileData, `http`) {
						fileURL = file.FileData
					} else if file.FileData != `` {
						encoded := strings.TrimPrefix(file.FileData, `data:application/octet-stream;base64,`)
						if decoded, err := base64.StdEncoding.DecodeString(encoded); err == nil && len(decoded) > 10*1024*1024 {
							return nil, fmt.Errorf(`aliyun guardrail file size must not exceed 10MB`)
						}
					}
				}
			}
		}
		if imageURL != `` || fileURL != `` {
			serviceName = `MultiModalGuard`
		}
		if imageURL != `` {
			params[`imageUrls`] = []string{imageURL}
		}
		if fileURL != `` {
			params[`fileUrls`] = []string{fileURL}
		}
		if imageBase64 != `` {
			return checkAliyunGuardrailBase64Image(c, info, params, imageBase64, `input`)
		}
	}
	if content == `` && params[`imageUrls`] == nil && params[`fileUrls`] == nil {
		return nil, nil
	}
	return checkAliyunGuardrail(c, info, params, serviceName, `input`)
}

func CheckAliyunGuardrailTaskInput(c *gin.Context, info *relaycommon.RelayInfo, request relaycommon.TaskSubmitReq) (*AliyunGuardrailResult, error) {
	if !setting.ShouldCheckAliyunGuardrailInputForUser(aliyunGuardrailUserID(c)) {
		return nil, nil
	}
	return checkAliyunGuardrailInputMeta(c, info, request.GetModerationMeta())
}

func CheckAliyunGuardrailOutput(c *gin.Context, info *relaycommon.RelayInfo, content string) (*AliyunGuardrailResult, error) {
	if !setting.ShouldCheckAliyunGuardrailOutputForUser(aliyunGuardrailUserID(c)) || content == `` {
		return nil, nil
	}
	return checkAliyunGuardrail(c, info, map[string]any{`content`: truncateAliyunGuardrailContent(content)}, `response_security_check`, `output`)
}

func CheckAliyunGuardrailImageOutput(c *gin.Context, info *relaycommon.RelayInfo, imageURLs []string) (*AliyunGuardrailResult, error) {
	if !setting.ShouldCheckAliyunGuardrailOutputForUser(aliyunGuardrailUserID(c)) || len(imageURLs) == 0 {
		return nil, nil
	}
	return checkAliyunGuardrail(c, info, map[string]any{`imageUrls`: imageURLs[:1]}, `MultiModalGuard`, `output`)
}

// CheckAliyunVideoGuardrail submits or polls an asynchronous Alibaba Cloud video moderation task.
// It returns complete=false while the moderation result is still pending.
func CheckAliyunVideoGuardrail(ctx context.Context, task *model.Task, videoURL string) (complete bool, blocked bool, err error) {
	if task == nil || !setting.ShouldCheckAliyunGuardrailVideoForUser(task.UserId) || strings.TrimSpace(videoURL) == `` {
		return true, false, nil
	}
	client, err := newAliyunGuardrailClient()
	if err != nil {
		return false, false, fmt.Errorf(`create aliyun video guardrail client: %w`, err)
	}
	if task.PrivateData.AliyunVideoGuardrailTaskID == `` {
		payload, err := common.Marshal(map[string]string{`url`: videoURL, `dataId`: task.TaskID})
		if err != nil {
			return false, false, err
		}
		response, err := client.VideoModerationWithContext(ctx, (&green.VideoModerationRequest{}).SetService(`videoDetection`).SetServiceParameters(string(payload)), aliyunGuardrailRuntimeOptions())
		if err != nil {
			return false, false, fmt.Errorf(`submit aliyun video guardrail: %w`, err)
		}
		if response == nil || response.Body == nil || response.Body.Code == nil || *response.Body.Code != 200 || response.Body.Data == nil || response.Body.Data.TaskId == nil {
			return false, false, fmt.Errorf(`aliyun video guardrail submit returned invalid response`)
		}
		task.PrivateData.AliyunVideoGuardrailTaskID = *response.Body.Data.TaskId
		task.PrivateData.AliyunVideoGuardrailStatus = `pending`
		task.PrivateData.AliyunVideoGuardrailURL = videoURL
		return false, false, nil
	}
	payload, err := common.Marshal(map[string]string{`taskId`: task.PrivateData.AliyunVideoGuardrailTaskID})
	if err != nil {
		return false, false, err
	}
	response, err := client.VideoModerationResultWithContext(ctx, (&green.VideoModerationResultRequest{}).SetService(`videoDetection`).SetServiceParameters(string(payload)), aliyunGuardrailRuntimeOptions())
	if err != nil {
		return false, false, fmt.Errorf(`query aliyun video guardrail: %w`, err)
	}
	if response == nil || response.Body == nil || response.Body.Code == nil {
		return false, false, fmt.Errorf(`aliyun video guardrail result returned invalid response`)
	}
	if *response.Body.Code == 280 {
		return false, false, nil
	}
	if *response.Body.Code != 200 || response.Body.Data == nil {
		return false, false, fmt.Errorf(`aliyun video guardrail result unavailable`)
	}
	riskLevel := strings.ToLower(valueOrEmpty(response.Body.Data.RiskLevel))
	task.PrivateData.AliyunVideoGuardrailStatus = `passed`
	if riskLevel == `high` {
		task.PrivateData.AliyunVideoGuardrailStatus = `blocked`
		return true, true, nil
	}
	return true, false, nil
}

func aliyunGuardrailUserID(c *gin.Context) int {
	if c == nil {
		return 0
	}
	return c.GetInt(`id`)
}

func checkAliyunGuardrailInputMeta(c *gin.Context, info *relaycommon.RelayInfo, meta *types.TokenCountMeta) (*AliyunGuardrailResult, error) {
	serviceName := `query_security_check`
	content := ``
	if meta != nil {
		content = truncateAliyunGuardrailContent(meta.CombineText)
	}
	params := map[string]any{`content`: content}
	if meta != nil {
		hasImage := false
		for _, file := range meta.Files {
			if file == nil || file.Source == nil {
				continue
			}
			if file.FileType != types.FileTypeImage && file.FileType != types.FileTypeFile {
				continue
			}
			if file.FileType == types.FileTypeFile && params[`fileUrls`] != nil {
				return nil, fmt.Errorf(`aliyun guardrail supports only one file per request`)
			}
			rawData := strings.TrimSpace(file.Source.GetRawData())
			if rawData == `` {
				continue
			}
			if file.FileType == types.FileTypeImage && hasImage {
				continue
			}
			if file.Source.IsURL() {
				serviceName = `MultiModalGuard`
				if file.FileType == types.FileTypeImage {
					hasImage = true
					params[`imageUrls`] = []string{rawData}
				} else {
					params[`fileUrls`] = []string{rawData}
				}
				continue
			}
			if file.Source.IsBase64() && file.FileType == types.FileTypeImage {
				hasImage = true
				return checkAliyunGuardrailBase64Image(c, info, params, rawData, `input`)
			}
		}
	}
	if content == `` && params[`imageUrls`] == nil && params[`fileUrls`] == nil {
		return nil, nil
	}
	return checkAliyunGuardrail(c, info, params, serviceName, `input`)
}

func checkAliyunGuardrail(c *gin.Context, info *relaycommon.RelayInfo, params map[string]any, serviceName, direction string) (*AliyunGuardrailResult, error) {
	content := common.Interface2String(params[`content`])
	client, err := newAliyunGuardrailClient()
	if err != nil {
		guardrailErr := fmt.Errorf(`create aliyun guardrail client: %w`, err)
		recordAliyunGuardrailError(c, info, direction, content, serviceName, guardrailErr)
		return nil, guardrailErr
	}
	params[`chatId`] = c.GetString(common.RequestIdKey)
	payload, err := common.Marshal(params)
	if err != nil {
		recordAliyunGuardrailError(c, info, direction, content, serviceName, err)
		return nil, err
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	result := &AliyunGuardrailResult{}
	if serviceName == `MultiModalGuard` {
		multiModalService := `query_security_check`
		if direction == `output` {
			multiModalService = `response_security_check`
		}
		response, err := client.MultiModalGuardWithContext(ctx, (&green.MultiModalGuardRequest{}).SetService(multiModalService).SetServiceParameters(string(payload)), aliyunGuardrailRuntimeOptions())
		if err != nil {
			guardrailErr := fmt.Errorf(`aliyun multimodal guardrail request: %w`, err)
			recordAliyunGuardrailError(c, info, direction, content, serviceName, guardrailErr)
			return nil, guardrailErr
		}
		if response == nil || response.Body == nil || response.Body.Code == nil || *response.Body.Code != 200 || response.Body.Data == nil {
			guardrailErr := fmt.Errorf(`aliyun multimodal guardrail invalid response`)
			recordAliyunGuardrailError(c, info, direction, content, serviceName, guardrailErr)
			return nil, guardrailErr
		}
		result.RiskLevel = strings.ToLower(valueOrEmpty(response.Body.Data.Suggestion))
		result.Detail = response.Body.Data.String()
		result.Blocked = result.RiskLevel == `block`
	} else {
		response, err := client.TextModerationPlusWithContext(ctx, (&green.TextModerationPlusRequest{}).SetService(serviceName).SetServiceParameters(string(payload)), aliyunGuardrailRuntimeOptions())
		if err != nil {
			guardrailErr := fmt.Errorf(`aliyun guardrail request: %w`, err)
			recordAliyunGuardrailError(c, info, direction, content, serviceName, guardrailErr)
			return nil, guardrailErr
		}
		if response == nil || response.Body == nil || response.Body.Code == nil || *response.Body.Code != 200 || response.Body.Data == nil {
			guardrailErr := fmt.Errorf(`aliyun guardrail invalid response`)
			recordAliyunGuardrailError(c, info, direction, content, serviceName, guardrailErr)
			return nil, guardrailErr
		}
		result.RiskLevel = strings.ToLower(valueOrEmpty(response.Body.Data.RiskLevel))
		result.Detail = response.Body.Data.String()
		result.Blocked = result.RiskLevel == `high`
	}
	recordAliyunGuardrailLog(c, info, direction, content, serviceName, result)
	return result, nil
}

func checkAliyunGuardrailBase64Image(c *gin.Context, info *relaycommon.RelayInfo, params map[string]any, imageBase64, direction string) (*AliyunGuardrailResult, error) {
	const serviceName = `MultiModalGuardForBase64`
	content := common.Interface2String(params[`content`])
	client, err := newAliyunGuardrailClient()
	if err != nil {
		guardrailErr := fmt.Errorf(`create aliyun guardrail client: %w`, err)
		recordAliyunGuardrailError(c, info, direction, content, serviceName, guardrailErr)
		return nil, guardrailErr
	}
	_, imageBase64, err = DecodeBase64FileData(imageBase64)
	if err != nil {
		guardrailErr := fmt.Errorf(`decode guardrail image: %w`, err)
		recordAliyunGuardrailError(c, info, direction, content, serviceName, guardrailErr)
		return nil, guardrailErr
	}
	params[`chatId`] = c.GetString(common.RequestIdKey)
	payload, err := common.Marshal(params)
	if err != nil {
		recordAliyunGuardrailError(c, info, direction, content, serviceName, err)
		return nil, err
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	response, err := client.MultiModalGuardForBase64WithContext(ctx, (&green.MultiModalGuardForBase64Request{}).SetImageBase64Str(imageBase64).SetService(`query_security_check`).SetServiceParameters(string(payload)), aliyunGuardrailRuntimeOptions())
	if err != nil {
		guardrailErr := fmt.Errorf(`aliyun base64 image guardrail request: %w`, err)
		recordAliyunGuardrailError(c, info, direction, content, serviceName, guardrailErr)
		return nil, guardrailErr
	}
	if response == nil || response.Body == nil || response.Body.Code == nil || *response.Body.Code != 200 || response.Body.Data == nil {
		guardrailErr := fmt.Errorf(`aliyun base64 image guardrail invalid response`)
		recordAliyunGuardrailError(c, info, direction, content, serviceName, guardrailErr)
		return nil, guardrailErr
	}
	result := &AliyunGuardrailResult{RiskLevel: strings.ToLower(valueOrEmpty(response.Body.Data.Suggestion)), Detail: response.Body.Data.String()}
	result.Blocked = result.RiskLevel == `block`
	recordAliyunGuardrailLog(c, info, direction, content, serviceName, result)
	return result, nil
}

func truncateAliyunGuardrailContent(content string) string {
	if utf8.RuneCountInString(content) <= aliyunGuardrailMaxRunes {
		return content
	}
	return string([]rune(content)[:aliyunGuardrailMaxRunes])
}

func aliyunGuardrailRuntimeOptions() *dara.RuntimeOptions {
	return &dara.RuntimeOptions{}
}

func newAliyunGuardrailClient() (*green.Client, error) {
	config := (&openapi.Config{}).
		SetAccessKeyId(strings.TrimSpace(setting.AliyunGuardrailAccessKeyID)).
		SetAccessKeySecret(strings.TrimSpace(setting.AliyunGuardrailAccessKeySecret)).
		SetRegionId(strings.TrimSpace(setting.AliyunGuardrailRegionID)).
		SetConnectTimeout(3000).
		SetReadTimeout(3000)
	return green.NewClient(config)
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ``
	}
	return *value
}

func recordAliyunGuardrailLog(c *gin.Context, info *relaycommon.RelayInfo, direction, content, serviceName string, result *AliyunGuardrailResult) {
	entry := model.NewAliyunGuardrailLog()
	entry.UserId = c.GetInt(`id`)
	entry.Username = c.GetString(`username`)
	entry.RequestId = c.GetString(common.RequestIdKey)
	entry.Direction = direction
	entry.RiskLevel = result.RiskLevel
	entry.Service = serviceName
	entry.Content = content
	entry.Detail = result.Detail
	if info != nil {
		entry.ModelName = info.OriginModelName
		if info.ChannelMeta != nil {
			entry.ChannelId = info.ChannelMeta.ChannelId
		}
	}
	if err := model.CreateAliyunGuardrailLog(entry); err != nil {
		common.SysError(`failed to record aliyun guardrail log: ` + err.Error())
	}
}

func recordAliyunGuardrailError(c *gin.Context, info *relaycommon.RelayInfo, direction, content, serviceName string, err error) {
	recordAliyunGuardrailLog(c, info, direction, content, serviceName, &AliyunGuardrailResult{
		RiskLevel: `error`,
		Detail:    err.Error(),
	})
}
