package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
)

func mustLogOther(t *testing.T, log *Log) map[string]interface{} {
	t.Helper()
	other, err := common.StrToMap(log.Other)
	if err != nil {
		t.Fatalf("parse log other: %v", err)
	}
	return other
}

func TestNormalizeLogBillingMetadataLegacySettlementMarker(t *testing.T) {
	log := &Log{
		Type:  LogTypeConsume,
		Quota: 0,
		Other: common.MapToJsonStr(map[string]interface{}{
			"task_id":              "task_1",
			"actual_quota":         7500000,
			"pre_consumed_quota":   7500000,
			"billing_mode":         "video_per_second",
			"video_final_quota":    7500000,
			"video_billed_quota":   7500000,
			"video_quota_per_unit": 500000,
		}),
	}

	normalizeLogBillingMetadata(log)
	other := mustLogOther(t, log)

	if other["billing_phase"] != BillingPhaseSettlementMarker {
		t.Fatalf("billing_phase = %v", other["billing_phase"])
	}
	if other["affects_balance"] != false {
		t.Fatalf("affects_balance = %v", other["affects_balance"])
	}
	if got, _ := logOtherNumber(other["display_quota"]); int(got) != 7500000 {
		t.Fatalf("display_quota = %v", other["display_quota"])
	}
	if got, _ := logOtherNumber(other["balance_delta"]); int(got) != 0 {
		t.Fatalf("balance_delta = %v", other["balance_delta"])
	}
	if !isTaskSettlementMarkerLog(log, other) {
		t.Fatalf("expected settlement marker")
	}
}

func TestNormalizeLogBillingMetadataLegacyPreCharge(t *testing.T) {
	log := &Log{
		Type:  LogTypeConsume,
		Quota: 7500000,
		Other: common.MapToJsonStr(map[string]interface{}{
			"task_id":       "task_1",
			"billing_mode":  "video_per_second",
			"request_path":  "/v1/video/generations",
			"video_seconds": 5,
		}),
	}

	normalizeLogBillingMetadata(log)
	other := mustLogOther(t, log)

	if other["billing_phase"] != BillingPhasePreCharge {
		t.Fatalf("billing_phase = %v", other["billing_phase"])
	}
	if other["affects_balance"] != true {
		t.Fatalf("affects_balance = %v", other["affects_balance"])
	}
	if got, _ := logOtherNumber(other["display_quota"]); int(got) != 7500000 {
		t.Fatalf("display_quota = %v", other["display_quota"])
	}
	if got, _ := logOtherNumber(other["balance_delta"]); int(got) != -7500000 {
		t.Fatalf("balance_delta = %v", other["balance_delta"])
	}
}
