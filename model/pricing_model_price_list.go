package model

import (
	"math"
	"sort"
	"strconv"

	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

const defaultFactorySupplierAlias = "词元工厂"

// ChannelPricingMeta 可见渠道及其供应商展示字段（supplier_application_id=0 时别名为词元工厂）。
type ChannelPricingMeta struct {
	ChannelID             int
	SupplierApplicationID int
	SupplierAlias         string
}

// ModelPriceListChannelItem model_price_list 嵌套的渠道项。
type ModelPriceListChannelItem struct {
	ChannelID             int     `json:"channel_id"`
	SupplierApplicationID int     `json:"supplier_application_id"`
	SupplierAlias         string  `json:"supplier_alias"`
	ModelPrice            float64 `json:"model_price"`
	ModelRatio            float64 `json:"model_ratio"`
	CompletionRatio       float64 `json:"completion_ratio"`
}

// ModelPriceListSupplierItem model_price_list 嵌套的关联供应商（supplier_id 为 supplier_applications.id）。
type ModelPriceListSupplierItem struct {
	SupplierID    int    `json:"supplier_id"`
	SupplierAlias string `json:"supplier_alias"`
}

// ModelPriceListItem 在保留 Pricing 全部 JSON 字段基础上增加区间与渠道/供应商嵌套列表。
type ModelPriceListItem struct {
	Pricing
	MinModelPrice      float64 `json:"minModelPrice"`
	MinModelRatio      float64 `json:"minModelRatio"`
	MinCompletionRatio float64 `json:"minCompletionRatio"`
	MaxModelPrice      float64 `json:"maxModelPrice"`
	MaxModelRatio      float64 `json:"maxModelRatio"`
	MaxCompletionRatio float64 `json:"maxCompletionRatio"`
	SupplierList       []ModelPriceListSupplierItem `json:"supplier_list"`
	ChannelList        []ModelPriceListChannelItem  `json:"channel_list"`
}

// ListChannelsPricingMeta 查询可见渠道的 id、名称、supplier_application_id，并关联 supplier_applications.supplier_alias。
func ListChannelsPricingMeta(visibleChannelIDs map[int]struct{}) ([]ChannelPricingMeta, error) {
	type scanRow struct {
		ChannelID             int     `gorm:"column:channel_id"`
		SupplierApplicationID int     `gorm:"column:supplier_application_id"`
		SupplierAlias         *string `gorm:"column:supplier_alias"`
	}
	q := DB.Table("channels").
		Select("channels.id AS channel_id, channels.name AS channel_name, channels.supplier_application_id, supplier_applications.supplier_alias AS supplier_alias").
		Joins("LEFT JOIN supplier_applications ON supplier_applications.id = channels.supplier_application_id").
		Order("channels.id ASC")
	ids := make([]int, 0, len(visibleChannelIDs))
	for id := range visibleChannelIDs {
		ids = append(ids, id)
	}
	if len(ids) > 0 {
		q = q.Where("channels.id IN ?", ids)
	}
	var rows []scanRow
	if err := q.Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]ChannelPricingMeta, 0, len(rows))
	for _, r := range rows {
		alias := ""
		if r.SupplierApplicationID == 0 {
			alias = defaultFactorySupplierAlias
		} else if r.SupplierAlias != nil {
			alias = *r.SupplierAlias
		}
		out = append(out, ChannelPricingMeta{
			ChannelID:             r.ChannelID,
			SupplierApplicationID: r.SupplierApplicationID,
			SupplierAlias:         alias,
		})
	}
	return out, nil
}

func channelMapGet(m map[string]map[string]float64, channelKey, modelName string) (float64, bool) {
	sub, ok := m[channelKey]
	if !ok {
		return 0, false
	}
	k := ratio_setting.FormatMatchingModelName(modelName)
	v, ok := sub[k]
	return v, ok
}

func channelHasExplicitModelPricing(channelKey, modelName string, chPrice, chRatio, chComp map[string]map[string]float64) bool {
	_, chPOk := channelMapGet(chPrice, channelKey, modelName)
	_, chROk := channelMapGet(chRatio, channelKey, modelName)
	_, chCOk := channelMapGet(chComp, channelKey, modelName)
	return chPOk || chROk || chCOk
}

func modelHasExplicitVisibleChannelPricing(modelName string, metas []ChannelPricingMeta, chPrice, chRatio, chComp map[string]map[string]float64) bool {
	for _, ch := range metas {
		key := strconv.Itoa(ch.ChannelID)
		if channelHasExplicitModelPricing(key, modelName, chPrice, chRatio, chComp) {
			return true
		}
	}
	return false
}

func shouldIncludeModelInPriceList(modelName string, metas []ChannelPricingMeta, chPrice, chRatio, chComp map[string]map[string]float64) bool {
	if ratio_setting.ModelHasConfiguredPricing(modelName) {
		return true
	}
	return modelHasExplicitVisibleChannelPricing(modelName, metas, chPrice, chRatio, chComp)
}

// resolveChannelModelEffective 单渠道下有效价格：ChannelModelPrice > 全局价；否则 ChannelModelRatio / ChannelCompletionRatio 优先于全局倍率/完成倍率。
func resolveChannelModelEffective(
	channelKey string,
	modelName string,
	chPrice, chRatio, chComp map[string]map[string]float64,
) (price, ratio, completion float64, priceMode bool) {
	chP, chPOk := channelMapGet(chPrice, channelKey, modelName)
	chR, chROk := channelMapGet(chRatio, channelKey, modelName)
	chC, chCOk := channelMapGet(chComp, channelKey, modelName)

	globalP, globalPOk := ratio_setting.GetModelPrice(modelName, false)
	globalR, globalROK, _ := ratio_setting.GetModelRatio(modelName)
	globalC := ratio_setting.GetCompletionRatio(modelName)

	if chPOk {
		return chP, 0, 0, true
	}
	if globalPOk {
		return globalP, 0, 0, true
	}

	var r float64
	if chROk {
		r = chR
	} else if globalROK {
		r = globalR
	} else {
		r = 0
	}
	var c float64
	if chCOk {
		c = chC
	} else {
		c = globalC
	}
	return 0, r, c, false
}

