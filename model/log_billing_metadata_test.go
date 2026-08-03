package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
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

func TestNormalizeLogBillingMetadataPositiveQuotaSettlementMarker(t *testing.T) {
	log := &Log{
		Type:  LogTypeConsume,
		Quota: 550000,
		Other: common.MapToJsonStr(map[string]interface{}{
			"task_id":              "task_legacy_positive_marker",
			"billing_phase":        BillingPhaseSettlementMarker,
			"affects_balance":      false,
			"actual_quota":         550000,
			"pre_consumed_quota":   750000,
			"billing_mode":         "video_token_output",
			"video_final_quota":    550000,
			"video_billed_quota":   550000,
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
	if got, _ := logOtherNumber(other["display_quota"]); int(got) != 550000 {
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

func setupSettlementMergeTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("sql db: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	oldDB, oldLogDB := DB, LOG_DB
	DB = db
	LOG_DB = db
	common.UsingSQLite = true
	t.Cleanup(func() {
		DB = oldDB
		LOG_DB = oldLogDB
	})
	if err := db.AutoMigrate(&Log{}, &Task{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
}

func TestMergeSettlementMarkersIntoPreChargeLogsBatch(t *testing.T) {
	setupSettlementMergeTestDB(t)

	preA := &Log{
		UserId: 7, Type: LogTypeConsume, Quota: 1000, CreatedAt: 1_000,
		Other: common.MapToJsonStr(map[string]interface{}{
			"task_id": "task_a", "billing_mode": "video_per_second",
		}),
	}
	preB := &Log{
		UserId: 7, Type: LogTypeConsume, Quota: 2000, CreatedAt: 1_100,
		Other: common.MapToJsonStr(map[string]interface{}{
			"task_id": "task_b", "billing_mode": "video_per_second",
		}),
	}
	markerA := &Log{
		UserId: 7, Type: LogTypeConsume, Quota: 0, CreatedAt: 1_200,
		Other: common.MapToJsonStr(SetBillingLogMetadata(map[string]interface{}{
			"task_id": "task_a", "actual_quota": 800, "pre_consumed_quota": 1000,
		}, BillingPhaseSettlementMarker, false, 800, 0)),
	}
	markerB := &Log{
		UserId: 7, Type: LogTypeConsume, Quota: 0, CreatedAt: 1_300,
		Other: common.MapToJsonStr(SetBillingLogMetadata(map[string]interface{}{
			"task_id": "task_b", "actual_quota": 1500, "pre_consumed_quota": 2000,
		}, BillingPhaseSettlementMarker, false, 1500, 0)),
	}
	// 其他用户同名模式不应串数据
	noise := &Log{
		UserId: 99, Type: LogTypeConsume, Quota: 0, CreatedAt: 1_250,
		Other: common.MapToJsonStr(SetBillingLogMetadata(map[string]interface{}{
			"task_id": "task_a", "actual_quota": 1, "pre_consumed_quota": 1,
		}, BillingPhaseSettlementMarker, false, 1, 0)),
	}
	if err := LOG_DB.Create([]*Log{preA, preB, markerA, markerB, noise}).Error; err != nil {
		t.Fatalf("seed logs: %v", err)
	}

	logs := []*Log{preA, preB}
	mergeSettlementMarkersIntoPreChargeLogs(logs)

	otherA := mustLogOther(t, logs[0])
	if otherA["billing_phase"] != BillingPhaseSettlementMerged {
		t.Fatalf("task_a billing_phase = %v", otherA["billing_phase"])
	}
	if got, _ := logOtherNumber(otherA["actual_quota"]); int(got) != 800 {
		t.Fatalf("task_a actual_quota = %v", otherA["actual_quota"])
	}
	if got, _ := logOtherNumber(otherA["display_quota"]); int(got) != 800 {
		t.Fatalf("task_a display_quota = %v", otherA["display_quota"])
	}

	otherB := mustLogOther(t, logs[1])
	if otherB["billing_phase"] != BillingPhaseSettlementMerged {
		t.Fatalf("task_b billing_phase = %v", otherB["billing_phase"])
	}
	if got, _ := logOtherNumber(otherB["actual_quota"]); int(got) != 1500 {
		t.Fatalf("task_b actual_quota = %v", otherB["actual_quota"])
	}
}

func TestFillTaskUseTimeBatch(t *testing.T) {
	setupSettlementMergeTestDB(t)
	if err := DB.Create([]*Task{
		{TaskID: "task_u1", SubmitTime: 100, FinishTime: 140},
		{TaskID: "task_u2", StartTime: 200, FinishTime: 255},
	}).Error; err != nil {
		t.Fatalf("seed tasks: %v", err)
	}

	logs := []*Log{
		{Other: common.MapToJsonStr(map[string]interface{}{"task_id": "task_u1"})},
		{Other: common.MapToJsonStr(map[string]interface{}{"task_id": "task_u2"})},
		{UseTime: 9, Other: common.MapToJsonStr(map[string]interface{}{"task_id": "task_u1"})},
	}
	fillTaskUseTime(logs)
	if logs[0].UseTime != 40 {
		t.Fatalf("log0 use_time = %d", logs[0].UseTime)
	}
	if logs[1].UseTime != 55 {
		t.Fatalf("log1 use_time = %d", logs[1].UseTime)
	}
	if logs[2].UseTime != 9 {
		t.Fatalf("existing use_time should stay 9, got %d", logs[2].UseTime)
	}
}
