package controller

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	perfmetrics "github.com/QuantumNous/new-api/pkg/perf_metrics"
	"github.com/QuantumNous/new-api/relay"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/samber/lo"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func resolveRelayPriceData(c *gin.Context, relayInfo *relaycommon.RelayInfo, tokens int, meta *types.TokenCountMeta) (types.PriceData, error) {
	if relayInfo != nil && relayInfo.ChannelMeta != nil && constant.IsASRChannel(relayInfo.ChannelType) {
		// 阿里云 ASR 按秒计费：预估 token 按 1000 token/分钟（EstimateRequestToken 音频口径）折算回秒数；
		// URL 模式 tokens=0 表示不预扣，结算时按上游返回时长一次性扣费。
		estSeconds := float64(tokens) * 60.0 / 1000.0
		return helper.ModelPriceHelperASR(c, relayInfo, estSeconds)
	}
	if relayInfo != nil &&
		(relayInfo.RelayMode == relayconstant.RelayModeImagesGenerations ||
			relayInfo.RelayMode == relayconstant.RelayModeImagesEdits) {
		channelID := 0
		if relayInfo.ChannelMeta != nil {
			channelID = relayInfo.ChannelId
		}
		hasImageTable := helper.HasImagePerImageTablePricing(channelID, relayInfo.OriginModelName) ||
			helper.HasImagePerImageTablePricingForInfo(channelID, relayInfo)
		if priceData, ok, err := helper.TryModelPriceHelperImage(c, relayInfo); err != nil {
			return types.PriceData{}, err
		} else if ok {
			return priceData, nil
		}
		if hasImageTable {
			return types.PriceData{}, helper.ImageModelPriceMatchError(c, channelID, relayInfo)
		}
		return helper.ModelPriceHelperForImageFallback(c, relayInfo, tokens, meta)
	}
	return helper.ModelPriceHelper(c, relayInfo, tokens, meta)
}

func relayHandler(c *gin.Context, info *relaycommon.RelayInfo) *types.TokenFactoryError {
	var err *types.TokenFactoryError
	switch info.RelayMode {
	case relayconstant.RelayModeImagesGenerations, relayconstant.RelayModeImagesEdits:
		err = relay.ImageHelper(c, info)
	case relayconstant.RelayModeAudioSpeech:
		fallthrough
	case relayconstant.RelayModeAudioTranslation:
		fallthrough
	case relayconstant.RelayModeAudioTranscription:
		err = relay.AudioHelper(c, info)
	case relayconstant.RelayModeRerank:
		err = relay.RerankHelper(c, info)
	case relayconstant.RelayModeEmbeddings:
		err = relay.EmbeddingHelper(c, info)
	case relayconstant.RelayModeResponses, relayconstant.RelayModeResponsesCompact:
		err = relay.ResponsesHelper(c, info)
	default:
		err = relay.TextHelper(c, info)
	}
	return err
}

func geminiRelayHandler(c *gin.Context, info *relaycommon.RelayInfo) *types.TokenFactoryError {
	var err *types.TokenFactoryError
	if strings.Contains(c.Request.URL.Path, "embed") {
		err = relay.GeminiEmbeddingHandler(c, info)
	} else {
		err = relay.GeminiHelper(c, info)
	}
	return err
}

