package model

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	ChannelModelHotOverrideForceHot    = "force_hot"
	ChannelModelHotOverrideForceNotHot = "force_not_hot"
	ChannelModelHotOverrideMaxRank     = 1000000
)

// ChannelModelHotOverride stores an explicit homepage hot-state override for
// one channel-model pair. Missing rows mean "follow automatic ranking".
type ChannelModelHotOverride struct {
	ID           int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ChannelID    int       `gorm:"not null;uniqueIndex:idx_channel_model_hot_override,priority:1;index" json:"channel_id"`
	ModelName    string    `gorm:"type:varchar(255);not null;uniqueIndex:idx_channel_model_hot_override,priority:2;index" json:"model_name"`
	OverrideMode string    `gorm:"type:varchar(32);not null" json:"override_mode"`
	ManualRank   int       `gorm:"not null;default:0" json:"manual_rank"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (ChannelModelHotOverride) TableName() string {
	return "channel_model_hot_overrides"
}

func NormalizeChannelModelHotOverrideMode(mode string) string {
	return strings.ToLower(strings.TrimSpace(mode))
}

func IsValidChannelModelHotOverrideMode(mode string) bool {
	switch NormalizeChannelModelHotOverrideMode(mode) {
	case ChannelModelHotOverrideForceHot, ChannelModelHotOverrideForceNotHot:
		return true
	default:
		return false
	}
}

func GetAllChannelModelHotOverrides() ([]ChannelModelHotOverride, error) {
	var overrides []ChannelModelHotOverride
	err := DB.Order("manual_rank ASC, id ASC").Find(&overrides).Error
	return overrides, err
}

// SaveChannelModelHotOverride creates or updates a non-automatic override.
// Passing an empty mode restores automatic ranking by deleting the row.
func SaveChannelModelHotOverride(override *ChannelModelHotOverride) error {
	if override == nil || override.ChannelID <= 0 || strings.TrimSpace(override.ModelName) == "" {
		return errors.New("invalid channel-model hot override")
	}
	override.ModelName = strings.TrimSpace(override.ModelName)
	override.OverrideMode = NormalizeChannelModelHotOverrideMode(override.OverrideMode)
	if override.OverrideMode == "" || override.OverrideMode == "auto" {
		return DeleteChannelModelHotOverride(override.ChannelID, override.ModelName)
	}
	if !IsValidChannelModelHotOverrideMode(override.OverrideMode) {
		return errors.New("invalid hot override mode")
	}
	if override.ManualRank < 0 {
		override.ManualRank = 0
	} else if override.ManualRank > ChannelModelHotOverrideMaxRank {
		override.ManualRank = ChannelModelHotOverrideMaxRank
	}

	var existing ChannelModelHotOverride
	err := DB.Where("channel_id = ? AND model_name = ?", override.ChannelID, override.ModelName).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return DB.Create(override).Error
	}
	if err != nil {
		return err
	}
	existing.OverrideMode = override.OverrideMode
	existing.ManualRank = override.ManualRank
	return DB.Save(&existing).Error
}

func BatchSaveChannelModelHotOverrides(overrides []ChannelModelHotOverride) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		for i := range overrides {
			override := overrides[i]
			if override.ChannelID <= 0 || strings.TrimSpace(override.ModelName) == "" {
				return errors.New("invalid channel-model hot override")
			}
			override.ModelName = strings.TrimSpace(override.ModelName)
			override.OverrideMode = NormalizeChannelModelHotOverrideMode(override.OverrideMode)
			if override.ManualRank < 0 {
				override.ManualRank = 0
			} else if override.ManualRank > ChannelModelHotOverrideMaxRank {
				override.ManualRank = ChannelModelHotOverrideMaxRank
			}
			if override.OverrideMode == "" || override.OverrideMode == "auto" {
				if err := tx.Where("channel_id = ? AND model_name = ?", override.ChannelID, override.ModelName).
					Delete(&ChannelModelHotOverride{}).Error; err != nil {
					return err
				}
				continue
			}
			if !IsValidChannelModelHotOverrideMode(override.OverrideMode) {
				return errors.New("invalid hot override mode")
			}

			var existing ChannelModelHotOverride
			err := tx.Where("channel_id = ? AND model_name = ?", override.ChannelID, override.ModelName).First(&existing).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if err := tx.Create(&override).Error; err != nil {
					return err
				}
				continue
			}
			if err != nil {
				return err
			}
			existing.OverrideMode = override.OverrideMode
			existing.ManualRank = override.ManualRank
			if err := tx.Save(&existing).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func DeleteChannelModelHotOverride(channelID int, modelName string) error {
	return DB.Where("channel_id = ? AND model_name = ?", channelID, strings.TrimSpace(modelName)).
		Delete(&ChannelModelHotOverride{}).Error
}

func DeleteChannelModelHotOverridesByChannel(channelID int) error {
	return DB.Where("channel_id = ?", channelID).Delete(&ChannelModelHotOverride{}).Error
}
