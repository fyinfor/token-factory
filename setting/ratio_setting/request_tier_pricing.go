package ratio_setting

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/types"
)

const (
	RequestTierModeProgressive = "progressive"

	RequestTierDimensionInputTokens = "input_tokens"

	// RequestTierBoundaryLt：prev ≤ tokens < up_to（边界落入下一档），默认，兼容旧逻辑
	RequestTierBoundaryLt = "lt"
	// RequestTierBoundaryLte：首档 0 ≤ tokens ≤ up_to，其后 prev < tokens ≤ up_to
	RequestTierBoundaryLte = "lte"

	RequestTierCurrencyUSD    = "USD"
	RequestTierCurrencyCNY    = "CNY"
	RequestTierCurrencyCustom = "CUSTOM"

	// TierRatioBase 价格($/1M) ↔ 内部 ratio 换算基准（与 model_ratio 默认 2 对齐）
	TierRatioBase = 2.0
)

// RequestTierPrices 单档四类单价（基准货币 /1M tokens；计费时换算为 USD）
type RequestTierPrices struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cache_read"`
	CacheWrite float64 `json:"cache_write"`
}

// RequestTierBand 单个输入 Token 区间档位
type RequestTierBand struct {
	UpTo   int64             `json:"up_to"`
	Prices RequestTierPrices `json:"prices"`
}

// RequestTierPricing 统一阶梯计费规则
// dimension 预留扩展（当前仅 input_tokens）；boundary 控制上限开闭。
// Currency 为价格基准货币（USD/CNY/CUSTOM），prices 存该货币下的单价；
// 使用时换算为系统内部 USD 计价；与系统展示货币一致时前端可直接展示。
type RequestTierPricing struct {
	Mode      string            `json:"mode,omitempty"`
	Dimension string            `json:"dimension,omitempty"`
	Boundary  string            `json:"boundary,omitempty"` // lt | lte
	Currency  string            `json:"currency,omitempty"` // USD | CNY | CUSTOM
	Tiers     []RequestTierBand `json:"tiers,omitempty"`
}

// RequestTierHit 命中档结果（含渠道/全局价格，供计费与展示）
type RequestTierHit struct {
	FromToken           int64
	ToToken             int64 // 0 = ∞
	Boundary            string
	Label               string
	ChannelPrices       RequestTierPrices // 已换算为 USD；无渠道时为零值且 HasChannel=false
	GlobalPrices        RequestTierPrices // 已换算为 USD
	HasChannel          bool
	HasGlobal           bool
	EffectiveInput      float64 // 内部 ratio（usdPrice/TierRatioBase 经折扣后）
	EffectiveOutput     float64
	EffectiveCacheRead  float64
	EffectiveCacheWrite float64
	InputUnitPrice      float64 // 平台展示单价 $/1M
	OutputUnitPrice     float64
	CacheReadUnitPrice  float64
	CacheWriteUnitPrice float64
}

var modelRequestTierPricingMap = types.NewRWMap[string, RequestTierPricing]()
var channelModelRequestTierPricingMap = types.NewRWMap[string, map[string]RequestTierPricing]()

// ---------- 价格 / ratio 换算 ----------

func PriceToTierRatio(price float64) float64 {
	if price < 0 || !isFiniteFloat(price) {
		return 0
	}
	return price / TierRatioBase
}

func TierRatioToPrice(ratio float64) float64 {
	if ratio < 0 || !isFiniteFloat(ratio) {
		return 0
	}
	return ratio * TierRatioBase
}

func isFiniteFloat(v float64) bool {
	return !((v != v) || v > 1e308 || v < -1e308)
}

func NormalizeRequestTierCurrency(currency string) string {
	switch strings.ToUpper(strings.TrimSpace(currency)) {
	case RequestTierCurrencyCNY:
		return RequestTierCurrencyCNY
	case RequestTierCurrencyCustom:
		return RequestTierCurrencyCustom
	default:
		return RequestTierCurrencyUSD
	}
}

