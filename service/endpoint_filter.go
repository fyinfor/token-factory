package service

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
)

func ChannelSupportsRelayMode(channel *model.Channel, modelName string, relayMode int) bool {
	if channel == nil {
		return false
	}
	modelTags := ""
	if channel.Type == constant.ChannelTypeTokenFactoryOpen {
		modelTags = model.GetModelTagsByName(modelName)
	}
	endpointTypes := common.GetEndpointTypesByChannelTypeWithTags(channel.Type, modelName, modelTags)
	supports := func(targets ...constant.EndpointType) bool {
		for _, endpointType := range endpointTypes {
			for _, target := range targets {
				if endpointType == target {
					return true
				}
			}
		}
		return false
	}
	switch relayMode {
	case relayconstant.RelayModeVideoSubmit:
		return supports(
			constant.EndpointTypeOpenAIVideo,
			constant.EndpointTypeOpenAIVideoGW,
			constant.EndpointTypeTokenFactoryVideo,
			constant.EndpointTypeVideoGenerator,
			constant.EndpointTypeTencentCloudVODVideo,
			constant.EndpointTypeAliVideo,
			constant.EndpointTypeSeedanceVideo,
			constant.EndpointTypeMiniMaxH3Video,
		)
	case relayconstant.RelayModeImagesGenerations, relayconstant.RelayModeImagesEdits:
		return supports(constant.EndpointTypeImageGeneration, constant.EndpointTypeTencentCloudVODImage)
	default:
		return true
	}
}

func RelayModeChannelFilter(relayMode int, modelName string) func(*model.Channel) bool {
	switch relayMode {
	case relayconstant.RelayModeVideoSubmit,
		relayconstant.RelayModeImagesGenerations,
		relayconstant.RelayModeImagesEdits:
		return func(channel *model.Channel) bool {
			return ChannelSupportsRelayMode(channel, modelName, relayMode)
		}
	default:
		return nil
	}
}

func TryFilteredRouteChannel(candidates []*model.Channel, filter func(*model.Channel) bool) *model.Channel {
	for _, channel := range candidates {
		if channel != nil && (filter == nil || filter(channel)) {
			return channel
		}
	}
	return nil
}

func TryGroupRouteChannelWithFilter(group string, modelName string, filter func(*model.Channel) bool) (*model.Channel, bool) {
	ids := model.GetGroupEnabledChannelIDs(group, modelName)
	if len(ids) == 0 {
		return nil, false
	}
	channels := make([]*model.Channel, 0, len(ids))
	for _, id := range ids {
		channel, err := model.GetChannelById(id, true)
		if err != nil || channel == nil || channel.Status != common.ChannelStatusEnabled {
			continue
		}
		if !model.IsChannelEnabledForGroupModel(group, modelName, channel.Id) {
			continue
		}
		channels = append(channels, channel)
	}
	if channel := TryFilteredRouteChannel(channels, filter); channel != nil {
		return channel, true
	}
	return nil, false
}
