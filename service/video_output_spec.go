package service

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

// resolveVideoOutputSpecFromUpstream 仅从任务完成后的上游查询回包解析视频规格（不含用户提交参数回退）。
func resolveVideoOutputSpecFromUpstream(task *model.Task, taskResult *relaycommon.TaskInfo) seedanceVideoSpec {
	spec := seedanceVideoSpec{}
	if taskResult != nil {
		mergeVideoSpecFields(&spec, taskResult.Resolution, taskResult.Duration, taskResult.Ratio)
	}
	if task != nil && len(task.Data) > 0 {
		var upstream struct {
			Resolution string `json:"resolution"`
			Duration   int    `json:"duration"`
			Ratio      string `json:"ratio"`
		}
		if err := common.Unmarshal(task.Data, &upstream); err == nil {
			mergeVideoSpecFields(&spec, upstream.Resolution, upstream.Duration, upstream.Ratio)
		}
	}
	return spec
}

func mergeVideoSpecFields(spec *seedanceVideoSpec, resolution string, duration int, ratio string) {
	if spec == nil {
		return
	}
	if spec.Resolution == "" {
		if label := common.FormatVideoResolutionLabel(strings.TrimSpace(resolution)); label != "" {
			spec.Resolution = label
		}
	}
	if spec.Duration <= 0 && duration > 0 {
		spec.Duration = duration
	}
	if spec.Ratio == "" {
		if r := strings.TrimSpace(ratio); r != "" {
			spec.Ratio = r
		}
	}
}

// videoMetadataFromTaskCompletion 优先从上游 resolution 解析成片元数据，无 resolution 时回退请求 size。
func videoMetadataFromTaskCompletion(task *model.Task, taskResult *relaycommon.TaskInfo) (*VideoMetadata, bool) {
	duration := 0.0
	hasAudio := false
	if spec := resolveVideoOutputSpecFromUpstream(task, taskResult); spec.Duration > 0 {
		duration = float64(spec.Duration)
	}
	if taskResult != nil && duration <= 0 && taskResult.Duration > 0 {
		duration = float64(taskResult.Duration)
	}
	if task != nil {
		var req relaycommon.TaskSubmitReq
		if err := common.UnmarshalJsonStr(task.Properties.Input, &req); err == nil {
			hasAudio = taskRequestHasAudio(req)
			if duration <= 0 {
				duration = float64(videoDurationFromTaskRequest(req))
			}
		}
	}
	if meta, ok := extractVideoMetadataFromTaskData(task); ok {
		if duration <= 0 && meta.DurationSec > 0 {
			duration = meta.DurationSec
		}
		if meta.HasAudio {
			hasAudio = true
		}
	}
	if duration <= 0 {
		return nil, false
	}
	if w, h, ok := videoDimensionsFromTaskCompletion(task, taskResult); ok {
		return &VideoMetadata{
			DurationSec: duration,
			Width:       w,
			Height:      h,
			HasAudio:    hasAudio,
		}, true
	}
	return nil, false
}

// videoDimensionsFromTaskCompletion 计费匹配：上游 resolution > 请求 resolution > 请求 size > 回包其它推断。
func videoDimensionsFromTaskCompletion(task *model.Task, taskResult *relaycommon.TaskInfo) (int, int, bool) {
	spec := resolveVideoOutputSpecFromUpstream(task, taskResult)
	if strings.TrimSpace(spec.Resolution) != "" {
		if w, h, ok := common.ParseVideoResolutionAndRatio(spec.Resolution, spec.Ratio); ok {
			return w, h, true
		}
	}
	if task != nil {
		var req relaycommon.TaskSubmitReq
		if err := common.UnmarshalJsonStr(task.Properties.Input, &req); err == nil {
			if w, h, ok := common.ResolveVideoDimensionsFromRequest(req.Size, req.Resolution, req.Ratio, req.Metadata); ok {
				return w, h, true
			}
		}
	}
	if meta, ok := extractVideoMetadataFromTaskData(task); ok && meta.Width > 0 && meta.Height > 0 {
		return meta.Width, meta.Height, true
	}
	return 0, 0, false
}

// VideoBillingResolutionLabelFromRequest 提取用于计费档位匹配的 resolution 标识（如 720p）。
func VideoBillingResolutionLabelFromRequest(req relaycommon.TaskSubmitReq) string {
	if raw := videoResolutionParamFromRequest(req); raw != "" {
		if label := common.FormatVideoResolutionLabel(raw); label != "" {
			return label
		}
		return strings.TrimSpace(raw)
	}
	return ""
}

// VideoBillingResolutionLabelForTask 成片结算时优先取上游 resolution，否则回退请求参数。
func VideoBillingResolutionLabelForTask(task *model.Task, taskResult *relaycommon.TaskInfo) string {
	spec := resolveVideoOutputSpecFromUpstream(task, taskResult)
	if raw := strings.TrimSpace(spec.Resolution); raw != "" {
		if label := common.FormatVideoResolutionLabel(raw); label != "" {
			return label
		}
		return raw
	}
	if task != nil {
		var req relaycommon.TaskSubmitReq
		if err := common.UnmarshalJsonStr(task.Properties.Input, &req); err == nil {
			return VideoBillingResolutionLabelFromRequest(req)
		}
	}
	return ""
}
