package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	pb "github.com/QuantumNous/new-api/proto/route"
)

// AdminTokenFactorySyncUsers 手动触发一次用户快照同步到 TokenFactory。
// 管理员在后台可调用此端点，把当前 token-factory 的用户数据通过 gRPC 推送到 TokenFactory 的 external_users 表。
// 鉴权：AdminAuth；JWT 用 ROOT_USER 兜底（uid=1, role=100）以满足 TokenFactory 端校验。
func AdminTokenFactorySyncUsers(c *gin.Context) {
	// 1) 读取全部活跃用户（脱敏：不导出密码等）
	var users []model.User
	if err := model.DB.Select("id, username, display_name, role, status").
		Where("status = ?", common.UserStatusEnabled).
		Order("id ASC").
		Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "拉取用户失败: " + err.Error(),
		})
		return
	}

	// 2) 转换为 ExternalUserInfo
	infos := make([]*pb.ExternalUserInfo, 0, len(users))
	for _, u := range users {
		displayName := u.DisplayName
		if displayName == "" {
			displayName = u.Username
		}
		infos = append(infos, &pb.ExternalUserInfo{
			Uid:         int32(u.Id),
			Username:    u.Username,
			DisplayName: displayName,
			Role:        int32(u.Role),
			Status:      int32(u.Status),
		})
	}

	// 3) 用 ROOT_USER (uid=1) 签发 JWT
	jwt, err := common.IssueTokenFactoryJWT(1, 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "签发 JWT 失败: " + err.Error(),
		})
		return
	}

	// 4) 调用 gRPC 推送到 TokenFactory
	count, err := service.SyncUsersToTF(jwt, infos)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"success": false,
			"message": "gRPC 同步失败: " + err.Error(),
			"pushed":  count,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "用户同步成功",
		"pushed":  count,
		"total":   len(infos),
	})
}
