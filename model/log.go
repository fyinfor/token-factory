package model

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"

	"github.com/gin-gonic/gin"

	"github.com/bytedance/gopkg/util/gopool"
	"gorm.io/gorm"
)

type Log struct {
	Id               int    `json:"id" gorm:"index:idx_created_at_id,priority:1;index:idx_user_id_id,priority:2"`
	UserId           int    `json:"user_id" gorm:"index;index:idx_user_id_id,priority:1"`
	CreatedAt        int64  `json:"created_at" gorm:"bigint;index:idx_created_at_id,priority:2;index:idx_created_at_type"`
	Type             int    `json:"type" gorm:"index:idx_created_at_type"`
	Content          string `json:"content"`
	Username         string `json:"username" gorm:"index;index:index_username_model_name,priority:2;default:''"`
	TokenName        string `json:"token_name" gorm:"index;default:''"`
	ModelName        string `json:"model_name" gorm:"index;index:index_username_model_name,priority:1;default:''"`
	Quota            int    `json:"quota" gorm:"default:0"`
	PromptTokens     int    `json:"prompt_tokens" gorm:"default:0"`
	CompletionTokens int    `json:"completion_tokens" gorm:"default:0"`
	UseTime          int    `json:"use_time" gorm:"default:0"`
	IsStream         bool   `json:"is_stream"`
	ChannelId        int    `json:"channel" gorm:"index"`
	// 不在 API 中暴露渠道展示名称（控制台日志仅展示渠道编号）。
	ChannelName string `json:"-" gorm:"->"`
	// ChannelDisplay is a read-only API display value, e.g. route_slug_supplier_type.
	ChannelDisplay string `json:"channel_display,omitempty" gorm:"-"`
	// RouteSlug is the channel route suffix for user-facing logs (model/{route_slug}).
	RouteSlug string `json:"route_slug,omitempty" gorm:"-"`
	TokenId   int    `json:"token_id" gorm:"default:0;index"`
	Group     string `json:"group" gorm:"index"`
	Ip        string `json:"ip" gorm:"index;default:''"`
	RequestId string `json:"request_id,omitempty" gorm:"type:varchar(64);index:idx_logs_request_id;default:''"`
	Other     string `json:"other"`
}

// don't use iota, avoid change log type value
const (
	LogTypeUnknown = 0
	LogTypeTopup   = 1
	LogTypeConsume = 2
	LogTypeManage  = 3
	LogTypeSystem  = 4
	LogTypeError   = 5
	LogTypeRefund  = 6
)

const (
	BillingPhaseNormal           = "normal"
	BillingPhasePreCharge        = "pre_charge"
	BillingPhaseSettlementMarker = "settlement_marker"
	BillingPhaseSettlementMerged = "settlement_merged"
	BillingPhaseDeltaCharge      = "delta_charge"
	BillingPhaseDeltaRefund      = "delta_refund"
	BillingPhaseRefund           = "refund"
)

func SetBillingLogMetadata(other map[string]interface{}, phase string, affectsBalance bool, displayQuota int, balanceDelta int64) map[string]interface{} {
	if other == nil {
		other = make(map[string]interface{})
	}
	if strings.TrimSpace(phase) == "" {
		phase = BillingPhaseNormal
	}
	other["billing_phase"] = phase
	other["affects_balance"] = affectsBalance
	other["display_quota"] = displayQuota
	other["balance_delta"] = balanceDelta
	return other
}

func logOtherNumber(v interface{}) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case int32:
		return float64(x), true
	case uint:
		return float64(x), true
	case uint64:
		return float64(x), true
	case uint32:
		return float64(x), true
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
		return f, err == nil
	}
	return 0, false
}

func logOtherHasPositiveNumber(other map[string]interface{}, key string) bool {
	if other == nil {
		return false
	}
	n, ok := logOtherNumber(other[key])
	return ok && n > 0
}

func isTaskSettlementMarkerLog(log *Log, other map[string]interface{}) bool {
	if log == nil || other == nil {
		return false
	}
	if log.Type != LogTypeConsume {
		return false
	}
	_, hasTaskID := other["task_id"].(string)
	hasSettlementNumbers := hasTaskID &&
		logOtherHasPositiveNumber(other, "actual_quota") &&
		logOtherHasPositiveNumber(other, "pre_consumed_quota")
	if !hasSettlementNumbers {
		return false
	}
	phase, _ := other["billing_phase"].(string)
	if phase == BillingPhaseSettlementMarker {
		return true
	}
	return log.Quota == 0
}

func inferBillingPhase(log *Log, other map[string]interface{}) string {
	if log == nil {
		return BillingPhaseNormal
	}
	if phase, _ := other["billing_phase"].(string); strings.TrimSpace(phase) != "" {
		return phase
	}
	if _, ok := other["task_id"]; ok {
		switch log.Type {
		case LogTypeRefund:
			return BillingPhaseRefund
		case LogTypeConsume:
			hasSettlementNumbers := logOtherHasPositiveNumber(other, "actual_quota") &&
				logOtherHasPositiveNumber(other, "pre_consumed_quota")
			if log.Quota == 0 && hasSettlementNumbers {
				return BillingPhaseSettlementMarker
			}
			if log.Quota > 0 && hasSettlementNumbers {
				if actual, _ := logOtherNumber(other["actual_quota"]); actual > 0 {
					if pre, _ := logOtherNumber(other["pre_consumed_quota"]); pre > actual {
						return BillingPhaseDeltaRefund
					}
				}
				return BillingPhaseDeltaCharge
			}
			if log.Quota > 0 {
				return BillingPhasePreCharge
			}
		}
	}
	return BillingPhaseNormal
}

