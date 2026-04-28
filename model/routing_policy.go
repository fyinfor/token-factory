// Package model: routing_policy 维护「用户级路由策略」实体——
// 用户可在「路由偏好」页配置 N 条策略（按价格 / 按时延 / 按吞吐 / 均衡 / 自定义），
// 每条策略可绑定一组候选渠道与候选模型组成路由白名单，并可选开启 fallback 兜底。
//
// 本文件只负责落库 + 增删改查，PR3 由 service.ResolveRoutingPolicy 把策略翻译成
// router-engine 的 provider JSON + 候选过滤集合后再交给 distributor 接入。
//
// 表设计要点：
//   - routing_policies：主表，每条策略一行；is_default=1 的同 user 至多 1 条，由
//     SetDefaultRoutingPolicy 在事务里维护互斥。
//   - routing_policy_targets：候选池，三种 target_type：
//     channel（任意模型走指定渠道）/ model（任意渠道走指定模型）/ channel_model（精准绑定）。
//     UNIQUE(policy_id, target_type, channel_id, model_name) 防止同一组合重复入库。
//
// 兼容性：所有新表通过 GORM AutoMigrate 直接创建；老库无表会在下次启动自动建出来，
// 不影响存量数据。GORM 默认表名分别为 routing_policies / routing_policy_targets。
package model

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// === 策略类型常量 ===
const (
	// RoutingStrategyPrice 低价优先：按候选渠道单价升序，对应 router-engine sort=price。
	RoutingStrategyPrice = "price"
	// RoutingStrategyLatency 低时延优先：按 LatencyP50Seconds 升序，对应 sort=latency。
	RoutingStrategyLatency = "latency"
	// RoutingStrategyThroughput 高吞吐优先：按 ThroughputTps 降序，对应 sort=throughput。
	RoutingStrategyThroughput = "throughput"
	// RoutingStrategyBalanced 均衡：不显式 sort，让 router-engine 走 1/p^2 加权随机（default_price_lb）。
	RoutingStrategyBalanced = "balanced"
	// RoutingStrategyCustom 自定义：直接透传 ProviderOverridesJSON 给 router-engine（高级用户）。
	RoutingStrategyCustom = "custom"
)

// AllRoutingStrategies 用于校验策略类型枚举（controller 层校验）。
var AllRoutingStrategies = []string{
	RoutingStrategyPrice,
	RoutingStrategyLatency,
	RoutingStrategyThroughput,
	RoutingStrategyBalanced,
	RoutingStrategyCustom,
}

// === Fallback 策略常量（主候选耗尽后用什么策略兜底再选一轮） ===
const (
	// RoutingFallbackNone 不兜底：主候选用完即报错，严格闭环（适合预算/合规场景）。
	RoutingFallbackNone = ""
	// RoutingFallbackPrice 价格兜底：候选池外按全局最低价兜一次。
	RoutingFallbackPrice = "price"
	// RoutingFallbackLatency 时延兜底：候选池外按全局最快兜一次。
	RoutingFallbackLatency = "latency"
	// RoutingFallbackAny 任意兜底：交给 router-engine default_price_lb 兜底。
	RoutingFallbackAny = "any"
)

// AllRoutingFallbacks 用于校验 fallback_strategy 枚举。
var AllRoutingFallbacks = []string{
	RoutingFallbackNone,
	RoutingFallbackPrice,
	RoutingFallbackLatency,
	RoutingFallbackAny,
}

// === Target type 常量（候选池条目类型） ===
const (
	// RoutingTargetTypeChannel 仅限指定渠道；model_name 为空。
	RoutingTargetTypeChannel = "channel"
	// RoutingTargetTypeModel 仅限指定模型；channel_id 为 0。
	RoutingTargetTypeModel = "model"
	// RoutingTargetTypeChannelModel 精准绑定（渠道 × 模型）。
	RoutingTargetTypeChannelModel = "channel_model"
)

// AllRoutingTargetTypes 用于校验 target_type 枚举。
var AllRoutingTargetTypes = []string{
	RoutingTargetTypeChannel,
	RoutingTargetTypeModel,
	RoutingTargetTypeChannelModel,
}

// === 策略状态 ===
const (
	// RoutingPolicyStatusEnabled 启用；resolver 会读取该策略。
	RoutingPolicyStatusEnabled = 1
	// RoutingPolicyStatusDisabled 禁用；resolver 跳过；is_default 也不会生效。
	RoutingPolicyStatusDisabled = 0
)