func Relay(c *gin.Context, relayFormat types.RelayFormat) {

	requestId := c.GetString(common.RequestIdKey)
	//group := common.GetContextKeyString(c, constant.ContextKeyUsingGroup)
	//originalModel := common.GetContextKeyString(c, constant.ContextKeyOriginalModel)

	var (
		tokenFactoryError *types.TokenFactoryError
		ws                *websocket.Conn
	)

	if relayFormat == types.RelayFormatOpenAIRealtime {
		var err error
		ws, err = upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			helper.WssError(c, ws, types.NewError(err, types.ErrorCodeGetChannelFailed, types.ErrOptionWithSkipRetry()).ToOpenAIError())
			return
		}
		defer ws.Close()
	}

	defer func() {
		if tokenFactoryError != nil {
			if blocked, _ := c.Get("aliyun_guardrail_output_blocked"); blocked == true {
				if isStream, _ := c.Get("aliyun_guardrail_output_stream"); isStream == true {
					c.Header("Content-Type", "text/event-stream")
					_ = helper.ObjectData(c, gin.H{
						"object": "chat.completion.chunk",
						"choices": []gin.H{{
							"index":         0,
							"delta":         gin.H{"content": setting.AliyunGuardrailBlockedReply},
							"finish_reason": "content_filter",
						}},
					})
					helper.Done(c)
					return
				}
				c.JSON(http.StatusOK, gin.H{
					"object": "chat.completion",
					"choices": []gin.H{{
						"index":         0,
						"message":       gin.H{"role": "assistant", "content": setting.AliyunGuardrailBlockedReply},
						"finish_reason": "content_filter",
					}},
				})
				return
			}
			logger.LogError(c, fmt.Sprintf("relay error [来源=%s] (error_type=%s, error_code=%s): %s",
				tokenFactoryError.LogErrorOriginHint(), tokenFactoryError.GetErrorType(), tokenFactoryError.GetErrorCode(), tokenFactoryError.Error()))
			tokenFactoryError.SetMessage(common.MessageWithRequestId(tokenFactoryError.Error(), requestId))
			switch relayFormat {
			case types.RelayFormatOpenAIRealtime:
				helper.WssError(c, ws, tokenFactoryError.ToOpenAIError())
			case types.RelayFormatClaude:
				c.JSON(tokenFactoryError.StatusCode, gin.H{
					"type":  "error",
					"error": tokenFactoryError.ToClaudeError(),
				})
			default:
				c.JSON(tokenFactoryError.StatusCode, gin.H{
					"error": tokenFactoryError.ToOpenAIError(),
				})
			}
		}
	}()

	request, err := helper.GetAndValidateRequest(c, relayFormat)
	if err != nil {
		// Map "request body too large" to 413 so clients can handle it correctly
		if common.IsRequestBodyTooLargeError(err) || errors.Is(err, common.ErrRequestBodyTooLarge) {
			tokenFactoryError = types.NewErrorWithStatusCode(err, types.ErrorCodeReadRequestBodyFailed, http.StatusRequestEntityTooLarge, types.ErrOptionWithSkipRetry())
		} else {
			tokenFactoryError = types.NewError(err, types.ErrorCodeInvalidRequest)
		}
		return
	}

	relayInfo, err := relaycommon.GenRelayInfo(c, relayFormat, request, ws)
	if err != nil {
		tokenFactoryError = types.NewError(err, types.ErrorCodeGenRelayInfoFailed)
		return
	}

	needSensitiveCheck := setting.ShouldCheckPromptSensitive()
	needCountToken := constant.CountToken
	// Avoid building huge CombineText (strings.Join) when token counting and sensitive check are both disabled.
	var meta *types.TokenCountMeta
	if needSensitiveCheck || needCountToken {
		meta = request.GetTokenCountMeta()
	} else {
		meta = fastTokenCountMetaForPricing(request)
	}

	if needSensitiveCheck && meta != nil {
		contains, words := service.CheckSensitiveText(meta.CombineText)
		if contains {
			logger.LogWarn(c, fmt.Sprintf("user sensitive words detected: %s", strings.Join(words, ", ")))
			tokenFactoryError = types.NewError(err, types.ErrorCodeSensitiveWordsDetected)
			return
		}
	}

	guardrailResult, guardrailErr := service.CheckAliyunGuardrailInput(c, relayInfo, request)
	if guardrailErr != nil {
		logger.LogWarn(c, fmt.Sprintf("aliyun guardrail check skipped: %s", guardrailErr.Error()))
	}
	if guardrailResult != nil && guardrailResult.Blocked {
		c.Set("aliyun_guardrail_output_blocked", true)
		c.Set("aliyun_guardrail_output_stream", relayInfo.IsStream)
		tokenFactoryError = types.NewErrorWithStatusCode(fmt.Errorf(setting.AliyunGuardrailBlockedReply), types.ErrorCodeSensitiveWordsDetected, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
		return
	}

	tokens, err := service.EstimateRequestToken(c, meta, relayInfo)
	if err != nil {
		tokenFactoryError = types.NewError(err, types.ErrorCodeCountTokenFailed)
		return
	}

	relayInfo.SetEstimatePromptTokens(tokens)

	retryParam := &service.RetryParam{
		Ctx:        c,
		TokenGroup: relayInfo.TokenGroup,
		ModelName:  relayInfo.OriginModelName,
		Retry:      common.GetPointer(0),
	}
	relayInfo.RetryIndex = 0
	relayInfo.LastError = nil

	// Select first channel before pricing so pre-consume uses selected channel pricing.
	firstChannel, firstChannelErr := getChannel(c, relayInfo, retryParam)
	if firstChannelErr != nil {
		logger.LogError(c, firstChannelErr.Error())
		tokenFactoryError = firstChannelErr
		return
	}
	if relayFormat != types.RelayFormatTask {
		if tfErr := errVideoTaskChannelOnNonTaskRelay(firstChannel); tfErr != nil {
			tokenFactoryError = tfErr
			return
		}
	}
	if relayInfo.ChannelMeta == nil {
		relayInfo.InitChannelMeta(c)
	}

	priceData, err := resolveRelayPriceData(c, relayInfo, tokens, meta)
	if err != nil {
		tokenFactoryError = types.NewError(err, types.ErrorCodeModelPriceError)
		return
	}

	// common.SetContextKey(c, constant.ContextKeyTokenCountMeta, meta)

	if priceData.FreeModel {
		logger.LogInfo(c, fmt.Sprintf("模型 %s 免费，跳过预扣费", relayInfo.OriginModelName))
	} else {
		tokenFactoryError = service.PreConsumeBilling(c, priceData.QuotaToPreConsume, relayInfo)
		if tokenFactoryError != nil {
			return
		}
	}

	defer func() {
		// Only return quota if downstream failed and quota was actually pre-consumed
		if tokenFactoryError != nil {
			tokenFactoryError = service.NormalizeViolationFeeError(tokenFactoryError)
			if relayInfo.Billing != nil {
				relayInfo.Billing.Refund(c)
			}
			service.ChargeViolationFeeIfNeeded(c, relayInfo, tokenFactoryError)
		}
	}()

	maxRelayRetries := service.RouteOrderedMaxRetries(c)
	for ; retryParam.GetRetry() <= maxRelayRetries; retryParam.IncreaseRetry() {
		relayInfo.RetryIndex = retryParam.GetRetry()
		channel := firstChannel
		if retryParam.GetRetry() > 0 {
			var channelErr *types.TokenFactoryError
			channel, channelErr = getChannel(c, relayInfo, retryParam)
			if channelErr != nil {
				logger.LogError(c, channelErr.Error())
				tokenFactoryError = channelErr
				break
			}
		}
		if relayFormat != types.RelayFormatTask {
			if tfErr := errVideoTaskChannelOnNonTaskRelay(channel); tfErr != nil {
				tokenFactoryError = tfErr
				relayInfo.LastError = tokenFactoryError
				processChannelError(c, *types.NewChannelError(channel.Id, channel.Type, channel.Name, channel.ChannelInfo.IsMultiKey, common.GetContextKeyString(c, constant.ContextKeyChannelKey), channel.GetAutoBan()), tokenFactoryError)
				break
			}
		}

		// Refresh PriceData after final channel decision of this attempt.
		if relayInfo.ChannelMeta == nil {
			relayInfo.InitChannelMeta(c)
		}
		if _, priceErr := resolveRelayPriceData(c, relayInfo, tokens, meta); priceErr != nil {
			tokenFactoryError = types.NewError(priceErr, types.ErrorCodeModelPriceError)
			relayInfo.LastError = tokenFactoryError
			processChannelError(c, *types.NewChannelError(channel.Id, channel.Type, channel.Name, channel.ChannelInfo.IsMultiKey, common.GetContextKeyString(c, constant.ContextKeyChannelKey), channel.GetAutoBan()), tokenFactoryError)
			if !shouldRetry(c, tokenFactoryError, maxRelayRetries-retryParam.GetRetry()) {
				break
			}
			continue
		}

		addUsedChannel(c, channel.Id)
		bodyStorage, bodyErr := common.GetBodyStorage(c)
		if bodyErr != nil {
			// Ensure consistent 413 for oversized bodies even when error occurs later (e.g., retry path)
			if common.IsRequestBodyTooLargeError(bodyErr) || errors.Is(bodyErr, common.ErrRequestBodyTooLarge) {
				tokenFactoryError = types.NewErrorWithStatusCode(bodyErr, types.ErrorCodeReadRequestBodyFailed, http.StatusRequestEntityTooLarge, types.ErrOptionWithSkipRetry())
			} else {
				tokenFactoryError = types.NewErrorWithStatusCode(bodyErr, types.ErrorCodeReadRequestBodyFailed, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
			}
			break
		}
		c.Request.Body = io.NopCloser(bodyStorage)

		switch relayFormat {
		case types.RelayFormatOpenAIRealtime:
			tokenFactoryError = relay.WssHelper(c, relayInfo)
		case types.RelayFormatClaude:
			tokenFactoryError = relay.ClaudeHelper(c, relayInfo)
		case types.RelayFormatGemini:
			tokenFactoryError = geminiRelayHandler(c, relayInfo)
		default:
			tokenFactoryError = relayHandler(c, relayInfo)
		}

		if tokenFactoryError == nil {
			relayInfo.LastError = nil
			// 反馈给 TokenFactory 黏性熔断器：成功则清零该渠道的连续报错计数。
			service.RecordTFRouteResult(c, channel.Id, true)
			return
		}

		tokenFactoryError = service.NormalizeViolationFeeError(tokenFactoryError)
		relayInfo.LastError = tokenFactoryError

		processChannelError(c, *types.NewChannelError(channel.Id, channel.Type, channel.Name, channel.ChannelInfo.IsMultiKey, common.GetContextKeyString(c, constant.ContextKeyChannelKey), channel.GetAutoBan()), tokenFactoryError)
		// 反馈给 TokenFactory 黏性熔断器：仅统计 TF 选中的那个渠道的连续报错。
		service.RecordTFRouteResult(c, channel.Id, false)

		if !shouldRetry(c, tokenFactoryError, maxRelayRetries-retryParam.GetRetry()) {
			break
		}
	}

	useChannel := c.GetStringSlice("use_channel")
	if len(useChannel) > 1 {
		retryLogStr := fmt.Sprintf("重试：%s", strings.Trim(strings.Join(strings.Fields(fmt.Sprint(useChannel)), "->"), "[]"))
		logger.LogInfo(c, retryLogStr)
	}
	if tokenFactoryError != nil && !relayInfo.IsChannelTest {
		gopool.Go(func() {
			perfmetrics.RecordRelaySample(relayInfo, false, 0, 0, 0)
		})
	}
}

var upgrader = websocket.Upgrader{
	Subprotocols: []string{"realtime"}, // WS 握手支持的协议，如果有使用 Sec-WebSocket-Protocol，则必须在此声明对应的 Protocol TODO add other protocol
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许跨域
	},
}

