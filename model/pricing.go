package model

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"
)

type Pricing struct {
	ModelName              string                  `json:"model_name"`
	Description            string                  `json:"description,omitempty"`
	Icon                   string                  `json:"icon,omitempty"`
	Tags                   string                  `json:"tags,omitempty"`
	VendorID               int                     `json:"vendor_id,omitempty"`
	QuotaType              int                     `json:"quota_type"`
	ModelRatio             float64                 `json:"model_ratio"`
	ModelPrice             float64                 `json:"model_price"`
	OwnerBy                string                  `json:"owner_by"`
	CompletionRatio        float64                 `json:"completion_ratio"`
	CacheRatio             *float64                `json:"cache_ratio,omitempty"`
	CreateCacheRatio       *float64                `json:"create_cache_ratio,omitempty"`
	ImageRatio             *float64                `json:"image_ratio,omitempty"`
	AudioRatio             *float64                `json:"audio_ratio,omitempty"`
	AudioCompletionRatio   *float64                `json:"audio_completion_ratio,omitempty"`
	VideoRatio             *float64                `json:"video_ratio,omitempty"`
	VideoCompletionRatio   *float64                `json:"video_completion_ratio,omitempty"`
	VideoPrice             *float64                `json:"video_price,omitempty"`
	EnableGroup            []string                `json:"enable_groups"`
	SupportedEndpointTypes []constant.EndpointType `json:"supported_endpoint_types"`
	PricingVersion         string                  `json:"pricing_version,omitempty"`
}

type PricingVendor struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Icon        string `json:"icon,omitempty"`
}

// PricingSupplierItem 定价 data 中的供应商摘要。
type PricingSupplierItem struct {
	SupplierID     int    `json:"supplier_id"`
	SupplierAlias  string `json:"supplier_alias"`
	CompanyLogoURL string `json:"company_logo_url"`
	SupplierType   string `json:"supplier_type"`
}

// PricingChannelItem 某模型在各渠道上的定价摘要。
type PricingChannelItem struct {
	ChannelID             int     `json:"channel_id"`
	SupplierApplicationID int     `json:"supplier_application_id"`
	ChannelNo             string  `json:"channel_no"`
	SupplierAlias         string  `json:"supplier_alias"`
	CompanyLogoURL        string  `json:"company_logo_url"`
	SupplierType          string  `json:"supplier_type"`
	TestResponseTimeMs    int     `json:"test_response_time_ms"`
	ModelPrice            float64 `json:"model_price"`
	ModelRatio            float64 `json:"model_ratio"`
	CompletionRatio       float64 `json:"completion_ratio"`
	CacheRatio            float64 `json:"cache_ratio"`
	CreateCacheRatio      float64 `json:"create_cache_ratio"`
	PriceDiscountPercent  float64 `json:"price_discount_percent"`
}

// PricingAPIItem 在 Pricing 基础上扩展渠道维度统计字段（定价接口 data 元素类型）。
type PricingAPIItem struct {
	Pricing
	SupplierList []PricingSupplierItem `json:"supplier_list"`
	ChannelList  []PricingChannelItem  `json:"channel_list"`
}

func resolveChannelPricingTriple(channelID int, supplierApplicationID int, modelName string) (mp, mr, cr float64) {
	cr = ResolveSupplierScopedCompletionRatio(channelID, supplierApplicationID, modelName)
	// 优先级：供应商渠道表 > 供应商全局表 > Option 渠道 > 平台全局 > 旧 SupplierOption
	if v, ok := ResolveSupplierScopedFixedModelPrice(channelID, supplierApplicationID, modelName); ok {
		return v, 0, cr
	}
	mr, _, _ = ResolveSupplierScopedModelRatio(channelID, supplierApplicationID, modelName)
	return 0, mr, cr
}

func resolveChannelCachePair(channelID int, supplierApplicationID int, modelName string) (cacheRatio, createCacheRatio float64) {
	return ResolveSupplierScopedCacheRatios(channelID, supplierApplicationID, modelName)
}

func pricingSupplierAliasFromMeta(supplierApplicationID int, alias *string) string {
	if supplierApplicationID == 0 {
		return "P0"
	}
	if alias != nil {
		s := strings.TrimSpace(*alias)
		if s == "0" {
			return "P0"
		}
		if s != "" {
			return s
		}
	}
	return SupplierApplicationAutoAlias(supplierApplicationID)
}

