package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
)

func TestParseNoFailoverValue(t *testing.T) {
	cases := map[string]bool{
		"":        false,
		"0":       false,
		"false":   false,
		"no":      false,
		"1":       true,
		"true":    true,
		"TRUE":    true,
		" yes ":   true,
		"on":      true,
		"enabled": false,
	}
	for in, want := range cases {
		if got := parseNoFailoverValue(in); got != want {
			t.Fatalf("parseNoFailoverValue(%q)=%v want %v", in, got, want)
		}
	}
}

func TestApplyNoFailoverHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	c.Request.Header.Set(HeaderNoFailover, "true")

	ApplyNoFailoverHeader(c)
	if !ContextNoFailover(c) {
		t.Fatal("expected ContextNoFailover after ApplyNoFailoverHeader")
	}
	v, ok := common.GetContextKey(c, constant.ContextKeyNoFailover)
	if !ok || v != true {
		t.Fatalf("context key not set: ok=%v v=%v", ok, v)
	}
}

func TestContextNoFailover_HeaderOnlyWithoutApply(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	c.Request.Header.Set(HeaderNoFailover, "1")
	if !ContextNoFailover(c) {
		t.Fatal("ContextNoFailover should honor request header even if Apply was skipped")
	}
}
