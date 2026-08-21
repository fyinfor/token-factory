package controller

import (
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
)

// ── 用户智能路由策略 HTTP API（进程内本地实现）──────────────────
//
// 读写本地 route_* / model_group_* / user_* 表，不再代理 TokenFactory gRPC。
// TOKENFACTORY_ROUTE_ENABLED=false 时返回 404，前端据此隐藏相关 UI。

func rejectTokenFactoryRouteDisabled(c *gin.Context) bool {
	if common.TokenFactoryRouteEnabled() {
		return false
	}
	c.JSON(http.StatusNotFound, gin.H{
		"success":         false,
		"error":           "smart routing is not enabled on this site",
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

	isAdmin := userRole >= common.RoleAdminUser
	policy, err := service.GetLocalUserRoutePolicy(userID, isAdmin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "load route policy: " + err.Error()})
		return
	}

	groups := make([]gin.H, 0, len(policy.Groups))
	for _, g := range policy.Groups {
		channels := make([]gin.H, 0, len(g.Channels))
		for _, ch := range g.Channels {
			channels = append(channels, gin.H{
				"channel_id":        ch.ChannelID,
				"route_slug":        ch.RouteSlug,
				"name":              ch.Name,
				"masked_name":       ch.MaskedName,
				"provider_slug":     ch.ProviderSlug,
				"supplier_alias":    ch.SupplierAlias,
				"status":            ch.Status,
				"models_in_group":   ch.ModelsInGroup,
				"user_weight":       ch.UserWeight,
				"user_weight_id":    ch.UserWeightID,
				"user_enabled":      ch.UserEnabled,
				"user_configured":   ch.UserConfigured,
				"global_weight":     ch.GlobalWeight,
				"global_enabled":    ch.GlobalEnabled,
				"global_configured": ch.GlobalConfigured,
				"price":             ch.Price,
			})
		}
		groups = append(groups, gin.H{
			"group_key":      g.GroupKey,
			"display_name":   g.DisplayName,
			"models":         g.Models,
			"channel_count":  g.ChannelCount,
			"channels":       channels,
			"route_disabled": g.RouteDisabled,
		})
	}

	userOverrides := make([]gin.H, 0, len(policy.UserOverrides))
	for _, o := range policy.UserOverrides {
		userOverrides = append(userOverrides, gin.H{
			"id": o.ID, "raw_model": o.RawModel, "group_key": o.GroupKey, "is_user": o.IsUser,
		})
	}
	globalOverrides := make([]gin.H, 0, len(policy.GlobalOverrides))
	for _, o := range policy.GlobalOverrides {
		globalOverrides = append(globalOverrides, gin.H{
			"id": o.ID, "raw_model": o.RawModel, "group_key": o.GroupKey, "is_user": o.IsUser,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success":          true,
		"mode":             policy.Mode,
		"global_mode":      policy.GlobalMode,
		"groups":           groups,
		"user_overrides":   userOverrides,
		"global_overrides": globalOverrides,
	})
}

// UserUpdateRouteMode 更新当前用户的路由模式。
func UserUpdateRouteMode(c *gin.Context) {
	if rejectTokenFactoryRouteDisabled(c) {
		return
	}
	userID := c.GetInt("id")
	if userID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		Mode      string `json:"mode"`
		ResetMode bool   `json:"reset_mode"`
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
	if req.Mode == "" && !req.ResetMode {
		c.JSON(http.StatusBadRequest, gin.H{"error": "mode or reset_mode is required"})
		return
	}

	if req.ResetMode {
		if err := model.DeleteUserRouteConfig(userID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "mode reset"})
		return
	}

	if _, err := model.SaveUserRouteConfig(userID, req.Mode); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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

	// 用户指定价约束：不可配置未放行渠道的权重。
	if !service.UserRouteChannelVisibleInGroup(userID, req.GroupKey, req.ChannelID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该渠道不在您的可用渠道范围内"})
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	if _, err := model.UpsertUserModelGroupWeight(userID, req.GroupKey, req.ChannelID, req.Weight, enabled); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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

	if err := model.DeleteUserModelGroupWeightByID(uint(id), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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

	if _, err := model.UpsertUserModelGroupOverride(userID, req.RawModel, req.GroupKey); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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

	if err := model.DeleteUserModelGroupOverrideByID(uint(id), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "override deleted"})
}

// UserUpdateGroupRoute 更新某归类是否关闭智能路由。
func UserUpdateGroupRoute(c *gin.Context) {
	if rejectTokenFactoryRouteDisabled(c) {
		return
	}
	userID := c.GetInt("id")
	if userID <= 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		GroupKey string `json:"group_key"`
		Disabled *bool  `json:"disabled"`
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
	if req.GroupKey == "" || req.Disabled == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "group_key and disabled are required"})
		return
	}

	if err := model.SetUserModelGroupRouteDisabled(userID, req.GroupKey, *req.Disabled); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "group route updated", "disabled": *req.Disabled})
}
