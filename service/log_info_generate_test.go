package service

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/require"
)

func TestAppendTimePricingInfoIncludesScheduleWindowSnapshot(t *testing.T) {
	other := map[string]interface{}{}
	appendTimePricingInfo(types.PriceData{
		TimePricingScheduleID:    3,
		TimePricingPlanID:        5,
		TimePricingPlanVersion:   2,
		TimePricingScheduleName:  "午间高峰",
		TimePricingPlanName:      "动态费率",
		TimePricingTimezone:      "Asia/Shanghai",
		TimePricingWeekdays:      0x7f,
		TimePricingStartMinute:   720,
		TimePricingEndMinute:     992,
		TimePricingEffectiveFrom: "2026-08-01",
		TimePricingEffectiveTo:   "2026-08-31",
		TimePricingMatchedAt:     1786852800,
	}, other)

	require.Equal(t, "午间高峰", other["time_pricing_schedule_name"])
	require.Equal(t, 0x7f, other["time_pricing_weekdays"])
	require.Equal(t, 720, other["time_pricing_start_minute"])
	require.Equal(t, 992, other["time_pricing_end_minute"])
	require.Equal(t, "2026-08-01", other["time_pricing_effective_from"])
	require.Equal(t, "2026-08-31", other["time_pricing_effective_to"])
}

func TestTaskBillingOtherIncludesTimePricingScheduleWindowSnapshot(t *testing.T) {
	task := &model.Task{
		PrivateData: model.TaskPrivateData{
			BillingContext: &model.TaskBillingContext{
				TimePricingScheduleID:    3,
				TimePricingPlanID:        5,
				TimePricingPlanVersion:   2,
				TimePricingScheduleName:  "午间高峰",
				TimePricingPlanName:      "动态费率",
				TimePricingTimezone:      "Asia/Shanghai",
				TimePricingWeekdays:      0x7f,
				TimePricingStartMinute:   720,
				TimePricingEndMinute:     992,
				TimePricingEffectiveFrom: "2026-08-01",
				TimePricingEffectiveTo:   "2026-08-31",
				TimePricingMatchedAt:     1786852800,
			},
		},
	}

	other := taskBillingOther(task)
	require.Equal(t, "午间高峰", other["time_pricing_schedule_name"])
	require.Equal(t, 0x7f, other["time_pricing_weekdays"])
	require.Equal(t, 720, other["time_pricing_start_minute"])
	require.Equal(t, 992, other["time_pricing_end_minute"])
}
