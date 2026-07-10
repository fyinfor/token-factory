package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const (
	InvoiceRequestStatusPending    = "pending"
	InvoiceRequestStatusProcessing = "processing"
	InvoiceRequestStatusIssued     = "issued"
	InvoiceRequestStatusRejected   = "rejected"
	InvoiceRequestStatusCancelled  = "cancelled"

	InvoiceTitleTypePersonal = "personal"
	InvoiceTitleTypeCompany  = "company"
)

// InvoiceProfile 用户开票信息（电子普票）。
type InvoiceProfile struct {
	Id        int    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId    int    `json:"user_id" gorm:"not null;uniqueIndex"`
	TitleType string `json:"title_type" gorm:"type:varchar(16);not null;default:personal"`
	Title     string `json:"title" gorm:"type:varchar(256);not null"`
	TaxNo     string `json:"tax_no" gorm:"type:varchar(64);default:''"`
	Email     string `json:"email" gorm:"type:varchar(128);not null"`
	Phone     string `json:"phone" gorm:"type:varchar(32);default:''"`
	CreatedAt int64  `json:"created_at" gorm:"autoCreateTime;bigint"`
	UpdatedAt int64  `json:"updated_at" gorm:"autoUpdateTime;bigint"`
}

func (InvoiceProfile) TableName() string { return "invoice_profiles" }

// InvoiceRequest 发票申请。
type InvoiceRequest struct {
	Id              int     `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId          int     `json:"user_id" gorm:"not null;index"`
	RequestNo       string  `json:"request_no" gorm:"type:varchar(64);not null;uniqueIndex"`
	Status          string  `json:"status" gorm:"type:varchar(32);not null;index"`
	TotalAmount     float64 `json:"total_amount" gorm:"type:decimal(20,6);not null"`
	InvoiceType     string  `json:"invoice_type" gorm:"type:varchar(32);not null;default:electronic_ordinary"`
	ProfileSnapshot string  `json:"profile_snapshot" gorm:"type:text;not null"`
	Remark          string  `json:"remark" gorm:"type:text"`
	IssuedAt        int64   `json:"issued_at" gorm:"bigint;default:0"`
	InvoiceUrl      string  `json:"invoice_url" gorm:"type:varchar(512);default:''"`
	InvoiceCode     string  `json:"invoice_code" gorm:"type:varchar(64);default:''"`
	AdminNote       string  `json:"admin_note" gorm:"type:text"`
	CreatedAt       int64   `json:"created_at" gorm:"autoCreateTime;bigint;index"`
	UpdatedAt       int64   `json:"updated_at" gorm:"autoUpdateTime;bigint"`
}

func (InvoiceRequest) TableName() string { return "invoice_requests" }

// InvoiceRequestItem 发票申请明细（支持合并开票）。
type InvoiceRequestItem struct {
	Id               int     `json:"id" gorm:"primaryKey;autoIncrement"`
	InvoiceRequestId int     `json:"invoice_request_id" gorm:"not null;index"`
	TopUpId          int     `json:"topup_id" gorm:"not null;index"`
	TradeNo          string  `json:"trade_no" gorm:"type:varchar(255);not null"`
	InvoiceAmount    float64 `json:"invoice_amount" gorm:"type:decimal(20,6);not null"`
	CreatedAt        int64   `json:"created_at" gorm:"autoCreateTime;bigint"`
}

func (InvoiceRequestItem) TableName() string { return "invoice_request_items" }

// TopUpConsumeAttribution 充值订单消耗归因（FIFO）。
type TopUpConsumeAttribution struct {
	Id             int   `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId         int   `json:"user_id" gorm:"not null;index:idx_topup_attr_user_topup,unique"`
	TopUpId        int   `json:"topup_id" gorm:"not null;index:idx_topup_attr_user_topup,unique"`
	ConsumedQuota  int   `json:"consumed_quota" gorm:"not null;default:0"`
	UpdatedAt      int64 `json:"updated_at" gorm:"autoUpdateTime;bigint"`
}

func (TopUpConsumeAttribution) TableName() string { return "topup_consume_attributions" }

// invoiceTopUpIDColumn 返回充值订单外键列名（PostgreSQL 为 top_up_id，MySQL 为 topup_id）。
func invoiceTopUpIDColumn() string {
	if common.UsingPostgreSQL {
		return "top_up_id"
	}
	return "topup_id"
}

