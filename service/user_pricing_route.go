package service

import (
	"errors"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/fyinfor/router-engine/pkg/router"
)

// ── 用户指定价 × 选路约束 ────────────────────────────────────
//
// 命中用户指定价时，用户最终支付价 = 全局官方价 × 总折扣（固定，与渠道无关）。
//
// Mode=price_cap：排除「渠道有效单价 > 用户指定价上限」的渠道。
// Mode=channel_list：仅允许管理员勾选的渠道；智能路由路径在勾选集内按智能规则选，
// 非智能路由路径（SelectChannelLocal Fallback）按手动 priority 升序。

// 浮点比较容差，避免同价渠道因精度误差被排除。
const userPriceCapEpsilon = 1e-9

// UserPricingRouteConstraint 用户指定价对某模型的选路约束。
type UserPricingRouteConstraint struct {
	Mode     string         // price_cap | channel_list
	Cap      float64        // price_cap 时有效
	CapOK    bool           // price_cap 且全局官方价可定义
	AllowSet map[int]int    // channel_list: channel_id → priority
	Order    []int          // channel_list: 按 priority 升序的 channel_id
}

// ResolveUserPricingRouteConstraint 解析用户×模型选路约束；无启用指定价时 ok=false。
func ResolveUserPricingRouteConstraint(userID int, modelName string) (UserPricingRouteConstraint, bool) {
	if userID <= 0 || modelName == "" {
		return UserPricingRouteConstraint{}, false
	}
	ov, ok := model.GetEnabledUserModelPricingOverride(userID, modelName)
	if !ok {
		return UserPricingRouteConstraint{}, false
	}
	mode := ov.NormalizedMode()
	c := UserPricingRouteConstraint{Mode: mode}
	if mode == model.UserPricingModeChannelList {
		allow, _ := model.GetEnabledUserModelPricingChannelAllowSet(userID, modelName)
		if allow == nil {
			allow = map[int]int{}
		}
		c.AllowSet = allow
		order := make([]int, 0, len(allow))
		for id := range allow {
			order = append(order, id)
		}
		sort.Slice(order, func(i, j int) bool {
			pi, pj := allow[order[i]], allow[order[j]]
			if pi != pj {
				return pi < pj
			}
			return order[i] < order[j]
		})
		c.Order = order
		return c, true
	}
	cap, capOK := UnitPriceCapForTotalPercent(modelName, ov.TotalPercent())
	c.Cap = cap
	c.CapOK = capOK
	return c, true
}

