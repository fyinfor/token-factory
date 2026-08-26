package controller

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

// ── 用户指定价管理（仅管理员）────────────────────────────────
//
// 对「用户 × 模型」维度单独管理成本折扣 / 经营成本 / 加价折扣三项，
// 命中后计费为「全局官方价 × 三折扣总和」。
// Mode=price_cap：选路排除单价超上限的渠道。
// Mode=channel_list：仅勾选渠道可用；展示与智能路由 UI 同步过滤。

type userModelPricingItem struct {
	model.UserModelPricingOverride
	Username     string                               `json:"username"`
	TotalPercent float64                              `json:"total_percent"`
	Channels     []model.UserModelPricingChannelBinding `json:"channels"`
}

// ListUserModelPricing GET /api/user_model_pricing?user_id=&model_name=
func ListUserModelPricing(c *gin.Context) {
	userId, _ := strconv.Atoi(c.Query("user_id"))
	modelName := c.Query("model_name")
	rows, err := model.ListUserModelPricingOverrides(userId, modelName)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	ids := make([]int, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.UserId)
	}
	usernames := model.GetUsernamesByIds(ids)

	channelsByUserModel := map[int]map[string][]model.UserModelPricingChannelBinding{}
	if userId > 0 {
		chMap, chErr := model.ListUserModelPricingChannelsByUser(userId)
		if chErr != nil {
			common.ApiError(c, chErr)
			return
		}
		channelsByUserModel[userId] = chMap
	} else {
		// 全量列表：按行懒加载渠道（管理端通常带 user_id）
		for _, r := range rows {
			if channelsByUserModel[r.UserId] != nil {
				continue
			}
			chMap, chErr := model.ListUserModelPricingChannelsByUser(r.UserId)
			if chErr != nil {
				common.ApiError(c, chErr)
				return
			}
			channelsByUserModel[r.UserId] = chMap
		}
	}

	items := make([]userModelPricingItem, 0, len(rows))
	for _, r := range rows {
		chs := channelsByUserModel[r.UserId][r.ModelName]
		if chs == nil {
			chs = []model.UserModelPricingChannelBinding{}
		}
		items = append(items, userModelPricingItem{
			UserModelPricingOverride: r,
			Username:                 usernames[r.UserId],
			TotalPercent:             r.TotalPercent(),
			Channels:                 chs,
		})
	}
	common.ApiSuccess(c, items)
}

type upsertUserModelPricingChannelReq struct {
	ChannelId int `json:"channel_id"`
	Priority  int `json:"priority"`
}

type upsertUserModelPricingReq struct {
	UserId               int                                `json:"user_id"`
	ModelName            string                             `json:"model_name"`
	Mode                 string                             `json:"mode"`
	PriceDiscountPercent float64                            `json:"price_discount_percent"`
	OperatingCostPercent float64                            `json:"operating_cost_percent"`
	MarkupDiscountRate   float64                            `json:"markup_discount_rate"`
	Enabled              bool                               `json:"enabled"`
	Channels             []upsertUserModelPricingChannelReq `json:"channels"`
}

func validatePricingPercent(v float64) bool {
	return v >= 0 && v <= 1000
}

