package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/types"
)

func MidjourneyErrorWrapper(code int, desc string) *dto.MidjourneyResponse {
	return &dto.MidjourneyResponse{
		Code:        code,
		Description: desc,
	}
}

func MidjourneyErrorWithStatusCodeWrapper(code int, desc string, statusCode int) *dto.MidjourneyResponseWithStatusCode {
	return &dto.MidjourneyResponseWithStatusCode{
		StatusCode: statusCode,
		Response:   *MidjourneyErrorWrapper(code, desc),
	}
}

//// OpenAIErrorWrapper wraps an error into an OpenAIErrorWithStatusCode
//func OpenAIErrorWrapper(err error, code string, statusCode int) *dto.OpenAIErrorWithStatusCode {
//	text := err.Error()
//	lowerText := strings.ToLower(text)
//	if !strings.HasPrefix(lowerText, "get file base64 from url") && !strings.HasPrefix(lowerText, "mime type is not supported") {
//		if strings.Contains(lowerText, "post") || strings.Contains(lowerText, "dial") || strings.Contains(lowerText, "http") {
//			common.SysLog(fmt.Sprintf("error: %s", text))
//			text = "请求上游地址失败"
//		}
//	}
//	openAIError := dto.OpenAIError{
//		Message: text,
//		Type:    "token_factory_error",
//		Code:    code,
//	}
//	return &dto.OpenAIErrorWithStatusCode{
//		Error:      openAIError,
//		StatusCode: statusCode,
//	}
//}
//
//func OpenAIErrorWrapperLocal(err error, code string, statusCode int) *dto.OpenAIErrorWithStatusCode {
//	openaiErr := OpenAIErrorWrapper(err, code, statusCode)
//	openaiErr.LocalError = true
//	return openaiErr
//}

func ClaudeErrorWrapper(err error, code string, statusCode int) *dto.ClaudeErrorWithStatusCode {
	text := err.Error()
	lowerText := strings.ToLower(text)
	if !strings.HasPrefix(lowerText, "get file base64 from url") {
		if strings.Contains(lowerText, "post") || strings.Contains(lowerText, "dial") || strings.Contains(lowerText, "http") {
			common.SysLog(fmt.Sprintf("error: %s", text))
			text = "请求上游地址失败"
		}
	}
	claudeError := types.ClaudeError{
		Message: text,
		Type:    "token_factory_error",
	}
	return &dto.ClaudeErrorWithStatusCode{
		Error:      claudeError,
		StatusCode: statusCode,
	}
}

func ClaudeErrorWrapperLocal(err error, code string, statusCode int) *dto.ClaudeErrorWithStatusCode {
	claudeErr := ClaudeErrorWrapper(err, code, statusCode)
	claudeErr.LocalError = true
	return claudeErr
}

func RelayErrorHandler(ctx context.Context, resp *http.Response, showBodyWhenFail bool) (tokenFactoryErr *types.TokenFactoryError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return types.NewOpenAIError(
			fmt.Errorf("bad response status code %d, read body failed: %w", resp.StatusCode, err),
			types.ErrorCodeBadResponseStatusCode,
			resp.StatusCode,
		)
	}
	CloseResponseBodyGracefully(resp)

	bodyText := strings.TrimSpace(string(responseBody))
	truncatedBody := truncateUpstreamErrorBody(bodyText)
	buildMessage := func(message string) string {
		if message == "" {
			if truncatedBody == "" {
				return fmt.Sprintf("bad response status code %d", resp.StatusCode)
			}
			return fmt.Sprintf("bad response status code %d, body: %s", resp.StatusCode, truncatedBody)
		}
		if showBodyWhenFail && truncatedBody != "" {
			return fmt.Sprintf("bad response status code %d, message: %s, body: %s", resp.StatusCode, message, truncatedBody)
		}
		// 形如 "<400> xxx.InvalidParameter: ..." 的上游文本/非常规 JSON：保留原文，避免只剩状态码
		if strings.HasPrefix(message, "<") || strings.Contains(message, "InvalidParameter") {
			return message
		}
		return message
	}

	var errResponse dto.GeneralErrorResponse
	err = common.Unmarshal(responseBody, &errResponse)
	if err != nil {
		// 非 JSON（MaaS 常见纯文本 "<400> xxx.InvalidParameter: ..."）直接回传正文
		logger.LogError(ctx, fmt.Sprintf("bad response status code %d, body: %s", resp.StatusCode, truncatedBody))
		if bodyText != "" {
			return types.NewOpenAIError(errors.New(truncatedBody), types.ErrorCodeBadResponseStatusCode, resp.StatusCode)
		}
		return types.NewOpenAIError(
			fmt.Errorf("bad response status code %d", resp.StatusCode),
			types.ErrorCodeBadResponseStatusCode,
			resp.StatusCode,
		)
	}

	if common.GetJsonType(errResponse.Error) == "object" {
		// General format error (OpenAI, Anthropic, Gemini, etc.)
		oaiError := errResponse.TryToOpenAIError()
		if oaiError != nil {
			msg := strings.TrimSpace(oaiError.Message)
			if msg == "" {
				msg = buildMessage("")
			} else if showBodyWhenFail {
				msg = buildMessage(msg)
			}
			oaiError.Message = msg
			if oaiError.Type == "" {
				oaiError.Type = string(types.ErrorCodeBadResponseStatusCode)
			}
			if oaiError.Code == nil {
				oaiError.Code = types.ErrorCodeBadResponseStatusCode
			}
			return types.WithOpenAIError(*oaiError, resp.StatusCode)
		}
	}

	msg := strings.TrimSpace(errResponse.ToMessage())
	if msg == "" {
		// JSON 可解析但无 message 字段（或仅有 code）：把原始 body 带给客户端便于排查
		msg = buildMessage("")
		logger.LogError(ctx, fmt.Sprintf("bad response status code %d, empty message, body: %s", resp.StatusCode, truncatedBody))
	} else if showBodyWhenFail {
		msg = buildMessage(msg)
	}
	return types.NewOpenAIError(errors.New(msg), types.ErrorCodeBadResponseStatusCode, resp.StatusCode)
}

