package model

import (
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupSupplierRevenuePushTestDB(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:supplier_revenue_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&Log{},
		&SupplierRevenuePeriod{},
		&SupplierRevenueDelivery{},
		&SupplierRevenueAttempt{},
		&SupplierRevenuePushConfig{},
	))
	originalDB, originalLogDB := DB, LOG_DB
	DB, LOG_DB = db, db
	t.Cleanup(func() {
		DB, LOG_DB = originalDB, originalLogDB
	})
}

func TestSumSupplierRevenueQuotaIncludesRefundAsNegativeAdjustment(t *testing.T) {
	setupSupplierRevenuePushTestDB(t)
	require.NoError(t, LOG_DB.Create(&[]Log{
		{CreatedAt: 100, Type: LogTypeConsume, Quota: 10, ChannelId: 1},
		{CreatedAt: 110, Type: LogTypeRefund, Quota: 3, ChannelId: 1},
		{CreatedAt: 120, Type: LogTypeConsume, Quota: 100, ChannelId: 2},
		{CreatedAt: 200, Type: LogTypeConsume, Quota: 50, ChannelId: 1},
	}).Error)

	rawQuota, err := SumSupplierRevenueQuota([]int{1}, 100, 200)
	require.NoError(t, err)
	require.Equal(t, int64(7), rawQuota)
}

func TestSupplierRevenuePeriodUniqueAndDeliverySettlement(t *testing.T) {
	setupSupplierRevenuePushTestDB(t)
	period := &SupplierRevenuePeriod{
		SupplierID: 1, ScheduleType: SupplierRevenueScheduleDaily,
		PeriodStart: 100, PeriodEnd: 200, RawQuota: 10,
		RawAmount: "0.1", Amount: "0.100000", Currency: "USD",
		ExchangeRate: "1", Status: SupplierRevenuePeriodPending,
	}
	created, err := CreateSupplierRevenuePeriod(period)
	require.NoError(t, err)
	require.True(t, created)
	duplicate := *period
	duplicate.ID = 0
	created, err = CreateSupplierRevenuePeriod(&duplicate)
	require.NoError(t, err)
	require.False(t, created)

	delivery := &SupplierRevenueDelivery{
		SupplierID: 1, BatchNo: "SRP-TEST-1", Kind: SupplierRevenueDeliveryKindScheduled,
		PeriodStart: 100, PeriodEnd: 200, Amount: "0.100000", Currency: "USD",
		Status: SupplierRevenueDeliveryCreated, MaxAttempts: 4,
	}
	require.NoError(t, CreateSupplierRevenueDelivery(delivery, []*SupplierRevenuePeriod{period}))
	require.NoError(t, MarkSupplierRevenueDeliverySuccess(delivery.ID, 1))

	var stored SupplierRevenuePeriod
	require.NoError(t, DB.First(&stored, period.ID).Error)
	require.Equal(t, SupplierRevenuePeriodSettled, stored.Status)
	require.Equal(t, delivery.ID, stored.SettledDeliveryID)
}

func TestListRetryableSupplierRevenueDeliveriesOnlyReturnsEnabledSuppliers(t *testing.T) {
	setupSupplierRevenuePushTestDB(t)
	require.NoError(t, DB.Create(&[]SupplierRevenuePushConfig{
		{SupplierID: 1, Enabled: true},
		{SupplierID: 2, Enabled: false},
	}).Error)
	require.NoError(t, DB.Create(&[]SupplierRevenueDelivery{
		{SupplierID: 1, BatchNo: "SRP-ENABLED", Amount: "1.000000", Currency: "USD", Status: SupplierRevenueDeliveryRetrying, NextRetryAt: 100},
		{SupplierID: 2, BatchNo: "SRP-DISABLED", Amount: "1.000000", Currency: "USD", Status: SupplierRevenueDeliveryRetrying, NextRetryAt: 100},
	}).Error)

	deliveries, err := ListRetryableSupplierRevenueDeliveries(100, 10)
	require.NoError(t, err)
	require.Len(t, deliveries, 1)
	require.Equal(t, 1, deliveries[0].SupplierID)
}

func TestFinalizeRetryingManualSupplierRevenueDeliveries(t *testing.T) {
	setupSupplierRevenuePushTestDB(t)
	delivery := &SupplierRevenueDelivery{
		SupplierID: 1, BatchNo: "SRP-MANUAL-LEGACY", Kind: SupplierRevenueDeliveryKindManual,
		Amount: "1.000000", Currency: SupplierRevenueCurrencyUSD,
		Status: SupplierRevenueDeliveryRetrying, MaxAttempts: 4, NextRetryAt: time.Now().Add(time.Hour).Unix(),
	}
	require.NoError(t, DB.Create(delivery).Error)
	require.NoError(t, FinalizeRetryingManualSupplierRevenueDeliveries())

	stored, err := GetSupplierRevenueDelivery(delivery.ID)
	require.NoError(t, err)
	require.Equal(t, SupplierRevenueDeliveryFailed, stored.Status)
	require.Zero(t, stored.NextRetryAt)
	require.NotZero(t, stored.CompletedAt)
}
