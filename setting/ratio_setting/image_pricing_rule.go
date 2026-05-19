package ratio_setting

import (
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"
)

// ImageResolutionPerImageRule is fixed USD per generated image for a resolution tier
// (same monetary unit as ImagePrice / ModelPrice: dollars per image).
type ImageResolutionPerImageRule struct {
	Resolution string  `json:"resolution"`
	ImagePrice float64 `json:"image_price"`
}

type ImagePricingRules struct {
	TextToImagePerImage  []ImageResolutionPerImageRule `json:"text_to_image_per_image,omitempty"`
	ImageToImagePerImage []ImageResolutionPerImageRule `json:"image_to_image_per_image,omitempty"`
	SimilarityThreshold  float64                       `json:"similarity_threshold,omitempty"`
	PriceUnit            string                        `json:"price_unit,omitempty"`
}

var imagePricingRulesMap = types.NewRWMap[string, ImagePricingRules]()
var channelImagePricingRulesMap = types.NewRWMap[string, map[string]ImagePricingRules]()

func normalizeImageRules(v ImagePricingRules) ImagePricingRules {
	if v.SimilarityThreshold <= 0 {
		v.SimilarityThreshold = 0.35
	}
	for i := range v.TextToImagePerImage {
		v.TextToImagePerImage[i].Resolution = strings.TrimSpace(v.TextToImagePerImage[i].Resolution)
	}
	for i := range v.ImageToImagePerImage {
		v.ImageToImagePerImage[i].Resolution = strings.TrimSpace(v.ImageToImagePerImage[i].Resolution)
	}
	return v
}

func HasUsableImagePerImageRules(v ImagePricingRules) bool {
	for _, r := range v.TextToImagePerImage {
		if r.ImagePrice > 0 {
			return true
		}
	}
	for _, r := range v.ImageToImagePerImage {
		if r.ImagePrice > 0 {
			return true
		}
	}
	return false
}

func normalizeImageRulesMap(src map[string]ImagePricingRules) map[string]ImagePricingRules {
	dst := make(map[string]ImagePricingRules, len(src))
	for model, rules := range src {
		name := FormatMatchingModelName(strings.TrimSpace(model))
		if name == "" {
			continue
		}
		dst[name] = normalizeImageRules(rules)
	}
	return dst
}

func UpdateImagePricingRulesByJSONString(jsonStr string) error {
	trimmed := strings.TrimSpace(jsonStr)
	if trimmed == "" {
		imagePricingRulesMap.Clear()
		return nil
	}
	var parsed map[string]ImagePricingRules
	if err := common.UnmarshalJsonStr(trimmed, &parsed); err != nil {
		return err
	}
	imagePricingRulesMap.Clear()
	imagePricingRulesMap.AddAll(normalizeImageRulesMap(parsed))
	InvalidateExposedDataCache()
	return nil
}

func ImagePricingRules2JSONString() string {
	jsonBytes, err := common.Marshal(imagePricingRulesMap.ReadAll())
	if err != nil {
		common.SysError("error marshalling image pricing rules: " + err.Error())
		return "{}"
	}
	return string(jsonBytes)
}

func GetImagePricingRules(modelName string) (ImagePricingRules, bool) {
	name := FormatMatchingModelName(modelName)
	rules, ok := imagePricingRulesMap.Get(name)
	return rules, ok
}

func UpdateChannelImagePricingRulesByJSONString(jsonStr string) error {
	trimmed := strings.TrimSpace(jsonStr)
	if trimmed == "" {
		channelImagePricingRulesMap.Clear()
		return nil
	}
	var parsed map[string]map[string]ImagePricingRules
	if err := common.UnmarshalJsonStr(trimmed, &parsed); err != nil {
		return err
	}
	normalized := make(map[string]map[string]ImagePricingRules, len(parsed))
	for channelID, modelRules := range parsed {
		if _, err := strconv.Atoi(channelID); err != nil {
			continue
		}
		normalized[channelID] = normalizeImageRulesMap(modelRules)
	}
	channelImagePricingRulesMap.Clear()
	channelImagePricingRulesMap.AddAll(normalized)
	return nil
}

func ChannelImagePricingRules2JSONString() string {
	jsonBytes, err := common.Marshal(channelImagePricingRulesMap.ReadAll())
	if err != nil {
		common.SysError("error marshalling channel image pricing rules: " + err.Error())
		return "{}"
	}
	return string(jsonBytes)
}

func GetChannelImagePricingRules(channelID int, modelName string) (ImagePricingRules, bool) {
	key := normalizeChannelID(channelID)
	if key == "" {
		return ImagePricingRules{}, false
	}
	channelMap, ok := channelImagePricingRulesMap.Get(key)
	if !ok {
		return ImagePricingRules{}, false
	}
	rules, ok := channelMap[FormatMatchingModelName(modelName)]
	return rules, ok
}