// GetRequestTierCurrencyRate 返回 1 USD = X <currency>
func GetRequestTierCurrencyRate(currency string) float64 {
	switch NormalizeRequestTierCurrency(currency) {
	case RequestTierCurrencyCNY:
		if operation_setting.USDExchangeRate > 0 {
			return operation_setting.USDExchangeRate
		}
		return 1
	case RequestTierCurrencyCustom:
		rate := operation_setting.GetGeneralSetting().CustomCurrencyExchangeRate
		if rate > 0 {
			return rate
		}
		return 1
	default:
		return 1
	}
}

// ConvertRequestTierPriceToUSD 将基准货币单价换算为 USD（同币种或 USD 时原样返回）
func ConvertRequestTierPriceToUSD(price float64, currency string) float64 {
	if !isFiniteFloat(price) || price == 0 {
		return 0
	}
	rate := GetRequestTierCurrencyRate(currency)
	if rate <= 0 || rate == 1 {
		return price
	}
	return price / rate
}

func pricesToUSD(p RequestTierPrices, currency string) RequestTierPrices {
	return RequestTierPrices{
		Input:      ConvertRequestTierPriceToUSD(p.Input, currency),
		Output:     ConvertRequestTierPriceToUSD(p.Output, currency),
		CacheRead:  ConvertRequestTierPriceToUSD(p.CacheRead, currency),
		CacheWrite: ConvertRequestTierPriceToUSD(p.CacheWrite, currency),
	}
}

func pricesToRatios(p RequestTierPrices) (input, output, cacheRead, cacheWrite float64) {
	return PriceToTierRatio(p.Input), PriceToTierRatio(p.Output), PriceToTierRatio(p.CacheRead), PriceToTierRatio(p.CacheWrite)
}

// ---------- 规范化 / 校验 ----------

func NormalizeRequestTierBoundary(boundary string) string {
	switch strings.TrimSpace(strings.ToLower(boundary)) {
	case RequestTierBoundaryLte:
		return RequestTierBoundaryLte
	default:
		return RequestTierBoundaryLt
	}
}

func normalizeRequestTierPricing(rule RequestTierPricing) RequestTierPricing {
	if strings.TrimSpace(rule.Mode) == "" {
		rule.Mode = RequestTierModeProgressive
	}
	if strings.TrimSpace(rule.Dimension) == "" {
		rule.Dimension = RequestTierDimensionInputTokens
	}
	rule.Boundary = NormalizeRequestTierBoundary(rule.Boundary)
	rule.Currency = NormalizeRequestTierCurrency(rule.Currency)
	rule.Tiers = normalizeRequestTierBands(rule.Tiers)
	return rule
}

func normalizeRequestTierBands(tiers []RequestTierBand) []RequestTierBand {
	out := make([]RequestTierBand, 0, len(tiers))
	for _, tier := range tiers {
		if tier.Prices.Input < 0 || tier.Prices.Output < 0 || tier.Prices.CacheRead < 0 || tier.Prices.CacheWrite < 0 {
			continue
		}
		out = append(out, tier)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].UpTo == 0 {
			return false
		}
		if out[j].UpTo == 0 {
			return true
		}
		return out[i].UpTo < out[j].UpTo
	})
	return out
}

func ValidateRequestTierPricing(rule RequestTierPricing) error {
	mode := strings.TrimSpace(rule.Mode)
	if mode == "" {
		mode = RequestTierModeProgressive
	}
	if mode != RequestTierModeProgressive {
		return errors.New("仅支持 progressive 阶梯计费模式")
	}
	dimension := strings.TrimSpace(rule.Dimension)
	if dimension == "" {
		dimension = RequestTierDimensionInputTokens
	}
	if dimension != RequestTierDimensionInputTokens {
		return fmt.Errorf("暂不支持 dimension=%s，当前仅允许 %s", dimension, RequestTierDimensionInputTokens)
	}
	boundary := NormalizeRequestTierBoundary(rule.Boundary)
	if boundary != RequestTierBoundaryLt && boundary != RequestTierBoundaryLte {
		return errors.New("boundary 仅允许 lt 或 lte")
	}
	if len(rule.Tiers) == 0 {
		return nil
	}
	previous := int64(0)
	hasPositiveInput := false
	for i, tier := range rule.Tiers {
		if tier.Prices.Input < 0 || tier.Prices.Output < 0 || tier.Prices.CacheRead < 0 || tier.Prices.CacheWrite < 0 {
			return fmt.Errorf("第 %d 档价格不能为负数", i+1)
		}
		if tier.Prices.Input > 0 {
			hasPositiveInput = true
		}
		if tier.UpTo == 0 {
			if i != len(rule.Tiers)-1 {
				return fmt.Errorf("只有最后一档 up_to 可以为 0（无限）")
			}
			continue
		}
		if tier.UpTo <= previous {
			return fmt.Errorf("第 %d 档 up_to 必须递增", i+1)
		}
		previous = tier.UpTo
	}
	if !hasPositiveInput {
		return errors.New("至少需要一档输入价格大于 0")
	}
	return nil
}

