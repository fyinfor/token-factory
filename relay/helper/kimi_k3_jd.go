package helper

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/gin-gonic/gin"
)

// JD 采K3测试技术对接 - kimi‑k3专属特殊适配
// 本文件为 kimi-k3 文本模型独立校验，勿并入通用参数校验；后续可整文件删除。

const kimiK3TextModelName = "kimi-k3"

const kimiK3EmptyUserMessageFmt = "Invalid request: the message at position %d with role 'user' must not be empty"

const kimiK3EmptyUserErrorType = "invalid_request_error"

func kimiK3EmptyUserErrorMessage(position int) string {
	return fmt.Sprintf(kimiK3EmptyUserMessageFmt, position)
}

type kimiK3EmptyUserErrorResponse struct {
	Error kimiK3EmptyUserError `json:"error"`
}

type kimiK3EmptyUserError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

func isKimiK3TextModel(model string) bool {
	model = strings.TrimSpace(strings.ToLower(model))
	if model == "" {
		return false
	}
	if i := strings.LastIndex(model, "/"); i >= 0 {
		model = model[i+1:]
	}
	return model == kimiK3TextModelName
}

func kimiK3EmptyUserMessageIndex(req *dto.GeneralOpenAIRequest) (int, bool) {
	if req == nil {
		return -1, false
	}
	for i, msg := range req.Messages {
		if msg.Role != "user" {
			continue
		}
		content, ok := msg.Content.(string)
		if ok && content == "" {
			return i, true
		}
	}
	return -1, false
}

func shouldRejectKimiK3EmptyUser(c *gin.Context, request dto.Request) (int, bool) {
	textReq, ok := request.(*dto.GeneralOpenAIRequest)
	if !ok || textReq == nil {
		return -1, false
	}
	originModel := ""
	if c != nil {
		originModel = common.GetContextKeyString(c, constant.ContextKeyOriginalModel)
	}
	if !isKimiK3TextModel(textReq.Model) && !isKimiK3TextModel(originModel) {
		return -1, false
	}
	return kimiK3EmptyUserMessageIndex(textReq)
}

// AbortIfKimiK3EmptyUser 命中 kimi-k3 任意 user 空 content 时写入固定 400 并返回 true。
func AbortIfKimiK3EmptyUser(c *gin.Context, request dto.Request) bool {
	position, hit := shouldRejectKimiK3EmptyUser(c, request)
	if !hit {
		return false
	}
	c.AbortWithStatusJSON(http.StatusBadRequest, kimiK3EmptyUserErrorResponse{
		Error: kimiK3EmptyUserError{
			Message: kimiK3EmptyUserErrorMessage(position),
			Type:    kimiK3EmptyUserErrorType,
		},
	})
	return true
}
