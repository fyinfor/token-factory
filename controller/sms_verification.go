package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// SendSMSVerification 发送注册短信验证码。
func SendSMSVerification(c *gin.Context) {
	if !common.RegisterEnabled {
		common.ApiErrorI18n(c, i18n.MsgUserRegisterClosed)
		return
	}
	if !common.SMSVerificationEnabled {
		common.ApiErrorI18n(c, i18n.MsgUserSMSNotEnabled)
		return
	}
	phone := common.NormalizePhone(c.Query("phone"))
	if !common.IsValidLoginPhone(phone) {
		common.ApiErrorI18n(c, i18n.MsgUserPhoneInvalid)
		return
	}
	if model.IsPhoneAlreadyTaken(phone) {
		common.ApiErrorI18n(c, i18n.MsgUserPhoneTaken)
		return
	}
	if common.IsSMSPhoneBlacklisted(phone) {
		common.ApiErrorI18n(c, i18n.MsgUserPhoneBlacklisted)
		return
	}
	if err := common.CheckSMSCanSend(phone); err != nil {
		common.ApiError(c, err)
		return
	}

	// 阿里云数字验证码模板要求 code 变量必须为纯数字。
	code := common.GenerateNumericVerificationCode(6)
	if err := service.SendAliyunSMSCode(phone, code); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := common.RecordSMSSend(phone); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := common.StoreSMSVerificationCode(phone, code); err != nil {
		common.ApiErrorI18n(c, i18n.MsgUserSMSCodeStoreFailed)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

// SendSMSBindVerification 向待绑定手机号发送短信验证码（须已登录；手机号不可被其他用户占用）。
func SendSMSBindVerification(c *gin.Context) {
	if !common.SMSVerificationEnabled {
		common.ApiErrorI18n(c, i18n.MsgUserSMSNotEnabled)
		return
	}
	userID := c.GetInt("id")
	if userID <= 0 {
		common.ApiErrorI18n(c, i18n.MsgUnauthorized)
		return
	}
	phone := common.NormalizePhone(c.Query("phone"))
	if !common.IsValidLoginPhone(phone) {
		common.ApiErrorI18n(c, i18n.MsgUserPhoneInvalid)
		return
	}
	if model.IsPhoneTakenByOtherUser(phone, userID) {
		common.ApiErrorI18n(c, i18n.MsgUserPhoneTaken)
		return
	}
	if common.IsSMSPhoneBlacklisted(phone) {
		common.ApiErrorI18n(c, i18n.MsgUserPhoneBlacklisted)
		return
	}
	if err := common.CheckSMSCanSend(phone); err != nil {
		common.ApiError(c, err)
		return
	}

	code := common.GenerateNumericVerificationCode(6)
	if err := service.SendAliyunSMSCode(phone, code); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := common.RecordSMSSend(phone); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := common.StoreSMSVerificationCode(phone, code); err != nil {
		common.ApiErrorI18n(c, i18n.MsgUserSMSCodeStoreFailed)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}
