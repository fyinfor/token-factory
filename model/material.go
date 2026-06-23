package model

import (
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/setting/system_setting"
	"gorm.io/gorm"
)

// MaterialGroup 素材库分组。用户首次上传素材时自动创建，一个用户对应一个分组。
// 分组命名规则：格式化后的服务器地址_用户ID（移除 http:// 或 https://，"." 替换为 "_"）。
type MaterialGroup struct {
	Id        int    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId    int    `json:"user_id" gorm:"uniqueIndex:idx_material_group_user;not null"`
	GroupName string `json:"group_name" gorm:"type:varchar(255);not null"`
	GroupId   string `json:"group_id" gorm:"type:varchar(128);index"` // 上游素材库接口返回的分组 ID
	CreatedAt int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt int64  `json:"updated_at" gorm:"bigint"`
}

func (MaterialGroup) TableName() string {
	return "material_groups"
}

// MaterialAsset 素材记录。每次上传成功后落库，供素材列表查询及资源地址复制使用。
type MaterialAsset struct {
	Id        int    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserId    int    `json:"user_id" gorm:"index;not null"`
	GroupId   string `json:"group_id" gorm:"type:varchar(128);index"` // 上游分组 ID
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

// GetMaterialGroupByUserId 查询用户的素材分组，不存在时返回 (nil, nil)。
func GetMaterialGroupByUserId(userId int) (*MaterialGroup, error) {
	var group MaterialGroup
	err := DB.Where("user_id = ?", userId).First(&group).Error
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
