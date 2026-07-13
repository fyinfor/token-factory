package model

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	DistributorBindRequestStatusPending  = 1
	DistributorBindRequestStatusAccepted = 2
	DistributorBindRequestStatusRejected = 3
)

const (
	UserMessageBizTypeDistributorBindRequest = "distributor_bind_request"
	UserMessageTypeDistributorBindRequest    = "distributor_bind_request"
	UserMessageTypeDistributorBindResult     = "distributor_bind_result"
)

// DistributorBindRequest 代理主动绑定普通用户的确认请求。
type DistributorBindRequest struct {
	ID                int   `json:"id" gorm:"primaryKey;comment:主键ID"`
	DistributorUserID int   `json:"distributor_user_id" gorm:"type:int;index;not null;comment:发起绑定的代理用户ID"`
	TargetUserID      int   `json:"target_user_id" gorm:"type:int;index;not null;comment:被请求绑定的用户ID"`
	Status            int   `json:"status" gorm:"type:int;index;not null;default:1;comment:状态 1待处理 2已接受 3已拒绝"`
	MessageID         int   `json:"message_id" gorm:"type:int;index;default:0;comment:绑定请求站内消息ID"`
	CreatedAt         int64 `json:"created_at" gorm:"type:bigint;index;comment:创建时间戳"`
	UpdatedAt         int64 `json:"updated_at" gorm:"type:bigint;comment:更新时间戳"`
	RespondedAt       int64 `json:"responded_at" gorm:"type:bigint;default:0;comment:响应时间戳"`
}

func (DistributorBindRequest) TableName() string {
	return "distributor_bind_requests"
}

