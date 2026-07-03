package alivideo

import (
	"encoding/json"
	"testing"

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
