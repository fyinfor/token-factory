package relay

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/require"
)

func TestRestoreASRPriceDataSnapshot(t *testing.T) {
	expected := types.PriceData{
		UsePrice:               true,
		ModelPrice:             0.002,
		GlobalModelPrice:       0.002,
		TimePricingScheduleID:  7,
		TimePricingPlanID:      9,
		TimePricingPlanVersion: 3,
		TimePricingWeekdays:    0x7f,
		TimePricingStartMinute: 720,
		TimePricingEndMinute:   992,
		TimePricingPayload:     `{"ASRPrice":0.002}`,
	}
	raw, err := common.Marshal(expected)
	require.NoError(t, err)
	task := &model.AsrTask{PriceDataSnapshot: string(raw)}
	info := &relaycommon.RelayInfo{}

	require.True(t, restoreASRPriceDataSnapshot(task, info))
	require.Equal(t, expected.ModelPrice, info.PriceData.ModelPrice)
	require.Equal(t, expected.TimePricingPlanID, info.PriceData.TimePricingPlanID)
	require.Equal(t, expected.TimePricingWeekdays, info.PriceData.TimePricingWeekdays)
	require.Equal(t, expected.TimePricingStartMinute, info.PriceData.TimePricingStartMinute)
	require.Equal(t, expected.TimePricingEndMinute, info.PriceData.TimePricingEndMinute)
	require.Equal(t, expected.TimePricingPayload, info.PriceData.TimePricingPayload)
}
