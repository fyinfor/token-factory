package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
)

// ─── 导出字段键常量 ────────────────────────────────────────────────────────────

const (
	chFieldName           = "name"
	chFieldSyncKey        = "syncKey"
	chFieldDiscountRate   = "discountRate"
	chFieldOperatingCost  = "operatingCostRate"
	chFieldMarkupDiscount = "markupDiscountRate"
	chFieldRouteSlug      = "routeSlug"
	chFieldQuota          = "quota"
	chFieldDisabled       = "disabled"
	chFieldSupplierName   = "supplierName"
	chFieldType           = "type"
	chFieldLogo           = "logo"
	chFieldProviderType   = "providerType"
	chFieldApiKey         = "apiKey"
	chFieldApiBaseUrl     = "apiBaseUrl"
	chFieldModels         = "models"
	chFieldGroups         = "groups"
	chFieldModelRedirect  = "modelRedirect"
	chFieldOtherInfo      = "otherInfo"
)

// chAllowedExportFields 允许导出的合法字段集合，防止非法字段注入。
var chAllowedExportFields = map[string]bool{
	chFieldName: true, chFieldSyncKey: true, chFieldDiscountRate: true, chFieldOperatingCost: true, chFieldMarkupDiscount: true, chFieldRouteSlug: true,
	chFieldQuota: true, chFieldDisabled: true,
	chFieldSupplierName: true, chFieldType: true, chFieldLogo: true,
	chFieldProviderType: true, chFieldApiKey: true, chFieldApiBaseUrl: true,
	chFieldModels: true, chFieldGroups: true, chFieldModelRedirect: true,
	chFieldOtherInfo: true,
}

// ─── DTO 定义 ──────────────────────────────────────────────────────────────────

// ChannelExportRequest 渠道导出请求体。
type ChannelExportRequest struct {
	ChannelIDs []int    `json:"channel_ids"` // 需要导出的渠道 ID 列表
	Fields     []string `json:"fields"`      // 用户选择的字段列表
	Mode       string   `json:"mode"`        // 导出模式: "standard"(默认) | "site_builder"(建站用户导出)
}

// ChannelExportPayload 导出响应的数据结构（可直接用于后续导入）。
type ChannelExportPayload struct {
	Version    string                   `json:"version"`
	ExportTime string                   `json:"exportTime"`
	Channels   []map[string]interface{} `json:"channels"`
}

// ChannelImportRequest 导入请求结构（与导出结构兼容）。
type ChannelImportRequest struct {
	Version           string                   `json:"version"`
	ExportTime        string                   `json:"exportTime"`
	Channels          []map[string]interface{} `json:"channels"`
	SiteBuilderApiKey string                   `json:"site_builder_api_key,omitempty"` // 建站模式统一密钥，导入 type=60 渠道时若 apiKey 为空则原样写入渠道 Key
}

// ChannelImportResult 导入操作的结果统计。
type ChannelImportResult struct {
	Added    int                    `json:"added"`
	Updated  int                    `json:"updated"`
	Failed   int                    `json:"failed"`
	Failures []ChannelImportFailure `json:"failures"`
}

// ChannelImportFailure 单条导入失败的详情。
type ChannelImportFailure struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

// ─── 导出接口 ─────────────────────────────────────────────────────────────────

// ExportChannels 按渠道 ID 列表导出指定字段。
// POST /api/channel/export
// mode=standard (默认): 原样导出渠道数据
// mode=site_builder: 建站用户导出，type 强制为 60，apiKey 置空（由导入方指定建站密钥），
// apiBaseUrl 为本平台 ServerAddress，otherInfo 中标记来源与路由信息。
func ExportChannels(c *gin.Context) {
	var req ChannelExportRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请求格式错误"})
		return
	}
	if len(req.ChannelIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请先选择需要导出的渠道"})
		return
	}

	// 过滤非法字段，只保留允许导出的合法字段
	fieldSet := make(map[string]bool)
	for _, f := range req.Fields {
		if chAllowedExportFields[f] {
			fieldSet[f] = true
		}
	}
	if len(fieldSet) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请至少选择一个导出字段"})
		return
	}
	// 始终包含 name / syncKey，以便导入时做身份匹配
	fieldSet[chFieldName] = true
	fieldSet[chFieldSyncKey] = true

	channels, err := model.GetChannelsByIDs(req.ChannelIDs)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	isSiteBuilder := req.Mode == "site_builder"

	items := make([]map[string]interface{}, 0, len(channels))
	for _, ch := range channels {
		if isSiteBuilder {
			items = append(items, buildSiteBuilderExportItem(c, ch, fieldSet))
		} else {
			items = append(items, buildChannelExportItem(ch, fieldSet))
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": ChannelExportPayload{
			Version:    "1.0",
			ExportTime: time.Now().UTC().Format(time.RFC3339),
			Channels:   items,
		},
	})
}

