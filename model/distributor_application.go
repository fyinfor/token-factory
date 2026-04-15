package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

// 分销商申请状态：1=待审核 2=已通过 3=已驳回
const (
	DistributorAppStatusPending  = 1
	DistributorAppStatusApproved = 2
	DistributorAppStatusRejected  = 3
)

// DistributorApplication 分销商入驻申请（每个用户最多一条记录，驳回后可更新重新提交）
type DistributorApplication struct {
	Id                 int    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId             int    `json:"user_id" gorm:"not null;uniqueIndex:idx_dist_app_user"`
	RealName           string `json:"real_name" gorm:"type:varchar(64);not null;column:real_name"`
	IdCardNo           string `json:"id_card_no" gorm:"type:varchar(32);not null;column:id_card_no"`
	QualificationUrls  string `json:"qualification_urls" gorm:"type:text;not null;column:qualification_urls"` // JSON 数组 URL 字符串
	Contact            string `json:"contact" gorm:"type:varchar(128);not null;column:contact"`
	Status             int    `json:"status" gorm:"type:int;not null;default:1;index:idx_dist_app_status"`
	RejectReason       string `json:"reject_reason" gorm:"type:varchar(512);column:reject_reason"`
	ReviewerId         int    `json:"reviewer_id" gorm:"column:reviewer_id"`
	ReviewedAt         int64  `json:"reviewed_at" gorm:"column:reviewed_at"`
	CreatedAt          int64  `json:"created_at" gorm:"autoCreateTime;bigint"`
	UpdatedAt          int64  `json:"updated_at" gorm:"autoUpdateTime;bigint"`
}

func (DistributorApplication) TableName() string {
	return "distributor_applications"
}

// UpsertDistributorApplication 用户提交或驳回后重新提交
func UpsertDistributorApplication(userId int, realName, idCardNo, qualificationUrlsJSON, contact string) error {
	if userId <= 0 {
		return errors.New("invalid user")
	}
	realName = strings.TrimSpace(realName)
	idCardNo = strings.TrimSpace(idCardNo)
	contact = strings.TrimSpace(contact)
	qualificationUrlsJSON = strings.TrimSpace(qualificationUrlsJSON)
	if realName == "" || idCardNo == "" || qualificationUrlsJSON == "" || contact == "" {
		return errors.New("请填写完整资料")
	}
	u, err := GetUserById(userId, false)
	if err != nil {
		return err
	}
	if u.Role == common.RoleDistributorUser {
		return errors.New("您已是分销商")
	}
	if u.Role >= common.RoleAdminUser {
		return errors.New("管理员无需申请")
	}
	var app DistributorApplication
	err = DB.Where("user_id = ?", userId).First(&app).Error
	ts := common.GetTimestamp()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		app = DistributorApplication{
			UserId:            userId,
			RealName:          realName,
			IdCardNo:          idCardNo,
			QualificationUrls: qualificationUrlsJSON,
			Contact:           contact,
			Status:            DistributorAppStatusPending,
			CreatedAt:         ts,
			UpdatedAt:         ts,
		}
		return DB.Create(&app).Error
	}
	if err != nil {
		return err
	}
	if app.Status == DistributorAppStatusPending {
		return errors.New("申请正在审核中，请耐心等待")
	}
	if app.Status == DistributorAppStatusApproved {
		// 记录仍为「已通过」，但账号已被降级为普通用户时需允许再次提交（与驳回后重提相同，重置为待审核）
		if u.Role == common.RoleDistributorUser {
			return errors.New("申请已通过")
		}
		app.RealName = realName
		app.IdCardNo = idCardNo
		app.QualificationUrls = qualificationUrlsJSON
		app.Contact = contact
		app.Status = DistributorAppStatusPending
		app.RejectReason = ""
		app.ReviewerId = 0
		app.ReviewedAt = 0
		app.UpdatedAt = ts
		return DB.Save(&app).Error
	}
	// rejected -> resubmit
	app.RealName = realName
	app.IdCardNo = idCardNo
	app.QualificationUrls = qualificationUrlsJSON
	app.Contact = contact
	app.Status = DistributorAppStatusPending
	app.RejectReason = ""
	app.ReviewerId = 0
	app.ReviewedAt = 0
	app.UpdatedAt = ts
	return DB.Save(&app).Error
}

// GetDistributorApplicationByUserId 当前用户申请记录
func GetDistributorApplicationByUserId(userId int) (*DistributorApplication, error) {
	if userId <= 0 {
		return nil, errors.New("invalid user")
	}
	var app DistributorApplication
	err := DB.Where("user_id = ?", userId).First(&app).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &app, err
}

// DistributorApplicationListQuery 管理端筛选
type DistributorApplicationListQuery struct {
	Keyword   string
	Status    int // 0 = 全部
	DateFrom  int64
	DateTo    int64
	PageInfo  *common.PageInfo
}

