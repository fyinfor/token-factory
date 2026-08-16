package model

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func withChannelModelTimePricingDB(t *testing.T) {
	t.Helper()
	previousDB := DB
	db, err := gorm.Open(
		sqlite.Open(fmt.Sprintf("file:channel_model_time_pricing_%s_%d?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"), time.Now().UnixNano())),
		&gorm.Config{},
	)
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&ChannelModelPricePlan{},
		&ChannelModelPriceSchedule{},
		&Channel{},
		&ChannelModelDoc{},
		&ModelTestResult{},
	))
	DB = db
	ClearChannelModelTimePricingCache()
	t.Cleanup(func() {
		DB = previousDB
		ClearChannelModelTimePricingCache()
	})
}

func float64Pointer(value float64) *float64 { return &value }

func TestResolveActiveChannelModelPricePlan(t *testing.T) {
	withChannelModelTimePricingDB(t)

	plan := &ChannelModelPricePlan{
		ChannelID: 12, ModelName: "demo-model", Name: "工作日晚高峰",
		Enabled: true, CreatedByUserID: 1, UpdatedByUserID: 1,
	}
	require.NoError(t, CreateChannelModelPricePlan(plan, ChannelModelPricePlanPayload{
		ModelRatio: float64Pointer(1.25), CompletionRatio: float64Pointer(4),
	}))

	schedule := &ChannelModelPriceSchedule{
		ChannelID: 12, ModelName: "demo-model", PricePlanID: plan.ID,
		Name: "工作日 18-23", Timezone: DefaultChannelModelPricingTimezone,
		Weekdays: (1 << int(time.Monday)) | (1 << int(time.Tuesday)) | (1 << int(time.Wednesday)) |
			(1 << int(time.Thursday)) | (1 << int(time.Friday)),
		StartMinute: 18 * 60, EndMinute: 23 * 60, Enabled: true,
	}
	require.NoError(t, CreateChannelModelPriceSchedule(schedule))

	location, err := time.LoadLocation(DefaultChannelModelPricingTimezone)
	require.NoError(t, err)
	active, ok := ResolveActiveChannelModelPricePlan(12, "demo-model", time.Date(2026, 8, 17, 19, 30, 0, 0, location))
	require.True(t, ok)
	require.Equal(t, plan.ID, active.Plan.ID)
	require.NotNil(t, active.Payload.ModelRatio)
	require.InDelta(t, 1.25, *active.Payload.ModelRatio, 1e-9)

	_, ok = ResolveActiveChannelModelPricePlan(12, "demo-model", time.Date(2026, 8, 17, 23, 0, 0, 0, location))
	require.False(t, ok, "end time must be exclusive")
	_, ok = ResolveActiveChannelModelPricePlan(12, "demo-model", time.Date(2026, 8, 16, 19, 30, 0, 0, location))
	require.False(t, ok, "Sunday must not match a weekday schedule")
}

func TestChannelModelPriceScheduleCrossMidnight(t *testing.T) {
	schedule := ChannelModelPriceSchedule{
		Name: "Friday night", Timezone: DefaultChannelModelPricingTimezone,
		Weekdays: 1 << int(time.Friday), StartMinute: 22 * 60, EndMinute: 2 * 60, Enabled: true,
	}
	require.NoError(t, ValidateChannelModelPriceSchedule(&schedule))
	location, err := time.LoadLocation(DefaultChannelModelPricingTimezone)
	require.NoError(t, err)
	require.True(t, scheduleMatchesAt(schedule, time.Date(2026, 8, 14, 23, 0, 0, 0, location)))
	require.True(t, scheduleMatchesAt(schedule, time.Date(2026, 8, 15, 1, 59, 0, 0, location)))
	require.False(t, scheduleMatchesAt(schedule, time.Date(2026, 8, 15, 2, 0, 0, 0, location)))
}

