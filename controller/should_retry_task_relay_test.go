package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/gin-gonic/gin"
)

func newRelayTestContext() *gin.Context {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/video/generations", nil)
	return c
}

func TestShouldRetryTaskRelay_NoRetryOnInvalidRequestCode(t *testing.T) {
	c := newRelayTestContext()
	common.SetContextKey(c, constant.ContextKeySmartRouteChannelOrder, []int{101, 102})
	c.Set("use_channel", []string{"101"})

	taskErr := &dto.TaskError{
		StatusCode: http.StatusBadRequest,
		Code:       "invalid_request",
		Message:    "bad params",
		LocalError: false,
	}
	if shouldRetryTaskRelay(c, 101, taskErr, 3) {
		t.Fatal("invalid_request must not failover across channels")
	}
}

func TestShouldRetryTaskRelay_LocalErrorNeverRetries(t *testing.T) {
	c := newRelayTestContext()
	common.SetContextKey(c, constant.ContextKeySmartRouteChannelOrder, []int{101, 102})
	taskErr := &dto.TaskError{
		StatusCode: http.StatusInternalServerError,
		Code:       "upstream_error",
		LocalError: true,
	}
	if shouldRetryTaskRelay(c, 101, taskErr, 3) {
		t.Fatal("LocalError must not retry")
	}
}

func TestShouldRetryTaskRelay_NoFailoverHeader(t *testing.T) {
	c := newRelayTestContext()
	common.SetContextKey(c, constant.ContextKeySmartRouteChannelOrder, []int{101, 102})
	common.SetContextKey(c, constant.ContextKeyNoFailover, true)
	taskErr := &dto.TaskError{StatusCode: http.StatusBadGateway, Code: "upstream_error"}
	if shouldRetryTaskRelay(c, 101, taskErr, 3) {
		t.Fatal("X-TF-No-Failover must disable task channel failover")
	}
}
