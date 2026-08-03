package model

import (
	"errors"
	"fmt"
	"net/mail"
	"sort"
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

	InvoiceTypeElectronicOrdinary = "electronic_ordinary"
	InvoiceTypeElectronicSpecial  = "electronic_special"
)

// InvoiceProfile 用户开票信息。
type InvoiceProfile struct {
	Id          int    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId      int    `json:"user_id" gorm:"not null;uniqueIndex"`
	TitleType   string `json:"title_type" gorm:"type:varchar(16);not null;default:personal"`
	InvoiceType string `json:"invoice_type" gorm:"type:varchar(32);not null;default:electronic_ordinary"`
	Title       string `json:"title" gorm:"type:varchar(256);not null"`
	TaxNo       string `json:"tax_no" gorm:"type:varchar(64);default:''"`
	Email       string `json:"email" gorm:"type:varchar(128);not null"`
	Phone       string `json:"phone" gorm:"type:varchar(32);default:''"`
	CreatedAt   int64  `json:"created_at" gorm:"autoCreateTime;bigint"`
	UpdatedAt   int64  `json:"updated_at" gorm:"autoUpdateTime;bigint"`
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
	TopUpId          int     `json:"topup_id" gorm:"column:top_up_id;not null;index"`
	TradeNo          string  `json:"trade_no" gorm:"type:varchar(255);not null"`
	PaymentMethod    string  `json:"payment_method" gorm:"column:payment_method;->;-:migration"`
	InvoiceAmount    float64 `json:"invoice_amount" gorm:"type:decimal(20,6);not null"`
	CreatedAt        int64   `json:"created_at" gorm:"autoCreateTime;bigint"`
}

func (InvoiceRequestItem) TableName() string { return "invoice_request_items" }

