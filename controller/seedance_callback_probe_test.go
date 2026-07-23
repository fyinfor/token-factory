package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSeedanceCallbackProbe_CreateReceiveInspect(t *testing.T) {
	gin.SetMode(gin.TestMode)
	seedanceCallbackProbes = &seedanceCallbackProbeStore{
		sessions: make(map[string]*seedanceCallbackProbeSession),
	}

	createW := httptest.NewRecorder()
	createC, _ := gin.CreateTestContext(createW)
	createC.Request = httptest.NewRequest(http.MethodPost, "/api/debug/seedance/callback", nil)
	CreateSeedanceCallbackProbe(createC)
	require.Equal(t, http.StatusOK, createW.Code)

	var createResp map[string]any
	require.NoError(t, common.Unmarshal(createW.Body.Bytes(), &createResp))
	require.Equal(t, true, createResp["success"])
	data, ok := createResp["data"].(map[string]any)
	require.True(t, ok)
	token, _ := data["token"].(string)
	require.NotEmpty(t, token)
	callbackURL, _ := data["callback_url"].(string)
	require.Contains(t, callbackURL, token)

	payload := []byte(`{"id":"cgt-test","status":"succeeded","content":{"video_url":"https://res.mp4","last_frame_url":"https://res.jpg"}}`)
	recvW := httptest.NewRecorder()
	recvC, _ := gin.CreateTestContext(recvW)
	recvC.Params = gin.Params{{Key: "token", Value: token}}
	recvC.Request = httptest.NewRequest(http.MethodPost, "/api/debug/seedance/callback/"+token, bytes.NewReader(payload))
	recvC.Request.Header.Set("Content-Type", "application/json")
	ReceiveSeedanceCallbackProbe(recvC)
	require.Equal(t, http.StatusOK, recvW.Code)

	inspectW := httptest.NewRecorder()
	inspectC, _ := gin.CreateTestContext(inspectW)
	inspectC.Params = gin.Params{{Key: "token", Value: token}}
	inspectC.Request = httptest.NewRequest(http.MethodGet, "/api/debug/seedance/callback/"+token, nil)
	InspectSeedanceCallbackProbe(inspectC)
	require.Equal(t, http.StatusOK, inspectW.Code)

	var inspectResp map[string]any
	require.NoError(t, common.Unmarshal(inspectW.Body.Bytes(), &inspectResp))
	inspectData, ok := inspectResp["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, inspectData["called"])
	require.Equal(t, float64(1), inspectData["call_count"])
	last, ok := inspectData["last_event"].(map[string]any)
	require.True(t, ok)
	require.Contains(t, last["body"], "last_frame_url")
}
