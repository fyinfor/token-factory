package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTextProfitShareAmountsKeepsRoundedZeroAuditRow(t *testing.T) {
	slice, reward, shouldRecord := textProfitShareAmounts(1, 1, 1000, 25)

	require.Zero(t, slice)
	require.Zero(t, reward)
	require.True(t, shouldRecord)
}

func TestTextProfitShareAmountsNormalReward(t *testing.T) {
	slice, reward, shouldRecord := textProfitShareAmounts(1500, 1000, 2500, 50)

	require.Equal(t, 500, slice)
	require.Equal(t, 125, reward)
	require.True(t, shouldRecord)
}

func TestTextProfitShareAmountsRecordsRequestWithoutMarkup(t *testing.T) {
	slice, reward, shouldRecord := textProfitShareAmounts(1000, 1000, 2500, 0)

	require.Zero(t, slice)
	require.Zero(t, reward)
	require.True(t, shouldRecord)
}

func TestTextProfitShareAmountsSkipsUnbilledRequest(t *testing.T) {
	slice, reward, shouldRecord := textProfitShareAmounts(0, 0, 2500, 0)

	require.Zero(t, slice)
	require.Zero(t, reward)
	require.False(t, shouldRecord)
}
