package controller

import (
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// SupplierModelUsageItem 供应商模型使用统计项。
type SupplierModelUsageItem struct {
	ModelName string `json:"model_name"`
	Requests  int    `json:"requests"`
	Tokens    int    `json:"tokens"`
	Quota     int    `json:"quota"`
}

// parseSupplierDashboardTimeRange 解析供应商看板时间范围（默认最近24小时）。
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

// GetSupplierDashboardData 返回供应商数据看板（供应商看自己，管理员看全部供应商模型）。
func GetSupplierDashboardData(c *gin.Context) {
	startTimestamp, endTimestamp := parseSupplierDashboardTimeRange(c)

	var (
		modelNamesMap map[string]struct{}
		err           error
	)

	// 管理员默认查看全部供应商模型；当传 supplier_id 时查看指定供应商。
	if c.GetInt("role") >= common.RoleAdminUser {
		supplierID, _ := strconv.Atoi(c.Query("supplier_id"))
		if supplierID > 0 {
			modelNamesMap, err = collectSupplierOwnedModelNamesBySupplierID(supplierID)
		} else {
			modelNamesMap, err = collectAllSupplierOwnedModelNames()
		}
	} else {
		modelNamesMap, err = collectSupplierOwnedModelNames(c.GetInt("id"))
	}
	if err != nil {
		common.ApiError(c, err)
		return
	}

	modelNames := toSortedModelSlice(modelNamesMap)
	quotaData, err := model.GetQuotaDataByModelNames(startTimestamp, endTimestamp, modelNames)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	stat, err := model.SumUsedQuotaByModelNames(startTimestamp, endTimestamp, modelNames)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	usageMap := make(map[string]*SupplierModelUsageItem)
	totalRequests := 0
	totalTokens := 0
	totalQuota := 0

	for _, item := range quotaData {
		if item == nil {
			continue
		}
		totalRequests += item.Count
		totalTokens += item.TokenUsed
		totalQuota += item.Quota

		usageItem, ok := usageMap[item.ModelName]
		if !ok {
			usageItem = &SupplierModelUsageItem{
				ModelName: item.ModelName,
			}
			usageMap[item.ModelName] = usageItem
		}
		usageItem.Requests += item.Count
		usageItem.Tokens += item.TokenUsed
		usageItem.Quota += item.Quota
	}

	modelUsageStats := make([]*SupplierModelUsageItem, 0, len(usageMap))
	for _, usageItem := range usageMap {
		modelUsageStats = append(modelUsageStats, usageItem)
	}
	sort.Slice(modelUsageStats, func(i, j int) bool {
		return modelUsageStats[i].Quota > modelUsageStats[j].Quota
	})

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"start_timestamp":   startTimestamp,
			"end_timestamp":     endTimestamp,
			"model_names":       modelNames,
			"quota_data":        quotaData,
			"model_usage_stats": modelUsageStats,
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
				// provided_model_count: 供应商配置过的模型总数（无请求也计入）。
				"provided_model_count": len(modelNames),
				// active_model_count: 时间范围内有调用数据的模型数。
				"active_model_count": len(modelUsageStats),
				// model_count 保留为兼容字段，语义等同 provided_model_count。
				"model_count": len(modelNames),
				"top_models":  modelUsageStats,
			},
		},
	})
}
