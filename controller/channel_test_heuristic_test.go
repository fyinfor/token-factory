package controller

import "testing"

func TestTokenFactoryOpenVideoTestHeuristic_DoubaoLLMNotVideo(t *testing.T) {
	llmModels := []string{
		"doubao-seed-2-0-code-preview-260215",
		"doubao-seed-2-0-pro-260215",
		"doubao-seed-1-6-thinking-250715",
		"doubao-seedream-4-0-250828",
	}
	for _, m := range llmModels {
		if tokenFactoryOpenVideoTestHeuristic(m) {
			t.Fatalf("model %q should not be treated as video for channel test", m)
		}
	}
}

func TestTokenFactoryOpenVideoTestHeuristic_DoubaoVideo(t *testing.T) {
	videoModels := []string{
		"doubao-seedance-1-0-pro-250528",
		"doubao-seedance-2-0-260128",
		"doubao-seedance-2-0-fast-260128",
	}
	for _, m := range videoModels {
		if !tokenFactoryOpenVideoTestHeuristic(m) {
			t.Fatalf("model %q should be treated as video for channel test", m)
		}
	}
}

func TestTokenFactoryOpenVideoTestHeuristic_MiniMaxLLMNotVideo(t *testing.T) {
	llmModels := []string{
		"MiniMax-M2.7",
		"MiniMax-Text-01",
		"abab6.5s-chat",
	}
	for _, m := range llmModels {
		if tokenFactoryOpenVideoTestHeuristic(m) {
			t.Fatalf("model %q should not be treated as video for channel test", m)
		}
	}
}

func TestTokenFactoryOpenVideoTestHeuristic_MiniMaxVideo(t *testing.T) {
	videoModels := []string{
		"MiniMax-Hailuo-2.3",
		"MiniMax-Hailuo-2.3-Fast",
		"MiniMax-Hailuo-02",
	}
	for _, m := range videoModels {
		if !tokenFactoryOpenVideoTestHeuristic(m) {
			t.Fatalf("model %q should be treated as video for channel test", m)
		}
	}
}

func TestTokenFactoryOpenVideoTestHeuristic_OtherVideoFamilies(t *testing.T) {
	cases := map[string]bool{
		"sora-2":              true,
		"kling-v1":            true,
		"Video-abc123":        true,
		"wanx2.1-i2v-plus":    true,
		"gpt-4o-mini":         false,
		"claude-3-5-sonnet":   false,
		"MiniMax-M2.7":        false,
	}
	for model, want := range cases {
		got := tokenFactoryOpenVideoTestHeuristic(model)
		if got != want {
			t.Fatalf("model %q: got video=%v want %v", model, got, want)
		}
	}
}
