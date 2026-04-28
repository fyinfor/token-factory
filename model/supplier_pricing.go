package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"gorm.io/gorm"
)

// supplierPricingModelsJoined 收集一组映射中出现的全部模型名（用于批量入库）。
// NormalizeOwnedModelsForPricing 将自有模型名与 FormatMatching 规范化名一并纳入权限校验集合。
func NormalizeOwnedModelsForPricing(owned map[string]struct{}) map[string]struct{} {
	out := make(map[string]struct{}, len(owned)*2)
	for k := range owned {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		out[k] = struct{}{}
		out[ratio_setting.FormatMatchingModelName(k)] = struct{}{}
	}
	return out
}

func supplierPricingModelsJoined(maps map[string]map[string]float64) map[string]struct{} {
	out := make(map[string]struct{})
	for _, m := range maps {
		for name := range m {
			name = strings.TrimSpace(name)
			if name != "" {
				out[name] = struct{}{}
			}
		}
	}
	return out
}

// UpsertSupplierModelPricingMaps 将前端提交的模型→数值映射合并写入 supplier_model_pricings（覆盖同一供应商下同模型名）。
func UpsertSupplierModelPricingMaps(supplierApplicationID int, userID int, maps map[string]map[string]float64, ownedModels map[string]struct{}) error {
	modelNames := supplierPricingModelsJoined(maps)
	for name := range modelNames {
		if _, ok := ownedModels[name]; !ok {
			return fmt.Errorf("无权配置模型: %s", name)
		}
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		for rawName := range modelNames {
			modelName := ratio_setting.FormatMatchingModelName(rawName)
			row := SupplierModelPricing{
				SupplierApplicationID: supplierApplicationID,
				ModelName:             modelName,
				QuotaType:             0,
				UpdatedByUserID:       userID,
			}
			if v, ok := pickFloat(maps["ModelPrice"], rawName); ok {
				row.QuotaType = 1
				row.ModelPrice = floatPtr(v)
			}
			if v, ok := pickFloat(maps["ModelRatio"], rawName); ok {
				row.ModelRatio = floatPtr(v)
				if row.ModelPrice == nil {
					row.QuotaType = 0
				}
			}
			if v, ok := pickFloat(maps["CompletionRatio"], rawName); ok {
				row.CompletionRatio = floatPtr(v)
			}
			if v, ok := pickFloat(maps["CacheRatio"], rawName); ok {
				row.CacheRatio = floatPtr(v)
			}
			if v, ok := pickFloat(maps["CreateCacheRatio"], rawName); ok {
				row.CreateCacheRatio = floatPtr(v)
			}
			if v, ok := pickFloat(maps["ImageRatio"], rawName); ok {
				row.ImageRatio = floatPtr(v)
			}
			if v, ok := pickFloat(maps["AudioRatio"], rawName); ok {
				row.AudioRatio = floatPtr(v)
			}
			if v, ok := pickFloat(maps["AudioCompletionRatio"], rawName); ok {
				row.AudioCompletionRatio = floatPtr(v)
			}
			if row.ModelPrice == nil && row.ModelRatio == nil &&
				row.CompletionRatio == nil && row.CacheRatio == nil && row.CreateCacheRatio == nil &&
				row.ImageRatio == nil && row.AudioRatio == nil && row.AudioCompletionRatio == nil {
				if err := tx.Where("supplier_application_id = ? AND model_name = ?", supplierApplicationID, modelName).
					Delete(&SupplierModelPricing{}).Error; err != nil {
					return err
				}
				continue
			}
			var existing SupplierModelPricing
			err := tx.Where("supplier_application_id = ? AND model_name = ?", supplierApplicationID, modelName).First(&existing).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if err := tx.Create(&row).Error; err != nil {
					return err
				}
				continue
			}
			if err != nil {
				return err
			}
			row.ID = existing.ID
			if err := tx.Save(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// UpsertSupplierChannelModelPricingMaps 写入供应商渠道维度定价映射。
func UpsertSupplierChannelModelPricingMaps(supplierApplicationID int, channelID int, userID int, maps map[string]map[string]float64, ownedModels map[string]struct{}) error {
	modelNames := supplierPricingModelsJoined(maps)
	for name := range modelNames {
		if _, ok := ownedModels[name]; !ok {
			return fmt.Errorf("无权配置模型: %s", name)
		}
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		for rawName := range modelNames {
			modelName := ratio_setting.FormatMatchingModelName(rawName)
			row := SupplierChannelModelPricing{
				SupplierApplicationID: supplierApplicationID,
				ChannelID:             channelID,
				ModelName:             modelName,
				QuotaType:             0,
				UpdatedByUserID:       userID,
			}
			if v, ok := pickFloat(maps["ModelPrice"], rawName); ok {
				row.QuotaType = 1
				row.ModelPrice = floatPtr(v)
			}
			if v, ok := pickFloat(maps["ModelRatio"], rawName); ok {
				row.ModelRatio = floatPtr(v)
				if row.ModelPrice == nil {
					row.QuotaType = 0
				}
			}
			if v, ok := pickFloat(maps["CompletionRatio"], rawName); ok {
				row.CompletionRatio = floatPtr(v)
			}
			if v, ok := pickFloat(maps["CacheRatio"], rawName); ok {
				row.CacheRatio = floatPtr(v)
			}
			if v, ok := pickFloat(maps["CreateCacheRatio"], rawName); ok {
				row.CreateCacheRatio = floatPtr(v)
			}
			if v, ok := pickFloat(maps["ImageRatio"], rawName); ok {
				row.ImageRatio = floatPtr(v)
			}
			if v, ok := pickFloat(maps["AudioRatio"], rawName); ok {
				row.AudioRatio = floatPtr(v)
			}
			if v, ok := pickFloat(maps["AudioCompletionRatio"], rawName); ok {
				row.AudioCompletionRatio = floatPtr(v)
			}
			if row.ModelPrice == nil && row.ModelRatio == nil &&
				row.CompletionRatio == nil && row.CacheRatio == nil && row.CreateCacheRatio == nil &&
				row.ImageRatio == nil && row.AudioRatio == nil && row.AudioCompletionRatio == nil {
				if err := tx.Where("supplier_application_id = ? AND channel_id = ? AND model_name = ?", supplierApplicationID, channelID, modelName).
					Delete(&SupplierChannelModelPricing{}).Error; err != nil {
					return err
				}
				continue
			}
			var existing SupplierChannelModelPricing
			err := tx.Where("supplier_application_id = ? AND channel_id = ? AND model_name = ?", supplierApplicationID, channelID, modelName).First(&existing).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if err := tx.Create(&row).Error; err != nil {
					return err
				}
				continue
			}
			if err != nil {
				return err
			}
			row.ID = existing.ID
			if err := tx.Save(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func pickFloat(m map[string]float64, key string) (float64, bool) {
	if m == nil {
		return 0, false
	}
	v, ok := m[key]
	return v, ok
}

func floatPtr(v float64) *float64 {
	return &v
}

// GetSupplierModelPricingRow 读取供应商全局定价单行（model_name 存库为 FormatMatching 规范化值）。
func GetSupplierModelPricingRow(supplierApplicationID int, rawModelName string) (*SupplierModelPricing, error) {
	if supplierApplicationID <= 0 {
		return nil, nil
	}
	key := ratio_setting.FormatMatchingModelName(rawModelName)
	var row SupplierModelPricing
	err := DB.Where("supplier_application_id = ? AND model_name = ?", supplierApplicationID, key).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// GetSupplierChannelModelPricingRow 读取供应商渠道定价单行。
func GetSupplierChannelModelPricingRow(supplierApplicationID int, channelID int, rawModelName string) (*SupplierChannelModelPricing, error) {
	if supplierApplicationID <= 0 || channelID <= 0 {
		return nil, nil
	}
	key := ratio_setting.FormatMatchingModelName(rawModelName)
	var row SupplierChannelModelPricing
	err := DB.Where("supplier_application_id = ? AND channel_id = ? AND model_name = ?", supplierApplicationID, channelID, key).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// ListSupplierModelPricingsForSupplier 列出某供应商全部全局定价行（用于 GET API）。
func ListSupplierModelPricingsForSupplier(supplierApplicationID int) ([]SupplierModelPricing, error) {
	var rows []SupplierModelPricing
	err := DB.Where("supplier_application_id = ?", supplierApplicationID).Order("model_name asc").Find(&rows).Error
	return rows, err
}

// ListSupplierChannelModelPricings 列出某渠道下全部供应商渠道定价行。
func ListSupplierChannelModelPricings(supplierApplicationID int, channelID int) ([]SupplierChannelModelPricing, error) {
	var rows []SupplierChannelModelPricing
	err := DB.Where("supplier_application_id = ? AND channel_id = ?", supplierApplicationID, channelID).Order("model_name asc").Find(&rows).Error
	return rows, err
}

// BuildOptionLikeMapsFromSupplierGlobalRows 将表行转为与 Option JSON 相同结构的 map（便于前端复用编辑器）。
func BuildOptionLikeMapsFromSupplierGlobalRows(rows []SupplierModelPricing) map[string]map[string]float64 {
	out := map[string]map[string]float64{
		"ModelPrice":           {},
		"ModelRatio":           {},
		"CompletionRatio":      {},
		"CacheRatio":           {},
		"CreateCacheRatio":     {},
		"ImageRatio":           {},
		"AudioRatio":           {},
		"AudioCompletionRatio": {},
	}
	for _, r := range rows {
		name := r.ModelName
		if r.ModelPrice != nil {
			out["ModelPrice"][name] = *r.ModelPrice
		}
		if r.ModelRatio != nil {
			out["ModelRatio"][name] = *r.ModelRatio
		}
		if r.CompletionRatio != nil {
			out["CompletionRatio"][name] = *r.CompletionRatio
		}
		if r.CacheRatio != nil {
			out["CacheRatio"][name] = *r.CacheRatio
		}
		if r.CreateCacheRatio != nil {
			out["CreateCacheRatio"][name] = *r.CreateCacheRatio
		}
		if r.ImageRatio != nil {
			out["ImageRatio"][name] = *r.ImageRatio
		}
		if r.AudioRatio != nil {
			out["AudioRatio"][name] = *r.AudioRatio
		}
		if r.AudioCompletionRatio != nil {
			out["AudioCompletionRatio"][name] = *r.AudioCompletionRatio
		}
	}
	return out
}

// BuildOptionLikeMapsFromSupplierChannelRows 将渠道定价行转为 map。
func BuildOptionLikeMapsFromSupplierChannelRows(rows []SupplierChannelModelPricing) map[string]map[string]float64 {
	out := map[string]map[string]float64{
		"ModelPrice":           {},
		"ModelRatio":           {},
		"CompletionRatio":      {},
		"CacheRatio":           {},
		"CreateCacheRatio":     {},
		"ImageRatio":           {},
		"AudioRatio":           {},
		"AudioCompletionRatio": {},
	}
	for _, r := range rows {
		name := r.ModelName
		if r.ModelPrice != nil {
			out["ModelPrice"][name] = *r.ModelPrice
		}
		if r.ModelRatio != nil {
			out["ModelRatio"][name] = *r.ModelRatio
		}
		if r.CompletionRatio != nil {
			out["CompletionRatio"][name] = *r.CompletionRatio
		}
		if r.CacheRatio != nil {
			out["CacheRatio"][name] = *r.CacheRatio
		}
		if r.CreateCacheRatio != nil {
			out["CreateCacheRatio"][name] = *r.CreateCacheRatio
		}
		if r.ImageRatio != nil {
			out["ImageRatio"][name] = *r.ImageRatio
		}
		if r.AudioRatio != nil {
			out["AudioRatio"][name] = *r.AudioRatio
		}
		if r.AudioCompletionRatio != nil {
			out["AudioCompletionRatio"][name] = *r.AudioCompletionRatio
		}
	}
	return out
}

// ResolveSupplierScopedFixedModelPrice 解析按次固定价（$/次）：供应商渠道表 > 供应商全局表 > Option 渠道价 > 平台全局 > 旧 SupplierModelPrice。
func ResolveSupplierScopedFixedModelPrice(channelID int, supplierApplicationID int, modelName string) (float64, bool) {
	modelName = ratio_setting.FormatMatchingModelName(modelName)
	if supplierApplicationID > 0 && channelID > 0 {
		if row, _ := GetSupplierChannelModelPricingRow(supplierApplicationID, channelID, modelName); row != nil && row.ModelPrice != nil {
			return *row.ModelPrice, true
		}
		if row, _ := GetSupplierModelPricingRow(supplierApplicationID, modelName); row != nil && row.ModelPrice != nil {
			return *row.ModelPrice, true
		}
	}
	if v, ok := ratio_setting.GetChannelModelPrice(channelID, modelName); ok {
		return v, true
	}
	if v, ok := ratio_setting.GetModelPrice(modelName, false); ok {
		return v, true
	}
	if supplierApplicationID > 0 {
		if v, ok := ratio_setting.GetSupplierModelPrice(supplierApplicationID, modelName); ok {
			return v, true
		}
	}
	return 0, false
}

// ResolveSupplierScopedModelRatio 解析输入倍率：供应商渠道表 > 供应商全局表 > Option 渠道倍率 > 旧 SupplierModelRatio > 平台全局。
func ResolveSupplierScopedModelRatio(channelID int, supplierApplicationID int, modelName string) (float64, bool, string) {
	modelName = ratio_setting.FormatMatchingModelName(modelName)
	if supplierApplicationID > 0 && channelID > 0 {
		if row, _ := GetSupplierChannelModelPricingRow(supplierApplicationID, channelID, modelName); row != nil && row.ModelRatio != nil {
			return *row.ModelRatio, true, ""
		}
		if row, _ := GetSupplierModelPricingRow(supplierApplicationID, modelName); row != nil && row.ModelRatio != nil {
			return *row.ModelRatio, true, ""
		}
	}
	if v, ok := ratio_setting.GetChannelModelRatio(channelID, modelName); ok {
		return v, true, ""
	}
	if supplierApplicationID > 0 {
		if v, ok := ratio_setting.GetSupplierModelRatio(supplierApplicationID, modelName); ok {
			return v, true, ""
		}
	}
	return ratio_setting.GetModelRatio(modelName)
}

// ResolveSupplierScopedCompletionRatio 解析输出倍率：供应商渠道行 > 供应商全局行 > Option 渠道 > 平台全局。
func ResolveSupplierScopedCompletionRatio(channelID int, supplierApplicationID int, modelName string) float64 {
	modelName = ratio_setting.FormatMatchingModelName(modelName)
	if supplierApplicationID > 0 && channelID > 0 {
		if row, _ := GetSupplierChannelModelPricingRow(supplierApplicationID, channelID, modelName); row != nil && row.CompletionRatio != nil {
			return *row.CompletionRatio
		}
		if row, _ := GetSupplierModelPricingRow(supplierApplicationID, modelName); row != nil && row.CompletionRatio != nil {
			return *row.CompletionRatio
		}
	}
	if v, ok := ratio_setting.GetChannelCompletionRatio(channelID, modelName); ok {
		return v
	}
	return ratio_setting.GetCompletionRatio(modelName)
}

// ResolveSupplierScopedImageRatio 供应商渠道/全局行优先，其次 Option 渠道与平台全局。
func ResolveSupplierScopedImageRatio(channelID int, supplierApplicationID int, modelName string) (float64, bool) {
	modelName = ratio_setting.FormatMatchingModelName(modelName)
	if supplierApplicationID > 0 && channelID > 0 {
		if row, _ := GetSupplierChannelModelPricingRow(supplierApplicationID, channelID, modelName); row != nil && row.ImageRatio != nil {
			return *row.ImageRatio, true
		}
	}
	if supplierApplicationID > 0 {
		if row, _ := GetSupplierModelPricingRow(supplierApplicationID, modelName); row != nil && row.ImageRatio != nil {
			return *row.ImageRatio, true
		}
	}
	if v, ok := ratio_setting.GetChannelImageRatio(channelID, modelName); ok {
		return v, true
	}
	return ratio_setting.GetImageRatio(modelName)
}

// ResolveSupplierScopedAudioRatio 音频输入倍率。
func ResolveSupplierScopedAudioRatio(channelID int, supplierApplicationID int, modelName string) float64 {
	modelName = ratio_setting.FormatMatchingModelName(modelName)
	if supplierApplicationID > 0 && channelID > 0 {
		if row, _ := GetSupplierChannelModelPricingRow(supplierApplicationID, channelID, modelName); row != nil && row.AudioRatio != nil {
			return *row.AudioRatio
		}
	}
	if supplierApplicationID > 0 {
		if row, _ := GetSupplierModelPricingRow(supplierApplicationID, modelName); row != nil && row.AudioRatio != nil {
			return *row.AudioRatio
		}
	}
	if v, ok := ratio_setting.GetChannelAudioRatio(channelID, modelName); ok {
		return v
	}
	return ratio_setting.GetAudioRatio(modelName)
}

// ResolveSupplierScopedAudioCompletionRatio 音频输出相对输入倍率。
func ResolveSupplierScopedAudioCompletionRatio(channelID int, supplierApplicationID int, modelName string) float64 {
	modelName = ratio_setting.FormatMatchingModelName(modelName)
	if supplierApplicationID > 0 && channelID > 0 {
		if row, _ := GetSupplierChannelModelPricingRow(supplierApplicationID, channelID, modelName); row != nil && row.AudioCompletionRatio != nil {
			return *row.AudioCompletionRatio
		}
	}
	if supplierApplicationID > 0 {
		if row, _ := GetSupplierModelPricingRow(supplierApplicationID, modelName); row != nil && row.AudioCompletionRatio != nil {
			return *row.AudioCompletionRatio
		}
	}
	if v, ok := ratio_setting.GetChannelAudioCompletionRatio(channelID, modelName); ok {
		return v
	}
	return ratio_setting.GetAudioCompletionRatio(modelName)
}

// ResolveSupplierScopedCacheRatios 解析缓存读写倍率（供应商渠道/全局行优先）。
func ResolveSupplierScopedCacheRatios(channelID int, supplierApplicationID int, modelName string) (cacheRatio float64, createCacheRatio float64) {
	modelName = ratio_setting.FormatMatchingModelName(modelName)
	var chRow *SupplierChannelModelPricing
	var glRow *SupplierModelPricing
	if supplierApplicationID > 0 && channelID > 0 {
		chRow, _ = GetSupplierChannelModelPricingRow(supplierApplicationID, channelID, modelName)
	}
	if supplierApplicationID > 0 {
		glRow, _ = GetSupplierModelPricingRow(supplierApplicationID, modelName)
	}
	if chRow != nil && chRow.CacheRatio != nil {
		cacheRatio = *chRow.CacheRatio
	} else if glRow != nil && glRow.CacheRatio != nil {
		cacheRatio = *glRow.CacheRatio
	} else if v, ok := ratio_setting.GetChannelCacheRatio(channelID, modelName); ok {
		cacheRatio = v
	} else {
		cacheRatio, _ = ratio_setting.GetCacheRatio(modelName)
	}

	if chRow != nil && chRow.CreateCacheRatio != nil {
		createCacheRatio = *chRow.CreateCacheRatio
	} else if glRow != nil && glRow.CreateCacheRatio != nil {
		createCacheRatio = *glRow.CreateCacheRatio
	} else if v, ok := ratio_setting.GetChannelCreateCacheRatio(channelID, modelName); ok {
		createCacheRatio = v
	} else {
		createCacheRatio, _ = ratio_setting.GetCreateCacheRatio(modelName)
	}
	return cacheRatio, createCacheRatio
}
