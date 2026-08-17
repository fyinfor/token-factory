package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type channelModelPricePlanRequest struct {
	ModelName string                             `json:"model_name"`
	Name      string                             `json:"name"`
	Enabled   *bool                              `json:"enabled"`
	Pricing   model.ChannelModelPricePlanPayload `json:"pricing"`
}

type channelModelPricePlanResponse struct {
	model.ChannelModelPricePlan
	Pricing model.ChannelModelPricePlanPayload `json:"pricing"`
}

type channelModelPriceScheduleRequest struct {
	ModelName     string `json:"model_name"`
	PricePlanID   int    `json:"price_plan_id"`
	Name          string `json:"name"`
	Timezone      string `json:"timezone"`
	Weekdays      int    `json:"weekdays"`
	StartMinute   int    `json:"start_minute"`
	EndMinute     int    `json:"end_minute"`
	EffectiveFrom string `json:"effective_from"`
	EffectiveTo   string `json:"effective_to"`
	Enabled       *bool  `json:"enabled"`
}

func parsePositivePathInt(c *gin.Context, name string) (int, bool) {
	value, err := strconv.Atoi(c.Param(name))
	if err != nil || value <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的参数"})
		return 0, false
	}
	return value, true
}

func authorizeChannelTimePricing(c *gin.Context, channelID int) (*model.Channel, bool) {
	channel, err := model.GetChannelById(channelID, false)
	if err != nil || channel == nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "渠道不存在"})
		return nil, false
	}
	if c.GetInt("role") >= common.RoleAdminUser {
		return channel, true
	}
	application, err := model.GetApprovedSupplierApplicationByApplicant(c.GetInt("id"))
	if err != nil || channel.OwnerUserID != c.GetInt("id") || channel.SupplierApplicationID != application.ID {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "无权管理该渠道的分时定价"})
		return nil, false
	}
	return channel, true
}

func normalizeChannelPricingModel(channel *model.Channel, rawName string) (string, bool) {
	name := ratio_setting.FormatMatchingModelName(strings.TrimSpace(rawName))
	if channel == nil || name == "" {
		return "", false
	}
	models := channel.GetModels()
	if len(models) == 0 {
		return name, true
	}
	for _, item := range models {
		if ratio_setting.FormatMatchingModelName(item) == name {
			return name, true
		}
	}
	return "", false
}

func respondChannelTimePricingError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, model.ErrChannelModelScheduleConflict), errors.Is(err, model.ErrChannelModelPricePlanInUse):
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": err.Error()})
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "记录不存在"})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
	}
}

func pricePlanResponse(plan model.ChannelModelPricePlan) (channelModelPricePlanResponse, error) {
	payload, err := model.ParseChannelModelPricePlanPayload(plan.PricePayload)
	if err != nil {
		return channelModelPricePlanResponse{}, err
	}
	return channelModelPricePlanResponse{ChannelModelPricePlan: plan, Pricing: payload}, nil
}

func GetChannelModelTimePricing(c *gin.Context) {
	channelID, ok := parsePositivePathInt(c, "channel_id")
	if !ok {
		return
	}
	channel, ok := authorizeChannelTimePricing(c, channelID)
	if !ok {
		return
	}
	modelName, ok := normalizeChannelPricingModel(channel, c.Query("model_name"))
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "模型不属于该渠道"})
		return
	}
	plans, err := model.ListChannelModelPricePlans(channelID, modelName)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	planResponses := make([]channelModelPricePlanResponse, 0, len(plans))
	for _, plan := range plans {
		response, parseErr := pricePlanResponse(plan)
		if parseErr != nil {
			continue
		}
		planResponses = append(planResponses, response)
	}
	schedules, err := model.ListChannelModelPriceSchedules(channelID, modelName)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	model.SortChannelModelSchedulesForDisplay(schedules)
	var activeScheduleID, activePlanID int
	if active, matched := model.ResolveActiveChannelModelPricePlan(channelID, modelName, time.Now()); matched {
		activeScheduleID = active.Schedule.ID
		activePlanID = active.Plan.ID
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"model_name": modelName, "timezone": model.DefaultChannelModelPricingTimezone,
			"plans": planResponses, "schedules": schedules,
			"active_schedule_id": activeScheduleID, "active_plan_id": activePlanID,
			"channel_rates": gin.H{
				"price_discount_percent": channel.ResolvedPriceDiscountPercent(),
				"operating_cost_percent": channel.ResolvedOperatingCostPercent(),
				"effective_cost_percent": channel.ResolvedEffectiveCostPercent(),
				"markup_discount_rate":   channel.ResolvedMarkupDiscountRate(),
			},
		},
	})
}

func CreateChannelModelPricePlan(c *gin.Context) {
	channelID, ok := parsePositivePathInt(c, "channel_id")
	if !ok {
		return
	}
	channel, ok := authorizeChannelTimePricing(c, channelID)
	if !ok {
		return
	}
	var request channelModelPricePlanRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的 JSON"})
		return
	}
	modelName, ok := normalizeChannelPricingModel(channel, request.ModelName)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "模型不属于该渠道"})
		return
	}
	enabled := true
	if request.Enabled != nil {
		enabled = *request.Enabled
	}
	plan := &model.ChannelModelPricePlan{
		ChannelID: channelID, SupplierApplicationID: channel.SupplierApplicationID,
		ModelName: modelName, Name: request.Name, Enabled: enabled,
		CreatedByUserID: c.GetInt("id"), UpdatedByUserID: c.GetInt("id"),
	}
	if err := model.CreateChannelModelPricePlan(plan, request.Pricing); err != nil {
		respondChannelTimePricingError(c, err)
		return
	}
	response, err := pricePlanResponse(*plan)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": response})
}

