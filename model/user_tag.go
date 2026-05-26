package model

import "strings"

type UserTag struct {
	ID   int    `json:"id"`
	Name string `json:"name" gorm:"size:64;not null;uniqueIndex:uk_user_tag_name"`
	Note string `json:"note,omitempty" gorm:"type:varchar(255)"`
}

func GetAllUserTagNames() ([]string, error) {
	var tags []string
	err := DB.Model(&UserTag{}).
		Order("id ASC").
		Pluck("name", &tags).Error
	if err != nil {
		return nil, err
	}
	return tags, nil
}

func UpsertUserTags(tagNames []string) error {
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
		if err := DB.Where("name = ?", name).FirstOrCreate(&UserTag{}, &UserTag{Name: name}).Error; err != nil {
			return err
		}
	}
	return nil
}

func normalizeUserTags(tags []string) []string {
	result := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		name := strings.TrimSpace(tag)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		result = append(result, name)
	}
	return result
}

func splitUserTagsCSV(csv string) []string {
	if strings.TrimSpace(csv) == "" {
		return nil
	}
	return normalizeUserTags(strings.Split(csv, ","))
}

func JoinUserTags(tags []string) string {
	return strings.Join(normalizeUserTags(tags), ",")
}

func GetUserTagsList(tags string) []string {
	return splitUserTagsCSV(tags)
}
