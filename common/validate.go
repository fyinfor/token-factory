package common

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

var Validate *validator.Validate

func init() {
	Validate = validator.New()
}

// fieldLabelMap maps struct field names to user-friendly Chinese labels for
// validation error messages. Add new fields here as needed.
var fieldLabelMap = map[string]string{
	"Username":    "用户名",
	"Password":    "密码",
	"Email":       "邮箱",
	"DisplayName": "显示名",
	"Remark":      "备注",
	"Phone":       "手机号",
	"VerificationCode": "邮箱验证码",
	"SMSCode":     "短信验证码",
}

// FormatValidationError converts a validator error into a friendly Chinese
// message. It handles validator.ValidationErrors (Struct validation) and
// falls back to err.Error() for other error types.
//
// Example: for `RegisterRequest.Username` failed on the `max` tag with param
// "20", this returns "用户名长度不能超过 20 个字符".
func FormatValidationError(err error) string {
	if err == nil {
		return ""
	}

	valErrs, ok := err.(validator.ValidationErrors)
	if !ok {
		return err.Error()
	}

	var msgs []string
	for _, fe := range valErrs {
		msgs = append(msgs, formatFieldError(fe))
	}
	return strings.Join(msgs, "；")
}

// formatFieldError translates a single validator.FieldError into a friendly
// Chinese message based on the failed tag (required, min, max, email, etc.).
func formatFieldError(fe validator.FieldError) string {
	label, ok := fieldLabelMap[fe.Field()]
	if !ok {
		label = fe.Field()
	}

	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s不能为空", label)
	case "min":
		return fmt.Sprintf("%s长度不能少于 %s 个字符", label, fe.Param())
	case "max":
		return fmt.Sprintf("%s长度不能超过 %s 个字符", label, fe.Param())
	case "email":
		return fmt.Sprintf("%s格式不正确", label)
	case "omitempty":
		return fmt.Sprintf("%s格式不正确", label)
	default:
		return fmt.Sprintf("%s校验失败（%s）", label, fe.Tag())
	}
}
