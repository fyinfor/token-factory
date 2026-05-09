package ratio_setting

import (
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"
)

type VideoResolutionPriceRule struct {
	Resolution       string  `json:"resolution"`
	TokenPrice       float64 `json:"token_price"`
	PixelCompression float64 `json:"pixel_compression"`
}

// VideoResolutionPerVideoRule is fixed USD per completed video for a resolution
// tier (same monetary unit as VideoPrice / ModelPrice: dollars per job).
type VideoResolutionPerVideoRule struct {
	Resolution string  `json:"resolution"`
	VideoPrice float64 `json:"video_price"`
}

// VideoResolutionAudioPriceRule represents price by resolution + audio flag.
// Price semantics are USD-based internal unit (same as ModelPrice/VideoPrice).
type VideoResolutionAudioPriceRule struct {
	Resolution string  `json:"resolution"`
	HasAudio   bool    `json:"has_audio"`
	Price      float64 `json:"price"`
}

type VideoImagePriceRule struct {
	TokenPrice       float64 `json:"token_price"`
	PixelCompression float64 `json:"pixel_compression"`
}

type VideoPricingRules struct {
	TextToVideo         []VideoResolutionPriceRule `json:"text_to_video,omitempty"`
	ImageToVideo        *VideoImagePriceRule       `json:"image_to_video,omitempty"`
	ImageToVideoRules   []VideoResolutionPriceRule `json:"image_to_video_rules,omitempty"`
	VideoToVideo        []VideoResolutionPriceRule `json:"video_to_video,omitempty"`
	VideoToVideoInput   []VideoResolutionPriceRule `json:"video_to_video_input,omitempty"`
	VideoToVideoOutput  []VideoResolutionPriceRule `json:"video_to_video_output,omitempty"`
	SimilarityThreshold float64                    `json:"similarity_threshold,omitempty"`
	// Per-video (flat $ per output) by resolution; same dollar semantics as VideoPrice.
	TextToVideoPerVideo        []VideoResolutionPerVideoRule `json:"text_to_video_per_video,omitempty"`
	ImageToVideoPerVideo       []VideoResolutionPerVideoRule `json:"image_to_video_per_video,omitempty"`
	VideoToVideoInputPerVideo  []VideoResolutionPerVideoRule `json:"video_to_video_input_per_video,omitempty"`
	VideoToVideoOutputPerVideo []VideoResolutionPerVideoRule `json:"video_to_video_output_per_video,omitempty"`
	// New billing tables:
	// - *_per_second: by ceil(seconds) × unit price
	// - *_per_item: by generated video count
	TextToVideoPerSecond  []VideoResolutionAudioPriceRule `json:"text_to_video_per_second,omitempty"`
	ImageToVideoPerSecond []VideoResolutionAudioPriceRule `json:"image_to_video_per_second,omitempty"`
	VideoToVideoPerSecond []VideoResolutionAudioPriceRule `json:"video_to_video_per_second,omitempty"`
	TextToVideoPerItem    []VideoResolutionAudioPriceRule `json:"text_to_video_per_item,omitempty"`
	ImageToVideoPerItem   []VideoResolutionAudioPriceRule `json:"image_to_video_per_item,omitempty"`
	VideoToVideoPerItem   []VideoResolutionAudioPriceRule `json:"video_to_video_per_item,omitempty"`
}

var videoPricingRulesMap = types.NewRWMap[string, VideoPricingRules]()
var channelVideoPricingRulesMap = types.NewRWMap[string, map[string]VideoPricingRules]()

func normalizeVideoRules(v VideoPricingRules) VideoPricingRules {
	if v.SimilarityThreshold <= 0 {
		v.SimilarityThreshold = 0.35
	}
	for i := range v.TextToVideo {
		if v.TextToVideo[i].PixelCompression <= 0 {
			v.TextToVideo[i].PixelCompression = 1024
		}
	}
	for i := range v.VideoToVideo {
		if v.VideoToVideo[i].PixelCompression <= 0 {
			v.VideoToVideo[i].PixelCompression = 1024
		}
	}
	for i := range v.ImageToVideoRules {
		if v.ImageToVideoRules[i].PixelCompression <= 0 {
			v.ImageToVideoRules[i].PixelCompression = 1024
		}
	}
	for i := range v.VideoToVideoInput {
		if v.VideoToVideoInput[i].PixelCompression <= 0 {
			v.VideoToVideoInput[i].PixelCompression = 1024
		}
	}
	for i := range v.VideoToVideoOutput {
		if v.VideoToVideoOutput[i].PixelCompression <= 0 {
			v.VideoToVideoOutput[i].PixelCompression = 1024
		}
	}
	if v.ImageToVideo != nil && v.ImageToVideo.PixelCompression <= 0 {
		v.ImageToVideo.PixelCompression = 1024
	}
	for i := range v.TextToVideoPerVideo {
		v.TextToVideoPerVideo[i].Resolution = strings.TrimSpace(v.TextToVideoPerVideo[i].Resolution)
	}
	for i := range v.ImageToVideoPerVideo {
		v.ImageToVideoPerVideo[i].Resolution = strings.TrimSpace(v.ImageToVideoPerVideo[i].Resolution)
	}
	for i := range v.VideoToVideoInputPerVideo {
		v.VideoToVideoInputPerVideo[i].Resolution = strings.TrimSpace(v.VideoToVideoInputPerVideo[i].Resolution)
	}
	for i := range v.VideoToVideoOutputPerVideo {
		v.VideoToVideoOutputPerVideo[i].Resolution = strings.TrimSpace(v.VideoToVideoOutputPerVideo[i].Resolution)
	}
	for i := range v.TextToVideoPerSecond {
		v.TextToVideoPerSecond[i].Resolution = strings.TrimSpace(v.TextToVideoPerSecond[i].Resolution)
	}
	for i := range v.ImageToVideoPerSecond {
		v.ImageToVideoPerSecond[i].Resolution = strings.TrimSpace(v.ImageToVideoPerSecond[i].Resolution)
	}
	for i := range v.VideoToVideoPerSecond {
		v.VideoToVideoPerSecond[i].Resolution = strings.TrimSpace(v.VideoToVideoPerSecond[i].Resolution)
	}
	for i := range v.TextToVideoPerItem {
		v.TextToVideoPerItem[i].Resolution = strings.TrimSpace(v.TextToVideoPerItem[i].Resolution)
	}
	for i := range v.ImageToVideoPerItem {
		v.ImageToVideoPerItem[i].Resolution = strings.TrimSpace(v.ImageToVideoPerItem[i].Resolution)
	}
	for i := range v.VideoToVideoPerItem {
		v.VideoToVideoPerItem[i].Resolution = strings.TrimSpace(v.VideoToVideoPerItem[i].Resolution)
	}
	return v
}

