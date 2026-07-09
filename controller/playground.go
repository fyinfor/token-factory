package controller

import (
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

func setupPlaygroundTokenContext(c *gin.Context, relayFormat types.RelayFormat) (*types.TokenFactoryError, *relaycommon.RelayInfo) {
	useAccessToken := c.GetBool("use_access_token")
	if useAccessToken {
		return types.NewError(errors.New("暂不支持使用 access token"), types.ErrorCodeAccessDenied, types.ErrOptionWithSkipRetry()), nil
	}

	relayInfo, err := relaycommon.GenRelayInfo(c, relayFormat, nil, nil)
	if err != nil {
		return types.NewError(err, types.ErrorCodeInvalidRequest, types.ErrOptionWithSkipRetry()), nil
	}

	userId := c.GetInt("id")

	// Write user context to ensure acceptUnsetRatio is available
	userCache, err := model.GetUserCache(userId)
	if err != nil {
		return types.NewError(err, types.ErrorCodeQueryDataError, types.ErrOptionWithSkipRetry()), nil
	}
	userCache.WriteContext(c)

	tempToken := &model.Token{
		UserId: userId,
		Name:   fmt.Sprintf("playground-%s", relayInfo.UsingGroup),
		Group:  relayInfo.UsingGroup,
	}
	_ = middleware.SetupContextForToken(c, tempToken)
	return nil, relayInfo
}

func Playground(c *gin.Context) {
	var tokenFactoryError *types.TokenFactoryError

	defer func() {
		if tokenFactoryError != nil {
			c.JSON(tokenFactoryError.StatusCode, gin.H{
				"error": tokenFactoryError.ToOpenAIError(),
			})
		}
	}()

	tokenFactoryError, _ = setupPlaygroundTokenContext(c, types.RelayFormatOpenAI)
	if tokenFactoryError != nil {
		return
	}

	Relay(c, types.RelayFormatOpenAI)
}

func playgroundImageRelay(c *gin.Context, relayMode int) {
	var tokenFactoryError *types.TokenFactoryError
	defer func() {
		if tokenFactoryError != nil {
			c.JSON(tokenFactoryError.StatusCode, gin.H{
				"error": tokenFactoryError.ToOpenAIError(),
			})
		}
	}()
	tokenFactoryError, _ = setupPlaygroundTokenContext(c, types.RelayFormatOpenAIImage)
	if tokenFactoryError != nil {
		return
	}
	// 兜底：操练场路径无法被 Path2RelayMode 识别，需显式设置 relay_mode
	c.Set("relay_mode", relayMode)
	Relay(c, types.RelayFormatOpenAIImage)
}

// PlaygroundImage 文生图：无参考图时走 /v1/images/generations
func PlaygroundImage(c *gin.Context) {
	playgroundImageRelay(c, relayconstant.RelayModeImagesGenerations)
}

// PlaygroundImageEdits 图生图：有参考图时走 /v1/images/edits
func PlaygroundImageEdits(c *gin.Context) {
	playgroundImageRelay(c, relayconstant.RelayModeImagesEdits)
}

func PlaygroundVideo(c *gin.Context) {
	var tokenFactoryError *types.TokenFactoryError
	defer func() {
		if tokenFactoryError != nil {
			c.JSON(tokenFactoryError.StatusCode, gin.H{
				"error": tokenFactoryError.ToOpenAIError(),
			})
		}
	}()
	tokenFactoryError, _ = setupPlaygroundTokenContext(c, types.RelayFormatTask)
	if tokenFactoryError != nil {
		return
	}
	RelayTask(c)
}

func PlaygroundVideoFetch(c *gin.Context) {
	var tokenFactoryError *types.TokenFactoryError
	defer func() {
		if tokenFactoryError != nil {
			c.JSON(tokenFactoryError.StatusCode, gin.H{
				"error": tokenFactoryError.ToOpenAIError(),
			})
		}
	}()
	tokenFactoryError, _ = setupPlaygroundTokenContext(c, types.RelayFormatTask)
	if tokenFactoryError != nil {
		return
	}
	RelayTaskFetch(c)
}

func PlaygroundImageFetch(c *gin.Context) {
	var tokenFactoryError *types.TokenFactoryError
	defer func() {
		if tokenFactoryError != nil {
			c.JSON(tokenFactoryError.StatusCode, gin.H{
				"error": tokenFactoryError.ToOpenAIError(),
			})
		}
	}()
	tokenFactoryError, _ = setupPlaygroundTokenContext(c, types.RelayFormatTask)
	if tokenFactoryError != nil {
		return
	}
	RelayTaskFetch(c)
}
