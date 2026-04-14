package model

import (
	"errors"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm/clause"
)

const (
	// SupplierApplicationStatusPending 表示待审核。
	SupplierApplicationStatusPending = 0
	// SupplierApplicationStatusApproved 表示审核通过。
	SupplierApplicationStatusApproved = 1
	// SupplierApplicationStatusRejected 表示审核驳回。
	SupplierApplicationStatusRejected = 2
)

const (
	// SupplierApplicationAuditActionSubmit 表示提交申请。
	SupplierApplicationAuditActionSubmit = 0
	// SupplierApplicationAuditActionApprove 表示审核通过。
	SupplierApplicationAuditActionApprove = 1
	// SupplierApplicationAuditActionReject 表示审核驳回。
	SupplierApplicationAuditActionReject = 2
)

const (
	// UserMessageBizTypeSupplierApplication 供应商申请业务类型。
	UserMessageBizTypeSupplierApplication = "supplier_application"
	// UserMessageTypeSupplierSubmitted 供应商提交待审核消息。
	UserMessageTypeSupplierSubmitted = "supplier_submitted"
	// UserMessageTypeSupplierApproved 供应商审核通过消息。
	UserMessageTypeSupplierApproved = "supplier_approved"
	// UserMessageTypeSupplierRejected 供应商审核驳回消息。
	UserMessageTypeSupplierRejected = "supplier_rejected"
)

var (
	// ErrSupplierApplicationAlreadyReviewed 表示申请已被其他管理员处理。
	ErrSupplierApplicationAlreadyReviewed = errors.New("supplier application already reviewed")
)

// SupplierApplication 供应商入驻申请主表。
type SupplierApplication struct {
	ID                int    `json:"id" gorm:"primaryKey"`
	ApplicantUserID   int    `json:"applicant_user_id" gorm:"index;not null"`
	CompanyName       string `json:"company_name" gorm:"type:varchar(255);not null"`
	CreditCode        string `json:"credit_code" gorm:"type:varchar(32);not null;uniqueIndex"`
	BusinessLicenseURL string `json:"business_license_url" gorm:"type:varchar(1024);not null"`
	LegalRepresentative string `json:"legal_representative" gorm:"type:varchar(128);not null"`
	CompanySize       string `json:"company_size" gorm:"type:varchar(64)"`
	ContactName       string `json:"contact_name" gorm:"type:varchar(128);not null"`
	ContactMobile     string `json:"contact_mobile" gorm:"type:varchar(32);not null"`
	ContactWechat     string `json:"contact_wechat" gorm:"type:varchar(128);not null"`
	Status            int    `json:"status" gorm:"type:int;index;default:0;not null"`
	ReviewReason      string `json:"review_reason" gorm:"type:text"`
	ReviewedBy        int    `json:"reviewed_by" gorm:"type:int;index;default:0"`
	ReviewedAt        int64  `json:"reviewed_at" gorm:"type:bigint;default:0"`
	CreatedAt         int64  `json:"created_at" gorm:"type:bigint;index"`
	UpdatedAt         int64  `json:"updated_at" gorm:"type:bigint"`
}

// SupplierApplicationAudit 供应商审核极简审计表。
type SupplierApplicationAudit struct {
	ID             int    `json:"id" gorm:"primaryKey"`
	ApplicationID  int    `json:"application_id" gorm:"index;not null"`
	OperatorUserID int    `json:"operator_user_id" gorm:"index;not null"`
	Action         int    `json:"action" gorm:"type:int;index;not null"`
	FromStatus     int    `json:"from_status" gorm:"type:int;not null"`
	ToStatus       int    `json:"to_status" gorm:"type:int;not null"`
	Reason         string `json:"reason" gorm:"type:text"`
	CreatedAt      int64  `json:"created_at" gorm:"type:bigint;index"`
}

// UserMessage 通用站内消息表（支持按用户与按角色广播）。
type UserMessage struct {
	ID              int    `json:"id" gorm:"primaryKey"`
	ReceiverUserID  int    `json:"receiver_user_id" gorm:"type:int;index;default:0"`
	ReceiverMinRole int    `json:"receiver_min_role" gorm:"type:int;index;default:0"`
	Type            string `json:"type" gorm:"type:varchar(64);index"`
	Title           string `json:"title" gorm:"type:varchar(255);not null"`
	Content         string `json:"content" gorm:"type:text;not null"`
	BizType         string `json:"biz_type" gorm:"type:varchar(64);index"`
	BizID           int    `json:"biz_id" gorm:"type:int;index;default:0"`
	IsRead          bool   `json:"is_read" gorm:"type:boolean;index;default:false"`
	ReadAt          int64  `json:"read_at" gorm:"type:bigint;default:0"`
	CreatedAt       int64  `json:"created_at" gorm:"type:bigint;index"`
}

