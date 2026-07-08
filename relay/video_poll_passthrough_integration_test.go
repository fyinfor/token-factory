package relay

import (
	"strconv"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	taskdoubao "github.com/QuantumNous/new-api/relay/channel/task/doubao"
	taskopenaivideo "github.com/QuantumNous/new-api/relay/channel/task/openaivideo"
	"github.com/stretchr/testify/require"
)

const seedanceUpstreamPollJSON = `{
    "content": {"video_url": "https://res.mp4"},
    "created_at": 1781174278,
    "duration": 5,
    "id": "task_WZNVZhE1MQ1LX9vTy7pTdv2bmTYNx6x8",
    "model": "doubao-seedance-2-0-260128",
    "ratio": "16:9",
    "resolution": "480p",
    "status": "succeeded",
    "updated_at": 1781174392,
    "usage": {"completion_tokens": 50638, "total_tokens": 50638}
}`

func TestBuildOpenAIVideoPollResponse_SeedanceUpstreamShape(t *testing.T) {
	upstream := []byte(seedanceUpstreamPollJSON)
	task := &model.Task{
		TaskID:     "local_task",
		Platform:   constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeSeedance)),
		Data:       upstream,
		SubmitTime: 1700000000,
		FinishTime: 1700000100,
	}
	for _, path := range []string{
		"/v1/video/generations/local_task",
		"/v1/videos/local_task",
	} {
		out := buildOpenAIVideoPollResponse(task, upstream, path)
		require.NotEmpty(t, out, path)
		s := string(out)
		require.Contains(t, s, `"ratio":"16:9"`, path)
		require.Contains(t, s, `"resolution":"480p"`, path)
		require.Contains(t, s, `"duration":5`, path)
		require.Contains(t, s, `"usage":`, path)
		if path == "/v1/videos/local_task" {
			require.Contains(t, s, `"created_at":1700000000`, path)
			require.Contains(t, s, `"completed_at":1700000100`, path)
		} else {
			require.Contains(t, s, dto.FormatTimeUnixRFC3339(1700000000), path)
			require.Contains(t, s, dto.FormatTimeUnixRFC3339(1700000100), path)
		}
	}
}

func TestVideoPollPassthrough_DoubaoConvert(t *testing.T) {
	upstream := []byte(seedanceUpstreamPollJSON)
	task := &model.Task{
		TaskID:   "local_task",
		Platform: constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeDoubaoVideo)),
		Data:     upstream,
		Status:   model.TaskStatusSuccess,
	}
	raw, err := (&taskdoubao.TaskAdaptor{}).ConvertToOpenAIVideo(task)
	require.NoError(t, err)
	ts := dto.VideoPollTimestampContext{SubmitTime: 1700000000, FinishTime: 1700000100}
	out, err := dto.FinalizeVideoPollResponseJSON(raw, upstream, "/v1/video/generations/local_task", ts)
	require.NoError(t, err)
	s := string(out)
	require.Contains(t, s, `"ratio":"16:9"`)
	require.Contains(t, s, `"resolution":"480p"`)
	require.Contains(t, s, `"duration":5`)
}

func TestVideoPollPassthrough_OpenAIVideoConvert(t *testing.T) {
	upstream := []byte(seedanceUpstreamPollJSON)
	task := &model.Task{
		TaskID:   "local_task",
		Platform: constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeOpenAIVideo)),
		Data:     upstream,
		Status:   model.TaskStatusSuccess,
	}
	raw, err := (&taskopenaivideo.TaskAdaptor{}).ConvertToOpenAIVideo(task)
	require.NoError(t, err)
	ts := dto.VideoPollTimestampContext{SubmitTime: 1700000000, FinishTime: 1700000100}
	out, err := dto.FinalizeVideoPollResponseJSON(raw, upstream, "/v1/video/generations/local_task", ts)
	require.NoError(t, err)
	require.Contains(t, string(out), `"ratio":"16:9"`)
}

func TestVideoPollPassthrough_EmptyTaskData(t *testing.T) {
	task := &model.Task{TaskID: "local_task", Data: nil}
	raw, err := (&taskopenaivideo.TaskAdaptor{}).ConvertToOpenAIVideo(task)
	require.NoError(t, err)
	out, err := dto.MergeVideoPollPassthroughFields(raw, task.Data)
	require.NoError(t, err)
	require.False(t, strings.Contains(string(out), `"ratio"`))
}
