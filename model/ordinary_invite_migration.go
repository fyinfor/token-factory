package model

import (
	"errors"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const ordinaryInviteSplitMigrationOption = "OrdinaryInviteRelationSplitMigratedV1"

// MigrateLegacyOrdinaryInvitesIfNeeded 将旧版“普通邀请也写 users.inviter_id”的关系拆到独立表。
// 只处理当前邀请人不是分销商、且关系尚无任何分销收益的记录；已产生收益的数据保持原状，避免破坏结算。
func MigrateLegacyOrdinaryInvitesIfNeeded() error {
	var option Option
	err := DB.Where("key = ?", ordinaryInviteSplitMigrationOption).First(&option).Error
	if err == nil && option.Value == "1" {
		return nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	return DB.Transaction(func(tx *gorm.DB) error {
		var invitees []User
		if err := tx.Select("id", "inviter_id").Where("inviter_id > ?", 0).Find(&invitees).Error; err != nil {
			return err
		}
		migratedCounts := make(map[int]int)
		now := common.GetTimestamp()
		for _, invitee := range invitees {
			var inviter User
			if err := tx.Select("id", "role", "is_distributor").Where("id = ?", invitee.InviterId).First(&inviter).Error; err != nil {
				continue
			}
			if UserIsDistributor(&inviter) {
				continue
			}
			var affRelation AffInviteRelation
			relErr := tx.Where("inviter_id = ? AND invitee_user_id = ?", inviter.Id, invitee.Id).First(&affRelation).Error
			if relErr != nil && !errors.Is(relErr, gorm.ErrRecordNotFound) {
				return relErr
			}
			if relErr == nil && (affRelation.CommissionEarnedQuota > 0 || affRelation.ProfitShareEarnedQuota > 0) {
				continue
			}
			ordinaryRelation := OrdinaryInviteRelation{
				InviterUserId: inviter.Id,
				InviteeUserId: invitee.Id,
				RewardQuota:   0,
				Status:        OrdinaryInviteStatusActive,
				CreatedAt:     now,
				UpdatedAt:     now,
			}
			if err := tx.Where("inviter_user_id = ? AND invitee_user_id = ?", inviter.Id, invitee.Id).
				FirstOrCreate(&ordinaryRelation).Error; err != nil {
				return err
			}
			if err := tx.Model(&User{}).Where("id = ? AND inviter_id = ?", invitee.Id, inviter.Id).
				Update("inviter_id", 0).Error; err != nil {
				return err
			}
			if relErr == nil {
				if err := tx.Delete(&affRelation).Error; err != nil {
					return err
				}
			}
			migratedCounts[inviter.Id]++
		}
		for inviterId := range migratedCounts {
			var count int64
			if err := tx.Model(&User{}).Where("inviter_id = ?", inviterId).Count(&count).Error; err != nil {
				return err
			}
			if err := tx.Model(&User{}).Where("id = ?", inviterId).Update("aff_count", count).Error; err != nil {
				return err
			}
		}
		option = Option{Key: ordinaryInviteSplitMigrationOption, Value: strconv.Itoa(1)}
		return tx.Save(&option).Error
	})
}
