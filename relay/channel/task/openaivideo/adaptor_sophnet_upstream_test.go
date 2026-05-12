package openaivideo

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newSophnetAdaptor() *TaskAdaptor {
	// 与真实 happyhorse 渠道一致：Sophnet 协议由 base URL 触发；测试中直接固定 protocol。
	return &TaskAdaptor{
		baseURL:  "https://www.sophnet.com/api/open-apis/projects/easyllms",
		protocol: ProtocolSophnet,
		apiKey:   "test-key",
	}
}

func newTestGinAndInfo() (*gin.Context, *relaycommon.RelayInfo, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	info := &relaycommon.RelayInfo{
		TaskRelayInfo: &relaycommon.TaskRelayInfo{
			PublicTaskID: "task_upstream_shape_test",
		},
		OriginModelName: "happyhorse",
	}
	return c, info, w
}

// TestSophnetDoResponse_InformalUpstreamBodies 模拟 Sophnet/happyhorse 上游返回非规范或错误 JSON 时，
// 提交阶段 DoResponse 的错误码与分支（与预扣费之后的解析行为对齐，便于回归）。
func TestSophnetDoResponse_InformalUpstreamBodies(t *testing.T) {
	a := newSophnetAdaptor()

	t.Run("non_json", func(t *testing.T) {
		c, info, w := newTestGinAndInfo()
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(`not json at all`)),
		}
		_, _, taskErr := a.DoResponse(c, resp, info)
		require.NotNil(t, taskErr)
		assert.Equal(t, "unmarshal_response_body_failed", taskErr.Code)
		assert.Equal(t, http.StatusInternalServerError, taskErr.StatusCode)
		assert.Empty(t, w.Body.String())
	})

	t.Run("html_502", func(t *testing.T) {
		c, info, w := newTestGinAndInfo()
		resp := &http.Response{
			StatusCode: http.StatusBadGateway,
			Body:       io.NopCloser(bytes.NewBufferString(`<html><body>502 Bad Gateway</body></html>`)),
		}
		_, _, taskErr := a.DoResponse(c, resp, info)
		require.NotNil(t, taskErr)
		assert.Equal(t, "unmarshal_response_body_failed", taskErr.Code)
		assert.Empty(t, w.Body.String())
	})

	t.Run("empty_object_no_task_id", func(t *testing.T) {
		c, info, w := newTestGinAndInfo()
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(`{}`)),
		}
		_, _, taskErr := a.DoResponse(c, resp, info)
		require.NotNil(t, taskErr)
		assert.Equal(t, "invalid_response", taskErr.Code)
		assert.Contains(t, taskErr.Message, "task id is empty")
		assert.Empty(t, w.Body.String())
	})

	t.Run("status_20109_upstream_business_error", func(t *testing.T) {
		c, info, w := newTestGinAndInfo()
		body := `{"status":20109,"message":"余额不足","result":null}`
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(body)),
		}
		_, _, taskErr := a.DoResponse(c, resp, info)
		require.NotNil(t, taskErr)
		assert.Equal(t, "video_submit_failed", taskErr.Code)
		assert.Equal(t, http.StatusBadRequest, taskErr.StatusCode)
		assert.Contains(t, taskErr.Message, "余额不足")
		assert.Empty(t, w.Body.String())
	})

	t.Run("status_zero_but_task_id_missing", func(t *testing.T) {
		c, info, w := newTestGinAndInfo()
		body := `{"status":0,"message":"","result":{}}`
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(body)),
		}
		_, _, taskErr := a.DoResponse(c, resp, info)
		require.NotNil(t, taskErr)
		assert.Equal(t, "invalid_response", taskErr.Code)
		assert.Contains(t, taskErr.Message, "task id is empty")
		assert.Empty(t, w.Body.String())
	})

	t.Run("success_returns_upstream_task_id", func(t *testing.T) {
		c, info, w := newTestGinAndInfo()
		body := `{"status":0,"message":"","result":{"task_id":"upstream-task-abc"}}`
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewBufferString(body)),
		}
		taskID, raw, taskErr := a.DoResponse(c, resp, info)
		require.Nil(t, taskErr)
		assert.Equal(t, "upstream-task-abc", taskID)
		assert.Equal(t, body, string(raw))
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "task_upstream_shape_test")
	})
}

// TestSophnetParseTaskResult_InformalBodies 轮询阶段 ParseTaskResult：非 JSON / 非 Sophnet 形 / 业务失败。
func TestSophnetParseTaskResult_InformalBodies(t *testing.T) {
	a := newSophnetAdaptor()

	t.Run("garbage_falls_through_ark_and_errors", func(t *testing.T) {
		_, err := a.ParseTaskResult([]byte(`{"id":123}`))
		require.Error(t, err)
	})

	t.Run("sophnet_top_level_failure", func(t *testing.T) {
		body := `{"status":500,"message":"内部错误","result":null}`
		ti, err := a.ParseTaskResult([]byte(body))
		require.NoError(t, err)
		require.NotNil(t, ti)
		assert.Equal(t, model.TaskStatusFailure, ti.Status)
		assert.Equal(t, "100%", ti.Progress)
		assert.Contains(t, ti.Reason, "内部错误")
	})

	t.Run("sophnet_wrapped_ark_success", func(t *testing.T) {
		body := `{"status":0,"message":"","result":{"id":"vid-1","model":"happyhorse-1.0-t2v","status":"succeeded","content":{"video_url":"https://example.com/v.mp4"}}}`
		ti, err := a.ParseTaskResult([]byte(body))
		require.NoError(t, err)
		require.NotNil(t, ti)
		assert.Contains(t, ti.Url, "v.mp4")
	})
}

