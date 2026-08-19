package controller

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func TestFormatChannelModelScheduleConflict(t *testing.T) {
	schedule := model.ChannelModelPriceSchedule{
		ID:            42,
		ModelName:     "gpt-4o",
		Name:          "工作日晚高峰测试版",
		Weekdays:      0x3e,
		StartMinute:   18 * 60,
		EndMinute:     23 * 60,
		EffectiveFrom: "2026-08-01",
		EffectiveTo:   "2026-08-31",
	}

	require.Equal(t,
		"模型「gpt-4o」的动态费率与已启用规则「工作日晚高峰测试版」（规则 ID：42，工作日 18:00–23:00，生效日期：2026-08-01 至 2026-08-31）冲突。请调整重复日期、时间范围或生效日期，或先停用该规则。",
		formatChannelModelScheduleConflict(schedule),
	)
}

func TestFormatChannelPricingConflictRangeDetails(t *testing.T) {
	weekendMask := (1 << int(time.Saturday)) | (1 << int(time.Sunday))
	require.Equal(t, "周六、周日", formatChannelPricingWeekdays(weekendMask))
	require.Equal(t, "24:00", formatChannelPricingMinute(1440))
	require.Equal(t, "长期有效", formatChannelPricingDateRange("", ""))
	require.Equal(t, "自 2026-08-01 起", formatChannelPricingDateRange("2026-08-01", ""))
	require.Equal(t, "截至 2026-08-31", formatChannelPricingDateRange("", "2026-08-31"))
}
