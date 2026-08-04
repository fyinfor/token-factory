package model

import (
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/setting/system_setting"
	"gorm.io/gorm"
)

// MaterialGroup 素材库分组。用户首次上传素材时自动创建。
// GroupType 区分虚拟分组（virtual）与真人认证分组（real）：
//   - virtual：虚拟人像素材组，一个用户最多一个（原有 ensureMaterialGroup 流程）；
//   - real：真人认证成功后创建的分组，一个用户可有多个。
//
// 分组命名规则：格式化后的服务器地址_用户ID（移除 http:// 或 https://，"." 替换为 "_"）。
type MaterialGroup struct {
	Id          int    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId      int    `json:"user_id" gorm:"index:idx_material_group_user;not null"`
	GroupName   string `json:"group_name" gorm:"type:varchar(255);not null"`
	Description string `json:"description" gorm:"type:varchar(512);default:''"`
	GroupId     string `json:"group_id" gorm:"type:varchar(128);index"` // 上游素材库接口返回的分组 ID
	GroupType   string `json:"group_type" gorm:"type:varchar(32);default:'virtual';index:idx_material_group_type"`
	CreatedAt   int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt   int64  `json:"updated_at" gorm:"bigint"`
}

func (MaterialGroup) TableName() string {
	return "material_groups"
}

// MaterialGroupType 素材分组类型枚举。
const (
	MaterialGroupTypeVirtual = "virtual"
	MaterialGroupTypeReal    = "real"
)

// MaterialAsset 素材记录。每次上传成功后落库，供素材列表查询及资源地址复制使用。
// GroupType 标识素材所属分组类型（virtual/real），与 MaterialGroup.GroupType 对齐，
// 用于虚拟人像与真人素材的业务隔离。
type MaterialAsset struct {
	Id        int    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId    int    `json:"user_id" gorm:"index;not null"`
	GroupId   string `json:"group_id" gorm:"type:varchar(128);index"` // 上游分组 ID
	GroupType string `json:"group_type" gorm:"type:varchar(32);default:'virtual';index"`
	AssetId   string `json:"asset_id" gorm:"type:varchar(128);index"` // 上游素材 ID，如 asset-xxxx
	Name      string `json:"name" gorm:"type:varchar(255)"`
	AssetType string `json:"asset_type" gorm:"type:varchar(32)"` // Image / Video / Audio
	URL       string `json:"url" gorm:"type:text"`               // 上传后生成的公网 URL
	Status    string `json:"status" gorm:"type:varchar(32)"`     // Active / Pending / Failed
	CreatedAt int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt int64  `json:"updated_at" gorm:"bigint"`
}

func (MaterialAsset) TableName() string {
	return "material_assets"
}

// FormatServerAddressForGroup 按分组命名规则格式化服务器地址：
// 移除 http:// 或 https:// 前缀，将 "." 替换为 "_"，并去除端口/路径中的非法字符。
func FormatServerAddressForGroup(serverAddress string) string {
	addr := strings.TrimSpace(serverAddress)
	addr = strings.TrimPrefix(addr, "https://")
	addr = strings.TrimPrefix(addr, "http://")
	addr = strings.TrimRight(addr, "/")
	// 仅保留主机部分，去掉可能存在的路径
	if idx := strings.Index(addr, "/"); idx >= 0 {
		addr = addr[:idx]
	}
	addr = strings.ReplaceAll(addr, ".", "_")
	addr = strings.ReplaceAll(addr, ":", "_")
	return addr
}

// BuildMaterialGroupName 根据系统通用设置中的服务器地址与用户 ID 生成分组名称。
func BuildMaterialGroupName(userId int) string {
	prefix := FormatServerAddressForGroup(system_setting.ServerAddress)
	if prefix == "" {
		prefix = "local"
	}
	return prefix + "_" + strconv.Itoa(userId)
}

