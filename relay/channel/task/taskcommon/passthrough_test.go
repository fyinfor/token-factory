package taskcommon

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/model_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsPassThroughBodyEnabled(t *testing.T) {
	origin := model_setting.GetGlobalSettings().PassThroughRequestEnabled
	t.Cleanup(func() {
		model_setting.GetGlobalSettings().PassThroughRequestEnabled = origin
	})

	model_setting.GetGlobalSettings().PassThroughRequestEnabled = false
	assert.False(t, IsPassThroughBodyEnabled(nil))
	assert.False(t, IsPassThroughBodyEnabled(&relaycommon.RelayInfo{}))
	assert.True(t, IsPassThroughBodyEnabled(&relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelSetting: dto.ChannelSettings{PassThroughBodyEnabled: true},
		},
	}))

	model_setting.GetGlobalSettings().PassThroughRequestEnabled = true
	assert.True(t, IsPassThroughBodyEnabled(&relaycommon.RelayInfo{}))
}

func TestBuildPassThroughRequestBody_JSONStripsPromptAndRewritesModel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{
		"model": "Seedance 2.0/cy2",
		"prompt": "小猫在客厅沙发睡觉",
		"content": [
			{"type": "text", "text": "小猫在沙发睡觉"},
			{"type": "image_url", "image_url": {"url": "https://example.com/cat.jpg"}, "role": "reference_image"},
			{"type": "video_url", "video_url": {"url": "https://example.com/ref.mp4"}, "role": "reference_video"}
		],
		"resolution": "480p",
		"ratio": "16:9",
		"duration": 5,
		"watermark": false
	}`)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/video/generations", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "doubao-seedance-2-0-260128",
			ChannelSetting:    dto.ChannelSettings{PassThroughBodyEnabled: true},
		},
	}
	reader, err := BuildPassThroughRequestBody(c, info)
	require.NoError(t, err)
	out, err := io.ReadAll(reader)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, common.Unmarshal(out, &got))
	assert.Equal(t, "doubao-seedance-2-0-260128", got["model"])
	_, hasPrompt := got["prompt"]
	assert.False(t, hasPrompt, "prompt must not be forwarded upstream")
	assert.Equal(t, "480p", got["resolution"])
	assert.Equal(t, "16:9", got["ratio"])
	assert.Equal(t, float64(5), got["duration"])
	assert.Equal(t, false, got["watermark"])

	content, ok := got["content"].([]any)
	require.True(t, ok)
	require.Len(t, content, 3)
	first, _ := content[0].(map[string]any)
	assert.Equal(t, "text", first["type"])
	assert.Equal(t, "小猫在沙发睡觉", first["text"])
	second, _ := content[1].(map[string]any)
	assert.Equal(t, "image_url", second["type"])
	third, _ := content[2].(map[string]any)
	assert.Equal(t, "video_url", third["type"])
}

func TestRewriteJSONPassThroughBody_KeepsUnknownFields(t *testing.T) {
	raw := []byte(`{"model":"m1","prompt":"p","generate_audio":true,"seed":7}`)
	out, err := rewriteJSONPassThroughBody(raw, "upstream-model")
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, common.Unmarshal(out, &got))
	assert.Equal(t, "upstream-model", got["model"])
	assert.Equal(t, true, got["generate_audio"])
	assert.Equal(t, float64(7), got["seed"])
	_, hasPrompt := got["prompt"]
	assert.False(t, hasPrompt)
}

func TestBuildPassThroughRequestBody_KeepsCallbackURLAndReturnLastFrame(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{
		"model": "Seedance 2.0",
		"prompt": "gateway only",
		"content": [{"type": "text", "text": "一只猫在窗台晒太阳"}],
		"resolution": "720p",
		"duration": 5,
		"callback_url": "https://example.com/api/debug/seedance/callback/tok_test",
		"return_last_frame": true
	}`)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/video/generations", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "doubao-seedance-2-0-260128",
			ChannelSetting:    dto.ChannelSettings{PassThroughBodyEnabled: true},
		},
	}
	reader, err := BuildPassThroughRequestBody(c, info)
	require.NoError(t, err)
	out, err := io.ReadAll(reader)
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, common.Unmarshal(out, &got))
	assert.Equal(t, "doubao-seedance-2-0-260128", got["model"])
	_, hasPrompt := got["prompt"]
	assert.False(t, hasPrompt)
	assert.Equal(t, "https://example.com/api/debug/seedance/callback/tok_test", got["callback_url"])
	assert.Equal(t, true, got["return_last_frame"])
}
