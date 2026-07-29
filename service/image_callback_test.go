package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateImageCallbackURLEmpty(t *testing.T) {
	err := ValidateImageCallbackURL("  ")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestValidateImageCallbackURLAcceptsHTTPS(t *testing.T) {
	err := ValidateImageCallbackURL("https://example.com/hooks/image")
	if err != nil {
		assert.Contains(t, err.Error(), "callback_url")
	}
}

func TestBuildImageSuccessCallbackPayloadKeepsData(t *testing.T) {
	raw := []byte(`{"created":1710000000,"data":[{"url":"https://cdn.example.com/a.png","b64_json":""}],"usage":{"total_tokens":10}}`)
	payload := BuildImageSuccessCallbackPayload("req_1", 1700000000, raw)
	require.NotNil(t, payload)
	assert.Equal(t, "req_1", payload.ID)
	assert.Equal(t, "success", payload.Status)
	assert.Equal(t, int64(1710000000), payload.Created)
	require.Len(t, payload.Data, 1)
	assert.Equal(t, "https://cdn.example.com/a.png", payload.Data[0].Url)
	require.Contains(t, payload.Extra, "usage")

	body, err := payload.MarshalJSON()
	require.NoError(t, err)
	assert.Contains(t, string(body), `"url":"https://cdn.example.com/a.png"`)
	assert.Contains(t, string(body), `"usage"`)
}

func TestBuildImageSuccessCallbackPayloadExtractsImagesArray(t *testing.T) {
	raw := []byte(`{"images":["https://cdn.example.com/edit.png"],"created":1710000001}`)
	payload := BuildImageSuccessCallbackPayload("req_2", 1700000000, raw)
	require.Len(t, payload.Data, 1)
	assert.Equal(t, "https://cdn.example.com/edit.png", payload.Data[0].Url)
}

func TestBuildImageSuccessCallbackPayloadExtractsImageURLField(t *testing.T) {
	raw := []byte(`{"data":[{"image_url":"https://cdn.example.com/from-edit.png"}]}`)
	payload := BuildImageSuccessCallbackPayload("req_3", 1700000000, raw)
	require.Len(t, payload.Data, 1)
	assert.Equal(t, "https://cdn.example.com/from-edit.png", payload.Data[0].Url)
}
