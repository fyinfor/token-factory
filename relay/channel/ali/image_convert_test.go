package ali

import (
	"encoding/json"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
)

func TestOaiImage2AliImageRequest_SyncWithHTTPSImage(t *testing.T) {
	info := &relaycommon.RelayInfo{}
	req := dto.ImageRequest{
		Model:  "qwen-image-2.0-pro",
		Prompt: "小猫睡觉",
		N:      common.GetPointer(uint(1)),
		Size:   "2048x2048",
		Image:  json.RawMessage(`"https://xxx.png"`),
	}
	watermark := false
	req.Watermark = &watermark

	got, err := oaiImage2AliImageRequest(info, req, true)
	require.NoError(t, err)
	require.Equal(t, "qwen-image-2.0-pro", got.Model)
	require.Equal(t, "2048*2048", got.Parameters.Size)
	require.Equal(t, 1, got.Parameters.N)
	require.NotNil(t, got.Parameters.Watermark)
	require.False(t, *got.Parameters.Watermark)

	input, ok := got.Input.(AliImageInput)
	require.True(t, ok)
	require.Len(t, input.Messages, 1)
	content, ok := input.Messages[0].Content.([]AliMediaContent)
	require.True(t, ok)
	require.Equal(t, []AliMediaContent{
		{Image: "https://xxx.png"},
		{Text: "小猫睡觉"},
	}, content)
}

func TestOaiImage2AliImageRequest_SyncTextOnly(t *testing.T) {
	info := &relaycommon.RelayInfo{}
	req := dto.ImageRequest{
		Model:  "qwen-image-plus",
		Prompt: "一只猫",
		N:      lo.ToPtr(uint(1)),
	}

	got, err := oaiImage2AliImageRequest(info, req, true)
	require.NoError(t, err)

	input, ok := got.Input.(AliImageInput)
	require.True(t, ok)
	content, ok := input.Messages[0].Content.([]AliMediaContent)
	require.True(t, ok)
	require.Equal(t, []AliMediaContent{{Text: "一只猫"}}, content)
}

func TestOaiImage2AliImageRequest_NativeInputPassthrough(t *testing.T) {
	info := &relaycommon.RelayInfo{}
	req := dto.ImageRequest{
		Model:  "qwen-image-2.0-pro",
		Prompt: "ignored-when-input-present",
		Extra: map[string]json.RawMessage{
			"input": json.RawMessage(`{
				"messages":[{
					"role":"user",
					"content":[
						{"image":"https://xxx.png"},
						{"text":"小猫睡觉"}
					]
				}]
			}`),
			"parameters": json.RawMessage(`{
				"n":1,
				"negative_prompt":" ",
				"prompt_extend":true,
				"watermark":false,
				"size":"2048*2048"
			}`),
		},
	}

	got, err := oaiImage2AliImageRequest(info, req, true)
	require.NoError(t, err)
	require.Equal(t, 1, got.Parameters.N)
	require.Equal(t, " ", got.Parameters.NegativePrompt)
	require.NotNil(t, got.Parameters.PromptExtend)
	require.True(t, *got.Parameters.PromptExtend)
	require.Equal(t, "2048*2048", got.Parameters.Size)
	require.NotNil(t, got.Input)
}

func TestBuildAliSyncImageInput_MultipleImages(t *testing.T) {
	input := buildAliSyncImageInput("edit me", []string{"https://a.png", "https://b.png"})
	content, ok := input.Messages[0].Content.([]AliMediaContent)
	require.True(t, ok)
	require.Equal(t, []AliMediaContent{
		{Image: "https://a.png"},
		{Image: "https://b.png"},
		{Text: "edit me"},
	}, content)
}

func TestOaiImage2AliImageRequest_SyncMultiImageFusion(t *testing.T) {
	info := &relaycommon.RelayInfo{}
	img1 := "https://help-static-aliyun-doc.aliyuncs.com/file-manage-files/zh-CN/20260310/rdsgaa/image+%2815%29.png"
	img2 := "https://help-static-aliyun-doc.aliyuncs.com/file-manage-files/zh-CN/20260310/qokhtl/image+%2816%29.png"
	prompt := "使用图一的城市照片作为底图。请勿更改照片中的真实建筑、街道、车辆或人物。"
	imageJSON, err := json.Marshal([]string{img1, img2})
	require.NoError(t, err)

	req := dto.ImageRequest{
		Model:  "qwen-image-2.0-pro",
		Prompt: prompt,
		N:      common.GetPointer(uint(1)),
		Image:  imageJSON,
	}

	got, err := oaiImage2AliImageRequest(info, req, true)
	require.NoError(t, err)

	input, ok := got.Input.(AliImageInput)
	require.True(t, ok)
	content, ok := input.Messages[0].Content.([]AliMediaContent)
	require.True(t, ok)
	require.Equal(t, []AliMediaContent{
		{Image: img1},
		{Image: img2},
		{Text: prompt},
	}, content)
}
