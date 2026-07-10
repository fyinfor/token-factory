package model

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/bytedance/gopkg/util/gopool"
	"gorm.io/gorm"
)

// ErrRedeemFailed is returned when redemption fails due to database error.
var ErrRedeemFailed = errors.New("redeem.failed")

var ErrRedemptionQuotaInsufficient = errors.New("quota insufficient")

type Redemption struct {
	Id           int            `json:"id"`
	UserId       int            `json:"user_id"`
	Key          string         `json:"key" gorm:"type:char(32);uniqueIndex"`
	Status       int            `json:"status" gorm:"default:1"`
	Name         string         `json:"name" gorm:"index"`
	Quota        int            `json:"quota" gorm:"default:100"`
	CreatedTime  int64          `json:"created_time" gorm:"bigint"`
	RedeemedTime int64          `json:"redeemed_time" gorm:"bigint"`
	Count        int            `json:"count" gorm:"-:all"`
	UsedUserId   int            `json:"used_user_id"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`
	ExpiredTime  int64          `json:"expired_time" gorm:"bigint"`

	CreatorName  string `json:"creator_name" gorm:"column:creator_name;->;-:migration"`
	RedeemerName string `json:"redeemer_name" gorm:"column:redeemer_name;->;-:migration"`
}

type RedemptionQueryOptions struct {
	OperatorId     int
	OperatorRole   int
	Keyword        string
	StartTimestamp int64
	EndTimestamp   int64
}

func redemptionOperatorIsAdmin(role int) bool {
	return role >= common.RoleAdminUser
}

func RedemptionOperatorAllowed(userId int, role int) bool {
	if redemptionOperatorIsAdmin(role) {
		return true
	}
	user, err := GetUserById(userId, false)
	return err == nil && UserIsDistributor(user)
}

func syncUserQuotaCacheForRedemption(userId int, delta int) {
	if userId <= 0 || delta == 0 {
		return
	}
	gopool.Go(func() {
		var err error
		if delta > 0 {
			err = cacheIncrUserQuota(userId, int64(delta))
		} else {
			err = cacheDecrUserQuota(userId, int64(-delta))
		}
		if err != nil {
			common.SysLog("failed to sync user quota cache for redemption: " + err.Error())
		}
	})
}

func redemptionBaseQuery(tx *gorm.DB, opts RedemptionQueryOptions) *gorm.DB {
	query := tx.Model(&Redemption{}).
		Select("redemptions.*, creator.username AS creator_name, redeemer.username AS redeemer_name").
		Joins("LEFT JOIN users creator ON creator.id = redemptions.user_id").
		Joins("LEFT JOIN users redeemer ON redeemer.id = redemptions.used_user_id")

	if !redemptionOperatorIsAdmin(opts.OperatorRole) {
		query = query.Where("redemptions.user_id = ?", opts.OperatorId)
	}
	if opts.StartTimestamp > 0 {
		query = query.Where("redemptions.created_time >= ?", opts.StartTimestamp)
	}
	if opts.EndTimestamp > 0 {
		query = query.Where("redemptions.created_time <= ?", opts.EndTimestamp)
	}

	keyword := strings.TrimSpace(opts.Keyword)
	if keyword == "" {
		return query
	}

	like := keyword + "%"
	if id, err := strconv.Atoi(keyword); err == nil {
		return query.Where(
			"redemptions.id = ? OR redemptions.user_id = ? OR redemptions.used_user_id = ? OR redemptions.name LIKE ? OR redemptions.key LIKE ? OR creator.username LIKE ? OR creator.display_name LIKE ? OR redeemer.username LIKE ? OR redeemer.display_name LIKE ?",
			id, id, id, like, like, like, like, like, like,
		)
	}
	return query.Where(
		"redemptions.name LIKE ? OR redemptions.key LIKE ? OR creator.username LIKE ? OR creator.display_name LIKE ? OR redeemer.username LIKE ? OR redeemer.display_name LIKE ?",
		like, like, like, like, like, like,
	)
}

