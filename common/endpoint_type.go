package common

import "github.com/QuantumNous/new-api/constant"

// GetEndpointTypesByChannelType 获取渠道最优先端点类型（所有的渠道都支持 OpenAI 端点）。
// TokenFactoryOpen(60) 需结合模型标签判断视频/图片端点，无标签时请用 GetEndpointTypesByChannelTypeWithTags。
func GetEndpointTypesByChannelType(channelType int, modelName string) []constant.EndpointType {
	return GetEndpointTypesByChannelTypeWithTags(channelType, modelName, "")
}

// GetEndpointTypesByChannelTypeWithTags 在 GetEndpointTypesByChannelType 基础上传入 models.tags，
// 供 TokenFactoryOpen(60) 按「视频」「图片」标签附加 tokenfactory-video / image-generation。
func GetEndpointTypesByChannelTypeWithTags(channelType int, modelName string, modelTags string) []constant.EndpointType {
	var endpointTypes []constant.EndpointType
	switch channelType {
	case constant.ChannelTypeJina:
		endpointTypes = []constant.EndpointType{constant.EndpointTypeJinaRerank}
	//case constant.ChannelTypeMidjourney, constant.ChannelTypeMidjourneyPlus:
	//	endpointTypes = []constant.EndpointType{constant.EndpointTypeMidjourney}
	//case constant.ChannelTypeSunoAPI:
	//	endpointTypes = []constant.EndpointType{constant.EndpointTypeSuno}
	//case constant.ChannelTypeKling:
	//	endpointTypes = []constant.EndpointType{constant.EndpointTypeKling}
	//case constant.ChannelTypeJimeng:
	//	endpointTypes = []constant.EndpointType{constant.EndpointTypeJimeng}
	case constant.ChannelTypeAws:
		fallthrough
	case constant.ChannelTypeAnthropic:
		endpointTypes = []constant.EndpointType{constant.EndpointTypeAnthropic, constant.EndpointTypeOpenAI}
	case constant.ChannelTypeVertexAi:
		fallthrough
	case constant.ChannelTypeGemini:
		endpointTypes = []constant.EndpointType{constant.EndpointTypeGemini, constant.EndpointTypeOpenAI}
	case constant.ChannelTypeOpenRouter: // OpenRouter 只支持 OpenAI 端点
		endpointTypes = []constant.EndpointType{constant.EndpointTypeOpenAI}
	case constant.ChannelTypeXai:
		endpointTypes = []constant.EndpointType{constant.EndpointTypeOpenAI, constant.EndpointTypeOpenAIResponse}
	case constant.ChannelTypeSora:
		endpointTypes = []constant.EndpointType{constant.EndpointTypeOpenAIVideo}
	case constant.ChannelTypeOpenAIVideo:
		endpointTypes = []constant.EndpointType{constant.EndpointTypeOpenAIVideoGW}
	case constant.ChannelTypeTokenFactoryOpen:
		// 建站渠道默认仅声明 OpenAI 文本入口；带「视频」「图片」标签的模型再附加对应端点，
		// 避免所有模型带上 tokenfactory-video 导致首页误走视频计价卡片。
		endpointTypes = []constant.EndpointType{constant.EndpointTypeOpenAI}
		if ModelTagsIndicateVideoPricing(modelTags) {
			endpointTypes = append([]constant.EndpointType{constant.EndpointTypeTokenFactoryVideo}, endpointTypes...)
		}
		if ModelTagsIndicateImagePricing(modelTags) {
			endpointTypes = append([]constant.EndpointType{constant.EndpointTypeImageGeneration}, endpointTypes...)
		}
	case constant.ChannelTypeVideoGenerator:
		endpointTypes = []constant.EndpointType{constant.EndpointTypeVideoGenerator}
	case constant.ChannelTypeTencentCloudVideo:
		endpointTypes = []constant.EndpointType{constant.EndpointTypeTencentCloudVODVideo}
	case constant.ChannelTypeTencentCloudImage:
		endpointTypes = []constant.EndpointType{constant.EndpointTypeTencentCloudVODImage}
	case constant.ChannelTypeOpenAIImage, constant.ChannelTypeAliImage, constant.ChannelTypeHiDreamImage:
		endpointTypes = []constant.EndpointType{constant.EndpointTypeImageGeneration}
	case constant.ChannelTypeAliVideo:
		endpointTypes = []constant.EndpointType{constant.EndpointTypeAliVideo}
	case constant.ChannelTypeDoubaoVideo, constant.ChannelTypeSeedance:
		endpointTypes = []constant.EndpointType{constant.EndpointTypeSeedanceVideo}
	case constant.ChannelTypeMiniMaxH3Video:
		endpointTypes = []constant.EndpointType{constant.EndpointTypeMiniMaxH3Video}
	case constant.ChannelTypeAliASRSync:
		endpointTypes = []constant.EndpointType{constant.EndpointTypeAliASRSync}
	case constant.ChannelTypeAliASRAsync:
		endpointTypes = []constant.EndpointType{constant.EndpointTypeAliASRAsync}
	default:
		if IsOpenAIResponseOnlyModel(modelName) {
			endpointTypes = []constant.EndpointType{constant.EndpointTypeOpenAIResponse}
		} else {
			endpointTypes = []constant.EndpointType{constant.EndpointTypeOpenAI}
		}
	}
	if IsImageGenerationModel(modelName) {
		// add to first
		endpointTypes = append([]constant.EndpointType{constant.EndpointTypeImageGeneration}, endpointTypes...)
	}
	return endpointTypes
}
