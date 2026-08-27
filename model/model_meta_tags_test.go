package model

import "testing"

func TestResolveModelTagsFromRows(t *testing.T) {
	rows := []Model{
		{ModelName: "gpt-4o", Tags: "文本", NameRule: NameRuleExact},
		{ModelName: "Kling", Tags: "视频", NameRule: NameRulePrefix},
		{ModelName: "mini", Tags: "文本", NameRule: NameRuleSuffix},
	}

	tests := []struct {
		model string
		want  string
	}{
		{model: "gpt-4o", want: "文本"},
		{model: "Kling-3.0", want: "视频"},
		{model: "gpt-4o-mini", want: "文本"},
		{model: "unknown-model", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			if got := resolveModelTagsFromRows(tt.model, rows); got != tt.want {
				t.Fatalf("resolveModelTagsFromRows(%q) = %q, want %q", tt.model, got, tt.want)
			}
		})
	}
}

func TestResolveExactModelTagsFromRowsRejectsNameRuleFallback(t *testing.T) {
	rows := []Model{
		{ModelName: "gpt-4o", Tags: "文本", NameRule: NameRuleExact},
		{ModelName: "Kling", Tags: "视频", NameRule: NameRulePrefix},
		{ModelName: "mini", Tags: "文本", NameRule: NameRuleSuffix},
		{ModelName: "qwen", Tags: "文本", NameRule: NameRuleContains},
	}

	tests := []struct {
		model string
		want  string
	}{
		{model: "gpt-4o", want: "文本"},
		{model: " gpt-4o ", want: "文本"},
		{model: "GPT-4o", want: ""},
		{model: "Kling-3.0", want: ""},
		{model: "Kling", want: "视频"},
		{model: "gpt-4o-mini", want: ""},
		{model: "qwen-plus", want: ""},
		{model: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			if got := resolveExactModelTagsFromRows(tt.model, rows); got != tt.want {
				t.Fatalf("resolveExactModelTagsFromRows(%q) = %q, want %q", tt.model, got, tt.want)
			}
		})
	}
}
