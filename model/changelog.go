package model

import (
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

type Changelog struct {
	Id        int    `json:"id" gorm:"primaryKey"`
	Date      string `json:"date" gorm:"type:varchar(32);not null;index"`
	Content   string `json:"content" gorm:"type:text;not null"`
	CreatedAt int64  `json:"created_at" gorm:"bigint;index"`
	UpdatedAt int64  `json:"updated_at" gorm:"bigint"`
}

func (Changelog) TableName() string {
	return "changelogs"
}

func normalizeChangelogForSave(changelog *Changelog) {
	changelog.Date = strings.TrimSpace(changelog.Date)
	changelog.Content = strings.TrimSpace(changelog.Content)
	now := time.Now().Unix()
	if changelog.CreatedAt == 0 {
		changelog.CreatedAt = now
	}
	changelog.UpdatedAt = now
}

func ListChangelogs(offset, limit int) ([]*Changelog, int64, error) {
	if limit <= 0 {
		limit = 10
	}
	var total int64
	if err := DB.Model(&Changelog{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var changelogs []*Changelog
	err := DB.Order("date DESC").Order("id DESC").Offset(offset).Limit(limit).Find(&changelogs).Error
	return changelogs, total, err
}

func ListAllChangelogs() ([]*Changelog, error) {
	var changelogs []*Changelog
	err := DB.Order("date DESC").Order("id DESC").Find(&changelogs).Error
	return changelogs, err
}

func CreateChangelog(changelog *Changelog) error {
	normalizeChangelogForSave(changelog)
	return DB.Create(changelog).Error
}

func UpdateChangelog(changelog *Changelog) error {
	normalizeChangelogForSave(changelog)
	return DB.Model(&Changelog{}).
		Where("id = ?", changelog.Id).
		Updates(map[string]interface{}{
			"date":       changelog.Date,
			"content":    changelog.Content,
			"updated_at": changelog.UpdatedAt,
		}).Error
}

func DeleteChangelog(id int) error {
	return DB.Delete(&Changelog{}, id).Error
}

type legacyChangelogEntry struct {
	Date    string `json:"date"`
	Content string `json:"content"`
}

func MigrateLegacyChangelogOption() error {
	var count int64
	if err := DB.Model(&Changelog{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	var option Option
	if err := DB.Where(commonKeyCol+" = ?", "Changelog").First(&option).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}
	if strings.TrimSpace(option.Value) == "" {
		return nil
	}

	var legacyEntries []legacyChangelogEntry
	if err := common.UnmarshalJsonStr(option.Value, &legacyEntries); err != nil {
		return nil
	}
	for _, legacyEntry := range legacyEntries {
		changelog := Changelog{
			Date:    legacyEntry.Date,
			Content: legacyEntry.Content,
		}
		if strings.TrimSpace(changelog.Date) == "" || strings.TrimSpace(changelog.Content) == "" {
			continue
		}
		if err := CreateChangelog(&changelog); err != nil {
			return err
		}
	}
	return nil
}
