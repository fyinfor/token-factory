package controller

import (
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

type playgroundMediaRouteOption struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	RouteSlug    string `json:"route_slug,omitempty"`
	SupplierType string `json:"supplier_type,omitempty"`
	Label        string `json:"label"`
}

type playgroundMediaModelOption struct {
	Model  string                       `json:"model"`
	Routes []playgroundMediaRouteOption `json:"routes,omitempty"`
}

type playgroundMediaResolutionOption struct {
	Label         string `json:"label"`
	Value         string `json:"value"`
	RawResolution string `json:"raw_resolution"`
	Lane          string `json:"lane,omitempty"`
	BillingMode   string `json:"billing_mode,omitempty"`
}

type playgroundMediaOptionsResponse struct {
	Image []playgroundMediaModelOption `json:"image"`
	Video []playgroundMediaModelOption `json:"video"`
}

func tokenAllowsDiscoveryModel(c *gin.Context, modelName string) bool {
	if !common.GetContextKeyBool(c, constant.ContextKeyTokenModelLimitEnabled) {
		return true
	}
	raw, ok := common.GetContextKey(c, constant.ContextKeyTokenModelLimit)
	if !ok {
		return false
	}
	limits, ok := raw.(map[string]bool)
	if !ok {
		return false
	}
	return model.ModelLimitMapAllows(limits, modelName)
}

func playgroundDiscoveryGroupsForToken(c *gin.Context) ([]string, bool) {
	userID := c.GetInt("id")
	user, err := model.GetUserCache(userID)
	if err != nil {
		common.ApiError(c, err)
		return nil, false
	}
	tokenGroup := common.GetContextKeyString(c, constant.ContextKeyTokenGroup)
	if tokenGroup == "auto" {
		groups := service.GetUserAutoGroup(user.Group)
		if len(groups) > 0 {
			return groups, true
		}
	}
	if tokenGroup != "" {
		return []string{tokenGroup}, true
	}
	usingGroup := common.GetContextKeyString(c, constant.ContextKeyUsingGroup)
	if usingGroup != "" && usingGroup != "auto" {
		return []string{usingGroup}, true
	}
	return []string{user.Group}, true
}

func collectDiscoveryModels(c *gin.Context, groups []string) []string {
	pricingShowable := CollectPricingShowableModelNames(c.GetInt("id"))
	seen := make(map[string]struct{})
	out := make([]string, 0)
	for _, group := range groups {
		for _, modelName := range model.GetGroupEnabledModels(group) {
			modelName = strings.TrimSpace(modelName)
			if modelName == "" {
				continue
			}
			if !pricingShowable[modelName] || !tokenAllowsDiscoveryModel(c, modelName) {
				continue
			}
			allowed, err := model.UserCanAccessModel(c.GetInt("id"), modelName)
			if err != nil || !allowed {
				continue
			}
			if _, ok := seen[modelName]; ok {
				continue
			}
			seen[modelName] = struct{}{}
			out = append(out, modelName)
		}
	}
	sort.Strings(out)
	return out
}

func splitDiscoveryTags(csv string) []string {
	if strings.TrimSpace(csv) == "" {
		return nil
	}
	parts := strings.FieldsFunc(csv, func(r rune) bool {
		return r == ',' || r == '，' || r == '、'
	})
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{})
	for _, part := range parts {
		tag := strings.TrimSpace(part)
		if tag == "" {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		out = append(out, tag)
	}
	return out
}

