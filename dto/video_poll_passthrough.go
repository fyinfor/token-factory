package dto

import (
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

// videoPollPassthroughFieldKeys 为视频任务查询需从上游原样透传给用户的字段。
// content 含 video_url / last_frame_url（return_last_frame=true 时）。
var videoPollPassthroughFieldKeys = []string{"ratio", "resolution", "duration", "usage", "content"}

// VideoPollTimestampContext 提供比上游更可靠的任务时间戳，用于校正 created_at / completed_at。
type VideoPollTimestampContext struct {
	SubmitTime int64
	CreatedAt  int64
	FinishTime int64
}

// IsVideoGenerationsFetchPath 判断是否为 GET /v1/video/generations/{task_id} 查询路由。
func IsVideoGenerationsFetchPath(path string) bool {
	return strings.HasPrefix(strings.TrimSpace(path), "/v1/video/generations/")
}

// IsContentsGenerationsFetchPath 判断是否为 GET /api/v3/contents/generations/tasks/{task_id} 查询路由。
func IsContentsGenerationsFetchPath(path string) bool {
	p := strings.TrimSpace(path)
	return strings.HasPrefix(p, "/api/v3/contents/generations/tasks/")
}

// ExtractVideoPollPassthroughFields 从上游查询原始 JSON 中提取需透传的字段。
// 依次扫描顶层、data、result、resultSummary 对象；字段存在即保留原值（含 0 / false / 空字符串）。
func ExtractVideoPollPassthroughFields(upstreamJSON []byte) map[string]any {
	if len(upstreamJSON) == 0 {
		return nil
	}
	var root map[string]any
	if err := common.Unmarshal(upstreamJSON, &root); err != nil {
		return nil
	}
	LiftVideoPollResultSummary(root)
	sources := videoPollPassthroughSources(root)
	if len(sources) == 0 {
		return nil
	}
	out := make(map[string]any)
	for _, key := range videoPollPassthroughFieldKeys {
		for _, src := range sources {
			v, ok := src[key]
			if !ok {
				continue
			}
			out[key] = v
			break
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func videoPollPassthroughSources(root map[string]any) []map[string]any {
	if root == nil {
		return nil
	}
	sources := []map[string]any{root}
	if data, ok := root["data"].(map[string]any); ok {
		sources = append(sources, data)
	}
	if result, ok := root["result"].(map[string]any); ok {
		sources = append(sources, result)
	}
	if summary := videoPollResultSummaryMap(root); summary != nil {
		sources = append(sources, summary)
	}
	return sources
}

// LiftVideoPollResultSummaryJSON 将 resultSummary / result_summary 中的成片字段提升到顶层后重新序列化。
// 供渠道 ParseTaskResult / ConvertToOpenAIVideo 把聚合网关回包还原为 Seedance/Ark 标准形态。
func LiftVideoPollResultSummaryJSON(raw []byte) []byte {
	if len(raw) == 0 {
		return raw
	}
	var root map[string]any
	if err := common.Unmarshal(raw, &root); err != nil || root == nil {
		return raw
	}
	LiftVideoPollResultSummary(root)
	out, err := common.Marshal(root)
	if err != nil {
		return raw
	}
	return out
}

// LiftVideoPollResultSummary 兼容火山方舟 Seedance 聚合查询回包：成片 URL / 时长 / 分辨率 / usage 放在 resultSummary 中。
// 仅在顶层缺失时回填，并把字符串 duration（如 "4"）规范为数字。
func LiftVideoPollResultSummary(root map[string]any) {
	if root == nil {
		return
	}
	summary := videoPollResultSummaryMap(root)
	if summary != nil {
		liftVideoPollContent(root, summary)
		for _, key := range []string{"duration", "resolution", "ratio", "usage", "error", "seed", "framespersecond", "generate_audio", "draft", "extra"} {
			if videoPollValueEmpty(root[key]) && !videoPollValueEmpty(summary[key]) {
				root[key] = summary[key]
			}
		}
		if videoPollValueEmpty(root["status"]) {
			if st := firstNonEmptyVideoPollValue(summary["upstreamStatus"], summary["upstream_status"], summary["status"]); !videoPollValueEmpty(st) {
				root["status"] = st
			}
		}
	}
	coerceVideoPollDuration(root)
}

func videoPollResultSummaryMap(root map[string]any) map[string]any {
	if root == nil {
		return nil
	}
	if summary, ok := root["resultSummary"].(map[string]any); ok && len(summary) > 0 {
		return summary
	}
	if summary, ok := root["result_summary"].(map[string]any); ok && len(summary) > 0 {
		return summary
	}
	return nil
}

func liftVideoPollContent(root, summary map[string]any) {
	sumContent, _ := summary["content"].(map[string]any)
	if len(sumContent) == 0 {
		return
	}
	rootContent, _ := root["content"].(map[string]any)
	if videoPollContentVideoURL(rootContent) == "" {
		if rootContent == nil {
			root["content"] = sumContent
			return
		}
		for _, key := range []string{"video_url", "last_frame_url"} {
			if videoPollValueEmpty(rootContent[key]) && !videoPollValueEmpty(sumContent[key]) {
				rootContent[key] = sumContent[key]
			}
		}
	}
}

func videoPollContentVideoURL(content map[string]any) string {
	if content == nil {
		return ""
	}
	if u, _ := content["video_url"].(string); strings.TrimSpace(u) != "" {
		return strings.TrimSpace(u)
	}
	if u, _ := content["url"].(string); strings.TrimSpace(u) != "" {
		return strings.TrimSpace(u)
	}
	return ""
}

func coerceVideoPollDuration(root map[string]any) {
	if root == nil {
		return
	}
	raw, ok := root["duration"]
	if !ok || raw == nil {
		return
	}
	s, isStr := raw.(string)
	if !isStr {
		return
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return
	}
	if n, err := strconv.Atoi(s); err == nil {
		root["duration"] = n
		return
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		root["duration"] = f
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

func firstNonEmptyVideoPollValue(values ...any) any {
	for _, v := range values {
		if !videoPollValueEmpty(v) {
			return v
		}
	}
	return nil
}

// MergeVideoPollPassthroughFields 将上游存在的透传字段合并进响应 JSON 顶层。
func MergeVideoPollPassthroughFields(responseJSON, upstreamJSON []byte) ([]byte, error) {
	fields := ExtractVideoPollPassthroughFields(upstreamJSON)
	if len(fields) == 0 {
		return responseJSON, nil
	}
	var resp map[string]any
	if err := common.Unmarshal(responseJSON, &resp); err != nil {
		return responseJSON, err
	}
	for k, v := range fields {
		resp[k] = v
	}
	return common.Marshal(resp)
}

// CorrectVideoPollTimestamps 校正响应中的 created_at / completed_at。
// 上游 Seedance/Ark 回包时间戳不可靠，以本站任务落库时间为准；无落库时间时不改写已有字段。
func CorrectVideoPollTimestamps(resp map[string]any, ts VideoPollTimestampContext, requestPath string) {
	if resp == nil {
		return
	}
	createdSec := pickVideoPollCreatedUnix(ts)
	if createdSec > 0 {
		resp["created_at"] = formatVideoPollTimestamp(createdSec, requestPath)
	}
	completedSec := pickVideoPollCompletedUnix(ts)
	if completedSec > 0 {
		resp["completed_at"] = formatVideoPollTimestamp(completedSec, requestPath)
	}
}

func pickVideoPollCreatedUnix(ts VideoPollTimestampContext) int64 {
	if ts.SubmitTime > 0 {
		return ts.SubmitTime
	}
	return ts.CreatedAt
}

func pickVideoPollCompletedUnix(ts VideoPollTimestampContext) int64 {
	return ts.FinishTime
}

func formatVideoPollTimestamp(sec int64, requestPath string) any {
	if sec <= 0 {
		return nil
	}
	// /v1/video/generations 与上游 Seedance/Ark 一致返回 Unix 秒，便于与任务日志（本地时区展示）对齐。
	// /v1/videos* 亦为 int64（new-api 兼容）。
	if IsOpenAIVideosCompatPath(requestPath) || IsVideoGenerationsFetchPath(requestPath) {
		return sec
	}
	return FormatTimeUnixRFC3339(sec)
}

// VideoPollTimestampContextFromTaskFields 从任务表字段构建时间校正上下文。
func VideoPollTimestampContextFromTaskFields(submitTime, createdAt, finishTime int64) VideoPollTimestampContext {
	return VideoPollTimestampContext{
		SubmitTime: submitTime,
		CreatedAt:  createdAt,
		FinishTime: finishTime,
	}
}

// FinalizeVideoPollResponseJSON 合并上游透传字段、校正时间戳，再按路由适配时间格式。
// 必须先 merge 再 AdaptOpenAIVideoJSONForPath，避免 struct 反序列化丢弃 ratio 等扩展字段。
func FinalizeVideoPollResponseJSON(responseJSON, upstreamJSON []byte, requestPath string, ts VideoPollTimestampContext) ([]byte, error) {
	merged, err := MergeVideoPollPassthroughFields(responseJSON, upstreamJSON)
	if err != nil {
		return responseJSON, err
	}
	var resp map[string]any
	if err := common.Unmarshal(merged, &resp); err != nil {
		return merged, nil
	}
	CorrectVideoPollTimestamps(resp, ts, requestPath)
	corrected, err := common.Marshal(resp)
	if err != nil {
		return merged, err
	}
	return AdaptOpenAIVideoJSONForPath(requestPath, corrected)
}

// FinalizeVideoPollPassthroughJSON 用于渠道全量透传场景：保留上游 JSON 结构，仅补齐透传字段并校正时间。
func FinalizeVideoPollPassthroughJSON(respJSON, upstreamJSON []byte, requestPath string, ts VideoPollTimestampContext) ([]byte, error) {
	merged, err := MergeVideoPollPassthroughFields(respJSON, upstreamJSON)
	if err != nil {
		return respJSON, err
	}
	var resp map[string]any
	if err := common.Unmarshal(merged, &resp); err != nil {
		return merged, nil
	}
	CorrectVideoPollTimestamps(resp, ts, requestPath)
	out, err := common.Marshal(resp)
	if err != nil {
		return merged, err
	}
	return AdaptOpenAIVideoJSONForPath(requestPath, out)
}
