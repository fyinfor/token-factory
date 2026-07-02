package dto

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
)

const (
	VideoStatusUnknown    = "unknown"
	VideoStatusQueued     = "queued"
	VideoStatusInProgress = "in_progress"
	VideoStatusCompleted  = "completed"
	VideoStatusFailed     = "failed"
	// ObjectVideoGeneration is the canonical object type for async video generation tasks.
	ObjectVideoGeneration = "video.generation"
)

// OpenAIVideoOutput holds primary generation artifacts (industrial / OpenAI-style task contract).
type OpenAIVideoOutput struct {
	VideoURL     string `json:"video_url,omitempty"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
	SrtURL       string `json:"srt_url,omitempty"`
}

type OpenAIVideoUsage struct {
	CompletionTokens int `json:"completion_tokens,omitempty"`
	TotalTokens      int `json:"total_tokens,omitempty"`
}

// OpenAIVideo is the unified response for POST submit and GET poll on /v1/videos/* (OpenAI-style video task API).
// Do not put the result URL in metadata; use Output.VideoURL only. Timestamps are RFC3339 in UTC by default;
// on /v1/videos* routes they are serialized as Unix int64 for downstream new-api compatibility.
type OpenAIVideo struct {
	ID                 string             `json:"id"`
	Object             string             `json:"object"`
	Model              string             `json:"model,omitempty"`
	Status             string             `json:"status"`
	Progress           int                `json:"progress"`
	CreatedAt          string             `json:"created_at,omitempty"`
	CompletedAt        string             `json:"completed_at,omitempty"`
	Seconds            string             `json:"seconds,omitempty"`
	Size               string             `json:"size,omitempty"`
	RemixedFromVideoID string             `json:"remixed_from_video_id,omitempty"`
	Error              *OpenAIVideoError  `json:"error"`
	Output             *OpenAIVideoOutput `json:"output,omitempty"`
	Metadata           map[string]any     `json:"metadata,omitempty"`
	Usage              *OpenAIVideoUsage  `json:"usage,omitempty"`
}

// FormatTimeUnixRFC3339 converts a Unix second timestamp to RFC3339 (UTC), or returns "" if unset.
func FormatTimeUnixRFC3339(sec int64) string {
	if sec <= 0 {
		return ""
	}
	return time.Unix(sec, 0).UTC().Format(time.RFC3339)
}

// IsOpenAIVideosCompatPath reports whether the request path is the OpenAI /v1/videos surface.
// Excludes /v1/video/generations and /v1/videos/generations*.
func IsOpenAIVideosCompatPath(path string) bool {
	path = strings.TrimSpace(path)
	if path == "/v1/videos" {
		return true
	}
	if !strings.HasPrefix(path, "/v1/videos/") {
		return false
	}
	rest := strings.TrimPrefix(path, "/v1/videos/")
	return rest != "generations" && !strings.HasPrefix(rest, "generations/")
}

// ConvertTimestampsToInt64 converts RFC3339 timestamp strings to int64 Unix timestamps
// for new-api /v1/videos compatibility.
func ConvertTimestampsToInt64(video *OpenAIVideo) ([]byte, error) {
	var videoMap map[string]any
	data, err := common.Marshal(video)
	if err != nil {
		return nil, err
	}
	if err := common.Unmarshal(data, &videoMap); err != nil {
		return nil, err
	}

	if createdAtStr, ok := videoMap["created_at"].(string); ok && createdAtStr != "" {
		if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			videoMap["created_at"] = t.Unix()
		}
	}

	if completedAtStr, ok := videoMap["completed_at"].(string); ok && completedAtStr != "" {
		if t, err := time.Parse(time.RFC3339, completedAtStr); err == nil {
			videoMap["completed_at"] = t.Unix()
		}
	}

	if expiresAtStr, ok := videoMap["expires_at"].(string); ok && expiresAtStr != "" {
		if t, err := time.Parse(time.RFC3339, expiresAtStr); err == nil {
			videoMap["expires_at"] = t.Unix()
		}
	}

	return common.Marshal(videoMap)
}

// AdaptOpenAIVideoJSONForPath applies /v1/videos timestamp conversion when needed.
func AdaptOpenAIVideoJSONForPath(path string, data []byte) ([]byte, error) {
	if !IsOpenAIVideosCompatPath(path) {
		return data, nil
	}
	var video OpenAIVideo
	if err := common.Unmarshal(data, &video); err != nil {
		return data, err
	}
	return ConvertTimestampsToInt64(&video)
}

func (m *OpenAIVideo) SetProgressStr(progress string) {
	progress = strings.TrimSuffix(progress, "%")
	m.Progress, _ = strconv.Atoi(progress)
}

// SetOutputVideoURL sets the primary video URL on Output (not metadata).
func (m *OpenAIVideo) SetOutputVideoURL(url string) {
	url = strings.TrimSpace(url)
	if url == "" {
		return
	}
	if m.Output == nil {
		m.Output = &OpenAIVideoOutput{}
	}
	m.Output.VideoURL = url
}

// SetMetadata sets supplemental key/values only. URL-like keys are routed to Output, not metadata.
func (m *OpenAIVideo) SetMetadata(k string, v any) {
	if k == "url" || k == "video_url" {
		if s, ok := v.(string); ok {
			m.SetOutputVideoURL(s)
		}
		return
	}
	if m.Metadata == nil {
		m.Metadata = make(map[string]any)
	}
	m.Metadata[k] = v
}

func NewOpenAIVideo() *OpenAIVideo {
	return &OpenAIVideo{
		Object: ObjectVideoGeneration,
		Status: VideoStatusQueued,
	}
}

type OpenAIVideoError struct {
	Message string `json:"message"`
	Code    string `json:"code"`
}

// MarshalJSON encodes a null error explicitly as JSON null.
func (m OpenAIVideo) MarshalJSON() ([]byte, error) {
	type alias struct {
		ID                 string             `json:"id"`
		Object             string             `json:"object"`
		Model              string             `json:"model,omitempty"`
		Status             string             `json:"status"`
		Progress           int                `json:"progress"`
		CreatedAt          string             `json:"created_at,omitempty"`
		CompletedAt        string             `json:"completed_at,omitempty"`
		Seconds            string             `json:"seconds,omitempty"`
		Size               string             `json:"size,omitempty"`
		RemixedFromVideoID string             `json:"remixed_from_video_id,omitempty"`
		Error              *OpenAIVideoError  `json:"error"`
		Output             *OpenAIVideoOutput `json:"output,omitempty"`
		Metadata           map[string]any     `json:"metadata,omitempty"`
		Usage              *OpenAIVideoUsage  `json:"usage,omitempty"`
	}
	a := alias{
		ID:                 m.ID,
		Object:             m.Object,
		Model:              m.Model,
		Status:             m.Status,
		Progress:           m.Progress,
		CreatedAt:          m.CreatedAt,
		CompletedAt:        m.CompletedAt,
		Seconds:            m.Seconds,
		Size:               m.Size,
		RemixedFromVideoID: m.RemixedFromVideoID,
		Error:              m.Error,
		Output:             m.Output,
		Metadata:           m.Metadata,
		Usage:              m.Usage,
	}
	return json.Marshal(a)
}