func GetAllRedemptions(opts RedemptionQueryOptions, startIdx int, num int) (redemptions []*Redemption, total int64, err error) {
	if err = SettleExpiredRedemptions(opts.OperatorId, opts.OperatorRole); err != nil {
		return nil, 0, err
	}

	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	query := redemptionBaseQuery(tx, opts)
	if err = query.Select("redemptions.id").Count(&total).Error; err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	query = redemptionBaseQuery(tx, opts)
	if err = query.Order("redemptions.created_time desc, redemptions.id desc").Limit(num).Offset(startIdx).Find(&redemptions).Error; err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}
	return redemptions, total, nil
}

func SearchRedemptions(opts RedemptionQueryOptions, startIdx int, num int) (redemptions []*Redemption, total int64, err error) {
	return GetAllRedemptions(opts, startIdx, num)
}

func GetRedemptionById(id int, operatorId int, operatorRole int) (*Redemption, error) {
	if id == 0 {
		return nil, errors.New("id is empty")
	}
	opts := RedemptionQueryOptions{OperatorId: operatorId, OperatorRole: operatorRole}
	redemption := Redemption{}
	err := redemptionBaseQuery(DB, opts).First(&redemption, "redemptions.id = ?", id).Error
	return &redemption, err
}

func CreateRedemptionsWithQuota(userId int, name string, quota int, count int, expiredTime int64) ([]string, error) {
	if userId <= 0 {
		return nil, errors.New("invalid user id")
	}
	if quota <= 0 || count <= 0 {
		return nil, errors.New("quota and count must be positive")
	}
	totalQuota := quota * count
	if quota != 0 && totalQuota/quota != count {
		return nil, errors.New("quota too large")
	}

	keys := make([]string, 0, count)
	err := DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&User{}).
			Where("id = ? AND quota >= ?", userId, totalQuota).
			UpdateColumn("quota", gorm.Expr("quota - ?", totalQuota))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrRedemptionQuotaInsufficient
		}

		now := common.GetTimestamp()
		for i := 0; i < count; i++ {
			key := common.GetUUID()
			redemption := Redemption{
				UserId:      userId,
				Name:        name,
				Key:         key,
				CreatedTime: now,
				Quota:       quota,
				ExpiredTime: expiredTime,
			}
			if err := tx.Create(&redemption).Error; err != nil {
				return err
			}
			keys = append(keys, key)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	syncUserQuotaCacheForRedemption(userId, -totalQuota)
	RecordLog(userId, LogTypeManage, fmt.Sprintf("创建兑换码 %d 个，冻结额度 %s", count, logger.LogQuotaManage(totalQuota)))
	return keys, nil
}

func refundRedemptionQuota(tx *gorm.DB, redemption *Redemption) error {
	if redemption == nil || redemption.UserId <= 0 || redemption.Quota <= 0 {
		return nil
	}
	return tx.Model(&User{}).
		Where("id = ?", redemption.UserId).
		UpdateColumn("quota", gorm.Expr("quota + ?", redemption.Quota)).Error
}

func redemptionOwnerScopedQuery(tx *gorm.DB, id int, operatorId int, operatorRole int) *gorm.DB {
	query := tx.Set("gorm:query_option", "FOR UPDATE")
	if !redemptionOperatorIsAdmin(operatorRole) {
		query = query.Where("user_id = ?", operatorId)
	}
	return query.Where("id = ?", id)
}

func DisableRedemptionById(id int, operatorId int, operatorRole int, reason string) (*Redemption, error) {
	if id == 0 {
		return nil, errors.New("id is empty")
	}
	if strings.TrimSpace(reason) == "" {
		reason = "禁用兑换码"
	}

	var redemption Redemption
	refunded := false
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := redemptionOwnerScopedQuery(tx, id, operatorId, operatorRole).First(&redemption).Error; err != nil {
			return err
		}
		if redemption.Status == common.RedemptionCodeStatusUsed || redemption.Status == common.RedemptionCodeStatusDisabled {
			return nil
		}
		if err := refundRedemptionQuota(tx, &redemption); err != nil {
			return err
		}
		refunded = true
		redemption.Status = common.RedemptionCodeStatusDisabled
		return tx.Model(&redemption).Select("status").Updates(&redemption).Error
	})
	if err != nil {
		return nil, err
	}
	if refunded {
		syncUserQuotaCacheForRedemption(redemption.UserId, redemption.Quota)
		RecordLog(redemption.UserId, LogTypeManage, fmt.Sprintf("%s，兑换码ID %d 失效，退回额度 %s", reason, redemption.Id, logger.LogQuotaManage(redemption.Quota)))
	}
	return &redemption, nil
}

