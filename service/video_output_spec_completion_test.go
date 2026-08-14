package service

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
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

func withUpscalePrice(t *testing.T, modelName string) {
	t.Helper()
	prev := ratio_setting.VideoPricingRules2JSONString()
	t.Cleanup(func() { _ = ratio_setting.UpdateVideoPricingRulesByJSONString(prev) })
	cfg := `{"` + modelName + `":{"video_upscale_per_second":[{"resolution":"720p","source_resolution":"480p","price":0.02}]}}`
	require.NoError(t, ratio_setting.UpdateVideoPricingRulesByJSONString(cfg))
}

func TestVideoDimensionsFromTaskCompletion_UpscaleSkipsUpstreamCalibration(t *testing.T) {
	modelName := "upscale-keep-target-res"
	withUpscalePrice(t, modelName)
	task := &model.Task{
		Properties: model.Properties{
			OriginModelName: modelName,
			Input:           `{"resolution":"480p","metadata":{"resolution":"480p","ratio":"16:9"}}`,
		},
		Data: []byte(`{"id":"task_x","status":"succeeded","duration":4,"resolution":"480p","ratio":"16:9"}`),
		PrivateData: model.TaskPrivateData{
			VideoUpscale: &model.TaskVideoUpscaleInfo{
				SourceResolution: "480p",
				TargetResolution: "720p",
				Status:           model.TaskVideoUpscaleStatusSuccess,
			},
		},
	}
	w, h, ok := videoDimensionsFromTaskCompletion(task, &relaycommon.TaskInfo{
		Resolution: "480p",
		Ratio:      "16:9",
		Duration:   4,
	})
	require.True(t, ok)
	require.Equal(t, 1280, w)
	require.Equal(t, 720, h)
}

func TestVideoDimensionsFromTaskCompletion_UpscaleWithoutPriceCalibratesUpstream(t *testing.T) {
	task := &model.Task{
		Properties: model.Properties{
			OriginModelName: "no-upscale-price-model",
			Input:           `{"resolution":"720p","metadata":{"resolution":"720p","ratio":"16:9"}}`,
		},
		Data: []byte(`{"id":"task_x","status":"succeeded","duration":4,"resolution":"480p","ratio":"16:9"}`),
		PrivateData: model.TaskPrivateData{
			VideoUpscale: &model.TaskVideoUpscaleInfo{
				SourceResolution: "480p",
				TargetResolution: "720p",
				Status:           model.TaskVideoUpscaleStatusSuccess,
			},
		},
	}
	w, h, ok := videoDimensionsFromTaskCompletion(task, &relaycommon.TaskInfo{
		Resolution: "480p",
		Ratio:      "16:9",
		Duration:   4,
	})
	require.True(t, ok)
	require.Equal(t, 854, w)
	require.Equal(t, 480, h)
}

func TestVideoMetadataFromTaskCompletion_UpscaleDurationPreferred(t *testing.T) {
	modelName := "upscale-duration-meta"
	withUpscalePrice(t, modelName)
	task := &model.Task{
		Properties: model.Properties{
			OriginModelName: modelName,
			Input:           `{"resolution":"720p","metadata":{"resolution":"720p","ratio":"16:9","has_audio":true}}`,
		},
		Data: []byte(`{"id":"task_x","status":"succeeded","duration":5,"resolution":"480p","ratio":"16:9"}`),
		PrivateData: model.TaskPrivateData{
			VideoUpscale: &model.TaskVideoUpscaleInfo{
				SourceResolution: "480p",
				TargetResolution: "720p",
				Status:           model.TaskVideoUpscaleStatusSuccess,
				DurationSec:      4.2,
			},
		},
	}
	meta, ok := videoMetadataFromTaskCompletion(task, &relaycommon.TaskInfo{
		Resolution: "480p",
		Duration:   5,
	})
	require.True(t, ok)
	require.InDelta(t, 4.2, meta.DurationSec, 0.001)
	require.Equal(t, 1280, meta.Width)
	require.Equal(t, 720, meta.Height)
	require.True(t, meta.HasAudio)
}

func TestShouldKeepUpscaleTargetResolution(t *testing.T) {
	modelName := "upscale-gate-model"
	withUpscalePrice(t, modelName)
	task := &model.Task{
		Properties: model.Properties{OriginModelName: modelName},
		PrivateData: model.TaskPrivateData{
			VideoUpscale: &model.TaskVideoUpscaleInfo{
				SourceResolution: "480p",
				TargetResolution: "720p",
				Status:           model.TaskVideoUpscaleStatusSuccess,
			},
		},
	}
	require.True(t, shouldKeepUpscaleTargetResolution(task))

	task.PrivateData.VideoUpscale.Status = model.TaskVideoUpscaleStatusFailed
	require.False(t, shouldKeepUpscaleTargetResolution(task))

	task.PrivateData.VideoUpscale.Status = model.TaskVideoUpscaleStatusSuccess
	task.Properties.OriginModelName = "no-price"
	require.False(t, shouldKeepUpscaleTargetResolution(task))
}
