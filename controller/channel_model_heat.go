package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// GetChannelModelHeats 获取所有渠道-模型组合的热力配置
func GetChannelModelHeats(c *gin.Context) {
	heats, err := model.GetAllChannelModelHeats()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    heats,
	})
}

// GetChannelModelHeatsByChannel 获取指定渠道的所有模型热力配置
func GetChannelModelHeatsByChannel(c *gin.Context) {
	channelID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无效的渠道ID",
		})
		return
	}

	heats, err := model.GetChannelModelHeatsByChannel(channelID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    heats,
	})
}

// SaveChannelModelHeatRequest 保存渠道-模型热力配置的请求结构
type SaveChannelModelHeatRequest struct {
	ChannelID          int     `json:"channel_id"`
	ModelName          string  `json:"model_name"`
	ModelSortWeight    float64 `json:"model_sort_weight"`
	ChannelSortWeight  float64 `json:"channel_sort_weight"`
	ManualBaseReqCount int64   `json:"manual_base_req_count"`
}

// SaveChannelModelHeat 保存单个渠道-模型组合的热力配置
func SaveChannelModelHeat(c *gin.Context) {
	var req SaveChannelModelHeatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	heat := &model.ChannelModelHeat{
		ChannelID:          req.ChannelID,
		ModelName:          req.ModelName,
		ModelSortWeight:    req.ModelSortWeight,
		ChannelSortWeight:  req.ChannelSortWeight,
		ManualBaseReqCount: req.ManualBaseReqCount,
	}

	if err := model.SaveChannelModelHeat(heat); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "保存成功",
	})
}

// BatchSaveChannelModelHeatsRequest 批量保存请求结构
type BatchSaveChannelModelHeatsRequest struct {
	Heats []SaveChannelModelHeatRequest `json:"heats"`
}

// BatchSaveChannelModelHeats 批量保存渠道-模型组合的热力配置
func BatchSaveChannelModelHeats(c *gin.Context) {
	var req BatchSaveChannelModelHeatsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	heats := make([]model.ChannelModelHeat, 0, len(req.Heats))
	for _, h := range req.Heats {
		heats = append(heats, model.ChannelModelHeat{
			ChannelID:          h.ChannelID,
			ModelName:          h.ModelName,
			ModelSortWeight:    h.ModelSortWeight,
			ChannelSortWeight:  h.ChannelSortWeight,
			ManualBaseReqCount: h.ManualBaseReqCount,
		})
	}

	if err := model.BatchSaveChannelModelHeats(heats); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "批量保存成功",
	})
}

// GetHeatStatPeriod 获取热度统计周期配置
func GetHeatStatPeriod(c *gin.Context) {
	common.OptionMapRWMutex.RLock()
	period := common.OptionMap["HeatStatPeriod"]
	common.OptionMapRWMutex.RUnlock()
	if period == "" {
		period = model.HeatStatPeriod7d
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    period,
	})
}

// SetHeatStatPeriod 设置热度统计周期配置
func SetHeatStatPeriod(c *gin.Context) {
	var req struct {
		Period string `json:"period"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	switch req.Period {
	case model.HeatStatPeriod7d, model.HeatStatPeriod30d, model.HeatStatPeriodAll:
	default:
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的周期值，可选: 7d, 30d, all"})
		return
	}
	if err := model.UpdateOption("HeatStatPeriod", req.Period); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "保存成功"})
}

// DeleteChannelModelHeat 删除渠道-模型组合的热力配置
func DeleteChannelModelHeat(c *gin.Context) {
	channelID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无效的渠道ID",
		})
		return
	}

	modelName := c.Param("model_name")
	if modelName == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "模型名称不能为空",
		})
		return
	}

	if err := model.DeleteChannelModelHeat(channelID, modelName); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "删除成功",
	})
}
