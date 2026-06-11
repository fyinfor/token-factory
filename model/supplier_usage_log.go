package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"gorm.io/gorm"
)

// logHourBucketExpr 将 Unix 秒时间戳对齐到小时桶（与 quota_data 的 LogQuotaData 一致）。
const logHourBucketExpr = "(created_at - (created_at % 3600))"

// supplierLogTokenSumExpr 跨库安全的 Token 汇总（避免 NULL 导致 sum 结果为 NULL）。
const supplierLogTokenSumExpr = "COALESCE(SUM(prompt_tokens), 0) + COALESCE(SUM(completion_tokens), 0)"

// SupplierChannelLogsStat 供应商渠道使用日志统计（与数据看板 AggregateSupplierUsageFromLogs 同源）。
type SupplierChannelLogsStat struct {
	Quota         int     `json:"quota"`
	DisplayAmount float64 `json:"display_amount"`
	Rpm           int     `json:"rpm"`
	Tpm           int     `json:"tpm"`
	TotalTokens   int     `json:"total_tokens"`
	TotalRequests int     `json:"total_requests"`
}

// SupplierUsageByModel 供应商看板按模型聚合（仅消费日志、限定供应商自有渠道）。
type SupplierUsageByModel struct {
	ModelName string `json:"model_name" gorm:"column:model_name"`
	Count     int    `json:"count" gorm:"column:count"`
	TokenUsed int    `json:"token_used" gorm:"column:token_used"`
	Quota     int    `json:"quota" gorm:"column:quota"`
}

// SupplierUsageByUser 供应商看板按模型+用户聚合（同一模型下各用户的请求/Token/额度）。
type SupplierUsageByUser struct {
	UserID    int    `json:"user_id" gorm:"column:user_id"`
	Username  string `json:"username" gorm:"column:username"`
	Requests  int    `json:"requests" gorm:"column:count"`
	TokenUsed int    `json:"tokens" gorm:"column:token_used"`
	Quota     int    `json:"quota" gorm:"column:quota"`
}

