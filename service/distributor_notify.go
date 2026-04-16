package service

import (
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
)

func notifyDistributorUser(userId int, notifyType string, title string, content string) {
	if userId <= 0 {
		return
	}
	u, err := model.GetUserById(userId, false)
	if err != nil || u == nil {
		return
	}
	msg := &model.UserMessage{
		ReceiverUserID: userId,
		Type:           notifyType,
		Title:          title,
		Content:        content,
		BizType:        "distributor",
	}
	if err := PublishUserMessage(msg); err != nil {
		common.SysLog("distributor notify (站内): " + err.Error())
	}
	if err := NotifyUser(userId, u.Email, u.GetSetting(), dto.NewNotify(notifyType, title, content, nil)); err != nil {
		common.SysLog(fmt.Sprintf("distributor notify (channel): user=%d %s", userId, err.Error()))
	}
}

// NotifyDistributorApplicationApproved 资料审核通过，已成为分销商。
func NotifyDistributorApplicationApproved(userId int) {
	notifyDistributorUser(userId, dto.NotifyTypeDistributorApplicationApproved, "分销商认证已通过",
		"您的分销商资料审核已通过，已开通分销中心相关功能。您可在个人中心查看邀请与收益。")
}

// NotifyDistributorApplicationRejected 资料审核被驳回。
func NotifyDistributorApplicationRejected(userId int, reason string) {
	content := "您的分销商入驻申请未通过审核。"
	if reason != "" {
		content += "原因：" + reason
	}
	notifyDistributorUser(userId, dto.NotifyTypeDistributorApplicationRejected, "分销商认证未通过", content)
}

// NotifyDistributorRoleGranted 管理员将账号设为分销商。
func NotifyDistributorRoleGranted(userId int) {
	notifyDistributorUser(userId, dto.NotifyTypeDistributorRoleGranted, "已设为分销商",
		"管理员已为您的账号开通分销商资格，可使用分销中心邀请与收益功能。")
}

// NotifyDistributorRoleRevoked 管理员取消分销商资格。
func NotifyDistributorRoleRevoked(userId int) {
	notifyDistributorUser(userId, dto.NotifyTypeDistributorRoleRevoked, "已取消分销商资格",
		"管理员已取消您的分销商资格，分销中心相关功能将不可用。")
}

// NotifyDistributorWithdrawalSubmitted 用户提交线下提现申请。
func NotifyDistributorWithdrawalSubmitted(userId int, quotaAmount int) {
	notifyDistributorUser(userId, dto.NotifyTypeDistributorWithdrawalSubmitted, "提现申请已提交",
		fmt.Sprintf("您已提交一笔线下提现申请，申请额度：%s，请等待审核。", logger.LogQuota(quotaAmount)))
}

// NotifyDistributorWithdrawalApproved 提现审核通过。
func NotifyDistributorWithdrawalApproved(userId int) {
	notifyDistributorUser(userId, dto.NotifyTypeDistributorWithdrawalApproved, "提现审核已通过",
		"您提交的线下提现申请已通过审核。")
}

// NotifyDistributorWithdrawalRejected 提现被驳回。
func NotifyDistributorWithdrawalRejected(userId int, reason string) {
	content := "您提交的线下提现申请未通过审核。"
	if reason != "" {
		content += "原因：" + reason
	}
	notifyDistributorUser(userId, dto.NotifyTypeDistributorWithdrawalRejected, "提现审核未通过", content)
}

// NotifyUserDemotedFromAdmin 管理员将用户从管理员降为普通用户。
func NotifyUserDemotedFromAdmin(userId int) {
	notifyDistributorUser(userId, dto.NotifyTypeUserDemotedFromAdmin, "账号身份已调整",
		"您的账号已由管理员调整为普通用户。")
}
