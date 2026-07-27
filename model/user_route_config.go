package model

import (
	"fmt"
	"strings"
	"time"
)

// ── 用户级路由策略（单站点）──────────────────────────────────────
//
// 每个用户可独立配置路由策略，仅影响自己的模型路由调用。
// 用户级配置优先于全局默认；Mode 为空字符串时表示「跟随全局默认」。
// SelectChannel 查找顺序：用户级 → 全局 fallback。

// UserRouteConfig 用户级路由配置。
type UserRouteConfig struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	UserID    int       `gorm:"uniqueIndex:idx_user_route;not null" json:"user_id"`
	// Mode 空字符串表示跟随全局；可选 default|weight|price。
	Mode string `gorm:"size:32;default:''" json:"mode"`
}

// GetUserRouteConfig 读取用户级路由配置；不存在时返回 nil。
func GetUserRouteConfig(userID int) *UserRouteConfig {
	var cfg UserRouteConfig
	if err := DB.Where("user_id = ?", userID).First(&cfg).Error; err != nil {
		return nil
	}
	return &cfg
}

// SaveUserRouteConfig 创建或更新用户级路由配置。
func SaveUserRouteConfig(userID int, mode string) (UserRouteConfig, error) {
	var cfg UserRouteConfig
	result := DB.Where("user_id = ?", userID).First(&cfg)
	if result.RowsAffected == 0 {
		cfg = UserRouteConfig{UserID: userID, Mode: mode}
		if err := DB.Create(&cfg).Error; err != nil {
			return cfg, err
		}
		return cfg, nil
	}
	if err := DB.Model(&cfg).Update("mode", mode).Error; err != nil {
		return cfg, err
	}
	_ = DB.First(&cfg, cfg.ID).Error
	return cfg, nil
}

// DeleteUserRouteConfig 删除用户级路由配置（恢复跟随全局）。
func DeleteUserRouteConfig(userID int) error {
	return DB.Where("user_id = ?", userID).Delete(&UserRouteConfig{}).Error
}

// UserModelGroupWeight 用户级归类权重。(user_id, group_key, channel_id) 唯一。
type UserModelGroupWeight struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	UserID    int       `gorm:"uniqueIndex:idx_user_group_channel;not null" json:"user_id"`
	GroupKey  string    `gorm:"uniqueIndex:idx_user_group_channel;size:256;not null" json:"group_key"`
	ChannelID int       `gorm:"uniqueIndex:idx_user_group_channel;not null" json:"channel_id"`
	Weight    int       `gorm:"default:100" json:"weight"`
	Enabled   bool      `gorm:"default:true" json:"enabled"`
}

// LoadUserModelGroupWeights 加载某用户在某归类下的所有权重配置。
func LoadUserModelGroupWeights(userID int, groupKey string) ([]UserModelGroupWeight, error) {
	var rows []UserModelGroupWeight
	if err := DB.Where("user_id = ? AND group_key = ?", userID, groupKey).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// LoadAllUserModelGroupWeights 加载某用户的所有归类权重配置。
func LoadAllUserModelGroupWeights(userID int) ([]UserModelGroupWeight, error) {
	var rows []UserModelGroupWeight
	if err := DB.Where("user_id = ?", userID).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// UpsertUserModelGroupWeight 创建或更新用户级归类权重。
func UpsertUserModelGroupWeight(userID int, groupKey string, channelID int, weight int, enabled bool) (UserModelGroupWeight, error) {
	var existing UserModelGroupWeight
	result := DB.Where("user_id = ? AND group_key = ? AND channel_id = ?", userID, groupKey, channelID).First(&existing)
	if result.RowsAffected == 0 {
		w := UserModelGroupWeight{
			UserID:    userID,
			GroupKey:  groupKey,
			ChannelID: channelID,
			Weight:    weight,
			Enabled:   enabled,
		}
		if err := DB.Create(&w).Error; err != nil {
			return w, err
		}
		return w, nil
	}
	updates := map[string]interface{}{
		"weight":  weight,
		"enabled": enabled,
	}
	if err := DB.Model(&existing).Updates(updates).Error; err != nil {
		return existing, err
	}
	_ = DB.First(&existing, existing.ID).Error
	return existing, nil
}

// DeleteUserModelGroupWeightByID 按 ID 删除用户级归类权重（校验 user_id）。
func DeleteUserModelGroupWeightByID(id uint, userID int) error {
	return DB.Where("id = ? AND user_id = ?", id, userID).Delete(&UserModelGroupWeight{}).Error
}

// UserModelGroupOverride 用户级模型归类覆盖。(user_id, raw_model) 唯一。
type UserModelGroupOverride struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	UserID    int       `gorm:"uniqueIndex:idx_user_raw_model;not null" json:"user_id"`
	RawModel  string    `gorm:"uniqueIndex:idx_user_raw_model;size:256;not null" json:"raw_model"`
	GroupKey  string    `gorm:"index;size:256;not null" json:"group_key"`
}

// LoadUserModelGroupOverrides 加载某用户的所有模型归类覆盖。
func LoadUserModelGroupOverrides(userID int) (map[string]string, error) {
	var rows []UserModelGroupOverride
	if err := DB.Where("user_id = ?", userID).Find(&rows).Error; err != nil {
		return nil, err
	}
	m := make(map[string]string, len(rows))
	for _, r := range rows {
		key := r.RawModel
		if key == "" || r.GroupKey == "" {
			continue
		}
		m[key] = r.GroupKey
	}
	return m, nil
}

// LoadAllUserModelGroupOverrides 加载某用户的所有模型归类覆盖（原始记录）。
func LoadAllUserModelGroupOverrides(userID int) ([]UserModelGroupOverride, error) {
	var rows []UserModelGroupOverride
	if err := DB.Where("user_id = ?", userID).Order("raw_model ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// UpsertUserModelGroupOverride 新增或更新用户级模型归类覆盖。
func UpsertUserModelGroupOverride(userID int, rawModel string, groupKey string) (UserModelGroupOverride, error) {
	raw := strings.ToLower(strings.TrimSpace(rawModel))
	key := groupKey
	if raw == "" || key == "" {
		return UserModelGroupOverride{}, fmt.Errorf("raw_model and group_key required")
	}
	var existing UserModelGroupOverride
	result := DB.Where("user_id = ? AND raw_model = ?", userID, raw).First(&existing)
	if result.RowsAffected == 0 {
		ov := UserModelGroupOverride{UserID: userID, RawModel: raw, GroupKey: key}
		if err := DB.Create(&ov).Error; err != nil {
			return ov, err
		}
		return ov, nil
	}
	if err := DB.Model(&existing).Update("group_key", key).Error; err != nil {
		return existing, err
	}
	_ = DB.First(&existing, existing.ID).Error
	return existing, nil
}

// DeleteUserModelGroupOverrideByID 按 ID 删除用户级模型归类覆盖（校验 user_id）。
func DeleteUserModelGroupOverrideByID(id uint, userID int) error {
	return DB.Where("id = ? AND user_id = ?", id, userID).Delete(&UserModelGroupOverride{}).Error
}
