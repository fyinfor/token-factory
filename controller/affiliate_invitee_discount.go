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
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

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

var distributorModelDiscountTemplateExportHeaders = []string{
	"模型 / 通道路径",
	"模型类型",
	"官方价格（全局价）",
	"调用折扣",
	"折扣后价格",
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

// ExportDistributorModelDiscountTemplate 导出分销商模版调用折扣。
// GET /api/distributor/model-discount-template/export
func ExportDistributorModelDiscountTemplate(c *gin.Context) {
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

	items, _, _, err := model.GetDistributorModelDiscountTemplate(userId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	items = filterInviteeModelDiscountExportItems(items, c.Query("q"), c.Query("supplier_type"))

	data, err := buildDistributorModelDiscountTemplateExportWorkbook(items)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	filename := fmt.Sprintf("调用折扣-%s.xlsx", time.Now().Format("20060102-150405"))
	c.Header("Content-Type", inviteeModelDiscountExportContentType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	c.Header("Content-Length", strconv.Itoa(len(data)))
	c.Data(http.StatusOK, inviteeModelDiscountExportContentType, data)
}

func buildDistributorModelDiscountTemplateExportWorkbook(items []model.InviteeModelMarkupDiscountRateItem) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "调用折扣"
	defaultSheet := f.GetSheetName(0)
	if err := f.SetSheetName(defaultSheet, sheet); err != nil {
		return nil, err
	}

	for i, header := range distributorModelDiscountTemplateExportHeaders {
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

	_ = f.SetCellStyle(sheet, "A1", "E1", headerStyle)
	_ = f.SetColWidth(sheet, "A", "A", 46)
	_ = f.SetColWidth(sheet, "B", "B", 16)
	_ = f.SetColWidth(sheet, "C", "C", 46)
	_ = f.SetColWidth(sheet, "D", "D", 16)
	_ = f.SetColWidth(sheet, "E", "E", 46)
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
			formatDistributorModelDiscountSupplierType(item.SupplierType),
			formatDistributorModelDiscountOfficialPrice(item, 1),
			formatInviteeModelDiscountMarkupRate(item.ChannelPriceDiscountPercent),
			formatDistributorModelDiscountOfficialPrice(item, distributorModelDiscountCallMultiplier(item.ChannelPriceDiscountPercent)),
		}
		for col, value := range values {
			cell, _ := excelize.CoordinatesToCellName(col+1, row)
			if err := f.SetCellValue(sheet, cell, value); err != nil {
				return nil, err
			}
		}
		_ = f.SetRowHeight(sheet, row, distributorModelDiscountExportRowHeight(item))
		_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), bodyStyle)
		_ = f.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("E%d", row), centerStyle)
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func formatDistributorModelDiscountSupplierType(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	return value
}

func distributorModelDiscountCallMultiplier(percent float64) float64 {
	if !isFiniteInviteeModelDiscountFloat(percent) || percent < 0 {
		return math.NaN()
	}
	return percent / 100
}

func formatDistributorModelDiscountPriceNumber(value float64) string {
	if !isFiniteInviteeModelDiscountFloat(value) || value < 0 {
		return "-"
	}
	const precision = 1_000_000.0
	truncated := math.Trunc(value*precision) / precision
	return strings.TrimRight(strings.TrimRight(strconv.FormatFloat(truncated, 'f', 6, 64), "0"), ".")
}

type distributorModelDiscountExportCurrency struct {
	symbol string
	rate   float64
}

func currentDistributorModelDiscountExportCurrency() distributorModelDiscountExportCurrency {
	displayType := operation_setting.GetQuotaDisplayType()
	if displayType != operation_setting.QuotaDisplayTypeUSD &&
		displayType != operation_setting.QuotaDisplayTypeCNY &&
		displayType != operation_setting.QuotaDisplayTypeCustom {
		// 首页在 TOKENS 模式下把价格卡片回退到 USD 展示；导出保持相同行为。
		return distributorModelDiscountExportCurrency{symbol: "$", rate: 1}
	}

	symbol := strings.TrimSpace(operation_setting.GetCurrencySymbol())
	if symbol == "" {
		symbol = "$"
	}
	rate := operation_setting.GetUsdToCurrencyRate(operation_setting.USDExchangeRate)
	if !isFiniteInviteeModelDiscountFloat(rate) || rate <= 0 {
		rate = 1
	}
	return distributorModelDiscountExportCurrency{symbol: symbol, rate: rate}
}

