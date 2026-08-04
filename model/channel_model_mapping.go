package model

import (
	"encoding/json"
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

// ApplyChannelModelMapping 按渠道 model_mapping 做链式重定向，返回最终模型名。
// mappingJSON 为空或未命中时原样返回 startModel；循环映射时停在已访问节点以免死循环。
func ApplyChannelModelMapping(mappingJSON, startModel string) string {
	startModel = strings.TrimSpace(startModel)
	mappingJSON = strings.TrimSpace(mappingJSON)
	if startModel == "" || mappingJSON == "" || mappingJSON == "{}" {
		return startModel
	}
	modelMap := make(map[string]string)
	if err := json.Unmarshal([]byte(mappingJSON), &modelMap); err != nil || len(modelMap) == 0 {
		return startModel
	}
	current := startModel
	visited := map[string]bool{current: true}
	for {
		next, exists := modelMap[current]
		if !exists || next == "" || next == current {
			break
		}
		if visited[next] {
			break
		}
		visited[next] = true
		current = next
	}
	return current
}

// ResolvePricingModelName 解析计价所用模型名：别名自身已配置定价则用别名，否则回退到映射链尾规范模型。
func ResolvePricingModelName(mappingJSON, originModel string) string {
	originModel = strings.TrimSpace(originModel)
	if originModel == "" {
		return originModel
	}
	if ratio_setting.ModelHasConfiguredPricing(originModel) {
		return originModel
	}
	canonical := ApplyChannelModelMapping(mappingJSON, originModel)
	if canonical != "" && canonical != originModel && ratio_setting.ModelHasConfiguredPricing(canonical) {
		return canonical
	}
	return originModel
}

// enabledAliasCanonical 由 updatePricing 刷新：alias → 具备定价配置的规范模型名。
var (
	enabledAliasCanonical     = map[string]string{}
	enabledAliasCanonicalLock sync.RWMutex
)

func setEnabledAliasCanonicalIndex(index map[string]string) {
	enabledAliasCanonicalLock.Lock()
	defer enabledAliasCanonicalLock.Unlock()
	if index == nil {
		enabledAliasCanonical = map[string]string{}
		return
	}
	enabledAliasCanonical = index
}

// ResolveDisplayPricingModelName 展示/门禁用：别名自身有定价则返回别名，否则用缓存的渠道映射规范模型。
func ResolveDisplayPricingModelName(modelName string) string {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return modelName
	}
	if ratio_setting.ModelHasConfiguredPricing(modelName) {
		return modelName
	}
	enabledAliasCanonicalLock.RLock()
	canonical := enabledAliasCanonical[modelName]
	enabledAliasCanonicalLock.RUnlock()
	if canonical != "" && ratio_setting.ModelHasConfiguredPricing(canonical) {
		return canonical
	}
	return modelName
}

// ModelHasDisplayConfiguredPricing 模型自身或可通过渠道映射继承到已配置定价的规范模型时返回 true。
func ModelHasDisplayConfiguredPricing(modelName string) bool {
	return ratio_setting.ModelHasConfiguredPricing(ResolveDisplayPricingModelName(modelName))
}

// LookupCachedAliasCanonical 返回缓存的别名→规范模型（无映射或不在缓存中返回空串）。
func LookupCachedAliasCanonical(alias string) string {
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return ""
	}
	enabledAliasCanonicalLock.RLock()
	defer enabledAliasCanonicalLock.RUnlock()
	return enabledAliasCanonical[alias]
}

// buildAliasCanonicalIndexFromMeta 扫描启用渠道映射，构造「无自身定价的别名 → 已配置定价的规范模型」。
func buildAliasCanonicalIndexFromMeta(metas []ChannelPricingMeta) map[string]string {
	out := make(map[string]string)
	for _, row := range metas {
		mapping := strings.TrimSpace(row.ModelMapping)
		if mapping == "" || mapping == "{}" {
			continue
		}
		modelMap := make(map[string]string)
		if err := json.Unmarshal([]byte(mapping), &modelMap); err != nil {
			continue
		}
		for alias := range modelMap {
			alias = strings.TrimSpace(alias)
			if alias == "" || ratio_setting.ModelHasConfiguredPricing(alias) {
				continue
			}
			if !ChannelModelsRawContains(row.Models, alias) {
				continue
			}
			canonical := ApplyChannelModelMapping(mapping, alias)
			if canonical == "" || canonical == alias || !ratio_setting.ModelHasConfiguredPricing(canonical) {
				continue
			}
			if _, exists := out[alias]; !exists {
				out[alias] = canonical
			}
		}
	}
	return out
}

// fillPricingFieldsFromRatioSetting 按模型名写入全局定价字段（固定价优先于倍率）。
func fillPricingFieldsFromRatioSetting(pricing *Pricing, modelName string) {
	if pricing == nil {
		return
	}
	modelPrice, findPrice := ratio_setting.GetModelPrice(modelName, false)
	if findPrice {
		pricing.ModelPrice = modelPrice
		pricing.QuotaType = 1
	} else {
		modelRatio, _, _ := ratio_setting.GetModelRatio(modelName)
		pricing.ModelRatio = modelRatio
		if ratio_setting.ContainsCompletionRatio(modelName) {
			cr := ratio_setting.GetCompletionRatio(modelName)
			pricing.CompletionRatio = &cr
		}
		pricing.QuotaType = 0
	}
	if cacheRatio, ok := ratio_setting.GetCacheRatio(modelName); ok {
		pricing.CacheRatio = &cacheRatio
	}
	if createCacheRatio, ok := ratio_setting.GetCreateCacheRatio(modelName); ok {
		pricing.CreateCacheRatio = &createCacheRatio
	}
	if rule, ok := ratio_setting.GetModelRequestTierPricing(modelName); ok && len(rule.Tiers) > 0 {
		pricing.RequestTierPricing = rule
		if pricing.QuotaType != 1 {
			pricing.QuotaType = 3
		}
	}
	if imageRatio, ok := ratio_setting.GetImageRatio(modelName); ok {
		pricing.ImageRatio = &imageRatio
	}
	if ratio_setting.ContainsAudioRatio(modelName) {
		audioRatio := ratio_setting.GetAudioRatio(modelName)
		pricing.AudioRatio = &audioRatio
	}
	if ratio_setting.ContainsAudioCompletionRatio(modelName) {
		audioCompletionRatio := ratio_setting.GetAudioCompletionRatio(modelName)
		pricing.AudioCompletionRatio = &audioCompletionRatio
	}
	if ratio_setting.ContainsVideoRatio(modelName) {
		videoRatio := ratio_setting.GetVideoRatio(modelName)
		pricing.VideoRatio = &videoRatio
	}
	if ratio_setting.ContainsVideoCompletionRatio(modelName) {
		videoCompletionRatio := ratio_setting.GetVideoCompletionRatio(modelName)
		pricing.VideoCompletionRatio = &videoCompletionRatio
	}
	if ratio_setting.ContainsVideoPrice(modelName) {
		videoPrice, _ := ratio_setting.GetVideoPrice(modelName)
		pricing.VideoPrice = &videoPrice
	}
}
