package model

import (
	"sort"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

type videoUpscaleAcc struct {
	label  string
	prices []float64
}

func attachVideoUpscaleTiers(hint *VideoFlatClipPricingHint, tiers []VideoUpscaleTierRow) *VideoFlatClipPricingHint {
	if len(tiers) == 0 {
		return hint
	}
	if hint == nil {
		hint = &VideoFlatClipPricingHint{}
	}
	hint.UpscaleTiers = tiers
	return hint
}

func buildVideoUpscaleTiers(channelID int, modelName string, costDiscPercent, markupDiscPercent float64) []VideoUpscaleTierRow {
	var channelRows, globalRows []ratio_setting.VideoUpscalePriceRule
	if channelID > 0 {
		if rules, ok := ratio_setting.GetChannelVideoPricingRules(channelID, modelName); ok {
			channelRows = rules.VideoUpscalePerSecond
		}
	}
	if rules, ok := ratio_setting.GetVideoPricingRules(modelName); ok {
		globalRows = rules.VideoUpscalePerSecond
	}
	return mergeVideoUpscaleTiers(channelRows, globalRows, costDiscPercent, markupDiscPercent)
}

// mergeVideoUpscaleTiers 按目标分辨率去重：同档多条取最低原价；渠道档覆盖全局档。
// 有效价与实扣一致：EffectiveRuleUnitPrice(found, 0, cost, markup)。
func mergeVideoUpscaleTiers(
	channelRows, globalRows []ratio_setting.VideoUpscalePriceRule,
	costDiscPercent, markupDiscPercent float64,
) []VideoUpscaleTierRow {
	globalAcc := collectVideoUpscaleByTarget(globalRows)
	channelAcc := collectVideoUpscaleByTarget(channelRows)
	keys := make([]string, 0, len(globalAcc)+len(channelAcc))
	seen := make(map[string]struct{}, len(globalAcc)+len(channelAcc))
	for key := range globalAcc {
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	for key := range channelAcc {
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	if len(keys) == 0 {
		return nil
	}
	sort.Slice(keys, func(i, j int) bool {
		return videoUpscaleResolutionRank(keys[i]) < videoUpscaleResolutionRank(keys[j])
	})
	out := make([]VideoUpscaleTierRow, 0, len(keys))
	for _, key := range keys {
		acc, ok := channelAcc[key]
		if !ok {
			acc = globalAcc[key]
		}
		raw := minPositiveFloat(acc.prices)
		if raw <= 0 {
			continue
		}
		usd := EffectiveRuleUnitPrice(raw, 0, costDiscPercent, markupDiscPercent)
		if usd <= 0 {
			continue
		}
		out = append(out, VideoUpscaleTierRow{
			Resolution:              acc.label,
			UsdAfterChannelDiscount: usd,
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func collectVideoUpscaleByTarget(rows []ratio_setting.VideoUpscalePriceRule) map[string]videoUpscaleAcc {
	out := make(map[string]videoUpscaleAcc)
	for _, row := range rows {
		if row.Price <= 0 {
			continue
		}
		label := common.FormatVideoResolutionLabel(row.Resolution)
		if label == "" {
			label = strings.TrimSpace(row.Resolution)
		}
		if label == "" {
			continue
		}
		key := strings.ToLower(label)
		acc := out[key]
		acc.label = label
		acc.prices = append(acc.prices, row.Price)
		out[key] = acc
	}
	return out
}

func minPositiveFloat(values []float64) float64 {
	min := 0.0
	for _, v := range values {
		if v <= 0 {
			continue
		}
		if min <= 0 || v < min {
			min = v
		}
	}
	return min
}

func videoUpscaleResolutionRank(label string) int {
	s := strings.ToLower(strings.TrimSpace(label))
	if s == "" {
		return 1 << 30
	}
	if strings.HasSuffix(s, "k") {
		n, err := strconv.Atoi(strings.TrimSuffix(s, "k"))
		if err == nil && n > 0 {
			return n * 720
		}
	}
	n, err := strconv.Atoi(strings.TrimSuffix(s, "p"))
	if err == nil && n > 0 {
		return n
	}
	return 1 << 30
}
