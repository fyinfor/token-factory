package model

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

var distWithdrawMonthRe = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

// 分销商线下提现申请状态
const (
	DistWithdrawStatusPending   = 1 // 提现中（待审核）
	DistWithdrawStatusApproved  = 2 // 提现成功
	DistWithdrawStatusRejected  = 3 // 提现失败（驳回）
	DistWithdrawStatusCancelled = 4 // 已取消
)

// DistributorWithdrawal 分销商线下提现申请（提交后暂扣 aff_quota，驳回/取消退回）
type DistributorWithdrawal struct {
	Id            int    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId        int    `json:"user_id" gorm:"not null;index:idx_dist_wd_user"`
	RealName      string `json:"real_name" gorm:"type:varchar(64);not null;column:real_name"`
	BankName      string `json:"bank_name" gorm:"type:varchar(128);not null;column:bank_name"`
	BankAccount   string `json:"bank_account" gorm:"type:varchar(64);not null;column:bank_account"`
	VoucherUrls   string `json:"voucher_urls" gorm:"type:text;not null;column:voucher_urls"` // JSON 数组 URL
	WithdrawMonth string `json:"withdraw_month" gorm:"type:varchar(16);not null;column:withdraw_month"` // YYYY-MM
	QuotaAmount   int    `json:"quota_amount" gorm:"not null;column:quota_amount"`
	Status        int    `json:"status" gorm:"not null;default:1;index:idx_dist_wd_status"`
	RejectReason  string `json:"reject_reason" gorm:"type:varchar(512);column:reject_reason"`
	ReviewerId    int    `json:"reviewer_id" gorm:"column:reviewer_id"`
	ReviewedAt    int64  `json:"reviewed_at" gorm:"column:reviewed_at"`
	CancelledAt   int64  `json:"cancelled_at" gorm:"column:cancelled_at"`
	CreatedAt     int64  `json:"created_at" gorm:"autoCreateTime;bigint;index"`
	UpdatedAt     int64  `json:"updated_at" gorm:"autoUpdateTime;bigint"`
}

func (DistributorWithdrawal) TableName() string {
	return "distributor_withdrawals"
}

// GetDistributorWithdrawalByID 按主键查询提现记录。
func GetDistributorWithdrawalByID(id int) (*DistributorWithdrawal, error) {
	if id <= 0 {
		return nil, errors.New("invalid id")
	}
	var w DistributorWithdrawal
	if err := DB.Where("id = ?", id).First(&w).Error; err != nil {
		return nil, err
	}
	return &w, nil
}

// GetDistributorMinWithdrawQuota 最低提现额度（内部点数），未配置时与 QuotaPerUnit 一致（约等于展示 1 单位）
func GetDistributorMinWithdrawQuota() int {
	common.OptionMapRWMutex.RLock()
	raw := common.Interface2String(common.OptionMap["DistributorMinWithdrawQuota"])
	common.OptionMapRWMutex.RUnlock()
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return int(common.QuotaPerUnit)
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return int(common.QuotaPerUnit)
	}
	return n
}

func distWithdrawRefundAffQuota(tx *gorm.DB, userId int, quota int) error {
	if userId <= 0 || quota <= 0 {
		return nil
	}
	res := tx.Model(&User{}).Where("id = ?", userId).UpdateColumn("aff_quota", gorm.Expr("aff_quota + ?", quota))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("user not found: %d", userId)
	}
	return nil
}

// CreateDistributorWithdrawal 提交提现：校验最低额度与余额，暂扣 aff_quota
func CreateDistributorWithdrawal(userId int, realName, bankName, bankAccount, voucherUrlsJSON, withdrawMonth string, quotaAmount int) error {
	realName = strings.TrimSpace(realName)
	bankName = strings.TrimSpace(bankName)
	bankAccount = strings.TrimSpace(bankAccount)
	withdrawMonth = strings.TrimSpace(withdrawMonth)
	voucherUrlsJSON = strings.TrimSpace(voucherUrlsJSON)
	if realName == "" {
		return errors.New("请填写真实姓名")
	}
	if bankName == "" {
		return errors.New("请填写开户行")
	}
	if bankAccount == "" {
		return errors.New("请填写银行卡号")
	}
	if voucherUrlsJSON == "" {
		return errors.New("请上传票据")
	}
	if withdrawMonth == "" {
		withdrawMonth = time.Now().Format("2006-01")
	} else if !distWithdrawMonthRe.MatchString(withdrawMonth) {
		return errors.New("提现月份格式应为 YYYY-MM")
	}
	var vurls []string
	if err := common.UnmarshalJsonStr(voucherUrlsJSON, &vurls); err != nil || len(vurls) == 0 {
		return errors.New("请上传票据")
	}
	for _, u := range vurls {
		if strings.TrimSpace(u) == "" {
			return errors.New("票据地址无效")
		}
	}
	if quotaAmount <= 0 {
		return errors.New("提现额度无效")
	}
	u, err := GetUserById(userId, false)
	if err != nil {
		return err
	}
	if !UserIsDistributor(u) {
		return errors.New("仅分销商可申请提现")
	}
	if u.AffQuota < quotaAmount {
		return errors.New("待使用收益不足")
	}
	minQ := GetDistributorMinWithdrawQuota()
	// 余额达到系统最低提现额度时，不得低于该门槛；余额不足该门槛时，允许在 1～当前余额之间提现
	if u.AffQuota >= minQ {
		if quotaAmount < minQ {
			return fmt.Errorf("提现额度不能低于系统下限")
		}
	} else {
		if quotaAmount < 1 || quotaAmount > u.AffQuota {
			return errors.New("提现额度须在 1 与当前待使用余额之间")
		}
	}

	ts := common.GetTimestamp()
	return DB.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&User{}).Where("id = ? AND aff_quota >= ?", userId, quotaAmount).
			UpdateColumn("aff_quota", gorm.Expr("aff_quota - ?", quotaAmount))
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errors.New("待使用收益不足")
		}
		w := DistributorWithdrawal{
			UserId:        userId,
			RealName:      realName,
			BankName:      bankName,
			BankAccount:   bankAccount,
			VoucherUrls:   voucherUrlsJSON,
			WithdrawMonth: withdrawMonth,
			QuotaAmount:   quotaAmount,
			Status:        DistWithdrawStatusPending,
			CreatedAt:     ts,
			UpdatedAt:     ts,
		}
		return tx.Create(&w).Error
	})
}

