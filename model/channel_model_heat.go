package model

import (
	"time"

	"gorm.io/gorm"
)

// ChannelModelHeat 存储渠道-模型组合的热力排序配置
type ChannelModelHeat struct {
	ID                 int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ChannelID          int       `gorm:"not null;index:idx_channel_model,unique;index:idx_channel" json:"channel_id"`
	ModelName          string    `gorm:"type:varchar(255);not null;index:idx_channel_model,unique;index:idx_model" json:"model_name"`
	ModelSortWeight    float64   `gorm:"type:double precision;default:1" json:"model_sort_weight"`
	ChannelSortWeight  float64   `gorm:"column:channel_sort_weight;type:double precision;default:1" json:"channel_sort_weight"`
	ManualBaseReqCount int64     `gorm:"column:manual_base_req_count;default:0" json:"manual_base_req_count"`
	CreatedAt          time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (ChannelModelHeat) TableName() string {
	return "channel_model_heats"
}

// GetChannelModelHeat 获取指定渠道-模型组合的热力配置
func GetChannelModelHeat(channelID int, modelName string) (*ChannelModelHeat, error) {
	var heat ChannelModelHeat
	err := DB.Where("channel_id = ? AND model_name = ?", channelID, modelName).First(&heat).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &heat, nil
}

// GetChannelModelHeatsByChannel 获取指定渠道的所有模型热力配置
func GetChannelModelHeatsByChannel(channelID int) ([]ChannelModelHeat, error) {
	var heats []ChannelModelHeat
	err := DB.Where("channel_id = ?", channelID).Find(&heats).Error
	return heats, err
}

// GetAllChannelModelHeats 获取所有渠道-模型组合的热力配置
func GetAllChannelModelHeats() ([]ChannelModelHeat, error) {
	var heats []ChannelModelHeat
	err := DB.Find(&heats).Error
	return heats, err
}

// SaveChannelModelHeat 保存或更新渠道-模型组合的热力配置
func SaveChannelModelHeat(heat *ChannelModelHeat) error {
	var existing ChannelModelHeat
	err := DB.Where("channel_id = ? AND model_name = ?", heat.ChannelID, heat.ModelName).First(&existing).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 创建新记录
			return DB.Create(heat).Error
		}
		return err
	}
	// 更新现有记录
	existing.ModelSortWeight = heat.ModelSortWeight
	existing.ChannelSortWeight = heat.ChannelSortWeight
	existing.ManualBaseReqCount = heat.ManualBaseReqCount
	return DB.Save(&existing).Error
}

// BatchSaveChannelModelHeats 批量保存渠道-模型组合的热力配置
func BatchSaveChannelModelHeats(heats []ChannelModelHeat) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		for _, heat := range heats {
			var existing ChannelModelHeat
			err := tx.Where("channel_id = ? AND model_name = ?", heat.ChannelID, heat.ModelName).First(&existing).Error
			if err != nil {
				if err == gorm.ErrRecordNotFound {
					if err := tx.Create(&heat).Error; err != nil {
						return err
					}
					continue
				}
				return err
			}
			existing.ModelSortWeight = heat.ModelSortWeight
			existing.ChannelSortWeight = heat.ChannelSortWeight
			existing.ManualBaseReqCount = heat.ManualBaseReqCount
			if err := tx.Save(&existing).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// DeleteChannelModelHeat 删除指定渠道-模型组合的热力配置
func DeleteChannelModelHeat(channelID int, modelName string) error {
	return DB.Where("channel_id = ? AND model_name = ?", channelID, modelName).Delete(&ChannelModelHeat{}).Error
}

// DeleteChannelModelHeatsByChannel 删除指定渠道的所有热力配置
func DeleteChannelModelHeatsByChannel(channelID int) error {
	return DB.Where("channel_id = ?", channelID).Delete(&ChannelModelHeat{}).Error
}