func addUsedChannel(c *gin.Context, channelId int) {
	useChannel := c.GetStringSlice("use_channel")
	useChannel = append(useChannel, fmt.Sprintf("%d", channelId))
	c.Set("use_channel", useChannel)
}

func fastTokenCountMetaForPricing(request dto.Request) *types.TokenCountMeta {
	if request == nil {
		return &types.TokenCountMeta{}
	}
	meta := &types.TokenCountMeta{
		TokenType: types.TokenTypeTokenizer,
	}
	switch r := request.(type) {
	case *dto.GeneralOpenAIRequest:
		maxCompletionTokens := lo.FromPtrOr(r.MaxCompletionTokens, uint(0))
		maxTokens := lo.FromPtrOr(r.MaxTokens, uint(0))
		if maxCompletionTokens > maxTokens {
			meta.MaxTokens = int(maxCompletionTokens)
		} else {
			meta.MaxTokens = int(maxTokens)
		}
	case *dto.OpenAIResponsesRequest:
		meta.MaxTokens = int(lo.FromPtrOr(r.MaxOutputTokens, uint(0)))
	case *dto.ClaudeRequest:
		meta.MaxTokens = int(lo.FromPtr(r.MaxTokens))
	case *dto.ImageRequest:
		// Pricing for image requests depends on ImagePriceRatio; safe to compute even when CountToken is disabled.
		return r.GetTokenCountMeta()
	default:
		// Best-effort: leave CombineText empty to avoid large allocations.
	}
	return meta
}