// buildChannelExportItem 根据字段集合构建单个渠道的导出 map（未选字段不出现在结果中）。
func buildChannelExportItem(ch *model.Channel, fields map[string]bool) map[string]interface{} {
	item := make(map[string]interface{})

	if fields[chFieldName] {
		item[chFieldName] = ch.Name
	}
	if fields[chFieldSyncKey] {
		item[chFieldSyncKey] = ch.SyncKey
	}
	if fields[chFieldDiscountRate] {
		item[chFieldDiscountRate] = ch.PriceDiscountPercent
	}
	if fields[chFieldOperatingCost] {
		item[chFieldOperatingCost] = ch.OperatingCostPercent
	}
	if fields[chFieldMarkupDiscount] {
		item[chFieldMarkupDiscount] = ch.MarkupDiscountRate
	}
	if fields[chFieldRouteSlug] {
		item[chFieldRouteSlug] = ch.RouteSlug
	}
	if fields[chFieldQuota] {
		item[chFieldQuota] = ch.Balance
	}
	if fields[chFieldDisabled] {
		// Status=2 表示禁用，其他值表示启用
		item[chFieldDisabled] = ch.Status == 2
	}
	if fields[chFieldSupplierName] {
		item[chFieldSupplierName] = ch.SupplierName
	}
	if fields[chFieldType] {
		item[chFieldType] = ch.Type
	}
	if fields[chFieldLogo] {
		item[chFieldLogo] = ch.CompanyLogoURL
	}
	if fields[chFieldProviderType] {
		item[chFieldProviderType] = ch.SupplierType
	}
	if fields[chFieldApiKey] {
		item[chFieldApiKey] = ch.Key
	}
	if fields[chFieldApiBaseUrl] {
		baseURL := ""
		if ch.BaseURL != nil {
			baseURL = *ch.BaseURL
		}
		item[chFieldApiBaseUrl] = baseURL
	}
	if fields[chFieldModels] {
		// Models 字段存储为逗号分隔字符串，导出时转换为数组
		item[chFieldModels] = ch.GetModels()
	}
	if fields[chFieldGroups] {
		// Group 字段存储为逗号分隔字符串，导出时转换为数组
		item[chFieldGroups] = ch.GetGroups()
	}
	if fields[chFieldModelRedirect] {
		redirect := map[string]string{}
		if ch.ModelMapping != nil && *ch.ModelMapping != "" {
			_ = common.UnmarshalJsonStr(*ch.ModelMapping, &redirect)
		}
		item[chFieldModelRedirect] = redirect
	}
	if fields[chFieldOtherInfo] {
		otherInfo := ch.GetOtherInfo()
		if len(otherInfo) > 0 {
			item[chFieldOtherInfo] = otherInfo
		}
	}

	return item
}