func TestCreateChannelModelPriceScheduleRejectsOverlap(t *testing.T) {
	withChannelModelTimePricingDB(t)
	plan := &ChannelModelPricePlan{ChannelID: 1, ModelName: "m", Name: "peak", Enabled: true}
	require.NoError(t, CreateChannelModelPricePlan(plan, ChannelModelPricePlanPayload{ModelPrice: float64Pointer(1)}))

	first := &ChannelModelPriceSchedule{
		ChannelID: 1, ModelName: "m", PricePlanID: plan.ID, Name: "A",
		Timezone: DefaultChannelModelPricingTimezone, Weekdays: 1 << int(time.Monday),
		StartMinute: 9 * 60, EndMinute: 12 * 60, Enabled: true,
	}
	require.NoError(t, CreateChannelModelPriceSchedule(first))
	second := &ChannelModelPriceSchedule{
		ChannelID: 1, ModelName: "m", PricePlanID: plan.ID, Name: "B",
		Timezone: DefaultChannelModelPricingTimezone, Weekdays: 1 << int(time.Monday),
		StartMinute: 11 * 60, EndMinute: 13 * 60, Enabled: true,
	}
	err := CreateChannelModelPriceSchedule(second)
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrChannelModelScheduleConflict))
}

func TestDeleteChannelModelPricePlanInUse(t *testing.T) {
	withChannelModelTimePricingDB(t)
	plan := &ChannelModelPricePlan{ChannelID: 1, ModelName: "m", Name: "peak", Enabled: true}
	require.NoError(t, CreateChannelModelPricePlan(plan, ChannelModelPricePlanPayload{ModelPrice: float64Pointer(1)}))
	require.NoError(t, CreateChannelModelPriceSchedule(&ChannelModelPriceSchedule{
		ChannelID: 1, ModelName: "m", PricePlanID: plan.ID, Name: "all day",
		Timezone: DefaultChannelModelPricingTimezone, Weekdays: 0x7f,
		StartMinute: 0, EndMinute: 1440, Enabled: true,
	}))
	require.ErrorIs(t, DeleteChannelModelPricePlan(plan.ID), ErrChannelModelPricePlanInUse)
}

func TestUpdateChannelModelPricePlanRefreshesPayloadAndVersion(t *testing.T) {
	withChannelModelTimePricingDB(t)
	plan := &ChannelModelPricePlan{ChannelID: 1, ModelName: "m", Name: "peak", Enabled: true}
	require.NoError(t, CreateChannelModelPricePlan(plan, ChannelModelPricePlanPayload{ModelPrice: float64Pointer(1)}))

	plan.Name = "peak v2"
	require.NoError(t, UpdateChannelModelPricePlan(plan, ChannelModelPricePlanPayload{ModelPrice: float64Pointer(2)}))
	require.Equal(t, 2, plan.Version)
	payload, err := ParseChannelModelPricePlanPayload(plan.PricePayload)
	require.NoError(t, err)
	require.NotNil(t, payload.ModelPrice)
	require.InDelta(t, 2, *payload.ModelPrice, 1e-9)
}

func TestChannelModelRatePlanRequiresCompleteRates(t *testing.T) {
	payload := ChannelModelPricePlanPayload{
		Mode:                 ChannelModelPricePlanModeRate,
		PriceDiscountPercent: float64Pointer(60),
		OperatingCostPercent: float64Pointer(5),
	}
	require.Error(t, payload.NormalizeAndValidate())
	payload.MarkupDiscountRate = float64Pointer(10)
	require.NoError(t, payload.NormalizeAndValidate())
	require.Equal(t, ChannelModelPricePlanModeRate, payload.ResolvedMode())
}

func TestValidateChannelModelPriceScheduleRejectsOtherTimezone(t *testing.T) {
	schedule := &ChannelModelPriceSchedule{
		Name: "UTC schedule", Timezone: "UTC", Weekdays: 0x7f,
		StartMinute: 0, EndMinute: 60, Enabled: true,
	}
	require.Error(t, ValidateChannelModelPriceSchedule(schedule))
}

