package openaivideo

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

// 本站内部视频接口 /video/generations 轮询回包存在两种 JSON 形态。
// ParseTaskResult / ConvertToOpenAIVideo 先走本文件识别，再回落到原 MaaS/Sophnet/Ark 解析。
//
// 格式1（标准直接返回）：顶层即任务对象，usage 在根级。
//   识别：顶层 status 为字符串，且存在 object=video.generation，或根级 usage / content / output。
//   token：usage.total_tokens（为 0 时回退 completion_tokens）。
//
// 格式2（包装嵌套）：外层 code/data，真实任务在 data.data。
//   识别：data 为对象，且其下再嵌套 data，或 progress 为 "100%" 这类百分数字符串。
//   token：data.data.usage.completion_tokens 作为总 token 数。
//   progress 为字符串；status 可能同时出现在 data.status（SUCCESS）与 data.data.status（succeeded）。

type videoGenerationsCompatFields struct {
	TaskID           string
	Model            string
	RawStatus        string
	Progress         string
	VideoURL         string
	ErrorMessage     string
	ErrorCode        string
	CompletionTokens int
	TotalTokens      int
	Duration         int
	Ratio            string
	Resolution       string
}

func parseVideoGenerationsCompatResult(respBody []byte) (*relaycommon.TaskInfo, bool, error) {
	fields, ok := extractVideoGenerationsCompatFields(respBody)
	if !ok {
		return nil, false, nil
	}
	return fields.toTaskInfo(), true, nil
}

func extractVideoGenerationsCompatFields(respBody []byte) (videoGenerationsCompatFields, bool) {
	var empty videoGenerationsCompatFields
	if len(respBody) == 0 {
		return empty, false
	}
	var root map[string]any
	if err := common.Unmarshal(respBody, &root); err != nil || root == nil {
		return empty, false
	}
	dto.LiftVideoPollResultSummary(root)
	if isVideoGenerationsFormat2(root) {
		return extractFormat2Fields(root), true
	}
	if isVideoGenerationsFormat1(root) {
		return extractFormat1Fields(root), true
	}
	return empty, false
}

// isVideoGenerationsFormat2 格式2：外层套 code/data，真实任务数据嵌套在 data.data。
func isVideoGenerationsFormat2(root map[string]any) bool {
	if root == nil {
		return false
	}
	data := compatAsMap(root["data"])
	if data == nil {
		return false
	}
	nested := compatAsMap(data["data"])
	if nested != nil {
		if compatAsMap(nested["usage"]) != nil || compatAsMap(nested["content"]) != nil || compatAsMap(nested["output"]) != nil {
			return true
		}
		if strings.TrimSpace(compatAsString(nested["status"])) != "" && strings.TrimSpace(compatAsString(nested["model"])) != "" {
			return true
		}
	}
	_, codeIsNum := compatAsFloat64(root["code"])
	progress, progressIsStr := data["progress"].(string)
	if !codeIsNum && root["code"] != nil {
		if nested != nil {
			return true
		}
		if progressIsStr && strings.Contains(progress, "%") {
			return true
		}
		if strings.TrimSpace(compatAsString(data["task_id"])) != "" && strings.TrimSpace(compatAsString(data["status"])) != "" {
			return true
		}
	}
	return false
}

// isVideoGenerationsFormat1 格式1：顶层直接包含任务信息，usage 在根级别。
func isVideoGenerationsFormat1(root map[string]any) bool {
	if root == nil || isVideoGenerationsFormat2(root) {
		return false
	}
	if _, hasResult := root["result"]; hasResult {
		if _, ok := compatAsFloat64(root["code"]); ok {
			return false
		}
		if _, ok := compatAsFloat64(root["status"]); ok {
			return false
		}
	}
	if _, statusIsNum := compatAsFloat64(root["status"]); statusIsNum {
		return false
	}
	if strings.TrimSpace(compatAsString(root["status"])) == "" {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(compatAsString(root["object"])), dto.ObjectVideoGeneration) {
		return true
	}
	if compatAsMap(root["usage"]) != nil {
		return true
	}
	if compatAsMap(root["content"]) != nil || compatAsMap(root["output"]) != nil {
		return true
	}
	return false
}