// buildSiteBuilderExportItem 构建建站用户导出项。
// 核心差异：type 固定为 60 (TokenFactoryOpen)，apiKey 置空（由导入方在导入时指定建站密钥），
// apiBaseUrl 为本平台 ServerAddress，otherInfo 中标记来源和路由信息。
// 折扣：导出给建站用户的成本折扣 = 本平台成本折扣 + 经营成本；
// 经营成本与加价折扣置 0，避免下游再叠一层重复计费。
func buildSiteBuilderExportItem(c *gin.Context, ch *model.Channel, fields map[string]bool) map[string]interface{} {
	item := buildChannelExportItem(ch, fields)

	applySiteBuilderDiscountExport(item, ch, fields)

	// 强制覆盖 type = 60 (TokenFactoryOpen)
	if fields[chFieldType] {
		item[chFieldType] = constant.ChannelTypeTokenFactoryOpen
	}

	// 强制覆盖 apiBaseUrl 为本平台 ServerAddress
	serverAddr := strings.TrimRight(system_setting.ServerAddress, "/")
	if serverAddr == "" {
		// fallback: 从请求中推导
		scheme := "http"
		if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		serverAddr = fmt.Sprintf("%s://%s", scheme, c.Request.Host)
	}
	item[chFieldApiBaseUrl] = serverAddr
	// 确保字段集合中包含 apiBaseUrl，即使原先未勾选
	fields[chFieldApiBaseUrl] = true

	// 建站模式下 apiKey 置空，由导入方通过 site_builder_api_key 参数统一指定密钥。
	// 导入时：若渠道 type=60 且 apiKey 为空，则使用导入请求中的 site_builder_api_key。
	item[chFieldApiKey] = ""
	// 确保字段集合中包含 apiKey
	fields[chFieldApiKey] = true

	// 对于建站导出，在 otherInfo 中标记来源为 tokenfactory_open，
	// 并保留上游路由信息（route_slug 等），以便导入方可正确路由请求到上游渠道。
	otherInfo := ch.GetOtherInfo()
	if otherInfo == nil {
		otherInfo = make(map[string]interface{})
	}
	otherInfo["source"] = "tokenfactory_open"
	// 保留原渠道的 route_slug 作为 upstream_route_slug
	if ch.Id > 0 {
		otherInfo["upstream_channel_id"] = ch.Id
	}
	if ch.RouteSlug != "" {
		otherInfo["upstream_route_slug"] = ch.RouteSlug
	}
	// 保留原渠道的 supplier 信息
	if alias := chUpstreamSupplierAlias(ch); alias != "" {
		otherInfo["upstream_supplier_alias"] = alias
	}
	if ch.ChannelNo != "" {
		otherInfo["upstream_channel_no"] = ch.ChannelNo
	}
	item[chFieldOtherInfo] = otherInfo
	fields[chFieldOtherInfo] = true

	return item
}

// applySiteBuilderDiscountExport 建站导出折扣改写：
// discountRate = ResolvedPriceDiscountPercent + ResolvedOperatingCostPercent，
// 即本平台实际成本率（与计费口径 ResolvedEffectiveCostPercent 一致）。
// 经营成本与加价折扣导出为 0：前者已并入成本折扣，后者由建站侧自行定价，
// 均避免下游在此基础上再叠一层重复计费。
func applySiteBuilderDiscountExport(item map[string]interface{}, ch *model.Channel, fields map[string]bool) {
	if item == nil || ch == nil || fields == nil {
		return
	}
	if !fields[chFieldDiscountRate] && !fields[chFieldOperatingCost] && !fields[chFieldMarkupDiscount] {
		return
	}
	// 勾选任一折扣相关字段时，都写出合并后的成本折扣，保证建站侧拿到完整成本口径。
	item[chFieldDiscountRate] = ch.ResolvedEffectiveCostPercent()
	fields[chFieldDiscountRate] = true
	if fields[chFieldOperatingCost] {
		item[chFieldOperatingCost] = float64(0)
	}
	if fields[chFieldMarkupDiscount] {
		item[chFieldMarkupDiscount] = float64(0)
	}
}

// ─── 导入接口 ──────────────────────────────────────────────────────────────

