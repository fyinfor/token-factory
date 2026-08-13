package model

import (
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
)

// ── 用户指定价（用户 × 模型 维度的三折扣覆盖）────────────────────
//
// 普通用户命中覆盖时，计费改为「全局官方价 × (成本折扣 + 经营成本 + 加价折扣)」，与渠道无关。
// 代理身份（UserIsDistributor）命中覆盖时：计费不改写，自用仍按渠道成本价（加价=0）；
// 指定价只约束选路 / 可见渠道（与 Mode 一致）。
//
// 选路 / 展示约束由 Mode 决定：
//   - price_cap：排除「渠道有效单价 > 用户指定价上限」的渠道（默认）
//   - channel_list：仅允许子表勾选的渠道；手动 priority 在非智能路由路径生效

const (
	UserPricingModePriceCap     = "price_cap"
	UserPricingModeChannelList  = "channel_list"
)

// UserModelPricingOverride 用户指定价配置，(user_id, model_name) 唯一。
type UserModelPricingOverride struct {
	Id        int    `json:"id" gorm:"primaryKey"`
	UserId    int    `json:"user_id" gorm:"uniqueIndex:idx_user_model_pricing_user_model;not null"`
	ModelName string `json:"model_name" gorm:"uniqueIndex:idx_user_model_pricing_user_model;size:255;not null"`
	// Mode 选路模式：price_cap | channel_list；空串按 price_cap。
	Mode string `json:"mode" gorm:"type:varchar(32);default:'price_cap'"`
	// PriceDiscountPercent 成本折扣（百分数，如 42 表示 42%）。
	PriceDiscountPercent float64 `json:"price_discount_percent" gorm:"type:double precision;default:100"`
	// OperatingCostPercent 经营成本（百分数，如 6 表示 6%）。
	OperatingCostPercent float64 `json:"operating_cost_percent" gorm:"type:double precision;default:0"`
	// MarkupDiscountRate 加价折扣（百分数，如 2 表示 2%）。
	MarkupDiscountRate float64 `json:"markup_discount_rate" gorm:"type:double precision;default:0"`
	Enabled            bool    `json:"enabled" gorm:"default:true"`
	CreatedTime        int64   `json:"created_time" gorm:"bigint"`
	UpdatedTime        int64   `json:"updated_time" gorm:"bigint"`
}

func (UserModelPricingOverride) TableName() string {
	return "user_model_pricing_overrides"
}

// NormalizedMode 返回规范化后的 mode；空串视为 price_cap。
func (o *UserModelPricingOverride) NormalizedMode() string {
	if o == nil {
		return UserPricingModePriceCap
	}
	switch strings.TrimSpace(o.Mode) {
	case UserPricingModeChannelList:
		return UserPricingModeChannelList
	default:
		return UserPricingModePriceCap
	}
}

// TotalPercent 三折扣总和（百分数），即用户最终价相对全局官方价的比例。
func (o *UserModelPricingOverride) TotalPercent() float64 {
	if o == nil {
		return 100
	}
	return clampChannelPriceDiscountPercent(o.PriceDiscountPercent) +
		clampChannelPriceDiscountPercent(o.OperatingCostPercent) +
		clampChannelMarkupDiscountRate(o.MarkupDiscountRate)
}

// UserModelPricingChannel 用户指定价「渠道清单」模式的勾选渠道与优先级。
type UserModelPricingChannel struct {
	Id         int    `json:"id" gorm:"primaryKey"`
	UserId     int    `json:"user_id" gorm:"uniqueIndex:idx_ump_ch_user_model_ch;index:idx_ump_ch_user_model_pri;not null"`
	ModelName  string `json:"model_name" gorm:"uniqueIndex:idx_ump_ch_user_model_ch;index:idx_ump_ch_user_model_pri;size:255;not null"`
	ChannelId  int    `json:"channel_id" gorm:"uniqueIndex:idx_ump_ch_user_model_ch;not null"`
	Priority   int    `json:"priority" gorm:"index:idx_ump_ch_user_model_pri;not null;default:1"`
	CreatedTime int64 `json:"created_time" gorm:"bigint"`
	UpdatedTime int64 `json:"updated_time" gorm:"bigint"`
}