// InvoiceEligibleOrder 待开票订单列表项。
type InvoiceEligibleOrder struct {
	TopUpId          int     `json:"topup_id"`
	TradeNo          string  `json:"trade_no"`
	Money            float64 `json:"money"`
	QuotaToAdd       int     `json:"quota_to_add"`
	ConsumedQuota    int     `json:"consumed_quota"`
	ConsumedAmount   float64 `json:"consumed_amount"`
	InvoicedAmount   float64 `json:"invoiced_amount"`
	PendingAmount    float64 `json:"pending_amount"`
	InvoiceableAmount float64 `json:"invoiceable_amount"`
	CreateTime       int64   `json:"create_time"`
	Status           string  `json:"status"`
}

func GetInvoiceProfileByUserID(userID int) (*InvoiceProfile, error) {
	if userID <= 0 {
		return nil, errors.New("invalid user id")
	}
	var profile InvoiceProfile
	err := DB.Where("user_id = ?", userID).First(&profile).Error
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

func UpsertInvoiceProfile(profile *InvoiceProfile) error {
	if profile == nil || profile.UserId <= 0 {
		return errors.New("invalid profile")
	}
	profile.Title = strings.TrimSpace(profile.Title)
	profile.Email = strings.TrimSpace(profile.Email)
	profile.TaxNo = strings.TrimSpace(profile.TaxNo)
	profile.Phone = strings.TrimSpace(profile.Phone)
	if profile.Title == "" || profile.Email == "" {
		return errors.New("title and email are required")
	}
	if profile.TitleType == "" {
		profile.TitleType = InvoiceTitleTypePersonal
	}
	var existing InvoiceProfile
	err := DB.Where("user_id = ?", profile.UserId).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return DB.Create(profile).Error
	}
	if err != nil {
		return err
	}
	profile.Id = existing.Id
	return DB.Save(profile).Error
}

func topUpMoneyPerQuota(topUp *TopUp) decimal.Decimal {
	if topUp == nil {
		return decimal.Zero
	}
	quota := topUp.ResolveQuotaToAdd()
	if quota <= 0 {
		return decimal.Zero
	}
	return decimal.NewFromFloat(topUp.Money).Div(decimal.NewFromInt(int64(quota)))
}

func consumedAmountForTopUp(topUp *TopUp, consumedQuota int) decimal.Decimal {
	if topUp == nil || consumedQuota <= 0 {
		return decimal.Zero
	}
	return topUpMoneyPerQuota(topUp).Mul(decimal.NewFromInt(int64(consumedQuota)))
}

func getTopUpPendingInvoiceAmount(topUpID int) (decimal.Decimal, error) {
	var total decimal.Decimal
	rows, err := getPendingInvoiceItemsByTopUp(topUpID)
	if err != nil {
		return decimal.Zero, err
	}
	for _, row := range rows {
		total = total.Add(decimal.NewFromFloat(row.InvoiceAmount))
	}
	return total, nil
}

func getPendingInvoiceItemsByTopUp(topUpID int) ([]InvoiceRequestItem, error) {
	var items []InvoiceRequestItem
	err := DB.Table("invoice_request_items").
		Select("invoice_request_items.*").
		Joins("JOIN invoice_requests ON invoice_requests.id = invoice_request_items.invoice_request_id").
		Where("invoice_request_items."+invoiceTopUpIDColumn()+" = ? AND invoice_requests.status IN ?", topUpID, []string{
			InvoiceRequestStatusPending,
			InvoiceRequestStatusProcessing,
		}).
		Find(&items).Error
	return items, err
}

func GetTopUpInvoiceableAmount(topUp *TopUp, consumedQuota int) (float64, error) {
	if topUp == nil || topUp.Status != common.TopUpStatusSuccess {
		return 0, nil
	}
	consumedMoney := consumedAmountForTopUp(topUp, consumedQuota)
	available := consumedMoney.Sub(decimal.NewFromFloat(topUp.InvoicedAmount))
	pending, err := getTopUpPendingInvoiceAmount(topUp.Id)
	if err != nil {
		return 0, err
	}
	available = available.Sub(pending)
	if available.IsNegative() {
		return 0, nil
	}
	f, _ := available.Float64()
	return f, nil
}

