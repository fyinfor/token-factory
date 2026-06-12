package ratio_setting

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/shopspring/decimal"
)

const RequestTierModeProgressive = "progressive"

type RequestTierSegment struct {
	UpTo  int64   `json:"up_to"`
	Ratio float64 `json:"ratio"`
}

// RequestTierPricingRule 用于模板系统，保留用于兼容性
type RequestTierPricingRule struct {
	Mode       string               `json:"mode,omitempty"`
	Input      []RequestTierSegment `json:"input,omitempty"`
	Output     []RequestTierSegment `json:"output,omitempty"`
	CacheRead  []RequestTierSegment `json:"cache_read,omitempty"`
	CacheWrite []RequestTierSegment `json:"cache_write,omitempty"`
}

// 新的独立阶梯倍率结构
type TierSegments struct {
	Segments []RequestTierSegment `json:"segments,omitempty"`
}

type RequestTierPricingTemplate struct {
	Name string `json:"name,omitempty"`
	RequestTierPricingRule
}

type RequestTierPricingBreakdownItem struct {
	From   int64   `json:"from"`
	To     int64   `json:"to"`
	Tokens string  `json:"tokens"`
	Ratio  float64 `json:"ratio"`
	Result string  `json:"result"`
}

type RequestTierPricingBreakdown struct {
	InputBefore      string                                       `json:"input_before,omitempty"`
	InputAfter       string                                       `json:"input_after,omitempty"`
	OutputBefore     string                                       `json:"output_before,omitempty"`
	OutputAfter      string                                       `json:"output_after,omitempty"`
	CacheReadBefore  string                                       `json:"cache_read_before,omitempty"`
	CacheReadAfter   string                                       `json:"cache_read_after,omitempty"`
	CacheWriteBefore string                                       `json:"cache_write_before,omitempty"`
	CacheWriteAfter  string                                       `json:"cache_write_after,omitempty"`
	Details          map[string][]RequestTierPricingBreakdownItem `json:"details,omitempty"`
}

var requestTierPricingTemplatesMap = types.NewRWMap[string, RequestTierPricingTemplate]()

// 新的四个独立阶梯倍率 Map
var modelTierRatioMap = types.NewRWMap[string, TierSegments]()
var completionTierRatioMap = types.NewRWMap[string, TierSegments]()
var cacheTierRatioMap = types.NewRWMap[string, TierSegments]()
var createCacheTierRatioMap = types.NewRWMap[string, TierSegments]()

// 渠道级别的四个独立阶梯倍率 Map
var channelModelTierRatioMap = types.NewRWMap[string, map[string]TierSegments]()
var channelCompletionTierRatioMap = types.NewRWMap[string, map[string]TierSegments]()
var channelCacheTierRatioMap = types.NewRWMap[string, map[string]TierSegments]()
var channelCreateCacheTierRatioMap = types.NewRWMap[string, map[string]TierSegments]()

// BuildTierLabel 根据实际 token 数量确定所属阶梯档位并生成展示标签
// tokens: 实际使用的 token 数量，用于确定所属档位
// 示例: "输入token<32k"
func BuildTierLabel(tokenType string, segments []RequestTierSegment, tokens int64) string {
	if len(segments) == 0 {
		return ""
	}
	prefix := tokenType + "token"
	// 找到 token 实际所属的档位（按 up_to 递增排列，最后一个 up_to=0 表示无上限）
	var matchedSegment *RequestTierSegment
	previous := int64(0)
	for i := range segments {
		seg := &segments[i]
		if seg.UpTo == 0 {
			// 无上限档位：tokens >= previous 即命中此档
			matchedSegment = seg
			break
		}
		if tokens < seg.UpTo {
			matchedSegment = seg
			break
		}
		previous = seg.UpTo
	}
	if matchedSegment == nil {
		// 兜底：使用最后一个档位
		matchedSegment = &segments[len(segments)-1]
	}
	if matchedSegment.UpTo > 0 {
		return fmt.Sprintf("%s<%d", prefix, matchedSegment.UpTo)
	}
	// 无上限档位
	return fmt.Sprintf("%s≥%d", prefix, previous)
}

