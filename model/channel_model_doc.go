package model

import (
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ChannelModelDoc overrides the model-level fallback document for one concrete
// model exposed by a channel. Route aliases are deliberately not part of its identity.
type ChannelModelDoc struct {
	ID                int    `json:"id" gorm:"primaryKey"`
	ChannelID         int    `json:"channel_id" gorm:"not null;uniqueIndex:uk_channel_model_doc,priority:1;index"`
	ModelName         string `json:"model_name" gorm:"size:255;not null;uniqueIndex:uk_channel_model_doc,priority:2;index"`
	DocIntroduction   string `json:"doc_introduction,omitempty" gorm:"type:text"`
	DocIntroductionEn string `json:"doc_introduction_en,omitempty" gorm:"type:text"`
	ApiDocs           string `json:"api_docs,omitempty" gorm:"type:text"`
	ApiDocsMarkdown   string `json:"api_docs_markdown,omitempty" gorm:"type:text"`
	ApiDocsMarkdownEn string `json:"api_docs_markdown_en,omitempty" gorm:"type:text"`
	CreatedTime       int64  `json:"created_time" gorm:"bigint"`
	UpdatedTime       int64  `json:"updated_time" gorm:"bigint"`
}

type ChannelModelDocItem struct {
	ChannelID         int    `json:"channel_id"`
	ChannelName       string `json:"channel_name"`
	ChannelType       int    `json:"channel_type"`
	ChannelStatus     int    `json:"channel_status"`
	ChannelNo         string `json:"channel_no,omitempty"`
	RouteSlug         string `json:"route_slug,omitempty"`
	ModelName         string `json:"model_name"`
	Configured        bool   `json:"configured"`
	DocIntroduction   string `json:"doc_introduction,omitempty"`
	DocIntroductionEn string `json:"doc_introduction_en,omitempty"`
	ApiDocs           string `json:"api_docs,omitempty"`
	ApiDocsMarkdown   string `json:"api_docs_markdown,omitempty"`
	ApiDocsMarkdownEn string `json:"api_docs_markdown_en,omitempty"`
	UpdatedTime       int64  `json:"updated_time,omitempty"`
}

type ChannelModelDocTemplate struct {
	ID                int    `json:"id"`
	ChannelID         int    `json:"channel_id"`
	ChannelName       string `json:"channel_name"`
	ChannelNo         string `json:"channel_no,omitempty"`
	RouteSlug         string `json:"route_slug,omitempty"`
	ModelName         string `json:"model_name"`
	DocIntroduction   string `json:"doc_introduction,omitempty"`
	DocIntroductionEn string `json:"doc_introduction_en,omitempty"`
	ApiDocs           string `json:"api_docs,omitempty"`
	ApiDocsMarkdown   string `json:"api_docs_markdown,omitempty"`
	ApiDocsMarkdownEn string `json:"api_docs_markdown_en,omitempty"`
	UpdatedTime       int64  `json:"updated_time,omitempty"`
}

func channelModelDocKey(channelID int, modelName string) string {
	return strconv.Itoa(channelID) + "\x00" + strings.TrimSpace(modelName)
}

func GetAllChannelModelDocsMap() (map[string]ChannelModelDoc, error) {
	var docs []ChannelModelDoc
	if err := DB.Find(&docs).Error; err != nil {
		return nil, err
	}
	result := make(map[string]ChannelModelDoc, len(docs))
	for _, doc := range docs {
		result[channelModelDocKey(doc.ChannelID, doc.ModelName)] = doc
	}
	return result, nil
}

func ListChannelModelDocTemplates() ([]ChannelModelDocTemplate, error) {
	var templates []ChannelModelDocTemplate
	err := DB.Table("channel_model_docs").
		Select("channel_model_docs.id, channel_model_docs.channel_id, channels.name AS channel_name, channels.channel_no, channels.route_slug, channel_model_docs.model_name, channel_model_docs.doc_introduction, channel_model_docs.doc_introduction_en, channel_model_docs.api_docs, channel_model_docs.api_docs_markdown, channel_model_docs.api_docs_markdown_en, channel_model_docs.updated_time").
		Joins("LEFT JOIN channels ON channels.id = channel_model_docs.channel_id").
		Where("channel_model_docs.doc_introduction <> ? OR channel_model_docs.doc_introduction_en <> ? OR channel_model_docs.api_docs <> ? OR channel_model_docs.api_docs_markdown <> ? OR channel_model_docs.api_docs_markdown_en <> ?", "", "", "", "", "").
		Order("channel_model_docs.updated_time DESC, channel_model_docs.id DESC").
		Scan(&templates).Error
	if err != nil {
		return nil, err
	}
	return templates, nil
}

// ListChannelModelDocItems returns concrete channel-model bindings matched by
// a model metadata rule and resolves unconfigured rows to the legacy fallback.
func ListChannelModelDocItems(meta *Model) ([]ChannelModelDocItem, error) {
	if meta == nil || meta.Id <= 0 {
		return []ChannelModelDocItem{}, nil
	}

	var items []ChannelModelDocItem
	err := DB.Table("abilities").
		Select("channels.id AS channel_id, channels.name AS channel_name, channels.type AS channel_type, channels.status AS channel_status, channels.channel_no, channels.route_slug, abilities.model AS model_name").
		Joins("JOIN channels ON channels.id = abilities.channel_id").
		Distinct().
		Order("channels.id ASC, abilities.model ASC").
		Scan(&items).Error
	if err != nil {
		return nil, err
	}

	filtered := make([]ChannelModelDocItem, 0, len(items))
	channelIDs := make([]int, 0, len(items))
	seenChannelIDs := make(map[int]struct{})
	for _, item := range items {
		item.ModelName = strings.TrimSpace(item.ModelName)
		if !modelNameMatchesRule(meta.ModelName, item.ModelName, meta.NameRule) {
			continue
		}
		filtered = append(filtered, item)
		if _, ok := seenChannelIDs[item.ChannelID]; !ok {
			seenChannelIDs[item.ChannelID] = struct{}{}
			channelIDs = append(channelIDs, item.ChannelID)
		}
	}

	docMap := make(map[string]ChannelModelDoc)
	if len(channelIDs) > 0 {
		var docs []ChannelModelDoc
		if err := DB.Where("channel_id IN ?", channelIDs).Find(&docs).Error; err != nil {
			return nil, err
		}
		for _, doc := range docs {
			docMap[channelModelDocKey(doc.ChannelID, doc.ModelName)] = doc
		}
	}

	for i := range filtered {
		item := &filtered[i]
		if doc, ok := docMap[channelModelDocKey(item.ChannelID, item.ModelName)]; ok {
			item.Configured = true
			item.DocIntroduction = doc.DocIntroduction
			item.DocIntroductionEn = doc.DocIntroductionEn
			if strings.TrimSpace(item.DocIntroductionEn) == "" {
				item.DocIntroductionEn = meta.DocIntroductionEn
			}
			item.ApiDocs = doc.ApiDocs
			item.ApiDocsMarkdown = doc.ApiDocsMarkdown
			item.ApiDocsMarkdownEn = doc.ApiDocsMarkdownEn
			item.UpdatedTime = doc.UpdatedTime
			continue
		}
		item.DocIntroduction = meta.DocIntroduction
		item.DocIntroductionEn = meta.DocIntroductionEn
		item.ApiDocs = meta.ApiDocs
	}
	return filtered, nil
}

func ValidateChannelModelDocBinding(meta *Model, channelID int, modelName string) error {
	items, err := ListChannelModelDocItems(meta)
	if err != nil {
		return err
	}
	modelName = strings.TrimSpace(modelName)
	for _, item := range items {
		if item.ChannelID == channelID && item.ModelName == modelName {
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}

func upsertChannelModelDoc(db *gorm.DB, doc *ChannelModelDoc) error {
	if doc == nil || doc.ChannelID <= 0 || strings.TrimSpace(doc.ModelName) == "" {
		return gorm.ErrInvalidData
	}
	doc.ModelName = strings.TrimSpace(doc.ModelName)
	now := common.GetTimestamp()
	doc.CreatedTime = now
	doc.UpdatedTime = now
	return db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "channel_id"}, {Name: "model_name"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"doc_introduction",
			"doc_introduction_en",
			"api_docs",
			"api_docs_markdown",
			"api_docs_markdown_en",
			"updated_time",
		}),
	}).Create(doc).Error
}

func UpsertChannelModelDoc(doc *ChannelModelDoc) error {
	return upsertChannelModelDoc(DB, doc)
}

func UpsertChannelModelDocWithDB(db *gorm.DB, doc *ChannelModelDoc) error {
	if db == nil {
		return gorm.ErrInvalidDB
	}
	return upsertChannelModelDoc(db, doc)
}

func DeleteChannelModelDoc(channelID int, modelName string) error {
	return DB.Where("channel_id = ? AND model_name = ?", channelID, strings.TrimSpace(modelName)).
		Delete(&ChannelModelDoc{}).Error
}

func DeleteChannelModelDocsByChannel(channelID int) error {
	return deleteChannelModelDocsByChannelIDs(DB, []int{channelID})
}

func deleteChannelModelDocsByChannelIDs(db *gorm.DB, channelIDs []int) error {
	if len(channelIDs) == 0 {
		return nil
	}
	return db.Where("channel_id IN ?", channelIDs).Delete(&ChannelModelDoc{}).Error
}
