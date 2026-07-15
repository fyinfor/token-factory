package model

import (
	"math"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupUsedQuotaTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	DB = db
	LOG_DB = db
	common.UsingSQLite = true
	common.BatchUpdateEnabled = false
	common.QuotaPerUnit = 500000
	operation_setting.USDExchangeRate = 6.82
	require.NoError(t, db.AutoMigrate(&User{}, &Log{}, &TopUp{}))
}

func TestDecreaseUserUsedQuota_FloorsAtZero(t *testing.T) {
	setupUsedQuotaTestDB(t)
	require.NoError(t, DB.Create(&User{Id: 1, Username: "u1", UsedQuota: 100, Status: 1}).Error)

	DecreaseUserUsedQuota(1, 40)
	var used int
	require.NoError(t, DB.Model(&User{}).Where("id = ?", 1).Select("used_quota").Scan(&used).Error)
	assert.Equal(t, 60, used)

	DecreaseUserUsedQuota(1, 9999)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", 1).Select("used_quota").Scan(&used).Error)
	assert.Equal(t, 0, used)
}

func TestReconcileUserUsedQuota_CreditsMinusRemain(t *testing.T) {
	setupUsedQuotaTestDB(t)
	// 充值 200 CNY → quota_to_add = 200/6.82*500000 ≈ 14662757
	topupQuota := int(math.Round(200.0 / 6.82 * 500000))
	remain := 1000000
	require.NoError(t, DB.Create(&User{
		Id: 2, Username: "u2", Quota: remain, UsedQuota: 1, Status: 1,
	}).Error)
	require.NoError(t, DB.Create(&TopUp{
		UserId: 2, Status: common.TopUpStatusSuccess, QuotaToAdd: topupQuota, Money: 200,
		TradeNo: "test-trade-u2",
	}).Error)
	require.NoError(t, LOG_DB.Create(&Log{
		UserId: 2, Type: LogTypeSystem, Content: "新用户注册赠送 ¥0.999600 额度",
	}).Error)

	oldUsed, newUsed, err := ReconcileUserUsedQuotaFromLogs(2)
	require.NoError(t, err)
	assert.Equal(t, 1, oldUsed)

	giftQ := int(creditQuotaFromCNYDisplay(0.999600))
	wantUsed := topupQuota + giftQ - remain
	assert.Equal(t, wantUsed, newUsed)

	var used int
	require.NoError(t, DB.Model(&User{}).Where("id = ?", 2).Select("used_quota").Scan(&used).Error)
	assert.Equal(t, wantUsed, used)
	// 剩余 + 已用 = 累计到账
	assert.Equal(t, topupQuota+giftQ, remain+used)
}

func TestRecordTaskBillingLog_RefundDecreasesUsedQuota(t *testing.T) {
	setupUsedQuotaTestDB(t)
	require.NoError(t, DB.Create(&User{Id: 3, Username: "u3", Quota: 10000, UsedQuota: 4000, Status: 1}).Error)

	RecordTaskBillingLog(RecordTaskBillingLogParams{
		UserId:  3,
		LogType: LogTypeRefund,
		Quota:   1500,
		Other:   SetBillingLogMetadata(map[string]interface{}{}, BillingPhaseRefund, true, 1500, 1500),
	})

	var used int
	require.NoError(t, DB.Model(&User{}).Where("id = ?", 3).Select("used_quota").Scan(&used).Error)
	assert.Equal(t, 2500, used)
}