func normalizeLogBillingMetadata(log *Log) {
	if log == nil || strings.TrimSpace(log.Other) == "" {
		return
	}
	other, err := common.StrToMap(log.Other)
	if err != nil || other == nil {
		return
	}
	phase := inferBillingPhase(log, other)
	if phase == BillingPhaseNormal {
		return
	}
	displayQuota := log.Quota
	if phase == BillingPhaseSettlementMarker {
		if actual, ok := logOtherNumber(other["actual_quota"]); ok && actual > 0 {
			displayQuota = int(actual)
		}
	}
	affectsBalance := LogTypeChargeable(log.Type) && log.Quota > 0
	balanceDelta := SignedLogDelta(log.Quota, log.Type)
	if phase == BillingPhaseSettlementMarker {
		affectsBalance = false
		balanceDelta = 0
	}
	if v, ok := other["affects_balance"].(bool); ok && !v {
		affectsBalance = false
		balanceDelta = 0
	}
	SetBillingLogMetadata(other, phase, affectsBalance, displayQuota, balanceDelta)
	log.Other = common.MapToJsonStr(other)
}

func normalizeLogsBillingMetadata(logs []*Log) {
	for _, log := range logs {
		normalizeLogBillingMetadata(log)
	}
}

func logTaskID(other map[string]interface{}) string {
	if other == nil {
		return ""
	}
	taskID, _ := other["task_id"].(string)
	return strings.TrimSpace(taskID)
}

func querySettlementMarkerByTaskID(taskID string) (*Log, map[string]interface{}) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return nil, nil
	}
	var marker Log
	err := LOG_DB.
		Where("logs.type = ? AND logs.other LIKE ? AND logs.other LIKE ? AND logs.other LIKE ?",
			LogTypeConsume,
			"%"+taskID+"%",
			"%\"actual_quota\"%",
			"%\"pre_consumed_quota\"%",
		).
		Order("logs.id desc").
		Limit(1).
		Find(&marker).Error
	if err != nil || marker.Id == 0 {
		return nil, nil
	}
	other, err := common.StrToMap(marker.Other)
	if err != nil || other == nil || !isTaskSettlementMarkerLog(&marker, other) {
		return nil, nil
	}
	return &marker, other
}

func queryTaskUseTimeByTaskID(taskID string) int {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return 0
	}
	var task Task
	err := DB.
		Select("id", "submit_time", "start_time", "finish_time").
		Where("task_id = ?", taskID).
		Limit(1).
		Find(&task).Error
	if err != nil || task.ID == 0 || task.FinishTime <= 0 {
		return 0
	}
	start := task.SubmitTime
	if start <= 0 {
		start = task.StartTime
	}
	if start <= 0 || task.FinishTime <= start {
		return 0
	}
	return int(task.FinishTime - start)
}

func fillTaskUseTime(logs []*Log) {
	cache := make(map[string]int)
	for _, log := range logs {
		if log == nil || log.UseTime > 0 || strings.TrimSpace(log.Other) == "" {
			continue
		}
		other, err := common.StrToMap(log.Other)
		if err != nil || other == nil {
			continue
		}
		taskID := logTaskID(other)
		if taskID == "" {
			continue
		}
		useTime, ok := cache[taskID]
		if !ok {
			useTime = queryTaskUseTimeByTaskID(taskID)
			cache[taskID] = useTime
		}
		if useTime > 0 {
			log.UseTime = useTime
		}
	}
}

func mergeSettlementMarkersIntoPreChargeLogs(logs []*Log) {
	for _, log := range logs {
		if log == nil || log.Type != LogTypeConsume || log.Quota <= 0 {
			continue
		}
		other, err := common.StrToMap(log.Other)
		if err != nil || other == nil {
			continue
		}
		if logOtherHasPositiveNumber(other, "actual_quota") {
			continue
		}
		taskID := logTaskID(other)
		if taskID == "" {
			continue
		}
		marker, markerOther := querySettlementMarkerByTaskID(taskID)
		if marker == nil || markerOther == nil {
			continue
		}
		for key, value := range markerOther {
			switch key {
			case "request_path", "billing_phase", "affects_balance", "balance_delta", "display_quota":
				continue
			default:
				other[key] = value
			}
		}
		actualQuota := log.Quota
		if actual, ok := logOtherNumber(markerOther["actual_quota"]); ok && actual > 0 {
			actualQuota = int(actual)
		}
		other["source_log_ids"] = map[string]interface{}{
			"pre_charge": log.Id,
			"settlement": marker.Id,
		}
		SetBillingLogMetadata(other, BillingPhaseSettlementMerged, true, actualQuota, SignedLogDelta(log.Quota, log.Type))
		log.Other = common.MapToJsonStr(other)
	}
}

func applyBillingLogVisibility(tx *gorm.DB, includeRawBillingLogs bool) *gorm.DB {
	if includeRawBillingLogs {
		return tx
	}
	// Hide async task settlement markers by default. They confirm the final task
	// price but do not change balance; raw mode can still return them for audit.
	return tx.Where(
		"NOT (logs.type = ? AND logs.other LIKE ? AND logs.other LIKE ? AND logs.other LIKE ?)",
		LogTypeConsume,
		"%\"actual_quota\"%",
		"%\"pre_consumed_quota\"%",
		"%\"affects_balance\":false%",
	)
}

