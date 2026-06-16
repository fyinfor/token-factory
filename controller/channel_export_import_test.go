package controller

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupChannelImportTestDB(t *testing.T) {
	t.Helper()
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	require.NoError(t, db.AutoMigrate(&model.Channel{}, &model.Ability{}, &model.SupplierApplication{}))
}

func TestFindExistingForImportSiteBuilderDoesNotMatchOrdinaryChannel(t *testing.T) {
	setupChannelImportTestDB(t)

	require.NoError(t, model.DB.Create(&model.Channel{
		Name:   "same-name",
		Type:   1,
		Status: common.ChannelStatusEnabled,
	}).Error)

	existing, err := chFindExistingForImport("same-name", map[string]interface{}{
		"type":       float64(constant.ChannelTypeTokenFactoryOpen),
		"apiBaseUrl": "https://upstream.example",
		"otherInfo": map[string]interface{}{
			"source":              "tokenfactory_open",
			"upstream_route_slug": "uAb12",
		},
	})

	require.NoError(t, err)
	require.Nil(t, existing)
}

func TestFindExistingForImportSiteBuilderMatchesRouteIdentity(t *testing.T) {
	setupChannelImportTestDB(t)

	baseURL := "https://upstream.example/"
	otherInfo, err := common.Marshal(map[string]interface{}{
		"source":              "tokenfactory_open",
		"upstream_route_slug": "uAb12",
	})
	require.NoError(t, err)
	require.NoError(t, model.DB.Create(&model.Channel{
		Name:      "same-name",
		Type:      constant.ChannelTypeTokenFactoryOpen,
		Status:    common.ChannelStatusEnabled,
		BaseURL:   &baseURL,
		OtherInfo: string(otherInfo),
	}).Error)

	existing, err := chFindExistingForImport("same-name", map[string]interface{}{
		"type":       float64(constant.ChannelTypeTokenFactoryOpen),
		"apiBaseUrl": "https://upstream.example",
		"otherInfo": map[string]interface{}{
			"source":              "tokenfactory_open",
			"upstream_route_slug": "uAb12",
		},
	})

	require.NoError(t, err)
	require.NotNil(t, existing)
	require.Equal(t, constant.ChannelTypeTokenFactoryOpen, existing.Type)
}

func TestApplyToExistingRefreshesAbilitiesWhenModelsChange(t *testing.T) {
	setupChannelImportTestDB(t)

	channel := &model.Channel{
		Name:   "ability-refresh",
		Type:   1,
		Status: common.ChannelStatusEnabled,
		Models: "old-model",
		Group:  "default",
	}
	require.NoError(t, model.DB.Create(channel).Error)
	require.NoError(t, channel.AddAbilities(nil))

	err := chApplyToExisting(channel, map[string]interface{}{
		"models": []interface{}{"new-model"},
		"groups": []interface{}{"default"},
	}, "")
	require.NoError(t, err)

	var oldCount int64
	require.NoError(t, model.DB.Model(&model.Ability{}).
		Where("channel_id = ? AND model = ?", channel.Id, "old-model").
		Count(&oldCount).Error)
	require.Zero(t, oldCount)

	var newCount int64
	require.NoError(t, model.DB.Model(&model.Ability{}).
		Where("channel_id = ? AND model = ?", channel.Id, "new-model").
		Count(&newCount).Error)
	require.EqualValues(t, 1, newCount)
}
