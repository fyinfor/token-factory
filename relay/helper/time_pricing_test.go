package helper

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestModelPriceHelperPerCallIgnoresLegacyIndependentTimePricingPlan(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousDB := model.DB
	db, err := gorm.Open(
		sqlite.Open(fmt.Sprintf("file:helper_time_pricing_%s_%d?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"), time.Now().UnixNano())),
		&gorm.Config{},
	)
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.ChannelModelPricePlan{}, &model.ChannelModelPriceSchedule{}))
	model.DB = db
	model.ClearChannelModelTimePricingCache()

	previousPrice := ratio_setting.ModelPrice2JSONString()
	t.Cleanup(func() {
		model.DB = previousDB
		model.ClearChannelModelTimePricingCache()
		_ = ratio_setting.UpdateModelPriceByJSONString(previousPrice)
	})

	const (
		channelID     = 90817263
		modelName     = "scheduled-per-call-model"
		officialPrice = 0.01
		planPrice     = 0.025
	)
	require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{"`+modelName+`":0.01}`))
	plan := &model.ChannelModelPricePlan{
		ChannelID: channelID, ModelName: modelName, Name: "peak", Enabled: true,
	}
	legacyPayload, err := (model.ChannelModelPricePlanPayload{
		ModelPrice: float64PointerForHelperTest(planPrice),
	}).MarshalJSONString()
	require.NoError(t, err)
	plan.PricePayload = legacyPayload
	plan.Version = 1
	require.NoError(t, db.Create(plan).Error)
	require.NoError(t, db.Create(&model.ChannelModelPriceSchedule{
		ChannelID: channelID, ModelName: modelName, PricePlanID: plan.ID,
		Name: "all day", Timezone: model.DefaultChannelModelPricingTimezone,
		Weekdays: 0x7f, StartMinute: 0, EndMinute: 1440, Enabled: true,
	}).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/test", nil)
	info := &relaycommon.RelayInfo{
		OriginModelName: modelName,
		UsingGroup:      "default",
		UserGroup:       "default",
		ChannelMeta:     &relaycommon.ChannelMeta{ChannelId: channelID},
	}

	priceData, err := ModelPriceHelperPerCall(ctx, info)
	require.NoError(t, err)
	require.InDelta(t, officialPrice, priceData.ModelPrice, 1e-9)
	require.InDelta(t, officialPrice, priceData.GlobalModelPrice, 1e-9)
	require.Zero(t, priceData.TimePricingPlanID)
	require.Empty(t, priceData.TimePricingPayload)
}

func TestModelPriceHelperPerCallUsesActiveTimeRateWithRegularChannelPrice(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousDB := model.DB
	db, err := gorm.Open(
		sqlite.Open(fmt.Sprintf("file:helper_time_rate_%s_%d?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"), time.Now().UnixNano())),
		&gorm.Config{},
	)
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.ChannelModelPricePlan{}, &model.ChannelModelPriceSchedule{}))
	model.DB = db
	model.ClearChannelModelTimePricingCache()

	previousPrice := ratio_setting.ModelPrice2JSONString()
	previousChannelPrice := ratio_setting.ChannelModelPrice2JSONString()
	t.Cleanup(func() {
		model.DB = previousDB
		model.ClearChannelModelTimePricingCache()
		_ = ratio_setting.UpdateModelPriceByJSONString(previousPrice)
		_ = ratio_setting.UpdateChannelModelPriceByJSONString(previousChannelPrice)
	})

	const (
		channelID     = 90817264
		modelName     = "scheduled-rate-per-call-model"
		officialPrice = 0.01
		channelPrice  = 0.02
	)
	require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{"`+modelName+`":0.01}`))
	require.NoError(t, ratio_setting.UpdateChannelModelPriceByJSONString(fmt.Sprintf(`{"%d":{"%s":0.02}}`, channelID, modelName)))
	plan := &model.ChannelModelPricePlan{
		ChannelID: channelID, ModelName: modelName, Name: "peak rate", Enabled: true,
	}
	require.NoError(t, model.CreateChannelModelPricePlan(plan, model.ChannelModelPricePlanPayload{
		Mode:                 model.ChannelModelPricePlanModeRate,
		PriceDiscountPercent: float64PointerForHelperTest(50),
		OperatingCostPercent: float64PointerForHelperTest(10),
		MarkupDiscountRate:   float64PointerForHelperTest(20),
	}))
	require.NoError(t, model.CreateChannelModelPriceSchedule(&model.ChannelModelPriceSchedule{
		ChannelID: channelID, ModelName: modelName, PricePlanID: plan.ID,
		Name: "all day", Timezone: model.DefaultChannelModelPricingTimezone,
		Weekdays: 0x7f, StartMinute: 0, EndMinute: 1440, Enabled: true,
	}))

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/test", nil)
	info := &relaycommon.RelayInfo{
		OriginModelName: modelName,
		UsingGroup:      "default",
		UserGroup:       "default",
		ChannelMeta:     &relaycommon.ChannelMeta{ChannelId: channelID},
	}

	priceData, err := ModelPriceHelperPerCall(ctx, info)
	require.NoError(t, err)
	require.InDelta(t, channelPrice, priceData.ModelPrice, 1e-9)
	require.InDelta(t, officialPrice, priceData.GlobalModelPrice, 1e-9)
	require.InDelta(t, 50, priceData.RawPriceDiscountPercent, 1e-9)
	require.InDelta(t, 10, priceData.OperatingCostPercent, 1e-9)
	require.InDelta(t, 60, priceData.CostDiscountPercent, 1e-9)
	require.InDelta(t, 20, priceData.MarkupDiscountPercent, 1e-9)
	require.Equal(t, plan.ID, priceData.TimePricingPlanID)
	require.Equal(t, int((channelPrice*0.6+officialPrice*0.2)*common.QuotaPerUnit), priceData.Quota)
}