// errVideoTaskChannelOnNonTaskRelay 视频任务类渠道（Sora/OpenAI 视频/腾讯云视频等）只能走 RelayTask 与 ModelPriceHelperVideo；
// 若误命中 /v1/chat/completions 等会按文本 token 计费，导致控制台日志与利润分成错误。
func errVideoTaskChannelOnNonTaskRelay(ch *model.Channel) *types.TokenFactoryError {
	if ch == nil || !constant.IsVideoTaskChannel(ch.Type) {
		return nil
	}
	return types.NewErrorWithStatusCode(
		fmt.Errorf(
			"当前模型命中了视频任务渠道，仅支持视频任务接口（如 POST /v1/videos 或操练场 POST /api/playground/videos），"+
				"不能使用聊天补全、嵌入、图生等非任务接口，否则会按文本 token 误计费；请改用视频任务 API 或为该模型配置文本类渠道",
		),
		types.ErrorCodeInvalidRequest,
		http.StatusBadRequest,
		types.ErrOptionWithSkipRetry(),
	)
}

func getChannel(c *gin.Context, info *relaycommon.RelayInfo, retryParam *service.RetryParam) (*model.Channel, *types.TokenFactoryError) {
	if info.ChannelMeta == nil {
		info.InitChannelMeta(c)
	}
	// 首轮优先复用分发中间件已经选中的渠道，避免重复选路覆盖 specific_channel_id 语义。
	if retryParam.GetRetry() == 0 {
		if selectedID := common.GetContextKeyInt(c, constant.ContextKeyChannelId); selectedID > 0 {
			if ch, chErr := model.CacheGetChannel(selectedID); chErr == nil && ch != nil && ch.Status == common.ChannelStatusEnabled {
				return ch, nil
			}
		}
	}
	// playground specific_channel_id / 硬指定渠道路由（{alias}/{model}/cN）：仅允许首轮命中已选渠道，
	// 禁止在重试阶段切换到 smart-route 或随机候选池。
	// 注意：{model}/{route_slug} 使用 PreferredChannelID + 有序候选，允许保底切换；
	// 若带 X-TF-No-Failover 则同样禁止切换。
	if retryParam.GetRetry() > 0 {
		if _, ok := common.GetContextKey(c, constant.ContextKeyTokenSpecificChannelId); ok {
			return nil, types.NewError(
				fmt.Errorf("已指定渠道，禁用重试切换渠道"),
				types.ErrorCodeGetChannelFailed,
				types.ErrOptionWithSkipRetry(),
			)
		}
		if _, ok := common.GetContextKey(c, constant.ContextKeyForcedChannelID); ok {
			return nil, types.NewError(
				fmt.Errorf("已指定渠道，禁用重试切换渠道"),
				types.ErrorCodeGetChannelFailed,
				types.ErrOptionWithSkipRetry(),
			)
		}
		if service.ContextNoFailover(c) {
			return nil, types.NewError(
				fmt.Errorf("已禁用渠道切换（X-TF-No-Failover）"),
				types.ErrorCodeGetChannelFailed,
				types.ErrOptionWithSkipRetry(),
			)
		}
	}
	if order, ok := service.RouteOrderedChannelIDs(c); ok {
		if ch, picked := service.PickNextOrderedRouteChannel(c, order); picked {
			info.PriceData.GroupRatioInfo = helper.HandleGroupRatio(c, info)
			if tfErr := middleware.SetupContextForSelectedChannel(c, ch, info.OriginModelName); tfErr != nil {
				return nil, tfErr
			}
			if tfErr := middleware.ChannelModelRateLimitError(ch, info.OriginModelName); tfErr != nil {
				return nil, tfErr
			}
			service.RefreshTFRouteStickyChannel(c, ch.Id)
			return ch, nil
		}
		return nil, types.NewError(
			fmt.Errorf("有序路由候选已用尽"),
			types.ErrorCodeGetChannelFailed,
			types.ErrOptionWithSkipRetry(),
		)
	}
	// 命中「指定供应商 + 任意渠道」时，候选池已限定；order 列表耗尽意味着供应商内无更多可用渠道，
	// 不应回落到全局的 CacheGetRandomSatisfiedChannel（那会跨供应商），直接结束重试。
	if _, forced := service.ForcedSupplierFromContext(c); forced {
		return nil, types.NewError(fmt.Errorf("分组 %s 下模型 %s 在指定供应商内已无可用渠道（retry）", retryParam.TokenGroup, info.OriginModelName), types.ErrorCodeGetChannelFailed, types.ErrOptionWithSkipRetry())
	}
	channel, selectGroup, err := service.CacheGetRandomSatisfiedChannel(retryParam)

	info.PriceData.GroupRatioInfo = helper.HandleGroupRatio(c, info)

	if err != nil {
		return nil, types.NewError(fmt.Errorf("获取分组 %s 下模型 %s 的可用渠道失败（retry）: %s", selectGroup, info.OriginModelName, err.Error()), types.ErrorCodeGetChannelFailed, types.ErrOptionWithSkipRetry())
	}
	if channel == nil {
		return nil, types.NewError(fmt.Errorf("分组 %s 下模型 %s 的可用渠道不存在（retry）", selectGroup, info.OriginModelName), types.ErrorCodeGetChannelFailed, types.ErrOptionWithSkipRetry())
	}

	// 用户指定价：重试随机兜底选中的渠道也不允许超出用户价格上限。
	if !service.ChannelWithinUserPriceCap(c.GetInt("id"), info.OriginModelName, channel) {
		return nil, types.NewError(
			fmt.Errorf("分组 %s 下模型 %s 已无满足用户指定价上限的可用渠道（retry）", selectGroup, info.OriginModelName),
			types.ErrorCodeGetChannelFailed,
			types.ErrOptionWithSkipRetry(),
		)
	}

	tokenFactoryError := middleware.SetupContextForSelectedChannel(c, channel, info.OriginModelName)
	if tokenFactoryError != nil {
		return nil, tokenFactoryError
	}
	if retryParam.GetRetry() > 0 {
		if tfErr := middleware.ChannelModelRateLimitError(channel, info.OriginModelName); tfErr != nil {
			return nil, tfErr
		}
	}
	return channel, nil
}

