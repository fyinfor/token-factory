package controller

import (
	"log"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// AdminTokenFactorySyncChannels 手动触发一次渠道快照同步到 TokenFactory。
// 仅同步本站渠道（按 TOKENFACTORY_SITE_KEY 隔离），不含渠道密钥。
// TokenFactory 端仅更新 channel_snapshots，不会修改归类权重等路由配置。
func AdminTokenFactorySyncChannels(c *gin.Context) {
	if !common.TokenFactoryRouteEnabled() {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "TokenFactory route not enabled",
		})
		return
	}

	pushed, total, err := service.SyncChannelsToTokenFactory()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "channels synced to TokenFactory",
		"data": gin.H{
			"pushed":   pushed,
			"total":    total,
			"site_key": common.TokenFactorySiteKey(),
		},
	})
}
