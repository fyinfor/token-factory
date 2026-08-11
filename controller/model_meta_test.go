package controller

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestSyncChannelModelDocsCopiesAPIDocsAndPreservesIntroductions(t *testing.T) {
	previousDB := model.DB
	t.Cleanup(func() { model.DB = previousDB })

	db, err := gorm.Open(
		sqlite.Open(fmt.Sprintf("file:model_meta_controller_%d?mode=memory&cache=shared", time.Now().UnixNano())),
		&gorm.Config{},
	)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Model{}, &model.Channel{}, &model.Ability{}, &model.ChannelModelDoc{}); err != nil {
		t.Fatalf("migrate sqlite: %v", err)
	}
	model.DB = db

	meta := model.Model{
		ModelName:         "demo-model",
		NameRule:          model.NameRuleExact,
		Status:            1,
		DocIntroduction:   "shared introduction",
		DocIntroductionEn: "shared introduction en",
	}
	if err := db.Create(&meta).Error; err != nil {
		t.Fatalf("create model: %v", err)
	}
	channels := []model.Channel{
		{Name: "source", Status: 1, Models: "demo-model"},
		{Name: "configured target", Status: 1, Models: "demo-model"},
		{Name: "inherited target", Status: 1, Models: "demo-model"},
	}
	if err := db.Create(&channels).Error; err != nil {
		t.Fatalf("create channels: %v", err)
	}
	for _, channel := range channels {
		if err := db.Create(&model.Ability{
			Group:     "default",
			Model:     "demo-model",
			ChannelId: channel.Id,
			Enabled:   true,
		}).Error; err != nil {
			t.Fatalf("create ability: %v", err)
		}
	}

	existingDocs := []model.ChannelModelDoc{
		{
			ChannelID:         channels[0].Id,
			ModelName:         "demo-model",
			DocIntroduction:   "source introduction",
			DocIntroductionEn: "source introduction en",
			ApiDocsMarkdown:   "old source docs",
		},
		{
			ChannelID:         channels[1].Id,
			ModelName:         "demo-model",
			DocIntroduction:   "target introduction",
			DocIntroductionEn: "target introduction en",
			ApiDocsMarkdown:   "old target docs",
		},
	}
	for i := range existingDocs {
		if err := model.UpsertChannelModelDoc(&existingDocs[i]); err != nil {
			t.Fatalf("create channel document: %v", err)
		}
	}

	body, err := common.Marshal(map[string]any{
		"source_channel_id":    channels[0].Id,
		"source_model_name":    "demo-model",
		"api_docs":             `[{"path":"/v1/demo"}]`,
		"api_docs_markdown":    "# Shared API docs",
		"api_docs_markdown_en": "# Shared API docs EN",
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/models/:id/channel_docs/sync", SyncChannelModelDocs)
	request := httptest.NewRequest(
		http.MethodPost,
		fmt.Sprintf("/api/models/%d/channel_docs/sync", meta.Id),
		bytes.NewReader(body),
	)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status %d: %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Success bool `json:"success"`
		Data    struct {
			Synced      int `json:"synced"`
			OtherSynced int `json:"other_synced"`
		} `json:"data"`
	}
	if err := common.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Success || response.Data.Synced != 3 || response.Data.OtherSynced != 2 {
		t.Fatalf("unexpected sync response: %+v", response)
	}

	var docs []model.ChannelModelDoc
	if err := db.Order("channel_id ASC").Find(&docs).Error; err != nil {
		t.Fatalf("load synchronized documents: %v", err)
	}
	if len(docs) != 3 {
		t.Fatalf("expected one document per channel, got %d", len(docs))
	}
	wantIntroductions := map[int][2]string{
		channels[0].Id: {"source introduction", "source introduction en"},
		channels[1].Id: {"target introduction", "target introduction en"},
		channels[2].Id: {"shared introduction", "shared introduction en"},
	}
	for _, doc := range docs {
		wantIntroduction := wantIntroductions[doc.ChannelID]
		if doc.DocIntroduction != wantIntroduction[0] || doc.DocIntroductionEn != wantIntroduction[1] {
			t.Fatalf("channel %d introduction was overwritten: %+v", doc.ChannelID, doc)
		}
		if doc.ApiDocs != `[{"path":"/v1/demo"}]` || doc.ApiDocsMarkdown != "# Shared API docs" || doc.ApiDocsMarkdownEn != "# Shared API docs EN" {
			t.Fatalf("channel %d did not receive synchronized API docs: %+v", doc.ChannelID, doc)
		}
	}
}
