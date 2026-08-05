package dto

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAudioRequest_GetInputText_String(t *testing.T) {
	raw := []byte(`{"model":"tts-1","input":"hello world","voice":"alloy"}`)
	var req AudioRequest
	require.NoError(t, common.Unmarshal(raw, &req))
	assert.Equal(t, "hello world", req.GetInputText())
	assert.Equal(t, "tts-1", req.Model)
}

func TestAudioRequest_GetInputText_Object(t *testing.T) {
	raw := []byte(`{
		"model":"fun-asr-flash",
		"input":{"messages":[{"role":"user","content":[{"type":"input_audio"}]}]},
		"parameters":{"format":"mp3"}
	}`)
	var req AudioRequest
	require.NoError(t, common.Unmarshal(raw, &req))
	assert.Equal(t, "", req.GetInputText())
	assert.Equal(t, "fun-asr-flash", req.Model)
	assert.NotEmpty(t, req.Input)
}

func TestAudioRequest_AudioURLFields(t *testing.T) {
	raw := []byte(`{"model":"fun-asr-flash","audio_url":"https://example.com/a.mp3"}`)
	var req AudioRequest
	require.NoError(t, common.Unmarshal(raw, &req))
	assert.Equal(t, "https://example.com/a.mp3", req.AudioURL)
}
