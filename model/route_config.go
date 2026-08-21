package model

import "time"

// ── 全局路由模式（进程内智能路由）────────────────────────────────
//
// 从 TokenFactory 剥离：全局只有「一个」路由模式，作用于所有归类。
// 读全局模式 → 计算请求模型的归类 → 按该模式对候选渠道排序。

const (
	// RouteModeDefault 路由关闭：不走智能路由，按渠道 ID 升序选首条可用。
	RouteModeDefault = "default"
	// RouteModeWeight 权重模式：按「归类 × 渠道」配置的权重，优先高权重渠道。
	RouteModeWeight = "weight"
	// RouteModePrice 价格优模式：按候选渠道最终单价升序，优先低价。
	RouteModePrice = "price"
	// RouteModePerformance 性能模式（占位，暂未实现）。
	RouteModePerformance = "performance"
	// RouteModePricePerf 价格/性能占比模式（占位，暂未实现）。
	RouteModePricePerf = "price_perf"
)

// IsImplementedRouteMode 返回该模式是否启用智能路由策略（weight / price）。
func IsImplementedRouteMode(mode string) bool {
	return mode == RouteModeWeight || mode == RouteModePrice
}

// IsRouteDisabled 返回路由是否关闭（default 或未识别模式）。
func IsRouteDisabled(mode string) bool {
	return !IsImplementedRouteMode(mode)
}

// RouteConfig 全局路由配置（单例，固定 ID=1）。
type RouteConfig struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	UpdatedAt time.Time `json:"updated_at"`
	// Mode 当前全局路由模式：default|weight|price|performance|price_perf。
	// 系统默认价格优先：未配置用户按最终单价升序调度。
	Mode string `gorm:"size:32;default:'price'" json:"mode"`
	// PricePerfRatio 价格/性能占比（0-100，占位字段）。
	PricePerfRatio int `gorm:"default:50" json:"price_perf_ratio"`
}

// GetRouteConfig 读取全局路由配置；不存在时以价格优先模式初始化。
func GetRouteConfig() RouteConfig {
	var cfg RouteConfig
	if err := DB.First(&cfg, 1).Error; err != nil {
		cfg = RouteConfig{ID: 1, Mode: RouteModePrice, PricePerfRatio: 50}
		_ = DB.Create(&cfg).Error
	}
	if cfg.Mode == "" {
		cfg.Mode = RouteModePrice
	}
	return cfg
}

// EnsureDefaultRouteModePrice 将全局默认路由改为价格优先，并让仍为「默认/空」的用户跟随全局。
// 幂等：已显式配置 weight/price 的用户不受影响。
func EnsureDefaultRouteModePrice() error {
	if DB == nil {
		return nil
	}
	if err := DB.Model(&RouteConfig{}).
		Where("id = ? AND (mode = ? OR mode = ? OR mode IS NULL)", 1, RouteModeDefault, "").
		Update("mode", RouteModePrice).Error; err != nil {
		return err
	}
	// 用户曾选「默认（原生）」的，改为空串以跟随新的全局价格优先。
	return DB.Model(&UserRouteConfig{}).
		Where("mode = ?", RouteModeDefault).
		Update("mode", "").Error
}

// SaveRouteConfig 更新全局路由配置（单例 upsert）。
func SaveRouteConfig(mode string, pricePerfRatio int) (RouteConfig, error) {
	cfg := GetRouteConfig()
	cfg.ID = 1
	cfg.Mode = mode
	cfg.PricePerfRatio = pricePerfRatio
	err := DB.Save(&cfg).Error
	return cfg, err
}

// ModelGroupWeight 归类权重：在某个归类下，为特定渠道配置权重（权重模式使用）。
// (group_key, channel_id) 唯一。
type ModelGroupWeight struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	GroupKey  string    `gorm:"uniqueIndex:idx_group_channel;size:256;not null" json:"group_key"`
	ChannelID int       `gorm:"uniqueIndex:idx_group_channel;not null" json:"channel_id"`
	Weight    int       `gorm:"default:100" json:"weight"`
	Enabled   bool      `gorm:"default:true" json:"enabled"`
}

// LoadModelGroupWeights 加载某归类下的全部全局权重。
func LoadModelGroupWeights(groupKey string) ([]ModelGroupWeight, error) {
	var rows []ModelGroupWeight
	if groupKey == "" {
		return rows, nil
	}
	if err := DB.Where("group_key = ?", groupKey).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// LoadAllModelGroupWeights 加载全部全局归类权重。
func LoadAllModelGroupWeights() ([]ModelGroupWeight, error) {
	var rows []ModelGroupWeight
	if err := DB.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// UpsertModelGroupWeight 创建或更新全局归类权重。
func UpsertModelGroupWeight(groupKey string, channelID int, weight int, enabled bool) (ModelGroupWeight, error) {
	var existing ModelGroupWeight
	result := DB.Where("group_key = ? AND channel_id = ?", groupKey, channelID).First(&existing)
	if result.RowsAffected == 0 {
		w := ModelGroupWeight{
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

// DeleteModelGroupWeightByID 按 ID 删除全局归类权重。
func DeleteModelGroupWeightByID(id uint) error {
	return DB.Delete(&ModelGroupWeight{}, id).Error
}