// UpsertUserModelPricing POST /api/user_model_pricing
func UpsertUserModelPricing(c *gin.Context) {
	var req upsertUserModelPricingReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	req.ModelName = strings.TrimSpace(req.ModelName)
	if req.UserId <= 0 || req.ModelName == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "user_id 和 model_name 不能为空"})
		return
	}
	if !validatePricingPercent(req.PriceDiscountPercent) ||
		!validatePricingPercent(req.OperatingCostPercent) ||
		!validatePricingPercent(req.MarkupDiscountRate) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "折扣百分比需在 0-1000 之间"})
		return
	}
	mode := strings.TrimSpace(req.Mode)
	if mode == "" {
		mode = model.UserPricingModePriceCap
	}
	if mode != model.UserPricingModePriceCap && mode != model.UserPricingModeChannelList {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "mode 须为 price_cap 或 channel_list"})
		return
	}
	if _, err := model.GetUserById(req.UserId, false); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "用户不存在"})
		return
	}

	bindings := make([]model.UserModelPricingChannelBinding, 0, len(req.Channels))
	if mode == model.UserPricingModeChannelList {
		enabledIDs := model.GetEnabledChannelIDsByModel(req.ModelName)
		enabledSet := make(map[int]struct{}, len(enabledIDs))
		for _, id := range enabledIDs {
			enabledSet[id] = struct{}{}
		}
		for _, ch := range req.Channels {
			if ch.ChannelId <= 0 {
				continue
			}
			if _, ok := enabledSet[ch.ChannelId]; !ok {
				// 已关闭或不支持该模型的渠道：与预览一致，跳过而不是整单失败
				continue
			}
			cached, cacheErr := model.CacheGetChannel(ch.ChannelId)
			if cacheErr != nil || cached == nil || cached.Status != common.ChannelStatusEnabled {
				continue
			}
			bindings = append(bindings, model.UserModelPricingChannelBinding{
				ChannelId: ch.ChannelId,
				Priority:  ch.Priority,
			})
		}
		if len(bindings) == 0 {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "渠道清单模式至少勾选一个渠道"})
			return
		}
	}

	ov := &model.UserModelPricingOverride{
		UserId:               req.UserId,
		ModelName:            req.ModelName,
		Mode:                 mode,
		PriceDiscountPercent: req.PriceDiscountPercent,
		OperatingCostPercent: req.OperatingCostPercent,
		MarkupDiscountRate:   req.MarkupDiscountRate,
		Enabled:              req.Enabled,
	}
	saved, err := model.UpsertUserModelPricingOverrideWithChannels(ov, bindings)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	chs, _ := model.ListUserModelPricingChannels(saved.UserId, saved.ModelName)
	common.ApiSuccess(c, userModelPricingItem{
		UserModelPricingOverride: *saved,
		Username:                 "",
		TotalPercent:             saved.TotalPercent(),
		Channels:                 chs,
	})
}

// DeleteUserModelPricing DELETE /api/user_model_pricing/:id
func DeleteUserModelPricing(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.DeleteUserModelPricingOverrideById(id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

type userModelPricingPreviewChannel struct {
	ChannelId   int     `json:"channel_id"`
	ChannelName string  `json:"channel_name"`
	UnitPrice   float64 `json:"unit_price"`
	WithinCap   bool    `json:"within_cap"`
	Selected    bool    `json:"selected"`
	Priority    int     `json:"priority"`
}

// PreviewUserModelPricing GET /api/user_model_pricing/preview?...
// 按给定三折扣实时预览渠道；mode=channel_list 时附带勾选/排序参考。
func PreviewUserModelPricing(c *gin.Context) {
	modelName := strings.TrimSpace(c.Query("model_name"))
	if modelName == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "model_name 不能为空"})
		return
	}
	mode := strings.TrimSpace(c.DefaultQuery("mode", model.UserPricingModePriceCap))
	if mode != model.UserPricingModeChannelList {
		mode = model.UserPricingModePriceCap
	}
	costDisc, _ := strconv.ParseFloat(c.DefaultQuery("price_discount_percent", "100"), 64)
	operating, _ := strconv.ParseFloat(c.DefaultQuery("operating_cost_percent", "0"), 64)
	markup, _ := strconv.ParseFloat(c.DefaultQuery("markup_discount_rate", "0"), 64)
	totalPercent := costDisc + operating + markup

	cap, capOK := service.UnitPriceCapForTotalPercent(modelName, totalPercent)

	selectedSet := map[int]int{}
	if selectedRaw := strings.TrimSpace(c.Query("selected_channel_ids")); selectedRaw != "" {
		for i, part := range strings.Split(selectedRaw, ",") {
			id, err := strconv.Atoi(strings.TrimSpace(part))
			if err != nil || id <= 0 {
				continue
			}
			if _, ok := selectedSet[id]; !ok {
				selectedSet[id] = i + 1
			}
		}
	}

	channelIDs := model.GetEnabledChannelIDsByModel(modelName)
	channels := make([]userModelPricingPreviewChannel, 0, len(channelIDs))
	withinCount := 0
	for _, id := range channelIDs {
		ch, err := model.CacheGetChannel(id)
		if err != nil || ch == nil || ch.Status != common.ChannelStatusEnabled {
			continue
		}
		price := service.ResolveChannelModelUnitPrice(ch, modelName)
		within := !capOK || price <= cap*(1+1e-9)
		if within {
			withinCount++
		}
		pri, selected := selectedSet[ch.Id]
		channels = append(channels, userModelPricingPreviewChannel{
			ChannelId:   ch.Id,
			ChannelName: ch.Name,
			UnitPrice:   price,
			WithinCap:   within,
			Selected:    selected,
			Priority:    pri,
		})
	}
	common.ApiSuccess(c, gin.H{
		"mode":           mode,
		"total_percent":  totalPercent,
		"cap":            cap,
		"cap_defined":    capOK,
		"channels":       channels,
		"within_count":   withinCount,
		"total_channels": len(channels),
		"selected_count": len(selectedSet),
	})
}