// === 错误 ===
var (
	// ErrRoutingPolicyNotFound 找不到指定策略，或非该用户拥有。
	ErrRoutingPolicyNotFound = errors.New("routing policy not found")
	// ErrRoutingPolicyInvalidStrategy 策略类型不合法。
	ErrRoutingPolicyInvalidStrategy = errors.New("invalid routing strategy")
	// ErrRoutingPolicyInvalidFallback fallback 策略类型不合法。
	ErrRoutingPolicyInvalidFallback = errors.New("invalid fallback strategy")
	// ErrRoutingPolicyInvalidTarget target 条目不合法（target_type 错 / channel_id+model_name 都为空等）。
	ErrRoutingPolicyInvalidTarget = errors.New("invalid routing policy target")
	// ErrRoutingPolicyEmptyName 策略名为空。
	ErrRoutingPolicyEmptyName = errors.New("routing policy name is required")
	// ErrRoutingPolicyDisabledCannotDefault 禁用策略不可设为默认。
	ErrRoutingPolicyDisabledCannotDefault = errors.New("disabled policy cannot be default")
)

// RoutingPolicy 「用户路由策略」主表。
//
// 字段语义：
//   - Strategy：主策略（price / latency / throughput / balanced / custom），
//     resolver 会翻译成 router-engine 的 provider JSON 的 sort 字段；custom 直接透传。
//   - AllowFallbacks：true 表示允许 router-engine 在主候选失败时 fall back 到剩余候选；
//     false 表示候选只取一个，失败即报错（严格闭环）。
//   - FallbackStrategy：当 AllowFallbacks=true 但主候选池整体不可用时，是否再扩展到候选池外做兜底；
//     '' 表示不扩；其它值见 AllRoutingFallbacks。
//   - MaxPrice：>0 时硬性筛掉单价超过该值的渠道（同 router-engine max_price.completion）。
//   - MaxLatencyMs：>0 时硬性筛掉 P50 时延超过该值的渠道（毫秒）。
//   - MinThroughputTps：>0 时硬性筛掉吞吐低于该值的渠道。
//   - ProviderOverridesJSON：高级旁路：custom 策略下直接透传给 router-engine；
//     非 custom 时由 resolver 在合并前作为 base，policy 字段再覆盖关键项。
//   - IsDefault：用户当前生效的默认策略；同 user 至多 1 条。SetDefaultRoutingPolicy 维护互斥。
//   - Status：启用 / 禁用；禁用策略不会被 resolver 选中，也不能成为默认。
//   - Targets：关联候选池条目；GORM 不展开，PR3 显式 LoadTargets。
type RoutingPolicy struct {
	ID                    int64                 `json:"id" gorm:"primaryKey;comment:主键ID"`
	UserID                int                   `json:"user_id" gorm:"type:int;index;not null;comment:所属用户ID"`
	Name                  string                `json:"name" gorm:"type:varchar(128);not null;comment:策略名"`
	Description           string                `json:"description" gorm:"type:text;default:'';comment:策略描述"`
	Strategy              string                `json:"strategy" gorm:"type:varchar(32);not null;default:'balanced';comment:策略类型 price|latency|throughput|balanced|custom"`
	// AllowFallbacks 默认由 controller 层在「字段缺省」时填 true（OpenRouter 兼容默认）；
	// 这里不设 GORM default:true，否则用户显式传 false 会被 GORM 用 default 救回 true。
	AllowFallbacks        bool                  `json:"allow_fallbacks" gorm:"type:boolean;not null;comment:是否允许router-engine在主候选失败时fallback"`
	FallbackStrategy      string                `json:"fallback_strategy" gorm:"type:varchar(32);default:'';comment:候选池整体不可用时的兜底策略 ''/price/latency/any"`
	MaxPrice              float64               `json:"max_price" gorm:"type:decimal(20,10);default:0;comment:>0时硬性过滤单价超过该值的渠道"`
	MaxLatencyMs          int                   `json:"max_latency_ms" gorm:"type:int;default:0;comment:>0时硬性过滤P50时延(ms)超过该值的渠道"`
	MinThroughputTps      float64               `json:"min_throughput_tps" gorm:"type:decimal(20,10);default:0;comment:>0时硬性过滤吞吐(tps)低于该值的渠道"`
	ProviderOverridesJSON string                `json:"provider_overrides_json" gorm:"type:text;default:'';comment:custom策略下原样透传给router-engine的provider JSON"`
	// Status 不设 GORM default:1，原因同 AllowFallbacks：希望尊重「显式 0=disabled」。
	// controller 在 Create 路径上用 *int 兜底；调用 model 层 API 的非 controller 路径需自行明确传值。
	Status    int  `json:"status" gorm:"type:int;index;not null;comment:状态 1启用 0禁用"`
	IsDefault bool `json:"is_default" gorm:"type:boolean;index;not null;comment:是否为该用户的默认策略 同user至多1条"`
	Priority              int                   `json:"priority" gorm:"type:int;default:0;comment:多策略时的排序权重 预留"`
	CreatedAt             int64                 `json:"created_at" gorm:"type:bigint;index;comment:创建时间戳"`
	UpdatedAt             int64                 `json:"updated_at" gorm:"type:bigint;comment:更新时间戳"`
	Targets               []RoutingPolicyTarget `json:"targets,omitempty" gorm:"-"`
}