func ListInvoiceEligibleOrders(userID int, tradeNoKeyword string) ([]InvoiceEligibleOrder, error) {
	if userID <= 0 {
		return nil, errors.New("invalid user id")
	}
	var topups []*TopUp
	tx := DB.Where("user_id = ? AND status = ?", userID, common.TopUpStatusSuccess).Order("create_time asc")
	if kw := strings.TrimSpace(tradeNoKeyword); kw != "" {
		tx = tx.Where("trade_no LIKE ?", "%"+kw+"%")
	}
	if err := tx.Find(&topups).Error; err != nil {
		return nil, err
	}
	attrMap, err := getTopUpAttributionMap(userID)
	if err != nil {
		return nil, err
	}
	out := make([]InvoiceEligibleOrder, 0, len(topups))
	for _, topUp := range topups {
		consumedQuota := attrMap[topUp.Id]
		consumedAmountDec := consumedAmountForTopUp(topUp, consumedQuota)
		consumedAmount, _ := consumedAmountDec.Float64()
		invoiceable, err := GetTopUpInvoiceableAmount(topUp, consumedQuota)
		if err != nil {
			return nil, err
		}
		pending, err := getTopUpPendingInvoiceAmount(topUp.Id)
		if err != nil {
			return nil, err
		}
		pendingF, _ := pending.Float64()
		out = append(out, InvoiceEligibleOrder{
			TopUpId:           topUp.Id,
			TradeNo:           topUp.TradeNo,
			Money:             topUp.Money,
			QuotaToAdd:        topUp.ResolveQuotaToAdd(),
			ConsumedQuota:     consumedQuota,
			ConsumedAmount:    consumedAmount,
			InvoicedAmount:    topUp.InvoicedAmount,
			PendingAmount:     pendingF,
			InvoiceableAmount: invoiceable,
			CreateTime:        topUp.CreateTime,
			Status:            topUp.Status,
		})
	}
	return out, nil
}

func getTopUpAttributionMap(userID int) (map[int]int, error) {
	var rows []TopUpConsumeAttribution
	if err := DB.Where("user_id = ?", userID).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[int]int, len(rows))
	for _, row := range rows {
		out[row.TopUpId] = row.ConsumedQuota
	}
	return out, nil
}

