package common

import "testing"

func TestModelCategory(t *testing.T) {
	cases := []struct {
		name     string
		model    string
		expected string
	}{
		// Seedance 应归入 video（与系统其他位置保持一致：playground、供应商能力）
		{"seedance-lite", "doubao-seedance-1.0-lite", RankingCategoryVideo},
		{"seedance-pro", "seedance-1.0-pro", RankingCategoryVideo},
		{"seedance-mixed-case", "Doubao-Seedance-pro-250528", RankingCategoryVideo},

		// 通用 video
		{"kling", "kling-v1-5", RankingCategoryVideo},
		{"vidu", "vidu-1.0", RankingCategoryVideo},
		{"sora", "sora-1.0-turbo", RankingCategoryVideo},
		{"hunyuan-video", "hunyuan-video-1.5", RankingCategoryVideo},
		{"wan-video", "wan-video-1.0", RankingCategoryVideo},
		{"wan2", "wan2-t2v-1.0", RankingCategoryVideo},
		{"ali-video", "ali-video-turbo", RankingCategoryVideo},
		{"tencentcloud-vod-video", "tencentcloud-vod-video-1", RankingCategoryVideo},
		{"openai-video", "openai-video-sora-1", RankingCategoryVideo},
		{"video-01", "kling-video-01", RankingCategoryVideo},
		{"video-02", "kling-video-02", RankingCategoryVideo},
		{"video-generation", "myvendor-video-generation-1", RankingCategoryVideo},

		// 文生图 image
		{"dall-e-3", "dall-e-3", RankingCategoryImage},
		{"dall-e-2", "dall-e-2", RankingCategoryImage},
		{"gpt-image", "gpt-image-1", RankingCategoryImage},
		{"imagen", "imagen-3.0", RankingCategoryImage},
		{"flux", "flux-pro-1.0", RankingCategoryImage},
		{"sdxl", "sdxl-1.0", RankingCategoryImage},
		{"stable-diffusion", "stable-diffusion-xl", RankingCategoryImage},
		{"wanx", "wanx-v1", RankingCategoryImage},
		{"kolors", "kolors-1.0", RankingCategoryImage},
		{"cogview", "cogview-3", RankingCategoryImage},
		{"hunyuan-dit", "hunyuan-dit-1.2", RankingCategoryImage},
		{"image-alpha", "vendor-image-alpha-v1", RankingCategoryImage},

		// 文本对话 text（默认）
		{"gpt-4o", "gpt-4o", RankingCategoryText},
		{"claude", "claude-3.5-sonnet", RankingCategoryText},
		{"deepseek", "deepseek-v3", RankingCategoryText},
		{"qwen", "qwen-2.5-72b", RankingCategoryText},
		{"unknown", "totally-custom-model", RankingCategoryText},

		// 边界：Seedance 不会被识别为 image（即使包含 "image" 子串）
		{"seedance-not-image", "seedance-image-alpha-test", RankingCategoryVideo},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ModelCategory(tc.model)
			if got != tc.expected {
				t.Errorf("ModelCategory(%q) = %q, want %q", tc.model, got, tc.expected)
			}
		})
	}
}

func TestIsRankingCategorySupported(t *testing.T) {
	supported := []string{"", "all", "text", "image", "video"}
	for _, c := range supported {
		if !IsRankingCategorySupported(c) {
			t.Errorf("IsRankingCategorySupported(%q) = false, want true", c)
		}
	}
	for _, c := range []string{"foo", "audio", "t2i", "t2v", "seedance", "chat"} {
		if IsRankingCategorySupported(c) {
			t.Errorf("IsRankingCategorySupported(%q) = true, want false", c)
		}
	}
}

func TestSeedanceIsClassifiedAsVideo(t *testing.T) {
	// Seedance 必须是 video 而非 image/text。
	if ModelCategory("doubao-seedance-lite") != RankingCategoryVideo {
		t.Error("seedance should be classified as video")
	}
	if !IsVideoGenerationModel("doubao-seedance-lite") {
		t.Error("seedance should match IsVideoGenerationModel")
	}
	if IsTextToImageModel("kling-v1") {
		t.Error("kling should not be classified as image")
	}
}
