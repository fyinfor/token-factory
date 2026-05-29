package doubao

import (
	"strings"
)

// SubmitURL returns the upstream task creation endpoint for Seedance / Doubao video.
func SubmitURL(baseURL string) string {
	return strings.TrimRight(strings.TrimSpace(baseURL), "/") + "/api/v3/contents/generations/tasks"
}

// FetchURL returns the upstream task status endpoint.
func FetchURL(baseURL, taskID string) string {
	return strings.TrimRight(strings.TrimSpace(baseURL), "/") + "/api/v3/contents/generations/tasks/" + strings.TrimSpace(taskID)
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
