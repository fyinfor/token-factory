package minimaxh3

import (
	"net/http"
	"net/url"
	"strings"
)

// NormalizeBaseURL 规范化用户填写的 V2 基础地址。
// 只接受 host + /v2，自动去掉误填的接口后缀与尾部斜杠。
func NormalizeBaseURL(raw string) string {
	u := strings.TrimSpace(raw)
	u = strings.TrimRight(u, "/")
	if u == "" {
		return DefaultBaseURL
	}
	lower := strings.ToLower(u)
	for _, suffix := range []string{
		"/query/video_generation",
		"/video_generation",
	} {
		if strings.HasSuffix(lower, suffix) {
			u = strings.TrimRight(u[:len(u)-len(suffix)], "/")
			lower = strings.ToLower(u)
		}
	}
	if u == "" {
		return DefaultBaseURL
	}
	switch strings.ToLower(u) {
	case "https://api.minimaxi.com", "http://api.minimaxi.com":
		return DefaultBaseURL
	}
	return u
}

// SubmitURL 创建视频任务完整路径：{baseUrl}/video_generation
func SubmitURL(baseURL string) string {
	return NormalizeBaseURL(baseURL) + VideoGenerationPath
}

// QueryURL 查询视频任务完整路径：{baseUrl}/query/video_generation/{task_id}
func QueryURL(baseURL, taskID string) string {
	id := strings.TrimSpace(taskID)
	return NormalizeBaseURL(baseURL) + QueryTaskPathPrefix + url.PathEscape(id)
}

// ApplyAuthHeaders 统一鉴权：Authorization: Bearer {apiKey}、Content-Type: application/json。
func ApplyAuthHeaders(h http.Header, apiKey string) {
	if h == nil {
		return
	}
	h.Set("Authorization", "Bearer "+strings.TrimSpace(apiKey))
	h.Set("Content-Type", "application/json")
	h.Set("Accept", "application/json")
}
