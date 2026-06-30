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
	// TextToImageModels 文生图（T2I）模型：模型名包含这些子串/前缀即视为文生图。
	TextToImageModels = []string{
		"dall-e",
		"gpt-image",
		"prefix:imagen-",
		"prefix:flux-",
		"prefix:flux.",
		"prefix:flux_",
		"sdxl",
		"stable-diffusion",
		"prefix:wanx-",
		"prefix:kolors-",
		"prefix:cogview-",
		"prefix:hunyuan-dit-",
		"image-alpha",
		"prefix:t2i-",
	}
	// VideoGenerationModels 通用文生视频（T2V）/ 图生视频（I2V）模型名关键词。
	// Seedance 等专用视频模型也归入此列表。
	VideoGenerationModels = []string{
		"prefix:kling-",
		"prefix:kling_",
		"prefix:vidu-",
		"prefix:sora-",
		"prefix:openai-video-",
		"prefix:hidream-video-",
		"prefix:tokenfactory-video-",
		"prefix:tencentcloud-vod-video-",
		"prefix:ali-video-",
		"prefix:seedance-",
		"prefix:seedance.",
		"prefix:seedance_",
		"prefix:doubao-seedance-",
		"prefix:doubao-seedance.",
		// 兼容 OpenAI-Video 适配器下 Seedance 2.0 的可读别名（playground-facing）。
		// 案例：Seedance2.0、Seedance2.0-fast。模型名已统一小写后做子串匹配。
		"seedance2",
		// 兼容 OpenAI-Video 网关上 Seedance/Doubao 系列返回的「Video-xxx」opaque ID。
		"prefix:video-",
		"hunyuan-video",
		"prefix:cogvideox-",
		"prefix:wan2-",
		"prefix:wan-video-",
		"video-01",
		"video-02",
		"video-generation",
		"prefix:t2v-",
		"prefix:i2v-",
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

// matchModelKeyword 复用 ImageGenerationModels 的子串/前缀匹配规则。
func matchModelKeyword(modelName string, list []string) bool {
	modelName = strings.ToLower(modelName)
	for _, m := range list {
		if m == "" {
			continue
		}
		if strings.HasPrefix(m, "prefix:") {
			if strings.HasPrefix(modelName, strings.TrimPrefix(m, "prefix:")) {
				return true
			}
			continue
		}
		if strings.Contains(modelName, m) {
			return true
		}
	}
	return false
}

// IsTextToImageModel 判断模型名是否属于文生图（T2I）。
func IsTextToImageModel(modelName string) bool {
	return matchModelKeyword(modelName, TextToImageModels)
}

// IsVideoGenerationModel 判断模型名是否属于视频生成（T2V / I2V / 通用 video）。
// 文生图/通用 image 模型不会被识别为视频生成。
func IsVideoGenerationModel(modelName string) bool {
	return matchModelKeyword(modelName, VideoGenerationModels)
}

// RankingCategory 排行分类常量。值与系统内其他位置（playground、供应商能力）保持一致：
// text / image / video，加上 all 表示不过滤。
const (
	RankingCategoryAll   = "all"
	RankingCategoryText  = "text"  // 文本对话模型（GPT/Claude/DeepSeek/Qwen 等）
	RankingCategoryImage = "image" // 文生图 / 图生图
	RankingCategoryVideo = "video" // 文生视频 / 图生视频（包含 Seedance 等专用视频模型）
)

// ModelCategory 统一返回模型在排行中的分类标签。
// 与 system 中其他模块（playground display_mode、供应商能力多模态选项）保持一致。
// Seedance 等专用视频模型归入 video，不单独成类。
func ModelCategory(modelName string) string {
	if IsVideoGenerationModel(modelName) {
		return RankingCategoryVideo
	}
	if IsTextToImageModel(modelName) {
		return RankingCategoryImage
	}
	return RankingCategoryText
}

// IsRankingCategorySupported 当前支持过滤的分类集合。
func IsRankingCategorySupported(category string) bool {
	switch category {
	case "", RankingCategoryAll, RankingCategoryText, RankingCategoryImage, RankingCategoryVideo:
		return true
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
