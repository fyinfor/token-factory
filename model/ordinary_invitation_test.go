package model

import (
	"fmt"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupOrdinaryInvitationTestDB(t *testing.T) {
	t.Helper()
	previousDB := DB
	previousLogDB := LOG_DB
	previousQuotaForInviter := common.QuotaForInviter
	previousDefaultCommissionBps := common.AffiliateDefaultCommissionBps
	previousCommissionMode := common.DistributorCommissionMode

	dsn := fmt.Sprintf("file:ordinary_invitation_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	DB = db
	LOG_DB = db
	common.QuotaForInviter = 500
	common.AffiliateDefaultCommissionBps = 1000
	common.DistributorCommissionMode = common.DistributorCommissionModeTopup
	require.NoError(t, DB.AutoMigrate(&User{}, &AffInviteRelation{}, &AffInviteCommissionLog{}, &Log{}))

	t.Cleanup(func() {
		DB = previousDB
		LOG_DB = previousLogDB
		common.QuotaForInviter = previousQuotaForInviter
		common.AffiliateDefaultCommissionBps = previousDefaultCommissionBps
		common.DistributorCommissionMode = previousCommissionMode
		_ = sqlDB.Close()
	})
}

func TestOrdinaryInvitationRewardsThenOnlyFutureCommissionAfterUpgrade(t *testing.T) {
	setupOrdinaryInvitationTestDB(t)

	inviter := User{
		Username:  "ordinary-inviter",
		AffCode:   "ordinary-inviter-code",
		Role:      common.RoleCommonUser,
		Status:    common.UserStatusEnabled,
		Quota:     100,
		GiftQuota: 20,
	}
	require.NoError(t, DB.Create(&inviter).Error)
	invitee := User{
		Username:  "ordinary-invitee",
		AffCode:   "ordinary-invitee-code",
		Role:      common.RoleCommonUser,
		Status:    common.UserStatusEnabled,
		InviterId: inviter.Id,
	}
	require.NoError(t, DB.Create(&invitee).Error)

	require.NoError(t, EnsureAffInviteRelation(inviter.Id, invitee.Id))
	require.NoError(t, inviteUser(inviter.Id))

	var reloaded User
	require.NoError(t, DB.First(&reloaded, inviter.Id).Error)
	require.Equal(t, 1, reloaded.AffCount)
	require.Equal(t, 600, reloaded.Quota)
	require.Equal(t, 520, reloaded.GiftQuota)

	var relation AffInviteRelation
	require.NoError(t, DB.Where("inviter_id = ? AND invitee_user_id = ?", inviter.Id, invitee.Id).First(&relation).Error)
	require.Zero(t, relation.CommissionRatioBps)

	// 普通用户阶段发生的充值不产生分成，也不会在升级后补发。
	ApplyAffiliateTopupReward(invitee.Id, 10000)
	require.NoError(t, DB.First(&reloaded, inviter.Id).Error)
	require.Zero(t, reloaded.AffQuota)
	require.Zero(t, reloaded.AffHistoryQuota)

	require.NoError(t, DB.Model(&User{}).Where("id = ?", inviter.Id).Updates(map[string]any{
		"is_distributor":             common.DistributorFlagYes,
		"distributor_commission_bps": 1500,
	}).Error)

	// 升级后的新充值按升级后的当前比例结算。
	ApplyAffiliateTopupReward(invitee.Id, 10000)
	require.NoError(t, DB.First(&reloaded, inviter.Id).Error)
	require.Equal(t, 1500, reloaded.AffQuota)
	require.Equal(t, 1500, reloaded.AffHistoryQuota)

	var logs []AffInviteCommissionLog
	require.NoError(t, DB.Find(&logs).Error)
	require.Len(t, logs, 1)
	require.Equal(t, 1500, logs[0].RewardQuota)
	require.Equal(t, 1500, logs[0].CommissionBps)
}

func TestInvitationCodeOnlyAcceptsEnabledNonAdminUsers(t *testing.T) {
	setupOrdinaryInvitationTestDB(t)

	ordinary := User{Username: "eligible-inviter", AffCode: "eligible-code", Role: common.RoleCommonUser, Status: common.UserStatusEnabled}
	disabled := User{Username: "disabled-inviter", AffCode: "disabled-code", Role: common.RoleCommonUser, Status: common.UserStatusDisabled}
	admin := User{Username: "admin-inviter", AffCode: "admin-code", Role: common.RoleAdminUser, Status: common.UserStatusEnabled}
	require.NoError(t, DB.Create(&ordinary).Error)
	require.NoError(t, DB.Create(&disabled).Error)
	require.NoError(t, DB.Create(&admin).Error)

	id, err := GetUserIdByAffCode(" eligible-code ")
	require.NoError(t, err)
	require.Equal(t, ordinary.Id, id)

	_, err = GetUserIdByAffCode(disabled.AffCode)
	require.Error(t, err)
	_, err = GetUserIdByAffCode(admin.AffCode)
	require.Error(t, err)
}

func TestEffectiveAffiliateCommissionUsesCurrentDefaultsForNewRelation(t *testing.T) {
	setupOrdinaryInvitationTestDB(t)

	inviter := User{
		Username:                 "ratio-inviter",
		AffCode:                  "ratio-inviter-code",
		Role:                     common.RoleCommonUser,
		Status:                   common.UserStatusEnabled,
		IsDistributor:            common.DistributorFlagYes,
		DistributorCommissionBps: 1600,
	}
	require.NoError(t, DB.Create(&inviter).Error)
	invitee := User{Username: "ratio-invitee", AffCode: "ratio-invitee-code", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, InviterId: inviter.Id}
	require.NoError(t, DB.Create(&invitee).Error)
	require.NoError(t, EnsureAffInviteRelation(inviter.Id, invitee.Id))

	require.Equal(t, 1600, EffectiveAffiliateCommissionBps(&inviter, invitee.Id))
	items, total, err := ListAffInvitees(inviter.Id, "", &common.PageInfo{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, items, 1)
	require.Equal(t, 1600, items[0].CommissionRatioBps)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", inviter.Id).Update("distributor_commission_bps", 0).Error)
	inviter.DistributorCommissionBps = 0
	require.Equal(t, 1000, EffectiveAffiliateCommissionBps(&inviter, invitee.Id))
	require.NoError(t, DB.Model(&AffInviteRelation{}).
		Where("inviter_id = ? AND invitee_user_id = ?", inviter.Id, invitee.Id).
		Update("commission_ratio_bps", 700).Error)
	require.Equal(t, 700, EffectiveAffiliateCommissionBps(&inviter, invitee.Id))
}
