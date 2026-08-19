package service

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
)

func TestRankingResponsesIncludeVendorEnglishName(t *testing.T) {
	totals := []model.RankingQuotaTotal{{
		ModelName:   "gpt-test",
		TotalTokens: 100,
	}}
	meta := map[string]rankingModelMeta{
		"gpt-test": {
			vendor:       "测试供应商",
			vendorNameEn: "Test vendor",
		},
	}

	models := buildRankedModels(totals, 100, map[string]int{"gpt-test": 2}, map[string]int64{}, meta, true)
	if len(models) != 1 || models[0].VendorNameEn != "Test vendor" {
		t.Fatalf("model response vendor_name_en = %q, want %q", models[0].VendorNameEn, "Test vendor")
	}

	vendors := buildRankedVendors(totals, nil, 100, meta, true)
	if len(vendors) != 1 || vendors[0].VendorNameEn != "Test vendor" {
		t.Fatalf("vendor response vendor_name_en = %q, want %q", vendors[0].VendorNameEn, "Test vendor")
	}

	movers, _ := buildRankingMovers(models)
	if len(movers) != 1 || movers[0].VendorNameEn != "Test vendor" {
		t.Fatalf("mover response vendor_name_en = %q, want %q", movers[0].VendorNameEn, "Test vendor")
	}
}
