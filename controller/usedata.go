package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

const maxUserQuotaRangeSeconds int64 = 30 * 24 * 60 * 60

func GetAllQuotaDates(c *gin.Context) {
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	username := c.Query("username")
	mxxh := float64(1)
	if role := c.GetInt("role"); role == common.RoleAdminUser || role == common.RoleRootUser {
		var err error
		mxxh, err = model.GetModelConsumptionDistributionMultiplier()
		if err != nil {
			common.ApiError(c, err)
			return
		}
	}
	dates, stat, err := model.AggregateQuotaDataFromLogs(startTimestamp, endTimestamp, 0, username)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    dates,
		"mxxh":    mxxh,
		"stat": gin.H{
			"quota":          stat.Quota,
			"display_amount": stat.DisplayAmount,
			"rpm":            stat.Rpm,
			"tpm":            stat.Tpm,
		},
	})
	return
}

func GetUserQuotaDates(c *gin.Context) {
	userId := c.GetInt("id")
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	// 用户看板最多查询最近 30 天，前端会预留一分钟边界。
	if endTimestamp-startTimestamp > maxUserQuotaRangeSeconds {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "时间跨度不能超过 1 个月",
		})
		return
	}
	mxxh := float64(1)
	if role := c.GetInt("role"); role == common.RoleAdminUser || role == common.RoleRootUser {
		var err error
		mxxh, err = model.GetModelConsumptionDistributionMultiplier()
		if err != nil {
			common.ApiError(c, err)
			return
		}
	}
	dates, stat, err := model.AggregateQuotaDataFromLogs(startTimestamp, endTimestamp, userId, "")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    dates,
		"mxxh":    mxxh,
		"stat": gin.H{
			"quota":          stat.Quota,
			"display_amount": stat.DisplayAmount,
			"rpm":            stat.Rpm,
			"tpm":            stat.Tpm,
		},
	})
	return
}
