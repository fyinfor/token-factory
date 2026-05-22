package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

// AffInviteProfitShareLog 利润分成模式下，单次请求结算产生的分销入账流水。
type AffInviteProfitShareLog struct {
	Id               int    `json:"id" gorm:"primaryKey;autoIncrement"`
	InviterId        int    `json:"inviter_id" gorm:"not null;index:idx_aff_ps_inv_inv,priority:1"`
	InviteeUserId    int    `json:"invitee_user_id" gorm:"not null;index:idx_aff_ps_inv_inv,priority:2;index"`
	ChannelId        int    `json:"channel_id" gorm:"not null;column:channel_id"`
	RouteSlug        string `json:"route_slug,omitempty" gorm:"-"`
	ModelName        string `json:"model_name" gorm:"type:varchar(255);not null;default:'';column:model_name"`
	UserQuotaCharged int    `json:"user_quota_charged" gorm:"not null;column:user_quota_charged"`
	MarkupSliceQuota int    `json:"markup_slice_quota" gorm:"not null;column:markup_slice_quota"`
	RewardQuota      int    `json:"reward_quota" gorm:"not null;column:reward_quota"`
	CreatedAt        int64  `json:"created_at" gorm:"bigint;index"`
}

func (AffInviteProfitShareLog) TableName() string {
	return "aff_invite_profit_share_logs"
}

// CreditDistributorProfitShare 将「加价切片」额度记入邀请人 aff_quota，并累计关系表 profit_share_earned_quota。
func CreditDistributorProfitShare(inviterId, inviteeUserId, channelId int, modelName string, userQuotaCharged, markupSliceQuota int) error {
	if inviterId <= 0 || inviteeUserId <= 0 || markupSliceQuota <= 0 {
		return nil
	}
	reward := markupSliceQuota
	modelName = strings.TrimSpace(modelName)
	ts := common.GetTimestamp()

	return DB.Transaction(func(tx *gorm.DB) error {
		up := tx.Model(&User{}).Where("id = ?", inviterId).Updates(map[string]interface{}{
			"aff_quota":   gorm.Expr("aff_quota + ?", reward),
			"aff_history": gorm.Expr("aff_history + ?", reward),
		})
		if up.Error != nil {
			return up.Error
		}
		if up.RowsAffected == 0 {
			return fmt.Errorf("inviter user not found: %d", inviterId)
		}
		res := tx.Model(&AffInviteRelation{}).
			Where("inviter_id = ? AND invitee_user_id = ?", inviterId, inviteeUserId).
			UpdateColumn("profit_share_earned_quota", gorm.Expr("profit_share_earned_quota + ?", reward))
		if res.Error != nil {
			return res.Error
		}
		row := AffInviteProfitShareLog{
			InviterId:        inviterId,
			InviteeUserId:    inviteeUserId,
			ChannelId:        channelId,
			ModelName:        modelName,
			UserQuotaCharged: userQuotaCharged,
			MarkupSliceQuota: markupSliceQuota,
			RewardQuota:      reward,
			CreatedAt:        ts,
		}
		return tx.Create(&row).Error
	})
}

// ListAffInviteProfitShareLogs 分页返回某邀请人对某一被邀请人的利润分成明细。
func ListAffInviteProfitShareLogs(inviterId, inviteeUserId int, pageInfo *common.PageInfo) ([]AffInviteProfitShareLog, int64, error) {
	if inviterId <= 0 || inviteeUserId <= 0 {
		return nil, 0, errors.New("invalid id")
	}
	var total int64
	base := DB.Model(&AffInviteProfitShareLog{}).Where("inviter_id = ? AND invitee_user_id = ?", inviterId, inviteeUserId)
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []AffInviteProfitShareLog
	err := DB.Where("inviter_id = ? AND invitee_user_id = ?", inviterId, inviteeUserId).
		Order("created_at DESC").
		Limit(pageInfo.GetPageSize()).
		Offset(pageInfo.GetStartIdx()).
		Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	if rows == nil {
		rows = []AffInviteProfitShareLog{}
	}
	attachProfitShareLogRouteSlugs(rows)
	return rows, total, nil
}

func attachProfitShareLogRouteSlugs(rows []AffInviteProfitShareLog) {
	if len(rows) == 0 {
		return
	}
	ids := make([]int, 0, len(rows))
	seen := make(map[int]struct{}, len(rows))
	for i := range rows {
		cid := rows[i].ChannelId
		if cid <= 0 {
			continue
		}
		if _, ok := seen[cid]; ok {
			continue
		}
		seen[cid] = struct{}{}
		ids = append(ids, cid)
	}
	slugMap := GetRouteSlugsByChannelIDs(ids)
	for i := range rows {
		cid := rows[i].ChannelId
		if cid <= 0 {
			continue
		}
		slug := ""
		if slugMap != nil {
			slug = strings.TrimSpace(slugMap[cid])
		}
		if slug == "" {
			slug = DefaultRouteSlugFromChannelID(int64(cid))
		}
		rows[i].RouteSlug = slug
	}
}