func extractFormat1Fields(root map[string]any) videoGenerationsCompatFields {
	fields := videoGenerationsCompatFields{
		TaskID:     firstNonEmpty(compatAsString(root["id"]), compatAsString(root["task_id"])),
		Model:      compatAsString(root["model"]),
		RawStatus:  compatAsString(root["status"]),
		Progress:   normalizeVideoProgress(root["progress"]),
		VideoURL:   firstNonEmpty(compatNestedVideoURL(root["content"]), compatNestedVideoURL(root["output"])),
		Duration:   compatAsInt(root["duration"]),
		Ratio:      compatAsString(root["ratio"]),
		Resolution: compatAsString(root["resolution"]),
	}
	fields.ErrorMessage, fields.ErrorCode = compatExtractError(root["error"])
	if msg := strings.TrimSpace(compatAsString(root["message"])); fields.ErrorMessage == "" && isFailedVideoStatus(fields.RawStatus) {
		fields.ErrorMessage = msg
	}
	completion, total := tokensFromUsageMap(compatAsMap(root["usage"]), false)
	fields.CompletionTokens = completion
	fields.TotalTokens = total
	return fields
}

func extractFormat2Fields(root map[string]any) videoGenerationsCompatFields {
	data := compatAsMap(root["data"])
	if data == nil {
		data = map[string]any{}
	}
	nested := compatAsMap(data["data"])
	if nested == nil {
		nested = map[string]any{}
	}

	outerStatus := compatAsString(data["status"])
	innerStatus := compatAsString(nested["status"])
	rawStatus := firstNonEmpty(innerStatus, outerStatus)

	fields := videoGenerationsCompatFields{
		TaskID:     firstNonEmpty(compatAsString(data["task_id"]), compatAsString(nested["id"]), compatAsString(data["id"])),
		Model:      firstNonEmpty(compatAsString(nested["model"]), compatAsString(data["model"])),
		RawStatus:  rawStatus,
		Progress:   firstNonEmpty(normalizeVideoProgress(data["progress"]), normalizeVideoProgress(nested["progress"])),
		VideoURL:   firstNonEmpty(compatNestedVideoURL(nested["content"]), compatNestedVideoURL(nested["output"]), compatNestedVideoURL(data["content"]), compatNestedVideoURL(data["output"])),
		Duration:   compatAsInt(firstNonNil(nested["duration"], data["duration"])),
		Ratio:      firstNonEmpty(compatAsString(nested["ratio"]), compatAsString(data["ratio"])),
		Resolution: firstNonEmpty(compatAsString(nested["resolution"]), compatAsString(data["resolution"])),
	}
	fields.ErrorMessage, fields.ErrorCode = compatExtractError(firstNonNil(nested["error"], data["error"], root["error"]))
	if fields.ErrorMessage == "" {
		if msg := strings.TrimSpace(compatAsString(root["message"])); msg != "" && (isFailedVideoStatus(rawStatus) || isFormat2EnvelopeFailure(root)) {
			fields.ErrorMessage = msg
		}
	}
	if isFormat2EnvelopeFailure(root) && fields.RawStatus == "" {
		fields.RawStatus = "failed"
	}
	// 格式2 token：data.data.usage.completion_tokens 作为总 token 数。
	completion, total := tokensFromUsageMap(compatAsMap(nested["usage"]), true)
	if total <= 0 {
		completion, total = tokensFromUsageMap(compatAsMap(data["usage"]), true)
	}
	fields.CompletionTokens = completion
	fields.TotalTokens = total
	return fields
}

