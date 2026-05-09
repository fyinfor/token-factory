package openaivideo

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
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
