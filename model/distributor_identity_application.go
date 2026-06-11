package model

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

// DistributorIdentityApplication stores a distributor's request to switch the
// effective settlement identity between personal and enterprise.
type DistributorIdentityApplication struct {
	Id                int    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId            int    `json:"user_id" gorm:"not null;index:idx_dist_identity_app_user"`
	SourceApplyType   int    `json:"source_apply_type" gorm:"type:int;not null;default:0;column:source_apply_type"`
	SourceRealName    string `json:"source_real_name" gorm:"type:varchar(64);column:source_real_name"`
	TargetApplyType   int    `json:"target_apply_type" gorm:"type:int;not null;column:target_apply_type"`
	RealName          string `json:"real_name" gorm:"type:varchar(64);not null;column:real_name"`
	IdCardNo          string `json:"id_card_no" gorm:"type:varchar(32);not null;column:id_card_no"`
	QualificationUrls string `json:"qualification_urls" gorm:"type:text;not null;column:qualification_urls"`
	Contact           string `json:"contact" gorm:"type:varchar(128);not null;column:contact"`
	Status            int    `json:"status" gorm:"type:int;not null;default:1;index:idx_dist_identity_app_status"`
	RejectReason      string `json:"reject_reason" gorm:"type:varchar(512);column:reject_reason"`
	ReviewerId        int    `json:"reviewer_id" gorm:"column:reviewer_id"`
	ReviewedAt        int64  `json:"reviewed_at" gorm:"column:reviewed_at"`
	CreatedAt         int64  `json:"created_at" gorm:"autoCreateTime;bigint"`
	UpdatedAt         int64  `json:"updated_at" gorm:"autoUpdateTime;bigint"`
}

func (DistributorIdentityApplication) TableName() string {
	return "distributor_identity_applications"
}

func oppositeDistributorApplyType(applyType int) int {
	if applyType == DistributorApplyTypeEnterprise {
		return DistributorApplyTypePersonal
	}
	return DistributorApplyTypeEnterprise
}

type DistributorIdentityApplicationListItem struct {
	Application      DistributorIdentityApplication
	Username         string
	CurrentApplyType int
	CurrentRealName  string
}

type DistributorIdentityApplicationListQuery struct {
	Keyword         string
	Status          int
	TargetApplyType int
	PageInfo        *common.PageInfo
}

func validateDistributorIdentityApplicationInput(applyType int, realName, idCardNo, qualificationUrlsJSON, contact string) (string, string, string, string, error) {
	if applyType != DistributorApplyTypePersonal && applyType != DistributorApplyTypeEnterprise {
		return "", "", "", "", errors.New("申请身份无效")
	}
	realName = strings.TrimSpace(realName)
	idCardNo = strings.TrimSpace(idCardNo)
	contact = strings.TrimSpace(contact)
	qualJSON, err := NormalizeDistributorQualificationURLsJSON(qualificationUrlsJSON)
	if err != nil {
		return "", "", "", "", err
	}
	if !distributorQualificationURLsNonEmpty(qualJSON) {
		return "", "", "", "", errors.New("请上传资格证书")
	}
	if realName == "" || idCardNo == "" || contact == "" {
		return "", "", "", "", errors.New("请填写完整资料")
	}
	return realName, idCardNo, contact, qualJSON, nil
}

