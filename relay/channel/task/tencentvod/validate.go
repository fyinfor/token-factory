package tencentvod

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

const contextKeyNativeBody = "tencentvod_native_body"
const contextKeyBillingSpec = "tencentvod_billing_spec"

// 官方 ModelName 取值（CreateAigcVideoTask）。
var allowedModelNames = map[string]struct{}{
	"Kling": {}, "Vidu": {}, "Hailuo": {}, "Hunyuan": {},
	"Mingmou": {}, "GV": {}, "OS": {}, "PixVerse": {},
}

// 官方 Resolution 并集（按模型约束在 validateResolution 内再收紧）。
var resolutionByModel = map[string]map[string]struct{}{
	"Kling":    {"720P": {}, "1080P": {}},
	"Hailuo":   {"768P": {}, "1080P": {}},
	"Vidu":     {"720P": {}, "1080P": {}},
	"GV":       {"720P": {}, "1080P": {}},
	"OS":       {"720P": {}},
	"PixVerse": {"540p": {}, "720p": {}, "1080p": {}, "2k": {}, "4k": {}},
}

// 官方 AspectRatio（Hailuo 官方暂不支持，网关仍要求显式传入以保证计费可比对）。
var aspectRatioByModel = map[string]map[string]struct{}{
	"Kling":    {"16:9": {}, "9:16": {}, "1:1": {}},
	"Vidu":     {"16:9": {}, "9:16": {}, "4:3": {}, "3:4": {}, "1:1": {}},
	"GV":       {"16:9": {}, "9:16": {}},
	"OS":       {"16:9": {}, "9:16": {}},
	"Hailuo":   {"16:9": {}, "9:16": {}, "1:1": {}},
	"PixVerse": {"16:9": {}, "4:3": {}, "1:1": {}, "3:4": {}, "9:16": {}, "2:3": {}, "3:2": {}, "21:9": {}},
}

// validateNativeOrLegacyRequest 提交前校验：原生腾讯云 JSON 或 OpenAI 风格 TaskSubmitReq。
func (a *TaskAdaptor) validateNativeOrLegacyRequest(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	raw, err := common.GetBodyStorage(c)
	if err != nil {
		return service.TaskErrorWrapperLocal(err, "read_body_failed", http.StatusBadRequest)
	}
	body, err := raw.Bytes()
	if err != nil {
		return service.TaskErrorWrapperLocal(err, "read_body_failed", http.StatusBadRequest)
	}

	var probe map[string]any
	if err := common.Unmarshal(body, &probe); err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}

	if isNativeTencentVideoBody(probe) {
		var req CreateAigcVideoTaskRequest
		if err := common.Unmarshal(body, &req); err != nil {
			return service.TaskErrorWrapperLocal(err, "invalid_tencent_vod_body", http.StatusBadRequest)
		}
		cred, credErr := ParseCredentials(a.apiKey)
		if credErr != nil {
			return service.TaskErrorWrapperLocal(credErr, "invalid_credentials", http.StatusBadRequest)
		}
		if err := validateCreateAigcVideoTaskRequest(&req, cred.SubAppID); err != nil {
			logger.LogWarn(c.Request.Context(), fmt.Sprintf("[tencentvod] 提交前参数校验失败: %v", err))
			return service.TaskErrorWrapperLocal(err, "invalid_tencent_vod_params", http.StatusBadRequest)
		}
		enrichNativeCreateRequestAudio(&req, true)
		// 强制使用渠道密钥中的 SubAppId，避免前端传错导致计费/资源错位。
		req.SubAppId = cred.SubAppID
		normalized, marshalErr := common.Marshal(req)
		if marshalErr != nil {
			return service.TaskErrorWrapperLocal(marshalErr, "marshal_request_failed", http.StatusInternalServerError)
		}
		c.Set(contextKeyNativeBody, normalized)
		c.Set(contextKeyBillingSpec, req.OutputConfig.ToBillingSpec())
		info.Action = constant.TaskActionGenerate
		if len(req.FileInfos) == 0 && strings.TrimSpace(req.Prompt) != "" {
			info.Action = constant.TaskActionTextGenerate
		}
		// 同步写入 TaskSubmitReq，供预扣费与任务 Properties.Input 比对。
		storeLegacyFromNative(c, info, &req)
		return nil
	}

	if taskErr := relaycommon.ValidateBasicTaskRequest(c, info, constant.TaskActionGenerate); taskErr != nil {
		return taskErr
	}
	taskReq, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	modelName, modelVersion := SplitCombinedModel(taskcommonUpstreamModel(info, taskReq.Model))
	if strings.TrimSpace(modelName) == "" || strings.TrimSpace(modelVersion) == "" {
		return service.TaskErrorWrapperLocal(
			fmt.Errorf("model 须为 ModelName-ModelVersion，例如 GV-3.1"),
			"invalid_model", http.StatusBadRequest,
		)
	}
	oc, err := outputConfigFromTaskSubmitReq(taskReq)
	if err != nil {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("[tencentvod] OpenAI风格入参缺少计费核心字段: %v", err))
		return service.TaskErrorWrapperLocal(err, "missing_output_config", http.StatusBadRequest)
	}
	if err := validateOutputConfig(modelName, oc); err != nil {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("[tencentvod] OutputConfig 枚举校验失败: %v", err))
		return service.TaskErrorWrapperLocal(err, "invalid_output_config", http.StatusBadRequest)
	}
	c.Set(contextKeyBillingSpec, oc.ToBillingSpec())
	return nil
}

