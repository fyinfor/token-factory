package controller

import (
	"errors"
	"fmt"
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

type channelModelRateRuleRequest struct {
	ModelNames           []string `json:"model_names"`
	Name                 string   `json:"name"`
	PriceDiscountPercent float64  `json:"price_discount_percent"`
	OperatingCostPercent float64  `json:"operating_cost_percent"`
	MarkupDiscountRate   float64  `json:"markup_discount_rate"`
	Timezone             string   `json:"timezone"`
	Weekdays             int      `json:"weekdays"`
	StartMinute          int      `json:"start_minute"`
	EndMinute            int      `json:"end_minute"`
	EffectiveFrom        string   `json:"effective_from"`
	EffectiveTo          string   `json:"effective_to"`
	Enabled              *bool    `json:"enabled"`
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
	var conflictErr *model.ChannelModelScheduleConflictError
	switch {
	case errors.As(err, &conflictErr):
		conflicting := conflictErr.ConflictingSchedule
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"code":    "channel_model_schedule_conflict",
			"message": formatChannelModelScheduleConflict(conflicting),
			"conflict": gin.H{
				"schedule_id":    conflicting.ID,
				"model_name":     conflicting.ModelName,
				"name":           conflicting.Name,
				"weekdays":       conflicting.Weekdays,
				"start_minute":   conflicting.StartMinute,
				"end_minute":     conflicting.EndMinute,
				"effective_from": conflicting.EffectiveFrom,
				"effective_to":   conflicting.EffectiveTo,
			},
		})
	case errors.Is(err, model.ErrChannelModelScheduleConflict):
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"code":    "channel_model_schedule_conflict",
			"message": "动态费率的生效范围与另一条已启用规则冲突，请调整重复日期、时间范围或生效日期。",
		})
	case errors.Is(err, model.ErrChannelModelPricePlanInUse):
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": "价格方案仍被已启用的动态费率使用，无法删除。"})
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "记录不存在"})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
	}
}

func formatChannelModelScheduleConflict(schedule model.ChannelModelPriceSchedule) string {
	return fmt.Sprintf(
		"模型「%s」的动态费率与已启用规则「%s」（规则 ID：%d，%s %s–%s，生效日期：%s）冲突。请调整重复日期、时间范围或生效日期，或先停用该规则。",
		schedule.ModelName,
		schedule.Name,
		schedule.ID,
		formatChannelPricingWeekdays(schedule.Weekdays),
		formatChannelPricingMinute(schedule.StartMinute),
		formatChannelPricingMinute(schedule.EndMinute),
		formatChannelPricingDateRange(schedule.EffectiveFrom, schedule.EffectiveTo),
	)
}

func formatChannelPricingWeekdays(mask int) string {
	if mask == 0x7f {
		return "每天"
	}
	if mask == 0x3e {
		return "工作日"
	}
	labels := []struct {
		value int
		label string
	}{
		{1, "周一"}, {2, "周二"}, {3, "周三"}, {4, "周四"},
		{5, "周五"}, {6, "周六"}, {0, "周日"},
	}
	selected := make([]string, 0, 7)
	for _, item := range labels {
		if mask&(1<<item.value) != 0 {
			selected = append(selected, item.label)
		}
	}
	return strings.Join(selected, "、")
}

func formatChannelPricingMinute(minute int) string {
	if minute == 1440 {
		return "24:00"
	}
	return fmt.Sprintf("%02d:%02d", minute/60, minute%60)
}

func formatChannelPricingDateRange(from, to string) string {
	switch {
	case from == "" && to == "":
		return "长期有效"
	case from == "":
		return "截至 " + to
	case to == "":
		return "自 " + from + " 起"
	case from == to:
		return from
	default:
		return from + " 至 " + to
	}
}

func pricePlanResponse(plan model.ChannelModelPricePlan) (channelModelPricePlanResponse, error) {
	payload, err := model.ParseChannelModelPricePlanPayload(plan.PricePayload)
	if err != nil {
		return channelModelPricePlanResponse{}, err
	}
	return channelModelPricePlanResponse{ChannelModelPricePlan: plan, Pricing: payload}, nil
}

