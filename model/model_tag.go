package model

import "strings"

type ModelTag struct {
	ID   int    `json:"id"`
	Name string `json:"name" gorm:"size:64;not null;uniqueIndex:uk_model_tag_name"`
	Note string `json:"note,omitempty" gorm:"type:varchar(255)"`
}

func GetAllModelTagNames() ([]string, error) {
	var tags []string
	err := DB.Model(&ModelTag{}).
		Order("id ASC").
		Pluck("name", &tags).Error
	if err != nil {
		return nil, err
	}
	return tags, nil
}

func UpsertModelTags(tagNames []string) error {
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
	if len(cleaned) == 0 {
		return nil
	}
	for _, name := range cleaned {
		if err := DB.Where("name = ?", name).FirstOrCreate(&ModelTag{}, &ModelTag{Name: name}).Error; err != nil {
			return err
		}
	}
	return nil
}
