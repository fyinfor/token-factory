package service

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
)

func TestVideoDimensionsFromTaskCompletion_UpstreamResolutionBeforeRequestSize(t *testing.T) {
	task := &model.Task{
		Properties: model.Properties{
			Input: `{"size":"1920x1080","metadata":{"resolution":"720p","ratio":"16:9"}}`,
		},
		Data: []byte(`{"id":"task_x","status":"succeeded","duration":5,"resolution":"480p","ratio":"16:9"}`),
	}
	w, h, ok := videoDimensionsFromTaskCompletion(task, nil)
	require.True(t, ok)
	require.Equal(t, 854, w)
	require.Equal(t, 480, h)
}

func TestVideoDimensionsFromTaskCompletion_FallsBackToRequestResolution(t *testing.T) {
	task := &model.Task{
		Properties: model.Properties{
			Input: `{"size":"1920x1080","metadata":{"resolution":"720p","ratio":"16:9"}}`,
		},
		Data: []byte(`{"id":"task_x","status":"succeeded","duration":5}`),
	}
	w, h, ok := videoDimensionsFromTaskCompletion(task, nil)
	require.True(t, ok)
	require.Equal(t, 1280, w)
	require.Equal(t, 720, h)
}

func TestVideoDimensionsFromTaskCompletion_FallsBackToRequestSize(t *testing.T) {
	task := &model.Task{
		Properties: model.Properties{
			Input: `{"size":"1920x1080","duration":5}`,
		},
		Data: []byte(`{"id":"task_x","status":"succeeded","duration":5}`),
	}
	w, h, ok := videoDimensionsFromTaskCompletion(task, nil)
	require.True(t, ok)
	require.Equal(t, 1920, w)
	require.Equal(t, 1080, h)
}

func TestVideoDimensionsFromTaskCompletion_TaskResultResolution(t *testing.T) {
	task := &model.Task{
		Properties: model.Properties{
			Input: `{"size":"1920x1080","metadata":{"resolution":"720p","ratio":"16:9"}}`,
		},
	}
	taskResult := &relaycommon.TaskInfo{
		Resolution: "480p",
		Ratio:      "16:9",
		Duration:   5,
	}
	w, h, ok := videoDimensionsFromTaskCompletion(task, taskResult)
	require.True(t, ok)
	require.Equal(t, 854, w)
	require.Equal(t, 480, h)
}