// HasUsableVideoPerVideoRules reports whether any per-resolution flat video price tier exists
// with a positive video_price (USD per completed video, same unit as VideoPrice).
func HasUsableVideoPerVideoRules(v VideoPricingRules) bool {
	for _, r := range v.TextToVideoPerItem {
		if r.Price > 0 {
			return true
		}
	}
	for _, r := range v.ImageToVideoPerItem {
		if r.Price > 0 {
			return true
		}
	}
	for _, r := range v.VideoToVideoPerItem {
		if r.Price > 0 {
			return true
		}
	}
	for _, r := range v.TextToVideoPerVideo {
		if r.VideoPrice > 0 {
			return true
		}
	}
	for _, r := range v.ImageToVideoPerVideo {
		if r.VideoPrice > 0 {
			return true
		}
	}
	for _, r := range v.VideoToVideoInputPerVideo {
		if r.VideoPrice > 0 {
			return true
		}
	}
	for _, r := range v.VideoToVideoOutputPerVideo {
		if r.VideoPrice > 0 {
			return true
		}
	}
	return false
}

func HasUsableVideoPerSecondRules(v VideoPricingRules) bool {
	for _, r := range v.TextToVideoPerSecond {
		if r.Price > 0 {
			return true
		}
	}
	for _, r := range v.ImageToVideoPerSecond {
		if r.Price > 0 {
			return true
		}
	}
	for _, r := range v.VideoToVideoPerSecond {
		if r.Price > 0 {
			return true
		}
	}
	return false
}

func normalizeVideoRulesMap(src map[string]VideoPricingRules) map[string]VideoPricingRules {
	dst := make(map[string]VideoPricingRules, len(src))
	for model, rules := range src {
		name := FormatMatchingModelName(strings.TrimSpace(model))
		if name == "" {
			continue
		}
		dst[name] = normalizeVideoRules(rules)
	}
	return dst
}

func UpdateVideoPricingRulesByJSONString(jsonStr string) error {
	trimmed := strings.TrimSpace(jsonStr)
	if trimmed == "" {
		videoPricingRulesMap.Clear()
		return nil
	}
	var parsed map[string]VideoPricingRules
	if err := common.UnmarshalJsonStr(trimmed, &parsed); err != nil {
		return err
	}
	videoPricingRulesMap.Clear()
	videoPricingRulesMap.AddAll(normalizeVideoRulesMap(parsed))
	InvalidateExposedDataCache()
	return nil
}

func VideoPricingRules2JSONString() string {
	jsonBytes, err := common.Marshal(videoPricingRulesMap.ReadAll())
	if err != nil {
		common.SysError("error marshalling video pricing rules: " + err.Error())
		return "{}"
	}
	return string(jsonBytes)
}

func GetVideoPricingRules(modelName string) (VideoPricingRules, bool) {
	name := FormatMatchingModelName(modelName)
	rules, ok := videoPricingRulesMap.Get(name)
	return rules, ok
}

func UpdateChannelVideoPricingRulesByJSONString(jsonStr string) error {
	trimmed := strings.TrimSpace(jsonStr)
	if trimmed == "" {
		channelVideoPricingRulesMap.Clear()
		return nil
	}
	var parsed map[string]map[string]VideoPricingRules
	if err := common.UnmarshalJsonStr(trimmed, &parsed); err != nil {
		return err
	}
	normalized := make(map[string]map[string]VideoPricingRules, len(parsed))
	for channelID, modelRules := range parsed {
		if _, err := strconv.Atoi(channelID); err != nil {
			continue
		}
		normalized[channelID] = normalizeVideoRulesMap(modelRules)
	}
	channelVideoPricingRulesMap.Clear()
	channelVideoPricingRulesMap.AddAll(normalized)
	return nil
}

func ChannelVideoPricingRules2JSONString() string {
	jsonBytes, err := common.Marshal(channelVideoPricingRulesMap.ReadAll())
	if err != nil {
		common.SysError("error marshalling channel video pricing rules: " + err.Error())
		return "{}"
	}
	return string(jsonBytes)
}

func GetChannelVideoPricingRules(channelID int, modelName string) (VideoPricingRules, bool) {
	key := normalizeChannelID(channelID)
	if key == "" {
		return VideoPricingRules{}, false
	}
	channelMap, ok := channelVideoPricingRulesMap.Get(key)
	if !ok {
		return VideoPricingRules{}, false
	}
	rules, ok := channelMap[FormatMatchingModelName(modelName)]
	return rules, ok
}
