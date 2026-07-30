package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestCreditDistributorProfitShareRecordsZeroRewardWithoutCreditingBalance(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&AffInviteRelation{}, &AffInviteProfitShareLog{}))
	DB.Exec("DELETE FROM aff_invite_profit_share_logs")
	DB.Exec("DELETE FROM aff_invite_relations")
	DB.Exec("DELETE FROM users")
	t.Cleanup(func() {
		DB.Exec("DELETE FROM aff_invite_profit_share_logs")
		DB.Exec("DELETE FROM aff_invite_relations")
		DB.Exec("DELETE FROM users")
	})

	require.NoError(t, DB.Create(&User{
		Id:              801,
		Username:        "zero-reward-dist",
		Role:            common.RoleCommonUser,
		Status:          common.UserStatusEnabled,
		AffQuota:        123,
		AffCode:         "zero-reward-dist-code",
		AffHistoryQuota: 456,
	}).Error)
	require.NoError(t, DB.Create(&User{
		Id:        802,
		Username:  "zero-reward-invitee",
		Role:      common.RoleCommonUser,
		Status:    common.UserStatusEnabled,
		InviterId: 801,
		AffCode:   "zero-reward-invitee-code",
	}).Error)
	require.NoError(t, DB.Create(&AffInviteRelation{
		InviterId:              801,
		InviteeUserId:          802,
		CommissionRatioBps:     1000,
		ProfitShareEarnedQuota: 789,
	}).Error)

	require.NoError(t, CreditDistributorProfitShare(801, 802, 9, "small-text-model", 1, 0, 0, 1000, 2, "text"))

	var inviter User
	require.NoError(t, DB.First(&inviter, 801).Error)
	require.Equal(t, 123, inviter.AffQuota)
	require.Equal(t, 456, inviter.AffHistoryQuota)

	var relation AffInviteRelation
	require.NoError(t, DB.Where("inviter_id = ? AND invitee_user_id = ?", 801, 802).First(&relation).Error)
	require.Equal(t, 789, relation.ProfitShareEarnedQuota)

	var logs []AffInviteProfitShareLog
	require.NoError(t, DB.Find(&logs).Error)
	require.Len(t, logs, 1)
	require.Equal(t, 1, logs[0].UserQuotaCharged)
	require.Zero(t, logs[0].MarkupSliceQuota)
	require.Zero(t, logs[0].RewardQuota)
	require.Equal(t, "text", logs[0].BillingMode)
}

func TestListAffInviteProfitShareLogsCanHideZeroReward(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&AffInviteProfitShareLog{}))
	DB.Exec("DELETE FROM aff_invite_profit_share_logs")
	t.Cleanup(func() {
		DB.Exec("DELETE FROM aff_invite_profit_share_logs")
	})

	require.NoError(t, DB.Create(&[]AffInviteProfitShareLog{
		{InviterId: 901, InviteeUserId: 902, ModelName: "zero", RewardQuota: 0, CreatedAt: 1},
		{InviterId: 901, InviteeUserId: 902, ModelName: "positive", RewardQuota: 10, CreatedAt: 2},
	}).Error)

	pageInfo := &common.PageInfo{Page: 1, PageSize: 10}
	all, total, err := ListAffInviteProfitShareLogs(901, 902, "", false, pageInfo)
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	require.Len(t, all, 2)

	withReward, filteredTotal, err := ListAffInviteProfitShareLogs(901, 902, "", true, pageInfo)
	require.NoError(t, err)
	require.EqualValues(t, 1, filteredTotal)
	require.Len(t, withReward, 1)
	require.Equal(t, "positive", withReward[0].ModelName)
}