func shouldRetry(c *gin.Context, openaiErr *types.TokenFactoryError, retryTimes int) bool {
	if openaiErr == nil {
		return false
	}
	if service.ShouldSkipRetryAfterChannelAffinityFailure(c) {
		return false
	}
	// 明确指定渠道（playground specific_channel_id / {alias}/{model}/cN 硬指定）时，不允许重试切换渠道。
	// {model}/{route_slug} 偏好渠道走有序候选保底，不在此拦截；X-TF-No-Failover 显式关闭切换。
	if _, ok := c.Get("specific_channel_id"); ok {
		return false
	}
	if _, ok := common.GetContextKey(c, constant.ContextKeyTokenSpecificChannelId); ok {
		return false
	}
	if _, ok := common.GetContextKey(c, constant.ContextKeyForcedChannelID); ok {
		return false
	}
	if service.ContextNoFailover(c) {
		return false
	}
	if types.IsChannelError(openaiErr) {
		return true
	}
	if types.IsSkipRetryError(openaiErr) {
		return false
	}
	if retryTimes <= 0 {
		return false
	}
	code := openaiErr.StatusCode
	if code >= 200 && code < 300 {
		return false
	}
	// 有序智能路由尚有未尝试渠道时：4xx/5xx（含 400/408/504/524）一律切换，优先保证可用性。
	if code >= 400 && code <= 599 && service.HasUnusedOrderedRouteChannel(c) {
		return true
	}
	if code < 100 || code > 599 {
		return true
	}
	if operation_setting.IsAlwaysSkipRetryCode(openaiErr.GetErrorCode()) {
		return false
	}
	return operation_setting.ShouldRetryByStatusCode(code)
}

