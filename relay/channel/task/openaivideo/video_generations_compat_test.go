package openaivideo

import (
	"fmt"
	"io"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildRequestURL_VideoGeneratorUsesInternalPath(t *testing.T) {
	a := newSophnetAdaptor()
	got, err := a.BuildRequestURL(nil)
	require.NoError(t, err)
	assert.Equal(t, "https://www.sophnet.com/api/open-apis/projects/easyllms/video/generations", got)
	assert.NotContains(t, got, "/videogenerator/generate")
}

func TestInit_VideoGeneratorForcesInternalProtocol(t *testing.T) {
	a := &TaskAdaptor{}
	a.Init(&relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:    constant.ChannelTypeVideoGenerator,
			ChannelBaseUrl: "https://internal.example.com",
		},
	})
	assert.Equal(t, ProtocolSophnet, a.protocol)
}

func TestFetchTask_VideoGeneratorPollPathFormat(t *testing.T) {
	assert.Equal(t, "/video/generations", sophnetSubmitPath)
	assert.Equal(t, "/video/generations/%s", sophnetResultFmt)
	assert.Equal(t, "https://example.com/video/generations/task_abc",
		fmt.Sprintf("%s%s", "https://example.com", fmt.Sprintf(sophnetResultFmt, "task_abc")))
}

func TestBuildRequestBody_VideoGeneratorSendsPromptNotContentArray(t *testing.T) {
	a := newSophnetAdaptor()
	c, info, _ := newTestGinAndInfo()
	info.ChannelMeta = &relaycommon.ChannelMeta{
		ChannelType:       constant.ChannelTypeVideoGenerator,
		UpstreamModelName: "seedance-2.0-fast",
	}
	c.Set("task_request", relaycommon.TaskSubmitReq{
		Model:      "seedance-2.0-fast",
		Prompt:     "a cute cat dancing in a sunny garden",
		Duration:   5,
		Resolution: "720p",
		Ratio:      "16:9",
	})

	reader, err := a.BuildRequestBody(c, info)
	require.NoError(t, err)
	raw, err := io.ReadAll(reader)
	require.NoError(t, err)

	var body map[string]any
	require.NoError(t, common.Unmarshal(raw, &body))
	assert.Equal(t, "a cute cat dancing in a sunny garden", body["prompt"])
	assert.Equal(t, "seedance-2.0-fast", body["model"])
	assert.NotContains(t, body, "content")
}

func TestParseTaskResult_Format1DirectUsageTotalTokens(t *testing.T) {
	a := &TaskAdaptor{}
	body := []byte(`{
		"completed_at": 1787050657,
		"content": { "video_url": "https://cdn.example.com/a.mp4" },
		"created_at": 1787050528,
		"duration": 4,
		"error": null,
		"id": "cgt-20260818185526-5qgpz",
		"model": "Seedance 2.0",
		"object": "video.generation",
		"output": { "video_url": "https://cdn.example.com/a.mp4" },
		"progress": 100,
		"ratio": "16:9",
		"resolution": "480p",
		"status": "completed",
		"usage": { "completion_tokens": 40594, "total_tokens": 40594 }
	}`)
	ti, err := a.ParseTaskResult(body)
	require.NoError(t, err)
	require.NotNil(t, ti)
	assert.Equal(t, string(model.TaskStatusSuccess), ti.Status)
	assert.Equal(t, "100%", ti.Progress)
	assert.Equal(t, "https://cdn.example.com/a.mp4", ti.Url)
	assert.Equal(t, 40594, ti.TotalTokens)
	assert.Equal(t, 40594, ti.CompletionTokens)
	assert.Equal(t, 4, ti.Duration)
	assert.Equal(t, "16:9", ti.Ratio)
	assert.Equal(t, "480p", ti.Resolution)
}

func TestParseTaskResult_Format2WrappedCompletionTokens(t *testing.T) {
	a := &TaskAdaptor{}
	body := []byte(`{
		"code": "success",
		"message": "",
		"data": {
			"id": 125999,
			"task_id": "task_svzXN30GKCcnW14RzRIiKjUrtjVzWY5h",
			"status": "SUCCESS",
			"progress": "100%",
			"data": {
				"content": { "video_url": "https://cdn.example.com/b.mp4" },
				"model": "seedance-2.0-standard-multi",
				"status": "succeeded",
				"usage": { "completion_tokens": 191254, "total_tokens": 2008000 }
			}
		}
	}`)
	ti, err := a.ParseTaskResult(body)
	require.NoError(t, err)
	require.NotNil(t, ti)
	assert.Equal(t, string(model.TaskStatusSuccess), ti.Status)
	assert.Equal(t, "100%", ti.Progress)
	assert.Equal(t, "https://cdn.example.com/b.mp4", ti.Url)
	assert.Equal(t, 191254, ti.TotalTokens)
	assert.Equal(t, 191254, ti.CompletionTokens)
}