// CancelDistributorWithdrawal 用户取消待审核申请，退回 aff_quota
func CancelDistributorWithdrawal(userId, withdrawalId int) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var w DistributorWithdrawal
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ?", withdrawalId).First(&w).Error; err != nil {
			return errors.New("记录不存在")
		}
		if w.UserId != userId {
			return errors.New("无权操作")
		}
		if w.Status != DistWithdrawStatusPending {
			return errors.New("当前状态不可取消")
		}
		if err := distWithdrawRefundAffQuota(tx, userId, w.QuotaAmount); err != nil {
			return err
		}
		ts := common.GetTimestamp()
		return tx.Model(&DistributorWithdrawal{}).Where("id = ?", withdrawalId).Updates(map[string]interface{}{
			"status":       DistWithdrawStatusCancelled,
			"cancelled_at": ts,
			"updated_at":   ts,
		}).Error
	})
}

// ListDistributorWithdrawals 当前用户提现记录
func ListDistributorWithdrawals(userId int, pageInfo *common.PageInfo) ([]DistributorWithdrawal, int64, error) {
	if userId <= 0 {
		return nil, 0, errors.New("invalid user")
	}
	var total int64
	base := DB.Model(&DistributorWithdrawal{}).Where("user_id = ?", userId)
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []DistributorWithdrawal
	err := base.Order("id DESC").
		Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	if rows == nil {
		rows = []DistributorWithdrawal{}
	}
	return rows, total, nil
}

type DistributorWithdrawalAdminRow struct {
	DistributorWithdrawal
	Username string `json:"username"`
}

// ListDistributorWithdrawalsAdmin 管理端列表
func ListDistributorWithdrawalsAdmin(status int, keyword string, pageInfo *common.PageInfo) ([]DistributorWithdrawalAdminRow, int64, error) {
	base := DB.Model(&DistributorWithdrawal{})
	if status > 0 {
		base = base.Where("status = ?", status)
	}
	if kw := strings.TrimSpace(keyword); kw != "" {
		like := "%" + kw + "%"
		base = base.Where(
			"real_name LIKE ? OR bank_account LIKE ? OR user_id IN (SELECT id FROM users WHERE username LIKE ?)",
			like, like, like,
		)
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []DistributorWithdrawal
	err := base.Order("id DESC").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&list).Error
	if err != nil {
		return nil, 0, err
	}
	out := make([]DistributorWithdrawalAdminRow, 0, len(list))
	for i := range list {
		row := DistributorWithdrawalAdminRow{DistributorWithdrawal: list[i]}
		var u User
		if e := DB.Select("username").Where("id = ?", list[i].UserId).First(&u).Error; e == nil {
			row.Username = u.Username
		}
		out = append(out, row)
	}
	return out, total, nil
}

// ApproveDistributorWithdrawalAdmin 审核通过（额度已在提交时扣除，此处仅改状态）
func ApproveDistributorWithdrawalAdmin(withdrawalId, reviewerId int) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var w DistributorWithdrawal
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ?", withdrawalId).First(&w).Error; err != nil {
			return errors.New("记录不存在")
		}
		if w.Status != DistWithdrawStatusPending {
			return errors.New("当前状态不可审核")
		}
		ts := common.GetTimestamp()
		return tx.Model(&DistributorWithdrawal{}).Where("id = ?", withdrawalId).Updates(map[string]interface{}{
			"status":      DistWithdrawStatusApproved,
			"reviewer_id": reviewerId,
			"reviewed_at": ts,
			"updated_at":  ts,
		}).Error
	})
}

// RejectDistributorWithdrawalAdmin 驳回：退回 aff_quota
func RejectDistributorWithdrawalAdmin(withdrawalId, reviewerId int, reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return errors.New("请填写驳回原因")
	}
	if len(reason) > 500 {
		reason = reason[:500]
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var w DistributorWithdrawal
		if err := tx.Set("gorm:query_option", "FOR UPDATE").Where("id = ?", withdrawalId).First(&w).Error; err != nil {
			return errors.New("记录不存在")
		}
		if w.Status != DistWithdrawStatusPending {
			return errors.New("当前状态不可审核")
		}
		if err := distWithdrawRefundAffQuota(tx, w.UserId, w.QuotaAmount); err != nil {
			return err
		}
		ts := common.GetTimestamp()
		return tx.Model(&DistributorWithdrawal{}).Where("id = ?", withdrawalId).Updates(map[string]interface{}{
			"status":        DistWithdrawStatusRejected,
			"reject_reason": reason,
			"reviewer_id":   reviewerId,
			"reviewed_at":   ts,
			"updated_at":    ts,
		}).Error
	})
}
