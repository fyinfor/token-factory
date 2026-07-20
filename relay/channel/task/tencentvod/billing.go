package tencentvod

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"

	"github.com/gin-gonic/gin"
)

// EstimateBilling 按提交 OutputConfig.Duration 预扣时长系数。
func (a *TaskAdaptor) EstimateBilling(c *gin.Context, _ *relaycommon.RelayInfo) map[string]float64 {
	if v, ok := c.Get(contextKeyBillingSpec); ok {
		if spec, ok := v.(BillingOutputSpec); ok && spec.Duration > 0 {
			return map[string]float64{"seconds": float64(spec.Duration)}
		}
	}
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil
	}
	oc, err := outputConfigFromTaskSubmitReq(req)
	if err != nil || oc == nil || oc.Duration <= 0 {
		return nil
	}
	return map[string]float64{"seconds": float64(oc.ToBillingSpec().Duration)}
}

// AdjustBillingOnComplete 任务完成时比对提交 OutputConfig 与回包 AigcVideoTask.Input.OutputConfig。
// 分辨率/时长/比例任一不一致则返回按实际规格重算后的额度，由 settleTaskBillingOnComplete → RecalculateTaskQuota 做多退少补与日志修正。
// 若通用按秒结算已先完成，本方法不会被调用；此处作为无按秒规则时的兜底。
func (a *TaskAdaptor) AdjustBillingOnComplete(task *model.Task, taskResult *relaycommon.TaskInfo) int {
	ctx := context.Background()
	if task == nil || task.Status != model.TaskStatusSuccess {
		return 0
	}
	submitted, submitOK := extractSubmittedBillingSpec(task)
	actual, actualOK := extractActualBillingSpec(task, taskResult)
	if !actualOK {
		logger.LogWarn(ctx, fmt.Sprintf("[tencentvod] task=%s 回包缺少 AigcVideoTask.Input.OutputConfig 计费字段，跳过 adaptor 差额结算", task.TaskID))
		return 0
	}
	mismatched, reasons := billingSpecMismatch(submitted, actual, submitOK)
	if !mismatched {
		return 0
	}
	logger.LogWarn(ctx, fmt.Sprintf(
		"[tencentvod] 计费核心字段不一致，触发重算 task=%s submitted={res:%s dur:%d ar:%s} actual={res:%s dur:%d ar:%s} reasons=%v",
		task.TaskID,
		submitted.Resolution, submitted.Duration, submitted.AspectRatio,
		actual.Resolution, actual.Duration, actual.AspectRatio,
		reasons,
	))
	pre := task.Quota
	if pre <= 0 {
		return 0
	}
	if submitOK && submitted.Duration > 0 && actual.Duration > 0 && submitted.Duration != actual.Duration {
		scaled := int(math.Round(float64(pre) * float64(actual.Duration) / float64(submitted.Duration)))
		if scaled <= 0 && pre > 0 {
			scaled = 1
		}
		return scaled
	}
	logger.LogWarn(ctx, fmt.Sprintf("[tencentvod] task=%s 仅分辨率/比例不一致且无时长差，adaptor 无法独立重算，依赖按秒结算链路", task.TaskID))
	return 0
}

func extractSubmittedBillingSpec(task *model.Task) (BillingOutputSpec, bool) {
	if task == nil {
		return BillingOutputSpec{}, false
	}
	input := strings.TrimSpace(task.Properties.Input)
	if input == "" {
		return BillingOutputSpec{}, false
	}
	var native CreateAigcVideoTaskRequest
	if err := common.UnmarshalJsonStr(input, &native); err == nil && native.OutputConfig != nil {
		spec := native.OutputConfig.ToBillingSpec()
		if spec.HasBillingCore() {
			return spec, true
		}
	}
	var req relaycommon.TaskSubmitReq
	if err := common.UnmarshalJsonStr(input, &req); err == nil {
		if oc, err := outputConfigFromTaskSubmitReq(req); err == nil {
			spec := oc.ToBillingSpec()
			if spec.HasBillingCore() {
				return spec, true
			}
		}
	}
	return BillingOutputSpec{}, false
}

func extractActualBillingSpec(task *model.Task, taskResult *relaycommon.TaskInfo) (BillingOutputSpec, bool) {
	if taskResult != nil {
		spec := BillingOutputSpec{
			Resolution:  strings.TrimSpace(taskResult.Resolution),
			Duration:    taskResult.Duration,
			AspectRatio: strings.TrimSpace(taskResult.Ratio),
		}
		if spec.HasBillingCore() {
			return spec, true
		}
	}
	if task != nil && len(task.Data) > 0 {
		if oc := ParseInputOutputConfigFromDescribeBody(task.Data); oc != nil {
			spec := oc.ToBillingSpec()
			if spec.HasBillingCore() {
				return spec, true
			}
		}
	}
	return BillingOutputSpec{}, false
}

func billingSpecMismatch(submitted, actual BillingOutputSpec, submitOK bool) (bool, []string) {
	if !submitOK {
		return true, []string{"submitted_missing"}
	}
	var reasons []string
	if !equalResolution(submitted.Resolution, actual.Resolution) {
		reasons = append(reasons, "resolution")
	}
	if submitted.Duration != actual.Duration {
		reasons = append(reasons, "duration")
	}
	if !equalAspectRatio(submitted.AspectRatio, actual.AspectRatio) {
		reasons = append(reasons, "aspect_ratio")
	}
	return len(reasons) > 0, reasons
}

func equalResolution(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}

func equalAspectRatio(a, b string) bool {
	return strings.TrimSpace(a) == strings.TrimSpace(b)
}

// ParseInputOutputConfigFromDescribeBody 从 DescribeTaskDetail 回包解析 AigcVideoTask.Input.OutputConfig。
func ParseInputOutputConfigFromDescribeBody(respBody []byte) *AigcVideoOutputConfig {
	if len(respBody) == 0 {
		return nil
	}
	var env struct {
		Response *struct {
			AigcVideoTask *struct {
				Input *struct {
					OutputConfig *AigcVideoOutputConfig `json:"OutputConfig,omitempty"`
				} `json:"Input,omitempty"`
			} `json:"AigcVideoTask,omitempty"`
		} `json:"Response"`
	}
	if err := common.Unmarshal(respBody, &env); err != nil {
		return nil
	}
	if env.Response == nil || env.Response.AigcVideoTask == nil || env.Response.AigcVideoTask.Input == nil {
		return nil
	}
	return env.Response.AigcVideoTask.Input.OutputConfig
}

// ApplyActualOutputConfigToTaskInfo 将回包 Input.OutputConfig 写入 TaskInfo 供结算链路消费。
func ApplyActualOutputConfigToTaskInfo(ti *relaycommon.TaskInfo, oc *AigcVideoOutputConfig) {
	if ti == nil || oc == nil {
		return
	}
	spec := oc.ToBillingSpec()
	if spec.Resolution != "" {
		ti.Resolution = spec.Resolution
	}
	if spec.Duration > 0 {
		ti.Duration = spec.Duration
	}
	if spec.AspectRatio != "" {
		ti.Ratio = spec.AspectRatio
	}
}