// TierSegmentPair 渠道与全局阶梯 segments（与前端 resolveTierSegmentSources 一致）
type TierSegmentPair struct {
	Channel []RequestTierSegment
	Global  []RequestTierSegment
}

// LoadTierSegmentPair 分别加载渠道与全局阶梯 segments
func LoadTierSegmentPair(
	channelID int,
	model string,
	getChannel func(int, string) (TierSegments, bool),
	getGlobal func(string) (TierSegments, bool),
) TierSegmentPair {
	var pair TierSegmentPair
	if globalTier, ok := getGlobal(model); ok {
		pair.Global = globalTier.Segments
	}
	if channelTier, ok := getChannel(channelID, model); ok {
		pair.Channel = channelTier.Segments
	}
	return pair
}

// SelectBandSegments 区间结构优先渠道阶梯，否则全局阶梯
func SelectBandSegments(pair TierSegmentPair) []RequestTierSegment {
	if len(pair.Channel) > 0 {
		return pair.Channel
	}
	return pair.Global
}

// FindTierBandFromTokens 根据 token 数量确定所属区间起点（fromToken）
func FindTierBandFromTokens(tokens int64, segments []RequestTierSegment) int64 {
	if len(segments) == 0 {
		return 0
	}
	previous := int64(0)
	for i := range segments {
		seg := &segments[i]
		if seg.UpTo == 0 {
			return previous
		}
		if tokens < seg.UpTo {
			return previous
		}
		previous = seg.UpTo
	}
	return previous
}

// FindTierRatioAtBand 按区间起点在 segments 中查找倍率
func FindTierRatioAtBand(segments []RequestTierSegment, fromToken int64) (float64, bool) {
	for _, seg := range segments {
		if seg.UpTo == 0 || fromToken < seg.UpTo {
			return seg.Ratio, true
		}
	}
	return 0, false
}

func tierEffectiveRate(channelRatio, globalRatio, costDiscPercent, markupDiscPercent float64) float64 {
	return channelRatio*(costDiscPercent/100.0) + globalRatio*(markupDiscPercent/100.0)
}

// TierPlatformUsdPerM 计算命中档位下的平台展示单价（$/1M tokens，含分组倍率）
// 与前端 buildTokenTierPreviewItems 公式一致：
//
//	(baseRatio × costDisc% + globalRatio × markupDisc%) × 2 × groupRatio
func TierPlatformUsdPerM(
	channelSegments, globalSegments []RequestTierSegment,
	bandFrom int64,
	costDiscPercent, markupDiscPercent, groupRatio float64,
) float64 {
	globalRatio, globalOk := FindTierRatioAtBand(globalSegments, bandFrom)
	if !globalOk {
		globalRatio = 0
	}
	baseRatio := globalRatio
	if len(channelSegments) > 0 {
		if channelRatio, ok := FindTierRatioAtBand(channelSegments, bandFrom); ok {
			baseRatio = channelRatio
		}
	}
	effRate := tierEffectiveRate(baseRatio, globalRatio, costDiscPercent, markupDiscPercent)
	return effRate * 2 * groupRatio
}

