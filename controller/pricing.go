package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
)

// filterChannelPricingMapByVisibleChannels 仅保留可见渠道的渠道倍率配置。
func filterChannelPricingMapByVisibleChannels(source map[string]map[string]float64, visibleChannelIDs map[int]struct{}) map[string]map[string]float64 {
	filtered := make(map[string]map[string]float64, len(source))
	for channelID, modelRatio := range source {
		id, err := model.ParseSupplierChannelIDFilter(channelID)
		if err != nil {
			continue
		}
		if _, ok := visibleChannelIDs[id]; !ok {
			continue
		}
		filtered[channelID] = modelRatio
	}
	return filtered
}

// getPricingVisibleChannelsForUser 根据登录用户返回定价页可见渠道列表（供应商仅可见自己渠道）。
func getPricingVisibleChannelsForUser(c *gin.Context) ([]model.ChannelSimplePricingItem, map[int]struct{}, error) {
	channels, err := model.ListChannelsForPricing()
	if err != nil {
		return nil, nil, err
	}
	userID, exists := c.Get("id")
	if !exists {
		visibleChannelIDs := make(map[int]struct{}, len(channels))
		for _, item := range channels {
			visibleChannelIDs[item.ChannelID] = struct{}{}
		}
		return channels, visibleChannelIDs, nil
	}
	user, err := model.GetUserById(userID.(int), false)
	if err == nil && user.Role >= common.RoleAdminUser {
		visibleChannelIDs := make(map[int]struct{}, len(channels))
		for _, item := range channels {
			visibleChannelIDs[item.ChannelID] = struct{}{}
		}
		return channels, visibleChannelIDs, nil
	}
	if _, err := model.GetApprovedSupplierApplicationByApplicant(userID.(int)); err != nil {
		visibleChannelIDs := make(map[int]struct{}, len(channels))
		for _, item := range channels {
			visibleChannelIDs[item.ChannelID] = struct{}{}
		}
		return channels, visibleChannelIDs, nil
	}
	ownerUserID := userID.(int)
	ownedChannels, _, err := model.SearchSupplierChannels(&ownerUserID, 0, 100000, model.SupplierChannelSearchFilter{})
	if err != nil {
		return nil, nil, err
	}
	visibleChannelIDs := make(map[int]struct{}, len(ownedChannels))
	visibleChannels := make([]model.ChannelSimplePricingItem, 0, len(ownedChannels))
	for _, item := range ownedChannels {
		visibleChannelIDs[item.Id] = struct{}{}
		visibleChannels = append(visibleChannels, model.ChannelSimplePricingItem{
			ChannelID:   item.Id,
			ChannelName: item.Name,
		})
	}
	return visibleChannels, visibleChannelIDs, nil
}

