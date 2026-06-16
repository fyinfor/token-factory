package common

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
)

func TestModelTagsContain(t *testing.T) {
	if !ModelTagsContain("文本,视频,热门", "视频") {
		t.Fatal("expected 视频 tag match")
	}
	if ModelTagsContain("文本,多模态", "视频") {
		t.Fatal("unexpected 视频 tag match")
	}
	if !ModelTagsContain("文本, 图片", "图片") {
		t.Fatal("expected trimmed 图片 tag match")
	}
}

func TestTokenFactoryOpenEndpointTypesByTag(t *testing.T) {
	textOnly := GetEndpointTypesByChannelTypeWithTags(
		constant.ChannelTypeTokenFactoryOpen,
		"glm-5.1",
		"文本,热门",
	)
	if len(textOnly) != 1 || textOnly[0] != constant.EndpointTypeOpenAI {
		t.Fatalf("text model should only have openai endpoint, got %v", textOnly)
	}

	videoTagged := GetEndpointTypesByChannelTypeWithTags(
		constant.ChannelTypeTokenFactoryOpen,
		"kling-v3",
		"视频",
	)
	if len(videoTagged) != 2 ||
		videoTagged[0] != constant.EndpointTypeTokenFactoryVideo ||
		videoTagged[1] != constant.EndpointTypeOpenAI {
		t.Fatalf("video-tagged model endpoints = %v", videoTagged)
	}

	imageTagged := GetEndpointTypesByChannelTypeWithTags(
		constant.ChannelTypeTokenFactoryOpen,
		"Qwen-Image",
		"图片",
	)
	if len(imageTagged) != 2 ||
		imageTagged[0] != constant.EndpointTypeImageGeneration ||
		imageTagged[1] != constant.EndpointTypeOpenAI {
		t.Fatalf("image-tagged model endpoints = %v", imageTagged)
	}
}