func processChannelError(c *gin.Context, channelError types.ChannelError, err *types.TokenFactoryError) {
	logger.LogError(c, fmt.Sprintf("channel error [来源=%s] (channel #%d, status_code=%d, error_type=%s, error_code=%s): %s",
		err.LogErrorOriginHint(), channelError.ChannelId, err.StatusCode, err.GetErrorType(), err.GetErrorCode(), err.Error()))
	// 不要使用context获取渠道信息，异步处理时可能会出现渠道信息不一致的情况
	// do not use context to get channel info, there may be inconsistent channel info when processing asynchronously
	if service.ShouldDisableChannel(channelError.ChannelType, err) && channelError.AutoBan {
		gopool.Go(func() {
			service.DisableChannel(channelError, err.ErrorWithStatusCode())
		})
	}

	// ASR 上游错误始终写入使用日志（不受 ERROR_LOG_ENABLED 默认关闭影响）
	if constant.IsASRChannel(channelError.ChannelType) && types.IsRecordErrorLog(err) {
		// 以本次失败渠道为准写入，避免重试时上下文渠道信息不一致
		c.Set("channel_id", channelError.ChannelId)
		c.Set("channel_name", channelError.ChannelName)
		c.Set("channel_type", channelError.ChannelType)
		service.RecordASRErrorLog(c, nil, err, "")
		return
	}

	if constant.ErrorLogEnabled && types.IsRecordErrorLog(err) {
		// 保存错误日志到mysql中
		userId := c.GetInt("id")
		tokenName := c.GetString("token_name")
		modelName := c.GetString("original_model")
		tokenId := c.GetInt("token_id")
		userGroup := c.GetString("group")
		channelId := c.GetInt("channel_id")
		other := make(map[string]interface{})
		if c.Request != nil && c.Request.URL != nil {
			other["request_path"] = c.Request.URL.Path
		}
		other["error_type"] = err.GetErrorType()
		other["error_code"] = err.GetErrorCode()
		other["status_code"] = err.StatusCode
		other["channel_id"] = channelId
		other["channel_name"] = c.GetString("channel_name")
		other["channel_type"] = c.GetInt("channel_type")
		adminInfo := make(map[string]interface{})
		adminInfo["use_channel"] = c.GetStringSlice("use_channel")
		isMultiKey := common.GetContextKeyBool(c, constant.ContextKeyChannelIsMultiKey)
		if isMultiKey {
			adminInfo["is_multi_key"] = true
			adminInfo["multi_key_index"] = common.GetContextKeyInt(c, constant.ContextKeyChannelMultiKeyIndex)
		}
		service.AppendChannelAffinityAdminInfo(c, adminInfo)
		other["admin_info"] = adminInfo
		startTime := common.GetContextKeyTime(c, constant.ContextKeyRequestStartTime)
		if startTime.IsZero() {
			startTime = time.Now()
		}
		useTimeSeconds := int(time.Since(startTime).Seconds())
		// content / other 写入 DB 供 /log 查看，保持不变；来源提示仅打在服务端日志。
		model.RecordErrorLog(c, userId, channelId, modelName, tokenName, err.MaskSensitiveErrorWithStatusCode(), tokenId, useTimeSeconds, false, userGroup, other, err.LogErrorOriginHint())
	}

}

func RelayMidjourney(c *gin.Context) {
	relayInfo, err := relaycommon.GenRelayInfo(c, types.RelayFormatMjProxy, nil, nil)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"description": fmt.Sprintf("failed to generate relay info: %s", err.Error()),
			"type":        "upstream_error",
			"code":        4,
		})
		return
	}

	var mjErr *dto.MidjourneyResponse
	switch relayInfo.RelayMode {
	case relayconstant.RelayModeMidjourneyNotify:
		mjErr = relay.RelayMidjourneyNotify(c)
	case relayconstant.RelayModeMidjourneyTaskFetch, relayconstant.RelayModeMidjourneyTaskFetchByCondition:
		mjErr = relay.RelayMidjourneyTask(c, relayInfo.RelayMode)
	case relayconstant.RelayModeMidjourneyTaskImageSeed:
		mjErr = relay.RelayMidjourneyTaskImageSeed(c)
	case relayconstant.RelayModeSwapFace:
		mjErr = relay.RelaySwapFace(c, relayInfo)
	default:
		mjErr = relay.RelayMidjourneySubmit(c, relayInfo)
	}
	//err = relayMidjourneySubmit(c, relayMode)
	log.Println(mjErr)
	if mjErr != nil {
		statusCode := http.StatusBadRequest
		if mjErr.Code == 30 {
			mjErr.Result = "当前分组负载已饱和，请稍后再试，或升级账户以提升服务质量。"
			statusCode = http.StatusTooManyRequests
		}
		c.JSON(statusCode, gin.H{
			"description": fmt.Sprintf("%s %s", mjErr.Description, mjErr.Result),
			"type":        "upstream_error",
			"code":        mjErr.Code,
		})
		channelId := c.GetInt("channel_id")
		logger.LogError(c, fmt.Sprintf("relay error [来源=上游] (channel #%d, status_code=%d): %s", channelId, statusCode, fmt.Sprintf("%s %s", mjErr.Description, mjErr.Result)))
	}
}

func RelayNotImplemented(c *gin.Context) {
	err := types.OpenAIError{
		Message: "API not implemented",
		Type:    "token_factory_error",
		Param:   "",
		Code:    "api_not_implemented",
	}
	c.JSON(http.StatusNotImplemented, gin.H{
		"error": err,
	})
}

func RelayNotFound(c *gin.Context) {
	err := types.OpenAIError{
		Message: fmt.Sprintf("Invalid URL (%s %s)", c.Request.Method, c.Request.URL.Path),
		Type:    "invalid_request_error",
		Param:   "",
		Code:    "",
	}
	c.JSON(http.StatusNotFound, gin.H{
		"error": err,
	})
}

func RelayTaskFetch(c *gin.Context) {
	relayInfo, err := relaycommon.GenRelayInfo(c, types.RelayFormatTask, nil, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, &dto.TaskError{
			Code:       "gen_relay_info_failed",
			Message:    err.Error(),
			StatusCode: http.StatusInternalServerError,
		})
		return
	}
	if taskErr := relay.RelayTaskFetch(c, relayInfo.RelayMode); taskErr != nil {
		respondTaskError(c, taskErr)
	}
}

