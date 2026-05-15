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
	chFieldName          = "name"
	chFieldDiscountRate  = "discountRate"
	chFieldRouteSlug     = "routeSlug"
	chFieldQuota         = "quota"
	chFieldDisabled      = "disabled"
	chFieldSupplierName  = "supplierName"
	chFieldType          = "type"
	chFieldLogo          = "logo"
	chFieldProviderType  = "providerType"
	chFieldApiKey        = "apiKey"
	chFieldApiBaseUrl    = "apiBaseUrl"
	chFieldModels        = "models"
	chFieldGroups        = "groups"
	chFieldModelRedirect = "modelRedirect"
)

// chAllowedExportFields 允许导出的合法字段集合，防止非法字段注入。
var chAllowedExportFields = map[string]bool{
	chFieldName: true, chFieldDiscountRate: true, chFieldRouteSlug: true,
	chFieldQuota: true, chFieldDisabled: true,
	chFieldSupplierName: true, chFieldType: true, chFieldLogo: true,
	chFieldProviderType: true, chFieldApiKey: true, chFieldApiBaseUrl: true,
	chFieldModels: true, chFieldGroups: true, chFieldModelRedirect: true,
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
	Version    string                   `json:"version"`
	ExportTime string                   `json:"exportTime"`
	Channels   []map[string]interface{} `json:"channels"`
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
// mode=site_builder: 建站用户导出，type 强制为 60，apiKey 为新生成的令牌 key，
// apiBaseUrl 为本平台 ServerAddress，每个渠道创建独立令牌并绑定渠道模型范围。
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
	// 始终包含 name，以便导入时做名称匹配
	fieldSet[chFieldName] = true

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

	// 建站用户导出时过滤掉令牌创建失败的条目
	if isSiteBuilder {
		filtered := make([]map[string]interface{}, 0, len(items))
		for _, item := range items {
			if _, failed := item["__export_failed__"]; !failed {
				filtered = append(filtered, item)
			}
		}
		// 清理内部标记
		for _, item := range filtered {
			delete(item, "__export_failed__")
		}
		items = filtered
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
	if fields[chFieldDiscountRate] {
		item[chFieldDiscountRate] = ch.PriceDiscountPercent
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

	return item
}

// buildSiteBuilderExportItem 构建建站用户导出项。
// 核心差异：type 固定为 60 (TokenFactoryOpen)，apiKey 为新生成的令牌 key（sk-前缀），
// apiBaseUrl 为本平台 ServerAddress，同时为每个渠道创建独立令牌并限定模型范围。
func buildSiteBuilderExportItem(c *gin.Context, ch *model.Channel, fields map[string]bool) map[string]interface{} {
	item := buildChannelExportItem(ch, fields)

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

	// 为该渠道生成独立令牌，限定模型范围
	channelModels := ch.GetModels()
	modelLimits := strings.Join(channelModels, ",")
	tokenName := fmt.Sprintf("建站导出-%s", ch.Name)

	key, err := common.GenerateKey()
	if err != nil {
		common.SysError("建站用户导出: 生成令牌密钥失败: " + err.Error())
		item["__export_failed__"] = true
		return item
	}

	userId := c.GetInt("id")
	if userId == 0 {
		// 管理员接口应该有用户 ID，兜底使用 1
		userId = 1
	}

	newToken := &model.Token{
		UserId:             userId,
		Name:               tokenName,
		Key:                key,
		CreatedTime:        common.GetTimestamp(),
		AccessedTime:       common.GetTimestamp(),
		ExpiredTime:        -1,
		UnlimitedQuota:     true,
		ModelLimitsEnabled: len(channelModels) > 0,
		ModelLimits:        modelLimits,
		Group:              ch.Group,
	}
	if err := newToken.Insert(); err != nil {
		common.SysError(fmt.Sprintf("建站用户导出: 创建令牌失败 (渠道 %s): %v", ch.Name, err))
		item["__export_failed__"] = true
		return item
	}

	// 覆盖 apiKey 为新令牌的 key（带 sk- 前缀，与 TokenFactoryOpen 导入格式一致）
	item[chFieldApiKey] = "sk-" + key
	// 确保字段集合中包含 apiKey
	fields[chFieldApiKey] = true

	return item
}

// ─── 导入接口 ──────────────────────────────────────────────────────────────

// ImportChannels 按名称匹配导入渠道配置。
// 核心规则：仅通过 name 匹配；同名则更新（仅更新 JSON 中存在的字段）；不存在则新增；
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

		// 按名称查询是否存在同名渠道
		existing, err := model.GetChannelByName(name)
		if err != nil {
			result.Failed++
			result.Failures = append(result.Failures, ChannelImportFailure{Name: name, Reason: "查询渠道失败: " + err.Error()})
			continue
		}

		if existing != nil {
			// 同名渠道已存在：仅更新 JSON 中存在的字段，不清空其他字段
			if err := chApplyToExisting(existing, item); err != nil {
				result.Failed++
				result.Failures = append(result.Failures, ChannelImportFailure{Name: name, Reason: "更新失败: " + err.Error()})
				continue
			}
			result.Updated++
		} else {
			// 不存在同名渠道：新增
			newCh := &model.Channel{}
			if err := chApplyToNew(newCh, item); err != nil {
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
	return nil
}

// chApplyToExisting 将导入数据应用到已存在的渠道（精确更新，仅更新 JSON 中存在的字段）。
// 通过 GORM Select+Updates 确保只写入指定列，不影响其他列。
func chApplyToExisting(ch *model.Channel, item map[string]interface{}) error {
	cols := make([]string, 0, len(item))
	updates := &model.Channel{}

	if v, ok := item["discountRate"]; ok {
		f := chToFloat64(v)
		updates.PriceDiscountPercent = &f
		cols = append(cols, "price_discount_percent")
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

	if len(cols) == 0 {
		// 没有可更新的字段，直接跳过（不报错）
		return nil
	}

	// 使用精确列选择更新，确保只写入指定列
	return model.PartialUpdateChannelFields(ch.Id, cols, updates)
}

// chApplyToNew 将导入数据写入新渠道对象，用于新增场景。
func chApplyToNew(ch *model.Channel, item map[string]interface{}) error {
	name, ok := chGetStr(item, "name")
	if !ok || strings.TrimSpace(name) == "" {
		return fmt.Errorf("name 字段缺失")
	}
	ch.Name = strings.TrimSpace(name)

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

	return nil
}
