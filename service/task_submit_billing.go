package service

import (
	"math"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
)

// ResolveActualTaskQuotaOnSubmit 在任务提交成功后，按优先级计算本次应结算额度：
// 1) 上游返回 total_tokens -> 优先按 token 结算；
// 2) 无 token 时，视频任务按上游返回真实成片元数据（时长/分辨率/音轨）结算；
// 3) 都不可用时，回退 estimatedQuota（估算值）。
func ResolveActualTaskQuotaOnSubmit(c *gin.Context, info *relaycommon.RelayInfo, taskData []byte, estimatedQuota int) int {
	if info == nil {
		return estimatedQuota
	}
	if totalTokens := extractTotalTokensFromTaskData(taskData); totalTokens > 0 {
		if quota := calcQuotaByUpstreamTokens(info, totalTokens); quota > 0 {
			return quota
		}
	}
	if constant.IsVideoTaskChannel(info.ChannelType) {
		if quota := calcVideoPerSecondQuotaByTaskData(c, info, taskData); quota > 0 {
			return quota
		}
	}
	return estimatedQuota
}

func calcQuotaByUpstreamTokens(info *relaycommon.RelayInfo, totalTokens int) int {
	if info == nil || totalTokens <= 0 {
		return 0
	}
	modelRatio := info.PriceData.ModelRatio
	if modelRatio <= 0 {
		return 0
	}
	groupRatio := info.PriceData.GroupRatioInfo.GroupRatio
	if groupRatio <= 0 {
		groupRatio = 1
	}
	rawQuota := int(math.Round(float64(totalTokens) * modelRatio * groupRatio))
	return model.ApplyChannelPriceDiscountToQuota(rawQuota, model.ResolveChannelPriceDiscountPercent(info.ChannelId))
}

func calcVideoPerSecondQuotaByTaskData(c *gin.Context, info *relaycommon.RelayInfo, taskData []byte) int {
	if info == nil || len(taskData) == 0 {
		return 0
	}
	meta, ok := extractVideoMetadataFromTaskDataBytes(taskData)
	if !ok {
		return 0
	}
	modelName := strings.TrimSpace(info.OriginModelName)
	if modelName == "" {
		return 0
	}
	rules, ok := ratio_setting.GetChannelVideoPricingRules(info.ChannelId, modelName)
	if !ok || !ratio_setting.HasUsableVideoPerSecondRules(rules) {
		var globalOK bool
		rules, globalOK = ratio_setting.GetVideoPricingRules(modelName)
		if !globalOK || !ratio_setting.HasUsableVideoPerSecondRules(rules) {
			return 0
		}
	}
	mode := detectVideoBillingModeFromSubmitRequest(c)
	pricePerSec, ok := matchPerSecondPrice(rules, mode, meta.Width, meta.Height, meta.HasAudio)
	if !ok || pricePerSec <= 0 {
		return 0
	}
	seconds := int(math.Ceil(meta.DurationSec))
	if seconds <= 0 {
		return 0
	}
	groupRatio := info.PriceData.GroupRatioInfo.GroupRatio
	if groupRatio <= 0 {
		groupRatio = 1
	}
	rawQuota := float64(seconds) * pricePerSec * common.QuotaPerUnit * groupRatio
	quota := model.ApplyChannelPriceDiscountToQuota(int(math.Round(rawQuota)), model.ResolveChannelPriceDiscountPercent(info.ChannelId))
	if quota <= 0 && rawQuota > 0 {
		return 1
	}
	return quota
}

func detectVideoBillingModeFromSubmitRequest(c *gin.Context) string {
	if c == nil {
		return "text_to_video"
	}
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return "text_to_video"
	}
	if strings.TrimSpace(req.InputReference) != "" {
		return "video_to_video"
	}
	if strings.TrimSpace(req.Image) != "" || len(req.Images) > 0 {
		return "image_to_video"
	}
	return "text_to_video"
}

func extractTotalTokensFromTaskData(taskData []byte) int {
	if len(taskData) == 0 {
		return 0
	}
	var payload any
	if err := common.Unmarshal(taskData, &payload); err != nil {
		return 0
	}
	return findTokenCount(payload)
}

func findTokenCount(node any) int {
	switch v := node.(type) {
	case map[string]any:
		for k, raw := range v {
			lk := strings.ToLower(strings.TrimSpace(k))
			if lk == "totaltokens" || lk == "total_tokens" {
				if n := submitToInt(raw); n > 0 {
					return n
				}
			}
		}
		for _, child := range v {
			if n := findTokenCount(child); n > 0 {
				return n
			}
		}
	case []any:
		for _, child := range v {
			if n := findTokenCount(child); n > 0 {
				return n
			}
		}
	}
	return 0
}

func extractVideoMetadataFromTaskDataBytes(taskData []byte) (*VideoMetadata, bool) {
	if len(taskData) == 0 {
		return nil, false
	}
	var payload map[string]any
	if err := common.Unmarshal(taskData, &payload); err != nil {
		return nil, false
	}
	response, _ := payload["Response"].(map[string]any)
	if response == nil {
		return nil, false
	}
	aigcVideoTask, _ := response["AigcVideoTask"].(map[string]any)
	if aigcVideoTask == nil {
		return nil, false
	}
	output, _ := aigcVideoTask["Output"].(map[string]any)
	if output == nil {
		return nil, false
	}
	fileInfos, _ := output["FileInfos"].([]any)
	if len(fileInfos) == 0 {
		return nil, false
	}
	firstFile, _ := fileInfos[0].(map[string]any)
	if firstFile == nil {
		return nil, false
	}
	metaMap, _ := firstFile["MetaData"].(map[string]any)
	if metaMap == nil {
		return nil, false
	}

	duration := submitToFloat64(metaMap["Duration"])
	if duration <= 0 {
		duration = submitToFloat64(metaMap["VideoDuration"])
	}
	width := submitToInt(metaMap["Width"])
	height := submitToInt(metaMap["Height"])
	audioDuration := submitToFloat64(metaMap["AudioDuration"])
	hasAudio := audioDuration > 0
	if !hasAudio {
		if audioStreams, ok := metaMap["AudioStreamSet"].([]any); ok && len(audioStreams) > 0 {
			hasAudio = true
		}
	}
	if duration <= 0 || width <= 0 || height <= 0 {
		return nil, false
	}
	return &VideoMetadata{
		DurationSec: duration,
		Width:       width,
		Height:      height,
		HasAudio:    hasAudio,
	}, true
}

func submitToFloat64(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case int:
		return float64(x)
	case int64:
		return float64(x)
	case int32:
		return float64(x)
	case uint:
		return float64(x)
	case uint64:
		return float64(x)
	case uint32:
		return float64(x)
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
		if err == nil {
			return f
		}
	}
	return 0
}

func submitToInt(v any) int {
	switch x := v.(type) {
	case int:
		return x
	case int64:
		return int(x)
	case int32:
		return int(x)
	case uint:
		return int(x)
	case uint64:
		return int(x)
	case uint32:
		return int(x)
	case float64:
		return int(x)
	case float32:
		return int(x)
	case string:
		i, err := strconv.Atoi(strings.TrimSpace(x))
		if err == nil {
			return i
		}
	}
	return 0
}
