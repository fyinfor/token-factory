package alivideo

import (
	"encoding/json"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

func TestSubmitURL(t *testing.T) {
	got := SubmitURL("https://dashscope.aliyuncs.com/api")
	want := "https://dashscope.aliyuncs.com/api/v1/services/aigc/video-generation/video-synthesis"
	if got != want {
		t.Fatalf("SubmitURL() = %q, want %q", got, want)
	}
	got2 := SubmitURL("https://dashscope.aliyuncs.com")
	want2 := "https://dashscope.aliyuncs.com/api/v1/services/aigc/video-generation/video-synthesis"
	if got2 != want2 {
		t.Fatalf("SubmitURL() = %q, want %q", got2, want2)
	}
}

func TestParseTaskResult_UsageBillingFields(t *testing.T) {
	body := []byte(`{
		"request_id": "req-1",
		"output": {
			"task_id": "task-1",
			"task_status": "SUCCEEDED",
			"video_url": "https://example.com/out.mp4"
		},
		"usage": {
			"duration": 23.94,
			"input_video_duration": 11.97,
			"output_video_duration": 11.97,
			"video_count": 1,
			"SR": 720,
			"ratio": "16:9"
		}
	}`)
	a := &TaskAdaptor{}
	info, err := a.ParseTaskResult(body)
	if err != nil {
		t.Fatal(err)
	}
	if info.Duration != 24 {
		t.Fatalf("Duration = %d, want 24 from usage.duration", info.Duration)
	}
	if info.Resolution != "720p" {
		t.Fatalf("Resolution = %q, want 720p", info.Resolution)
	}
	if info.Ratio != "16:9" {
		t.Fatalf("Ratio = %q, want 16:9", info.Ratio)
	}
	if info.Url != "https://example.com/out.mp4" {
		t.Fatalf("Url = %q", info.Url)
	}
}

func TestConvertToOpenAIVideo_PreservesAliVideoUsage(t *testing.T) {
	task := &model.Task{
		TaskID:     "task-1",
		Status:     model.TaskStatusSuccess,
		Data:       []byte(`{"output":{"video_url":"https://example.com/out.mp4"},"usage":{"SR":720,"duration":23.94,"input_video_duration":11.97,"output_video_duration":11.97,"video_count":1}}`),
		Properties: model.Properties{OriginModelName: "happyhorse-1.0-t2v"},
	}
	body, err := (&TaskAdaptor{}).ConvertToOpenAIVideo(task)
	if err != nil {
		t.Fatal(err)
	}
	var response dto.OpenAIVideo
	if err := common.Unmarshal(body, &response); err != nil {
		t.Fatal(err)
	}
	if response.Usage == nil {
		t.Fatal("expected usage")
	}
	if response.Usage.Duration == nil || *response.Usage.Duration != 23.94 || response.Usage.InputVideoDuration == nil || *response.Usage.InputVideoDuration != 11.97 || response.Usage.OutputVideoDuration == nil || *response.Usage.OutputVideoDuration != 11.97 || response.Usage.VideoCount == nil || *response.Usage.VideoCount != 1 || response.Usage.SR == nil || *response.Usage.SR != 720 {
		t.Fatalf("usage = %+v", response.Usage)
	}
}

func TestConvertToAliRequest_TextToVideo(t *testing.T) {
	a := &TaskAdaptor{}
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{}}
	req := relaycommon.TaskSubmitReq{
		Model:  "happyhorse-1.0-t2v",
		Prompt: "test prompt",
		Size:   "720P",
	}
	aliReq, err := a.convertToAliRequest(info, req)
	if err != nil {
		t.Fatal(err)
	}
	if aliReq.Model != "happyhorse-1.0-t2v" {
		t.Fatalf("model = %q", aliReq.Model)
	}
	if aliReq.Input.Prompt != "test prompt" {
		t.Fatalf("prompt = %q", aliReq.Input.Prompt)
	}
	if aliReq.Parameters.Resolution != "720P" {
		t.Fatalf("resolution = %q", aliReq.Parameters.Resolution)
	}
	if aliReq.Parameters.Watermark == nil || *aliReq.Parameters.Watermark != false {
		t.Fatalf("watermark = %v, want false", aliReq.Parameters.Watermark)
	}
}

