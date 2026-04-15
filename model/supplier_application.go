package model

import (
	"errors"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	// SupplierApplicationStatusPending 表示待审核。
	SupplierApplicationStatusPending = 0
	// SupplierApplicationStatusApproved 表示审核通过。
	SupplierApplicationStatusApproved = 1
	// SupplierApplicationStatusRejected 表示审核驳回。
	SupplierApplicationStatusRejected = 2
	// SupplierApplicationStatusDeactivated 表示供应商已注销。
	SupplierApplicationStatusDeactivated = 3
)

const (
	// SupplierApplicationAuditActionSubmit 表示提交申请。
	SupplierApplicationAuditActionSubmit = 0
	// SupplierApplicationAuditActionApprove 表示审核通过。
	SupplierApplicationAuditActionApprove = 1
	// SupplierApplicationAuditActionReject 表示审核驳回。
	SupplierApplicationAuditActionReject = 2
	// SupplierApplicationAuditActionDeactivate 表示供应商注销。
	SupplierApplicationAuditActionDeactivate = 3
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
	// ErrSupplierApplicationStatusNotEditable 表示申请当前状态不可修改（已审核通过）。
	ErrSupplierApplicationStatusNotEditable = errors.New("supplier application status is not editable")
	// ErrSupplierApplicationStatusNotApproved 表示申请当前不处于审核通过状态，不能执行供应商注销。
	ErrSupplierApplicationStatusNotApproved = errors.New("supplier application status is not approved")
)

// SupplierApplication 供应商入驻申请主表。
type SupplierApplication struct {
	ID                  int    `json:"id" gorm:"primaryKey;comment:主键ID"`
	ApplicantUserID     int    `json:"applicant_user_id" gorm:"index;not null;comment:申请人用户ID"`
	CompanyName         string `json:"company_name" gorm:"type:varchar(255);not null;comment:企业或主体名称"`
	CreditCode          string `json:"credit_code" gorm:"type:varchar(32);not null;uniqueIndex;comment:统一社会信用代码"`
	BusinessLicenseURL  string `json:"business_license_url" gorm:"type:varchar(1024);not null;comment:营业执照文件URL"`
	BusinessLicenseFile string `json:"business_license_file" gorm:"type:varchar(255);not null;default:'';comment:营业执照文件名称"`
	LegalRepresentative string `json:"legal_representative" gorm:"type:varchar(128);not null;comment:法人或经营者姓名"`
	CompanySize         string `json:"company_size" gorm:"type:varchar(64);comment:企业规模"`
	ContactName         string `json:"contact_name" gorm:"type:varchar(128);not null;comment:对接人姓名"`
	ContactMobile       string `json:"contact_mobile" gorm:"type:varchar(32);not null;comment:对接人手机号"`
	ContactWechat       string `json:"contact_wechat" gorm:"type:varchar(128);not null;comment:对接人微信或企业微信"`
	Status              int    `json:"status" gorm:"type:int;index;default:0;not null;comment:审核状态 0待审核 1已通过 2已驳回 3已注销"`
	ReviewReason        string `json:"review_reason" gorm:"type:text;comment:审核备注或驳回原因"`
	ReviewedBy          int    `json:"reviewed_by" gorm:"type:int;index;default:0;comment:审核人用户ID"`
	ReviewedAt          int64  `json:"reviewed_at" gorm:"type:bigint;default:0;comment:审核时间戳"`
	CreatedAt           int64  `json:"created_at" gorm:"type:bigint;index;comment:创建时间戳"`
	UpdatedAt           int64  `json:"updated_at" gorm:"type:bigint;comment:更新时间戳"`
}

// SupplierApplicationAudit 供应商审核极简审计表。
type SupplierApplicationAudit struct {
	ID             int    `json:"id" gorm:"primaryKey;comment:主键ID"`
	ApplicationID  int    `json:"application_id" gorm:"index;not null;comment:供应商申请ID"`
	OperatorUserID int    `json:"operator_user_id" gorm:"index;not null;comment:操作人用户ID"`
	Action         int    `json:"action" gorm:"type:int;index;not null;comment:操作类型 0提交 1通过 2驳回"`
	FromStatus     int    `json:"from_status" gorm:"type:int;not null;comment:变更前状态"`
	ToStatus       int    `json:"to_status" gorm:"type:int;not null;comment:变更后状态"`
	Reason         string `json:"reason" gorm:"type:text;comment:审核备注或驳回原因"`
	CreatedAt      int64  `json:"created_at" gorm:"type:bigint;index;comment:创建时间戳"`
}

