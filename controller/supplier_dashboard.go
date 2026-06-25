package controller

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// SupplierModelUsageItem 供应商模型使用统计项。
type SupplierModelUsageItem struct {
	ModelName string                    `json:"model_name"`
	Requests  int                       `json:"requests"`
	Tokens    int                       `json:"tokens"`
	Quota     int                       `json:"quota"`
	Users     []model.SupplierUsageByUser `json:"users,omitempty"`
}

// loadSupplierDashboardAccount 加载供应商对接人（申请人）在平台上的剩余额度与历史累计已用额度，与使用日志中的额度/花费字段同源。
func loadSupplierDashboardAccount(c *gin.Context, adminSupplierID int) (quota int, usedQuota int) {
	if c.GetInt("role") >= common.RoleAdminUser {
		if adminSupplierID <= 0 {
			return 0, 0
		}
		app, err := model.GetSupplierByID(adminSupplierID)
		if err != nil || app == nil || app.ApplicantUserID <= 0 {
			return 0, 0
		}
		u, err := model.GetUserById(app.ApplicantUserID, false)
		if err != nil {
			return 0, 0
		}
		return u.Quota, u.UsedQuota
	}
	u, err := model.GetUserById(c.GetInt("id"), false)
	if err != nil {
		return 0, 0
	}
	return u.Quota, u.UsedQuota
}

// parseSupplierDashboardTimeRange 解析供应商看板时间范围：请求参数 start_timestamp、end_timestamp 为 Unix 秒；未传或非法时默认最近 24 小时。
func parseSupplierDashboardTimeRange(c *gin.Context) (int64, int64) {
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	if endTimestamp <= 0 {
		endTimestamp = time.Now().Unix()
	}
	if startTimestamp <= 0 || startTimestamp >= endTimestamp {
		startTimestamp = endTimestamp - 24*3600
	}
	return startTimestamp, endTimestamp
}

// toSortedModelSlice 将模型集合转换为稳定排序切片。
func toSortedModelSlice(modelsMap map[string]struct{}) []string {
	modelNames := make([]string, 0, len(modelsMap))
	for modelName := range modelsMap {
		modelNames = append(modelNames, modelName)
	}
	sort.Strings(modelNames)
	return modelNames
}

// mergeConfiguredAndActiveModelNames 合并渠道配置模型与日志中实际出现过的模型名。
func mergeConfiguredAndActiveModelNames(configured map[string]struct{}, usageByModel []model.SupplierUsageByModel) []string {
	merged := make(map[string]struct{}, len(configured)+len(usageByModel))
	for name := range configured {
		merged[name] = struct{}{}
	}
	for _, row := range usageByModel {
		name := row.ModelName
		if name == "" {
			continue
		}
		merged[name] = struct{}{}
	}
	return toSortedModelSlice(merged)
}

// GetSupplierDashboardData 返回供应商数据看板：仅统计自有渠道上的全部模型消费（与按渠道筛选的使用日志一致）。
func GetSupplierDashboardData(c *gin.Context) {
	startTimestamp, endTimestamp := parseSupplierDashboardTimeRange(c)
	adminSupplierID, _ := strconv.Atoi(c.Query("supplier_id"))
	accountQuota, accountUsedQuota := loadSupplierDashboardAccount(c, adminSupplierID)

	var (
		scope supplierDashboardScope
		err   error
	)

	if c.GetInt("role") >= common.RoleAdminUser {
		if adminSupplierID > 0 {
			scope, err = collectSupplierDashboardScopeBySupplierID(adminSupplierID)
		} else {
			scope, err = collectAllSupplierDashboardScope()
		}
	} else {
		scope, err = collectSupplierDashboardScope(c.GetInt("id"))
	}
	if err != nil {
		common.ApiError(c, err)
		return
	}

	quotaData, usageByModel, stat, err := model.AggregateSupplierUsageFromLogs(
		startTimestamp, endTimestamp, scope.ChannelIDs,
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	usageByModelUser, err := model.AggregateSupplierUsageAllModelUsers(
		startTimestamp, endTimestamp, scope.ChannelIDs,
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	usersByModel := model.GroupSupplierUsageByModelUsers(usageByModelUser)

	usageMap := make(map[string]*SupplierModelUsageItem, len(usageByModel))
	totalRequests := 0
	totalTokens := 0
	totalQuota := 0

	for _, row := range usageByModel {
		totalRequests += row.Count
		totalTokens += row.TokenUsed
		totalQuota += row.Quota
		usageMap[row.ModelName] = &SupplierModelUsageItem{
			ModelName: row.ModelName,
			Requests:  row.Count,
			Tokens:    row.TokenUsed,
			Quota:     row.Quota,
			Users:     usersByModel[row.ModelName],
		}
	}

	modelUsageStats := make([]*SupplierModelUsageItem, 0, len(usageMap))
	for _, usageItem := range usageMap {
		modelUsageStats = append(modelUsageStats, usageItem)
	}
	sort.Slice(modelUsageStats, func(i, j int) bool {
		return modelUsageStats[i].Quota > modelUsageStats[j].Quota
	})

	configuredModelNames := toSortedModelSlice(scope.ConfiguredModelNames)
	modelNames := mergeConfiguredAndActiveModelNames(scope.ConfiguredModelNames, usageByModel)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"start_timestamp": startTimestamp,
			"end_timestamp":   endTimestamp,
			"usage_time_range": gin.H{
				"start_timestamp": startTimestamp,
				"end_timestamp":   endTimestamp,
				"bucket":          "hour",
			},
			"account": gin.H{
				"quota":      accountQuota,
				"used_quota": accountUsedQuota,
			},
			"model_names":              modelNames,
			"configured_model_names":   configuredModelNames,
			"channel_ids":              scope.ChannelIDs,
			"channel_count":            len(scope.ChannelIDs),
			"quota_data":               quotaData,
			"model_usage_stats":        modelUsageStats,
			"resource_consumption": gin.H{
				"total_requests": totalRequests,
				"total_tokens":   totalTokens,
				"total_quota":    totalQuota,
			},
			"performance_metrics": gin.H{
				"rpm": stat.Rpm,
				"tpm": stat.Tpm,
			},
			"model_data_analysis": gin.H{
				"provided_model_count": len(scope.ConfiguredModelNames),
				"active_model_count":   len(modelUsageStats),
				"model_count":          len(scope.ConfiguredModelNames),
				"top_models":           modelUsageStats,
			},
		},
	})
}