func normalizeRequestTierSegments(segments []RequestTierSegment) []RequestTierSegment {
	out := make([]RequestTierSegment, 0, len(segments))
	for _, segment := range segments {
		if segment.Ratio < 0 {
			continue
		}
		out = append(out, segment)
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

func normalizeRequestTierRule(rule RequestTierPricingRule) RequestTierPricingRule {
	if strings.TrimSpace(rule.Mode) == "" {
		rule.Mode = RequestTierModeProgressive
	}
	rule.Input = normalizeRequestTierSegments(rule.Input)
	rule.Output = normalizeRequestTierSegments(rule.Output)
	rule.CacheRead = normalizeRequestTierSegments(rule.CacheRead)
	rule.CacheWrite = normalizeRequestTierSegments(rule.CacheWrite)
	return rule
}

func validateRequestTierSegments(name string, segments []RequestTierSegment) error {
	if len(segments) == 0 {
		return nil
	}
	previous := int64(0)
	for i, segment := range segments {
		if segment.Ratio < 0 {
			return fmt.Errorf("%s 第 %d 档 ratio 不能小于 0", name, i+1)
		}
		if segment.UpTo == 0 {
			if i != len(segments)-1 {
				return fmt.Errorf("%s 只有最后一档 up_to 可以为 0", name)
			}
			continue
		}
		if segment.UpTo <= previous {
			return fmt.Errorf("%s 第 %d 档 up_to 必须递增", name, i+1)
		}
		previous = segment.UpTo
	}
	return nil
}

func ValidateRequestTierRule(rule RequestTierPricingRule) error {
	mode := strings.TrimSpace(rule.Mode)
	if mode == "" {
		mode = RequestTierModeProgressive
	}
	if mode != RequestTierModeProgressive {
		return errors.New("仅支持 progressive 阶梯计费模式")
	}
	if err := validateRequestTierSegments("input", rule.Input); err != nil {
		return err
	}
	if err := validateRequestTierSegments("output", rule.Output); err != nil {
		return err
	}
	if err := validateRequestTierSegments("cache_read", rule.CacheRead); err != nil {
		return err
	}
	if err := validateRequestTierSegments("cache_write", rule.CacheWrite); err != nil {
		return err
	}
	return nil
}

func normalizeRequestTierRuleMap(src map[string]RequestTierPricingRule) (map[string]RequestTierPricingRule, error) {
	dst := make(map[string]RequestTierPricingRule, len(src))
	for modelName, rule := range src {
		name := FormatMatchingModelName(strings.TrimSpace(modelName))
		if name == "" {
			continue
		}
		rule = normalizeRequestTierRule(rule)
		if err := ValidateRequestTierRule(rule); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		dst[name] = rule
	}
	return dst, nil
}

// ========== 新的四个独立阶梯倍率处理函数 ==========

func normalizeTierSegmentsMap(src map[string]TierSegments) (map[string]TierSegments, error) {
	dst := make(map[string]TierSegments, len(src))
	for modelName, tier := range src {
		name := FormatMatchingModelName(strings.TrimSpace(modelName))
		if name == "" {
			continue
		}
		tier.Segments = normalizeRequestTierSegments(tier.Segments)
		if err := validateRequestTierSegments("segments", tier.Segments); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		dst[name] = tier
	}
	return dst, nil
}

// ModelTierRatio
func ModelTierRatio2JSONString() string {
	return modelTierRatioMap.MarshalJSONString()
}

func UpdateModelTierRatioByJSONString(jsonStr string) error {
	trimmed := strings.TrimSpace(jsonStr)
	if trimmed == "" {
		modelTierRatioMap.Clear()
		InvalidateExposedDataCache()
		return nil
	}
	var parsed map[string]TierSegments
	if err := common.UnmarshalJsonStr(trimmed, &parsed); err != nil {
		return err
	}
	normalized, err := normalizeTierSegmentsMap(parsed)
	if err != nil {
		return err
	}
	modelTierRatioMap.Clear()
	modelTierRatioMap.AddAll(normalized)
	InvalidateExposedDataCache()
	return nil
}

func GetModelTierRatio(model string) (TierSegments, bool) {
	return modelTierRatioMap.Get(FormatMatchingModelName(model))
}

func GetModelTierRatioCopy() map[string]TierSegments {
	return modelTierRatioMap.ReadAll()
}

// ChannelModelTierRatio
func ChannelModelTierRatio2JSONString() string {
	return channelModelTierRatioMap.MarshalJSONString()
}

func UpdateChannelModelTierRatioByJSONString(jsonStr string) error {
	trimmed := strings.TrimSpace(jsonStr)
	if trimmed == "" {
		channelModelTierRatioMap.Clear()
		InvalidateExposedDataCache()
		return nil
	}
	var parsed map[string]map[string]TierSegments
	if err := common.UnmarshalJsonStr(trimmed, &parsed); err != nil {
		return err
	}
	normalized := make(map[string]map[string]TierSegments, len(parsed))
	for channelID, rules := range parsed {
		id, convErr := strconv.Atoi(strings.TrimSpace(channelID))
		if convErr != nil {
			continue
		}
		key := normalizeChannelID(id)
		if key == "" {
			continue
		}
		normalizedRules, err := normalizeTierSegmentsMap(rules)
		if err != nil {
			return err
		}
		normalized[key] = normalizedRules
	}
	channelModelTierRatioMap.Clear()
	channelModelTierRatioMap.AddAll(normalized)
	InvalidateExposedDataCache()
	return nil
}

func GetChannelModelTierRatio(channelID int, model string) (TierSegments, bool) {
	key := normalizeChannelID(channelID)
	if key == "" {
		return TierSegments{}, false
	}
	channelRules, ok := channelModelTierRatioMap.Get(key)
	if !ok {
		return TierSegments{}, false
	}
	rule, ok := channelRules[FormatMatchingModelName(model)]
	return rule, ok
}

func GetChannelModelTierRatioCopy() map[string]map[string]TierSegments {
	return channelModelTierRatioMap.ReadAll()
}

// CompletionTierRatio
func CompletionTierRatio2JSONString() string {
	return completionTierRatioMap.MarshalJSONString()
}

func UpdateCompletionTierRatioByJSONString(jsonStr string) error {
	trimmed := strings.TrimSpace(jsonStr)
	if trimmed == "" {
		completionTierRatioMap.Clear()
		InvalidateExposedDataCache()
		return nil
	}
	var parsed map[string]TierSegments
	if err := common.UnmarshalJsonStr(trimmed, &parsed); err != nil {
		return err
	}
	normalized, err := normalizeTierSegmentsMap(parsed)
	if err != nil {
		return err
	}
	completionTierRatioMap.Clear()
	completionTierRatioMap.AddAll(normalized)
	InvalidateExposedDataCache()
	return nil
}

func GetCompletionTierRatio(model string) (TierSegments, bool) {
	return completionTierRatioMap.Get(FormatMatchingModelName(model))
}

func GetCompletionTierRatioCopy() map[string]TierSegments {
	return completionTierRatioMap.ReadAll()
}

// ChannelCompletionTierRatio
func ChannelCompletionTierRatio2JSONString() string {
	return channelCompletionTierRatioMap.MarshalJSONString()
}

func UpdateChannelCompletionTierRatioByJSONString(jsonStr string) error {
	trimmed := strings.TrimSpace(jsonStr)
	if trimmed == "" {
		channelCompletionTierRatioMap.Clear()
		InvalidateExposedDataCache()
		return nil
	}
	var parsed map[string]map[string]TierSegments
	if err := common.UnmarshalJsonStr(trimmed, &parsed); err != nil {
		return err
	}
	normalized := make(map[string]map[string]TierSegments, len(parsed))
	for channelID, rules := range parsed {
		id, convErr := strconv.Atoi(strings.TrimSpace(channelID))
		if convErr != nil {
			continue
		}
		key := normalizeChannelID(id)
		if key == "" {
			continue
		}
		normalizedRules, err := normalizeTierSegmentsMap(rules)
		if err != nil {
			return err
		}
		normalized[key] = normalizedRules
	}
	channelCompletionTierRatioMap.Clear()
	channelCompletionTierRatioMap.AddAll(normalized)
	InvalidateExposedDataCache()
	return nil
}

func GetChannelCompletionTierRatio(channelID int, model string) (TierSegments, bool) {
	key := normalizeChannelID(channelID)
	if key == "" {
		return TierSegments{}, false
	}
	channelRules, ok := channelCompletionTierRatioMap.Get(key)
	if !ok {
		return TierSegments{}, false
	}
	rule, ok := channelRules[FormatMatchingModelName(model)]
	return rule, ok
}

func GetChannelCompletionTierRatioCopy() map[string]map[string]TierSegments {
	return channelCompletionTierRatioMap.ReadAll()
}

// CacheTierRatio
func CacheTierRatio2JSONString() string {
	return cacheTierRatioMap.MarshalJSONString()
}

func UpdateCacheTierRatioByJSONString(jsonStr string) error {
	trimmed := strings.TrimSpace(jsonStr)
	if trimmed == "" {
		cacheTierRatioMap.Clear()
		InvalidateExposedDataCache()
		return nil
	}
	var parsed map[string]TierSegments
	if err := common.UnmarshalJsonStr(trimmed, &parsed); err != nil {
		return err
	}
	normalized, err := normalizeTierSegmentsMap(parsed)
	if err != nil {
		return err
	}
	cacheTierRatioMap.Clear()
	cacheTierRatioMap.AddAll(normalized)
	InvalidateExposedDataCache()
	return nil
}

func GetCacheTierRatio(model string) (TierSegments, bool) {
	return cacheTierRatioMap.Get(FormatMatchingModelName(model))
}

func GetCacheTierRatioCopy() map[string]TierSegments {
	return cacheTierRatioMap.ReadAll()
}

// ChannelCacheTierRatio
func ChannelCacheTierRatio2JSONString() string {
	return channelCacheTierRatioMap.MarshalJSONString()
}

func UpdateChannelCacheTierRatioByJSONString(jsonStr string) error {
	trimmed := strings.TrimSpace(jsonStr)
	if trimmed == "" {
		channelCacheTierRatioMap.Clear()
		InvalidateExposedDataCache()
		return nil
	}
	var parsed map[string]map[string]TierSegments
	if err := common.UnmarshalJsonStr(trimmed, &parsed); err != nil {
		return err
	}
	normalized := make(map[string]map[string]TierSegments, len(parsed))
	for channelID, rules := range parsed {
		id, convErr := strconv.Atoi(strings.TrimSpace(channelID))
		if convErr != nil {
			continue
		}
		key := normalizeChannelID(id)
		if key == "" {
			continue
		}
		normalizedRules, err := normalizeTierSegmentsMap(rules)
		if err != nil {
			return err
		}
		normalized[key] = normalizedRules
	}
	channelCacheTierRatioMap.Clear()
	channelCacheTierRatioMap.AddAll(normalized)
	InvalidateExposedDataCache()
	return nil
}

func GetChannelCacheTierRatio(channelID int, model string) (TierSegments, bool) {
	key := normalizeChannelID(channelID)
	if key == "" {
		return TierSegments{}, false
	}
	channelRules, ok := channelCacheTierRatioMap.Get(key)
	if !ok {
		return TierSegments{}, false
	}
	rule, ok := channelRules[FormatMatchingModelName(model)]
	return rule, ok
}

func GetChannelCacheTierRatioCopy() map[string]map[string]TierSegments {
	return channelCacheTierRatioMap.ReadAll()
}

// CreateCacheTierRatio
func CreateCacheTierRatio2JSONString() string {
	return createCacheTierRatioMap.MarshalJSONString()
}

func UpdateCreateCacheTierRatioByJSONString(jsonStr string) error {
	trimmed := strings.TrimSpace(jsonStr)
	if trimmed == "" {
		createCacheTierRatioMap.Clear()
		InvalidateExposedDataCache()
		return nil
	}
	var parsed map[string]TierSegments
	if err := common.UnmarshalJsonStr(trimmed, &parsed); err != nil {
		return err
	}
	normalized, err := normalizeTierSegmentsMap(parsed)
	if err != nil {
		return err
	}
	createCacheTierRatioMap.Clear()
	createCacheTierRatioMap.AddAll(normalized)
	InvalidateExposedDataCache()
	return nil
}

func GetCreateCacheTierRatio(model string) (TierSegments, bool) {
	return createCacheTierRatioMap.Get(FormatMatchingModelName(model))
}

func GetCreateCacheTierRatioCopy() map[string]TierSegments {
	return createCacheTierRatioMap.ReadAll()
}

// ChannelCreateCacheTierRatio
func ChannelCreateCacheTierRatio2JSONString() string {
	return channelCreateCacheTierRatioMap.MarshalJSONString()
}

func UpdateChannelCreateCacheTierRatioByJSONString(jsonStr string) error {
	trimmed := strings.TrimSpace(jsonStr)
	if trimmed == "" {
		channelCreateCacheTierRatioMap.Clear()
		InvalidateExposedDataCache()
		return nil
	}
	var parsed map[string]map[string]TierSegments
	if err := common.UnmarshalJsonStr(trimmed, &parsed); err != nil {
		return err
	}
	normalized := make(map[string]map[string]TierSegments, len(parsed))
	for channelID, rules := range parsed {
		id, convErr := strconv.Atoi(strings.TrimSpace(channelID))
		if convErr != nil {
			continue
		}
		key := normalizeChannelID(id)
		if key == "" {
			continue
		}
		normalizedRules, err := normalizeTierSegmentsMap(rules)
		if err != nil {
			return err
		}
		normalized[key] = normalizedRules
	}
	channelCreateCacheTierRatioMap.Clear()
	channelCreateCacheTierRatioMap.AddAll(normalized)
	InvalidateExposedDataCache()
	return nil
}

func GetChannelCreateCacheTierRatio(channelID int, model string) (TierSegments, bool) {
	key := normalizeChannelID(channelID)
	if key == "" {
		return TierSegments{}, false
	}
	channelRules, ok := channelCreateCacheTierRatioMap.Get(key)
	if !ok {
		return TierSegments{}, false
	}
	rule, ok := channelRules[FormatMatchingModelName(model)]
	return rule, ok
}

func GetChannelCreateCacheTierRatioCopy() map[string]map[string]TierSegments {
	return channelCreateCacheTierRatioMap.ReadAll()
}

func RequestTierPricingTemplates2JSONString() string {
	return requestTierPricingTemplatesMap.MarshalJSONString()
}

func requestTierTemplateID(template RequestTierPricingTemplate, index int, existing map[string]RequestTierPricingTemplate) string {
	base := strings.TrimSpace(template.Name)
	if base == "" {
		base = "template"
	}
	base = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			return r
		}
		return '_'
	}, strings.ToLower(base))
	base = strings.Trim(base, "_")
	if base == "" {
		base = "template"
	}
	prefix := fmt.Sprintf("tpl_%s_%d", base, time.Now().UnixNano())
	id := fmt.Sprintf("%s_%d", prefix, index+1)
	for suffix := 2; ; suffix++ {
		if _, ok := existing[id]; !ok {
			return id
		}
		id = fmt.Sprintf("%s_%d_%d", prefix, index+1, suffix)
	}
}

