package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
)

func TestChannelSupportsRelayMode(t *testing.T) {
	tests := []struct {
		name      string
		channel   *model.Channel
		modelName string
		relayMode int
		want      bool
	}{
		{
			name: "OpenAI chat channel supports chat",
			channel: &model.Channel{
				Type: constant.ChannelTypeOpenAI,
			},
			modelName: "gpt-4",
			relayMode: relayconstant.RelayModeChatCompletions,
			want:      true,
		},
		{
			name: "OpenAI video channel supports video submit",
			channel: &model.Channel{
				Type: constant.ChannelTypeOpenAIVideo,
			},
			modelName: "sora-2",
			relayMode: relayconstant.RelayModeVideoSubmit,
			want:      true,
		},
		{
			name: "TokenFactoryOpen channel supports video submit when model has video tag",
			channel: &model.Channel{
				Type: constant.ChannelTypeTokenFactoryOpen,
			},
			modelName: "video-model",
			relayMode: relayconstant.RelayModeVideoSubmit,
			want:      false,
		},
		{
			name: "OpenAI chat channel does not support video submit",
			channel: &model.Channel{
				Type: constant.ChannelTypeOpenAI,
			},
			modelName: "gpt-4",
			relayMode: relayconstant.RelayModeVideoSubmit,
			want:      false,
		},
		{
			name: "OpenAI image channel supports image generations",
			channel: &model.Channel{
				Type: constant.ChannelTypeOpenAIImage,
			},
			modelName: "dall-e-3",
			relayMode: relayconstant.RelayModeImagesGenerations,
			want:      true,
		},
		{
			name: "Ali Qwen image channel supports image generations",
			channel: &model.Channel{
				Type: constant.ChannelTypeAliImage,
			},
			modelName: "qwen-image-2.0-pro",
			relayMode: relayconstant.RelayModeImagesGenerations,
			want:      true,
		},
		{
			name: "HiDream image channel supports image generations",
			channel: &model.Channel{
				Type: constant.ChannelTypeHiDreamImage,
			},
			modelName: "hidream-H4.5-image",
			relayMode: relayconstant.RelayModeImagesGenerations,
			want:      true,
		},
		{
			name: "nil channel returns false",
			channel:   nil,
			modelName: "any",
			relayMode: relayconstant.RelayModeVideoSubmit,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ChannelSupportsRelayMode(tt.channel, tt.modelName, tt.relayMode)
			if got != tt.want {
				t.Errorf("ChannelSupportsRelayMode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRelayModeChannelFilter(t *testing.T) {
	videoFilter := RelayModeChannelFilter(relayconstant.RelayModeVideoSubmit, "video-model")
	if videoFilter == nil {
		t.Fatal("RelayModeChannelFilter for video should not return nil")
	}

	chatFilter := RelayModeChannelFilter(relayconstant.RelayModeChatCompletions, "gpt-4")
	if chatFilter != nil {
		t.Error("RelayModeChannelFilter for chat should return nil (no filtering)")
	}

	imageFilter := RelayModeChannelFilter(relayconstant.RelayModeImagesGenerations, "dall-e-3")
	if imageFilter == nil {
		t.Fatal("RelayModeChannelFilter for image should not return nil")
	}
}

func TestTryFilteredRouteChannel(t *testing.T) {
	channels := []*model.Channel{
		{Type: constant.ChannelTypeOpenAI},
		{Type: constant.ChannelTypeOpenAIVideo},
		{Type: constant.ChannelTypeTokenFactoryOpen},
	}

	videoFilter := RelayModeChannelFilter(relayconstant.RelayModeVideoSubmit, "video-model")
	selected := TryFilteredRouteChannel(channels, videoFilter)
	if selected == nil {
		t.Error("TryFilteredRouteChannel should find a video-capable channel")
	}
	if selected.Type != constant.ChannelTypeOpenAIVideo && selected.Type != constant.ChannelTypeTokenFactoryOpen {
		t.Errorf("TryFilteredRouteChannel selected wrong channel type: %d", selected.Type)
	}

	chatFilter := RelayModeChannelFilter(relayconstant.RelayModeChatCompletions, "gpt-4")
	selected = TryFilteredRouteChannel(channels, chatFilter)
	if selected == nil {
		t.Error("TryFilteredRouteChannel with nil filter should return first channel")
	}
	if selected.Type != constant.ChannelTypeOpenAI {
		t.Errorf("TryFilteredRouteChannel with nil filter should return first channel, got type %d", selected.Type)
	}
}

func TestGetEndpointTypesByChannelTypeForVideo(t *testing.T) {
	videoChannelTypes := []int{
		constant.ChannelTypeOpenAIVideo,
		constant.ChannelTypeTokenFactoryOpen,
		constant.ChannelTypeVideoGenerator,
		constant.ChannelTypeTencentCloudVideo,
		constant.ChannelTypeAliVideo,
		constant.ChannelTypeDoubaoVideo,
		constant.ChannelTypeSeedance,
	}

	for _, channelType := range videoChannelTypes {
		var endpointTypes []constant.EndpointType
		if channelType == constant.ChannelTypeTokenFactoryOpen {
			endpointTypes = common.GetEndpointTypesByChannelTypeWithTags(channelType, "any-video-model", "视频")
		} else {
			endpointTypes = common.GetEndpointTypesByChannelType(channelType, "any-video-model")
		}
		hasVideoEndpoint := false
		for _, et := range endpointTypes {
			switch et {
			case constant.EndpointTypeOpenAIVideo,
				constant.EndpointTypeOpenAIVideoGW,
				constant.EndpointTypeTokenFactoryVideo,
				constant.EndpointTypeVideoGenerator,
				constant.EndpointTypeTencentCloudVODVideo,
				constant.EndpointTypeAliVideo,
				constant.EndpointTypeSeedanceVideo:
				hasVideoEndpoint = true
			}
		}
		if !hasVideoEndpoint {
			t.Errorf("ChannelType %d should have at least one video endpoint type, got %v", channelType, endpointTypes)
		}
	}
}
