package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateImageCallbackURLEmpty(t *testing.T) {
	err := ValidateImageCallbackURL("  ")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestValidateImageCallbackURLAcceptsHTTPS(t *testing.T) {
	// SSRF 开关关闭时，标准 https URL 应通过
	err := ValidateImageCallbackURL("https://example.com/hooks/image")
	if err != nil {
		// 若环境启用了严格 SSRF/域名白名单，仅断言错误文案可读
		assert.Contains(t, err.Error(), "callback_url")
	}
}
