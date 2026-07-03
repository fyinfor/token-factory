package openaivideo

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestClassifyTfOpenVideoClientPath_PlaygroundVideos(t *testing.T) {
	require.Equal(t, tfStyleOpenAIVideos, classifyTfOpenVideoClientPath("/api/playground/videos"))
	require.Equal(t, tfStyleOpenAIVideos, classifyTfOpenVideoClientPath("https://host/api/playground/videos"))
	require.Equal(t, tfStyleOpenAIVideos, classifyTfOpenVideoClientPath("/v1/videos"))
	require.Equal(t, tfStyleVideoGenerations, classifyTfOpenVideoClientPath("/v1/video/generations"))
}

func TestPassthroughOpenAIVideoJSON_ReplacesPublicTaskID(t *testing.T) {
	body := []byte(`{"id":"upstream-id","status":"queued","object":"video.generation","created_at":"2026-07-03T03:16:23Z"}`)
	out, err := PassthroughOpenAIVideoJSON(body, "task_local_1", "/api/playground/videos")
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, common.Unmarshal(out, &m))
	require.Equal(t, "task_local_1", m["id"])
	require.Equal(t, "queued", m["status"])
}

func TestDoResponse_TokenFactoryOpenPassthroughSubmit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/api/playground/videos", nil)

	a := &TaskAdaptor{protocol: ProtocolTokenFactory}
	info := &relaycommon.RelayInfo{
		OriginModelName: "Seedance 2.0",
		TaskRelayInfo: &relaycommon.TaskRelayInfo{
			PublicTaskID: "task_public_1",
		},
		TfOpenVideoUpstreamStyle: tfStyleOpenAIVideos,
	}
	respBody := []byte(`{"id":"upstream-abc","status":"queued","object":"video.generation"}`)
	httpResp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(respBody)),
		Header:     make(http.Header),
	}

	taskID, raw, taskErr := a.DoResponse(c, httpResp, info)
	require.Nil(t, taskErr)
	require.Equal(t, "upstream-abc", taskID)
	require.JSONEq(t, string(respBody), string(raw))
}