// BuildPricingAPIItems 为定价接口组装带渠道统计的 data 列表。
// 渠道项价格为：基础定价（resolveChannelPricingTriple）× 渠道专属折扣；用户/分组倍率由前端用 group_ratio 再乘（与 calculateModelPrice 一致）。
func BuildPricingAPIItems(filtered []Pricing, visibleChannelIDs map[int]struct{}, metas []ChannelPricingMeta) []PricingAPIItem {
	testSuccessByChannel, err := LoadChannelPricingTestSuccessIndex()
	if err != nil {
		common.SysLog(fmt.Sprintf("LoadChannelPricingTestSuccessIndex error: %v", err))
		testSuccessByChannel = nil
	}
	visibleIDs := make([]int, 0, len(visibleChannelIDs))
	for id := range visibleChannelIDs {
		visibleIDs = append(visibleIDs, id)
	}

	out := make([]PricingAPIItem, 0, len(filtered))
	for _, p := range filtered {
		item := PricingAPIItem{Pricing: p}
		var chItems []PricingChannelItem

		modelName := p.ModelName
		// 为当前模型预加载各可见渠道的测试耗时：手动覆盖耗时优先，否则使用最近一次成功测试耗时。
		testResponseTimeByChannel := make(map[int]int)
		if len(visibleIDs) > 0 {
			rows, err := GetModelTestResultsByModelNameAndChannelIDs(modelName, visibleIDs)
			if err != nil {
				common.SysLog(fmt.Sprintf("GetModelTestResultsByModelNameAndChannelIDs error: model=%s err=%v", modelName, err))
			} else {
				for i := range rows {
					r := rows[i]
					if r.ChannelId <= 0 {
						continue
					}
					if r.ManualDisplayResponseTime > 0 {
						testResponseTimeByChannel[r.ChannelId] = r.ManualDisplayResponseTime
						continue
					}
					if r.LastTestSuccess && r.LastResponseTime > 0 {
						testResponseTimeByChannel[r.ChannelId] = r.LastResponseTime
					}
				}
			}
		}
		for _, row := range metas {
			if row.ChannelID <= 0 {
				continue
			}
			if _, ok := visibleChannelIDs[row.ChannelID]; !ok {
				continue
			}
			if !ChannelModelsRawContains(row.Models, modelName) {
				continue
			}
			// 单测门禁：仅当该渠道在库中已有「至少一条」成功单测记录时，才要求本模型也有成功记录。
			// 否则新渠道/供应商从未跑过单测时 names 为空，旧逻辑会对所有模型 continue，导致供应商只见自有渠道时 data 全空。
			if testSuccessByChannel != nil {
				namesOK := testSuccessByChannel[row.ChannelID]
				if len(namesOK) > 0 && !ChannelPricingRowMatchesLastTestSuccess(testSuccessByChannel, row.ChannelID, modelName) {
					continue
				}
			}
			baseMp, baseMr, cr := resolveChannelPricingTriple(row.ChannelID, row.SupplierApplicationID, modelName)
			chCache, chCreate := resolveChannelCachePair(row.ChannelID, row.SupplierApplicationID, modelName)
			alias := pricingSupplierAliasFromMeta(row.SupplierApplicationID, row.SupplierAlias)
			d := 100.0
			if row.PriceDiscountPercent != nil {
				d = *row.PriceDiscountPercent
			}
			mult := ChannelPriceDiscountMultiplierForPricing(d)
			mp := baseMp * mult
			mr := baseMr * mult
			chItems = append(chItems, PricingChannelItem{
				ChannelID:             row.ChannelID,
				SupplierApplicationID: row.SupplierApplicationID,
				ChannelNo:             row.ChannelNo,
				SupplierAlias:         alias,
				CompanyLogoURL:        strings.TrimSpace(row.CompanyLogoURL),
				SupplierType:          strings.TrimSpace(row.SupplierType),
				TestResponseTimeMs:    testResponseTimeByChannel[row.ChannelID],
				ModelPrice:            mp,
				ModelRatio:            mr,
				CompletionRatio:       cr,
				CacheRatio:            chCache,
				CreateCacheRatio:      chCreate,
				PriceDiscountPercent:  d,
			})
		}

		if len(chItems) == 0 {
			continue
		}

		sort.Slice(chItems, func(i, j int) bool {
			var ai, aj float64
			if p.QuotaType == 1 {
				ai, aj = chItems[i].ModelPrice, chItems[j].ModelPrice
			} else {
				ai, aj = chItems[i].ModelRatio, chItems[j].ModelRatio
			}
			if ai != aj {
				return ai < aj
			}
			return chItems[i].ChannelID < chItems[j].ChannelID
		})

		item.ChannelList = chItems

		supplierSeen := make(map[int]struct{})
		suppliers := make([]PricingSupplierItem, 0)
		for _, ch := range chItems {
			sid := ch.SupplierApplicationID
			if _, ok := supplierSeen[sid]; ok {
				continue
			}
			supplierSeen[sid] = struct{}{}
			suppliers = append(suppliers, PricingSupplierItem{
				SupplierID:     sid,
				SupplierAlias:  ch.SupplierAlias,
				CompanyLogoURL: ch.CompanyLogoURL,
				SupplierType:   ch.SupplierType,
			})
		}
		item.SupplierList = suppliers
		out = append(out, item)
	}
	return out
}