func formatUserLogs(logs []*Log, startIdx int) {
	for i := range logs {
		logs[i].ChannelName = ""
		// Hide admin-only channel display; route_slug remains for user-facing logs.
		logs[i].ChannelDisplay = ""
		var otherMap map[string]interface{}
		otherMap, _ = common.StrToMap(logs[i].Other)
		if otherMap != nil {
			// Remove admin-only debug fields.
			delete(otherMap, "admin_info")
			// delete(otherMap, "reject_reason")
			delete(otherMap, "stream_status")
		}
		logs[i].Other = common.MapToJsonStr(otherMap)
		logs[i].Id = startIdx + i + 1
	}
}

func formatChannelDisplay(routeSlug string, supplierType string, channelID int) string {
	routeSlug = strings.TrimSpace(routeSlug)
	supplierType = strings.TrimSpace(supplierType)
	if routeSlug == "" && channelID > 0 {
		routeSlug = DefaultRouteSlugFromChannelID(int64(channelID))
	}
	if routeSlug == "" {
		return ""
	}
	if supplierType == "" {
		return routeSlug
	}
	return routeSlug + "_" + supplierType
}

// userFacingLogRouteSlug 返回控制台日志面向用户的路由后缀。
// 优先展示渠道当前有效 route_slug（含自定义如 tx）；缺失或非法时回退为默认 u+base62(id)。
func userFacingLogRouteSlug(routeSlug string, channelID int) string {
	if channelID <= 0 {
		return ""
	}
	defaultSlug := DefaultRouteSlugFromChannelID(int64(channelID))
	slug := strings.TrimSpace(routeSlug)
	if slug == "" {
		return defaultSlug
	}
	if IsValidRouteSlug(slug) {
		return slug
	}
	return defaultSlug
}

func attachLogChannelDisplays(logs []*Log) {
	if len(logs) == 0 {
		return
	}
	channelIDs := make([]int, 0, len(logs))
	seen := make(map[int]struct{}, len(logs))
	for _, log := range logs {
		if log == nil || log.ChannelId <= 0 {
			continue
		}
		if _, ok := seen[log.ChannelId]; ok {
			continue
		}
		seen[log.ChannelId] = struct{}{}
		channelIDs = append(channelIDs, log.ChannelId)
	}
	if len(channelIDs) == 0 {
		return
	}

	type channelDisplayRow struct {
		Id           int
		RouteSlug    string
		SupplierType string
	}
	var rows []channelDisplayRow
	if err := DB.Model(&Channel{}).
		Select("id", "route_slug", "supplier_type").
		Where("id IN ?", channelIDs).
		Find(&rows).Error; err != nil {
		common.SysError("failed to attach log channel display: " + err.Error())
		return
	}

	displayMap := make(map[int]string, len(rows))
	routeSlugMap := make(map[int]string, len(rows))
	for _, row := range rows {
		routeSlug := strings.TrimSpace(row.RouteSlug)
		if routeSlug == "" && row.Id > 0 {
			routeSlug = DefaultRouteSlugFromChannelID(int64(row.Id))
		}
		if routeSlug != "" {
			routeSlugMap[row.Id] = routeSlug
		}
		display := formatChannelDisplay(row.RouteSlug, row.SupplierType, row.Id)
		if display != "" {
			displayMap[row.Id] = display
		}
	}
	for i := range logs {
		channelID := logs[i].ChannelId
		if channelID <= 0 {
			continue
		}
		storedSlug := routeSlugMap[channelID]
		logs[i].RouteSlug = userFacingLogRouteSlug(storedSlug, channelID)
		if display, ok := displayMap[channelID]; ok {
			logs[i].ChannelDisplay = display
		}
	}
}

func GetLogByTokenId(tokenId int) (logs []*Log, err error) {
	err = LOG_DB.Model(&Log{}).Where("token_id = ?", tokenId).Order("id desc").Limit(common.MaxRecentItems).Find(&logs).Error
	attachLogChannelDisplays(logs)
	formatUserLogs(logs, 0)
	return logs, err
}

func RecordLog(userId int, logType int, content string) {
	if logType == LogTypeConsume && !common.LogConsumeEnabled {
		return
	}
	username, _ := GetUsernameById(userId, false)
	content = prependUsernameBeforeRole(content, username)
	log := &Log{
		UserId:    userId,
		Username:  username,
		CreatedAt: common.GetTimestamp(),
		Type:      logType,
		Content:   content,
	}
	err := LOG_DB.Create(log).Error
	if err != nil {
		common.SysLog("failed to record log: " + err.Error())
	}
}

// prependUsernameBeforeRole 在日志详情以角色词开头时，自动在角色前补充用户名，便于审计操作者身份。
func prependUsernameBeforeRole(content string, username string) string {
	if content == "" || username == "" {
		return content
	}
	rolePrefixes := []string{"管理员", "用户", "分销商", "供应商"}
	for _, rolePrefix := range rolePrefixes {
		withUsername := username + rolePrefix
		if strings.HasPrefix(content, withUsername) {
			return content
		}
		if strings.HasPrefix(content, rolePrefix) {
			return username + content
		}
	}
	return content
}