func (f videoGenerationsCompatFields) toTaskInfo() *relaycommon.TaskInfo {
	taskResult := &relaycommon.TaskInfo{Code: 0}
	mapped := mapVideoGenerationsStatus(f.RawStatus)
	taskResult.Status = mapped.status
	taskResult.Progress = firstNonEmpty(f.Progress, mapped.progress)
	taskResult.Url = f.VideoURL
	taskResult.CompletionTokens = f.CompletionTokens
	taskResult.TotalTokens = f.TotalTokens
	taskResult.Duration = f.Duration
	taskResult.Ratio = f.Ratio
	taskResult.Resolution = f.Resolution
	if mapped.failed {
		taskResult.Reason = firstNonEmpty(f.ErrorMessage, f.ErrorCode, fmt.Sprintf("video task %s", f.RawStatus))
		if taskResult.Progress == "" {
			taskResult.Progress = "100%"
		}
	} else if f.ErrorMessage != "" || f.ErrorCode != "" {
		taskResult.Status = model.TaskStatusFailure
		taskResult.Progress = "100%"
		taskResult.Reason = firstNonEmpty(f.ErrorMessage, f.ErrorCode)
	}
	if taskResult.Status == model.TaskStatusSuccess && taskResult.Progress == "" {
		taskResult.Progress = "100%"
	}
	return taskResult
}

func applyVideoGenerationsCompatToOpenAIVideo(ov *dto.OpenAIVideo, respBody []byte) bool {
	if ov == nil {
		return false
	}
	fields, ok := extractVideoGenerationsCompatFields(respBody)
	if !ok {
		return false
	}
	if st := mapArkVideoStatusToOpenAI(fields.RawStatus); st != "" {
		ov.Status = st
	}
	if fields.Progress != "" {
		ov.SetProgressStr(fields.Progress)
	}
	if fields.VideoURL != "" {
		ov.SetMetadata("url", fields.VideoURL)
	}
	if strings.TrimSpace(ov.Model) == "" && fields.Model != "" {
		ov.Model = fields.Model
	}
	if fields.ErrorMessage != "" || fields.ErrorCode != "" || ov.Status == dto.VideoStatusFailed {
		if ov.Error == nil && (fields.ErrorMessage != "" || fields.ErrorCode != "" || fields.RawStatus != "") {
			ov.Error = &dto.OpenAIVideoError{
				Message: firstNonEmpty(fields.ErrorMessage, fields.RawStatus, "video task failed"),
				Code:    firstNonEmpty(fields.ErrorCode, "video_task_failed"),
			}
		}
	}
	if fields.TotalTokens > 0 || fields.CompletionTokens > 0 {
		ov.Usage = &dto.OpenAIVideoUsage{
			CompletionTokens: fields.CompletionTokens,
			TotalTokens:      fields.TotalTokens,
		}
		if ov.Usage.TotalTokens == 0 {
			ov.Usage.TotalTokens = ov.Usage.CompletionTokens
		}
	}
	return true
}

type mappedVideoStatus struct {
	status   string
	progress string
	failed   bool
}

func mapVideoGenerationsStatus(raw string) mappedVideoStatus {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "queued", "pending", "submitted", "not_start", "not-start":
		return mappedVideoStatus{status: model.TaskStatusQueued, progress: taskcommon.ProgressQueued}
	case "running", "in_progress", "processing":
		return mappedVideoStatus{status: model.TaskStatusInProgress, progress: taskcommon.ProgressInProgress}
	case "succeeded", "completed", "success", "done":
		return mappedVideoStatus{status: model.TaskStatusSuccess, progress: "100%"}
	case "failed", "failure", "expired", "cancelled", "canceled", "error":
		return mappedVideoStatus{status: model.TaskStatusFailure, progress: "100%", failed: true}
	default:
		if raw == "" {
			return mappedVideoStatus{status: model.TaskStatusInProgress, progress: taskcommon.ProgressInProgress}
		}
		return mappedVideoStatus{status: model.TaskStatusInProgress, progress: taskcommon.ProgressInProgress}
	}
}

func isFailedVideoStatus(raw string) bool {
	return mapVideoGenerationsStatus(raw).failed
}

func isFormat2EnvelopeFailure(root map[string]any) bool {
	if root == nil {
		return false
	}
	code := strings.ToLower(strings.TrimSpace(compatAsString(root["code"])))
	if code == "" {
		return false
	}
	switch code {
	case "success", "ok", "0", "succeeded", "completed":
		return false
	default:
		return true
	}
}

