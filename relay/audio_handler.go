package relay

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

func AudioHelper(c *gin.Context, info *relaycommon.RelayInfo) (tokenFactoryError *types.TokenFactoryError) {
	info.InitChannelMeta(c)

	audioReq, ok := info.Request.(*dto.AudioRequest)
	if !ok {
		return types.NewError(errors.New("invalid request type"), types.ErrorCodeInvalidRequest, types.ErrOptionWithSkipRetry())
	}

	request, err := common.DeepCopy(audioReq)
	if err != nil {
		return types.NewError(fmt.Errorf("failed to copy request to AudioRequest: %w", err), types.ErrorCodeInvalidRequest, types.ErrOptionWithSkipRetry())
	}

	err = helper.ModelMappedHelper(c, info, request)
	if err != nil {
		return types.NewError(err, types.ErrorCodeChannelModelMappedError, types.ErrOptionWithSkipRetry())
	}

	adaptor := GetAdaptor(info.ApiType)
	if adaptor == nil {
		return types.NewError(fmt.Errorf("invalid api type: %d", info.ApiType), types.ErrorCodeInvalidApiType, types.ErrOptionWithSkipRetry())
	}
	adaptor.Init(info)

	var requestBody io.Reader
	// 同步 ASR 透传仅适用于 application/json 且已是上游原生体（如 DashScope input.messages）。
	// multipart / OpenAI 兼容 audio_url JSON 必须走 ConvertAudioRequest，否则上游会直接非 200。
	passThroughEnabled := model_setting.GetGlobalSettings().PassThroughRequestEnabled || info.ChannelSetting.PassThroughBodyEnabled
	passThroughJSON := passThroughEnabled && strings.HasPrefix(strings.ToLower(c.ContentType()), "application/json")
	if passThroughJSON {
		storage, err := common.GetBodyStorage(c)
		if err != nil {
			return types.NewErrorWithStatusCode(err, types.ErrorCodeReadRequestBodyFailed, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
		}
		body, err := storage.Bytes()
		if err != nil {
			return types.NewErrorWithStatusCode(err, types.ErrorCodeReadRequestBodyFailed, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
		}
		if audioJSONHasNativeInputMessages(body) {
			upstreamModel := strings.TrimSpace(request.Model)
			if upstreamModel == "" {
				upstreamModel = strings.TrimSpace(info.UpstreamModelName)
			}
			if upstreamModel != "" {
				rewritten, rewriteErr := rewriteAudioJSONModelField(body, upstreamModel)
				if rewriteErr != nil {
					return types.NewError(fmt.Errorf("透传改写 model 失败: %w", rewriteErr), types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
				}
				body = rewritten
			}
			if common.DebugEnabled {
				println("audio requestBody: ", string(body))
			}
			requestBody = bytes.NewReader(body)
		} else {
			// 透传开关打开但 body 仍是 OpenAI 兼容格式：交给渠道转换
			ioReader, err := adaptor.ConvertAudioRequest(c, info, *request)
			if err != nil {
				return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
			}
			requestBody = ioReader
		}
	} else {
		ioReader, err := adaptor.ConvertAudioRequest(c, info, *request)
		if err != nil {
			return types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
		}
		requestBody = ioReader
	}

	resp, err := adaptor.DoRequest(c, info, requestBody)
	if err != nil {
		return types.NewError(err, types.ErrorCodeDoRequestFailed)
	}
	statusCodeMappingStr := c.GetString("status_code_mapping")

	var httpResp *http.Response
	if resp != nil {
		httpResp = resp.(*http.Response)
		if httpResp.StatusCode != http.StatusOK {
			// ASR 上游（尤其 MaaS）常返回纯文本或非常规 JSON 错误体，带上 body 便于定位
			tokenFactoryError = service.RelayErrorHandler(c.Request.Context(), httpResp, true)
			// reset status code 重置状态码
			service.ResetStatusCode(tokenFactoryError, statusCodeMappingStr)
			return tokenFactoryError
		}
	}

	usage, tokenFactoryError := adaptor.DoResponse(c, httpResp, info)
	if tokenFactoryError != nil {
		// reset status code 重置状态码
		service.ResetStatusCode(tokenFactoryError, statusCodeMappingStr)
		return tokenFactoryError
	}

	// 阿里云 ASR 按秒计费：DoResponse 已将真实音频秒数写入 PriceData.OtherRatios["seconds"]
	if constant.IsASRChannel(info.ChannelType) {
		seconds := info.PriceData.OtherRatios["seconds"]
		if tfErr := service.PostASRConsumeQuota(c, info, seconds, "", ""); tfErr != nil {
			return tfErr
		}
		return nil
	}

	if usage.(*dto.Usage).CompletionTokenDetails.AudioTokens > 0 || usage.(*dto.Usage).PromptTokensDetails.AudioTokens > 0 {
		service.PostAudioConsumeQuota(c, info, usage.(*dto.Usage), "")
	} else {
		service.PostTextConsumeQuota(c, info, usage.(*dto.Usage), nil)
	}

	return nil
}

// audioJSONHasNativeInputMessages 判断 JSON 是否已是上游原生 multimodal 体（含 input.messages）。
func audioJSONHasNativeInputMessages(body []byte) bool {
	var peek struct {
		Input *struct {
			Messages json.RawMessage `json:"messages"`
		} `json:"input"`
	}
	if err := common.Unmarshal(body, &peek); err != nil {
		return false
	}
	return peek.Input != nil && len(peek.Input.Messages) > 0
}

// rewriteAudioJSONModelField 在 JSON 透传体中写入映射后的上游模型名。
func rewriteAudioJSONModelField(cachedBody []byte, upstreamModel string) ([]byte, error) {
	var payload map[string]any
	if err := common.Unmarshal(cachedBody, &payload); err != nil {
		return nil, err
	}
	payload["model"] = upstreamModel
	return common.Marshal(payload)
}