func normalizeRequestTierPricingMap(src map[string]RequestTierPricing) (map[string]RequestTierPricing, error) {
	dst := make(map[string]RequestTierPricing, len(src))
	for modelName, rule := range src {
		name := FormatMatchingModelName(strings.TrimSpace(modelName))
		if name == "" {
			continue
		}
		rule = normalizeRequestTierPricing(rule)
		if err := ValidateRequestTierPricing(rule); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		if len(rule.Tiers) == 0 {
			continue
		}
		dst[name] = rule
	}
	return dst, nil
}

// ---------- 命中逻辑 ----------

// FindRequestTierBandIndex 按 dimensionValue（当前为输入 token）与 boundary 命中档位下标；无档返回 -1
func FindRequestTierBandIndex(tokens int64, tiers []RequestTierBand, boundary string) int {
	if len(tiers) == 0 {
		return -1
	}
	boundary = NormalizeRequestTierBoundary(boundary)
	for i := range tiers {
		upTo := tiers[i].UpTo
		if upTo == 0 {
			return i
		}
		if boundary == RequestTierBoundaryLte {
			if tokens <= upTo {
				return i
			}
		} else if tokens < upTo {
			return i
		}
	}
	return len(tiers) - 1
}

// BandRangeFromIndex 返回档位区间 [from, to]，to=0 表示 ∞
func BandRangeFromIndex(tiers []RequestTierBand, index int) (from, to int64) {
	if index < 0 || index >= len(tiers) {
		return 0, 0
	}
	to = tiers[index].UpTo
	if index == 0 {
		return 0, to
	}
	return tiers[index-1].UpTo, to
}

// BuildRequestTierLabel 生成档位展示标签，随 boundary 变化比较符
func BuildRequestTierLabel(tokenType string, tiers []RequestTierBand, tokens int64, boundary string) string {
	idx := FindRequestTierBandIndex(tokens, tiers, boundary)
	if idx < 0 {
		return ""
	}
	boundary = NormalizeRequestTierBoundary(boundary)
	from, to := BandRangeFromIndex(tiers, idx)
	prefix := tokenType + "token"
	if to == 0 {
		if from <= 0 {
			return prefix + "≥0"
		}
		return fmt.Sprintf("%s≥%d", prefix, from)
	}
	if boundary == RequestTierBoundaryLte {
		return fmt.Sprintf("%s≤%d", prefix, to)
	}
	return fmt.Sprintf("%s<%d", prefix, to)
}

func tierEffectiveRate(channelRatio, globalRatio, costDiscPercent, markupDiscPercent float64) float64 {
	return channelRatio*(costDiscPercent/100.0) + globalRatio*(markupDiscPercent/100.0)
}

func platformUsdPerMFromPrices(channelPrice, globalPrice float64, hasChannel bool, costDiscPercent, markupDiscPercent, groupRatio float64) float64 {
	globalRatio := PriceToTierRatio(globalPrice)
	baseRatio := globalRatio
	if hasChannel {
		baseRatio = PriceToTierRatio(channelPrice)
	}
	return tierEffectiveRate(baseRatio, globalRatio, costDiscPercent, markupDiscPercent) * TierRatioBase * groupRatio
}

// ResolveRequestTierPricing 渠道完整规则优先，否则全局；禁止跨源拼装
func ResolveRequestTierPricing(channelID int, model string) (RequestTierPricing, bool) {
	if rule, ok := GetChannelModelRequestTierPricing(channelID, model); ok && len(rule.Tiers) > 0 {
		return rule, true
	}
	if rule, ok := GetModelRequestTierPricing(model); ok && len(rule.Tiers) > 0 {
		return rule, true
	}
	return RequestTierPricing{}, false
}

