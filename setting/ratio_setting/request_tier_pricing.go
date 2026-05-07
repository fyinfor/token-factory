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

type RequestTierPricingRule struct {
	Mode       string               `json:"mode,omitempty"`
	Input      []RequestTierSegment `json:"input,omitempty"`
	Output     []RequestTierSegment `json:"output,omitempty"`
	CacheRead  []RequestTierSegment `json:"cache_read,omitempty"`
	CacheWrite []RequestTierSegment `json:"cache_write,omitempty"`
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

var requestTierPricingMap = types.NewRWMap[string, RequestTierPricingRule]()
var channelRequestTierPricingMap = types.NewRWMap[string, map[string]RequestTierPricingRule]()
var requestTierPricingTemplatesMap = types.NewRWMap[string, RequestTierPricingTemplate]()

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

func RequestTierPricing2JSONString() string {
	return requestTierPricingMap.MarshalJSONString()
}

func UpdateRequestTierPricingByJSONString(jsonStr string) error {
	trimmed := strings.TrimSpace(jsonStr)
	if trimmed == "" {
		requestTierPricingMap.Clear()
		InvalidateExposedDataCache()
		return nil
	}
	var parsed map[string]RequestTierPricingRule
	if err := common.UnmarshalJsonStr(trimmed, &parsed); err != nil {
		return err
	}
	normalized, err := normalizeRequestTierRuleMap(parsed)
	if err != nil {
		return err
	}
	requestTierPricingMap.Clear()
	requestTierPricingMap.AddAll(normalized)
	InvalidateExposedDataCache()
	return nil
}

func GetRequestTierPricing(model string) (RequestTierPricingRule, bool) {
	return requestTierPricingMap.Get(FormatMatchingModelName(model))
}

func GetRequestTierPricingCopy() map[string]RequestTierPricingRule {
	return requestTierPricingMap.ReadAll()
}

func ChannelRequestTierPricing2JSONString() string {
	return channelRequestTierPricingMap.MarshalJSONString()
}

func UpdateChannelRequestTierPricingByJSONString(jsonStr string) error {
	trimmed := strings.TrimSpace(jsonStr)
	if trimmed == "" {
		channelRequestTierPricingMap.Clear()
		InvalidateExposedDataCache()
		return nil
	}
	var parsed map[string]map[string]RequestTierPricingRule
	if err := common.UnmarshalJsonStr(trimmed, &parsed); err != nil {
		return err
	}
	normalized := make(map[string]map[string]RequestTierPricingRule, len(parsed))
	for channelID, rules := range parsed {
		id, convErr := strconv.Atoi(strings.TrimSpace(channelID))
		if convErr != nil {
			continue
		}
		key := normalizeChannelID(id)
		if key == "" {
			continue
		}
		normalizedRules, err := normalizeRequestTierRuleMap(rules)
		if err != nil {
			return err
		}
		normalized[key] = normalizedRules
	}
	channelRequestTierPricingMap.Clear()
	channelRequestTierPricingMap.AddAll(normalized)
	InvalidateExposedDataCache()
	return nil
}

func GetChannelRequestTierPricing(channelID int, model string) (RequestTierPricingRule, bool) {
	key := normalizeChannelID(channelID)
	if key == "" {
		return RequestTierPricingRule{}, false
	}
	channelRules, ok := channelRequestTierPricingMap.Get(key)
	if !ok {
		return RequestTierPricingRule{}, false
	}
	rule, ok := channelRules[FormatMatchingModelName(model)]
	return rule, ok
}

func GetChannelRequestTierPricingCopy() map[string]map[string]RequestTierPricingRule {
	return channelRequestTierPricingMap.ReadAll()
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

func ResolveRequestTierPricing(channelID int, model string) (RequestTierPricingRule, bool) {
	if requestTierPricingMap == nil || channelRequestTierPricingMap == nil {
		return RequestTierPricingRule{}, false
	}
	if requestTierPricingMap.Len() == 0 && channelRequestTierPricingMap.Len() == 0 {
		return RequestTierPricingRule{}, false
	}
	if rule, ok := GetChannelRequestTierPricing(channelID, model); ok {
		return rule, true
	}
	return GetRequestTierPricing(model)
}
