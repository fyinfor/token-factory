package controller

import (
	"errors"
	"io"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

// HandleMaterialAction 终端用户素材库 Action 网关入口。
// 路由：POST /api/material?Action={ActionName}
// 鉴权：Bearer Token（sk-xxx），与 Web 控制台 REST 路由隔离。
func HandleMaterialAction(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		common.MaterialActionError(c, common.MaterialCodeUnauthorized, "未授权，请携带有效的 API Token")
		return
	}

	action := service.NormalizeMaterialAction(c.Query("Action"))

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		common.MaterialActionError(c, common.MaterialCodeInvalidParameter, "读取请求体失败")
		return
	}
	if action == "" {
		action = extractActionFromBody(body)
	}
	if action == "" {
		common.MaterialActionError(c, common.MaterialCodeInvalidParameter, "Action 参数不能为空")
		return
	}

	data, err := service.DispatchMaterialAction(userId, action, body)
	if err != nil {
		bizErr := asMaterialActionError(err)
		common.MaterialActionError(c, bizErr.Code, bizErr.Message)
		return
	}
	common.MaterialActionSuccess(c, data)
}

// extractActionFromBody 从 JSON Body 中提取 Action 字段（Query 缺失时的兜底）。
func extractActionFromBody(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	var payload struct {
		Action string `json:"Action"`
	}
	if err := common.Unmarshal(body, &payload); err != nil {
		return ""
	}
	return service.NormalizeMaterialAction(payload.Action)
}

func asMaterialActionError(err error) *service.MaterialActionError {
	if err == nil {
		return nil
	}
	var bizErr *service.MaterialActionError
	if errors.As(err, &bizErr) {
		return bizErr
	}
	return &service.MaterialActionError{Code: common.MaterialCodeInternalError, Message: err.Error()}
}
