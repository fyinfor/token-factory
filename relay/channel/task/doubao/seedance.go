package doubao

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

// Seedance（火山方舟 ChannelTypeSeedance=65）专属：
// 1) 未传 prompt 时填充默认值，仅用于绕过网关校验；
// 2) 将完整原始请求 JSON 写入 context，供任务 Properties.Input 落库调试；
// 3) 从 content[] 提取参考图/视频，修正任务 action / 计费模式识别。

const seedanceDefaultPrompt = "帮我生成一个视频"

// validateSeedanceTaskRequest 仅用于 ChannelTypeSeedance。
func validateSeedanceTaskRequest(c *gin.Context, info *relaycommon.RelayInfo) *dto.TaskError {
	if taskErr := ensureSeedanceDefaultPromptAndPersistRawBody(c); taskErr != nil {
		return taskErr
	}
	if taskErr := relaycommon.ValidateBasicTaskRequest(c, info, constant.TaskActionGenerate); taskErr != nil {
		return taskErr
	}
	enrichSeedanceTaskRequestFromContent(c, info)
	return nil
}

// ensureSeedanceDefaultPromptAndPersistRawBody 补齐默认 prompt，并持久化完整原始请求体。
func ensureSeedanceDefaultPromptAndPersistRawBody(c *gin.Context) *dto.TaskError {
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return service.TaskErrorWrapperLocal(err, "read_request_body_failed", http.StatusBadRequest)
	}
	raw, err := storage.Bytes()
	if err != nil {
		return service.TaskErrorWrapperLocal(err, "read_request_body_failed", http.StatusBadRequest)
	}
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed[0] != '{' {
		// 非 JSON：不做 Seedance 专属改写，交给通用校验。
		return nil
	}

	var bodyMap map[string]any
	if err := common.Unmarshal(raw, &bodyMap); err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	if bodyMap == nil {
		bodyMap = map[string]any{}
	}

	persistBytes := raw
	if promptMissing(bodyMap["prompt"]) {
		bodyMap["prompt"] = seedanceDefaultPrompt
		filled, mErr := common.Marshal(bodyMap)
		if mErr != nil {
			return service.TaskErrorWrapperLocal(mErr, "marshal_request_failed", http.StatusInternalServerError)
		}
		if err := common.ReplaceRequestBody(c, filled); err != nil {
			return service.TaskErrorWrapperLocal(err, "replace_request_body_failed", http.StatusInternalServerError)
		}
		persistBytes = filled
		logger.LogInfo(c, fmt.Sprintf("seedance: empty prompt filled with default %q for gateway validation", seedanceDefaultPrompt))
	}

	// 落库用完整 JSON（含 content / resolution 等透传字段），便于线上排查。
	relaycommon.SetTaskPersistedInput(c, string(persistBytes))
	return nil
}

func promptMissing(v any) bool {
	switch p := v.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(p) == ""
	default:
		return strings.TrimSpace(fmt.Sprint(p)) == ""
	}
}

// enrichSeedanceTaskRequestFromContent 把 content[] 中的参考图/视频同步到 TaskSubmitReq，
// 以便 ResolveTaskActionToStore / DetectVideoBillingMode 识别正确任务类型。
func enrichSeedanceTaskRequestFromContent(c *gin.Context, info *relaycommon.RelayInfo) {
	raw, ok := relaycommon.GetTaskPersistedInput(c)
	if !ok {
		return
	}
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return
	}

	var bodyMap map[string]any
	if err := common.UnmarshalJsonStr(raw, &bodyMap); err != nil {
		return
	}
	contentRaw, ok := bodyMap["content"]
	if !ok {
		return
	}
	items, ok := contentRaw.([]any)
	if !ok || len(items) == 0 {
		return
	}

	changed := false
	if req.Metadata == nil {
		req.Metadata = make(map[string]any)
	}
	videoURLs := collectMetadataStringList(req.Metadata["video_urls"])

	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		typ, _ := m["type"].(string)
		switch strings.ToLower(strings.TrimSpace(typ)) {
		case "image_url":
			if u := mediaURLFromContentItem(m, "image_url"); u != "" && !containsStringFold(req.Images, u) {
				req.Images = append(req.Images, u)
				changed = true
			}
		case "video_url":
			if u := mediaURLFromContentItem(m, "video_url"); u != "" && !containsStringFold(videoURLs, u) {
				videoURLs = append(videoURLs, u)
				changed = true
			}
		}
	}
	if len(videoURLs) > 0 {
		req.Metadata["video_urls"] = videoURLs
	}
	if !changed && len(videoURLs) == 0 {
		return
	}

	action := relaycommon.ResolveTaskActionToStore(info, constant.TaskActionGenerate, &req)
	relaycommon.StoreTaskRequest(c, info, action, req)
	logger.LogInfo(c, fmt.Sprintf("seedance: enriched task request from content images=%d videos=%d action=%s",
		len(req.Images), len(videoURLs), action))
}

func mediaURLFromContentItem(item map[string]any, key string) string {
	raw, ok := item[key]
	if !ok || raw == nil {
		return ""
	}
	switch v := raw.(type) {
	case string:
		return strings.TrimSpace(v)
	case map[string]any:
		if u, ok := v["url"].(string); ok {
			return strings.TrimSpace(u)
		}
	}
	return ""
}

func collectMetadataStringList(v any) []string {
	switch arr := v.(type) {
	case []string:
		out := make([]string, 0, len(arr))
		for _, s := range arr {
			if t := strings.TrimSpace(s); t != "" {
				out = append(out, t)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(arr))
		for _, it := range arr {
			if s, ok := it.(string); ok {
				if t := strings.TrimSpace(s); t != "" {
					out = append(out, t)
				}
			}
		}
		return out
	case string:
		if t := strings.TrimSpace(arr); t != "" {
			return []string{t}
		}
	}
	return nil
}

func containsStringFold(list []string, target string) bool {
	target = strings.TrimSpace(target)
	if target == "" {
		return false
	}
	for _, s := range list {
		if strings.EqualFold(strings.TrimSpace(s), target) {
			return true
		}
	}
	return false
}
