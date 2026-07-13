package model

import (
	"testing"

	"github.com/QuantumNous/new-api/setting/operation_setting"
)

func TestSalesDiscountPercent(t *testing.T) {
	got := SalesDiscountPercent(85, 3, 7)
	if got != 95 {
		t.Fatalf("expected 95, got %v", got)
	}
}

func TestComputeSettlementPriceBreakdownFixedPrice(t *testing.T) {
	other := map[string]interface{}{
		"global_model_price":             35.0,
		"group_ratio":                    1.0,
		"price_discount_percent":         85.0,
		"operating_cost_percent":         3.0,
		"channel_price_discount_percent": 88.0,
		"markup_discount_rate":           7.0,
		"sales_discount_percent":         95.0,
	}
	bd := ComputeSettlementPriceBreakdown(10, 5, 0, 16625000, other)
	if bd.OfficialTotal != 35 {
		t.Fatalf("official total: got %v", bd.OfficialTotal)
	}
	if bd.CostPrice != 29.75 {
		t.Fatalf("cost price: got %v", bd.CostPrice)
	}
	if bd.OperatingPrice != 30.8 {
		t.Fatalf("operating price: got %v", bd.OperatingPrice)
	}
	if bd.SalesPrice != 33.25 {
		t.Fatalf("sales price: got %v", bd.SalesPrice)
	}
}

func TestComputeSettlementPriceBreakdownTokenBased(t *testing.T) {
	other := map[string]interface{}{
		"global_model_ratio":             1.75,
		"global_completion_ratio":        2.0,
		"group_ratio":                    1.0,
		"price_discount_percent":         100.0,
		"channel_price_discount_percent": 100.0,
		"sales_discount_percent":         100.0,
	}
	// 1000 input + 500 output, official = 1000*1.75*2/1e6 + 500*1.75*2*2/1e6 = 0.0035 + 0.0035 = 0.007
	bd := ComputeSettlementPriceBreakdown(1000, 500, 0, 3500, other)
	if bd.OfficialTotal < 0.0069 || bd.OfficialTotal > 0.0071 {
		t.Fatalf("official total: got %v", bd.OfficialTotal)
	}
}

func TestComputeSettlementPriceBreakdownQuotaFallback(t *testing.T) {
	other := map[string]interface{}{
		"sales_discount_percent": 95.0,
	}
	// quota 16625000 => $33.25 sales, official = 33.25/0.95 = 35
	bd := ComputeSettlementPriceBreakdown(100, 50, 0, 16625000, other)
	if bd.SalesPrice < 33.24 || bd.SalesPrice > 33.26 {
		t.Fatalf("sales price from quota: got %v", bd.SalesPrice)
	}
	if bd.OfficialTotal < 34.99 || bd.OfficialTotal > 35.01 {
		t.Fatalf("official from quota fallback: got %v", bd.OfficialTotal)
	}
}

func TestParseSettlementDiscountSnapshotLegacy(t *testing.T) {
	other := map[string]interface{}{
		"channel_price_discount_percent": 88.0,
		"markup_discount_rate":           7.0,
	}
	snap := ParseSettlementDiscountSnapshot(other)
	if snap.OperatingDiscountPercent != 88 {
		t.Fatalf("operating discount: got %v", snap.OperatingDiscountPercent)
	}
	if snap.SalesDiscountPercent != 95 {
		t.Fatalf("sales discount: got %v", snap.SalesDiscountPercent)
	}
}

func TestFormatSettlementMoneyUSD(t *testing.T) {
	oldType := operation_setting.GetGeneralSetting().QuotaDisplayType
	defer func() {
		operation_setting.GetGeneralSetting().QuotaDisplayType = oldType
	}()
	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeUSD
	got := FormatSettlementMoney(11.73)
	if got != "$11.73" {
		t.Fatalf("expected $11.73, got %s", got)
	}
}

func TestFormatSettlementMoneyCNY(t *testing.T) {
	oldType := operation_setting.GetGeneralSetting().QuotaDisplayType
	oldRate := operation_setting.USDExchangeRate
	defer func() {
		operation_setting.GetGeneralSetting().QuotaDisplayType = oldType
		operation_setting.USDExchangeRate = oldRate
	}()
	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeCNY
	operation_setting.USDExchangeRate = 7.0
	got := FormatSettlementMoney(1.0)
	if got != "¥7.00" {
		t.Fatalf("expected ¥7.00, got %s", got)
	}
}