// ImportChannels 导入渠道配置。
// 匹配优先级：syncKey >（建站）上游身份 > name；命中则更新（仅更新 JSON 中存在的字段）；不存在则新增；
// 绝对禁止清空/覆盖未传字段；绝对禁止删除已有渠道。
// POST /api/channel/import
func ImportChannels(c *gin.Context) {
	var req ChannelImportRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "JSON 格式错误，请上传合法的导出文件"})
		return
	}
	if req.Channels == nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "channels 字段不能为空"})
		return
	}

	result := &ChannelImportResult{Failures: []ChannelImportFailure{}}

	for _, item := range req.Channels {
		// 校验 name 字段
		name, ok := chGetStr(item, "name")
		if !ok || strings.TrimSpace(name) == "" {
			result.Failed++
			result.Failures = append(result.Failures, ChannelImportFailure{Name: "(未知)", Reason: "缺少或无效的 name 字段"})
			continue
		}
		name = strings.TrimSpace(name)

		// 校验字段类型合法性（models/groups 必须为数组，modelRedirect 必须为对象）
		if err := chValidateItem(item); err != nil {
			result.Failed++
			result.Failures = append(result.Failures, ChannelImportFailure{Name: name, Reason: err.Error()})
			continue
		}

		// 建站用户导入不能按同名覆盖普通渠道；仅在已有 TokenFactoryOpen
		// 建站渠道中按上游身份匹配，否则新增。
		existing, err := chFindExistingForImport(name, item)
		if err != nil {
			result.Failed++
			result.Failures = append(result.Failures, ChannelImportFailure{Name: name, Reason: "查询渠道失败: " + err.Error()})
			continue
		}

		if existing != nil {
			// 同名渠道已存在：仅更新 JSON 中存在的字段，不清空其他字段
			if err := chApplyToExisting(existing, item, req.SiteBuilderApiKey); err != nil {
				result.Failed++
				result.Failures = append(result.Failures, ChannelImportFailure{Name: name, Reason: "更新失败: " + err.Error()})
				continue
			}
			result.Updated++
		} else {
			// 不存在同名渠道：新增
			newCh := &model.Channel{}
			if err := chApplyToNew(newCh, item, req.SiteBuilderApiKey); err != nil {
				result.Failed++
				result.Failures = append(result.Failures, ChannelImportFailure{Name: name, Reason: "构建新增数据失败: " + err.Error()})
				continue
			}
			if err := newCh.Insert(); err != nil {
				result.Failed++
				result.Failures = append(result.Failures, ChannelImportFailure{Name: name, Reason: "创建渠道失败: " + err.Error()})
				continue
			}
			result.Added++
		}
	}

	common.SysLog(fmt.Sprintf("渠道导入完成：新增 %d，更新 %d，失败 %d", result.Added, result.Updated, result.Failed))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "渠道导入完成",
		"data":    result,
	})
}

// ─── 内部工具函数 ──────────────────────────────────────────────────────────────

// chGetStr 从 map 中安全读取字符串值。
func chGetStr(m map[string]interface{}, key string) (string, bool) {
	v, ok := m[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// chToFloat64 将 JSON 数字（float64）或其他数值类型统一转为 float64。
func chToFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	}
	return 0
}

func chIsSiteBuilderItem(item map[string]interface{}) bool {
	if v, ok := item["type"]; ok && int(chToFloat64(v)) == constant.ChannelTypeTokenFactoryOpen {
		return true
	}
	if m, ok := item["otherInfo"].(map[string]interface{}); ok {
		source := strings.TrimSpace(common.Interface2String(m["source"]))
		return source == "tokenfactory_open"
	}
	return false
}

func chNormalizeBaseURL(s string) string {
	return strings.TrimRight(strings.TrimSpace(s), "/")
}

func chUpstreamSupplierAlias(ch *model.Channel) string {
	if ch == nil {
		return ""
	}
	if ch.SupplierApplicationID == 0 {
		return "P0"
	}
	var row struct {
		SupplierAlias *string `gorm:"column:supplier_alias"`
	}
	if err := model.DB.Model(&model.SupplierApplication{}).
		Select("supplier_alias").
		Where("id = ?", ch.SupplierApplicationID).
		Scan(&row).Error; err == nil && row.SupplierAlias != nil {
		if alias := strings.TrimSpace(*row.SupplierAlias); alias != "" {
			return alias
		}
	}
	return model.SupplierApplicationAutoAlias(ch.SupplierApplicationID)
}

type chSiteBuilderIdentity struct {
	BaseURL       string
	UpstreamID    int
	UpstreamRoute string
	Alias         string
	ChannelNo     string
}