func UpdateRequestTierPricingTemplatesByJSONString(jsonStr string) error {
	trimmed := strings.TrimSpace(jsonStr)
	if trimmed == "" {
		requestTierPricingTemplatesMap.Clear()
		return nil
	}
	var parsed map[string]RequestTierPricingTemplate
	if err := common.UnmarshalJsonStr(trimmed, &parsed); err != nil {
		var list []RequestTierPricingTemplate
		if listErr := common.UnmarshalJsonStr(trimmed, &list); listErr != nil {
			return err
		}
		parsed = make(map[string]RequestTierPricingTemplate, len(list))
		for index, template := range list {
			parsed[requestTierTemplateID(template, index, parsed)] = template
		}
	}
	normalized := make(map[string]RequestTierPricingTemplate, len(parsed))
	index := 0
	for key, template := range parsed {
		name := strings.TrimSpace(key)
		if name == "" {
			name = requestTierTemplateID(template, index, normalized)
		}
		template.RequestTierPricingRule = normalizeRequestTierRule(template.RequestTierPricingRule)
		if err := ValidateRequestTierRule(template.RequestTierPricingRule); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		normalized[name] = template
		index++
	}
	requestTierPricingTemplatesMap.Clear()
	requestTierPricingTemplatesMap.AddAll(normalized)
	return nil
}

