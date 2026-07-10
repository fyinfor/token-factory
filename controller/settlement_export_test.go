package controller

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

func TestSettlementMoneyHeaderUsesRuntimeCurrency(t *testing.T) {
	oldType := operation_setting.GetGeneralSetting().QuotaDisplayType
	oldRate := operation_setting.USDExchangeRate
	defer func() {
		operation_setting.GetGeneralSetting().QuotaDisplayType = oldType
		operation_setting.USDExchangeRate = oldRate
	}()

	headerFn := settlementMoneyHeader("官方输入价格", "Official input price")

	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeUSD
	if got := headerFn("zh-CN"); got != "官方输入价格(USD)" {
		t.Fatalf("USD header: got %q", got)
	}

	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeCNY
	operation_setting.USDExchangeRate = 7.0
	if got := headerFn("zh-CN"); got != "官方输入价格(CNY)" {
		t.Fatalf("CNY header: got %q", got)
	}
	if got := model.FormatSettlementMoney(1.0); got != "¥7.00" {
		t.Fatalf("CNY value: got %q", got)
	}
}

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
