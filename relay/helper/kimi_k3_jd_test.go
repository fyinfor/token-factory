package helper

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestIsKimiK3TextModel(t *testing.T) {
	require.True(t, isKimiK3TextModel("kimi-k3"))
	require.True(t, isKimiK3TextModel("Kimi-K3"))
	require.True(t, isKimiK3TextModel(" moonshot/kimi-k3 "))
	require.False(t, isKimiK3TextModel("kimi-k2"))
	require.False(t, isKimiK3TextModel("kimi-k2.5"))
	require.False(t, isKimiK3TextModel("kimi-k3.5"))
	require.False(t, isKimiK3TextModel("kimi-k3-instruct"))
	require.False(t, isKimiK3TextModel("gpt-4o"))
	require.False(t, isKimiK3TextModel(""))
}

func TestKimiK3HasEmptyFirstUserMessage(t *testing.T) {
	require.False(t, kimiK3HasEmptyFirstUserMessage(nil))
	require.False(t, kimiK3HasEmptyFirstUserMessage(&dto.GeneralOpenAIRequest{}))
	require.True(t, kimiK3HasEmptyFirstUserMessage(&dto.GeneralOpenAIRequest{
		Messages: []dto.Message{{Role: "user", Content: ""}},
	}))
	require.False(t, kimiK3HasEmptyFirstUserMessage(&dto.GeneralOpenAIRequest{
		Messages: []dto.Message{{Role: "user", Content: "hello"}},
	}))
	require.False(t, kimiK3HasEmptyFirstUserMessage(&dto.GeneralOpenAIRequest{
		Messages: []dto.Message{{Role: "assistant", Content: ""}},
	}))
	require.False(t, kimiK3HasEmptyFirstUserMessage(&dto.GeneralOpenAIRequest{
		Messages: []dto.Message{{Role: "user", Content: nil}},
	}))
	require.False(t, kimiK3HasEmptyFirstUserMessage(&dto.GeneralOpenAIRequest{
		Messages: []dto.Message{{Role: "user", Content: []any{}}},
	}))
}

func TestShouldRejectKimiK3EmptyUser_SkipsOtherModels(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(string(constant.ContextKeyOriginalModel), "gpt-4o")

	req := &dto.GeneralOpenAIRequest{
		Model:    "gpt-4o",
		Messages: []dto.Message{{Role: "user", Content: ""}},
	}
	require.False(t, shouldRejectKimiK3EmptyUser(c, req))
}

func TestShouldRejectKimiK3EmptyUser_OriginModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(string(constant.ContextKeyOriginalModel), "kimi-k3")

	req := &dto.GeneralOpenAIRequest{
		Model:    "alias-mapped-away",
		Messages: []dto.Message{{Role: "user", Content: ""}},
	}
	require.True(t, shouldRejectKimiK3EmptyUser(c, req))
}

func TestAbortIfKimiK3EmptyUser_ExactJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req := &dto.GeneralOpenAIRequest{
		Model:    "kimi-k3",
		Messages: []dto.Message{{Role: "user", Content: ""}},
	}
	require.True(t, AbortIfKimiK3EmptyUser(c, req))
	require.Equal(t, http.StatusBadRequest, w.Code)

	var body map[string]any
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, kimiK3EmptyUserMessage, errObj["message"])
	require.Equal(t, kimiK3EmptyUserErrorType, errObj["type"])
	require.Len(t, errObj, 2)
	require.Len(t, body, 1)
}

func TestAbortIfKimiK3EmptyUser_NoOpForOtherModels(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req := &dto.GeneralOpenAIRequest{
		Model:    "kimi-k2.5",
		Messages: []dto.Message{{Role: "user", Content: ""}},
	}
	require.False(t, AbortIfKimiK3EmptyUser(c, req))
	require.Equal(t, http.StatusOK, w.Code)
	require.Empty(t, w.Body.String())
}
