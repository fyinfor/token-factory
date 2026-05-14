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
	if info != nil {
		info.TFOpenUpstreamRouteApplied = false
	}
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
	// TFOpen 上游渠道精准路由：
	// 新版：route_slug 格式（优先），将 UpstreamModelName 改写为 "{model}/{route_slug}"，
	// 上游的 ParseModelRouteIndex 解析此格式精准路由到对应渠道。
	// 旧版（兼容）：alias|channelNo 三段式路由，格式为 "legacy|{alias}|{channelNo}"，
	// 将 UpstreamModelName 改写为 "{alias}/{model}/{channelNo}"。
	// 当上游也是 TokenFactory 平台时，使用原始模型名（上游可识别的本地模型名）而非
	// model_mapping 映射后的名称（如 HuggingFace 格式），避免上游 distributor 误解析。
	if tfRoute := c.GetString(string(constant.ContextKeyTFOpenUpstreamChannelRoute)); tfRoute != "" {
		// 使用原始模型名（而非映射后的名称），因为上游 TF 平台理解本地原始模型名
		modelForUpstream := info.OriginModelName
		if isResponsesCompact && strings.HasSuffix(modelForUpstream, ratio_setting.CompactModelSuffix) {
			modelForUpstream = strings.TrimSuffix(modelForUpstream, ratio_setting.CompactModelSuffix)
		}

		if strings.HasPrefix(tfRoute, "legacy|") {
			// 旧版三段式路由兼容：legacy|alias|channelNo → alias/model/channelNo
			parts := strings.SplitN(tfRoute, "|", 3)
			if len(parts) == 3 {
				alias := parts[1]
				channelNo := parts[2]
				if alias != "" && channelNo != "" {
					info.UpstreamModelName = alias + "/" + modelForUpstream + "/" + channelNo
					info.IsModelMapped = false
					info.TFOpenUpstreamRouteApplied = true
				}
			}
		} else {
			// 新版二段式路由：route_slug → model/route_slug
			routeSlug := strings.TrimSpace(tfRoute)
			if routeSlug != "" {
				info.UpstreamModelName = modelForUpstream + "/" + routeSlug
				info.IsModelMapped = false
				info.TFOpenUpstreamRouteApplied = true
			}
		}
	}

	if request != nil {
		request.SetModelName(info.UpstreamModelName)
	}
	return nil
}
