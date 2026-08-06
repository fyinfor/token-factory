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
// 命中后计费为「全局官方价 × 三折扣总和」，且选路排除单价超上限的渠道。

type userModelPricingItem struct {
	model.UserModelPricingOverride
	Username     string  `json:"username"`
	TotalPercent float64 `json:"total_percent"`
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
	items := make([]userModelPricingItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, userModelPricingItem{
			UserModelPricingOverride: r,
			Username:                 usernames[r.UserId],
			TotalPercent:             r.TotalPercent(),
		})
	}
	common.ApiSuccess(c, items)
}

type upsertUserModelPricingReq struct {
	UserId               int     `json:"user_id"`
	ModelName            string  `json:"model_name"`
	PriceDiscountPercent float64 `json:"price_discount_percent"`
	OperatingCostPercent float64 `json:"operating_cost_percent"`
	MarkupDiscountRate   float64 `json:"markup_discount_rate"`
	Enabled              bool    `json:"enabled"`
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
	if _, err := model.GetUserById(req.UserId, false); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "用户不存在"})
		return
	}
	ov := &model.UserModelPricingOverride{
		UserId:               req.UserId,
		ModelName:            req.ModelName,
		PriceDiscountPercent: req.PriceDiscountPercent,
		OperatingCostPercent: req.OperatingCostPercent,
		MarkupDiscountRate:   req.MarkupDiscountRate,
		Enabled:              req.Enabled,
	}
	saved, err := model.UpsertUserModelPricingOverride(ov)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, saved)
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
}

// PreviewUserModelPricing GET /api/user_model_pricing/preview?model_name=&price_discount_percent=&operating_cost_percent=&markup_discount_rate=
// 按给定三折扣实时预览：价格上限，以及该模型各渠道是否在上限内（保存前校验用）。
func PreviewUserModelPricing(c *gin.Context) {
	modelName := strings.TrimSpace(c.Query("model_name"))
	if modelName == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "model_name 不能为空"})
		return
	}
	costDisc, _ := strconv.ParseFloat(c.DefaultQuery("price_discount_percent", "100"), 64)
	operating, _ := strconv.ParseFloat(c.DefaultQuery("operating_cost_percent", "0"), 64)
	markup, _ := strconv.ParseFloat(c.DefaultQuery("markup_discount_rate", "0"), 64)
	totalPercent := costDisc + operating + markup

	cap, capOK := service.UnitPriceCapForTotalPercent(modelName, totalPercent)

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
		channels = append(channels, userModelPricingPreviewChannel{
			ChannelId:   ch.Id,
			ChannelName: ch.Name,
			UnitPrice:   price,
			WithinCap:   within,
		})
	}
	common.ApiSuccess(c, gin.H{
		"total_percent":  totalPercent,
		"cap":            cap,
		"cap_defined":    capOK,
		"channels":       channels,
		"within_count":   withinCount,
		"total_channels": len(channels),
	})
}