// HasAnyRequestTierPricing 是否存在任意全局/渠道阶梯配置
func HasAnyRequestTierPricing() bool {
	return modelRequestTierPricingMap.Len() > 0 || channelModelRequestTierPricingMap.Len() > 0
}

// ResolveRequestTierHit 按输入 token 命中档位，并计算有效倍率与展示单价。
// 渠道有规则时整规则使用渠道（含 boundary）；全局价用于加价折扣侧（若全局同 band 存在）。
func ResolveRequestTierHit(
	channelID int,
	model string,
	inputTokens int64,
	costDiscPercent, markupDiscPercent, groupRatio float64,
) (RequestTierHit, bool) {
	channelRule, hasChannel := GetChannelModelRequestTierPricing(channelID, model)
	globalRule, hasGlobal := GetModelRequestTierPricing(model)
	if hasChannel && len(channelRule.Tiers) == 0 {
		hasChannel = false
	}
	if hasGlobal && len(globalRule.Tiers) == 0 {
		hasGlobal = false
	}
	if !hasChannel && !hasGlobal {
		return RequestTierHit{}, false
	}

	bandRule := globalRule
	if hasChannel {
		bandRule = channelRule
	}
	bandRule = normalizeRequestTierPricing(bandRule)
	idx := FindRequestTierBandIndex(inputTokens, bandRule.Tiers, bandRule.Boundary)
	if idx < 0 {
		return RequestTierHit{}, false
	}
	from, to := BandRangeFromIndex(bandRule.Tiers, idx)

	hit := RequestTierHit{
		FromToken: from,
		ToToken:   to,
		Boundary:  bandRule.Boundary,
		Label:     BuildRequestTierLabel("输入", bandRule.Tiers, inputTokens, bandRule.Boundary),
		HasChannel: hasChannel,
		HasGlobal:  hasGlobal,
	}

	if hasChannel {
		chRule := normalizeRequestTierPricing(channelRule)
		chIdx := FindRequestTierBandIndex(inputTokens, chRule.Tiers, chRule.Boundary)
		if chIdx >= 0 {
			hit.ChannelPrices = pricesToUSD(chRule.Tiers[chIdx].Prices, chRule.Currency)
		}
	}
	if hasGlobal {
		gRule := normalizeRequestTierPricing(globalRule)
		gIdx := FindRequestTierBandIndex(inputTokens, gRule.Tiers, gRule.Boundary)
		if gIdx >= 0 {
			hit.GlobalPrices = pricesToUSD(gRule.Tiers[gIdx].Prices, gRule.Currency)
		}
	}

	// 有效倍率：有渠道用渠道价作 base，全局价作 markup 侧；仅全局时 base=global
	chIn, chOut, chCR, chCW := pricesToRatios(hit.ChannelPrices)
	gIn, gOut, gCR, gCW := pricesToRatios(hit.GlobalPrices)
	if !hasChannel {
		chIn, chOut, chCR, chCW = gIn, gOut, gCR, gCW
	}
	hit.EffectiveInput = tierEffectiveRate(chIn, gIn, costDiscPercent, markupDiscPercent)
	hit.EffectiveOutput = tierEffectiveRate(chOut, gOut, costDiscPercent, markupDiscPercent)
	hit.EffectiveCacheRead = tierEffectiveRate(chCR, gCR, costDiscPercent, markupDiscPercent)
	hit.EffectiveCacheWrite = tierEffectiveRate(chCW, gCW, costDiscPercent, markupDiscPercent)

	hit.InputUnitPrice = platformUsdPerMFromPrices(hit.ChannelPrices.Input, hit.GlobalPrices.Input, hasChannel, costDiscPercent, markupDiscPercent, groupRatio)
	hit.OutputUnitPrice = platformUsdPerMFromPrices(hit.ChannelPrices.Output, hit.GlobalPrices.Output, hasChannel, costDiscPercent, markupDiscPercent, groupRatio)
	hit.CacheReadUnitPrice = platformUsdPerMFromPrices(hit.ChannelPrices.CacheRead, hit.GlobalPrices.CacheRead, hasChannel, costDiscPercent, markupDiscPercent, groupRatio)
	hit.CacheWriteUnitPrice = platformUsdPerMFromPrices(hit.ChannelPrices.CacheWrite, hit.GlobalPrices.CacheWrite, hasChannel, costDiscPercent, markupDiscPercent, groupRatio)

	return hit, true
}

