package model

import (
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestChannelModelDocsUseChannelIDAndModelName(t *testing.T) {
	previousDB := DB
	t.Cleanup(func() { DB = previousDB })

	db, err := gorm.Open(
		sqlite.Open(fmt.Sprintf("file:channel_model_docs_%d?mode=memory&cache=shared", time.Now().UnixNano())),
		&gorm.Config{},
	)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Channel{}, &Ability{}, &Model{}, &ChannelModelDoc{}, &ModelTestResult{}); err != nil {
		t.Fatalf("migrate sqlite: %v", err)
	}
	DB = db

	meta := Model{ModelName: "demo-model", NameRule: NameRuleExact, Status: 1, DocIntroduction: "model fallback", ApiDocs: "[]"}
	if err := DB.Create(&meta).Error; err != nil {
		t.Fatalf("create model: %v", err)
	}
	channels := []Channel{
		{Name: "channel-a", RouteSlug: "a", Status: 1, Models: "demo-model"},
		{Name: "channel-b", RouteSlug: "b", Status: 2, Models: "demo-model"},
	}
	if err := DB.Create(&channels).Error; err != nil {
		t.Fatalf("create channels: %v", err)
	}
	for _, channel := range channels {
		if err := DB.Create(&Ability{Group: "default", Model: "demo-model", ChannelId: channel.Id, Enabled: channel.Status == 1}).Error; err != nil {
			t.Fatalf("create ability: %v", err)
		}
	}

	items, err := ListChannelModelDocItems(&meta)
	if err != nil || len(items) != 2 {
		t.Fatalf("list fallback docs: len=%d err=%v", len(items), err)
	}
	if items[0].Configured || items[0].DocIntroduction != "model fallback" {
		t.Fatalf("expected model fallback before channel override: %+v", items[0])
	}

	if err := UpsertChannelModelDoc(&ChannelModelDoc{ChannelID: channels[0].Id, ModelName: "demo-model"}); err != nil {
		t.Fatalf("save channel doc: %v", err)
	}
	if err := UpsertChannelModelDoc(&ChannelModelDoc{
		ChannelID:         channels[1].Id,
		ModelName:         "demo-model",
		DocIntroduction:   "channel-b docs",
		ApiDocs:           `[{"path":"/v1/demo"}]`,
		ApiDocsMarkdown:   "```http\nPOST /v1/demo\n```",
		ApiDocsMarkdownEn: "```http\nPOST /v1/demo\n```\n\nEnglish docs",
	}); err != nil {
		t.Fatalf("save second channel doc: %v", err)
	}
	if err := UpsertChannelModelDoc(&ChannelModelDoc{
		ChannelID:         channels[1].Id,
		ModelName:         "demo-model",
		DocIntroduction:   "channel-b updated",
		ApiDocs:           `[{"path":"/v1/demo-updated"}]`,
		ApiDocsMarkdown:   "```http\nPOST /v1/demo-updated\n```",
		ApiDocsMarkdownEn: "```http\nPOST /v1/demo-updated\n```\n\nUpdated English docs",
	}); err != nil {
		t.Fatalf("update second channel doc: %v", err)
	}
	var secondDoc ChannelModelDoc
	if err := DB.Where("channel_id = ? AND model_name = ?", channels[1].Id, "demo-model").First(&secondDoc).Error; err != nil {
		t.Fatalf("load updated second channel doc: %v", err)
	}
	if secondDoc.DocIntroduction != "channel-b updated" {
		t.Fatalf("expected upsert to update channel doc: %+v", secondDoc)
	}
	if secondDoc.ApiDocsMarkdown != "```http\nPOST /v1/demo-updated\n```" {
		t.Fatalf("expected upsert to update Markdown document: %+v", secondDoc)
	}
	if secondDoc.ApiDocsMarkdownEn != "```http\nPOST /v1/demo-updated\n```\n\nUpdated English docs" {
		t.Fatalf("expected upsert to update English Markdown document: %+v", secondDoc)
	}
	templates, err := ListChannelModelDocTemplates()
	if err != nil {
		t.Fatalf("list channel document templates: %v", err)
	}
	if len(templates) != 1 {
		t.Fatalf("expected only the non-empty channel document template, got %d", len(templates))
	}
	if templates[0].ChannelID != channels[1].Id || templates[0].ChannelName != "channel-b" || templates[0].ApiDocsMarkdown != secondDoc.ApiDocsMarkdown {
		t.Fatalf("unexpected channel document template: %+v", templates[0])
	}
	pricingItems := BuildPricingAPIItems(
		[]Pricing{{
			ModelName:       "demo-model",
			DocIntroduction: "pricing fallback",
			ApiDocs:         []any{},
		}},
		map[int]struct{}{channels[0].Id: {}, channels[1].Id: {}},
		[]ChannelPricingMeta{
			{ChannelID: channels[0].Id, Models: "demo-model"},
			{ChannelID: channels[1].Id, Models: "demo-model"},
		},
		true,
	)
	if len(pricingItems) != 2 {
		t.Fatalf("expected one pricing item per channel, got %d", len(pricingItems))
	}
	pricingDocs := make(map[int]PricingChannelItem, len(pricingItems))
	for _, item := range pricingItems {
		if len(item.ChannelList) != 1 {
			t.Fatalf("expected flattened channel pricing item: %+v", item)
		}
		channelItem := item.ChannelList[0]
		pricingDocs[channelItem.ChannelID] = channelItem
	}
	if item := pricingDocs[channels[0].Id]; !item.DocConfigured || item.DocIntroduction != "" || item.ApiDocsMarkdown != "" {
		t.Fatalf("expected explicit empty channel document in pricing response: %+v", item)
	}
	if item := pricingDocs[channels[1].Id]; !item.DocConfigured || item.ApiDocsMarkdown != secondDoc.ApiDocsMarkdown || item.ApiDocsMarkdownEn != secondDoc.ApiDocsMarkdownEn {
		t.Fatalf("expected channel Markdown document in pricing response: %+v", item)
	}
	if err := DB.Model(&Channel{}).Where("id = ?", channels[0].Id).Update("route_slug", "renamed").Error; err != nil {
		t.Fatalf("rename route: %v", err)
	}
	items, err = ListChannelModelDocItems(&meta)
	if err != nil {
		t.Fatalf("list channel docs: %v", err)
	}
	var overridden ChannelModelDocItem
	for _, item := range items {
		if item.ChannelID == channels[0].Id {
			overridden = item
		}
	}
	if !overridden.Configured || overridden.DocIntroduction != "" || overridden.RouteSlug != "renamed" {
		t.Fatalf("expected explicit empty override to survive route rename: %+v", overridden)
	}

	if err := DeleteChannelModelDoc(channels[0].Id, "demo-model"); err != nil {
		t.Fatalf("delete channel doc: %v", err)
	}
	items, err = ListChannelModelDocItems(&meta)
	if err != nil {
		t.Fatalf("list restored docs: %v", err)
	}
	for _, item := range items {
		if item.ChannelID == channels[0].Id && (item.Configured || item.DocIntroduction != "model fallback") {
			t.Fatalf("expected fallback after deleting override: %+v", item)
		}
	}

	if err := BatchDeleteChannels([]int{channels[1].Id}); err != nil {
		t.Fatalf("batch delete channel: %v", err)
	}
	var remaining int64
	if err := DB.Model(&ChannelModelDoc{}).Where("channel_id = ?", channels[1].Id).Count(&remaining).Error; err != nil {
		t.Fatalf("count deleted channel docs: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("expected batch channel delete to clean docs, got %d", remaining)
	}
}
