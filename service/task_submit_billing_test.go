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

func TestExtractTotalTokensFromTaskData_VideoGenerationsTwoFormats(t *testing.T) {
	format1 := []byte(`{"status":"completed","usage":{"completion_tokens":40594,"total_tokens":40594}}`)
	require.Equal(t, 40594, extractTotalTokensFromTaskData(format1))

	format2 := []byte(`{
		"code":"success",
		"data":{
			"task_id":"task_x",
			"status":"SUCCESS",
			"data":{"usage":{"completion_tokens":191254,"total_tokens":2008000}}
		}
	}`)
	require.Equal(t, 191254, extractTotalTokensFromTaskData(format2))

	resultSummary := []byte(`{
		"id":"01a0231f-ace7-7c3f-a7ec-02f8f6dea411",
		"status":"succeeded",
		"resultSummary":{"duration":"4","usage":{"completion_tokens":38830,"total_tokens":38830}}
	}`)
	require.Equal(t, 38830, extractTotalTokensFromTaskData(resultSummary))
	require.Equal(t, 4, extractDurationFromTaskData(resultSummary))
}
