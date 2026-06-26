package controller

import (
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
)

// ── 用户智能路由策略 HTTP API ──────────────────────────────────
//
// 代理到 TokenFactory gRPC 服务，供 token-factory 用户控制台调用。
// 所有 API 从 JWT session 获取当前用户 ID 和角色。
// TOKENFACTORY_ROUTE_ENABLED=false 时返回 404，前端据此隐藏相关 UI。

func rejectTokenFactoryRouteDisabled(c *gin.Context) bool {
	if common.TokenFactoryRouteEnabled() {
		return false
	}
	c.JSON(http.StatusNotFound, gin.H{
		"success":         false,
		"error":           "TokenFactory smart routing is not enabled on this site",
		"feature_enabled": false,
	})
	return true
}

// UserGetRoutePolicy 获取当前用户的路由策略完整视图。
func UserGetRoutePolicy(c *gin.Context) {
	if rejectTokenFactoryRouteDisabled(c) {
		return
	}
	userID := c.GetInt("id")
	userRole := c.GetInt("role")
	if userID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	jwt, err := common.IssueTokenFactoryJWT(userID, userRole)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "issue JWT failed: " + err.Error()})
		return
	}

	policy, err := service.GetUserRoutePolicyFromTF(jwt, userID, userRole)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "TokenFactory gRPC error: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":         true,
		"mode":            policy.Mode,
		"global_mode":     policy.GlobalMode,
		"groups":          policy.Groups,
		"user_overrides":  policy.UserOverrides,
		"global_overrides": policy.GlobalOverrides,
	})
}

// UserUpdateRouteMode 更新当前用户的路由模式。
func UserUpdateRouteMode(c *gin.Context) {
	if rejectTokenFactoryRouteDisabled(c) {
		return
	}
	userID := c.GetInt("id")
	userRole := c.GetInt("role")
	if userID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		Mode      string `json:"mode"`
		ResetMode bool   `json:"reset_mode"`
	}
	// 手动读取 body 并解析，避免 gzip/MaxBytes 中间件包裹后 ShouldBindJSON 偶发 EOF。
	bodyBytes, readErr := io.ReadAll(c.Request.Body)
	if readErr != nil && readErr != io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{"error": "read body: " + readErr.Error()})
		return
	}
	if len(bodyBytes) > 0 {
		if err := common.Unmarshal(bodyBytes, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json: " + err.Error()})
			return
		}
	}

	// 至少要有一个明确意图，否则视为无效请求。
	if req.Mode == "" && !req.ResetMode {
		c.JSON(http.StatusBadRequest, gin.H{"error": "mode or reset_mode is required"})
		return
	}

	jwt, err := common.IssueTokenFactoryJWT(userID, userRole)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "issue JWT failed: " + err.Error()})
		return
	}

	if err := service.UpsertUserRouteModeToTF(jwt, userID, userRole, req.Mode, req.ResetMode); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "TokenFactory gRPC error: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "mode updated"})
}

// UserUpsertRouteWeight 创建或更新用户级归类权重。
func UserUpsertRouteWeight(c *gin.Context) {
	if rejectTokenFactoryRouteDisabled(c) {
		return
	}
	userID := c.GetInt("id")
	userRole := c.GetInt("role")
	if userID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		GroupKey  string `json:"group_key"`
		ChannelID int    `json:"channel_id"`
		Weight    int    `json:"weight"`
		Enabled   *bool  `json:"enabled"`
	}
	bodyBytes, readErr := io.ReadAll(c.Request.Body)
	if readErr != nil && readErr != io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{"error": "read body: " + readErr.Error()})
		return
	}
	if len(bodyBytes) > 0 {
		if err := common.Unmarshal(bodyBytes, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json: " + err.Error()})
			return
		}
	}
	if req.GroupKey == "" || req.ChannelID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "group_key and channel_id are required"})
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	jwt, err := common.IssueTokenFactoryJWT(userID, userRole)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "issue JWT failed: " + err.Error()})
		return
	}

	if err := service.UpsertUserWeightToTF(jwt, userID, userRole, req.GroupKey, req.ChannelID, req.Weight, enabled); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "TokenFactory gRPC error: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "weight updated"})
}

// UserDeleteRouteWeight 删除用户级归类权重。
func UserDeleteRouteWeight(c *gin.Context) {
	if rejectTokenFactoryRouteDisabled(c) {
		return
	}
	userID := c.GetInt("id")
	userRole := c.GetInt("role")
	if userID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	jwt, err := common.IssueTokenFactoryJWT(userID, userRole)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "issue JWT failed: " + err.Error()})
		return
	}

	if err := service.DeleteUserWeightFromTF(jwt, userID, userRole, uint32(id)); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "TokenFactory gRPC error: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "weight deleted"})
}

// UserUpsertRouteOverride 创建或更新用户级模型归类覆盖。
func UserUpsertRouteOverride(c *gin.Context) {
	if rejectTokenFactoryRouteDisabled(c) {
		return
	}
	userID := c.GetInt("id")
	userRole := c.GetInt("role")
	if userID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		RawModel string `json:"raw_model"`
		GroupKey string `json:"group_key"`
	}
	bodyBytes, readErr := io.ReadAll(c.Request.Body)
	if readErr != nil && readErr != io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{"error": "read body: " + readErr.Error()})
		return
	}
	if len(bodyBytes) > 0 {
		if err := common.Unmarshal(bodyBytes, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json: " + err.Error()})
			return
		}
	}
	if req.RawModel == "" || req.GroupKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "raw_model and group_key are required"})
		return
	}

	jwt, err := common.IssueTokenFactoryJWT(userID, userRole)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "issue JWT failed: " + err.Error()})
		return
	}

	if err := service.UpsertUserOverrideToTF(jwt, userID, userRole, req.RawModel, req.GroupKey); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "TokenFactory gRPC error: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "override updated"})
}

// UserDeleteRouteOverride 删除用户级模型归类覆盖。
func UserDeleteRouteOverride(c *gin.Context) {
	if rejectTokenFactoryRouteDisabled(c) {
		return
	}
	userID := c.GetInt("id")
	userRole := c.GetInt("role")
	if userID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	jwt, err := common.IssueTokenFactoryJWT(userID, userRole)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "issue JWT failed: " + err.Error()})
		return
	}

	if err := service.DeleteUserOverrideFromTF(jwt, userID, userRole, uint32(id)); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "TokenFactory gRPC error: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": "override deleted"})
}
