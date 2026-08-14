package service

import (
	"encoding/json"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/stretchr/testify/require"
)

func TestApplyVideoUpscaleToClientVideoResponse_URLAndResolution(t *testing.T) {
	modelName := "upscale-client-resp"
	prev := ratio_setting.VideoPricingRules2JSONString()
	t.Cleanup(func() { _ = ratio_setting.UpdateVideoPricingRulesByJSONString(prev) })
	cfg := `{"` + modelName + `":{"video_upscale_per_second":[{"resolution":"720p","source_resolution":"480p","price":0.02}]}}`
	require.NoError(t, ratio_setting.UpdateVideoPricingRulesByJSONString(cfg))

	task := &model.Task{
		Properties: model.Properties{OriginModelName: modelName},
		PrivateData: model.TaskPrivateData{
			ResultURL: "https://bucket.cos.ap-guangzhou.myqcloud.com/video/output.mp4",
			VideoUpscale: &model.TaskVideoUpscaleInfo{
				SourceResolution: "480p",
				TargetResolution: "720p",
				Status:           model.TaskVideoUpscaleStatusSuccess,
			},
		},
	}
	in := []byte(`{
		"id":"t1",
		"status":"completed",
		"resolution":"480p",
		"url":"https://origin.example/raw.mp4",
		"result_url":"https://origin.example/raw.mp4",
		"content":{"url":"https://origin.example/raw.mp4","video_url":"https://origin.example/raw.mp4"},
		"output":{"url":"https://origin.example/raw.mp4","video_url":"https://origin.example/raw.mp4"}
	}`)
	out := ApplyVideoUpscaleToClientVideoResponse(in, task)
	require.Contains(t, string(out), `"resolution":"720p"`)
	require.Contains(t, string(out), "https://bucket.cos.ap-guangzhou.myqcloud.com/video/output.mp4")
	require.NotContains(t, string(out), "https://origin.example/raw.mp4")

	var root map[string]any
	require.NoError(t, common.Unmarshal(out, &root))
	_, hasURL := root["url"]
	_, hasResultURL := root["result_url"]
	require.False(t, hasURL, "top-level url must be removed")
	require.False(t, hasResultURL, "top-level result_url must be removed")

	content, _ := root["content"].(map[string]any)
	require.NotNil(t, content)
	require.Equal(t, "https://bucket.cos.ap-guangzhou.myqcloud.com/video/output.mp4", content["video_url"])
	_, hasContentURL := content["url"]
	require.False(t, hasContentURL, "content.url must be removed")

	output, _ := root["output"].(map[string]any)
	require.NotNil(t, output)
	require.Equal(t, "https://bucket.cos.ap-guangzhou.myqcloud.com/video/output.mp4", output["video_url"])
	_, hasOutputURL := output["url"]
	require.False(t, hasOutputURL, "output.url must be removed")
}

func TestVideoUpscaleBillingOnComplete_IncludesFeeWhenMarkupZero(t *testing.T) {
	modelName := "upscale-billing-markup-zero"
	prev := ratio_setting.VideoPricingRules2JSONString()
	t.Cleanup(func() { _ = ratio_setting.UpdateVideoPricingRulesByJSONString(prev) })
	cfg := `{"` + modelName + `":{"video_upscale_per_second":[{"resolution":"720p","source_resolution":"480p","price":0.0217}]}}`
	require.NoError(t, ratio_setting.UpdateVideoPricingRulesByJSONString(cfg))

	cost := 100.0
	task := &model.Task{
		UserId:    0,
		ChannelId: 0,
		Properties: model.Properties{OriginModelName: modelName},
		PrivateData: model.TaskPrivateData{
			BillingContext: &model.TaskBillingContext{
				GroupRatio:           1,
				EffectiveCostPercent: &cost,
			},
			VideoUpscale: &model.TaskVideoUpscaleInfo{
				SourceResolution: "480p",
				TargetResolution: "720p",
				Status:           model.TaskVideoUpscaleStatusSuccess,
				DurationSec:      5.2,
			},
		},
	}

	quota, other := videoUpscaleBillingOnComplete(task)
	require.Greater(t, quota, 0, "upscale fee must be charged when markup discount is 0%")
	require.NotNil(t, other)
	require.EqualValues(t, 6, other["video_upscale_seconds"])
	require.InDelta(t, 0.0217, other["video_upscale_price_per_second"], 1e-9)

	merged, upsOther := appendVideoUpscaleBilling(task, 1_000_000)
	require.Equal(t, 1_000_000+quota, merged)
	require.EqualValues(t, quota, upsOther["video_upscale_quota"])
}

func TestApplyVideoUpscaleToClientVideoResponse_CreatesNestedVideoURL(t *testing.T) {
	task := &model.Task{
		PrivateData: model.TaskPrivateData{
			ResultURL: "https://cdn.example/upscaled.mp4",
			VideoUpscale: &model.TaskVideoUpscaleInfo{
				Status: model.TaskVideoUpscaleStatusSuccess,
			},
		},
	}
	out := ApplyVideoUpscaleToClientVideoResponse([]byte(`{"id":"t1","status":"completed"}`), task)
	var root map[string]any
	require.NoError(t, json.Unmarshal(out, &root))
	content, _ := root["content"].(map[string]any)
	output, _ := root["output"].(map[string]any)
	require.Equal(t, "https://cdn.example/upscaled.mp4", content["video_url"])
	require.Equal(t, "https://cdn.example/upscaled.mp4", output["video_url"])
}
