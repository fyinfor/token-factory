package dto

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestOpenAIVideoJSONShape(t *testing.T) {
	v := NewOpenAIVideo()
	v.ID = "task_test"
	v.Model = "Seedance2.0"
	v.Status = VideoStatusInProgress
	v.Progress = 30
	v.CreatedAt = FormatTimeUnixRFC3339(1778292296)

	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)

	if strings.Contains(s, `"task_id"`) {
		t.Fatalf("unexpected task_id in JSON: %s", s)
	}
	if strings.Contains(s, `"video_url"`) {
		t.Fatalf("unexpected top-level video_url in JSON: %s", s)
	}
	if !strings.Contains(s, `"object":"video.generation"`) {
		t.Fatalf("expected object video.generation, got: %s", s)
	}
	if !strings.Contains(s, `"created_at":"`) {
		t.Fatalf("expected RFC3339 created_at string, got: %s", s)
	}

	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if _, ok := m["completed_at"]; ok {
		t.Fatalf("completed_at should be omitted when empty, got: %s", s)
	}
}

func TestIsOpenAIVideosCompatPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/v1/videos", true},
		{"/v1/videos/task_abc", true},
		{"/v1/videos/vid/remix", true},
		{"/v1/video/generations", false},
		{"/v1/video/generations/task_abc", false},
		{"/v1/videos/generations", false},
		{"/v1/videos/generations/task_abc", false},
		{"/api/playground/videos/task_abc", false},
	}
	for _, tt := range tests {
		if got := IsOpenAIVideosCompatPath(tt.path); got != tt.want {
			t.Fatalf("IsOpenAIVideosCompatPath(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestAdaptOpenAIVideoJSONForPath(t *testing.T) {
	v := NewOpenAIVideo()
	v.ID = "task_test"
	v.CreatedAt = FormatTimeUnixRFC3339(1778292296)

	defaultJSON, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}

	unchanged, err := AdaptOpenAIVideoJSONForPath("/v1/video/generations/task_abc", defaultJSON)
	if err != nil {
		t.Fatal(err)
	}
	if string(unchanged) != string(defaultJSON) {
		t.Fatalf("expected unchanged JSON for non-compat path")
	}

	converted, err := AdaptOpenAIVideoJSONForPath("/v1/videos/task_abc", defaultJSON)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(converted), `"created_at":1778292296`) {
		t.Fatalf("expected int64 created_at for /v1/videos path, got: %s", converted)
	}

	withExtra := []byte(`{"id":"task_test","created_at":"2026-07-03T03:16:23Z","ratio":"16:9","resolution":"480p","duration":5}`)
	convertedExtra, err := AdaptOpenAIVideoJSONForPath("/v1/videos/task_abc", withExtra)
	if err != nil {
		t.Fatal(err)
	}
	s := string(convertedExtra)
	if !strings.Contains(s, `"ratio":"16:9"`) || !strings.Contains(s, `"resolution":"480p"`) {
		t.Fatalf("expected passthrough fields preserved on /v1/videos path, got: %s", s)
	}
}