// GetMaterialGroupByUserId 查询用户的虚拟素材分组，不存在时返回 (nil, nil)。
// 兼容旧数据：group_type 为空时视为 virtual。
func GetMaterialGroupByUserId(userId int) (*MaterialGroup, error) {
	var group MaterialGroup
	err := DB.Where("user_id = ? AND (group_type = ? OR group_type = '' OR group_type IS NULL)", userId, MaterialGroupTypeVirtual).First(&group).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &group, nil
}

// CreateMaterialGroup 落库新建素材分组。
func CreateMaterialGroup(group *MaterialGroup) error {
	now := time.Now().Unix()
	group.CreatedAt = now
	group.UpdatedAt = now
	return DB.Create(group).Error
}

// CreateMaterialAsset 落库新建素材记录。
func CreateMaterialAsset(asset *MaterialAsset) error {
	now := time.Now().Unix()
	asset.CreatedAt = now
	asset.UpdatedAt = now
	return DB.Create(asset).Error
}

// ListMaterialAssets 分页查询用户素材列表（按创建时间倒序）。
func ListMaterialAssets(userId int, groupId string, offset int, limit int) ([]*MaterialAsset, int64, error) {
	var assets []*MaterialAsset
	var total int64
	query := DB.Model(&MaterialAsset{}).Where("user_id = ?", userId)
	if groupId != "" {
		query = query.Where("group_id = ?", groupId)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Order("id desc").Offset(offset).Limit(limit).Find(&assets).Error
	if err != nil {
		return nil, 0, err
	}
	return assets, total, nil
}

// UpdateMaterialAssetStatus 更新素材状态（用于状态刷新）。
func UpdateMaterialAssetStatus(id int, status string) error {
	return DB.Model(&MaterialAsset{}).Where("id = ?", id).
		Updates(map[string]interface{}{"status": status, "updated_at": time.Now().Unix()}).Error
}

// UpdateMaterialAssetInfo 刷新素材的状态/URL/类型（GetAsset 轮询拿到上游永久 URL 时使用）。
// 仅更新非空字段，避免空值覆盖已有数据。
func UpdateMaterialAssetInfo(id int, status string, url string, assetType string) error {
	updates := map[string]interface{}{"updated_at": time.Now().Unix()}
	if strings.TrimSpace(status) != "" {
		updates["status"] = status
	}
	if strings.TrimSpace(url) != "" {
		updates["url"] = url
	}
	if strings.TrimSpace(assetType) != "" {
		updates["asset_type"] = assetType
	}
	return DB.Model(&MaterialAsset{}).Where("id = ?", id).Updates(updates).Error
}

// UpdateMaterialAssetGroupType 更新素材的分组类型（用于真人素材上传后标记）。
func UpdateMaterialAssetGroupType(id int, groupType string) error {
	return DB.Model(&MaterialAsset{}).Where("id = ?", id).
		Updates(map[string]interface{}{"group_type": groupType, "updated_at": time.Now().Unix()}).Error
}

// UpdateMaterialAssetName 更新素材名称（UpdateAsset 本地同步）。
func UpdateMaterialAssetName(id int, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	return DB.Model(&MaterialAsset{}).Where("id = ?", id).
		Updates(map[string]interface{}{"name": name, "updated_at": time.Now().Unix()}).Error
}

// GetMaterialAssetByAssetIdAndUser 按上游 asset_id + user_id 查询素材，不存在时返回 (nil, nil)。
func GetMaterialAssetByAssetIdAndUser(assetId string, userId int) (*MaterialAsset, error) {
	var asset MaterialAsset
	err := DB.Where("asset_id = ? AND user_id = ?", assetId, userId).First(&asset).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &asset, nil
}

// GetMaterialAssetByIdAndUser 按主键查询素材并校验归属用户，不存在时返回 (nil, nil)。
func GetMaterialAssetByIdAndUser(id int, userId int) (*MaterialAsset, error) {
	var asset MaterialAsset
	err := DB.Where("id = ? AND user_id = ?", id, userId).First(&asset).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &asset, nil
}

// DeleteMaterialAsset 按主键物理删除素材记录。
func DeleteMaterialAsset(id int) error {
	return DB.Delete(&MaterialAsset{}, id).Error
}

// ---------------------------------------------------------------------------
// 真人认证分组与素材查询
// ---------------------------------------------------------------------------

// GetRealMaterialGroupsByUserId 查询用户的所有真人认证分组（按创建时间倒序）。
func GetRealMaterialGroupsByUserId(userId int) ([]*MaterialGroup, error) {
	var groups []*MaterialGroup
	err := DB.Where("user_id = ? AND group_type = ?", userId, MaterialGroupTypeReal).
		Order("id desc").Find(&groups).Error
	if err != nil {
		return nil, err
	}
	return groups, nil
}

// GetMaterialGroupByGroupIdAndUser 按上游分组 ID + 用户 ID 查询分组，校验归属。
func GetMaterialGroupByGroupIdAndUser(groupId string, userId int) (*MaterialGroup, error) {
	var group MaterialGroup
	err := DB.Where("group_id = ? AND user_id = ?", groupId, userId).First(&group).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &group, nil
}

// GetMaterialGroupByGroupId 按上游分组 ID 查询分组（不限定用户），用于认证成功去重。
func GetMaterialGroupByGroupId(groupId string) (*MaterialGroup, error) {
	var group MaterialGroup
	err := DB.Where("group_id = ?", groupId).First(&group).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &group, nil
}

// DeleteMaterialGroup 按主键物理删除分组记录。
func DeleteMaterialGroup(id int) error {
	return DB.Delete(&MaterialGroup{}, id).Error
}

// UpdateMaterialGroup 更新素材分组的名称和描述（仅允许分组成员本人修改）。
// groupName 为空时不更新名称；description 允许清空。
func UpdateMaterialGroup(id int, userId int, groupName string, description string) error {
	updates := map[string]interface{}{"updated_at": time.Now().Unix()}
	name := strings.TrimSpace(groupName)
	if name != "" {
		updates["group_name"] = name
	}
	updates["description"] = strings.TrimSpace(description)
	return DB.Model(&MaterialGroup{}).Where("id = ? AND user_id = ?", id, userId).Updates(updates).Error
}

// ListMaterialAssetsByGroupType 按分组类型分页查询用户素材（按创建时间倒序）。
// groupType 为空时查全部类型。
func ListMaterialAssetsByGroupType(userId int, groupType string, offset int, limit int) ([]*MaterialAsset, int64, error) {
	var assets []*MaterialAsset
	var total int64
	query := DB.Model(&MaterialAsset{}).Where("user_id = ?", userId)
	if groupType != "" {
		query = query.Where("group_type = ?", groupType)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Order("id desc").Offset(offset).Limit(limit).Find(&assets).Error
	if err != nil {
		return nil, 0, err
	}
	return assets, total, nil
}

// MaterialGroupListFilter Action ListAssetGroups 本地筛选条件。
// UserId=0 表示不按用户过滤（管理员查全部）。
type MaterialGroupListFilter struct {
	UserId    int
	GroupType string   // virtual / real；空表示不限
	GroupIds  []string // 上游 group_id 列表
	Name      string   // 名称模糊匹配
	SortBy    string   // created_at / updated_at
	SortOrder string   // asc / desc
	Offset    int
	Limit     int
}

// MaterialAssetListFilter Action ListAssets 本地筛选条件。
// UserId=0 表示不按用户过滤（管理员查全部）。
type MaterialAssetListFilter struct {
	UserId    int
	GroupType string   // virtual / real；空表示不限
	GroupIds  []string // 上游 group_id 列表
	Statuses  []string // Active / Pending / Failed
	Name      string   // 名称模糊匹配
	SortBy    string   // created_at / updated_at / group_id
	SortOrder string   // asc / desc
	Offset    int
	Limit     int
}

func applyMaterialSort(query *gorm.DB, sortBy string, sortOrder string, defaultCol string) *gorm.DB {
	col := strings.TrimSpace(sortBy)
	if col == "" {
		col = defaultCol
	}
	order := strings.ToLower(strings.TrimSpace(sortOrder))
	if order != "asc" {
		order = "desc"
	}
	return query.Order(col + " " + order)
}

func applyMaterialGroupTypeFilter(query *gorm.DB, groupType string) *gorm.DB {
	groupType = strings.TrimSpace(groupType)
	if groupType == "" {
		return query
	}
	if groupType == MaterialGroupTypeVirtual {
		// 兼容旧数据：group_type 为空时视为 virtual
		return query.Where("(group_type = ? OR group_type = '' OR group_type IS NULL)", MaterialGroupTypeVirtual)
	}
	return query.Where("group_type = ?", groupType)
}

// ListMaterialGroupsFiltered 按筛选条件分页查询素材组（供 ListAssetGroups Action 使用）。
func ListMaterialGroupsFiltered(filter MaterialGroupListFilter) ([]*MaterialGroup, int64, error) {
	var groups []*MaterialGroup
	var total int64
	query := DB.Model(&MaterialGroup{})
	if filter.UserId > 0 {
		query = query.Where("user_id = ?", filter.UserId)
	}
	query = applyMaterialGroupTypeFilter(query, filter.GroupType)
	if len(filter.GroupIds) > 0 {
		query = query.Where("group_id IN ?", filter.GroupIds)
	}
	if name := strings.TrimSpace(filter.Name); name != "" {
		query = query.Where("group_name LIKE ?", "%"+name+"%")
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	query = applyMaterialSort(query, filter.SortBy, filter.SortOrder, "created_at")
	err := query.Offset(filter.Offset).Limit(filter.Limit).Find(&groups).Error
	if err != nil {
		return nil, 0, err
	}
	return groups, total, nil
}

// ListMaterialAssetsFiltered 按筛选条件分页查询素材（供 ListAssets Action 使用）。
func ListMaterialAssetsFiltered(filter MaterialAssetListFilter) ([]*MaterialAsset, int64, error) {
	var assets []*MaterialAsset
	var total int64
	query := DB.Model(&MaterialAsset{})
	if filter.UserId > 0 {
		query = query.Where("user_id = ?", filter.UserId)
	}
	query = applyMaterialGroupTypeFilter(query, filter.GroupType)
	if len(filter.GroupIds) > 0 {
		query = query.Where("group_id IN ?", filter.GroupIds)
	}
	if len(filter.Statuses) > 0 {
		query = query.Where("status IN ?", filter.Statuses)
	}
	if name := strings.TrimSpace(filter.Name); name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	query = applyMaterialSort(query, filter.SortBy, filter.SortOrder, "created_at")
	err := query.Offset(filter.Offset).Limit(filter.Limit).Find(&assets).Error
	if err != nil {
		return nil, 0, err
	}
	return assets, total, nil
}

// ---------------------------------------------------------------------------
// 真人认证会话（BytedToken 仅后端存储，禁止前端传输/展示）
// ---------------------------------------------------------------------------

// MaterialVisualSession 真人认证会话记录。BytedToken 仅用于后端轮询，json:"-" 防止序列化泄露。
type MaterialVisualSession struct {
	Id           int    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId       int    `json:"user_id" gorm:"index;not null"`
	BytedToken   string `json:"-" gorm:"type:varchar(255);index"`        // 仅后端轮询使用，禁止前端传输
	H5Link       string `json:"h5_link" gorm:"type:text"`                // H5 认证链接（5 分钟有效）
	QrCode       string `json:"qr_code" gorm:"type:text"`                // H5 链接的 base64 二维码
	Status       string `json:"status" gorm:"type:varchar(32);index"`    // pending/success/failed/expired
	GroupId      string `json:"group_id" gorm:"type:varchar(128);index"` // 认证成功后的真人分组 ID
	ErrorMessage string `json:"error_message" gorm:"type:text"`          // 失败原因
	ExpiresAt    int64  `json:"expires_at" gorm:"bigint"`                // 会话过期时间（created_at + 300s）
	CreatedAt    int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt    int64  `json:"updated_at" gorm:"bigint"`
}

func (MaterialVisualSession) TableName() string {
	return "material_visual_sessions"
}

// VisualSessionStatus 真人认证会话状态枚举。
const (
	VisualSessionStatusPending = "pending"
	VisualSessionStatusSuccess = "success"
	VisualSessionStatusFailed  = "failed"
	VisualSessionStatusExpired = "expired"
)

// CreateVisualSession 落库新建真人认证会话。
func CreateVisualSession(session *MaterialVisualSession) error {
	now := time.Now().Unix()
	session.CreatedAt = now
	session.UpdatedAt = now
	return DB.Create(session).Error
}

// GetVisualSessionByIdAndUser 按主键 + 用户 ID 查询会话，校验归属。
func GetVisualSessionByIdAndUser(id int, userId int) (*MaterialVisualSession, error) {
	var session MaterialVisualSession
	err := DB.Where("id = ? AND user_id = ?", id, userId).First(&session).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &session, nil
}

// GetVisualSessionByBytedTokenAndUser 按 BytedToken + 用户 ID 查询会话，校验归属。
func GetVisualSessionByBytedTokenAndUser(bytedToken string, userId int) (*MaterialVisualSession, error) {
	var session MaterialVisualSession
	err := DB.Where("byted_token = ? AND user_id = ?", bytedToken, userId).First(&session).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &session, nil
}

// UpdateVisualSessionStatus 更新会话状态（及可选的分组 ID / 错误信息）。
func UpdateVisualSessionStatus(id int, status string, groupId string, errorMessage string) error {
	updates := map[string]interface{}{"status": status, "updated_at": time.Now().Unix()}
	if strings.TrimSpace(groupId) != "" {
		updates["group_id"] = groupId
	}
	if errorMessage != "" {
		updates["error_message"] = errorMessage
	}
	return DB.Model(&MaterialVisualSession{}).Where("id = ?", id).Updates(updates).Error
}

// migrateMaterialGroupType 迁移素材分组类型字段：
//  1. 删除 material_groups 上旧的 user_id 唯一索引（idx_material_group_user），允许一个用户拥有多个分组（虚拟 + 真人）。
//  2. 回填 group_type 为空/NULL 的旧记录为 "virtual"（material_groups 与 material_assets 两表）。
//
// 跨库兼容：DB.Migrator().DropIndex 在 SQLite/MySQL/PostgreSQL 上均以索引名为参数，不涉及列引用差异。
func migrateMaterialGroupType() error {
	// 删除旧唯一索引（AutoMigrate 已按新 tag 创建普通索引 idx_material_group_user，此处仅清理旧约束）。
	_ = DB.Migrator().DropIndex(&MaterialGroup{}, "idx_material_group_user")

	// 回填 material_groups.group_type 为空/NULL 的旧记录为 virtual。
	if err := DB.Model(&MaterialGroup{}).Where("group_type = '' OR group_type IS NULL").
		Update("group_type", MaterialGroupTypeVirtual).Error; err != nil {
		return err
	}
	// 回填 material_assets.group_type 为空/NULL 的旧记录为 virtual。
	if err := DB.Model(&MaterialAsset{}).Where("group_type = '' OR group_type IS NULL").
		Update("group_type", MaterialGroupTypeVirtual).Error; err != nil {
		return err
	}
	return nil
}