// TopUpConsumeAttribution 充值订单消耗归因（FIFO）。
type TopUpConsumeAttribution struct {
	Id            int   `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId        int   `json:"user_id" gorm:"not null;index:idx_topup_attr_user_topup,unique"`
	TopUpId       int   `json:"topup_id" gorm:"column:top_up_id;not null;index:idx_topup_attr_user_topup,unique"`
	ConsumedQuota int   `json:"consumed_quota" gorm:"not null;default:0"`
	UpdatedAt     int64 `json:"updated_at" gorm:"autoUpdateTime;bigint"`
}

func (TopUpConsumeAttribution) TableName() string { return "topup_consume_attributions" }

// invoiceTopUpIDColumn 返回充值订单外键列名。
// GORM 将 TopUpId 映射为 top_up_id；历史 MySQL 若仍为 topup_id，由 migrateInvoiceTopUpIDColumns 先重命名。
func invoiceTopUpIDColumn() string {
	return "top_up_id"
}

// migrateInvoiceTopUpIDColumns 将历史 MySQL 列 topup_id 重命名为 GORM 标准列 top_up_id。
func migrateInvoiceTopUpIDColumns() error {
	tables := []string{"invoice_request_items", "topup_consume_attributions"}
	for _, table := range tables {
		if !DB.Migrator().HasTable(table) {
			continue
		}
		legacyExists, err := invoiceTableHasPhysicalColumn(table, "topup_id")
		if err != nil {
			return err
		}
		canonicalExists, err := invoiceTableHasPhysicalColumn(table, "top_up_id")
		if err != nil {
			return err
		}
		if !legacyExists || canonicalExists {
			continue
		}
		if err := renameInvoiceTopUpIDColumn(table); err != nil {
			return fmt.Errorf("rename %s.topup_id -> top_up_id: %w", table, err)
		}
		common.SysLog(fmt.Sprintf("migrated %s.topup_id to top_up_id", table))
	}
	return nil
}

func renameInvoiceTopUpIDColumn(table string) error {
	if common.UsingPostgreSQL {
		return DB.Exec(fmt.Sprintf(`ALTER TABLE %s RENAME COLUMN topup_id TO top_up_id`, table)).Error
	}
	if common.UsingSQLite {
		return DB.Exec(fmt.Sprintf(`ALTER TABLE %s RENAME COLUMN topup_id TO top_up_id`, table)).Error
	}
	// MySQL 5.7 不支持 RENAME COLUMN，需用 CHANGE 并保留原列类型。
	var columnType string
	if err := DB.Raw(`SELECT COLUMN_TYPE FROM information_schema.columns
		WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?`,
		table, "topup_id").Scan(&columnType).Error; err != nil {
		return err
	}
	if columnType == "" {
		return fmt.Errorf("empty COLUMN_TYPE for %s.topup_id", table)
	}
	var isNullable string
	if err := DB.Raw(`SELECT IS_NULLABLE FROM information_schema.columns
		WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?`,
		table, "topup_id").Scan(&isNullable).Error; err != nil {
		return err
	}
	nullSQL := "NULL"
	if strings.EqualFold(isNullable, "NO") {
		nullSQL = "NOT NULL"
	}
	return DB.Exec(fmt.Sprintf("ALTER TABLE `%s` CHANGE `topup_id` `top_up_id` %s %s", table, columnType, nullSQL)).Error
}

func invoiceTableHasPhysicalColumn(tableName, columnName string) (bool, error) {
	if common.UsingPostgreSQL {
		var count int64
		err := DB.Raw(`SELECT COUNT(1) FROM information_schema.columns
			WHERE table_schema = current_schema() AND table_name = ? AND column_name = ?`,
			tableName, columnName).Scan(&count).Error
		return count > 0, err
	}
	if common.UsingMySQL {
		var count int64
		err := DB.Raw(`SELECT COUNT(1) FROM information_schema.columns
			WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?`,
			tableName, columnName).Scan(&count).Error
		return count > 0, err
	}
	// SQLite：pragma_table_info 的表名不能参数绑定。
	var count int64
	err := DB.Raw(fmt.Sprintf("SELECT COUNT(1) FROM pragma_table_info('%s') WHERE name = ?", tableName), columnName).Scan(&count).Error
	return count > 0, err
}

// InvoiceEligibleOrder 待开票订单列表项。
type InvoiceEligibleOrder struct {
	TopUpId           int     `json:"topup_id"`
	TradeNo           string  `json:"trade_no"`
	Money             float64 `json:"money"`
	QuotaToAdd        int     `json:"quota_to_add"`
	ConsumedQuota     int     `json:"consumed_quota"`
	ConsumedAmount    float64 `json:"consumed_amount"`
	InvoicedAmount    float64 `json:"invoiced_amount"`
	PendingAmount     float64 `json:"pending_amount"`
	InvoiceableAmount float64 `json:"invoiceable_amount"`
	CreateTime        int64   `json:"create_time"`
	Status            string  `json:"status"`
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
	if profile.TitleType != InvoiceTitleTypePersonal && profile.TitleType != InvoiceTitleTypeCompany {
		return errors.New("invalid invoice title type")
	}
	if profile.InvoiceType == "" {
		profile.InvoiceType = InvoiceTypeElectronicOrdinary
	}
	if profile.InvoiceType != InvoiceTypeElectronicOrdinary && profile.InvoiceType != InvoiceTypeElectronicSpecial {
		return errors.New("invalid invoice type")
	}
	if profile.TitleType == InvoiceTitleTypeCompany && profile.TaxNo == "" {
		return errors.New("company tax number is required")
	}
	if _, err := mail.ParseAddress(profile.Email); err != nil {
		return errors.New("invalid invoice email")
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

func GetTopUpInvoiceableAmount(topUp *TopUp) (float64, error) {
	if topUp == nil || topUp.Status != common.TopUpStatusSuccess {
		return 0, nil
	}
	available := decimal.NewFromFloat(topUp.Money).Sub(decimal.NewFromFloat(topUp.InvoicedAmount))
	available = available.Sub(decimal.NewFromFloat(topUp.PendingInvoiceAmount))
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
		invoiceable, err := GetTopUpInvoiceableAmount(topUp)
		if err != nil {
			return nil, err
		}
		out = append(out, InvoiceEligibleOrder{
			TopUpId:           topUp.Id,
			TradeNo:           topUp.TradeNo,
			Money:             topUp.Money,
			QuotaToAdd:        topUp.ResolveQuotaToAdd(),
			ConsumedQuota:     consumedQuota,
			ConsumedAmount:    consumedAmount,
			InvoicedAmount:    topUp.InvoicedAmount,
			PendingAmount:     topUp.PendingInvoiceAmount,
			InvoiceableAmount: invoiceable,
			CreateTime:        topUp.CreateTime,
			Status:            topUp.Status,
		})
	}
	return out, nil
}

func getTopUpAttributionMap(userID int) (map[int]int, error) {
	return getTopUpAttributionMapWithDB(DB, userID)
}

func getTopUpAttributionMapWithDB(db *gorm.DB, userID int) (map[int]int, error) {
	var rows []TopUpConsumeAttribution
	if err := db.Where("user_id = ?", userID).Find(&rows).Error; err != nil {
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
		if err := tx.Where("user_id = ? AND status = ? AND "+topUpInvoiceEligibleWhere(), userID, common.TopUpStatusSuccess).
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

func ReleaseConsumeQuotaFromTopUps(userID, refundQuota int) error {
	if userID <= 0 || refundQuota <= 0 {
		return nil
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var rows []TopUpConsumeAttribution
		if err := tx.Where("user_id = ? AND consumed_quota > 0", userID).
			Order("updated_at desc, id desc").Find(&rows).Error; err != nil {
			return err
		}
		remaining := refundQuota
		for _, row := range rows {
			if remaining <= 0 {
				break
			}
			release := remaining
			if release > row.ConsumedQuota {
				release = row.ConsumedQuota
			}
			result := tx.Model(&TopUpConsumeAttribution{}).
				Where("id = ? AND consumed_quota >= ?", row.Id, release).
				Update("consumed_quota", gorm.Expr("consumed_quota - ?", release))
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return fmt.Errorf("consume attribution changed for topup %d", row.TopUpId)
			}
			remaining -= release
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
	profileJSON, err := common.Marshal(profile)
	if err != nil {
		return nil, err
	}
	merged := make(map[int]decimal.Decimal, len(items))
	for _, item := range items {
		if item.TopUpId <= 0 || item.InvoiceAmount <= 0 {
			return nil, fmt.Errorf("invalid invoice item for topup %d", item.TopUpId)
		}
		amount := decimal.NewFromFloat(item.InvoiceAmount).Round(6)
		merged[item.TopUpId] = merged[item.TopUpId].Add(amount)
	}
	topUpIDs := make([]int, 0, len(merged))
	for topUpID := range merged {
		topUpIDs = append(topUpIDs, topUpID)
	}
	sort.Ints(topUpIDs)
	var created *InvoiceRequest
	err = DB.Transaction(func(tx *gorm.DB) error {
		total := decimal.Zero
		prepared := make([]InvoiceRequestItem, 0, len(topUpIDs))
		for _, topUpID := range topUpIDs {
			amount := merged[topUpID]
			var topUp TopUp
			if err := tx.Where("id = ? AND user_id = ? AND status = ?", topUpID, userID, common.TopUpStatusSuccess).First(&topUp).Error; err != nil {
				return fmt.Errorf("topup %d not found or not payable", topUpID)
			}
			maxReserved := decimal.NewFromFloat(topUp.Money).Sub(decimal.NewFromFloat(topUp.InvoicedAmount)).Round(6)
			if maxReserved.IsNegative() {
				maxReserved = decimal.Zero
			}
			amountF, _ := amount.Float64()
			maxReservedF, _ := maxReserved.Float64()
			reserve := tx.Model(&TopUp{}).
				Where("id = ? AND user_id = ? AND status = ? AND pending_invoice_amount + ? <= ?", topUp.Id, userID, common.TopUpStatusSuccess, amountF, maxReservedF+0.000001).
				Update("pending_invoice_amount", gorm.Expr("pending_invoice_amount + ?", amountF))
			if reserve.Error != nil {
				return reserve.Error
			}
			if reserve.RowsAffected != 1 {
				return fmt.Errorf("invoice amount exceeds available for %s", topUp.TradeNo)
			}
			total = total.Add(amount)
			prepared = append(prepared, InvoiceRequestItem{
				TopUpId:       topUp.Id,
				TradeNo:       topUp.TradeNo,
				InvoiceAmount: amountF,
			})
		}
		totalF, _ := total.Float64()
		invoiceType := profile.InvoiceType
		if invoiceType == "" {
			invoiceType = InvoiceTypeElectronicOrdinary
		}
		if invoiceType != InvoiceTypeElectronicOrdinary && invoiceType != InvoiceTypeElectronicSpecial {
			return errors.New("invalid invoice type")
		}
		req := &InvoiceRequest{
			UserId:          userID,
			RequestNo:       generateInvoiceRequestNo(),
			Status:          InvoiceRequestStatusPending,
			TotalAmount:     totalF,
			InvoiceType:     invoiceType,
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

func ListInvoiceRequestsAdmin(status, keyword string, pageInfo *common.PageInfo) ([]*InvoiceRequest, int64, error) {
	var total int64
	tx := DB.Model(&InvoiceRequest{})
	if s := strings.TrimSpace(status); s != "" {
		tx = tx.Where("status = ?", s)
	}
	if kw := strings.TrimSpace(keyword); kw != "" {
		like := "%" + kw + "%"
		var userIDs []int
		if err := DB.Model(&User{}).Where("username LIKE ? OR email LIKE ?", like, like).Pluck("id", &userIDs).Error; err != nil {
			return nil, 0, err
		}
		if len(userIDs) > 0 {
			tx = tx.Where("request_no LIKE ? OR user_id IN ?", like, userIDs)
		} else {
			tx = tx.Where("request_no LIKE ?", like)
		}
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
	err := DB.Model(&InvoiceRequestItem{}).
		Select("invoice_request_items.*, top_ups.payment_method AS payment_method").
		Joins("LEFT JOIN top_ups ON top_ups.id = invoice_request_items."+invoiceTopUpIDColumn()).
		Where("invoice_request_items.invoice_request_id = ?", requestID).
		Order("invoice_request_items.id asc").
		Find(&items).Error
	return items, err
}

func IssueInvoiceRequest(requestID int, invoiceCode, invoiceURL, adminNote string) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var req InvoiceRequest
		if err := tx.Where("id = ?", requestID).First(&req).Error; err != nil {
			return err
		}
		var items []InvoiceRequestItem
		if err := tx.Where("invoice_request_id = ?", requestID).Find(&items).Error; err != nil {
			return err
		}
		now := time.Now().Unix()
		update := tx.Model(&InvoiceRequest{}).
			Where("id = ? AND status IN ?", requestID, []string{InvoiceRequestStatusPending, InvoiceRequestStatusProcessing}).
			Updates(map[string]interface{}{
				"status":       InvoiceRequestStatusIssued,
				"invoice_code": strings.TrimSpace(invoiceCode),
				"invoice_url":  strings.TrimSpace(invoiceURL),
				"admin_note":   strings.TrimSpace(adminNote),
				"issued_at":    now,
			})
		if update.Error != nil {
			return update.Error
		}
		if update.RowsAffected != 1 {
			return fmt.Errorf("request status not issuable: %s", req.Status)
		}
		for _, item := range items {
			result := tx.Model(&TopUp{}).Where("id = ? AND pending_invoice_amount + ? >= ?", item.TopUpId, 0.000001, item.InvoiceAmount).
				Updates(map[string]interface{}{
					"invoiced_amount":        gorm.Expr("invoiced_amount + ?", item.InvoiceAmount),
					"pending_invoice_amount": gorm.Expr("CASE WHEN pending_invoice_amount - ? < 0 THEN 0 ELSE pending_invoice_amount - ? END", item.InvoiceAmount, item.InvoiceAmount),
				})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return fmt.Errorf("pending invoice reservation missing for topup %d", item.TopUpId)
			}
		}
		return nil
	})
}

func RejectInvoiceRequest(requestID int, adminNote string) error {
	if strings.TrimSpace(adminNote) == "" {
		return errors.New("rejection reason is required")
	}
	return transitionInvoiceRequestAndRelease(requestID, 0, InvoiceRequestStatusRejected, adminNote, []string{InvoiceRequestStatusPending, InvoiceRequestStatusProcessing})
}

func CancelInvoiceRequest(userID, requestID int) error {
	return transitionInvoiceRequestAndRelease(requestID, userID, InvoiceRequestStatusCancelled, "", []string{InvoiceRequestStatusPending})
}

func MarkInvoiceRequestProcessing(requestID int) error {
	result := DB.Model(&InvoiceRequest{}).
		Where("id = ? AND status = ?", requestID, InvoiceRequestStatusPending).
		Update("status", InvoiceRequestStatusProcessing)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("request is no longer pending")
	}
	return nil
}

func transitionInvoiceRequestAndRelease(requestID, userID int, targetStatus, adminNote string, allowedStatuses []string) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		query := tx.Model(&InvoiceRequest{}).Where("id = ? AND status IN ?", requestID, allowedStatuses)
		if userID > 0 {
			query = query.Where("user_id = ?", userID)
		}
		updates := map[string]interface{}{"status": targetStatus}
		if targetStatus == InvoiceRequestStatusRejected {
			updates["admin_note"] = strings.TrimSpace(adminNote)
		}
		result := query.Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return fmt.Errorf("invoice request status changed, please refresh")
		}
		var items []InvoiceRequestItem
		if err := tx.Where("invoice_request_id = ?", requestID).Find(&items).Error; err != nil {
			return err
		}
		for _, item := range items {
			if err := tx.Model(&TopUp{}).Where("id = ?", item.TopUpId).
				Update("pending_invoice_amount", gorm.Expr("CASE WHEN pending_invoice_amount - ? < 0 THEN 0 ELSE pending_invoice_amount - ? END", item.InvoiceAmount, item.InvoiceAmount)).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func BackfillPendingInvoiceAmounts() error {
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&TopUp{}).Where("pending_invoice_amount <> 0").Update("pending_invoice_amount", 0).Error; err != nil {
			return err
		}
		type pendingRow struct {
			TopUpID int
			Amount  float64
		}
		var rows []pendingRow
		if err := tx.Table("invoice_request_items").
			Select("invoice_request_items."+invoiceTopUpIDColumn()+" AS top_up_id, SUM(invoice_request_items.invoice_amount) AS amount").
			Joins("JOIN invoice_requests ON invoice_requests.id = invoice_request_items.invoice_request_id").
			Where("invoice_requests.status IN ?", []string{InvoiceRequestStatusPending, InvoiceRequestStatusProcessing}).
			Group("invoice_request_items." + invoiceTopUpIDColumn()).Scan(&rows).Error; err != nil {
			return err
		}
		for _, row := range rows {
			if err := tx.Model(&TopUp{}).Where("id = ?", row.TopUpID).Update("pending_invoice_amount", row.Amount).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
