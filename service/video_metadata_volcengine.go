package service

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

// extractVolcEngineVideoMetadata parses VolcEngine / Seedance contents-generations task payloads
// (top-level id, status, duration, resolution, content.video_url, or resultSummary 嵌套形态).
func extractVolcEngineVideoMetadata(payload map[string]any) (*VideoMetadata, bool) {
	if len(payload) == 0 {
		return nil, false
	}
	dto.LiftVideoPollResultSummary(payload)
	// DashScope / Tencent shapes are handled elsewhere.
	if dashScopeOutput(payload) != nil {
		return nil, false
	}
	if _, ok := payload["Response"].(map[string]any); ok {
		return nil, false
	}
	status, _ := payload["status"].(string)
	if strings.TrimSpace(status) == "" {
		return nil, false
	}
	if _, ok := payload["id"].(string); !ok {
		return nil, false
	}

	duration := metadataToFloat64(payload["duration"])
	if duration <= 0 {
		return nil, false
	}

	resolution, _ := payload["resolution"].(string)
	ratio, _ := payload["ratio"].(string)
	var width, height int
	if strings.TrimSpace(resolution) != "" {
		if w, h, ok := common.ParseVideoResolutionAndRatio(resolution, ratio); ok {
			width, height = w, h
		}
	}
	if width <= 0 || height <= 0 {
		width = metadataToInt(payload["width"])
		height = metadataToInt(payload["height"])
	}
	if width <= 0 || height <= 0 {
		if size, _ := payload["size"].(string); strings.TrimSpace(size) != "" {
			if w, h, ok := common.ParseVideoResolutionAndRatio(size, ratio); ok {
				width, height = w, h
			}
		}
	}
	if width <= 0 || height <= 0 {
		if u := volcEngineVideoURL(payload); u != "" {
			width, height = dashScopeResolutionFromURL(u)
		}
	}
	if width <= 0 || height <= 0 {
		width, height = 1280, 720
	}

	hasAudio := false
	for _, key := range []string{"generate_audio", "has_audio"} {
		if v, ok := payload[key]; ok {
			switch x := v.(type) {
			case bool:
				hasAudio = x
			case string:
				hasAudio = strings.EqualFold(strings.TrimSpace(x), "true")
			}
			if hasAudio {
				break
			}
		}
	}

	return &VideoMetadata{
		DurationSec: duration,
		Width:       width,
		Height:      height,
		HasAudio:    hasAudio,
	}, true
}

func volcEngineVideoURL(payload map[string]any) string {
	content, _ := payload["content"].(map[string]any)
	if content == nil {
		return ""
	}
	if u, _ := content["video_url"].(string); strings.TrimSpace(u) != "" {
		return strings.TrimSpace(u)
	}
	return ""
}
