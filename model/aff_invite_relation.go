package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"

	"gorm.io/gorm"
)

// AffInviteRelation 邀请人与被邀请人关系表：为每个被邀请人单独配置充值分销比例。
// CommissionRatioBps 存储单位为万分之一（相对于「百分比」）：1 表示 0.01%，100 表示 1%，10000 表示 100%。
type AffInviteRelation struct {
	Id                 int   `json:"id" gorm:"primaryKey;autoIncrement"`
	InviterId          int   `json:"inviter_id" gorm:"not null;uniqueIndex:idx_aff_inv_pair"`
	InviteeUserId      int   `json:"invitee_user_id" gorm:"not null;uniqueIndex:idx_aff_inv_pair;column:invitee_user_id"`
	CommissionRatioBps int   `json:"commission_ratio_bps" gorm:"not null;default:0;column:commission_ratio_bps"`
	// 自动时间戳：创建/更新时 GORM 自动赋值
	CreatedAt          int64 `json:"created_at" gorm:"autoCreateTime;bigint;comment:创建时间"`
	UpdatedAt          int64 `json:"updated_at" gorm:"autoUpdateTime;bigint;comment:更新时间"`
}

func (AffInviteRelation) TableName() string {
	return "aff_invite_relations"
}

const maxAffiliateCommissionBps = 10000

// AffInviteeListItem 邀请人视角下的被邀请人列表项
type AffInviteeListItem struct {
	InviteeId          int    `json:"invitee_id"`
	Username           string `json:"username"`
	DisplayName        string `json:"display_name"`
	CommissionRatioBps int    `json:"commission_ratio_bps"` // 万分之一单位（1=0.01%），前端展示为百分比
}

// EnsureAffInviteRelation 注册成功后建立关系行，比例初始为系统默认 AffiliateDefaultCommissionBps。
func EnsureAffInviteRelation(inviterId, inviteeUserId int) error {
	if inviterId <= 0 || inviteeUserId <= 0 {
		return nil
	}
	var cnt int64
	err := DB.Model(&AffInviteRelation{}).Where("inviter_id = ? AND invitee_user_id = ?", inviterId, inviteeUserId).Count(&cnt).Error
	if err != nil {
		return err
	}
	if cnt > 0 {
		return nil
	}
	ts := common.GetTimestamp()
	rel := AffInviteRelation{
		InviterId:          inviterId,
		InviteeUserId:      inviteeUserId,
		CommissionRatioBps: common.AffiliateDefaultCommissionBps,
		CreatedAt:          ts,
		UpdatedAt:          ts,
	}
	return DB.Create(&rel).Error
}

// BackfillAffInviteRelationsIfNeeded 表为空时执行一次历史数据补全，避免每次启动全表扫描。
func BackfillAffInviteRelationsIfNeeded() error {
	var cnt int64
	if err := DB.Model(&AffInviteRelation{}).Count(&cnt).Error; err != nil {
		return err
	}
	if cnt > 0 {
		return nil
	}
	return BackfillAffInviteRelationsFromUsers()
}

// BackfillAffInviteRelationsFromUsers 为历史数据补全关系行。
func BackfillAffInviteRelationsFromUsers() error {
	var users []User
	err := DB.Unscoped().Model(&User{}).Select("id", "inviter_id").Where("inviter_id > ?", 0).Find(&users).Error
	if err != nil {
		return err
	}
	for i := range users {
		if err := EnsureAffInviteRelation(users[i].InviterId, users[i].Id); err != nil {
			common.SysError("backfill aff_invite_relations: " + err.Error())
		}
	}
	return nil
}

func resolveCommissionBps(inviterId, inviteeId int) int {
	var rel AffInviteRelation
	err := DB.Where("inviter_id = ? AND invitee_user_id = ?", inviterId, inviteeId).First(&rel).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.AffiliateDefaultCommissionBps
		}
		common.SysError("resolveCommissionBps: " + err.Error())
		return 0
	}
	// 历史数据中 0 表示未按新默认落库，按系统默认比例计奖；显式配置由管理员在后台写入正数比例
	if rel.CommissionRatioBps <= 0 {
		return common.AffiliateDefaultCommissionBps
	}
	return rel.CommissionRatioBps
}