func formatDistributorModelDiscountUnitPrice(valueUSD float64, unit string) string {
	currency := currentDistributorModelDiscountExportCurrency()
	formatted := formatDistributorModelDiscountPriceNumber(valueUSD * currency.rate)
	if formatted == "-" {
		return formatted
	}
	return fmt.Sprintf("%s%s / %s", currency.symbol, formatted, unit)
}

func formatDistributorModelDiscountOfficialPrice(item model.InviteeModelMarkupDiscountRateItem, multiplier float64) string {
	if !isFiniteInviteeModelDiscountFloat(multiplier) || multiplier < 0 {
		return "-"
	}
	// 分销商模板没有渠道加价：折扣后价只按全局官方价 × 调用折扣计算。
	// 不使用 ChannelBasePrice，也不叠加 Default/CurrentMarkupDiscountRate。
	sections := make([]string, 0, 3)
	var basePrice string
	switch item.OfficialPricingQuotaType {
	case 1:
		value := item.OfficialBasePrice * multiplier
		formatted := formatDistributorModelDiscountUnitPrice(value, "次")
		if formatted != "-" {
			basePrice = formatted
		}
	case 3:
		basePrice = formatDistributorModelDiscountTierPricing(item.OfficialRequestTierPricing, multiplier)
	default:
		basePrice = formatDistributorModelDiscountTokenPricing(item, multiplier)
	}
	if basePrice != "" && basePrice != "-" {
		sections = append(sections, basePrice)
	}
	if imagePricing := formatDistributorModelDiscountImagePricing(item, multiplier); imagePricing != "" {
		sections = append(sections, imagePricing)
	}
	if videoPricing := formatDistributorModelDiscountVideoPricing(item, multiplier); videoPricing != "" {
		sections = append(sections, videoPricing)
	}
	if len(sections) == 0 {
		return "-"
	}
	return strings.Join(sections, "\n")
}

func formatDistributorModelDiscountTokenPricing(item model.InviteeModelMarkupDiscountRateItem, multiplier float64) string {
	if !isFiniteInviteeModelDiscountFloat(item.OfficialBasePrice) || item.OfficialBasePrice < 0 {
		return "-"
	}
	inputPrice := item.OfficialBasePrice * ratio_setting.TierRatioBase * multiplier
	completionRatio := item.OfficialCompletionRatio
	if !isFiniteInviteeModelDiscountFloat(completionRatio) || completionRatio < 0 {
		return "-"
	}
	parts := []string{
		"输入 " + formatDistributorModelDiscountUnitPrice(inputPrice, "1M tokens"),
		"输出 " + formatDistributorModelDiscountUnitPrice(inputPrice*completionRatio, "1M tokens"),
	}
	if item.OfficialCacheRatio != nil {
		parts = append(parts, "缓存读 "+formatDistributorModelDiscountUnitPrice(inputPrice*(*item.OfficialCacheRatio), "1M tokens"))
	}
	if item.OfficialCreateCacheRatio != nil {
		parts = append(parts, "缓存写 "+formatDistributorModelDiscountUnitPrice(inputPrice*(*item.OfficialCreateCacheRatio), "1M tokens"))
	}
	return "文本按量\n" + strings.Join(parts, "；")
}

func distributorModelDiscountGlobalModelRatio(item model.InviteeModelMarkupDiscountRateItem) float64 {
	if isFiniteInviteeModelDiscountFloat(item.OfficialModelRatio) && item.OfficialModelRatio > 0 {
		return item.OfficialModelRatio
	}
	if item.OfficialPricingQuotaType != 1 && isFiniteInviteeModelDiscountFloat(item.OfficialBasePrice) && item.OfficialBasePrice > 0 {
		return item.OfficialBasePrice
	}
	return 0
}

func formatDistributorModelDiscountImagePricing(item model.InviteeModelMarkupDiscountRateItem, multiplier float64) string {
	sections := make([]string, 0, 2)
	if rules := item.OfficialImagePricingRules; rules != nil && ratio_setting.HasUsableImagePerImageRules(*rules) {
		lines := []string{"图片生成"}
		lines = appendDistributorModelDiscountImageRows(lines, "文生图", rules.TextToImagePerImage, multiplier)
		lines = appendDistributorModelDiscountImageRows(lines, "图生图", rules.ImageToImagePerImage, multiplier)
		if len(lines) > 1 {
			sections = append(sections, strings.Join(lines, "\n"))
		}
	} else if item.OfficialImagePrice != nil && *item.OfficialImagePrice > 0 {
		sections = append(sections, "图片生成："+
			formatDistributorModelDiscountUnitPrice(*item.OfficialImagePrice*multiplier, "张"))
	}

	if item.OfficialImageRatio != nil && *item.OfficialImageRatio > 0 {
		globalRatio := distributorModelDiscountGlobalModelRatio(item)
		if globalRatio > 0 {
			price := globalRatio * ratio_setting.TierRatioBase * *item.OfficialImageRatio * multiplier
			sections = append(sections, "图片输入："+
				formatDistributorModelDiscountUnitPrice(price, "1M tokens"))
		}
	}
	return strings.Join(sections, "\n")
}

