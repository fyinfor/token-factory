package doubao

import (
	"io"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
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

// metadata.video_urls 支持素材库 asset:// 引用（不依赖视频扩展名），必须封装为上游 video_url 结构。
func TestConvertToRequestPayload_Video2VideoAssetURI(t *testing.T) {
	a := &TaskAdaptor{}
	body, err := a.convertToRequestPayload(&relaycommon.TaskSubmitReq{
		Model:  "doubao-seedance-2-0-260128",
		Prompt: "基于素材库参考视频续写",
		Metadata: map[string]interface{}{
			"video_urls": []interface{}{"asset://asset-2026xxxx"},
			"resolution": "480p",
			"ratio":      "16:9",
		},
	})
	require.NoError(t, err)
	require.Len(t, body.Content, 2)
	assert.Equal(t, "text", body.Content[0].Type)
	assert.Equal(t, "video_url", body.Content[1].Type)
	assert.Equal(t, "reference_video", body.Content[1].Role)
	require.NotNil(t, body.Content[1].VideoURL)
	assert.Equal(t, "asset://asset-2026xxxx", body.Content[1].VideoURL.URL)
}

// 参考音频列表：metadata.audio_urls → content[].type=audio_url + role=reference_audio
func TestConvertToRequestPayload_ReferenceAudio(t *testing.T) {
	a := &TaskAdaptor{}
	body, err := a.convertToRequestPayload(&relaycommon.TaskSubmitReq{
		Model:  "doubao-seedance-2-0-260128",
		Prompt: "根据参考音频生成口型一致的视频",
		Images: []string{"https://example.com/portrait.png"},
		Metadata: map[string]interface{}{
			"audio_urls": []interface{}{
				"https://example.com/ref1.mp3",
				"https://example.com/ref2.wav",
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, body.Content, 4)
	assert.Equal(t, "text", body.Content[0].Type)
	assert.Equal(t, "image_url", body.Content[1].Type)
	assert.Equal(t, "reference_image", body.Content[1].Role)
	assert.Equal(t, "audio_url", body.Content[2].Type)
	assert.Equal(t, "reference_audio", body.Content[2].Role)
	assert.Equal(t, "https://example.com/ref1.mp3", body.Content[2].AudioURL.URL)
	assert.Equal(t, "audio_url", body.Content[3].Type)
	assert.Equal(t, "reference_audio", body.Content[3].Role)
	assert.Equal(t, "https://example.com/ref2.wav", body.Content[3].AudioURL.URL)
}

// metadata.audio_urls 支持素材库 asset:// 引用（不依赖音频扩展名）
func TestConvertToRequestPayload_ReferenceAudioAssetURI(t *testing.T) {
	a := &TaskAdaptor{}
	body, err := a.convertToRequestPayload(&relaycommon.TaskSubmitReq{
		Model:  "doubao-seedance-2-0-260128",
		Prompt: "根据素材库参考音频生成口型一致的视频",
		Images: []string{"asset://asset-img-001"},
		Metadata: map[string]interface{}{
			"audio_urls": []interface{}{
				"asset://asset-audio-001",
			},
			"generate_audio": true,
		},
	})
	require.NoError(t, err)
	require.Len(t, body.Content, 3)
	assert.Equal(t, "image_url", body.Content[1].Type)
	assert.Equal(t, "asset://asset-img-001", body.Content[1].ImageURL.URL)
	assert.Equal(t, "audio_url", body.Content[2].Type)
	assert.Equal(t, "reference_audio", body.Content[2].Role)
	assert.Equal(t, "asset://asset-audio-001", body.Content[2].AudioURL.URL)
	require.NotNil(t, body.GenerateAudio)
	assert.True(t, bool(*body.GenerateAudio))
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

func TestConvertToRequestPayload_CallbackURLAndReturnLastFrame(t *testing.T) {
	a := &TaskAdaptor{}
	body, err := a.convertToRequestPayload(&relaycommon.TaskSubmitReq{
		Model:  "doubao-seedance-2-0-260128",
		Prompt: "一只猫",
		Metadata: map[string]interface{}{
			"callback_url":      "https://example.com/api/debug/seedance/callback/tok",
			"return_last_frame": true,
			"duration":          5,
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/api/debug/seedance/callback/tok", body.CallbackURL)
	require.NotNil(t, body.ReturnLastFrame)
	assert.True(t, bool(*body.ReturnLastFrame))

	data, err := common.Marshal(body)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"callback_url":"https://example.com/api/debug/seedance/callback/tok"`)
	assert.Contains(t, string(data), `"return_last_frame":true`)
}

func TestConvertToOpenAIVideo_IncludesLastFrameURLInMetadata(t *testing.T) {
	a := &TaskAdaptor{}
	upstream := []byte(`{
		"id":"cgt-1",
		"status":"succeeded",
		"content":{"video_url":"https://res.mp4","last_frame_url":"https://res-last.jpg"},
		"usage":{"completion_tokens":10,"total_tokens":10}
	}`)
	task := &model.Task{
		TaskID:   "local_task",
		Status:   model.TaskStatusSuccess,
		Progress: "100%",
		Data:     upstream,
		Properties: model.Properties{OriginModelName: "doubao-seedance-2-0-260128"},
	}
	out, err := a.ConvertToOpenAIVideo(task)
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, common.Unmarshal(out, &got))
	meta, ok := got["metadata"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "https://res-last.jpg", meta["last_frame_url"])
	output, ok := got["output"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "https://res.mp4", output["video_url"])
}
