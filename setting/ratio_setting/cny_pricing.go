package ratio_setting

import (
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"
)

// CNYUnitPrices 国内模型人民币标价（元 / 1M tokens），展示直接使用，结算时按汇率换内部倍率。
type CNYUnitPrices struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cache_read"`
	CacheWrite float64 `json:"cache_write"`
}

// InternalTokenRatios 由人民币单价换算得到的内部 USD 倍率（与 model_ratio 语义一致）。
type InternalTokenRatios struct {
	ModelRatio       float64
	CompletionRatio  float64
	CacheRatio       float64
	CreateCacheRatio float64
}

var modelCNYPricingMap = types.NewRWMap[string, CNYUnitPrices]()
var channelModelCNYPricingMap = types.NewRWMap[string, map[string]CNYUnitPrices]()

func (p CNYUnitPrices) Valid() bool {
	return isFiniteFloat(p.Input) && p.Input > 0
}

func normalizeCNYUnitPrices(p CNYUnitPrices) CNYUnitPrices {
	if !isFiniteFloat(p.Input) || p.Input < 0 {
		p.Input = 0
	}
	if !isFiniteFloat(p.Output) || p.Output < 0 {
		p.Output = 0
	}
	if !isFiniteFloat(p.CacheRead) || p.CacheRead < 0 {
		p.CacheRead = 0
	}
	if !isFiniteFloat(p.CacheWrite) || p.CacheWrite < 0 {
		p.CacheWrite = 0
	}
	return p
}

func normalizeCNYPricingMap(src map[string]CNYUnitPrices) map[string]CNYUnitPrices {
	out := make(map[string]CNYUnitPrices, len(src))
	for name, prices := range src {
		key := FormatMatchingModelName(strings.TrimSpace(name))
		if key == "" {
			continue
		}
		prices = normalizeCNYUnitPrices(prices)
		if !prices.Valid() {
			continue
		}
		out[key] = prices
	}
	return out
}

// CNYUnitPricesToInternalRatios 将 ¥/1M 换算为内部倍率：
//
//	model_ratio = (input_cny / USDExchangeRate) / 2
//	completion/cache 仍为相对输入价的倍数（与币种无关）
func CNYUnitPricesToInternalRatios(p CNYUnitPrices) InternalTokenRatios {
	p = normalizeCNYUnitPrices(p)
	usdInput := ConvertRequestTierPriceToUSD(p.Input, RequestTierCurrencyCNY)
	out := InternalTokenRatios{
		ModelRatio: PriceToTierRatio(usdInput),
	}
	if p.Input > 0 {
		if p.Output > 0 {
			out.CompletionRatio = p.Output / p.Input
		}
		if p.CacheRead > 0 {
			out.CacheRatio = p.CacheRead / p.Input
		}
		if p.CacheWrite > 0 {
			out.CreateCacheRatio = p.CacheWrite / p.Input
		}
	}
	return out
}

func GetModelCNYPricing(model string) (CNYUnitPrices, bool) {
	p, ok := modelCNYPricingMap.Get(FormatMatchingModelName(model))
	if !ok || !p.Valid() {
		return CNYUnitPrices{}, false
	}
	return p, true
}

func GetModelCNYPricingCopy() map[string]CNYUnitPrices {
	return modelCNYPricingMap.ReadAll()
}

func GetChannelModelCNYPricing(channelID int, model string) (CNYUnitPrices, bool) {
	key := normalizeChannelID(channelID)
	if key == "" {
		return CNYUnitPrices{}, false
	}
	byModel, ok := channelModelCNYPricingMap.Get(key)
	if !ok {
		return CNYUnitPrices{}, false
	}
	p, ok := byModel[FormatMatchingModelName(model)]
	if !ok || !p.Valid() {
		return CNYUnitPrices{}, false
	}
	return p, true
}

func GetChannelModelCNYPricingCopy() map[string]map[string]CNYUnitPrices {
	return channelModelCNYPricingMap.ReadAll()
}

// ResolveCNYPricing 渠道人民币价优先，否则回退全局。
func ResolveCNYPricing(channelID int, model string) (CNYUnitPrices, bool) {
	if p, ok := GetChannelModelCNYPricing(channelID, model); ok {
		return p, true
	}
	return GetModelCNYPricing(model)
}

// ResolveCNYTokenRatios 返回渠道/全局两套内部倍率。ok=false 表示该模型未配置人民币标价。
func ResolveCNYTokenRatios(channelID int, model string) (channelRatios, globalRatios InternalTokenRatios, ok bool) {
	global, hasGlobal := GetModelCNYPricing(model)
	channel, hasChannel := GetChannelModelCNYPricing(channelID, model)
	if !hasChannel && !hasGlobal {
		return InternalTokenRatios{}, InternalTokenRatios{}, false
	}
	if hasChannel {
		channelRatios = CNYUnitPricesToInternalRatios(channel)
	} else {
		channelRatios = CNYUnitPricesToInternalRatios(global)
	}
	if hasGlobal {
		globalRatios = CNYUnitPricesToInternalRatios(global)
	} else {
		globalRatios = channelRatios
	}
	return channelRatios, globalRatios, true
}

func ModelCNYPricing2JSONString() string {
	return modelCNYPricingMap.MarshalJSONString()
}

func UpdateModelCNYPricingByJSONString(jsonStr string) error {
	trimmed := strings.TrimSpace(jsonStr)
	if trimmed == "" {
		modelCNYPricingMap.Clear()
		InvalidateExposedDataCache()
		return nil
	}
	var parsed map[string]CNYUnitPrices
	if err := common.UnmarshalJsonStr(trimmed, &parsed); err != nil {
		return err
	}
	modelCNYPricingMap.Clear()
	modelCNYPricingMap.AddAll(normalizeCNYPricingMap(parsed))
	InvalidateExposedDataCache()
	return nil
}

func ChannelModelCNYPricing2JSONString() string {
	return channelModelCNYPricingMap.MarshalJSONString()
}

func UpdateChannelModelCNYPricingByJSONString(jsonStr string) error {
	trimmed := strings.TrimSpace(jsonStr)
	if trimmed == "" {
		channelModelCNYPricingMap.Clear()
		InvalidateExposedDataCache()
		return nil
	}
	var parsed map[string]map[string]CNYUnitPrices
	if err := common.UnmarshalJsonStr(trimmed, &parsed); err != nil {
		return err
	}
	normalized := make(map[string]map[string]CNYUnitPrices, len(parsed))
	for channelID, prices := range parsed {
		id, convErr := strconv.Atoi(strings.TrimSpace(channelID))
		if convErr != nil {
			continue
		}
		key := normalizeChannelID(id)
		if key == "" {
			continue
		}
		inner := normalizeCNYPricingMap(prices)
		if len(inner) == 0 {
			continue
		}
		normalized[key] = inner
	}
	channelModelCNYPricingMap.Clear()
	channelModelCNYPricingMap.AddAll(normalized)
	InvalidateExposedDataCache()
	return nil
}
