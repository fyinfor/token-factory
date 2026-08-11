package controller

import (
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
)

func TestParseUserModelPricingExportFields_Default(t *testing.T) {
	cols, errMsg := parseUserModelPricingExportFields("")
	if errMsg != "" {
		t.Fatalf("unexpected error: %s", errMsg)
	}
	if len(cols) != len(userModelPricingExportDefaultFields) {
		t.Fatalf("got %d cols, want %d", len(cols), len(userModelPricingExportDefaultFields))
	}
	for i, col := range cols {
		if col.Key != userModelPricingExportDefaultFields[i] {
			t.Fatalf("col[%d]=%s, want %s", i, col.Key, userModelPricingExportDefaultFields[i])
		}
	}
}

func TestParseUserModelPricingExportFields_SubsetAndDedup(t *testing.T) {
	cols, errMsg := parseUserModelPricingExportFields(
		"model_name, enabled, model_name ,price_discount_percent",
	)
	if errMsg != "" {
		t.Fatalf("unexpected error: %s", errMsg)
	}
	want := []string{"model_name", "enabled", "price_discount_percent"}
	if len(cols) != len(want) {
		t.Fatalf("got %d cols, want %d", len(cols), len(want))
	}
	for i, col := range cols {
		if col.Key != want[i] {
			t.Fatalf("col[%d]=%s, want %s", i, col.Key, want[i])
		}
	}
}

func TestParseUserModelPricingExportFields_Invalid(t *testing.T) {
	_, errMsg := parseUserModelPricingExportFields("model_name,foo")
	if !strings.Contains(errMsg, "不支持的导出字段") {
		t.Fatalf("want unsupported field error, got %q", errMsg)
	}
	_, errMsg = parseUserModelPricingExportFields(" , , ")
	if errMsg != "请至少选择一个导出字段" {
		t.Fatalf("want empty fields error, got %q", errMsg)
	}
}

func TestBuildUserModelPricingExportWorkbook(t *testing.T) {
	cols, errMsg := parseUserModelPricingExportFields(
		"model_name,price_discount_percent,enabled",
	)
	if errMsg != "" {
		t.Fatal(errMsg)
	}
	rows := []model.UserModelPricingOverride{
		{
			UserId:               7,
			ModelName:            "gpt-4o",
			PriceDiscountPercent: 42,
			OperatingCostPercent: 6,
			MarkupDiscountRate:   2,
			Enabled:              true,
			UpdatedTime:          time.Date(2026, 8, 10, 12, 0, 0, 0, time.Local).Unix(),
		},
	}
	data, err := buildUserModelPricingExportWorkbook(rows, "alice", cols)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 100 {
		t.Fatalf("workbook too small: %d bytes", len(data))
	}
}
