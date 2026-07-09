/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

package controller

import (
	"bytes"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

const inviteeModelDiscountExportContentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

var inviteeModelDiscountExportHeaders = []string{
	"模型 / 通道路径",
	"成本折扣",
	"平台加价折扣比例",
	"平台售价比例",
	"当前代理加价比例",
	"修改后售价比例",
}

// GetInviteeModelDiscounts 获取被邀请用户的模型折扣列表
// GET /api/distributor/invitee-model-discounts?invitee_id=xxx
func GetInviteeModelDiscounts(c *gin.Context) {
	userId := c.GetInt("id")
	u, err := model.GetUserById(userId, false)
	if err != nil || !model.UserIsDistributor(u) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "仅分销商可查看"})
		return
	}
	if !common.IsDistributorProfitShareMode() {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "当前站点未启用利润分成模式"})
		return
	}

	inviteeId, err := strconv.Atoi(c.Query("invitee_id"))
	if err != nil || inviteeId <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	items, _, err := model.GetInviteeModelDiscounts(userId, inviteeId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"items": items,
			"total": len(items),
		},
	})
}

func validateAdminInviteeModelDiscountTarget(distributorId, inviteeId int) string {
	if distributorId <= 0 || inviteeId <= 0 {
		return "参数错误"
	}
	if !common.IsDistributorProfitShareMode() {
		return "当前站点未启用利润分成模式"
	}
	dist, err := model.GetUserById(distributorId, false)
	if err != nil || dist == nil || !model.UserIsDistributor(dist) {
		return "用户不是分销商"
	}
	invitee, err := model.GetUserById(inviteeId, false)
	if err != nil || invitee == nil || invitee.InviterId != distributorId {
		return "该用户不是此分销商邀请的下级"
	}
	return ""
}

func parseAdminInviteeModelDiscountQuery(c *gin.Context) (int, int, string) {
	distributorId, err := strconv.Atoi(c.Query("distributor_id"))
	if err != nil || distributorId <= 0 {
		return 0, 0, "参数错误"
	}
	inviteeId, err := strconv.Atoi(c.Query("invitee_id"))
	if err != nil || inviteeId <= 0 {
		return 0, 0, "参数错误"
	}
	if msg := validateAdminInviteeModelDiscountTarget(distributorId, inviteeId); msg != "" {
		return 0, 0, msg
	}
	return distributorId, inviteeId, ""
}

// GetInviteeModelDiscountsAdmin 管理员查看某个代理下级用户的模型折扣列表。
// GET /api/distributor/admin/invitee-model-discounts?distributor_id=xxx&invitee_id=xxx
func GetInviteeModelDiscountsAdmin(c *gin.Context) {
	distributorId, inviteeId, msg := parseAdminInviteeModelDiscountQuery(c)
	if msg != "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": msg})
		return
	}

	items, _, err := model.GetInviteeModelDiscounts(distributorId, inviteeId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"items": items,
			"total": len(items),
		},
	})
}

