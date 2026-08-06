package model

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
)

// ── 用户指定价（用户 × 模型 维度的三折扣覆盖）────────────────────
//
// 命中覆盖时，该用户调用该模型的计费改为「全局官方价 × (成本折扣 + 经营成本 + 加价折扣)」，
// 与渠道无关；同时选路层会排除「渠道有效单价 > 用户指定价上限」的渠道。

// UserModelPricingOverride 用户指定价配置，(user_id, model_name) 唯一。
type UserModelPricingOverride struct {
	Id        int    `json:"id" gorm:"primaryKey"`
	UserId    int    `json:"user_id" gorm:"uniqueIndex:idx_user_model_pricing_user_model;not null"`
	ModelName string `json:"model_name" gorm:"uniqueIndex:idx_user_model_pricing_user_model;size:255;not null"`
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

// TotalPercent 三折扣总和（百分数），即用户最终价相对全局官方价的比例。
func (o *UserModelPricingOverride) TotalPercent() float64 {
	if o == nil {
		return 100
	}
	return clampChannelPriceDiscountPercent(o.PriceDiscountPercent) +
		clampChannelPriceDiscountPercent(o.OperatingCostPercent) +
		clampChannelMarkupDiscountRate(o.MarkupDiscountRate)
}

// ── 进程内缓存（计费与选路热路径均会查询）──────────────────────

const userModelPricingCacheTTL = 30 * time.Second

var (
	userModelPricingCacheMu sync.RWMutex
	userModelPricingCache   map[int]map[string]UserModelPricingOverride
	userModelPricingLoaded  time.Time
)

func loadUserModelPricingCacheLocked() {
	var rows []UserModelPricingOverride
	if err := DB.Where("enabled = ?", true).Find(&rows).Error; err != nil {
		common.SysError("load user model pricing overrides: " + err.Error())
		// 加载失败时保留旧缓存，避免瞬时全量放开。
		if userModelPricingCache == nil {
			userModelPricingCache = map[int]map[string]UserModelPricingOverride{}
		}
		userModelPricingLoaded = time.Now()
		return
	}
	next := make(map[int]map[string]UserModelPricingOverride, len(rows))
	for _, r := range rows {
		m, ok := next[r.UserId]
		if !ok {
			m = make(map[string]UserModelPricingOverride)
			next[r.UserId] = m
		}
		m[strings.TrimSpace(r.ModelName)] = r
	}
	userModelPricingCache = next
	userModelPricingLoaded = time.Now()
}

func userModelPricingSnapshot() map[int]map[string]UserModelPricingOverride {
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
func GetEnabledUserModelPricingOverride(userId int, modelNames ...string) (UserModelPricingOverride, bool) {
	if userId <= 0 {
		return UserModelPricingOverride{}, false
	}
	snap := userModelPricingSnapshot()
	byModel, ok := snap[userId]
	if !ok || len(byModel) == 0 {
		return UserModelPricingOverride{}, false
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
	return UserModelPricingOverride{}, false
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
	return rows, err
}

// UpsertUserModelPricingOverride 按 (user_id, model_name) 创建或更新配置。
func UpsertUserModelPricingOverride(ov *UserModelPricingOverride) (*UserModelPricingOverride, error) {
	if ov == nil {
		return nil, errors.New("override is nil")
	}
	ov.ModelName = strings.TrimSpace(ov.ModelName)
	if ov.UserId <= 0 || ov.ModelName == "" {
		return nil, errors.New("user_id 和 model_name 不能为空")
	}
	now := time.Now().Unix()
	var existing UserModelPricingOverride
	result := DB.Where("user_id = ? AND model_name = ?", ov.UserId, ov.ModelName).First(&existing)
	if result.Error != nil {
		ov.Id = 0
		ov.CreatedTime = now
		ov.UpdatedTime = now
		if err := DB.Create(ov).Error; err != nil {
			return nil, err
		}
		InvalidateUserModelPricingCache()
		return ov, nil
	}
	updates := map[string]interface{}{
		"price_discount_percent": ov.PriceDiscountPercent,
		"operating_cost_percent": ov.OperatingCostPercent,
		"markup_discount_rate":   ov.MarkupDiscountRate,
		"enabled":                ov.Enabled,
		"updated_time":           now,
	}
	if err := DB.Model(&existing).Updates(updates).Error; err != nil {
		return nil, err
	}
	InvalidateUserModelPricingCache()
	_ = DB.First(&existing, existing.Id).Error
	return &existing, nil
}

// DeleteUserModelPricingOverrideById 删除一条配置。
func DeleteUserModelPricingOverrideById(id int) error {
	if id <= 0 {
		return errors.New("invalid id")
	}
	err := DB.Delete(&UserModelPricingOverride{}, id).Error
	if err == nil {
		InvalidateUserModelPricingCache()
	}
	return err
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
	return ids
}
