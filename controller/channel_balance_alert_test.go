package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupBalanceAlertTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	model.DB = db
	if err := db.AutoMigrate(&model.Channel{}, &model.UserMessage{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}
}

func TestNotifyChannelBalanceAlertIfNeeded(t *testing.T) {
	setupBalanceAlertTestDB(t)

	common.OptionMapRWMutex.Lock()
	common.OptionMap = map[string]string{
		"ChannelBalanceAlertEnabled":       "true",
		"ChannelBalanceSoftAlertThreshold": "50",
		"ChannelBalanceRiskAlertThreshold": "20",
	}
	common.OptionMapRWMutex.Unlock()

	ch := &model.Channel{
		Name:    "test-channel",
		Balance: 100,
	}
	if err := model.DB.Create(ch).Error; err != nil {
		t.Fatalf("create channel failed: %v", err)
	}

	notifyChannelBalanceAlertIfNeeded(ch, 100, 10)

	var firstCount int64
	if err := model.DB.Model(&model.UserMessage{}).Count(&firstCount).Error; err != nil {
		t.Fatalf("count messages failed: %v", err)
	}
	if firstCount != 1 {
		t.Fatalf("expected 1 message after entering risk level, got %d", firstCount)
	}
	var firstMsg model.UserMessage
	if err := model.DB.Order("id desc").First(&firstMsg).Error; err != nil {
		t.Fatalf("load latest message failed: %v", err)
	}
	if firstMsg.ReceiverMinRole != common.RoleAdminUser {
		t.Fatalf("expected receiver_min_role=%d, got %d", common.RoleAdminUser, firstMsg.ReceiverMinRole)
	}

	notifyChannelBalanceAlertIfNeeded(ch, 10, 5)
	var secondCount int64
	if err := model.DB.Model(&model.UserMessage{}).Count(&secondCount).Error; err != nil {
		t.Fatalf("count messages failed: %v", err)
	}
	if secondCount != 1 {
		t.Fatalf("expected no duplicate message in same level, got %d", secondCount)
	}
}
