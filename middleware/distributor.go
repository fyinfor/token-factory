package middleware

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

type ModelRequest struct {
	Model string `json:"model"`
	Group string `json:"group,omitempty"`
	// SpecificChannelID 指定 playground 请求直连某个渠道（channels.id）。
	// nil 表示按默认逻辑随机/智能路由。
	SpecificChannelID *int `json:"specific_channel_id,omitempty"`
}

func Distribute() func(c *gin.Context) {
	return func(c *gin.Context) {
		var channel *model.Channel
		// 压测/归因：X-TF-No-Failover=true 时禁止渠道级 failover（见 shouldRetry / getChannel）。
		service.ApplyNoFailoverHeader(c)
		modelRequest, shouldSelectChannel, err := getModelRequest(c)
		if err != nil {
			abortWithOpenAiMessage(c, http.StatusBadRequest, i18n.T(c, i18n.MsgDistributorInvalidRequest, map[string]any{"Error": err.Error()}))
			return
		}
		channelId, ok := common.GetContextKey(c, constant.ContextKeyTokenSpecificChannelId)

		// 解析特殊模型名形式，按优先级识别：
		//   1) {supplier_alias}/{model}/{channel_no} —— 旧格式：指定渠道直连（向后兼容）；
		//   2) {model}/{route_slug}                  —— 全局渠道路由后缀（channels.route_slug，整渠道唯一）；
		//   3) {supplier_alias}/{model}              —— 旧格式：指定供应商下任意渠道。
		// 命中后把真实模型名回写到 modelRequest.Model 与请求体，后续路由/日志使用真实模型名。
		if shouldSelectChannel && modelRequest != nil && strings.Contains(modelRequest.Model, "/") {
			route, matched, routeErr := service.ParseForcedChannelModelName(modelRequest.Model)
			if matched && routeErr != nil {
				abortWithOpenAiMessage(c, http.StatusBadRequest, i18n.T(c, i18n.MsgDistributorInvalidRequest, map[string]any{"Error": routeErr.Error()}))
				return
			}
			if route != nil {
				originalModelKey := modelRequest.Model
				if err := service.ApplyForcedChannelOnRequestBody(c, route, originalModelKey); err != nil {
					abortWithOpenAiMessage(c, http.StatusBadRequest, i18n.T(c, i18n.MsgDistributorInvalidRequest, map[string]any{"Error": err.Error()}))
					return
				}
				modelRequest.Model = route.ModelName
			} else {
				// 尝试 {model}/{route_slug}（全局渠道路由后缀）。
				indexRoute, _, _ := service.ParseModelRouteIndex(modelRequest.Model)
				if indexRoute != nil {
					originalModelKey := modelRequest.Model
					userID := c.GetInt("id")
					// routeOn=false 覆盖：全局关闭路由 / 用户选关闭 / 该模型归类关闭。
					routeOn := service.EffectiveRouteStrategyEnabled(userID, indexRoute.ModelName)
					playgroundPin := isPlaygroundChannelPinPath(c)
					if indexRoute.ChannelID <= 0 {
						ch := model.LookupChannelByRouteSlug(indexRoute.RouteSlug)
						// 智能路由开启（且非操练场）：后缀不可用/不存在 → 剥后缀走同模型智能路由，失败可切换。
						// 路由关闭 / 操练场：后缀必须可用，否则直接拒绝。
						if routeOn && !playgroundPin {
							reason := "not_found"
							if ch != nil && ch.Status != common.ChannelStatusEnabled {
								reason = fmt.Sprintf("disabled(id=%d)", ch.Id)
							} else if ch != nil {
								reason = fmt.Sprintf("missing_model(id=%d)", ch.Id)
							}
							logger.LogInfo(c, fmt.Sprintf("route_slug=%s unavailable (%s) for model=%s, fallback to smart route", indexRoute.RouteSlug, reason, indexRoute.ModelName))
							if err := service.ApplyModelRouteOnRequestBody(c, indexRoute, originalModelKey); err != nil {
								abortWithOpenAiMessage(c, http.StatusBadRequest, i18n.T(c, i18n.MsgDistributorInvalidRequest, map[string]any{"Error": err.Error()}))
								return
							}
							modelRequest.Model = indexRoute.ModelName
						} else {
							logger.LogInfo(c, fmt.Sprintf("route_slug=%s unavailable for model=%s (channel_found=%v, route_on=%v)", indexRoute.RouteSlug, indexRoute.ModelName, ch != nil, routeOn))
							if ch != nil && ch.Status != common.ChannelStatusEnabled {
								abortWithOpenAiMessage(c, http.StatusForbidden, i18n.T(c, i18n.MsgDistributorChannelDisabled))
								return
							}
							abortWithOpenAiMessage(c, http.StatusBadRequest, i18n.T(c, i18n.MsgDistributorInvalidChannelId), types.ErrorCodeInvalidRequest)
							return
						}
					} else {
						if err := service.ApplyModelRouteOnRequestBody(c, indexRoute, originalModelKey); err != nil {
							abortWithOpenAiMessage(c, http.StatusBadRequest, i18n.T(c, i18n.MsgDistributorInvalidRequest, map[string]any{"Error": err.Error()}))
							return
						}
						modelRequest.Model = indexRoute.ModelName
						if !routeOn || playgroundPin {
							// 路由关闭 / 归类关闭 / 操练场：硬指定，失败不切换。
							common.SetContextKey(c, constant.ContextKeyTokenSpecificChannelId, strconv.Itoa(indexRoute.ChannelID))
						}
						// 智能路由开启：仅写 PreferredChannelID，首跳优先该渠道，失败后有序切换。
					}
				} else {
					// 未命中以上两种格式时再尝试两段形式（{alias}/{model}）。
					supplierRoute, supplierMatched, supplierErr := service.ParseForcedSupplierModelName(modelRequest.Model)
					if supplierMatched && supplierErr != nil {
						abortWithOpenAiMessage(c, http.StatusBadRequest, i18n.T(c, i18n.MsgDistributorInvalidRequest, map[string]any{"Error": supplierErr.Error()}))
						return
					}
					if supplierRoute != nil {
						originalModelKey := modelRequest.Model
						if err := service.ApplyForcedSupplierOnRequestBody(c, supplierRoute, originalModelKey); err != nil {
							abortWithOpenAiMessage(c, http.StatusBadRequest, i18n.T(c, i18n.MsgDistributorInvalidRequest, map[string]any{"Error": err.Error()}))
							return
						}
						modelRequest.Model = supplierRoute.ModelName
					}
				}
			}
		}
		if ok {
			id, err := strconv.Atoi(channelId.(string))
			if err != nil {
				abortWithOpenAiMessage(c, http.StatusBadRequest, i18n.T(c, i18n.MsgDistributorInvalidChannelId))
				return
			}
			channel, err = model.GetChannelById(id, true)
			if err != nil {
				abortWithOpenAiMessage(c, http.StatusBadRequest, i18n.T(c, i18n.MsgDistributorInvalidChannelId))
				return
			}
			if channel.Status != common.ChannelStatusEnabled {
				abortWithOpenAiMessage(c, http.StatusForbidden, i18n.T(c, i18n.MsgDistributorChannelDisabled))
				return
			}
			// playground 已指定本地渠道时，支持 "{model}/{n}" 语义：
			// 若该渠道来自 TokenFactoryOpen 同步，则将 n 解释为上游 channel_no（c<n>）强制路由。
			if strings.HasPrefix(c.Request.URL.Path, "/pg/chat/completions") {
				if _, hasForced := common.GetContextKey(c, constant.ContextKeyForcedChannelID); !hasForced {
					if _, hasPreferred := common.GetContextKey(c, constant.ContextKeyPreferredChannelID); !hasPreferred {
						otherInfo := channel.GetOtherInfo()
						if source, _ := otherInfo["source"].(string); source == "tokenfactory_open" {
							if parsedModel, upstreamChannelNo, ok := parsePlaygroundTFOpenUpstreamRoute(modelRequest.Model); ok {
								modelRequest.Model = parsedModel
								common.SetContextKey(c, constant.ContextKeyTFOpenUpstreamChannelNoOverride, upstreamChannelNo)
								_ = rewriteRequestModelField(c, parsedModel)
							}
						}
					}
				}
			}
		} else {
			// Select a channel for the user
			// check token model mapping
			// GET 查询/remix 等无请求体模型名：此时不应拦截，等拿到任务模型后再复用同一套白名单。
			if !shouldDeferTokenModelLimitCheck(shouldSelectChannel, modelRequest.Model) {
				modelLimitEnable := common.GetContextKeyBool(c, constant.ContextKeyTokenModelLimitEnabled)
				if modelLimitEnable {
					s, ok := common.GetContextKey(c, constant.ContextKeyTokenModelLimit)
					if !ok {
						// token model limit is empty, all models are not allowed
						abortWithOpenAiMessage(c, http.StatusForbidden, i18n.T(c, i18n.MsgDistributorTokenNoModelAccess))
						return
					}
					tokenModelLimit, ok := s.(map[string]bool)
					if !ok || len(tokenModelLimit) == 0 {
						abortWithOpenAiMessage(c, http.StatusForbidden, i18n.T(c, i18n.MsgDistributorTokenNoModelAccess))
						return
					}
					// 修复点：使用与 GetModelLimitsMap 一致的匹配逻辑，兼容 gizmo/thinking-* 等规范化模型名
					if !model.ModelLimitMapAllows(tokenModelLimit, modelRequest.Model) {
						abortWithOpenAiMessage(c, http.StatusForbidden, i18n.T(c, i18n.MsgDistributorTokenModelForbidden, map[string]any{"Model": modelRequest.Model}))
						return
					}
				}
			}

			if shouldSelectChannel {
				if modelRequest.Model == "" {
					abortWithOpenAiMessage(c, http.StatusBadRequest, i18n.T(c, i18n.MsgDistributorModelNameRequired))
					return
				}

				if allowed, err := model.UserCanAccessAnyModelName(c.GetInt("id"), modelRequest.Model); err != nil {
					abortWithOpenAiMessage(c, http.StatusInternalServerError, err.Error())
					return
				} else if !allowed {
					abortWithOpenAiMessage(c, http.StatusForbidden, fmt.Sprintf("无权访问模型 %s", modelRequest.Model), types.ErrorCodeAccessDenied)
					return
				}

				// 命中「指定渠道直连」（{alias}/{model}/cN）：跳过所有自动路由，失败不切换渠道。
				if rawForced, hasForced := common.GetContextKey(c, constant.ContextKeyForcedChannelID); hasForced {
					if forcedID, fok := rawForced.(int); fok && forcedID > 0 {
						forcedChannel, ferr := model.CacheGetChannel(forcedID)
						if ferr != nil || forcedChannel == nil {
							abortWithOpenAiMessage(c, http.StatusBadRequest, i18n.T(c, i18n.MsgDistributorInvalidChannelId))
							return
						}
						if forcedChannel.Status != common.ChannelStatusEnabled {
							abortWithOpenAiMessage(c, http.StatusForbidden, i18n.T(c, i18n.MsgDistributorChannelDisabled))
							return
						}
						// 用户指定价：硬指定渠道超出价格上限时直接拒绝（硬指定语义不切换渠道）。
						if !service.ChannelWithinUserPriceCap(c.GetInt("id"), modelRequest.Model, forcedChannel) {
							abortWithOpenAiMessage(c, http.StatusForbidden, fmt.Sprintf("指定渠道价格超出您对模型 %s 的指定价上限，禁止调用", modelRequest.Model))
							return
						}
						common.SetContextKey(c, constant.ContextKeyTokenSpecificChannelId, strconv.Itoa(forcedID))
						channel = forcedChannel
						common.SetContextKey(c, constant.ContextKeyRequestStartTime, time.Now())
						logSelectedUpstream(c, channel, modelRequest.Model)
						SetupContextForSelectedChannel(c, channel, modelRequest.Model)
						proceedRelayWithChannel(c, channel, modelRequest.Model)
						return
					}
				}

				service.IngestChatCompletionRoutingHints(c, modelRequest.Model)
				var selectGroup string
				usingGroup := common.GetContextKeyString(c, constant.ContextKeyUsingGroup)
				userGroup := common.GetContextKeyString(c, constant.ContextKeyUserGroup)
				relayMode := c.GetInt("relay_mode")
				endpointChannelFilter := service.RelayModeChannelFilter(relayMode, modelRequest.Model)
				// 视频创建：保底候选必须限定在视频能力渠道；图片首期仍用 endpoint first-match。
				var videoSubmitFilter func(*model.Channel) bool
				if relayMode == relayconstant.RelayModeVideoSubmit {
					videoSubmitFilter = endpointChannelFilter
				}

				// 命中「route_slug 偏好渠道」（{model}/{route_slug}）：
				// - 智能路由开启：优先该渠道，失败按同模型有序候选切换；
				// - 路由关闭 / 操练场：硬指定，失败不切换。
				if rawPref, hasPref := common.GetContextKey(c, constant.ContextKeyPreferredChannelID); hasPref {
					if preferredID, pok := rawPref.(int); pok && preferredID > 0 {
						preferredChannel, perr := model.CacheGetChannel(preferredID)
						preferOK := perr == nil && preferredChannel != nil && preferredChannel.Status == common.ChannelStatusEnabled
						routeOn := service.EffectiveRouteStrategyEnabled(c.GetInt("id"), modelRequest.Model)
						hardPin := !routeOn || isPlaygroundChannelPinPath(c) || service.ContextNoFailover(c)
						if _, hasSpecific := common.GetContextKey(c, constant.ContextKeyTokenSpecificChannelId); hasSpecific {
							hardPin = true
						}

						if preferOK && !service.ChannelWithinUserPriceCap(c.GetInt("id"), modelRequest.Model, preferredChannel) {
							if hardPin {
								abortWithOpenAiMessage(c, http.StatusForbidden, fmt.Sprintf("指定渠道价格超出您对模型 %s 的指定价上限，禁止调用", modelRequest.Model))
								return
							}
							logger.LogInfo(c, fmt.Sprintf("route_slug preferred channel (id=%d) exceeds user price cap, fallback to same-model smart route for model=%s", preferredID, modelRequest.Model))
							preferOK = false
						}
						if preferOK && videoSubmitFilter != nil && !videoSubmitFilter(preferredChannel) {
							if hardPin {
								abortWithOpenAiMessage(c, http.StatusBadRequest, i18n.T(c, i18n.MsgDistributorInvalidChannelId), types.ErrorCodeInvalidRequest)
								return
							}
							logger.LogInfo(c, fmt.Sprintf("route_slug preferred channel (id=%d) lacks video endpoint, fallback to same-model video smart route for model=%s", preferredID, modelRequest.Model))
							preferOK = false
						}
						if !preferOK {
							if hardPin {
								abortWithOpenAiMessage(c, http.StatusForbidden, i18n.T(c, i18n.MsgDistributorChannelDisabled))
								return
							}
							logger.LogInfo(c, fmt.Sprintf("route_slug preferred channel unavailable (id=%d), fallback to same-model smart route for model=%s", preferredID, modelRequest.Model))
							common.SetContextKey(c, constant.ContextKeyPreferredChannelID, 0)
						} else {
							var ordered []int
							if hardPin {
								ordered = []int{preferredID}
								common.SetContextKey(c, constant.ContextKeyTokenSpecificChannelId, strconv.Itoa(preferredID))
							} else if videoSubmitFilter != nil {
								ordered = service.BuildPreferredChannelFailoverOrderWithFilter(c, modelRequest.Model, usingGroup, preferredID, videoSubmitFilter)
								if len(ordered) == 0 {
									ordered = []int{preferredID}
								}
							} else {
								ordered = service.BuildPreferredChannelFailoverOrder(c, modelRequest.Model, usingGroup, preferredID)
								if len(ordered) == 0 {
									ordered = []int{preferredID}
								}
							}
							common.SetContextKey(c, constant.ContextKeySmartRouteChannelOrder, ordered)
							channel = preferredChannel
							common.SetContextKey(c, constant.ContextKeyRequestStartTime, time.Now())
							logSelectedUpstream(c, channel, modelRequest.Model, "route_slug_preferred")
							SetupContextForSelectedChannel(c, channel, modelRequest.Model)
							proceedRelayWithChannel(c, channel, modelRequest.Model)
							return
						}
					}
				}
				// check path is /pg/chat/completions
				if strings.HasPrefix(c.Request.URL.Path, "/pg/chat/completions") {
					playgroundRequest := &dto.PlayGroundRequest{}
					err = common.UnmarshalBodyReusable(c, playgroundRequest)
					if err != nil {
						abortWithOpenAiMessage(c, http.StatusBadRequest, i18n.T(c, i18n.MsgDistributorInvalidPlayground, map[string]any{"Error": err.Error()}))
						return
					}
					if playgroundRequest.Group != "" {
						if !service.GroupInUserUsableGroups(usingGroup, playgroundRequest.Group) && playgroundRequest.Group != usingGroup {
							abortWithOpenAiMessage(c, http.StatusForbidden, i18n.T(c, i18n.MsgDistributorGroupAccessDenied))
							return
						}
						usingGroup = playgroundRequest.Group
						common.SetContextKey(c, constant.ContextKeyUsingGroup, usingGroup)
					}
				}

				// 统一 router-engine 选路在 forced supplier / 亲和渠道之后执行（见下方 channel == nil 分支）。

				// 命中「指定供应商 + 任意渠道」：跳过亲和选择，直接在供应商内按 SmartRouter / 优先级挑选。
				// 若候选池为空，直接报"无可用渠道"，不再回落到跨供应商的全局池，保持用户显式意图。
				if forcedSupplierID, hasForcedSupplier := service.ForcedSupplierFromContext(c); hasForcedSupplier {
					providerJSON := common.GetContextKeyString(c, constant.ContextKeyOpenRouterProviderJSON)
					service.IngestChatCompletionRoutingHints(c, modelRequest.Model)
					ch, sg, routeOK := service.TrySupplierEngineRoute(c, usingGroup, userGroup, modelRequest.Model, providerJSON, forcedSupplierID, videoSubmitFilter)
					if !routeOK || ch == nil {
						abortWithOpenAiMessage(c, http.StatusServiceUnavailable, i18n.T(c, i18n.MsgDistributorNoAvailableChannel, map[string]any{"Group": usingGroup, "Model": modelRequest.Model}), types.ErrorCodeModelNotFound)
						return
					}
					channel = ch
					selectGroup = sg
					common.SetContextKey(c, constant.ContextKeyRequestStartTime, time.Now())
					logSelectedUpstream(c, channel, modelRequest.Model)
					SetupContextForSelectedChannel(c, channel, modelRequest.Model)
					proceedRelayWithChannel(c, channel, modelRequest.Model)
					_ = selectGroup
					return
				}

				if preferredChannelID, found := service.GetPreferredChannelByAffinity(c, modelRequest.Model, usingGroup); found {
					preferred, err := model.CacheGetChannel(preferredChannelID)
					if err == nil && preferred != nil {
						if preferred.Status != common.ChannelStatusEnabled {
							if service.ShouldSkipRetryAfterChannelAffinityFailure(c) {
								abortWithOpenAiMessage(c, http.StatusForbidden, i18n.T(c, i18n.MsgDistributorChannelDisabled))
								return
							}
						} else if endpointChannelFilter != nil && !endpointChannelFilter(preferred) {
						} else if !service.ChannelWithinUserPriceCap(c.GetInt("id"), modelRequest.Model, preferred) {
							// 用户指定价：亲和渠道超出价格上限时跳过，走正常选路。
						} else if usingGroup == "auto" {
							autoGroups := service.GetUserAutoGroup(userGroup)
							for _, g := range autoGroups {
								if model.IsChannelEnabledForGroupModel(g, modelRequest.Model, preferred.Id) {
									selectGroup = g
									common.SetContextKey(c, constant.ContextKeyAutoGroup, g)
									channel = preferred
									service.MarkChannelAffinityUsed(c, g, preferred.Id)
									break
								}
							}
						} else if model.IsChannelEnabledForGroupModel(usingGroup, modelRequest.Model, preferred.Id) {
							channel = preferred
							selectGroup = usingGroup
							service.MarkChannelAffinityUsed(c, usingGroup, preferred.Id)
						}
					}
					// 视频创建：亲和命中后仍补齐同模型视频有序候选，创建失败可保底切换。
					if channel != nil && videoSubmitFilter != nil && !service.ContextNoFailover(c) {
						ordered := service.BuildPreferredChannelFailoverOrderWithFilter(c, modelRequest.Model, usingGroup, channel.Id, videoSubmitFilter)
						if len(ordered) > 0 {
							common.SetContextKey(c, constant.ContextKeySmartRouteChannelOrder, ordered)
						}
					}
				}

				if channel == nil {
					userGroup := common.GetContextKeyString(c, constant.ContextKeyUserGroup)
					providerJSON := common.GetContextKeyString(c, constant.ContextKeyOpenRouterProviderJSON)
					var routeFilter func(*model.Channel) bool
					if videoSubmitFilter != nil {
						routeFilter = videoSubmitFilter
					} else if endpointChannelFilter != nil {
						routeFilter = endpointChannelFilter
					}
					if ch, sg, routeOK := service.TryEngineRoute(c, usingGroup, userGroup, modelRequest.Model, providerJSON, routeFilter); routeOK {
						channel = ch
						selectGroup = sg
					}
					if channel == nil {
						showGroup := usingGroup
						if usingGroup == "auto" && selectGroup != "" {
							showGroup = fmt.Sprintf("auto(%s)", selectGroup)
						}
						abortWithOpenAiMessage(c, http.StatusServiceUnavailable, i18n.T(c, i18n.MsgDistributorNoAvailableChannel, map[string]any{"Group": showGroup, "Model": modelRequest.Model}), types.ErrorCodeModelNotFound)
						return
					}
				}
			}
		}
		common.SetContextKey(c, constant.ContextKeyRequestStartTime, time.Now())
		logSelectedUpstream(c, channel, modelRequest.Model)
		SetupContextForSelectedChannel(c, channel, modelRequest.Model)
		proceedRelayWithChannel(c, channel, modelRequest.Model)
	}
}

