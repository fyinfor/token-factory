package model

import (
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const TopUpPaymentMethodCorporate = "corporate"

// GrantUserGiftQuota 赠送不可开票额度（同时增加 quota 与 gift_quota）。
func GrantUserGiftQuota(userID, quota int) error {
	if userID <= 0 || quota <= 0 {
		return errors.New("invalid gift quota")
	}
	tx := DB.Model(&User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"quota":      gorm.Expr("quota + ?", quota),
		"gift_quota": gorm.Expr("gift_quota + ?", quota),
	})
	if tx.Error != nil {
		return tx.Error
	}
	if tx.RowsAffected == 0 {
		return fmt.Errorf("user not found: %d", userID)
	}
	return nil
}

// RestoreUserWalletQuota 退还钱包额度（按原赠送/充值比例恢复 gift_quota）。
func RestoreUserWalletQuota(userID, total, giftPart int) error {
	if userID <= 0 || total <= 0 {
		return nil
	}
	if giftPart < 0 {
		giftPart = 0
	}
	if giftPart > total {
		giftPart = total
	}
	return DB.Model(&User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"quota":      gorm.Expr("quota + ?", total),
		"gift_quota": gorm.Expr("gift_quota + ?", giftPart),
	}).Error
}

// ComputePaidConsumeWithGiftOffset 按「先扣赠送」计算本次消耗中应归因到可开票充值订单的额度。
// giftConsumedEarlier 为本请求内已消耗的赠送额度（如预扣费阶段），用于还原请求开始时的赠送余额。
func ComputePaidConsumeWithGiftOffset(userID, consumeAmount, giftConsumedEarlier int) int {
	if userID <= 0 || consumeAmount <= 0 {
		return 0
	}
	var user User
	if err := DB.Select("gift_quota").Where("id = ?", userID).First(&user).Error; err != nil {
		return consumeAmount
	}
	giftAtRequestStart := user.GiftQuota + giftConsumedEarlier
	if giftAtRequestStart < 0 {
		giftAtRequestStart = 0
	}
	giftForTotal := consumeAmount
	if giftForTotal > giftAtRequestStart {
		giftForTotal = giftAtRequestStart
	}
	return consumeAmount - giftForTotal
}

func decreaseUserQuotaGiftFirst(id, amount int) (paid, gift int, err error) {
	if amount <= 0 {
		return 0, 0, nil
	}
	err = DB.Transaction(func(tx *gorm.DB) error {
		var user User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&user).Error; err != nil {
			return err
		}
		gift = amount
		if gift > user.GiftQuota {
			gift = user.GiftQuota
		}
		paid = amount - gift
		return tx.Model(&User{}).Where("id = ?", id).Updates(map[string]interface{}{
			"quota":      gorm.Expr("quota - ?", amount),
			"gift_quota": gorm.Expr("gift_quota - ?", gift),
		}).Error
	})
	return paid, gift, err
}

// InvoiceBalanceSummary 用户开票相关余额摘要。
type InvoiceBalanceSummary struct {
	GiftQuota           int     `json:"gift_quota"`
	PaidQuota           int     `json:"paid_quota"`
	TotalQuota          int     `json:"total_quota"`
	InvoiceableAmount   float64 `json:"invoiceable_amount"`
	InvoiceableOrderCnt int     `json:"invoiceable_order_count"`
}

func GetInvoiceBalanceSummary(userID int) (*InvoiceBalanceSummary, error) {
	if userID <= 0 {
		return nil, errors.New("invalid user id")
	}
	var user User
	if err := DB.Select("id", "quota", "gift_quota").Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, err
	}
	gift := user.GiftQuota
	if gift < 0 {
		gift = 0
	}
	if gift > user.Quota {
		gift = user.Quota
	}
	paid := user.Quota - gift
	if paid < 0 {
		paid = 0
	}

	orders, err := ListInvoiceEligibleOrders(userID, "")
	if err != nil {
		return nil, err
	}
	totalInvoiceable := 0.0
	invoiceableCnt := 0
	for _, order := range orders {
		if order.InvoiceableAmount > 0 {
			invoiceableCnt++
			totalInvoiceable += order.InvoiceableAmount
		}
	}

	return &InvoiceBalanceSummary{
		GiftQuota:           gift,
		PaidQuota:           paid,
		TotalQuota:          user.Quota,
		InvoiceableAmount:   totalInvoiceable,
		InvoiceableOrderCnt: invoiceableCnt,
	}, nil
}

func topUpInvoiceEligibleWhere() string {
	if common.UsingPostgreSQL {
		return "invoice_eligible IS DISTINCT FROM FALSE"
	}
	return "(invoice_eligible = 1 OR invoice_eligible IS NULL)"
}