func applyTierSegments(tokens decimal.Decimal, segments []RequestTierSegment) (decimal.Decimal, []RequestTierPricingBreakdownItem) {
	if tokens.LessThanOrEqual(decimal.Zero) || len(segments) == 0 {
		return tokens, nil
	}
	remaining := tokens
	previous := decimal.Zero
	result := decimal.Zero
	items := make([]RequestTierPricingBreakdownItem, 0, len(segments))
	for _, segment := range segments {
		if remaining.LessThanOrEqual(decimal.Zero) {
			break
		}
		var size decimal.Decimal
		var to int64
		if segment.UpTo == 0 {
			size = remaining
			to = 0
		} else {
			upper := decimal.NewFromInt(segment.UpTo)
			if upper.LessThanOrEqual(previous) {
				continue
			}
			capacity := upper.Sub(previous)
			if remaining.GreaterThan(capacity) {
				size = capacity
			} else {
				size = remaining
			}
			to = segment.UpTo
		}
		segResult := size.Mul(decimal.NewFromFloat(segment.Ratio))
		result = result.Add(segResult)
		items = append(items, RequestTierPricingBreakdownItem{
			From:   previous.IntPart(),
			To:     to,
			Tokens: size.String(),
			Ratio:  segment.Ratio,
			Result: segResult.String(),
		})
		remaining = remaining.Sub(size)
		if segment.UpTo == 0 {
			previous = previous.Add(size)
		} else {
			previous = decimal.NewFromInt(segment.UpTo)
		}
	}
	if remaining.GreaterThan(decimal.Zero) {
		result = result.Add(remaining)
	}
	return result, items
}