func isPlaygroundChannelPinPath(c *gin.Context) bool {
	if c == nil || c.Request == nil {
		return false
	}
	path := c.Request.URL.Path
	return strings.Contains(path, "/playground/") ||
		strings.Contains(path, "/api/playground/") ||
		strings.HasPrefix(path, "/pg/")
}

func parsePlaygroundTFOpenUpstreamRoute(rawModel string) (string, string, bool) {
	modelName := strings.TrimSpace(rawModel)
	if modelName == "" || !strings.Contains(modelName, "/") {
		return "", "", false
	}
	lastSlash := strings.LastIndex(modelName, "/")
	if lastSlash <= 0 || lastSlash >= len(modelName)-1 {
		return "", "", false
	}
	baseModel := strings.TrimSpace(modelName[:lastSlash])
	suffix := strings.TrimSpace(modelName[lastSlash+1:])
	if baseModel == "" || !isAllDigits(suffix) {
		return "", "", false
	}
	return baseModel, "c" + suffix, true
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func rewriteRequestModelField(c *gin.Context, modelName string) error {
	contentType := c.Request.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		return nil
	}

	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return err
	}
	body, err := storage.Bytes()
	if err != nil {
		return err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return nil
	}

	var obj map[string]json.RawMessage
	if err := common.Unmarshal(body, &obj); err != nil {
		return nil
	}
	if _, ok := obj["model"]; !ok {
		return nil
	}
	newModel, err := json.Marshal(modelName)
	if err != nil {
		return err
	}
	obj["model"] = newModel
	newBody, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	return common.ReplaceRequestBody(c, newBody)
}