func (UserModelPricingChannel) TableName() string {
	return "user_model_pricing_channels"
}

// UserModelPricingChannelBinding API / 缓存用的轻量绑定。
type UserModelPricingChannelBinding struct {
	ChannelId int `json:"channel_id"`
	Priority  int `json:"priority"`
}

// ── 进程内缓存（计费与选路热路径均会查询）──────────────────────

type cachedUserModelPricing struct {
	Override UserModelPricingOverride
	Channels []UserModelPricingChannelBinding // 已按 priority 升序
}

const userModelPricingCacheTTL = 30 * time.Second

var (
	userModelPricingCacheMu sync.RWMutex
	userModelPricingCache   map[int]map[string]cachedUserModelPricing
	userModelPricingLoaded  time.Time
)

func loadUserModelPricingCacheLocked() {
	var rows []UserModelPricingOverride
	if err := DB.Where("enabled = ?", true).Find(&rows).Error; err != nil {
		common.SysError("load user model pricing overrides: " + err.Error())
		if userModelPricingCache == nil {
			userModelPricingCache = map[int]map[string]cachedUserModelPricing{}
		}
		userModelPricingLoaded = time.Now()
		return
	}

	var chRows []UserModelPricingChannel
	if err := DB.Order("priority ASC, id ASC").Find(&chRows).Error; err != nil {
		common.SysError("load user model pricing channels: " + err.Error())
		// 渠道清单加载失败时仍加载主表，channel_list 视为空清单（更安全：不可调用）
		chRows = nil
	}
	chByUserModel := make(map[int]map[string][]UserModelPricingChannelBinding)
	for _, r := range chRows {
		uid := r.UserId
		mn := strings.TrimSpace(r.ModelName)
		if uid <= 0 || mn == "" || r.ChannelId <= 0 {
			continue
		}
		if chByUserModel[uid] == nil {
			chByUserModel[uid] = make(map[string][]UserModelPricingChannelBinding)
		}
		chByUserModel[uid][mn] = append(chByUserModel[uid][mn], UserModelPricingChannelBinding{
			ChannelId: r.ChannelId,
			Priority:  r.Priority,
		})
	}

	next := make(map[int]map[string]cachedUserModelPricing, len(rows))
	for _, r := range rows {
		mn := strings.TrimSpace(r.ModelName)
		r.Mode = normalizeUserPricingMode(r.Mode)
		m, ok := next[r.UserId]
		if !ok {
			m = make(map[string]cachedUserModelPricing)
			next[r.UserId] = m
		}
		var channels []UserModelPricingChannelBinding
		if r.NormalizedMode() == UserPricingModeChannelList {
			channels = chByUserModel[r.UserId][mn]
			if channels == nil {
				channels = []UserModelPricingChannelBinding{}
			}
		}
		m[mn] = cachedUserModelPricing{Override: r, Channels: channels}
	}
	userModelPricingCache = next
	userModelPricingLoaded = time.Now()
}

func normalizeUserPricingMode(mode string) string {
	switch strings.TrimSpace(mode) {
	case UserPricingModeChannelList:
		return UserPricingModeChannelList
	default:
		return UserPricingModePriceCap
	}
}

func userModelPricingSnapshot() map[int]map[string]cachedUserModelPricing {
	userModelPricingCacheMu.RLock()
	if userModelPricingCache != nil && time.Since(userModelPricingLoaded) < userModelPricingCacheTTL {
		snap := userModelPricingCache
		userModelPricingCacheMu.RUnlock()
		return snap
	}
	userModelPricingCacheMu.RUnlock()

	userModelPricingCacheMu.Lock()
	defer userModelPricingCacheMu.Unlock()
	if userModelPricingCache == nil || time.Since(userModelPricingLoaded) >= userModelPricingCacheTTL {
		loadUserModelPricingCacheLocked()
	}
	return userModelPricingCache
}

