package controller

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

// supplierEditableModelOptionKeys 定义供应商可操作的模型倍率相关配置键。
var supplierEditableModelOptionKeys = map[string]struct{}{
	"ModelPrice":           {},
	"ModelRatio":           {},
	"CompletionRatio":      {},
	"CacheRatio":           {},
	"CreateCacheRatio":     {},
	"ImageRatio":           {},
	"AudioRatio":           {},
	"AudioCompletionRatio": {},
	"VideoRatio":           {},
	"VideoCompletionRatio": {},
	"VideoPrice":           {},
}

// collectSupplierOwnedModelNames 收集供应商名下渠道与模型中的模型名集合。
func collectSupplierOwnedModelNames(userID int) (map[string]struct{}, error) {
	ownedModels := make(map[string]struct{})

	channels, _, err := model.SearchSupplierChannels(&userID, 0, 100000, model.SupplierChannelSearchFilter{})
	if err != nil {
		return nil, err
	}
	for _, channel := range channels {
		for _, modelName := range channel.GetModels() {
			modelName = strings.TrimSpace(modelName)
			if modelName == "" {
				continue
			}
			ownedModels[modelName] = struct{}{}
		}
	}

	models, _, err := model.SearchSupplierModels(&userID, "", "", 0, 100000)
	if err != nil {
		return nil, err
	}
	for _, item := range models {
		modelName := strings.TrimSpace(item.ModelName)
		if modelName == "" {
			continue
		}
		ownedModels[modelName] = struct{}{}
	}

	return ownedModels, nil
}

// collectAllSupplierOwnedModelNames 收集全部供应商名下的模型名集合（管理员统计用）。
func collectAllSupplierOwnedModelNames() (map[string]struct{}, error) {
	ownedModels := make(map[string]struct{})

	channels, _, err := model.SearchSupplierChannels(nil, 0, 100000, model.SupplierChannelSearchFilter{})
	if err != nil {
		return nil, err
	}
	for _, channel := range channels {
		for _, modelName := range channel.GetModels() {
			modelName = strings.TrimSpace(modelName)
			if modelName == "" {
				continue
			}
			ownedModels[modelName] = struct{}{}
		}
	}

	models, _, err := model.SearchSupplierModels(nil, "", "", 0, 100000)
	if err != nil {
		return nil, err
	}
	for _, item := range models {
		modelName := strings.TrimSpace(item.ModelName)
		if modelName == "" {
			continue
		}
		ownedModels[modelName] = struct{}{}
	}
	return ownedModels, nil
}

// filterModelJSONByOwnedModels 仅保留属于供应商自有模型的 JSON 键值。
func filterModelJSONByOwnedModels(raw string, ownedModels map[string]struct{}) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "{}", nil
	}
	var origin map[string]any
	if err := common.UnmarshalJsonStr(raw, &origin); err != nil {
		return "", err
	}
	filtered := make(map[string]any)
	for modelName, value := range origin {
		if _, ok := ownedModels[modelName]; !ok {
			continue
		}
		filtered[modelName] = value
	}
	bytes, err := common.Marshal(filtered)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// mergeModelJSONByOwnedModels 仅允许供应商更新自有模型键，其余键保持原值。
func mergeModelJSONByOwnedModels(currentRaw string, incomingRaw string, ownedModels map[string]struct{}) (string, error) {
	base := make(map[string]any)
	currentRaw = strings.TrimSpace(currentRaw)
	if currentRaw != "" {
		if err := common.UnmarshalJsonStr(currentRaw, &base); err != nil {
			return "", err
		}
	}

	patch := make(map[string]any)
	if err := common.UnmarshalJsonStr(strings.TrimSpace(incomingRaw), &patch); err != nil {
		return "", err
	}
	for modelName, value := range patch {
		if _, ok := ownedModels[modelName]; !ok {
			continue
		}
		base[modelName] = value
	}
	bytes, err := common.Marshal(base)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