func RecordErrorLog(c *gin.Context, userId int, channelId int, modelName string, tokenName string, content string, tokenId int, useTimeSeconds int,
	isStream bool, group string, other map[string]interface{}) {
	logger.LogInfo(c, fmt.Sprintf("record error log: userId=%d, channelId=%d, modelName=%s, tokenName=%s, content=%s", userId, channelId, modelName, tokenName, content))
	username := c.GetString("username")
	requestId := c.GetString(common.RequestIdKey)
	otherStr := common.MapToJsonStr(other)
	// 判断是否需要记录 IP
	needRecordIp := false
	if settingMap, err := GetUserSetting(userId, false); err == nil {
		if settingMap.RecordIpLog {
			needRecordIp = true
		}
	}
	log := &Log{
		UserId:           userId,
		Username:         username,
		CreatedAt:        common.GetTimestamp(),
		Type:             LogTypeError,
		Content:          content,
		PromptTokens:     0,
		CompletionTokens: 0,
		TokenName:        tokenName,
		ModelName:        modelName,
		Quota:            0,
		ChannelId:        channelId,
		TokenId:          tokenId,
		UseTime:          useTimeSeconds,
		IsStream:         isStream,
		Group:            group,
		Ip: func() string {
			if needRecordIp {
				return c.ClientIP()
			}
			return ""
		}(),
		RequestId: requestId,
		Other:     otherStr,
	}
	err := LOG_DB.Create(log).Error
	if err != nil {
		logger.LogError(c, "failed to record log: "+err.Error())
	}
}

type RecordConsumeLogParams struct {
	ChannelId        int `json:"channel_id"`
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	// TokenUsed 显式指定写入 quota_data.token_used 的值；<=0 时回退为 PromptTokens+CompletionTokens。
	// 异步任务预扣路径（视频按 token/按秒/按次）没有真实 token 数，可将预扣额度作为消耗量上报，
	// 以便 /rankings 排行（按 quota_data.token_used 聚合）也能覆盖到 Seedance/Kling/Sora 等异步视频模型。
	TokenUsed      int                    `json:"token_used"`
	ModelName      string                 `json:"model_name"`
	TokenName      string                 `json:"token_name"`
	Quota          int                    `json:"quota"`
	Content        string                 `json:"content"`
	TokenId        int                    `json:"token_id"`
	UseTimeSeconds int                    `json:"use_time_seconds"`
	IsStream       bool                   `json:"is_stream"`
	Group          string                 `json:"group"`
	Other          map[string]interface{} `json:"other"`
}

func RecordConsumeLog(c *gin.Context, userId int, params RecordConsumeLogParams) {
	if !common.LogConsumeEnabled {
		return
	}
	logger.LogInfo(c, fmt.Sprintf("record consume log: userId=%d, params=%s", userId, common.GetJsonString(params)))
	username := c.GetString("username")
	requestId := c.GetString(common.RequestIdKey)
	otherStr := common.MapToJsonStr(params.Other)
	// 判断是否需要记录 IP
	needRecordIp := false
	if settingMap, err := GetUserSetting(userId, false); err == nil {
		if settingMap.RecordIpLog {
			needRecordIp = true
		}
	}
	log := &Log{
		UserId:           userId,
		Username:         username,
		CreatedAt:        common.GetTimestamp(),
		Type:             LogTypeConsume,
		Content:          params.Content,
		PromptTokens:     params.PromptTokens,
		CompletionTokens: params.CompletionTokens,
		TokenName:        params.TokenName,
		ModelName:        params.ModelName,
		Quota:            params.Quota,
		ChannelId:        params.ChannelId,
		TokenId:          params.TokenId,
		UseTime:          params.UseTimeSeconds,
		IsStream:         params.IsStream,
		Group:            params.Group,
		Ip: func() string {
			if needRecordIp {
				return c.ClientIP()
			}
			return ""
		}(),
		RequestId: requestId,
		Other:     otherStr,
	}
	err := LOG_DB.Create(log).Error
	if err != nil {
		logger.LogError(c, "failed to record log: "+err.Error())
	}
	if common.DataExportEnabled {
		gopool.Go(func() {
			tokenUsed := params.PromptTokens + params.CompletionTokens
			if params.TokenUsed > 0 {
				// 异步任务（视频/Seedance 等）没有真实 token 数，用调用方显式上报的消耗量。
				tokenUsed = params.TokenUsed
			}
			LogQuotaData(userId, username, params.ModelName, params.Quota, common.GetTimestamp(), tokenUsed)
		})
	}
}

type RecordTaskBillingLogParams struct {
	UserId         int
	LogType        int
	Content        string
	ChannelId      int
	ModelName      string
	TokenName      string
	Quota          int
	TokenId        int
	UseTimeSeconds int
	Group          string
	Other          map[string]interface{}
}