func applySupplierChannelScope(tx *gorm.DB, startTimestamp, endTimestamp int64, channelIDs []int) *gorm.DB {
	if startTimestamp != 0 {
		tx = tx.Where("created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		tx = tx.Where("created_at <= ?", endTimestamp)
	}
	if len(channelIDs) > 0 {
		tx = tx.Where("channel_id IN ?", channelIDs)
	} else {
		tx = tx.Where("1 = 0")
	}
	return tx
}

func applySupplierChannelLogFilters(tx *gorm.DB, startTimestamp, endTimestamp int64, channelIDs []int) *gorm.DB {
	return applySupplierChannelScope(tx, startTimestamp, endTimestamp, channelIDs).
		Where("type = ?", LogTypeConsume)
}

// AggregateSupplierUsageFromLogs 从使用日志聚合供应商看板：仅统计指定渠道上的全部模型（按 model_name 汇总请求/Token/额度）。
func AggregateSupplierUsageFromLogs(startTimestamp, endTimestamp int64, channelIDs []int) (
	hourly []*QuotaData,
	byModel []SupplierUsageByModel,
	stat Stat,
	err error,
) {
	if len(channelIDs) == 0 {
		return []*QuotaData{}, []SupplierUsageByModel{}, Stat{}, nil
	}

	base := LOG_DB.Table("logs")
	base = applySupplierChannelLogFilters(base, startTimestamp, endTimestamp, channelIDs)

	var hourlyRows []*QuotaData
	err = base.Session(&gorm.Session{}).
		Select("model_name, sum(quota) as quota, count(*) as count, "+supplierLogTokenSumExpr+" as token_used, "+logHourBucketExpr+" as created_at").
		Group("model_name, " + logHourBucketExpr).
		Order("created_at asc").
		Find(&hourlyRows).Error
	if err != nil {
		return nil, nil, stat, errors.New("查询供应商看板时序数据失败")
	}

	var modelRows []SupplierUsageByModel
	err = base.Session(&gorm.Session{}).
		Select("model_name, count(*) as count, sum(quota) as quota, "+supplierLogTokenSumExpr+" as token_used").
		Group("model_name").
		Find(&modelRows).Error
	if err != nil {
		return nil, nil, stat, errors.New("查询供应商看板模型统计失败")
	}

	var quotaRows []struct {
		Quota int `gorm:"column:quota"`
	}
	quotaTx := LOG_DB.Table("logs").Select("quota")
	quotaTx = applySupplierChannelLogFilters(quotaTx, startTimestamp, endTimestamp, channelIDs)
	if err = quotaTx.Find(&quotaRows).Error; err != nil {
		return nil, nil, stat, errors.New("查询供应商看板额度统计失败")
	}
	for _, row := range quotaRows {
		stat.Quota += row.Quota
		stat.DisplayAmount += logger.QuotaToRoundedDisplayAmount(row.Quota, 2)
	}

	var rpmTpmRow struct {
		Rpm int `gorm:"column:rpm"`
		Tpm int `gorm:"column:tpm"`
	}
	rpmTpmQuery := LOG_DB.Table("logs").Select("count(*) rpm, coalesce(sum(prompt_tokens),0) + coalesce(sum(completion_tokens),0) as tpm")
	rpmTpmQuery = applySupplierChannelLogFilters(rpmTpmQuery, startTimestamp, endTimestamp, channelIDs)
	if err = rpmTpmQuery.Scan(&rpmTpmRow).Error; err != nil {
		return nil, nil, stat, errors.New("查询供应商看板 RPM/TPM 失败")
	}
	stat.Rpm = rpmTpmRow.Rpm
	stat.Tpm = rpmTpmRow.Tpm

	return hourlyRows, modelRows, stat, nil
}

// SupplierUsageByModelUser 供应商看板按模型+用户聚合（一次查询全量模型）。
type SupplierUsageByModelUser struct {
	ModelName string `json:"model_name" gorm:"column:model_name"`
	UserID    int    `json:"user_id" gorm:"column:user_id"`
	Username  string `json:"username" gorm:"column:username"`
	Requests  int    `json:"requests" gorm:"column:count"`
	TokenUsed int    `json:"tokens" gorm:"column:token_used"`
	Quota     int    `json:"quota" gorm:"column:quota"`
}

// AggregateSupplierUsageAllModelUsers 按模型与用户聚合全部消费（供看板明细与花费展示口径一致）。
func AggregateSupplierUsageAllModelUsers(
	startTimestamp, endTimestamp int64,
	channelIDs []int,
) ([]SupplierUsageByModelUser, error) {
	if len(channelIDs) == 0 {
		return []SupplierUsageByModelUser{}, nil
	}

	base := LOG_DB.Table("logs")
	base = applySupplierChannelLogFilters(base, startTimestamp, endTimestamp, channelIDs)

	var rows []SupplierUsageByModelUser
	err := base.Session(&gorm.Session{}).
		Select("model_name, user_id, username, count(*) as count, sum(quota) as quota, " + supplierLogTokenSumExpr + " as token_used").
		Group("model_name, user_id, username").
		Order("model_name asc, quota desc, count desc").
		Find(&rows).Error
	if err != nil {
		return nil, errors.New("查询供应商模型用户统计失败")
	}
	return rows, nil
}

// GroupSupplierUsageByModelUsers 将模型+用户聚合行按 model_name 分组为 users 列表。
func GroupSupplierUsageByModelUsers(rows []SupplierUsageByModelUser) map[string][]SupplierUsageByUser {
	out := make(map[string][]SupplierUsageByUser)
	for _, row := range rows {
		if row.ModelName == "" {
			continue
		}
		out[row.ModelName] = append(out[row.ModelName], SupplierUsageByUser{
			UserID:    row.UserID,
			Username:  row.Username,
			Requests:  row.Requests,
			TokenUsed: row.TokenUsed,
			Quota:     row.Quota,
		})
	}
	return out
}

// AggregateSupplierUsageByModelAndUser 按指定模型与用户聚合供应商渠道消费（与看板 model_usage_stats 同源范围）。
func AggregateSupplierUsageByModelAndUser(
	startTimestamp, endTimestamp int64,
	channelIDs []int,
	modelName string,
) ([]SupplierUsageByUser, error) {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" || len(channelIDs) == 0 {
		return []SupplierUsageByUser{}, nil
	}

	base := LOG_DB.Table("logs")
	base = applySupplierChannelLogFilters(base, startTimestamp, endTimestamp, channelIDs).
		Where("model_name = ?", modelName)

	var rows []SupplierUsageByUser
	err := base.Session(&gorm.Session{}).
		Select("user_id, username, count(*) as count, sum(quota) as quota, " + supplierLogTokenSumExpr + " as token_used").
		Group("user_id, username").
		Order("quota desc, count desc").
		Find(&rows).Error
	if err != nil {
		return nil, errors.New("查询供应商模型用户统计失败")
	}
	return rows, nil
}

// SummarizeSupplierChannelLogs 汇总供应商渠道日志（请求数、Token、额度、RPM/TPM），供看板与渠道日志统计接口复用。
func SummarizeSupplierChannelLogs(startTimestamp, endTimestamp int64, channelIDs []int) (SupplierChannelLogsStat, error) {
	out := SupplierChannelLogsStat{}
	_, byModel, stat, err := AggregateSupplierUsageFromLogs(startTimestamp, endTimestamp, channelIDs)
	if err != nil {
		return out, err
	}
	out.Quota = stat.Quota
	out.DisplayAmount = stat.DisplayAmount
	out.Rpm = stat.Rpm
	out.Tpm = stat.Tpm
	for _, row := range byModel {
		out.TotalRequests += row.Count
		out.TotalTokens += row.TokenUsed
	}
	return out, nil
}

