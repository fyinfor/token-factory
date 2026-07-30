package model

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const documentAIPromptSettingsID = 1

type DocumentAIPromptSettings struct {
	ID              int    `json:"-" gorm:"primaryKey"`
	PolishPrompt    string `json:"polish_prompt" gorm:"type:text;not null"`
	TranslatePrompt string `json:"translate_prompt" gorm:"type:text;not null"`
	UpdatedTime     int64  `json:"updated_time" gorm:"bigint"`
}

func GetDocumentAIPromptSettings() (*DocumentAIPromptSettings, error) {
	var settings DocumentAIPromptSettings
	err := DB.First(&settings, documentAIPromptSettingsID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &settings, nil
}

func SaveDocumentAIPromptSettings(settings *DocumentAIPromptSettings) error {
	if settings == nil || strings.TrimSpace(settings.PolishPrompt) == "" || strings.TrimSpace(settings.TranslatePrompt) == "" {
		return gorm.ErrInvalidData
	}
	settings.ID = documentAIPromptSettingsID
	settings.PolishPrompt = strings.TrimSpace(settings.PolishPrompt)
	settings.TranslatePrompt = strings.TrimSpace(settings.TranslatePrompt)
	settings.UpdatedTime = common.GetTimestamp()
	return DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"polish_prompt",
			"translate_prompt",
			"updated_time",
		}),
	}).Create(settings).Error
}

func ResetDocumentAIPromptSettings() error {
	return DB.Delete(&DocumentAIPromptSettings{}, documentAIPromptSettingsID).Error
}
