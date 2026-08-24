package doubao

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
)

const fetchBodyKeyAPI = "seedance_fetch_api"

// QueryTask 统一查询火山方舟 / Seedance 视频任务结果。
//
// 双接口由 fetchAPI 配置选择，查询逻辑集中在本函数，避免 FetchTask / 轮询各写一套：
//   - video_generations：旧接口 GET {base}/v1/video/generations/{task_id}（逻辑保持不变）
//   - contents_generations：新接口 GET {base}/api/v3/contents/generations/tasks/{task_id}
//
// 使用原生 net/http GET + Bearer，不引入方舟 SDK。非 2xx 响应原样返回给调用方解析。
func QueryTask(baseURL, key, taskID, fetchAPI, proxy string) (*http.Response, error) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, fmt.Errorf("invalid task_id")
	}
	uri := FetchURLByAPI(baseURL, taskID, fetchAPI)
	if strings.TrimSpace(uri) == "" {
		return nil, fmt.Errorf("invalid seedance fetch url")
	}

	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return nil, fmt.Errorf("new seedance fetch request failed: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	if k := strings.TrimSpace(key); k != "" {
		req.Header.Set("Authorization", "Bearer "+k)
	}

	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("seedance fetch task failed: %w", err)
	}
	if resp == nil {
		return nil, fmt.Errorf("seedance fetch task returned empty response")
	}
	return resp, nil
}

func fetchAPIFromBody(body map[string]any) string {
	if body == nil {
		return ""
	}
	s, _ := body[fetchBodyKeyAPI].(string)
	return s
}

// AdaptFetchResponseJSON 将两种查询回包归一为火山方舟 Contents API 任务对象。
// ParseTaskResult / ConvertToOpenAIVideo 仍按原 content.video_url 映射，不改旧解析分支。
//
// 兼容来源：
//  1. GET /api/v3/contents/generations/tasks/{id}：顶层 id/status/content.video_url
//  2. GET /v1/video/generations/{id}：OpenAI 形态 output.video_url / metadata.url
//  3. 聚合网关 resultSummary 嵌套（先 Lift 再补字段）
func AdaptFetchResponseJSON(raw []byte) []byte {
	if len(raw) == 0 {
		return raw
	}
	var root map[string]any
	if err := common.Unmarshal(raw, &root); err != nil || root == nil {
		return raw
	}
	dto.LiftVideoPollResultSummary(root)
	adaptVideoGenerationsFieldsToContents(root)
	out, err := common.Marshal(root)
	if err != nil {
		return raw
	}
	return out
}

func adaptVideoGenerationsFieldsToContents(root map[string]any) {
	if root == nil {
		return
	}
	if data, ok := root["data"].(map[string]any); ok && len(data) > 0 {
		dto.LiftVideoPollResultSummary(data)
		if contentVideoURL(root) == "" {
			mergeMissingSeedanceFields(root, data)
		}
	}

	content, _ := root["content"].(map[string]any)
	if content == nil {
		content = map[string]any{}
	}
	videoURL := strings.TrimSpace(anyString(content["video_url"]))
	lastFrame := strings.TrimSpace(anyString(content["last_frame_url"]))

	if output, ok := root["output"].(map[string]any); ok {
		if videoURL == "" {
			videoURL = firstNonEmptyString(anyString(output["video_url"]), anyString(output["url"]))
		}
		if lastFrame == "" {
			lastFrame = strings.TrimSpace(anyString(output["last_frame_url"]))
		}
	}
	if meta, ok := root["metadata"].(map[string]any); ok {
		if videoURL == "" {
			videoURL = firstNonEmptyString(anyString(meta["video_url"]), anyString(meta["url"]))
		}
		if lastFrame == "" {
			lastFrame = strings.TrimSpace(anyString(meta["last_frame_url"]))
		}
	}
	if videoURL != "" {
		content["video_url"] = videoURL
	}
	if lastFrame != "" {
		content["last_frame_url"] = lastFrame
	}
	if len(content) > 0 {
		root["content"] = content
	}
}