// ListDistributorApplicationsAdmin 分页列表（keyword 匹配姓名、用户名、联系方式）
func ListDistributorApplicationsAdmin(q DistributorApplicationListQuery) ([]DistributorApplication, []string, int64, error) {
	tx := DB.Model(&DistributorApplication{}).Joins("LEFT JOIN users ON users.id = distributor_applications.user_id")
	if q.Status > 0 {
		tx = tx.Where("distributor_applications.status = ?", q.Status)
	}
	if q.DateFrom > 0 {
		tx = tx.Where("distributor_applications.created_at >= ?", q.DateFrom)
	}
	if q.DateTo > 0 {
		tx = tx.Where("distributor_applications.created_at <= ?", q.DateTo)
	}
	kw := strings.TrimSpace(q.Keyword)
	if kw != "" {
		pattern := "%" + kw + "%"
		tx = tx.Where(
			"(distributor_applications.real_name LIKE ? OR distributor_applications.contact LIKE ? OR users.username LIKE ?)",
			pattern, pattern, pattern,
		)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, nil, 0, err
	}
	var rows []DistributorApplication
	pi := q.PageInfo
	if pi == nil {
		pi = &common.PageInfo{}
	}
	err := tx.Select("distributor_applications.*").
		Order("distributor_applications.id desc").
		Limit(pi.GetPageSize()).
		Offset(pi.GetStartIdx()).
		Find(&rows).Error
	if err != nil {
		return nil, nil, 0, err
	}
	usernames := make([]string, len(rows))
	for i := range rows {
		var u User
		if e := DB.Select("username").Where("id = ?", rows[i].UserId).First(&u).Error; e == nil {
			usernames[i] = u.Username
		}
	}
	return rows, usernames, total, nil
}

// GetDistributorApplicationByIdAdmin 详情
func GetDistributorApplicationByIdAdmin(id int) (*DistributorApplication, string, error) {
	if id <= 0 {
		return nil, "", errors.New("invalid id")
	}
	var app DistributorApplication
	if err := DB.Where("id = ?", id).First(&app).Error; err != nil {
		return nil, "", err
	}
	var u User
	_ = DB.Select("username").Where("id = ?", app.UserId).First(&u).Error
	return &app, u.Username, nil
}

// ApproveDistributorApplication 通过：用户角色改为分销商，申请状态已通过
func ApproveDistributorApplication(appId, reviewerId int) error {
	if appId <= 0 || reviewerId <= 0 {
		return errors.New("invalid params")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var app DistributorApplication
		if err := tx.Where("id = ?", appId).First(&app).Error; err != nil {
			return err
		}
		if app.Status != DistributorAppStatusPending {
			return errors.New("申请状态不是待审核")
		}
		var u User
		if err := tx.Where("id = ?", app.UserId).First(&u).Error; err != nil {
			return err
		}
		if u.Role >= common.RoleAdminUser {
			return errors.New("不能将管理员设为分销商")
		}
		if u.Role == common.RoleDistributorUser {
			return errors.New("用户已是分销商")
		}
		ts := common.GetTimestamp()
		app.Status = DistributorAppStatusApproved
		app.ReviewerId = reviewerId
		app.ReviewedAt = ts
		app.RejectReason = ""
		if err := tx.Save(&app).Error; err != nil {
			return err
		}
		err := tx.Model(&User{}).Where("id = ?", app.UserId).Update("role", common.RoleDistributorUser).Error
		if err != nil {
			return err
		}
		return nil
	})
}

// RejectDistributorApplication 驳回
func RejectDistributorApplication(appId, reviewerId int, reason string) error {
	reason = strings.TrimSpace(reason)
	if appId <= 0 || reviewerId <= 0 {
		return errors.New("invalid params")
	}
	if reason == "" {
		return errors.New("请填写驳回原因")
	}
	if len(reason) > 500 {
		return errors.New("驳回原因过长")
	}
	var app DistributorApplication
	if err := DB.Where("id = ?", appId).First(&app).Error; err != nil {
		return err
	}
	if app.Status != DistributorAppStatusPending {
		return errors.New("申请状态不是待审核")
	}
	ts := common.GetTimestamp()
	app.Status = DistributorAppStatusRejected
	app.RejectReason = reason
	app.ReviewerId = reviewerId
	app.ReviewedAt = ts
	return DB.Save(&app).Error
}

// ListDistributorsAdmin 分销商用户列表
func ListDistributorsAdmin(keyword string, pageInfo *common.PageInfo) ([]User, int64, error) {
	tx := DB.Model(&User{}).Where("role = ?", common.RoleDistributorUser)
	kw := strings.TrimSpace(keyword)
	if kw != "" {
		pattern := "%" + kw + "%"
		tx = tx.Where("(username LIKE ? OR display_name LIKE ?)", pattern, pattern)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var users []User
	err := tx.Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&users).Error
	return users, total, err
}

// SetUserDistributorCommissionBps 管理员设置单个分销商默认分成比例（万分之一）
func SetUserDistributorCommissionBps(userId, bps int) error {
	if userId <= 0 {
		return errors.New("invalid user")
	}
	if bps < 0 || bps > 10000 {
		return fmt.Errorf("commission bps must be 0..10000")
	}
	var u User
	if err := DB.Where("id = ?", userId).First(&u).Error; err != nil {
		return err
	}
	if u.Role != common.RoleDistributorUser {
		return errors.New("用户不是分销商")
	}
	return DB.Model(&User{}).Where("id = ?", userId).Update("distributor_commission_bps", bps).Error
}

// AdminSettleDistributorAffQuota 结账：清空待结算分销收益额度 aff_quota
func AdminSettleDistributorAffQuota(userId int) error {
	if userId <= 0 {
		return errors.New("invalid user")
	}
	var u User
	if err := DB.Where("id = ?", userId).First(&u).Error; err != nil {
		return err
	}
	if u.Role != common.RoleDistributorUser {
		return errors.New("用户不是分销商")
	}
	return DB.Model(&User{}).Where("id = ?", userId).Update("aff_quota", 0).Error
}
