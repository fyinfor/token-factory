package common

import "testing"

func TestModelCategory(t *testing.T) {
	cases := []struct {
		name     string
		model    string
		expected string
	}{
		// Seedance should win over generic video
		{"seedance-lite", "doubao-seedance-1.0-lite", RankingCategorySeedance},
		{"seedance-pro", "seedance-1.0-pro", RankingCategorySeedance},
		{"seedance-mixed-case", "Doubao-Seedance-pro-250528", RankingCategorySeedance},

		// Generic video
		{"kling", "kling-v1-5", RankingCategoryT2V},
		{"vidu", "vidu-1.0", RankingCategoryT2V},
		{"sora", "sora-1.0-turbo", RankingCategoryT2V},
		{"hunyuan-video", "hunyuan-video-1.5", RankingCategoryT2V},
		{"wan-video", "wan-video-1.0", RankingCategoryT2V},
		{"wan2", "wan2-t2v-1.0", RankingCategoryT2V},
		{"ali-video", "ali-video-turbo", RankingCategoryT2V},
		{"tencentcloud-vod-video", "tencentcloud-vod-video-1", RankingCategoryT2V},
		{"openai-video", "openai-video-sora-1", RankingCategoryT2V},
		{"video-01", "kling-video-01", RankingCategoryT2V},
		{"video-02", "kling-video-02", RankingCategoryT2V},
		{"video-generation", "myvendor-video-generation-1", RankingCategoryT2V},

		// Text-to-image
		{"dall-e-3", "dall-e-3", RankingCategoryT2I},
		{"dall-e-2", "dall-e-2", RankingCategoryT2I},
		{"gpt-image", "gpt-image-1", RankingCategoryT2I},
		{"imagen", "imagen-3.0", RankingCategoryT2I},
		{"flux", "flux-pro-1.0", RankingCategoryT2I},
		{"sdxl", "sdxl-1.0", RankingCategoryT2I},
		{"stable-diffusion", "stable-diffusion-xl", RankingCategoryT2I},
		{"wanx", "wanx-v1", RankingCategoryT2I},
		{"kolors", "kolors-1.0", RankingCategoryT2I},
		{"cogview", "cogview-3", RankingCategoryT2I},
		{"hunyuan-dit", "hunyuan-dit-1.2", RankingCategoryT2I},
		{"image-alpha", "vendor-image-alpha-v1", RankingCategoryT2I},

		// Chat (default)
		{"gpt-4o", "gpt-4o", RankingCategoryChat},
		{"claude", "claude-3.5-sonnet", RankingCategoryChat},
		{"deepseek", "deepseek-v3", RankingCategoryChat},
		{"qwen", "qwen-2.5-72b", RankingCategoryChat},
		{"unknown", "totally-custom-model", RankingCategoryChat},

		// Edge: Seedance should NOT be classified as image even if it contained "image-alpha"
		{"seedance-not-image", "seedance-image-alpha-test", RankingCategorySeedance},
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
	supported := []string{"", "all", "t2i", "t2v", "seedance"}
	for _, c := range supported {
		if !IsRankingCategorySupported(c) {
			t.Errorf("IsRankingCategorySupported(%q) = false, want true", c)
		}
	}
	if IsRankingCategorySupported("foo") {
		t.Error("IsRankingCategorySupported(\"foo\") = true, want false")
	}
	if IsRankingCategorySupported("audio") {
		t.Error("IsRankingCategorySupported(\"audio\") = true, want false (not in current set)")
	}
	// "chat" 是内部默认分类，不作为前端可选项
	if IsRankingCategorySupported("chat") {
		t.Error("IsRankingCategorySupported(\"chat\") = true, want false (chat is the implicit default, not a filter)")
	}
}

func TestIsSeedanceModel_DoesNotMatchImage(t *testing.T) {
	// Sanity: ensure seedance/video/image helpers don't bleed into each other.
	if !IsSeedanceModel("doubao-seedance-lite") {
		t.Error("expected seedance model to match IsSeedanceModel")
	}
	if IsImageGenerationModel("doubao-seedance-lite") {
		t.Error("seedance should not be classified as image")
	}
	if IsTextToImageModel("kling-v1") {
		t.Error("kling should not be classified as t2i")
	}
}
