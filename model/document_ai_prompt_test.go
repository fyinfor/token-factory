package model

import (
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestDocumentAIPromptSettingsLifecycle(t *testing.T) {
	previousDB := DB
	t.Cleanup(func() { DB = previousDB })

	db, err := gorm.Open(
		sqlite.Open(fmt.Sprintf("file:document_ai_prompts_%d?mode=memory&cache=shared", time.Now().UnixNano())),
		&gorm.Config{},
	)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&DocumentAIPromptSettings{}); err != nil {
		t.Fatalf("migrate sqlite: %v", err)
	}
	DB = db

	settings, err := GetDocumentAIPromptSettings()
	if err != nil || settings != nil {
		t.Fatalf("expected no custom settings: settings=%+v err=%v", settings, err)
	}

	if err := SaveDocumentAIPromptSettings(&DocumentAIPromptSettings{
		PolishPrompt:    "  polish v1  ",
		TranslatePrompt: "  translate v1  ",
	}); err != nil {
		t.Fatalf("save settings: %v", err)
	}
	settings, err = GetDocumentAIPromptSettings()
	if err != nil {
		t.Fatalf("load settings: %v", err)
	}
	if settings.PolishPrompt != "polish v1" || settings.TranslatePrompt != "translate v1" || settings.UpdatedTime <= 0 {
		t.Fatalf("unexpected saved settings: %+v", settings)
	}

	if err := SaveDocumentAIPromptSettings(&DocumentAIPromptSettings{
		PolishPrompt:    "polish v2",
		TranslatePrompt: "translate v2",
	}); err != nil {
		t.Fatalf("update settings: %v", err)
	}
	var count int64
	if err := DB.Model(&DocumentAIPromptSettings{}).Count(&count).Error; err != nil {
		t.Fatalf("count settings: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected a single settings row, got %d", count)
	}
	settings, err = GetDocumentAIPromptSettings()
	if err != nil || settings.PolishPrompt != "polish v2" || settings.TranslatePrompt != "translate v2" {
		t.Fatalf("unexpected updated settings: settings=%+v err=%v", settings, err)
	}

	if err := ResetDocumentAIPromptSettings(); err != nil {
		t.Fatalf("reset settings: %v", err)
	}
	settings, err = GetDocumentAIPromptSettings()
	if err != nil || settings != nil {
		t.Fatalf("expected settings to be reset: settings=%+v err=%v", settings, err)
	}
}
