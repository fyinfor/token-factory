package controller

import (
	"errors"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/relay"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

// RelayASRTaskSubmit POST /v1/audio/transcriptions/async
// 提交阿里云 ASR 异步转写任务：渠道分发沿用 Distribute 中间件，此处复用首个选中渠道，
// 不做渠道重试以避免重复创建上游任务；提交预扣 60 秒费用，成功取结果后按 usage.duration 补差价。
func RelayASRTaskSubmit(c *gin.Context) {
	requestId := c.GetString(common.RequestIdKey)
	var tokenFactoryError *types.TokenFactoryError
	defer func() {
		if tokenFactoryError != nil {
			logger.LogError(c, "asr task submit error: "+tokenFactoryError.Error())
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

	tokenFactoryError = relay.SubmitASRTask(c, relayInfo, submitReq, channelModel.GetSetting().Proxy)
}

// RelayASRTaskFetch GET /v1/audio/transcriptions/async/:task_id
// 查询异步转写任务结果。该路由不经过渠道分发（无请求体模型名），任务归属按登录用户校验。
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