func parseVideoGeneratorSubmit(body []byte) (taskID string, failMsg string, unmarshalErr error) {
	var root map[string]any
	if err := common.Unmarshal(body, &root); err != nil {
		return "", "", err
	}
	if root == nil {
		return "", "", nil
	}

	if isVideoGenerationsFormat2(root) {
		if isFormat2EnvelopeFailure(root) {
			msg := firstNonEmpty(compatAsString(root["message"]), compatAsString(root["code"]), "video upstream returned error")
			return "", msg, nil
		}
		data := compatAsMap(root["data"])
		id := ""
		if data != nil {
			id = firstNonEmpty(compatAsString(data["task_id"]), compatAsString(data["id"]))
			if nested := compatAsMap(data["data"]); nested != nil {
				id = firstNonEmpty(id, compatAsString(nested["id"]), compatAsString(nested["task_id"]))
			}
		}
		return id, "", nil
	}

	if _, hasResult := root["result"]; hasResult {
		if n, ok := compatAsFloat64(root["status"]); ok && n != 0 {
			msg := firstNonEmpty(compatAsString(root["message"]), fmt.Sprintf("video upstream returned status=%v", root["status"]))
			return "", msg, nil
		}
		if result := compatAsMap(root["result"]); result != nil {
			if id := firstNonEmpty(compatAsString(result["task_id"]), compatAsString(result["id"])); id != "" {
				return id, "", nil
			}
		}
	}

	if msg, code := compatExtractError(root["error"]); msg != "" || code != "" {
		return "", firstNonEmpty(msg, code, "video upstream returned error"), nil
	}

	return firstNonEmpty(compatAsString(root["id"]), compatAsString(root["task_id"])), "", nil
}

func tokensFromUsageMap(usage map[string]any, preferCompletion bool) (completion, total int) {
	if usage == nil {
		return 0, 0
	}
	completion = compatAsInt(usage["completion_tokens"])
	total = compatAsInt(usage["total_tokens"])
	if preferCompletion {
		if completion > 0 {
			total = completion
		}
		return completion, total
	}
	if total == 0 && completion > 0 {
		total = completion
	}
	return completion, total
}

func compatNestedVideoURL(node any) string {
	m := compatAsMap(node)
	if m == nil {
		return ""
	}
	return firstNonEmpty(compatAsString(m["video_url"]), compatAsString(m["url"]))
}

func compatExtractError(v any) (message, code string) {
	if v == nil {
		return "", ""
	}
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s), ""
	}
	m := compatAsMap(v)
	if m == nil {
		return "", ""
	}
	return firstNonEmpty(compatAsString(m["message"]), compatAsString(m["msg"])), compatAsString(m["code"])
}

func normalizeVideoProgress(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		s = strings.TrimSpace(s)
		if s == "" {
			return ""
		}
		if strings.HasSuffix(s, "%") {
			return s
		}
		return s + "%"
	}
	if f, ok := compatAsFloat64(v); ok {
		return strconv.Itoa(int(f)) + "%"
	}
	return ""
}

func compatAsMap(v any) map[string]any {
	if v == nil {
		return nil
	}
	m, _ := v.(map[string]any)
	return m
}

func compatAsString(v any) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	case float64:
		if x == float64(int64(x)) {
			return strconv.FormatInt(int64(x), 10)
		}
		return strconv.FormatFloat(x, 'f', -1, 64)
	case float32:
		return compatAsString(float64(x))
	case int:
		return strconv.Itoa(x)
	case int32:
		return strconv.FormatInt(int64(x), 10)
	case int64:
		return strconv.FormatInt(x, 10)
	case jsonNumberStringer:
		return strings.TrimSpace(x.String())
	default:
		return strings.TrimSpace(fmt.Sprint(x))
	}
}

func compatAsFloat64(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int32:
		return float64(x), true
	case int64:
		return float64(x), true
	case jsonNumberStringer:
		f, err := strconv.ParseFloat(x.String(), 64)
		return f, err == nil
	default:
		return 0, false
	}
}

func compatAsInt(v any) int {
	if f, ok := compatAsFloat64(v); ok {
		return int(f)
	}
	if s := strings.TrimSpace(compatAsString(v)); s != "" {
		if n, err := strconv.Atoi(strings.TrimSuffix(s, "%")); err == nil {
			return n
		}
	}
	return 0
}

func firstNonNil(values ...any) any {
	for _, v := range values {
		if v != nil {
			return v
		}
	}
	return nil
}

type jsonNumberStringer interface {
	String() string
}
