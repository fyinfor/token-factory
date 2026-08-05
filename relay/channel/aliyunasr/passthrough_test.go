package aliyunasr

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRewriteJSONModelField(t *testing.T) {
	raw := []byte(`{"model":"client-model","input":{"messages":[{"role":"user"}]},"parameters":{"format":"mp3"},"extra":1}`)
	out, err := rewriteJSONModelField(raw, "upstream-model")
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, common.Unmarshal(out, &got))
	assert.Equal(t, "upstream-model", got["model"])
	assert.Equal(t, float64(1), got["extra"])
	_, ok := got["input"]
	assert.True(t, ok)
}

func TestTryPassThroughNativeSyncJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{
		"model": "fun-asr-flash",
		"input": {
			"messages": [
				{
					"role": "user",
					"content": [
						{
							"type": "input_audio",
							"input_audio": {"data": "https://example.com/a.mp3"}
						}
					]
				}
			]
		},
		"parameters": {"format": "mp3", "sample_rate": "16000"}
	}`)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/audio/transcriptions", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	require.NoError(t, common.ReplaceRequestBody(c, body))

	req := &dto.AudioRequest{Model: "mapped-fun-asr"}
	info := &relaycommon.RelayInfo{}
	reader, ok, err := tryPassThroughNativeSyncJSON(c, info, req)
	require.NoError(t, err)
	require.True(t, ok)
	require.NotNil(t, reader)

	out, err := io.ReadAll(reader)
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, common.Unmarshal(out, &got))
	assert.Equal(t, "mapped-fun-asr", got["model"])
	assert.NotNil(t, got["input"])
	assert.NotNil(t, got["parameters"])
}

func TestTryPassThroughNativeSyncJSON_RejectsAudioURLOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"fun-asr-flash","audio_url":"https://example.com/a.mp3"}`)

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/audio/transcriptions", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	require.NoError(t, common.ReplaceRequestBody(c, body))

	reader, ok, err := tryPassThroughNativeSyncJSON(c, &relaycommon.RelayInfo{}, &dto.AudioRequest{Model: "fun-asr-flash"})
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Nil(t, reader)
}