func truncateUpstreamErrorBody(s string) string {
	const maxLen = 800
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "...(truncated)"
}

func ResetStatusCode(tokenFactoryErr *types.TokenFactoryError, statusCodeMappingStr string) {
	if tokenFactoryErr == nil {
		return
	}
	if statusCodeMappingStr == "" || statusCodeMappingStr == "{}" {
		return
	}
	statusCodeMapping := make(map[string]any)
	err := common.Unmarshal([]byte(statusCodeMappingStr), &statusCodeMapping)
	if err != nil {
		return
	}
	if tokenFactoryErr.StatusCode == http.StatusOK {
		return
	}
	codeStr := strconv.Itoa(tokenFactoryErr.StatusCode)
	if value, ok := statusCodeMapping[codeStr]; ok {
		intCode, ok := parseStatusCodeMappingValue(value)
		if !ok {
			return
		}
		tokenFactoryErr.StatusCode = intCode
	}
}

func parseStatusCodeMappingValue(value any) (int, bool) {
	switch v := value.(type) {
	case string:
		if v == "" {
			return 0, false
		}
		statusCode, err := strconv.Atoi(v)
		if err != nil {
			return 0, false
		}
		return statusCode, true
	case float64:
		if v != math.Trunc(v) {
			return 0, false
		}
		return int(v), true
	case int:
		return v, true
	case json.Number:
		statusCode, err := strconv.Atoi(v.String())
		if err != nil {
			return 0, false
		}
		return statusCode, true
	default:
		return 0, false
	}
}

func TaskErrorWrapperLocal(err error, code string, statusCode int) *dto.TaskError {
	openaiErr := TaskErrorWrapper(err, code, statusCode)
	openaiErr.LocalError = true
	return openaiErr
}

func TaskErrorWrapper(err error, code string, statusCode int) *dto.TaskError {
	text := err.Error()
	lowerText := strings.ToLower(text)
	if strings.Contains(lowerText, "post") || strings.Contains(lowerText, "dial") || strings.Contains(lowerText, "http") {
		common.SysLog(fmt.Sprintf("error: %s", text))
		//text = "请求上游地址失败"
		text = common.MaskSensitiveInfo(text)
	}
	//避免暴露内部错误
	taskError := &dto.TaskError{
		Code:       code,
		Message:    text,
		StatusCode: statusCode,
		Error:      err,
	}

	return taskError
}

// TaskErrorFromAPIError 将 PreConsumeBilling 返回的 TokenFactoryError 转换为 TaskError。
func TaskErrorFromAPIError(apiErr *types.TokenFactoryError) *dto.TaskError {
	if apiErr == nil {
		return nil
	}
	return &dto.TaskError{
		Code:       string(apiErr.GetErrorCode()),
		Message:    apiErr.Err.Error(),
		StatusCode: apiErr.StatusCode,
		Error:      apiErr.Err,
	}
}
