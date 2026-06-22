package model

import (
	"testing"

	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

func TestNormalizeModelLimitNames(t *testing.T) {
	t.Parallel()

	got := NormalizeModelLimitNames([]string{" gpt-4 ", "gpt-4", "", "claude-3-5-sonnet"})
	if len(got) != 2 {
		t.Fatalf("expected 2 unique models, got %v", got)
	}
	if got[0] != "gpt-4" || got[1] != "claude-3-5-sonnet" {
		t.Fatalf("unexpected normalized models: %v", got)
	}
}

func TestGetModelLimitsMapIncludesFormattedKeys(t *testing.T) {
	t.Parallel()

	token := &Token{
		ModelLimits: "gpt-4-gizmo-abc123",
	}
	limitsMap := token.GetModelLimitsMap()

	rawKey := "gpt-4-gizmo-abc123"
	formattedKey := ratio_setting.FormatMatchingModelName(rawKey)
	if !limitsMap[rawKey] {
		t.Fatalf("expected raw model key %q in limits map", rawKey)
	}
	if !limitsMap[formattedKey] {
		t.Fatalf("expected formatted model key %q in limits map", formattedKey)
	}
}

func TestModelLimitMapAllowsConfiguredModel(t *testing.T) {
	t.Parallel()

	token := &Token{
		ModelLimitsEnabled: true,
		ModelLimits:        "gpt-4,gemini-2.5-flash-thinking-8192",
	}
	limitsMap := token.GetModelLimitsMap()

	cases := []struct {
		name  string
		model string
		want  bool
	}{
		{name: "exact configured model", model: "gpt-4", want: true},
		{name: "thinking budget model via formatted key", model: "gemini-2.5-flash-thinking-8192", want: true},
		{name: "unconfigured model", model: "gpt-3.5-turbo", want: false},
		{name: "empty model", model: "", want: false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := ModelLimitMapAllows(limitsMap, tc.model); got != tc.want {
				t.Fatalf("ModelLimitMapAllows(%q) = %v, want %v", tc.model, got, tc.want)
			}
		})
	}
}

func TestModelLimitMapAllowsBlocksUnconfiguredModel(t *testing.T) {
	t.Parallel()

	token := &Token{
		ModelLimitsEnabled: true,
		ModelLimits:        "gpt-4o",
	}
	limitsMap := token.GetModelLimitsMap()

	if ModelLimitMapAllows(limitsMap, "claude-3-5-sonnet-20241022") {
		t.Fatal("expected unconfigured model to be blocked")
	}
}

func TestSyncModelLimits(t *testing.T) {
	t.Parallel()

	token := &Token{
		ModelLimitsEnabled: true,
		ModelLimits:        " gpt-4 , ,gpt-4 ",
	}
	token.SyncModelLimits()

	if !token.ModelLimitsEnabled {
		t.Fatal("expected model_limits_enabled to remain true when limits exist")
	}
	if token.ModelLimits != "gpt-4" {
		t.Fatalf("expected cleaned model_limits %q, got %q", "gpt-4", token.ModelLimits)
	}

	emptyToken := &Token{
		ModelLimitsEnabled: true,
		ModelLimits:        " , ",
	}
	emptyToken.SyncModelLimits()
	if emptyToken.ModelLimitsEnabled {
		t.Fatal("expected model_limits_enabled to be false when limits are empty")
	}
	if emptyToken.ModelLimits != "" {
		t.Fatalf("expected empty model_limits, got %q", emptyToken.ModelLimits)
	}
}
