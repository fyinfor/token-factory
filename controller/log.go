package controller

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

func includeRawBillingLogs(c *gin.Context) bool {
	billingView := strings.TrimSpace(strings.ToLower(c.Query("billing_view")))
	includeRaw := strings.TrimSpace(strings.ToLower(c.Query("include_raw_billing_logs")))
	return billingView == "raw" || includeRaw == "1" || includeRaw == "true" || includeRaw == "yes"
}

func GetAllLogs(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	logTypes := model.ParseLogTypesQuery(c.Query("type"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	username := c.Query("username")
	tokenName := c.Query("token_name")
	modelName := c.Query("model_name")
	channel, _ := strconv.Atoi(c.Query("channel"))
	group := c.Query("group")
	requestId := c.Query("request_id")
	logs, total, err := model.GetAllLogs(logTypes, startTimestamp, endTimestamp, modelName, username, tokenName, pageInfo.GetStartIdx(), pageInfo.GetPageSize(), channel, group, requestId, includeRawBillingLogs(c))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(logs)
	common.ApiSuccess(c, pageInfo)
	return
}

func GetUserLogs(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	userId := c.GetInt("id")
	logTypes := model.ParseLogTypesQuery(c.Query("type"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	tokenName := c.Query("token_name")
	modelName := c.Query("model_name")
	group := c.Query("group")
	requestId := c.Query("request_id")
	logs, total, err := model.GetUserLogs(userId, logTypes, startTimestamp, endTimestamp, modelName, tokenName, pageInfo.GetStartIdx(), pageInfo.GetPageSize(), group, requestId, includeRawBillingLogs(c))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(logs)
	common.ApiSuccess(c, pageInfo)
	return
}

// Deprecated: SearchAllLogs 已废弃，前端未使用该接口。
func SearchAllLogs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": false,
		"message": "该接口已废弃",
	})
}

// Deprecated: SearchUserLogs 已废弃，前端未使用该接口。
func SearchUserLogs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": false,
		"message": "该接口已废弃",
	})
}

// GetLogByKey 通过令牌 API Key 查询该令牌的使用日志。
// 鉴权：TokenAuthReadOnly（禁用令牌/封禁用户不可查；过期/耗尽可查）。
// 筛选条件与用户侧日志查询一致（不含 token_name，已按当前令牌限定）。
func GetLogByKey(c *gin.Context) {
	tokenId := c.GetInt("token_id")
	userId := c.GetInt("id")
	if tokenId == 0 {
		common.ApiErrorMsg(c, "无效的令牌")
		return
	}
	pageInfo := common.GetPageQuery(c)
	logTypes := model.ParseLogTypesQuery(c.Query("type"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	modelName := c.Query("model_name")
	group := c.Query("group")
	requestId := c.Query("request_id")
	logs, total, err := model.GetTokenLogs(userId, tokenId, logTypes, startTimestamp, endTimestamp, modelName, pageInfo.GetStartIdx(), pageInfo.GetPageSize(), group, requestId, includeRawBillingLogs(c))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(logs)
	common.ApiSuccess(c, pageInfo)
}

// GetUserLogsByToken 通过令牌 API Key 查询当前用户的使用日志。
// 鉴权：TokenAuthReadOnly（禁用令牌/封禁用户不可查；过期/耗尽可查）；筛选条件与 GET /api/log/self 一致。
func GetUserLogsByToken(c *gin.Context) {
	GetUserLogs(c)
}

func GetLogsStat(c *gin.Context) {
	logType, _ := strconv.Atoi(c.Query("type"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	tokenName := c.Query("token_name")
	username := c.Query("username")
	modelName := c.Query("model_name")
	channel, _ := strconv.Atoi(c.Query("channel"))
	group := c.Query("group")
	stat, err := model.SumUsedQuota(logType, startTimestamp, endTimestamp, modelName, username, tokenName, channel, group)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	//tokenNum := model.SumUsedToken(logType, startTimestamp, endTimestamp, modelName, username, "")
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"quota":                stat.Quota,
			"display_amount":       stat.DisplayAmount,
			"text_display_amount":  stat.TextDisplayAmount,
			"image_display_amount": stat.ImageDisplayAmount,
			"video_display_amount": stat.VideoDisplayAmount,
			"rpm":                  stat.Rpm,
			"tpm":                  stat.Tpm,
		},
	})
	return
}

func GetLogsSelfStat(c *gin.Context) {
	username := c.GetString("username")
	logType, _ := strconv.Atoi(c.Query("type"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	tokenName := c.Query("token_name")
	modelName := c.Query("model_name")
	channel, _ := strconv.Atoi(c.Query("channel"))
	group := c.Query("group")
	quotaNum, err := model.SumUsedQuota(logType, startTimestamp, endTimestamp, modelName, username, tokenName, channel, group)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	//tokenNum := model.SumUsedToken(logType, startTimestamp, endTimestamp, modelName, username, tokenName)
	c.JSON(200, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"quota":                quotaNum.Quota,
			"display_amount":       quotaNum.DisplayAmount,
			"text_display_amount":  quotaNum.TextDisplayAmount,
			"image_display_amount": quotaNum.ImageDisplayAmount,
			"video_display_amount": quotaNum.VideoDisplayAmount,
			"rpm":                  quotaNum.Rpm,
			"tpm":                  quotaNum.Tpm,
			//"token": tokenNum,
		},
	})
	return
}

func DeleteHistoryLogs(c *gin.Context) {
	targetTimestamp, _ := strconv.ParseInt(c.Query("target_timestamp"), 10, 64)
	if targetTimestamp == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "target timestamp is required",
		})
		return
	}
	count, err := model.DeleteOldLog(c.Request.Context(), targetTimestamp, 100)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    count,
	})
	return
}