// ApplyAffiliateTopupReward 被邀请用户获得充值额度 quotaAdded 后，按邀请关系比例将提成记入邀请人 aff_quota / aff_history（与 resolveCommissionBps 一致，不增加 quota）。
// 须在支付回调完成入账后调用，与订单事务解耦。
func ApplyAffiliateTopupReward(inviteeUserId int, quotaAdded int) {
	if inviteeUserId <= 0 || quotaAdded <= 0 {
		return
	}
	invitee, err := GetUserById(inviteeUserId, false)
	if err != nil {
		return
	}
	inviterId := invitee.InviterId
	if inviterId <= 0 {
		return
	}
	bps := resolveCommissionBps(inviterId, inviteeUserId)
	if bps <= 0 {
		return
	}
	if bps > maxAffiliateCommissionBps {
		bps = maxAffiliateCommissionBps
	}
	reward := int(int64(quotaAdded) * int64(bps) / int64(maxAffiliateCommissionBps))
	if reward <= 0 {
		return
	}
	if err := IncreaseUserAffCommissionQuota(inviterId, reward); err != nil {
		common.SysError(fmt.Sprintf("ApplyAffiliateTopupReward: inviter=%d invitee=%d reward=%d err=%v", inviterId, inviteeUserId, reward, err))
		return
	}
	inviteeLabel := strings.TrimSpace(invitee.Username)
	if inviteeLabel == "" {
		inviteeLabel = strings.TrimSpace(invitee.DisplayName)
	}
	if inviteeLabel == "" {
		inviteeLabel = fmt.Sprintf("ID:%d", invitee.Id)
	}
	pct := logger.FormatCommissionRatioAsPercent(bps)
	amt := logger.LogQuotaConcise(reward)
	RecordLog(inviterId, LogTypeSystem, fmt.Sprintf("邀请分销奖励（被邀请用户 %s 充值）%s，分成比例 %s", inviteeLabel, amt, pct))
}

// ListAffInvitees 分页返回当前用户邀请注册的用户及其分销比例。
func ListAffInvitees(inviterId int, pageInfo *common.PageInfo) ([]AffInviteeListItem, int64, error) {
	if inviterId <= 0 {
		return nil, 0, errors.New("invalid inviter")
	}
	var total int64
	tx := DB.Model(&User{}).Where("inviter_id = ?", inviterId)
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var users []User
	err := DB.Where("inviter_id = ?", inviterId).Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&users).Error
	if err != nil {
		return nil, 0, err
	}
	if len(users) == 0 {
		return []AffInviteeListItem{}, total, nil
	}
	ids := make([]int, 0, len(users))
	for _, u := range users {
		ids = append(ids, u.Id)
	}
	var rels []AffInviteRelation
	_ = DB.Where("inviter_id = ? AND invitee_user_id IN ?", inviterId, ids).Find(&rels).Error
	bpsMap := make(map[int]int, len(rels))
	for _, r := range rels {
		bpsMap[r.InviteeUserId] = r.CommissionRatioBps
	}
	defaultBps := common.AffiliateDefaultCommissionBps
	items := make([]AffInviteeListItem, 0, len(users))
	for _, u := range users {
		bps, ok := bpsMap[u.Id]
		if !ok {
			bps = defaultBps
		} else if bps <= 0 {
			bps = defaultBps
		}
		items = append(items, AffInviteeListItem{
			InviteeId:          u.Id,
			Username:           u.Username,
			DisplayName:        u.DisplayName,
			CommissionRatioBps: bps,
		})
	}
	return items, total, nil
}

// UpdateAffInviteeCommission 邀请人修改某一被邀请人的分销比例（验证被邀请人确实属于当前邀请人）。
func UpdateAffInviteeCommission(inviterId, inviteeUserId, commissionBps int) error {
	if inviterId <= 0 || inviteeUserId <= 0 {
		return errors.New("invalid id")
	}
	if commissionBps < 0 || commissionBps > maxAffiliateCommissionBps {
		return fmt.Errorf("commission_ratio_bps must be 0..%d (万分之一单位，1=0.01%%)", maxAffiliateCommissionBps)
	}
	invitee, err := GetUserById(inviteeUserId, false)
	if err != nil {
		return errors.New("user not found")
	}
	if invitee.InviterId != inviterId {
		return errors.New("not your invitee")
	}
	ts := common.GetTimestamp()
	var rel AffInviteRelation
	err = DB.Where("inviter_id = ? AND invitee_user_id = ?", inviterId, inviteeUserId).First(&rel).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		rel = AffInviteRelation{
			InviterId:          inviterId,
			InviteeUserId:      inviteeUserId,
			CommissionRatioBps: commissionBps,
			CreatedAt:          ts,
			UpdatedAt:          ts,
		}
		return DB.Create(&rel).Error
	}
	if err != nil {
		return err
	}
	rel.CommissionRatioBps = commissionBps
	rel.UpdatedAt = ts
	return DB.Save(&rel).Error
}
