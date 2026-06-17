package model

import (
	"errors"

	"github.com/QuantumNous/new-api/common"
)

type DistributorInviteeUnbindLog struct {
	Id                      int    `json:"id" gorm:"primaryKey;autoIncrement"`
	DistributorId           int    `json:"distributor_id" gorm:"not null;index"`
	InviteeUserId           int    `json:"invitee_user_id" gorm:"not null;index"`
	OperatorId              int    `json:"operator_id" gorm:"not null;index"`
	InviteeUsername         string `json:"invitee_username" gorm:"type:varchar(255)"`
	InviteeDisplayName      string `json:"invitee_display_name" gorm:"type:varchar(255)"`
	OperatorUsername        string `json:"operator_username" gorm:"type:varchar(255)"`
	Reason                  string `json:"reason" gorm:"type:text"`
	CommissionRatioBps      int    `json:"commission_ratio_bps"`
	CommissionEarnedQuota   int    `json:"commission_earned_quota"`
	ProfitShareEarnedQuota  int    `json:"profit_share_earned_quota"`
	ModelMarkupDiscountRate string `json:"model_markup_discount_rate" gorm:"type:text"`
	CreatedAt               int64  `json:"created_at" gorm:"autoCreateTime;bigint"`
}

func (DistributorInviteeUnbindLog) TableName() string {
	return "distributor_invitee_unbind_logs"
}

func ListDistributorInviteeUnbindLogs(distributorId int, pageInfo *common.PageInfo) ([]DistributorInviteeUnbindLog, int64, error) {
	if distributorId <= 0 {
		return nil, 0, errors.New("invalid distributor")
	}
	var total int64
	base := DB.Model(&DistributorInviteeUnbindLog{}).Where("distributor_id = ?", distributorId)
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []DistributorInviteeUnbindLog
	err := DB.Where("distributor_id = ?", distributorId).
		Order("id desc").
		Limit(pageInfo.GetPageSize()).
		Offset(pageInfo.GetStartIdx()).
		Find(&rows).Error
	return rows, total, err
}

func HasDistributorInviteeUnbindLog(distributorId, inviteeUserId int) (bool, error) {
	if distributorId <= 0 || inviteeUserId <= 0 {
		return false, nil
	}
	var count int64
	err := DB.Model(&DistributorInviteeUnbindLog{}).
		Where("distributor_id = ? AND invitee_user_id = ?", distributorId, inviteeUserId).
		Count(&count).Error
	return count > 0, err
}
