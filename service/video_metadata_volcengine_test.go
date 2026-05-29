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
