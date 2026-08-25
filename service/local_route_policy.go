package service

import (
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

// LocalUserRoutePolicy 用户路由策略完整视图（本地实现，供控制台 API）。
type LocalUserRoutePolicy struct {
	Mode            string
	GlobalMode      string
	Groups          []LocalUserModelGroup
	UserOverrides   []LocalUserOverrideItem
	GlobalOverrides []LocalUserOverrideItem
}

// LocalUserModelGroup 用户视图中的模型分组。
type LocalUserModelGroup struct {
	GroupKey      string
	DisplayName   string
	Models        []string
	ChannelCount  int
	Channels      []LocalUserGroupChannel
	RouteDisabled bool // 用户对该归类关闭智能路由
}

// LocalUserGroupChannel 用户视图中的渠道信息。
type LocalUserGroupChannel struct {
	ChannelID        int
	RouteSlug        string
	Name             string
	MaskedName       string
	ProviderSlug     string
	SupplierAlias    string
	Status           int
	ModelsInGroup    []string
	UserWeight       int
	UserWeightID     uint
	UserEnabled      bool
	UserConfigured   bool
	GlobalWeight     int
	GlobalEnabled    bool
	GlobalConfigured bool
	Price            float64
	UserDiscount     float64 // 当前用户相对官方价的优惠百分点（归类内模型取最优）
}

// LocalUserOverrideItem 归类覆盖项。
type LocalUserOverrideItem struct {
	ID       uint
	RawModel string
	GroupKey string
	IsUser   bool
}

// GetLocalUserRoutePolicy 基于本地 channels 表构建用户路由策略视图。
func GetLocalUserRoutePolicy(userID int, isAdmin bool) (*LocalUserRoutePolicy, error) {
	mode := ""
	if userCfg := model.GetUserRouteConfig(userID); userCfg != nil {
		mode = userCfg.Mode
	}
	globalCfg := model.GetRouteConfig()

	userOverrides, _ := model.LoadAllUserModelGroupOverrides(userID)
	globalOverrideMap, _ := model.LoadModelGroupOverrides()
	userOverrideMap, _ := model.LoadUserModelGroupOverrides(userID)

	allUserWeights, _ := model.LoadAllUserModelGroupWeights(userID)
	userWeightMap := make(map[string]map[int]model.UserModelGroupWeight)
	for _, w := range allUserWeights {
		if userWeightMap[w.GroupKey] == nil {
			userWeightMap[w.GroupKey] = make(map[int]model.UserModelGroupWeight)
		}
		userWeightMap[w.GroupKey][w.ChannelID] = w
	}

	routeDisabledMap, _ := model.LoadUserModelGroupRouteDisabledMap(userID)

	globalWeights, _ := model.LoadAllModelGroupWeights()
	globalWeightMap := make(map[string]map[int]model.ModelGroupWeight)
	for _, w := range globalWeights {
		if globalWeightMap[w.GroupKey] == nil {
			globalWeightMap[w.GroupKey] = make(map[int]model.ModelGroupWeight)
		}
		globalWeightMap[w.GroupKey][w.ChannelID] = w
	}

	var metas []model.ModelGroupMeta
	_ = model.DB.Find(&metas).Error
	displayMap := make(map[string]string, len(metas))
	for _, m := range metas {
		if m.DisplayName != "" {
			displayMap[m.GroupKey] = m.DisplayName
		}
	}

	channels, err := model.GetAllChannels(0, 0, true, true)
	if err != nil {
		return nil, err
	}

	groupRatio := 1.0
	if userID > 0 {
		if g, gErr := model.GetUserGroup(userID, false); gErr == nil && strings.TrimSpace(g) != "" {
			groupRatio = ratio_setting.GetGroupRatio(g)
			if groupRatio <= 0 {
				groupRatio = 1
			}
		}
	}

	aliasByAppID := loadSupplierAliasMap(channels)

	testIndex, testErr := model.LoadChannelPricingTestSuccessIndex()
	if testErr != nil || testIndex == nil {
		testIndex = map[int][]string{}
	}

	groupModels := make(map[string]map[string]bool)
	groupChannels := make(map[string]map[int]*LocalUserGroupChannel)

	for _, ch := range channels {
		if ch == nil {
			continue
		}
		// 已关闭渠道用户无法调用，不进入智能路由展示（与选路/预览口径一致）
		if ch.Status != common.ChannelStatusEnabled {
			continue
		}
		rawModels := ch.GetModels()
		if len(rawModels) == 0 {
			continue
		}
		supplierAlias := aliasByAppID[ch.SupplierApplicationID]
		providerSlug := strings.ToLower(strings.TrimSpace(ch.SupplierType))

		for _, m := range rawModels {
			m = strings.TrimSpace(m)
			if m == "" {
				continue
			}
			// 用户指定价：price_cap / channel_list 下不可用渠道不进入智能路由 UI。
			if userID > 0 && !ChannelAllowedForUserPricing(userID, m, ch) {
				continue
			}
			if !model.ChannelModelHasPassedConnectivityTest(testIndex, ch.Id, m) {
				continue
			}
			key := model.ResolveModelGroupKey(m, globalOverrideMap)
			if uov, ok := userOverrideMap[strings.ToLower(m)]; ok && uov != "" {
				key = uov
			}
			if key == "" {
				continue
			}
			if groupModels[key] == nil {
				groupModels[key] = make(map[string]bool)
			}
			groupModels[key][m] = true

			if groupChannels[key] == nil {
				groupChannels[key] = make(map[int]*LocalUserGroupChannel)
			}
			entry := groupChannels[key][ch.Id]
			if entry == nil {
				fullName := sanitizeUTF8(ch.Name)
				entry = &LocalUserGroupChannel{
					ChannelID:     ch.Id,
					RouteSlug:     sanitizeUTF8(ch.RouteSlug),
					Name:          fullName,
					MaskedName:    maskChannelName(fullName),
					ProviderSlug:  sanitizeUTF8(providerSlug),
					SupplierAlias: sanitizeUTF8(supplierAlias),
					Status:        ch.Status,
				}
				groupChannels[key][ch.Id] = entry
			}
			entry.ModelsInGroup = append(entry.ModelsInGroup, sanitizeUTF8(m))
			price := ResolveChannelModelConfiguredUnitPriceOrZero(ch, m)
			if price > 0 && (entry.Price <= 0 || price < entry.Price) {
				entry.Price = price
			}
			if disc := resolveUserChannelModelDiscount(userID, ch, m, groupRatio); disc > entry.UserDiscount {
				entry.UserDiscount = disc
			}
		}
	}

	groups := make([]LocalUserModelGroup, 0, len(groupModels))
	for key, modelSet := range groupModels {
		rawModels := make([]string, 0, len(modelSet))
		for m := range modelSet {
			rawModels = append(rawModels, sanitizeUTF8(m))
		}
		sort.Strings(rawModels)

		chans := make([]LocalUserGroupChannel, 0, len(groupChannels[key]))
		for _, entry := range groupChannels[key] {
			sort.Strings(entry.ModelsInGroup)
			if uwMap, ok := userWeightMap[key]; ok {
				if uw, ok2 := uwMap[entry.ChannelID]; ok2 {
					entry.UserWeight = uw.Weight
					entry.UserWeightID = uw.ID
					entry.UserEnabled = uw.Enabled
					entry.UserConfigured = true
				}
			}
			if gwMap, ok := globalWeightMap[key]; ok {
				if gw, ok2 := gwMap[entry.ChannelID]; ok2 {
					entry.GlobalWeight = gw.Weight
					entry.GlobalEnabled = gw.Enabled
					entry.GlobalConfigured = true
				}
			}
			if !isAdmin {
				entry.Name = entry.MaskedName
			}
			chans = append(chans, *entry)
		}

		sort.Slice(chans, func(i, j int) bool {
			ci, cj := chans[i], chans[j]
			if ci.UserConfigured != cj.UserConfigured {
				return ci.UserConfigured
			}
			ei := localEffectiveEnabled(ci)
			ej := localEffectiveEnabled(cj)
			if ei != ej {
				return ei
			}
			wi := localEffectiveWeight(ci)
			wj := localEffectiveWeight(cj)
			if wi != wj {
				return wi > wj
			}
			return ci.ChannelID < cj.ChannelID
		})

		// 跳过无可用渠道的分组（用户指定价过滤后可能为空）
		if len(chans) == 0 {
			continue
		}

		display := sanitizeUTF8(displayMap[key])
		if display == "" {
			display = sanitizeUTF8(key)
		}
		groups = append(groups, LocalUserModelGroup{
			GroupKey:      sanitizeUTF8(key),
			DisplayName:   display,
			Models:        rawModels,
			ChannelCount:  len(chans),
			Channels:      chans,
			RouteDisabled: routeDisabledMap[key],
		})
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].GroupKey < groups[j].GroupKey })

	userOverrideItems := make([]LocalUserOverrideItem, 0, len(userOverrides))
	for _, o := range userOverrides {
		userOverrideItems = append(userOverrideItems, LocalUserOverrideItem{
			ID: o.ID, RawModel: o.RawModel, GroupKey: o.GroupKey, IsUser: true,
		})
	}
	globalRows, _ := model.LoadAllModelGroupOverrides()
	globalOverrideItems := make([]LocalUserOverrideItem, 0, len(globalRows))
	for _, o := range globalRows {
		globalOverrideItems = append(globalOverrideItems, LocalUserOverrideItem{
			ID: o.ID, RawModel: o.RawModel, GroupKey: o.GroupKey, IsUser: false,
		})
	}

	return &LocalUserRoutePolicy{
		Mode:            mode,
		GlobalMode:      globalCfg.Mode,
		Groups:          groups,
		UserOverrides:   userOverrideItems,
		GlobalOverrides: globalOverrideItems,
	}, nil
}