type aggExtrema struct {
	minPrice, maxPrice           float64
	minRatio, maxRatio           float64
	minCompletion, maxCompletion float64
	hasPrice                     bool
	hasRatio                     bool
}

func updateExtrema(ex *aggExtrema, price, ratio, comp float64, priceMode bool) {
	if priceMode {
		if !ex.hasPrice {
			ex.minPrice = price
			ex.maxPrice = price
			ex.hasPrice = true
		} else {
			ex.minPrice = math.Min(ex.minPrice, price)
			ex.maxPrice = math.Max(ex.maxPrice, price)
		}
		return
	}
	if !ex.hasRatio {
		ex.minRatio = ratio
		ex.maxRatio = ratio
		ex.minCompletion = comp
		ex.maxCompletion = comp
		ex.hasRatio = true
	} else {
		ex.minRatio = math.Min(ex.minRatio, ratio)
		ex.maxRatio = math.Max(ex.maxRatio, ratio)
		ex.minCompletion = math.Min(ex.minCompletion, comp)
		ex.maxCompletion = math.Max(ex.maxCompletion, comp)
	}
}

func finalizeExtrema(ex aggExtrema) (minP, maxP, minR, maxR, minC, maxC float64) {
	if ex.hasPrice {
		minP, maxP = ex.minPrice, ex.maxPrice
	}
	if ex.hasRatio {
		minR, maxR = ex.minRatio, ex.maxRatio
		minC, maxC = ex.minCompletion, ex.maxCompletion
	}
	return
}

// BuildModelPriceList 按渠道优先级解析每条渠道有效价，聚合 min/max，并组装 supplier_list / channel_list。
func BuildModelPriceList(
	allPricing []Pricing,
	channelMetas []ChannelPricingMeta,
	chPrice, chRatio, chComp map[string]map[string]float64,
) []ModelPriceListItem {
	out := make([]ModelPriceListItem, 0)
	for _, p := range allPricing {
		if !shouldIncludeModelInPriceList(p.ModelName, channelMetas, chPrice, chRatio, chComp) {
			continue
		}
		hasGlobal := ratio_setting.ModelHasConfiguredPricing(p.ModelName)
		var ex aggExtrema
		chItems := make([]ModelPriceListChannelItem, 0, len(channelMetas))

		for _, ch := range channelMetas {
			chKey := strconv.Itoa(ch.ChannelID)
			if !hasGlobal && !channelHasExplicitModelPricing(chKey, p.ModelName, chPrice, chRatio, chComp) {
				continue
			}
			price, ratio, comp, priceMode := resolveChannelModelEffective(chKey, p.ModelName, chPrice, chRatio, chComp)
			updateExtrema(&ex, price, ratio, comp, priceMode)
			chItems = append(chItems, ModelPriceListChannelItem{
				ChannelID:             ch.ChannelID,
				SupplierApplicationID: ch.SupplierApplicationID,
				SupplierAlias:         ch.SupplierAlias,
				ModelPrice:            price,
				ModelRatio:            ratio,
				CompletionRatio:       comp,
			})
		}

		if len(channelMetas) == 0 {
			price, ratio, comp, priceMode := buildGlobalTupleFixed(p.ModelName)
			updateExtrema(&ex, price, ratio, comp, priceMode)
		} else if len(chItems) == 0 && hasGlobal {
			price, ratio, comp, priceMode := buildGlobalTupleFixed(p.ModelName)
			updateExtrema(&ex, price, ratio, comp, priceMode)
		}

		if !ex.hasPrice && !ex.hasRatio {
			continue
		}

		minP, maxP, minR, maxR, minC, maxC := finalizeExtrema(ex)
		item := ModelPriceListItem{
			Pricing:            p,
			MinModelPrice:      minP,
			MaxModelPrice:      maxP,
			MinModelRatio:      minR,
			MaxModelRatio:      maxR,
			MinCompletionRatio: minC,
			MaxCompletionRatio: maxC,
			ChannelList:        chItems,
		}
		supMap := make(map[int]string)
		for _, ci := range chItems {
			supMap[ci.SupplierApplicationID] = ci.SupplierAlias
		}
		supList := make([]ModelPriceListSupplierItem, 0, len(supMap))
		for sid, salias := range supMap {
			supList = append(supList, ModelPriceListSupplierItem{SupplierID: sid, SupplierAlias: salias})
		}
		sort.Slice(supList, func(i, j int) bool { return supList[i].SupplierID < supList[j].SupplierID })
		item.SupplierList = supList
		out = append(out, item)
	}
	return out
}

func buildGlobalTupleFixed(modelName string) (price, ratio, completion float64, priceMode bool) {
	globalP, globalPOk := ratio_setting.GetModelPrice(modelName, false)
	if globalPOk {
		return globalP, 0, 0, true
	}
	globalR, globalROK, _ := ratio_setting.GetModelRatio(modelName)
	globalC := ratio_setting.GetCompletionRatio(modelName)
	return 0, globalR, globalC, globalROK
}