// RoutingPolicyTarget 「策略候选池」条目。
//
// 一条策略可包含 N 个候选条目，三种类型择一：
//   - channel：限制必须用某个渠道（model_name=''）；
//   - model：限制必须用某个模型（channel_id=0）；
//   - channel_model：精准绑定（同时限定渠道与模型）。
//
// resolver 在合并多条候选条目时取并集（任一命中即视为允许）。
//
// UNIQUE(policy_id, target_type, channel_id, model_name) 防止重复入库；不同 target_type
// 之间允许同名（例如 channel=10 与 channel_model=10/gpt-4o 可共存）。
type RoutingPolicyTarget struct {
	ID         int64  `json:"id" gorm:"primaryKey;comment:主键ID"`
	PolicyID   int64  `json:"policy_id" gorm:"type:bigint;index;not null;uniqueIndex:idx_routing_policy_targets_unique,priority:1;comment:所属策略ID"`
	TargetType string `json:"target_type" gorm:"type:varchar(16);not null;default:'channel_model';uniqueIndex:idx_routing_policy_targets_unique,priority:2;comment:候选条目类型 channel|model|channel_model"`
	ChannelID  int    `json:"channel_id" gorm:"type:int;default:0;uniqueIndex:idx_routing_policy_targets_unique,priority:3;comment:渠道ID 0表示按模型限制"`
	ModelName  string `json:"model_name" gorm:"type:varchar(128);default:'';uniqueIndex:idx_routing_policy_targets_unique,priority:4;comment:模型名 空串表示按渠道限制"`
	CreatedAt  int64  `json:"created_at" gorm:"type:bigint;comment:创建时间戳"`
}

// TableName 显式表名（避免 GORM 复数化拼出 routing_policys）。
func (RoutingPolicy) TableName() string { return "routing_policies" }

// TableName 显式表名。
func (RoutingPolicyTarget) TableName() string { return "routing_policy_targets" }

// === 校验辅助 ===

func validateRoutingStrategy(s string) error {
	for _, v := range AllRoutingStrategies {
		if v == s {
			return nil
		}
	}
	return ErrRoutingPolicyInvalidStrategy
}

func validateRoutingFallback(s string) error {
	for _, v := range AllRoutingFallbacks {
		if v == s {
			return nil
		}
	}
	return ErrRoutingPolicyInvalidFallback
}

func validateRoutingTargetType(s string) error {
	for _, v := range AllRoutingTargetTypes {
		if v == s {
			return nil
		}
	}
	return ErrRoutingPolicyInvalidTarget
}

// ValidateRoutingPolicy 在写入前做基础合法性校验，避免脏数据落库。
//
// 仅做结构性校验：不查渠道是否存在、不验证用户是否有权使用，这类业务校验放在 resolver/distributor。
func ValidateRoutingPolicy(p *RoutingPolicy) error {
	if p == nil {
		return ErrRoutingPolicyNotFound
	}
	if strings.TrimSpace(p.Name) == "" {
		return ErrRoutingPolicyEmptyName
	}
	if err := validateRoutingStrategy(p.Strategy); err != nil {
		return err
	}
	if err := validateRoutingFallback(p.FallbackStrategy); err != nil {
		return err
	}
	if p.MaxPrice < 0 || p.MaxLatencyMs < 0 || p.MinThroughputTps < 0 {
		return fmt.Errorf("threshold values must be >= 0")
	}
	for i := range p.Targets {
		if err := validateRoutingTarget(&p.Targets[i]); err != nil {
			return err
		}
	}
	return nil
}

