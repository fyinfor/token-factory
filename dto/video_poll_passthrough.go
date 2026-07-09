package dto

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
)

// videoPollPassthroughFieldKeys 为视频任务查询需从上游原样透传给用户的字段。
var videoPollPassthroughFieldKeys = []string{"ratio", "resolution", "duration", "usage"}

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

// ExtractVideoPollPassthroughFields 从上游查询原始 JSON 中提取需透传的字段。
// 依次扫描顶层、data、result 对象；字段存在即保留原值（含 0 / false / 空字符串）。
func ExtractVideoPollPassthroughFields(upstreamJSON []byte) map[string]any {
	if len(upstreamJSON) == 0 {
		return nil
	}
	var root map[string]any
	if err := common.Unmarshal(upstreamJSON, &root); err != nil {
		return nil
	}
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
	return sources
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

// FinalizeVideoPollPassthroughJSON 用于渠道全量透传场景：保留上游 JSON 结构，仅补齐四字段并校正时间。
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