var (
	pricingMap           []Pricing
	vendorsList          []PricingVendor
	supportedEndpointMap map[string]common.EndpointInfo
	lastGetPricingTime   time.Time
	updatePricingLock    sync.Mutex

	// 缓存映射：模型名 -> 启用分组 / 计费类型
	modelEnableGroups     = make(map[string][]string)
	modelQuotaTypeMap     = make(map[string]int)
	modelEnableGroupsLock = sync.RWMutex{}
)

var (
	modelSupportEndpointTypes = make(map[string][]constant.EndpointType)
	modelSupportEndpointsLock = sync.RWMutex{}
)

func GetPricing() []Pricing {
	if time.Since(lastGetPricingTime) > time.Minute*1 || len(pricingMap) == 0 {
		updatePricingLock.Lock()
		defer updatePricingLock.Unlock()
		// Double check after acquiring the lock
		if time.Since(lastGetPricingTime) > time.Minute*1 || len(pricingMap) == 0 {
			modelSupportEndpointsLock.Lock()
			defer modelSupportEndpointsLock.Unlock()
			updatePricing()
		}
	}
	return pricingMap
}

// GetVendors 返回当前定价接口使用到的供应商信息
func GetVendors() []PricingVendor {
	if time.Since(lastGetPricingTime) > time.Minute*1 || len(pricingMap) == 0 {
		// 保证先刷新一次
		GetPricing()
	}
	return vendorsList
}

func GetModelSupportEndpointTypes(model string) []constant.EndpointType {
	if model == "" {
		return make([]constant.EndpointType, 0)
	}
	modelSupportEndpointsLock.RLock()
	defer modelSupportEndpointsLock.RUnlock()
	if endpoints, ok := modelSupportEndpointTypes[model]; ok {
		return endpoints
	}
	return make([]constant.EndpointType, 0)
}

