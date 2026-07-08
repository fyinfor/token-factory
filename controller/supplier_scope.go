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
	"VideoPricingRules":    {},
	"ImagePrice":           {},
	"ImagePricingRules":    {},
}

// supplierDashboardScope 供应商看板范围：统计这些渠道上的全部模型消费（不按模型名录截断）。
type supplierDashboardScope struct {
	ChannelIDs           []int
	ConfiguredModelNames map[string]struct{}
}

func mergeChannelModelsIntoScope(scope *supplierDashboardScope, channels []*model.Channel) {
	seenChannel := make(map[int]struct{}, len(channels))
	for _, channel := range channels {
		if channel != nil && channel.Id > 0 {
			if _, ok := seenChannel[channel.Id]; !ok {
				seenChannel[channel.Id] = struct{}{}
				scope.ChannelIDs = append(scope.ChannelIDs, channel.Id)
			}
		}
		for _, modelName := range channel.GetModels() {
			modelName = strings.TrimSpace(modelName)
			if modelName == "" {
				continue
			}
			scope.ConfiguredModelNames[modelName] = struct{}{}
		}
	}
}

func mergeSupplierModelsIntoScope(scope *supplierDashboardScope, models []*model.Model) {
	for _, item := range models {
		if item == nil {
			continue
		}
		modelName := strings.TrimSpace(item.ModelName)
		if modelName == "" {
			continue
		}
		scope.ConfiguredModelNames[modelName] = struct{}{}
	}
}

func appendChannelsBySupplierApplicationID(scope *supplierDashboardScope, supplierApplicationID int) error {
	if supplierApplicationID <= 0 {
		return nil
	}
	var channels []*model.Channel
	if err := model.DB.Where("supplier_application_id = ?", supplierApplicationID).
		Omit("key").Find(&channels).Error; err != nil {
		return err
	}
	mergeChannelModelsIntoScope(scope, channels)
	return nil
}

// supplierApplicationIDsForUser 收集用户关联的供应商申请 ID（users.supplier_id 与审核通过申请均计入）。
func supplierApplicationIDsForUser(userID int) ([]int, error) {
	seen := make(map[int]struct{})
	add := func(id int) {
		if id > 0 {
			seen[id] = struct{}{}
		}
	}
	if user, err := model.GetUserById(userID, false); err == nil && user != nil {
		add(user.SupplierID)
	}
	if app, err := model.GetApprovedSupplierApplicationByApplicant(userID); err == nil && app != nil {
		add(int(app.ID))
	}
	ids := make([]int, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	return ids, nil
}

// collectSupplierDashboardScope 收集指定供应商对接人名下的全部渠道 ID（owner + supplier_application_id）。
func collectSupplierDashboardScope(userID int) (supplierDashboardScope, error) {
	scope := supplierDashboardScope{
		ConfiguredModelNames: make(map[string]struct{}),
	}
	channels, _, err := model.SearchSupplierChannels(&userID, 0, 100000, model.SupplierChannelSearchFilter{})
	if err != nil {
		return scope, err
	}
	mergeChannelModelsIntoScope(&scope, channels)

	appIDs, err := supplierApplicationIDsForUser(userID)
	if err != nil {
		return scope, err
	}
	for _, appID := range appIDs {
		if err := appendChannelsBySupplierApplicationID(&scope, appID); err != nil {
			return scope, err
		}
	}

	models, _, err := model.SearchSupplierModels(&userID, "", "", "", "", 0, 100000)
	if err != nil {
		return scope, err
	}
	mergeSupplierModelsIntoScope(&scope, models)
	return scope, nil
}

// collectAllSupplierDashboardScope 收集全部供应商渠道（管理员汇总视图）。
func collectAllSupplierDashboardScope() (supplierDashboardScope, error) {
	scope := supplierDashboardScope{
		ConfiguredModelNames: make(map[string]struct{}),
	}
	channels, _, err := model.SearchSupplierChannels(nil, 0, 100000, model.SupplierChannelSearchFilter{})
	if err != nil {
		return scope, err
	}
	mergeChannelModelsIntoScope(&scope, channels)

	models, _, err := model.SearchSupplierModels(nil, "", "", "", "", 0, 100000)
	if err != nil {
		return scope, err
	}
	mergeSupplierModelsIntoScope(&scope, models)
	return scope, nil
}

// collectSupplierDashboardScopeBySupplierID 按供应商申请 ID 收集看板范围。
func collectSupplierDashboardScopeBySupplierID(supplierID int) (supplierDashboardScope, error) {
	app, err := model.GetSupplierByID(supplierID)
	if err != nil {
		return supplierDashboardScope{}, err
	}
	scope, err := collectSupplierDashboardScope(app.ApplicantUserID)
	if err != nil {
		return scope, err
	}
	if err := appendChannelsBySupplierApplicationID(&scope, supplierID); err != nil {
		return scope, err
	}
	return scope, nil
}

// collectSupplierOwnedModelNames 收集供应商名下渠道与模型中的模型名集合。
func collectSupplierOwnedModelNames(userID int) (map[string]struct{}, error) {
	scope, err := collectSupplierDashboardScope(userID)
	if err != nil {
		return nil, err
	}
	return scope.ConfiguredModelNames, nil
}

// collectAllSupplierOwnedModelNames 收集全部供应商名下的模型名集合（管理员统计用）。
func collectAllSupplierOwnedModelNames() (map[string]struct{}, error) {
	scope, err := collectAllSupplierDashboardScope()
	if err != nil {
		return nil, err
	}
	return scope.ConfiguredModelNames, nil
}

// collectSupplierOwnedModelNamesBySupplierID 收集指定供应商申请（supplier_application_id）名下模型集合。
func collectSupplierOwnedModelNamesBySupplierID(supplierID int) (map[string]struct{}, error) {
	scope, err := collectSupplierDashboardScopeBySupplierID(supplierID)
	if err != nil {
		return nil, err
	}
	return scope.ConfiguredModelNames, nil
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
