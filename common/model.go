package common

import "strings"

var (
	// OpenAIResponseOnlyModels is a list of models that are only available for OpenAI responses.
	OpenAIResponseOnlyModels = []string{
		"o3-pro",
		"o3-deep-research",
		"o4-mini-deep-research",
	}
	ImageGenerationModels = []string{
		"dall-e-3",
		"dall-e-2",
		"gpt-image-1",
		"prefix:imagen-",
		"flux-",
		"flux.1-",
	}
	OpenAITextModels = []string{
		"gpt-",
		"o1",
		"o3",
		"o4",
		"chatgpt",
	}
)

func IsOpenAIResponseOnlyModel(modelName string) bool {
	for _, m := range OpenAIResponseOnlyModels {
		if strings.Contains(modelName, m) {
			return true
		}
	}
	return false
}

func IsImageGenerationModel(modelName string) bool {
	modelName = strings.ToLower(modelName)
	for _, m := range ImageGenerationModels {
		if strings.Contains(modelName, m) {
			return true
		}
		if strings.HasPrefix(m, "prefix:") && strings.HasPrefix(modelName, strings.TrimPrefix(m, "prefix:")) {
			return true
		}
	}
	return false
}

func IsOpenAITextModel(modelName string) bool {
	modelName = strings.ToLower(modelName)
	for _, m := range OpenAITextModels {
		if strings.Contains(modelName, m) {
			return true
		}
	}
	return false
}

// ModelTagsContain 判断 models.tags 逗号列表是否包含指定标签（去空格后精确匹配）。
func ModelTagsContain(tags string, keyword string) bool {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return false
	}
	for _, t := range strings.Split(tags, ",") {
		if strings.TrimSpace(t) == keyword {
			return true
		}
	}
	return false
}

// ModelTagsIndicateVideoPricing 模型带「视频」标签时，定价卡片走视频展示口径。
func ModelTagsIndicateVideoPricing(tags string) bool {
	return ModelTagsContain(tags, "视频")
}

// ModelTagsIndicateImagePricing 模型带「图片」标签时，定价卡片走图片展示口径。
func ModelTagsIndicateImagePricing(tags string) bool {
	return ModelTagsContain(tags, "图片")
}