func AttributeConsumeQuotaToTopUps(userID, consumeQuota int) error {
	if userID <= 0 || consumeQuota <= 0 {
		return nil
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var topups []TopUp
		if err := tx.Where("user_id = ? AND status = ?", userID, common.TopUpStatusSuccess).
			Order("create_time asc").Find(&topups).Error; err != nil {
			return err
		}
		remaining := consumeQuota
		for _, topUp := range topups {
			if remaining <= 0 {
				break
			}
			capacity := topUp.ResolveQuotaToAdd()
			if capacity <= 0 {
				continue
			}
			var attr TopUpConsumeAttribution
			err := tx.Where("user_id = ? AND "+invoiceTopUpIDColumn()+" = ?", userID, topUp.Id).First(&attr).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				attr = TopUpConsumeAttribution{UserId: userID, TopUpId: topUp.Id}
			} else if err != nil {
				return err
			}
			room := capacity - attr.ConsumedQuota
			if room <= 0 {
				continue
			}
			add := remaining
			if add > room {
				add = room
			}
			attr.ConsumedQuota += add
			remaining -= add
			if err := tx.Save(&attr).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

type InvoiceRequestItemInput struct {
	TopUpId       int     `json:"topup_id"`
	InvoiceAmount float64 `json:"invoice_amount"`
}

func CreateInvoiceRequest(userID int, items []InvoiceRequestItemInput, remark string, profile *InvoiceProfile) (*InvoiceRequest, error) {
	if userID <= 0 || len(items) == 0 || profile == nil {
		return nil, errors.New("invalid invoice request")
	}
	profileJSON, err := json.Marshal(profile)
	if err != nil {
		return nil, err
	}
	var created *InvoiceRequest
	err = DB.Transaction(func(tx *gorm.DB) error {
		attrMap, err := getTopUpAttributionMap(userID)
		if err != nil {
			return err
		}
		total := decimal.Zero
		prepared := make([]InvoiceRequestItem, 0, len(items))
		for _, item := range items {
			if item.TopUpId <= 0 || item.InvoiceAmount <= 0 {
				return fmt.Errorf("invalid invoice item for topup %d", item.TopUpId)
			}
			var topUp TopUp
			if err := tx.Where("id = ? AND user_id = ? AND status = ?", item.TopUpId, userID, common.TopUpStatusSuccess).First(&topUp).Error; err != nil {
				return fmt.Errorf("topup %d not found or not payable", item.TopUpId)
			}
			invoiceable, err := GetTopUpInvoiceableAmount(&topUp, attrMap[topUp.Id])
			if err != nil {
				return err
			}
			if item.InvoiceAmount > invoiceable+0.000001 {
				return fmt.Errorf("invoice amount exceeds available for %s", topUp.TradeNo)
			}
			total = total.Add(decimal.NewFromFloat(item.InvoiceAmount))
			prepared = append(prepared, InvoiceRequestItem{
				TopUpId:       topUp.Id,
				TradeNo:       topUp.TradeNo,
				InvoiceAmount: item.InvoiceAmount,
			})
		}
		totalF, _ := total.Float64()
		req := &InvoiceRequest{
			UserId:          userID,
			RequestNo:       generateInvoiceRequestNo(),
			Status:          InvoiceRequestStatusPending,
			TotalAmount:     totalF,
			InvoiceType:     "electronic_ordinary",
			ProfileSnapshot: string(profileJSON),
			Remark:          strings.TrimSpace(remark),
		}
		if err := tx.Create(req).Error; err != nil {
			return err
		}
		for i := range prepared {
			prepared[i].InvoiceRequestId = req.Id
			if err := tx.Create(&prepared[i]).Error; err != nil {
				return err
			}
		}
		created = req
		return nil
	})
	return created, err
}

func generateInvoiceRequestNo() string {
	return fmt.Sprintf("INV%s%04d", time.Now().Format("20060102150405"), time.Now().Nanosecond()%10000)
}

func ListInvoiceRequestsByUser(userID int, pageInfo *common.PageInfo) ([]*InvoiceRequest, int64, error) {
	if userID <= 0 {
		return nil, 0, errors.New("invalid user id")
	}
	var total int64
	tx := DB.Model(&InvoiceRequest{}).Where("user_id = ?", userID)
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []*InvoiceRequest
	if err := tx.Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func ListInvoiceRequestsAdmin(status string, pageInfo *common.PageInfo) ([]*InvoiceRequest, int64, error) {
	var total int64
	tx := DB.Model(&InvoiceRequest{})
	if s := strings.TrimSpace(status); s != "" {
		tx = tx.Where("status = ?", s)
	}
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []*InvoiceRequest
	if err := tx.Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func GetInvoiceRequestByID(id int) (*InvoiceRequest, error) {
	var req InvoiceRequest
	if err := DB.Where("id = ?", id).First(&req).Error; err != nil {
		return nil, err
	}
	return &req, nil
}

func GetInvoiceRequestItems(requestID int) ([]InvoiceRequestItem, error) {
	var items []InvoiceRequestItem
	err := DB.Where("invoice_request_id = ?", requestID).Order("id asc").Find(&items).Error
	return items, err
}

func IssueInvoiceRequest(requestID int, invoiceCode, invoiceURL, adminNote string) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var req InvoiceRequest
		if err := tx.Where("id = ?", requestID).First(&req).Error; err != nil {
			return err
		}
		if req.Status != InvoiceRequestStatusPending && req.Status != InvoiceRequestStatusProcessing {
			return fmt.Errorf("request status not issuable: %s", req.Status)
		}
		var items []InvoiceRequestItem
		if err := tx.Where("invoice_request_id = ?", requestID).Find(&items).Error; err != nil {
			return err
		}
		for _, item := range items {
			var topUp TopUp
			if err := tx.Where("id = ?", item.TopUpId).First(&topUp).Error; err != nil {
				return err
			}
			topUp.InvoicedAmount += item.InvoiceAmount
			if err := tx.Save(&topUp).Error; err != nil {
				return err
			}
		}
		now := time.Now().Unix()
		req.Status = InvoiceRequestStatusIssued
		req.InvoiceCode = strings.TrimSpace(invoiceCode)
		req.InvoiceUrl = strings.TrimSpace(invoiceURL)
		req.AdminNote = strings.TrimSpace(adminNote)
		req.IssuedAt = now
		return tx.Save(&req).Error
	})
}

func RejectInvoiceRequest(requestID int, adminNote string) error {
	var req InvoiceRequest
	if err := DB.Where("id = ?", requestID).First(&req).Error; err != nil {
		return err
	}
	if req.Status != InvoiceRequestStatusPending && req.Status != InvoiceRequestStatusProcessing {
		return fmt.Errorf("request status not rejectable: %s", req.Status)
	}
	req.Status = InvoiceRequestStatusRejected
	req.AdminNote = strings.TrimSpace(adminNote)
	return DB.Save(&req).Error
}
