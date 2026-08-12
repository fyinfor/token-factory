package model

import (
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	OrdinaryInviteStatusActive    = 1
	OrdinaryInviteStatusConverted = 2
)

// OrdinaryInviteRelation 记录普通用户的注册邀请关系。
// 它与分销上下级关系彻底分离：只有 converted 后，才会另外创建 users.inviter_id / aff_invite_relations。
type OrdinaryInviteRelation struct {
	Id                    int   `json:"id" gorm:"primaryKey;autoIncrement"`
	InviterUserId         int   `json:"inviter_user_id" gorm:"not null;index;uniqueIndex:idx_ordinary_invite_pair"`
	InviteeUserId         int   `json:"invitee_user_id" gorm:"not null;index;uniqueIndex:idx_ordinary_invite_pair"`
	RewardQuota           int   `json:"reward_quota" gorm:"not null;default:0"`
	Status                int   `json:"status" gorm:"not null;default:1;index"`
	ConvertedAt           int64 `json:"converted_at" gorm:"not null;default:0"`
	ConvertedBy           int   `json:"converted_by" gorm:"not null;default:0"`
	DistributorRelationId int   `json:"distributor_relation_id" gorm:"not null;default:0"`
	CreatedAt             int64 `json:"created_at" gorm:"autoCreateTime;bigint"`
	UpdatedAt             int64 `json:"updated_at" gorm:"autoUpdateTime;bigint"`
}

func (OrdinaryInviteRelation) TableName() string {
	return "ordinary_invite_relations"
}

type OrdinaryInviteConversionPreview struct {
	Total        int64 `json:"total"`
	Convertible  int64 `json:"convertible"`
	AlreadyBound int64 `json:"already_bound"`
	Ineligible   int64 `json:"ineligible"`
}

type OrdinaryInviteConversionResult struct {
	Requested int64 `json:"requested"`
	Converted int64 `json:"converted"`
	Skipped   int64 `json:"skipped"`
}

// CreateRegistrationInviteRelation 根据邀请人当前身份创建注册邀请关系。
// 分销商邀请直接成为正式下级；普通用户邀请只进入 ordinary_invite_relations。
func CreateRegistrationInviteRelation(inviterUserId, inviteeUserId, rewardQuota int) error {
	if inviterUserId <= 0 || inviteeUserId <= 0 || inviterUserId == inviteeUserId {
		return errors.New("invalid registration invitation")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var inviter User
		if err := tx.Where("id = ?", inviterUserId).First(&inviter).Error; err != nil {
			return err
		}
		if !UserIsDistributor(&inviter) {
			now := common.GetTimestamp()
			relation := OrdinaryInviteRelation{
				InviterUserId: inviterUserId,
				InviteeUserId: inviteeUserId,
				RewardQuota:   rewardQuota,
				Status:        OrdinaryInviteStatusActive,
				CreatedAt:     now,
				UpdatedAt:     now,
			}
			return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&relation).Error
		}

		var invitee User
		if err := tx.Select("id", "role", "is_distributor", "inviter_id").Where("id = ?", inviteeUserId).First(&invitee).Error; err != nil {
			return err
		}
		if invitee.Role != common.RoleCommonUser || UserIsDistributor(&invitee) {
			return errors.New("invitee cannot be bound as distributor invitee")
		}
		if invitee.InviterId != 0 && invitee.InviterId != inviterUserId {
			return errors.New("invitee already bound to another distributor")
		}
		if invitee.InviterId == 0 {
			res := tx.Model(&User{}).Where("id = ? AND inviter_id = ?", inviteeUserId, 0).
				Update("inviter_id", inviterUserId)
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return errors.New("invitee binding changed")
			}
		}

		var count int64
		if err := tx.Model(&AffInviteRelation{}).
			Where("inviter_id = ? AND invitee_user_id = ?", inviterUserId, inviteeUserId).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return nil
		}
		now := common.GetTimestamp()
		affRelation := AffInviteRelation{
			InviterId:               inviterUserId,
			InviteeUserId:           inviteeUserId,
			CommissionRatioBps:      defaultCommissionBpsForNewInviteRelation(inviterUserId),
			CommissionEarnedQuota:   0,
			ProfitShareEarnedQuota:  0,
			ModelMarkupDiscountRate: defaultModelMarkupDiscountRateForNewInviteRelation(tx, inviterUserId),
			CreatedAt:               now,
			UpdatedAt:               now,
		}
		if err := tx.Create(&affRelation).Error; err != nil {
			return err
		}
		return tx.Model(&User{}).Where("id = ?", inviterUserId).
			UpdateColumn("aff_count", gorm.Expr("aff_count + ?", 1)).Error
	})
}

func CountOrdinaryInvitees(inviterUserId int) (int64, error) {
	if inviterUserId <= 0 {
		return 0, errors.New("invalid inviter")
	}
	var count int64
	err := DB.Model(&OrdinaryInviteRelation{}).
		Where("inviter_user_id = ?", inviterUserId).
		Count(&count).Error
	return count, err
}