func loadSupplierAliasMap(channels []*model.Channel) map[int]string {
	ids := make([]int, 0)
	seen := make(map[int]bool)
	for _, ch := range channels {
		if ch == nil || ch.SupplierApplicationID <= 0 || seen[ch.SupplierApplicationID] {
			continue
		}
		seen[ch.SupplierApplicationID] = true
		ids = append(ids, ch.SupplierApplicationID)
	}
	out := make(map[int]string)
	if len(ids) == 0 {
		return out
	}
	var apps []model.SupplierApplication
	_ = model.DB.Select("id, supplier_alias").Where("id IN ?", ids).Find(&apps).Error
	for _, app := range apps {
		if app.SupplierAlias != nil {
			out[app.ID] = *app.SupplierAlias
		}
	}
	return out
}

func localEffectiveWeight(ch LocalUserGroupChannel) int {
	if ch.UserConfigured {
		return ch.UserWeight
	}
	if ch.GlobalConfigured {
		return ch.GlobalWeight
	}
	return 0
}

func localEffectiveEnabled(ch LocalUserGroupChannel) bool {
	if ch.UserConfigured {
		return ch.UserEnabled
	}
	if ch.GlobalConfigured {
		return ch.GlobalEnabled
	}
	return true
}

func maskChannelName(name string) string {
	name = sanitizeUTF8(name)
	runes := []rune(name)
	if len(runes) == 0 {
		return "***"
	}
	if len(runes) <= 2 {
		return string(runes[:1]) + "***"
	}
	if len(runes) <= 4 {
		return string(runes[:1]) + "***" + string(runes[len(runes)-1:])
	}
	return string(runes[:2]) + "***" + string(runes[len(runes)-2:])
}