// TestDetectProtocol_SophnetMarkers 与 happyhorse 渠道 base_url 判定一致。
func TestDetectProtocol_SophnetMarkers(t *testing.T) {
	assert.Equal(t, ProtocolSophnet, DetectProtocol("https://www.sophnet.com/api/open-apis/projects/easyllms/foo"))
	assert.Equal(t, ProtocolSophnet, DetectProtocol("http://127.0.0.1:9999/videogenerator/api"))
	assert.Equal(t, ProtocolMaaS, DetectProtocol("https://maas.hidreamai.com/api/maas/gw"))
	assert.Equal(t, ProtocolMaaS, DetectProtocol("https://hiharness.hidreamai.com"))
	assert.Equal(t, ProtocolArk, DetectProtocol("https://reseller.example.com/v1"))
}

func TestNormalizeMaaSBaseURL_HiHarness(t *testing.T) {
	assert.Equal(t, "https://hiharness.hidreamai.com/api/maas/gw", normalizeMaaSBaseURL("https://hiharness.hidreamai.com"))
	assert.Equal(t, "https://hiharness.hidreamai.com/api/maas/gw", normalizeMaaSBaseURL("https://hiharness.hidreamai.com/api/maas/gw"))
	assert.Equal(t, "https://maas.hidreamai.com", normalizeMaaSBaseURL("https://maas.hidreamai.com"))
}

func TestBuildMaasPayloadMap_AutoConvertsPlaygroundVideoFields(t *testing.T) {
	a := &TaskAdaptor{}
	req := &relaycommon.TaskSubmitReq{
		Model:    "Seedance2.0",
		Prompt:   "跳舞的小女孩",
		Images:   []string{"https://example.com/ref.png"},
		Size:     "1280x720",
		Duration: 5,
		Metadata: map[string]any{
			"generate_audio": true,
			"video_urls":     []any{"https://example.com/ref.mp4?token=abc"},
			"audio_urls":     []string{"https://example.com/ref.mp3"},
		},
	}

	body, err := a.buildMaasPayloadMap(req)
	require.NoError(t, err)

	assert.Equal(t, "Seedance2.0", body["model_id"])
	assert.Equal(t, 5, body["duration"])
	assert.Equal(t, "720p", body["resolution"])
	assert.Equal(t, "16:9", body["ratio"])
	assert.Equal(t, true, body["generate_audio"])
	assert.NotContains(t, body, "video_urls")
	assert.NotContains(t, body, "audio_urls")

	content, ok := body["content"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, content, 4)
	assert.Equal(t, "text", content[0]["type"])
	assert.Equal(t, "image_url", content[1]["type"])
	assert.Equal(t, "video_url", content[2]["type"])
	assert.Equal(t, "audio_url", content[3]["type"])
}

func TestConvertToOpenAIVideo_MaaSCompletedShape(t *testing.T) {
	a := &TaskAdaptor{}
	task := &model.Task{
		TaskID:     "task_wjlMGb4cfEgrqq7oubXjWPdXPmSjfRfe",
		Status:     model.TaskStatusSuccess,
		Progress:   "100%",
		CreatedAt:  1778293580,
		FinishTime: 1778293807,
		Properties: model.Properties{OriginModelName: "Seedance2.0"},
		Data: []byte(`{
			"code": 0,
			"message": "Success",
			"result": {
				"status": 1,
				"sub_task_results": [
					{
						"url": "https://media.hidreamai.com/03fbb389-91ad-4f59-b5ce-c57c41770209.mp4",
						"task_status": 1
					}
				]
			}
		}`),
	}

	body, err := a.ConvertToOpenAIVideo(task)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, common.Unmarshal(body, &got))
	assert.Equal(t, "task_wjlMGb4cfEgrqq7oubXjWPdXPmSjfRfe", got["id"])
	assert.Equal(t, "video.generation", got["object"])
	assert.Equal(t, "Seedance2.0", got["model"])
	assert.Equal(t, "completed", got["status"])
	assert.Equal(t, float64(100), got["progress"])
	assert.Nil(t, got["error"])
	output, ok := got["output"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "https://media.hidreamai.com/03fbb389-91ad-4f59-b5ce-c57c41770209.mp4", output["video_url"])
}

func TestDetectResponseProtocol_SophnetNumericStatusWithResult(t *testing.T) {
	raw := []byte(`{"status":0,"result":{"task_id":"x"}}`)
	assert.Equal(t, ProtocolSophnet, detectResponseProtocol(raw))
}

func TestDetectResponseProtocol_MalformedUsesArk(t *testing.T) {
	raw := []byte(`not json`)
	assert.Equal(t, ProtocolArk, detectResponseProtocol(raw))
}

func TestDetectResponseProtocol_ArrayBodyUsesArk(t *testing.T) {
	raw := []byte(`[{"status":0}]`)
	assert.Equal(t, ProtocolArk, detectResponseProtocol(raw))
}
