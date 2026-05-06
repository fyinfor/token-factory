package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

func GetRateLimitBlacklistUsers(c *gin.Context) {
	limit := int64(200)
	if raw := c.Query("limit"); raw != "" {
		if n, err := strconv.ParseInt(raw, 10, 64); err == nil && n > 0 {
			limit = n
		}
	}
	items, err := service.ListUserRateLimitBlacklist(limit)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    items,
	})
}

type removeRateLimitBlacklistRequest struct {
	UserID int `json:"user_id"`
}

func DeleteRateLimitBlacklistUser(c *gin.Context) {
	var req removeRateLimitBlacklistRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil || req.UserID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的用户ID",
		})
		return
	}
	if err := service.RemoveUserRateLimitBlacklist(req.UserID); err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}
