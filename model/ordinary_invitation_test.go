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
	require.NoError(t, DB.AutoMigrate(&User{}, &OrdinaryInviteRelation{}, &AffInviteRelation{}, &AffInviteCommissionLog{}, &Log{}))

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
		Username: "ordinary-invitee",
		AffCode:  "ordinary-invitee-code",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, DB.Create(&invitee).Error)

	require.NoError(t, CreateRegistrationInviteRelation(inviter.Id, invitee.Id, common.QuotaForInviter))
	require.NoError(t, inviteUser(inviter.Id))

	var reloaded User
	require.NoError(t, DB.First(&reloaded, inviter.Id).Error)
	require.Zero(t, reloaded.AffCount)
	require.Equal(t, 600, reloaded.Quota)
	require.Equal(t, 520, reloaded.GiftQuota)

	var ordinaryRelation OrdinaryInviteRelation
	require.NoError(t, DB.Where("inviter_user_id = ? AND invitee_user_id = ?", inviter.Id, invitee.Id).First(&ordinaryRelation).Error)
	require.Equal(t, OrdinaryInviteStatusActive, ordinaryRelation.Status)
	var affRelationCount int64
	require.NoError(t, DB.Model(&AffInviteRelation{}).Where("inviter_id = ? AND invitee_user_id = ?", inviter.Id, invitee.Id).Count(&affRelationCount).Error)
	require.Zero(t, affRelationCount)
	require.NoError(t, DB.First(&invitee, invitee.Id).Error)
	require.Zero(t, invitee.InviterId)

	// 普通用户阶段发生的充值不产生分成，也不会在升级后补发。
	ApplyAffiliateTopupReward(invitee.Id, 10000)
	require.NoError(t, DB.First(&reloaded, inviter.Id).Error)
	require.Zero(t, reloaded.AffQuota)
	require.Zero(t, reloaded.AffHistoryQuota)

	require.NoError(t, DB.Model(&User{}).Where("id = ?", inviter.Id).Updates(map[string]any{
		"is_distributor":             common.DistributorFlagYes,
		"distributor_commission_bps": 1500,
	}).Error)

	// 只升级身份但未确认转换时，历史普通邀请仍不产生分成。
	ApplyAffiliateTopupReward(invitee.Id, 10000)
	require.NoError(t, DB.First(&reloaded, inviter.Id).Error)
	require.Zero(t, reloaded.AffQuota)

	preview, err := GetOrdinaryInviteConversionPreview(inviter.Id)
	require.NoError(t, err)
	require.EqualValues(t, 1, preview.Convertible)
	var conversion *OrdinaryInviteConversionResult
	require.NoError(t, DB.Transaction(func(tx *gorm.DB) error {
		var convertErr error
		conversion, convertErr = ConvertOrdinaryInvitesToDistributor(tx, inviter.Id, 999)
		return convertErr
	}))
	require.EqualValues(t, 1, conversion.Converted)

	// 明确转换完成后的新充值，按升级后的当前比例结算。
	ApplyAffiliateTopupReward(invitee.Id, 10000)
	require.NoError(t, DB.First(&reloaded, inviter.Id).Error)
	require.Equal(t, 1500, reloaded.AffQuota)
	require.Equal(t, 1500, reloaded.AffHistoryQuota)

	var logs []AffInviteCommissionLog
	require.NoError(t, DB.Find(&logs).Error)
	require.Len(t, logs, 1)
	require.Equal(t, 1500, logs[0].RewardQuota)
	require.Equal(t, 1500, logs[0].CommissionBps)
	require.NoError(t, DB.First(&invitee, invitee.Id).Error)
	require.Equal(t, inviter.Id, invitee.InviterId)
	require.NoError(t, DB.First(&ordinaryRelation, ordinaryRelation.Id).Error)
	require.Equal(t, OrdinaryInviteStatusConverted, ordinaryRelation.Status)
}

func TestDistributorRegistrationCreatesOnlyDistributorRelation(t *testing.T) {
	setupOrdinaryInvitationTestDB(t)

	inviter := User{
		Username:      "distributor-inviter",
		AffCode:       "distributor-inviter-code",
		Role:          common.RoleCommonUser,
		Status:        common.UserStatusEnabled,
		IsDistributor: common.DistributorFlagYes,
	}
	invitee := User{Username: "direct-distributor-invitee", AffCode: "direct-distributor-invitee-code", Role: common.RoleCommonUser, Status: common.UserStatusEnabled}
	require.NoError(t, DB.Create(&inviter).Error)
	require.NoError(t, DB.Create(&invitee).Error)
	require.NoError(t, CreateRegistrationInviteRelation(inviter.Id, invitee.Id, common.QuotaForInviter))

	require.NoError(t, DB.First(&invitee, invitee.Id).Error)
	require.Equal(t, inviter.Id, invitee.InviterId)
	var ordinaryCount int64
	require.NoError(t, DB.Model(&OrdinaryInviteRelation{}).Count(&ordinaryCount).Error)
	require.Zero(t, ordinaryCount)
	var affCount int64
	require.NoError(t, DB.Model(&AffInviteRelation{}).Where("inviter_id = ? AND invitee_user_id = ?", inviter.Id, invitee.Id).Count(&affCount).Error)
	require.EqualValues(t, 1, affCount)
	require.NoError(t, DB.First(&inviter, inviter.Id).Error)
	require.Equal(t, 1, inviter.AffCount)
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
