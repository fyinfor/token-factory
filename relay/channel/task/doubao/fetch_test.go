package doubao

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchURLByAPI(t *testing.T) {
	base := "https://ark.cn-beijing.volces.com/"
	taskID := "cgt-20260821150113-fh2fm"

	assert.Equal(t,
		"https://ark.cn-beijing.volces.com/api/v3/contents/generations/tasks/"+taskID,
		FetchURL(base, taskID),
		"默认查询路径应保持 Contents API")
	assert.Equal(t,
		"https://ark.cn-beijing.volces.com/api/v3/contents/generations/tasks/"+taskID,
		FetchURLByAPI(base, taskID, FetchAPIContentsGenerations))
	assert.Equal(t,
		"https://ark.cn-beijing.volces.com/v1/video/generations/"+taskID,
		FetchURLByAPI(base, taskID, FetchAPIVideoGenerations))
}

func TestNormalizeFetchAPI(t *testing.T) {
	assert.Equal(t, FetchAPIContentsGenerations, NormalizeFetchAPI(""))
	assert.Equal(t, FetchAPIContentsGenerations, NormalizeFetchAPI("contents_generations"))
	assert.Equal(t, FetchAPIVideoGenerations, NormalizeFetchAPI("video_generations"))
	assert.Equal(t, FetchAPIVideoGenerations, NormalizeFetchAPI("v1"))
}

func TestQueryTask_InvalidTaskID(t *testing.T) {
	resp, err := QueryTask("https://example.com", "sk-test", "  ", FetchAPIContentsGenerations, "")
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "invalid task_id")
}

func TestFetchTask_NilBody(t *testing.T) {
	a := &TaskAdaptor{}
	resp, err := a.FetchTask("https://example.com", "sk-test", nil, "")
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "invalid task_id")
}

func TestAdaptFetchResponseJSON_ContentsAPIPassthrough(t *testing.T) {
	raw := []byte(`{
		"id":"cgt-1",
		"status":"succeeded",
		"content":{"video_url":"https://res.mp4","last_frame_url":"https://last.jpg"},
		"usage":{"completion_tokens":10,"total_tokens":10}
	}`)
	out := AdaptFetchResponseJSON(raw)
	var got map[string]any
	require.NoError(t, common.Unmarshal(out, &got))
	content := got["content"].(map[string]any)
	assert.Equal(t, "https://res.mp4", content["video_url"])
	assert.Equal(t, "https://last.jpg", content["last_frame_url"])
}

func TestAdaptFetchResponseJSON_VideoGenerationsOutput(t *testing.T) {
	raw := []byte(`{
		"id":"task_abc",
		"object":"video.generation",
		"status":"completed",
		"output":{"video_url":"https://cdn.example.com/a.mp4"},
		"metadata":{"url":"https://cdn.example.com/a.mp4","last_frame_url":"https://cdn.example.com/last.jpg"},
		"usage":{"completion_tokens":88,"total_tokens":88}
	}`)
	out := AdaptFetchResponseJSON(raw)
	var got map[string]any
	require.NoError(t, common.Unmarshal(out, &got))
	content := got["content"].(map[string]any)
	assert.Equal(t, "https://cdn.example.com/a.mp4", content["video_url"])
	assert.Equal(t, "https://cdn.example.com/last.jpg", content["last_frame_url"])
}

func TestParseTaskResult_VideoGenerationsOutputURL(t *testing.T) {
	a := &TaskAdaptor{}
	ti, err := a.ParseTaskResult([]byte(`{
		"id":"task_abc",
		"status":"completed",
		"output":{"video_url":"https://cdn.example.com/out.mp4"},
		"duration":5,
		"resolution":"720p",
		"usage":{"completion_tokens":100,"total_tokens":100}
	}`))
	require.NoError(t, err)
	require.NotNil(t, ti)
	assert.Equal(t, string(model.TaskStatusSuccess), ti.Status)
	assert.Equal(t, "https://cdn.example.com/out.mp4", ti.Url)
	assert.Equal(t, 5, ti.Duration)
	assert.Equal(t, "720p", ti.Resolution)
	assert.Equal(t, 100, ti.TotalTokens)
}

func TestBuildContentsGenerationsClientJSON(t *testing.T) {
	task := &model.Task{
		TaskID:     "cgt-public-id",
		Status:     model.TaskStatusSuccess,
		SubmitTime: 1700000000,
		FinishTime: 1700000100,
		Data: []byte(`{
			"id":"upstream-id",
			"status":"completed",
			"output":{"video_url":"https://cdn.example.com/out.mp4"},
			"usage":{"total_tokens":12}
		}`),
		Properties: model.Properties{OriginModelName: "doubao-seedance-2-0-260128"},
	}
	out, err := BuildContentsGenerationsClientJSON(task, task.Data)
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, common.Unmarshal(out, &got))
	assert.Equal(t, "cgt-public-id", got["id"])
	assert.Equal(t, "completed", got["status"])
	assert.Equal(t, "doubao-seedance-2-0-260128", got["model"])
	content := got["content"].(map[string]any)
	assert.Equal(t, "https://cdn.example.com/out.mp4", content["video_url"])
	_, hasOutput := got["output"]
	assert.False(t, hasOutput)
	_, hasObject := got["object"]
	assert.False(t, hasObject)
	assert.Equal(t, float64(1700000000), got["created_at"])
	assert.Equal(t, float64(1700000100), got["updated_at"])
}
