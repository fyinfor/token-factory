package service

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/fyinfor/router-engine/pkg/router"
)

// ── 用户指定价 × 选路：价格上限过滤 ────────────────────────────
//
// 命中用户指定价时，用户最终支付价 = 全局官方价 × 总折扣（固定，与渠道无关）。
// 选路层必须排除「渠道有效单价 > 用户指定价上限」的渠道，避免平台亏本服务。
// 上限与 ResolveChannelModelUnitPrice 同口径（固定价模型比固定价，倍率模型比倍率）。

// 浮点比较容差，避免同价渠道因精度误差被排除。
const userPriceCapEpsilon = 1e-9

// UserModelUnitPriceCap 返回用户对该模型的路由单价上限。
// 无指定价配置、或全局官方价未配置（上限无法定义）时返回 false（不限制）。
func UserModelUnitPriceCap(userID int, modelName string) (float64, bool) {
	if userID <= 0 || modelName == "" {
		return 0, false
	}
	ov, ok := model.GetEnabledUserModelPricingOverride(userID, modelName)
	if !ok {
		return 0, false
	}
	return UnitPriceCapForTotalPercent(modelName, ov.TotalPercent())
}

// UnitPriceCapForTotalPercent 按「全局官方价 × 总折扣%」计算路由单价上限，
// 与 ResolveChannelModelUnitPrice 的口径对齐：全局固定价优先，其次全局倍率。
func UnitPriceCapForTotalPercent(modelName string, totalPercent float64) (float64, bool) {
	total := totalPercent / 100.0
	if globalPrice, ok := ratio_setting.GetModelPrice(modelName, false); ok && globalPrice > 0 {
		return globalPrice * total, true
	}
	if globalRatio, _, _ := ratio_setting.GetModelRatio(modelName); globalRatio > 0 {
		return globalRatio * total, true
	}
	return 0, false
}

func unitPriceWithinCap(price, cap float64) bool {
	return price <= cap*(1+userPriceCapEpsilon)
}

// ChannelWithinUserPriceCap 判断某渠道对该用户是否可用（价格不超上限）。
// 用户无指定价时恒为 true。
func ChannelWithinUserPriceCap(userID int, modelName string, ch *model.Channel) bool {
	if ch == nil {
		return true
	}
	cap, ok := UserModelUnitPriceCap(userID, modelName)
	if !ok {
		return true
	}
	return unitPriceWithinCap(ResolveChannelModelUnitPrice(ch, modelName), cap)
}

// FilterRouteCandidatesByUserPriceCap 过滤归类路由候选（Price 字段已是单价信号）。
// 无指定价时原样返回。
func FilterRouteCandidatesByUserPriceCap(userID int, modelName string, candidates []RouteChannelCandidate) []RouteChannelCandidate {
	cap, ok := UserModelUnitPriceCap(userID, modelName)
	if !ok {
		return candidates
	}
	out := make([]RouteChannelCandidate, 0, len(candidates))
	for _, cand := range candidates {
		if unitPriceWithinCap(cand.Price, cap) {
			out = append(out, cand)
		}
	}
	return out
}

// filterEndpointCandidatesByUserPriceCap 过滤 SmartRouter 候选（UnitPrice 已是单价信号）。
func filterEndpointCandidatesByUserPriceCap(userID int, modelName string, cands []*router.EndpointCandidate) []*router.EndpointCandidate {
	cap, ok := UserModelUnitPriceCap(userID, modelName)
	if !ok {
		return cands
	}
	out := make([]*router.EndpointCandidate, 0, len(cands))
	for _, cand := range cands {
		if unitPriceWithinCap(cand.UnitPrice, cap) {
			out = append(out, cand)
		}
	}
	return out
}

// UserModelPricingImportPreviewItem 一键导入预览：每个模型抄自最便宜渠道的当前三项折扣。
type UserModelPricingImportPreviewItem struct {
	ModelName            string  `json:"model_name"`
	ChannelId            int     `json:"channel_id"`
	ChannelName          string  `json:"channel_name"`
	UnitPrice            float64 `json:"unit_price"`
	PriceDiscountPercent float64 `json:"price_discount_percent"`
	OperatingCostPercent float64 `json:"operating_cost_percent"`
	MarkupDiscountRate   float64 `json:"markup_discount_rate"`
	TotalPercent         float64 `json:"total_percent"`
}

// BuildUserModelPricingImportFromCheapestChannels 为用户构建「从当前最便宜渠道抄折扣」的导入行。
// 跳过找不到已配置单价渠道的模型。
func BuildUserModelPricingImportFromCheapestChannels(userID int, enabled bool) (rows []model.UserModelPricingOverride, preview []UserModelPricingImportPreviewItem) {
	if userID <= 0 {
		return nil, nil
	}
	models := model.ListImportablePricedModels()
	rows = make([]model.UserModelPricingOverride, 0, len(models))
	preview = make([]UserModelPricingImportPreviewItem, 0, len(models))
	for _, name := range models {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		ch, price, ok := findCheapestEnabledChannelForModel(name)
		if !ok || ch == nil {
			continue
		}
		raw := ch.ResolvedPriceDiscountPercent()
		operating := ch.ResolvedOperatingCostPercent()
		markup := ch.ResolvedMarkupDiscountRate()
		rows = append(rows, model.UserModelPricingOverride{
			UserId:               userID,
			ModelName:            name,
			PriceDiscountPercent: raw,
			OperatingCostPercent: operating,
			MarkupDiscountRate:   markup,
			Enabled:              enabled,
		})
		preview = append(preview, UserModelPricingImportPreviewItem{
			ModelName:            name,
			ChannelId:            ch.Id,
			ChannelName:          ch.Name,
			UnitPrice:            price,
			PriceDiscountPercent: raw,
			OperatingCostPercent: operating,
			MarkupDiscountRate:   markup,
			TotalPercent:         raw + operating + markup,
		})
	}
	return rows, preview
}

func findCheapestEnabledChannelForModel(modelName string) (*model.Channel, float64, bool) {
	ids := model.GetEnabledChannelIDsByModel(modelName)
	var best *model.Channel
	var bestPrice float64
	found := false
	for _, id := range ids {
		ch, err := model.CacheGetChannel(id)
		if err != nil || ch == nil || ch.Status != common.ChannelStatusEnabled {
			continue
		}
		price, ok := ResolveChannelModelConfiguredUnitPrice(ch, modelName)
		if !ok || price <= 0 {
			continue
		}
		if !found || price < bestPrice {
			best = ch
			bestPrice = price
			found = true
		}
	}
	return best, bestPrice, found
}
