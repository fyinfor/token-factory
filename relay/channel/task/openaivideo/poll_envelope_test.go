package openaivideo

import (
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
)

func TestParseTaskResult_CodeDataEnvelopeArkSucceeded(t *testing.T) {
	a := &TaskAdaptor{}
	body := []byte(`{
		"code": 0,
		"message": "success",
		"data": {
			"id": "cgt-upstream-1",
			"status": "succeeded",
			"content": { "video_url": "https://example.com/out.mp4" }
		}
	}`)
	ti, err := a.ParseTaskResult(body)
	require.NoError(t, err)
	require.Equal(t, string(model.TaskStatusSuccess), ti.Status)
	require.Equal(t, "https://example.com/out.mp4", ti.Url)
}

func TestParseTaskResult_CodeDataEnvelopeArkOutputVideoURL(t *testing.T) {
	a := &TaskAdaptor{}
	body := []byte(`{
		"code": 0,
		"data": {
			"id": "cgt-2",
			"status": "completed",
			"output": { "video_url": "https://cdn.example.com/v.mp4" }
		}
	}`)
	ti, err := a.ParseTaskResult(body)
	require.NoError(t, err)
	require.Equal(t, string(model.TaskStatusSuccess), ti.Status)
	require.Equal(t, "https://cdn.example.com/v.mp4", ti.Url)
}

func TestParseTaskResult_CodeDataEnvelopeArkFailedWithoutOutput(t *testing.T) {
	a := &TaskAdaptor{}
	body := []byte(`{
		"code": 0,
		"message": "success",
		"data": {
			"id": "cgt-failed-1",
			"status": "failed",
			"error": { "message": "generation failed" }
		}
	}`)
	ti, err := a.ParseTaskResult(body)
	require.NoError(t, err)
	require.Equal(t, string(model.TaskStatusFailure), ti.Status)
	require.Equal(t, "generation failed", ti.Reason)
}

func TestParseTaskResult_CodeDataEnvelopeArkTaskIDFailed(t *testing.T) {
	a := &TaskAdaptor{}
	body := []byte(`{
		"code": 0,
		"data": {
			"task_id": "task-upstream-1",
			"status": "failed",
			"error": { "code": "video_failed" }
		}
	}`)
	ti, err := a.ParseTaskResult(body)
	require.NoError(t, err)
	require.Equal(t, string(model.TaskStatusFailure), ti.Status)
	require.Equal(t, "video_failed", ti.Reason)
}

func TestApplyTokenFactoryTaskBillingHeaderOverridesLocalEstimate(t *testing.T) {
	chDiscount := float64(100)
	info := &relaycommon.RelayInfo{
		PriceData: types.PriceData{
			ModelPrice: 0.01,
			Quota:      22500,
			GroupRatioInfo: types.GroupRatioInfo{
				GroupRatio: 1,
			},
		},
	}
	resp := &http.Response{Header: make(http.Header)}
	resp.Header.Set("X-New-Api-Task-Billing", `{
		"ModelPrice": 0,
		"ModelRatio": 0,
		"VideoOutputTokens": 50000,
		"UsePrice": true,
		"ChannelPriceDiscount": 100,
		"Quota": 1183335,
		"VideoRuleUnit": "`+service.VideoRuleUnitPerToken+`",
		"VideoBillingMode": "text_to_video",
		"VideoChannelRulePrice": 47.3334,
		"VideoGlobalRulePrice": 0,
		"VideoRuleWidth": 1280,
		"VideoRuleHeight": 720,
		"VideoRuleHasAudio": false
	}`)
	info.PriceData.ChannelPriceDiscount = &chDiscount

	applyTokenFactoryTaskBillingHeader(resp, info)

	require.Equal(t, 1183335, info.PriceData.Quota)
	require.Equal(t, 1183335, info.PriceData.QuotaToPreConsume)
	require.Equal(t, 50000, info.PriceData.VideoOutputTokens)
	require.Equal(t, service.VideoRuleUnitPerToken, info.PriceData.VideoRuleUnit)
	require.Equal(t, "text_to_video", info.PriceData.VideoBillingMode)
	require.InDelta(t, 47.3334, info.PriceData.VideoChannelRulePrice, 1e-9)
	require.Equal(t, 1.0, info.PriceData.GroupRatioInfo.GroupRatio)
}

func TestApplyTokenFactoryTaskBillingHeaderCopiesUpstreamOther(t *testing.T) {
	info := &relaycommon.RelayInfo{
		PriceData: types.PriceData{
			Quota: 22500,
			GroupRatioInfo: types.GroupRatioInfo{
				GroupRatio: 1,
			},
		},
	}
	resp := &http.Response{Header: make(http.Header)}
	resp.Header.Set("X-New-Api-Task-Billing", `{"Quota":16700000,"UsePrice":true,"ModelPrice":0}`)
	resp.Header.Set(service.TaskBillingOtherHeader, `{
		"billing_mode":"video_per_second",
		"video_seconds":5,
		"video_resolution":"720p",
		"video_price_per_second":3.34,
		"video_billed_quota":16700000,
		"video_quota_per_unit":500000
	}`)

	applyTokenFactoryTaskBillingHeader(resp, info)

	require.Equal(t, 16700000, info.PriceData.Quota)
	require.Equal(t, "video_per_second", info.UpstreamTaskBillingOther["billing_mode"])
	require.Equal(t, float64(5), info.UpstreamTaskBillingOther["video_seconds"])
	require.Equal(t, "720p", info.UpstreamTaskBillingOther["video_resolution"])
	require.Equal(t, float64(16700000), info.UpstreamTaskBillingOther["video_billed_quota"])
}
