package dto

import (
	"encoding/json"
	"strconv"
	"strings"
)

const (
	VideoStatusUnknown    = "unknown"
	VideoStatusQueued     = "queued"
	VideoStatusInProgress = "in_progress"
	VideoStatusCompleted  = "completed"
	VideoStatusFailed     = "failed"
)

type OpenAIVideo struct {
	ID                 string            `json:"id"`
	TaskID             string            `json:"task_id,omitempty"` //兼容旧接口 待废弃
	Object             string            `json:"object"`
	Model              string            `json:"model"`
	Status             string            `json:"status"` // Should use VideoStatus constants: VideoStatusQueued, VideoStatusInProgress, VideoStatusCompleted, VideoStatusFailed
	Progress           int               `json:"progress"`
	CreatedAt          int64             `json:"created_at"`
	CompletedAt        int64             `json:"completed_at"`
	ExpiresAt          int64             `json:"expires_at,omitempty"`
	Seconds            string            `json:"seconds,omitempty"`
	Size               string            `json:"size,omitempty"`
	RemixedFromVideoID string            `json:"remixed_from_video_id,omitempty"`
	Error              *OpenAIVideoError `json:"error"`
	VideoURL           string            `json:"video_url"`
	Metadata           map[string]any    `json:"metadata,omitempty"`
}

func (m OpenAIVideo) MarshalJSON() ([]byte, error) {
	type openAIVideoAlias OpenAIVideo
	payload := struct {
		openAIVideoAlias
		CompletedAt *int64 `json:"completed_at"`
	}{
		openAIVideoAlias: openAIVideoAlias(m),
	}
	if m.CompletedAt > 0 {
		v := m.CompletedAt
		payload.CompletedAt = &v
	}
	return json.Marshal(payload)
}

func (m *OpenAIVideo) SetProgressStr(progress string) {
	progress = strings.TrimSuffix(progress, "%")
	m.Progress, _ = strconv.Atoi(progress)
}
func (m *OpenAIVideo) SetMetadata(k string, v any) {
	if k == "url" || k == "video_url" {
		if s, ok := v.(string); ok {
			m.VideoURL = s
		}
	}
	if m.Metadata == nil {
		m.Metadata = make(map[string]any)
	}
	m.Metadata[k] = v
}
func NewOpenAIVideo() *OpenAIVideo {
	return &OpenAIVideo{
		Object: "video",
		Status: VideoStatusQueued,
	}
}

type OpenAIVideoError struct {
	Message string `json:"message"`
	Code    string `json:"code"`
}