// ---------- Option CRUD：全局 ----------

func ModelRequestTierPricing2JSONString() string {
	return modelRequestTierPricingMap.MarshalJSONString()
}

func UpdateModelRequestTierPricingByJSONString(jsonStr string) error {
	trimmed := strings.TrimSpace(jsonStr)
	if trimmed == "" {
		modelRequestTierPricingMap.Clear()
		InvalidateExposedDataCache()
		return nil
	}
	var parsed map[string]RequestTierPricing
	if err := common.UnmarshalJsonStr(trimmed, &parsed); err != nil {
		return err
	}
	normalized, err := normalizeRequestTierPricingMap(parsed)
	if err != nil {
		return err
	}
	modelRequestTierPricingMap.Clear()
	modelRequestTierPricingMap.AddAll(normalized)
	InvalidateExposedDataCache()
	return nil
}

func GetModelRequestTierPricing(model string) (RequestTierPricing, bool) {
	return modelRequestTierPricingMap.Get(FormatMatchingModelName(model))
}

func GetModelRequestTierPricingCopy() map[string]RequestTierPricing {
	return modelRequestTierPricingMap.ReadAll()
}

// ---------- Option CRUD：渠道 ----------

func ChannelModelRequestTierPricing2JSONString() string {
	return channelModelRequestTierPricingMap.MarshalJSONString()
}

func UpdateChannelModelRequestTierPricingByJSONString(jsonStr string) error {
	trimmed := strings.TrimSpace(jsonStr)
	if trimmed == "" {
		channelModelRequestTierPricingMap.Clear()
		InvalidateExposedDataCache()
		return nil
	}
	var parsed map[string]map[string]RequestTierPricing
	if err := common.UnmarshalJsonStr(trimmed, &parsed); err != nil {
		return err
	}
	normalized := make(map[string]map[string]RequestTierPricing, len(parsed))
	for channelID, rules := range parsed {
		id, convErr := strconv.Atoi(strings.TrimSpace(channelID))
		if convErr != nil {
			continue
		}
		key := normalizeChannelID(id)
		if key == "" {
			continue
		}
		normalizedRules, err := normalizeRequestTierPricingMap(rules)
		if err != nil {
			return err
		}
		if len(normalizedRules) == 0 {
			continue
		}
		normalized[key] = normalizedRules
	}
	channelModelRequestTierPricingMap.Clear()
	channelModelRequestTierPricingMap.AddAll(normalized)
	InvalidateExposedDataCache()
	return nil
}

func GetChannelModelRequestTierPricing(channelID int, model string) (RequestTierPricing, bool) {
	key := normalizeChannelID(channelID)
	if key == "" {
		return RequestTierPricing{}, false
	}
	channelRules, ok := channelModelRequestTierPricingMap.Get(key)
	if !ok {
		return RequestTierPricing{}, false
	}
	rule, ok := channelRules[FormatMatchingModelName(model)]
	return rule, ok
}

func GetChannelModelRequestTierPricingCopy() map[string]map[string]RequestTierPricing {
	return channelModelRequestTierPricingMap.ReadAll()
}

type legacySegment struct {
	UpTo  int64   `json:"up_to"`
	Ratio float64 `json:"ratio"`
}