func TestModelPriceHelperPerCallTimeRateFallsBackToOfficialPrice(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousDB := model.DB
	db, err := gorm.Open(
		sqlite.Open(fmt.Sprintf("file:helper_time_rate_global_fallback_%s_%d?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"), time.Now().UnixNano())),
		&gorm.Config{},
	)
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.ChannelModelPricePlan{}, &model.ChannelModelPriceSchedule{}))
	model.DB = db
	model.ClearChannelModelTimePricingCache()

	previousPrice := ratio_setting.ModelPrice2JSONString()
	t.Cleanup(func() {
		model.DB = previousDB
		model.ClearChannelModelTimePricingCache()
		_ = ratio_setting.UpdateModelPriceByJSONString(previousPrice)
	})

	const (
		channelID     = 90817265
		modelName     = "scheduled-rate-global-fallback-model"
		officialPrice = 0.01
	)
	require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{"`+modelName+`":0.01}`))
	plan := &model.ChannelModelPricePlan{
		ChannelID: channelID, ModelName: modelName, Name: "peak rate", Enabled: true,
	}
	require.NoError(t, model.CreateChannelModelPricePlan(plan, model.ChannelModelPricePlanPayload{
		Mode:                 model.ChannelModelPricePlanModeRate,
		PriceDiscountPercent: float64PointerForHelperTest(50),
		OperatingCostPercent: float64PointerForHelperTest(10),
		MarkupDiscountRate:   float64PointerForHelperTest(20),
	}))
	require.NoError(t, model.CreateChannelModelPriceSchedule(&model.ChannelModelPriceSchedule{
		ChannelID: channelID, ModelName: modelName, PricePlanID: plan.ID,
		Name: "all day", Timezone: model.DefaultChannelModelPricingTimezone,
		Weekdays: 0x7f, StartMinute: 0, EndMinute: 1440, Enabled: true,
	}))

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/test", nil)
	info := &relaycommon.RelayInfo{
		OriginModelName: modelName,
		UsingGroup:      "default",
		UserGroup:       "default",
		ChannelMeta:     &relaycommon.ChannelMeta{ChannelId: channelID},
	}

	priceData, err := ModelPriceHelperPerCall(ctx, info)
	require.NoError(t, err)
	require.InDelta(t, officialPrice, priceData.ModelPrice, 1e-9)
	require.InDelta(t, officialPrice, priceData.GlobalModelPrice, 1e-9)
	require.InDelta(t, 60, priceData.CostDiscountPercent, 1e-9)
	require.InDelta(t, 20, priceData.MarkupDiscountPercent, 1e-9)
	require.Equal(t, plan.ID, priceData.TimePricingPlanID)
	require.Equal(t, int((officialPrice*0.6+officialPrice*0.2)*common.QuotaPerUnit), priceData.Quota)
}

func TestModelPriceHelperPerCallUserModelPricingOverridesTimeRate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousDB := model.DB
	db, err := gorm.Open(
		sqlite.Open(fmt.Sprintf("file:helper_user_rate_override_%s_%d?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"), time.Now().UnixNano())),
		&gorm.Config{},
	)
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&model.ChannelModelPricePlan{},
		&model.ChannelModelPriceSchedule{},
		&model.UserModelPricingOverride{},
		&model.UserModelPricingChannel{},
	))
	model.DB = db
	model.ClearChannelModelTimePricingCache()
	model.InvalidateUserModelPricingCache()

	previousPrice := ratio_setting.ModelPrice2JSONString()
	previousChannelPrice := ratio_setting.ChannelModelPrice2JSONString()
	t.Cleanup(func() {
		model.DB = previousDB
		model.ClearChannelModelTimePricingCache()
		model.InvalidateUserModelPricingCache()
		_ = ratio_setting.UpdateModelPriceByJSONString(previousPrice)
		_ = ratio_setting.UpdateChannelModelPriceByJSONString(previousChannelPrice)
	})

	const (
		channelID     = 90817266
		userID        = 7719201
		modelName     = "scheduled-rate-user-override-model"
		officialPrice = 0.01
	)
	require.NoError(t, ratio_setting.UpdateModelPriceByJSONString(`{"`+modelName+`":0.01}`))
	require.NoError(t, ratio_setting.UpdateChannelModelPriceByJSONString(fmt.Sprintf(`{"%d":{"%s":0.02}}`, channelID, modelName)))
	plan := &model.ChannelModelPricePlan{
		ChannelID: channelID, ModelName: modelName, Name: "peak rate", Enabled: true,
	}
	require.NoError(t, model.CreateChannelModelPricePlan(plan, model.ChannelModelPricePlanPayload{
		Mode:                 model.ChannelModelPricePlanModeRate,
		PriceDiscountPercent: float64PointerForHelperTest(50),
		OperatingCostPercent: float64PointerForHelperTest(10),
		MarkupDiscountRate:   float64PointerForHelperTest(20),
	}))
	require.NoError(t, model.CreateChannelModelPriceSchedule(&model.ChannelModelPriceSchedule{
		ChannelID: channelID, ModelName: modelName, PricePlanID: plan.ID,
		Name: "all day", Timezone: model.DefaultChannelModelPricingTimezone,
		Weekdays: 0x7f, StartMinute: 0, EndMinute: 1440, Enabled: true,
	}))
	require.NoError(t, db.Create(&model.UserModelPricingOverride{
		UserId: userID, ModelName: modelName, Mode: model.UserPricingModePriceCap,
		PriceDiscountPercent: 70, OperatingCostPercent: 5, MarkupDiscountRate: 10,
		Enabled: true,
	}).Error)
	model.InvalidateUserModelPricingCache()

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/test", nil)
	info := &relaycommon.RelayInfo{
		UserId:          userID,
		OriginModelName: modelName,
		UsingGroup:      "default",
		UserGroup:       "default",
		ChannelMeta:     &relaycommon.ChannelMeta{ChannelId: channelID},
	}

	priceData, err := ModelPriceHelperPerCall(ctx, info)
	require.NoError(t, err)
	require.InDelta(t, officialPrice, priceData.ModelPrice, 1e-9)
	require.InDelta(t, 70, priceData.RawPriceDiscountPercent, 1e-9)
	require.InDelta(t, 5, priceData.OperatingCostPercent, 1e-9)
	require.InDelta(t, 75, priceData.CostDiscountPercent, 1e-9)
	require.InDelta(t, 10, priceData.MarkupDiscountPercent, 1e-9)
	require.Zero(t, priceData.TimePricingPlanID)
	require.Equal(t, int(officialPrice*0.85*common.QuotaPerUnit), priceData.Quota)
}

func TestTryModelPriceHelperImageUsesActiveTimeRate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousDB := model.DB
	db, err := gorm.Open(
		sqlite.Open(fmt.Sprintf("file:helper_time_pricing_image_%s_%d?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"), time.Now().UnixNano())),
		&gorm.Config{},
	)
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Channel{}, &model.ChannelModelPricePlan{}, &model.ChannelModelPriceSchedule{}))
	model.DB = db
	model.ClearChannelModelTimePricingCache()
	previousImageRules := ratio_setting.ImagePricingRules2JSONString()
	t.Cleanup(func() {
		model.DB = previousDB
		model.ClearChannelModelTimePricingCache()
		_ = ratio_setting.UpdateImagePricingRulesByJSONString(previousImageRules)
	})

	const (
		channelID  = 90817267
		modelName  = "scheduled-image-model"
		imagePrice = 0.031
	)
	require.NoError(t, ratio_setting.UpdateImagePricingRulesByJSONString(
		`{"`+modelName+`":{"text_to_image_per_image":[{"resolution":"1024x1024","image_price":0.031}]}}`,
	))
	plan := &model.ChannelModelPricePlan{
		ChannelID: channelID, ModelName: modelName, Name: "image rate", Enabled: true,
	}
	require.NoError(t, model.CreateChannelModelPricePlan(plan, model.ChannelModelPricePlanPayload{
		Mode: model.ChannelModelPricePlanModeRate, PriceDiscountPercent: float64PointerForHelperTest(50),
		OperatingCostPercent: float64PointerForHelperTest(10), MarkupDiscountRate: float64PointerForHelperTest(20),
	}))
	require.NoError(t, model.CreateChannelModelPriceSchedule(&model.ChannelModelPriceSchedule{
		ChannelID: channelID, ModelName: modelName, PricePlanID: plan.ID,
		Name: "image all day", Timezone: model.DefaultChannelModelPricingTimezone,
		Weekdays: 0x7f, StartMinute: 0, EndMinute: 1440, Enabled: true,
	}))

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	info := &relaycommon.RelayInfo{
		OriginModelName: modelName,
		UsingGroup:      "default",
		UserGroup:       "default",
		ChannelMeta:     &relaycommon.ChannelMeta{ChannelId: channelID},
		Request: &dto.ImageRequest{
			Model: modelName,
			Size:  "1024x1024",
		},
	}

	priceData, ok, err := TryModelPriceHelperImage(ctx, info)
	require.NoError(t, err)
	require.True(t, ok)
	require.True(t, priceData.UsePrice)
	require.Equal(t, plan.ID, priceData.TimePricingPlanID)
	require.NotNil(t, info.ImageBilling)
	require.InDelta(t, imagePrice*0.8, info.ImageBilling.UsdPerImage, 1e-9)
	require.InDelta(t, 60, priceData.CostDiscountPercent, 1e-9)
	require.InDelta(t, 20, priceData.MarkupDiscountPercent, 1e-9)
	require.Equal(t, int(imagePrice*0.8*common.QuotaPerUnit), priceData.Quota)
}

func TestModelPriceHelperVideoUsesActiveTimeRate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousDB := model.DB
	db, err := gorm.Open(
		sqlite.Open(fmt.Sprintf("file:helper_time_pricing_video_%s_%d?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"), time.Now().UnixNano())),
		&gorm.Config{},
	)
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Channel{}, &model.ChannelModelPricePlan{}, &model.ChannelModelPriceSchedule{}))
	model.DB = db
	model.ClearChannelModelTimePricingCache()
	previousVideoRules := ratio_setting.VideoPricingRules2JSONString()
	t.Cleanup(func() {
		model.DB = previousDB
		model.ClearChannelModelTimePricingCache()
		_ = ratio_setting.UpdateVideoPricingRulesByJSONString(previousVideoRules)
	})

	const (
		channelID      = 90817268
		modelName      = "scheduled-video-model"
		pricePerSecond = 0.012
		duration       = 5
	)
	require.NoError(t, ratio_setting.UpdateVideoPricingRulesByJSONString(
		`{"`+modelName+`":{"text_to_video_per_second":[{"resolution":"540p","has_audio":false,"price":0.012}]}}`,
	))
	plan := &model.ChannelModelPricePlan{
		ChannelID: channelID, ModelName: modelName, Name: "video rate", Enabled: true,
	}
	require.NoError(t, model.CreateChannelModelPricePlan(plan, model.ChannelModelPricePlanPayload{
		Mode: model.ChannelModelPricePlanModeRate, PriceDiscountPercent: float64PointerForHelperTest(50),
		OperatingCostPercent: float64PointerForHelperTest(10), MarkupDiscountRate: float64PointerForHelperTest(20),
	}))
	require.NoError(t, model.CreateChannelModelPriceSchedule(&model.ChannelModelPriceSchedule{
		ChannelID: channelID, ModelName: modelName, PricePlanID: plan.ID,
		Name: "video all day", Timezone: model.DefaultChannelModelPricingTimezone,
		Weekdays: 0x7f, StartMinute: 0, EndMinute: 1440, Enabled: true,
	}))

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Set("task_request", relaycommon.TaskSubmitReq{
		Model: modelName, Size: "960x540", Duration: duration,
	})
	info := &relaycommon.RelayInfo{
		OriginModelName: modelName,
		UsingGroup:      "default",
		UserGroup:       "default",
		ChannelMeta:     &relaycommon.ChannelMeta{ChannelId: channelID},
	}

	priceData, err := ModelPriceHelperVideo(ctx, info)
	require.NoError(t, err)
	require.True(t, priceData.UsePrice)
	require.Equal(t, plan.ID, priceData.TimePricingPlanID)
	require.Equal(t, "per_second", priceData.VideoRuleUnit)
	require.InDelta(t, pricePerSecond, priceData.VideoChannelRulePrice, 1e-9)
	require.InDelta(t, 60, priceData.CostDiscountPercent, 1e-9)
	require.InDelta(t, 20, priceData.MarkupDiscountPercent, 1e-9)
	require.Equal(t, int(pricePerSecond*duration*0.8*common.QuotaPerUnit), priceData.Quota)
}

func float64PointerForHelperTest(value float64) *float64 { return &value }
