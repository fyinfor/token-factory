package controller

import (
	"errors"
	"net/http"
	"strings"

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

// getPricingVisibleChannelsForUser 返回定价/模型广场可见的渠道列表及 channel_* Option 过滤用的 ID 集合。
// 当前策略：所有角色（包含已审核供应商）均可见全部渠道，与普通用户保持一致。
func getPricingVisibleChannelsForUser(c *gin.Context) ([]model.ChannelSimplePricingItem, map[int]struct{}, error) {
	channels, err := model.ListChannelsForPricing()
	if err != nil {
		return nil, nil, err
	}
	visibleChannelIDs := make(map[int]struct{}, len(channels))
	for _, item := range channels {
		visibleChannelIDs[item.ChannelID] = struct{}{}
	}
	return channels, visibleChannelIDs, nil
}

// shouldBlurPricing 检查 HeaderNavModules 配置中是否有任一模块开启了 blurPricing。
func shouldBlurPricing() bool {
	common.OptionMapRWMutex.RLock()
	raw := common.OptionMap["HeaderNavModules"]
	common.OptionMapRWMutex.RUnlock()
	if raw == "" {
		return false
	}
	var modules map[string]any
	if err := common.Unmarshal([]byte(raw), &modules); err != nil {
		return false
	}
	for _, key := range []string{"home", "pricing"} {
		switch v := modules[key].(type) {
		case map[string]any:
			if bp, ok := v["blurPricing"]; ok {
				if b, ok := bp.(bool); ok && b {
					return true
				}
			}
		}
	}
	return false
}

// sanitizePricingData 将定价数据中的价格和供应商信息置零/清空。
func sanitizePricingData(data []model.PricingAPIItem) {
	for i := range data {
		data[i].ModelRatio = 0
		data[i].ModelPrice = 0
		data[i].CompletionRatio = 0
		data[i].CacheRatio = nil
		data[i].CreateCacheRatio = nil
		data[i].ImageRatio = nil
		data[i].AudioRatio = nil
		data[i].AudioCompletionRatio = nil
		data[i].VideoRatio = nil
		data[i].VideoCompletionRatio = nil
		data[i].VideoPrice = nil
		for j := range data[i].ChannelList {
			data[i].ChannelList[j].ModelPrice = 0
			data[i].ChannelList[j].ModelRatio = 0
			data[i].ChannelList[j].CompletionRatio = 0
			data[i].ChannelList[j].CacheRatio = 0
			data[i].ChannelList[j].CreateCacheRatio = 0
			data[i].ChannelList[j].PriceDiscountPercent = 0
			data[i].ChannelList[j].SupplierAlias = ""
			data[i].ChannelList[j].CompanyLogoURL = ""
			data[i].ChannelList[j].SupplierType = ""
		}
		for j := range data[i].SupplierList {
			data[i].SupplierList[j].SupplierAlias = ""
			data[i].SupplierList[j].CompanyLogoURL = ""
			data[i].SupplierList[j].SupplierType = ""
		}
	}
}

func buildPricingAPIData() []model.PricingAPIItem {
	pricing := model.GetPricing()
	filtered := make([]model.Pricing, 0, len(pricing))
	for _, p := range pricing {
		if ratio_setting.ModelHasConfiguredPricing(p.ModelName) {
			filtered = append(filtered, p)
		}
	}
	channels, err := model.ListChannelsForPricing()
	if err != nil {
		channels = nil
	}
	visibleChannelIDs := make(map[int]struct{}, len(channels))
	for _, item := range channels {
		visibleChannelIDs[item.ChannelID] = struct{}{}
	}
	channelPricingMeta, err := model.ListChannelPricingMeta()
	if err != nil {
		channelPricingMeta = nil
	}
	return model.BuildPricingAPIItems(filtered, visibleChannelIDs, channelPricingMeta, true)
}

// CollectPricingShowableModelNames 返回 /pricing 接口前端可展示的模型名集合（与 GetPricing 同源条件）。
// 判定条件与 /pricing 完全一致：
//  1. 模型已配置定价（ratio_setting.ModelHasConfiguredPricing）。
//  2. 至少存在一个 (model, 可见渠道) 满足 model.BuildPricingAPIItems 的单测门禁
//     （ManualDisplayResponseTime>0 或 LastTestSuccess && LastResponseTime>0；该渠道若已有任何成功单测，则本模型也需通过模糊匹配）。
//
// 用于操练场等需要"配好定价 + 测试连通性通过"判定与定价页保持一致的位置，避免两端各自实现的判定门槛漂移导致少展示。
func CollectPricingShowableModelNames() map[string]bool {
	pricing := model.GetPricing()
	filtered := make([]model.Pricing, 0, len(pricing))
	for _, p := range pricing {
		if ratio_setting.ModelHasConfiguredPricing(p.ModelName) {
			filtered = append(filtered, p)
		}
	}
	visibleChannelIDs := make(map[int]struct{})
	if channels, err := model.ListChannelsForPricing(); err == nil {
		for _, item := range channels {
			visibleChannelIDs[item.ChannelID] = struct{}{}
		}
	}
	metas, err := model.ListChannelPricingMeta()
	if err != nil {
		metas = nil
	}
	items := model.BuildPricingAPIItems(filtered, visibleChannelIDs, metas, false)
	out := make(map[string]bool, len(items))
	for i := range items {
		name := strings.TrimSpace(items[i].ModelName)
		if name == "" {
			continue
		}
		out[name] = true
	}
	return out
}

func validateAdminIssuedToken(rawToken string) error {
	tokenKey := strings.TrimSpace(rawToken)
	if strings.HasPrefix(strings.ToLower(tokenKey), "bearer ") {
		tokenKey = strings.TrimSpace(tokenKey[7:])
	}
	tokenKey = strings.TrimPrefix(tokenKey, "sk-")
	token, err := model.ValidateUserToken(tokenKey)
	if err != nil {
		return err
	}
	if token == nil || !model.IsAdmin(token.UserId) {
		return errors.New("令牌不是管理员签发")
	}
	return nil
}

func PriceSync(c *gin.Context) {
	var req struct {
		Token string `json:"token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请求参数格式错误"})
		return
	}
	if err := validateAdminIssuedToken(req.Token); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "token 验证失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    buildPricingAPIData(),
	})
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
	_, visibleChannelIDs, err := getPricingVisibleChannelsForUser(c)
	if err != nil {
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
	channelVideoRatio := map[string]map[string]float64{}
	channelVideoCompletionRatio := map[string]map[string]float64{}
	channelVideoPrice := map[string]map[string]float64{}
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
	for channelID, modelRatio := range ratio_setting.GetChannelVideoRatioCopy() {
		channelVideoRatio[channelID] = modelRatio
	}
	for channelID, modelRatio := range ratio_setting.GetChannelVideoCompletionRatioCopy() {
		channelVideoCompletionRatio[channelID] = modelRatio
	}
	for channelID, modelPrice := range ratio_setting.GetChannelVideoPriceCopy() {
		channelVideoPrice[channelID] = modelPrice
	}
	channelModelPrice = filterChannelPricingMapByVisibleChannels(channelModelPrice, visibleChannelIDs)
	channelModelRatio = filterChannelPricingMapByVisibleChannels(channelModelRatio, visibleChannelIDs)
	channelCompletionRatio = filterChannelPricingMapByVisibleChannels(channelCompletionRatio, visibleChannelIDs)
	channelCacheRatio = filterChannelPricingMapByVisibleChannels(channelCacheRatio, visibleChannelIDs)
	channelCreateCacheRatio = filterChannelPricingMapByVisibleChannels(channelCreateCacheRatio, visibleChannelIDs)
	channelImageRatio = filterChannelPricingMapByVisibleChannels(channelImageRatio, visibleChannelIDs)
	channelAudioRatio = filterChannelPricingMapByVisibleChannels(channelAudioRatio, visibleChannelIDs)
	channelAudioCompletionRatio = filterChannelPricingMapByVisibleChannels(channelAudioCompletionRatio, visibleChannelIDs)
	channelVideoRatio = filterChannelPricingMapByVisibleChannels(channelVideoRatio, visibleChannelIDs)
	channelVideoCompletionRatio = filterChannelPricingMapByVisibleChannels(channelVideoCompletionRatio, visibleChannelIDs)
	channelVideoPrice = filterChannelPricingMapByVisibleChannels(channelVideoPrice, visibleChannelIDs)
	for supplierID, modelPrice := range ratio_setting.GetSupplierModelPriceCopy() {
		supplierModelPrice[supplierID] = modelPrice
	}
	for supplierID, modelRatio := range ratio_setting.GetSupplierModelRatioCopy() {
		supplierModelRatio[supplierID] = modelRatio
	}

	channelPricingMeta, err := model.ListChannelPricingMeta()
	if err != nil {
		channelPricingMeta = nil
	}
	pricingData := model.BuildPricingAPIItems(filtered, visibleChannelIDs, channelPricingMeta, false)

	blurPricing := false
	if !exists && shouldBlurPricing() {
		blurPricing = true
		sanitizePricingData(pricingData)
		groupRatio = map[string]float64{}
		groupModelPrice = map[string]map[string]float64{}
		groupModelRatio = map[string]map[string]float64{}
		channelModelPrice = map[string]map[string]float64{}
		channelModelRatio = map[string]map[string]float64{}
		channelCompletionRatio = map[string]map[string]float64{}
		channelCacheRatio = map[string]map[string]float64{}
		channelCreateCacheRatio = map[string]map[string]float64{}
		channelImageRatio = map[string]map[string]float64{}
		channelAudioRatio = map[string]map[string]float64{}
		channelAudioCompletionRatio = map[string]map[string]float64{}
		channelVideoRatio = map[string]map[string]float64{}
		channelVideoCompletionRatio = map[string]map[string]float64{}
		channelVideoPrice = map[string]map[string]float64{}
		supplierModelPrice = map[string]map[string]float64{}
		supplierModelRatio = map[string]map[string]float64{}
	}

	c.JSON(200, gin.H{
		"success":      true,
		"data":         pricingData,
		"blur_pricing": blurPricing,
		"vendors":      model.GetVendors(),
		// "channels":                       channels,
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
		"channel_video_ratio":            channelVideoRatio,
		"channel_video_completion_ratio": channelVideoCompletionRatio,
		"channel_video_price":            channelVideoPrice,
		"supplier_model_price":           supplierModelPrice,
		"supplier_model_ratio":           supplierModelRatio,
		"usable_group":                   usableGroup,
		"supported_endpoint":             model.GetSupportedEndpointMap(),
		"auto_groups":                    service.GetUserAutoGroup(group),
		"pricing_version":                "b58e1c9a3f7d4e2a8c0b1d6e9f4a2c7d8e0f1b2a3",
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