func RecordTaskBillingLog(params RecordTaskBillingLogParams) {
	if params.LogType == LogTypeConsume && !common.LogConsumeEnabled {
		return
	}
	username, _ := GetUsernameById(params.UserId, false)
	tokenName := params.TokenName
	if params.TokenId > 0 {
		if token, err := GetTokenById(params.TokenId); err == nil {
			tokenName = token.Name
		}
	} else if tokenName == "" {
		// playground/default token：避免任务结算日志中令牌名为空，导致前端不展示。
		tokenName = "playground-default"
	}
	log := &Log{
		UserId:    params.UserId,
		Username:  username,
		CreatedAt: common.GetTimestamp(),
		Type:      params.LogType,
		Content:   params.Content,
		TokenName: tokenName,
		ModelName: params.ModelName,
		Quota:     params.Quota,
		ChannelId: params.ChannelId,
		TokenId:   params.TokenId,
		UseTime:   params.UseTimeSeconds,
		Group:     params.Group,
		Other:     common.MapToJsonStr(params.Other),
	}
	err := LOG_DB.Create(log).Error
	if err != nil {
		common.SysLog("failed to record task billing log: " + err.Error())
	}
	// 异步任务结算（refund / delta_charge / delta_refund）也写一份 quota_data，
	// 使 /rankings 排行（按 quota_data.token_used 聚合）能覆盖到 Seedance/Kling/Sora 等异步视频模型。
	// - pre_charge 已经由 LogTaskConsumption → RecordConsumeLog 写入 quota_data；
	// - settlement_marker (Quota=0, affectsBalance=false) 仅作展示用，不重复写；
	// - 这里只对真正影响余额的事件做带符号追加/扣减。
	if common.DataExportEnabled && params.Quota > 0 && LogTypeChargeable(params.LogType) {
		affectsBalance := true
		if params.Other != nil {
			if v, ok := params.Other["affects_balance"].(bool); ok {
				affectsBalance = v
			}
		}
		if affectsBalance {
			signedQuota := params.Quota
			if params.LogType == LogTypeRefund {
				signedQuota = -params.Quota
			}
			gopool.Go(func() {
				LogQuotaData(params.UserId, username, params.ModelName, signedQuota, common.GetTimestamp(), signedQuota)
			})
		}
	}
}