// InvalidateUserModelPricingCache 配置写入后失效缓存，下次查询时重载。
func InvalidateUserModelPricingCache() {
	userModelPricingCacheMu.Lock()
	userModelPricingCache = nil
	userModelPricingCacheMu.Unlock()
}

// GetEnabledUserModelPricingOverride 查询用户对某模型的启用中指定价。
// 可传入多个候选模型名（如原始名与计价名），按顺序返回第一个命中的配置。
// 注意：命中仅表示存在启用配置（选路/展示仍可用）；计费是否改写见 UserPricingBillingApplies。
func GetEnabledUserModelPricingOverride(userId int, modelNames ...string) (UserModelPricingOverride, bool) {
	cached, ok := getEnabledUserModelPricingCached(userId, modelNames...)
	if !ok {
		return UserModelPricingOverride{}, false
	}
	return cached.Override, true
}

// UserPricingBillingApplies 指定价是否改写该用户的实扣计费。
// 代理自用保持渠道成本价（加价=0），指定价仅作选路/可见渠道上限。
func UserPricingBillingApplies(userId int) bool {
	if userId <= 0 {
		return false
	}
	u, err := GetUserById(userId, false)
	if err != nil || u == nil {
		// 查用户失败时保持「可改写计费」，与改造前行为一致，避免误放过。
		return true
	}
	return !UserIsDistributor(u)
}

// GetEnabledUserModelPricingBillingOverride 返回应用于计费改写的指定价。
// 无启用配置、或用户为代理（仅选路约束）时 ok=false。
func GetEnabledUserModelPricingBillingOverride(userId int, modelNames ...string) (UserModelPricingOverride, bool) {
	if !UserPricingBillingApplies(userId) {
		return UserModelPricingOverride{}, false
	}
	return GetEnabledUserModelPricingOverride(userId, modelNames...)
}

// GetEnabledUserModelPricingChannels 返回 channel_list 模式下的勾选渠道（已按 priority 升序）。
// 无配置、非 channel_list 或清单为空时返回 nil, false。
func GetEnabledUserModelPricingChannels(userId int, modelNames ...string) ([]UserModelPricingChannelBinding, bool) {
	cached, ok := getEnabledUserModelPricingCached(userId, modelNames...)
	if !ok {
		return nil, false
	}
	if cached.Override.NormalizedMode() != UserPricingModeChannelList {
		return nil, false
	}
	if len(cached.Channels) == 0 {
		return []UserModelPricingChannelBinding{}, true
	}
	out := make([]UserModelPricingChannelBinding, len(cached.Channels))
	copy(out, cached.Channels)
	return out, true
}

// GetEnabledUserModelPricingChannelAllowSet 返回 channel_id → priority（越小越优先）。
func GetEnabledUserModelPricingChannelAllowSet(userId int, modelNames ...string) (map[int]int, bool) {
	channels, ok := GetEnabledUserModelPricingChannels(userId, modelNames...)
	if !ok {
		return nil, false
	}
	out := make(map[int]int, len(channels))
	for _, ch := range channels {
		if ch.ChannelId <= 0 {
			continue
		}
		if prev, exists := out[ch.ChannelId]; !exists || ch.Priority < prev {
			out[ch.ChannelId] = ch.Priority
		}
	}
	return out, true
}

func getEnabledUserModelPricingCached(userId int, modelNames ...string) (cachedUserModelPricing, bool) {
	if userId <= 0 {
		return cachedUserModelPricing{}, false
	}
	snap := userModelPricingSnapshot()
	byModel, ok := snap[userId]
	if !ok || len(byModel) == 0 {
		return cachedUserModelPricing{}, false
	}
	for _, name := range modelNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if ov, ok := byModel[name]; ok {
			return ov, true
		}
	}
	return cachedUserModelPricing{}, false
}

