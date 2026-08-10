package controller

import (
	"fmt"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupModelExportImportTestDB(t *testing.T) {
	t.Helper()
	previousDB := model.DB
	previousSQLite := common.UsingSQLite
	previousMySQL := common.UsingMySQL
	previousPostgreSQL := common.UsingPostgreSQL
	t.Cleanup(func() {
		model.DB = previousDB
		common.UsingSQLite = previousSQLite
		common.UsingMySQL = previousMySQL
		common.UsingPostgreSQL = previousPostgreSQL
	})

	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	db, err := gorm.Open(
		sqlite.Open(fmt.Sprintf("file:model_export_import_%d?mode=memory&cache=shared", time.Now().UnixNano())),
		&gorm.Config{},
	)
	require.NoError(t, err)
	model.DB = db
	require.NoError(t, db.AutoMigrate(
		&model.Channel{},
		&model.Ability{},
		&model.ChannelModelDoc{},
	))
}

func TestBuildChannelModelDocExportItemsIncludesMarkdownAndEmptyOverrides(t *testing.T) {
	setupModelExportImportTestDB(t)

	channel := &model.Channel{Name: "renamed-channel", SyncKey: "sync-doc", Status: 1}
	require.NoError(t, model.DB.Create(channel).Error)
	require.NoError(t, model.DB.Create(&model.ChannelModelDoc{
		ChannelID:         channel.Id,
		ModelName:         "video-model",
		DocIntroduction:   "intro",
		DocIntroductionEn: "intro en",
		ApiDocs:           `[{"path":"/v1/videos"}]`,
		ApiDocsMarkdown:   "# 中文文档",
		ApiDocsMarkdownEn: "# English docs",
	}).Error)
	require.NoError(t, model.DB.Create(&model.ChannelModelDoc{
		ChannelID: channel.Id,
		ModelName: "explicit-empty-model",
	}).Error)

	items, err := buildChannelModelDocExportItems()
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.Equal(t, "sync-doc", items[0].ChannelSyncKey)
	require.Equal(t, "renamed-channel", items[0].ChannelName)
	require.Equal(t, "intro en", *items[0].DocIntroductionEn)
	require.Equal(t, "# 中文文档", *items[0].ApiDocsMarkdown)
	require.Equal(t, "# English docs", *items[0].ApiDocsMarkdownEn)
	require.NotNil(t, items[1].ApiDocsMarkdown)
	require.Empty(t, *items[1].ApiDocsMarkdown)
}

func TestImportChannelModelDocsUsesSyncKeyAndReportsSkips(t *testing.T) {
	setupModelExportImportTestDB(t)

	channel := &model.Channel{Name: "target-renamed", SyncKey: "sync-target", Status: 1}
	require.NoError(t, model.DB.Create(channel).Error)
	require.NoError(t, model.DB.Create(&model.Ability{
		Group: "default", Model: "video-model", ChannelId: channel.Id, Enabled: true,
	}).Error)

	zh := "# 新中文文档"
	en := "# New English docs"
	result := &ModelImportResult{Failures: []ModelImportFailureItem{}, SkippedDocs: []ModelImportFailureItem{}}
	importChannelModelDocs([]ChannelModelDocExportItem{
		{
			ModelName:         "video-model",
			ChannelSyncKey:    "sync-target",
			ChannelName:       "source-old-name",
			ApiDocsMarkdown:   &zh,
			ApiDocsMarkdownEn: &en,
		},
		{
			ModelName:      "video-model",
			ChannelSyncKey: "different-sync-key",
			ChannelName:    "target-renamed",
		},
	}, result)

	require.Equal(t, 1, result.ChannelDocsAdded)
	require.Equal(t, 1, result.ChannelDocsSkipped)
	require.Zero(t, result.ChannelDocsFailed)
	require.Contains(t, result.SkippedDocs[0].Reason, "different-sync-key")

	var stored model.ChannelModelDoc
	require.NoError(t, model.DB.Where("channel_id = ? AND model_name = ?", channel.Id, "video-model").First(&stored).Error)
	require.Equal(t, zh, stored.ApiDocsMarkdown)
	require.Equal(t, en, stored.ApiDocsMarkdownEn)
}

func TestImportChannelModelDocsFallsBackToNameWithoutSyncKey(t *testing.T) {
	setupModelExportImportTestDB(t)

	channel := &model.Channel{Name: "legacy-channel", SyncKey: "local-sync", Status: 1}
	require.NoError(t, model.DB.Create(channel).Error)
	require.NoError(t, model.DB.Create(&model.Ability{
		Group: "default", Model: "image-model", ChannelId: channel.Id, Enabled: true,
	}).Error)
	oldMarkdown := "# Old"
	require.NoError(t, model.DB.Create(&model.ChannelModelDoc{
		ChannelID: channel.Id, ModelName: "image-model", ApiDocsMarkdown: oldMarkdown,
	}).Error)

	newMarkdown := "# New"
	result := &ModelImportResult{Failures: []ModelImportFailureItem{}, SkippedDocs: []ModelImportFailureItem{}}
	importChannelModelDocs([]ChannelModelDocExportItem{{
		ModelName:       "image-model",
		ChannelName:     "legacy-channel",
		ApiDocsMarkdown: &newMarkdown,
	}}, result)

	require.Equal(t, 1, result.ChannelDocsUpdated)
	var stored model.ChannelModelDoc
	require.NoError(t, model.DB.Where("channel_id = ? AND model_name = ?", channel.Id, "image-model").First(&stored).Error)
	require.Equal(t, newMarkdown, stored.ApiDocsMarkdown)
}