func logSelectedUpstream(c *gin.Context, channel *model.Channel, modelName string, via ...string) {
	if c == nil || channel == nil {
		return
	}
	upstreamName := channel.Name
	upstreamBaseURL := channel.GetBaseURL()
	prefix := "upstream selected"
	if len(via) > 0 && via[0] != "" {
		prefix = via[0] + " selected"
	}
	msg := fmt.Sprintf("%s: channel=%s(id=%d) base_url=%s model=%s", prefix, upstreamName, channel.Id, upstreamBaseURL, modelName)
	logger.LogInfo(c, msg)
}

// getModelFromRequest 从请求中读取模型信息
// 根据 Content-Type 自动处理：
// - application/json
// - application/x-www-form-urlencoded
// - multipart/form-data
func getModelFromRequest(c *gin.Context) (*ModelRequest, error) {
	var modelRequest ModelRequest
	err := common.UnmarshalBodyReusable(c, &modelRequest)
	if err != nil {
		return nil, errors.New(i18n.T(c, i18n.MsgDistributorInvalidRequest, map[string]any{"Error": err.Error()}))
	}
	return &modelRequest, nil
}

// shouldDeferTokenModelLimitCheck 查询类接口 GET 无请求体，模型名此时为空。
// 若在分发阶段用空模型做白名单校验，会误报 "This token has no access to model"。
func shouldDeferTokenModelLimitCheck(shouldSelectChannel bool, modelName string) bool {
	return !shouldSelectChannel && strings.TrimSpace(modelName) == ""
}

