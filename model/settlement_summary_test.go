package model

import "testing"

func TestBuildSettlementSummaryFromLogsGroupsByChannel(t *testing.T) {
	logs := []*Log{
		{Id: 1, UserId: 10, Username: "u1", ChannelId: 1, PromptTokens: 100, CompletionTokens: 20, Quota: 1000, Other: `{}`},
		{Id: 2, UserId: 11, Username: "u2", ChannelId: 1, PromptTokens: 50, CompletionTokens: 10, Quota: 500, Other: `{}`},
		{Id: 3, UserId: 12, Username: "u3", ChannelId: 2, PromptTokens: 30, CompletionTokens: 5, Quota: 200, Other: `{}`},
	}
	logs[0].ChannelDisplay = "ch-a"
	logs[1].ChannelDisplay = "ch-a"
	logs[2].ChannelDisplay = "ch-b"

	summary := BuildSettlementSummaryFromLogs(logs, "platform", nil)
	if summary.Totals.RecordCount != 3 {
		t.Fatalf("totals record_count: got %d", summary.Totals.RecordCount)
	}
	if summary.Totals.PromptTokens != 180 {
		t.Fatalf("totals prompt_tokens: got %d", summary.Totals.PromptTokens)
	}
	if len(summary.Groups) != 2 {
		t.Fatalf("groups: got %d", len(summary.Groups))
	}
	if summary.Groups[0].Key != "1" || summary.Groups[0].RecordCount != 2 {
		t.Fatalf("top group: %+v", summary.Groups[0])
	}
}

func TestMergeSettlementGroupsByAgent(t *testing.T) {
	groupAcc := map[string]*settlementSummaryAccumulator{
		"10": {RecordCount: 2, PromptTokens: 100, UserPaidUSD: 1.5},
		"11": {RecordCount: 1, PromptTokens: 50, UserPaidUSD: 0.5},
		"12": {RecordCount: 3, PromptTokens: 80, UserPaidUSD: 2.0},
	}
	agentMap := map[int]string{
		10: "agent-a",
		11: "agent-a",
	}
	mergedAcc, mergedLabel := mergeSettlementGroupsByAgent(groupAcc, agentMap)
	if len(mergedAcc) != 2 {
		t.Fatalf("merged groups: got %d", len(mergedAcc))
	}
	a := mergedAcc["agent-a"]
	if a == nil || a.RecordCount != 3 || a.PromptTokens != 150 {
		t.Fatalf("agent-a: %+v", a)
	}
	none := mergedAcc["(无代理)"]
	if none == nil || none.RecordCount != 3 {
		t.Fatalf("无代理: %+v", none)
	}
	if mergedLabel["agent-a"] != "agent-a" {
		t.Fatalf("label: %q", mergedLabel["agent-a"])
	}
}
