package model

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

const (
	NameRuleExact = iota
	NameRulePrefix
	NameRuleContains
	NameRuleSuffix
)

type BoundChannel struct {
	Name      string `json:"name"`
	Type      int    `json:"type"`
	RouteSlug string `json:"route_slug,omitempty"`
}

type Model struct {
	Id                int            `json:"id"`
	ModelName         string         `json:"model_name" gorm:"size:128;not null;uniqueIndex:uk_model_name_delete_at,priority:1"`
	Description       string         `json:"description,omitempty" gorm:"type:text"`
	DescriptionEn     string         `json:"description_en,omitempty" gorm:"type:text"`
	DocIntroduction   string         `json:"doc_introduction,omitempty" gorm:"type:text"`
	DocIntroductionEn string         `json:"doc_introduction_en,omitempty" gorm:"type:text"`
	ApiDocs           string         `json:"api_docs,omitempty" gorm:"type:text"`
	Icon              string         `json:"icon,omitempty" gorm:"type:varchar(128)"`
	Tags              string         `json:"tags,omitempty" gorm:"type:varchar(255)"`
	VendorID          int            `json:"vendor_id,omitempty" gorm:"index"`
	Endpoints         string         `json:"endpoints,omitempty" gorm:"type:text"`
	Status            int            `json:"status" gorm:"default:1"`
	SyncOfficial      int            `json:"sync_official" gorm:"default:1"`
	CreatedTime       int64          `json:"created_time" gorm:"bigint"`
	UpdatedTime       int64          `json:"updated_time" gorm:"bigint"`
	DeletedAt         gorm.DeletedAt `json:"-" gorm:"index;uniqueIndex:uk_model_name_delete_at,priority:2"`

	BoundChannels         []BoundChannel `json:"bound_channels,omitempty" gorm:"-"`
	EnableGroups          []string       `json:"enable_groups,omitempty" gorm:"-"`
	QuotaTypes            []int          `json:"quota_types,omitempty" gorm:"-"`
	NameRule              int            `json:"name_rule" gorm:"default:0"`
	OwnerUserID           int            `json:"owner_user_id" gorm:"type:int;index;default:0"`           // 模型归属用户ID（供应商场景）
	SupplierApplicationID int            `json:"supplier_application_id" gorm:"type:int;index;default:0"` // 关联 supplier_applications.id

	MatchedModels    []string `json:"matched_models,omitempty" gorm:"-"`
	MatchedCount     int      `json:"matched_count,omitempty" gorm:"-"`
	Visibility       string   `json:"visibility" gorm:"-"`
	VisibilitySetIDs []int    `json:"visibility_set_ids,omitempty" gorm:"-"`

	// 排序权重和手动调用次数（用于热门排序干预）
	SortWeight         float64 `json:"sort_weight" gorm:"default:1"`
	ManualBaseReqCount int64   `json:"manual_base_req_count" gorm:"default:0"` // 手动设置调用基数
}

func (mi *Model) Insert() error {
	now := common.GetTimestamp()
	mi.CreatedTime = now
	mi.UpdatedTime = now

	// 保存原始值（因为 Create 后可能被 GORM 的 default 标签覆盖为 1）
	originalStatus := mi.Status
	originalSyncOfficial := mi.SyncOfficial

	// 先创建记录（GORM 会对零值字段应用默认值）
	if err := DB.Create(mi).Error; err != nil {
		return err
	}

	// 使用保存的原始值进行更新，确保零值能正确保存
	return DB.Model(&Model{}).Where("id = ?", mi.Id).Updates(map[string]interface{}{
		"status":        originalStatus,
		"sync_official": originalSyncOfficial,
	}).Error
}

func IsModelNameDuplicated(id int, name string) (bool, error) {
	if name == "" {
		return false, nil
	}
	var cnt int64
	err := DB.Model(&Model{}).Where("model_name = ? AND id <> ?", name, id).Count(&cnt).Error
	return cnt > 0, err
}

func (mi *Model) Update() error {
	mi.UpdatedTime = common.GetTimestamp()
	return DB.Transaction(func(tx *gorm.DB) error {
		var existing Model
		if err := tx.Select("model_name").Where("id = ?", mi.Id).First(&existing).Error; err != nil {
			return err
		}
		// 使用 Select 强制更新所有字段，包括零值
		if err := tx.Model(&Model{}).Where("id = ?", mi.Id).
			Select("model_name", "description", "description_en", "doc_introduction", "doc_introduction_en", "api_docs", "icon", "tags", "vendor_id", "endpoints", "status", "sync_official", "name_rule", "owner_user_id", "supplier_application_id", "updated_time").
			Updates(mi).Error; err != nil {
			return err
		}
		if existing.ModelName != mi.ModelName {
			return tx.Model(&ChannelModelHotOverride{}).
				Where("model_name = ?", existing.ModelName).
				Update("model_name", mi.ModelName).Error
		}
		return nil
	})
}

func (mi *Model) Delete() error {
	return DB.Delete(mi).Error
}