// ListUserModelPricingChannels 管理端：列出某用户×模型的渠道绑定（含禁用覆盖）。
func ListUserModelPricingChannels(userId int, modelName string) ([]UserModelPricingChannelBinding, error) {
	modelName = strings.TrimSpace(modelName)
	if userId <= 0 || modelName == "" {
		return nil, nil
	}
	var rows []UserModelPricingChannel
	err := DB.Where("user_id = ? AND model_name = ?", userId, modelName).
		Order("priority ASC, id ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]UserModelPricingChannelBinding, 0, len(rows))
	for _, r := range rows {
		out = append(out, UserModelPricingChannelBinding{
			ChannelId: r.ChannelId,
			Priority:  r.Priority,
		})
	}
	return out, nil
}

// ListUserModelPricingChannelsByUser 批量拉取某用户全部渠道绑定：model_name → bindings。
func ListUserModelPricingChannelsByUser(userId int) (map[string][]UserModelPricingChannelBinding, error) {
	out := make(map[string][]UserModelPricingChannelBinding)
	if userId <= 0 {
		return out, nil
	}
	var rows []UserModelPricingChannel
	err := DB.Where("user_id = ?", userId).Order("model_name ASC, priority ASC, id ASC").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		mn := strings.TrimSpace(r.ModelName)
		out[mn] = append(out[mn], UserModelPricingChannelBinding{
			ChannelId: r.ChannelId,
			Priority:  r.Priority,
		})
	}
	return out, nil
}

// NormalizeUserPricingChannelBindings 去重并按提交顺序重编 priority（从 1 起）。
func NormalizeUserPricingChannelBindings(channels []UserModelPricingChannelBinding) []UserModelPricingChannelBinding {
	if len(channels) == 0 {
		return nil
	}
	seen := make(map[int]struct{}, len(channels))
	out := make([]UserModelPricingChannelBinding, 0, len(channels))
	for _, ch := range channels {
		if ch.ChannelId <= 0 {
			continue
		}
		if _, ok := seen[ch.ChannelId]; ok {
			continue
		}
		seen[ch.ChannelId] = struct{}{}
		out = append(out, UserModelPricingChannelBinding{ChannelId: ch.ChannelId})
	}
	for i := range out {
		out[i].Priority = i + 1
	}
	return out
}

// ── CRUD ──────────────────────────────────────────────────────

// ListUserModelPricingOverrides 按可选条件列出配置（含禁用项，供管理界面展示）。
func ListUserModelPricingOverrides(userId int, modelName string) ([]UserModelPricingOverride, error) {
	var rows []UserModelPricingOverride
	tx := DB.Model(&UserModelPricingOverride{})
	if userId > 0 {
		tx = tx.Where("user_id = ?", userId)
	}
	if modelName = strings.TrimSpace(modelName); modelName != "" {
		tx = tx.Where("model_name LIKE ?", "%"+modelName+"%")
	}
	err := tx.Order("id DESC").Find(&rows).Error
	for i := range rows {
		rows[i].Mode = normalizeUserPricingMode(rows[i].Mode)
	}
	return rows, err
}

// UpsertUserModelPricingOverride 按 (user_id, model_name) 创建或更新配置（不含渠道清单）。
// 若目标 mode 为 channel_list，请使用 UpsertUserModelPricingOverrideWithChannels。
func UpsertUserModelPricingOverride(ov *UserModelPricingOverride) (*UserModelPricingOverride, error) {
	return UpsertUserModelPricingOverrideWithChannels(ov, nil)
}

