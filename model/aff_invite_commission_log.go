package model

import (
	"errors"

	"github.com/QuantumNous/new-api/common"
)

// AffInviteCommissionLog 单次充值产生的分销记录（供分销商查看「按笔」明细）。
type AffInviteCommissionLog struct {
	Id                int   `json:"id" gorm:"primaryKey;autoIncrement"`
	InviterId         int   `json:"inviter_id" gorm:"not null;index:idx_aff_comm_inv_inv,priority:1"`
	InviteeUserId     int   `json:"invitee_user_id" gorm:"not null;index:idx_aff_comm_inv_inv,priority:2;index"`
	InviteeQuotaAdded int   `json:"invitee_quota_added" gorm:"not null;column:invitee_quota_added"` // 被邀请用户本次充值入账的额度（与 ApplyAffiliateTopupReward 的 quotaAdded 一致）
	CommissionBps     int   `json:"commission_bps" gorm:"not null;column:commission_bps"`         // 当时采用的万分之一比例
	RewardQuota       int   `json:"reward_quota" gorm:"not null;column:reward_quota"`               // 邀请人本次获得的 aff 额度
	CreatedAt         int64 `json:"created_at" gorm:"bigint;index"`
}

func (AffInviteCommissionLog) TableName() string {
	return "aff_invite_commission_logs"
}

// InsertAffInviteCommissionLog 写入单笔分销明细（与 IncreaseUserAffCommissionQuota 成功后在同一逻辑路径调用）。
func InsertAffInviteCommissionLog(inviterId, inviteeUserId, inviteeQuotaAdded, commissionBps, rewardQuota int) error {
	if inviterId <= 0 || inviteeUserId <= 0 || inviteeQuotaAdded <= 0 || rewardQuota <= 0 {
		return nil
	}
	row := AffInviteCommissionLog{
		InviterId:         inviterId,
		InviteeUserId:     inviteeUserId,
		InviteeQuotaAdded: inviteeQuotaAdded,
		CommissionBps:     commissionBps,
		RewardQuota:       rewardQuota,
		CreatedAt:         common.GetTimestamp(),
	}
	return DB.Create(&row).Error
}

// ListAffInviteCommissionLogs 分页返回某邀请人对某一被邀请人的充值分成明细。
func ListAffInviteCommissionLogs(inviterId, inviteeUserId int, pageInfo *common.PageInfo) ([]AffInviteCommissionLog, int64, error) {
	if inviterId <= 0 || inviteeUserId <= 0 {
		return nil, 0, errors.New("invalid id")
	}
	var total int64
	base := DB.Model(&AffInviteCommissionLog{}).Where("inviter_id = ? AND invitee_user_id = ?", inviterId, inviteeUserId)
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []AffInviteCommissionLog
	err := DB.Where("inviter_id = ? AND invitee_user_id = ?", inviterId, inviteeUserId).
		Order("created_at DESC").
		Limit(pageInfo.GetPageSize()).
		Offset(pageInfo.GetStartIdx()).
		Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	if rows == nil {
		rows = []AffInviteCommissionLog{}
	}
	return rows, total, nil
}
