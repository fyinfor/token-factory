package model

import (
	"sort"
	"strings"

	"gorm.io/gorm"
)

type ModelTag struct {
	ID   int    `json:"id"`
	Name string `json:"name" gorm:"size:64;not null;uniqueIndex:uk_model_tag_name"`
	Note string `json:"note,omitempty" gorm:"type:varchar(255)"`
}

func normalizeModelTagNames(tagNames []string) []string {
	cleaned := make([]string, 0, len(tagNames))
	seen := make(map[string]struct{}, len(tagNames))
	for _, tag := range tagNames {
		name := strings.TrimSpace(tag)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		cleaned = append(cleaned, name)
	}
	return cleaned
}

func splitModelTagsCSV(csv string) []string {
	if strings.TrimSpace(csv) == "" {
		return nil
	}
	return normalizeModelTagNames(strings.Split(csv, ","))
}

// CollectInUseModelTagNames 汇总 models.tags 中当前仍在使用的标签。
func CollectInUseModelTagNames() ([]string, error) {
	var allTagCSVs []string
	if err := DB.Model(&Model{}).Where("tags <> ?", "").Pluck("tags", &allTagCSVs).Error; err != nil {
		return nil, err
	}

	merged := make([]string, 0, len(allTagCSVs))
	for _, csv := range allTagCSVs {
		merged = append(merged, splitModelTagsCSV(csv)...)
	}
	return normalizeModelTagNames(merged), nil
}

// SyncModelTags 将 model_tags 表与给定标签列表对齐：保留/创建在用标签，删除未使用标签。
func SyncModelTags(tagNames []string) error {
	cleaned := normalizeModelTagNames(tagNames)
	return DB.Transaction(func(tx *gorm.DB) error {
		if len(cleaned) == 0 {
			return tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&ModelTag{}).Error
		}
		for _, name := range cleaned {
			if err := tx.Where("name = ?", name).FirstOrCreate(&ModelTag{}, &ModelTag{Name: name}).Error; err != nil {
				return err
			}
		}
		return tx.Where("name NOT IN ?", cleaned).Delete(&ModelTag{}).Error
	})
}

// SyncModelTagsFromModels 根据 models 表现有 tags 同步 model_tags 表并返回当前在用标签。
func SyncModelTagsFromModels() ([]string, error) {
	tags, err := CollectInUseModelTagNames()
	if err != nil {
		return nil, err
	}
	if err := SyncModelTags(tags); err != nil {
		return nil, err
	}
	sort.Strings(tags)
	return tags, nil
}

func GetAllModelTagNames() ([]string, error) {
	var tags []string
	err := DB.Model(&ModelTag{}).
		Order("name ASC").
		Pluck("name", &tags).Error
	if err != nil {
		return nil, err
	}
	return tags, nil
}
