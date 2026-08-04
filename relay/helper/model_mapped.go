package helper

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/service"
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

	// TokenFactoryOpen 渠道同样先应用本地 model_mapping（本站别名 → 上游真实模型名），
	// 再在下方 tfRoute 逻辑中用映射后的名称拼接路由；OriginModelName 保持客户侧原始名用于计费/日志。
	channelType := common.GetContextKeyInt(c, constant.ContextKeyChannelType)
	isTFOpenUpstream := channelType == constant.ChannelTypeTokenFactoryOpen

	// map model name
	modelMapping := c.GetString("model_mapping")
	if modelMapping != "" && modelMapping != "{}" {
		modelMap := make(map[string]string)
		err := json.Unmarshal([]byte(modelMapping), &modelMap)
		if err != nil {
			return fmt.Errorf("unmarshal_model_mapping_failed")
		}

		// 若模型名形如「Seedance2.0/route_slug」：优先用已解析的路由得到基础名；
		// 若路由未命中（子站与上游库不一致、slug 在上游不存在等），仍用「最后一段为合法 route_slug」时的基础名走 model_mapping，
		// 避免把整串当作上游真实 model_id 送给外部网关（会导致 Invalid input params）。
		currentModel := mappingModelName
		if idx, matched, _ := service.ParseModelRouteIndex(mappingModelName); matched && idx != nil {
			currentModel = idx.ModelName
		} else if strings.Contains(mappingModelName, "/") {
			lastSlash := strings.LastIndex(mappingModelName, "/")
			if lastSlash > 0 && lastSlash < len(mappingModelName)-1 {
				potentialSlug := strings.TrimSpace(mappingModelName[lastSlash+1:])
				potentialBase := strings.TrimSpace(mappingModelName[:lastSlash])
				if potentialBase != "" && model.IsValidRouteSlug(potentialSlug) {
					currentModel = potentialBase
				}
			}
		}

		// 支持链式模型重定向，最终使用链尾的模型
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
	// TFOpen 上游渠道精准路由（仅 ChannelTypeTokenFactoryOpen）：
	// 新版：route_slug 格式（优先），将 UpstreamModelName 改写为 "{model}/{route_slug}"，
	// 上游的 ParseModelRouteIndex 解析此格式精准路由到对应渠道。
	// 旧版（兼容）：alias|channelNo 三段式路由，格式为 "legacy|{alias}|{channelNo}"，
	// 将 UpstreamModelName 改写为 "{alias}/{model}/{channelNo}"。
	// 使用映射后的上游模型名（未映射时等同 OriginModelName），避免本站别名穿透到上游。
	// 非 60 类型渠道即使上下文误带了路由提示也不拼接后缀（真实 OpenAI 网关不识别）。
	if isTFOpenUpstream {
		if tfRoute := c.GetString(string(constant.ContextKeyTFOpenUpstreamChannelRoute)); tfRoute != "" {
			modelForUpstream := info.UpstreamModelName
			if modelForUpstream == "" {
				modelForUpstream = info.OriginModelName
			}
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
	}

	// 未命中 model_mapping、且未走 TFOpen 精准路由时：请求里仍可能是「Seedance2.0/route_slug」
	//（例如子站 other_info 里的 slug 在上游库不存在，Distribute 未能改写 body）。
	// 此时至少剥掉「最后一段为合法 route_slug」的后缀，避免把整串当作外部视频网关的 model_id
	//（Hidream/MaaS 会返回 Invalid input params）。
	if info != nil && !isTFOpenUpstream && !info.TFOpenUpstreamRouteApplied && !info.IsModelMapped {
		um := strings.TrimSpace(info.UpstreamModelName)
		if um == "" {
			um = strings.TrimSpace(mappingModelName)
		}
		if um != "" && strings.Contains(um, "/") {
			if idx, matched, _ := service.ParseModelRouteIndex(um); matched && idx != nil {
				info.UpstreamModelName = idx.ModelName
			} else {
				lastSlash := strings.LastIndex(um, "/")
				if lastSlash > 0 && lastSlash < len(um)-1 {
					potentialSlug := strings.TrimSpace(um[lastSlash+1:])
					potentialBase := strings.TrimSpace(um[:lastSlash])
					if potentialBase != "" && model.IsValidRouteSlug(potentialSlug) {
						info.UpstreamModelName = potentialBase
					}
				}
			}
		}
	}

	if request != nil {
		request.SetModelName(info.UpstreamModelName)
	}
	return nil
}

// ApplyChannelModelMapping 按渠道 model_mapping 做链式重定向，委托 model 包统一实现。
func ApplyChannelModelMapping(mappingJSON, startModel string) string {
	return model.ApplyChannelModelMapping(mappingJSON, startModel)
}
