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

func TestFindExistingForImportBySyncKeyRenames(t *testing.T) {
	setupChannelImportTestDB(t)

	require.NoError(t, model.DB.Create(&model.Channel{
		Name:    "old-name",
		SyncKey: "sync-abc",
		Type:    1,
		Status:  common.ChannelStatusEnabled,
	}).Error)

	existing, err := chFindExistingForImport("new-name", map[string]interface{}{
		"name":    "new-name",
		"syncKey": "sync-abc",
	})
	require.NoError(t, err)
	require.NotNil(t, existing)
	require.Equal(t, "old-name", existing.Name)
	require.Equal(t, "sync-abc", existing.SyncKey)

	err = chApplyToExisting(existing, map[string]interface{}{
		"name":    "new-name",
		"syncKey": "sync-abc",
	}, "")
	require.NoError(t, err)

	var updated model.Channel
	require.NoError(t, model.DB.First(&updated, existing.Id).Error)
	require.Equal(t, "new-name", updated.Name)
	require.Equal(t, "sync-abc", updated.SyncKey)
}

func TestFindExistingForImportNameFallbackBindsSyncKey(t *testing.T) {
	setupChannelImportTestDB(t)

	require.NoError(t, model.DB.Create(&model.Channel{
		Name:    "same-name",
		SyncKey: "",
		Type:    1,
		Status:  common.ChannelStatusEnabled,
	}).Error)

	existing, err := chFindExistingForImport("same-name", map[string]interface{}{
		"name":    "same-name",
		"syncKey": "bind-001",
	})
	require.NoError(t, err)
	require.NotNil(t, existing)

	err = chApplyToExisting(existing, map[string]interface{}{
		"syncKey": "bind-001",
	}, "")
	require.NoError(t, err)

	var updated model.Channel
	require.NoError(t, model.DB.First(&updated, existing.Id).Error)
	require.Equal(t, "bind-001", updated.SyncKey)
}

func TestFindExistingForImportNameConflictWithDifferentSyncKey(t *testing.T) {
	setupChannelImportTestDB(t)

	require.NoError(t, model.DB.Create(&model.Channel{
		Name:    "same-name",
		SyncKey: "existing-key",
		Type:    1,
		Status:  common.ChannelStatusEnabled,
	}).Error)

	existing, err := chFindExistingForImport("same-name", map[string]interface{}{
		"name":    "same-name",
		"syncKey": "other-key",
	})
	require.NoError(t, err)
	require.Nil(t, existing)
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

func TestApplySiteBuilderDiscountExportCombinesCostAndOperating(t *testing.T) {
	cost := 60.0
	markup := 20.0
	operating := 5.0
	ch := &model.Channel{
		PriceDiscountPercent: &cost,
		MarkupDiscountRate:   &markup,
		OperatingCostPercent: &operating,
	}

	fields := map[string]bool{
		chFieldDiscountRate:   true,
		chFieldMarkupDiscount: true,
		chFieldOperatingCost:  true,
	}
	item := buildChannelExportItem(ch, fields)
	applySiteBuilderDiscountExport(item, ch, fields)

	// 成本折扣 60 + 经营成本 5 = 65；加价折扣不参与合并
	require.Equal(t, 65.0, item[chFieldDiscountRate])
	require.Equal(t, float64(0), item[chFieldOperatingCost])
	require.Equal(t, float64(0), item[chFieldMarkupDiscount])
}

func TestApplySiteBuilderDiscountExportForcesDiscountWhenOnlyOperatingSelected(t *testing.T) {
	cost := 85.0
	markup := 7.0
	operating := 3.0
	ch := &model.Channel{
		PriceDiscountPercent: &cost,
		MarkupDiscountRate:   &markup,
		OperatingCostPercent: &operating,
	}

	fields := map[string]bool{
		chFieldOperatingCost: true,
	}
	item := buildChannelExportItem(ch, fields)
	applySiteBuilderDiscountExport(item, ch, fields)

	require.Equal(t, 88.0, item[chFieldDiscountRate])
	require.Equal(t, float64(0), item[chFieldOperatingCost])
	require.True(t, fields[chFieldDiscountRate])
}

func TestApplySiteBuilderDiscountExportNilDefaults(t *testing.T) {
	ch := &model.Channel{}
	fields := map[string]bool{chFieldDiscountRate: true}
	item := buildChannelExportItem(ch, fields)
	applySiteBuilderDiscountExport(item, ch, fields)

	// nil 成本折扣按 100，nil 经营成本按 0 → 100
	require.Equal(t, 100.0, item[chFieldDiscountRate])
}
