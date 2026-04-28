package helper

import (
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

// https://docs.claude.com/en/docs/build-with-claude/prompt-caching#1-hour-cache-duration
const claudeCacheCreation1hMultiplier = 6 / 3.75

func resolveSupplierIDByChannel(info *relaycommon.RelayInfo) int {
	if info == nil || info.ChannelMeta == nil || info.ChannelId <= 0 {
		return 0
	}
	channel, err := model.CacheGetChannel(info.ChannelId)
	if err != nil || channel == nil {
		return 0
	}
	return channel.SupplierApplicationID
}

// HandleGroupRatio checks for "auto_group" in the context and updates the group ratio and relayInfo.UsingGroup if present
func HandleGroupRatio(ctx *gin.Context, relayInfo *relaycommon.RelayInfo) types.GroupRatioInfo {
	groupRatioInfo := types.GroupRatioInfo{
		GroupRatio:        1.0, // default ratio
		GroupSpecialRatio: -1,
	}

	// check auto group
	autoGroup, exists := ctx.Get("auto_group")
	if exists {
		logger.LogDebug(ctx, fmt.Sprintf("final group: %s", autoGroup))
		relayInfo.UsingGroup = autoGroup.(string)
	}

	// check user group special ratio
	userGroupRatio, ok := ratio_setting.GetGroupGroupRatio(relayInfo.UserGroup, relayInfo.UsingGroup)
	if ok {
		// user group special ratio
		groupRatioInfo.GroupSpecialRatio = userGroupRatio
		groupRatioInfo.GroupRatio = userGroupRatio
		groupRatioInfo.HasSpecialRatio = true
	} else {
		// normal group ratio
		groupRatioInfo.GroupRatio = ratio_setting.GetGroupRatio(relayInfo.UsingGroup)
	}

	return groupRatioInfo
}

func ModelPriceHelper(c *gin.Context, info *relaycommon.RelayInfo, promptTokens int, meta *types.TokenCountMeta) (types.PriceData, error) {
	if info == nil {
		return types.PriceData{}, fmt.Errorf("relay info is nil")
	}
	channelID := 0
	if info.ChannelMeta != nil {
		channelID = info.ChannelId
	}
	groupRatioInfo := HandleGroupRatio(c, info)
	supplierID := resolveSupplierIDByChannel(info)
	modelPrice, usePrice := model.ResolveSupplierScopedFixedModelPrice(channelID, supplierID, info.OriginModelName)
	if groupPrice, ok := ratio_setting.GetGroupModelPrice(info.UsingGroup, info.OriginModelName); ok {
		modelPrice = groupPrice
		usePrice = true
	}
	channelVideoRatio, hasChannelVideoRatio := ratio_setting.GetChannelVideoRatio(channelID, info.OriginModelName)
	channelVideoCompletionRatio, hasChannelVideoCompletionRatio := ratio_setting.GetChannelVideoCompletionRatio(channelID, info.OriginModelName)
	channelCompletionRatio, hasChannelCompletionRatio := ratio_setting.GetChannelCompletionRatio(channelID, info.OriginModelName)
	channelCacheRatio, hasChannelCacheRatio := ratio_setting.GetChannelCacheRatio(channelID, info.OriginModelName)
	channelCreateCacheRatio, hasChannelCreateCacheRatio := ratio_setting.GetChannelCreateCacheRatio(channelID, info.OriginModelName)
	channelImageRatio, hasChannelImageRatio := ratio_setting.GetChannelImageRatio(channelID, info.OriginModelName)
	channelAudioRatio, hasChannelAudioRatio := ratio_setting.GetChannelAudioRatio(channelID, info.OriginModelName)
	channelAudioCompletionRatio, hasChannelAudioCompletionRatio := ratio_setting.GetChannelAudioCompletionRatio(channelID, info.OriginModelName)

	var preConsumedQuota int
	var modelRatio float64
	var completionRatio float64
	var cacheRatio float64
	var imageRatio float64
	var cacheCreationRatio float64
	var cacheCreationRatio5m float64
	var cacheCreationRatio1h float64
	var audioRatio float64
	var audioCompletionRatio float64
	var videoRatio float64
	var videoCompletionRatio float64
	var freeModel bool
	if !usePrice {
		preConsumedTokens := common.Max(promptTokens, common.PreConsumedQuota)
		if meta.MaxTokens != 0 {
			preConsumedTokens += meta.MaxTokens
		}
		var success bool
		var matchName string
		modelRatio, success, matchName = model.ResolveSupplierScopedModelRatio(channelID, supplierID, info.OriginModelName)
		if groupModelRatio, ok := ratio_setting.GetGroupModelRatio(info.UsingGroup, info.OriginModelName); ok {
			modelRatio = groupModelRatio
			success = true
		}
		if !success {
			acceptUnsetRatio := false
			if info.UserSetting.AcceptUnsetRatioModel {
				acceptUnsetRatio = true
			}
			if !acceptUnsetRatio {
				return types.PriceData{}, fmt.Errorf("模型 %s 倍率或价格未配置，请联系管理员设置或开始自用模式；Model %s ratio or price not set, please set or start self-use mode", matchName, matchName)
			}
		}
		completionRatio = model.ResolveSupplierScopedCompletionRatio(channelID, supplierID, info.OriginModelName)
		cacheRatio, cacheCreationRatio = model.ResolveSupplierScopedCacheRatios(channelID, supplierID, info.OriginModelName)
		cacheCreationRatio5m = cacheCreationRatio
		// 固定1h和5min缓存写入价格的比例
		cacheCreationRatio1h = cacheCreationRatio * claudeCacheCreation1hMultiplier

		imageRatio, _ = ratio_setting.GetImageRatio(info.OriginModelName)
		audioRatio = ratio_setting.GetAudioRatio(info.OriginModelName)
		audioCompletionRatio = ratio_setting.GetAudioCompletionRatio(info.OriginModelName)
		videoRatio = ratio_setting.GetVideoRatio(info.OriginModelName)
		videoCompletionRatio = ratio_setting.GetVideoCompletionRatio(info.OriginModelName)
		if hasChannelCompletionRatio {
			completionRatio = channelCompletionRatio
		}
		if hasChannelCacheRatio {
			cacheRatio = channelCacheRatio
		}
		if hasChannelCreateCacheRatio {
			cacheCreationRatio = channelCreateCacheRatio
		}
		if hasChannelImageRatio {
			imageRatio = channelImageRatio
		}
		if hasChannelAudioRatio {
			audioRatio = channelAudioRatio
		}
		if hasChannelAudioCompletionRatio {
			audioCompletionRatio = channelAudioCompletionRatio
		}

		if hasChannelVideoRatio {
			videoRatio = channelVideoRatio
		}
		if hasChannelVideoCompletionRatio {
			videoCompletionRatio = channelVideoCompletionRatio
		}
		ratio := modelRatio * groupRatioInfo.GroupRatio
		preConsumedQuota = int(float64(preConsumedTokens) * ratio)
	} else {
		if meta.ImagePriceRatio != 0 {
			modelPrice = modelPrice * meta.ImagePriceRatio
		}
		preConsumedQuota = int(modelPrice * common.QuotaPerUnit * groupRatioInfo.GroupRatio)
	}

	// check if free model pre-consume is disabled
	if !operation_setting.GetQuotaSetting().EnableFreeModelPreConsume {
		// if model price or ratio is 0, do not pre-consume quota
		if groupRatioInfo.GroupRatio == 0 {
			preConsumedQuota = 0
			freeModel = true
		} else if usePrice {
			if modelPrice == 0 {
				preConsumedQuota = 0
				freeModel = true
			}
		} else {
			if modelRatio == 0 {
				preConsumedQuota = 0
				freeModel = true
			}
		}
	}

	chDisc := model.ResolveChannelPriceDiscountPercent(channelID)
	chDiscCopy := chDisc
	preConsumedQuota = model.ApplyChannelPriceDiscountToQuota(preConsumedQuota, chDisc)

	priceData := types.PriceData{
		FreeModel:            freeModel,
		ModelPrice:           modelPrice,
		ModelRatio:           modelRatio,
		CompletionRatio:      completionRatio,
		GroupRatioInfo:       groupRatioInfo,
		UsePrice:             usePrice,
		CacheRatio:           cacheRatio,
		ImageRatio:           imageRatio,
		AudioRatio:           audioRatio,
		AudioCompletionRatio: audioCompletionRatio,
		VideoRatio:           videoRatio,
		VideoCompletionRatio: videoCompletionRatio,
		CacheCreationRatio:   cacheCreationRatio,
		CacheCreation5mRatio: cacheCreationRatio5m,
		CacheCreation1hRatio: cacheCreationRatio1h,
		ChannelPriceDiscount: &chDiscCopy,
		QuotaToPreConsume:    preConsumedQuota,
	}

	if common.DebugEnabled {
		println(fmt.Sprintf("model_price_helper result: %s", priceData.ToSetting()))
	}
	info.PriceData = priceData
	return priceData, nil
}

// ModelPriceHelperPerCall 按次计费的 PriceHelper (MJ、Task)
func ModelPriceHelperPerCall(c *gin.Context, info *relaycommon.RelayInfo) (types.PriceData, error) {
	if info == nil {
		return types.PriceData{}, fmt.Errorf("relay info is nil")
	}
	channelID := 0
	if info.ChannelMeta != nil {
		channelID = info.ChannelId
	}
	groupRatioInfo := HandleGroupRatio(c, info)

	supplierID := resolveSupplierIDByChannel(info)
	modelPrice, success := model.ResolveSupplierScopedFixedModelPrice(channelID, supplierID, info.OriginModelName)
	if groupPrice, ok := ratio_setting.GetGroupModelPrice(info.UsingGroup, info.OriginModelName); ok {
		modelPrice = groupPrice
		success = true
	}
	// 如果没有配置价格，检查模型倍率配置
	if !success {

		// 没有配置费用，也要使用默认费用,否则按费率计费模型无法使用
		defaultPrice, ok := ratio_setting.GetDefaultModelPriceMap()[info.OriginModelName]
		if ok {
			modelPrice = defaultPrice
		} else {
			// 没有配置倍率也不接受没配置,那就返回错误
			_, ratioSuccess, matchName := model.ResolveSupplierScopedModelRatio(channelID, supplierID, info.OriginModelName)
			acceptUnsetRatio := false
			if info.UserSetting.AcceptUnsetRatioModel {
				acceptUnsetRatio = true
			}
			if !ratioSuccess && !acceptUnsetRatio {
				return types.PriceData{}, fmt.Errorf("模型 %s 倍率或价格未配置，请联系管理员设置或开始自用模式；Model %s ratio or price not set, please set or start self-use mode", matchName, matchName)
			}
			// 未配置价格但配置了倍率，使用默认预扣价格
			modelPrice = float64(common.PreConsumedQuota) / common.QuotaPerUnit
		}

	}
	quota := int(modelPrice * common.QuotaPerUnit * groupRatioInfo.GroupRatio)

	// 免费模型检测（与 ModelPriceHelper 对齐）
	freeModel := false
	if !operation_setting.GetQuotaSetting().EnableFreeModelPreConsume {
		if groupRatioInfo.GroupRatio == 0 || modelPrice == 0 {
			quota = 0
			freeModel = true
		}
	}
	chDisc := model.ResolveChannelPriceDiscountPercent(channelID)
	chDiscCopy := chDisc
	quota = model.ApplyChannelPriceDiscountToQuota(quota, chDisc)

	priceData := types.PriceData{
		FreeModel:            freeModel,
		ModelPrice:           modelPrice,
		Quota:                quota,
		GroupRatioInfo:       groupRatioInfo,
		ChannelPriceDiscount: &chDiscCopy,
	}
	return priceData, nil
}

func ContainPriceOrRatio(modelName string) bool {
	_, ok := ratio_setting.GetModelPrice(modelName, false)
	if ok {
		return true
	}
	_, ok, _ = ratio_setting.GetModelRatio(modelName)
	if ok {
		return true
	}
	return false
}