func TestBuildPricingAPIItemsUsesActiveTimePriceAndMappedModel(t *testing.T) {
	withChannelModelTimePricingDB(t)

	previousModelRatio := ratio_setting.ModelRatio2JSONString()
	previousCompletionRatio := ratio_setting.CompletionRatio2JSONString()
	previousCacheRatio := ratio_setting.CacheRatio2JSONString()
	previousCreateCacheRatio := ratio_setting.CreateCacheRatio2JSONString()
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(previousModelRatio))
		require.NoError(t, ratio_setting.UpdateCompletionRatioByJSONString(previousCompletionRatio))
		require.NoError(t, ratio_setting.UpdateCacheRatioByJSONString(previousCacheRatio))
		require.NoError(t, ratio_setting.UpdateCreateCacheRatioByJSONString(previousCreateCacheRatio))
	})
	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"canonical-model":5}`))
	require.NoError(t, ratio_setting.UpdateCompletionRatioByJSONString(`{"canonical-model":2}`))
	require.NoError(t, ratio_setting.UpdateCacheRatioByJSONString(`{"canonical-model":0.2}`))
	require.NoError(t, ratio_setting.UpdateCreateCacheRatioByJSONString(`{"canonical-model":1.5}`))

	plan := &ChannelModelPricePlan{
		ChannelID: 7, ModelName: "canonical-model", Name: "peak price", Enabled: true,
	}
	require.NoError(t, CreateChannelModelPricePlan(plan, ChannelModelPricePlanPayload{
		ModelRatio:      float64Pointer(20),
		CompletionRatio: float64Pointer(1),
		CacheRatio:      float64Pointer(0.1),
	}))
	require.NoError(t, CreateChannelModelPriceSchedule(&ChannelModelPriceSchedule{
		ChannelID: 7, ModelName: "canonical-model", PricePlanID: plan.ID,
		Name: "weekday peak", Timezone: DefaultChannelModelPricingTimezone,
		Weekdays: 1 << int(time.Monday), StartMinute: 12 * 60, EndMinute: 16 * 60,
		Enabled: true,
	}))

	officialCompletion := 2.0
	officialCache := 0.2
	officialCreateCache := 1.5
	pricing := []Pricing{{
		ModelName: "display-model", ModelRatio: 5, QuotaType: 0,
		CompletionRatio: &officialCompletion, CacheRatio: &officialCache,
		CreateCacheRatio: &officialCreateCache,
	}}
	metas := []ChannelPricingMeta{{
		ChannelID: 7, Models: "display-model",
		ModelMapping: `{"display-model":"canonical-model"}`,
	}}
	location, err := time.LoadLocation(DefaultChannelModelPricingTimezone)
	require.NoError(t, err)

	activeItems := buildPricingAPIItemsAt(
		pricing,
		map[int]struct{}{7: {}},
		metas,
		true,
		time.Date(2026, 8, 17, 13, 0, 0, 0, location),
	)
	require.Len(t, activeItems, 1)
	require.Len(t, activeItems[0].ChannelList, 1)
	activeChannel := activeItems[0].ChannelList[0]
	require.InDelta(t, 20, activeChannel.ModelRatio, 1e-9)
	require.InDelta(t, 1, activeChannel.CompletionRatio, 1e-9)
	require.InDelta(t, 0.1, activeChannel.CacheRatio, 1e-9)
	require.InDelta(t, 1.5, activeChannel.CreateCacheRatio, 1e-9, "unset child ratios must fall back to official pricing")
	require.NotNil(t, activeChannel.TimePricing)
	require.True(t, activeChannel.TimePricing.Active)
	require.Equal(t, "peak price", activeChannel.TimePricing.ActivePlanName)
	require.InDelta(t, 5, activeChannel.TimePricing.RegularPricing.ModelRatio, 1e-9)
	require.InDelta(t, 2, activeChannel.TimePricing.RegularPricing.CompletionRatio, 1e-9)
	require.InDelta(t, 0.2, activeChannel.TimePricing.RegularPricing.CacheRatio, 1e-9)
	require.InDelta(t, 1.5, activeChannel.TimePricing.RegularPricing.CreateCacheRatio, 1e-9)
	require.Len(t, activeChannel.TimePricing.Schedules, 1)
	require.True(t, activeChannel.TimePricing.Schedules[0].Active)

	regularItems := buildPricingAPIItemsAt(
		pricing,
		map[int]struct{}{7: {}},
		metas,
		true,
		time.Date(2026, 8, 17, 17, 0, 0, 0, location),
	)
	require.Len(t, regularItems, 1)
	regularChannel := regularItems[0].ChannelList[0]
	require.InDelta(t, 5, regularChannel.ModelRatio, 1e-9)
	require.NotNil(t, regularChannel.TimePricing)
	require.False(t, regularChannel.TimePricing.Active)
	require.InDelta(t, 5, regularChannel.TimePricing.RegularPricing.ModelRatio, 1e-9)
	require.Len(t, regularChannel.TimePricing.Schedules, 1)
}

func TestBuildPricingAPIItemsUsesActiveTimeRateWithoutReplacingRegularPrice(t *testing.T) {
	withChannelModelTimePricingDB(t)

	previousModelRatio := ratio_setting.ModelRatio2JSONString()
	previousCompletionRatio := ratio_setting.CompletionRatio2JSONString()
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(previousModelRatio))
		require.NoError(t, ratio_setting.UpdateCompletionRatioByJSONString(previousCompletionRatio))
	})
	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"rate-model":5}`))
	require.NoError(t, ratio_setting.UpdateCompletionRatioByJSONString(`{"rate-model":2}`))

	plan := &ChannelModelPricePlan{
		ChannelID: 8, ModelName: "rate-model", Name: "peak rate", Enabled: true,
	}
	require.NoError(t, CreateChannelModelPricePlan(plan, ChannelModelPricePlanPayload{
		Mode:                 ChannelModelPricePlanModeRate,
		PriceDiscountPercent: float64Pointer(80),
		OperatingCostPercent: float64Pointer(5),
		MarkupDiscountRate:   float64Pointer(20),
	}))
	require.NoError(t, CreateChannelModelPriceSchedule(&ChannelModelPriceSchedule{
		ChannelID: 8, ModelName: "rate-model", PricePlanID: plan.ID,
		Name: "all day", Timezone: DefaultChannelModelPricingTimezone,
		Weekdays: 0x7f, StartMinute: 0, EndMinute: 1440, Enabled: true,
	}))

	regularDiscount := 60.0
	regularOperating := 5.0
	regularMarkup := 10.0
	pricing := []Pricing{{ModelName: "rate-model", ModelRatio: 5, QuotaType: 0}}
	metas := []ChannelPricingMeta{{
		ChannelID: 8, Models: "rate-model",
		PriceDiscountPercent: &regularDiscount,
		OperatingCostPercent: &regularOperating,
		MarkupDiscountRate:   &regularMarkup,
	}}
	location, err := time.LoadLocation(DefaultChannelModelPricingTimezone)
	require.NoError(t, err)
	items := buildPricingAPIItemsAt(
		pricing,
		map[int]struct{}{8: {}},
		metas,
		true,
		time.Date(2026, 8, 17, 13, 0, 0, 0, location),
	)
	require.Len(t, items, 1)
	channel := items[0].ChannelList[0]
	require.InDelta(t, 5, channel.ModelRatio, 1e-9, "rate mode must retain the regular channel price")
	require.InDelta(t, 80, channel.RawPriceDiscountPercent, 1e-9)
	require.InDelta(t, 5, channel.OperatingCostPercent, 1e-9)
	require.InDelta(t, 85, channel.PriceDiscountPercent, 1e-9)
	require.InDelta(t, 20, channel.MarkupDiscountRate, 1e-9)
	require.NotNil(t, channel.TimePricing)
	require.InDelta(t, 65, channel.TimePricing.RegularRates.EffectiveCostPercent, 1e-9)
	require.Len(t, channel.TimePricing.Schedules, 1)
	require.Equal(t, ChannelModelPricePlanModeRate, channel.TimePricing.Schedules[0].Mode)
	require.NotNil(t, channel.TimePricing.Schedules[0].Rates)
	require.InDelta(t, 5, channel.TimePricing.Schedules[0].Pricing.ModelRatio, 1e-9)
}

