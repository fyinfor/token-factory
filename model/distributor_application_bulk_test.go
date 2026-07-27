package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupDistributorBulkTestDB(t *testing.T) {
	t.Helper()
	previousDB := DB
	t.Cleanup(func() { DB = previousDB })

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	DB = db
	require.NoError(t, DB.AutoMigrate(&User{}, &DistributorApplication{}))
}

func TestSetDistributorsCommissionBpsScopes(t *testing.T) {
	setupDistributorBulkTestDB(t)

	users := []User{
		{Username: "personal-agent", DisplayName: "Personal", AffCode: "bulk-personal", Role: common.RoleCommonUser, IsDistributor: common.DistributorFlagYes, DistributorCommissionBps: 100},
		{Username: "enterprise-agent", DisplayName: "Enterprise", AffCode: "bulk-enterprise", Role: common.RoleCommonUser, IsDistributor: common.DistributorFlagYes, DistributorCommissionBps: 200},
		{Username: "other-agent", DisplayName: "Other", AffCode: "bulk-other", Role: common.RoleCommonUser, IsDistributor: common.DistributorFlagYes, DistributorCommissionBps: 300},
		{Username: "ordinary-user", DisplayName: "Ordinary", AffCode: "bulk-ordinary", Role: common.RoleCommonUser, IsDistributor: common.DistributorFlagNo, DistributorCommissionBps: 400},
		{Username: "admin-user", DisplayName: "Admin", AffCode: "bulk-admin", Role: common.RoleAdminUser, IsDistributor: common.DistributorFlagYes, DistributorCommissionBps: 500},
	}
	for i := range users {
		require.NoError(t, DB.Create(&users[i]).Error)
	}
	require.NoError(t, DB.Create(&DistributorApplication{
		UserId: users[0].Id, ApplyType: DistributorApplyTypePersonal, RealName: "张三", Contact: "13800000000",
	}).Error)
	require.NoError(t, DB.Create(&DistributorApplication{
		UserId: users[1].Id, ApplyType: DistributorApplyTypeEnterprise, RealName: "星河科技", Contact: "13900000000",
	}).Error)

	updated, err := SetDistributorsCommissionBps(1200, "星河", 0)
	require.NoError(t, err)
	require.EqualValues(t, 1, updated)

	updated, err = SetDistributorsCommissionBps(800, "", DistributorApplyTypePersonal)
	require.NoError(t, err)
	require.EqualValues(t, 1, updated)

	updated, err = SetDistributorsCommissionBps(0, "", 0)
	require.NoError(t, err)
	require.EqualValues(t, 3, updated)

	var reloaded []User
	require.NoError(t, DB.Order("id ASC").Find(&reloaded).Error)
	require.Equal(t, 0, reloaded[0].DistributorCommissionBps)
	require.Equal(t, 0, reloaded[1].DistributorCommissionBps)
	require.Equal(t, 0, reloaded[2].DistributorCommissionBps)
	require.Equal(t, 400, reloaded[3].DistributorCommissionBps)
	require.Equal(t, 500, reloaded[4].DistributorCommissionBps)
}

func TestSetDistributorsCommissionBpsRejectsOutOfRange(t *testing.T) {
	setupDistributorBulkTestDB(t)

	_, err := SetDistributorsCommissionBps(10001, "", 0)
	require.Error(t, err)

	_, err = SetDistributorsCommissionBps(1000, "", 99)
	require.Error(t, err)
}
