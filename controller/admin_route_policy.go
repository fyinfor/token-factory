package controller

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
)

// ── 管理员：全局路由模板 + 指定用户路由策略 ──────────────────────

func rejectAdminRouteDisabled(c *gin.Context) bool {
	return rejectTokenFactoryRouteDisabled(c)
}

func writeLocalUserRoutePolicyJSON(c *gin.Context, policy *service.LocalUserRoutePolicy) {
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
				"user_discount":     ch.UserDiscount,
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

// AdminGetRouteConfig 返回全局路由配置（模板默认模式）。
func AdminGetRouteConfig(c *gin.Context) {
	if rejectAdminRouteDisabled(c) {
		return
	}
	cfg := model.GetRouteConfig()
	c.JSON(http.StatusOK, gin.H{"success": true, "config": cfg})
}

// AdminUpdateRouteConfig 更新全局路由模式（未单独配置用户跟随此模板）。
func AdminUpdateRouteConfig(c *gin.Context) {
	if rejectAdminRouteDisabled(c) {
		return
	}
	var req struct {
		Mode           string `json:"mode"`
		PricePerfRatio *int   `json:"price_perf_ratio"`
	}
	bodyBytes, readErr := io.ReadAll(c.Request.Body)
	if readErr != nil && readErr != io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "read body: " + readErr.Error()})
		return
	}
	if len(bodyBytes) > 0 {
		if err := common.Unmarshal(bodyBytes, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid json: " + err.Error()})
			return
		}
	}
	mode := strings.TrimSpace(req.Mode)
	switch mode {
	case model.RouteModeDefault, model.RouteModeWeight, model.RouteModePrice,
		model.RouteModePerformance, model.RouteModePricePerf:
	default:
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid mode: " + mode})
		return
	}
	ratio := model.GetRouteConfig().PricePerfRatio
	if req.PricePerfRatio != nil {
		ratio = *req.PricePerfRatio
		if ratio < 0 {
			ratio = 0
		}
		if ratio > 100 {
			ratio = 100
		}
	}
	cfg, err := model.SaveRouteConfig(mode, ratio)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "config": cfg})
}

// AdminGetRoutePolicyTemplate 返回全局模板视图（不含用户级覆盖权重）。
func AdminGetRoutePolicyTemplate(c *gin.Context) {
	if rejectAdminRouteDisabled(c) {
		return
	}
	policy, err := service.GetLocalUserRoutePolicy(0, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "load route policy: " + err.Error()})
		return
	}
	weightIDByKey := map[string]uint{}
	if rows, _ := model.LoadAllModelGroupWeights(); len(rows) > 0 {
		for _, w := range rows {
			weightIDByKey[w.GroupKey+"\x00"+strconv.Itoa(w.ChannelID)] = w.ID
		}
	}
	// 模板页把全局权重映射为可编辑的「生效权重」字段，便于复用前端卡片。
	for i := range policy.Groups {
		for j := range policy.Groups[i].Channels {
			ch := &policy.Groups[i].Channels[j]
			ch.UserWeight = ch.GlobalWeight
			ch.UserEnabled = ch.GlobalEnabled
			ch.UserConfigured = ch.GlobalConfigured
			ch.UserWeightID = weightIDByKey[policy.Groups[i].GroupKey+"\x00"+strconv.Itoa(ch.ChannelID)]
		}
	}
	policy.Mode = policy.GlobalMode
	writeLocalUserRoutePolicyJSON(c, policy)
}

// AdminUpsertTemplateWeight 写入全局归类权重模板。
func AdminUpsertTemplateWeight(c *gin.Context) {
	if rejectAdminRouteDisabled(c) {
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
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "read body: " + readErr.Error()})
		return
	}
	if len(bodyBytes) > 0 {
		if err := common.Unmarshal(bodyBytes, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid json: " + err.Error()})
			return
		}
	}
	if req.GroupKey == "" || req.ChannelID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "group_key and channel_id are required"})
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	if _, err := model.UpsertModelGroupWeight(req.GroupKey, req.ChannelID, req.Weight, enabled); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "weight updated"})
}

// AdminDeleteTemplateWeight 删除全局归类权重。
func AdminDeleteTemplateWeight(c *gin.Context) {
	if rejectAdminRouteDisabled(c) {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid id"})
		return
	}
	if err := model.DeleteModelGroupWeightByID(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "weight deleted"})
}

// AdminUpsertTemplateOverride 写入全局模型归类覆盖。
func AdminUpsertTemplateOverride(c *gin.Context) {
	if rejectAdminRouteDisabled(c) {
		return
	}
	var req struct {
		RawModel string `json:"raw_model"`
		GroupKey string `json:"group_key"`
	}
	bodyBytes, readErr := io.ReadAll(c.Request.Body)
	if readErr != nil && readErr != io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "read body: " + readErr.Error()})
		return
	}
	if len(bodyBytes) > 0 {
		if err := common.Unmarshal(bodyBytes, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid json: " + err.Error()})
			return
		}
	}
	if _, err := model.UpsertModelGroupOverride(req.RawModel, req.GroupKey); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "override updated"})
}

// AdminDeleteTemplateOverride 删除全局模型归类覆盖。
func AdminDeleteTemplateOverride(c *gin.Context) {
	if rejectAdminRouteDisabled(c) {
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid id"})
		return
	}
	if err := model.DeleteModelGroupOverrideByID(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "override deleted"})
}

