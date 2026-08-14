package relay

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/relay/channel"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/gin-gonic/gin"
)

const imageAsyncTimeout = 15 * time.Minute

type imageResponseCaptureWriter struct {
	gin.ResponseWriter
	buf         *bytes.Buffer
	captureOnly bool
	statusCode  int
}

func (w *imageResponseCaptureWriter) Write(data []byte) (int, error) {
	if w.buf != nil {
		_, _ = w.buf.Write(data)
	}
	if w.captureOnly {
		return len(data), nil
	}
	return w.ResponseWriter.Write(data)
}

func (w *imageResponseCaptureWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

func (w *imageResponseCaptureWriter) WriteHeader(code int) {
	w.statusCode = code
	if !w.captureOnly {
		w.ResponseWriter.WriteHeader(code)
	}
}

func ImageHelper(c *gin.Context, info *relaycommon.RelayInfo) (tokenFactoryError *types.TokenFactoryError) {
	info.InitChannelMeta(c)

	imageReq, ok := info.Request.(*dto.ImageRequest)
	if !ok {
		return types.NewErrorWithStatusCode(fmt.Errorf("invalid request type, expected dto.ImageRequest, got %T", info.Request), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}

	request, err := common.DeepCopy(imageReq)
	if err != nil {
		return types.NewError(fmt.Errorf("failed to copy request to ImageRequest: %w", err), types.ErrorCodeInvalidRequest, types.ErrOptionWithSkipRetry())
	}

	callbackURL := strings.TrimSpace(request.CallbackURL)
	request.CallbackURL = "" // 网关侧参数，不转发上游
	if imageReq != nil {
		imageReq.CallbackURL = ""
	}

	if callbackURL != "" {
		if err := service.ValidateImageCallbackURL(callbackURL); err != nil {
			return types.NewErrorWithStatusCode(err, types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
		}
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

	requestBodyBytes, contentType, bodyErr := buildImageUpstreamRequestBody(c, info, adaptor, request)
	if bodyErr != nil {
		return bodyErr
	}

	if callbackURL != "" {
		return launchImageAsyncCallback(c, info, request, requestBodyBytes, contentType, callbackURL)
	}

	_, tokenFactoryError = executeImageRelay(c, info, request, bytes.NewReader(requestBodyBytes), false)
	return tokenFactoryError
}

func buildImageUpstreamRequestBody(c *gin.Context, info *relaycommon.RelayInfo, adaptor channel.Adaptor, request *dto.ImageRequest) ([]byte, string, *types.TokenFactoryError) {
	contentType := c.Request.Header.Get("Content-Type")
	if model_setting.GetGlobalSettings().PassThroughRequestEnabled || info.ChannelSetting.PassThroughBodyEnabled {
		storage, err := common.GetBodyStorage(c)
		if err != nil {
			return nil, "", types.NewErrorWithStatusCode(err, types.ErrorCodeReadRequestBodyFailed, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
		}
		data, err := storage.Bytes()
		if err != nil {
			return nil, "", types.NewErrorWithStatusCode(err, types.ErrorCodeReadRequestBodyFailed, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
		}
		stripped, stripErr := stripCallbackURLFromRequestBody(data, contentType)
		if stripErr != nil {
			logger.LogWarn(c, fmt.Sprintf("strip callback_url from passthrough body failed: %s", stripErr.Error()))
			return data, contentType, nil
		}
		return stripped, contentType, nil
	}

	convertedRequest, err := adaptor.ConvertImageRequest(c, info, *request)
	if err != nil {
		return nil, "", types.NewError(err, types.ErrorCodeConvertRequestFailed)
	}
	relaycommon.AppendRequestConversionFromRequest(info, convertedRequest)
	contentType = c.Request.Header.Get("Content-Type")

	switch body := convertedRequest.(type) {
	case *bytes.Buffer:
		return body.Bytes(), contentType, nil
	case []byte:
		return body, contentType, nil
	default:
		jsonData, err := common.Marshal(convertedRequest)
		if err != nil {
			return nil, "", types.NewError(err, types.ErrorCodeConvertRequestFailed, types.ErrOptionWithSkipRetry())
		}

		if len(info.ParamOverride) > 0 {
			jsonData, err = relaycommon.ApplyParamOverrideWithRelayInfo(jsonData, info)
			if err != nil {
				return nil, "", tokenFactoryErrorFromParamOverride(err)
			}
		}

		if common.DebugEnabled {
			logger.LogDebug(c, fmt.Sprintf("image request body: %s", string(jsonData)))
		}
		if contentType == "" {
			contentType = "application/json"
		}
		return jsonData, contentType, nil
	}
}

func stripCallbackURLFromRequestBody(data []byte, contentType string) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}
	ct := strings.ToLower(contentType)
	if strings.Contains(ct, "multipart/form-data") {
		// multipart 透传场景下字段剥离成本高；callback_url 已在结构化路径剥离。
		return data, nil
	}
	var rawMap map[string]any
	if err := common.Unmarshal(data, &rawMap); err != nil {
		return data, nil
	}
	if _, ok := rawMap["callback_url"]; !ok {
		return data, nil
	}
	delete(rawMap, "callback_url")
	return common.Marshal(rawMap)
}

func launchImageAsyncCallback(c *gin.Context, info *relaycommon.RelayInfo, request *dto.ImageRequest, requestBody []byte, contentType string, callbackURL string) *types.TokenFactoryError {
	requestID := strings.TrimSpace(c.GetString(common.RequestIdKey))
	if requestID == "" {
		requestID = info.RequestId
	}
	if requestID == "" {
		requestID = common.GetTimeString() + common.GetRandomString(8)
	}
	created := time.Now().Unix()
	if !info.StartTime.IsZero() {
		created = info.StartTime.Unix()
	}

	asyncCtx, cancel, err := cloneGinContextForImageAsync(c, requestID)
	if err != nil {
		return types.NewError(err, types.ErrorCodeInvalidRequest, types.ErrOptionWithSkipRetry())
	}
	if contentType != "" {
		asyncCtx.Request.Header.Set("Content-Type", contentType)
	}

	bodyCopy := append([]byte(nil), requestBody...)
	requestCopy, copyErr := common.DeepCopy(request)
	if copyErr != nil {
		cancel()
		return types.NewError(fmt.Errorf("failed to copy async image request: %w", copyErr), types.ErrorCodeInvalidRequest, types.ErrOptionWithSkipRetry())
	}

	c.JSON(http.StatusOK, dto.ImageAsyncSubmitResponse{
		ID:      requestID,
		Created: created,
	})

	gopool.Go(func() {
		defer cancel()
		defer func() {
			if r := recover(); r != nil {
				common.SysLog(fmt.Sprintf("image async callback panic: %v", r))
				if info.Billing != nil {
					info.Billing.Refund(asyncCtx)
				}
				_ = service.PostImageCallback(callbackURL, &dto.ImageCallbackPayload{
					ID:      requestID,
					Created: created,
					Status:  dto.ImageCallbackStatusFailed,
					Error: &dto.ImageCallbackError{
						Message: fmt.Sprintf("internal error: %v", r),
						Type:    "server_error",
					},
				})
			}
		}()

		responseBody, tokenFactoryError := executeImageRelay(asyncCtx, info, requestCopy, bytes.NewReader(bodyCopy), true)
		if tokenFactoryError != nil {
			logger.LogError(asyncCtx, fmt.Sprintf("image async relay failed: %s", tokenFactoryError.Error()))
			if info.Billing != nil {
				info.Billing.Refund(asyncCtx)
			}
			openaiErr := tokenFactoryError.ToOpenAIError()
			_ = service.PostImageCallback(callbackURL, &dto.ImageCallbackPayload{
				ID:      requestID,
				Created: created,
				Status:  dto.ImageCallbackStatusFailed,
				Error: &dto.ImageCallbackError{
					Message: openaiErr.Message,
					Type:    openaiErr.Type,
					Code:    openaiErr.Code,
				},
			})
			return
		}

		payload := service.BuildImageSuccessCallbackPayload(requestID, created, responseBody)
		if len(payload.Data) == 0 && len(responseBody) == 0 {
			logger.LogWarn(asyncCtx, "image async callback success but response body is empty")
		}
		if err := service.PostImageCallback(callbackURL, payload); err != nil {
			logger.LogError(asyncCtx, fmt.Sprintf("post image callback failed: %s", err.Error()))
		}
	})

	return nil
}

func cloneGinContextForImageAsync(src *gin.Context, requestID string) (*gin.Context, context.CancelFunc, error) {
	if src == nil || src.Request == nil {
		return nil, nil, fmt.Errorf("invalid gin context for image async")
	}

	recorder := httptest.NewRecorder()
	dst, _ := gin.CreateTestContext(recorder)

	bgCtx, cancel := context.WithTimeout(context.Background(), imageAsyncTimeout)
	bgCtx = context.WithValue(bgCtx, common.RequestIdKey, requestID)

	req := src.Request.Clone(bgCtx)
	req.Body = http.NoBody
	req.ContentLength = 0
	req.GetBody = nil
	dst.Request = req

	if dst.Keys == nil {
		dst.Keys = make(map[any]any)
	}
	for k, v := range src.Keys {
		dst.Keys[k] = v
	}
	dst.Set(common.RequestIdKey, requestID)
	dst.Set(common.KeyBodyStorage, nil)
	dst.Set(string(constant.ContextKeyFileSourcesToCleanup), nil)

	return dst, cancel, nil
}

func executeImageRelay(c *gin.Context, info *relaycommon.RelayInfo, request *dto.ImageRequest, requestBody io.Reader, asyncCapture bool) (responseBody []byte, tokenFactoryError *types.TokenFactoryError) {
	adaptor := GetAdaptor(info.ApiType)
	if adaptor == nil {
		return nil, types.NewError(fmt.Errorf("invalid api type: %d", info.ApiType), types.ErrorCodeInvalidApiType, types.ErrOptionWithSkipRetry())
	}
	adaptor.Init(info)

	statusCodeMappingStr := c.GetString("status_code_mapping")

	shouldCheckGuardrailOutput := setting.ShouldCheckAliyunGuardrailOutputForUser(c.GetInt(`id`))
	captureImageResponse := asyncCapture || shouldCaptureImageResponse(info) || shouldCheckGuardrailOutput
	var responseCapture *bytes.Buffer
	originalWriter := c.Writer
	var captureWriter *imageResponseCaptureWriter
	if captureImageResponse {
		responseCapture = &bytes.Buffer{}
		captureWriter = &imageResponseCaptureWriter{
			ResponseWriter: c.Writer,
			buf:            responseCapture,
			captureOnly:    asyncCapture || shouldCheckGuardrailOutput,
		}
		c.Writer = captureWriter
	}

	resp, err := adaptor.DoRequest(c, info, requestBody)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeDoRequestFailed, http.StatusInternalServerError)
	}
	var httpResp *http.Response
	if resp != nil {
		httpResp = resp.(*http.Response)
		info.IsStream = info.IsStream || strings.HasPrefix(httpResp.Header.Get("Content-Type"), "text/event-stream")
		if httpResp.StatusCode != http.StatusOK {
			if httpResp.StatusCode == http.StatusCreated && info.ApiType == constant.APITypeReplicate {
				// replicate channel returns 201 Created when using Prefer: wait, treat it as success.
				httpResp.StatusCode = http.StatusOK
			} else {
				tokenFactoryError = service.RelayErrorHandler(c.Request.Context(), httpResp, false)
				service.ResetStatusCode(tokenFactoryError, statusCodeMappingStr)
				return nil, tokenFactoryError
			}
		}
	}

	usage, tokenFactoryError := adaptor.DoResponse(c, httpResp, info)
	if tokenFactoryError != nil {
		service.ResetStatusCode(tokenFactoryError, statusCodeMappingStr)
		return nil, tokenFactoryError
	}
	if captureWriter != nil && captureWriter.captureOnly {
		c.Writer = originalWriter
		captured := append([]byte(nil), responseCapture.Bytes()...)
		var imageResponse dto.ImageResponse
		if err := common.Unmarshal(captured, &imageResponse); err == nil {
			imageURLs := make([]string, 0, len(imageResponse.Data))
			for _, image := range imageResponse.Data {
				if strings.HasPrefix(image.Url, "http") {
					imageURLs = append(imageURLs, image.Url)
				}
			}
			guardrailResult, guardrailErr := service.CheckAliyunGuardrailImageOutput(c, info, imageURLs)
			if guardrailErr != nil {
				logger.LogWarn(c, fmt.Sprintf("aliyun guardrail image output check skipped: %s", guardrailErr.Error()))
			}
			if guardrailResult != nil && guardrailResult.Blocked {
				c.Set("aliyun_guardrail_output_blocked", true)
				return nil, types.NewOpenAIError(fmt.Errorf("aliyun guardrail blocked image output"), types.ErrorCodeSensitiveWordsDetected, http.StatusBadRequest)
			}
		}
		if asyncCapture {
			responseBody = captured
		} else {
			if captureWriter.statusCode > 0 {
				originalWriter.WriteHeader(captureWriter.statusCode)
			}
			_, _ = originalWriter.Write(captured)
			originalWriter.Flush()
		}
	} else if asyncCapture && responseCapture != nil {
		responseBody = append([]byte(nil), responseCapture.Bytes()...)
	}

	imageN := uint(1)
	if request.N != nil {
		imageN = *request.N
	}
	if captureImageResponse && responseCapture != nil {
		helper.FinalizeImagePerImageBilling(c, info, request, responseCapture.Bytes())
		if n, ok := info.PriceData.OtherRatios["n"]; ok && n > 0 {
			imageN = uint(n)
		}
	} else if _, hasN := info.PriceData.OtherRatios["n"]; !hasN {
		info.PriceData.AddOtherRatio("n", float64(imageN))
	}

	usageDto, _ := usage.(*dto.Usage)
	if usageDto == nil {
		usageDto = &dto.Usage{}
	}
	if usageDto.TotalTokens == 0 {
		usageDto.TotalTokens = 1
	}
	if usageDto.PromptTokens == 0 {
		usageDto.PromptTokens = 1
	}

	quality := "standard"
	if request.Quality == "hd" {
		quality = "hd"
	}

	var logContent []string

	sizeLabel := strings.TrimSpace(request.Size)
	if info.ImageBilling != nil && info.ImageBilling.Width > 0 && info.ImageBilling.Height > 0 {
		sizeLabel = fmt.Sprintf("%dx%d", info.ImageBilling.Width, info.ImageBilling.Height)
	}
	if sizeLabel != "" {
		logContent = append(logContent, fmt.Sprintf("大小 %s", sizeLabel))
	}
	if len(quality) > 0 {
		logContent = append(logContent, fmt.Sprintf("品质 %s", quality))
	}
	if imageN > 0 {
		logContent = append(logContent, fmt.Sprintf("生成数量 %d", imageN))
	}
	if asyncCapture {
		logContent = append(logContent, "异步回调")
	}

	service.PostTextConsumeQuota(c, info, usageDto, logContent)
	return responseBody, nil
}

func shouldCaptureImageResponse(info *relaycommon.RelayInfo) bool {
	if info == nil || info.ImageBilling == nil || !info.PriceData.UsePrice {
		return false
	}
	return info.RelayMode == relayconstant.RelayModeImagesGenerations ||
		info.RelayMode == relayconstant.RelayModeImagesEdits
}