func updatePricing() {
	//modelRatios := common.GetModelRatios()
	enableAbilities, err := GetAllEnableAbilityWithChannels()
	if err != nil {
		common.SysLog(fmt.Sprintf("GetAllEnableAbilityWithChannels error: %v", err))
		return
	}
	// 预加载模型元数据与供应商一次，避免循环查询
	var allMeta []Model
	_ = DB.Find(&allMeta).Error
	metaMap := make(map[string]*Model)
	prefixList := make([]*Model, 0)
	suffixList := make([]*Model, 0)
	containsList := make([]*Model, 0)
	for i := range allMeta {
		m := &allMeta[i]
		if m.NameRule == NameRuleExact {
			metaMap[m.ModelName] = m
		} else {
			switch m.NameRule {
			case NameRulePrefix:
				prefixList = append(prefixList, m)
			case NameRuleSuffix:
				suffixList = append(suffixList, m)
			case NameRuleContains:
				containsList = append(containsList, m)
			}
		}
	}

	// 将非精确规则模型匹配到 metaMap
	for _, m := range prefixList {
		for _, pricingModel := range enableAbilities {
			if strings.HasPrefix(pricingModel.Model, m.ModelName) {
				if _, exists := metaMap[pricingModel.Model]; !exists {
					metaMap[pricingModel.Model] = m
				}
			}
		}
	}
	for _, m := range suffixList {
		for _, pricingModel := range enableAbilities {
			if strings.HasSuffix(pricingModel.Model, m.ModelName) {
				if _, exists := metaMap[pricingModel.Model]; !exists {
					metaMap[pricingModel.Model] = m
				}
			}
		}
	}
	for _, m := range containsList {
		for _, pricingModel := range enableAbilities {
			if strings.Contains(pricingModel.Model, m.ModelName) {
				if _, exists := metaMap[pricingModel.Model]; !exists {
					metaMap[pricingModel.Model] = m
				}
			}
		}
	}

	// 预加载供应商
	var vendors []Vendor
	_ = DB.Find(&vendors).Error
	vendorMap := make(map[int]*Vendor)
	for i := range vendors {
		vendorMap[vendors[i].Id] = &vendors[i]
	}

	// 初始化默认供应商映射
	initDefaultVendorMapping(metaMap, vendorMap, enableAbilities)

	// 构建对前端友好的供应商列表
	vendorsList = make([]PricingVendor, 0, len(vendorMap))
	for _, v := range vendorMap {
		vendorsList = append(vendorsList, PricingVendor{
			ID:          v.Id,
			Name:        v.Name,
			Description: v.Description,
			Icon:        v.Icon,
		})
	}

	modelGroupsMap := make(map[string]*types.Set[string])

	for _, ability := range enableAbilities {
		groups, ok := modelGroupsMap[ability.Model]
		if !ok {
			groups = types.NewSet[string]()
			modelGroupsMap[ability.Model] = groups
		}
		groups.Add(ability.Group)
	}

	//这里使用切片而不是Set，因为一个模型可能支持多个端点类型，并且第一个端点是优先使用端点
	modelSupportEndpointsStr := make(map[string][]string)

	// 先根据已有能力填充原生端点
	for _, ability := range enableAbilities {
		endpoints := modelSupportEndpointsStr[ability.Model]
		channelTypes := common.GetEndpointTypesByChannelType(ability.ChannelType, ability.Model)
		for _, channelType := range channelTypes {
			if !common.StringsContains(endpoints, string(channelType)) {
				endpoints = append(endpoints, string(channelType))
			}
		}
		modelSupportEndpointsStr[ability.Model] = endpoints
	}

	// 再补充模型自定义端点：若配置有效则替换默认端点，不做合并
	for modelName, meta := range metaMap {
		if strings.TrimSpace(meta.Endpoints) == "" {
			continue
		}
		var raw map[string]interface{}
		if err := json.Unmarshal([]byte(meta.Endpoints), &raw); err == nil {
			endpoints := make([]string, 0, len(raw))
			for k, v := range raw {
				switch v.(type) {
				case string, map[string]interface{}:
					if !common.StringsContains(endpoints, k) {
						endpoints = append(endpoints, k)
					}
				}
			}
			if len(endpoints) > 0 {
				modelSupportEndpointsStr[modelName] = endpoints
			}
		}
	}

	modelSupportEndpointTypes = make(map[string][]constant.EndpointType)
	for model, endpoints := range modelSupportEndpointsStr {
		supportedEndpoints := make([]constant.EndpointType, 0)
		for _, endpointStr := range endpoints {
			endpointType := constant.EndpointType(endpointStr)
			supportedEndpoints = append(supportedEndpoints, endpointType)
		}
		modelSupportEndpointTypes[model] = supportedEndpoints
	}

	// 构建全局 supportedEndpointMap（默认 + 自定义覆盖）
	supportedEndpointMap = make(map[string]common.EndpointInfo)
	// 1. 默认端点
	for _, endpoints := range modelSupportEndpointTypes {
		for _, et := range endpoints {
			if info, ok := common.GetDefaultEndpointInfo(et); ok {
				if _, exists := supportedEndpointMap[string(et)]; !exists {
					supportedEndpointMap[string(et)] = info
				}
			}
		}
	}
	// 2. 自定义端点（models 表）覆盖默认
	for _, meta := range metaMap {
		if strings.TrimSpace(meta.Endpoints) == "" {
			continue
		}
		var raw map[string]interface{}
		if err := json.Unmarshal([]byte(meta.Endpoints), &raw); err == nil {
			for k, v := range raw {
				switch val := v.(type) {
				case string:
					supportedEndpointMap[k] = common.EndpointInfo{Path: val, Method: "POST"}
				case map[string]interface{}:
					ep := common.EndpointInfo{Method: "POST"}
					if p, ok := val["path"].(string); ok {
						ep.Path = p
					}
					if m, ok := val["method"].(string); ok {
						ep.Method = strings.ToUpper(m)
					}
					supportedEndpointMap[k] = ep
				default:
					// ignore unsupported types
				}
			}
		}
	}

	pricingMap = make([]Pricing, 0)
	for model, groups := range modelGroupsMap {
		pricing := Pricing{
			ModelName:              model,
			EnableGroup:            groups.Items(),
			SupportedEndpointTypes: modelSupportEndpointTypes[model],
		}

		// 补充模型元数据（描述、标签、供应商、状态）
		if meta, ok := metaMap[model]; ok {
			// 若模型被禁用(status!=1)，则直接跳过，不返回给前端
			if meta.Status != 1 {
				continue
			}
			pricing.Description = meta.Description
			pricing.Icon = meta.Icon
			pricing.Tags = meta.Tags
			pricing.VendorID = meta.VendorID
		}
		modelPrice, findPrice := ratio_setting.GetModelPrice(model, false)
		if findPrice {
			pricing.ModelPrice = modelPrice
			pricing.QuotaType = 1
		} else {
			modelRatio, _, _ := ratio_setting.GetModelRatio(model)
			pricing.ModelRatio = modelRatio
			pricing.CompletionRatio = ratio_setting.GetCompletionRatio(model)
			pricing.QuotaType = 0
		}
		if cacheRatio, ok := ratio_setting.GetCacheRatio(model); ok {
			pricing.CacheRatio = &cacheRatio
		}
		if createCacheRatio, ok := ratio_setting.GetCreateCacheRatio(model); ok {
			pricing.CreateCacheRatio = &createCacheRatio
		}
		if imageRatio, ok := ratio_setting.GetImageRatio(model); ok {
			pricing.ImageRatio = &imageRatio
		}
		if ratio_setting.ContainsAudioRatio(model) {
			audioRatio := ratio_setting.GetAudioRatio(model)
			pricing.AudioRatio = &audioRatio
		}
		if ratio_setting.ContainsAudioCompletionRatio(model) {
			audioCompletionRatio := ratio_setting.GetAudioCompletionRatio(model)
			pricing.AudioCompletionRatio = &audioCompletionRatio
		}
		if ratio_setting.ContainsVideoRatio(model) {
			videoRatio := ratio_setting.GetVideoRatio(model)
			pricing.VideoRatio = &videoRatio
		}
		if ratio_setting.ContainsVideoCompletionRatio(model) {
			videoCompletionRatio := ratio_setting.GetVideoCompletionRatio(model)
			pricing.VideoCompletionRatio = &videoCompletionRatio
		}
		if ratio_setting.ContainsVideoPrice(model) {
			videoPrice, _ := ratio_setting.GetVideoPrice(model)
			pricing.VideoPrice = &videoPrice
		}
		pricingMap = append(pricingMap, pricing)
	}

	// 防止大更新后数据不通用
	if len(pricingMap) > 0 {
		pricingMap[0].PricingVersion = "5a90f2b86c08bd983a9a2e6d66c255f4eaef9c4bc934386d2b6ae84ef0ff1f1f"
	}

	// 刷新缓存映射，供高并发快速查询
	modelEnableGroupsLock.Lock()
	modelEnableGroups = make(map[string][]string)
	modelQuotaTypeMap = make(map[string]int)
	for _, p := range pricingMap {
		modelEnableGroups[p.ModelName] = p.EnableGroup
		modelQuotaTypeMap[p.ModelName] = p.QuotaType
	}
	modelEnableGroupsLock.Unlock()

	lastGetPricingTime = time.Now()
}

// GetSupportedEndpointMap 返回全局端点到路径的映射
func GetSupportedEndpointMap() map[string]common.EndpointInfo {
	return supportedEndpointMap
}
