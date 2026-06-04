package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// resolveSupplierDashboardScopeForRequest 解析当前请求对应的供应商看板渠道范围。
func resolveSupplierDashboardScopeForRequest(c *gin.Context) (supplierDashboardScope, error) {
	adminSupplierID, _ := strconv.Atoi(c.Query("supplier_id"))
	if c.GetInt("role") >= common.RoleAdminUser {
		if adminSupplierID > 0 {
			return collectSupplierDashboardScopeBySupplierID(adminSupplierID)
		}
		return collectAllSupplierDashboardScope()
	}
	return collectSupplierDashboardScope(c.GetInt("id"))
}

// GetSupplierChannelLogs 供应商渠道使用日志（与数据看板相同 channel_id 范围，不限 user_id）。
func GetSupplierChannelLogs(c *gin.Context) {
	scope, err := resolveSupplierDashboardScopeForRequest(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo := common.GetPageQuery(c)
	logTypes := model.ParseLogTypesQuery(c.Query("type"))
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	channelFilter, _ := strconv.Atoi(c.Query("channel"))
	logs, total, err := model.GetSupplierChannelLogs(
		scope.ChannelIDs,
		logTypes,
		startTimestamp,
		endTimestamp,
		c.Query("model_name"),
		c.Query("token_name"),
		c.Query("group"),
		c.Query("request_id"),
		channelFilter,
		pageInfo.GetStartIdx(),
		pageInfo.GetPageSize(),
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(logs)
	common.ApiSuccess(c, pageInfo)
}

// GetSupplierChannelLogsStat 供应商渠道使用日志统计（与数据看板 resource_consumption 同源）。
func GetSupplierChannelLogsStat(c *gin.Context) {
	scope, err := resolveSupplierDashboardScopeForRequest(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	startTimestamp, endTimestamp := parseSupplierDashboardTimeRange(c)
	stat, err := model.SummarizeSupplierChannelLogs(startTimestamp, endTimestamp, scope.ChannelIDs)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    stat,
	})
}
