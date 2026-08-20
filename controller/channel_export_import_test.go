package controller

import (
	"testing"
	"time"

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
	item, err := buildChannelExportItem(ch, fields)
	require.NoError(t, err)
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
	item, err := buildChannelExportItem(ch, fields)
	require.NoError(t, err)
	applySiteBuilderDiscountExport(item, ch, fields)

	require.Equal(t, 88.0, item[chFieldDiscountRate])
	require.Equal(t, float64(0), item[chFieldOperatingCost])
	require.True(t, fields[chFieldDiscountRate])
}

func TestApplySiteBuilderDiscountExportNilDefaults(t *testing.T) {
	ch := &model.Channel{}
	fields := map[string]bool{chFieldDiscountRate: true}
	item, err := buildChannelExportItem(ch, fields)
	require.NoError(t, err)
	applySiteBuilderDiscountExport(item, ch, fields)

	// nil 成本折扣按 100，nil 经营成本按 0 → 100
	require.Equal(t, 100.0, item[chFieldDiscountRate])
}

func setupChannelDynamicRatesExportTestDB(t *testing.T) *model.Channel {
	t.Helper()
	setupChannelImportTestDB(t)
	require.NoError(t, model.DB.AutoMigrate(&model.ChannelModelPricePlan{}, &model.ChannelModelPriceSchedule{}))
	model.ClearChannelModelTimePricingCache()

	channel := &model.Channel{
		Name:   "dynamic-rate-channel",
		Type:   1,
		Status: common.ChannelStatusEnabled,
		Models: "gpt-4o",
		Group:  "default",
	}
	require.NoError(t, model.DB.Create(channel).Error)

	mutation := &model.ChannelModelRateRuleMutation{
		ChannelID: channel.Id, ModelNames: []string{"gpt-4o"}, Name: "晚高峰",
		PriceDiscountPercent: 70, OperatingCostPercent: 5, MarkupDiscountRate: 80,
		Timezone: model.DefaultChannelModelPricingTimezone,
		Weekdays: (1 << int(time.Monday)) | (1 << int(time.Tuesday)),
		StartMinute: 18 * 60, EndMinute: 22 * 60, Enabled: true, UserID: 1,
	}
	require.NoError(t, model.CreateChannelModelRateRules(mutation))
	return channel
}

func TestBuildChannelExportItemIncludesDynamicRates(t *testing.T) {
	channel := setupChannelDynamicRatesExportTestDB(t)

	item, err := buildChannelExportItem(channel, map[string]bool{chFieldDynamicRates: true})
	require.NoError(t, err)
	rawRules, ok := item[chFieldDynamicRates].([]map[string]interface{})
	require.True(t, ok)
	require.Len(t, rawRules, 1)
	require.Equal(t, "gpt-4o", rawRules[0]["model_name"])
	require.Equal(t, "晚高峰", rawRules[0]["name"])
	require.Equal(t, 70.0, rawRules[0]["price_discount_percent"])
	require.Equal(t, 5.0, rawRules[0]["operating_cost_percent"])
	require.Equal(t, 80.0, rawRules[0]["markup_discount_rate"])
}

func TestImportChannelsReplacesDynamicRates(t *testing.T) {
	channel := setupChannelDynamicRatesExportTestDB(t)

	exported, err := buildChannelDynamicRatesExport(channel.Id)
	require.NoError(t, err)
	require.Len(t, exported, 1)

	exported[0]["name"] = "导入后规则"
	exported[0]["price_discount_percent"] = 55.0

	item := map[string]interface{}{
		"name":          channel.Name,
		chFieldDynamicRates: exported,
	}
	require.NoError(t, chImportDynamicRatesIfPresent(channel.Id, item, 1))

	rules, err := model.ListChannelModelRateRules(channel.Id)
	require.NoError(t, err)
	require.Len(t, rules, 1)
	require.Equal(t, "导入后规则", rules[0].Name)
	require.InDelta(t, 55, rules[0].PriceDiscountPercent, 1e-9)
}

func TestImportChannelsClearsDynamicRatesWithEmptyArray(t *testing.T) {
	channel := setupChannelDynamicRatesExportTestDB(t)

	item := map[string]interface{}{
		"name":              channel.Name,
		chFieldDynamicRates: []interface{}{},
	}
	require.NoError(t, chImportDynamicRatesIfPresent(channel.Id, item, 1))

	rules, err := model.ListChannelModelRateRules(channel.Id)
	require.NoError(t, err)
	require.Empty(t, rules)
}

func setupChannelVideoUpscaleRulesExportTestDB(t *testing.T) *model.Channel {
	t.Helper()
	setupChannelImportTestDB(t)

	settingsJSON, err := common.Marshal(map[string]interface{}{
		"video_upscale_rules": []map[string]interface{}{
			{
				"source_resolution": "480p",
				"target_resolution": "1080p",
				"template_id":       10001,
			},
		},
	})
	require.NoError(t, err)

	channel := &model.Channel{
		Name:          "video-upscale-channel",
		Type:          1,
		Status:        common.ChannelStatusEnabled,
		Models:        "seedance-video",
		Group:         "default",
		OtherSettings: string(settingsJSON),
	}
	require.NoError(t, model.DB.Create(channel).Error)
	return channel
}

func TestBuildChannelExportItemIncludesVideoUpscaleRules(t *testing.T) {
	channel := setupChannelVideoUpscaleRulesExportTestDB(t)

	item, err := buildChannelExportItem(channel, map[string]bool{chFieldVideoUpscaleRules: true})
	require.NoError(t, err)
	rawRules, ok := item[chFieldVideoUpscaleRules].([]map[string]interface{})
	require.True(t, ok)
	require.Len(t, rawRules, 1)
	require.Equal(t, "480p", rawRules[0]["source_resolution"])
	require.Equal(t, "1080p", rawRules[0]["target_resolution"])
	require.EqualValues(t, 10001, rawRules[0]["template_id"])
}

func TestImportChannelsReplacesVideoUpscaleRules(t *testing.T) {
	channel := setupChannelVideoUpscaleRulesExportTestDB(t)

	item := map[string]interface{}{
		"name": channel.Name,
		chFieldVideoUpscaleRules: []map[string]interface{}{
			{
				"source_resolution": "720p",
				"target_resolution": "2K",
				"template_id":       20002,
			},
		},
	}
	require.NoError(t, chImportVideoUpscaleRulesIfPresent(channel.Id, item))

	updated, err := model.GetChannelById(channel.Id, false)
	require.NoError(t, err)
	rules := updated.GetOtherSettings().VideoUpscaleRules
	require.Len(t, rules, 1)
	require.Equal(t, "720p", rules[0].SourceResolution)
	require.Equal(t, "2K", rules[0].TargetResolution)
	require.EqualValues(t, 20002, rules[0].TemplateId)
}

func TestImportChannelsClearsVideoUpscaleRulesWithEmptyArray(t *testing.T) {
	channel := setupChannelVideoUpscaleRulesExportTestDB(t)

	item := map[string]interface{}{
		"name":                   channel.Name,
		chFieldVideoUpscaleRules: []interface{}{},
	}
	require.NoError(t, chImportVideoUpscaleRulesIfPresent(channel.Id, item))

	updated, err := model.GetChannelById(channel.Id, false)
	require.NoError(t, err)
	require.Nil(t, updated.GetOtherSettings().VideoUpscaleRules)
}