func EnableRedemptionById(id int, operatorId int, operatorRole int) (*Redemption, error) {
	if id == 0 {
		return nil, errors.New("id is empty")
	}

	var redemption Redemption
	charged := false
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := redemptionOwnerScopedQuery(tx, id, operatorId, operatorRole).First(&redemption).Error; err != nil {
			return err
		}
		if redemption.Status == common.RedemptionCodeStatusUsed {
			return errors.New("used redemption code cannot be enabled")
		}
		if redemption.ExpiredTime != 0 && redemption.ExpiredTime < common.GetTimestamp() {
			return errors.New("expired redemption code cannot be enabled")
		}
		creator, err := GetUserById(redemption.UserId, false)
		if err != nil {
			return err
		}
		if !redemptionOperatorIsAdmin(creator.Role) && !UserIsDistributor(creator) {
			return errors.New("redemption creator is not allowed to enable redemption codes")
		}
		if redemption.Status == common.RedemptionCodeStatusEnabled {
			return nil
		}
		result := tx.Model(&User{}).
			Where("id = ? AND quota >= ?", redemption.UserId, redemption.Quota).
			UpdateColumn("quota", gorm.Expr("quota - ?", redemption.Quota))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrRedemptionQuotaInsufficient
		}
		charged = true
		redemption.Status = common.RedemptionCodeStatusEnabled
		return tx.Model(&redemption).Select("status").Updates(&redemption).Error
	})
	if err != nil {
		return nil, err
	}
	if charged {
		syncUserQuotaCacheForRedemption(redemption.UserId, -redemption.Quota)
		RecordLog(redemption.UserId, LogTypeManage, fmt.Sprintf("启用兑换码ID %d，重新冻结额度 %s", redemption.Id, logger.LogQuotaManage(redemption.Quota)))
	}
	return &redemption, nil
}

func UpdateRedemptionInfo(redemption *Redemption, operatorId int, operatorRole int) (*Redemption, error) {
	if redemption == nil || redemption.Id == 0 {
		return nil, errors.New("id is empty")
	}

	var current Redemption
	delta := 0
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := redemptionOwnerScopedQuery(tx, redemption.Id, operatorId, operatorRole).First(&current).Error; err != nil {
			return err
		}
		if current.Status != common.RedemptionCodeStatusEnabled {
			return errors.New("only unused and enabled redemption codes can be edited")
		}
		if current.ExpiredTime != 0 && current.ExpiredTime < common.GetTimestamp() {
			return errors.New("expired redemption code cannot be edited")
		}

		delta = redemption.Quota - current.Quota
		if delta > 0 {
			result := tx.Model(&User{}).
				Where("id = ? AND quota >= ?", current.UserId, delta).
				UpdateColumn("quota", gorm.Expr("quota - ?", delta))
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return ErrRedemptionQuotaInsufficient
			}
		} else if delta < 0 {
			if err := tx.Model(&User{}).
				Where("id = ?", current.UserId).
				UpdateColumn("quota", gorm.Expr("quota + ?", -delta)).Error; err != nil {
				return err
			}
		}

		current.Name = redemption.Name
		current.Quota = redemption.Quota
		current.ExpiredTime = redemption.ExpiredTime
		return tx.Model(&current).Select("name", "quota", "expired_time").Updates(&current).Error
	})
	if err != nil {
		return nil, err
	}
	if delta != 0 {
		syncUserQuotaCacheForRedemption(current.UserId, -delta)
		if delta > 0 {
			RecordLog(current.UserId, LogTypeManage, fmt.Sprintf("更新兑换码ID %d，追加冻结额度 %s", current.Id, logger.LogQuotaManage(delta)))
		} else {
			RecordLog(current.UserId, LogTypeManage, fmt.Sprintf("更新兑换码ID %d，退回额度 %s", current.Id, logger.LogQuotaManage(-delta)))
		}
	}
	return &current, nil
}