func appendDistributorModelDiscountImageRows(lines []string, mode string, rows []ratio_setting.ImageResolutionPerImageRule, multiplier float64) []string {
	for _, row := range rows {
		if row.ImagePrice <= 0 {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s · %s：%s",
			mode,
			distributorModelDiscountResolutionLabel(row.Resolution),
			formatDistributorModelDiscountUnitPrice(row.ImagePrice*multiplier, "张"),
		))
	}
	return lines
}

func formatDistributorModelDiscountVideoPricing(item model.InviteeModelMarkupDiscountRateItem, multiplier float64) string {
	if rules := item.OfficialVideoPricingRules; rules != nil {
		switch {
		case ratio_setting.HasUsableVideoPerTokenRules(*rules):
			lines := []string{"视频生成（按 token）"}
			groups := distributorModelDiscountVideoPriceGroups{}
			groups.addAudioMode("文生视频", rules.TextToVideoPerToken)
			groups.addAudioMode("图生视频", rules.ImageToVideoPerToken)
			groups.addAudioMode("视频生视频", rules.VideoToVideoPerToken)
			lines = groups.appendTo(lines, multiplier, "1M tokens")
			if len(lines) > 1 {
				return strings.Join(lines, "\n")
			}
		case ratio_setting.HasUsableVideoPerSecondRules(*rules):
			lines := []string{"视频生成（按秒）"}
			groups := distributorModelDiscountVideoPriceGroups{}
			groups.addAudioMode("文生视频", rules.TextToVideoPerSecond)
			groups.addAudioMode("图生视频", rules.ImageToVideoPerSecond)
			groups.addAudioMode("视频生视频", rules.VideoToVideoPerSecond)
			lines = groups.appendTo(lines, multiplier, "秒")
			if len(lines) > 1 {
				return strings.Join(lines, "\n")
			}
		case ratio_setting.HasUsableVideoPerVideoRules(*rules):
			lines := []string{"视频生成（按条）"}
			groups := distributorModelDiscountVideoPriceGroups{}
			groups.addPerVideoMode("文生视频", rules.TextToVideoPerVideo, rules.TextToVideoPerItem)
			groups.addPerVideoMode("图生视频", rules.ImageToVideoPerVideo, rules.ImageToVideoPerItem)
			if hasPositiveDistributorModelDiscountLegacyVideoRows(rules.VideoToVideoInputPerVideo) ||
				hasPositiveDistributorModelDiscountLegacyVideoRows(rules.VideoToVideoOutputPerVideo) {
				groups.addLegacyMode("视频生视频（输入）", rules.VideoToVideoInputPerVideo)
				groups.addLegacyMode("视频生视频（输出）", rules.VideoToVideoOutputPerVideo)
			} else {
				groups.addAudioMode("视频生视频", rules.VideoToVideoPerItem)
			}
			lines = groups.appendTo(lines, multiplier, "条")
			if len(lines) > 1 {
				return strings.Join(lines, "\n")
			}
		}
	}

	if item.OfficialVideoPrice != nil && *item.OfficialVideoPrice > 0 {
		return "视频生成：" +
			formatDistributorModelDiscountUnitPrice(*item.OfficialVideoPrice*multiplier, "条")
	}
	if item.OfficialVideoRatio != nil && *item.OfficialVideoRatio > 0 {
		globalRatio := distributorModelDiscountGlobalModelRatio(item)
		completionRatio := item.OfficialVideoCompletionRatio
		if globalRatio > 0 && isFiniteInviteeModelDiscountFloat(completionRatio) && completionRatio > 0 {
			inputPrice := globalRatio * ratio_setting.TierRatioBase * *item.OfficialVideoRatio * multiplier
			return fmt.Sprintf("视频 token：输入 %s；输出 %s",
				formatDistributorModelDiscountUnitPrice(inputPrice, "1M tokens"),
				formatDistributorModelDiscountUnitPrice(inputPrice*completionRatio, "1M tokens"),
			)
		}
	}
	return ""
}

const (
	distributorModelDiscountVideoPriceGeneral = iota
	distributorModelDiscountVideoPriceSilent
	distributorModelDiscountVideoPriceAudio
)

