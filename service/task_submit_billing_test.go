package service

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/require"
)

func TestResolveActualTaskQuotaOnSubmit_TokenFactoryOpenKeepsLocalEstimate(t *testing.T) {
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType: constant.ChannelTypeTokenFactoryOpen,
		},
		UpstreamTaskBillingOther: map[string]interface{}{
			"video_total_tokens": 999999,
			"video_resolution":   "720p",
		},
		PriceData: types.PriceData{
			Quota:      2366703,
			ModelRatio: 100,
			GroupRatioInfo: types.GroupRatioInfo{
				GroupRatio: 1,
			},
		},
	}
	taskData := []byte(`{"total_tokens":999999999}`)

	actual := ResolveActualTaskQuotaOnSubmit(nil, info, taskData, 2366703)

	require.Equal(t, 2366703, actual)
}
