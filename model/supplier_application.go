package model

import (
	"errors"
	"strconv"
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
	ID                  int                 `json:"id" gorm:"primaryKey;comment:主键ID"`
	ApplicantUserID     int                 `json:"applicant_user_id" gorm:"index;not null;comment:申请人用户ID"`
	ApplicantUsername   string              `json:"applicant_username" gorm:"column:applicant_username;->;comment:申请人用户名（关联 users.username）"`
	CompanyName         string              `json:"company_name" gorm:"type:varchar(255);not null;comment:企业或主体名称"`
	CreditCode          string              `json:"credit_code" gorm:"type:varchar(32);not null;uniqueIndex;comment:统一社会信用代码"`
	BusinessLicenseURL  string              `json:"business_license_url" gorm:"type:varchar(1024);not null;comment:营业执照文件URL"`
	BusinessLicenseFile string              `json:"business_license_file" gorm:"type:varchar(255);not null;default:'';comment:营业执照文件名称"`
	CompanyLogoURL      string              `json:"company_logo_url" gorm:"type:varchar(1024);not null;default:'';comment:企业Logo图片URL"`
	SupplierType        string              `json:"supplier_type" gorm:"type:varchar(64);not null;default:'';comment:供应商类型"`
	LegalRepresentative string              `json:"legal_representative" gorm:"type:varchar(128);not null;comment:法人或经营者姓名"`
	CompanySize         string              `json:"company_size" gorm:"type:varchar(64);comment:企业规模"`
	ContactName         string              `json:"contact_name" gorm:"type:varchar(128);not null;comment:对接人姓名"`
	ContactMobile       string              `json:"contact_mobile" gorm:"type:varchar(32);not null;comment:对接人手机号"`
	ContactWechat       string              `json:"contact_wechat" gorm:"type:varchar(128);not null;comment:对接人微信或企业微信"`
	SupplierAlias       *string             `json:"supplier_alias" gorm:"type:varchar(128);uniqueIndex;comment:供应商别名，创建时默认 P+主键 id，可修改"`
	SupplierCapability  *SupplierCapability `json:"supplier_capability" gorm:"-"`
	Status              int                 `json:"status" gorm:"type:int;index;default:0;not null;comment:审核状态 0待审核 1已通过 2已驳回 3已注销"`
	ReviewReason        string              `json:"review_reason" gorm:"type:text;comment:审核备注或驳回原因"`
	ReviewedBy          int                 `json:"reviewed_by" gorm:"type:int;index;default:0;comment:审核人用户ID"`
	ReviewedAt          int64               `json:"reviewed_at" gorm:"type:bigint;default:0;comment:审核时间戳"`
	CreatedAt           int64               `json:"created_at" gorm:"type:bigint;index;comment:创建时间戳"`
	UpdatedAt           int64               `json:"updated_at" gorm:"type:bigint;comment:更新时间戳"`
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

// UserMessageRead 站内广播消息的用户已读记录。
// 仅用于 receiver_user_id=0 的广播消息按用户追踪已读状态。
type UserMessageRead struct {
	ID        int   `json:"id" gorm:"primaryKey;comment:主键ID"`
	UserID    int   `json:"user_id" gorm:"type:int;not null;uniqueIndex:idx_user_message_reads_user_message,priority:1;comment:用户ID"`
	MessageID int   `json:"message_id" gorm:"type:int;not null;uniqueIndex:idx_user_message_reads_user_message,priority:2;comment:消息ID"`
	ReadAt    int64 `json:"read_at" gorm:"type:bigint;not null;default:0;comment:已读时间戳"`
	CreatedAt int64 `json:"created_at" gorm:"type:bigint;index;comment:创建时间戳"`
}

// SupplierSimplePricingItem pricing 接口使用的供应商精简信息。
type SupplierSimplePricingItem struct {
	SupplierID   int    `json:"supplier_id"`
	SupplierName string `json:"supplier_name"`
}

// SupplierApplicationAutoAlias 供应商库内编号：P + 主键 id（创建/编辑后由服务端写入，无需前端传入）。
func SupplierApplicationAutoAlias(id int) string {
	if id <= 0 {
		return ""
	}
	return "P" + strconv.Itoa(id)
}

func ptrSupplierApplicationAlias(id int) *string {
	if id <= 0 {
		return nil
	}
	s := SupplierApplicationAutoAlias(id)
	return &s
}

// CreateSupplierApplication 创建供应商申请记录。
func CreateSupplierApplication(app *SupplierApplication) error {
	now := time.Now().Unix()
	app.CreatedAt = now
	app.UpdatedAt = now
	if err := DB.Create(app).Error; err != nil {
		return err
	}
	alias := SupplierApplicationAutoAlias(app.ID)
	if err := DB.Model(app).Update("supplier_alias", alias).Error; err != nil {
		return err
	}
	app.SupplierAlias = ptrSupplierApplicationAlias(app.ID)
	return nil
}

// CreateSupplierApplicationAutoApproved 创建直接审核通过的供应商申请。
// 该方法用于管理员及以上角色提交申请时，保证申请创建、用户绑定与审计记录在同一事务内完成。
func CreateSupplierApplicationAutoApproved(app *SupplierApplication, reviewerUserID int) error {
	tx := DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	now := time.Now().Unix()
	app.Status = SupplierApplicationStatusApproved
	app.ReviewedBy = reviewerUserID
	app.ReviewedAt = now
	app.ReviewReason = ""
	app.CreatedAt = now
	app.UpdatedAt = now
	if err := tx.Create(app).Error; err != nil {
		tx.Rollback()
		return err
	}
	alias := SupplierApplicationAutoAlias(app.ID)
	if err := tx.Model(&SupplierApplication{}).Where("id = ?", app.ID).Update("supplier_alias", alias).Error; err != nil {
		tx.Rollback()
		return err
	}
	app.SupplierAlias = ptrSupplierApplicationAlias(app.ID)
	if err := tx.Model(&User{}).Where("id = ?", app.ApplicantUserID).
		Updates(map[string]any{
			"supplier_id": app.ID,
		}).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Create(&SupplierApplicationAudit{
		ApplicationID:  app.ID,
		OperatorUserID: app.ApplicantUserID,
		Action:         SupplierApplicationAuditActionSubmit,
		FromStatus:     SupplierApplicationStatusPending,
		ToStatus:       SupplierApplicationStatusPending,
		Reason:         "",
		CreatedAt:      now,
	}).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Create(&SupplierApplicationAudit{
		ApplicationID:  app.ID,
		OperatorUserID: reviewerUserID,
		Action:         SupplierApplicationAuditActionApprove,
		FromStatus:     SupplierApplicationStatusPending,
		ToStatus:       SupplierApplicationStatusApproved,
		Reason:         "管理员提交自动通过",
		CreatedAt:      now,
	}).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
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

// ListSuppliersByCompanyName 分页查询供应商列表（支持按供应商名称模糊搜索与状态筛选）。
func ListSuppliersByCompanyName(companyName string, statuses []int, pageInfo *common.PageInfo) ([]*SupplierApplication, int64, error) {
	var (
		items []*SupplierApplication
		total int64
	)
	query := DB.Model(&SupplierApplication{}).
		Joins("LEFT JOIN users ON users.id = supplier_applications.applicant_user_id")
	if len(statuses) > 0 {
		query = query.Where("supplier_applications.status IN ?", statuses)
	}
	if strings.TrimSpace(companyName) != "" {
		query = query.Where("company_name LIKE ?", "%"+strings.TrimSpace(companyName)+"%")
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.
		Select("supplier_applications.*, users.username AS applicant_username").
		Order("supplier_applications.id desc").
		Limit(pageInfo.GetPageSize()).
		Offset(pageInfo.GetStartIdx()).
		Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetSupplierByID 根据供应商申请ID查询供应商详情（含申请人用户名）。
func GetSupplierByID(supplierID int) (*SupplierApplication, error) {
	var item SupplierApplication
	err := DB.Model(&SupplierApplication{}).
		Select("supplier_applications.*, users.username AS applicant_username").
		Joins("LEFT JOIN users ON users.id = supplier_applications.applicant_user_id").
		Where("supplier_applications.id = ?", supplierID).
		First(&item).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
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
		"company_logo_url":      req.CompanyLogoURL,
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
	app.CompanyLogoURL = req.CompanyLogoURL
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

// AdminUpdateSupplierApplication 管理员更新指定供应商申请资料。
// 与用户侧不同：即使申请处于审核通过状态，也允许管理员更新；更新后保持原状态不变。
func AdminUpdateSupplierApplication(applicationID int, req *SupplierApplication) (*SupplierApplication, error) {
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
	now := time.Now().Unix()
	aliasVal := ""
	if req.SupplierAlias != nil {
		aliasVal = strings.TrimSpace(*req.SupplierAlias)
	}
	if aliasVal == "" {
		aliasVal = SupplierApplicationAutoAlias(applicationID)
	}
	updates := map[string]any{
		"company_name":          req.CompanyName,
		"credit_code":           req.CreditCode,
		"business_license_url":  req.BusinessLicenseURL,
		"business_license_file": req.BusinessLicenseFile,
		"company_logo_url":      req.CompanyLogoURL,
		"supplier_type":         req.SupplierType,
		"legal_representative":  req.LegalRepresentative,
		"company_size":          req.CompanySize,
		"contact_name":          req.ContactName,
		"contact_mobile":        req.ContactMobile,
		"contact_wechat":        req.ContactWechat,
		"supplier_alias":        aliasVal,
		"updated_at":            now,
	}
	if err := tx.Model(&SupplierApplication{}).
		Where("id = ?", app.ID).
		Updates(updates).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	app.CompanyName = req.CompanyName
	app.CreditCode = req.CreditCode
	app.BusinessLicenseURL = req.BusinessLicenseURL
	app.BusinessLicenseFile = req.BusinessLicenseFile
	app.CompanyLogoURL = req.CompanyLogoURL
	app.SupplierType = req.SupplierType
	app.LegalRepresentative = req.LegalRepresentative
	app.CompanySize = req.CompanySize
	app.ContactName = req.ContactName
	app.ContactMobile = req.ContactMobile
	app.ContactWechat = req.ContactWechat
	savedAlias := aliasVal
	app.SupplierAlias = &savedAlias
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
// supplierAlias 为可选：通过时若为空则写入默认 P+id，否则写入管理员指定别名。
func ReviewSupplierApplication(applicationID int, reviewerUserID int, toStatus int, reason string, supplierAlias string, supplierType string) (*SupplierApplication, error) {
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
	if toStatus == SupplierApplicationStatusApproved {
		trimmedAlias := strings.TrimSpace(supplierAlias)
		trimmedType := strings.TrimSpace(supplierType)
		if trimmedAlias != "" {
			updates["supplier_alias"] = trimmedAlias
		} else {
			updates["supplier_alias"] = SupplierApplicationAutoAlias(applicationID)
		}
		updates["supplier_type"] = trimmedType
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
	if toStatus == SupplierApplicationStatusApproved {
		trimmedAlias := strings.TrimSpace(supplierAlias)
		trimmedType := strings.TrimSpace(supplierType)
		if trimmedAlias != "" {
			app.SupplierAlias = &trimmedAlias
		} else {
			app.SupplierAlias = ptrSupplierApplicationAlias(applicationID)
		}
		app.SupplierType = trimmedType
	}
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}
	return &app, nil
}

// DeactivateSupplierApplication 注销供应商（仅允许审核通过状态）。
// 管理员（role>=RoleAdminUser）可按ID注销任意供应商；普通用户仅可注销自己提交的供应商。
// 注销后会将 supplier_applications.status 置为已注销，并清空申请人用户表 supplier_id。
func DeactivateSupplierApplication(operatorUserID int, operatorRole int, supplierID int, reason string) (*SupplierApplication, error) {
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
	query := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", supplierID)
	if operatorRole < common.RoleAdminUser {
		query = query.Where("applicant_user_id = ?", operatorUserID)
	}
	if err := query.First(&app).Error; err != nil {
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
	if err := tx.Model(&User{}).Where("id = ?", app.ApplicantUserID).
		Updates(map[string]any{
			"supplier_id": 0,
		}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := tx.Create(&SupplierApplicationAudit{
		ApplicationID:  app.ID,
		OperatorUserID: operatorUserID,
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
// readStatus: all/read/unread；titleKeyword: 标题模糊查询关键字。
func ListUserMessagesForUser(userID int, role int, pageInfo *common.PageInfo, titleKeyword string, readStatus string) ([]*UserMessage, int64, error) {
	var (
		items []*UserMessage
		total int64
	)
	query := DB.Model(&UserMessage{}).Where("receiver_user_id = ? OR (receiver_user_id = 0 AND receiver_min_role > 0 AND receiver_min_role <= ?)", userID, role)
	if strings.TrimSpace(titleKeyword) != "" {
		query = query.Where("title LIKE ?", "%"+strings.TrimSpace(titleKeyword)+"%")
	}
	if readStatus == "read" {
		query = query.Where("(receiver_user_id = ? AND is_read = ?) OR (receiver_user_id = 0 AND EXISTS (SELECT 1 FROM user_message_reads umr WHERE umr.message_id = user_messages.id AND umr.user_id = ?))", userID, true, userID)
	} else if readStatus == "unread" {
		query = query.Where("(receiver_user_id = ? AND is_read = ?) OR (receiver_user_id = 0 AND NOT EXISTS (SELECT 1 FROM user_message_reads umr WHERE umr.message_id = user_messages.id AND umr.user_id = ?))", userID, false, userID)
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
	broadcastIDs := make([]int, 0)
	for _, item := range items {
		if item.ReceiverUserID == 0 {
			broadcastIDs = append(broadcastIDs, item.ID)
		}
	}
	if len(broadcastIDs) > 0 {
		var readRows []UserMessageRead
		if err := DB.Model(&UserMessageRead{}).
			Select("message_id").
			Where("user_id = ? AND message_id IN ?", userID, broadcastIDs).
			Find(&readRows).Error; err != nil {
			return nil, 0, err
		}
		readMap := make(map[int]bool, len(readRows))
		for _, row := range readRows {
			readMap[row.MessageID] = true
		}
		for _, item := range items {
			if item.ReceiverUserID == 0 {
				item.IsRead = readMap[item.ID]
			}
		}
	}
	return items, total, nil
}

// CountUnreadUserMessages 统计当前用户未读消息（支持广播消息按用户已读追踪）。
func CountUnreadUserMessages(userID int, role int) (int64, error) {
	var (
		directTotal    int64
		broadcastTotal int64
	)
	if err := DB.Model(&UserMessage{}).
		Where("receiver_user_id = ? AND is_read = ?", userID, false).
		Count(&directTotal).Error; err != nil {
		return 0, err
	}
	if err := DB.Model(&UserMessage{}).
		Where("receiver_user_id = 0 AND receiver_min_role > 0 AND receiver_min_role <= ? AND NOT EXISTS (SELECT 1 FROM user_message_reads umr WHERE umr.message_id = user_messages.id AND umr.user_id = ?)", role, userID).
		Count(&broadcastTotal).Error; err != nil {
		return 0, err
	}
	return directTotal + broadcastTotal, nil
}

// MarkUserMessageAsRead 将用户可见消息标记为已读。
// 定向消息更新 user_messages.is_read；广播消息写入 user_message_reads 用户已读记录。
func MarkUserMessageAsRead(messageID int, userID int, role int) (bool, error) {
	var msg UserMessage
	if err := DB.Where("id = ? AND (receiver_user_id = ? OR (receiver_user_id = 0 AND receiver_min_role > 0 AND receiver_min_role <= ?))", messageID, userID, role).
		First(&msg).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	now := time.Now().Unix()
	if msg.ReceiverUserID == userID {
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
	read := UserMessageRead{
		UserID:    userID,
		MessageID: messageID,
		ReadAt:    now,
		CreatedAt: now,
	}
	res := DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "message_id"}},
		DoUpdates: clause.Assignments(map[string]any{"read_at": now}),
	}).Create(&read)
	if res.Error != nil {
		return false, res.Error
	}
	return true, nil
}

// MarkAllUserMessagesAsRead 将当前用户可见消息全部标记为已读。
// 定向消息写回 user_messages；广播消息写入 user_message_reads。
func MarkAllUserMessagesAsRead(userID int, role int) (int64, error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	now := time.Now().Unix()
	res := tx.Model(&UserMessage{}).
		Where("receiver_user_id = ? AND is_read = ?", userID, false).
		Updates(map[string]any{
			"is_read": true,
			"read_at": now,
		})
	if res.Error != nil {
		tx.Rollback()
		return 0, res.Error
	}
	updatedCount := res.RowsAffected
	var broadcastIDs []int
	if err := tx.Model(&UserMessage{}).
		Select("id").
		Where("receiver_user_id = 0 AND receiver_min_role > 0 AND receiver_min_role <= ?", role).
		Find(&broadcastIDs).Error; err != nil {
		tx.Rollback()
		return 0, err
	}
	if len(broadcastIDs) > 0 {
		reads := make([]UserMessageRead, 0, len(broadcastIDs))
		for _, messageID := range broadcastIDs {
			reads = append(reads, UserMessageRead{
				UserID:    userID,
				MessageID: messageID,
				ReadAt:    now,
				CreatedAt: now,
			})
		}
		insertRes := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "message_id"}},
			DoUpdates: clause.Assignments(map[string]any{"read_at": now}),
		}).Create(&reads)
		if insertRes.Error != nil {
			tx.Rollback()
			return 0, insertRes.Error
		}
		updatedCount += insertRes.RowsAffected
	}
	if err := tx.Commit().Error; err != nil {
		return 0, err
	}
	return updatedCount, nil
}

// IsSupplierApplicationNotFound 判断是否未找到供应商申请记录。
func IsSupplierApplicationNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

// BackfillSupplierApplicationAlias 将 supplier_alias 统一为 P+id（迁移/修复用，可安全重复执行）。
func BackfillSupplierApplicationAlias() error {
	switch {
	case common.UsingMySQL:
		return DB.Exec("UPDATE supplier_applications SET supplier_alias = CONCAT('P', id) WHERE id > 0").Error
	case common.UsingPostgreSQL:
		return DB.Exec(`UPDATE supplier_applications SET supplier_alias = 'P' || id::text WHERE id > 0`).Error
	default:
		return DB.Exec("UPDATE supplier_applications SET supplier_alias = 'P' || id WHERE id > 0").Error
	}
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

// IsSupplierAliasDuplicateError 判断是否为供应商别名重复错误。
func IsSupplierAliasDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	lowerMsg := strings.ToLower(err.Error())
	if !strings.Contains(lowerMsg, "supplier_alias") {
		return false
	}
	// 兼容 MySQL / PostgreSQL / SQLite 常见唯一约束错误文案
	return strings.Contains(lowerMsg, "duplicate") ||
		strings.Contains(lowerMsg, "duplicated") ||
		strings.Contains(lowerMsg, "unique constraint") ||
		strings.Contains(lowerMsg, "unique failed") ||
		strings.Contains(lowerMsg, "idx_supplier_applications_supplier_alias")
}
