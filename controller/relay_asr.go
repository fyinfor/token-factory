package controller

import (
	"errors"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/relay"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

// RelayAudioTranscription POST /v1/audio/transcriptions
// 阿里云 ASR 异步渠道：内部提交上游异步任务并轮询，对外同步返回识别结果。
// 其它渠道（含阿里云 ASR 真同步、OpenAI Whisper 等）仍走原有 AudioHelper。
func RelayAudioTranscription(c *gin.Context) {
	if constant.IsASRAsyncChannel(common.GetContextKeyInt(c, constant.ContextKeyChannelType)) {
		RelayASRTaskSubmitWait(c)
		return
	}
	Relay(c, types.RelayFormatOpenAIAudio)
}

// RelayASRTaskSubmit POST /v1/audio/transcriptions/async
// 提交阿里云 ASR 异步转写任务：渠道分发沿用 Distribute 中间件，此处复用首个选中渠道，
// 不做渠道重试以避免重复创建上游任务；提交预扣 60 秒费用，成功取结果后按 usage.duration 补差价。
// 支持 JSON/multipart 的 audio_url，或 multipart file（先上传操练场附件库再以上游可拉取 URL 提交）。
func RelayASRTaskSubmit(c *gin.Context) {
	relayASRAsyncSubmit(c, false)
}

// RelayASRTaskSubmitWait POST /v1/audio/transcriptions（阿里云 ASR 异步渠道）
// 上游入参与异步提交一致；提交后由网关轮询任务状态，完成后同步返回识别文本。
func RelayASRTaskSubmitWait(c *gin.Context) {
	relayASRAsyncSubmit(c, true)
}

func relayASRAsyncSubmit(c *gin.Context, waitForResult bool) {
	requestId := c.GetString(common.RequestIdKey)
	var tokenFactoryError *types.TokenFactoryError
	defer func() {
		if tokenFactoryError != nil {
			logPrefix := "asr task submit error: "
			if waitForResult {
				logPrefix = "asr sync wait error: "
			}
			logger.LogError(c, logPrefix+tokenFactoryError.Error())
			tokenFactoryError.SetMessage(common.MessageWithRequestId(tokenFactoryError.Error(), requestId))
			c.JSON(tokenFactoryError.StatusCode, gin.H{
				"error": tokenFactoryError.ToOpenAIError(),
			})
		}
	}()

	// 支持 JSON body 与 multipart 表单两种提交方式
	submitReq := &dto.ASRTaskSubmitRequest{}
	if err := common.UnmarshalBodyReusable(c, submitReq); err != nil {
		tokenFactoryError = types.NewError(err, types.ErrorCodeInvalidRequest, types.ErrOptionWithSkipRetry())
		return
	}
	if strings.TrimSpace(submitReq.Model) == "" {
		tokenFactoryError = types.NewError(errors.New("model is required"), types.ErrorCodeInvalidRequest, types.ErrOptionWithSkipRetry())
		return
	}

	relayInfo, err := relaycommon.GenRelayInfo(c, types.RelayFormatOpenAIAudio, submitReq, nil)
	if err != nil {
		tokenFactoryError = types.NewError(err, types.ErrorCodeGenRelayInfoFailed)
		return
	}

	retryParam := &service.RetryParam{
		Ctx:        c,
		TokenGroup: relayInfo.TokenGroup,
		ModelName:  relayInfo.OriginModelName,
		Retry:      common.GetPointer(0),
	}
	channelModel, channelErr := getChannel(c, relayInfo, retryParam)
	if channelErr != nil {
		logger.LogError(c, channelErr.Error())
		tokenFactoryError = channelErr
		return
	}
	if relayInfo.ChannelMeta == nil {
		relayInfo.InitChannelMeta(c)
	}
	addUsedChannel(c, channelModel.Id)

	if waitForResult {
		tokenFactoryError = relay.SubmitASRTaskAndWait(c, relayInfo, submitReq, channelModel.GetSetting().Proxy)
	} else {
		tokenFactoryError = relay.SubmitASRTask(c, relayInfo, submitReq, channelModel.GetSetting().Proxy)
	}
	if tokenFactoryError != nil {
		skipChannelError := waitForResult &&
			(tokenFactoryError.StatusCode == http.StatusGatewayTimeout ||
				tokenFactoryError.GetErrorCode() == types.ErrorCodeBadResponse)
		if !skipChannelError {
			processChannelError(c,
				*types.NewChannelError(channelModel.Id, channelModel.Type, channelModel.Name, channelModel.ChannelInfo.IsMultiKey,
					common.GetContextKeyString(c, constant.ContextKeyChannelKey), channelModel.GetAutoBan()),
				tokenFactoryError)
		}
	}
}

// RelayASRTaskFetch GET /v1/audio/transcriptions/async/:task_id
// 查询异步转写任务结果。无请求体模型名，不在分发阶段做模型白名单拦截；
// 查到任务后按令牌模型限制列表校验，管理员可跨用户查询。
func RelayASRTaskFetch(c *gin.Context) {
	requestId := c.GetString(common.RequestIdKey)
	var tokenFactoryError *types.TokenFactoryError
	defer func() {
		if tokenFactoryError != nil {
			logger.LogError(c, "asr task fetch error: "+tokenFactoryError.Error())
			tokenFactoryError.SetMessage(common.MessageWithRequestId(tokenFactoryError.Error(), requestId))
			c.JSON(tokenFactoryError.StatusCode, gin.H{
				"error": tokenFactoryError.ToOpenAIError(),
			})
		}
	}()

	taskID := strings.TrimSpace(c.Param("task_id"))
	if taskID == "" {
		tokenFactoryError = types.NewErrorWithStatusCode(errors.New("task_id is required"), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
		return
	}

	tokenFactoryError = relay.FetchASRTaskResult(c, taskID)
}