func channelModelRateRuleMutationFromRequest(
	channel *model.Channel,
	channelID int,
	request channelModelRateRuleRequest,
	userID int,
) (*model.ChannelModelRateRuleMutation, error) {
	if channel == nil {
		return nil, errors.New("渠道不存在")
	}
	if len(channel.GetModels()) == 0 {
		return nil, errors.New("该渠道还没有已保存的模型")
	}
	modelNames := make([]string, 0, len(request.ModelNames))
	for _, rawName := range request.ModelNames {
		name, ok := normalizeChannelPricingModel(channel, rawName)
		if !ok {
			return nil, fmt.Errorf("模型不属于该渠道: %s", rawName)
		}
		modelNames = append(modelNames, name)
	}
	enabled := true
	if request.Enabled != nil {
		enabled = *request.Enabled
	}
	return &model.ChannelModelRateRuleMutation{
		ChannelID: channelID, SupplierApplicationID: channel.SupplierApplicationID,
		ModelNames: modelNames, Name: request.Name,
		PriceDiscountPercent: request.PriceDiscountPercent,
		OperatingCostPercent: request.OperatingCostPercent,
		MarkupDiscountRate:   request.MarkupDiscountRate,
		Timezone:             request.Timezone, Weekdays: request.Weekdays,
		StartMinute: request.StartMinute, EndMinute: request.EndMinute,
		EffectiveFrom: request.EffectiveFrom, EffectiveTo: request.EffectiveTo,
		Enabled: enabled, UserID: userID,
	}, nil
}

func GetChannelModelRateRules(c *gin.Context) {
	channelID, ok := parsePositivePathInt(c, "channel_id")
	if !ok {
		return
	}
	channel, ok := authorizeChannelTimePricing(c, channelID)
	if !ok {
		return
	}
	rules, err := model.ListChannelModelRateRules(channelID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"rules":    rules,
			"timezone": model.DefaultChannelModelPricingTimezone,
			"channel_rates": gin.H{
				"price_discount_percent": channel.ResolvedPriceDiscountPercent(),
				"operating_cost_percent": channel.ResolvedOperatingCostPercent(),
				"effective_cost_percent": channel.ResolvedEffectiveCostPercent(),
				"markup_discount_rate":   channel.ResolvedMarkupDiscountRate(),
			},
		},
	})
}

func CreateChannelModelRateRules(c *gin.Context) {
	channelID, ok := parsePositivePathInt(c, "channel_id")
	if !ok {
		return
	}
	channel, ok := authorizeChannelTimePricing(c, channelID)
	if !ok {
		return
	}
	var request channelModelRateRuleRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的 JSON"})
		return
	}
	mutation, err := channelModelRateRuleMutationFromRequest(channel, channelID, request, c.GetInt("id"))
	if err != nil {
		respondChannelTimePricingError(c, err)
		return
	}
	if err := model.CreateChannelModelRateRules(mutation); err != nil {
		respondChannelTimePricingError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func UpdateChannelModelRateRule(c *gin.Context) {
	channelID, ok := parsePositivePathInt(c, "channel_id")
	if !ok {
		return
	}
	channel, ok := authorizeChannelTimePricing(c, channelID)
	if !ok {
		return
	}
	scheduleID, ok := parsePositivePathInt(c, "schedule_id")
	if !ok {
		return
	}
	var request channelModelRateRuleRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的 JSON"})
		return
	}
	mutation, err := channelModelRateRuleMutationFromRequest(channel, channelID, request, c.GetInt("id"))
	if err != nil {
		respondChannelTimePricingError(c, err)
		return
	}
	if err := model.UpdateChannelModelRateRule(scheduleID, mutation); err != nil {
		respondChannelTimePricingError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func DeleteChannelModelRateRule(c *gin.Context) {
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
	if err := model.DeleteChannelModelRateRule(channelID, scheduleID); err != nil {
		respondChannelTimePricingError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
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
	ratePlanIDs := make(map[int]struct{}, len(plans))
	for _, plan := range plans {
		response, parseErr := pricePlanResponse(plan)
		if parseErr != nil || response.Pricing.ResolvedMode() != model.ChannelModelPricePlanModeRate || !response.Pricing.HasRateOverrides() {
			continue
		}
		ratePlanIDs[plan.ID] = struct{}{}
		planResponses = append(planResponses, response)
	}
	schedules, err := model.ListChannelModelPriceSchedules(channelID, modelName)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	filteredSchedules := schedules[:0]
	for _, schedule := range schedules {
		if _, ok := ratePlanIDs[schedule.PricePlanID]; ok {
			filteredSchedules = append(filteredSchedules, schedule)
		}
	}
	schedules = filteredSchedules
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
