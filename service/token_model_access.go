package service

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

const contextKeyIsAdmin = "tf_fetch_is_admin"

// TokenAllowsModel 判断当前请求令牌是否允许访问指定模型。
// 未启用模型限制时放行；启用时与生成接口使用同一套 ModelLimitMapAllows 匹配规则。
func TokenAllowsModel(c *gin.Context, modelName string) bool {
	if c == nil {
		return false
	}
	if !common.GetContextKeyBool(c, constant.ContextKeyTokenModelLimitEnabled) {
		return true
	}
	raw, ok := common.GetContextKey(c, constant.ContextKeyTokenModelLimit)
	if !ok {
		return false
	}
	limits, ok := raw.(map[string]bool)
	if !ok || len(limits) == 0 {
		return false
	}
	return model.ModelLimitMapAllows(limits, modelName)
}

// TokenModelForbiddenMessage 与分发中间件生成请求使用同一句拒绝文案。
func TokenModelForbiddenMessage(c *gin.Context, modelName string) string {
	return i18n.T(c, i18n.MsgDistributorTokenModelForbidden, map[string]any{"Model": modelName})
}

// ContextUserIsAdmin 判断当前请求用户是否为管理员（含 Root）。
// TokenAuth 路径未必写入 role（role=0），此时回退到按 user id 查库。
func ContextUserIsAdmin(c *gin.Context) bool {
	if c == nil {
		return false
	}
	if cached, exists := c.Get(contextKeyIsAdmin); exists {
		isAdmin, _ := cached.(bool)
		return isAdmin
	}
	role := c.GetInt("role")
	isAdmin := false
	switch {
	case role >= common.RoleAdminUser:
		isAdmin = true
	case role > common.RoleGuestUser:
		isAdmin = false
	default:
		isAdmin = model.IsAdmin(c.GetInt("id"))
	}
	c.Set(contextKeyIsAdmin, isAdmin)
	return isAdmin
}

// FetchOwnerUserID 异步查询的归属过滤用户 ID：管理员返回 0 表示不按所属人限制。
func FetchOwnerUserID(c *gin.Context) int {
	if c == nil {
		return -1
	}
	if ContextUserIsAdmin(c) {
		return 0
	}
	userID := c.GetInt("id")
	if userID <= 0 {
		return -1
	}
	return userID
}
