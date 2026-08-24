package service

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
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
	if output != nil {
		fileInfos, _ := output["FileInfos"].([]any)
		if len(fileInfos) > 0 {
			firstFile, _ := fileInfos[0].(map[string]any)
			if firstFile != nil {
				metaMap, _ := firstFile["MetaData"].(map[string]any)
				if metaMap != nil {
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
					if duration > 0 && width > 0 && height > 0 {
						return &VideoMetadata{
							DurationSec: duration,
							Width:       width,
							Height:      height,
							HasAudio:    hasAudio,
						}, true
					}
				}
			}
		}
	}

	// Temporary 存储常无 MetaData：回退 AigcVideoTask.Input.OutputConfig（计费核心三字段）
	input, _ := aigcVideoTask["Input"].(map[string]any)
	if input == nil {
		return nil, false
	}
	oc, _ := input["OutputConfig"].(map[string]any)
	if oc == nil {
		return nil, false
	}
	duration := metadataToFloat64(oc["Duration"])
	resolution, _ := oc["Resolution"].(string)
	resolution = strings.TrimSpace(resolution)
	aspectRatio, _ := oc["AspectRatio"].(string)
	aspectRatio = strings.TrimSpace(aspectRatio)
	if duration <= 0 {
		return nil, false
	}
	width, height, ok := common.ParseVideoResolutionAndRatio(resolution, aspectRatio)
	if !ok {
		return nil, false
	}
	return &VideoMetadata{
		DurationSec: duration,
		Width:       width,
		Height:      height,
		HasAudio:    false,
	}, true
}

func extractDashScopeVideoMetadata(payload map[string]any) (*VideoMetadata, bool) {
	output := dashScopeOutput(payload)
	if output == nil && dashScopeUsage(payload) == nil {
		return nil, false
	}
	duration := dashScopeDurationFromOutput(output)
	if duration <= 0 {
		duration = dashScopeDurationFromUsage(payload)
	}
	ratio := dashScopeRatioFromUsage(payload)
	resolution := dashScopeResolutionLabelFromUsage(payload)
	width, height := 0, 0
	if resolution != "" {
		if w, h, ok := common.ParseVideoResolutionAndRatio(resolution, ratio); ok {
			width, height = w, h
		}
	}
	if width <= 0 || height <= 0 {
		videoURL := dashScopeVideoURL(output)
		if videoURL != "" {
			width, height = dashScopeResolutionFromURL(videoURL)
		}
	}
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
	if payload == nil {
		return nil
	}
	if output, _ := payload["output"].(map[string]any); output != nil {
		return output
	}
	if output, _ := payload["Output"].(map[string]any); output != nil {
		return output
	}
	return nil
}

// dashScopeUsage 读取 DashScope 查询回包顶层 usage（计费权威来源）。
func dashScopeUsage(payload map[string]any) map[string]any {
	if payload == nil {
		return nil
	}
	if usage, _ := payload["usage"].(map[string]any); usage != nil {
		return usage
	}
	if usage, _ := payload["Usage"].(map[string]any); usage != nil {
		return usage
	}
	// 兼容误嵌套在 output 内的 usage
	if output := dashScopeOutput(payload); output != nil {
		if usage, _ := output["usage"].(map[string]any); usage != nil {
			return usage
		}
	}
	return nil
}

func dashScopeVideoURL(output map[string]any) string {
	if output == nil {
		return ""
	}
	for _, key := range []string{"video_url", "VideoURL", "videoUrl"} {
		if u, _ := output[key].(string); strings.TrimSpace(u) != "" {
			return strings.TrimSpace(u)
		}
	}
	return ""
}

func dashScopeDurationFromOutput(output map[string]any) float64 {
	if output == nil {
		return 0
	}
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
	return 0
}

func dashScopeDurationFromUsage(payload map[string]any) float64 {
	usage := dashScopeUsage(payload)
	if usage == nil {
		return 0
	}
	// duration 是 DashScope usage 的计费时长，不能用 output_video_duration 替代。
	for _, key := range []string{
		"duration", "Duration",
		"video_duration", "VideoDuration",
		"output_video_duration", "OutputVideoDuration",
	} {
		if d := metadataToFloat64(usage[key]); d > 0 {
			return d
		}
	}
	return 0
}

func dashScopeRatioFromUsage(payload map[string]any) string {
	usage := dashScopeUsage(payload)
	if usage == nil {
		return ""
	}
	for _, key := range []string{"ratio", "Ratio", "aspect_ratio", "AspectRatio"} {
		if r, _ := usage[key].(string); strings.TrimSpace(r) != "" {
			return strings.TrimSpace(r)
		}
	}
	return ""
}

func dashScopeResolutionLabelFromUsage(payload map[string]any) string {
	usage := dashScopeUsage(payload)
	if usage == nil {
		return ""
	}
	for _, key := range []string{"SR", "sr", "resolution", "Resolution"} {
		if v, ok := usage[key]; ok {
			switch x := v.(type) {
			case string:
				if label := common.FormatVideoResolutionLabel(x); label != "" {
					return label
				}
			default:
				if n := metadataToInt(v); n > 0 {
					if label := common.FormatVideoResolutionLabel(strconv.Itoa(n)); label != "" {
						return label
					}
				}
			}
		}
	}
	return ""
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
