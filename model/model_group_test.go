package model

import "testing"

func TestNormalizeModelName(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"deepseek-v4-flash-0221", "deepseek-v4-flash"},
		{"deepseek-v4-flash", "deepseek-v4-flash"},
		{"claude-3-5-sonnet-20241022", "claude-3-5-sonnet"},
		{"gpt-4o-mini", "gpt-4o-mini"},
		{"gpt-4o", "gpt-4o"},
		{"some-model-latest", "some-model"},
		{"", ""},
		{"  GPT-4o-MINI  ", "gpt-4o-mini"},
	}
	for _, tc := range cases {
		if got := NormalizeModelName(tc.in); got != tc.want {
			t.Errorf("NormalizeModelName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestResolveModelGroupKey(t *testing.T) {
	overrides := map[string]string{
		"claude-3-5-sonnet-20241022": "claude-sonnet",
	}
	if got := ResolveModelGroupKey("claude-3-5-sonnet-20241022", overrides); got != "claude-sonnet" {
		t.Fatalf("override not applied: %q", got)
	}
	if got := ResolveModelGroupKey("deepseek-v4-flash-0221", overrides); got != "deepseek-v4-flash" {
		t.Fatalf("normalize fallback failed: %q", got)
	}
}
