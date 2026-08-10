package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ─── 导出/导入 DTO 定义 ──────────────────────────────────────────────────────

// ModelExportPayload 模型元数据导出响应结构
type ModelExportPayload struct {
	Version     string                      `json:"version"`
	ExportTime  string                      `json:"exportTime"`
	Vendors     []VendorExportItem          `json:"vendors"`
	Models      []ModelExportItem           `json:"models"`
	ChannelDocs []ChannelModelDocExportItem `json:"channel_docs"`
}

// VendorExportItem 模型类型（供应商）导出项
type VendorExportItem struct {
	Name        string `json:"name"`
	NameEn      string `json:"name_en,omitempty"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

// ModelExportItem 模型数据导出项
type ModelExportItem struct {
	ModelName       string  `json:"model_name"`
	NameRule        int     `json:"name_rule"`
	Icon            string  `json:"icon"`
	Description     string  `json:"description"`
	DescriptionEn   string  `json:"description_en,omitempty"`
	DocIntroduction *string `json:"doc_introduction,omitempty"`
	ApiDocs         *string `json:"api_docs,omitempty"`
	Tags            string  `json:"tags"`
	Vendor          string  `json:"vendor"`
	Endpoints       string  `json:"endpoints"`
	SyncOfficial    int     `json:"sync_official"`
	Status          int     `json:"status"`
}

// ChannelModelDocExportItem 渠道模型文档导出项。渠道 ID 不跨环境复用，使用 sync_key 重新绑定。
type ChannelModelDocExportItem struct {
	ModelName         string  `json:"model_name"`
	ChannelSyncKey    string  `json:"channel_sync_key,omitempty"`
	ChannelName       string  `json:"channel_name"`
	DocIntroduction   *string `json:"doc_introduction,omitempty"`
	ApiDocs           *string `json:"api_docs,omitempty"`
	ApiDocsMarkdown   *string `json:"api_docs_markdown,omitempty"`
	ApiDocsMarkdownEn *string `json:"api_docs_markdown_en,omitempty"`
}

// ModelImportRequest 模型导入请求结构
type ModelImportRequest struct {
	Version     string                      `json:"version"`
	ExportTime  string                      `json:"exportTime"`
	Vendors     []VendorExportItem          `json:"vendors"`
	Models      []ModelExportItem           `json:"models"`
	ChannelDocs []ChannelModelDocExportItem `json:"channel_docs"`
}

// ModelImportResult 导入结果统计
type ModelImportResult struct {
	VendorsAdded       int                      `json:"vendors_added"`
	VendorsUpdated     int                      `json:"vendors_updated"`
	ModelsAdded        int                      `json:"models_added"`
	ModelsUpdated      int                      `json:"models_updated"`
	ModelsFailed       int                      `json:"models_failed"`
	VendorsFailed      int                      `json:"vendors_failed"`
	ChannelDocsAdded   int                      `json:"channel_docs_added"`
	ChannelDocsUpdated int                      `json:"channel_docs_updated"`
	ChannelDocsSkipped int                      `json:"channel_docs_skipped"`
	ChannelDocsFailed  int                      `json:"channel_docs_failed"`
	Failures           []ModelImportFailureItem `json:"failures"`
	SkippedDocs        []ModelImportFailureItem `json:"skipped_docs"`
}

// ModelImportFailureItem 单条导入失败详情
type ModelImportFailureItem struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
	Type   string `json:"type"` // "vendor" | "model" | "channel_doc"
}

func stringPtr(s string) *string {
	return &s
}

// ─── 导出接口 ──────────────────────────────────────────────────────────────

// ExportModelsMeta 导出模型元数据（模型类型 + 模型数据）
// GET /api/models/export
func ExportModelsMeta(c *gin.Context) {
	// 导出所有模型类型（供应商）
	var vendors []*model.Vendor
	if err := model.DB.Find(&vendors).Error; err != nil {
		common.ApiError(c, err)
		return
	}

	vendorItems := make([]VendorExportItem, 0, len(vendors))
	for _, v := range vendors {
		vendorItems = append(vendorItems, VendorExportItem{
			Name:        v.Name,
			NameEn:      v.NameEn,
			Description: v.Description,
			Icon:        v.Icon,
		})
	}

	// 导出所有模型数据
	var models []*model.Model
	if err := model.DB.Order("id ASC").Find(&models).Error; err != nil {
		common.ApiError(c, err)
		return
	}

	// 构建 vendorID -> vendorName 映射，用于将 vendor_id 转为 vendor name
	vendorNameMap := make(map[int]string)
	for _, v := range vendors {
		vendorNameMap[v.Id] = v.Name
	}

	modelItems := make([]ModelExportItem, 0, len(models))
	for _, m := range models {
		vendorName := vendorNameMap[m.VendorID]
		modelItems = append(modelItems, ModelExportItem{
			ModelName:       m.ModelName,
			NameRule:        m.NameRule,
			Icon:            m.Icon,
			Description:     m.Description,
			DescriptionEn:   m.DescriptionEn,
			DocIntroduction: stringPtr(m.DocIntroduction),
			ApiDocs:         stringPtr(m.ApiDocs),
			Tags:            m.Tags,
			Vendor:          vendorName,
			Endpoints:       m.Endpoints,
			SyncOfficial:    m.SyncOfficial,
			Status:          m.Status,
		})
	}

	channelDocItems, err := buildChannelModelDocExportItems()
	if err != nil {
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": ModelExportPayload{
			Version:     "1.1",
			ExportTime:  time.Now().UTC().Format(time.RFC3339),
			Vendors:     vendorItems,
			Models:      modelItems,
			ChannelDocs: channelDocItems,
		},
	})
}

func buildChannelModelDocExportItems() ([]ChannelModelDocExportItem, error) {
	type channelModelDocExportRow struct {
		model.ChannelModelDoc
		ChannelSyncKey string `gorm:"column:channel_sync_key"`
		ChannelName    string `gorm:"column:channel_name"`
	}

	var rows []channelModelDocExportRow
	err := model.DB.Table("channel_model_docs").
		Select("channel_model_docs.*, channels.sync_key AS channel_sync_key, channels.name AS channel_name").
		Joins("JOIN channels ON channels.id = channel_model_docs.channel_id").
		Order("channel_model_docs.id ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	items := make([]ChannelModelDocExportItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, ChannelModelDocExportItem{
			ModelName:         row.ModelName,
			ChannelSyncKey:    row.ChannelSyncKey,
			ChannelName:       row.ChannelName,
			DocIntroduction:   stringPtr(row.DocIntroduction),
			ApiDocs:           stringPtr(row.ApiDocs),
			ApiDocsMarkdown:   stringPtr(row.ApiDocsMarkdown),
			ApiDocsMarkdownEn: stringPtr(row.ApiDocsMarkdownEn),
		})
	}
	return items, nil
}

// ─── 导入接口 ──────────────────────────────────────────────────────────────

// ImportModelsMeta 导入模型元数据（增量：名称匹配覆盖，不存在新增，不删除）
// POST /api/models/import
func ImportModelsMeta(c *gin.Context) {
	var req ModelImportRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "JSON 格式错误，请上传合法的导出文件"})
		return
	}

	result := &ModelImportResult{
		Failures:    []ModelImportFailureItem{},
		SkippedDocs: []ModelImportFailureItem{},
	}

	// 1. 导入模型类型（供应商）
	if len(req.Vendors) > 0 {
		for _, vItem := range req.Vendors {
			vName := strings.TrimSpace(vItem.Name)
			if vName == "" {
				result.VendorsFailed++
				result.Failures = append(result.Failures, ModelImportFailureItem{
					Name:   "(未知)",
					Reason: "供应商名称不能为空",
					Type:   "vendor",
				})
				continue
			}

			// 按名称查找是否已存在
			var existing model.Vendor
			err := model.DB.Where("name = ?", vName).First(&existing).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				// 非常见错误
				result.VendorsFailed++
				result.Failures = append(result.Failures, ModelImportFailureItem{
					Name:   vName,
					Reason: "查询供应商失败: " + err.Error(),
					Type:   "vendor",
				})
				continue
			}

			if existing.Id > 0 {
				// 已存在：仅更新描述、英文名称和图标
				updates := map[string]interface{}{}
				if vItem.Description != "" {
					updates["description"] = vItem.Description
				}
				if vItem.NameEn != "" {
					updates["name_en"] = vItem.NameEn
				}
				if vItem.Icon != "" {
					updates["icon"] = vItem.Icon
				}
				if len(updates) > 0 {
					updates["updated_time"] = common.GetTimestamp()
					if err := model.DB.Model(&model.Vendor{}).Where("id = ?", existing.Id).Updates(updates).Error; err != nil {
						result.VendorsFailed++
						result.Failures = append(result.Failures, ModelImportFailureItem{
							Name:   vName,
							Reason: "更新供应商失败: " + err.Error(),
							Type:   "vendor",
						})
						continue
					}
				}
				result.VendorsUpdated++
			} else {
				// 不存在：新增
				newVendor := &model.Vendor{
					Name:        vName,
					NameEn:      vItem.NameEn,
					Description: vItem.Description,
					Icon:        vItem.Icon,
					Status:      1,
				}
				if err := newVendor.Insert(); err != nil {
					result.VendorsFailed++
					result.Failures = append(result.Failures, ModelImportFailureItem{
						Name:   vName,
						Reason: "创建供应商失败: " + err.Error(),
						Type:   "vendor",
					})
					continue
				}
				result.VendorsAdded++
			}
		}
	}

	// 2. 导入模型数据
	if len(req.Models) > 0 {
		// 刷新 vendorName -> vendorID 映射（可能刚新增了供应商）
		vendorIDMap, err := buildVendorNameIDMap()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "构建供应商映射失败: " + err.Error()})
			return
		}

		for _, mItem := range req.Models {
			mName := strings.TrimSpace(mItem.ModelName)
			if mName == "" {
				result.ModelsFailed++
				result.Failures = append(result.Failures, ModelImportFailureItem{
					Name:   "(未知)",
					Reason: "模型名称不能为空",
					Type:   "model",
				})
				continue
			}

			// 按名称查找是否已存在
			var existing model.Model
			err := model.DB.Where("model_name = ?", mName).First(&existing).Error
			if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				result.ModelsFailed++
				result.Failures = append(result.Failures, ModelImportFailureItem{
					Name:   mName,
					Reason: "查询模型失败: " + err.Error(),
					Type:   "model",
				})
				continue
			}

			// 解析 vendor name -> vendor_id
			vendorID := 0
			if mItem.Vendor != "" {
				if id, ok := vendorIDMap[mItem.Vendor]; ok {
					vendorID = id
				}
			}

			if existing.Id > 0 {
				// 已存在：覆盖更新指定字段
				updates := map[string]interface{}{
					"name_rule":      mItem.NameRule,
					"icon":           mItem.Icon,
					"description":    mItem.Description,
					"description_en": mItem.DescriptionEn,
					"tags":           mItem.Tags,
					"endpoints":      mItem.Endpoints,
					"sync_official":  mItem.SyncOfficial,
					"status":         mItem.Status,
					"updated_time":   common.GetTimestamp(),
				}
				if mItem.DocIntroduction != nil {
					updates["doc_introduction"] = *mItem.DocIntroduction
				}
				if mItem.ApiDocs != nil {
					updates["api_docs"] = *mItem.ApiDocs
				}
				if vendorID > 0 {
					updates["vendor_id"] = vendorID
				}
				if err := model.DB.Model(&model.Model{}).Where("id = ?", existing.Id).Updates(updates).Error; err != nil {
					result.ModelsFailed++
					result.Failures = append(result.Failures, ModelImportFailureItem{
						Name:   mName,
						Reason: "更新模型失败: " + err.Error(),
						Type:   "model",
					})
					continue
				}
				result.ModelsUpdated++
			} else {
				// 不存在：新增
				newModel := &model.Model{
					ModelName:     mName,
					NameRule:      mItem.NameRule,
					Icon:          mItem.Icon,
					Description:   mItem.Description,
					DescriptionEn: mItem.DescriptionEn,
					Tags:          mItem.Tags,
					VendorID:      vendorID,
					Endpoints:     mItem.Endpoints,
					SyncOfficial:  mItem.SyncOfficial,
					Status:        mItem.Status,
				}
				if mItem.DocIntroduction != nil {
					newModel.DocIntroduction = *mItem.DocIntroduction
				}
				if mItem.ApiDocs != nil {
					newModel.ApiDocs = *mItem.ApiDocs
				}
				if err := newModel.Insert(); err != nil {
					result.ModelsFailed++
					result.Failures = append(result.Failures, ModelImportFailureItem{
						Name:   mName,
						Reason: "创建模型失败: " + err.Error(),
						Type:   "model",
					})
					continue
				}
				result.ModelsAdded++
			}
		}
	}

	importChannelModelDocs(req.ChannelDocs, result)

	// 刷新定价缓存
	model.RefreshPricing()
	if _, err := model.SyncModelTagsFromModels(); err != nil {
		common.SysError("sync model tags after import: " + err.Error())
	}

	common.SysLog(fmt.Sprintf("模型导入完成：供应商新增 %d/更新 %d/失败 %d，模型新增 %d/更新 %d/失败 %d，渠道文档新增 %d/更新 %d/跳过 %d/失败 %d",
		result.VendorsAdded, result.VendorsUpdated, result.VendorsFailed,
		result.ModelsAdded, result.ModelsUpdated, result.ModelsFailed,
		result.ChannelDocsAdded, result.ChannelDocsUpdated, result.ChannelDocsSkipped, result.ChannelDocsFailed))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "模型导入完成",
		"data":    result,
	})
}

func importChannelModelDocs(items []ChannelModelDocExportItem, result *ModelImportResult) {
	for _, item := range items {
		modelName := strings.TrimSpace(item.ModelName)
		channelName := strings.TrimSpace(item.ChannelName)
		displayName := modelName
		if channelName != "" {
			displayName = channelName + " / " + modelName
		}
		if modelName == "" {
			appendChannelDocFailure(result, displayName, "模型名称不能为空")
			continue
		}

		channel, err := resolveChannelModelDocImportChannel(item)
		if err != nil {
			appendChannelDocFailure(result, displayName, "查询渠道失败: "+err.Error())
			continue
		}
		if channel == nil {
			reason := "目标渠道不存在"
			if strings.TrimSpace(item.ChannelSyncKey) != "" {
				reason = "未找到同步编号为 " + strings.TrimSpace(item.ChannelSyncKey) + " 的渠道"
			}
			appendChannelDocSkip(result, displayName, reason)
			continue
		}

		var abilityCount int64
		if err := model.DB.Model(&model.Ability{}).
			Where("channel_id = ? AND model = ?", channel.Id, modelName).
			Count(&abilityCount).Error; err != nil {
			appendChannelDocFailure(result, displayName, "校验渠道模型失败: "+err.Error())
			continue
		}
		if abilityCount == 0 {
			appendChannelDocSkip(result, displayName, "目标渠道未提供该模型")
			continue
		}

		var doc model.ChannelModelDoc
		err = model.DB.Where("channel_id = ? AND model_name = ?", channel.Id, modelName).First(&doc).Error
		exists := err == nil
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			appendChannelDocFailure(result, displayName, "查询已有文档失败: "+err.Error())
			continue
		}
		if !exists {
			doc = model.ChannelModelDoc{ChannelID: channel.Id, ModelName: modelName}
		}
		applyChannelModelDocImportFields(&doc, item)
		if err := model.UpsertChannelModelDoc(&doc); err != nil {
			appendChannelDocFailure(result, displayName, "保存文档失败: "+err.Error())
			continue
		}
		if exists {
			result.ChannelDocsUpdated++
		} else {
			result.ChannelDocsAdded++
		}
	}
}

func resolveChannelModelDocImportChannel(item ChannelModelDocExportItem) (*model.Channel, error) {
	syncKey := model.NormalizeChannelSyncKey(item.ChannelSyncKey)
	if syncKey != "" {
		if !model.IsValidChannelSyncKey(syncKey) {
			return nil, fmt.Errorf("渠道同步编号格式无效")
		}
		return model.GetChannelBySyncKey(syncKey)
	}
	channelName := strings.TrimSpace(item.ChannelName)
	if channelName == "" {
		return nil, nil
	}
	return model.GetChannelByName(channelName)
}

func applyChannelModelDocImportFields(doc *model.ChannelModelDoc, item ChannelModelDocExportItem) {
	if item.DocIntroduction != nil {
		doc.DocIntroduction = *item.DocIntroduction
	}
	if item.ApiDocs != nil {
		doc.ApiDocs = *item.ApiDocs
	}
	if item.ApiDocsMarkdown != nil {
		doc.ApiDocsMarkdown = *item.ApiDocsMarkdown
	}
	if item.ApiDocsMarkdownEn != nil {
		doc.ApiDocsMarkdownEn = *item.ApiDocsMarkdownEn
	}
}

func appendChannelDocFailure(result *ModelImportResult, name, reason string) {
	result.ChannelDocsFailed++
	result.Failures = append(result.Failures, ModelImportFailureItem{
		Name: name, Reason: reason, Type: "channel_doc",
	})
}

func appendChannelDocSkip(result *ModelImportResult, name, reason string) {
	result.ChannelDocsSkipped++
	result.SkippedDocs = append(result.SkippedDocs, ModelImportFailureItem{
		Name: name, Reason: reason, Type: "channel_doc",
	})
}

// buildVendorNameIDMap 构建供应商名称到 ID 的映射
func buildVendorNameIDMap() (map[string]int, error) {
	var vendors []*model.Vendor
	if err := model.DB.Find(&vendors).Error; err != nil {
		return nil, err
	}
	m := make(map[string]int, len(vendors))
	for _, v := range vendors {
		m[v.Name] = v.Id
	}
	return m, nil
}