func validateRoutingTarget(t *RoutingPolicyTarget) error {
	if t == nil {
		return ErrRoutingPolicyInvalidTarget
	}
	if err := validateRoutingTargetType(t.TargetType); err != nil {
		return err
	}
	t.ModelName = strings.TrimSpace(t.ModelName)
	switch t.TargetType {
	case RoutingTargetTypeChannel:
		if t.ChannelID <= 0 {
			return ErrRoutingPolicyInvalidTarget
		}
		t.ModelName = ""
	case RoutingTargetTypeModel:
		if t.ModelName == "" {
			return ErrRoutingPolicyInvalidTarget
		}
		t.ChannelID = 0
	case RoutingTargetTypeChannelModel:
		if t.ChannelID <= 0 || t.ModelName == "" {
			return ErrRoutingPolicyInvalidTarget
		}
	}
	return nil
}

// === CRUD ===

// CreateRoutingPolicy 创建一条策略；同时落候选池。事务内写入，失败整体回滚。
//
// 写入策略：
//   - is_default=true 时自动复用 SetDefaultRoutingPolicy 的互斥逻辑（先把同 user 其它策略 IsDefault 置 false）；
//   - 候选池条目 PolicyID 由本函数补齐，调用方无需关心。
func CreateRoutingPolicy(p *RoutingPolicy) error {
	if err := ValidateRoutingPolicy(p); err != nil {
		return err
	}
	now := time.Now().Unix()
	p.CreatedAt = now
	p.UpdatedAt = now
	if p.Strategy == "" {
		p.Strategy = RoutingStrategyBalanced
	}
	tx := DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	if p.IsDefault && p.Status == RoutingPolicyStatusEnabled {
		if err := tx.Model(&RoutingPolicy{}).
			Where("user_id = ? AND is_default = ?", p.UserID, true).
			Updates(map[string]any{"is_default": false, "updated_at": now}).Error; err != nil {
			tx.Rollback()
			return err
		}
	} else if p.Status != RoutingPolicyStatusEnabled {
		// 禁用策略不能是默认。
		p.IsDefault = false
	}
	if err := tx.Create(p).Error; err != nil {
		tx.Rollback()
		return err
	}
	for i := range p.Targets {
		p.Targets[i].PolicyID = p.ID
		p.Targets[i].CreatedAt = now
	}
	if len(p.Targets) > 0 {
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&p.Targets).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}