// GetSupplierDashboardModelUserUsage 返回指定模型在供应商渠道上的按用户消费明细（与看板时间范围、渠道范围一致）。
func GetSupplierDashboardModelUserUsage(c *gin.Context) {
	modelName := strings.TrimSpace(c.Query("model_name"))
	if modelName == "" {
		common.ApiErrorMsg(c, "model_name 不能为空")
		return
	}

	scope, err := resolveSupplierDashboardScopeForRequest(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	startTimestamp, endTimestamp := parseSupplierDashboardTimeRange(c)
	rows, err := model.AggregateSupplierUsageByModelAndUser(
		startTimestamp, endTimestamp, scope.ChannelIDs, modelName,
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	totalRequests := 0
	totalTokens := 0
	totalQuota := 0
	for _, row := range rows {
		totalRequests += row.Requests
		totalTokens += row.TokenUsed
		totalQuota += row.Quota
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"model_name":      modelName,
			"start_timestamp": startTimestamp,
			"end_timestamp":   endTimestamp,
			"users":           rows,
			"summary": gin.H{
				"total_requests": totalRequests,
				"total_tokens":   totalTokens,
				"total_quota":    totalQuota,
			},
		},
	})
}

const supplierDashboardExportMaxRangeSec = 366 * 24 * 3600

// supplierDashboardExportQuery 供应商看板使用详情导出参数（与看板统计时间范围口径一致）。
type supplierDashboardExportQuery struct {
	StartTs, EndTs int64
	Lang           string
	SupplierName   string
}

// parseSupplierDashboardExportQuery 解析看板导出时间范围与语言；最长一年，与看板查询一致。
func parseSupplierDashboardExportQuery(c *gin.Context) (supplierDashboardExportQuery, error) {
	var q supplierDashboardExportQuery
	startTimestamp, endTimestamp := parseSupplierDashboardTimeRange(c)
	q.StartTs = startTimestamp
	q.EndTs = endTimestamp
	if q.EndTs-q.StartTs > supplierDashboardExportMaxRangeSec {
		return q, fmt.Errorf("时间范围超出限制(最多一年)")
	}
	q.SupplierName = strings.TrimSpace(c.Query("supplier_name"))
	q.Lang = c.Query("lang")
	switch q.Lang {
	case "zh-CN", "zh-TW", "en", "fr", "ru", "ja", "vi", "id", "ms", "th", "sw":
		// ok
	case "":
		q.Lang = "zh-CN"
	default:
		q.Lang = "zh-CN"
	}
	return q, nil
}

// ExportSupplierDashboardUsage 导出供应商看板使用详情账单（模型汇总 + 逐条明细，含输入/输出 Token）。
func ExportSupplierDashboardUsage(c *gin.Context) {
	role := c.GetInt("role")
	if role < common.RoleAdminUser {
		user, err := model.GetUserById(c.GetInt("id"), false)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		if user == nil || user.SupplierID == 0 {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "需要供应商权限"})
			return
		}
	}

	scope, err := resolveSupplierDashboardScopeForRequest(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if len(scope.ChannelIDs) == 0 {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "未找到可导出的供应商渠道"})
		return
	}

	query, err := parseSupplierDashboardExportQuery(c)
	if err != nil {
		c.JSON(400, gin.H{"success": false, "message": err.Error()})
		return
	}

	modelRows, err := model.AggregateSupplierUsageByModelWithTokenBreakdown(
		query.StartTs, query.EndTs, scope.ChannelIDs,
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	logTypes := model.ParseLogTypesQuery(c.Query("type"))
	logs, _, err := model.GetSupplierChannelLogsForExport(
		scope.ChannelIDs,
		logTypes,
		query.StartTs,
		query.EndTs,
		"",
		"",
		"",
		"",
		0,
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	dict := resolveSupplierDashboardExportDict(query.Lang)
	filename := fmt.Sprintf("supplier-usage-detail-%d.csv", time.Now().Unix())
	streamSupplierDashboardUsageCSV(c, modelRows, logs, query, filename, dict)
}
