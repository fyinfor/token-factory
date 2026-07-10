package controller

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/model"
)

func TestWriteSettlementSummarySection(t *testing.T) {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	summary := &model.SettlementSummaryResult{
		Scope:    "channel",
		Currency: "USD",
		Totals: model.SettlementSummaryAmounts{
			RecordCount:      3,
			PromptTokens:     100,
			CompletionTokens: 20,
			CacheTokens:      5,
			OfficialTotal:    "$0.12",
			CostPrice:        "$0.03",
			OperatingPrice:   "$0.03",
			SalesPrice:       "$0.03",
			UserPaid:         "$0.03",
		},
		Groups: []model.SettlementSummaryGroup{
			{
				Key:   "1",
				Label: "渠道A",
				SettlementSummaryAmounts: model.SettlementSummaryAmounts{
					RecordCount:      3,
					PromptTokens:     100,
					CompletionTokens: 20,
					CacheTokens:      5,
					OfficialTotal:    "$0.12",
					CostPrice:        "$0.03",
					OperatingPrice:   "$0.03",
					SalesPrice:       "$0.03",
					UserPaid:         "$0.03",
				},
			},
		},
	}
	if err := writeSettlementSummarySection(w, summary, "zh-CN"); err != nil {
		t.Fatalf("writeSettlementSummarySection: %v", err)
	}
	w.Flush()
	out := buf.String()
	for _, want := range []string{"【汇总合计】", "合计", "【渠道明细】", "渠道A", "请求笔数", "官方总价(USD)"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output:\n%s", want, out)
		}
	}
}