func GetVendorModelCounts() (map[int64]int64, error) {
	var stats []struct {
		VendorID int64
		Count    int64
	}
	if err := DB.Model(&Model{}).
		Select("vendor_id as vendor_id, count(*) as count").
		Group("vendor_id").
		Scan(&stats).Error; err != nil {
		return nil, err
	}
	m := make(map[int64]int64, len(stats))
	for _, s := range stats {
		m[s.VendorID] = s.Count
	}
	return m, nil
}

func GetAllModels(offset int, limit int) ([]*Model, error) {
	var models []*Model
	err := DB.Order("id DESC").Offset(offset).Limit(limit).Find(&models).Error
	return models, err
}

// ListModelsByOwnerUser 分页查询指定归属用户创建的模型。
func ListModelsByOwnerUser(ownerUserID int, offset int, limit int) ([]*Model, int64, error) {
	var (
		models []*Model
		total  int64
	)
	query := DB.Model(&Model{}).Where("owner_user_id = ?", ownerUserID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&models).Error; err != nil {
		return nil, 0, err
	}
	return models, total, nil
}

func applyModelTagFilter(db *gorm.DB, tag string) *gorm.DB {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return db
	}
	return db.Where("tags = ? OR tags LIKE ? OR tags LIKE ? OR tags LIKE ?",
		tag, tag+",%", "%,"+tag+",%", "%,"+tag)
}

// SearchSupplierModels 搜索供应商模型（供应商查自己，管理员查全部供应商）。
func SearchSupplierModels(ownerUserID *int, keyword string, vendor string, routeSlug string, tag string, offset int, limit int) ([]*Model, int64, error) {
	var (
		models []*Model
		total  int64
	)
	db := DB.Model(&Model{})
	if ownerUserID != nil {
		db = db.Where("owner_user_id = ?", *ownerUserID)
	} else {
		db = db.Where("owner_user_id > ? AND supplier_application_id > ?", 0, 0)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("model_name LIKE ? OR description LIKE ? OR tags LIKE ?", like, like, like)
	}
	if vendor != "" {
		if vid, err := strconv.Atoi(vendor); err == nil {
			db = db.Where("models.vendor_id = ?", vid)
		} else {
			db = db.Joins("JOIN vendors ON vendors.id = models.vendor_id").Where("vendors.name LIKE ?", "%"+vendor+"%")
		}
	}
	db = applyModelTagFilter(db, tag)
	db = applyModelRouteSlugFilter(db, routeSlug, ownerUserID)
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Order("models.id DESC").Offset(offset).Limit(limit).Find(&models).Error; err != nil {
		return nil, 0, err
	}
	return models, total, nil
}

func GetBoundChannelsByModelsMap(modelNames []string) (map[string][]BoundChannel, error) {
	result := make(map[string][]BoundChannel)
	if len(modelNames) == 0 {
		return result, nil
	}
	type row struct {
		Model     string
		Name      string
		Type      int
		RouteSlug string
	}
	var rows []row
	err := DB.Table("channels").
		Select("abilities.model as model, channels.name as name, channels.type as type, channels.route_slug as route_slug").
		Joins("JOIN abilities ON abilities.channel_id = channels.id").
		Where("abilities.model IN ? AND abilities.enabled = ?", modelNames, true).
		Distinct().
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		result[r.Model] = append(result[r.Model], BoundChannel{Name: r.Name, Type: r.Type, RouteSlug: r.RouteSlug})
	}
	return result, nil
}

func SearchModels(keyword string, vendor string, routeSlug string, tag string, offset int, limit int) ([]*Model, int64, error) {
	var models []*Model
	db := DB.Model(&Model{})
	if keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("model_name LIKE ? OR description LIKE ? OR tags LIKE ?", like, like, like)
	}
	if vendor != "" {
		if vid, err := strconv.Atoi(vendor); err == nil {
			db = db.Where("models.vendor_id = ?", vid)
		} else {
			db = db.Joins("JOIN vendors ON vendors.id = models.vendor_id").Where("vendors.name LIKE ?", "%"+vendor+"%")
		}
	}
	db = applyModelTagFilter(db, tag)
	db = applyModelRouteSlugFilter(db, routeSlug, nil)
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Order("models.id DESC").Offset(offset).Limit(limit).Find(&models).Error; err != nil {
		return nil, 0, err
	}
	return models, total, nil
}

