package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
)

func TestResolveVideoOutputSpecFromUpstream_TaskResultFirst(t *testing.T) {
	task := &model.Task{
		Data: []byte(`{"resolution":"720p","duration":8,"ratio":"9:16"}`),
	}
	taskResult := &relaycommon.TaskInfo{
		Resolution: "480p",
		Duration:   5,
		Ratio:      "16:9",
	}
	spec := resolveVideoOutputSpecFromUpstream(task, taskResult)
	require.Equal(t, "480p", spec.Resolution)
	require.Equal(t, 5, spec.Duration)
	require.Equal(t, "16:9", spec.Ratio)
}

func TestResolveVideoOutputSpecFromUpstream_DashScopeUsage(t *testing.T) {
	task := &model.Task{
		Data: []byte(`{
			"output": {"task_status":"SUCCEEDED","video_url":"https://example.com/a.mp4"},
			"usage": {"duration":5,"output_video_duration":5,"SR":720,"ratio":"16:9"}
		}`),
	}
	spec := resolveVideoOutputSpecFromUpstream(task, nil)
	require.Equal(t, "720p", spec.Resolution)
	require.Equal(t, 5, spec.Duration)
	require.Equal(t, "16:9", spec.Ratio)
}

func TestResolveSeedanceVideoSpec_FallsBackToRequest(t *testing.T) {
	task := &model.Task{
		Properties: model.Properties{
			Input: `{"size":"720p","duration":10,"metadata":{"ratio":"16:9"}}`,
		},
	}
	spec := resolveSeedanceVideoSpec(task, nil)
	require.Equal(t, "720p", spec.Resolution)
	require.Equal(t, 10, spec.Duration)
	require.Equal(t, "16:9", spec.Ratio)
}

func TestResolveSeedanceVideoSpec_UpstreamOverridesRequest(t *testing.T) {
	task := &model.Task{
		Properties: model.Properties{
			Input: `{"size":"720p","duration":10,"metadata":{"ratio":"16:9"}}`,
		},
	}
	taskResult := &relaycommon.TaskInfo{
		Resolution: "480p",
		Duration:   5,
		Ratio:      "16:9",
	}
	spec := resolveSeedanceVideoSpec(task, taskResult)
	require.Equal(t, "480p", spec.Resolution)
	require.Equal(t, 5, spec.Duration)
	require.Equal(t, "16:9", spec.Ratio)
}

func TestVideoMetadataFromTaskCompletion_SeedancePayload(t *testing.T) {
	raw := []byte(`{
		"id": "task_abc",
		"status": "succeeded",
		"duration": 5,
		"resolution": "480p",
		"ratio": "16:9",
		"content": {"video_url": "https://res.example.com/out.mp4"}
	}`)
	task := &model.Task{Data: raw}
	meta, ok := videoMetadataFromTaskCompletion(task, nil)
	require.True(t, ok)
	require.InDelta(t, 5, meta.DurationSec, 1e-9)
	require.Equal(t, 854, meta.Width)
	require.Equal(t, 480, meta.Height)
}

func TestVideoDimensionsFromTaskCompletion_TaskResultOnly(t *testing.T) {
	task := &model.Task{}
	taskResult := &relaycommon.TaskInfo{
		Resolution: "720p",
		Duration:   8,
		Ratio:      "16:9",
	}
	w, h, ok := videoDimensionsFromTaskCompletion(task, taskResult)
	require.True(t, ok)
	require.Equal(t, 1280, w)
	require.Equal(t, 720, h)
}

func TestVideoMetadataFromTaskCompletion_PrefersTaskDataOverURLProbeOrder(t *testing.T) {
	var payload map[string]any
	require.NoError(t, common.Unmarshal([]byte(`{
		"id":"task_x","status":"succeeded","duration":5,"resolution":"480p","ratio":"16:9",
		"content":{"video_url":"https://example.com/x.mp4"}
	}`), &payload))
	meta, ok := extractVideoMetadataFromMap(payload)
	require.True(t, ok)
	require.Equal(t, 5.0, meta.DurationSec)
}

func TestVideoPerTokenBillingDetailFromTask_PrefersUpstreamResolution(t *testing.T) {
	task := &model.Task{
		Data: []byte(`{"resolution":"480p","duration":5,"ratio":"16:9","status":"succeeded"}`),
		Properties: model.Properties{
			Input:           `{"size":"720p","duration":10,"metadata":{"ratio":"16:9"}}`,
			OriginModelName: "seedance-test",
		},
	}
	match := &videoTokenRuleMatch{
		Resolution:           "1280x720",
		RuleWidth:            1280,
		RuleHeight:           720,
		ChannelPricePerToken: 0.15,
	}
	spec := resolveSeedanceVideoSpec(task, &relaycommon.TaskInfo{
		Resolution: "480p",
		Duration:   5,
		Ratio:      "16:9",
	})
	detail := videoPerTokenBillingDetailFromTask(task, match, spec, 50000)
	require.NotNil(t, detail)
	require.Equal(t, "480p", detail.Resolution)
	require.Equal(t, 854, detail.Width)
	require.Equal(t, 480, detail.Height)
}