func RelayTask(c *gin.Context) {
	relayInfo, err := relaycommon.GenRelayInfo(c, types.RelayFormatTask, nil, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, &dto.TaskError{
			Code:       "gen_relay_info_failed",
			Message:    err.Error(),
			StatusCode: http.StatusInternalServerError,
		})
		return
	}

	if taskErr := relay.ResolveOriginTask(c, relayInfo); taskErr != nil {
		respondTaskError(c, taskErr)
		return
	}

	var result *relay.TaskSubmitResult
	var taskErr *dto.TaskError
	defer func() {
		if taskErr != nil && relayInfo.Billing != nil {
			relayInfo.Billing.Refund(c)
		}
	}()

	retryParam := &service.RetryParam{
		Ctx:        c,
		TokenGroup: relayInfo.TokenGroup,
		ModelName:  relayInfo.OriginModelName,
		Retry:      common.GetPointer(0),
	}

	maxRelayRetries := service.RouteOrderedMaxRetries(c)
	for ; retryParam.GetRetry() <= maxRelayRetries; retryParam.IncreaseRetry() {
		var channel *model.Channel

		if lockedCh, ok := relayInfo.LockedChannel.(*model.Channel); ok && lockedCh != nil {
			channel = lockedCh
			if retryParam.GetRetry() > 0 {
				if setupErr := middleware.SetupContextForSelectedChannel(c, channel, relayInfo.OriginModelName); setupErr != nil {
					taskErr = service.TaskErrorWrapperLocal(setupErr.Err, "setup_locked_channel_failed", http.StatusInternalServerError)
					break
				}
			}
		} else {
			var channelErr *types.TokenFactoryError
			channel, channelErr = getChannel(c, relayInfo, retryParam)
			if channelErr != nil {
				logger.LogError(c, channelErr.Error())
				taskErr = service.TaskErrorWrapperLocal(channelErr.Err, "get_channel_failed", http.StatusInternalServerError)
				break
			}
		}

		addUsedChannel(c, channel.Id)
		bodyStorage, bodyErr := common.GetBodyStorage(c)
		if bodyErr != nil {
			if common.IsRequestBodyTooLargeError(bodyErr) || errors.Is(bodyErr, common.ErrRequestBodyTooLarge) {
				taskErr = service.TaskErrorWrapperLocal(bodyErr, "read_request_body_failed", http.StatusRequestEntityTooLarge)
			} else {
				taskErr = service.TaskErrorWrapperLocal(bodyErr, "read_request_body_failed", http.StatusBadRequest)
			}
			break
		}
		c.Request.Body = io.NopCloser(bodyStorage)

		result, taskErr = relay.RelayTaskSubmit(c, relayInfo)
		if taskErr == nil {
			break
		}

		if !taskErr.LocalError {
			processChannelError(c,
				*types.NewChannelError(channel.Id, channel.Type, channel.Name, channel.ChannelInfo.IsMultiKey,
					common.GetContextKeyString(c, constant.ContextKeyChannelKey), channel.GetAutoBan()),
				types.NewOpenAIError(taskErr.Error, types.ErrorCodeBadResponseStatusCode, taskErr.StatusCode))
		}

		if !shouldRetryTaskRelay(c, channel.Id, taskErr, common.RetryTimes-retryParam.GetRetry()) {
			break
		}
	}

	useChannel := c.GetStringSlice("use_channel")
	if len(useChannel) > 1 {
		retryLogStr := fmt.Sprintf("重试：%s", strings.Trim(strings.Join(strings.Fields(fmt.Sprint(useChannel)), "->"), "[]"))
		logger.LogInfo(c, retryLogStr)
	}

	// ── 成功：结算 + 日志 + 插入任务 ──
	if taskErr == nil {
		actualQuota := service.ResolveActualTaskQuotaOnSubmit(c, relayInfo, result.TaskData, result.Quota)
		result.Quota = actualQuota
		relayInfo.PriceData.Quota = actualQuota
		if settleErr := service.SettleBilling(c, relayInfo, actualQuota); settleErr != nil {
			common.SysError("settle task billing error: " + settleErr.Error())
		}
		service.LogTaskConsumption(c, relayInfo)

		task := model.InitTask(result.Platform, relayInfo)
		// 优先使用渠道写入的完整原始请求体（如 Seedance 透传），便于线上排查。
		if raw, ok := relaycommon.GetTaskPersistedInput(c); ok {
			task.Properties.Input = raw
		} else if req, err := relaycommon.GetTaskRequest(c); err == nil {
			if reqBytes, mErr := common.Marshal(req); mErr == nil {
				task.Properties.Input = string(reqBytes)
			}
		}
		task.PrivateData.UpstreamTaskID = result.UpstreamTaskID
		task.PrivateData.TfOpenVideoUpstreamStyle = relayInfo.TfOpenVideoUpstreamStyle
		if k := strings.TrimSpace(relayInfo.ApiKey); k != "" {
			// 轮询上游（如腾讯云 DescribeTaskDetail）时使用与提交相同的密钥，避免多 Key 渠道错钥
			task.PrivateData.Key = k
		}
		task.PrivateData.BillingSource = relayInfo.BillingSource
		task.PrivateData.SubscriptionId = relayInfo.SubscriptionId
		if relayInfo.BillingSource == service.BillingSourceWallet {
			paidQuota := actualQuota - relayInfo.WalletGiftConsumed
			if paidQuota > 0 {
				task.PrivateData.WalletPaidQuota = paidQuota
			}
		}
		task.PrivateData.TokenId = relayInfo.TokenId
		task.PrivateData.TokenName = c.GetString("token_name")
		chDiscPct := model.ResolveChannelEffectiveCostPercent(relayInfo.ChannelId)
		if relayInfo.PriceData.ChannelPriceDiscount != nil {
			chDiscPct = *relayInfo.PriceData.ChannelPriceDiscount
		}
		markupDiscPct := relayInfo.PriceData.MarkupDiscountPercent
		rawPriceDiscountPct := relayInfo.PriceData.RawPriceDiscountPercent
		operatingCostPct := relayInfo.PriceData.OperatingCostPercent
		effectiveCostPct := chDiscPct
		task.PrivateData.BillingContext = &model.TaskBillingContext{
			ModelPrice:                  relayInfo.PriceData.ModelPrice,
			GroupRatio:                  relayInfo.PriceData.GroupRatioInfo.GroupRatio,
			ModelRatio:                  relayInfo.PriceData.ModelRatio,
			OtherRatios:                 relayInfo.PriceData.OtherRatios,
			OriginModelName:             relayInfo.OriginModelName,
			PerCallBilling:              common.StringsContains(constant.TaskPricePatches, relayInfo.OriginModelName),
			ChannelPriceDiscountPercent: chDiscPct,
			MarkupDiscountPercent:       &markupDiscPct,
			PriceDiscountPercent:        &rawPriceDiscountPct,
			OperatingCostPercent:        &operatingCostPct,
			EffectiveCostPercent:        &effectiveCostPct,
			VideoRuleUnit:               relayInfo.PriceData.VideoRuleUnit,
			VideoBillingMode:            relayInfo.PriceData.VideoBillingMode,
			VideoChannelRulePrice:       relayInfo.PriceData.VideoChannelRulePrice,
			VideoGlobalRulePrice:        relayInfo.PriceData.VideoGlobalRulePrice,
			VideoRuleWidth:              relayInfo.PriceData.VideoRuleWidth,
			VideoRuleHeight:             relayInfo.PriceData.VideoRuleHeight,
			VideoRuleHasAudio:           relayInfo.PriceData.VideoRuleHasAudio,
			UpstreamBillingOther:        relayInfo.UpstreamTaskBillingOther,
		}
		service.AttachVideoUpscaleToTask(c, task)
		task.Quota = actualQuota
		task.Data = result.TaskData
		task.Action = relayInfo.Action
		if insertErr := task.Insert(); insertErr != nil {
			common.SysError("insert task error: " + insertErr.Error())
		}
	}

	if taskErr != nil {
		respondTaskError(c, taskErr)
	}
}

