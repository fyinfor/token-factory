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

func TestKimiK3EmptyUserMessageIndex(t *testing.T) {
	_, hit := kimiK3EmptyUserMessageIndex(nil)
	require.False(t, hit)
	_, hit = kimiK3EmptyUserMessageIndex(&dto.GeneralOpenAIRequest{})
	require.False(t, hit)

	pos, hit := kimiK3EmptyUserMessageIndex(&dto.GeneralOpenAIRequest{
		Messages: []dto.Message{{Role: "user", Content: ""}},
	})
	require.True(t, hit)
	require.Equal(t, 0, pos)

	pos, hit = kimiK3EmptyUserMessageIndex(&dto.GeneralOpenAIRequest{
		Messages: []dto.Message{
			{Role: "system", Content: "你是一位资深的软件工程师，请用专业、简洁且易懂的语言回答用户的问题。"},
			{Role: "user", Content: ""},
		},
	})
	require.True(t, hit)
	require.Equal(t, 1, pos)

	_, hit = kimiK3EmptyUserMessageIndex(&dto.GeneralOpenAIRequest{
		Messages: []dto.Message{
			{Role: "system", Content: "sys"},
			{Role: "user", Content: "hello"},
		},
	})
	require.False(t, hit)
	_, hit = kimiK3EmptyUserMessageIndex(&dto.GeneralOpenAIRequest{
		Messages: []dto.Message{{Role: "assistant", Content: ""}},
	})
	require.False(t, hit)
	_, hit = kimiK3EmptyUserMessageIndex(&dto.GeneralOpenAIRequest{
		Messages: []dto.Message{{Role: "user", Content: nil}},
	})
	require.False(t, hit)
	_, hit = kimiK3EmptyUserMessageIndex(&dto.GeneralOpenAIRequest{
		Messages: []dto.Message{{Role: "user", Content: []any{}}},
	})
	require.False(t, hit)
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
	_, hit := shouldRejectKimiK3EmptyUser(c, req)
	require.False(t, hit)
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
	pos, hit := shouldRejectKimiK3EmptyUser(c, req)
	require.True(t, hit)
	require.Equal(t, 0, pos)
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
	require.Equal(t, kimiK3EmptyUserErrorMessage(0), errObj["message"])
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

func TestAbortIfKimiK3EmptyUser_SystemThenEmptyUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req := &dto.GeneralOpenAIRequest{
		Model: "kimi-k3",
		Messages: []dto.Message{
			{Role: "system", Content: "你是一位资深的软件工程师，请用专业、简洁且易懂的语言回答用户的问题。"},
			{Role: "user", Content: ""},
		},
	}
	require.True(t, AbortIfKimiK3EmptyUser(c, req))
	require.Equal(t, http.StatusBadRequest, w.Code)

	var body map[string]any
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &body))
	errObj, ok := body["error"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, kimiK3EmptyUserErrorMessage(1), errObj["message"])
	require.Equal(t, kimiK3EmptyUserErrorType, errObj["type"])
	require.Len(t, errObj, 2)
}
