package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
)

// AdminTokenFactorySyncUsers 手动触发一次用户快照同步到 TokenFactory。
// 管理员在后台可调用此端点，把当前 token-factory 的用户数据通过 gRPC 推送到 TokenFactory 的 external_users 表。
// 鉴权：AdminAuth；JWT 用 ROOT_USER 兜底（uid=1, role=100）以满足 TokenFactory 端校验。
func AdminTokenFactorySyncUsers(c *gin.Context) {
	if !common.TokenFactoryRouteEnabled() {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "TokenFactory route not enabled",
		})
		return
	}

	pushed, total, err := service.SyncUsersToTokenFactory()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
			"pushed":  pushed,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"message":  "用户同步成功",
		"pushed":   pushed,
		"total":    total,
		"site_key": common.TokenFactorySiteKey(),
	})
}