func UpdateChannelModelPricePlan(c *gin.Context) {
	channelID, ok := parsePositivePathInt(c, "channel_id")
	if !ok {
		return
	}
	if _, ok = authorizeChannelTimePricing(c, channelID); !ok {
		return
	}
	planID, ok := parsePositivePathInt(c, "plan_id")
	if !ok {
		return
	}
	existing, err := model.GetChannelModelPricePlanByID(planID)
	if err != nil || existing.ChannelID != channelID {
		respondChannelTimePricingError(c, gorm.ErrRecordNotFound)
		return
	}
	var request channelModelPricePlanRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的 JSON"})
		return
	}
	enabled := existing.Enabled
	if request.Enabled != nil {
		enabled = *request.Enabled
	}
	existing.Name = request.Name
	existing.Enabled = enabled
	existing.UpdatedByUserID = c.GetInt("id")
	if err := model.UpdateChannelModelPricePlan(existing, request.Pricing); err != nil {
		respondChannelTimePricingError(c, err)
		return
	}
	response, err := pricePlanResponse(*existing)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": response})
}

func DeleteChannelModelPricePlan(c *gin.Context) {
	channelID, ok := parsePositivePathInt(c, "channel_id")
	if !ok {
		return
	}
	if _, ok = authorizeChannelTimePricing(c, channelID); !ok {
		return
	}
	planID, ok := parsePositivePathInt(c, "plan_id")
	if !ok {
		return
	}
	existing, err := model.GetChannelModelPricePlanByID(planID)
	if err != nil || existing.ChannelID != channelID {
		respondChannelTimePricingError(c, gorm.ErrRecordNotFound)
		return
	}
	if err := model.DeleteChannelModelPricePlan(planID); err != nil {
		respondChannelTimePricingError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func CreateChannelModelPriceSchedule(c *gin.Context) {
	channelID, ok := parsePositivePathInt(c, "channel_id")
	if !ok {
		return
	}
	channel, ok := authorizeChannelTimePricing(c, channelID)
	if !ok {
		return
	}
	var request channelModelPriceScheduleRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的 JSON"})
		return
	}
	modelName, ok := normalizeChannelPricingModel(channel, request.ModelName)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "模型不属于该渠道"})
		return
	}
	enabled := true
	if request.Enabled != nil {
		enabled = *request.Enabled
	}
	schedule := &model.ChannelModelPriceSchedule{
		ChannelID: channelID, SupplierApplicationID: channel.SupplierApplicationID,
		ModelName: modelName, PricePlanID: request.PricePlanID, Name: request.Name,
		Timezone: request.Timezone, Weekdays: request.Weekdays,
		StartMinute: request.StartMinute, EndMinute: request.EndMinute,
		EffectiveFrom: request.EffectiveFrom, EffectiveTo: request.EffectiveTo,
		Enabled: enabled, CreatedByUserID: c.GetInt("id"), UpdatedByUserID: c.GetInt("id"),
	}
	if err := model.CreateChannelModelPriceSchedule(schedule); err != nil {
		respondChannelTimePricingError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": schedule})
}

func UpdateChannelModelPriceSchedule(c *gin.Context) {
	channelID, ok := parsePositivePathInt(c, "channel_id")
	if !ok {
		return
	}
	if _, ok = authorizeChannelTimePricing(c, channelID); !ok {
		return
	}
	scheduleID, ok := parsePositivePathInt(c, "schedule_id")
	if !ok {
		return
	}
	existing, err := model.GetChannelModelPriceScheduleByID(scheduleID)
	if err != nil || existing.ChannelID != channelID {
		respondChannelTimePricingError(c, gorm.ErrRecordNotFound)
		return
	}
	var request channelModelPriceScheduleRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的 JSON"})
		return
	}
	enabled := existing.Enabled
	if request.Enabled != nil {
		enabled = *request.Enabled
	}
	existing.PricePlanID = request.PricePlanID
	existing.Name = request.Name
	existing.Timezone = request.Timezone
	existing.Weekdays = request.Weekdays
	existing.StartMinute = request.StartMinute
	existing.EndMinute = request.EndMinute
	existing.EffectiveFrom = request.EffectiveFrom
	existing.EffectiveTo = request.EffectiveTo
	existing.Enabled = enabled
	existing.UpdatedByUserID = c.GetInt("id")
	if err := model.UpdateChannelModelPriceSchedule(existing); err != nil {
		respondChannelTimePricingError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": existing})
}

func DeleteChannelModelPriceSchedule(c *gin.Context) {
	channelID, ok := parsePositivePathInt(c, "channel_id")
	if !ok {
		return
	}
	if _, ok = authorizeChannelTimePricing(c, channelID); !ok {
		return
	}
	scheduleID, ok := parsePositivePathInt(c, "schedule_id")
	if !ok {
		return
	}
	existing, err := model.GetChannelModelPriceScheduleByID(scheduleID)
	if err != nil || existing.ChannelID != channelID {
		respondChannelTimePricingError(c, gorm.ErrRecordNotFound)
		return
	}
	if err := model.DeleteChannelModelPriceSchedule(scheduleID); err != nil {
		respondChannelTimePricingError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