// ApplyTierSegmentsForType 应用阶梯倍率到单个类型
func ApplyTierSegmentsForType(tokens decimal.Decimal, tier TierSegments) decimal.Decimal {
	if tokens.LessThanOrEqual(decimal.Zero) || len(tier.Segments) == 0 {
		return tokens
	}
	remaining := tokens
	previous := decimal.Zero
	result := decimal.Zero
	for _, segment := range tier.Segments {
		if remaining.LessThanOrEqual(decimal.Zero) {
			break
		}
		var size decimal.Decimal
		if segment.UpTo == 0 {
			size = remaining
		} else {
			upper := decimal.NewFromInt(segment.UpTo)
			if upper.LessThanOrEqual(previous) {
				continue
			}
			capacity := upper.Sub(previous)
			if remaining.GreaterThan(capacity) {
				size = capacity
			} else {
				size = remaining
			}
		}
		segResult := size.Mul(decimal.NewFromFloat(segment.Ratio))
		result = result.Add(segResult)
		remaining = remaining.Sub(size)
		if segment.UpTo == 0 {
			previous = previous.Add(size)
		} else {
			previous = decimal.NewFromInt(segment.UpTo)
		}
	}
	if remaining.GreaterThan(decimal.Zero) {
		result = result.Add(remaining)
	}
	return result
}

