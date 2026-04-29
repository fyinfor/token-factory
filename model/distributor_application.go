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
	DistributorAppStatusPending    = 1
	DistributorAppStatusApproved   = 2
	DistributorAppStatusRejected   = 3
	DistributorApplyTypePersonal   = 1 // 个人申请：real_name=姓名，id_card_no=身份证
	DistributorApplyTypeEnterprise = 2 // 企业申请：real_name=企业名称，id_card_no=统一社会信用代码
)

// DistributorApplication 分销商入驻申请（每个用户最多一条记录，驳回后可更新重新提交）
type DistributorApplication struct {
	Id                 int    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId             int    `json:"user_id" gorm:"not null;uniqueIndex:idx_dist_app_user"`
	ApplyType          int    `json:"apply_type" gorm:"type:int;not null;default:1;column:apply_type"` // 1=个人 2=企业
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

func distributorQualificationURLsNonEmpty(jsonStr string) bool {
	raw := strings.TrimSpace(jsonStr)
	if raw == "" {
		return false
	}
	var urls []string
	if common.UnmarshalJsonStr(raw, &urls) != nil {
		return false
	}
	for _, u := range urls {
		if strings.TrimSpace(u) != "" {
			return true
		}
	}
	return false
}

// NormalizeDistributorQualificationURLsJSON 解析并规范化资格证书 JSON 数组字符串
func NormalizeDistributorQualificationURLsJSON(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "[]", nil
	}
	var urls []string
	if err := common.UnmarshalJsonStr(raw, &urls); err != nil {
		return "", errors.New("资格证书格式无效")
	}
	out := make([]string, 0, len(urls))
	for _, u := range urls {
		u = strings.TrimSpace(u)
		if u != "" {
			out = append(out, u)
		}
	}
	b, err := common.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// UpsertDistributorApplication 用户提交或驳回后重新提交
func UpsertDistributorApplication(userId, applyType int, realName, idCardNo, qualificationUrlsJSON, contact string) error {
	if userId <= 0 {
		return errors.New("invalid user")
	}
	if applyType != DistributorApplyTypePersonal && applyType != DistributorApplyTypeEnterprise {
		return errors.New("申请类型无效")
	}
	realName = strings.TrimSpace(realName)
	idCardNo = strings.TrimSpace(idCardNo)
	contact = strings.TrimSpace(contact)
	qualJSON, err := NormalizeDistributorQualificationURLsJSON(qualificationUrlsJSON)
	if err != nil {
		return err
	}
	if !distributorQualificationURLsNonEmpty(qualJSON) {
		return errors.New("请上传资格证书")
	}
	if realName == "" || idCardNo == "" || contact == "" {
		return errors.New("请填写完整资料")
	}
	u, err := GetUserById(userId, false)
	if err != nil {
		return err
	}
	if UserIsDistributor(u) {
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
			ApplyType:         applyType,
			RealName:          realName,
			IdCardNo:          idCardNo,
			QualificationUrls: qualJSON,
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
		// 记录仍为「已通过」，但账号已被取消分销商资格时需允许再次提交（与驳回后重提相同，重置为待审核）
		if UserIsDistributor(u) {
			return errors.New("申请已通过")
		}
		app.ApplyType = applyType
		app.RealName = realName
		app.IdCardNo = idCardNo
		app.QualificationUrls = qualJSON
		app.Contact = contact
		app.Status = DistributorAppStatusPending
		app.RejectReason = ""
		app.ReviewerId = 0
		app.ReviewedAt = 0
		app.UpdatedAt = ts
		return DB.Save(&app).Error
	}
	// rejected -> resubmit
	app.ApplyType = applyType
	app.RealName = realName
	app.IdCardNo = idCardNo
	app.QualificationUrls = qualJSON
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
	ApplyType int // 0 = 全部 1=个人 2=企业
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
	if q.ApplyType == DistributorApplyTypePersonal || q.ApplyType == DistributorApplyTypeEnterprise {
		tx = tx.Where("distributor_applications.apply_type = ?", q.ApplyType)
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
			"(distributor_applications.real_name LIKE ? OR distributor_applications.contact LIKE ? OR distributor_applications.id_card_no LIKE ? OR users.username LIKE ?)",
			pattern, pattern, pattern, pattern,
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

// ApproveDistributorApplication 通过：用户角色改为分销商，申请状态已通过。
// distributorCommissionBps 非 nil 时写入该用户的 distributor_commission_bps（0～10000，万分之一；0 表示跟随系统默认）；nil 表示不修改该字段（兼容无请求体调用）。
func ApproveDistributorApplication(appId, reviewerId int, distributorCommissionBps *int) error {
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
		if UserIsDistributor(&u) {
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
		if err := tx.Model(&User{}).Where("id = ?", app.UserId).Update("is_distributor", common.DistributorFlagYes).Error; err != nil {
			return err
		}
		if distributorCommissionBps != nil {
			b := *distributorCommissionBps
			if b < 0 || b > 10000 {
				return fmt.Errorf("commission bps must be 0..10000")
			}
			if err := tx.Model(&User{}).Where("id = ?", app.UserId).Update("distributor_commission_bps", b).Error; err != nil {
				return err
			}
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

// DistributorAdminListItem 管理端分销商列表行（含申请真实姓名、是否需补录资料）
type DistributorAdminListItem struct {
	User
	ApplicationRealName  string `json:"application_real_name"`
	ApplicationApplyType int    `json:"application_apply_type"` // 无申请记录时为 0
	NeedsSupplement      bool   `json:"needs_supplement"`
}

// DistributorListAdminQuery 管理端代理人员列表筛选
type DistributorListAdminQuery struct {
	Keyword   string
	ApplyType int // 0=全部 1=个人 2=企业
	PageInfo  *common.PageInfo
}

// ListDistributorsAdmin 分销商用户列表（LEFT JOIN 申请资料；关键字可搜用户名、显示名、申请姓名、联系方式、身份证）
func ListDistributorsAdmin(param DistributorListAdminQuery) ([]DistributorAdminListItem, int64, error) {
	tx := DB.Table("users").
		Joins("LEFT JOIN distributor_applications ON distributor_applications.user_id = users.id").
		Where("users.is_distributor = ? AND users.role < ?", common.DistributorFlagYes, common.RoleAdminUser)
	if param.ApplyType == DistributorApplyTypePersonal || param.ApplyType == DistributorApplyTypeEnterprise {
		tx = tx.Where("distributor_applications.apply_type = ?", param.ApplyType)
	}
	kw := strings.TrimSpace(param.Keyword)
	if kw != "" {
		pattern := "%" + kw + "%"
		tx = tx.Where(
			"(users.username LIKE ? OR users.display_name LIKE ? OR distributor_applications.real_name LIKE ? OR distributor_applications.contact LIKE ? OR distributor_applications.id_card_no LIKE ?)",
			pattern, pattern, pattern, pattern, pattern,
		)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	pageInfo := param.PageInfo
	if pageInfo == nil {
		pageInfo = &common.PageInfo{}
	}
	type distAdminScan struct {
		User
		AppRealName  string `gorm:"column:app_rn"`
		AppIdCard    string `gorm:"column:app_ic"`
		AppContact   string `gorm:"column:app_ct"`
		AppQual      string `gorm:"column:app_ql"`
		AppApplyType int    `gorm:"column:app_at"`
	}
	var scans []distAdminScan
	err := tx.Select(`users.*, distributor_applications.real_name AS app_rn, distributor_applications.id_card_no AS app_ic, distributor_applications.contact AS app_ct, distributor_applications.qualification_urls AS app_ql, COALESCE(distributor_applications.apply_type, 1) AS app_at`).
		Order("users.id DESC").
		Limit(pageInfo.GetPageSize()).
		Offset(pageInfo.GetStartIdx()).
		Scan(&scans).Error
	if err != nil {
		return nil, 0, err
	}
	out := make([]DistributorAdminListItem, 0, len(scans))
	for i := range scans {
		s := scans[i]
		fake := &DistributorApplication{
			ApplyType:         s.AppApplyType,
			RealName:          s.AppRealName,
			IdCardNo:          s.AppIdCard,
			Contact:           s.AppContact,
			QualificationUrls: s.AppQual,
		}
		rn := strings.TrimSpace(s.AppRealName)
		out = append(out, DistributorAdminListItem{
			User:                 s.User,
			ApplicationRealName:  rn,
			ApplicationApplyType: s.AppApplyType,
			NeedsSupplement:      !IsDistributorApplicationProfileComplete(fake),
		})
	}
	return out, total, nil
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
	if !UserIsDistributor(&u) {
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
	if !UserIsDistributor(&u) {
		return errors.New("用户不是分销商")
	}
	return DB.Model(&User{}).Where("id = ?", userId).Update("aff_quota", 0).Error
}

// IsDistributorApplicationProfileComplete 判断分销商申请资料是否已完整（用于手工开通后补录提示）
func IsDistributorApplicationProfileComplete(app *DistributorApplication) bool {
	if app == nil {
		return false
	}
	if strings.TrimSpace(app.RealName) == "" || strings.TrimSpace(app.IdCardNo) == "" || strings.TrimSpace(app.Contact) == "" {
		return false
	}
	return distributorQualificationURLsNonEmpty(app.QualificationUrls)
}

// GetDistributorApplicationProfileByUserIdAdmin 管理端：某分销商的申请资料；无记录或资料不全时 needsManualEntry 为 true
func GetDistributorApplicationProfileByUserIdAdmin(userId int) (username string, app *DistributorApplication, needsManualEntry bool, err error) {
	if userId <= 0 {
		return "", nil, false, errors.New("invalid id")
	}
	u, err := GetUserById(userId, false)
	if err != nil {
		return "", nil, false, err
	}
	if !UserIsDistributor(u) {
		return "", nil, false, errors.New("用户不是分销商")
	}
	app, err = GetDistributorApplicationByUserId(userId)
	if err != nil {
		return "", nil, false, err
	}
	needsManualEntry = !IsDistributorApplicationProfileComplete(app)
	return u.Username, app, needsManualEntry, nil
}

// AdminUpsertDistributorApplicationByUser 管理端补录/修改分销商申请资料（无记录时创建为已通过）
func AdminUpsertDistributorApplicationByUser(userId, reviewerId, applyType int, realName, idCardNo, qualificationUrlsJSON, contact string) error {
	if userId <= 0 || reviewerId <= 0 {
		return errors.New("invalid params")
	}
	if applyType != DistributorApplyTypePersonal && applyType != DistributorApplyTypeEnterprise {
		return errors.New("申请类型无效")
	}
	realName = strings.TrimSpace(realName)
	idCardNo = strings.TrimSpace(idCardNo)
	contact = strings.TrimSpace(contact)
	qualJSON, err := NormalizeDistributorQualificationURLsJSON(qualificationUrlsJSON)
	if err != nil {
		return err
	}
	if !distributorQualificationURLsNonEmpty(qualJSON) {
		return errors.New("请上传资格证书")
	}
	if realName == "" || idCardNo == "" || contact == "" {
		return errors.New("请填写完整资料")
	}
	u, err := GetUserById(userId, false)
	if err != nil {
		return err
	}
	if !UserIsDistributor(u) {
		return errors.New("用户不是分销商")
	}
	if u.Role >= common.RoleAdminUser {
		return errors.New("管理员账号无需维护申请资料")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var app DistributorApplication
		err := tx.Where("user_id = ?", userId).First(&app).Error
		ts := common.GetTimestamp()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			app = DistributorApplication{
				UserId:            userId,
				ApplyType:         applyType,
				RealName:          realName,
				IdCardNo:          idCardNo,
				QualificationUrls: qualJSON,
				Contact:           contact,
				Status:            DistributorAppStatusApproved,
				RejectReason:      "",
				ReviewerId:        reviewerId,
				ReviewedAt:        ts,
				CreatedAt:         ts,
				UpdatedAt:         ts,
			}
			return tx.Create(&app).Error
		}
		if err != nil {
			return err
		}
		app.ApplyType = applyType
		app.RealName = realName
		app.IdCardNo = idCardNo
		app.QualificationUrls = qualJSON
		app.Contact = contact
		app.RejectReason = ""
		app.ReviewerId = reviewerId
		app.ReviewedAt = ts
		if app.Status != DistributorAppStatusApproved {
			app.Status = DistributorAppStatusApproved
		}
		app.UpdatedAt = ts
		return tx.Save(&app).Error
	})
}

// migrateDropDistributorApplicationIsStudentColumn 删除 distributor_applications 表中已废弃的 is_student 列（模型已不再映射该字段）。
func migrateDropDistributorApplicationIsStudentColumn() error {
	if DB == nil {
		return nil
	}
	var stmt string
	switch {
	case common.UsingPostgreSQL:
		stmt = `ALTER TABLE distributor_applications DROP COLUMN IF EXISTS is_student`
	case common.UsingSQLite:
		stmt = `ALTER TABLE distributor_applications DROP COLUMN is_student`
	default:
		stmt = `ALTER TABLE distributor_applications DROP COLUMN is_student`
	}
	err := DB.Exec(stmt).Error
	if err == nil {
		common.SysLog("migrate: dropped distributor_applications.is_student")
		return nil
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "unknown column") ||
		strings.Contains(msg, "doesn't exist") ||
		strings.Contains(msg, "no such column") ||
		strings.Contains(msg, "check that column") ||
		strings.Contains(msg, "does not exist") {
		return nil
	}
	if common.UsingSQLite &&
		(strings.Contains(msg, "syntax error") || strings.Contains(msg, "near \"drop\"")) {
		common.SysLog("migrate: skip DROP is_student (SQLite may not support DROP COLUMN): " + err.Error())
		return nil
	}
	return err
}