func parseAdminTargetUserID(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid user id"})
		return 0, false
	}
	if _, err := model.GetUserById(id, false); err != nil {
		// 区分软删除与真正不存在，避免搜索结果含已删用户时只回笼统 404。
		var deleted model.User
		if e := model.DB.Unscoped().Select("id", "username", "deleted_at").First(&deleted, "id = ?", id).Error; e == nil && deleted.DeletedAt.Valid {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": fmt.Sprintf("用户 #%d（%s）已删除", id, deleted.Username)})
			return 0, false
		}
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "user not found"})
		return 0, false
	}
	return id, true
}

// AdminGetUserRoutePolicy 查看指定用户的路由策略。
func AdminGetUserRoutePolicy(c *gin.Context) {
	if rejectAdminRouteDisabled(c) {
		return
	}
	userID, ok := parseAdminTargetUserID(c)
	if !ok {
		return
	}
	policy, err := service.GetLocalUserRoutePolicy(userID, true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "load route policy: " + err.Error()})
		return
	}
	writeLocalUserRoutePolicyJSON(c, policy)
}

// AdminUpdateUserRouteMode 更新指定用户的路由模式。
func AdminUpdateUserRouteMode(c *gin.Context) {
	if rejectAdminRouteDisabled(c) {
		return
	}
	userID, ok := parseAdminTargetUserID(c)
	if !ok {
		return
	}
	var req struct {
		Mode      string `json:"mode"`
		ResetMode bool   `json:"reset_mode"`
	}
	bodyBytes, readErr := io.ReadAll(c.Request.Body)
	if readErr != nil && readErr != io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "read body: " + readErr.Error()})
		return
	}
	if len(bodyBytes) > 0 {
		if err := common.Unmarshal(bodyBytes, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid json: " + err.Error()})
			return
		}
	}
	if req.ResetMode {
		if err := model.DeleteUserRouteConfig(userID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "mode reset"})
		return
	}
	if req.Mode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "mode or reset_mode is required"})
		return
	}
	if _, err := model.SaveUserRouteConfig(userID, req.Mode); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "mode updated"})
}

// AdminUpdateUserGroupRoute 更新指定用户某归类是否关闭智能路由。
func AdminUpdateUserGroupRoute(c *gin.Context) {
	if rejectAdminRouteDisabled(c) {
		return
	}
	userID, ok := parseAdminTargetUserID(c)
	if !ok {
		return
	}
	var req struct {
		GroupKey string `json:"group_key"`
		Disabled *bool  `json:"disabled"`
	}
	bodyBytes, readErr := io.ReadAll(c.Request.Body)
	if readErr != nil && readErr != io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "read body: " + readErr.Error()})
		return
	}
	if len(bodyBytes) > 0 {
		if err := common.Unmarshal(bodyBytes, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid json: " + err.Error()})
			return
		}
	}
	if req.GroupKey == "" || req.Disabled == nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "group_key and disabled are required"})
		return
	}
	if err := model.SetUserModelGroupRouteDisabled(userID, req.GroupKey, *req.Disabled); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "group route updated", "disabled": *req.Disabled})
}

// AdminUpsertUserRouteWeight 写入指定用户的归类权重。
func AdminUpsertUserRouteWeight(c *gin.Context) {
	if rejectAdminRouteDisabled(c) {
		return
	}
	userID, ok := parseAdminTargetUserID(c)
	if !ok {
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
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "read body: " + readErr.Error()})
		return
	}
	if len(bodyBytes) > 0 {
		if err := common.Unmarshal(bodyBytes, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid json: " + err.Error()})
			return
		}
	}
	if req.GroupKey == "" || req.ChannelID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "group_key and channel_id are required"})
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	if _, err := model.UpsertUserModelGroupWeight(userID, req.GroupKey, req.ChannelID, req.Weight, enabled); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "weight updated"})
}

// AdminDeleteUserRouteWeight 删除指定用户的归类权重。
func AdminDeleteUserRouteWeight(c *gin.Context) {
	if rejectAdminRouteDisabled(c) {
		return
	}
	userID, ok := parseAdminTargetUserID(c)
	if !ok {
		return
	}
	wid, err := strconv.ParseUint(c.Param("wid"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid weight id"})
		return
	}
	if err := model.DeleteUserModelGroupWeightByID(uint(wid), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "weight deleted"})
}

// AdminUpsertUserRouteOverride 写入指定用户的模型归类覆盖。
func AdminUpsertUserRouteOverride(c *gin.Context) {
	if rejectAdminRouteDisabled(c) {
		return
	}
	userID, ok := parseAdminTargetUserID(c)
	if !ok {
		return
	}
	var req struct {
		RawModel string `json:"raw_model"`
		GroupKey string `json:"group_key"`
	}
	bodyBytes, readErr := io.ReadAll(c.Request.Body)
	if readErr != nil && readErr != io.EOF {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "read body: " + readErr.Error()})
		return
	}
	if len(bodyBytes) > 0 {
		if err := common.Unmarshal(bodyBytes, &req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid json: " + err.Error()})
			return
		}
	}
	if _, err := model.UpsertUserModelGroupOverride(userID, req.RawModel, req.GroupKey); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "override updated"})
}

// AdminDeleteUserRouteOverride 删除指定用户的模型归类覆盖。
func AdminDeleteUserRouteOverride(c *gin.Context) {
	if rejectAdminRouteDisabled(c) {
		return
	}
	userID, ok := parseAdminTargetUserID(c)
	if !ok {
		return
	}
	oid, err := strconv.ParseUint(c.Param("oid"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid override id"})
		return
	}
	if err := model.DeleteUserModelGroupOverrideByID(uint(oid), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "override deleted"})
}
