package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// ChannelModelRouteIndex 存储「模型名 + 路由索引 → 渠道 ID」的全局唯一映射。
//
// 路由调用格式：{model_name}/{route_index}，例如：
//
//	claude-opus-4-6/0   —— 该模型下第 1 个渠道
//	claude-opus-4-6/1   —— 第 2 个
//	claude-opus-4-6/A   —— 第 11 个（base-62：0-9 A-Z a-z）
//
// 索引对同一模型名全局唯一；按渠道创建顺序递增分配，不随渠道删除而复用。
type ChannelModelRouteIndex struct {
	ID         int64  `gorm:"primaryKey;autoIncrement"`
	ModelName  string `gorm:"type:varchar(255);not null;uniqueIndex:uq_cmri_model_route;uniqueIndex:uq_cmri_channel_model"`
	RouteIndex string `gorm:"type:varchar(16);not null;uniqueIndex:uq_cmri_model_route"`
	ChannelID  int    `gorm:"not null;uniqueIndex:uq_cmri_channel_model;index:idx_cmri_channel"`
}

// base62Chars 是路由索引的字母表：数字 → 大写字母 → 小写字母（共 62 个字符）。
const base62Chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// EncodeBase62 将非负整数编码为 base-62 字符串（"0" 对应 0，"A" 对应 10，"a" 对应 36…）。
// 导出版本供外部包（如 controller）使用；包内仍使用私有版本 encodeBase62。
func EncodeBase62(n int64) string { return encodeBase62(n) }

// encodeBase62 将非负整数编码为 base-62 字符串（"0" 对应 0，"A" 对应 10，"a" 对应 36…）。
func encodeBase62(n int64) string {
	if n == 0 {
		return "0"
	}
	base := int64(len(base62Chars))
	var buf []byte
	for n > 0 {
		buf = append([]byte{base62Chars[n%base]}, buf...)
		n /= base
	}
	return string(buf)
}

// IsValidRouteIndex 判断字符串是否为合法的路由索引：非空且全为 base-62 字符（0-9 A-Z a-z），
// 不含连字符、点号等特殊符号，以此与真实模型名（通常含 - 或 .）区分。
func IsValidRouteIndex(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')) {
			return false
		}
	}
	return true
}

// FindChannelIDByModelAndRouteIndex 根据模型名 + 路由索引查找启用状态的渠道 ID。
// 若索引不存在或对应渠道已禁用，返回 0 和非空 error。
func FindChannelIDByModelAndRouteIndex(modelName, routeIndex string) (int, error) {
	var entry ChannelModelRouteIndex
	err := DB.Where("model_name = ? AND route_index = ?", modelName, routeIndex).
		First(&entry).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, fmt.Errorf("路由 %s/%s 不存在", modelName, routeIndex)
		}
		return 0, err
	}
	var ch Channel
	if err := DB.Select("id, status").Where("id = ?", entry.ChannelID).First(&ch).Error; err != nil {
		return 0, fmt.Errorf("路由 %s/%s 对应渠道已不存在", modelName, routeIndex)
	}
	if ch.Status != common.ChannelStatusEnabled {
		return 0, fmt.Errorf("路由 %s/%s 对应渠道已禁用", modelName, routeIndex)
	}
	return entry.ChannelID, nil
}

// GetRouteIndexByChannelAndModel 返回某渠道某模型的路由索引；不存在时返回空字符串。
func GetRouteIndexByChannelAndModel(channelID int, modelName string) string {
	var entry ChannelModelRouteIndex
	if err := DB.Select("route_index").
		Where("channel_id = ? AND model_name = ?", channelID, modelName).
		First(&entry).Error; err != nil {
		return ""
	}
	return entry.RouteIndex
}

// GetRouteIndicesByChannels 批量返回一组渠道的 channel_id → (model_name → route_index) 映射，
// 用于定价接口一次性填充所有渠道的路由索引，避免 N+1 查询。
func GetRouteIndicesByChannels(channelIDs []int) map[int]map[string]string {
	if len(channelIDs) == 0 {
		return nil
	}
	var entries []ChannelModelRouteIndex
	if err := DB.Select("channel_id, model_name, route_index").
		Where("channel_id IN ?", channelIDs).
		Find(&entries).Error; err != nil {
		return nil
	}
	result := make(map[int]map[string]string, len(channelIDs))
	for _, e := range entries {
		if result[e.ChannelID] == nil {
			result[e.ChannelID] = make(map[string]string)
		}
		result[e.ChannelID][e.ModelName] = e.RouteIndex
	}
	return result
}

// AssignChannelModelRouteIndices 为渠道的各个模型分配路由索引（幂等：已有索引的模型跳过）。
// 此函数属于 best-effort，失败时仅记录日志，不中断主流程。
func AssignChannelModelRouteIndices(channelID int, models []string) {
	for _, raw := range models {
		m := strings.TrimSpace(raw)
		if m == "" {
			continue
		}
		assignSingleModelRouteIndex(channelID, m)
	}
}