// ParseLogTypesQuery 解析 type 查询参数，支持单个值或逗号分隔多值（如 "2" 或 "2,3,5"）。
// 空、"0" 或仅含 0 时返回 nil，表示不按类型过滤。
func ParseLogTypesQuery(typeQuery string) []int {
	typeQuery = strings.TrimSpace(typeQuery)
	if typeQuery == "" || typeQuery == "0" {
		return nil
	}
	parts := strings.Split(typeQuery, ",")
	types := make([]int, 0, len(parts))
	seen := make(map[int]struct{}, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == "0" {
			continue
		}
		v, err := strconv.Atoi(part)
		if err != nil || v <= 0 {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		types = append(types, v)
	}
	if len(types) == 0 {
		return nil
	}
	return types
}

func applyLogTypesFilter(tx *gorm.DB, logTypes []int) *gorm.DB {
	if len(logTypes) == 0 {
		return tx
	}
	if len(logTypes) == 1 {
		return tx.Where("logs.type = ?", logTypes[0])
	}
	return tx.Where("logs.type IN ?", logTypes)
}

func GetAllLogs(logTypes []int, startTimestamp int64, endTimestamp int64, modelName string, username string, tokenName string, startIdx int, num int, channel int, group string, requestId string, includeRawBillingLogs bool) (logs []*Log, total int64, err error) {
	tx := LOG_DB
	tx = applyLogTypesFilter(tx, logTypes)
	tx = applyBillingLogVisibility(tx, includeRawBillingLogs)

	if modelName != "" {
		tx = tx.Where("logs.model_name like ?", modelName)
	}
	if username != "" {
		tx = tx.Where("logs.username = ?", username)
	}
	if tokenName != "" {
		tx = tx.Where("logs.token_name = ?", tokenName)
	}
	if requestId != "" {
		tx = tx.Where("logs.request_id = ?", requestId)
	}
	if startTimestamp != 0 {
		tx = tx.Where("logs.created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		tx = tx.Where("logs.created_at <= ?", endTimestamp)
	}
	if channel != 0 {
		tx = tx.Where("logs.channel_id = ?", channel)
	}
	if group != "" {
		tx = tx.Where("logs."+logGroupCol+" = ?", group)
	}
	err = tx.Model(&Log{}).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	err = tx.Order("logs.id desc").Limit(num).Offset(startIdx).Find(&logs).Error
	if err != nil {
		return nil, 0, err
	}
	attachLogChannelDisplays(logs)
	normalizeLogsBillingMetadata(logs)
	fillTaskUseTime(logs)
	if !includeRawBillingLogs {
		mergeSettlementMarkersIntoPreChargeLogs(logs)
	}

	for i := range logs {
		if logs[i].Other == "" {
			continue
		}
		otherMap, errParse := common.StrToMap(logs[i].Other)
		if errParse != nil || otherMap == nil {
			continue
		}
		// 历史错误日志 other 中可能含 channel_name（渠道展示名），控制台不返回。
		delete(otherMap, "channel_name")
		logs[i].Other = common.MapToJsonStr(otherMap)
	}

	return logs, total, err
}

const logSearchCountLimit = 10000

func GetUserLogs(userId int, logTypes []int, startTimestamp int64, endTimestamp int64, modelName string, tokenName string, startIdx int, num int, group string, requestId string, includeRawBillingLogs bool) (logs []*Log, total int64, err error) {
	tx := LOG_DB.Where("logs.user_id = ?", userId)
	tx = applyLogTypesFilter(tx, logTypes)
	tx = applyBillingLogVisibility(tx, includeRawBillingLogs)

	if modelName != "" {
		modelNamePattern, err := sanitizeLikePattern(modelName)
		if err != nil {
			return nil, 0, err
		}
		tx = tx.Where("logs.model_name LIKE ? ESCAPE '!'", modelNamePattern)
	}
	if tokenName != "" {
		tx = tx.Where("logs.token_name = ?", tokenName)
	}
	if requestId != "" {
		tx = tx.Where("logs.request_id = ?", requestId)
	}
	if startTimestamp != 0 {
		tx = tx.Where("logs.created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		tx = tx.Where("logs.created_at <= ?", endTimestamp)
	}
	if group != "" {
		tx = tx.Where("logs."+logGroupCol+" = ?", group)
	}
	err = tx.Model(&Log{}).Limit(logSearchCountLimit).Count(&total).Error
	if err != nil {
		common.SysError("failed to count user logs: " + err.Error())
		return nil, 0, errors.New("查询日志失败")
	}
	err = tx.Order("logs.id desc").Limit(num).Offset(startIdx).Find(&logs).Error
	if err != nil {
		common.SysError("failed to search user logs: " + err.Error())
		return nil, 0, errors.New("查询日志失败")
	}

	normalizeLogsBillingMetadata(logs)
	fillTaskUseTime(logs)
	if !includeRawBillingLogs {
		mergeSettlementMarkersIntoPreChargeLogs(logs)
	}
	attachLogChannelDisplays(logs)
	formatUserLogs(logs, startIdx)
	return logs, total, err
}

// logConsumeAmountDigits 日志页消费统计展示金额固定小数位（与前端 LOG_CONSUME_AMOUNT_DIGITS 一致）。
const logConsumeAmountDigits = 6

type Stat struct {
	Quota              int     `json:"quota"`
	DisplayAmount      float64 `json:"display_amount"`
	TextDisplayAmount  float64 `json:"text_display_amount"`
	ImageDisplayAmount float64 `json:"image_display_amount"`
	VideoDisplayAmount float64 `json:"video_display_amount"`
	Rpm                int     `json:"rpm"`
	Tpm                int     `json:"tpm"`
}

const (
	logBillingTypeText  = "text"
	logBillingTypeImage = "image"
	logBillingTypeVideo = "video"
)

// classifyLogBillingType 将消费日志 other 归为文本 / 图片 / 视频三类计费。
func classifyLogBillingType(otherJSON string) string {
	other, _ := common.StrToMap(otherJSON)
	if other == nil {
		return logBillingTypeText
	}
	mode, _ := other["billing_mode"].(string)
	switch mode {
	case "image_per_image":
		return logBillingTypeImage
	case "video_per_second", "video_token_output", "video_per_video", "video_token":
		return logBillingTypeVideo
	}
	if other["video_billed_quota"] != nil || other["video_quota_per_unit"] != nil {
		return logBillingTypeVideo
	}
	if path, ok := other["request_path"].(string); ok && strings.Contains(path, "/videos") {
		return logBillingTypeVideo
	}
	return logBillingTypeText
}

type logConsumeQuotaRow struct {
	Quota     int    `gorm:"column:quota"`
	Other     string `gorm:"column:other"`
	ModelName string `gorm:"column:model_name"`
}

// accumulateLogConsumeDisplayStat 按「行级 6 位进一 → 类型分项再进一 → 总计再进一」汇总。
// 含文本、图片、视频全部计费类型。各模型分项进一逻辑见前端 aggregateLogConsumeStats。
func accumulateLogConsumeDisplayStat(rows []logConsumeQuotaRow, stat *Stat) {
	var textSum, imageSum, videoSum float64

	for _, row := range rows {
		stat.Quota += row.Quota
		lineAmount := logger.QuotaToCeilDisplayAmount(row.Quota, logConsumeAmountDigits)
		billingType := classifyLogBillingType(row.Other)
		switch billingType {
		case logBillingTypeImage:
			imageSum += lineAmount
		case logBillingTypeVideo:
			videoSum += lineAmount
		default:
			textSum += lineAmount
		}
	}

	stat.TextDisplayAmount = logger.CeilToFixedDecimals(textSum, logConsumeAmountDigits)
	stat.ImageDisplayAmount = logger.CeilToFixedDecimals(imageSum, logConsumeAmountDigits)
	stat.VideoDisplayAmount = logger.CeilToFixedDecimals(videoSum, logConsumeAmountDigits)
	// 总消耗 = 文本 + 图片 + 视频（三类均已进一）再收口 6 位
	stat.DisplayAmount = logger.CeilToFixedDecimals(
		stat.TextDisplayAmount+stat.ImageDisplayAmount+stat.VideoDisplayAmount,
		logConsumeAmountDigits,
	)
}

func SumUsedQuota(logType int, startTimestamp int64, endTimestamp int64, modelName string, username string, tokenName string, channel int, group string) (stat Stat, err error) {
	quotaListQuery := LOG_DB.Table("logs").Select("quota, other, model_name")
	rpmTpmQuery := LOG_DB.Table("logs").Select("count(*) rpm, COALESCE(sum(prompt_tokens), 0) + COALESCE(sum(completion_tokens), 0) tpm")
	quotaListQuery = applyBillingLogVisibility(quotaListQuery, false)
	rpmTpmQuery = applyBillingLogVisibility(rpmTpmQuery, false)

	if username != "" {
		quotaListQuery = quotaListQuery.Where("username = ?", username)
		rpmTpmQuery = rpmTpmQuery.Where("username = ?", username)
	}
	if tokenName != "" {
		quotaListQuery = quotaListQuery.Where("token_name = ?", tokenName)
		rpmTpmQuery = rpmTpmQuery.Where("token_name = ?", tokenName)
	}
	if startTimestamp != 0 {
		quotaListQuery = quotaListQuery.Where("created_at >= ?", startTimestamp)
		rpmTpmQuery = rpmTpmQuery.Where("created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		quotaListQuery = quotaListQuery.Where("created_at <= ?", endTimestamp)
		rpmTpmQuery = rpmTpmQuery.Where("created_at <= ?", endTimestamp)
	}
	if modelName != "" {
		modelNamePattern, err := sanitizeLikePattern(modelName)
		if err != nil {
			return stat, err
		}
		quotaListQuery = quotaListQuery.Where("model_name LIKE ? ESCAPE '!'", modelNamePattern)
		rpmTpmQuery = rpmTpmQuery.Where("model_name LIKE ? ESCAPE '!'", modelNamePattern)
	}
	if channel != 0 {
		quotaListQuery = quotaListQuery.Where("channel_id = ?", channel)
		rpmTpmQuery = rpmTpmQuery.Where("channel_id = ?", channel)
	}
	if group != "" {
		quotaListQuery = quotaListQuery.Where(logGroupCol+" = ?", group)
		rpmTpmQuery = rpmTpmQuery.Where(logGroupCol+" = ?", group)
	}

	quotaListQuery = quotaListQuery.Where("type = ?", LogTypeConsume)
	rpmTpmQuery = rpmTpmQuery.Where("type = ?", LogTypeConsume)

	var quotaRows []logConsumeQuotaRow
	if err := quotaListQuery.Find(&quotaRows).Error; err != nil {
		common.SysError("failed to query log stat: " + err.Error())
		return stat, errors.New("查询统计数据失败")
	}
	accumulateLogConsumeDisplayStat(quotaRows, &stat)

	var rpmTpmRow struct {
		Rpm int `gorm:"column:rpm"`
		Tpm int `gorm:"column:tpm"`
	}
	if err := rpmTpmQuery.Scan(&rpmTpmRow).Error; err != nil {
		common.SysError("failed to query rpm/tpm: " + err.Error())
		return stat, errors.New("查询统计数据失败")
	}
	stat.Rpm = rpmTpmRow.Rpm
	stat.Tpm = rpmTpmRow.Tpm

	return stat, nil
}

// SumUsedQuotaByModelNames 按模型集合统计消耗额度与筛选时间范围内的 rpm/tpm。
func SumUsedQuotaByModelNames(startTimestamp int64, endTimestamp int64, modelNames []string) (stat Stat, err error) {
	if len(modelNames) == 0 {
		return stat, nil
	}
	quotaListQuery := LOG_DB.Table("logs").Select("quota, other, model_name")
	rpmTpmQuery := LOG_DB.Table("logs").Select("count(*) rpm, COALESCE(sum(prompt_tokens), 0) + COALESCE(sum(completion_tokens), 0) tpm")

	if startTimestamp != 0 {
		quotaListQuery = quotaListQuery.Where("created_at >= ?", startTimestamp)
		rpmTpmQuery = rpmTpmQuery.Where("created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		quotaListQuery = quotaListQuery.Where("created_at <= ?", endTimestamp)
		rpmTpmQuery = rpmTpmQuery.Where("created_at <= ?", endTimestamp)
	}
	quotaListQuery = quotaListQuery.Where("model_name IN ?", modelNames)
	rpmTpmQuery = rpmTpmQuery.Where("model_name IN ?", modelNames)

	quotaListQuery = quotaListQuery.Where("type = ?", LogTypeConsume)
	rpmTpmQuery = rpmTpmQuery.Where("type = ?", LogTypeConsume)

	var quotaRows []logConsumeQuotaRow
	if err := quotaListQuery.Find(&quotaRows).Error; err != nil {
		common.SysError("failed to query supplier log stat: " + err.Error())
		return stat, errors.New("查询统计数据失败")
	}
	accumulateLogConsumeDisplayStat(quotaRows, &stat)

	var rpmTpmRow struct {
		Rpm int `gorm:"column:rpm"`
		Tpm int `gorm:"column:tpm"`
	}
	if err := rpmTpmQuery.Scan(&rpmTpmRow).Error; err != nil {
		common.SysError("failed to query supplier rpm/tpm stat: " + err.Error())
		return stat, errors.New("查询统计数据失败")
	}
	stat.Rpm = rpmTpmRow.Rpm
	stat.Tpm = rpmTpmRow.Tpm
	return stat, nil
}

func SumUsedToken(logType int, startTimestamp int64, endTimestamp int64, modelName string, username string, tokenName string) (token int) {
	tx := LOG_DB.Table("logs").Select("ifnull(sum(prompt_tokens),0) + ifnull(sum(completion_tokens),0)")
	if username != "" {
		tx = tx.Where("username = ?", username)
	}
	if tokenName != "" {
		tx = tx.Where("token_name = ?", tokenName)
	}
	if startTimestamp != 0 {
		tx = tx.Where("created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		tx = tx.Where("created_at <= ?", endTimestamp)
	}
	if modelName != "" {
		tx = tx.Where("model_name = ?", modelName)
	}
	tx.Where("type = ?", LogTypeConsume).Scan(&token)
	return token
}

// LogTypeChargeable 返回该类型日志是否参与"对账余额"计算（影响 User.Quota）。
// 仅 Consume/Topup/Refund 写入的 Quota 是真实发生额；Manage/System 的 Quota=0，
// 它们的金额已直接落到 User.Quota 上但日志里没存差异值，因此不参与累加。
func LogTypeChargeable(t int) bool {
	return t == LogTypeConsume || t == LogTypeTopup || t == LogTypeRefund
}

// SignedLogDelta 返回一条日志在"对账余额"意义上的带符号变动额。
// Consume 为负（扣费），Topup/Refund 为正（入账）。Manage/System 返回 0（不参与累加，
// 因为它们的 quota 字段为 0 且实际金额已直接落到 User.Quota）。
func SignedLogDelta(quota int, t int) int64 {
	switch t {
	case LogTypeConsume:
		return -int64(quota)
	case LogTypeTopup, LogTypeRefund:
		return int64(quota)
	default:
		return 0
	}
}

// GetChargeableDeltaByUser 在 [fromTs, toTs] 区间内对指定 user_id 累加 signed delta。
// 供 controller 在导出对账单时反推"期初余额"使用：pre_quota = current - sum(区间内)。
func GetChargeableDeltaByUser(userId int, fromTs int64, toTs int64) (int64, error) {
	type row struct {
		Sum int64 `gorm:"column:sum_delta"`
	}
	var r row
	err := LOG_DB.Table("logs").
		Select("COALESCE(SUM(CASE WHEN type = ? THEN -quota ELSE quota END), 0) AS sum_delta", LogTypeConsume).
		Where("user_id = ?", userId).
		Where("type IN ?", []int{LogTypeConsume, LogTypeTopup, LogTypeRefund}).
		Where("created_at >= ?", fromTs).
		Where("created_at <= ?", toTs).
		Scan(&r).Error
	if err != nil {
		return 0, err
	}
	return r.Sum, nil
}

// LogExportFilter 对账单/日志导出筛选条件，与控制台使用日志列表查询口径对齐。
type LogExportFilter struct {
	FromTs, ToTs                           int64
	ModelName, TokenName, Group, RequestID string
	LogTypes                               []int
}

// AdminLogExportFilter 管理员全站日志导出筛选，与 GetAllLogs 列表口径对齐。
type AdminLogExportFilter struct {
	LogExportFilter
	Username string
	Channel  int
}

// GetUserLogsForExport 拉取对账单导出所需的日志行（升序）。
// 内部使用 LOOP 形式而非 GORM Stream 保持实现简单；调用方负责后续流式写出。
// 上限 logExportCountLimit 条；超过则 controller 返回 400。
const logExportCountLimit = 100000

func GetUserLogsForExport(userId int, filter LogExportFilter) ([]*Log, int64, error) {
	tx := LOG_DB.Where("user_id = ?", userId)
	tx = applyLogTypesFilter(tx, filter.LogTypes)
	tx = applyBillingLogVisibility(tx, false)

	if filter.ModelName != "" {
		pattern, err := sanitizeLikePattern(filter.ModelName)
		if err != nil {
			return nil, 0, err
		}
		tx = tx.Where("model_name LIKE ? ESCAPE '!'", pattern)
	}
	if filter.TokenName != "" {
		tx = tx.Where("token_name = ?", filter.TokenName)
	}
	if filter.RequestID != "" {
		tx = tx.Where("request_id = ?", filter.RequestID)
	}
	if filter.Group != "" {
		tx = tx.Where(logGroupCol+" = ?", filter.Group)
	}
	if filter.FromTs > 0 {
		tx = tx.Where("created_at >= ?", filter.FromTs)
	}
	if filter.ToTs > 0 {
		tx = tx.Where("created_at <= ?", filter.ToTs)
	}

	var total int64
	if err := tx.Model(&Log{}).Limit(logExportCountLimit).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total > logExportCountLimit {
		return nil, total, fmt.Errorf("导出行数超过 %d 上限", logExportCountLimit)
	}

	var logs []*Log
	if err := tx.Order("id ASC").Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}

// GetAllLogsForExport 拉取管理员全站日志导出数据（升序，不分页），筛选与 GetAllLogs 一致。
func GetAllLogsForExport(filter AdminLogExportFilter) ([]*Log, int64, error) {
	tx := LOG_DB
	tx = applyLogTypesFilter(tx, filter.LogTypes)
	tx = applyBillingLogVisibility(tx, false)

	if filter.ModelName != "" {
		tx = tx.Where("logs.model_name LIKE ?", filter.ModelName)
	}
	if filter.Username != "" {
		tx = tx.Where("logs.username = ?", filter.Username)
	}
	if filter.TokenName != "" {
		tx = tx.Where("logs.token_name = ?", filter.TokenName)
	}
	if filter.RequestID != "" {
		tx = tx.Where("logs.request_id = ?", filter.RequestID)
	}
	if filter.FromTs > 0 {
		tx = tx.Where("logs.created_at >= ?", filter.FromTs)
	}
	if filter.ToTs > 0 {
		tx = tx.Where("logs.created_at <= ?", filter.ToTs)
	}
	if filter.Channel > 0 {
		tx = tx.Where("logs.channel_id = ?", filter.Channel)
	}
	if filter.Group != "" {
		tx = tx.Where("logs."+logGroupCol+" = ?", filter.Group)
	}

	var total int64
	if err := tx.Model(&Log{}).Limit(logExportCountLimit).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total > logExportCountLimit {
		return nil, total, fmt.Errorf("导出行数超过 %d 上限", logExportCountLimit)
	}

	var logs []*Log
	if err := tx.Order("logs.id ASC").Find(&logs).Error; err != nil {
		return nil, 0, err
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

func DeleteOldLog(ctx context.Context, targetTimestamp int64, limit int) (int64, error) {
	var total int64 = 0

	for {
		if nil != ctx.Err() {
			return total, ctx.Err()
		}

		result := LOG_DB.Where("created_at < ?", targetTimestamp).Limit(limit).Delete(&Log{})
		if nil != result.Error {
			return total, result.Error
		}

		total += result.RowsAffected

		if result.RowsAffected < int64(limit) {
			break
		}
	}

	return total, nil
}