func collectDiscoveryModelTags(models []string) map[string][]string {
	if len(models) == 0 {
		return nil
	}
	rows := make([]struct {
		ModelName string `gorm:"column:model_name"`
		Tags      string `gorm:"column:tags"`
		NameRule  int    `gorm:"column:name_rule"`
	}, 0)
	if err := model.DB.Model(&model.Model{}).
		Select("model_name", "tags", "name_rule").
		Where("status = ?", 1).
		Find(&rows).Error; err != nil {
		return nil
	}
	rulePriority := func(rule int) int {
		switch rule {
		case model.NameRuleExact:
			return 0
		case model.NameRulePrefix:
			return 1
		case model.NameRuleSuffix:
			return 2
		case model.NameRuleContains:
			return 3
		default:
			return 9
		}
	}
	matchRule := func(pattern, target string, rule int) bool {
		switch rule {
		case model.NameRuleExact:
			return target == pattern
		case model.NameRulePrefix:
			return strings.HasPrefix(target, pattern)
		case model.NameRuleSuffix:
			return strings.HasSuffix(target, pattern)
		case model.NameRuleContains:
			return strings.Contains(target, pattern)
		default:
			return false
		}
	}
	out := make(map[string][]string, len(models))
	for _, targetModelName := range models {
		bestIdx := -1
		for i := range rows {
			row := rows[i]
			if !matchRule(row.ModelName, targetModelName, row.NameRule) {
				continue
			}
			if bestIdx < 0 {
				bestIdx = i
				continue
			}
			cur := rows[bestIdx]
			curPriority := rulePriority(cur.NameRule)
			newPriority := rulePriority(row.NameRule)
			if newPriority < curPriority || (newPriority == curPriority && len(row.ModelName) > len(cur.ModelName)) {
				bestIdx = i
			}
		}
		if bestIdx >= 0 {
			out[targetModelName] = splitDiscoveryTags(rows[bestIdx].Tags)
		}
	}
	return out
}

func tagListContains(tags []string, want string) bool {
	for _, tag := range tags {
		if tag == want {
			return true
		}
	}
	return false
}

func endpointListHasMedia(modelName, mediaMode string) bool {
	for _, endpointType := range model.GetModelSupportEndpointTypes(modelName) {
		switch mediaMode {
		case "image":
			if endpointType == constant.EndpointTypeImageGeneration ||
				endpointType == constant.EndpointTypeTencentCloudVODImage {
				return true
			}
		case "video":
			if endpointType == constant.EndpointTypeOpenAIVideo ||
				endpointType == constant.EndpointTypeOpenAIVideoGW ||
				endpointType == constant.EndpointTypeTokenFactoryVideo ||
				endpointType == constant.EndpointTypeVideoGenerator ||
				endpointType == constant.EndpointTypeTencentCloudVODVideo ||
				endpointType == constant.EndpointTypeAliVideo ||
				endpointType == constant.EndpointTypeSeedanceVideo {
				return true
			}
		}
	}
	return false
}

func discoveryModelSupportsMedia(modelName string, tags []string, mediaMode string) bool {
	switch mediaMode {
	case "image":
		return tagListContains(tags, "图片") || endpointListHasMedia(modelName, "image")
	case "video":
		return tagListContains(tags, "视频") || endpointListHasMedia(modelName, "video")
	default:
		return false
	}
}

func collectDiscoveryChannelIDs(groups []string, modelName string) map[int]struct{} {
	out := make(map[int]struct{})
	for _, group := range groups {
		for _, channelID := range model.GetGroupEnabledChannelIDs(group, modelName) {
			out[channelID] = struct{}{}
		}
	}
	return out
}

func collectDiscoveryRoutes(channelIDs map[int]struct{}) []playgroundMediaRouteOption {
	routes := make([]playgroundMediaRouteOption, 0)
	seen := make(map[string]struct{})
	for channelID := range channelIDs {
		ch, err := model.CacheGetChannel(channelID)
		if err != nil || ch == nil || ch.Status != common.ChannelStatusEnabled {
			continue
		}
		routeSlug := strings.TrimSpace(ch.RouteSlug)
		if routeSlug == "" {
			continue
		}
		if _, ok := seen[routeSlug]; ok {
			continue
		}
		seen[routeSlug] = struct{}{}
		parts := []string{routeSlug}
		if supplierType := strings.TrimSpace(ch.SupplierType); supplierType != "" {
			parts = append(parts, supplierType)
		}
		routes = append(routes, playgroundMediaRouteOption{
			ID:           ch.Id,
			Name:         ch.Name,
			RouteSlug:    routeSlug,
			SupplierType: ch.SupplierType,
			Label:        strings.Join(parts, "-"),
		})
	}
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Label == routes[j].Label {
			return routes[i].ID < routes[j].ID
		}
		return routes[i].Label < routes[j].Label
	})
	return routes
}

func compactResolution(raw string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(raw), " ", ""))
}