func TestEnrichNativeAliVideoBody_DefaultWatermark(t *testing.T) {
	body := []byte(`{"model":"happyhorse-1.0-t2v","input":{"prompt":"x"},"parameters":{"duration":5}}`)
	out, err := enrichNativeAliVideoBody(body)
	if err != nil {
		t.Fatal(err)
	}
	var aliReq AliVideoRequest
	if err := json.Unmarshal(out, &aliReq); err != nil {
		t.Fatal(err)
	}
	if aliReq.Parameters == nil || aliReq.Parameters.Watermark == nil || *aliReq.Parameters.Watermark != false {
		t.Fatalf("watermark = %v, want false", aliReq.Parameters)
	}
}

func TestBuildMediaFromTaskReq_VideoURLsInMetadata(t *testing.T) {
	req := relaycommon.TaskSubmitReq{
		Metadata: map[string]interface{}{
			"video_urls": []interface{}{"https://example.com/src.mp4"},
		},
	}
	media := buildMediaFromTaskReq("happyhorse-1.0-v2v", req)
	if len(media) != 1 || media[0].Type != "video" {
		t.Fatalf("media = %+v", media)
	}
}

func TestVideoEdit_DedupeSameVideoURL(t *testing.T) {
	videoURL := "http://example.com/aigcVideoGenFile.mp4"
	req := relaycommon.TaskSubmitReq{
		Model:          "happyhorse-1.0-video-edit",
		Prompt:         "faster run",
		InputReference: videoURL,
		Metadata: map[string]interface{}{
			"video_urls": []interface{}{videoURL},
		},
	}
	a := &TaskAdaptor{}
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{}}
	aliReq, err := a.convertToAliRequest(info, req)
	if err != nil {
		t.Fatal(err)
	}
	videoCount := 0
	for _, m := range aliReq.Input.Media {
		if m.Type == "video" {
			videoCount++
		}
	}
	if videoCount != 1 {
		t.Fatalf("video media count = %d, want 1: %+v", videoCount, aliReq.Input.Media)
	}
}

func TestNormalizeAliVideoMedia_R2VWithVideoURL(t *testing.T) {
	req := relaycommon.TaskSubmitReq{
		Model:          "happyhorse-1.0-r2v",
		InputReference: "https://example.com/ref.mp4",
		Metadata: map[string]interface{}{
			"video_urls": []interface{}{"https://example.com/ref.mp4"},
			"input": map[string]interface{}{
				"prompt": "test",
				"media": []interface{}{
					map[string]interface{}{"type": "video", "url": "https://example.com/ref.mp4"},
				},
			},
		},
	}
	a := &TaskAdaptor{}
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{}}
	aliReq, err := a.convertToAliRequest(info, req)
	if err != nil {
		t.Fatal(err)
	}
	if len(aliReq.Input.Media) != 1 || aliReq.Input.Media[0].Type != "reference_image" {
		t.Fatalf("media = %+v", aliReq.Input.Media)
	}
}

func TestEnrichNativeAliVideoBody_ImgURL(t *testing.T) {
	body := []byte(`{"model":"happyhorse-1.0-i2v","input":{"prompt":"x","img_url":"https://example.com/a.png"},"parameters":{"duration":5}}`)
	out, err := enrichNativeAliVideoBody(body)
	if err != nil {
		t.Fatal(err)
	}
	var aliReq AliVideoRequest
	if err := json.Unmarshal(out, &aliReq); err != nil {
		t.Fatal(err)
	}
	if len(aliReq.Input.Media) != 1 || aliReq.Input.Media[0].Type != "first_frame" {
		t.Fatalf("media = %+v", aliReq.Input.Media)
	}
}

func TestBuildMediaFromTaskReq_TwoFrames(t *testing.T) {
	req := relaycommon.TaskSubmitReq{
		Images: []string{
			"https://example.com/first.png",
			"https://example.com/last.png",
		},
	}
	media := buildMediaFromTaskReq("happyhorse-1.0-i2v", req)
	if len(media) != 2 {
		t.Fatalf("media len = %d, want 2: %+v", len(media), media)
	}
	if media[0].Type != "first_frame" || media[1].Type != "last_frame" {
		t.Fatalf("media = %+v", media)
	}
}

func TestConvertToAliRequest_ImageToVideo(t *testing.T) {
	a := &TaskAdaptor{}
	info := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{}}
	req := relaycommon.TaskSubmitReq{
		Model:  "happyhorse-1.0-i2v",
		Prompt: "cat running",
		Images: []string{"https://example.com/frame.png"},
	}
	aliReq, err := a.convertToAliRequest(info, req)
	if err != nil {
		t.Fatal(err)
	}
	if len(aliReq.Input.Media) != 1 || aliReq.Input.Media[0].Type != "first_frame" {
		t.Fatalf("media = %+v", aliReq.Input.Media)
	}
}