type distributorModelDiscountVideoPriceLine struct {
	mode       string
	resolution string
	priceUSD   float64
}

type distributorModelDiscountVideoPriceGroups [3][]distributorModelDiscountVideoPriceLine

type distributorModelDiscountVideoAudioPrices struct {
	silent *float64
	audio  *float64
}

func (groups *distributorModelDiscountVideoPriceGroups) addAudioMode(mode string, rows []ratio_setting.VideoResolutionAudioPriceRule) {
	pricesByResolution := make(map[string]*distributorModelDiscountVideoAudioPrices)
	for _, row := range rows {
		if !isFiniteInviteeModelDiscountFloat(row.Price) || row.Price <= 0 {
			continue
		}
		resolution := distributorModelDiscountResolutionLabel(row.Resolution)
		prices := pricesByResolution[resolution]
		if prices == nil {
			prices = &distributorModelDiscountVideoAudioPrices{}
			pricesByResolution[resolution] = prices
		}
		price := row.Price
		if row.HasAudio {
			if prices.audio == nil {
				prices.audio = &price
			}
		} else if prices.silent == nil {
			prices.silent = &price
		}
	}

	resolutions := make([]string, 0, len(pricesByResolution))
	for resolution := range pricesByResolution {
		resolutions = append(resolutions, resolution)
	}
	sort.SliceStable(resolutions, func(i, j int) bool {
		return distributorModelDiscountResolutionLess(resolutions[i], resolutions[j])
	})

	for _, resolution := range resolutions {
		prices := pricesByResolution[resolution]
		if prices.silent != nil && prices.audio != nil && distributorModelDiscountVideoPricesEqual(*prices.silent, *prices.audio) {
			groups[distributorModelDiscountVideoPriceGeneral] = append(groups[distributorModelDiscountVideoPriceGeneral], distributorModelDiscountVideoPriceLine{
				mode: mode, resolution: resolution, priceUSD: *prices.silent,
			})
			continue
		}
		if prices.silent != nil {
			groups[distributorModelDiscountVideoPriceSilent] = append(groups[distributorModelDiscountVideoPriceSilent], distributorModelDiscountVideoPriceLine{
				mode: mode, resolution: resolution, priceUSD: *prices.silent,
			})
		}
		if prices.audio != nil {
			groups[distributorModelDiscountVideoPriceAudio] = append(groups[distributorModelDiscountVideoPriceAudio], distributorModelDiscountVideoPriceLine{
				mode: mode, resolution: resolution, priceUSD: *prices.audio,
			})
		}
	}
}

func (groups *distributorModelDiscountVideoPriceGroups) addPerVideoMode(
	mode string,
	legacyRows []ratio_setting.VideoResolutionPerVideoRule,
	itemRows []ratio_setting.VideoResolutionAudioPriceRule,
) {
	if hasPositiveDistributorModelDiscountLegacyVideoRows(legacyRows) {
		groups.addLegacyMode(mode, legacyRows)
		return
	}
	groups.addAudioMode(mode, itemRows)
}

func (groups *distributorModelDiscountVideoPriceGroups) addLegacyMode(mode string, rows []ratio_setting.VideoResolutionPerVideoRule) {
	sortedRows := append([]ratio_setting.VideoResolutionPerVideoRule(nil), rows...)
	sort.SliceStable(sortedRows, func(i, j int) bool {
		left := distributorModelDiscountResolutionLabel(sortedRows[i].Resolution)
		right := distributorModelDiscountResolutionLabel(sortedRows[j].Resolution)
		return distributorModelDiscountResolutionLess(left, right)
	})
	for _, row := range sortedRows {
		if !isFiniteInviteeModelDiscountFloat(row.VideoPrice) || row.VideoPrice <= 0 {
			continue
		}
		groups[distributorModelDiscountVideoPriceGeneral] = append(groups[distributorModelDiscountVideoPriceGeneral], distributorModelDiscountVideoPriceLine{
			mode: mode, resolution: distributorModelDiscountResolutionLabel(row.Resolution), priceUSD: row.VideoPrice,
		})
	}
}

func (groups distributorModelDiscountVideoPriceGroups) appendTo(lines []string, multiplier float64, unit string) []string {
	labels := [...]string{"通用", "无声", "有声"}
	for category, rows := range groups {
		if len(rows) == 0 {
			continue
		}
		lines = append(lines, labels[category])
		for _, row := range rows {
			lines = append(lines, fmt.Sprintf("%s · %s：%s",
				row.mode,
				row.resolution,
				formatDistributorModelDiscountUnitPrice(row.priceUSD*multiplier, unit),
			))
		}
	}
	return lines
}

