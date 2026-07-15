package service

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
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
		// 腾讯云 DescribeTaskDetail：AigcVideoTask.Input.OutputConfig（Temporary 存储时常无 MetaData）
		res, dur, ratio := extractTencentInputOutputConfig(task.Data)
		mergeVideoSpecFields(&spec, res, dur, ratio)
	}
	return spec
}

// extractTencentInputOutputConfig 解析腾讯云回包中的计费核心三字段。
func extractTencentInputOutputConfig(data []byte) (resolution string, duration int, ratio string) {
	var env struct {
		Response *struct {
			AigcVideoTask *struct {
				Input *struct {
					OutputConfig *struct {
						Resolution  string  `json:"Resolution"`
						Duration    float64 `json:"Duration"`
						AspectRatio string  `json:"AspectRatio"`
					} `json:"OutputConfig"`
				} `json:"Input"`
			} `json:"AigcVideoTask"`
		} `json:"Response"`
	}
	if err := common.Unmarshal(data, &env); err != nil {
		return "", 0, ""
	}
	if env.Response == nil || env.Response.AigcVideoTask == nil || env.Response.AigcVideoTask.Input == nil || env.Response.AigcVideoTask.Input.OutputConfig == nil {
		return "", 0, ""
	}
	oc := env.Response.AigcVideoTask.Input.OutputConfig
	resolution = strings.TrimSpace(oc.Resolution)
	ratio = strings.TrimSpace(oc.AspectRatio)
	if oc.Duration > 0 {
		duration = int(oc.Duration)
		if float64(duration) < oc.Duration {
			duration++
		}
		if duration <= 0 {
			duration = 1
		}
	}
	return resolution, duration, ratio
}

// logTencentVodBillingMismatch 查询结算路径下比对提交参数与回包 Input.OutputConfig，不一致时打点。
func logTencentVodBillingMismatch(ctx context.Context, task *model.Task, taskResult *relaycommon.TaskInfo) {
	if task == nil || len(task.Data) == 0 {
		return
	}
	actRes, actDur, actRatio := extractTencentInputOutputConfig(task.Data)
	if actRes == "" && actDur <= 0 && actRatio == "" && taskResult != nil {
		actRes = strings.TrimSpace(taskResult.Resolution)
		actDur = taskResult.Duration
		actRatio = strings.TrimSpace(taskResult.Ratio)
	}
	if actRes == "" && actDur <= 0 && actRatio == "" {
		return
	}
	subRes, subDur, subRatio := "", 0, ""
	input := strings.TrimSpace(task.Properties.Input)
	if input != "" {
		var native struct {
			OutputConfig *struct {
				Resolution  string  `json:"Resolution"`
				Duration    float64 `json:"Duration"`
				AspectRatio string  `json:"AspectRatio"`
			} `json:"OutputConfig"`
		}
		if err := common.UnmarshalJsonStr(input, &native); err == nil && native.OutputConfig != nil {
			subRes = strings.TrimSpace(native.OutputConfig.Resolution)
			subRatio = strings.TrimSpace(native.OutputConfig.AspectRatio)
			if native.OutputConfig.Duration > 0 {
				subDur = int(math.Ceil(native.OutputConfig.Duration))
			}
		} else {
			var req relaycommon.TaskSubmitReq
			if err := common.UnmarshalJsonStr(input, &req); err == nil {
				subRes = strings.TrimSpace(req.Resolution)
				subRatio = strings.TrimSpace(req.Ratio)
				subDur = req.Duration
				if subDur <= 0 {
					if sec, err := strconv.Atoi(strings.TrimSpace(req.Seconds)); err == nil {
						subDur = sec
					}
				}
				if subRes == "" && req.Metadata != nil {
					if v, ok := req.Metadata["resolution"].(string); ok {
						subRes = strings.TrimSpace(v)
					}
				}
				if subRatio == "" && req.Metadata != nil {
					if v, ok := req.Metadata["ratio"].(string); ok {
						subRatio = strings.TrimSpace(v)
					}
				}
			}
		}
	}
	mismatched := false
	if subRes == "" && subDur <= 0 && subRatio == "" {
		mismatched = true // 前端未传参
	} else {
		if subRes != "" && !strings.EqualFold(subRes, actRes) {
			mismatched = true
		}
		if subDur > 0 && actDur > 0 && subDur != actDur {
			mismatched = true
		}
		if subRatio != "" && actRatio != "" && subRatio != actRatio {
			mismatched = true
		}
	}
	if !mismatched {
		return
	}
	logger.LogWarn(ctx, fmt.Sprintf(
		"[tencentvod] 计费核心字段不一致(查询结算) task=%s submitted={res:%s dur:%d ar:%s} actual={res:%s dur:%d ar:%s}",
		task.TaskID, subRes, subDur, subRatio, actRes, actDur, actRatio,
	))
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
