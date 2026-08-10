package controller

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

const (
	homeHotModelLimitOption  = "HomeHotModelLimit"
	defaultHomeHotModelLimit = 8
)

type saveChannelModelHotOverrideRequest struct {
	ChannelID    int    `json:"channel_id"`
	ModelName    string `json:"model_name"`
	OverrideMode string `json:"override_mode"`
	ManualRank   int    `json:"manual_rank"`
}

func GetChannelModelHotOverrides(c *gin.Context) {
	overrides, err := model.GetAllChannelModelHotOverrides()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, overrides)
}

func SaveChannelModelHotOverride(c *gin.Context) {
	var req saveChannelModelHotOverrideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	if req.ChannelID <= 0 || strings.TrimSpace(req.ModelName) == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "渠道和模型不能为空"})
		return
	}
	override := &model.ChannelModelHotOverride{
		ChannelID:    req.ChannelID,
		ModelName:    req.ModelName,
		OverrideMode: req.OverrideMode,
		ManualRank:   req.ManualRank,
	}
	if err := model.SaveChannelModelHotOverride(override); err != nil {
		common.ApiError(c, err)
		return
	}
	model.RefreshPricing()
	common.ApiSuccess(c, nil)
}

func BatchSaveChannelModelHotOverrides(c *gin.Context) {
	var req struct {
		Overrides []saveChannelModelHotOverrideRequest `json:"overrides"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	overrides := make([]model.ChannelModelHotOverride, 0, len(req.Overrides))
	for _, item := range req.Overrides {
		overrides = append(overrides, model.ChannelModelHotOverride{
			ChannelID:    item.ChannelID,
			ModelName:    item.ModelName,
			OverrideMode: item.OverrideMode,
			ManualRank:   item.ManualRank,
		})
	}
	if err := model.BatchSaveChannelModelHotOverrides(overrides); err != nil {
		common.ApiError(c, err)
		return
	}
	model.RefreshPricing()
	common.ApiSuccess(c, gin.H{"updated": len(overrides)})
}

func getHomeHotSettings() (string, int) {
	common.OptionMapRWMutex.RLock()
	period := common.OptionMap["HeatStatPeriod"]
	limitText := common.OptionMap[homeHotModelLimitOption]
	common.OptionMapRWMutex.RUnlock()
	if period == "" {
		period = model.HeatStatPeriod7d
	}
	limit, err := strconv.Atoi(limitText)
	if err != nil || limit < 1 || limit > 100 {
		limit = defaultHomeHotModelLimit
	}
	return period, limit
}

func GetHomeHotSettings(c *gin.Context) {
	period, limit := getHomeHotSettings()
	common.ApiSuccess(c, gin.H{
		"period": period,
		"limit":  limit,
	})
}

func SetHomeHotSettings(c *gin.Context) {
	var req struct {
		Period string `json:"period"`
		Limit  int    `json:"limit"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	switch req.Period {
	case model.HeatStatPeriod7d, model.HeatStatPeriod30d, model.HeatStatPeriodAll:
	default:
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "统计周期仅支持 7d、30d 或 all"})
		return
	}
	if req.Limit < 1 || req.Limit > 100 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "自动热门数量必须在 1 到 100 之间"})
		return
	}
	if err := model.UpdateOption("HeatStatPeriod", req.Period); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.UpdateOption(homeHotModelLimitOption, strconv.Itoa(req.Limit)); err != nil {
		common.ApiError(c, err)
		return
	}
	model.InvalidateChannelModelHotStatsCache()
	model.RefreshPricing()
	common.ApiSuccess(c, nil)
}
