package doubao

import (
	"io"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubmitURL(t *testing.T) {
	assert.Equal(t, "https://api.tokenspace.net.cn/api/v3/contents/generations/tasks", SubmitURL("https://api.tokenspace.net.cn/"))
}

func TestConvertToRequestPayload_Text2Video(t *testing.T) {
	a := &TaskAdaptor{}
	body, err := a.convertToRequestPayload(&relaycommon.TaskSubmitReq{
		Model:  "doubao-seedance-2-0-260128",
		Prompt: "一只金色柴犬在樱花树下奔跑",
		Metadata: map[string]interface{}{
			"resolution": "480p",
			"ratio":      "16:9",
			"duration":   5,
			"watermark":  false,
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "doubao-seedance-2-0-260128", body.Model)
	require.Len(t, body.Content, 1)
	assert.Equal(t, "text", body.Content[0].Type)
	require.NotNil(t, body.Duration)
	assert.Equal(t, 5, int(*body.Duration))
}

func TestConvertToRequestPayload_Image2Video(t *testing.T) {
	a := &TaskAdaptor{}
	body, err := a.convertToRequestPayload(&relaycommon.TaskSubmitReq{
		Model:  "doubao-seedance-2-0-260128",
		Prompt: "让画面中的人物缓缓转身微笑",
		Images: []string{"https://example.com/photo.jpg"},
	})
	require.NoError(t, err)
	require.Len(t, body.Content, 2)
	assert.Equal(t, "text", body.Content[0].Type)
	assert.Equal(t, "image_url", body.Content[1].Type)
	assert.Equal(t, "reference_image", body.Content[1].Role)
	assert.Equal(t, "https://example.com/photo.jpg", body.Content[1].ImageURL.URL)
}

func TestConvertToRequestPayload_MultiImage2Video(t *testing.T) {
	a := &TaskAdaptor{}
	body, err := a.convertToRequestPayload(&relaycommon.TaskSubmitReq{
		Model:  "doubao-seedance-2-0-260128",
		Prompt: "多参考图生成视频",
		Images: []string{
			"https://example.com/ref1.png",
			"https://example.com/ref2.png",
		},
	})
	require.NoError(t, err)
	require.Len(t, body.Content, 3)
	assert.Equal(t, "text", body.Content[0].Type)
	for i := 1; i < len(body.Content); i++ {
		assert.Equal(t, "image_url", body.Content[i].Type)
		assert.Equal(t, "reference_image", body.Content[i].Role)
	}
	assert.Equal(t, "https://example.com/ref1.png", body.Content[1].ImageURL.URL)
	assert.Equal(t, "https://example.com/ref2.png", body.Content[2].ImageURL.URL)
}

func TestConvertToRequestPayload_Video2Video(t *testing.T) {
	a := &TaskAdaptor{}
	body, err := a.convertToRequestPayload(&relaycommon.TaskSubmitReq{
		Model:  "doubao-seedance-2-0-260128",
		Prompt: "修改一下这个视频",
		Metadata: map[string]interface{}{
			"video_urls": []interface{}{"https://example.com/reference-video.mp4"},
			"resolution": "480p",
			"ratio":      "16:9",
		},
	})
	require.NoError(t, err)
	require.Len(t, body.Content, 2)
	assert.Equal(t, "text", body.Content[0].Type)
	assert.Equal(t, "video_url", body.Content[1].Type)
	assert.Equal(t, "reference_video", body.Content[1].Role)
	assert.Equal(t, "https://example.com/reference-video.mp4", body.Content[1].VideoURL.URL)
}

func TestConvertToRequestPayload_SizeToResolution(t *testing.T) {
	a := &TaskAdaptor{}
	req := relaycommon.TaskSubmitReq{
		Model:    "doubao-seedance-2-0-260128",
		Prompt:   "test",
		Size:     "864x480",
		Metadata: map[string]interface{}{"duration": 5},
	}
	req.Metadata = common.NormalizeTaskVideoMetadata(req.Metadata, req.Size, nil, nil)
	body, err := a.convertToRequestPayload(&req)
	require.NoError(t, err)
	assert.Equal(t, "480p", body.Resolution)
	assert.Equal(t, "16:9", body.Ratio)
}

func TestBuildRequestBody_IntegrationShape(t *testing.T) {
	a := &TaskAdaptor{baseURL: "https://api.tokenspace.net.cn", apiKey: "sk-test"}
	req := relaycommon.TaskSubmitReq{
		Model:  "doubao-seedance-2-0-260128",
		Prompt: "test prompt",
		Metadata: map[string]interface{}{
			"resolution": "720p",
			"ratio":      "9:16",
			"duration":   8,
		},
	}
	payload, err := a.convertToRequestPayload(&req)
	require.NoError(t, err)
	data, err := common.Marshal(payload)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"type":"text"`)
	assert.Contains(t, string(data), `"resolution":"720p"`)
	_ = io.Discard
}
