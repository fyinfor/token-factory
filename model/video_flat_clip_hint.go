package model

import (
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

// VideoFlatClipTierRow 单档「按条成片」标价（已乘渠道展示折扣，未乘用户分组倍率）。
type VideoFlatClipTierRow struct {
	UsdAfterChannelDiscount float64 `json:"usd_after_channel_discount"`
	Resolution              string  `json:"resolution,omitempty"`
	HasAudio                *bool   `json:"has_audio,omitempty"`
	Lane                    string  `json:"lane,omitempty"`
}

// VideoFlatClipPricingHint 多档视频分辨率价在定价卡片上的摘要（按条或按秒，见 BillingMode）：
// MinUsdAfterChannelDiscount = min(规则标价)×渠道展示折扣系数（与 channel_list 中倍率口径一致），
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
	return ratio_setting.HasUsableVideoPerVideoRules(rules) ||
		ratio_setting.HasUsableVideoPerSecondRules(rules)
}

// resolveVideoRulesForPricingCardHint 拉取渠道或全局视频计价规则（按条或按秒任一存在即可）。
func resolveVideoRulesForPricingCardHint(channelID int, modelName string) (ratio_setting.VideoPricingRules, bool) {
	if channelID > 0 {
		if rules, ok := ratio_setting.GetChannelVideoPricingRules(channelID, modelName); ok && videoRulesUsableForPricingHint(rules) {
			return rules, true
		}
	}
	if rules, ok := ratio_setting.GetVideoPricingRules(modelName); ok && videoRulesUsableForPricingHint(rules) {
		return rules, true
	}
	return ratio_setting.VideoPricingRules{}, false
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

func buildSortedTierRows(tiers []videoFlatTier, discountMult float64) []VideoFlatClipTierRow {
	rows := make([]VideoFlatClipTierRow, 0, len(tiers))
	for _, ti := range tiers {
		usd := ti.RawUSD * discountMult
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

// BuildVideoFlatClipHint 汇总当前模型×渠道下视频分辨率档位（优先按条，否则按秒），返回最低价档（已乘渠道展示折扣）及总档位数。
func BuildVideoFlatClipHint(channelID int, modelName string, discountMult float64) *VideoFlatClipPricingHint {
	rules, ok := resolveVideoRulesForPricingCardHint(channelID, modelName)
	if !ok {
		return nil
	}
	tiers := collectVideoFlatTiers(rules)
	billingMode := "per_item"
	if len(tiers) == 0 {
		tiers = collectVideoPerSecondTiers(rules)
		billingMode = "per_second"
	}
	if len(tiers) == 0 {
		return nil
	}
	tiers = collapsePairedUnifiedAudioTiers(tiers)
	tiers = normalizeLegacyAllFalseToUnifiedHintTiers(tiers)
	best, ok := pickMinVideoFlatTier(tiers)
	if !ok || best.RawUSD <= 0 {
		return nil
	}
	var hasAudioPtr *bool
	if best.HasAudio != nil {
		v := *best.HasAudio
		hasAudioPtr = &v
	}
	rows := buildSortedTierRows(tiers, discountMult)
	return &VideoFlatClipPricingHint{
		MinUsdAfterChannelDiscount: best.RawUSD * discountMult,
		Resolution:                 strings.TrimSpace(best.Res),
		HasAudio:                   hasAudioPtr,
		Lane:                       best.Lane,
		TierCount:                  len(tiers),
		Tiers:                      rows,
		BillingMode:                billingMode,
	}
}