func DeleteRedemptionById(id int, operatorId int, operatorRole int) (err error) {
	if id == 0 {
		return errors.New("id is empty")
	}

	var redemption Redemption
	refunded := false
	err = DB.Transaction(func(tx *gorm.DB) error {
		if err := redemptionOwnerScopedQuery(tx, id, operatorId, operatorRole).First(&redemption).Error; err != nil {
			return err
		}
		if redemption.Status == common.RedemptionCodeStatusEnabled {
			if err := refundRedemptionQuota(tx, &redemption); err != nil {
				return err
			}
			refunded = true
		}
		return tx.Delete(&redemption).Error
	})
	if err != nil {
		return err
	}
	if refunded {
		syncUserQuotaCacheForRedemption(redemption.UserId, redemption.Quota)
		RecordLog(redemption.UserId, LogTypeManage, fmt.Sprintf("删除兑换码ID %d，退回额度 %s", redemption.Id, logger.LogQuotaManage(redemption.Quota)))
	} else {
		RecordLog(redemption.UserId, LogTypeManage, fmt.Sprintf("删除兑换码ID %d", redemption.Id))
	}
	return nil
}

func SettleExpiredRedemptions(operatorId int, operatorRole int) error {
	now := common.GetTimestamp()
	var redemptions []Redemption

	query := DB.Where("status = ? AND expired_time != 0 AND expired_time < ?", common.RedemptionCodeStatusEnabled, now)
	if !redemptionOperatorIsAdmin(operatorRole) {
		query = query.Where("user_id = ?", operatorId)
	}
	if err := query.Find(&redemptions).Error; err != nil {
		return err
	}

	for _, redemption := range redemptions {
		r := redemption
		refunded := false
		err := DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Set("gorm:query_option", "FOR UPDATE").
				First(&r, "id = ? AND status = ?", r.Id, common.RedemptionCodeStatusEnabled).Error; err != nil {
				return err
			}
			if r.ExpiredTime == 0 || r.ExpiredTime >= now {
				return nil
			}
			if err := refundRedemptionQuota(tx, &r); err != nil {
				return err
			}
			refunded = true
			r.Status = common.RedemptionCodeStatusDisabled
			return tx.Model(&r).Select("status").Updates(&r).Error
		})
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return err
		}
		if refunded {
			syncUserQuotaCacheForRedemption(r.UserId, r.Quota)
			RecordLog(r.UserId, LogTypeManage, fmt.Sprintf("兑换码ID %d 到期失效，退回额度 %s", r.Id, logger.LogQuotaManage(r.Quota)))
		}
	}
	return nil
}

func DisableUserActiveRedemptions(userId int, reason string) (int64, int, error) {
	if userId <= 0 {
		return 0, 0, nil
	}
	if strings.TrimSpace(reason) == "" {
		reason = "批量禁用兑换码"
	}

	var redemptions []Redemption
	if err := DB.Where("user_id = ? AND status = ?", userId, common.RedemptionCodeStatusEnabled).Find(&redemptions).Error; err != nil {
		return 0, 0, err
	}

	var count int64
	totalRefund := 0
	for _, redemption := range redemptions {
		r := redemption
		refunded := false
		err := DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Set("gorm:query_option", "FOR UPDATE").
				First(&r, "id = ? AND status = ?", r.Id, common.RedemptionCodeStatusEnabled).Error; err != nil {
				return err
			}
			if err := refundRedemptionQuota(tx, &r); err != nil {
				return err
			}
			refunded = true
			r.Status = common.RedemptionCodeStatusDisabled
			return tx.Model(&r).Select("status").Updates(&r).Error
		})
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return count, totalRefund, err
		}
		if refunded {
			count++
			totalRefund += r.Quota
		}
	}
	if totalRefund > 0 {
		syncUserQuotaCacheForRedemption(userId, totalRefund)
		RecordLog(userId, LogTypeManage, fmt.Sprintf("%s，失效兑换码 %d 个，退回额度 %s", reason, count, logger.LogQuotaManage(totalRefund)))
	}
	return count, totalRefund, nil
}