type DistributorBindableUser struct {
	UserID      int    `json:"user_id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Phone       string `json:"phone,omitempty"`
	Status      string `json:"status"`
	RequestID   int    `json:"request_id,omitempty"`
}

func distributorBindUserLabel(u *User) string {
	if u == nil {
		return ""
	}
	if s := strings.TrimSpace(u.DisplayName); s != "" {
		return s
	}
	if s := strings.TrimSpace(u.Username); s != "" {
		return s
	}
	return fmt.Sprintf("ID:%d", u.Id)
}

func SearchDistributorBindableUser(distributorUserID int, keyword string) (*DistributorBindableUser, error) {
	kw := strings.TrimSpace(keyword)
	if distributorUserID <= 0 || kw == "" {
		return nil, errors.New("请输入用户名或手机号")
	}
	var user User
	phone := common.NormalizePhone(kw)
	query := DB.Model(&User{}).Omit("password")
	if phone != "" {
		query = query.Where("username = ? OR phone = ?", kw, phone)
	} else {
		query = query.Where("username = ?", kw)
	}
	if err := query.First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	item := &DistributorBindableUser{
		UserID:      user.Id,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Phone:       user.Phone,
		Status:      "bindable",
	}
	if user.Id == distributorUserID {
		item.Status = "not_allowed"
		return item, nil
	}
	if UserIsDistributor(&user) {
		item.Status = "distributor"
		return item, nil
	}
	if user.Role != common.RoleCommonUser {
		item.Status = "not_allowed"
		return item, nil
	}
	if user.InviterId > 0 {
		item.Status = "bound"
		return item, nil
	}
	var pending DistributorBindRequest
	err := DB.Where("distributor_user_id = ? AND target_user_id = ? AND status = ?",
		distributorUserID, user.Id, DistributorBindRequestStatusPending).First(&pending).Error
	if err == nil {
		item.Status = "pending"
		item.RequestID = pending.ID
		return item, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return item, nil
}

func CreateDistributorBindRequest(distributorUserID, targetUserID int) (*DistributorBindRequest, error) {
	if distributorUserID <= 0 || targetUserID <= 0 || distributorUserID == targetUserID {
		return nil, errors.New("参数错误")
	}
	distributor, err := GetUserById(distributorUserID, false)
	if err != nil || !UserIsDistributor(distributor) {
		return nil, errors.New("仅代理用户可发起绑定")
	}
	target, err := GetUserById(targetUserID, false)
	if err != nil {
		return nil, errors.New("用户不存在")
	}
	if UserIsDistributor(target) {
		return nil, errors.New("不允许多级代理")
	}
	if target.Role != common.RoleCommonUser {
		return nil, errors.New("只能绑定普通用户")
	}
	if target.InviterId > 0 {
		return nil, errors.New("该用户已绑定代理")
	}
	var pending DistributorBindRequest
	err = DB.Where("distributor_user_id = ? AND target_user_id = ? AND status = ?",
		distributorUserID, targetUserID, DistributorBindRequestStatusPending).First(&pending).Error
	if err == nil {
		return &pending, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	now := time.Now().Unix()
	req := &DistributorBindRequest{
		DistributorUserID: distributorUserID,
		TargetUserID:      targetUserID,
		Status:            DistributorBindRequestStatusPending,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(req).Error; err != nil {
			return err
		}
		title := "代理绑定确认"
		content := fmt.Sprintf("代理用户 %s 希望将您绑定为邀请用户。接受后，您的账号将出现在对方的邀请用户列表中。", distributorBindUserLabel(distributor))
		msg := &UserMessage{
			ReceiverUserID: targetUserID,
			Type:           UserMessageTypeDistributorBindRequest,
			Title:          title,
			Content:        content,
			BizType:        UserMessageBizTypeDistributorBindRequest,
			BizID:          req.ID,
			CreatedAt:      now,
		}
		if err := tx.Create(msg).Error; err != nil {
			return err
		}
		req.MessageID = msg.ID
		return tx.Model(req).Update("message_id", msg.ID).Error
	}); err != nil {
		return nil, err
	}
	return req, nil
}

func RespondDistributorBindRequest(requestID, targetUserID int, accept bool) (*DistributorBindRequest, error) {
	if requestID <= 0 || targetUserID <= 0 {
		return nil, errors.New("参数错误")
	}
	var req DistributorBindRequest
	var distributor User
	var target User
	now := time.Now().Unix()
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND target_user_id = ?", requestID, targetUserID).First(&req).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("绑定请求不存在")
			}
			return err
		}
		if req.Status != DistributorBindRequestStatusPending {
			return errors.New("该绑定请求已处理")
		}
		if err := tx.Where("id = ?", req.DistributorUserID).First(&distributor).Error; err != nil {
			return errors.New("代理用户不存在")
		}
		if err := tx.Where("id = ?", req.TargetUserID).First(&target).Error; err != nil {
			return errors.New("目标用户不存在")
		}
		if accept {
			if !UserIsDistributor(&distributor) {
				return errors.New("发起用户已不是代理")
			}
			if target.Role != common.RoleCommonUser || UserIsDistributor(&target) {
				return errors.New("该用户不能绑定为邀请用户")
			}
			if target.InviterId > 0 {
				return errors.New("该用户已绑定代理")
			}
			res := tx.Model(&User{}).Where("id = ? AND inviter_id = ?", target.Id, 0).Update("inviter_id", distributor.Id)
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return errors.New("该用户已绑定代理")
			}
			res = tx.Model(&DistributorBindRequest{}).Where("id = ? AND status = ?", req.ID, DistributorBindRequestStatusPending).Updates(map[string]any{
				"status":       DistributorBindRequestStatusAccepted,
				"updated_at":   now,
				"responded_at": now,
			})
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return errors.New("该绑定请求已处理")
			}
			req.Status = DistributorBindRequestStatusAccepted
			req.UpdatedAt = now
			req.RespondedAt = now
			if err := tx.Model(&User{}).Where("id = ?", distributor.Id).UpdateColumn("aff_count", gorm.Expr("aff_count + ?", 1)).Error; err != nil {
				return err
			}
			rel := AffInviteRelation{
				InviterId:               distributor.Id,
				InviteeUserId:           target.Id,
				CommissionRatioBps:      defaultCommissionBpsForNewInviteRelation(distributor.Id),
				CommissionEarnedQuota:   0,
				ProfitShareEarnedQuota:  0,
				ModelMarkupDiscountRate: defaultModelMarkupDiscountRateForNewInviteRelation(tx, distributor.Id),
				CreatedAt:               now,
				UpdatedAt:               now,
			}
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&rel).Error; err != nil {
				return err
			}
		} else {
			req.Status = DistributorBindRequestStatusRejected
			req.UpdatedAt = now
			req.RespondedAt = now
			res := tx.Model(&DistributorBindRequest{}).Where("id = ? AND status = ?", req.ID, DistributorBindRequestStatusPending).Updates(map[string]any{
				"status":       req.Status,
				"updated_at":   req.UpdatedAt,
				"responded_at": req.RespondedAt,
			})
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				return errors.New("该绑定请求已处理")
			}
		}
		if req.MessageID > 0 {
			_ = tx.Model(&UserMessage{}).Where("id = ? AND receiver_user_id = ?", req.MessageID, targetUserID).Updates(map[string]any{
				"is_read": true,
				"read_at": now,
			}).Error
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	resultTitle := "代理绑定结果"
	resultText := "已拒绝"
	if accept {
		resultText = "已接受"
	}
	content := fmt.Sprintf("用户 %s %s您的绑定请求。", distributorBindUserLabel(&target), resultText)
	_ = CreateUserMessage(&UserMessage{
		ReceiverUserID: req.DistributorUserID,
		Type:           UserMessageTypeDistributorBindResult,
		Title:          resultTitle,
		Content:        content,
		BizType:        UserMessageBizTypeDistributorBindRequest,
		BizID:          req.ID,
	})
	return &req, nil
}

func GetDistributorBindRequestForUser(requestID, userID int) (*DistributorBindRequest, error) {
	if requestID <= 0 || userID <= 0 {
		return nil, errors.New("参数错误")
	}
	var req DistributorBindRequest
	if err := DB.Where("id = ? AND target_user_id = ?", requestID, userID).First(&req).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &req, nil
}