// UserModelUnitPriceCap 返回用户对该模型的路由单价上限（仅 price_cap）。
// 无指定价、channel_list、或全局官方价未配置时返回 false。
func UserModelUnitPriceCap(userID int, modelName string) (float64, bool) {
	c, ok := ResolveUserPricingRouteConstraint(userID, modelName)
	if !ok || c.Mode != model.UserPricingModePriceCap || !c.CapOK {
		return 0, false
	}
	return c.Cap, true
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

// ChannelWithinUserPriceCap 判断某渠道对该用户是否可用（兼容旧名）。
// price_cap：单价不超上限；channel_list：渠道在勾选集内；无指定价：恒 true。
func ChannelWithinUserPriceCap(userID int, modelName string, ch *model.Channel) bool {
	return ChannelAllowedForUserPricing(userID, modelName, ch)
}

// ChannelAllowedForUserPricing 判断某渠道对该用户×模型是否允许调用。
func ChannelAllowedForUserPricing(userID int, modelName string, ch *model.Channel) bool {
	if ch == nil {
		return true
	}
	c, ok := ResolveUserPricingRouteConstraint(userID, modelName)
	if !ok {
		return true
	}
	if c.Mode == model.UserPricingModeChannelList {
		_, allowed := c.AllowSet[ch.Id]
		return allowed
	}
	if !c.CapOK {
		return true
	}
	return unitPriceWithinCap(ResolveChannelModelUnitPrice(ch, modelName), c.Cap)
}

// FilterRouteCandidatesByUserPriceCap 按用户指定价过滤归类路由候选。
// channel_list 仅过滤不改序（智能路由自行排序）；调用方可在 Fallback 时再按手动序排。
func FilterRouteCandidatesByUserPriceCap(userID int, modelName string, candidates []RouteChannelCandidate) []RouteChannelCandidate {
	c, ok := ResolveUserPricingRouteConstraint(userID, modelName)
	if !ok {
		return candidates
	}
	out := make([]RouteChannelCandidate, 0, len(candidates))
	if c.Mode == model.UserPricingModeChannelList {
		for _, cand := range candidates {
			if _, allowed := c.AllowSet[cand.ChannelID]; allowed {
				out = append(out, cand)
			}
		}
		return out
	}
	if !c.CapOK {
		return candidates
	}
	for _, cand := range candidates {
		if unitPriceWithinCap(cand.Price, c.Cap) {
			out = append(out, cand)
		}
	}
	return out
}

// SortRouteCandidatesByUserPricingPriority 在 channel_list 模式下按手动 priority 升序；
// 其它模式原样返回。未出现在 allowset 的渠道排到末尾。
func SortRouteCandidatesByUserPricingPriority(userID int, modelName string, candidates []RouteChannelCandidate) []RouteChannelCandidate {
	c, ok := ResolveUserPricingRouteConstraint(userID, modelName)
	if !ok || c.Mode != model.UserPricingModeChannelList || len(candidates) <= 1 {
		return candidates
	}
	out := make([]RouteChannelCandidate, len(candidates))
	copy(out, candidates)
	sort.SliceStable(out, func(i, j int) bool {
		pi, oi := c.AllowSet[out[i].ChannelID]
		pj, oj := c.AllowSet[out[j].ChannelID]
		if oi != oj {
			return oi
		}
		if !oi {
			return out[i].ChannelID < out[j].ChannelID
		}
		if pi != pj {
			return pi < pj
		}
		return out[i].ChannelID < out[j].ChannelID
	})
	return out
}

// UserPricingUsesManualChannelPriority 是否应对「非智能路由」路径启用手动渠道路径优先级。
func UserPricingUsesManualChannelPriority(userID int, modelName string) bool {
	c, ok := ResolveUserPricingRouteConstraint(userID, modelName)
	return ok && c.Mode == model.UserPricingModeChannelList && len(c.Order) > 0
}

// filterEndpointCandidatesByUserPriceCap 过滤 SmartRouter 候选（UnitPrice 已是单价信号）。
func filterEndpointCandidatesByUserPriceCap(userID int, modelName string, cands []*router.EndpointCandidate) []*router.EndpointCandidate {
	c, ok := ResolveUserPricingRouteConstraint(userID, modelName)
	if !ok {
		return cands
	}
	out := make([]*router.EndpointCandidate, 0, len(cands))
	if c.Mode == model.UserPricingModeChannelList {
		for _, cand := range cands {
			if cand == nil {
				continue
			}
			if _, allowed := c.AllowSet[cand.ChannelID]; allowed {
				out = append(out, cand)
			}
		}
		return out
	}
	if !c.CapOK {
		return cands
	}
	for _, cand := range cands {
		if cand == nil {
			continue
		}
		if unitPriceWithinCap(cand.UnitPrice, c.Cap) {
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
// 跳过找不到已配置单价渠道的模型。导入一律为 price_cap 模式。
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
			Mode:                 model.UserPricingModePriceCap,
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

// UserModelPricingConvertItem 一键切到渠道清单的单模型结果。
type UserModelPricingConvertItem struct {
	ModelName     string  `json:"model_name"`
	ChannelCount  int     `json:"channel_count"`
	Skipped       bool    `json:"skipped"`
	SkipReason    string  `json:"skip_reason,omitempty"`
	PreviousMode  string  `json:"previous_mode"`
}

// BuildWithinCapChannelBindings 按单价升序勾选「有效单价 ≤ 指定售价」的启用渠道。
// 上限无法定义时返回全部已配置单价的启用渠道（按单价升序）；无可用渠道时返回空。
func BuildWithinCapChannelBindings(modelName string, totalPercent float64) []model.UserModelPricingChannelBinding {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return nil
	}
	cap, capOK := UnitPriceCapForTotalPercent(modelName, totalPercent)
	type priced struct {
		id    int
		price float64
	}
	pricedList := make([]priced, 0)
	for _, id := range model.GetEnabledChannelIDsByModel(modelName) {
		ch, err := model.CacheGetChannel(id)
		if err != nil || ch == nil || ch.Status != common.ChannelStatusEnabled {
			continue
		}
		price := ResolveChannelModelUnitPrice(ch, modelName)
		if price <= 0 {
			continue
		}
		if capOK && !unitPriceWithinCap(price, cap) {
			continue
		}
		pricedList = append(pricedList, priced{id: id, price: price})
	}
	sort.SliceStable(pricedList, func(i, j int) bool {
		if pricedList[i].price != pricedList[j].price {
			return pricedList[i].price < pricedList[j].price
		}
		return pricedList[i].id < pricedList[j].id
	})
	out := make([]model.UserModelPricingChannelBinding, 0, len(pricedList))
	for i, p := range pricedList {
		out = append(out, model.UserModelPricingChannelBinding{
			ChannelId: p.id,
			Priority:  i + 1,
		})
	}
	return out
}

// ConvertUserModelPricingToChannelList 将指定价改为 channel_list（兼容旧调用）。
// modelNames 为空时处理该用户全部配置；非空时仅处理名单内模型。
func ConvertUserModelPricingToChannelList(userID int, modelNames []string) (converted, skipped int, items []UserModelPricingConvertItem, err error) {
	return ConvertUserModelPricingMode(userID, modelNames, model.UserPricingModeChannelList)
}

// ConvertUserModelPricingMode 批量切换选路模式。
//   - targetMode=channel_list：每模型勾选未超指定售价渠道，按单价升序；无可用渠道则跳过。
//   - targetMode=price_cap：改回价格上限并清空渠道清单。
// modelNames 为空 / 不传时默认处理该用户全部配置（「默认不选 = 全切」）。
func ConvertUserModelPricingMode(userID int, modelNames []string, targetMode string) (converted, skipped int, items []UserModelPricingConvertItem, err error) {
	if userID <= 0 {
		return 0, 0, nil, errors.New("invalid user_id")
	}
	switch strings.TrimSpace(targetMode) {
	case "", model.UserPricingModeChannelList:
		targetMode = model.UserPricingModeChannelList
	case model.UserPricingModePriceCap:
		targetMode = model.UserPricingModePriceCap
	default:
		return 0, 0, nil, errors.New("target_mode 须为 price_cap 或 channel_list")
	}
	rows, err := model.ListUserModelPricingOverrides(userID, "")
	if err != nil {
		return 0, 0, nil, err
	}
	filter := map[string]struct{}{}
	for _, n := range modelNames {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		filter[n] = struct{}{}
	}
	useFilter := len(filter) > 0

	items = make([]UserModelPricingConvertItem, 0, len(rows))
	touched := false
	for _, row := range rows {
		if useFilter {
			if _, ok := filter[row.ModelName]; !ok {
				continue
			}
		}
		prevMode := row.NormalizedMode()
		ov := row

		if targetMode == model.UserPricingModePriceCap {
			ov.Mode = model.UserPricingModePriceCap
			if _, upsertErr := model.UpsertUserModelPricingOverrideWithChannelsOpt(&ov, nil, false); upsertErr != nil {
				skipped++
				items = append(items, UserModelPricingConvertItem{
					ModelName:    row.ModelName,
					Skipped:      true,
					SkipReason:   upsertErr.Error(),
					PreviousMode: prevMode,
				})
				continue
			}
			touched = true
			converted++
			items = append(items, UserModelPricingConvertItem{
				ModelName:    row.ModelName,
				ChannelCount: 0,
				PreviousMode: prevMode,
			})
			continue
		}

		bindings := BuildWithinCapChannelBindings(row.ModelName, row.TotalPercent())
		if len(bindings) == 0 {
			skipped++
			items = append(items, UserModelPricingConvertItem{
				ModelName:    row.ModelName,
				Skipped:      true,
				SkipReason:   "无未超价启用渠道可勾选",
				PreviousMode: prevMode,
			})
			continue
		}
		ov.Mode = model.UserPricingModeChannelList
		if _, upsertErr := model.UpsertUserModelPricingOverrideWithChannelsOpt(&ov, bindings, false); upsertErr != nil {
			skipped++
			items = append(items, UserModelPricingConvertItem{
				ModelName:    row.ModelName,
				Skipped:      true,
				SkipReason:   upsertErr.Error(),
				PreviousMode: prevMode,
			})
			continue
		}
		touched = true
		converted++
		items = append(items, UserModelPricingConvertItem{
			ModelName:    row.ModelName,
			ChannelCount: len(bindings),
			PreviousMode: prevMode,
		})
	}
	if touched {
		model.InvalidateUserModelPricingCache()
	}
	if useFilter && converted == 0 && skipped == 0 {
		return 0, 0, items, errors.New("所选模型均不在该用户指定价配置中")
	}
	return converted, skipped, items, nil
}