func chSiteBuilderIdentityFromItem(item map[string]interface{}) chSiteBuilderIdentity {
	var ident chSiteBuilderIdentity
	if s, ok := item["apiBaseUrl"].(string); ok {
		ident.BaseURL = chNormalizeBaseURL(s)
	}
	if m, ok := item["otherInfo"].(map[string]interface{}); ok {
		ident.UpstreamID = int(chToFloat64(m["upstream_channel_id"]))
		ident.UpstreamRoute = strings.TrimSpace(common.Interface2String(m["upstream_route_slug"]))
		ident.Alias = strings.TrimSpace(common.Interface2String(m["upstream_supplier_alias"]))
		ident.ChannelNo = strings.TrimSpace(common.Interface2String(m["upstream_channel_no"]))
	}
	return ident
}

func chSiteBuilderIdentityFromChannel(ch *model.Channel) chSiteBuilderIdentity {
	otherInfo := ch.GetOtherInfo()
	return chSiteBuilderIdentity{
		BaseURL:       chNormalizeBaseURL(ch.GetBaseURL()),
		UpstreamID:    int(chToFloat64(otherInfo["upstream_channel_id"])),
		UpstreamRoute: strings.TrimSpace(common.Interface2String(otherInfo["upstream_route_slug"])),
		Alias:         strings.TrimSpace(common.Interface2String(otherInfo["upstream_supplier_alias"])),
		ChannelNo:     strings.TrimSpace(common.Interface2String(otherInfo["upstream_channel_no"])),
	}
}

func chSiteBuilderIdentityHasRoute(ident chSiteBuilderIdentity) bool {
	return ident.UpstreamRoute != "" || ident.UpstreamID > 0 || (ident.Alias != "" && ident.ChannelNo != "")
}

func chSiteBuilderIdentityMatches(a, b chSiteBuilderIdentity) bool {
	if a.BaseURL != "" && b.BaseURL != "" && a.BaseURL != b.BaseURL {
		return false
	}
	if a.UpstreamRoute != "" && b.UpstreamRoute != "" {
		return a.UpstreamRoute == b.UpstreamRoute
	}
	if a.UpstreamID > 0 && b.UpstreamID > 0 {
		return a.UpstreamID == b.UpstreamID
	}
	if a.Alias != "" && a.ChannelNo != "" && b.Alias != "" && b.ChannelNo != "" {
		return a.Alias == b.Alias && a.ChannelNo == b.ChannelNo
	}
	return false
}

func chFindExistingForImport(name string, item map[string]interface{}) (*model.Channel, error) {
	if syncKey, ok := chGetStr(item, chFieldSyncKey); ok {
		syncKey = model.NormalizeChannelSyncKey(syncKey)
		if syncKey != "" {
			if !model.IsValidChannelSyncKey(syncKey) {
				return nil, fmt.Errorf("sync_key 格式无效")
			}
			byKey, err := model.GetChannelBySyncKey(syncKey)
			if err != nil {
				return nil, err
			}
			if byKey != nil {
				return byKey, nil
			}
		}
	}

	if !chIsSiteBuilderItem(item) {
		existing, err := model.GetChannelByName(name)
		if err != nil {
			return nil, err
		}
		if existing == nil {
			return nil, nil
		}
		// 名称命中但对方已有不同 sync_key：避免误覆盖绑定，视为未命中（后续走新增）
		if incoming, ok := chGetStr(item, chFieldSyncKey); ok {
			incoming = model.NormalizeChannelSyncKey(incoming)
			existingKey := model.NormalizeChannelSyncKey(existing.SyncKey)
			if incoming != "" && existingKey != "" && incoming != existingKey {
				return nil, nil
			}
		}
		return existing, nil
	}

	var candidates []model.Channel
	if err := model.DB.
		Where("name = ? AND type = ?", name, constant.ChannelTypeTokenFactoryOpen).
		Find(&candidates).Error; err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, nil
	}

	incoming := chSiteBuilderIdentityFromItem(item)
	if chSiteBuilderIdentityHasRoute(incoming) {
		for i := range candidates {
			existing := chSiteBuilderIdentityFromChannel(&candidates[i])
			if chSiteBuilderIdentityMatches(incoming, existing) {
				return &candidates[i], nil
			}
		}
		return nil, nil
	}

	if len(candidates) == 1 {
		return &candidates[0], nil
	}
	return nil, nil
}