// GetPricing 返回前端定价展示数据。
func GetPricing(c *gin.Context) {
	pricing := model.GetPricing()
	filtered := make([]model.Pricing, 0, len(pricing))
	for _, p := range pricing {
		if ratio_setting.ModelHasConfiguredPricing(p.ModelName) {
			filtered = append(filtered, p)
		}
	}
	channels, visibleChannelIDs, err := getPricingVisibleChannelsForUser(c)
	if err != nil {
		channels = []model.ChannelSimplePricingItem{}
		visibleChannelIDs = map[int]struct{}{}
	}
	userId, exists := c.Get("id")
	usableGroup := map[string]string{}
	groupRatio := map[string]float64{}
	groupModelPrice := map[string]map[string]float64{}
	groupModelRatio := map[string]map[string]float64{}
	channelModelPrice := map[string]map[string]float64{}
	channelModelRatio := map[string]map[string]float64{}
	channelCompletionRatio := map[string]map[string]float64{}
	channelCacheRatio := map[string]map[string]float64{}
	channelCreateCacheRatio := map[string]map[string]float64{}
	channelImageRatio := map[string]map[string]float64{}
	channelAudioRatio := map[string]map[string]float64{}
	channelAudioCompletionRatio := map[string]map[string]float64{}
	supplierModelPrice := map[string]map[string]float64{}
	supplierModelRatio := map[string]map[string]float64{}
	for s, f := range ratio_setting.GetGroupRatioCopy() {
		groupRatio[s] = f
	}
	var group string
	if exists {
		user, err := model.GetUserCache(userId.(int))
		if err == nil {
			group = user.Group
			for g := range groupRatio {
				ratio, ok := ratio_setting.GetGroupGroupRatio(group, g)
				if ok {
					groupRatio[g] = ratio
				}
			}
		}
	}

	usableGroup = service.GetUserUsableGroups(group)
	// check groupRatio contains usableGroup
	for group := range ratio_setting.GetGroupRatioCopy() {
		if _, ok := usableGroup[group]; !ok {
			delete(groupRatio, group)
		}
	}
	for group, modelPrice := range ratio_setting.GetGroupModelPriceCopy() {
		if _, ok := usableGroup[group]; ok {
			groupModelPrice[group] = modelPrice
		}
	}
	for group, modelRatioByGroup := range ratio_setting.GetGroupModelRatioCopy() {
		if _, ok := usableGroup[group]; ok {
			groupModelRatio[group] = modelRatioByGroup
		}
	}
	for channelID, modelPrice := range ratio_setting.GetChannelModelPriceCopy() {
		channelModelPrice[channelID] = modelPrice
	}
	for channelID, modelRatio := range ratio_setting.GetChannelModelRatioCopy() {
		channelModelRatio[channelID] = modelRatio
	}
	for channelID, modelRatio := range ratio_setting.GetChannelCompletionRatioCopy() {
		channelCompletionRatio[channelID] = modelRatio
	}
	for channelID, modelRatio := range ratio_setting.GetChannelCacheRatioCopy() {
		channelCacheRatio[channelID] = modelRatio
	}
	for channelID, modelRatio := range ratio_setting.GetChannelCreateCacheRatioCopy() {
		channelCreateCacheRatio[channelID] = modelRatio
	}
	for channelID, modelRatio := range ratio_setting.GetChannelImageRatioCopy() {
		channelImageRatio[channelID] = modelRatio
	}
	for channelID, modelRatio := range ratio_setting.GetChannelAudioRatioCopy() {
		channelAudioRatio[channelID] = modelRatio
	}
	for channelID, modelRatio := range ratio_setting.GetChannelAudioCompletionRatioCopy() {
		channelAudioCompletionRatio[channelID] = modelRatio
	}
	channelModelPrice = filterChannelPricingMapByVisibleChannels(channelModelPrice, visibleChannelIDs)
	channelModelRatio = filterChannelPricingMapByVisibleChannels(channelModelRatio, visibleChannelIDs)
	channelCompletionRatio = filterChannelPricingMapByVisibleChannels(channelCompletionRatio, visibleChannelIDs)
	channelCacheRatio = filterChannelPricingMapByVisibleChannels(channelCacheRatio, visibleChannelIDs)
	channelCreateCacheRatio = filterChannelPricingMapByVisibleChannels(channelCreateCacheRatio, visibleChannelIDs)
	channelImageRatio = filterChannelPricingMapByVisibleChannels(channelImageRatio, visibleChannelIDs)
	channelAudioRatio = filterChannelPricingMapByVisibleChannels(channelAudioRatio, visibleChannelIDs)
	channelAudioCompletionRatio = filterChannelPricingMapByVisibleChannels(channelAudioCompletionRatio, visibleChannelIDs)
	for supplierID, modelPrice := range ratio_setting.GetSupplierModelPriceCopy() {
		supplierModelPrice[supplierID] = modelPrice
	}
	for supplierID, modelRatio := range ratio_setting.GetSupplierModelRatioCopy() {
		supplierModelRatio[supplierID] = modelRatio
	}

	c.JSON(200, gin.H{
		"success":                        true,
		"data":                           filtered,
		"vendors":                        model.GetVendors(),
		"channels":                       channels,
		"group_ratio":                    groupRatio,
		"group_model_price":              groupModelPrice,
		"group_model_ratio":              groupModelRatio,
		"channel_model_price":            channelModelPrice,
		"channel_model_ratio":            channelModelRatio,
		"channel_completion_ratio":       channelCompletionRatio,
		"channel_cache_ratio":            channelCacheRatio,
		"channel_create_cache_ratio":     channelCreateCacheRatio,
		"channel_image_ratio":            channelImageRatio,
		"channel_audio_ratio":            channelAudioRatio,
		"channel_audio_completion_ratio": channelAudioCompletionRatio,
		"supplier_model_price":           supplierModelPrice,
		"supplier_model_ratio":           supplierModelRatio,
		"usable_group":                   usableGroup,
		"supported_endpoint":             model.GetSupportedEndpointMap(),
		"auto_groups":                    service.GetUserAutoGroup(group),
		"pricing_version":                "a42d372ccf0b5dd13ecf71203521f9d2",
	})
}

func ResetModelRatio(c *gin.Context) {
	defaultStr := ratio_setting.DefaultModelRatio2JSONString()
	err := model.UpdateOption("ModelRatio", defaultStr)
	if err != nil {
		c.JSON(200, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	err = ratio_setting.UpdateModelRatioByJSONString(defaultStr)
	if err != nil {
		c.JSON(200, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{
		"success": true,
		"message": "重置模型倍率成功",
	})
}