// GetSupplierChannelLogs 分页查询供应商自有渠道上的消费日志（不限 user_id，与数据看板统计范围一致）。
func GetSupplierChannelLogs(
	channelIDs []int,
	logTypes []int,
	startTimestamp, endTimestamp int64,
	modelName, tokenName, group, requestID string,
	channelFilter int,
	startIdx, num int,
) (logs []*Log, total int64, err error) {
	if len(channelIDs) == 0 {
		return []*Log{}, 0, nil
	}
	allowed := make(map[int]struct{}, len(channelIDs))
	for _, id := range channelIDs {
		if id > 0 {
			allowed[id] = struct{}{}
		}
	}
	if channelFilter > 0 {
		if _, ok := allowed[channelFilter]; !ok {
			return []*Log{}, 0, nil
		}
		channelIDs = []int{channelFilter}
	}

	tx := LOG_DB.Table("logs")
	tx = applySupplierChannelScope(tx, startTimestamp, endTimestamp, channelIDs)
	tx = applyLogTypesFilter(tx, logTypes)
	if modelName != "" {
		modelNamePattern, patErr := sanitizeLikePattern(modelName)
		if patErr != nil {
			return nil, 0, patErr
		}
		tx = tx.Where("logs.model_name LIKE ? ESCAPE '!'", modelNamePattern)
	}
	if tokenName != "" {
		tx = tx.Where("logs.token_name = ?", tokenName)
	}
	if requestID != "" {
		tx = tx.Where("logs.request_id = ?", requestID)
	}
	if group != "" {
		tx = tx.Where("logs."+logGroupCol+" = ?", group)
	}

	err = tx.Session(&gorm.Session{}).Model(&Log{}).Limit(logSearchCountLimit).Count(&total).Error
	if err != nil {
		return nil, 0, errors.New("查询供应商渠道日志失败")
	}
	err = tx.Order("logs.id desc").Limit(num).Offset(startIdx).Find(&logs).Error
	if err != nil {
		return nil, 0, errors.New("查询供应商渠道日志失败")
	}
	attachLogChannelDisplays(logs)
	for i := range logs {
		if logs[i].Other == "" {
			continue
		}
		otherMap, errParse := common.StrToMap(logs[i].Other)
		if errParse != nil || otherMap == nil {
			continue
		}
		delete(otherMap, "channel_name")
		logs[i].Other = common.MapToJsonStr(otherMap)
	}
	return logs, total, nil
}

// GetSupplierChannelLogsForExport 拉取供应商渠道日志导出数据（升序，不分页）。
func GetSupplierChannelLogsForExport(
	channelIDs []int,
	logTypes []int,
	startTimestamp, endTimestamp int64,
	modelName, tokenName, group, requestID string,
	channelFilter int,
) (logs []*Log, total int64, err error) {
	if len(channelIDs) == 0 {
		return []*Log{}, 0, nil
	}
	allowed := make(map[int]struct{}, len(channelIDs))
	for _, id := range channelIDs {
		if id > 0 {
			allowed[id] = struct{}{}
		}
	}
	if channelFilter > 0 {
		if _, ok := allowed[channelFilter]; !ok {
			return []*Log{}, 0, nil
		}
		channelIDs = []int{channelFilter}
	}

	tx := LOG_DB.Table("logs")
	tx = applySupplierChannelScope(tx, startTimestamp, endTimestamp, channelIDs)
	tx = applyLogTypesFilter(tx, logTypes)
	tx = applyBillingLogVisibility(tx, false)
	if modelName != "" {
		modelNamePattern, patErr := sanitizeLikePattern(modelName)
		if patErr != nil {
			return nil, 0, patErr
		}
		tx = tx.Where("logs.model_name LIKE ? ESCAPE '!'", modelNamePattern)
	}
	if tokenName != "" {
		tx = tx.Where("logs.token_name = ?", tokenName)
	}
	if requestID != "" {
		tx = tx.Where("logs.request_id = ?", requestID)
	}
	if group != "" {
		tx = tx.Where("logs."+logGroupCol+" = ?", group)
	}

	if err = tx.Session(&gorm.Session{}).Model(&Log{}).Limit(logExportCountLimit).Count(&total).Error; err != nil {
		return nil, 0, errors.New("查询供应商渠道日志失败")
	}
	if total > logExportCountLimit {
		return nil, total, fmt.Errorf("导出行数超过 %d 上限", logExportCountLimit)
	}
	if err = tx.Order("logs.id ASC").Find(&logs).Error; err != nil {
		return nil, 0, errors.New("查询供应商渠道日志失败")
	}
	attachLogChannelDisplays(logs)
	for i := range logs {
		if logs[i].Other == "" {
			continue
		}
		otherMap, errParse := common.StrToMap(logs[i].Other)
		if errParse != nil || otherMap == nil {
			continue
		}
		delete(otherMap, "channel_name")
		logs[i].Other = common.MapToJsonStr(otherMap)
	}
	return logs, total, nil
}
