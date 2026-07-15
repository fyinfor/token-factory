package model

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func resetDistributorAnalyticsTestData(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.AutoMigrate(
		&AffFunnelDaily{},
		&AffInviteRelation{},
		&AffInviteCommissionLog{},
		&AffInviteProfitShareLog{},
	))
	cleanup := func() {
		DB.Exec("DELETE FROM aff_invite_profit_share_logs")
		DB.Exec("DELETE FROM aff_invite_commission_logs")
		DB.Exec("DELETE FROM aff_invite_relations")
		DB.Exec("DELETE FROM aff_funnel_daily")
		DB.Exec("DELETE FROM users")
	}
	cleanup()
	t.Cleanup(cleanup)
}

func findAnalyticsDay(rows []DistributorAnalyticsDay, date string) DistributorAnalyticsDay {
	for _, row := range rows {
		if row.Date == date {
			return row
		}
	}
	return DistributorAnalyticsDay{}
}

func TestDistributorAnalyticsIncludesProfitShareLogs(t *testing.T) {
	resetDistributorAnalyticsTestData(t)

	today := time.Now().UTC().Truncate(24 * time.Hour)
	todayDate := today.Format("2006-01-02")
	todayTs := today.Add(2 * time.Hour).Unix()

	require.NoError(t, DB.Create(&User{
		Id:            101,
		Username:      "dist-a",
		Role:          common.RoleCommonUser,
		Status:        common.UserStatusEnabled,
		IsDistributor: common.DistributorFlagYes,
		AffCode:       "dist-a",
	}).Error)
	require.NoError(t, DB.Create(&User{
		Id:            102,
		Username:      "dist-b",
		Role:          common.RoleCommonUser,
		Status:        common.UserStatusEnabled,
		IsDistributor: common.DistributorFlagYes,
		AffCode:       "dist-b",
	}).Error)
	for _, u := range []User{
		{Id: 201, Username: "invitee-a", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, InviterId: 101, AffCode: "invitee-a"},
		{Id: 202, Username: "invitee-b", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, InviterId: 101, AffCode: "invitee-b"},
		{Id: 203, Username: "invitee-c", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, InviterId: 102, AffCode: "invitee-c"},
	} {
		user := u
		require.NoError(t, DB.Create(&user).Error)
	}
	require.NoError(t, DB.Create(&[]AffInviteRelation{
		{InviterId: 101, InviteeUserId: 201, CreatedAt: todayTs, UpdatedAt: todayTs},
		{InviterId: 101, InviteeUserId: 202, CreatedAt: todayTs, UpdatedAt: todayTs},
		{InviterId: 102, InviteeUserId: 203, CreatedAt: todayTs, UpdatedAt: todayTs},
	}).Error)
	require.NoError(t, DB.Create(&AffInviteCommissionLog{
		InviterId:         101,
		InviteeUserId:     201,
		InviteeQuotaAdded: 1000,
		CommissionBps:     1000,
		RewardQuota:       100,
		CreatedAt:         todayTs,
	}).Error)
	require.NoError(t, DB.Create(&[]AffInviteProfitShareLog{
		{
			InviterId:        101,
			InviteeUserId:    201,
			UserQuotaCharged: 2000,
			MarkupSliceQuota: 500,
			RewardQuota:      300,
			CommissionBps:    6000,
			CreatedAt:        todayTs,
		},
		{
			InviterId:        101,
			InviteeUserId:    202,
			UserQuotaCharged: 5000,
			MarkupSliceQuota: 700,
			RewardQuota:      500,
			CommissionBps:    7000,
			CreatedAt:        todayTs,
		},
		{
			InviterId:        102,
			InviteeUserId:    203,
			UserQuotaCharged: 8000,
			MarkupSliceQuota: 1000,
			RewardQuota:      800,
			CommissionBps:    8000,
			CreatedAt:        todayTs,
		},
	}).Error)

	selfSeries, err := BuildDistributorSelfAnalytics(101, 7)
	require.NoError(t, err)
	selfToday := findAnalyticsDay(selfSeries, todayDate)
	require.EqualValues(t, 900, selfToday.RewardQuota)
	require.EqualValues(t, 8000, selfToday.InviteeQuotaAdded)

	inviteeTop, err := ListInviteeTopForDistributorAnalytics(101, 10)
	require.NoError(t, err)
	require.Len(t, inviteeTop, 2)
	require.Equal(t, 202, inviteeTop[0].InviteeUserId)
	require.EqualValues(t, 500, inviteeTop[0].TotalRewardQuota)
	require.EqualValues(t, 5000, inviteeTop[0].TotalInviteeQuotaIn)
	require.Equal(t, 201, inviteeTop[1].InviteeUserId)
	require.EqualValues(t, 400, inviteeTop[1].TotalRewardQuota)
	require.EqualValues(t, 3000, inviteeTop[1].TotalInviteeQuotaIn)

	platformSeries, topTotal, topPeriod, _, err := BuildPlatformAffiliateAnalytics(7)
	require.NoError(t, err)
	platformToday := findAnalyticsDay(platformSeries, todayDate)
	require.EqualValues(t, 1700, platformToday.RewardQuota)
	require.EqualValues(t, 16000, platformToday.InviteeQuotaAdded)
	require.Len(t, topTotal, 2)
	require.Equal(t, 101, topTotal[0].UserId)
	require.EqualValues(t, 900, topTotal[0].TotalRewardQuota)
	require.Equal(t, 102, topTotal[1].UserId)
	require.EqualValues(t, 800, topTotal[1].TotalRewardQuota)
	require.Len(t, topPeriod, 2)
	require.Equal(t, 101, topPeriod[0].UserId)
	require.EqualValues(t, 900, topPeriod[0].TotalRewardQuota)
}
