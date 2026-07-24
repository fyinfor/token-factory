package model

import (
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

// VideoFlatClipTierRow 单档视频标价（已套用成本折扣+加价折扣的有效展示价，未乘用户分组倍率）。
type VideoFlatClipTierRow struct {
	UsdAfterChannelDiscount float64 `json:"usd_after_channel_discount"`
	UsdChannelRaw           float64 `json:"usd_channel_raw,omitempty"`
	UsdOfficial             float64 `json:"usd_official,omitempty"`
	Resolution              string  `json:"resolution,omitempty"`
	HasAudio                *bool   `json:"has_audio,omitempty"`
	Lane                    string  `json:"lane,omitempty"`
}

// VideoFlatClipPricingHint 多档视频分辨率价在定价卡片上的摘要（按条或按秒，见 BillingMode）：
// MinUsdAfterChannelDiscount = min(渠道规则价×成本折扣% + 全局规则价×加价折扣%)，与实扣公式一致；
// 前端再乘用户当前分组倍率后走 displayPrice。
// Tiers 为全部档位（同口径），供「查看更多价格」表格展示。
type VideoFlatClipPricingHint struct {
	MinUsdAfterChannelDiscount float64                `json:"min_usd_after_channel_discount"`
	Resolution                 string                 `json:"resolution,omitempty"`
	HasAudio                   *bool                  `json:"has_audio,omitempty"`
	Lane                       string                 `json:"lane,omitempty"`
	TierCount                  int                    `json:"tier_count"`
	Tiers                      []VideoFlatClipTierRow `json:"tiers,omitempty"`
	// BillingMode：per_item 为按条成片；per_second 为按秒（如 Seedance2.0 常见 text_to_video_per_second）。
	BillingMode string `json:"billing_mode,omitempty"`
}

type videoFlatTier struct {
	RawUSD   float64
	Res      string
	HasAudio *bool
	Lane     string
}

func laneOrderVideoFlat(l string) int {
	switch l {
	case "text_to_video":
		return 0
	case "image_to_video":
		return 1
	case "video_to_video":
		return 2
	case "text_to_video_legacy":
		return 3
	case "image_to_video_legacy":
		return 4
	case "video_to_video_input_legacy":
		return 5
	case "video_to_video_output_legacy":
		return 6
	case "text_to_video_per_second":
		return 10
	case "image_to_video_per_second":
		return 11
	case "video_to_video_per_second":
		return 12
	default:
		return 99
	}
}

func audioPtrRank(p *bool) int {
	if p == nil {
		return 2
	}
	if !*p {
		return 0
	}
	return 1
}

func tierLessVideoFlat(a, b videoFlatTier) bool {
	ar := strings.TrimSpace(strings.ToLower(a.Res))
	br := strings.TrimSpace(strings.ToLower(b.Res))
	if ar != br {
		return ar < br
	}
	if laneOrderVideoFlat(a.Lane) != laneOrderVideoFlat(b.Lane) {
		return laneOrderVideoFlat(a.Lane) < laneOrderVideoFlat(b.Lane)
	}
	return audioPtrRank(a.HasAudio) < audioPtrRank(b.HasAudio)
}

func tierDedupeKey(ti videoFlatTier) string {
	return ti.Lane + "\x00" + strings.TrimSpace(strings.ToLower(ti.Res)) + "\x00" +
		strconv.FormatFloat(ti.RawUSD, 'f', 8, 64)
}

// collapsePairedUnifiedAudioTiers 同 lane+分辨率+价格 下同时存在 has_audio true/false 时合并为一条（HasAudio=nil），供展示「统一」。
func collapsePairedUnifiedAudioTiers(tiers []videoFlatTier) []videoFlatTier {
	order := make([]string, 0)
	groups := make(map[string][]videoFlatTier)
	for _, ti := range tiers {
		k := tierDedupeKey(ti)
		if _, ok := groups[k]; !ok {
			order = append(order, k)
		}
		groups[k] = append(groups[k], ti)
	}
	out := make([]videoFlatTier, 0, len(tiers))
	for _, k := range order {
		g := groups[k]
		if len(g) == 2 {
			a, b := g[0], g[1]
			if a.HasAudio != nil && b.HasAudio != nil && *a.HasAudio != *b.HasAudio &&
				math.Abs(a.RawUSD-b.RawUSD) < 1e-9 &&
				strings.EqualFold(strings.TrimSpace(a.Res), strings.TrimSpace(b.Res)) &&
				a.Lane == b.Lane {
				out = append(out, videoFlatTier{
					RawUSD:   a.RawUSD,
					Res:      a.Res,
					Lane:     a.Lane,
					HasAudio: nil,
				})
				continue
			}
		}
		out = append(out, g...)
	}
	return out
}

// normalizeLegacyAllFalseToUnifiedHintTiers 旧版前端只写入 has_audio:false 的统一价；当每条档位均为 false（无 nil、无 true）时改为 nil 以便展示「统一」。
func normalizeLegacyAllFalseToUnifiedHintTiers(tiers []videoFlatTier) []videoFlatTier {
	if len(tiers) == 0 {
		return tiers
	}
	allExplicitFalse := true
	for i := range tiers {
		t := tiers[i]
		if t.HasAudio == nil {
			allExplicitFalse = false
			break
		}
		if *t.HasAudio {
			allExplicitFalse = false
			break
		}
	}
	if !allExplicitFalse {
		return tiers
	}
	out := make([]videoFlatTier, len(tiers))
	for i := range tiers {
		tt := tiers[i]
		tt.HasAudio = nil
		out[i] = tt
	}
	return out
}

func appendPerItemTiers(dst *[]videoFlatTier, rows []ratio_setting.VideoResolutionAudioPriceRule, lane string) {
	for i := range rows {
		r := rows[i]
		if r.Price <= 0 {
			continue
		}
		ha := r.HasAudio
		*dst = append(*dst, videoFlatTier{
			RawUSD:   r.Price,
			Res:      r.Resolution,
			HasAudio: &ha,
			Lane:     lane,
		})
	}
}

func appendLegacyPerVideoTiers(dst *[]videoFlatTier, rows []ratio_setting.VideoResolutionPerVideoRule, lane string) {
	for i := range rows {
		r := rows[i]
		if r.VideoPrice <= 0 {
			continue
		}
		*dst = append(*dst, videoFlatTier{
			RawUSD:   r.VideoPrice,
			Res:      r.Resolution,
			HasAudio: nil,
			Lane:     lane,
		})
	}
}

func collectVideoFlatTiers(rules ratio_setting.VideoPricingRules) []videoFlatTier {
	out := make([]videoFlatTier, 0, 48)
	appendPerItemTiers(&out, rules.TextToVideoPerItem, "text_to_video")
	appendPerItemTiers(&out, rules.ImageToVideoPerItem, "image_to_video")
	appendPerItemTiers(&out, rules.VideoToVideoPerItem, "video_to_video")
	appendLegacyPerVideoTiers(&out, rules.TextToVideoPerVideo, "text_to_video_legacy")
	appendLegacyPerVideoTiers(&out, rules.ImageToVideoPerVideo, "image_to_video_legacy")
	appendLegacyPerVideoTiers(&out, rules.VideoToVideoInputPerVideo, "video_to_video_input_legacy")
	appendLegacyPerVideoTiers(&out, rules.VideoToVideoOutputPerVideo, "video_to_video_output_legacy")
	return out
}

func collectVideoPerSecondTiers(rules ratio_setting.VideoPricingRules) []videoFlatTier {
	out := make([]videoFlatTier, 0, 24)
	appendPerItemTiers(&out, rules.TextToVideoPerSecond, "text_to_video_per_second")
	appendPerItemTiers(&out, rules.ImageToVideoPerSecond, "image_to_video_per_second")
	appendPerItemTiers(&out, rules.VideoToVideoPerSecond, "video_to_video_per_second")
	return out
}

func collectVideoPerTokenTiers(rules ratio_setting.VideoPricingRules) []videoFlatTier {
	out := make([]videoFlatTier, 0, 24)
	appendPerItemTiers(&out, rules.TextToVideoPerToken, "text_to_video_per_token")
	appendPerItemTiers(&out, rules.ImageToVideoPerToken, "image_to_video_per_token")
	appendPerItemTiers(&out, rules.VideoToVideoPerToken, "video_to_video_per_token")
	return out
}

func pickMinVideoFlatTier(tiers []videoFlatTier) (videoFlatTier, bool) {
	if len(tiers) == 0 {
		return videoFlatTier{}, false
	}
	best := 0
	for i := 1; i < len(tiers); i++ {
		a, b := tiers[best], tiers[i]
		if b.RawUSD < a.RawUSD-1e-12 {
			best = i
			continue
		}
		if math.Abs(b.RawUSD-a.RawUSD) < 1e-9 && tierLessVideoFlat(b, a) {
			best = i
		}
	}
	return tiers[best], true
}

func videoRulesUsableForPricingHint(rules ratio_setting.VideoPricingRules) bool {
	return ratio_setting.HasUsableVideoPerTokenRules(rules) ||
		ratio_setting.HasUsableVideoPerSecondRules(rules) ||
		ratio_setting.HasUsableVideoPerVideoRules(rules)
}

func resolveChannelVideoRulesForPricingCardHint(channelID int, modelName string) (ratio_setting.VideoPricingRules, bool) {
	if channelID > 0 {
		if rules, ok := ratio_setting.GetChannelVideoPricingRules(channelID, modelName); ok && videoRulesUsableForPricingHint(rules) {
			return rules, true
		}
	}
	return ratio_setting.VideoPricingRules{}, false
}

func resolveGlobalVideoRulesForPricingCardHint(modelName string) (ratio_setting.VideoPricingRules, bool) {
	if rules, ok := ratio_setting.GetVideoPricingRules(modelName); ok && videoRulesUsableForPricingHint(rules) {
		return rules, true
	}
	return ratio_setting.VideoPricingRules{}, false
}

func lookupVideoTierRawUSD(rules ratio_setting.VideoPricingRules, target videoFlatTier) float64 {
	if !videoRulesUsableForPricingHint(rules) {
		return 0
	}
	tiers := collectVideoPerTokenTiers(rules)
	if len(tiers) == 0 {
		tiers = collectVideoPerSecondTiers(rules)
	}
	if len(tiers) == 0 {
		tiers = collectVideoFlatTiers(rules)
	}
	for _, c := range tiers {
		if c.Lane != target.Lane {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(c.Res), strings.TrimSpace(target.Res)) {
			continue
		}
		if !videoTierAudioMatches(c.HasAudio, target.HasAudio) {
			continue
		}
		return c.RawUSD
	}
	return 0
}

func videoTierAudioMatches(a, b *bool) bool {
	if a == nil || b == nil {
		return true
	}
	return *a == *b
}

func effectiveVideoTierDisplayUSD(channelTier videoFlatTier, globalRules ratio_setting.VideoPricingRules, costDiscPercent, markupDiscPercent float64) float64 {
	globalRaw := lookupVideoTierRawUSD(globalRules, channelTier)
	return EffectiveRuleUnitPrice(channelTier.RawUSD, globalRaw, costDiscPercent, markupDiscPercent)
}

// VideoBillingModeToPerSecondLane 将计费模式映射为 VideoPricingRules 按秒档位 lane 名。
func VideoBillingModeToPerSecondLane(mode string) string {
	switch strings.TrimSpace(mode) {
	case "image_to_video":
		return "image_to_video_per_second"
	case "video_to_video":
		return "video_to_video_per_second"
	default:
		return "text_to_video_per_second"
	}
}

// VideoBillingModeToPerTokenLane 将计费模式映射为 VideoPricingRules 按 token 档位 lane 名。
func VideoBillingModeToPerTokenLane(mode string) string {
	switch strings.TrimSpace(mode) {
	case "image_to_video":
		return "image_to_video_per_token"
	case "video_to_video":
		return "video_to_video_per_token"
	default:
		return "text_to_video_per_token"
	}
}

// LookupGlobalVideoPerSecondUSD 按与渠道已匹配档位相同的 lane+分辨率+音轨，查全局 VideoPricingRules 原价。
// 与 BuildVideoFlatClipHint / 定价卡片展示一致；不得用成片实际像素去重匹全局档（否则会错档加价）。
func LookupGlobalVideoPerSecondUSD(modelName, billingMode, resolution string, hasAudio, unifiedAudio bool) float64 {
	globalRules, ok := resolveGlobalVideoRulesForPricingCardHint(modelName)
	if !ok {
		return 0
	}
	lane := VideoBillingModeToPerSecondLane(billingMode)
	var ha *bool
	if unifiedAudio {
		ha = nil
	} else {
		v := hasAudio
		ha = &v
	}
	return lookupVideoTierRawUSD(globalRules, videoFlatTier{
		Res:      strings.TrimSpace(resolution),
		Lane:     lane,
		HasAudio: ha,
	})
}

// LookupGlobalVideoPerTokenUSD 按与渠道已匹配档位相同的 lane+分辨率+音轨，查全局 per_token 原价。
func LookupGlobalVideoPerTokenUSD(modelName, billingMode, resolution string, hasAudio, unifiedAudio bool) float64 {
	globalRules, ok := ratio_setting.GetVideoPricingRules(modelName)
	if !ok || !ratio_setting.HasUsableVideoPerTokenRules(globalRules) {
		return 0
	}
	lane := VideoBillingModeToPerTokenLane(billingMode)
	var ha *bool
	if unifiedAudio {
		ha = nil
	} else {
		v := hasAudio
		ha = &v
	}
	tiers := collectVideoPerTokenTiers(globalRules)
	for _, c := range tiers {
		if c.Lane != lane {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(c.Res), strings.TrimSpace(resolution)) {
			continue
		}
		if !videoTierAudioMatches(c.HasAudio, ha) {
			continue
		}
		return c.RawUSD
	}
	return 0
}

func tierRowLess(a, b VideoFlatClipTierRow) bool {
	ar := strings.TrimSpace(strings.ToLower(a.Resolution))
	br := strings.TrimSpace(strings.ToLower(b.Resolution))
	if ar != br {
		return ar < br
	}
	if laneOrderVideoFlat(a.Lane) != laneOrderVideoFlat(b.Lane) {
		return laneOrderVideoFlat(a.Lane) < laneOrderVideoFlat(b.Lane)
	}
	return audioPtrRank(a.HasAudio) < audioPtrRank(b.HasAudio)
}

func buildSortedTierRows(tiers []videoFlatTier, globalRules ratio_setting.VideoPricingRules, costDiscPercent, markupDiscPercent float64) []VideoFlatClipTierRow {
	rows := make([]VideoFlatClipTierRow, 0, len(tiers))
	for _, ti := range tiers {
		officialUSD := lookupVideoTierRawUSD(globalRules, ti)
		usd := effectiveVideoTierDisplayUSD(ti, globalRules, costDiscPercent, markupDiscPercent)
		if usd <= 0 {
			continue
		}
		var ha *bool
		if ti.HasAudio != nil {
			v := *ti.HasAudio
			ha = &v
		}
		rows = append(rows, VideoFlatClipTierRow{
			UsdAfterChannelDiscount: usd,
			UsdChannelRaw:           ti.RawUSD,
			UsdOfficial:             officialUSD,
			Resolution:              strings.TrimSpace(ti.Res),
			HasAudio:                ha,
			Lane:                    ti.Lane,
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if math.Abs(a.UsdAfterChannelDiscount-b.UsdAfterChannelDiscount) > 1e-9 {
			return a.UsdAfterChannelDiscount < b.UsdAfterChannelDiscount
		}
		return tierRowLess(a, b)
	})
	return rows
}

func pickMinEffectiveVideoDisplayTier(tiers []videoFlatTier, globalRules ratio_setting.VideoPricingRules, costDiscPercent, markupDiscPercent float64) (videoFlatTier, float64, bool) {
	if len(tiers) == 0 {
		return videoFlatTier{}, 0, false
	}
	bestIdx := 0
	bestUsd := effectiveVideoTierDisplayUSD(tiers[0], globalRules, costDiscPercent, markupDiscPercent)
	for i := 1; i < len(tiers); i++ {
		u := effectiveVideoTierDisplayUSD(tiers[i], globalRules, costDiscPercent, markupDiscPercent)
		if u < bestUsd-1e-12 || (math.Abs(u-bestUsd) < 1e-9 && tierLessVideoFlat(tiers[i], tiers[bestIdx])) {
			bestIdx = i
			bestUsd = u
		}
	}
	if bestUsd <= 0 {
		return videoFlatTier{}, 0, false
	}
	return tiers[bestIdx], bestUsd, true
}

// BuildVideoFlatClipHint 汇总当前模型×渠道下视频分辨率档位（优先按 token，否则按秒，否则按条），
// 返回最低价档（含成本折扣与加价折扣）及总档位数。
func BuildVideoFlatClipHint(channelID int, modelName string, costDiscPercent, markupDiscPercent float64) *VideoFlatClipPricingHint {
	channelRules, chOK := resolveChannelVideoRulesForPricingCardHint(channelID, modelName)
	globalRules, glOK := resolveGlobalVideoRulesForPricingCardHint(modelName)
	if !chOK && !glOK {
		return nil
	}
	rulesForTiers := channelRules
	if !chOK {
		rulesForTiers = globalRules
	}
	tiers := collectVideoPerTokenTiers(rulesForTiers)
	billingMode := "per_token"
	if len(tiers) == 0 {
		tiers = collectVideoPerSecondTiers(rulesForTiers)
		billingMode = "per_second"
	}
	if len(tiers) == 0 {
		tiers = collectVideoFlatTiers(rulesForTiers)
		billingMode = "per_item"
	}
	if len(tiers) == 0 {
		return nil
	}
	tiers = collapsePairedUnifiedAudioTiers(tiers)
	tiers = normalizeLegacyAllFalseToUnifiedHintTiers(tiers)
	best, minUsd, ok := pickMinEffectiveVideoDisplayTier(tiers, globalRules, costDiscPercent, markupDiscPercent)
	if !ok {
		return nil
	}
	var hasAudioPtr *bool
	if best.HasAudio != nil {
		v := *best.HasAudio
		hasAudioPtr = &v
	}
	rows := buildSortedTierRows(tiers, globalRules, costDiscPercent, markupDiscPercent)
	return &VideoFlatClipPricingHint{
		MinUsdAfterChannelDiscount: minUsd,
		Resolution:                 strings.TrimSpace(best.Res),
		HasAudio:                   hasAudioPtr,
		Lane:                       best.Lane,
		TierCount:                  len(tiers),
		Tiers:                      rows,
		BillingMode:                billingMode,
	}
}