// MergeLegacyTierSegmentsToPricing 将旧四路 TierSegments（ratio）合并为统一规则；boundary 固定 lt
func MergeLegacyTierSegmentsToPricing(
	input, output, cacheRead, cacheWrite []legacySegment,
) RequestTierPricing {
	base := input
	if len(base) == 0 {
		base = output
	}
	if len(base) == 0 {
		base = cacheRead
	}
	if len(base) == 0 {
		base = cacheWrite
	}
	if len(base) == 0 {
		return RequestTierPricing{}
	}
	outMap := map[int64]float64{}
	for _, s := range output {
		outMap[s.UpTo] = s.Ratio
	}
	crMap := map[int64]float64{}
	for _, s := range cacheRead {
		crMap[s.UpTo] = s.Ratio
	}
	cwMap := map[int64]float64{}
	for _, s := range cacheWrite {
		cwMap[s.UpTo] = s.Ratio
	}
	inMap := map[int64]float64{}
	for _, s := range input {
		inMap[s.UpTo] = s.Ratio
	}

	tiers := make([]RequestTierBand, 0, len(base))
	for _, s := range base {
		tiers = append(tiers, RequestTierBand{
			UpTo: s.UpTo,
			Prices: RequestTierPrices{
				Input:      TierRatioToPrice(inMap[s.UpTo]),
				Output:     TierRatioToPrice(outMap[s.UpTo]),
				CacheRead:  TierRatioToPrice(crMap[s.UpTo]),
				CacheWrite: TierRatioToPrice(cwMap[s.UpTo]),
			},
		})
	}
	rule := RequestTierPricing{
		Mode:      RequestTierModeProgressive,
		Dimension: RequestTierDimensionInputTokens,
		Boundary:  RequestTierBoundaryLt,
		Currency:  RequestTierCurrencyUSD,
		Tiers:     tiers,
	}
	return normalizeRequestTierPricing(rule)
}

// MigrateLegacyTierRatioMaps 从旧 8 Key 内存结构迁移到新统一结构（仅当新 map 为空时）
// legacy JSON 形状：map[model]{segments:[{up_to,ratio}]} / map[channel]map[model]...
func MigrateLegacyTierRatioMaps(
	modelTier, completionTier, cacheTier, createCacheTier map[string]legacyTierSegmentsWire,
	channelModel, channelCompletion, channelCache, channelCreateCache map[string]map[string]legacyTierSegmentsWire,
) (globalMigrated int, channelMigrated int) {
	if modelRequestTierPricingMap.Len() == 0 {
		merged := make(map[string]RequestTierPricing)
		models := map[string]struct{}{}
		for k := range modelTier {
			models[k] = struct{}{}
		}
		for k := range completionTier {
			models[k] = struct{}{}
		}
		for k := range cacheTier {
			models[k] = struct{}{}
		}
		for k := range createCacheTier {
			models[k] = struct{}{}
		}
		for modelName := range models {
			rule := MergeLegacyTierSegmentsToPricing(
				modelTier[modelName].Segments,
				completionTier[modelName].Segments,
				cacheTier[modelName].Segments,
				createCacheTier[modelName].Segments,
			)
			if len(rule.Tiers) == 0 {
				continue
			}
			if err := ValidateRequestTierPricing(rule); err != nil {
				continue
			}
			merged[FormatMatchingModelName(modelName)] = rule
			globalMigrated++
		}
		if len(merged) > 0 {
			modelRequestTierPricingMap.AddAll(merged)
		}
	}

	if channelModelRequestTierPricingMap.Len() == 0 {
		mergedCh := make(map[string]map[string]RequestTierPricing)
		channelIDs := map[string]struct{}{}
		for k := range channelModel {
			channelIDs[k] = struct{}{}
		}
		for k := range channelCompletion {
			channelIDs[k] = struct{}{}
		}
		for k := range channelCache {
			channelIDs[k] = struct{}{}
		}
		for k := range channelCreateCache {
			channelIDs[k] = struct{}{}
		}
		for chID := range channelIDs {
			models := map[string]struct{}{}
			for k := range channelModel[chID] {
				models[k] = struct{}{}
			}
			for k := range channelCompletion[chID] {
				models[k] = struct{}{}
			}
			for k := range channelCache[chID] {
				models[k] = struct{}{}
			}
			for k := range channelCreateCache[chID] {
				models[k] = struct{}{}
			}
			perModel := make(map[string]RequestTierPricing)
			for modelName := range models {
				rule := MergeLegacyTierSegmentsToPricing(
					channelModel[chID][modelName].Segments,
					channelCompletion[chID][modelName].Segments,
					channelCache[chID][modelName].Segments,
					channelCreateCache[chID][modelName].Segments,
				)
				if len(rule.Tiers) == 0 {
					continue
				}
				if err := ValidateRequestTierPricing(rule); err != nil {
					continue
				}
				perModel[FormatMatchingModelName(modelName)] = rule
				channelMigrated++
			}
			if len(perModel) > 0 {
				mergedCh[chID] = perModel
			}
		}
		if len(mergedCh) > 0 {
			channelModelRequestTierPricingMap.AddAll(mergedCh)
		}
	}
	if globalMigrated > 0 || channelMigrated > 0 {
		InvalidateExposedDataCache()
	}
	return globalMigrated, channelMigrated
}

