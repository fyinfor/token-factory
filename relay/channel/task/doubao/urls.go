package doubao

import (
	"strings"
)

const (
	// FetchAPIVideoGenerations 旧查询路径：GET {base}/v1/video/generations/{task_id}
	FetchAPIVideoGenerations = "video_generations"
	// FetchAPIContentsGenerations 火山方舟 Contents API：GET {base}/api/v3/contents/generations/tasks/{task_id}
	FetchAPIContentsGenerations = "contents_generations"

	submitPath                   = "/api/v3/contents/generations/tasks"
	fetchPathContentsGenerations = "/api/v3/contents/generations/tasks/"
	fetchPathVideoGenerations    = "/v1/video/generations/"
)

// SubmitURL returns the upstream task creation endpoint for Seedance / Doubao video.
func SubmitURL(baseURL string) string {
	return joinBasePath(baseURL, submitPath)
}

// FetchURL 默认走火山方舟 Contents API 查询路径，保持现有渠道行为不变。
func FetchURL(baseURL, taskID string) string {
	return FetchURLByAPI(baseURL, taskID, FetchAPIContentsGenerations)
}

// FetchURLByAPI 按渠道配置拼接查询 URL。
//
//   - video_generations：旧接口 GET {base}/v1/video/generations/{task_id}
//   - 空 / contents_generations：新接口 GET {base}/api/v3/contents/generations/tasks/{task_id}
func FetchURLByAPI(baseURL, taskID, fetchAPI string) string {
	taskID = strings.TrimSpace(taskID)
	switch NormalizeFetchAPI(fetchAPI) {
	case FetchAPIVideoGenerations:
		return joinBasePath(baseURL, fetchPathVideoGenerations+taskID)
	default:
		return joinBasePath(baseURL, fetchPathContentsGenerations+taskID)
	}
}

// NormalizeFetchAPI 规范化渠道「任务查询接口」配置。
// 未配置时默认 contents_generations，与提交路径 /api/v3/contents/generations/tasks 对齐。
func NormalizeFetchAPI(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case FetchAPIVideoGenerations, "openai", "v1", "/v1/video/generations":
		return FetchAPIVideoGenerations
	default:
		return FetchAPIContentsGenerations
	}
}

func joinBasePath(baseURL, path string) string {
	return strings.TrimRight(strings.TrimSpace(baseURL), "/") + path
}

func isVideoURL(raw string) bool {
	u := strings.ToLower(strings.TrimSpace(raw))
	if u == "" {
		return false
	}
	if strings.HasPrefix(u, "data:video/") {
		return true
	}
	for _, ext := range []string{".mp4", ".mov", ".webm", ".avi", ".mkv", ".m4v"} {
		if strings.Contains(u, ext) {
			return true
		}
	}
	return false
}

// isAudioURL 判断 URL 是否为音频资源（用于 Seedance 2.0 参考音频封装）。
func isAudioURL(raw string) bool {
	u := strings.ToLower(strings.TrimSpace(raw))
	if u == "" {
		return false
	}
	if strings.HasPrefix(u, "data:audio/") {
		return true
	}
	for _, ext := range []string{".mp3", ".wav", ".m4a", ".aac", ".ogg", ".flac"} {
		if strings.Contains(u, ext) {
			return true
		}
	}
	return false
}
