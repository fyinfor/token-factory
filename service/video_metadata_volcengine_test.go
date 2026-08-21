package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
)

func TestExtractVolcEngineVideoMetadata_SeedanceSuccess(t *testing.T) {
	raw := []byte(`{
		"id": "task_abc",
		"status": "succeeded",
		"model": "doubao-seedance-2-0-260128",
		"duration": 5,
		"resolution": "480p",
		"ratio": "16:9",
		"content": {
			"video_url": "https://res.example.com/doubao-seedance-2-0/out.mp4"
		},
		"usage": {
			"completion_tokens": 100858
		}
	}`)
	var payload map[string]any
	if err := common.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	meta, ok := extractVolcEngineVideoMetadata(payload)
	if !ok {
		t.Fatal("expected volcengine metadata")
	}
	if meta.DurationSec != 5 {
		t.Fatalf("duration=%v", meta.DurationSec)
	}
	if meta.Width != 854 || meta.Height != 480 {
		t.Fatalf("resolution=%dx%d", meta.Width, meta.Height)
	}
}

func TestExtractVolcEngineVideoMetadata_ResultSummary(t *testing.T) {
	raw := []byte(`{
		"id": "01a0231f-ace7-7c3f-a7ec-02f8f6dea411",
		"upstreamTaskId": "cgt-20260821150113-fh2fm",
		"status": "succeeded",
		"model": "doubao-seedance-2-5-260628",
		"resultSummary": {
			"content": {"video_url": "https://example.com/seedance.mp4"},
			"duration": "4",
			"resolution": "480p",
			"upstreamStatus": "succeeded",
			"usage": {"completion_tokens": 38830, "total_tokens": 38830}
		}
	}`)
	var payload map[string]any
	if err := common.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	meta, ok := extractVolcEngineVideoMetadata(payload)
	if !ok {
		t.Fatal("expected volcengine metadata from resultSummary")
	}
	if meta.DurationSec != 4 {
		t.Fatalf("duration=%v", meta.DurationSec)
	}
	if meta.Width != 854 || meta.Height != 480 {
		t.Fatalf("resolution=%dx%d", meta.Width, meta.Height)
	}
}

func TestExtractVolcEngineVideoMetadata_ResolutionBeforeSize(t *testing.T) {
	payload := map[string]any{
		"id":         "task_abc",
		"status":     "succeeded",
		"duration":   5,
		"resolution": "720p",
		"ratio":      "16:9",
		"size":       "1920x1080",
	}
	meta, ok := extractVolcEngineVideoMetadata(payload)
	if !ok {
		t.Fatal("expected volcengine metadata")
	}
	if meta.Width != 1280 || meta.Height != 720 {
		t.Fatalf("got %dx%d want 1280x720 from resolution", meta.Width, meta.Height)
	}
}

func TestExtractVideoMetadataFromMap_Seedance(t *testing.T) {
	payload := map[string]any{
		"id":         "task_x",
		"status":     "succeeded",
		"duration":   8,
		"resolution": "720p",
		"ratio":      "16:9",
		"content": map[string]any{
			"video_url": "https://example.com/720p.mp4",
		},
	}
	meta, ok := extractVideoMetadataFromMap(payload)
	if !ok {
		t.Fatal("expected metadata")
	}
	if meta.DurationSec != 8 {
		t.Fatalf("duration=%v", meta.DurationSec)
	}
	if meta.Width != 1280 || meta.Height != 720 {
		t.Fatalf("got %dx%d", meta.Width, meta.Height)
	}
}
