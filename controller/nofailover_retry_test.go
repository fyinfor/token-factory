package controller

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

func TestShouldRetry_RespectsNoFailoverHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	c.Request.Header.Set(service.HeaderNoFailover, "true")
	service.ApplyNoFailoverHeader(c)

	err := types.NewErrorWithStatusCode(
		errors.New("upstream 502"),
		types.ErrorCodeBadResponseStatusCode,
		http.StatusBadGateway,
	)
	if shouldRetry(c, err, 3) {
		t.Fatal("shouldRetry must be false when X-TF-No-Failover is set")
	}
}

func TestShouldRetry_WithoutNoFailoverStillCanRetry(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	err := types.NewError(
		errors.New("channel down"),
		types.ErrorCodeChannelInvalidKey,
	)
	if !shouldRetry(c, err, 3) {
		t.Fatal("channel errors should still allow retry when No-Failover is absent")
	}
}