func GetOrdinaryInviteConversionPreview(inviterUserId int) (*OrdinaryInviteConversionPreview, error) {
	if inviterUserId <= 0 {
		return nil, errors.New("invalid inviter")
	}
	var relations []OrdinaryInviteRelation
	if err := DB.Where("inviter_user_id = ? AND status = ?", inviterUserId, OrdinaryInviteStatusActive).
		Find(&relations).Error; err != nil {
		return nil, err
	}
	preview := &OrdinaryInviteConversionPreview{Total: int64(len(relations))}
	if len(relations) == 0 {
		return preview, nil
	}
	ids := make([]int, 0, len(relations))
	for _, relation := range relations {
		ids = append(ids, relation.InviteeUserId)
	}
	var users []User
	if err := DB.Select("id", "role", "is_distributor", "inviter_id").Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}
	userMap := make(map[int]User, len(users))
	for _, user := range users {
		userMap[user.Id] = user
	}
	for _, relation := range relations {
		user, ok := userMap[relation.InviteeUserId]
		if !ok || user.Id == inviterUserId || user.Role != common.RoleCommonUser || UserIsDistributor(&user) {
			preview.Ineligible++
			continue
		}
		if user.InviterId > 0 {
			preview.AlreadyBound++
			continue
		}
		preview.Convertible++
	}
	return preview, nil
}

// ConvertOrdinaryInvitesToDistributor 将普通邀请转换为正式分销下级。
// 只转换仍未绑定代理的普通用户，且从此刻起才会参与后续分成；历史奖励与消费不做追溯。
func ConvertOrdinaryInvitesToDistributor(tx *gorm.DB, inviterUserId, operatorId int) (*OrdinaryInviteConversionResult, error) {
	if tx == nil || inviterUserId <= 0 || operatorId <= 0 {
		return nil, errors.New("invalid conversion params")
	}
	var inviter User
	if err := tx.Where("id = ?", inviterUserId).First(&inviter).Error; err != nil {
		return nil, err
	}
	if !UserIsDistributor(&inviter) {
		return nil, errors.New("user is not distributor")
	}
	var relations []OrdinaryInviteRelation
	if err := tx.Where("inviter_user_id = ? AND status = ?", inviterUserId, OrdinaryInviteStatusActive).
		Order("id asc").Find(&relations).Error; err != nil {
		return nil, err
	}
	result := &OrdinaryInviteConversionResult{Requested: int64(len(relations))}
	now := common.GetTimestamp()
	for _, ordinaryRelation := range relations {
		var invitee User
		err := tx.Where("id = ?", ordinaryRelation.InviteeUserId).First(&invitee).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			result.Skipped++
			continue
		}
		if err != nil {
			return nil, err
		}
		if invitee.Id == inviterUserId || invitee.Role != common.RoleCommonUser || UserIsDistributor(&invitee) || invitee.InviterId > 0 {
			result.Skipped++
			continue
		}
		res := tx.Model(&User{}).
			Where("id = ? AND inviter_id = ?", invitee.Id, 0).
			Update("inviter_id", inviterUserId)
		if res.Error != nil {
			return nil, res.Error
		}
		if res.RowsAffected == 0 {
			result.Skipped++
			continue
		}
		affRelation := AffInviteRelation{
			InviterId:               inviterUserId,
			InviteeUserId:           invitee.Id,
			CommissionRatioBps:      defaultCommissionBpsForNewInviteRelation(inviterUserId),
			CommissionEarnedQuota:   0,
			ProfitShareEarnedQuota:  0,
			ModelMarkupDiscountRate: defaultModelMarkupDiscountRateForNewInviteRelation(tx, inviterUserId),
			CreatedAt:               now,
			UpdatedAt:               now,
		}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&affRelation).Error; err != nil {
			return nil, err
		}
		if affRelation.Id == 0 {
			if err := tx.Where("inviter_id = ? AND invitee_user_id = ?", inviterUserId, invitee.Id).
				First(&affRelation).Error; err != nil {
				return nil, err
			}
		}
		updates := map[string]any{
			"status":                  OrdinaryInviteStatusConverted,
			"converted_at":            now,
			"converted_by":            operatorId,
			"distributor_relation_id": affRelation.Id,
			"updated_at":              now,
		}
		res = tx.Model(&OrdinaryInviteRelation{}).
			Where("id = ? AND status = ?", ordinaryRelation.Id, OrdinaryInviteStatusActive).
			Updates(updates)
		if res.Error != nil {
			return nil, res.Error
		}
		if res.RowsAffected == 0 {
			return nil, fmt.Errorf("ordinary invitation changed: %d", ordinaryRelation.Id)
		}
		result.Converted++
	}
	result.Skipped = result.Requested - result.Converted
	var distributorInviteeCount int64
	if err := tx.Model(&User{}).Where("inviter_id = ?", inviterUserId).Count(&distributorInviteeCount).Error; err != nil {
		return nil, err
	}
	if err := tx.Model(&User{}).Where("id = ?", inviterUserId).
		Update("aff_count", distributorInviteeCount).Error; err != nil {
		return nil, err
	}
	return result, nil
}