// ExportInviteeModelDiscounts 导出被邀请用户的模型折扣价格表
// GET /api/distributor/invitee-model-discounts/export?invitee_id=xxx
func ExportInviteeModelDiscounts(c *gin.Context) {
	userId := c.GetInt("id")
	u, err := model.GetUserById(userId, false)
	if err != nil || !model.UserIsDistributor(u) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "仅分销商可查看"})
		return
	}
	if !common.IsDistributorProfitShareMode() {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "当前站点未启用利润分成模式"})
		return
	}

	inviteeId, err := strconv.Atoi(c.Query("invitee_id"))
	if err != nil || inviteeId <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	items, _, err := model.GetInviteeModelDiscounts(userId, inviteeId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	items = filterInviteeModelDiscountExportItems(items, c.Query("q"), c.Query("supplier_type"))

	data, err := buildInviteeModelDiscountExportWorkbook(items)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	filename := fmt.Sprintf("agent-model-discount-prices-%s.xlsx", time.Now().Format("20060102-150405"))
	c.Header("Content-Type", inviteeModelDiscountExportContentType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	c.Header("Content-Length", strconv.Itoa(len(data)))
	c.Data(http.StatusOK, inviteeModelDiscountExportContentType, data)
}

// ExportInviteeModelDiscountsAdmin 管理员导出某个代理下级用户的模型折扣价格表。
// GET /api/distributor/admin/invitee-model-discounts/export?distributor_id=xxx&invitee_id=xxx
func ExportInviteeModelDiscountsAdmin(c *gin.Context) {
	distributorId, inviteeId, msg := parseAdminInviteeModelDiscountQuery(c)
	if msg != "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": msg})
		return
	}

	items, _, err := model.GetInviteeModelDiscounts(distributorId, inviteeId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	items = filterInviteeModelDiscountExportItems(items, c.Query("q"), c.Query("supplier_type"))

	data, err := buildInviteeModelDiscountExportWorkbook(items)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	filename := fmt.Sprintf("agent-model-discount-prices-%s.xlsx", time.Now().Format("20060102-150405"))
	c.Header("Content-Type", inviteeModelDiscountExportContentType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	c.Header("Content-Length", strconv.Itoa(len(data)))
	c.Data(http.StatusOK, inviteeModelDiscountExportContentType, data)
}

func filterInviteeModelDiscountExportItems(items []model.InviteeModelMarkupDiscountRateItem, keyword, supplierType string) []model.InviteeModelMarkupDiscountRateItem {
	supplierType = strings.TrimSpace(supplierType)
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	if supplierType == "" && keyword == "" {
		return items
	}
	out := make([]model.InviteeModelMarkupDiscountRateItem, 0, len(items))
	for _, item := range items {
		if supplierType != "" && strings.TrimSpace(item.SupplierType) != supplierType {
			continue
		}
		if keyword != "" {
			searchText := strings.ToLower(fmt.Sprintf("%s %s %d %s %s",
				item.ModelName,
				item.ChannelPath,
				item.ChannelID,
				item.SupplierType,
				item.ChannelName,
			))
			if !strings.Contains(searchText, keyword) {
				continue
			}
		}
		out = append(out, item)
	}
	return out
}

func buildInviteeModelDiscountExportWorkbook(items []model.InviteeModelMarkupDiscountRateItem) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "价格表"
	defaultSheet := f.GetSheetName(0)
	if err := f.SetSheetName(defaultSheet, sheet); err != nil {
		return nil, err
	}

	for i, header := range inviteeModelDiscountExportHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err := f.SetCellValue(sheet, cell, header); err != nil {
			return nil, err
		}
	}

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"4472C4"}, Pattern: 1},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
			WrapText:   true,
		},
	})
	bodyStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{
			Vertical: "center",
			WrapText: true,
		},
	})
	centerStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
			WrapText:   true,
		},
	})

	_ = f.SetCellStyle(sheet, "A1", "F1", headerStyle)
	_ = f.SetColWidth(sheet, "A", "A", 46)
	_ = f.SetColWidth(sheet, "B", "B", 20)
	_ = f.SetColWidth(sheet, "C", "F", 24)
	_ = f.SetRowHeight(sheet, 1, 28)
	_ = f.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})

	for idx, item := range items {
		row := idx + 2
		values := []any{
			inviteeModelDiscountExportModelPath(item),
			formatInviteeModelDiscountMarkupRate(item.ChannelPriceDiscountPercent),
			formatInviteeModelDiscountMarkupRate(item.DefaultMarkupDiscountRate),
			formatInviteeModelDiscountSaleFormula(item, item.DefaultMarkupDiscountRate),
			formatInviteeModelDiscountMarkupRate(item.CurrentMarkupDiscountRate),
			formatInviteeModelDiscountSaleFormula(item, item.CurrentMarkupDiscountRate),
		}
		for col, value := range values {
			cell, _ := excelize.CoordinatesToCellName(col+1, row)
			if err := f.SetCellValue(sheet, cell, value); err != nil {
				return nil, err
			}
		}
		_ = f.SetRowHeight(sheet, row, 38)
		_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), bodyStyle)
		_ = f.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("F%d", row), centerStyle)
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func inviteeModelDiscountExportModelPath(item model.InviteeModelMarkupDiscountRateItem) string {
	modelName := strings.TrimSpace(item.ModelName)
	channelPath := strings.TrimSpace(item.ChannelPath)
	if channelPath == "" {
		return modelName
	}
	return modelName + "\n" + channelPath
}

func formatInviteeModelDiscountMarkupRate(value float64) string {
	if !isFiniteInviteeModelDiscountFloat(value) {
		return "-"
	}
	return fmt.Sprintf("%.1f%%", value)
}

func formatInviteeModelDiscountSaleFormula(item model.InviteeModelMarkupDiscountRateItem, markupRate float64) string {
	costRate := normalizeInviteeModelDiscountCostRate(item.ChannelPriceDiscountPercent)
	markupRate = max(0, markupRate)
	return fmt.Sprintf("%.1f%% + %.1f%% = %.1f%%", costRate, markupRate, costRate+markupRate)
}

