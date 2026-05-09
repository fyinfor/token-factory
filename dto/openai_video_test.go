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
