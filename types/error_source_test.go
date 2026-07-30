package types

import (
	"errors"
	"net/http"
	"testing"
)

func TestGetErrorSource_UpstreamByType(t *testing.T) {
	err := WithClaudeError(ClaudeError{Type: "api_error", Message: "boom"}, http.StatusInternalServerError)
	if got := err.GetErrorSource(); got != ErrorSourceUpstream {
		t.Fatalf("GetErrorSource()=%s, want upstream", got)
	}
	if hint := err.LogErrorOriginHint(); hint != "上游" {
		t.Fatalf("LogErrorOriginHint()=%q, want 上游", hint)
	}
}

func TestGetErrorSource_BadResponseBody(t *testing.T) {
	err := NewError(errors.New(`json: cannot unmarshal string into Go struct field`), ErrorCodeBadResponseBody)
	if got := err.GetErrorSource(); got != ErrorSourceUpstream {
		t.Fatalf("GetErrorSource()=%s, want upstream", got)
	}
	if hint := err.LogErrorOriginHint(); hint != "上游(响应体解析失败)" {
		t.Fatalf("LogErrorOriginHint()=%q, want 上游(响应体解析失败)", hint)
	}
}

func TestGetErrorSource_PlatformQuota(t *testing.T) {
	err := NewErrorWithStatusCode(errors.New("insufficient quota"), ErrorCodeInsufficientUserQuota, http.StatusForbidden)
	if got := err.GetErrorSource(); got != ErrorSourcePlatform {
		t.Fatalf("GetErrorSource()=%s, want platform", got)
	}
	if hint := err.LogErrorOriginHint(); hint != "本平台" {
		t.Fatalf("LogErrorOriginHint()=%q, want 本平台", hint)
	}
}

func TestGetErrorSource_ChannelConfig(t *testing.T) {
	err := NewError(errors.New("no key"), ErrorCodeChannelNoAvailableKey)
	if got := err.GetErrorSource(); got != ErrorSourcePlatform {
		t.Fatalf("GetErrorSource()=%s, want platform", got)
	}
}
