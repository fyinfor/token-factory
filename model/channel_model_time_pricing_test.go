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
		Mode: ChannelModelPricePlanModeRate, PriceDiscountPercent: float64Pointer(60),
		OperatingCostPercent: float64Pointer(5), MarkupDiscountRate: float64Pointer(10),
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
	require.Equal(t, ChannelModelPricePlanModeRate, active.Payload.ResolvedMode())
	require.InDelta(t, 60, *active.Payload.PriceDiscountPercent, 1e-9)

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
	require.NoError(t, CreateChannelModelPricePlan(plan, ChannelModelPricePlanPayload{
		Mode: ChannelModelPricePlanModeRate, PriceDiscountPercent: float64Pointer(60),
		OperatingCostPercent: float64Pointer(5), MarkupDiscountRate: float64Pointer(10),
	}))

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
	var conflictErr *ChannelModelScheduleConflictError
	require.ErrorAs(t, err, &conflictErr)
	require.Equal(t, first.ID, conflictErr.ConflictingSchedule.ID)
	require.Equal(t, "A", conflictErr.ConflictingSchedule.Name)
}

func TestDeleteChannelModelPricePlanInUse(t *testing.T) {
	withChannelModelTimePricingDB(t)
	plan := &ChannelModelPricePlan{ChannelID: 1, ModelName: "m", Name: "peak", Enabled: true}
	require.NoError(t, CreateChannelModelPricePlan(plan, ChannelModelPricePlanPayload{
		Mode: ChannelModelPricePlanModeRate, PriceDiscountPercent: float64Pointer(60),
		OperatingCostPercent: float64Pointer(5), MarkupDiscountRate: float64Pointer(10),
	}))
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
	require.NoError(t, CreateChannelModelPricePlan(plan, ChannelModelPricePlanPayload{
		Mode: ChannelModelPricePlanModeRate, PriceDiscountPercent: float64Pointer(60),
		OperatingCostPercent: float64Pointer(5), MarkupDiscountRate: float64Pointer(10),
	}))

	plan.Name = "peak v2"
	require.NoError(t, UpdateChannelModelPricePlan(plan, ChannelModelPricePlanPayload{
		Mode: ChannelModelPricePlanModeRate, PriceDiscountPercent: float64Pointer(70),
		OperatingCostPercent: float64Pointer(6), MarkupDiscountRate: float64Pointer(12),
	}))
	require.Equal(t, 2, plan.Version)
	payload, err := ParseChannelModelPricePlanPayload(plan.PricePayload)
	require.NoError(t, err)
	require.Equal(t, ChannelModelPricePlanModeRate, payload.ResolvedMode())
	require.InDelta(t, 70, *payload.PriceDiscountPercent, 1e-9)
}