func chShouldRefreshAbilities(item map[string]interface{}) bool {
	if _, ok := item["models"]; ok {
		return true
	}
	if _, ok := item["groups"]; ok {
		return true
	}
	if _, ok := item["disabled"]; ok {
		return true
	}
	return false
}

// chValidateItem 校验导入条目中各字段的类型合法性。
// 非法字段跳过，不影响其他条目继续处理。
func chValidateItem(item map[string]interface{}) error {
	if v, ok := item["models"]; ok && v != nil {
		if _, ok := v.([]interface{}); !ok {
			return fmt.Errorf("models 字段必须为数组")
		}
	}
	if v, ok := item["groups"]; ok && v != nil {
		if _, ok := v.([]interface{}); !ok {
			return fmt.Errorf("groups 字段必须为数组")
		}
	}
	if v, ok := item["modelRedirect"]; ok && v != nil {
		if _, ok := v.(map[string]interface{}); !ok {
			return fmt.Errorf("modelRedirect 字段必须为对象")
		}
	}
	if v, ok := item["otherInfo"]; ok && v != nil {
		if _, ok := v.(map[string]interface{}); !ok {
			return fmt.Errorf("otherInfo 字段必须为对象")
		}
	}
	return nil
}

// chApplyToExisting 将导入数据应用到已存在的渠道（精确更新，仅更新 JSON 中存在的字段）。
// 通过 GORM Select+Updates 确保只写入指定列，不影响其他列。
// siteBuilderApiKey: 建站模式统一密钥，当渠道 type=60 且 apiKey 为空时使用此值。
func chApplyToExisting(ch *model.Channel, item map[string]interface{}, siteBuilderApiKey string) error {
	cols := make([]string, 0, len(item))
	updates := &model.Channel{}

	// sync_key 命中后允许改名；仅 name 命中时写入同名也无害
	if v, ok := item[chFieldName]; ok {
		if s, ok := v.(string); ok {
			s = strings.TrimSpace(s)
			if s != "" {
				updates.Name = s
				cols = append(cols, "name")
			}
		}
	}
	if v, ok := item[chFieldSyncKey]; ok {
		if s, ok := v.(string); ok {
			s = model.NormalizeChannelSyncKey(s)
			if s != "" {
				if !model.IsValidChannelSyncKey(s) {
					return fmt.Errorf("sync_key 格式无效")
				}
				if err := model.ValidateChannelSyncKeyUnique(ch.Id, s); err != nil {
					return err
				}
				updates.SyncKey = s
				cols = append(cols, "sync_key")
			}
		}
	}
	if v, ok := item["discountRate"]; ok {
		f := chToFloat64(v)
		updates.PriceDiscountPercent = &f
		cols = append(cols, "price_discount_percent")
	}
	if v, ok := item["operatingCostRate"]; ok {
		f := chToFloat64(v)
		updates.OperatingCostPercent = &f
		cols = append(cols, "operating_cost_percent")
	}
	if v, ok := item["markupDiscountRate"]; ok {
		f := chToFloat64(v)
		updates.MarkupDiscountRate = &f
		cols = append(cols, "markup_discount_rate")
	}
	if v, ok := item["disabled"]; ok {
		if b, isBool := v.(bool); isBool {
			if b {
				updates.Status = 2 // 禁用
			} else {
				updates.Status = 1 // 启用
			}
			cols = append(cols, "status")
		}
	}
	if v, ok := item["type"]; ok {
		updates.Type = int(chToFloat64(v))
		cols = append(cols, "type")
	}
	if v, ok := item["logo"]; ok {
		if s, ok := v.(string); ok {
			updates.CompanyLogoURL = s
			cols = append(cols, "company_logo_url")
		}
	}
	if v, ok := item["providerType"]; ok {
		if s, ok := v.(string); ok {
			updates.SupplierType = s
			cols = append(cols, "supplier_type")
		}
	}
	if v, ok := item["apiKey"]; ok {
		if s, ok := v.(string); ok {
			// 建站模式：当 apiKey 为空且渠道 type=60 时，使用导入请求中的统一密钥
			if strings.TrimSpace(s) == "" && siteBuilderApiKey != "" {
				channelType := 0
				if t, ok := item["type"]; ok {
					channelType = int(chToFloat64(t))
				} else {
					channelType = ch.Type
				}
				if channelType == constant.ChannelTypeTokenFactoryOpen {
					s = strings.TrimSpace(siteBuilderApiKey)
				}
			}
			updates.Key = s
			cols = append(cols, "key")
		}
	}
	if v, ok := item["apiBaseUrl"]; ok {
		if s, ok := v.(string); ok {
			updates.BaseURL = &s
			cols = append(cols, "base_url")
		}
	}
	if v, ok := item["models"]; ok {
		if arr, ok := v.([]interface{}); ok {
			parts := make([]string, 0, len(arr))
			for _, m := range arr {
				if s, ok := m.(string); ok && strings.TrimSpace(s) != "" {
					parts = append(parts, strings.TrimSpace(s))
				}
			}
			updates.Models = strings.Join(parts, ",")
			cols = append(cols, "models")
		}
	}
	if v, ok := item["groups"]; ok {
		if arr, ok := v.([]interface{}); ok {
			parts := make([]string, 0, len(arr))
			for _, g := range arr {
				if s, ok := g.(string); ok && strings.TrimSpace(s) != "" {
					parts = append(parts, strings.TrimSpace(s))
				}
			}
			updates.Group = strings.Join(parts, ",")
			cols = append(cols, "group")
		}
	}
	if v, ok := item["modelRedirect"]; ok {
		if m, ok := v.(map[string]interface{}); ok {
			redirect := make(map[string]string, len(m))
			for k, val := range m {
				if s, ok := val.(string); ok {
					redirect[k] = s
				}
			}
			b, err := common.Marshal(redirect)
			if err != nil {
				return fmt.Errorf("序列化 modelRedirect 失败: %w", err)
			}
			s := string(b)
			updates.ModelMapping = &s
			cols = append(cols, "model_mapping")
		}
	}
	if v, ok := item["quota"]; ok {
		updates.Balance = chToFloat64(v)
		cols = append(cols, "balance")
	}
	if v, ok := item["routeSlug"]; ok {
		if s, ok := v.(string); ok {
			updates.RouteSlug = s
			cols = append(cols, "route_slug")
		}
	}
	if v, ok := item["otherInfo"]; ok {
		if m, ok := v.(map[string]interface{}); ok {
			b, err := common.Marshal(m)
			if err != nil {
				return fmt.Errorf("序列化 otherInfo 失败: %w", err)
			}
			s := string(b)
			updates.OtherInfo = s
			cols = append(cols, "other_info")
		}
	}

	if len(cols) == 0 {
		// 没有可更新的字段，直接跳过（不报错）
		return nil
	}

	// 使用精确列选择更新，确保只写入指定列
	if err := model.PartialUpdateChannelFields(ch.Id, cols, updates); err != nil {
		return err
	}
	if chShouldRefreshAbilities(item) {
		refreshed, err := model.GetChannelById(ch.Id, true)
		if err != nil {
			return fmt.Errorf("刷新渠道能力失败: %w", err)
		}
		if err := refreshed.UpdateAbilities(nil); err != nil {
			return fmt.Errorf("刷新渠道能力失败: %w", err)
		}
	}
	return nil
}

