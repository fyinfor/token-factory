package model

import (
	"math"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

// ImagePerImageTierRow 单档图片按张标价（已乘渠道展示折扣，未乘用户分组倍率）。
type ImagePerImageTierRow struct {
	UsdAfterChannelDiscount float64 `json:"usd_after_channel_discount"`
	UsdOfficial             float64 `json:"usd_official,omitempty"`
	Resolution              string  `json:"resolution,omitempty"`
	Lane                    string  `json:"lane,omitempty"`
}

// ImagePerImagePricingHint 多档图片分辨率价在定价卡片上的摘要。
type ImagePerImagePricingHint struct {
	MinUsdAfterChannelDiscount float64                `json:"min_usd_after_channel_discount"`
	Resolution                 string                 `json:"resolution,omitempty"`
	Lane                       string                 `json:"lane,omitempty"`
	TierCount                  int                    `json:"tier_count"`
	Tiers                      []ImagePerImageTierRow `json:"tiers,omitempty"`
}

type imagePerImageTier struct {
	RawUSD float64
	Res    string
	Lane   string
}

func laneOrderImagePerImage(l string) int {
	switch l {
	case "text_to_image":
		return 0
	case "image_to_image":
		return 1
	default:
		return 99
	}
}

func tierLessImagePerImage(a, b imagePerImageTier) bool {
	ar := strings.TrimSpace(strings.ToLower(a.Res))
	br := strings.TrimSpace(strings.ToLower(b.Res))
	if ar != br {
		return ar < br
	}
	return laneOrderImagePerImage(a.Lane) < laneOrderImagePerImage(b.Lane)
}

func collectImagePerImageTiers(rules ratio_setting.ImagePricingRules) []imagePerImageTier {
	out := make([]imagePerImageTier, 0, 16)
	for _, r := range rules.TextToImagePerImage {
		if r.ImagePrice <= 0 {
			continue
		}
		out = append(out, imagePerImageTier{
			RawUSD: r.ImagePrice,
			Res:    r.Resolution,
			Lane:   "text_to_image",
		})
	}
	for _, r := range rules.ImageToImagePerImage {
		if r.ImagePrice <= 0 {
			continue
		}
		out = append(out, imagePerImageTier{
			RawUSD: r.ImagePrice,
			Res:    r.Resolution,
			Lane:   "image_to_image",
		})
	}
	return out
}

func pickMinImagePerImageTier(tiers []imagePerImageTier) (imagePerImageTier, bool) {
	if len(tiers) == 0 {
		return imagePerImageTier{}, false
	}
	best := 0
	for i := 1; i < len(tiers); i++ {
		a, b := tiers[best], tiers[i]
		if b.RawUSD < a.RawUSD-1e-12 {
			best = i
			continue
		}
		if math.Abs(b.RawUSD-a.RawUSD) < 1e-9 && tierLessImagePerImage(b, a) {
			best = i
		}
	}
	return tiers[best], true
}

func imageTierRowLess(a, b ImagePerImageTierRow) bool {
	ar := strings.TrimSpace(strings.ToLower(a.Resolution))
	br := strings.TrimSpace(strings.ToLower(b.Resolution))
	if ar != br {
		return ar < br
	}
	return laneOrderImagePerImage(a.Lane) < laneOrderImagePerImage(b.Lane)
}

func buildSortedImagePerImageTierRows(tiers []imagePerImageTier, globalRules ratio_setting.ImagePricingRules, costDiscPercent, markupDiscPercent float64) []ImagePerImageTierRow {
	rows := make([]ImagePerImageTierRow, 0, len(tiers))
	for _, ti := range tiers {
		globalRaw := lookupImageTierRawUSD(globalRules, ti)
		usd := EffectiveRuleUnitPrice(ti.RawUSD, globalRaw, costDiscPercent, markupDiscPercent)
		if usd <= 0 {
			continue
		}
		rows = append(rows, ImagePerImageTierRow{
			UsdAfterChannelDiscount: usd,
			UsdOfficial:             globalRaw,
			Resolution:              strings.TrimSpace(ti.Res),
			Lane:                    ti.Lane,
		})
	}
	sort.Slice(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if math.Abs(a.UsdAfterChannelDiscount-b.UsdAfterChannelDiscount) > 1e-9 {
			return a.UsdAfterChannelDiscount < b.UsdAfterChannelDiscount
		}
		return imageTierRowLess(a, b)
	})
	return rows
}

func resolveChannelImageRulesForPricingCardHint(channelID int, modelName string) (ratio_setting.ImagePricingRules, bool) {
	if channelID > 0 {
		if rules, ok := ratio_setting.GetChannelImagePricingRules(channelID, modelName); ok && ratio_setting.HasUsableImagePerImageRules(rules) {
			return rules, true
		}
	}
	return ratio_setting.ImagePricingRules{}, false
}

func resolveGlobalImageRulesForPricingCardHint(modelName string) (ratio_setting.ImagePricingRules, bool) {
	if rules, ok := ratio_setting.GetImagePricingRules(modelName); ok && ratio_setting.HasUsableImagePerImageRules(rules) {
		return rules, true
	}
	return ratio_setting.ImagePricingRules{}, false
}

func lookupImageTierRawUSD(rules ratio_setting.ImagePricingRules, target imagePerImageTier) float64 {
	for _, c := range collectImagePerImageTiers(rules) {
		if c.Lane != target.Lane {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(c.Res), strings.TrimSpace(target.Res)) {
			continue
		}
		return c.RawUSD
	}
	return 0
}

// BuildImagePerImageHint 汇总当前模型×渠道下图片按张档位，返回最低价档（含成本折扣与加价折扣）及全部档位。
func BuildImagePerImageHint(channelID int, modelName string, costDiscPercent, markupDiscPercent float64) *ImagePerImagePricingHint {
	channelRules, chOK := resolveChannelImageRulesForPricingCardHint(channelID, modelName)
	globalRules, glOK := resolveGlobalImageRulesForPricingCardHint(modelName)
	if !chOK && !glOK {
		return nil
	}
	rulesForTiers := channelRules
	if !chOK {
		rulesForTiers = globalRules
	}
	tiers := collectImagePerImageTiers(rulesForTiers)
	if len(tiers) == 0 {
		return nil
	}
	rows := buildSortedImagePerImageTierRows(tiers, globalRules, costDiscPercent, markupDiscPercent)
	if len(rows) == 0 {
		return nil
	}
	bestRow := rows[0]
	return &ImagePerImagePricingHint{
		MinUsdAfterChannelDiscount: bestRow.UsdAfterChannelDiscount,
		Resolution:                 bestRow.Resolution,
		Lane:                       bestRow.Lane,
		TierCount:                  len(tiers),
		Tiers:                      rows,
	}
}
