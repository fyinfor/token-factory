package controller

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

// ─── 导出/导入共用的数据结构 ──────────────────────────────────────────────────

// PriceExportModelMaps 一组模型定价映射（字段名与全局 Option key 保持一致）。
type PriceExportModelMaps struct {
	ModelPrice           map[string]float64 `json:"ModelPrice"`
	ModelRatio           map[string]float64 `json:"ModelRatio"`
	CompletionRatio      map[string]float64 `json:"CompletionRatio"`
	CacheRatio           map[string]float64 `json:"CacheRatio"`
	CreateCacheRatio     map[string]float64 `json:"CreateCacheRatio"`
	ImageRatio           map[string]float64 `json:"ImageRatio"`
	AudioRatio           map[string]float64 `json:"AudioRatio"`
	AudioCompletionRatio map[string]float64 `json:"AudioCompletionRatio"`
}

// PriceExportChannelEntry 单渠道价格导出/导入条目（用 channel_name 标识，不含 ID）。
type PriceExportChannelEntry struct {
	ChannelName string               `json:"channel_name"`
	Models      PriceExportModelMaps `json:"models"`
}

// PriceExportData 完整导出结构（可直接用于后续导入）。
type PriceExportData struct {
	GlobalPrices PriceExportModelMaps      `json:"global_prices"`
	Channels     []PriceExportChannelEntry `json:"channels"`
}

// PriceImportChannelStat 单渠道导入统计。
type PriceImportChannelStat struct {
	ChannelName string `json:"channel_name"`
	Updated     int    `json:"updated"`
	Added       int    `json:"added"`
}

// PriceImportResult 导入结果统计（返回给前端展示）。
type PriceImportResult struct {
	GlobalUpdated   int                      `json:"global_updated"`
	GlobalAdded     int                      `json:"global_added"`
	ChannelStats    []PriceImportChannelStat `json:"channel_stats"`
	SkippedChannels []string                 `json:"skipped_channels"`
}

// ─── 内部工具函数 ──────────────────────────────────────────────────────────────

// readOptionStr 从内存 OptionMap 安全读取字符串值（只读锁）。
func readOptionStr(key string) string {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	return common.Interface2String(common.OptionMap[key])
}

// parseFloatMapSafe 将 JSON 字符串解析为 map[string]float64，失败时返回空 map。
func parseFloatMapSafe(raw string) map[string]float64 {
	out := map[string]float64{}
	if strings.TrimSpace(raw) == "" {
		return out
	}
	_ = common.UnmarshalJsonStr(raw, &out)
	return out
}

// parseNestedFloatMapSafe 将 JSON 字符串解析为 map[string]map[string]float64，失败时返回空 map。
func parseNestedFloatMapSafe(raw string) map[string]map[string]float64 {
	out := map[string]map[string]float64{}
	if strings.TrimSpace(raw) == "" {
		return out
	}
	_ = common.UnmarshalJsonStr(raw, &out)
	return out
}

// safeFloatMap 确保返回非 nil 的 map。
func safeFloatMap(m map[string]float64) map[string]float64 {
	if m == nil {
		return map[string]float64{}
	}
	return m
}

