package doubao

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateSeedanceTaskRequest_FillsDefaultPromptAndPersistsRawBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{
		"model": "Seedance 2.0/cy2",
		"content": [
			{"type": "text", "text": "帮我把视频附件里面的食物换成图片附件里面的面霜"},
			{"type": "image_url", "image_url": {"url": "https://example.com/a.png"}, "role": "reference_image"},
			{"type": "video_url", "video_url": {"url": "https://example.com/b.mp4"}, "role": "reference_video"}
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
		ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeSeedance},
	}
	err := validateSeedanceTaskRequest(c, info)
	require.Nil(t, err)

	req, gErr := relaycommon.GetTaskRequest(c)
	require.NoError(t, gErr)
	assert.Equal(t, seedanceDefaultPrompt, req.Prompt)
	assert.Equal(t, constant.TaskActionReferenceGenerate, info.Action)
	require.Len(t, req.Images, 1)
	assert.Equal(t, "https://example.com/a.png", req.Images[0])
	videos, ok := req.Metadata["video_urls"].([]string)
	require.True(t, ok)
	require.Len(t, videos, 1)
	assert.Equal(t, "https://example.com/b.mp4", videos[0])

	persisted, ok := relaycommon.GetTaskPersistedInput(c)
	require.True(t, ok)
	var got map[string]any
	require.NoError(t, common.UnmarshalJsonStr(persisted, &got))
	assert.Equal(t, seedanceDefaultPrompt, got["prompt"])
	assert.Equal(t, "480p", got["resolution"])
	assert.Equal(t, false, got["watermark"])
	content, ok := got["content"].([]any)
	require.True(t, ok)
	require.Len(t, content, 3)
}

func TestValidateSeedanceTaskRequest_KeepsClientPrompt(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{
		"model": "Seedance 2.0",
		"prompt": "拟人小猫在客厅跳舞",
		"content": [{"type": "text", "text": "拟人小猫在客厅跳舞"}],
		"resolution": "480p",
		"duration": 5
	}`)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/video/generations", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeSeedance},
	}
	err := validateSeedanceTaskRequest(c, info)
	require.Nil(t, err)

	req, gErr := relaycommon.GetTaskRequest(c)
	require.NoError(t, gErr)
	assert.Equal(t, "拟人小猫在客厅跳舞", req.Prompt)

	persisted, ok := relaycommon.GetTaskPersistedInput(c)
	require.True(t, ok)
	var got map[string]any
	require.NoError(t, common.UnmarshalJsonStr(persisted, &got))
	assert.Equal(t, "拟人小猫在客厅跳舞", got["prompt"])
	_, hasContent := got["content"]
	assert.True(t, hasContent)
}

func TestValidateRequestAndSetAction_OnlySeedanceUsesSpecialPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"m","prompt":"p","content":[{"type":"image_url","image_url":{"url":"https://x.png"}}]}`)

	// DoubaoVideo: 不走 Seedance 专属，content 不会 enrich，action 仍为文生
	c1, _ := gin.CreateTestContext(httptest.NewRecorder())
	c1.Request = httptest.NewRequest(http.MethodPost, "/v1/video/generations", bytes.NewReader(body))
	c1.Request.Header.Set("Content-Type", "application/json")
	info1 := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeDoubaoVideo},
	}
	a1 := &TaskAdaptor{ChannelType: constant.ChannelTypeDoubaoVideo}
	require.Nil(t, a1.ValidateRequestAndSetAction(c1, info1))
	_, ok := relaycommon.GetTaskPersistedInput(c1)
	assert.False(t, ok)
	assert.Equal(t, constant.TaskActionTextGenerate, info1.Action)

	// Seedance: 专属路径
	c2, _ := gin.CreateTestContext(httptest.NewRecorder())
	c2.Request = httptest.NewRequest(http.MethodPost, "/v1/video/generations", bytes.NewReader(body))
	c2.Request.Header.Set("Content-Type", "application/json")
	info2 := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeSeedance},
	}
	a2 := &TaskAdaptor{ChannelType: constant.ChannelTypeSeedance}
	require.Nil(t, a2.ValidateRequestAndSetAction(c2, info2))
	_, ok = relaycommon.GetTaskPersistedInput(c2)
	assert.True(t, ok)
	assert.Equal(t, constant.TaskActionGenerate, info2.Action) // 仅有图 → 图生
}

func TestPromptMissing(t *testing.T) {
	assert.True(t, promptMissing(nil))
	assert.True(t, promptMissing(""))
	assert.True(t, promptMissing("  "))
	assert.False(t, promptMissing("hello"))
}

func TestValidateSeedanceTaskRequest_PromotesCallbackAndReturnLastFrame(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{
		"model": "Seedance 2.0",
		"prompt": "拟人小猫在客厅跳舞",
		"content": [{"type": "text", "text": "拟人小猫在客厅跳舞"}],
		"callback_url": "https://example.com/hook",
		"return_last_frame": true,
		"resolution": "480p",
		"duration": 5
	}`)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/video/generations", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeSeedance},
	}
	err := validateSeedanceTaskRequest(c, info)
	require.Nil(t, err)

	req, gErr := relaycommon.GetTaskRequest(c)
	require.NoError(t, gErr)
	require.NotNil(t, req.Metadata)
	assert.Equal(t, "https://example.com/hook", req.Metadata["callback_url"])
	assert.Equal(t, true, req.Metadata["return_last_frame"])
}