// ListUserModelPricingUsers GET /api/user_model_pricing/users
// 已配置指定价的用户汇总，供管理页「按用户」筛选。
func ListUserModelPricingUsers(c *gin.Context) {
	rows, err := model.ListUsersWithModelPricing()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, rows)
}

type importUserModelPricingReq struct {
	UserId  int   `json:"user_id"`
	Enabled *bool `json:"enabled"`
}

// PreviewImportUserModelPricing GET /api/user_model_pricing/import_preview?user_id=
// 预览：每个已定价模型将抄自当前最便宜启用渠道的三项折扣（不写入）。
func PreviewImportUserModelPricing(c *gin.Context) {
	userId, _ := strconv.Atoi(c.Query("user_id"))
	if userId <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "user_id 不能为空"})
		return
	}
	_, preview := service.BuildUserModelPricingImportFromCheapestChannels(userId, true)
	common.ApiSuccess(c, gin.H{
		"total_models": len(preview),
		"items":        preview,
	})
}

// ImportUserModelPricing POST /api/user_model_pricing/import
// 一键导入：将当前平台「已启用且已配置展示定价」的模型，按每个模型当前最便宜启用渠道的三项折扣绑定到指定用户（已存在则覆盖为该渠道当前折扣）。
// 新建为 price_cap；已存在 channel_list 仅更新三折扣，保留渠道清单。
func ImportUserModelPricing(c *gin.Context) {
	var req importUserModelPricingReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	if req.UserId <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "user_id 不能为空"})
		return
	}
	if _, err := model.GetUserById(req.UserId, false); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "用户不存在"})
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	rows, preview := service.BuildUserModelPricingImportFromCheapestChannels(req.UserId, enabled)
	if len(rows) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "当前没有可导入的已定价模型（或无已配置单价的启用渠道）"})
		return
	}
	created, updated, err := model.BulkUpsertUserModelPricingOverrideRows(rows)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"total_models": len(rows),
		"created":      created,
		"updated":      updated,
		"items":        preview,
	})
}

// DeleteUserModelPricingByUser DELETE /api/user_model_pricing/by_user/:user_id
// 清空某用户下全部指定价配置。
func DeleteUserModelPricingByUser(c *gin.Context) {
	userId, err := strconv.Atoi(c.Param("user_id"))
	if err != nil || userId <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的 user_id"})
		return
	}
	n, err := model.DeleteUserModelPricingOverridesByUserId(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"deleted": n})
}

type convertChannelListReq struct {
	UserId     int      `json:"user_id"`
	ModelNames []string `json:"model_names"`  // 空 = 该用户全部模型（默认不选 = 全切）
	TargetMode string   `json:"target_mode"`  // price_cap | channel_list；空 = channel_list
}

// ConvertUserModelPricingToChannelList POST /api/user_model_pricing/convert_channel_list
// 批量切换选路模式。model_names 为空或不传时默认全切。
// target_mode=channel_list（默认）：勾选未超指定售价渠道；
// target_mode=price_cap：改回价格上限并清空渠道清单。
func ConvertUserModelPricingToChannelList(c *gin.Context) {
	var req convertChannelListReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	if req.UserId <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "user_id 不能为空"})
		return
	}
	targetMode := strings.TrimSpace(req.TargetMode)
	if targetMode == "" {
		targetMode = model.UserPricingModeChannelList
	}
	if targetMode != model.UserPricingModePriceCap && targetMode != model.UserPricingModeChannelList {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "target_mode 须为 price_cap 或 channel_list"})
		return
	}
	if _, err := model.GetUserById(req.UserId, false); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "用户不存在"})
		return
	}
	converted, skipped, items, err := service.ConvertUserModelPricingMode(req.UserId, req.ModelNames, targetMode)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if converted == 0 && skipped == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "该用户暂无指定价配置"})
		return
	}
	if converted == 0 {
		msg := "没有可转换的模型"
		if targetMode == model.UserPricingModeChannelList {
			msg = "没有可转换的模型（均无未超价启用渠道）"
		}
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": msg,
			"data": gin.H{
				"converted":   converted,
				"skipped":     skipped,
				"items":       items,
				"target_mode": targetMode,
			},
		})
		return
	}
	common.ApiSuccess(c, gin.H{
		"converted":   converted,
		"skipped":     skipped,
		"items":       items,
		"scope":       len(req.ModelNames),
		"scope_all":   len(req.ModelNames) == 0,
		"target_mode": targetMode,
	})
}