func taskcommonUpstreamModel(info *relaycommon.RelayInfo, fallback string) string {
	if info != nil {
		if u := strings.TrimSpace(info.UpstreamModelName); u != "" {
			return u
		}
		if o := strings.TrimSpace(info.OriginModelName); o != "" {
			return o
		}
	}
	return strings.TrimSpace(fallback)
}

func isNativeTencentVideoBody(probe map[string]any) bool {
	if probe == nil {
		return false
	}
	if _, ok := probe["ModelName"]; ok {
		return true
	}
	if _, ok := probe["OutputConfig"]; ok {
		return true
	}
	if _, ok := probe["SubAppId"]; ok {
		return true
	}
	if _, ok := probe["FileInfos"]; ok {
		return true
	}
	return false
}

func storeLegacyFromNative(c *gin.Context, info *relaycommon.RelayInfo, req *CreateAigcVideoTaskRequest) {
	legacy := relaycommon.TaskSubmitReq{
		Prompt: strings.TrimSpace(req.Prompt),
		Model:  strings.TrimSpace(req.ModelName) + "-" + strings.TrimSpace(req.ModelVersion),
	}
	if req.OutputConfig != nil {
		spec := req.OutputConfig.ToBillingSpec()
		legacy.Duration = spec.Duration
		legacy.Resolution = spec.Resolution
		legacy.Ratio = spec.AspectRatio
		legacy.Metadata = map[string]interface{}{
			"resolution":   spec.Resolution,
			"ratio":        spec.AspectRatio,
			"duration":     spec.Duration,
			"OutputConfig": req.OutputConfig,
		}
		if ag := strings.TrimSpace(req.OutputConfig.AudioGeneration); ag != "" {
			legacy.Metadata["generate_audio"] = strings.EqualFold(ag, "Enabled")
		}
	}
	for _, fi := range req.FileInfos {
		typ := strings.TrimSpace(fi.Type)
		switch {
		case strings.EqualFold(typ, "Url") && strings.TrimSpace(fi.Url) != "":
			legacy.Images = append(legacy.Images, strings.TrimSpace(fi.Url))
		case strings.EqualFold(typ, "Base64") && strings.TrimSpace(fi.Base64) != "":
			legacy.Images = append(legacy.Images, "data:image/jpeg;base64,"+strings.TrimSpace(fi.Base64))
		}
	}
	if len(legacy.Images) == 1 {
		legacy.Image = legacy.Images[0]
	}
	if info.Action == "" {
		info.Action = constant.TaskActionGenerate
	}
	c.Set("task_request", legacy)
}