func DeleteInvalidRedemptions(operatorId int, operatorRole int) (int64, error) {
	if err := SettleExpiredRedemptions(operatorId, operatorRole); err != nil {
		return 0, err
	}
	query := DB.Where("status IN ?", []int{common.RedemptionCodeStatusUsed, common.RedemptionCodeStatusDisabled})
	if !redemptionOperatorIsAdmin(operatorRole) {
		query = query.Where("user_id = ?", operatorId)
	}
	result := query.Delete(&Redemption{})
	return result.RowsAffected, result.Error
}

func Redeem(key string, userId int) (quota int, err error) {
	if key == "" {
		return 0, errors.New("redemption code is required")
	}
	if userId == 0 {
		return 0, errors.New("invalid user id")
	}

	redemption := &Redemption{}
	keyCol := "`key`"
	if common.UsingPostgreSQL {
		keyCol = `"key"`
	}

	common.RandomSleep()
	expiredRefund := false
	var expiredRedeemErr error
	err = DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Set("gorm:query_option", "FOR UPDATE").Where(keyCol+" = ?", key).First(redemption).Error
		if err != nil {
			return errors.New("invalid redemption code")
		}
		if redemption.Status != common.RedemptionCodeStatusEnabled {
			return errors.New("redemption code has been used")
		}
		if redemption.ExpiredTime != 0 && redemption.ExpiredTime < common.GetTimestamp() {
			if err := refundRedemptionQuota(tx, redemption); err != nil {
				return err
			}
			redemption.Status = common.RedemptionCodeStatusDisabled
			if err := tx.Model(redemption).Select("status").Updates(redemption).Error; err != nil {
				return err
			}
			expiredRefund = true
			expiredRedeemErr = errors.New("redemption code has expired")
			return nil
		}
		err = tx.Model(&User{}).Where("id = ?", userId).Updates(map[string]interface{}{
			"quota":      gorm.Expr("quota + ?", redemption.Quota),
			"gift_quota": gorm.Expr("gift_quota + ?", redemption.Quota),
		}).Error
		if err != nil {
			return err
		}
		redemption.RedeemedTime = common.GetTimestamp()
		redemption.Status = common.RedemptionCodeStatusUsed
		redemption.UsedUserId = userId
		err = tx.Save(redemption).Error
		return err
	})
	if expiredRefund {
		syncUserQuotaCacheForRedemption(redemption.UserId, redemption.Quota)
		RecordLog(redemption.UserId, LogTypeManage, fmt.Sprintf("兑换码ID %d 到期失效，退回额度 %s", redemption.Id, logger.LogQuotaManage(redemption.Quota)))
		return 0, expiredRedeemErr
	}
	if err != nil {
		common.SysError("redemption failed: " + err.Error())
		return 0, ErrRedeemFailed
	}
	syncUserQuotaCacheForRedemption(userId, redemption.Quota)
	RecordLog(userId, LogTypeTopup, fmt.Sprintf("通过兑换码充值 %s，兑换码ID %d", logger.LogQuotaManage(redemption.Quota), redemption.Id))
	ApplyAffiliateTopupReward(userId, redemption.Quota)
	return redemption.Quota, nil
}

func (redemption *Redemption) Insert() error {
	return DB.Create(redemption).Error
}

func (redemption *Redemption) SelectUpdate() error {
	return DB.Model(redemption).Select("redeemed_time", "status").Updates(redemption).Error
}

func (redemption *Redemption) Update() error {
	return DB.Model(redemption).Select("name", "status", "quota", "redeemed_time", "expired_time").Updates(redemption).Error
}

func (redemption *Redemption) Delete() error {
	return DB.Delete(redemption).Error
}