func resolutionLabel(raw string) string {
	compact := compactResolution(raw)
	if compact == "" {
		return ""
	}
	if strings.HasSuffix(compact, "p") {
		return strings.TrimSuffix(compact, "p") + "p"
	}
	if _, err := strconv.Atoi(compact); err == nil {
		return compact + "p"
	}
	if strings.HasSuffix(compact, "k") {
		return strings.ToUpper(compact)
	}
	if strings.Contains(compact, "x") {
		parts := strings.Split(compact, "x")
		if len(parts) == 2 {
			w, wErr := strconv.Atoi(parts[0])
			h, hErr := strconv.Atoi(parts[1])
			if wErr == nil && hErr == nil && w > 0 && h > 0 {
				short := w
				if h < short {
					short = h
				}
				switch {
				case short >= 2160:
					return "4K"
				case short >= 1440:
					return "2K"
				case short >= 1080:
					return "1080p"
				case short >= 720:
					return "720p"
				case short >= 540:
					return "540p"
				case short >= 480:
					return "480p"
				default:
					return strconv.Itoa(short) + "p"
				}
			}
		}
	}
	return raw
}

func imageResolutionValue(raw string) string {
	compact := compactResolution(raw)
	switch compact {
	case "480", "480p":
		return "854x480"
	case "540", "540p":
		return "960x540"
	case "720", "720p":
		return "1280x720"
	case "1080", "1080p":
		return "1920x1080"
	case "2k":
		return "2560x1440"
	case "4k":
		return "3840x2160"
	}
	if strings.Contains(compact, "x") {
		return compact
	}
	return raw
}