// marshalToJSON 将值序列化为 JSON 字符串，失败返回 "{}"。
func marshalToJSON(v any) string {
	b, err := common.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// mergeFloatMapCounting 将 src 中的键值增量合并到 dst（不删除 dst 中已有键），返回 added/updated 数量。
func mergeFloatMapCounting(dst, src map[string]float64) (added, updated int) {
	for k, v := range src {
		if _, exists := dst[k]; exists {
			dst[k] = v
			updated++
		} else {
			dst[k] = v
			added++
		}
	}
	return
}

// isModelMapsEmpty 判断 PriceExportModelMaps 是否所有子 map 均为空。
func isModelMapsEmpty(m PriceExportModelMaps) bool {
	return len(m.ModelPrice) == 0 &&
		len(m.ModelRatio) == 0 &&
		len(m.CompletionRatio) == 0 &&
		len(m.CacheRatio) == 0 &&
		len(m.CreateCacheRatio) == 0 &&
		len(m.ImageRatio) == 0 &&
		len(m.AudioRatio) == 0 &&
		len(m.AudioCompletionRatio) == 0
}

// globalPriceFields 全局价格 Option 键与 PriceExportModelMaps 字段的绑定关系。
var globalPriceFields = []struct {
	optionKey string
	getField  func(*PriceExportModelMaps) map[string]float64
}{
	{"ModelPrice", func(m *PriceExportModelMaps) map[string]float64 { return m.ModelPrice }},
	{"ModelRatio", func(m *PriceExportModelMaps) map[string]float64 { return m.ModelRatio }},
	{"CompletionRatio", func(m *PriceExportModelMaps) map[string]float64 { return m.CompletionRatio }},
	{"CacheRatio", func(m *PriceExportModelMaps) map[string]float64 { return m.CacheRatio }},
	{"CreateCacheRatio", func(m *PriceExportModelMaps) map[string]float64 { return m.CreateCacheRatio }},
	{"ImageRatio", func(m *PriceExportModelMaps) map[string]float64 { return m.ImageRatio }},
	{"AudioRatio", func(m *PriceExportModelMaps) map[string]float64 { return m.AudioRatio }},
	{"AudioCompletionRatio", func(m *PriceExportModelMaps) map[string]float64 { return m.AudioCompletionRatio }},
}

// channelPriceFields 渠道价格 Option 键（Channel 前缀）与 PriceExportModelMaps 字段的绑定关系。
var channelPriceFields = []struct {
	optionKey string
	getField  func(*PriceExportModelMaps) map[string]float64
}{
	{"ChannelModelPrice", func(m *PriceExportModelMaps) map[string]float64 { return m.ModelPrice }},
	{"ChannelModelRatio", func(m *PriceExportModelMaps) map[string]float64 { return m.ModelRatio }},
	{"ChannelCompletionRatio", func(m *PriceExportModelMaps) map[string]float64 { return m.CompletionRatio }},
	{"ChannelCacheRatio", func(m *PriceExportModelMaps) map[string]float64 { return m.CacheRatio }},
	{"ChannelCreateCacheRatio", func(m *PriceExportModelMaps) map[string]float64 { return m.CreateCacheRatio }},
	{"ChannelImageRatio", func(m *PriceExportModelMaps) map[string]float64 { return m.ImageRatio }},
	{"ChannelAudioRatio", func(m *PriceExportModelMaps) map[string]float64 { return m.AudioRatio }},
	{"ChannelAudioCompletionRatio", func(m *PriceExportModelMaps) map[string]float64 { return m.AudioCompletionRatio }},
}

// ─── 导出 ─────────────────────────────────────────────────────────────────────

// ExportPrices 导出全局及各渠道模型价格配置。
// GET /api/admin/price/export
func ExportPrices(c *gin.Context) {
	// 读取全局价格
	globalPrices := PriceExportModelMaps{
		ModelPrice:           parseFloatMapSafe(readOptionStr("ModelPrice")),
		ModelRatio:           parseFloatMapSafe(readOptionStr("ModelRatio")),
		CompletionRatio:      parseFloatMapSafe(readOptionStr("CompletionRatio")),
		CacheRatio:           parseFloatMapSafe(readOptionStr("CacheRatio")),
		CreateCacheRatio:     parseFloatMapSafe(readOptionStr("CreateCacheRatio")),
		ImageRatio:           parseFloatMapSafe(readOptionStr("ImageRatio")),
		AudioRatio:           parseFloatMapSafe(readOptionStr("AudioRatio")),
		AudioCompletionRatio: parseFloatMapSafe(readOptionStr("AudioCompletionRatio")),
	}

	// 读取渠道维度价格（结构：channel_id(str) → model_name → value）
	chModelPrice := parseNestedFloatMapSafe(readOptionStr("ChannelModelPrice"))
	chModelRatio := parseNestedFloatMapSafe(readOptionStr("ChannelModelRatio"))
	chCompletionRatio := parseNestedFloatMapSafe(readOptionStr("ChannelCompletionRatio"))
	chCacheRatio := parseNestedFloatMapSafe(readOptionStr("ChannelCacheRatio"))
	chCreateCacheRatio := parseNestedFloatMapSafe(readOptionStr("ChannelCreateCacheRatio"))
	chImageRatio := parseNestedFloatMapSafe(readOptionStr("ChannelImageRatio"))
	chAudioRatio := parseNestedFloatMapSafe(readOptionStr("ChannelAudioRatio"))
	chAudioCompletionRatio := parseNestedFloatMapSafe(readOptionStr("ChannelAudioCompletionRatio"))

	// 收集所有出现过的 channel_id（字符串形式）
	channelIDSet := map[string]struct{}{}
	for _, nm := range []map[string]map[string]float64{
		chModelPrice, chModelRatio, chCompletionRatio, chCacheRatio,
		chCreateCacheRatio, chImageRatio, chAudioRatio, chAudioCompletionRatio,
	} {
		for id := range nm {
			channelIDSet[id] = struct{}{}
		}
	}

	// 查询 channel_id → channel_name 映射
	idNameMap, err := model.GetChannelIdNameMap()
	if err != nil {
		common.ApiError(c, err)
		return
	}

	// 构建渠道导出条目（每个 channel_id 对应一个条目，避免同名渠道数据混淆）
	channelEntries := make([]PriceExportChannelEntry, 0, len(channelIDSet))
	for idStr := range channelIDSet {
		name, ok := idNameMap[idStr]
		if !ok {
			// 渠道已删除：保留占位符，导入时会被自动跳过
			name = fmt.Sprintf("__deleted__channel_id_%s", idStr)
		}
		channelEntries = append(channelEntries, PriceExportChannelEntry{
			ChannelName: name,
			Models: PriceExportModelMaps{
				ModelPrice:           safeFloatMap(chModelPrice[idStr]),
				ModelRatio:           safeFloatMap(chModelRatio[idStr]),
				CompletionRatio:      safeFloatMap(chCompletionRatio[idStr]),
				CacheRatio:           safeFloatMap(chCacheRatio[idStr]),
				CreateCacheRatio:     safeFloatMap(chCreateCacheRatio[idStr]),
				ImageRatio:           safeFloatMap(chImageRatio[idStr]),
				AudioRatio:           safeFloatMap(chAudioRatio[idStr]),
				AudioCompletionRatio: safeFloatMap(chAudioCompletionRatio[idStr]),
			},
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": PriceExportData{
			GlobalPrices: globalPrices,
			Channels:     channelEntries,
		},
	})
}

// ─── 导入 ─────────────────────────────────────────────────────────────────────

// ImportPrices 导入价格配置（增量同步，仅新增/更新，不删除已有数据）。
// POST /api/admin/price/import
func ImportPrices(c *gin.Context) {
	var payload PriceExportData
	if err := common.DecodeJson(c.Request.Body, &payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "JSON 格式错误，请上传合法的导出文件",
		})
		return
	}

	// 防止空数据写入
	globalEmpty := isModelMapsEmpty(payload.GlobalPrices)
	if globalEmpty && len(payload.Channels) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "导入文件中未包含任何价格数据，已取消导入",
		})
		return
	}

	result := &PriceImportResult{
		ChannelStats:    []PriceImportChannelStat{},
		SkippedChannels: []string{},
	}

	// ── 1. 同步全局模型价格 ────────────────────────────────────────────────────
	if !globalEmpty {
		added, updated, err := doSyncGlobalPrices(payload.GlobalPrices)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": fmt.Sprintf("全局价格同步失败: %v", err),
			})
			return
		}
		result.GlobalAdded = added
		result.GlobalUpdated = updated
	}

	// ── 2. 同步渠道模型价格 ────────────────────────────────────────────────────
	for _, chEntry := range payload.Channels {
		chName := strings.TrimSpace(chEntry.ChannelName)
		if chName == "" {
			continue
		}
		// 跳过已删除渠道的占位符（导出时自动生成的前缀）
		if strings.HasPrefix(chName, "__deleted__channel_id_") {
			result.SkippedChannels = append(result.SkippedChannels, chName)
			continue
		}
		// 渠道模型数据为空时跳过
		if isModelMapsEmpty(chEntry.Models) {
			continue
		}

		channelIDs, err := model.GetChannelIDsByName(chName)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": fmt.Sprintf("查询渠道 '%s' 失败: %v", chName, err),
			})
			return
		}
		if len(channelIDs) == 0 {
			result.SkippedChannels = append(result.SkippedChannels, chName)
			continue
		}

		// 对所有同名渠道执行增量同步
		stat := PriceImportChannelStat{ChannelName: chName}
		for _, channelID := range channelIDs {
			added, updated, err := doSyncChannelPrices(channelID, chEntry.Models)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"message": fmt.Sprintf("渠道 '%s'(id=%d) 价格同步失败: %v", chName, channelID, err),
				})
				return
			}
			stat.Added += added
			stat.Updated += updated
		}
		result.ChannelStats = append(result.ChannelStats, stat)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "价格导入成功",
		"data":    result,
	})
}