func TestBuildPricingAPIItemsUsesTimePricingMediaHints(t *testing.T) {
	withChannelModelTimePricingDB(t)

	const (
		channelID        = 9
		imageModelName   = "time-pricing-image-display"
		videoModelName   = "time-pricing-video-display"
		imagePrice       = 0.031
		videoPricePerSec = 0.012
	)
	imagePlan := &ChannelModelPricePlan{
		ChannelID: channelID, ModelName: imageModelName, Name: "image peak", Enabled: true,
	}
	require.NoError(t, CreateChannelModelPricePlan(imagePlan, ChannelModelPricePlanPayload{
		ImagePricingRules: &ratio_setting.ImagePricingRules{
			TextToImagePerImage: []ratio_setting.ImageResolutionPerImageRule{
				{Resolution: "1024x1024", ImagePrice: imagePrice},
			},
		},
	}))
	require.NoError(t, CreateChannelModelPriceSchedule(&ChannelModelPriceSchedule{
		ChannelID: channelID, ModelName: imageModelName, PricePlanID: imagePlan.ID,
		Name: "image all day", Timezone: DefaultChannelModelPricingTimezone,
		Weekdays: 0x7f, StartMinute: 0, EndMinute: 1440, Enabled: true,
	}))

	videoPlan := &ChannelModelPricePlan{
		ChannelID: channelID, ModelName: videoModelName, Name: "video peak", Enabled: true,
	}
	require.NoError(t, CreateChannelModelPricePlan(videoPlan, ChannelModelPricePlanPayload{
		VideoPricingRules: &ratio_setting.VideoPricingRules{
			TextToVideoPerSecond: []ratio_setting.VideoResolutionAudioPriceRule{
				{Resolution: "540p", HasAudio: false, Price: videoPricePerSec},
			},
		},
	}))
	require.NoError(t, CreateChannelModelPriceSchedule(&ChannelModelPriceSchedule{
		ChannelID: channelID, ModelName: videoModelName, PricePlanID: videoPlan.ID,
		Name: "video all day", Timezone: DefaultChannelModelPricingTimezone,
		Weekdays: 0x7f, StartMinute: 0, EndMinute: 1440, Enabled: true,
	}))

	pricing := []Pricing{
		{ModelName: imageModelName, ModelRatio: 1, QuotaType: 0},
		{ModelName: videoModelName, ModelRatio: 1, QuotaType: 0},
	}
	metas := []ChannelPricingMeta{{
		ChannelID: channelID,
		Models:    imageModelName + "," + videoModelName,
	}}
	location, err := time.LoadLocation(DefaultChannelModelPricingTimezone)
	require.NoError(t, err)
	items := buildPricingAPIItemsAt(
		pricing,
		map[int]struct{}{channelID: {}},
		metas,
		true,
		time.Date(2026, 8, 17, 13, 0, 0, 0, location),
	)
	require.Len(t, items, 2)

	itemsByModel := make(map[string]PricingAPIItem, len(items))
	for _, item := range items {
		itemsByModel[item.ModelName] = item
	}
	imageItem := itemsByModel[imageModelName]
	require.NotNil(t, imageItem.ImagePerImageHint)
	require.InDelta(t, imagePrice, imageItem.ImagePerImageHint.MinUsdAfterChannelDiscount, 1e-9)
	require.Len(t, imageItem.ChannelList, 1)
	require.NotNil(t, imageItem.ChannelList[0].TimePricing)
	require.NotNil(t, imageItem.ChannelList[0].TimePricing.Schedules[0].Pricing.ImagePerImageHint)

	videoItem := itemsByModel[videoModelName]
	require.NotNil(t, videoItem.VideoFlatClipHint)
	require.Equal(t, "per_second", videoItem.VideoFlatClipHint.BillingMode)
	require.InDelta(t, videoPricePerSec, videoItem.VideoFlatClipHint.MinUsdAfterChannelDiscount, 1e-9)
	require.Len(t, videoItem.ChannelList, 1)
	require.NotNil(t, videoItem.ChannelList[0].TimePricing)
	require.NotNil(t, videoItem.ChannelList[0].TimePricing.Schedules[0].Pricing.VideoFlatClipHint)
}
