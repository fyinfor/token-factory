package service

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestIsAliyunInvalidCertNoError(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		message string
		want    bool
	}{
		{name: "invalid cert number", code: "401", message: "参数非法(certNo)", want: true},
		{name: "case insensitive", code: "401", message: "invalid CERTNO", want: true},
		{name: "different parameter", code: "401", message: "参数非法(certName)", want: false},
		{name: "different code", code: "500", message: "参数非法(certNo)", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isAliyunInvalidCertNoError(test.code, test.message); got != test.want {
				t.Fatalf("isAliyunInvalidCertNoError(%q, %q) = %v, want %v", test.code, test.message, got, test.want)
			}
		})
	}
}

func TestRecordRealNameVerificationRewardLog(t *testing.T) {
	previousDB, previousLogDB := model.DB, model.LOG_DB
	previousRedisEnabled := common.RedisEnabled
	db, err := gorm.Open(sqlite.Open("file:real_name_reward_log?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err = db.AutoMigrate(&model.User{}, &model.Log{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	model.DB, model.LOG_DB = db, db
	common.RedisEnabled = false
	t.Cleanup(func() {
		model.DB, model.LOG_DB = previousDB, previousLogDB
		common.RedisEnabled = previousRedisEnabled
	})

	user := &model.User{Username: "real-name-log-user", Password: "test-password"}
	if err = db.Create(user).Error; err != nil {
		t.Fatalf("create test user: %v", err)
	}

	recordRealNameVerificationRewardLog(&model.RealNameVerification{
		UserId:      user.Id,
		RewardQuota: int(common.QuotaPerUnit),
	})

	var log model.Log
	if err = db.Where("user_id = ?", user.Id).First(&log).Error; err != nil {
		t.Fatalf("query reward log: %v", err)
	}
	if log.Type != model.LogTypeSystem {
		t.Fatalf("log type = %d, want %d", log.Type, model.LogTypeSystem)
	}
	if !strings.Contains(log.Content, "实名认证成功") || !strings.Contains(log.Content, "已发放至赠送余额") {
		t.Fatalf("unexpected reward log content: %q", log.Content)
	}
}
