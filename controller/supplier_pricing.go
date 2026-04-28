package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// SupplierPricingMapsPayload 供应商 PUT 提交的模型定价映射（键与 Option / 编辑器一致）。
type SupplierPricingMapsPayload struct {
	ModelPrice           map[string]float64 `json:"ModelPrice"`
	ModelRatio           map[string]float64 `json:"ModelRatio"`
	CompletionRatio      map[string]float64 `json:"CompletionRatio"`
	CacheRatio           map[string]float64 `json:"CacheRatio"`
	CreateCacheRatio     map[string]float64 `json:"CreateCacheRatio"`
	ImageRatio           map[string]float64 `json:"ImageRatio"`
	AudioRatio           map[string]float64 `json:"AudioRatio"`
	AudioCompletionRatio map[string]float64 `json:"AudioCompletionRatio"`
}

// supplierPricingPayloadToUpsertMaps 将请求体转为 upsert 用的嵌套 map。
func supplierPricingPayloadToUpsertMaps(p *SupplierPricingMapsPayload) map[string]map[string]float64 {
	if p == nil {
		return nil
	}
	return map[string]map[string]float64{
		"ModelPrice":           nonNilMap(p.ModelPrice),
		"ModelRatio":           nonNilMap(p.ModelRatio),
		"CompletionRatio":      nonNilMap(p.CompletionRatio),
		"CacheRatio":           nonNilMap(p.CacheRatio),
		"CreateCacheRatio":     nonNilMap(p.CreateCacheRatio),
		"ImageRatio":           nonNilMap(p.ImageRatio),
		"AudioRatio":           nonNilMap(p.AudioRatio),
		"AudioCompletionRatio": nonNilMap(p.AudioCompletionRatio),
	}
}

func nonNilMap(m map[string]float64) map[string]float64 {
	if m == nil {
		return map[string]float64{}
	}
	return m
}

// GetSupplierGlobalPricing 返回当前登录供应商的全局模型定价（表存储），形状与编辑器 maps 一致。
func GetSupplierGlobalPricing(c *gin.Context) {
	app, err := model.GetApprovedSupplierApplicationByApplicant(c.GetInt("id"))
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "当前账号无已审核通过的供应商资质",
		})
		return
	}
	rows, err := model.ListSupplierModelPricingsForSupplier(app.ID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	data := model.BuildOptionLikeMapsFromSupplierGlobalRows(rows)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}

// PutSupplierGlobalPricing 写入供应商全局模型定价（仅自有模型）。
func PutSupplierGlobalPricing(c *gin.Context) {
	app, err := model.GetApprovedSupplierApplicationByApplicant(c.GetInt("id"))
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "当前账号无已审核通过的供应商资质",
		})
		return
	}
	var payload SupplierPricingMapsPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的 JSON"})
		return
	}
	owned, err := collectSupplierOwnedModelNames(c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	ownedNorm := model.NormalizeOwnedModelsForPricing(owned)
	maps := supplierPricingPayloadToUpsertMaps(&payload)
	if err := model.UpsertSupplierModelPricingMaps(app.ID, c.GetInt("id"), maps, ownedNorm); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}

// GetSupplierChannelPricing 返回指定渠道下的供应商渠道定价映射。
func GetSupplierChannelPricing(c *gin.Context) {
	channelID, err := strconv.Atoi(c.Param("channel_id"))
	if err != nil || channelID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的渠道 ID"})
		return
	}
	app, err := model.GetApprovedSupplierApplicationByApplicant(c.GetInt("id"))
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "当前账号无已审核通过的供应商资质",
		})
		return
	}
	ch, err := model.GetChannelById(channelID, false)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "渠道不存在"})
		return
	}
	if ch.OwnerUserID != c.GetInt("id") || ch.SupplierApplicationID != app.ID {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "无权访问该渠道"})
		return
	}
	rows, err := model.ListSupplierChannelModelPricings(app.ID, channelID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	data := model.BuildOptionLikeMapsFromSupplierChannelRows(rows)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}

// PutSupplierChannelPricing 写入供应商渠道维度定价。
func PutSupplierChannelPricing(c *gin.Context) {
	channelID, err := strconv.Atoi(c.Param("channel_id"))
	if err != nil || channelID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的渠道 ID"})
		return
	}
	app, err := model.GetApprovedSupplierApplicationByApplicant(c.GetInt("id"))
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "当前账号无已审核通过的供应商资质",
		})
		return
	}
	ch, err := model.GetChannelById(channelID, false)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "渠道不存在"})
		return
	}
	if ch.OwnerUserID != c.GetInt("id") || ch.SupplierApplicationID != app.ID {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "无权修改该渠道"})
		return
	}
	var payload SupplierPricingMapsPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的 JSON"})
		return
	}
	owned, err := collectSupplierOwnedModelNames(c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	ownedNorm := model.NormalizeOwnedModelsForPricing(owned)
	maps := supplierPricingPayloadToUpsertMaps(&payload)
	if err := model.UpsertSupplierChannelModelPricingMaps(app.ID, channelID, c.GetInt("id"), maps, ownedNorm); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}
