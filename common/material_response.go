package common

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// MaterialAction 网关标准错误码。
const (
	MaterialCodeSuccess           = 0
	MaterialCodeInvalidParameter  = 400
	MaterialCodeUnauthorized      = 401
	MaterialCodeForbidden         = 403
	MaterialCodeNotFound          = 404
	MaterialCodeInternalError     = 500
	MaterialCodeServiceUnavailable = 503
)

// MaterialActionResponse 终端用户素材库 Action 网关统一响应（code/msg/data）。
type MaterialActionResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data,omitempty"`
}

// MaterialActionSuccess 返回成功响应，HTTP 状态码固定 200。
func MaterialActionSuccess(c *gin.Context, data any) {
	c.JSON(http.StatusOK, MaterialActionResponse{
		Code: MaterialCodeSuccess,
		Msg:  "success",
		Data: data,
	})
}

// MaterialActionError 返回失败响应，HTTP 状态码固定 200。
func MaterialActionError(c *gin.Context, code int, msg string) {
	if msg == "" {
		msg = "请求失败"
	}
	c.JSON(http.StatusOK, MaterialActionResponse{
		Code: code,
		Msg:  msg,
	})
}