// resolveUserChannelModelDiscount 计算当前用户在该渠道×模型上的优惠百分点。
// 有用户指定价时按官方价 × 总折扣；否则按渠道有效单价 × 分组倍率相对官方价。
func resolveUserChannelModelDiscount(userID int, ch *model.Channel, modelName string, groupRatio float64) float64 {
	if ov, ok := model.GetEnabledUserModelPricingOverride(userID, modelName); ok {
		tp := ov.TotalPercent()
		if tp < 0 || tp >= 100 {
			return 0
		}
		return 100 - tp
	}
	var official float64
	if p, ok := ratio_setting.GetModelPrice(modelName, false); ok && p > 0 {
		official = p
	} else if r, _, _ := ratio_setting.GetModelRatio(modelName); r > 0 {
		official = r
	} else {
		return 0
	}
	current, ok := ResolveChannelModelConfiguredUnitPrice(ch, modelName)
	if !ok || current <= 0 {
		return 0
	}
	if groupRatio > 0 {
		current *= groupRatio
	}
	if current >= official {
		return 0
	}
	return (1 - current/official) * 100
}

// ResolveChannelModelConfiguredUnitPriceOrZero 用于 UI 展示：有配置定价则返回单价，否则 0。
func ResolveChannelModelConfiguredUnitPriceOrZero(ch *model.Channel, modelName string) float64 {
	if price, ok := ResolveChannelModelConfiguredUnitPrice(ch, modelName); ok {
		return price
	}
	return 0
}

// UserRouteChannelVisibleInGroup 判断渠道是否出现在用户智能路由视图的指定分组中
// （仅已启用渠道，并叠加用户指定价 price_cap / channel_list 过滤）。
func UserRouteChannelVisibleInGroup(userID int, groupKey string, channelID int) bool {
	if userID <= 0 || groupKey == "" || channelID <= 0 {
		return false
	}
	policy, err := GetLocalUserRoutePolicy(userID, true)
	if err != nil || policy == nil {
		return false
	}
	for _, g := range policy.Groups {
		if g.GroupKey != groupKey {
			continue
		}
		for _, ch := range g.Channels {
			if ch.ChannelID == channelID {
				return true
			}
		}
	}
	return false
}