func assignSingleModelRouteIndex(channelID int, modelName string) {
	// 幂等检查：(channel_id, model_name) 已有记录则直接返回
	var existing ChannelModelRouteIndex
	if err := DB.Where("channel_id = ? AND model_name = ?", channelID, modelName).
		First(&existing).Error; err == nil {
		return
	}

	// 从当前记录数出发，依次尝试分配下一个可用索引
	var count int64
	DB.Model(&ChannelModelRouteIndex{}).Where("model_name = ?", modelName).Count(&count)

	for attempt := int64(0); attempt < 1024; attempt++ {
		idx := encodeBase62(count + attempt)
		entry := ChannelModelRouteIndex{
			ModelName:  modelName,
			RouteIndex: idx,
			ChannelID:  channelID,
		}
		if err := DB.Create(&entry).Error; err == nil {
			return
		}
		// 冲突：可能是 route_index 已被并发写入，或 (channel_id, model_name) 已存在
		if DB.Where("channel_id = ? AND model_name = ?", channelID, modelName).
			First(&existing).Error == nil {
			return // 并发下已由其他 goroutine 完成分配
		}
		// route_index 被其他渠道抢占，继续尝试下一个
	}
	common.SysLog(fmt.Sprintf(
		"channel_model_route_index: channel=%d model=%s: failed to assign index after 1024 attempts",
		channelID, modelName,
	))
}

// RemoveChannelModelRouteIndicesForModels 删除渠道指定模型的路由索引（用于模型列表缩减时的清理）。
func RemoveChannelModelRouteIndicesForModels(channelID int, models []string) {
	if channelID <= 0 || len(models) == 0 {
		return
	}
	trimmed := make([]string, 0, len(models))
	for _, m := range models {
		if t := strings.TrimSpace(m); t != "" {
			trimmed = append(trimmed, t)
		}
	}
	if len(trimmed) == 0 {
		return
	}
	if err := DB.Where("channel_id = ? AND model_name IN ?", channelID, trimmed).
		Delete(&ChannelModelRouteIndex{}).Error; err != nil {
		common.SysLog(fmt.Sprintf(
			"channel_model_route_index: remove for channel=%d models=%v: %v", channelID, trimmed, err,
		))
	}
}

// RemoveAllChannelModelRouteIndices 删除某渠道的全部路由索引（用于渠道删除时的清理）。
func RemoveAllChannelModelRouteIndices(channelID int) {
	if channelID <= 0 {
		return
	}
	if err := DB.Where("channel_id = ?", channelID).
		Delete(&ChannelModelRouteIndex{}).Error; err != nil {
		common.SysLog(fmt.Sprintf(
			"channel_model_route_index: remove all for channel=%d: %v", channelID, err,
		))
	}
}

// RemoveAllChannelModelRouteIndicesBatch 批量删除多个渠道的全部路由索引。
func RemoveAllChannelModelRouteIndicesBatch(channelIDs []int) {
	if len(channelIDs) == 0 {
		return
	}
	if err := DB.Where("channel_id IN ?", channelIDs).
		Delete(&ChannelModelRouteIndex{}).Error; err != nil {
		common.SysLog(fmt.Sprintf(
			"channel_model_route_index: batch remove for channels=%v: %v", channelIDs, err,
		))
	}
}

// SyncChannelModelRouteIndices 在渠道模型列表发生变更时同步路由索引：
// 为新增模型分配索引，删除移除模型的索引。
func SyncChannelModelRouteIndices(channelID int, oldModelsCSV, newModelsCSV string) {
	oldSet := parseModelSet(oldModelsCSV)
	newList, newSet := parseModelList(newModelsCSV)

	var removed []string
	for m := range oldSet {
		if _, ok := newSet[m]; !ok {
			removed = append(removed, m)
		}
	}
	var added []string
	for _, m := range newList {
		if _, ok := oldSet[m]; !ok {
			added = append(added, m)
		}
	}
	if len(removed) > 0 {
		RemoveChannelModelRouteIndicesForModels(channelID, removed)
	}
	if len(added) > 0 {
		AssignChannelModelRouteIndices(channelID, added)
	}
}

func parseModelSet(csv string) map[string]struct{} {
	set := make(map[string]struct{})
	for _, m := range strings.Split(csv, ",") {
		if t := strings.TrimSpace(m); t != "" {
			set[t] = struct{}{}
		}
	}
	return set
}

func parseModelList(csv string) ([]string, map[string]struct{}) {
	var list []string
	set := make(map[string]struct{})
	for _, m := range strings.Split(csv, ",") {
		if t := strings.TrimSpace(m); t != "" {
			if _, ok := set[t]; !ok {
				list = append(list, t)
				set[t] = struct{}{}
			}
		}
	}
	return list, set
}

// BackfillChannelModelRouteIndices 为历史渠道补全路由索引（启动时幂等执行）。
func BackfillChannelModelRouteIndices() error {
	type row struct {
		ID     int
		Models string
	}
	var rows []row
	if err := DB.Model(&Channel{}).Select("id, models").
		Where("status > 0 AND models != ''").
		Order("id asc").Scan(&rows).Error; err != nil {
		return err
	}
	for _, r := range rows {
		if r.ID <= 0 {
			continue
		}
		models := strings.Split(r.Models, ",")
		AssignChannelModelRouteIndices(r.ID, models)
	}
	return nil
}