// UserMessage 通用站内消息表（支持按用户与按角色广播）。
type UserMessage struct {
	ID              int    `json:"id" gorm:"primaryKey;comment:主键ID"`
	ReceiverUserID  int    `json:"receiver_user_id" gorm:"type:int;index;default:0;comment:接收用户ID 0表示广播"`
	ReceiverMinRole int    `json:"receiver_min_role" gorm:"type:int;index;default:0;comment:广播最小角色门槛"`
	Type            string `json:"type" gorm:"type:varchar(64);index;comment:消息类型"`
	Title           string `json:"title" gorm:"type:varchar(255);not null;comment:消息标题"`
	Content         string `json:"content" gorm:"type:text;not null;comment:消息内容"`
	BizType         string `json:"biz_type" gorm:"type:varchar(64);index;comment:业务类型"`
	BizID           int    `json:"biz_id" gorm:"type:int;index;default:0;comment:业务ID"`
	IsRead          bool   `json:"is_read" gorm:"type:boolean;index;default:false;comment:是否已读"`
	ReadAt          int64  `json:"read_at" gorm:"type:bigint;default:0;comment:已读时间戳"`
	CreatedAt       int64  `json:"created_at" gorm:"type:bigint;index;comment:创建时间戳"`
}

// SupplierSimplePricingItem pricing 接口使用的供应商精简信息。
type SupplierSimplePricingItem struct {
	SupplierID   int    `json:"supplier_id"`
	SupplierName string `json:"supplier_name"`
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

// ListSuppliersByCompanyName 分页查询供应商列表（支持按供应商名称模糊搜索）。
func ListSuppliersByCompanyName(companyName string, pageInfo *common.PageInfo) ([]*SupplierApplication, int64, error) {
	var (
		items []*SupplierApplication
		total int64
	)
	query := DB.Model(&SupplierApplication{})
	if strings.TrimSpace(companyName) != "" {
		query = query.Where("company_name LIKE ?", "%"+strings.TrimSpace(companyName)+"%")
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

// GetMySupplierApplication 获取当前用户最近一条供应商申请（按ID倒序）。
func GetMySupplierApplication(applicantUserID int) (*SupplierApplication, error) {
	var app SupplierApplication
	err := DB.Where("applicant_user_id = ?", applicantUserID).
		Order("id desc").
		First(&app).Error
	if err != nil {
		return nil, err
	}
	return &app, nil
}

// UpdateMySupplierApplication 修改当前用户指定ID申请（仅已通过状态不可修改）。
func UpdateMySupplierApplication(applicantUserID int, applicationID int, req *SupplierApplication) (*SupplierApplication, error) {
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
		Where("applicant_user_id = ? AND id = ?", applicantUserID, applicationID).
		First(&app).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	if app.Status == SupplierApplicationStatusApproved {
		tx.Rollback()
		return nil, ErrSupplierApplicationStatusNotEditable
	}
	fromStatus := app.Status
	now := time.Now().Unix()
	updates := map[string]any{
		"company_name":          req.CompanyName,
		"credit_code":           req.CreditCode,
		"business_license_url":  req.BusinessLicenseURL,
		"business_license_file": req.BusinessLicenseFile,
		"legal_representative":  req.LegalRepresentative,
		"company_size":          req.CompanySize,
		"contact_name":          req.ContactName,
		"contact_mobile":        req.ContactMobile,
		"contact_wechat":        req.ContactWechat,
		"status":                SupplierApplicationStatusPending,
		"review_reason":         "",
		"reviewed_by":           0,
		"reviewed_at":           0,
		"updated_at":            now,
	}
	if err := tx.Model(&SupplierApplication{}).
		Where("id = ?", app.ID).
		Updates(updates).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := tx.Create(&SupplierApplicationAudit{
		ApplicationID:  app.ID,
		OperatorUserID: applicantUserID,
		Action:         SupplierApplicationAuditActionSubmit,
		FromStatus:     fromStatus,
		ToStatus:       SupplierApplicationStatusPending,
		Reason:         "申请资料已修改并重新提交",
		CreatedAt:      now,
	}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	app.CompanyName = req.CompanyName
	app.CreditCode = req.CreditCode
	app.BusinessLicenseURL = req.BusinessLicenseURL
	app.BusinessLicenseFile = req.BusinessLicenseFile
	app.LegalRepresentative = req.LegalRepresentative
	app.CompanySize = req.CompanySize
	app.ContactName = req.ContactName
	app.ContactMobile = req.ContactMobile
	app.ContactWechat = req.ContactWechat
	app.Status = SupplierApplicationStatusPending
	app.ReviewReason = ""
	app.ReviewedBy = 0
	app.ReviewedAt = 0
	app.UpdatedAt = now
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}
	return &app, nil

}

// ListApprovedSuppliersForPricing 查询定价页使用的已审核通过供应商列表。
func ListApprovedSuppliersForPricing() ([]SupplierSimplePricingItem, error) {
	items := make([]SupplierSimplePricingItem, 0)
	err := DB.Model(&SupplierApplication{}).
		Select("id as supplier_id, company_name as supplier_name").
		Where("status = ?", SupplierApplicationStatusApproved).
		Order("id desc").
		Scan(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}

// GetApprovedSupplierApplicationByApplicant 获取当前用户的审核通过供应商申请。
func GetApprovedSupplierApplicationByApplicant(applicantUserID int) (*SupplierApplication, error) {
	var app SupplierApplication
	err := DB.Where("applicant_user_id = ? AND status = ?", applicantUserID, SupplierApplicationStatusApproved).
		Order("id desc").
		First(&app).Error
	if err != nil {
		return nil, err
	}
	return &app, nil
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
	if toStatus == SupplierApplicationStatusApproved {
		// 审核通过后，回填用户表 supplier_id，建立用户与供应商关联。
		if err := tx.Model(&User{}).Where("id = ?", app.ApplicantUserID).
			Updates(map[string]any{
				"supplier_id": app.ID,
			}).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
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

// DeactivateSupplierApplication 注销供应商（仅允许审核通过状态）。
// 注销后会将 supplier_applications.status 置为已注销，并清空用户表 supplier_id。
func DeactivateSupplierApplication(applicantUserID int, reason string) (*SupplierApplication, error) {
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
		Where("applicant_user_id = ?", applicantUserID).
		Order("id desc").
		First(&app).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	if app.Status != SupplierApplicationStatusApproved {
		tx.Rollback()
		return nil, ErrSupplierApplicationStatusNotApproved
	}
	now := time.Now().Unix()
	if err := tx.Model(&SupplierApplication{}).
		Where("id = ?", app.ID).
		Updates(map[string]any{
			"status":        SupplierApplicationStatusDeactivated,
			"review_reason": strings.TrimSpace(reason),
			"updated_at":    now,
		}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := tx.Model(&User{}).Where("id = ?", applicantUserID).
		Updates(map[string]any{
			"supplier_id": 0,
		}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := tx.Create(&SupplierApplicationAudit{
		ApplicationID:  app.ID,
		OperatorUserID: applicantUserID,
		Action:         SupplierApplicationAuditActionDeactivate,
		FromStatus:     SupplierApplicationStatusApproved,
		ToStatus:       SupplierApplicationStatusDeactivated,
		Reason:         strings.TrimSpace(reason),
		CreatedAt:      now,
	}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	app.Status = SupplierApplicationStatusDeactivated
	app.ReviewReason = strings.TrimSpace(reason)
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

// IsSupplierApplicationNotFound 判断是否未找到供应商申请记录。
func IsSupplierApplicationNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

// IsSupplierCreditCodeDuplicateError 判断是否为统一社会信用代码重复错误。
func IsSupplierCreditCodeDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	lowerMsg := strings.ToLower(err.Error())
	if !strings.Contains(lowerMsg, "credit_code") {
		return false
	}
	// 兼容 MySQL / PostgreSQL / SQLite 常见唯一约束错误文案
	return strings.Contains(lowerMsg, "duplicate") ||
		strings.Contains(lowerMsg, "duplicated") ||
		strings.Contains(lowerMsg, "unique constraint") ||
		strings.Contains(lowerMsg, "unique failed") ||
		strings.Contains(lowerMsg, "idx_supplier_applications_credit_code")
}
