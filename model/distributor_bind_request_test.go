package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func TestAdminBindDistributorInviteeDirectly(t *testing.T) {
	setupOrdinaryInvitationTestDB(t)
	require.NoError(t, DB.AutoMigrate(&DistributorBindRequest{}, &UserMessage{}))

	distributor := User{
		Username:                               "direct-bind-distributor",
		AffCode:                                "direct-bind-distributor-code",
		Role:                                   common.RoleCommonUser,
		Status:                                 common.UserStatusEnabled,
		IsDistributor:                          common.DistributorFlagYes,
		DistributorModelDiscountAutoApply:      common.DistributorFlagYes,
		DistributorModelMarkupDiscountTemplate: `[{"model_name":"gpt-test","channel_id":1,"markup_discount_rate":80}]`,
	}
	require.NoError(t, DB.Create(&distributor).Error)
	target := User{Username: "direct-bind-target", AffCode: "direct-bind-target-code", Role: common.RoleCommonUser, Status: common.UserStatusEnabled}
	require.NoError(t, DB.Create(&target).Error)

	pending := DistributorBindRequest{
		DistributorUserID: distributor.Id,
		TargetUserID:      target.Id,
		Status:            DistributorBindRequestStatusPending,
		CreatedAt:         1,
		UpdatedAt:         1,
	}
	require.NoError(t, DB.Create(&pending).Error)
	message := UserMessage{
		ReceiverUserID: target.Id,
		Type:           UserMessageTypeDistributorBindRequest,
		Title:          "代理绑定确认",
		Content:        "待确认",
		BizType:        UserMessageBizTypeDistributorBindRequest,
		BizID:          pending.ID,
		CreatedAt:      1,
	}
	require.NoError(t, DB.Create(&message).Error)
	require.NoError(t, DB.Model(&pending).Update("message_id", message.ID).Error)

	require.NoError(t, AdminBindDistributorInvitee(distributor.Id, target.Id))

	var boundTarget User
	require.NoError(t, DB.First(&boundTarget, target.Id).Error)
	require.Equal(t, distributor.Id, boundTarget.InviterId)
	var boundDistributor User
	require.NoError(t, DB.First(&boundDistributor, distributor.Id).Error)
	require.Equal(t, 1, boundDistributor.AffCount)
	var relation AffInviteRelation
	require.NoError(t, DB.Where("inviter_id = ? AND invitee_user_id = ?", distributor.Id, target.Id).First(&relation).Error)
	require.Equal(t, distributor.DistributorModelMarkupDiscountTemplate, relation.ModelMarkupDiscountRate)
	var reloadedPending DistributorBindRequest
	require.NoError(t, DB.First(&reloadedPending, pending.ID).Error)
	require.Equal(t, DistributorBindRequestStatusSuperseded, reloadedPending.Status)
	var reloadedMessage UserMessage
	require.NoError(t, DB.First(&reloadedMessage, message.ID).Error)
	require.True(t, reloadedMessage.IsRead)

	err := AdminBindDistributorInvitee(distributor.Id, target.Id)
	require.EqualError(t, err, "该用户已绑定代理")
	require.NoError(t, DB.First(&boundDistributor, distributor.Id).Error)
	require.Equal(t, 1, boundDistributor.AffCount)
}

func TestAdminBindDistributorInviteeRejectsIneligibleUser(t *testing.T) {
	setupOrdinaryInvitationTestDB(t)

	distributor := User{Username: "eligible-distributor", AffCode: "eligible-distributor-code", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, IsDistributor: common.DistributorFlagYes}
	require.NoError(t, DB.Create(&distributor).Error)
	anotherDistributor := User{Username: "nested-distributor", AffCode: "nested-distributor-code", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, IsDistributor: common.DistributorFlagYes}
	require.NoError(t, DB.Create(&anotherDistributor).Error)
	admin := User{Username: "admin-target", AffCode: "admin-target-code", Role: common.RoleAdminUser, Status: common.UserStatusEnabled}
	require.NoError(t, DB.Create(&admin).Error)

	require.EqualError(t, AdminBindDistributorInvitee(distributor.Id, anotherDistributor.Id), "不允许多级代理")
	require.EqualError(t, AdminBindDistributorInvitee(distributor.Id, admin.Id), "只能绑定普通用户")
}