func distributorModelDiscountVideoPricesEqual(left, right float64) bool {
	scale := math.Max(1, math.Max(math.Abs(left), math.Abs(right)))
	return math.Abs(left-right) <= 1e-9*scale
}

func hasPositiveDistributorModelDiscountLegacyVideoRows(rows []ratio_setting.VideoResolutionPerVideoRule) bool {
	for _, row := range rows {
		if row.VideoPrice > 0 {
			return true
		}
	}
	return false
}

func distributorModelDiscountResolutionLabel(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "默认"
	}
	return value
}

func distributorModelDiscountResolutionLess(left, right string) bool {
	if left == "默认" {
		return right != "默认"
	}
	if right == "默认" {
		return false
	}
	leftValue, leftOK := distributorModelDiscountResolutionSortValue(left)
	rightValue, rightOK := distributorModelDiscountResolutionSortValue(right)
	if leftOK && rightOK && leftValue != rightValue {
		return leftValue < rightValue
	}
	if leftOK != rightOK {
		return leftOK
	}
	return left < right
}

func distributorModelDiscountResolutionSortValue(value string) (float64, bool) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return 0, false
	}
	if strings.HasSuffix(normalized, "p") {
		parsed, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(normalized, "p")), 64)
		return parsed, err == nil && parsed > 0
	}
	if strings.HasSuffix(normalized, "k") {
		parsed, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(normalized, "k")), 64)
		return parsed * 1000, err == nil && parsed > 0
	}
	if parts := strings.Split(normalized, "x"); len(parts) == 2 {
		parsed, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		return parsed, err == nil && parsed > 0
	}
	return 0, false
}

func formatDistributorModelDiscountTierPricing(rule *ratio_setting.RequestTierPricing, multiplier float64) string {
	if rule == nil || len(rule.Tiers) == 0 || !isFiniteInviteeModelDiscountFloat(multiplier) || multiplier < 0 {
		return "-"
	}
	currency := ratio_setting.NormalizeRequestTierCurrency(rule.Currency)
	lines := []string{"阶梯价"}
	previous := int64(0)
	boundary := ratio_setting.NormalizeRequestTierBoundary(rule.Boundary)
	for idx, tier := range rule.Tiers {
		rangeLabel := distributorModelDiscountTierRangeLabel(previous, tier.UpTo, boundary, idx == 0)
		prices := tier.Prices
		lines = append(lines, fmt.Sprintf(
			"%s：输入 %s；输出 %s；缓存读 %s；缓存写 %s",
			rangeLabel,
			formatDistributorModelDiscountUnitPrice(ratio_setting.ConvertRequestTierPriceToUSD(prices.Input, currency)*multiplier, "1M tokens"),
			formatDistributorModelDiscountUnitPrice(ratio_setting.ConvertRequestTierPriceToUSD(prices.Output, currency)*multiplier, "1M tokens"),
			formatDistributorModelDiscountUnitPrice(ratio_setting.ConvertRequestTierPriceToUSD(prices.CacheRead, currency)*multiplier, "1M tokens"),
			formatDistributorModelDiscountUnitPrice(ratio_setting.ConvertRequestTierPriceToUSD(prices.CacheWrite, currency)*multiplier, "1M tokens"),
		))
		if tier.UpTo > 0 {
			previous = tier.UpTo
		}
	}
	return strings.Join(lines, "\n")
}

func distributorModelDiscountTierRangeLabel(previous, upTo int64, boundary string, first bool) string {
	if upTo == 0 {
		if first || previous <= 0 {
			return "输入 token ≥ 0"
		}
		if boundary == ratio_setting.RequestTierBoundaryLte {
			return fmt.Sprintf("输入 token > %d", previous)
		}
		return fmt.Sprintf("输入 token ≥ %d", previous)
	}
	if first || previous <= 0 {
		if boundary == ratio_setting.RequestTierBoundaryLte {
			return fmt.Sprintf("0 ≤ 输入 token ≤ %d", upTo)
		}
		return fmt.Sprintf("0 ≤ 输入 token < %d", upTo)
	}
	if boundary == ratio_setting.RequestTierBoundaryLte {
		return fmt.Sprintf("%d < 输入 token ≤ %d", previous, upTo)
	}
	return fmt.Sprintf("%d ≤ 输入 token < %d", previous, upTo)
}

func distributorModelDiscountExportRowHeight(item model.InviteeModelMarkupDiscountRateItem) float64 {
	lineCount := strings.Count(formatDistributorModelDiscountOfficialPrice(item, 1), "\n") + 1
	return math.Min(409, math.Max(38, float64(lineCount)*22))
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
