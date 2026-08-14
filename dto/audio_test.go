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

func TestASRTaskSubmitRequest_DiarizationEnabledStringAndBool(t *testing.T) {
	var fromString ASRTaskSubmitRequest
	require.NoError(t, common.Unmarshal([]byte(`{"model":"fun-asr","diarization_enabled":"true"}`), &fromString))
	require.NotNil(t, fromString.DiarizationEnabled)
	assert.True(t, bool(*fromString.DiarizationEnabled))
	require.NotNil(t, fromString.DiarizationEnabled.BoolPtr())
	assert.True(t, *fromString.DiarizationEnabled.BoolPtr())

	var fromBool ASRTaskSubmitRequest
	require.NoError(t, common.Unmarshal([]byte(`{"model":"fun-asr","diarization_enabled":true}`), &fromBool))
	require.NotNil(t, fromBool.DiarizationEnabled)
	assert.True(t, bool(*fromBool.DiarizationEnabled))

	var fromOne ASRTaskSubmitRequest
	require.NoError(t, common.Unmarshal([]byte(`{"model":"fun-asr","diarization_enabled":"1"}`), &fromOne))
	require.NotNil(t, fromOne.DiarizationEnabled)
	assert.True(t, bool(*fromOne.DiarizationEnabled))

	var omitted ASRTaskSubmitRequest
	require.NoError(t, common.Unmarshal([]byte(`{"model":"fun-asr"}`), &omitted))
	assert.Nil(t, omitted.DiarizationEnabled)
	assert.Nil(t, omitted.DiarizationEnabled.BoolPtr())
}
