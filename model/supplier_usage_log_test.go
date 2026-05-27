package model

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupSupplierUsageLogTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&Log{}); err != nil {
		t.Fatal(err)
	}
	LOG_DB = db
}

func TestAggregateSupplierUsageFromLogs_scopedByChannelOnly(t *testing.T) {
	setupSupplierUsageLogTestDB(t)
	now := time.Now().Unix()
	hour := now - (now % 3600)

	logs := []Log{
		{UserId: 1, CreatedAt: now, Type: LogTypeConsume, ModelName: "GLM-5.1", Quota: 100, PromptTokens: 10, CompletionTokens: 5, ChannelId: 10},
		{UserId: 2, CreatedAt: now, Type: LogTypeConsume, ModelName: "GLM-5.1", Quota: 200, PromptTokens: 20, CompletionTokens: 10, ChannelId: 99},
		{UserId: 1, CreatedAt: now, Type: LogTypeError, ModelName: "GLM-5.1", Quota: 0, ChannelId: 10},
		{UserId: 1, CreatedAt: now, Type: LogTypeConsume, ModelName: "other-model", Quota: 50, PromptTokens: 1, CompletionTokens: 1, ChannelId: 10},
	}
	if err := LOG_DB.Create(&logs).Error; err != nil {
		t.Fatal(err)
	}

	_, byModel, stat, err := AggregateSupplierUsageFromLogs(hour-3600, hour+3600, []int{10})
	if err != nil {
		t.Fatal(err)
	}
	if stat.Quota != 150 {
		t.Fatalf("expected quota 150, got %d", stat.Quota)
	}
	if len(byModel) != 2 {
		t.Fatalf("expected 2 model rows, got %d", len(byModel))
	}
	byName := make(map[string]SupplierUsageByModel, len(byModel))
	for _, row := range byModel {
		byName[row.ModelName] = row
	}
	if byName["GLM-5.1"].Count != 1 || byName["GLM-5.1"].Quota != 100 {
		t.Fatalf("unexpected GLM-5.1: %+v", byName["GLM-5.1"])
	}
	if byName["other-model"].Count != 1 || byName["other-model"].Quota != 50 {
		t.Fatalf("unexpected other-model: %+v", byName["other-model"])
	}
}

func TestAggregateSupplierUsageByModelAndUser_groupsByUser(t *testing.T) {
	setupSupplierUsageLogTestDB(t)
	now := time.Now().Unix()

	logs := []Log{
		{UserId: 1, Username: "alice", CreatedAt: now, Type: LogTypeConsume, ModelName: "glm-4", Quota: 100, PromptTokens: 10, CompletionTokens: 5, ChannelId: 10},
		{UserId: 2, Username: "bob", CreatedAt: now, Type: LogTypeConsume, ModelName: "glm-4", Quota: 200, PromptTokens: 20, CompletionTokens: 10, ChannelId: 10},
		{UserId: 1, Username: "alice", CreatedAt: now, Type: LogTypeConsume, ModelName: "glm-4", Quota: 50, PromptTokens: 5, CompletionTokens: 2, ChannelId: 10},
		{UserId: 3, Username: "carol", CreatedAt: now, Type: LogTypeConsume, ModelName: "other", Quota: 999, PromptTokens: 1, CompletionTokens: 1, ChannelId: 10},
	}
	if err := LOG_DB.Create(&logs).Error; err != nil {
		t.Fatal(err)
	}

	rows, err := AggregateSupplierUsageByModelAndUser(now-3600, now+3600, []int{10}, "glm-4")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 users, got %d", len(rows))
	}
	byUser := make(map[int]SupplierUsageByUser, len(rows))
	for _, row := range rows {
		byUser[row.UserID] = row
	}
	if byUser[1].Requests != 2 || byUser[1].Quota != 150 {
		t.Fatalf("unexpected alice: %+v", byUser[1])
	}
	if byUser[2].Requests != 1 || byUser[2].Quota != 200 {
		t.Fatalf("unexpected bob: %+v", byUser[2])
	}
}

func TestAggregateSupplierUsageAllModelUsers_groupsByModelAndUser(t *testing.T) {
	setupSupplierUsageLogTestDB(t)
	now := time.Now().Unix()

	logs := []Log{
		{UserId: 1, Username: "alice", CreatedAt: now, Type: LogTypeConsume, ModelName: "glm-4", Quota: 100, PromptTokens: 10, CompletionTokens: 5, ChannelId: 10},
		{UserId: 2, Username: "bob", CreatedAt: now, Type: LogTypeConsume, ModelName: "glm-4", Quota: 200, PromptTokens: 20, CompletionTokens: 10, ChannelId: 10},
		{UserId: 1, Username: "alice", CreatedAt: now, Type: LogTypeConsume, ModelName: "other", Quota: 50, PromptTokens: 1, CompletionTokens: 1, ChannelId: 10},
	}
	if err := LOG_DB.Create(&logs).Error; err != nil {
		t.Fatal(err)
	}

	rows, err := AggregateSupplierUsageAllModelUsers(now-3600, now+3600, []int{10})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	grouped := GroupSupplierUsageByModelUsers(rows)
	if len(grouped["glm-4"]) != 2 {
		t.Fatalf("expected 2 users for glm-4, got %d", len(grouped["glm-4"]))
	}
}

func TestAggregateSupplierUsageFromLogs_emptyChannels(t *testing.T) {
	setupSupplierUsageLogTestDB(t)
	hourly, byModel, stat, err := AggregateSupplierUsageFromLogs(0, time.Now().Unix(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(hourly) != 0 || len(byModel) != 0 || stat.Quota != 0 {
		t.Fatalf("expected empty result, got hourly=%d byModel=%d quota=%d", len(hourly), len(byModel), stat.Quota)
	}
}