func getModelRequest(c *gin.Context) (*ModelRequest, bool, error) {
	var modelRequest ModelRequest
	shouldSelectChannel := true
	var err error
	if strings.Contains(c.Request.URL.Path, "/mj/") {
		relayMode := relayconstant.Path2RelayModeMidjourney(c.Request.URL.Path)
		if relayMode == relayconstant.RelayModeMidjourneyTaskFetch ||
			relayMode == relayconstant.RelayModeMidjourneyTaskFetchByCondition ||
			relayMode == relayconstant.RelayModeMidjourneyNotify ||
			relayMode == relayconstant.RelayModeMidjourneyTaskImageSeed {
			shouldSelectChannel = false
		} else {
			midjourneyRequest := dto.MidjourneyRequest{}
			err = common.UnmarshalBodyReusable(c, &midjourneyRequest)
			if err != nil {
				return nil, false, errors.New(i18n.T(c, i18n.MsgDistributorInvalidMidjourney, map[string]any{"Error": err.Error()}))
			}
			midjourneyModel, mjErr, success := service.GetMjRequestModel(relayMode, &midjourneyRequest)
			if mjErr != nil {
				return nil, false, fmt.Errorf("%s", mjErr.Description)
			}
			if midjourneyModel == "" {
				if !success {
					return nil, false, fmt.Errorf("%s", i18n.T(c, i18n.MsgDistributorInvalidParseModel))
				} else {
					// task fetch, task fetch by condition, notify
					shouldSelectChannel = false
				}
			}
			modelRequest.Model = midjourneyModel
		}
		c.Set("relay_mode", relayMode)
	} else if strings.Contains(c.Request.URL.Path, "/suno/") {
		relayMode := relayconstant.Path2RelaySuno(c.Request.Method, c.Request.URL.Path)
		if relayMode == relayconstant.RelayModeSunoFetch ||
			relayMode == relayconstant.RelayModeSunoFetchByID {
			shouldSelectChannel = false
		} else {
			modelName := service.CoverTaskActionToModelName(constant.TaskPlatformSuno, c.Param("action"))
			modelRequest.Model = modelName
		}
		c.Set("platform", string(constant.TaskPlatformSuno))
		c.Set("relay_mode", relayMode)
	} else if strings.HasPrefix(c.Request.URL.Path, "/api/playground/videos/") && c.Request.Method == http.MethodGet {
		// 操练场视频任务查询：GET 无请求体，避免走通用 JSON 解析导致 EOF
		relayMode := relayconstant.RelayModeVideoFetchByID
		c.Set("relay_mode", relayMode)
		shouldSelectChannel = false
	} else if (strings.HasSuffix(c.Request.URL.Path, "/api/playground/videos") ||
		strings.HasSuffix(c.Request.URL.Path, "/pg/videos")) && c.Request.Method == http.MethodPost {
		// 操练场视频提交：与 POST /v1/videos 对齐，确保 TokenFactoryOpen 走 /v1/videos 上游穿透
		relayMode := relayconstant.RelayModeVideoSubmit
		c.Set("relay_mode", relayMode)
		req, err := getModelFromRequest(c)
		if err != nil {
			return nil, false, err
		}
		if req != nil {
			modelRequest.Model = req.Model
			modelRequest.Group = req.Group
			modelRequest.SpecificChannelID = req.SpecificChannelID
			if req.SpecificChannelID != nil {
				if *req.SpecificChannelID <= 0 {
					return nil, false, errors.New(i18n.T(c, i18n.MsgDistributorInvalidPlayground, map[string]any{"Error": "specific_channel_id must be greater than 0"}))
				}
				common.SetContextKey(c, constant.ContextKeyTokenSpecificChannelId, strconv.Itoa(*req.SpecificChannelID))
			}
			common.SetContextKey(c, constant.ContextKeyTokenGroup, modelRequest.Group)
		}
	} else if strings.HasPrefix(c.Request.URL.Path, "/api/playground/images/generations/") && c.Request.Method == http.MethodGet {
		// 操练场图片任务查询：GET 无请求体，避免走通用 JSON 解析导致 EOF
		relayMode := relayconstant.RelayModeVideoFetchByID
		c.Set("relay_mode", relayMode)
		shouldSelectChannel = false
	} else if strings.HasPrefix(c.Request.URL.Path, "/api/playground/images/edits") {
		// 操练场图生图：有参考图时走 OpenAI /v1/images/edits
		relayMode := relayconstant.RelayModeImagesEdits
		c.Set("relay_mode", relayMode)
		if c.Request.Method == http.MethodPost {
			req, err := getModelFromRequest(c)
			if err != nil {
				return nil, false, err
			}
			if req != nil {
				modelRequest.Model = req.Model
			}
		}
	} else if strings.HasPrefix(c.Request.URL.Path, "/api/playground/images/generations") {
		// 操练场文生图：无参考图时走 OpenAI /v1/images/generations
		relayMode := relayconstant.RelayModeImagesGenerations
		c.Set("relay_mode", relayMode)
		if c.Request.Method == http.MethodPost {
			req, err := getModelFromRequest(c)
			if err != nil {
				return nil, false, err
			}
			if req != nil {
				modelRequest.Model = req.Model
			}
		} else {
			shouldSelectChannel = false
		}
	} else if strings.Contains(c.Request.URL.Path, "/v1/videos/") && strings.HasSuffix(c.Request.URL.Path, "/remix") {
		relayMode := relayconstant.RelayModeVideoSubmit
		c.Set("relay_mode", relayMode)
		shouldSelectChannel = false
	} else if strings.Contains(c.Request.URL.Path, "/v1/videos") {
		//curl https://api.openai.com/v1/videos \
		//  -H "Authorization: Bearer $OPENAI_API_KEY" \
		//  -F "model=sora-2" \
		//  -F "prompt=A calico cat playing a piano on stage"
		//	-F input_reference="@image.jpg"
		relayMode := relayconstant.RelayModeUnknown
		if c.Request.Method == http.MethodPost {
			relayMode = relayconstant.RelayModeVideoSubmit
			req, err := getModelFromRequest(c)
			if err != nil {
				return nil, false, err
			}
			if req != nil {
				modelRequest.Model = req.Model
			}
		} else if c.Request.Method == http.MethodGet {
			relayMode = relayconstant.RelayModeVideoFetchByID
			shouldSelectChannel = false
		}
		c.Set("relay_mode", relayMode)
	} else if strings.Contains(c.Request.URL.Path, "/v1/video/generations") {
		relayMode := relayconstant.RelayModeUnknown
		if c.Request.Method == http.MethodPost {
			req, err := getModelFromRequest(c)
			if err != nil {
				return nil, false, err
			}
			modelRequest.Model = req.Model
			relayMode = relayconstant.RelayModeVideoSubmit
		} else if c.Request.Method == http.MethodGet {
			relayMode = relayconstant.RelayModeVideoFetchByID
			shouldSelectChannel = false
		}
		if _, ok := c.Get("relay_mode"); !ok {
			c.Set("relay_mode", relayMode)
		}
	} else if strings.Contains(c.Request.URL.Path, "/api/v3/contents/generations/tasks") && c.Request.Method == http.MethodGet {
		c.Set("relay_mode", relayconstant.RelayModeVideoFetchByID)
		shouldSelectChannel = false
	} else if strings.HasPrefix(c.Request.URL.Path, "/v1beta/models/") || strings.HasPrefix(c.Request.URL.Path, "/v1/models/") {
		// Gemini API 路径处理: /v1beta/models/gemini-2.0-flash:generateContent
		relayMode := relayconstant.RelayModeGemini
		modelName := extractModelNameFromGeminiPath(c.Request.URL.Path)
		if modelName != "" {
			modelRequest.Model = modelName
		}
		c.Set("relay_mode", relayMode)
	} else if !strings.HasPrefix(c.Request.URL.Path, "/v1/audio/transcriptions") && !strings.Contains(c.Request.Header.Get("Content-Type"), "multipart/form-data") {
		req, err := getModelFromRequest(c)
		if err != nil {
			return nil, false, err
		}
		modelRequest.Model = req.Model
	}
	if strings.HasPrefix(c.Request.URL.Path, "/v1/realtime") {
		//wss://api.openai.com/v1/realtime?model=gpt-4o-realtime-preview-2024-10-01
		modelRequest.Model = c.Query("model")
	}
	if strings.HasPrefix(c.Request.URL.Path, "/v1/moderations") {
		if modelRequest.Model == "" {
			modelRequest.Model = "text-moderation-stable"
		}
	}
	if strings.HasSuffix(c.Request.URL.Path, "embeddings") {
		if modelRequest.Model == "" {
			modelRequest.Model = c.Param("model")
		}
	}
	if strings.HasPrefix(c.Request.URL.Path, "/v1/images/generations") {
		modelRequest.Model = common.GetStringIfEmpty(modelRequest.Model, "dall-e")
	} else if strings.HasPrefix(c.Request.URL.Path, "/v1/images/edits") {
		//modelRequest.Model = common.GetStringIfEmpty(c.PostForm("model"), "gpt-image-1")
		contentType := c.ContentType()
		if slices.Contains([]string{gin.MIMEPOSTForm, gin.MIMEMultipartPOSTForm}, contentType) {
			req, err := getModelFromRequest(c)
			if err == nil && req.Model != "" {
				modelRequest.Model = req.Model
			}
		}
	}
	if strings.HasPrefix(c.Request.URL.Path, "/v1/audio") {
		relayMode := relayconstant.RelayModeAudioSpeech
		if strings.HasPrefix(c.Request.URL.Path, "/v1/audio/speech") {

			modelRequest.Model = common.GetStringIfEmpty(modelRequest.Model, "tts-1")
		} else if strings.HasPrefix(c.Request.URL.Path, "/v1/audio/translations") {
			// 先尝试从请求读取
			if req, err := getModelFromRequest(c); err == nil && req.Model != "" {
				modelRequest.Model = req.Model
			}
			modelRequest.Model = common.GetStringIfEmpty(modelRequest.Model, "whisper-1")
			relayMode = relayconstant.RelayModeAudioTranslation
		} else if strings.HasPrefix(c.Request.URL.Path, "/v1/audio/transcriptions/async") {
			// 阿里云 ASR 异步转写：POST 提交任务需按模型选渠道；GET 查询按 task_id 定位任务与渠道，无需选渠道
			if c.Request.Method == http.MethodPost {
				if req, err := getModelFromRequest(c); err == nil && req.Model != "" {
					modelRequest.Model = req.Model
				}
				relayMode = relayconstant.RelayModeAudioTranscriptionAsyncSubmit
			} else {
				shouldSelectChannel = false
				relayMode = relayconstant.RelayModeAudioTranscriptionAsyncFetch
			}
		} else if strings.HasPrefix(c.Request.URL.Path, "/v1/audio/transcriptions") {
			// 先尝试从请求读取
			if req, err := getModelFromRequest(c); err == nil && req.Model != "" {
				modelRequest.Model = req.Model
			}
			modelRequest.Model = common.GetStringIfEmpty(modelRequest.Model, "whisper-1")
			relayMode = relayconstant.RelayModeAudioTranscription
		}
		c.Set("relay_mode", relayMode)
	}
	if strings.HasPrefix(c.Request.URL.Path, "/pg/chat/completions") {
		// playground chat completions
		req, err := getModelFromRequest(c)
		if err != nil {
			return nil, false, err
		}
		modelRequest.Model = req.Model
		modelRequest.Group = req.Group
		modelRequest.SpecificChannelID = req.SpecificChannelID
		if req.SpecificChannelID != nil {
			if *req.SpecificChannelID <= 0 {
				return nil, false, errors.New(i18n.T(c, i18n.MsgDistributorInvalidPlayground, map[string]any{"Error": "specific_channel_id 必须大于 0"}))
			}
			common.SetContextKey(c, constant.ContextKeyTokenSpecificChannelId, strconv.Itoa(*req.SpecificChannelID))
		}
		common.SetContextKey(c, constant.ContextKeyTokenGroup, modelRequest.Group)
	}

	if strings.HasPrefix(c.Request.URL.Path, "/v1/responses/compact") && modelRequest.Model != "" {
		modelRequest.Model = ratio_setting.WithCompactModelSuffix(modelRequest.Model)
	}
	return &modelRequest, shouldSelectChannel, nil
}

