package helper

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
)

func ModelMappedHelper(c *gin.Context, info *relaycommon.RelayInfo, request dto.Request) error {
	if info.ChannelMeta == nil {
		info.ChannelMeta = &relaycommon.ChannelMeta{}
	}

	isResponsesCompact := info.RelayMode == relayconstant.RelayModeResponsesCompact
	originModelName := info.OriginModelName
	mappingModelName := originModelName
	if isResponsesCompact && strings.HasSuffix(originModelName, ratio_setting.CompactModelSuffix) {
		mappingModelName = strings.TrimSuffix(originModelName, ratio_setting.CompactModelSuffix)
	}

	// TokenFactoryOpen 渠道指向上游 TokenFactory 平台，上游 distributor 会将含 "/" 的模型名
	// 误解析为路由格式（{model}/{route_slug} 或 {alias}/{model}/{channel_no}）。
	// 因此当上游是 TF 平台时，跳过 model_mapping，保留本地原始模型名。
	// TFOpen 同步渠道（source=tokenfactory_open）会在下方 tfRoute 逻辑中拼接三段式路由，
	// 同样使用原始模型名。
	channelType := common.GetContextKeyInt(c, constant.ContextKeyChannelType)
	isTFOpenUpstream := channelType == constant.ChannelTypeTokenFactoryOpen

	// map model name
	modelMapping := c.GetString("model_mapping")
	if modelMapping != "" && modelMapping != "{}" && !isTFOpenUpstream {
		modelMap := make(map[string]string)
		err := json.Unmarshal([]byte(modelMapping), &modelMap)
		if err != nil {
			return fmt.Errorf("unmarshal_model_mapping_failed")
		}

		// 支持链式模型重定向，最终使用链尾的模型
		currentModel := mappingModelName
		visitedModels := map[string]bool{
			currentModel: true,
		}
		for {
			if mappedModel, exists := modelMap[currentModel]; exists && mappedModel != "" {
				// 模型重定向循环检测，避免无限循环
				if visitedModels[mappedModel] {
					if mappedModel == currentModel {
						if currentModel == info.OriginModelName {
							info.IsModelMapped = false
							return nil
						} else {
							info.IsModelMapped = true
							break
						}
					}
					return errors.New("model_mapping_contains_cycle")
				}
				visitedModels[mappedModel] = true
				currentModel = mappedModel
				info.IsModelMapped = true
			} else {
				break
			}
		}
		if info.IsModelMapped {
			info.UpstreamModelName = currentModel
		}
	}

	if isResponsesCompact {
		finalUpstreamModelName := mappingModelName
		if info.IsModelMapped && info.UpstreamModelName != "" {
			finalUpstreamModelName = info.UpstreamModelName
		}
		info.UpstreamModelName = finalUpstreamModelName
		info.OriginModelName = ratio_setting.WithCompactModelSuffix(finalUpstreamModelName)
	}
	// TFOpen 上游渠道精准路由：若本地渠道来自 TokenFactoryOpen 同步且存在有效的
	// upstream_supplier_alias 与 upstream_channel_no，将 UpstreamModelName 改写为
	// "{alias}/{model}/{channel_no}" 格式，上游平台的 Distribute 中间件会将其解析为
	// ParseForcedChannelModelName 指定渠道路由，从而保证子站流量与上游同一渠道对齐。
	// 当上游也是 TokenFactory 平台时，应使用原始模型名（上游可识别的本地模型名）而非
	// model_mapping 映射后的名称（如 HuggingFace 格式），避免上游 distributor 误解析含 "/" 的模型名。
	if tfRoute := c.GetString(string(constant.ContextKeyTFOpenUpstreamChannelRoute)); tfRoute != "" {
		if idx := strings.IndexByte(tfRoute, '|'); idx > 0 {
			alias := tfRoute[:idx]
			channelNo := tfRoute[idx+1:]
			if alias != "" && channelNo != "" {
				// 使用原始模型名（而非映射后的名称）拼接路由，
				// 因为上游 TF 平台理解本地原始模型名，model_mapping 仅为非 TF 上游设计。
				modelForUpstream := info.OriginModelName
				if isResponsesCompact && strings.HasSuffix(modelForUpstream, ratio_setting.CompactModelSuffix) {
					modelForUpstream = strings.TrimSuffix(modelForUpstream, ratio_setting.CompactModelSuffix)
				}
				info.UpstreamModelName = alias + "/" + modelForUpstream + "/" + channelNo
				info.IsModelMapped = false
			}
		}
	}

	if request != nil {
		request.SetModelName(info.UpstreamModelName)
	}
	return nil
}