// UpdateRoutingPolicy 更新策略主体 + 候选池（全量替换）。
//
// 全量替换策略：
//   - 候选池采用 delete-then-insert，简化"哪个 target 是新的 / 改的 / 删的"判断；
//   - 调用方只需传完整 Targets 列表。空列表表示「无候选限制」（即放空，全部渠道可用）。
//   - is_default 的转移交由 SetDefaultRoutingPolicy；本函数不允许直接通过 update 设默认。
func UpdateRoutingPolicy(userID int, policyID int64, patch *RoutingPolicy) error {
	if err := ValidateRoutingPolicy(patch); err != nil {
		return err
	}
	tx := DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	var existing RoutingPolicy
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND user_id = ?", policyID, userID).
		First(&existing).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRoutingPolicyNotFound
		}
		return err
	}
	now := time.Now().Unix()
	updates := map[string]any{
		"name":                    strings.TrimSpace(patch.Name),
		"description":             patch.Description,
		"strategy":                patch.Strategy,
		"allow_fallbacks":         patch.AllowFallbacks,
		"fallback_strategy":       patch.FallbackStrategy,
		"max_price":               patch.MaxPrice,
		"max_latency_ms":          patch.MaxLatencyMs,
		"min_throughput_tps":      patch.MinThroughputTps,
		"provider_overrides_json": patch.ProviderOverridesJSON,
		"status":                  patch.Status,
		"priority":                patch.Priority,
		"updated_at":              now,
	}
	// 禁用策略不能保持默认；自动取消。
	if patch.Status != RoutingPolicyStatusEnabled && existing.IsDefault {
		updates["is_default"] = false
	}
	if err := tx.Model(&RoutingPolicy{}).Where("id = ?", policyID).Updates(updates).Error; err != nil {
		tx.Rollback()
		return err
	}
	// 全量替换候选池：先删后插。
	if err := tx.Where("policy_id = ?", policyID).Delete(&RoutingPolicyTarget{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	if len(patch.Targets) > 0 {
		newTargets := make([]RoutingPolicyTarget, 0, len(patch.Targets))
		for i := range patch.Targets {
			t := patch.Targets[i]
			t.ID = 0
			t.PolicyID = policyID
			t.CreatedAt = now
			newTargets = append(newTargets, t)
		}
		if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&newTargets).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}

// DeleteRoutingPolicy 删除策略 + 关联候选池（事务）。
func DeleteRoutingPolicy(userID int, policyID int64) error {
	tx := DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	res := tx.Where("id = ? AND user_id = ?", policyID, userID).Delete(&RoutingPolicy{})
	if res.Error != nil {
		tx.Rollback()
		return res.Error
	}
	if res.RowsAffected == 0 {
		tx.Rollback()
		return ErrRoutingPolicyNotFound
	}
	if err := tx.Where("policy_id = ?", policyID).Delete(&RoutingPolicyTarget{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

// SetDefaultRoutingPolicy 设置某条策略为该用户的默认策略；同 user 其它策略 IsDefault 置 false。
//
// 仅启用状态的策略可设为默认；禁用策略尝试设默认会返回 ErrRoutingPolicyDisabledCannotDefault。
func SetDefaultRoutingPolicy(userID int, policyID int64) error {
	tx := DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	var target RoutingPolicy
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ? AND user_id = ?", policyID, userID).
		First(&target).Error; err != nil {
		tx.Rollback()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRoutingPolicyNotFound
		}
		return err
	}
	if target.Status != RoutingPolicyStatusEnabled {
		tx.Rollback()
		return ErrRoutingPolicyDisabledCannotDefault
	}
	now := time.Now().Unix()
	if err := tx.Model(&RoutingPolicy{}).
		Where("user_id = ? AND id <> ? AND is_default = ?", userID, policyID, true).
		Updates(map[string]any{"is_default": false, "updated_at": now}).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Model(&RoutingPolicy{}).
		Where("id = ?", policyID).
		Updates(map[string]any{"is_default": true, "updated_at": now}).Error; err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit().Error
}

// ClearDefaultRoutingPolicy 取消该用户的默认策略（不删除策略本身）。
func ClearDefaultRoutingPolicy(userID int) error {
	now := time.Now().Unix()
	return DB.Model(&RoutingPolicy{}).
		Where("user_id = ? AND is_default = ?", userID, true).
		Updates(map[string]any{"is_default": false, "updated_at": now}).Error
}

// === 查询 ===

// ListRoutingPoliciesByUser 分页查询某用户的所有策略；按 IsDefault desc, Priority desc, ID desc 排序。
//
// 返回的 RoutingPolicy 不包含 Targets，避免 N+1；如需 Targets 调用 LoadRoutingPolicyTargets 批量加载。
func ListRoutingPoliciesByUser(userID int, pageInfo *common.PageInfo) ([]*RoutingPolicy, int64, error) {
	var (
		items []*RoutingPolicy
		total int64
	)
	q := DB.Model(&RoutingPolicy{}).Where("user_id = ?", userID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	query := q.Order("is_default desc, priority desc, id desc")
	if pageInfo != nil {
		query = query.Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx())
	}
	if err := query.Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetRoutingPolicyByID 取用户拥有的单条策略（自动加载 Targets）。
func GetRoutingPolicyByID(userID int, policyID int64) (*RoutingPolicy, error) {
	var p RoutingPolicy
	if err := DB.Where("id = ? AND user_id = ?", policyID, userID).First(&p).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRoutingPolicyNotFound
		}
		return nil, err
	}
	targets, err := ListRoutingPolicyTargets(p.ID)
	if err != nil {
		return nil, err
	}
	p.Targets = targets
	return &p, nil
}

// GetDefaultRoutingPolicyByUser 取该用户当前生效的默认策略；不存在或被禁用时返回 (nil, nil)。
//
// resolver 走快路径：缓存 miss 时直接读这条；找不到 default 即视为「无策略」。
func GetDefaultRoutingPolicyByUser(userID int) (*RoutingPolicy, error) {
	var p RoutingPolicy
	err := DB.Where("user_id = ? AND is_default = ? AND status = ?", userID, true, RoutingPolicyStatusEnabled).
		Order("id desc").
		First(&p).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	targets, err := ListRoutingPolicyTargets(p.ID)
	if err != nil {
		return nil, err
	}
	p.Targets = targets
	return &p, nil
}

// ListRoutingPolicyTargets 取某策略的所有候选条目；按 ID 升序，便于稳定展示。
func ListRoutingPolicyTargets(policyID int64) ([]RoutingPolicyTarget, error) {
	var rows []RoutingPolicyTarget
	if err := DB.Where("policy_id = ?", policyID).Order("id asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// IsRoutingPolicyNotFound 判断错误是否为「策略不存在」。
func IsRoutingPolicyNotFound(err error) bool {
	return errors.Is(err, ErrRoutingPolicyNotFound) || errors.Is(err, gorm.ErrRecordNotFound)
}