// respondTaskError 统一输出 Task 错误响应（含 429 限流提示改写）
func respondTaskError(c *gin.Context, taskErr *dto.TaskError) {
	if taskErr.StatusCode == http.StatusTooManyRequests {
		taskErr.Message = "当前分组上游负载已饱和，请稍后再试"
	}
	c.JSON(taskErr.StatusCode, taskErr)
}

func shouldRetryTaskRelay(c *gin.Context, channelId int, taskErr *dto.TaskError, retryTimes int) bool {
	if taskErr == nil {
		return false
	}
	if service.ShouldSkipRetryAfterChannelAffinityFailure(c) {
		return false
	}
	if _, ok := c.Get("specific_channel_id"); ok {
		return false
	}
	if _, ok := common.GetContextKey(c, constant.ContextKeyTokenSpecificChannelId); ok {
		return false
	}
	if _, ok := common.GetContextKey(c, constant.ContextKeyForcedChannelID); ok {
		return false
	}
	if service.ContextNoFailover(c) {
		return false
	}
	if taskErr.LocalError {
		return false
	}
	// 本地/业务明确的不可切换错误码：换渠也解决不了（参数、审核等）。
	switch strings.ToLower(strings.TrimSpace(taskErr.Code)) {
	case "invalid_request", "invalid_model", "sensitive_words_detected",
		"missing_output_config", "invalid_output_config", "invalid_tencent_vod_params",
		"invalid_tencent_vod_body", "invalid_credentials", "read_request_body_failed",
		"read_body_failed":
		return false
	}
	code := taskErr.StatusCode
	if code >= 200 && code < 300 {
		return false
	}
	// 有序路由尚有未尝试渠道时：对 4xx/5xx 继续保底切换（与聊天中继对齐，且不受 RetryTimes 剩余次数卡死）。
	if code >= 400 && code <= 599 && service.HasUnusedOrderedRouteChannel(c) {
		return true
	}
	if retryTimes <= 0 {
		return false
	}
	// 与聊天中继一致：默认对完整 4xx/5xx 切渠道，优先可用性。
	if code >= 400 && code <= 599 {
		return operation_setting.ShouldRetryByStatusCode(code)
	}
	if code == 307 || code == http.StatusTooManyRequests {
		return true
	}
	if code < 100 || code > 599 {
		return true
	}
	return operation_setting.ShouldRetryByStatusCode(code)
}
