package model

import (
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestChannelModelHotOverrideLifecycle(t *testing.T) {
	previousDB := DB
	t.Cleanup(func() { DB = previousDB })

	db, err := gorm.Open(
		sqlite.Open(fmt.Sprintf("file:channel_model_hot_override_%d?mode=memory&cache=shared", time.Now().UnixNano())),
		&gorm.Config{},
	)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&ChannelModelHotOverride{}); err != nil {
		t.Fatalf("migrate sqlite: %v", err)
	}
	DB = db

	override := &ChannelModelHotOverride{
		ChannelID:    12,
		ModelName:    "demo-model",
		OverrideMode: ChannelModelHotOverrideForceHot,
		ManualRank:   2,
	}
	if err := SaveChannelModelHotOverride(override); err != nil {
		t.Fatalf("save force-hot override: %v", err)
	}

	overrides, err := GetAllChannelModelHotOverrides()
	if err != nil || len(overrides) != 1 {
		t.Fatalf("list overrides: len=%d err=%v", len(overrides), err)
	}
	if overrides[0].OverrideMode != ChannelModelHotOverrideForceHot || overrides[0].ManualRank != 2 {
		t.Fatalf("unexpected override: %+v", overrides[0])
	}

	override.OverrideMode = ChannelModelHotOverrideForceNotHot
	override.ManualRank = 99
	if err := SaveChannelModelHotOverride(override); err != nil {
		t.Fatalf("update force-not-hot override: %v", err)
	}
	overrides, err = GetAllChannelModelHotOverrides()
	if err != nil || len(overrides) != 1 || overrides[0].OverrideMode != ChannelModelHotOverrideForceNotHot {
		t.Fatalf("unexpected updated overrides: %+v err=%v", overrides, err)
	}

	override.OverrideMode = "auto"
	if err := SaveChannelModelHotOverride(override); err != nil {
		t.Fatalf("restore automatic mode: %v", err)
	}
	overrides, err = GetAllChannelModelHotOverrides()
	if err != nil || len(overrides) != 0 {
		t.Fatalf("expected override deletion: %+v err=%v", overrides, err)
	}
}

func TestBatchSaveChannelModelHotOverrides(t *testing.T) {
	previousDB := DB
	t.Cleanup(func() { DB = previousDB })

	db, err := gorm.Open(
		sqlite.Open(fmt.Sprintf("file:channel_model_hot_override_batch_%d?mode=memory&cache=shared", time.Now().UnixNano())),
		&gorm.Config{},
	)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&ChannelModelHotOverride{}); err != nil {
		t.Fatalf("migrate sqlite: %v", err)
	}
	DB = db

	if err := BatchSaveChannelModelHotOverrides([]ChannelModelHotOverride{
		{ChannelID: 1, ModelName: "model-a", OverrideMode: ChannelModelHotOverrideForceHot, ManualRank: 1},
		{ChannelID: 2, ModelName: "model-b", OverrideMode: ChannelModelHotOverrideForceNotHot},
	}); err != nil {
		t.Fatalf("batch save overrides: %v", err)
	}
	if err := BatchSaveChannelModelHotOverrides([]ChannelModelHotOverride{
		{ChannelID: 1, ModelName: "model-a", OverrideMode: "auto"},
	}); err != nil {
		t.Fatalf("batch restore automatic mode: %v", err)
	}

	overrides, err := GetAllChannelModelHotOverrides()
	if err != nil || len(overrides) != 1 {
		t.Fatalf("list batch overrides: len=%d err=%v", len(overrides), err)
	}
	if overrides[0].ChannelID != 2 || overrides[0].ModelName != "model-b" {
		t.Fatalf("unexpected remaining override: %+v", overrides[0])
	}
}

func TestModelRenameMovesChannelModelHotOverrides(t *testing.T) {
	previousDB := DB
	t.Cleanup(func() { DB = previousDB })

	db, err := gorm.Open(
		sqlite.Open(fmt.Sprintf("file:model_hot_override_rename_%d?mode=memory&cache=shared", time.Now().UnixNano())),
		&gorm.Config{},
	)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Model{}, &ChannelModelHotOverride{}); err != nil {
		t.Fatalf("migrate sqlite: %v", err)
	}
	DB = db

	meta := Model{ModelName: "old-model", Status: 1, SyncOfficial: 1}
	if err := DB.Create(&meta).Error; err != nil {
		t.Fatalf("create model: %v", err)
	}
	if err := SaveChannelModelHotOverride(&ChannelModelHotOverride{
		ChannelID:    8,
		ModelName:    meta.ModelName,
		OverrideMode: ChannelModelHotOverrideForceHot,
		ManualRank:   1,
	}); err != nil {
		t.Fatalf("create override: %v", err)
	}

	meta.ModelName = "new-model"
	if err := meta.Update(); err != nil {
		t.Fatalf("rename model: %v", err)
	}

	overrides, err := GetAllChannelModelHotOverrides()
	if err != nil || len(overrides) != 1 {
		t.Fatalf("list renamed overrides: len=%d err=%v", len(overrides), err)
	}
	if overrides[0].ModelName != "new-model" {
		t.Fatalf("override did not follow model rename: %+v", overrides[0])
	}
}