// chApplyToNew 将导入数据写入新渠道对象，用于新增场景。
// siteBuilderApiKey: 建站模式统一密钥，当渠道 type=60 且 apiKey 为空时使用此值。
func chApplyToNew(ch *model.Channel, item map[string]interface{}, siteBuilderApiKey string) error {
	name, ok := chGetStr(item, "name")
	if !ok || strings.TrimSpace(name) == "" {
		return fmt.Errorf("name 字段缺失")
	}
	ch.Name = strings.TrimSpace(name)

	if syncKey, ok := chGetStr(item, chFieldSyncKey); ok {
		syncKey = model.NormalizeChannelSyncKey(syncKey)
		if syncKey != "" {
			if !model.IsValidChannelSyncKey(syncKey) {
				return fmt.Errorf("sync_key 格式无效")
			}
			if err := model.ValidateChannelSyncKeyUnique(0, syncKey); err != nil {
				return err
			}
			ch.SyncKey = syncKey
		}
	}
	model.EnsureChannelSyncKey(ch)

	// 默认启用；若 JSON 中 disabled=true 则禁用
	ch.Status = 1
	if v, ok := item["disabled"]; ok {
		if b, isBool := v.(bool); isBool && b {
			ch.Status = 2
		}
	}

	if v, ok := item["discountRate"]; ok {
		f := chToFloat64(v)
		ch.PriceDiscountPercent = &f
	}
	if v, ok := item["operatingCostRate"]; ok {
		f := chToFloat64(v)
		ch.OperatingCostPercent = &f
	}
	if v, ok := item["markupDiscountRate"]; ok {
		f := chToFloat64(v)
		ch.MarkupDiscountRate = &f
	}
	if v, ok := item["routeSlug"]; ok {
		if s, ok := v.(string); ok {
			ch.RouteSlug = s
		}
	}
	if v, ok := item["quota"]; ok {
		ch.Balance = chToFloat64(v)
	}
	if v, ok := item["type"]; ok {
		ch.Type = int(chToFloat64(v))
	}
	if v, ok := item["logo"]; ok {
		if s, ok := v.(string); ok {
			ch.CompanyLogoURL = s
		}
	}
	if v, ok := item["providerType"]; ok {
		if s, ok := v.(string); ok {
			ch.SupplierType = s
		}
	}
	if v, ok := item["apiKey"]; ok {
		if s, ok := v.(string); ok {
			// 建站模式：当 apiKey 为空且渠道 type=60 时，使用导入请求中的统一密钥
			if strings.TrimSpace(s) == "" && siteBuilderApiKey != "" {
				if ch.Type == constant.ChannelTypeTokenFactoryOpen {
					s = strings.TrimSpace(siteBuilderApiKey)
				}
			}
			ch.Key = s
		}
	}
	if v, ok := item["apiBaseUrl"]; ok {
		if s, ok := v.(string); ok {
			ch.BaseURL = &s
		}
	}
	if v, ok := item["models"]; ok {
		if arr, ok := v.([]interface{}); ok {
			parts := make([]string, 0, len(arr))
			for _, m := range arr {
				if s, ok := m.(string); ok && strings.TrimSpace(s) != "" {
					parts = append(parts, strings.TrimSpace(s))
				}
			}
			ch.Models = strings.Join(parts, ",")
		}
	}
	if v, ok := item["groups"]; ok {
		if arr, ok := v.([]interface{}); ok {
			parts := make([]string, 0, len(arr))
			for _, g := range arr {
				if s, ok := g.(string); ok && strings.TrimSpace(s) != "" {
					parts = append(parts, strings.TrimSpace(s))
				}
			}
			ch.Group = strings.Join(parts, ",")
		}
	} else {
		ch.Group = "default" // 新增渠道默认分组
	}
	if v, ok := item["modelRedirect"]; ok {
		if m, ok := v.(map[string]interface{}); ok {
			redirect := make(map[string]string, len(m))
			for k, val := range m {
				if s, ok := val.(string); ok {
					redirect[k] = s
				}
			}
			b, err := common.Marshal(redirect)
			if err != nil {
				return fmt.Errorf("序列化 modelRedirect 失败: %w", err)
			}
			s := string(b)
			ch.ModelMapping = &s
		}
	}
	if v, ok := item["otherInfo"]; ok {
		if m, ok := v.(map[string]interface{}); ok {
			b, err := common.Marshal(m)
			if err != nil {
				return fmt.Errorf("序列化 otherInfo 失败: %w", err)
			}
			ch.OtherInfo = string(b)
		}
	}

	return nil
}
