package types

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

type OpenAIError struct {
	Message  string          `json:"message"`
	Type     string          `json:"type"`
	Param    string          `json:"param"`
	Code     any             `json:"code"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
}

type ClaudeError struct {
	Type    string `json:"type,omitempty"`
	Message string `json:"message,omitempty"`
}

type ErrorType string

const (
	ErrorTypeTokenFactoryError ErrorType = "token_factory_error"
	ErrorTypeOpenAIError       ErrorType = "openai_error"
	ErrorTypeClaudeError       ErrorType = "claude_error"
	ErrorTypeMidjourneyError   ErrorType = "midjourney_error"
	ErrorTypeGeminiError       ErrorType = "gemini_error"
	ErrorTypeRerankError       ErrorType = "rerank_error"
	ErrorTypeUpstreamError     ErrorType = "upstream_error"
)

type ErrorCode string

const (
	ErrorCodeInvalidRequest         ErrorCode = "invalid_request"
	ErrorCodeSensitiveWordsDetected ErrorCode = "sensitive_words_detected"
	ErrorCodeViolationFeeGrokCSAM   ErrorCode = "violation_fee.grok.csam"

	// token factory error
	ErrorCodeCountTokenFailed   ErrorCode = "count_token_failed"
	ErrorCodeModelPriceError    ErrorCode = "model_price_error"
	ErrorCodeInvalidApiType     ErrorCode = "invalid_api_type"
	ErrorCodeJsonMarshalFailed  ErrorCode = "json_marshal_failed"
	ErrorCodeDoRequestFailed    ErrorCode = "do_request_failed"
	ErrorCodeGetChannelFailed   ErrorCode = "get_channel_failed"
	ErrorCodeGenRelayInfoFailed ErrorCode = "gen_relay_info_failed"

	// channel error
	ErrorCodeChannelNoAvailableKey        ErrorCode = "channel:no_available_key"
	ErrorCodeChannelParamOverrideInvalid  ErrorCode = "channel:param_override_invalid"
	ErrorCodeChannelHeaderOverrideInvalid ErrorCode = "channel:header_override_invalid"
	ErrorCodeChannelModelMappedError      ErrorCode = "channel:model_mapped_error"
	ErrorCodeChannelAwsClientError        ErrorCode = "channel:aws_client_error"
	ErrorCodeChannelInvalidKey            ErrorCode = "channel:invalid_key"
	ErrorCodeChannelResponseTimeExceeded  ErrorCode = "channel:response_time_exceeded"
	ErrorCodeChannelBaseUrlEmpty          ErrorCode = "channel:base_url_empty"

	// client request error
	ErrorCodeReadRequestBodyFailed ErrorCode = "read_request_body_failed"
	ErrorCodeConvertRequestFailed  ErrorCode = "convert_request_failed"
	ErrorCodeAccessDenied          ErrorCode = "access_denied"

	// request error
	ErrorCodeBadRequestBody ErrorCode = "bad_request_body"

	// response error
	ErrorCodeReadResponseBodyFailed ErrorCode = "read_response_body_failed"
	ErrorCodeBadResponseStatusCode  ErrorCode = "bad_response_status_code"
	ErrorCodeBadResponse            ErrorCode = "bad_response"
	ErrorCodeBadResponseBody        ErrorCode = "bad_response_body"
	ErrorCodeEmptyResponse          ErrorCode = "empty_response"
	ErrorCodeAwsInvokeError         ErrorCode = "aws_invoke_error"
	ErrorCodeModelNotFound          ErrorCode = "model_not_found"
	ErrorCodePlatformModelRateLimit ErrorCode = "platform_model_rate_limit_exceeded"
	ErrorCodePromptBlocked          ErrorCode = "prompt_blocked"

	// sql error
	ErrorCodeQueryDataError  ErrorCode = "query_data_error"
	ErrorCodeUpdateDataError ErrorCode = "update_data_error"

	// quota error
	ErrorCodeInsufficientUserQuota      ErrorCode = "insufficient_user_quota"
	ErrorCodePreConsumeTokenQuotaFailed ErrorCode = "pre_consume_token_quota_failed"
)

type TokenFactoryError struct {
	Err            error
	RelayError     any
	skipRetry      bool
	recordErrorLog *bool
	errorType      ErrorType
	errorCode      ErrorCode
	StatusCode     int
	Metadata       json.RawMessage
}

// Unwrap enables errors.Is / errors.As to work with TokenFactoryError by exposing the underlying error.
func (e *TokenFactoryError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func (e *TokenFactoryError) GetErrorCode() ErrorCode {
	if e == nil {
		return ""
	}
	return e.errorCode
}

func (e *TokenFactoryError) GetErrorType() ErrorType {
	if e == nil {
		return ""
	}
	return e.errorType
}

func (e *TokenFactoryError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err == nil {
		// fallback message when underlying error is missing
		return string(e.errorCode)
	}
	return e.Err.Error()
}

func (e *TokenFactoryError) ErrorWithStatusCode() string {
	if e == nil {
		return ""
	}
	msg := e.Error()
	if e.StatusCode == 0 {
		return msg
	}
	if msg == "" {
		return fmt.Sprintf("status_code=%d", e.StatusCode)
	}
	return fmt.Sprintf("status_code=%d, %s", e.StatusCode, msg)
}

func (e *TokenFactoryError) MaskSensitiveError() string {
	if e == nil {
		return ""
	}
	if e.Err == nil {
		return string(e.errorCode)
	}
	errStr := e.Err.Error()
	if e.errorCode == ErrorCodeCountTokenFailed {
		return errStr
	}
	return common.MaskSensitiveInfo(errStr)
}

func (e *TokenFactoryError) MaskSensitiveErrorWithStatusCode() string {
	if e == nil {
		return ""
	}
	msg := e.MaskSensitiveError()
	if e.StatusCode == 0 {
		return msg
	}
	if msg == "" {
		return fmt.Sprintf("status_code=%d", e.StatusCode)
	}
	return fmt.Sprintf("status_code=%d, %s", e.StatusCode, msg)
}

func (e *TokenFactoryError) SetMessage(message string) {
	e.Err = errors.New(message)
}

func (e *TokenFactoryError) ToOpenAIError() OpenAIError {
	var result OpenAIError
	switch e.errorType {
	case ErrorTypeOpenAIError:
		if openAIError, ok := e.RelayError.(OpenAIError); ok {
			result = openAIError
		}
	case ErrorTypeClaudeError:
		if claudeError, ok := e.RelayError.(ClaudeError); ok {
			result = OpenAIError{
				Message: e.Error(),
				Type:    claudeError.Type,
				Param:   "",
				Code:    e.errorCode,
			}
		}
	default:
		result = OpenAIError{
			Message: e.Error(),
			Type:    string(e.errorType),
			Param:   "",
			Code:    e.errorCode,
		}
	}
	// RelayError.Message 为空时（例如 InitOpenAIError 后仅更新了 Err），回退到底层 Err，避免客户端只看到 "openai_error"。
	if result.Message == "" {
		result.Message = e.Error()
	}
	if e.errorCode != ErrorCodeCountTokenFailed {
		result.Message = common.MaskSensitiveInfo(result.Message)
	}
	if result.Message == "" {
		result.Message = string(e.errorType)
	}
	return result
}

func (e *TokenFactoryError) ToClaudeError() ClaudeError {
	var result ClaudeError
	switch e.errorType {
	case ErrorTypeOpenAIError:
		if openAIError, ok := e.RelayError.(OpenAIError); ok {
			result = ClaudeError{
				Message: e.Error(),
				Type:    fmt.Sprintf("%v", openAIError.Code),
			}
		}
	case ErrorTypeClaudeError:
		if claudeError, ok := e.RelayError.(ClaudeError); ok {
			result = claudeError
		}
	default:
		result = ClaudeError{
			Message: e.Error(),
			Type:    string(e.errorType),
		}
	}
	if e.errorCode != ErrorCodeCountTokenFailed {
		result.Message = common.MaskSensitiveInfo(result.Message)
	}
	if result.Message == "" {
		result.Message = string(e.errorType)
	}
	return result
}

type TokenFactoryErrorOptions func(*TokenFactoryError)

func NewError(err error, errorCode ErrorCode, ops ...TokenFactoryErrorOptions) *TokenFactoryError {
	var newErr *TokenFactoryError
	// 保留深层传递的 new err
	if errors.As(err, &newErr) {
		for _, op := range ops {
			op(newErr)
		}
		return newErr
	}
	e := &TokenFactoryError{
		Err:        err,
		RelayError: nil,
		errorType:  ErrorTypeTokenFactoryError,
		StatusCode: http.StatusInternalServerError,
		errorCode:  errorCode,
	}
	for _, op := range ops {
		op(e)
	}
	return e
}

func NewOpenAIError(err error, errorCode ErrorCode, statusCode int, ops ...TokenFactoryErrorOptions) *TokenFactoryError {
	var newErr *TokenFactoryError
	// 保留深层传递的 new err
	if errors.As(err, &newErr) {
		if newErr.RelayError == nil {
			openaiError := OpenAIError{
				Message: newErr.Error(),
				Type:    string(errorCode),
				Code:    errorCode,
			}
			newErr.RelayError = openaiError
		}
		for _, op := range ops {
			op(newErr)
		}
		return newErr
	}
	openaiError := OpenAIError{
		Message: err.Error(),
		Type:    string(errorCode),
		Code:    errorCode,
	}
	return WithOpenAIError(openaiError, statusCode, ops...)
}

func InitOpenAIError(errorCode ErrorCode, statusCode int, ops ...TokenFactoryErrorOptions) *TokenFactoryError {
	openaiError := OpenAIError{
		Type: string(errorCode),
		Code: errorCode,
	}
	return WithOpenAIError(openaiError, statusCode, ops...)
}

func NewErrorWithStatusCode(err error, errorCode ErrorCode, statusCode int, ops ...TokenFactoryErrorOptions) *TokenFactoryError {
	e := &TokenFactoryError{
		Err: err,
		RelayError: OpenAIError{
			Message: err.Error(),
			Type:    string(errorCode),
		},
		errorType:  ErrorTypeTokenFactoryError,
		StatusCode: statusCode,
		errorCode:  errorCode,
	}
	for _, op := range ops {
		op(e)
	}

	return e
}

func WithOpenAIError(openAIError OpenAIError, statusCode int, ops ...TokenFactoryErrorOptions) *TokenFactoryError {
	code, ok := openAIError.Code.(string)
	if !ok {
		if openAIError.Code != nil {
			code = fmt.Sprintf("%v", openAIError.Code)
		} else {
			code = "unknown_error"
		}
	}
	if openAIError.Type == "" {
		openAIError.Type = "upstream_error"
	}
	e := &TokenFactoryError{
		RelayError: openAIError,
		errorType:  ErrorTypeOpenAIError,
		StatusCode: statusCode,
		Err:        errors.New(openAIError.Message),
		errorCode:  ErrorCode(code),
	}
	// OpenRouter
	if len(openAIError.Metadata) > 0 {
		openAIError.Message = fmt.Sprintf("%s (%s)", openAIError.Message, openAIError.Metadata)
		e.Metadata = openAIError.Metadata
		e.RelayError = openAIError
		e.Err = errors.New(openAIError.Message)
	}
	for _, op := range ops {
		op(e)
	}
	return e
}

func WithClaudeError(claudeError ClaudeError, statusCode int, ops ...TokenFactoryErrorOptions) *TokenFactoryError {
	if claudeError.Type == "" {
		claudeError.Type = "upstream_error"
	}
	e := &TokenFactoryError{
		RelayError: claudeError,
		errorType:  ErrorTypeClaudeError,
		StatusCode: statusCode,
		Err:        errors.New(claudeError.Message),
		errorCode:  ErrorCode(claudeError.Type),
	}
	for _, op := range ops {
		op(e)
	}
	return e
}

func IsChannelError(err *TokenFactoryError) bool {
	if err == nil {
		return false
	}
	return strings.HasPrefix(string(err.errorCode), "channel:")
}

// ErrorSource 标识错误来源：上游供应商 vs 本平台。
type ErrorSource string

const (
	ErrorSourceUpstream ErrorSource = "upstream"
	ErrorSourcePlatform ErrorSource = "platform"
)

// GetErrorSource 判断错误主要归属上游还是本平台。
// 上游：供应商 HTTP/响应体/业务错误，或本平台解析上游响应失败（根因仍在上游数据）。
// 本平台：鉴权、额度、路由、请求校验、渠道配置等本地逻辑。
func (e *TokenFactoryError) GetErrorSource() ErrorSource {
	if e == nil {
		return ErrorSourcePlatform
	}
	switch e.errorType {
	case ErrorTypeOpenAIError, ErrorTypeClaudeError, ErrorTypeGeminiError,
		ErrorTypeMidjourneyError, ErrorTypeRerankError, ErrorTypeUpstreamError:
		return ErrorSourceUpstream
	}
	switch e.errorCode {
	case ErrorCodeBadResponseStatusCode, ErrorCodeBadResponse, ErrorCodeBadResponseBody,
		ErrorCodeEmptyResponse, ErrorCodeReadResponseBodyFailed, ErrorCodeDoRequestFailed,
		ErrorCodeAwsInvokeError:
		return ErrorSourceUpstream
	default:
		return ErrorSourcePlatform
	}
}

// LogErrorOriginHint 供服务端运维日志使用的中文来源提示（不影响对外/API 错误体）。
func (e *TokenFactoryError) LogErrorOriginHint() string {
	if e == nil {
		return "本平台"
	}
	switch e.errorCode {
	case ErrorCodeBadResponseBody:
		return "上游(响应体解析失败)"
	case ErrorCodeBadResponseStatusCode:
		return "上游(HTTP状态码异常)"
	case ErrorCodeBadResponse, ErrorCodeEmptyResponse:
		return "上游(响应异常)"
	case ErrorCodeReadResponseBodyFailed:
		return "上游(读取响应失败)"
	case ErrorCodeDoRequestFailed:
		return "上游(请求上游失败)"
	case ErrorCodeAwsInvokeError:
		return "上游(AWS调用失败)"
	}
	if e.GetErrorSource() == ErrorSourceUpstream {
		return "上游"
	}
	return "本平台"
}

func IsSkipRetryError(err *TokenFactoryError) bool {
	if err == nil {
		return false
	}

	return err.skipRetry
}

func ErrOptionWithSkipRetry() TokenFactoryErrorOptions {
	return func(e *TokenFactoryError) {
		e.skipRetry = true
	}
}

func ErrOptionWithNoRecordErrorLog() TokenFactoryErrorOptions {
	return func(e *TokenFactoryError) {
		e.recordErrorLog = common.GetPointer(false)
	}
}

func ErrOptionWithHideErrMsg(replaceStr string) TokenFactoryErrorOptions {
	return func(e *TokenFactoryError) {
		if common.DebugEnabled {
			fmt.Printf("ErrOptionWithHideErrMsg: %s, origin error: %s", replaceStr, e.Err)
		}
		e.Err = errors.New(replaceStr)
	}
}

func IsRecordErrorLog(e *TokenFactoryError) bool {
	if e == nil {
		return false
	}
	if e.recordErrorLog == nil {
		// default to true if not set
		return true
	}
	return *e.recordErrorLog
}