func mergeMissingSeedanceFields(dst, src map[string]any) {
	if dst == nil || src == nil {
		return
	}
	for _, key := range []string{"id", "status", "model", "content", "output", "metadata", "usage", "error", "duration", "resolution", "ratio", "seed", "framespersecond", "generate_audio", "draft", "extra", "created_at", "updated_at"} {
		if videoPollValueEmpty(dst[key]) && !videoPollValueEmpty(src[key]) {
			dst[key] = src[key]
		}
	}
}

func videoPollValueEmpty(v any) bool {
	if v == nil {
		return true
	}
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x) == ""
	case map[string]any:
		return len(x) == 0
	default:
		return false
	}
}

func contentVideoURL(root map[string]any) string {
	if root == nil {
		return ""
	}
	content, _ := root["content"].(map[string]any)
	if content == nil {
		return ""
	}
	return strings.TrimSpace(anyString(content["video_url"]))
}

func anyString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

func firstNonEmptyString(values ...string) string {
	for _, v := range values {
		if t := strings.TrimSpace(v); t != "" {
			return t
		}
	}
	return ""
}

// BuildContentsGenerationsClientJSON 将任务落库 / 上游回包适配为 Contents API 查询形态，
// 供客户端 GET /api/v3/contents/generations/tasks/{task_id} 使用。
func BuildContentsGenerationsClientJSON(task *model.Task, upstreamJSON []byte) ([]byte, error) {
	raw := upstreamJSON
	if len(raw) == 0 && task != nil {
		raw = task.Data
	}
	adapted := AdaptFetchResponseJSON(raw)
	var root map[string]any
	if len(adapted) > 0 {
		if err := common.Unmarshal(adapted, &root); err != nil {
			return nil, fmt.Errorf("unmarshal seedance contents generations json failed: %w", err)
		}
	}
	if root == nil {
		root = map[string]any{}
	}

	if task != nil {
		if id := strings.TrimSpace(task.TaskID); id != "" {
			root["id"] = id
		}
		if modelName := strings.TrimSpace(task.Properties.OriginModelName); modelName != "" {
			if videoPollValueEmpty(root["model"]) {
				root["model"] = modelName
			}
		}
		if videoPollValueEmpty(root["status"]) {
			root["status"] = arkStatusFromTask(task.Status)
		}
		ts := dto.VideoPollTimestampContextFromTaskFields(task.SubmitTime, task.CreatedAt, task.FinishTime)
		created := ts.SubmitTime
		if created <= 0 {
			created = ts.CreatedAt
		}
		if created > 0 {
			root["created_at"] = created
		}
		if ts.FinishTime > 0 {
			root["updated_at"] = ts.FinishTime
		} else if created > 0 {
			if videoPollValueEmpty(root["updated_at"]) {
				root["updated_at"] = created
			}
		}
		if resultURL := strings.TrimSpace(task.GetResultURL()); resultURL != "" {
			content, _ := root["content"].(map[string]any)
			if content == nil {
				content = map[string]any{}
			}
			content["video_url"] = resultURL
			root["content"] = content
		}
	}

	// 去掉 OpenAI 兼容查询专有字段，保持 Contents API 回包形态。
	delete(root, "object")
	delete(root, "progress")
	delete(root, "output")
	delete(root, "metadata")
	delete(root, "completed_at")
	delete(root, "resultSummary")
	delete(root, "result_summary")

	out, err := common.Marshal(root)
	if err != nil {
		return nil, fmt.Errorf("marshal seedance contents generations json failed: %w", err)
	}
	return out, nil
}

func arkStatusFromTask(status model.TaskStatus) string {
	switch status {
	case model.TaskStatusQueued, model.TaskStatusNotStart, model.TaskStatusSubmitted:
		return "queued"
	case model.TaskStatusInProgress:
		return "running"
	case model.TaskStatusSuccess:
		return "succeeded"
	case model.TaskStatusFailure:
		return "failed"
	default:
		return strings.ToLower(strings.TrimSpace(string(status)))
	}
}