func applyModelRouteSlugFilter(db *gorm.DB, routeSlug string, ownerUserID *int) *gorm.DB {
	routeSlug = strings.TrimSpace(routeSlug)
	if routeSlug == "" {
		return db
	}

	prefixLike := "abilities.model LIKE (models.model_name || '%')"
	containsLike := "abilities.model LIKE ('%' || models.model_name || '%')"
	suffixLike := "abilities.model LIKE ('%' || models.model_name)"
	if common.UsingMySQL {
		prefixLike = "abilities.model LIKE CONCAT(models.model_name, '%')"
		containsLike = "abilities.model LIKE CONCAT('%', models.model_name, '%')"
		suffixLike = "abilities.model LIKE CONCAT('%', models.model_name)"
	}

	where := `EXISTS (
		SELECT 1
		FROM abilities
		JOIN channels ON channels.id = abilities.channel_id
		WHERE abilities.enabled = ?
		  AND channels.route_slug LIKE ?`
	args := []interface{}{true, "%" + routeSlug + "%"}
	if ownerUserID != nil {
		where += " AND channels.owner_user_id = ?"
		args = append(args, *ownerUserID)
	}
	where += ` AND (
			(models.name_rule = ? AND abilities.model = models.model_name)
			OR (models.name_rule = ? AND ` + prefixLike + `)
			OR (models.name_rule = ? AND ` + containsLike + `)
			OR (models.name_rule = ? AND ` + suffixLike + `)
		  )
	)`
	args = append(args, NameRuleExact, NameRulePrefix, NameRuleContains, NameRuleSuffix)
	return db.Where(where, args...)
}

// GetExistingModelNames 从给定名称列表中返回已在 model_meta 表中存在记录的模型名。
// 用于上架向导诊断：快速判断哪些模型需要手动去 /console/models 配置元数据。
func GetExistingModelNames(names []string) ([]string, error) {
	if len(names) == 0 {
		return nil, nil
	}
	var result []string
	err := DB.Model(&Model{}).
		Select("model_name").
		Where("model_name IN ?", names).
		Pluck("model_name", &result).Error
	return result, err
}

var (
	activeModelRowsCache     []Model
	activeModelRowsCacheMu   sync.RWMutex
	activeModelRowsCacheTime time.Time
)

const activeModelRowsCacheTTL = 30 * time.Second

func modelNameRulePriority(rule int) int {
	switch rule {
	case NameRuleExact:
		return 0
	case NameRulePrefix:
		return 1
	case NameRuleSuffix:
		return 2
	case NameRuleContains:
		return 3
	default:
		return 9
	}
}

func modelNameMatchesRule(pattern, target string, rule int) bool {
	switch rule {
	case NameRuleExact:
		return target == pattern
	case NameRulePrefix:
		return strings.HasPrefix(target, pattern)
	case NameRuleSuffix:
		return strings.HasSuffix(target, pattern)
	case NameRuleContains:
		return strings.Contains(target, pattern)
	default:
		return false
	}
}

// resolveModelTagsFromRows 按 models 表 name_rule（精确/前缀/后缀/包含）解析模型标签。
func resolveModelTagsFromRows(modelName string, rows []Model) string {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return ""
	}
	bestIdx := -1
	for i := range rows {
		row := rows[i]
		if !modelNameMatchesRule(row.ModelName, modelName, row.NameRule) {
			continue
		}
		if bestIdx < 0 {
			bestIdx = i
			continue
		}
		cur := rows[bestIdx]
		curPriority := modelNameRulePriority(cur.NameRule)
		newPriority := modelNameRulePriority(row.NameRule)
		if newPriority < curPriority {
			bestIdx = i
			continue
		}
		if newPriority == curPriority && len(row.ModelName) > len(cur.ModelName) {
			bestIdx = i
		}
	}
	if bestIdx < 0 {
		return ""
	}
	return strings.TrimSpace(rows[bestIdx].Tags)
}

func loadActiveModelRows() ([]Model, error) {
	if DB == nil {
		return nil, nil
	}
	var rows []Model
	err := DB.Model(&Model{}).
		Select("id", "model_name", "tags", "name_rule").
		Where("status = 1").
		Find(&rows).Error
	return rows, err
}

func getActiveModelRowsCached() []Model {
	activeModelRowsCacheMu.RLock()
	if time.Since(activeModelRowsCacheTime) < activeModelRowsCacheTTL && activeModelRowsCache != nil {
		rows := activeModelRowsCache
		activeModelRowsCacheMu.RUnlock()
		return rows
	}
	activeModelRowsCacheMu.RUnlock()

	activeModelRowsCacheMu.Lock()
	defer activeModelRowsCacheMu.Unlock()
	if time.Since(activeModelRowsCacheTime) < activeModelRowsCacheTTL && activeModelRowsCache != nil {
		return activeModelRowsCache
	}
	rows, err := loadActiveModelRows()
	if err != nil {
		if activeModelRowsCache != nil {
			return activeModelRowsCache
		}
		return nil
	}
	activeModelRowsCache = rows
	activeModelRowsCacheTime = time.Now()
	return rows
}

// InvalidateActiveModelRowsCache 在 models 元数据变更后使标签解析缓存失效。
func InvalidateActiveModelRowsCache() {
	activeModelRowsCacheMu.Lock()
	defer activeModelRowsCacheMu.Unlock()
	activeModelRowsCache = nil
	activeModelRowsCacheTime = time.Time{}
}

// GetModelTagsByName 读取 models 表 tags 字段（支持 name_rule 匹配；未找到返回空串）。
func GetModelTagsByName(modelName string) string {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" || DB == nil {
		return ""
	}
	return resolveModelTagsFromRows(modelName, getActiveModelRowsCached())
}