func validateCreateAigcVideoTaskRequest(req *CreateAigcVideoTaskRequest, credSubAppID uint64) error {
	if req == nil {
		return fmt.Errorf("request body is empty")
	}
	if req.SubAppId == 0 {
		// 前端未传：回填渠道密钥 SubAppId（风险场景②）
		req.SubAppId = credSubAppID
	}
	if req.SubAppId == 0 {
		return fmt.Errorf("SubAppId is required")
	}
	if credSubAppID > 0 && req.SubAppId != credSubAppID {
		return fmt.Errorf("SubAppId=%d 与渠道密钥 SubAppId=%d 不一致", req.SubAppId, credSubAppID)
	}
	modelName := strings.TrimSpace(req.ModelName)
	modelVersion := strings.TrimSpace(req.ModelVersion)
	if modelName == "" {
		return fmt.Errorf("ModelName is required")
	}
	if modelVersion == "" {
		return fmt.Errorf("ModelVersion is required")
	}
	if _, ok := allowedModelNames[modelName]; !ok {
		return fmt.Errorf("ModelName %q 不在官方枚举内", modelName)
	}
	if err := validateFileInfos(req.FileInfos); err != nil {
		return err
	}
	if req.OutputConfig == nil {
		return fmt.Errorf("OutputConfig 必填（含 Resolution/Duration/AspectRatio），缺失会导致请求与回包参数错位")
	}
	return validateOutputConfig(modelName, req.OutputConfig)
}

func validateFileInfos(files []AigcVideoFileInfo) error {
	for i, fi := range files {
		typ := strings.TrimSpace(fi.Type)
		if typ == "" {
			// 兼容仅传 FileId / Url / Base64 的宽松写法
			switch {
			case strings.TrimSpace(fi.FileId) != "":
				continue
			case strings.TrimSpace(fi.Url) != "":
				continue
			case strings.TrimSpace(fi.Base64) != "":
				continue
			default:
				return fmt.Errorf("FileInfos[%d]: Type 或 Url/Base64/FileId 至少提供一项", i)
			}
		}
		switch strings.ToLower(typ) {
		case "url":
			if strings.TrimSpace(fi.Url) == "" {
				return fmt.Errorf("FileInfos[%d]: Type=Url 时 Url 必填", i)
			}
		case "base64":
			if strings.TrimSpace(fi.Base64) == "" {
				return fmt.Errorf("FileInfos[%d]: Type=Base64 时 Base64 必填", i)
			}
		case "file":
			if strings.TrimSpace(fi.FileId) == "" {
				return fmt.Errorf("FileInfos[%d]: Type=File 时 FileId 必填", i)
			}
		default:
			return fmt.Errorf("FileInfos[%d]: Type=%q 非法，仅支持 Url/Base64/File", i, typ)
		}
	}
	return nil
}

func validateOutputConfig(modelName string, oc *AigcVideoOutputConfig) error {
	if oc == nil {
		return fmt.Errorf("OutputConfig is required")
	}
	res := strings.TrimSpace(oc.Resolution)
	if res == "" {
		return fmt.Errorf("OutputConfig.Resolution 必填")
	}
	if oc.Duration <= 0 {
		return fmt.Errorf("OutputConfig.Duration 必填且须 > 0")
	}
	ar := strings.TrimSpace(oc.AspectRatio)
	if ar == "" {
		return fmt.Errorf("OutputConfig.AspectRatio 必填")
	}
	if err := validateResolution(modelName, res); err != nil {
		return err
	}
	if err := validateAspectRatio(modelName, ar); err != nil {
		return err
	}
	if err := validateDuration(modelName, oc.Duration); err != nil {
		return err
	}
	return nil
}

func validateResolution(modelName, resolution string) error {
	allowed, ok := resolutionByModel[modelName]
	if !ok {
		// 未知模型：仅校验非空（已在上层保证）
		return nil
	}
	if _, hit := allowed[resolution]; hit {
		return nil
	}
	// PixVerse 大小写敏感（官方示例为小写 p）；其余模型官方为大写 P
	lower := strings.ToLower(resolution)
	for k := range allowed {
		if strings.ToLower(k) == lower {
			return nil
		}
	}
	keys := make([]string, 0, len(allowed))
	for k := range allowed {
		keys = append(keys, k)
	}
	return fmt.Errorf("OutputConfig.Resolution=%q 对 ModelName=%s 非法，允许: %s", resolution, modelName, strings.Join(keys, ","))
}

