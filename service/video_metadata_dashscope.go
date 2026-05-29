package service

import (
	"regexp"
	"strconv"
	"strings"
)

var dashScopeResolutionFromURLRe = regexp.MustCompile(`(?i)(?:^|[^0-9])(480|720|1080)p(?:[^0-9]|$)`)

// extractVideoMetadataFromMap 从任务上游回包中解析成片元数据（腾讯云 VOD / 阿里云 DashScope 等）。
func extractVideoMetadataFromMap(payload map[string]any) (*VideoMetadata, bool) {
	if len(payload) == 0 {
		return nil, false
	}
	if meta, ok := extractTencentVODVideoMetadata(payload); ok {
		return meta, true
	}
	if meta, ok := extractDashScopeVideoMetadata(payload); ok {
		return meta, true
	}
	if meta, ok := extractVolcEngineVideoMetadata(payload); ok {
		return meta, true
	}
	return nil, false
}

func extractTencentVODVideoMetadata(payload map[string]any) (*VideoMetadata, bool) {
	response, _ := payload["Response"].(map[string]any)
	if response == nil {
		return nil, false
	}
	aigcVideoTask, _ := response["AigcVideoTask"].(map[string]any)
	if aigcVideoTask == nil {
		return nil, false
	}
	output, _ := aigcVideoTask["Output"].(map[string]any)
	if output == nil {
		return nil, false
	}
	fileInfos, _ := output["FileInfos"].([]any)
	if len(fileInfos) == 0 {
		return nil, false
	}
	firstFile, _ := fileInfos[0].(map[string]any)
	if firstFile == nil {
		return nil, false
	}
	metaMap, _ := firstFile["MetaData"].(map[string]any)
	if metaMap == nil {
		return nil, false
	}

	duration := metadataToFloat64(metaMap["Duration"])
	if duration <= 0 {
		duration = metadataToFloat64(metaMap["VideoDuration"])
	}
	width := metadataToInt(metaMap["Width"])
	height := metadataToInt(metaMap["Height"])
	audioDuration := metadataToFloat64(metaMap["AudioDuration"])

	hasAudio := audioDuration > 0
	if !hasAudio {
		if audioStreams, ok := metaMap["AudioStreamSet"].([]any); ok && len(audioStreams) > 0 {
			hasAudio = true
		}
	}
	if duration <= 0 || width <= 0 || height <= 0 {
		return nil, false
	}
	return &VideoMetadata{
		DurationSec: duration,
		Width:       width,
		Height:      height,
		HasAudio:    hasAudio,
	}, true
}

func extractDashScopeVideoMetadata(payload map[string]any) (*VideoMetadata, bool) {
	output := dashScopeOutput(payload)
	if output == nil {
		return nil, false
	}
	duration := dashScopeDurationFromOutput(output)
	videoURL := dashScopeVideoURL(output)
	width, height := dashScopeResolutionFromURL(videoURL)
	if duration <= 0 || width <= 0 || height <= 0 {
		return nil, false
	}
	return &VideoMetadata{
		DurationSec: duration,
		Width:       width,
		Height:      height,
		HasAudio:    false,
	}, true
}

func dashScopeOutput(payload map[string]any) map[string]any {
	if output, _ := payload["output"].(map[string]any); output != nil {
		return output
	}
	if output, _ := payload["Output"].(map[string]any); output != nil {
		return output
	}
	return nil
}

func dashScopeVideoURL(output map[string]any) string {
	for _, key := range []string{"video_url", "VideoURL", "videoUrl"} {
		if u, _ := output[key].(string); strings.TrimSpace(u) != "" {
			return strings.TrimSpace(u)
		}
	}
	return ""
}

func dashScopeDurationFromOutput(output map[string]any) float64 {
	// 仅使用成片时长字段。切勿用 end_time-submit_time：那是任务排队/生成耗时，不是视频秒数。
	for _, key := range []string{
		"duration", "Duration", "video_duration", "VideoDuration",
		"output_video_duration", "OutputVideoDuration",
	} {
		if d := metadataToFloat64(output[key]); d > 0 {
			return d
		}
	}
	if params, _ := output["parameters"].(map[string]any); params != nil {
		if d := metadataToFloat64(params["duration"]); d > 0 {
			return d
		}
	}
	if usage, _ := output["usage"].(map[string]any); usage != nil {
		for _, key := range []string{"video_duration", "duration", "output_video_duration"} {
			if d := metadataToFloat64(usage[key]); d > 0 {
				return d
			}
		}
	}
	return 0
}

func dashScopeResolutionFromURL(videoURL string) (int, int) {
	lower := strings.ToLower(videoURL)
	if m := dashScopeResolutionFromURLRe.FindStringSubmatch(lower); len(m) > 1 {
		switch m[1] {
		case "480":
			return 854, 480
		case "720":
			return 1280, 720
		case "1080":
			return 1920, 1080
		}
	}
	return 1280, 720
}

func metadataToFloat64(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case int:
		return float64(x)
	case int64:
		return float64(x)
	case int32:
		return float64(x)
	case uint:
		return float64(x)
	case uint64:
		return float64(x)
	case uint32:
		return float64(x)
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
		if err == nil {
			return f
		}
	}
	return 0
}

func metadataToInt(v any) int {
	switch x := v.(type) {
	case int:
		return x
	case int64:
		return int(x)
	case int32:
		return int(x)
	case uint:
		return int(x)
	case uint64:
		return int(x)
	case uint32:
		return int(x)
	case float64:
		return int(x)
	case float32:
		return int(x)
	case string:
		i, err := strconv.Atoi(strings.TrimSpace(x))
		if err == nil {
			return i
		}
	}
	return 0
}