// SubmitDistributorIdentityApplication creates a pending identity switch request.
// Pending review never changes the currently effective distributor profile.
func SubmitDistributorIdentityApplication(userId, targetApplyType int, realName, idCardNo, qualificationUrlsJSON, contact string) error {
	if userId <= 0 {
		return errors.New("invalid user")
	}
	realName, idCardNo, contact, qualJSON, err := validateDistributorIdentityApplicationInput(targetApplyType, realName, idCardNo, qualificationUrlsJSON, contact)
	if err != nil {
		return err
	}
	u, err := GetUserById(userId, false)
	if err != nil {
		return err
	}
	if !UserIsDistributor(u) {
		return errors.New("仅代理用户可申请变更身份")
	}
	if u.Role >= common.RoleAdminUser {
		return errors.New("管理员无需申请")
	}
	currentApplyType, currentRealName, err := GetDistributorWithdrawAccountType(userId)
	if err != nil {
		return err
	}
	if currentApplyType == targetApplyType {
		return errors.New("申请身份与当前身份一致")
	}
	var pending int64
	if err := DB.Model(&DistributorIdentityApplication{}).
		Where("user_id = ? AND status = ?", userId, DistributorAppStatusPending).
		Count(&pending).Error; err != nil {
		return err
	}
	if pending > 0 {
		return errors.New("已有身份变更申请正在审核中")
	}
	ts := common.GetTimestamp()
	app := DistributorIdentityApplication{
		UserId:            userId,
		SourceApplyType:   currentApplyType,
		SourceRealName:    currentRealName,
		TargetApplyType:   targetApplyType,
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

func GetLatestDistributorIdentityApplicationByUserId(userId int) (*DistributorIdentityApplication, error) {
	if userId <= 0 {
		return nil, errors.New("invalid user")
	}
	var apps []DistributorIdentityApplication
	if err := DB.Where("user_id = ?", userId).Order("id desc").Limit(1).Find(&apps).Error; err != nil {
		return nil, err
	}
	if len(apps) == 0 {
		return nil, nil
	}
	return &apps[0], nil
}

func ListDistributorIdentityApplicationsAdmin(q DistributorIdentityApplicationListQuery) ([]DistributorIdentityApplicationListItem, int64, error) {
	tx := DB.Model(&DistributorIdentityApplication{})
	if q.Status > 0 {
		tx = tx.Where("status = ?", q.Status)
	}
	if q.TargetApplyType == DistributorApplyTypePersonal || q.TargetApplyType == DistributorApplyTypeEnterprise {
		tx = tx.Where("target_apply_type = ?", q.TargetApplyType)
	}
	kw := strings.TrimSpace(q.Keyword)
	if kw != "" {
		pattern := "%" + kw + "%"
		var userIds []int
		if err := DB.Model(&User{}).Where("username LIKE ?", pattern).Pluck("id", &userIds).Error; err != nil {
			return nil, 0, err
		}
		if len(userIds) > 0 {
			tx = tx.Where("real_name LIKE ? OR contact LIKE ? OR id_card_no LIKE ? OR user_id IN ?", pattern, pattern, pattern, userIds)
		} else {
			tx = tx.Where("real_name LIKE ? OR contact LIKE ? OR id_card_no LIKE ?", pattern, pattern, pattern)
		}
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	pi := q.PageInfo
	if pi == nil {
		pi = &common.PageInfo{}
	}
	var rows []DistributorIdentityApplication
	if err := tx.Order("id desc").Limit(pi.GetPageSize()).Offset(pi.GetStartIdx()).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	out := make([]DistributorIdentityApplicationListItem, 0, len(rows))
	for i := range rows {
		var username string
		var u User
		if err := DB.Select("username").Where("id = ?", rows[i].UserId).First(&u).Error; err == nil {
			username = u.Username
		}
		sourceStored := rows[i].SourceApplyType > 0
		currentApplyType, currentRealName := rows[i].SourceApplyType, strings.TrimSpace(rows[i].SourceRealName)
		if currentApplyType == 0 {
			currentApplyType = oppositeDistributorApplyType(rows[i].TargetApplyType)
		}
		if !sourceStored && currentRealName == "" {
			_, currentRealName, _ = GetDistributorWithdrawAccountType(rows[i].UserId)
		}
		out = append(out, DistributorIdentityApplicationListItem{
			Application:      rows[i],
			Username:         username,
			CurrentApplyType: currentApplyType,
			CurrentRealName:  currentRealName,
		})
	}
	return out, total, nil
}

func GetDistributorIdentityApplicationByIdAdmin(id int) (*DistributorIdentityApplicationListItem, error) {
	if id <= 0 {
		return nil, errors.New("invalid id")
	}
	var app DistributorIdentityApplication
	if err := DB.Where("id = ?", id).First(&app).Error; err != nil {
		return nil, err
	}
	var username string
	var u User
	if err := DB.Select("username").Where("id = ?", app.UserId).First(&u).Error; err == nil {
		username = u.Username
	}
	sourceStored := app.SourceApplyType > 0
	currentApplyType, currentRealName := app.SourceApplyType, strings.TrimSpace(app.SourceRealName)
	if currentApplyType == 0 {
		currentApplyType = oppositeDistributorApplyType(app.TargetApplyType)
	}
	if !sourceStored && currentRealName == "" {
		_, currentRealName, _ = GetDistributorWithdrawAccountType(app.UserId)
	}
	return &DistributorIdentityApplicationListItem{
		Application:      app,
		Username:         username,
		CurrentApplyType: currentApplyType,
		CurrentRealName:  currentRealName,
	}, nil
}

func ApproveDistributorIdentityApplication(appId, reviewerId int) error {
	if appId <= 0 || reviewerId <= 0 {
		return errors.New("invalid params")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var app DistributorIdentityApplication
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
		if !UserIsDistributor(&u) {
			return errors.New("用户不是代理")
		}
		if u.Role >= common.RoleAdminUser {
			return errors.New("管理员无需维护代理身份")
		}
		ts := common.GetTimestamp()
		var profile DistributorApplication
		err := tx.Where("user_id = ?", app.UserId).First(&profile).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			profile = DistributorApplication{
				UserId:            app.UserId,
				ApplyType:         app.TargetApplyType,
				RealName:          app.RealName,
				IdCardNo:          app.IdCardNo,
				QualificationUrls: app.QualificationUrls,
				Contact:           app.Contact,
				Status:            DistributorAppStatusApproved,
				ReviewerId:        reviewerId,
				ReviewedAt:        ts,
				CreatedAt:         ts,
				UpdatedAt:         ts,
			}
			if err := tx.Create(&profile).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			profile.ApplyType = app.TargetApplyType
			profile.RealName = app.RealName
			profile.IdCardNo = app.IdCardNo
			profile.QualificationUrls = app.QualificationUrls
			profile.Contact = app.Contact
			profile.Status = DistributorAppStatusApproved
			profile.RejectReason = ""
			profile.ReviewerId = reviewerId
			profile.ReviewedAt = ts
			profile.UpdatedAt = ts
			if err := tx.Save(&profile).Error; err != nil {
				return err
			}
		}
		app.Status = DistributorAppStatusApproved
		app.ReviewerId = reviewerId
		app.ReviewedAt = ts
		app.RejectReason = ""
		app.UpdatedAt = ts
		return tx.Save(&app).Error
	})
}

func RejectDistributorIdentityApplication(appId, reviewerId int, reason string) error {
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
	var app DistributorIdentityApplication
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
	app.UpdatedAt = ts
	return DB.Save(&app).Error
}

func BackfillDistributorIdentityApplicationSourcesIfNeeded() error {
	if DB == nil {
		return nil
	}
	var count int64
	if err := DB.Model(&DistributorIdentityApplication{}).
		Where("source_apply_type = ?", 0).
		Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return nil
	}
	var rows []DistributorIdentityApplication
	if err := DB.Order("user_id asc, id asc").Find(&rows).Error; err != nil {
		return err
	}
	type effectiveIdentity struct {
		applyType int
		realName  string
	}
	effectiveByUser := map[int]effectiveIdentity{}
	for i := range rows {
		row := rows[i]
		sourceType := row.SourceApplyType
		if sourceType == 0 {
			sourceType = oppositeDistributorApplyType(row.TargetApplyType)
		}
		sourceRealName := strings.TrimSpace(row.SourceRealName)
		if sourceRealName == "" {
			if cur, ok := effectiveByUser[row.UserId]; ok && cur.applyType == sourceType {
				sourceRealName = cur.realName
			}
		}
		if row.SourceApplyType == 0 || row.SourceRealName != sourceRealName {
			if err := DB.Model(&DistributorIdentityApplication{}).
				Where("id = ?", row.Id).
				Updates(map[string]interface{}{
					"source_apply_type": sourceType,
					"source_real_name":  sourceRealName,
				}).Error; err != nil {
				return err
			}
		}
		if row.Status == DistributorAppStatusApproved {
			effectiveByUser[row.UserId] = effectiveIdentity{
				applyType: row.TargetApplyType,
				realName:  strings.TrimSpace(row.RealName),
			}
		}
	}
	return nil
}