// UpsertUserModelPricingOverrideWithChannels 创建或更新指定价，并在 channel_list 模式下同步渠道清单。
// price_cap 模式下会清空该用户×模型的渠道绑定。
func UpsertUserModelPricingOverrideWithChannels(ov *UserModelPricingOverride, channels []UserModelPricingChannelBinding) (*UserModelPricingOverride, error) {
	if ov == nil {
		return nil, errors.New("override is nil")
	}
	ov.ModelName = strings.TrimSpace(ov.ModelName)
	if ov.UserId <= 0 || ov.ModelName == "" {
		return nil, errors.New("user_id 和 model_name 不能为空")
	}
	ov.Mode = normalizeUserPricingMode(ov.Mode)
	bindings := NormalizeUserPricingChannelBindings(channels)
	if ov.Mode == UserPricingModeChannelList && len(bindings) == 0 {
		return nil, errors.New("渠道清单模式至少勾选一个渠道")
	}

	now := time.Now().Unix()
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}

	var existing UserModelPricingOverride
	result := tx.Where("user_id = ? AND model_name = ?", ov.UserId, ov.ModelName).First(&existing)
	if result.Error != nil {
		ov.Id = 0
		ov.CreatedTime = now
		ov.UpdatedTime = now
		if err := tx.Create(ov).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
		existing = *ov
	} else {
		updates := map[string]interface{}{
			"mode":                   ov.Mode,
			"price_discount_percent": ov.PriceDiscountPercent,
			"operating_cost_percent": ov.OperatingCostPercent,
			"markup_discount_rate":   ov.MarkupDiscountRate,
			"enabled":                ov.Enabled,
			"updated_time":           now,
		}
		if err := tx.Model(&existing).Updates(updates).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
		if err := tx.First(&existing, existing.Id).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	// 先清再写，保证与 mode 一致。
	if err := tx.Where("user_id = ? AND model_name = ?", existing.UserId, existing.ModelName).
		Delete(&UserModelPricingChannel{}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	if existing.NormalizedMode() == UserPricingModeChannelList && len(bindings) > 0 {
		rows := make([]UserModelPricingChannel, 0, len(bindings))
		for _, b := range bindings {
			rows = append(rows, UserModelPricingChannel{
				UserId:      existing.UserId,
				ModelName:   existing.ModelName,
				ChannelId:   b.ChannelId,
				Priority:    b.Priority,
				CreatedTime: now,
				UpdatedTime: now,
			})
		}
		if err := tx.CreateInBatches(rows, 100).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}
	InvalidateUserModelPricingCache()
	existing.Mode = normalizeUserPricingMode(existing.Mode)
	return &existing, nil
}

// DeleteUserModelPricingOverrideById 删除一条配置及其渠道绑定。
func DeleteUserModelPricingOverrideById(id int) error {
	if id <= 0 {
		return errors.New("invalid id")
	}
	var existing UserModelPricingOverride
	if err := DB.First(&existing, id).Error; err != nil {
		return err
	}
	tx := DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	if err := tx.Where("user_id = ? AND model_name = ?", existing.UserId, existing.ModelName).
		Delete(&UserModelPricingChannel{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Delete(&UserModelPricingOverride{}, id).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Commit().Error; err != nil {
		return err
	}
	InvalidateUserModelPricingCache()
	return nil
}

// DeleteUserModelPricingOverridesByUserId 删除某用户下全部指定价配置及渠道绑定。
func DeleteUserModelPricingOverridesByUserId(userId int) (int64, error) {
	if userId <= 0 {
		return 0, errors.New("invalid user_id")
	}
	tx := DB.Begin()
	if tx.Error != nil {
		return 0, tx.Error
	}
	if err := tx.Where("user_id = ?", userId).Delete(&UserModelPricingChannel{}).Error; err != nil {
		tx.Rollback()
		return 0, err
	}
	result := tx.Where("user_id = ?", userId).Delete(&UserModelPricingOverride{})
	if result.Error != nil {
		tx.Rollback()
		return 0, result.Error
	}
	if err := tx.Commit().Error; err != nil {
		return 0, err
	}
	if result.RowsAffected > 0 {
		InvalidateUserModelPricingCache()
	}
	return result.RowsAffected, nil
}

// UserModelPricingUserSummary 已配置指定价的用户汇总（管理页按用户筛选用）。
type UserModelPricingUserSummary struct {
	UserId       int    `json:"user_id"`
	Username     string `json:"username"`
	ModelCount   int64  `json:"model_count"`
	EnabledCount int64  `json:"enabled_count"`
}

// ListUsersWithModelPricing 列出所有已有指定价配置的用户（按最近更新倒序）。
func ListUsersWithModelPricing() ([]UserModelPricingUserSummary, error) {
	type row struct {
		UserId       int
		ModelCount   int64
		EnabledCount int64
		MaxUpdated   int64
	}
	var rows []row
	err := DB.Model(&UserModelPricingOverride{}).
		Select("user_id, COUNT(*) AS model_count, SUM(CASE WHEN enabled THEN 1 ELSE 0 END) AS enabled_count, MAX(updated_time) AS max_updated").
		Group("user_id").
		Order("max_updated DESC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	ids := make([]int, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.UserId)
	}
	names := GetUsernamesByIds(ids)
	out := make([]UserModelPricingUserSummary, 0, len(rows))
	for _, r := range rows {
		out = append(out, UserModelPricingUserSummary{
			UserId:       r.UserId,
			Username:     names[r.UserId],
			ModelCount:   r.ModelCount,
			EnabledCount: r.EnabledCount,
		})
	}
	return out, nil
}

// ListImportablePricedModels 返回可一键导入的模型名：当前启用能力中且已配置展示定价的模型。
func ListImportablePricedModels() []string {
	enabled := GetEnabledModels()
	out := make([]string, 0, len(enabled))
	seen := make(map[string]struct{}, len(enabled))
	for _, name := range enabled {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		if !ModelHasDisplayConfiguredPricing(name) {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}

// BulkUpsertUserModelPricingOverrides 按统一三折扣批量为某用户 upsert 多个模型指定价。
// 返回新建条数、更新条数。
func BulkUpsertUserModelPricingOverrides(userId int, modelNames []string, priceDisc, operating, markup float64, enabled bool) (created int, updated int, err error) {
	if userId <= 0 {
		return 0, 0, errors.New("invalid user_id")
	}
	names := make([]string, 0, len(modelNames))
	seen := make(map[string]struct{}, len(modelNames))
	for _, n := range modelNames {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		names = append(names, n)
	}
	if len(names) == 0 {
		return 0, 0, nil
	}
	rows := make([]UserModelPricingOverride, 0, len(names))
	for _, name := range names {
		rows = append(rows, UserModelPricingOverride{
			UserId:               userId,
			ModelName:            name,
			Mode:                 UserPricingModePriceCap,
			PriceDiscountPercent: priceDisc,
			OperatingCostPercent: operating,
			MarkupDiscountRate:   markup,
			Enabled:              enabled,
		})
	}
	return BulkUpsertUserModelPricingOverrideRows(rows)
}

// BulkUpsertUserModelPricingOverrideRows 批量 upsert，每条可带各自的三折扣（用于从渠道当前定价导入）。
// 导入默认写入 price_cap；已存在 channel_list 的行仅覆盖三折扣与 enabled，保留 mode 与渠道清单。
func BulkUpsertUserModelPricingOverrideRows(rows []UserModelPricingOverride) (created int, updated int, err error) {
	if len(rows) == 0 {
		return 0, 0, nil
	}
	byUser := make(map[int][]UserModelPricingOverride)
	for _, r := range rows {
		r.ModelName = strings.TrimSpace(r.ModelName)
		if r.UserId <= 0 || r.ModelName == "" {
			continue
		}
		if strings.TrimSpace(r.Mode) == "" {
			r.Mode = UserPricingModePriceCap
		}
		byUser[r.UserId] = append(byUser[r.UserId], r)
	}
	if len(byUser) == 0 {
		return 0, 0, nil
	}

	now := time.Now().Unix()
	tx := DB.Begin()
	if tx.Error != nil {
		return 0, 0, tx.Error
	}
	totalCreated, totalUpdated := 0, 0
	for userId, userRows := range byUser {
		names := make([]string, 0, len(userRows))
		rowByName := make(map[string]UserModelPricingOverride, len(userRows))
		for _, r := range userRows {
			if _, ok := rowByName[r.ModelName]; !ok {
				names = append(names, r.ModelName)
			}
			rowByName[r.ModelName] = r
		}
		var existing []UserModelPricingOverride
		if err = tx.Where("user_id = ? AND model_name IN ?", userId, names).Find(&existing).Error; err != nil {
			tx.Rollback()
			return 0, 0, err
		}
		existByName := make(map[string]UserModelPricingOverride, len(existing))
		for _, e := range existing {
			existByName[e.ModelName] = e
		}

		toCreate := make([]UserModelPricingOverride, 0)
		for _, name := range names {
			src := rowByName[name]
			if e, ok := existByName[name]; ok {
				// 不覆盖已有 mode（避免一键导入冲掉渠道清单配置）
				if err = tx.Model(&UserModelPricingOverride{}).Where("id = ?", e.Id).Updates(map[string]interface{}{
					"price_discount_percent": src.PriceDiscountPercent,
					"operating_cost_percent": src.OperatingCostPercent,
					"markup_discount_rate":   src.MarkupDiscountRate,
					"enabled":                src.Enabled,
					"updated_time":           now,
				}).Error; err != nil {
					tx.Rollback()
					return 0, 0, err
				}
				totalUpdated++
				continue
			}
			toCreate = append(toCreate, UserModelPricingOverride{
				UserId:               userId,
				ModelName:            name,
				Mode:                 UserPricingModePriceCap,
				PriceDiscountPercent: src.PriceDiscountPercent,
				OperatingCostPercent: src.OperatingCostPercent,
				MarkupDiscountRate:   src.MarkupDiscountRate,
				Enabled:              src.Enabled,
				CreatedTime:          now,
				UpdatedTime:          now,
			})
		}
		if len(toCreate) > 0 {
			if err = tx.CreateInBatches(toCreate, 100).Error; err != nil {
				tx.Rollback()
				return 0, 0, err
			}
			totalCreated += len(toCreate)
		}
	}
	if err = tx.Commit().Error; err != nil {
		return 0, 0, err
	}
	InvalidateUserModelPricingCache()
	return totalCreated, totalUpdated, nil
}

// GetUsernamesByIds 批量查询用户名（管理界面展示用）。
func GetUsernamesByIds(ids []int) map[int]string {
	out := make(map[int]string, len(ids))
	if len(ids) == 0 {
		return out
	}
	var rows []struct {
		Id       int
		Username string
	}
	if err := DB.Model(&User{}).Select("id, username").Where("id IN ?", ids).Scan(&rows).Error; err != nil {
		common.SysError("get usernames by ids: " + err.Error())
		return out
	}
	for _, r := range rows {
		out[r.Id] = r.Username
	}
	return out
}

// GetEnabledChannelIDsByModel 返回启用能力中支持该模型的全部渠道 ID（用于指定价预览）。
func GetEnabledChannelIDsByModel(modelName string) []int {
	var ids []int
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return ids
	}
	err := DB.Model(&Ability{}).
		Where("model = ? AND enabled = ?", modelName, true).
		Distinct("channel_id").
		Pluck("channel_id", &ids).Error
	if err != nil {
		common.SysError("list channels by model: " + err.Error())
	}
	sort.Ints(ids)
	return ids
}
