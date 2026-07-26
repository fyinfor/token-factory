package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
)

func TestSelectChannelLocal_DefaultFallback(t *testing.T) {
	// 无 DB 时 GetRouteConfig 会尝试写库；此处仅测空候选 + 无用户时的 fallback 分支需 mock。
	// 空候选在 weight/price 下也会 Fallback；用 Status 过滤后为空即可。
	res := SelectChannelLocal("gpt-4o", 0, nil)
	if !res.Fallback {
		t.Fatalf("expected fallback for empty candidates or default mode, got %+v", res)
	}
}

func TestSortRouteCandidatesByPrice(t *testing.T) {
	in := []RouteChannelCandidate{
		{ChannelID: 2, Price: 3, Status: common.ChannelStatusEnabled},
		{ChannelID: 1, Price: 1, Status: common.ChannelStatusEnabled},
		{ChannelID: 3, Price: 2, Status: common.ChannelStatusEnabled},
	}
	out := sortRouteCandidatesByPrice(in)
	if out[0].ChannelID != 1 || out[1].ChannelID != 3 || out[2].ChannelID != 2 {
		t.Fatalf("unexpected order: %+v", out)
	}
}

func TestSortRouteCandidatesByWeightDesc(t *testing.T) {
	in := []RouteChannelCandidate{
		{ChannelID: 2, Weight: 10},
		{ChannelID: 1, Weight: 50},
		{ChannelID: 3, Weight: 50},
	}
	out := sortRouteCandidatesByWeightDesc(in)
	if out[0].ChannelID != 1 || out[1].ChannelID != 3 || out[2].ChannelID != 2 {
		t.Fatalf("unexpected order: %+v", out)
	}
}

func TestResolveModelGroupKeyWithUser(t *testing.T) {
	user := map[string]string{"a-b-20240101": "user-group"}
	global := map[string]string{"a-b-20240101": "global-group"}
	if got := ResolveModelGroupKeyWithUser("a-b-20240101", user, global); got != "user-group" {
		t.Fatalf("user override preferred, got %q", got)
	}
	if got := ResolveModelGroupKeyWithUser("a-b-20240101", nil, global); got != "global-group" {
		t.Fatalf("global override, got %q", got)
	}
	if got := ResolveModelGroupKeyWithUser("deepseek-v4-flash-0221", nil, nil); got != "deepseek-v4-flash" {
		t.Fatalf("normalize, got %q", got)
	}
}
