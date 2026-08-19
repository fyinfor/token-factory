package controller

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestPrepareDocumentAIRequestBuildsPrivateRelayRequest(t *testing.T) {
	previousDB := model.DB
	t.Cleanup(func() { model.DB = previousDB })
	db, err := gorm.Open(
		sqlite.Open(fmt.Sprintf("file:document_ai_controller_%d?mode=memory&cache=shared", time.Now().UnixNano())),
		&gorm.Config{},
	)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.DocumentAIPromptSettings{}); err != nil {
		t.Fatalf("migrate sqlite: %v", err)
	}
	model.DB = db
	if err := model.SaveDocumentAIPromptSettings(&model.DocumentAIPromptSettings{
		PolishPrompt:    "private polish prompt",
		TranslatePrompt: "private translate prompt",
	}); err != nil {
		t.Fatalf("save custom prompts: %v", err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	var captured dto.GeneralOpenAIRequest
	var relayMode int
	router.POST("/api/models/document_ai/generate", PrepareDocumentAIRequest, func(c *gin.Context) {
		relayMode = c.GetInt("relay_mode")
		if err := common.UnmarshalBodyReusable(c, &captured); err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		c.Status(http.StatusNoContent)
	})

	body := []byte(`{"model":"demo-model","action":"polish","document":"# Demo API"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/models/document_ai/generate", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("unexpected status %d: %s", recorder.Code, recorder.Body.String())
	}
	if captured.Model != "demo-model" || captured.Stream == nil || !*captured.Stream {
		t.Fatalf("unexpected relay request: %+v", captured)
	}
	if len(captured.Messages) != 2 || captured.Messages[0].Content != "private polish prompt" {
		t.Fatalf("private prompt was not applied: %+v", captured.Messages)
	}
	if captured.Messages[1].Content == nil || !bytes.Contains([]byte(captured.Messages[1].Content.(string)), []byte("# Demo API")) {
		t.Fatalf("source document missing from relay request: %+v", captured.Messages[1])
	}
	if request.URL.Path != "/api/playground/chat/completions" || relayMode != relayconstant.RelayModeChatCompletions {
		t.Fatalf("request was not converted to chat relay: path=%s mode=%d", request.URL.Path, relayMode)
	}
}

func TestPrepareDocumentAIRequestBuildsIntroductionSearchRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	var captured dto.GeneralOpenAIRequest
	router.POST("/api/models/document_ai/generate", PrepareDocumentAIRequest, func(c *gin.Context) {
		if err := common.UnmarshalBodyReusable(c, &captured); err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		c.Status(http.StatusNoContent)
	})

	body := []byte(`{"model":"search-model","section":"introduction","action":"generate","target_model":"demo-model"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/models/document_ai/generate", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("unexpected status %d: %s", recorder.Code, recorder.Body.String())
	}
	if captured.Model != "search-model" || captured.WebSearchOptions == nil || captured.WebSearchOptions.SearchContextSize != "medium" {
		t.Fatalf("expected web search request, got %+v", captured)
	}
	if len(captured.Messages) != 2 || captured.Messages[1].Content == nil || !bytes.Contains([]byte(captured.Messages[1].Content.(string)), []byte("demo-model")) {
		t.Fatalf("target model missing from introduction request: %+v", captured.Messages)
	}
}

func TestPrepareDocumentAIRequestBuildsAnnouncementTranslationRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	var captured dto.GeneralOpenAIRequest
	router.POST("/api/models/document_ai/generate", PrepareDocumentAIRequest, func(c *gin.Context) {
		if err := common.UnmarshalBodyReusable(c, &captured); err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		c.Status(http.StatusNoContent)
	})

	body := []byte(`{"model":"translate-model","section":"announcement","action":"translate","document":"# 系统维护"}`)
	request := httptest.NewRequest(http.MethodPost, "/api/models/document_ai/generate", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("unexpected status %d: %s", recorder.Code, recorder.Body.String())
	}
	if len(captured.Messages) != 2 || captured.Messages[0].Content != defaultAnnouncementTranslatePrompt {
		t.Fatalf("announcement translation prompt was not applied: %+v", captured.Messages)
	}
	if captured.Messages[1].Content == nil || !bytes.Contains([]byte(captured.Messages[1].Content.(string)), []byte("系统维护")) {
		t.Fatalf("announcement source missing from relay request: %+v", captured.Messages[1])
	}
}