func SetupContextForSelectedChannel(c *gin.Context, channel *model.Channel, modelName string) *types.TokenFactoryError {
	c.Set("original_model", modelName) // for retry
	if channel == nil {
		return types.NewError(errors.New("channel is nil"), types.ErrorCodeGetChannelFailed, types.ErrOptionWithSkipRetry())
	}
	common.SetContextKey(c, constant.ContextKeyChannelId, channel.Id)
	common.SetContextKey(c, constant.ContextKeyChannelName, channel.Name)
	common.SetContextKey(c, constant.ContextKeyChannelType, channel.Type)
	common.SetContextKey(c, constant.ContextKeyChannelCreateTime, channel.CreatedTime)
	common.SetContextKey(c, constant.ContextKeyChannelSetting, channel.GetSetting())
	common.SetContextKey(c, constant.ContextKeyChannelOtherSetting, channel.GetOtherSettings())
	paramOverride := channel.GetParamOverride()
	headerOverride := channel.GetHeaderOverride()
	if mergedParam, applied := service.ApplyChannelAffinityOverrideTemplate(c, paramOverride); applied {
		paramOverride = mergedParam
	}
	common.SetContextKey(c, constant.ContextKeyChannelParamOverride, paramOverride)
	common.SetContextKey(c, constant.ContextKeyChannelHeaderOverride, headerOverride)
	if nil != channel.OpenAIOrganization && *channel.OpenAIOrganization != "" {
		common.SetContextKey(c, constant.ContextKeyChannelOrganization, *channel.OpenAIOrganization)
	}
	common.SetContextKey(c, constant.ContextKeyChannelAutoBan, channel.GetAutoBan())
	common.SetContextKey(c, constant.ContextKeyChannelModelMapping, channel.GetModelMapping())
	common.SetContextKey(c, constant.ContextKeyChannelStatusCodeMapping, channel.GetStatusCodeMapping())

	key, index, tokenFactoryError := channel.GetNextEnabledKey()
	if tokenFactoryError != nil {
		return tokenFactoryError
	}
	if channel.ChannelInfo.IsMultiKey {
		common.SetContextKey(c, constant.ContextKeyChannelIsMultiKey, true)
		common.SetContextKey(c, constant.ContextKeyChannelMultiKeyIndex, index)
	} else {
		// 必须设置为 false，否则在重试到单个 key 的时候会导致日志显示错误
		common.SetContextKey(c, constant.ContextKeyChannelIsMultiKey, false)
	}
	// c.Request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", key))
	common.SetContextKey(c, constant.ContextKeyChannelKey, key)
	common.SetContextKey(c, constant.ContextKeyChannelBaseUrl, channel.GetBaseURL())

	common.SetContextKey(c, constant.ContextKeySystemPromptOverride, false)

	// TODO: api_version统一
	switch channel.Type {
	case constant.ChannelTypeAzure:
		c.Set("api_version", channel.Other)
	case constant.ChannelTypeVertexAi:
		c.Set("region", channel.Other)
	case constant.ChannelTypeXunfei:
		c.Set("api_version", channel.Other)
	case constant.ChannelTypeGemini:
		c.Set("api_version", channel.Other)
	case constant.ChannelTypeAli:
		c.Set("plugin", channel.Other)
	case constant.ChannelCloudflare:
		c.Set("api_version", channel.Other)
	case constant.ChannelTypeMokaAI:
		c.Set("api_version", channel.Other)
	case constant.ChannelTypeCoze:
		c.Set("bot_id", channel.Other)
	}

	// 仅当本地渠道类型为 TokenFactoryOpen(60) 时才拼装上游精准路由。
	// 从 TF Open 同步下来的普通 OpenAI 等渠道（source=tokenfactory_open 但 type≠60）
	// 已直连真实上游网关，模型名不得带 /route_slug，否则会被上游判为模型不存在。
	otherInfo := channel.GetOtherInfo()
	if channel.Type == constant.ChannelTypeTokenFactoryOpen {
		if source, _ := otherInfo["source"].(string); source == "tokenfactory_open" {
			upstreamRouteSlug := strings.TrimSpace(common.Interface2String(otherInfo["upstream_route_slug"]))
			if upstreamRouteSlug != "" && model.IsValidRouteSlug(upstreamRouteSlug) {
				common.SetContextKey(c, constant.ContextKeyTFOpenUpstreamChannelRoute, upstreamRouteSlug)
				logger.LogInfo(c, fmt.Sprintf("tfopen route selected: route_slug=%s channel=%s(id=%d) model=%s", upstreamRouteSlug, channel.Name, channel.Id, modelName))
			} else {
				// 回退到旧版 alias|channelNo 三段式路由（兼容未同步 route_slug 的旧渠道）
				alias := strings.TrimSpace(common.Interface2String(otherInfo["upstream_supplier_alias"]))
				if alias == "" {
					if strings.TrimSpace(common.Interface2String(otherInfo["upstream_supplier_app_id"])) == "0" {
						alias = "P0"
					}
				}
				channelNo := strings.TrimSpace(common.Interface2String(otherInfo["upstream_channel_no"]))
				if override := strings.TrimSpace(common.GetContextKeyString(c, constant.ContextKeyTFOpenUpstreamChannelNoOverride)); override != "" {
					channelNo = override
				}
				if alias != "" && channelNo != "" {
					common.SetContextKey(c, constant.ContextKeyTFOpenUpstreamChannelRoute, "legacy|"+alias+"|"+channelNo)
					logger.LogInfo(c, fmt.Sprintf("tfopen route selected (legacy): route=%s|%s channel=%s(id=%d) model=%s", alias, channelNo, channel.Name, channel.Id, modelName))
				}
			}
		}
	}

	return nil
}

// extractModelNameFromGeminiPath 从 Gemini API URL 路径中提取模型名
// 输入格式: /v1beta/models/gemini-2.0-flash:generateContent
// 输出: gemini-2.0-flash
func extractModelNameFromGeminiPath(path string) string {
	// 查找 "/models/" 的位置
	modelsPrefix := "/models/"
	modelsIndex := strings.Index(path, modelsPrefix)
	if modelsIndex == -1 {
		return ""
	}

	// 从 "/models/" 之后开始提取
	startIndex := modelsIndex + len(modelsPrefix)
	if startIndex >= len(path) {
		return ""
	}

	// 查找 ":" 的位置，模型名在 ":" 之前
	colonIndex := strings.Index(path[startIndex:], ":")
	if colonIndex == -1 {
		// 如果没有找到 ":"，返回从 "/models/" 到路径结尾的部分
		return path[startIndex:]
	}

	// 返回模型名部分
	return path[startIndex : startIndex+colonIndex]
}

// tryTokenFactoryRoute 已废弃，统一由 service.TryEngineRoute 处理。