func validateAspectRatio(modelName, aspectRatio string) error {
	allowed, ok := aspectRatioByModel[modelName]
	if !ok {
		return nil
	}
	if _, hit := allowed[aspectRatio]; hit {
		return nil
	}
	keys := make([]string, 0, len(allowed))
	for k := range allowed {
		keys = append(keys, k)
	}
	return fmt.Errorf("OutputConfig.AspectRatio=%q 对 ModelName=%s 非法，允许: %s", aspectRatio, modelName, strings.Join(keys, ","))
}

func validateDuration(modelName string, duration float64) error {
	if duration <= 0 {
		return fmt.Errorf("OutputConfig.Duration 须 > 0")
	}
	// 官方文档按时长为区间/离散值，且随 ModelVersion 变化；此处做宽松上界防呆，精确枚举由上游拒绝。
	switch modelName {
	case "Kling", "PixVerse":
		if duration > 15 {
			return fmt.Errorf("OutputConfig.Duration=%v 对 %s 超出上限 15 秒", duration, modelName)
		}
	case "Hailuo":
		if duration != 6 && duration != 10 {
			return fmt.Errorf("OutputConfig.Duration=%v 对 Hailuo 非法，允许 6 或 10 秒", duration)
		}
	case "Vidu":
		if duration > 10 {
			return fmt.Errorf("OutputConfig.Duration=%v 对 Vidu 超出上限 10 秒", duration)
		}
	case "OS":
		if duration != 4 && duration != 8 && duration != 12 {
			return fmt.Errorf("OutputConfig.Duration=%v 对 OS 非法，允许 4/8/12 秒", duration)
		}
	}
	return nil
}

func outputConfigFromTaskSubmitReq(req relaycommon.TaskSubmitReq) (*AigcVideoOutputConfig, error) {
	oc := &AigcVideoOutputConfig{StorageMode: "Temporary"}
	if req.Duration > 0 {
		oc.Duration = float64(req.Duration)
	} else if s := strings.TrimSpace(req.Seconds); s != "" {
		var sec int
		if _, err := fmt.Sscanf(s, "%d", &sec); err == nil && sec > 0 {
			oc.Duration = float64(sec)
		}
	}
	if v, ok := req.Metadata["duration"]; ok {
		switch x := v.(type) {
		case float64:
			if x > 0 && oc.Duration <= 0 {
				oc.Duration = x
			}
		case int:
			if x > 0 && oc.Duration <= 0 {
				oc.Duration = float64(x)
			}
		}
	}
	oc.Resolution = strings.TrimSpace(req.Resolution)
	if oc.Resolution == "" {
		if v, ok := req.Metadata["resolution"].(string); ok {
			oc.Resolution = strings.TrimSpace(v)
		}
	}
	oc.AspectRatio = strings.TrimSpace(req.Ratio)
	if oc.AspectRatio == "" {
		if v, ok := req.Metadata["ratio"].(string); ok {
			oc.AspectRatio = strings.TrimSpace(v)
		}
		if oc.AspectRatio == "" {
			if v, ok := req.Metadata["aspect_ratio"].(string); ok {
				oc.AspectRatio = strings.TrimSpace(v)
			}
		}
	}
	// 兼容 metadata.OutputConfig 直传
	if raw, ok := req.Metadata["OutputConfig"]; ok && raw != nil {
		b, err := common.Marshal(raw)
		if err == nil {
			var nested AigcVideoOutputConfig
			if err := common.Unmarshal(b, &nested); err == nil {
				if nested.Resolution != "" {
					oc.Resolution = nested.Resolution
				}
				if nested.AspectRatio != "" {
					oc.AspectRatio = nested.AspectRatio
				}
				if nested.Duration > 0 {
					oc.Duration = nested.Duration
				}
				if nested.StorageMode != "" {
					oc.StorageMode = nested.StorageMode
				}
				if nested.AudioGeneration != "" {
					oc.AudioGeneration = nested.AudioGeneration
				}
			}
		}
	}
	if strings.TrimSpace(oc.Resolution) == "" || oc.Duration <= 0 || strings.TrimSpace(oc.AspectRatio) == "" {
		return nil, fmt.Errorf("须提供 resolution/duration/ratio（或 metadata.OutputConfig），对应 OutputConfig.Resolution/Duration/AspectRatio")
	}
	applyGenerateAudioFromTaskSubmitReq(oc, req)
	return oc, nil
}
