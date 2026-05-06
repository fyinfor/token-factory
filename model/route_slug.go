package model

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const base62Chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// channelNoRoutePattern 与旧版三段式里 channel_no（c1、c2…）同形；route_slug 禁止使用该形态以免解析歧义。
var channelNoRoutePattern = regexp.MustCompile(`^c\d+$`)

// EncodeBase62 将非负整数编码为 base-62（供渠道名后缀、默认 route_slug 等使用）。
func EncodeBase62(n int64) string { return encodeBase62(n) }

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

// DefaultRouteSlugFromChannelID 返回渠道默认全局路由后缀（与 channels.id 一一对应）。
// 前缀 "u" 避免与旧 channel_no 段 c\d+ 混淆。
func DefaultRouteSlugFromChannelID(id int64) string {
	return "u" + encodeBase62(id)
}

// IsValidRouteSlug 判断字符串是否可作为全局 route_slug：2～32 位 base62，且不能为 c+数字（旧 channel_no 形态）。
func IsValidRouteSlug(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) < 2 || len(s) > 32 {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')) {
			return false
		}
	}
	if channelNoRoutePattern.MatchString(s) {
		return false
	}
	return true
}

// ResolveChannelIDByRouteSlugAndModel 按 route_slug 查找已启用渠道，并校验 models 列表包含 modelName。
// 未命中、已禁用或模型不在列表中时返回 0（供分发器静默降级为普通路由）。
func ResolveChannelIDByRouteSlugAndModel(slug, modelName string) int {
	slug = strings.TrimSpace(slug)
	if slug == "" || !IsValidRouteSlug(slug) {
		return 0
	}
	var ch Channel
	err := DB.Select("id", "models", "status").Where("route_slug = ?", slug).First(&ch).Error
	if err != nil {
		return 0
	}
	if ch.Status != common.ChannelStatusEnabled {
		return 0
	}
	if !ChannelModelsRawContains(ch.Models, modelName) {
		return 0
	}
	return ch.Id
}

// GetRouteSlugsByChannelIDs 批量返回 channel_id → route_slug（定价等场景）。
func GetRouteSlugsByChannelIDs(channelIDs []int) map[int]string {
	if len(channelIDs) == 0 {
		return nil
	}
	var rows []Channel
	if err := DB.Select("id", "route_slug").Where("id IN ?", channelIDs).Find(&rows).Error; err != nil {
		return nil
	}
	out := make(map[int]string, len(rows))
	for i := range rows {
		s := strings.TrimSpace(rows[i].RouteSlug)
		if s != "" {
			out[rows[i].Id] = s
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// assignRouteSlugInTx 在事务内为新建渠道写入 route_slug（空则按 id 生成；非空则校验格式与唯一性）。
func assignRouteSlugInTx(tx *gorm.DB, channelID int, requested string) (assigned string, err error) {
	if channelID <= 0 {
		return "", nil
	}
	req := strings.TrimSpace(requested)
	slug := req
	if slug == "" {
		slug = DefaultRouteSlugFromChannelID(int64(channelID))
	} else if !IsValidRouteSlug(slug) {
		return "", fmt.Errorf("route_slug 无效")
	}
	var cnt int64
	if err := tx.Model(&Channel{}).Where("route_slug = ? AND id <> ?", slug, channelID).Count(&cnt).Error; err != nil {
		return "", err
	}
	if cnt > 0 {
		return "", fmt.Errorf("route_slug 已被占用")
	}
	if err := tx.Model(&Channel{}).Where("id = ?", channelID).Update("route_slug", slug).Error; err != nil {
		return "", err
	}
	return slug, nil
}

// BackfillChannelRouteSlugs 为缺少 route_slug 的渠道写入默认值（幂等）。
func BackfillChannelRouteSlugs() error {
	if DB == nil || DB.Migrator() == nil {
		return nil
	}
	if !DB.Migrator().HasColumn(&Channel{}, "route_slug") {
		return nil
	}
	var ids []int
	if err := DB.Model(&Channel{}).Where("route_slug IS NULL OR route_slug = ?", "").Pluck("id", &ids).Error; err != nil {
		return err
	}
	for _, id := range ids {
		slug := DefaultRouteSlugFromChannelID(int64(id))
		if err := DB.Model(&Channel{}).Where("id = ?", id).Update("route_slug", slug).Error; err != nil {
			return fmt.Errorf("backfill route_slug channel_id=%d: %w", id, err)
		}
	}
	return nil
}

// ensureRouteSlugLookupIndex 创建 route_slug 普通索引（非唯一：批量插入时须先落库再逐行赋值 slug，避免空串唯一冲突）。
func ensureRouteSlugLookupIndex() error {
	sql := "CREATE INDEX IF NOT EXISTS idx_channels_route_slug ON channels (route_slug)"
	if common.UsingMySQL {
		sql = "CREATE INDEX idx_channels_route_slug ON channels (route_slug)"
	}
	err := DB.Exec(sql).Error
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "duplicate") || strings.Contains(msg, "already exists") || strings.Contains(msg, "exist") {
		return nil
	}
	return fmt.Errorf("ensure route_slug lookup index: %w", err)
}

// MigrateChannelRouteSlugAndDropLegacy 删除未上线的旧 route_index 表、补全 route_slug、建查询索引。
func MigrateChannelRouteSlugAndDropLegacy() error {
	if DB == nil || DB.Migrator() == nil {
		return nil
	}
	if DB.Migrator().HasTable("channel_model_route_indices") {
		if err := DB.Migrator().DropTable("channel_model_route_indices"); err != nil {
			return fmt.Errorf("drop channel_model_route_indices: %w", err)
		}
	}
	if err := BackfillChannelRouteSlugs(); err != nil {
		return err
	}
	return ensureRouteSlugLookupIndex()
}