func normalizeInviteeModelDiscountCostRate(value float64) float64 {
	if !isFiniteInviteeModelDiscountFloat(value) {
		return 100
	}
	if value < 0 {
		return 0
	}
	return math.Round(value*10) / 10
}

func isFiniteInviteeModelDiscountFloat(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

// PutInviteeModelDiscounts 更新被邀请用户的模型折扣配置
// PUT /api/distributor/invitee-model-discounts
type putInviteeModelDiscountsRequest struct {
	InviteeId int                                          `json:"invitee_id"`
	Discounts []model.ModelMarkupDiscountRateUpdateRequest `json:"discounts"`
}

func PutInviteeModelDiscounts(c *gin.Context) {
	userId := c.GetInt("id")
	u, err := model.GetUserById(userId, false)
	if err != nil || !model.UserIsDistributor(u) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "仅分销商可操作"})
		return
	}
	if !common.IsDistributorProfitShareMode() {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "当前站点未启用利润分成模式"})
		return
	}

	var req putInviteeModelDiscountsRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的请求"})
		return
	}

	if req.InviteeId <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "参数错误"})
		return
	}

	if err := model.UpdateInviteeModelDiscounts(userId, req.InviteeId, req.Discounts); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}

func GetDistributorModelDiscountTemplate(c *gin.Context) {
	userId := c.GetInt("id")
	u, err := model.GetUserById(userId, false)
	if err != nil || !model.UserIsDistributor(u) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "仅分销商可查看"})
		return
	}
	if !common.IsDistributorProfitShareMode() {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "当前站点未启用利润分成模式"})
		return
	}

	items, inviteeCount, autoApplyNewInvitees, err := model.GetDistributorModelDiscountTemplate(userId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"items":                   items,
			"total":                   len(items),
			"invitee_count":           inviteeCount,
			"auto_apply_new_invitees": autoApplyNewInvitees,
		},
	})
}

type putDistributorModelDiscountTemplateRequest struct {
	Discounts            []model.ModelMarkupDiscountRateUpdateRequest `json:"discounts"`
	AutoApplyNewInvitees *bool                                        `json:"auto_apply_new_invitees"`
}

func PutDistributorModelDiscountTemplate(c *gin.Context) {
	userId := c.GetInt("id")
	u, err := model.GetUserById(userId, false)
	if err != nil || !model.UserIsDistributor(u) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "仅分销商可操作"})
		return
	}
	if !common.IsDistributorProfitShareMode() {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "当前站点未启用利润分成模式"})
		return
	}
	var req putDistributorModelDiscountTemplateRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的请求"})
		return
	}
	if err := model.UpdateDistributorModelDiscountTemplate(userId, req.Discounts, req.AutoApplyNewInvitees); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}

type batchInviteeModelDiscountsRequest struct {
	Action     string `json:"action"`
	Scope      string `json:"scope"`
	InviteeIds []int  `json:"invitee_ids"`
}

func PostBatchInviteeModelDiscounts(c *gin.Context) {
	userId := c.GetInt("id")
	u, err := model.GetUserById(userId, false)
	if err != nil || !model.UserIsDistributor(u) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "仅分销商可操作"})
		return
	}
	if !common.IsDistributorProfitShareMode() {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "当前站点未启用利润分成模式"})
		return
	}
	var req batchInviteeModelDiscountsRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的请求"})
		return
	}
	all := strings.EqualFold(strings.TrimSpace(req.Scope), "all")
	action := strings.TrimSpace(req.Action)
	var affected int
	switch action {
	case "apply_template":
		affected, err = model.ApplyDistributorModelDiscountTemplate(userId, all, req.InviteeIds)
	case "reset_default":
		affected, err = model.ResetInviteeModelDiscountsBatch(userId, all, req.InviteeIds)
	default:
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "未知操作"})
		return
	}
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"affected_count": affected,
		},
	})
}

type putInviteeModelDiscountsAdminRequest struct {
	DistributorId int                                          `json:"distributor_id"`
	InviteeId     int                                          `json:"invitee_id"`
	Discounts     []model.ModelMarkupDiscountRateUpdateRequest `json:"discounts"`
}

// PutInviteeModelDiscountsAdmin 管理员为某个代理下级用户更新模型折扣配置。
// PUT /api/distributor/admin/invitee-model-discounts
func PutInviteeModelDiscountsAdmin(c *gin.Context) {
	var req putInviteeModelDiscountsAdminRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的请求"})
		return
	}
	if msg := validateAdminInviteeModelDiscountTarget(req.DistributorId, req.InviteeId); msg != "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": msg})
		return
	}
	if err := model.UpdateInviteeModelDiscounts(req.DistributorId, req.InviteeId, req.Discounts); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}