func normalizeDiscoveryResolutions(mediaMode string, rawTiers interface{}) []playgroundMediaResolutionOption {
	type rawTier struct {
		Resolution  string
		Lane        string
		BillingMode string
	}
	items := make([]rawTier, 0)
	switch tiers := rawTiers.(type) {
	case []playgroundImagePricingTier:
		for _, tier := range tiers {
			items = append(items, rawTier{Resolution: tier.Resolution, Lane: tier.Lane})
		}
	case []playgroundVideoPricingTier:
		for _, tier := range tiers {
			items = append(items, rawTier{Resolution: tier.Resolution, Lane: tier.Lane, BillingMode: tier.BillingMode})
		}
	}
	seen := make(map[string]struct{})
	out := make([]playgroundMediaResolutionOption, 0, len(items))
	for _, item := range items {
		raw := strings.TrimSpace(item.Resolution)
		if raw == "" {
			continue
		}
		value := resolutionLabel(raw)
		label := value
		if mediaMode == "image" {
			value = imageResolutionValue(raw)
			label = resolutionLabel(raw)
			if label != "" && value != "" && compactResolution(label) != compactResolution(value) {
				label = label + " (" + value + ")"
			}
		}
		if value == "" {
			value = raw
		}
		if label == "" {
			label = value
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, playgroundMediaResolutionOption{
			Label:         label,
			Value:         value,
			RawResolution: raw,
			Lane:          item.Lane,
			BillingMode:   item.BillingMode,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Value < out[j].Value
	})
	return out
}

func preferredDiscoveryDefault(mediaMode string, options []playgroundMediaResolutionOption) string {
	if len(options) == 0 {
		if mediaMode == "video" {
			return "720p"
		}
		return "1280x720"
	}
	preferred := "720p"
	if mediaMode == "image" {
		preferred = "1280x720"
	}
	for _, option := range options {
		if strings.EqualFold(option.Value, preferred) {
			return option.Value
		}
	}
	return options[0].Value
}

func resolutionSortRank(value string) int {
	compact := compactResolution(value)
	if compact == "" {
		return 1 << 30
	}
	if strings.Contains(compact, "x") {
		parts := strings.Split(compact, "x")
		if len(parts) == 2 {
			w, wErr := strconv.Atoi(parts[0])
			h, hErr := strconv.Atoi(parts[1])
			if wErr == nil && hErr == nil && w > 0 && h > 0 {
				if w < h {
					return w
				}
				return h
			}
		}
	}
	if strings.HasSuffix(compact, "k") {
		n, err := strconv.Atoi(strings.TrimSuffix(compact, "k"))
		if err == nil && n > 0 {
			return n * 720
		}
	}
	if strings.HasSuffix(compact, "p") {
		n, err := strconv.Atoi(strings.TrimSuffix(compact, "p"))
		if err == nil && n > 0 {
			return n
		}
	}
	if n, err := strconv.Atoi(compact); err == nil && n > 0 {
		return n
	}
	return 1 << 30
}

func filterRouteChannelIDs(channelIDs map[int]struct{}, routeSlug string) map[int]struct{} {
	routeSlug = strings.TrimSpace(routeSlug)
	if routeSlug == "" {
		return channelIDs
	}
	out := make(map[int]struct{})
	for channelID := range channelIDs {
		ch, err := model.CacheGetChannel(channelID)
		if err != nil || ch == nil || ch.Status != common.ChannelStatusEnabled {
			continue
		}
		if strings.TrimSpace(ch.RouteSlug) == routeSlug {
			out[channelID] = struct{}{}
		}
	}
	return out
}

// GetPlaygroundMediaOptions exposes token-authenticated media discovery for external assistants.
// It allows a regular API token to list usable image/video models and discover model resolution tiers.
func GetPlaygroundMediaOptions(c *gin.Context) {
	groups, ok := playgroundDiscoveryGroupsForToken(c)
	if !ok {
		return
	}
	models := collectDiscoveryModels(c, groups)
	tagsByModel := collectDiscoveryModelTags(models)
	mode := strings.ToLower(strings.TrimSpace(c.Query("mode")))
	modelName := strings.TrimSpace(c.Query("model"))
	routeSlug := strings.TrimSpace(c.Query("route_slug"))

	if mode != "" && mode != "image" && mode != "video" {
		common.ApiErrorMsg(c, "mode must be image or video")
		return
	}

	if modelName != "" {
		if mode == "" {
			common.ApiErrorMsg(c, "mode must be image or video")
			return
		}
		allowed := false
		for _, candidate := range models {
			if candidate == modelName {
				allowed = true
				break
			}
		}
		if !allowed || !discoveryModelSupportsMedia(modelName, tagsByModel[modelName], mode) {
			common.ApiErrorMsg(c, "model is not available for this token and media mode")
			return
		}
		channelIDs := filterRouteChannelIDs(collectDiscoveryChannelIDs(groups, modelName), routeSlug)
		if routeSlug != "" && len(channelIDs) == 0 {
			common.ApiErrorMsg(c, "route_slug is not available for this token and model")
			return
		}
		routes := collectDiscoveryRoutes(channelIDs)
		var resolutions []playgroundMediaResolutionOption
		if mode == "image" {
			resolutions = normalizeDiscoveryResolutions(mode, collectPlaygroundImagePricingTiers(modelName, channelIDs))
		} else {
			resolutions = normalizeDiscoveryResolutions(mode, collectPlaygroundVideoPricingTiers(modelName, channelIDs))
		}
		sort.Slice(resolutions, func(i, j int) bool {
			ri := resolutionSortRank(resolutions[i].Value)
			rj := resolutionSortRank(resolutions[j].Value)
			if ri == rj {
				return resolutions[i].Value < resolutions[j].Value
			}
			return ri < rj
		})
		defaultValue := preferredDiscoveryDefault(mode, resolutions)
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "",
			"data": gin.H{
				"mode":               mode,
				"model":              modelName,
				"route_slug":         routeSlug,
				"routes":             routes,
				"resolutions":        resolutions,
				"default_resolution": defaultValue,
				"default_size":       defaultValue,
			},
		})
		return
	}

	response := playgroundMediaOptionsResponse{
		Image: make([]playgroundMediaModelOption, 0),
		Video: make([]playgroundMediaModelOption, 0),
	}
	for _, modelName := range models {
		tags := tagsByModel[modelName]
		channelIDs := collectDiscoveryChannelIDs(groups, modelName)
		item := playgroundMediaModelOption{
			Model:  modelName,
			Routes: collectDiscoveryRoutes(channelIDs),
		}
		if mode == "" || mode == "image" {
			if discoveryModelSupportsMedia(modelName, tags, "image") {
				response.Image = append(response.Image, item)
			}
		}
		if mode == "" || mode == "video" {
			if discoveryModelSupportsMedia(modelName, tags, "video") {
				response.Video = append(response.Video, item)
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    response,
	})
}
