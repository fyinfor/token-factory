package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGetModelRequestPlaygroundVideoPreservesSpecificChannelID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(
		http.MethodPost,
		"/api/playground/videos",
		strings.NewReader(`{"model":"Seedance2.0","prompt":"test","duration":5,"size":"960x540","specific_channel_id":60}`),
	)
	c.Request.Header.Set("Content-Type", "application/json")

	modelRequest, shouldSelectChannel, err := getModelRequest(c)
	require.NoError(t, err)
	require.True(t, shouldSelectChannel)
	require.NotNil(t, modelRequest)
	require.Equal(t, "Seedance2.0", modelRequest.Model)
	require.NotNil(t, modelRequest.SpecificChannelID)
	require.Equal(t, 60, *modelRequest.SpecificChannelID)

	rawChannelID, ok := common.GetContextKey(c, constant.ContextKeyTokenSpecificChannelId)
	require.True(t, ok)
	require.Equal(t, "60", rawChannelID)
	require.Equal(t, relayconstant.RelayModeVideoSubmit, c.GetInt("relay_mode"))
}
