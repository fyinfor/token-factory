package dto

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestExtractVideoPollPassthroughFields(t *testing.T) {
	upstream := []byte(`{
		"status":"succeeded",
		"ratio":"16:9",
		"resolution":"480p",
		"duration":5,
		"usage":{"completion_tokens":50638,"total_tokens":50638}
	}`)
	fields := ExtractVideoPollPassthroughFields(upstream)
	require.Equal(t, "16:9", fields["ratio"])
	require.Equal(t, "480p", fields["resolution"])
	require.Equal(t, float64(5), fields["duration"])
	require.NotNil(t, fields["usage"])
}

func TestExtractVideoPollPassthroughFields_NestedData(t *testing.T) {
	upstream := []byte(`{"code":0,"data":{"ratio":"9:16","duration":3}}`)
	fields := ExtractVideoPollPassthroughFields(upstream)
	require.Equal(t, "9:16", fields["ratio"])
	require.Equal(t, float64(3), fields["duration"])
}

func TestMergeVideoPollPassthroughFields(t *testing.T) {
	response := []byte(`{"id":"local_task","status":"completed","object":"video.generation"}`)
	upstream := []byte(`{"status":"succeeded","ratio":"16:9","resolution":"480p","duration":5,"usage":{"total_tokens":10}}`)
	out, err := MergeVideoPollPassthroughFields(response, upstream)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, common.Unmarshal(out, &m))
	require.Equal(t, "local_task", m["id"])
	require.Equal(t, "16:9", m["ratio"])
	require.Equal(t, "480p", m["resolution"])
	require.Equal(t, float64(5), m["duration"])
	require.NotNil(t, m["usage"])
}

func TestCorrectVideoPollTimestamps_VideoGenerationsPath(t *testing.T) {
	resp := map[string]any{
		"id":           "local_task",
		"created_at":   "2020-01-01T00:00:00Z",
		"completed_at": "2020-01-02T00:00:00Z",
	}
	ts := VideoPollTimestampContext{SubmitTime: 1700000000, FinishTime: 1700000100}
	CorrectVideoPollTimestamps(resp, ts, "/v1/video/generations/local_task")
	require.Equal(t, int64(1700000000), resp["created_at"])
	require.Equal(t, int64(1700000100), resp["completed_at"])
}

func TestCorrectVideoPollTimestamps_VideosCompatPath(t *testing.T) {
	resp := map[string]any{"id": "local_task"}
	ts := VideoPollTimestampContext{SubmitTime: 1700000000, FinishTime: 1700000100}
	CorrectVideoPollTimestamps(resp, ts, "/v1/videos/local_task")
	require.Equal(t, int64(1700000000), resp["created_at"])
	require.Equal(t, int64(1700000100), resp["completed_at"])
}

func TestFinalizeVideoPollResponseJSON_PreservesPassthroughOnVideosPath(t *testing.T) {
	response := []byte(`{"id":"local_task","status":"completed","object":"video.generation","created_at":"2020-01-01T00:00:00Z"}`)
	upstream := []byte(`{
		"ratio":"16:9","resolution":"480p","duration":5,
		"created_at":1781174278,"updated_at":1781174392,
		"usage":{"total_tokens":10}
	}`)
	ts := VideoPollTimestampContext{SubmitTime: 1700000000, FinishTime: 1700000100}
	out, err := FinalizeVideoPollResponseJSON(response, upstream, "/v1/videos/local_task", ts)
	require.NoError(t, err)

	var m map[string]any
	require.NoError(t, common.Unmarshal(out, &m))
	require.Equal(t, "16:9", m["ratio"])
	require.Equal(t, "480p", m["resolution"])
	require.Equal(t, float64(5), m["duration"])
	require.Equal(t, float64(1700000000), m["created_at"])
	require.Equal(t, float64(1700000100), m["completed_at"])
}

func TestParseVideoGenerationsPollUpstream(t *testing.T) {
	raw := []byte(`{
		"content":{"video_url":"https://res.mp4"},
		"created_at":1781174278,
		"duration":5,
		"ratio":"16:9",
		"resolution":"480p",
		"status":"succeeded",
		"updated_at":1781174392,
		"usage":{"completion_tokens":50638,"total_tokens":50638}
	}`)
	upstream := ParseVideoGenerationsPollUpstream(raw)
	require.NotNil(t, upstream)
	require.Equal(t, "16:9", upstream.Ratio)
	require.Equal(t, "480p", upstream.Resolution)
	require.Equal(t, 5, upstream.Duration)
	require.NotNil(t, upstream.Usage)
	require.Equal(t, 50638, upstream.Usage.TotalTokens)
	require.NotNil(t, upstream.Content)
	require.Equal(t, "https://res.mp4", upstream.Content.VideoURL)
}

func TestCorrectVideoPollTimestamps_MatchesTaskLogNotUpstream(t *testing.T) {
	// 任务日志（UTC+8）：提交 16:47:48、结束 16:50:54；上游 created_at/updated_at 偏差约 20 分钟。
	const (
		taskSubmitUnix = int64(1783500468)
		taskFinishUnix = int64(1783500654)
		upstreamCreated = int64(1783499259)
	)
	resp := map[string]any{
		"id":         "local_task",
		"created_at": upstreamCreated,
		"updated_at": int64(1783499381),
	}
	ts := VideoPollTimestampContext{SubmitTime: taskSubmitUnix, FinishTime: taskFinishUnix}
	CorrectVideoPollTimestamps(resp, ts, "/v1/video/generations/local_task")
	require.Equal(t, taskSubmitUnix, resp["created_at"])
	require.Equal(t, taskFinishUnix, resp["completed_at"])
}

func TestIsVideoGenerationsFetchPath(t *testing.T) {
	require.True(t, IsVideoGenerationsFetchPath("/v1/video/generations/task_abc"))
	require.False(t, IsVideoGenerationsFetchPath("/v1/videos/task_abc"))
}