// CreateSupplierApplication 创建供应商申请记录。
func CreateSupplierApplication(app *SupplierApplication) error {
	now := time.Now().Unix()
	app.CreatedAt = now
	app.UpdatedAt = now
	return DB.Create(app).Error
}

// CreateSupplierApplicationAudit 创建供应商审核审计记录。
func CreateSupplierApplicationAudit(audit *SupplierApplicationAudit) error {
	audit.CreatedAt = time.Now().Unix()
	return DB.Create(audit).Error
}

// ListSupplierApplications 分页查询供应商申请（管理员）。
func ListSupplierApplications(status *int, pageInfo *common.PageInfo) ([]*SupplierApplication, int64, error) {
	var (
		items []*SupplierApplication
		total int64
	)
	query := DB.Model(&SupplierApplication{})
	if status != nil {
		query = query.Where("status = ?", *status)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("id desc").
		Limit(pageInfo.GetPageSize()).
		Offset(pageInfo.GetStartIdx()).
		Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// ListSupplierApplicationsByApplicant 分页查询当前用户提交的申请。
func ListSupplierApplicationsByApplicant(applicantUserID int, status *int, pageInfo *common.PageInfo) ([]*SupplierApplication, int64, error) {
	var (
		items []*SupplierApplication
		total int64
	)
	query := DB.Model(&SupplierApplication{}).Where("applicant_user_id = ?", applicantUserID)
	if status != nil {
		query = query.Where("status = ?", *status)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("id desc").
		Limit(pageInfo.GetPageSize()).
		Offset(pageInfo.GetStartIdx()).
		Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// ReviewSupplierApplication 审核申请（仅允许从待审核状态流转到通过/驳回）。
func ReviewSupplierApplication(applicationID int, reviewerUserID int, toStatus int, reason string) (*SupplierApplication, error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var app SupplierApplication
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", applicationID).
		First(&app).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	if app.Status != SupplierApplicationStatusPending {
		tx.Rollback()
		return nil, ErrSupplierApplicationAlreadyReviewed
	}

	now := time.Now().Unix()
	updates := map[string]any{
		"status":        toStatus,
		"review_reason": reason,
		"reviewed_by":   reviewerUserID,
		"reviewed_at":   now,
		"updated_at":    now,
	}
	result := tx.Model(&SupplierApplication{}).
		Where("id = ? AND status = ?", applicationID, SupplierApplicationStatusPending).
		Updates(updates)
	if result.Error != nil {
		tx.Rollback()
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		tx.Rollback()
		return nil, ErrSupplierApplicationAlreadyReviewed
	}

	action := SupplierApplicationAuditActionApprove
	if toStatus == SupplierApplicationStatusRejected {
		action = SupplierApplicationAuditActionReject
	}
	if err := tx.Create(&SupplierApplicationAudit{
		ApplicationID:  app.ID,
		OperatorUserID: reviewerUserID,
		Action:         action,
		FromStatus:     SupplierApplicationStatusPending,
		ToStatus:       toStatus,
		Reason:         reason,
		CreatedAt:      now,
	}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	app.Status = toStatus
	app.ReviewReason = reason
	app.ReviewedBy = reviewerUserID
	app.ReviewedAt = now
	app.UpdatedAt = now
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}
	return &app, nil
}

// CreateUserMessage 创建站内消息。
func CreateUserMessage(msg *UserMessage) error {
	msg.CreatedAt = time.Now().Unix()
	return DB.Create(msg).Error
}

// ListUserMessagesForUser 分页查询当前用户可见消息。
func ListUserMessagesForUser(userID int, role int, pageInfo *common.PageInfo) ([]*UserMessage, int64, error) {
	var (
		items []*UserMessage
		total int64
	)
	query := DB.Model(&UserMessage{}).Where("receiver_user_id = ? OR (receiver_user_id = 0 AND receiver_min_role > 0 AND receiver_min_role <= ?)", userID, role)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("id desc").
		Limit(pageInfo.GetPageSize()).
		Offset(pageInfo.GetStartIdx()).
		Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// CountUnreadUserMessages 统计当前用户未读消息（广播消息默认视为未读）。
func CountUnreadUserMessages(userID int, role int) (int64, error) {
	var total int64
	err := DB.Model(&UserMessage{}).
		Where("(receiver_user_id = ? AND is_read = ?) OR (receiver_user_id = 0 AND receiver_min_role > 0 AND receiver_min_role <= ?)", userID, false, role).
		Count(&total).Error
	return total, err
}

// MarkUserMessageAsRead 将用户定向消息标记已读（广播消息不做个人已读标记）。
func MarkUserMessageAsRead(messageID int, userID int) (bool, error) {
	now := time.Now().Unix()
	res := DB.Model(&UserMessage{}).
		Where("id = ? AND receiver_user_id = ? AND is_read = ?", messageID, userID, false).
		Updates(map[string]any{
			"is_read": true,
			"read_at": now,
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}