type legacyTierSegmentsWire struct {
	Segments []legacySegment `json:"segments,omitempty"`
}

// TryMigrateLegacyTierRatioOptionJSON 解析旧 Option JSON 并尝试迁入（供 option 加载后调用）
func TryMigrateLegacyTierRatioOptionJSON(
	modelTierJSON, completionTierJSON, cacheTierJSON, createCacheTierJSON string,
	channelModelJSON, channelCompletionJSON, channelCacheJSON, channelCreateCacheJSON string,
) (bool, error) {
	parseGlobal := func(s string) map[string]legacyTierSegmentsWire {
		out := map[string]legacyTierSegmentsWire{}
		s = strings.TrimSpace(s)
		if s == "" || s == "{}" {
			return out
		}
		_ = common.UnmarshalJsonStr(s, &out)
		return out
	}
	parseChannel := func(s string) map[string]map[string]legacyTierSegmentsWire {
		out := map[string]map[string]legacyTierSegmentsWire{}
		s = strings.TrimSpace(s)
		if s == "" || s == "{}" {
			return out
		}
		_ = common.UnmarshalJsonStr(s, &out)
		return out
	}
	g, c := MigrateLegacyTierRatioMaps(
		parseGlobal(modelTierJSON),
		parseGlobal(completionTierJSON),
		parseGlobal(cacheTierJSON),
		parseGlobal(createCacheTierJSON),
		parseChannel(channelModelJSON),
		parseChannel(channelCompletionJSON),
		parseChannel(channelCacheJSON),
		parseChannel(channelCreateCacheJSON),
	)
	return g > 0 || c > 0, nil
}

// ---------- 兼容导出：从统一规则派生旧 segments（供尚未切换的展示层过渡，可删）----------

// RequestTierSegment 旧 segment 形状（仅迁移/测试兼容）
type RequestTierSegment struct {
	UpTo  int64   `json:"up_to"`
	Ratio float64 `json:"ratio"`
}

type TierSegments struct {
	Segments []RequestTierSegment `json:"segments,omitempty"`
}

func pricingToLegacySegments(rule RequestTierPricing, pricePicker func(RequestTierPrices) float64) TierSegments {
	segs := make([]RequestTierSegment, 0, len(rule.Tiers))
	for _, tier := range rule.Tiers {
		segs = append(segs, RequestTierSegment{
			UpTo:  tier.UpTo,
			Ratio: PriceToTierRatio(pricePicker(tier.Prices)),
		})
	}
	return TierSegments{Segments: segs}
}

func LegacySegmentsFromRequestTierPricing(rule RequestTierPricing) (input, output, cacheRead, cacheWrite TierSegments) {
	return pricingToLegacySegments(rule, func(p RequestTierPrices) float64 { return p.Input }),
		pricingToLegacySegments(rule, func(p RequestTierPrices) float64 { return p.Output }),
		pricingToLegacySegments(rule, func(p RequestTierPrices) float64 { return p.CacheRead }),
		pricingToLegacySegments(rule, func(p RequestTierPrices) float64 { return p.CacheWrite })
}

// MergeLegacyFourTierSegments 供外部用旧 TierSegments 合并
func MergeLegacyFourTierSegments(input, output, cacheRead, cacheWrite TierSegments) RequestTierPricing {
	toWire := func(t TierSegments) []legacySegment {
		out := make([]legacySegment, 0, len(t.Segments))
		for _, s := range t.Segments {
			out = append(out, legacySegment{UpTo: s.UpTo, Ratio: s.Ratio})
		}
		return out
	}
	return MergeLegacyTierSegmentsToPricing(
		toWire(input), toWire(output), toWire(cacheRead), toWire(cacheWrite),
	)
}