func TestChannelModelIndependentPricePlanIsRejected(t *testing.T) {
	payload := ChannelModelPricePlanPayload{ModelPrice: float64Pointer(1)}
	require.ErrorContains(t, payload.NormalizeAndValidate(), "only dynamic rate plans")
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

func TestBuildPricingAPIItemsIgnoresLegacyIndependentPrice(t *testing.T) {
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
	legacyPayload := ChannelModelPricePlanPayload{
		ModelRatio:      float64Pointer(20),
		CompletionRatio: float64Pointer(1),
		CacheRatio:      float64Pointer(0.1),
	}
	rawLegacyPayload, err := legacyPayload.MarshalJSONString()
	require.NoError(t, err)
	plan.PricePayload = rawLegacyPayload
	plan.Version = 1
	require.NoError(t, DB.Create(plan).Error)
	require.NoError(t, DB.Create(&ChannelModelPriceSchedule{
		ChannelID: 7, ModelName: "canonical-model", PricePlanID: plan.ID,
		Name: "weekday peak", Timezone: DefaultChannelModelPricingTimezone,
		Weekdays: 1 << int(time.Monday), StartMinute: 12 * 60, EndMinute: 16 * 60,
		Enabled: true,
	}).Error)

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
	require.InDelta(t, 5, activeChannel.ModelRatio, 1e-9)
	require.InDelta(t, 2, activeChannel.CompletionRatio, 1e-9)
	require.InDelta(t, 0.2, activeChannel.CacheRatio, 1e-9)
	require.InDelta(t, 1.5, activeChannel.CreateCacheRatio, 1e-9)
	require.Nil(t, activeChannel.TimePricing)

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
	require.Nil(t, regularChannel.TimePricing)
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

func TestBuildPricingAPIItemsIgnoresLegacyIndependentMediaPricing(t *testing.T) {
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
	imagePayload, err := (ChannelModelPricePlanPayload{
		ImagePricingRules: &ratio_setting.ImagePricingRules{
			TextToImagePerImage: []ratio_setting.ImageResolutionPerImageRule{
				{Resolution: "1024x1024", ImagePrice: imagePrice},
			},
		},
	}).MarshalJSONString()
	require.NoError(t, err)
	imagePlan.PricePayload = imagePayload
	imagePlan.Version = 1
	require.NoError(t, DB.Create(imagePlan).Error)
	require.NoError(t, DB.Create(&ChannelModelPriceSchedule{
		ChannelID: channelID, ModelName: imageModelName, PricePlanID: imagePlan.ID,
		Name: "image all day", Timezone: DefaultChannelModelPricingTimezone,
		Weekdays: 0x7f, StartMinute: 0, EndMinute: 1440, Enabled: true,
	}).Error)

	videoPlan := &ChannelModelPricePlan{
		ChannelID: channelID, ModelName: videoModelName, Name: "video peak", Enabled: true,
	}
	videoPayload, err := (ChannelModelPricePlanPayload{
		VideoPricingRules: &ratio_setting.VideoPricingRules{
			TextToVideoPerSecond: []ratio_setting.VideoResolutionAudioPriceRule{
				{Resolution: "540p", HasAudio: false, Price: videoPricePerSec},
			},
		},
	}).MarshalJSONString()
	require.NoError(t, err)
	videoPlan.PricePayload = videoPayload
	videoPlan.Version = 1
	require.NoError(t, DB.Create(videoPlan).Error)
	require.NoError(t, DB.Create(&ChannelModelPriceSchedule{
		ChannelID: channelID, ModelName: videoModelName, PricePlanID: videoPlan.ID,
		Name: "video all day", Timezone: DefaultChannelModelPricingTimezone,
		Weekdays: 0x7f, StartMinute: 0, EndMinute: 1440, Enabled: true,
	}).Error)

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
	require.Nil(t, imageItem.ImagePerImageHint)
	require.Len(t, imageItem.ChannelList, 1)
	require.Nil(t, imageItem.ChannelList[0].TimePricing)

	videoItem := itemsByModel[videoModelName]
	require.Nil(t, videoItem.VideoFlatClipHint)
	require.Len(t, videoItem.ChannelList, 1)
	require.Nil(t, videoItem.ChannelList[0].TimePricing)
}

func TestCreateChannelModelRateRulesForMultipleModels(t *testing.T) {
	withChannelModelTimePricingDB(t)

	mutation := &ChannelModelRateRuleMutation{
		ChannelID: 21, ModelNames: []string{"model-a", "model-b", "model-a"}, Name: "晚高峰",
		PriceDiscountPercent: 70, OperatingCostPercent: 5, MarkupDiscountRate: 10,
		Timezone: DefaultChannelModelPricingTimezone, Weekdays: 0x7f,
		StartMinute: 18 * 60, EndMinute: 23 * 60, Enabled: true, UserID: 1,
	}
	require.NoError(t, CreateChannelModelRateRules(mutation))

	rules, err := ListChannelModelRateRules(21)
	require.NoError(t, err)
	require.Len(t, rules, 2)
	require.Equal(t, "model-a", rules[0].ModelName)
	require.Equal(t, "model-b", rules[1].ModelName)
	for _, rule := range rules {
		require.InDelta(t, 75, rule.EffectiveCostPercent, 1e-9)
		require.Equal(t, 1, rule.PlanVersion)
	}
}

func TestCreateDisabledChannelModelRateRuleStaysDisabled(t *testing.T) {
	withChannelModelTimePricingDB(t)

	mutation := &ChannelModelRateRuleMutation{
		ChannelID: 25, ModelNames: []string{"model-a"}, Name: "停用规则",
		PriceDiscountPercent: 70, OperatingCostPercent: 5, MarkupDiscountRate: 10,
		Timezone: DefaultChannelModelPricingTimezone, Weekdays: 0x7f,
		StartMinute: 18 * 60, EndMinute: 23 * 60, Enabled: false, UserID: 1,
	}
	require.NoError(t, CreateChannelModelRateRules(mutation))
	rules, err := ListChannelModelRateRules(25)
	require.NoError(t, err)
	require.Len(t, rules, 1)
	require.False(t, rules[0].Enabled)
	var plan ChannelModelPricePlan
	var schedule ChannelModelPriceSchedule
	require.NoError(t, DB.First(&plan, rules[0].PlanID).Error)
	require.NoError(t, DB.First(&schedule, rules[0].ScheduleID).Error)
	require.False(t, plan.Enabled)
	require.False(t, schedule.Enabled)
}

func TestCreateChannelModelRateRulesRollsBackWhenAnyModelConflicts(t *testing.T) {
	withChannelModelTimePricingDB(t)

	base := &ChannelModelRateRuleMutation{
		ChannelID: 22, ModelNames: []string{"model-b"}, Name: "已有规则",
		PriceDiscountPercent: 70, OperatingCostPercent: 5, MarkupDiscountRate: 10,
		Timezone: DefaultChannelModelPricingTimezone, Weekdays: 0x7f,
		StartMinute: 18 * 60, EndMinute: 23 * 60, Enabled: true, UserID: 1,
	}
	require.NoError(t, CreateChannelModelRateRules(base))

	conflicting := *base
	conflicting.ModelNames = []string{"model-a", "model-b"}
	conflicting.Name = "冲突规则"
	err := CreateChannelModelRateRules(&conflicting)
	require.ErrorIs(t, err, ErrChannelModelScheduleConflict)

	var modelAPlanCount int64
	require.NoError(t, DB.Model(&ChannelModelPricePlan{}).
		Where("channel_id = ? AND model_name = ?", 22, "model-a").Count(&modelAPlanCount).Error)
	require.Zero(t, modelAPlanCount, "the first model must be rolled back when a later model conflicts")
	rules, listErr := ListChannelModelRateRules(22)
	require.NoError(t, listErr)
	require.Len(t, rules, 1)
}

func TestUpdateAndDeleteChannelModelRateRule(t *testing.T) {
	withChannelModelTimePricingDB(t)

	mutation := &ChannelModelRateRuleMutation{
		ChannelID: 23, ModelNames: []string{"model-a"}, Name: "初始规则",
		PriceDiscountPercent: 70, OperatingCostPercent: 5, MarkupDiscountRate: 10,
		Timezone: DefaultChannelModelPricingTimezone, Weekdays: 0x7f,
		StartMinute: 18 * 60, EndMinute: 23 * 60, Enabled: true, UserID: 1,
	}
	require.NoError(t, CreateChannelModelRateRules(mutation))
	rules, err := ListChannelModelRateRules(23)
	require.NoError(t, err)
	require.Len(t, rules, 1)

	mutation.Name = "更新规则"
	mutation.PriceDiscountPercent = 55
	mutation.OperatingCostPercent = 7
	mutation.MarkupDiscountRate = 13
	mutation.StartMinute = 9 * 60
	mutation.EndMinute = 12 * 60
	mutation.Enabled = false
	require.NoError(t, UpdateChannelModelRateRule(rules[0].ScheduleID, mutation))

	rules, err = ListChannelModelRateRules(23)
	require.NoError(t, err)
	require.Len(t, rules, 1)
	require.Equal(t, "更新规则", rules[0].Name)
	require.Equal(t, 2, rules[0].PlanVersion)
	require.False(t, rules[0].Enabled)
	require.InDelta(t, 62, rules[0].EffectiveCostPercent, 1e-9)
	require.InDelta(t, 13, rules[0].MarkupDiscountRate, 1e-9)

	require.NoError(t, DeleteChannelModelRateRule(23, rules[0].ScheduleID))
	rules, err = ListChannelModelRateRules(23)
	require.NoError(t, err)
	require.Empty(t, rules)
	var planCount int64
	require.NoError(t, DB.Model(&ChannelModelPricePlan{}).Where("channel_id = ?", 23).Count(&planCount).Error)
	require.Zero(t, planCount)
}

func TestUpdateChannelModelRateRuleDetachesSharedLegacyPlan(t *testing.T) {
	withChannelModelTimePricingDB(t)

	payload := ChannelModelPricePlanPayload{
		Mode: ChannelModelPricePlanModeRate, PriceDiscountPercent: float64Pointer(70),
		OperatingCostPercent: float64Pointer(5), MarkupDiscountRate: float64Pointer(10),
	}
	raw, err := payload.MarshalJSONString()
	require.NoError(t, err)
	sharedPlan := &ChannelModelPricePlan{
		ChannelID: 24, ModelName: "model-a", Name: "共享旧计划", PricePayload: raw,
		Version: 3, Enabled: true, CreatedByUserID: 1, UpdatedByUserID: 1,
	}
	require.NoError(t, DB.Create(sharedPlan).Error)
	first := &ChannelModelPriceSchedule{
		ChannelID: 24, ModelName: "model-a", PricePlanID: sharedPlan.ID, Name: "上午",
		Timezone: DefaultChannelModelPricingTimezone, Weekdays: 0x7f,
		StartMinute: 9 * 60, EndMinute: 12 * 60, Enabled: true,
	}
	second := &ChannelModelPriceSchedule{
		ChannelID: 24, ModelName: "model-a", PricePlanID: sharedPlan.ID, Name: "下午",
		Timezone: DefaultChannelModelPricingTimezone, Weekdays: 0x7f,
		StartMinute: 14 * 60, EndMinute: 17 * 60, Enabled: true,
	}
	require.NoError(t, DB.Create(first).Error)
	require.NoError(t, DB.Create(second).Error)

	mutation := &ChannelModelRateRuleMutation{
		ChannelID: 24, ModelNames: []string{"model-a"}, Name: "上午更新",
		PriceDiscountPercent: 50, OperatingCostPercent: 5, MarkupDiscountRate: 20,
		Timezone: DefaultChannelModelPricingTimezone, Weekdays: 0x7f,
		StartMinute: 9 * 60, EndMinute: 12 * 60, Enabled: false, UserID: 2,
	}
	require.NoError(t, UpdateChannelModelRateRule(first.ID, mutation))

	var updatedFirst, unchangedSecond ChannelModelPriceSchedule
	require.NoError(t, DB.First(&updatedFirst, first.ID).Error)
	require.NoError(t, DB.First(&unchangedSecond, second.ID).Error)
	require.NotEqual(t, sharedPlan.ID, updatedFirst.PricePlanID)
	require.Equal(t, sharedPlan.ID, unchangedSecond.PricePlanID)
	var originalPlan, detachedPlan ChannelModelPricePlan
	require.NoError(t, DB.First(&originalPlan, sharedPlan.ID).Error)
	require.NoError(t, DB.First(&detachedPlan, updatedFirst.PricePlanID).Error)
	require.True(t, originalPlan.Enabled)
	require.False(t, detachedPlan.Enabled)
	require.Equal(t, 4, detachedPlan.Version)
	originalPayload, err := ParseChannelModelPricePlanPayload(originalPlan.PricePayload)
	require.NoError(t, err)
	require.InDelta(t, 70, *originalPayload.PriceDiscountPercent, 1e-9)
}