func TestParseTaskResult_ResultSummaryNestedVideoURL(t *testing.T) {
	a := &TaskAdaptor{}
	body := []byte(`{
		"id": "01a0231f-ace7-7c3f-a7ec-02f8f6dea411",
		"upstreamTaskId": "cgt-20260821150113-fh2fm",
		"status": "succeeded",
		"model": "doubao-seedance-2-5-260628",
		"resultSummary": {
			"content": {"video_url": "https://example.com/seedance.mp4"},
			"duration": "4",
			"resolution": "480p",
			"upstreamStatus": "succeeded",
			"usage": {"completion_tokens": 38830, "total_tokens": 38830}
		}
	}`)
	ti, err := a.ParseTaskResult(body)
	require.NoError(t, err)
	require.NotNil(t, ti)
	assert.Equal(t, string(model.TaskStatusSuccess), ti.Status)
	assert.Equal(t, "https://example.com/seedance.mp4", ti.Url)
	assert.Equal(t, 4, ti.Duration)
	assert.Equal(t, "480p", ti.Resolution)
	assert.Equal(t, 38830, ti.TotalTokens)
	assert.Equal(t, 38830, ti.CompletionTokens)
}

func TestParseTaskResult_Format2MissingKeysDoesNotPanic(t *testing.T) {
	a := &TaskAdaptor{}
	body := []byte(`{"code":"success","data":{"task_id":"t1","status":"RUNNING","progress":"30%","data":{}}}`)
	ti, err := a.ParseTaskResult(body)
	require.NoError(t, err)
	require.NotNil(t, ti)
	assert.Equal(t, string(model.TaskStatusInProgress), ti.Status)
	assert.Equal(t, "30%", ti.Progress)
	assert.Empty(t, ti.Url)
	assert.Equal(t, 0, ti.TotalTokens)
}

func TestParseVideoGeneratorSubmit_Format1AndFormat2AndLegacy(t *testing.T) {
	id, fail, err := parseVideoGeneratorSubmit([]byte(`{"id":"cgt-1","status":"queued","object":"video.generation"}`))
	require.NoError(t, err)
	assert.Empty(t, fail)
	assert.Equal(t, "cgt-1", id)

	id, fail, err = parseVideoGeneratorSubmit([]byte(`{"code":"success","data":{"task_id":"task_wrap","status":"PENDING"}}`))
	require.NoError(t, err)
	assert.Empty(t, fail)
	assert.Equal(t, "task_wrap", id)

	id, fail, err = parseVideoGeneratorSubmit([]byte(`{"status":0,"result":{"task_id":"legacy-1"}}`))
	require.NoError(t, err)
	assert.Empty(t, fail)
	assert.Equal(t, "legacy-1", id)

	_, fail, err = parseVideoGeneratorSubmit([]byte(`{"status":20109,"message":"余额不足","result":null}`))
	require.NoError(t, err)
	assert.Contains(t, fail, "余额不足")
}

func TestConvertToOpenAIVideo_Format2KeepsExternalShape(t *testing.T) {
	a := &TaskAdaptor{}
	task := &model.Task{
		TaskID:     "public_task_1",
		Status:     model.TaskStatusSuccess,
		Progress:   "100%",
		Properties: model.Properties{OriginModelName: "Seedance 2.0"},
		Data: []byte(`{
			"code": "success",
			"data": {
				"task_id": "task_inner",
				"status": "SUCCESS",
				"progress": "100%",
				"data": {
					"content": { "video_url": "https://cdn.example.com/c.mp4" },
					"model": "seedance-2.0-standard-multi",
					"status": "succeeded",
					"usage": { "completion_tokens": 12, "total_tokens": 99 }
				}
			}
		}`),
	}
	body, err := a.ConvertToOpenAIVideo(task)
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, common.Unmarshal(body, &got))
	assert.Equal(t, "public_task_1", got["id"])
	assert.Equal(t, "video.generation", got["object"])
	assert.Equal(t, "completed", got["status"])
	output, ok := got["output"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "https://cdn.example.com/c.mp4", output["video_url"])
	usage, ok := got["usage"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(12), usage["total_tokens"])
	assert.Equal(t, dto.ObjectVideoGeneration, got["object"])
}