// ResolveModelTierRatio 解析模型阶梯倍率（优先使用渠道配置）
func ResolveModelTierRatio(channelID int, model string) (TierSegments, bool) {
	if modelTierRatioMap == nil || channelModelTierRatioMap == nil {
		return TierSegments{}, false
	}
	if modelTierRatioMap.Len() == 0 && channelModelTierRatioMap.Len() == 0 {
		return TierSegments{}, false
	}
	if rule, ok := GetChannelModelTierRatio(channelID, model); ok {
		return rule, true
	}
	return GetModelTierRatio(model)
}

// ResolveCompletionTierRatio 解析完成阶梯倍率（优先使用渠道配置）
func ResolveCompletionTierRatio(channelID int, model string) (TierSegments, bool) {
	if completionTierRatioMap == nil || channelCompletionTierRatioMap == nil {
		return TierSegments{}, false
	}
	if completionTierRatioMap.Len() == 0 && channelCompletionTierRatioMap.Len() == 0 {
		return TierSegments{}, false
	}
	if rule, ok := GetChannelCompletionTierRatio(channelID, model); ok {
		return rule, true
	}
	return GetCompletionTierRatio(model)
}

// ResolveCacheTierRatio 解析缓存读取阶梯倍率（优先使用渠道配置）
func ResolveCacheTierRatio(channelID int, model string) (TierSegments, bool) {
	if cacheTierRatioMap == nil || channelCacheTierRatioMap == nil {
		return TierSegments{}, false
	}
	if cacheTierRatioMap.Len() == 0 && channelCacheTierRatioMap.Len() == 0 {
		return TierSegments{}, false
	}
	if rule, ok := GetChannelCacheTierRatio(channelID, model); ok {
		return rule, true
	}
	return GetCacheTierRatio(model)
}

