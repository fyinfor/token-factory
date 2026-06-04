package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// SendSMSVerification 发送注册短信验证码。
func SendSMSVerification(c *gin.Context) {
	if !common.RegisterEnabled {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "新用户注册已关闭",
		})
		return
	}
	if !common.SMSVerificationEnabled {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "短信验证码功能未启用",
		})
		return
	}
	phone := common.NormalizePhone(c.Query("phone"))
	if !common.IsValidLoginPhone(phone) {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "手机号格式无效，请输入 11 位中国大陆手机号或 +国码 国际号",
		})
		return
	}
	if model.IsPhoneAlreadyTaken(phone) {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "手机号已被占用",
		})
		return
	}
	if common.IsSMSPhoneBlacklisted(phone) {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "该手机号已被加入短信黑名单",
		})
		return
	}
	if err := common.CheckSMSCanSend(phone); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 阿里云数字验证码模板要求 code 变量必须为纯数字。
	code := common.GenerateNumericVerificationCode(6)
	if err := service.SendAliyunSMSCode(phone, code); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	if err := common.RecordSMSSend(phone); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	if err := common.StoreSMSVerificationCode(phone, code); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "短信验证码存储失败，请稍后重试",
		})
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
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "短信验证码功能未启用",
		})
		return
	}
	userID := c.GetInt("id")
	if userID <= 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "未登录或会话无效",
		})
		return
	}
	phone := common.NormalizePhone(c.Query("phone"))
	if !common.IsValidLoginPhone(phone) {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "手机号格式无效，请输入 11 位中国大陆手机号或 +国码 国际号",
		})
		return
	}
	if model.IsPhoneTakenByOtherUser(phone, userID) {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "手机号已被占用",
		})
		return
	}
	if common.IsSMSPhoneBlacklisted(phone) {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "该手机号已被加入短信黑名单",
		})
		return
	}
	if err := common.CheckSMSCanSend(phone); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	code := common.GenerateNumericVerificationCode(6)
	if err := service.SendAliyunSMSCode(phone, code); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	if err := common.RecordSMSSend(phone); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	if err := common.StoreSMSVerificationCode(phone, code); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "短信验证码存储失败，请稍后重试",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}
