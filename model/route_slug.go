package model

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// channelNoRoutePattern 与旧版三段式里 channel_no（c1、c2…）同形；route_slug 禁止使用该形态以免解析歧义。
var channelNoRoutePattern = regexp.MustCompile(`^c\d+$`)

const routeSlugLookupCacheTTL = 30 * time.Second

type routeSlugLookupCacheEntry struct {
	ch        *Channel
	expiresAt time.Time
}

var routeSlugLookupCache sync.Map // slug -> routeSlugLookupCacheEntry

// InvalidateRouteSlugLookupCache 在渠道 route_slug / status / models 变更后清除缓存。
// slug 为空时清空全部。
func InvalidateRouteSlugLookupCache(slug string) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		routeSlugLookupCache.Range(func(key, _ any) bool {
			routeSlugLookupCache.Delete(key)
			return true
		})
		return
	}
	routeSlugLookupCache.Delete(slug)
}

// DefaultRouteSlugFromChannelID 返回渠道默认全局路由后缀（与 channels.id 一一对应）。
// 前缀 "u" 避免与旧 channel_no 段 c\d+ 混淆。
func DefaultRouteSlugFromChannelID(id int64) string {
	return "u" + EncodeBase62(id)
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

// IsLegacyChannelNoSuffix 是否为旧版 channel_no 形态（c1、c23…）。
// 不能登记为 route_slug，但仍可能出现在 {model}/{suffix} 调用中，需按「指定后缀」解析以便剥后缀/回落。
func IsLegacyChannelNoSuffix(s string) bool {
	return channelNoRoutePattern.MatchString(strings.TrimSpace(s))
}

// IsRouteSuffixCandidate 判断最后一段是否应按「渠道路由后缀」解析（含合法 route_slug 与误用的 cN）。
func IsRouteSuffixCandidate(s string) bool {
	return IsValidRouteSlug(s) || IsLegacyChannelNoSuffix(s)
}

// ResolveChannelIDByRouteSlugAndModel 按 route_slug 查找已启用渠道，并校验 models 列表包含 modelName。
// 未命中、已禁用或模型不在列表中时返回 0。
func ResolveChannelIDByRouteSlugAndModel(slug, modelName string) int {
	ch := LookupChannelByRouteSlug(slug)
	if ch == nil {
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

// LookupChannelByRouteSlug 按 route_slug 查找渠道（不限启用状态）；未找到返回 nil。
// 命中结果带短 TTL 内存缓存，避免 {model}/{route_slug} 热路径每请求打库。
func LookupChannelByRouteSlug(slug string) *Channel {
	slug = strings.TrimSpace(slug)
	if slug == "" || !IsValidRouteSlug(slug) {
		return nil
	}
	now := time.Now()
	if raw, ok := routeSlugLookupCache.Load(slug); ok {
		if entry, ok := raw.(routeSlugLookupCacheEntry); ok && now.Before(entry.expiresAt) {
			if entry.ch == nil {
				return nil
			}
			// 返回浅拷贝，避免调用方改写缓存对象
			cp := *entry.ch
			return &cp
		}
	}

	var ch Channel
	err := DB.Select("id", "models", "status", "route_slug").Where("route_slug = ?", slug).First(&ch).Error
	if err != nil {
		routeSlugLookupCache.Store(slug, routeSlugLookupCacheEntry{ch: nil, expiresAt: now.Add(routeSlugLookupCacheTTL)})
		return nil
	}
	cp := ch
	routeSlugLookupCache.Store(slug, routeSlugLookupCacheEntry{ch: &cp, expiresAt: now.Add(routeSlugLookupCacheTTL)})
	out := ch
	return &out
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