// ResolveCreateCacheTierRatio 解析缓存写入阶梯倍率（优先使用渠道配置）
func ResolveCreateCacheTierRatio(channelID int, model string) (TierSegments, bool) {
	if createCacheTierRatioMap == nil || channelCreateCacheTierRatioMap == nil {
		return TierSegments{}, false
	}
	if createCacheTierRatioMap.Len() == 0 && channelCreateCacheTierRatioMap.Len() == 0 {
		return TierSegments{}, false
	}
	if rule, ok := GetChannelCreateCacheTierRatio(channelID, model); ok {
		return rule, true
	}
	return GetCreateCacheTierRatio(model)
}

func ApplyRequestTierPricingDecimal(rule RequestTierPricingRule, input, output, cacheRead, cacheWrite decimal.Decimal) (decimal.Decimal, decimal.Decimal, decimal.Decimal, decimal.Decimal, RequestTierPricingBreakdown) {
	rule = normalizeRequestTierRule(rule)
	inAfter, inItems := applyTierSegments(input, rule.Input)
	outAfter, outItems := applyTierSegments(output, rule.Output)
	cacheReadAfter, cacheReadItems := applyTierSegments(cacheRead, rule.CacheRead)
	cacheWriteAfter, cacheWriteItems := applyTierSegments(cacheWrite, rule.CacheWrite)
	breakdown := RequestTierPricingBreakdown{
		InputBefore:      input.String(),
		InputAfter:       inAfter.String(),
		OutputBefore:     output.String(),
		OutputAfter:      outAfter.String(),
		CacheReadBefore:  cacheRead.String(),
		CacheReadAfter:   cacheReadAfter.String(),
		CacheWriteBefore: cacheWrite.String(),
		CacheWriteAfter:  cacheWriteAfter.String(),
		Details:          map[string][]RequestTierPricingBreakdownItem{},
	}
	if len(inItems) > 0 {
		breakdown.Details["input"] = inItems
	}
	if len(outItems) > 0 {
		breakdown.Details["output"] = outItems
	}
	if len(cacheReadItems) > 0 {
		breakdown.Details["cache_read"] = cacheReadItems
	}
	if len(cacheWriteItems) > 0 {
		breakdown.Details["cache_write"] = cacheWriteItems
	}
	if len(breakdown.Details) == 0 {
		breakdown.Details = nil
	}
	return inAfter, outAfter, cacheReadAfter, cacheWriteAfter, breakdown
}
