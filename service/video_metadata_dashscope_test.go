package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
)

func TestExtractDashScopeVideoMetadata_ExplicitDuration(t *testing.T) {
	raw := []byte(`{
		"output": {
			"task_status": "SUCCEEDED",
			"duration": 5,
			"video_url": "https://example.com/metadata_video_720p_sample.mp4"
		}
	}`)
	var payload map[string]any
	if err := common.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	meta, ok := extractDashScopeVideoMetadata(payload)
	if !ok {
		t.Fatal("expected dashscope metadata")
	}
	if meta.Width != 1280 || meta.Height != 720 {
		t.Fatalf("resolution = %dx%d", meta.Width, meta.Height)
	}
	if meta.DurationSec != 5 {
		t.Fatalf("duration = %v, want 5", meta.DurationSec)
	}
}

func TestExtractDashScopeVideoMetadata_TopLevelUsage(t *testing.T) {
	raw := []byte(`{
		"request_id": "99243b47-ec5f-9413-9993-xxxxxx",
		"output": {
			"task_id": "4673458e-28be-4a05-bf2a-xxxxxx",
			"task_status": "SUCCEEDED",
			"video_url": "https://dashscope-result.oss-cn-beijing.aliyuncs.com/xxx.mp4?Expires=xxx"
		},
		"usage": {
			"duration": 5,
			"input_video_duration": 0,
			"output_video_duration": 5,
			"video_count": 1,
			"SR": 720,
			"ratio": "16:9"
		}
	}`)
	var payload map[string]any
	if err := common.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	meta, ok := extractDashScopeVideoMetadata(payload)
	if !ok {
		t.Fatal("expected dashscope metadata from top-level usage")
	}
	if meta.DurationSec != 5 {
		t.Fatalf("duration = %v, want 5", meta.DurationSec)
	}
	if meta.Width != 1280 || meta.Height != 720 {
		t.Fatalf("resolution = %dx%d, want 1280x720", meta.Width, meta.Height)
	}
}

func TestExtractDashScopeUsageSpec(t *testing.T) {
	raw := []byte(`{
		"usage": {
			"output_video_duration": 5.2,
			"SR": 720,
			"ratio": "16:9"
		}
	}`)
	res, dur, ratio := extractDashScopeUsageSpec(raw)
	if res != "720p" {
		t.Fatalf("resolution = %q, want 720p", res)
	}
	if dur != 6 {
		t.Fatalf("duration = %d, want 6 (ceil)", dur)
	}
	if ratio != "16:9" {
		t.Fatalf("ratio = %q", ratio)
	}
}

func TestDashScopeDurationFromOutput_IgnoresTaskWallClock(t *testing.T) {
	output := map[string]any{
		"submit_time": "2026-05-26 20:18:44.938",
		"end_time":    "2026-05-26 20:20:09.938",
		"video_url":   "https://example.com/720p.mp4",
	}
	if d := dashScopeDurationFromOutput(output); d != 0 {
		t.Fatalf("expected 0 duration from task timestamps, got %v", d)
	}
}

func TestExtractVideoMetadataFromMap_PrefersTencent(t *testing.T) {
	payload := map[string]any{
		"Response": map[string]any{
			"AigcVideoTask": map[string]any{
				"Output": map[string]any{
					"FileInfos": []any{
						map[string]any{
							"MetaData": map[string]any{
								"Duration": 8.5,
								"Width":    1920,
								"Height":   1080,
							},
						},
					},
				},
			},
		},
		"output": map[string]any{
			"video_url":   "https://example.com/480p.mp4",
			"submit_time": "2026-05-26 20:18:44.938",
			"end_time":    "2026-05-26 20:22:58.704",
		},
	}
	meta, ok := extractVideoMetadataFromMap(payload)
	if !ok {
		t.Fatal("expected metadata")
	}
	if meta.Width != 1920 || meta.Height != 1080 {
		t.Fatalf("expected tencent metadata, got %dx%d", meta.Width, meta.Height)
	}
	if meta.DurationSec != 8.5 {
		t.Fatalf("duration = %v", meta.DurationSec)
	}
}

func TestExtractTencentVODVideoMetadata_InputOutputConfigFallback(t *testing.T) {
	payload := map[string]any{
		"Response": map[string]any{
			"AigcVideoTask": map[string]any{
				"Input": map[string]any{
					"OutputConfig": map[string]any{
						"Duration":    15,
						"Resolution":  "720P",
						"AspectRatio": "16:9",
					},
				},
				"Output": map[string]any{
					"FileInfos": []any{},
				},
			},
		},
	}
	meta, ok := extractTencentVODVideoMetadata(payload)
	if !ok {
		t.Fatal("expected OutputConfig fallback metadata")
	}
	if meta.DurationSec != 15 {
		t.Fatalf("duration = %v", meta.DurationSec)
	}
	if meta.Width != 1280 || meta.Height != 720 {
		t.Fatalf("got %dx%d", meta.Width, meta.Height)
	}
}
