package helper

import (
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

const channelModelTimePricingContextKey = "channel_model_time_pricing_snapshot"

type channelModelTimePricingContextValue struct {
	ChannelID int
	ModelName string
	Active    *model.ActiveChannelModelPricePlan
}

func resolveChannelModelTimePricing(c *gin.Context, channelID int, modelName string) *model.ActiveChannelModelPricePlan {
	if c != nil {
		if value, exists := c.Get(channelModelTimePricingContextKey); exists {
			if cached, ok := value.(channelModelTimePricingContextValue); ok &&
				cached.ChannelID == channelID && cached.ModelName == modelName {
				return cached.Active
			}
		}
	}
	active, ok := model.ResolveActiveChannelModelPricePlan(channelID, modelName, time.Now())
	if !ok {
		active = nil
	}
	if c != nil {
		c.Set(channelModelTimePricingContextKey, channelModelTimePricingContextValue{
			ChannelID: channelID, ModelName: modelName, Active: active,
		})
	}
	return active
}

func attachChannelModelTimePricing(priceData *types.PriceData, active *model.ActiveChannelModelPricePlan) {
	if priceData == nil || active == nil {
		return
	}
	priceData.TimePricingScheduleID = active.Schedule.ID
	priceData.TimePricingPlanID = active.Plan.ID
	priceData.TimePricingPlanVersion = active.Plan.Version
	priceData.TimePricingScheduleName = active.Schedule.Name
	priceData.TimePricingPlanName = active.Plan.Name
	priceData.TimePricingTimezone = active.Schedule.Timezone
	priceData.TimePricingWeekdays = active.Schedule.Weekdays
	priceData.TimePricingStartMinute = active.Schedule.StartMinute
	priceData.TimePricingEndMinute = active.Schedule.EndMinute
	priceData.TimePricingEffectiveFrom = active.Schedule.EffectiveFrom
	priceData.TimePricingEffectiveTo = active.Schedule.EffectiveTo
	priceData.TimePricingMatchedAt = active.MatchedAt.Unix()
	priceData.TimePricingPayload = active.Plan.PricePayload
}

func copyChannelModelTimePricingSnapshot(target *types.PriceData, source types.PriceData) {
	if target == nil || source.TimePricingPlanID <= 0 {
		return
	}
	target.TimePricingScheduleID = source.TimePricingScheduleID
	target.TimePricingPlanID = source.TimePricingPlanID
	target.TimePricingPlanVersion = source.TimePricingPlanVersion
	target.TimePricingScheduleName = source.TimePricingScheduleName
	target.TimePricingPlanName = source.TimePricingPlanName
	target.TimePricingTimezone = source.TimePricingTimezone
	target.TimePricingWeekdays = source.TimePricingWeekdays
	target.TimePricingStartMinute = source.TimePricingStartMinute
	target.TimePricingEndMinute = source.TimePricingEndMinute
	target.TimePricingEffectiveFrom = source.TimePricingEffectiveFrom
	target.TimePricingEffectiveTo = source.TimePricingEffectiveTo
	target.TimePricingMatchedAt = source.TimePricingMatchedAt
	target.TimePricingPayload = source.TimePricingPayload
}

func usesIndependentTimePricing(active *model.ActiveChannelModelPricePlan) bool {
	return active != nil && active.Payload.ResolvedMode() == model.ChannelModelPricePlanModePrice
}

func resolveChannelBillingPercents(
	c *gin.Context,
	info *relaycommon.RelayInfo,
	channelID int,
	modelName string,
	active *model.ActiveChannelModelPricePlan,
) (rawPriceDiscount, operatingCost, effectiveCost, markupDiscount float64) {
	rawPriceDiscount, operatingCost, effectiveCost = resolveChannelCostPercents(channelID)
	baseMarkup := model.ResolveChannelMarkupDiscountRate(channelID)
	if active != nil && active.Payload.ResolvedMode() == model.ChannelModelPricePlanModeRate && active.Payload.HasRateOverrides() {
		rawPriceDiscount = *active.Payload.PriceDiscountPercent
		operatingCost = *active.Payload.OperatingCostPercent
		effectiveCost = model.EffectiveCostPercent(rawPriceDiscount, operatingCost)
		baseMarkup = *active.Payload.MarkupDiscountRate
	}
	markupDiscount = effectiveMarkupDiscountPercentWithBase(c, info, channelID, modelName, baseMarkup)
	return
}

func scheduledPricingDebugLabel(active *model.ActiveChannelModelPricePlan) string {
	if active == nil {
		return ""
	}
	return fmt.Sprintf("schedule=%d plan=%d@v%d", active.Schedule.ID, active.Plan.ID, active.Plan.Version)
}
