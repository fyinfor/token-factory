package hidreamimage

import (
	"testing"

	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

func TestResolveModelID(t *testing.T) {
	tests := []struct {
		model string
		want  string
	}{
		{"hidream-H4.5-image", "Image-qyoyq2bi"},
		{"hidream-Q3-pro-image", "Image-2zvskglc"},
		{"HiDream-O1-Image-1.5", "HiDream-O1-Image-1.5"},
		{"Image-aw47boi4", "Image-aw47boi4"},
		{"unknown-model", ""},
	}
	for _, tt := range tests {
		if got := resolveModelID(tt.model); got != tt.want {
			t.Errorf("resolveModelID(%q) = %q, want %q", tt.model, got, tt.want)
		}
	}
}

func TestOaiImage2HiDreamRequestHSeries(t *testing.T) {
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "hidream-H4.5-image",
		},
	}
	n := uint(2)
	req := dto.ImageRequest{
		Model:  "hidream-H4.5-image",
		Prompt: "a cat",
		Size:   "2048x2048",
		N:      &n,
	}
	body, err := oaiImage2HiDreamRequest(info, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body["model_id"] != "Image-qyoyq2bi" {
		t.Fatalf("model_id = %v", body["model_id"])
	}
	if body["prompt"] != "a cat" {
		t.Fatalf("prompt = %v", body["prompt"])
	}
	if body["size"] != "2048*2048" {
		t.Fatalf("size = %v", body["size"])
	}
	if body["n"] != 2 {
		t.Fatalf("n = %v", body["n"])
	}
}

func TestOaiImage2HiDreamRequestQSeries(t *testing.T) {
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "hidream-Q1-image",
		},
	}
	req := dto.ImageRequest{
		Model:  "hidream-Q1-image",
		Prompt: "a dog",
		Size:   "1024x1024",
	}
	body, err := oaiImage2HiDreamRequest(info, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body["aspect_ratio"] != "1:1" {
		t.Fatalf("aspect_ratio = %v", body["aspect_ratio"])
	}
}

func TestClassifyMaasSubTasks(t *testing.T) {
	done, failed, _ := classifyMaasSubTasks(&maasResultResponse{
		Result: struct {
			Status         int                 `json:"status"`
			SubTaskResults []maasSubTaskResult `json:"sub_task_results"`
		}{
			SubTaskResults: []maasSubTaskResult{
				{TaskStatus: 1, URL: "https://example.com/a.png"},
			},
		},
	})
	if !done || failed {
		t.Fatalf("expected success, got done=%v failed=%v", done, failed)
	}

	done, failed, _ = classifyMaasSubTasks(&maasResultResponse{
		Result: struct {
			Status         int                 `json:"status"`
			SubTaskResults []maasSubTaskResult `json:"sub_task_results"`
		}{
			SubTaskResults: []maasSubTaskResult{
				{TaskStatus: 3, ErrorMsg: "boom"},
			},
		},
	})
	if done || !failed {
		t.Fatalf("expected failure, got done=%v failed=%v", done, failed)
	}
}
