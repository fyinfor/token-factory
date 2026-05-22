package model

import "testing"

func TestEffectiveRuleUnitPrice_MarkupWhenGlobalUnset(t *testing.T) {
	// 仅渠道规则价、无全局规则价时，加价应仍作用于渠道价（回退基准）
	got := EffectiveRuleUnitPrice(3, 0, 100, 10)
	want := 3*1 + 3*0.1 // 3.3
	if got != want {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestEffectiveRuleUnitPrice_TwoTier(t *testing.T) {
	got := EffectiveRuleUnitPrice(2, 3, 100, 10)
	want := 2*1 + 3*0.1 // 2.3
	if got != want {
		t.Fatalf("got %v want %v", got, want)
	}
}