// ─── 内部同步实现 ──────────────────────────────────────────────────────────────

// doSyncGlobalPrices 增量合并全局模型价格到 Options，返回 (added, updated, error)。
// 逐 Option 键处理：读取当前值 → 合并 → 通过 model.UpdateOption 写回（同时刷新内存缓存与 ratio_setting）。
func doSyncGlobalPrices(incoming PriceExportModelMaps) (totalAdded, totalUpdated int, err error) {
	for _, field := range globalPriceFields {
		src := field.getField(&incoming)
		if len(src) == 0 {
			continue
		}
		current := parseFloatMapSafe(readOptionStr(field.optionKey))
		added, updated := mergeFloatMapCounting(current, src)
		totalAdded += added
		totalUpdated += updated

		if err = model.UpdateOption(field.optionKey, marshalToJSON(current)); err != nil {
			return 0, 0, fmt.Errorf("写入 Option[%s] 失败: %w", field.optionKey, err)
		}
	}
	return
}

// doSyncChannelPrices 增量合并单渠道模型价格到对应的渠道 Option，返回 (added, updated, error)。
// 每个渠道 Option 的 value 为 map[channel_id(str)]map[model_name]float64 的嵌套结构。
func doSyncChannelPrices(channelID int, incoming PriceExportModelMaps) (totalAdded, totalUpdated int, err error) {
	idStr := fmt.Sprintf("%d", channelID)

	for _, field := range channelPriceFields {
		src := field.getField(&incoming)
		if len(src) == 0 {
			continue
		}
		// 读取整个渠道 Option 的当前嵌套 map
		fullMap := parseNestedFloatMapSafe(readOptionStr(field.optionKey))
		if fullMap[idStr] == nil {
			fullMap[idStr] = map[string]float64{}
		}
		added, updated := mergeFloatMapCounting(fullMap[idStr], src)
		totalAdded += added
		totalUpdated += updated

		if err = model.UpdateOption(field.optionKey, marshalToJSON(fullMap)); err != nil {
			return 0, 0, fmt.Errorf("写入 Option[%s] 失败: %w", field.optionKey, err)
		}
	}
	return
}
