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
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
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
	"代理调用折扣",
	"代理折扣后价格",
	"平台折扣",
	"平台折扣后价格",
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
	exportDistributorModelDiscountTemplateResponse(c, items, userId)
}

// ExportDistributorModelDiscountTemplateAdmin exports the platform's default
// model×channel call-discount table. The route is protected by AdminAuth; it
// intentionally does not reuse the distributor identity check above.
// GET /api/distributor/admin/model-discount-template/export
func ExportDistributorModelDiscountTemplateAdmin(c *gin.Context) {
	if !common.IsDistributorProfitShareMode() {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "当前站点未启用利润分成模式"})
		return
	}
	items, err := model.GetDefaultDistributorModelDiscountTemplate()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	exportDistributorModelDiscountTemplateResponse(c, items, c.GetInt("id"))
}

func exportDistributorModelDiscountTemplateResponse(c *gin.Context, items []model.InviteeModelMarkupDiscountRateItem, userID int) {
	items = filterInviteeModelDiscountExportItems(items, c.Query("q"), c.Query("supplier_type"))

	data, err := buildDistributorModelDiscountTemplateExportWorkbookForUser(items, userID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("调用折扣-%s.xlsx", timestamp)
	asciiFilename := fmt.Sprintf("call-discount-%s.xlsx", timestamp)
	c.Header("Content-Type", inviteeModelDiscountExportContentType)
	c.Header("Content-Disposition", fmt.Sprintf(
		`attachment; filename=%q; filename*=UTF-8''%s`,
		asciiFilename,
		url.PathEscape(filename),
	))
	c.Header("Content-Length", strconv.Itoa(len(data)))
	c.Data(http.StatusOK, inviteeModelDiscountExportContentType, data)
}

func buildDistributorModelDiscountTemplateExportWorkbook(items []model.InviteeModelMarkupDiscountRateItem) ([]byte, error) {
	return buildDistributorModelDiscountTemplateExportWorkbookForUser(items, 0)
}

func buildDistributorModelDiscountTemplateExportWorkbookForUser(items []model.InviteeModelMarkupDiscountRateItem, userID int) ([]byte, error) {
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

	_ = f.SetCellStyle(sheet, "A1", "G1", headerStyle)
	_ = f.SetColWidth(sheet, "A", "A", 46)
	_ = f.SetColWidth(sheet, "B", "B", 16)
	_ = f.SetColWidth(sheet, "C", "C", 46)
	_ = f.SetColWidth(sheet, "D", "D", 16)
	_ = f.SetColWidth(sheet, "E", "E", 46)
	_ = f.SetColWidth(sheet, "F", "F", 22)
	_ = f.SetColWidth(sheet, "G", "G", 46)
	_ = f.SetRowHeight(sheet, 1, 28)
	_ = f.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})

	platformGroupContext := newDistributorModelDiscountPlatformGroupContext(userID)
	for idx, item := range items {
		row := idx + 2
		platformGroupRatio := platformGroupContext.ratioFor(item)
		values := []any{
			inviteeModelDiscountExportModelPath(item),
			formatDistributorModelDiscountSupplierType(item.SupplierType),
			formatDistributorModelDiscountOfficialPrice(item, 1),
			formatInviteeModelDiscountMarkupRate(item.ChannelPriceDiscountPercent),
			formatDistributorModelDiscountOfficialPrice(item, distributorModelDiscountCallMultiplier(item.ChannelPriceDiscountPercent)),
			formatInviteeModelDiscountMarkupRate(item.ChannelPriceDiscountPercent + item.DefaultMarkupDiscountRate),
			formatDistributorModelDiscountPlatformPrice(item, platformGroupRatio),
		}
		for col, value := range values {
			cell, _ := excelize.CoordinatesToCellName(col+1, row)
			if err := f.SetCellValue(sheet, cell, value); err != nil {
				return nil, err
			}
		}
		_ = f.SetRowHeight(sheet, row, distributorModelDiscountExportRowHeight(values...))
		_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), bodyStyle)
		_ = f.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("G%d", row), centerStyle)
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

// distributorModelDiscountPlatformGroupContext is built once per workbook so
// a large export does not repeatedly load the distributor or copy group maps.
type distributorModelDiscountPlatformGroupContext struct {
	groupRatio   map[string]float64
	usableGroups map[string]string
}

// newDistributorModelDiscountPlatformGroupContext mirrors the home pricing
// page's initial selectedGroup="all" state and controller/pricing's user
// group override/usable-group filtering.
func newDistributorModelDiscountPlatformGroupContext(userID int) distributorModelDiscountPlatformGroupContext {
	userGroup := ""
	if userID > 0 {
		if user, err := model.GetUserById(userID, false); err == nil && user != nil {
			userGroup = user.Group
		}
	}
	groupRatio := ratio_setting.GetGroupRatioCopy()
	for group := range groupRatio {
		if ratio, ok := ratio_setting.GetGroupGroupRatio(userGroup, group); ok {
			groupRatio[group] = ratio
		}
	}
	return distributorModelDiscountPlatformGroupContext{
		groupRatio:   groupRatio,
		usableGroups: service.GetUserUsableGroups(userGroup),
	}
}

func (ctx distributorModelDiscountPlatformGroupContext) ratioFor(item model.InviteeModelMarkupDiscountRateItem) float64 {
	enabledGroups := item.PlatformPricing
	if enabledGroups != nil && len(enabledGroups.EnableGroup) > 0 {
		best := math.Inf(1)
		for _, group := range enabledGroups.EnableGroup {
			if _, ok := ctx.usableGroups[group]; !ok {
				continue
			}
			if ratio, ok := ctx.groupRatio[group]; ok && isFiniteInviteeModelDiscountFloat(ratio) && ratio < best {
				best = ratio
			}
		}
		if !math.IsInf(best, 1) {
			return best
		}
	}
	return 1
}

func distributorModelDiscountPlatformMultiplier(item model.InviteeModelMarkupDiscountRateItem) float64 {
	cost := distributorModelDiscountCallMultiplier(item.ChannelPriceDiscountPercent)
	markup := item.DefaultMarkupDiscountRate / 100
	if !isFiniteInviteeModelDiscountFloat(cost) || !isFiniteInviteeModelDiscountFloat(markup) || markup < 0 {
		return math.NaN()
	}
	return cost + markup
}

func distributorModelDiscountPlatformUnitPrice(valueUSD float64, unit string, groupRatio float64) string {
	return formatDistributorModelDiscountUnitPrice(valueUSD*groupRatio, unit)
}

// distributorModelDiscountPlatformHidesTextTokenPrices mirrors
// isVideoPricingModel/isASRPricingModel in ModelChannelList. Video and ASR
// rows have their own side-sheet presentation and must not be exported as
// ordinary text token pricing.
func distributorModelDiscountPlatformHidesTextTokenPrices(p model.PricingAPIItem) bool {
	if p.ASRPrice != nil {
		return true
	}
	if p.VideoRatio != nil || p.VideoCompletionRatio != nil || p.VideoPrice != nil || p.VideoFlatClipHint != nil {
		return true
	}
	for _, endpointType := range p.SupportedEndpointTypes {
		switch string(endpointType) {
		case "openai-video", "hidream-video", "tokenfactory-video", "videogenerator", "tencentcloud-vod-video", "ali-video":
			return true
		}
	}
	return false
}

func formatDistributorModelDiscountPlatformPrice(item model.InviteeModelMarkupDiscountRateItem, groupRatio float64) string {
	if !isFiniteInviteeModelDiscountFloat(groupRatio) || groupRatio <= 0 {
		groupRatio = 1
	}
	p := item.PlatformPricing
	if p == nil || len(p.ChannelList) == 0 {
		return formatDistributorModelDiscountOfficialPrice(item, distributorModelDiscountPlatformMultiplier(item)*groupRatio)
	}
	ch := p.ChannelList[0]
	cost := ch.EffectiveCostPercent / 100
	if !isFiniteInviteeModelDiscountFloat(cost) || cost < 0 {
		cost = distributorModelDiscountCallMultiplier(item.ChannelPriceDiscountPercent)
	}
	markup := ch.MarkupDiscountRate / 100
	sections := make([]string, 0, 3)
	if !distributorModelDiscountPlatformHidesTextTokenPrices(*p) {
		switch ch.QuotaType {
		case 1:
			price := ch.ModelPrice*cost + p.ModelPrice*markup
			if price > 0 {
				sections = append(sections, distributorModelDiscountPlatformUnitPrice(price, "次", groupRatio))
			}
		case 3:
			if tier := formatDistributorModelDiscountPlatformTierPricing(item, ch, cost, markup, groupRatio); tier != "" {
				sections = append(sections, tier)
			}
		default:
			if text := formatDistributorModelDiscountPlatformTokenPricing(*p, ch, cost, markup, groupRatio); text != "" {
				sections = append(sections, text)
			}
		}
	}
	if image := formatDistributorModelDiscountPlatformImagePricing(*p, groupRatio); image != "" {
		sections = append(sections, image)
	}
	if !distributorModelDiscountPlatformHidesTextTokenPrices(*p) {
		if audio := formatDistributorModelDiscountPlatformAudioPricing(*p, ch, cost, markup, groupRatio); audio != "" {
			sections = append(sections, audio)
		}
	}
	if video := formatDistributorModelDiscountPlatformVideoPricing(*p, ch, groupRatio); video != "" {
		sections = append(sections, video)
	}
	if p.ASRPrice != nil && *p.ASRPrice > 0 {
		sections = append(sections, "语音识别："+
			distributorModelDiscountPlatformUnitPrice(*p.ASRPrice, "秒", groupRatio))
	}
	if len(sections) == 0 {
		return "-"
	}
	return strings.Join(sections, "\n")
}

func formatDistributorModelDiscountPlatformTokenPricing(p model.PricingAPIItem, ch model.PricingChannelItem, cost, markup, groupRatio float64) string {
	if p.ModelRatio <= 0 || ch.ModelRatio <= 0 {
		return ""
	}
	input := (ch.ModelRatio*cost + p.ModelRatio*markup) * ratio_setting.TierRatioBase
	parts := []string{"输入 " + distributorModelDiscountPlatformUnitPrice(input, "1M tokens", groupRatio)}
	if p.CompletionRatio != nil && *p.CompletionRatio > 0 && ch.CompletionRatio > 0 {
		output := (ch.ModelRatio*ch.CompletionRatio*cost + p.ModelRatio**p.CompletionRatio*markup) * ratio_setting.TierRatioBase
		parts = append(parts, "输出 "+distributorModelDiscountPlatformUnitPrice(output, "1M tokens", groupRatio))
	}
	if p.CacheRatio != nil && *p.CacheRatio > 0 && ch.CacheRatio > 0 {
		read := (ch.ModelRatio*ch.CacheRatio*cost + p.ModelRatio**p.CacheRatio*markup) * ratio_setting.TierRatioBase
		parts = append(parts, "缓存读 "+distributorModelDiscountPlatformUnitPrice(read, "1M tokens", groupRatio))
	}
	if p.CreateCacheRatio != nil && *p.CreateCacheRatio > 0 && ch.CreateCacheRatio > 0 {
		write := (ch.ModelRatio*ch.CreateCacheRatio*cost + p.ModelRatio**p.CreateCacheRatio*markup) * ratio_setting.TierRatioBase
		parts = append(parts, "缓存写 "+distributorModelDiscountPlatformUnitPrice(write, "1M tokens", groupRatio))
	}
	return "文本按量\n" + strings.Join(parts, "；")
}

// Audio billing uses the effective text input rate and the global audio
// multipliers. Channel AudioRatio options are intentionally not used here:
// relay/service/quota.go's calculateAudioQuota applies the global audio ratios
// after resolving the channel/global input rate.
func formatDistributorModelDiscountPlatformAudioPricing(p model.PricingAPIItem, ch model.PricingChannelItem, cost, markup, groupRatio float64) string {
	if p.AudioRatio == nil || *p.AudioRatio <= 0 || p.ModelRatio <= 0 || ch.ModelRatio <= 0 {
		return ""
	}
	effectiveInputRate := ch.ModelRatio*cost + p.ModelRatio*markup
	input := effectiveInputRate * *p.AudioRatio * ratio_setting.TierRatioBase
	parts := []string{"音频输入 " + distributorModelDiscountPlatformUnitPrice(input, "1M tokens", groupRatio)}
	if p.AudioCompletionRatio != nil && *p.AudioCompletionRatio > 0 {
		output := input * *p.AudioCompletionRatio
		parts = append(parts, "音频输出 "+distributorModelDiscountPlatformUnitPrice(output, "1M tokens", groupRatio))
	}
	return "音频按量\n" + strings.Join(parts, "；")
}

func requestTierPricingFromAny(value any) (*ratio_setting.RequestTierPricing, bool) {
	switch v := value.(type) {
	case ratio_setting.RequestTierPricing:
		return &v, true
	case *ratio_setting.RequestTierPricing:
		return v, v != nil
	default:
		return nil, false
	}
}

// distributorModelDiscountTierPricesAtBand mirrors the side sheet's
// findTierPriceAtBand(previousUpTo, ..., "lt") lookup. Global and channel
// rules may have different boundaries, so matching by slice index is wrong.
func distributorModelDiscountTierPricesAtBand(rule *ratio_setting.RequestTierPricing, from int64) ratio_setting.RequestTierPrices {
	if rule == nil {
		return ratio_setting.RequestTierPrices{}
	}
	for _, tier := range rule.Tiers {
		if tier.UpTo == 0 || from < tier.UpTo {
			return tier.Prices
		}
	}
	return ratio_setting.RequestTierPrices{}
}

func formatDistributorModelDiscountPlatformTierPricing(item model.InviteeModelMarkupDiscountRateItem, ch model.PricingChannelItem, cost, markup, groupRatio float64) string {
	channelRule, ok := requestTierPricingFromAny(ch.RequestTierPricing)
	if !ok || channelRule == nil || len(channelRule.Tiers) == 0 {
		return ""
	}
	globalRule := item.OfficialRequestTierPricing
	lines := []string{"阶梯价"}
	previous := int64(0)
	boundary := ratio_setting.NormalizeRequestTierBoundary(channelRule.Boundary)
	for index, tier := range channelRule.Tiers {
		global := distributorModelDiscountTierPricesAtBand(globalRule, previous)
		currency := ratio_setting.NormalizeRequestTierCurrency(channelRule.Currency)
		globalCurrency := ratio_setting.RequestTierCurrencyUSD
		if globalRule != nil {
			globalCurrency = ratio_setting.NormalizeRequestTierCurrency(globalRule.Currency)
		}
		price := func(raw, official float64) string {
			effective := model.EffectiveRuleUnitPrice(
				ratio_setting.ConvertRequestTierPriceToUSD(raw, currency),
				ratio_setting.ConvertRequestTierPriceToUSD(official, globalCurrency),
				cost*100, markup*100,
			)
			return distributorModelDiscountPlatformUnitPrice(effective, "1M tokens", groupRatio)
		}
		lines = append(lines, fmt.Sprintf("%s：输入 %s；输出 %s；缓存读 %s；缓存写 %s",
			distributorModelDiscountTierRangeLabel(previous, tier.UpTo, boundary, index == 0),
			price(tier.Prices.Input, global.Input), price(tier.Prices.Output, global.Output),
			price(tier.Prices.CacheRead, global.CacheRead), price(tier.Prices.CacheWrite, global.CacheWrite),
		))
		if tier.UpTo > 0 {
			previous = tier.UpTo
		}
	}
	return strings.Join(lines, "\n")
}

func formatDistributorModelDiscountPlatformImagePricing(p model.PricingAPIItem, groupRatio float64) string {
	if hint := p.ImagePerImageHint; hint != nil && len(hint.Tiers) > 0 {
		lines := []string{"图片生成"}
		for _, row := range hint.Tiers {
			mode := "文生图"
			if row.Lane == "image_to_image" {
				mode = "图生图"
			}
			lines = append(lines, fmt.Sprintf("%s · %s：%s", mode, distributorModelDiscountResolutionLabel(row.Resolution),
				distributorModelDiscountPlatformUnitPrice(row.UsdAfterChannelDiscount, "张", groupRatio)))
		}
		return strings.Join(lines, "\n")
	}
	return ""
}

func formatDistributorModelDiscountPlatformVideoPricing(p model.PricingAPIItem, ch model.PricingChannelItem, groupRatio float64) string {
	if hint := p.VideoFlatClipHint; hint != nil && len(hint.Tiers) > 0 {
		unit := "条"
		switch hint.BillingMode {
		case "per_second":
			unit = "秒"
		case "per_token":
			unit = "1M tokens"
		}
		layout := distributorModelDiscountVideoPriceLayout{}
		audioRowsByMode := make(map[string][]ratio_setting.VideoResolutionAudioPriceRule)
		for _, row := range hint.Tiers {
			mode := distributorModelDiscountVideoLaneLabel(row.Lane)
			layout.ensureMode(mode)
			if row.HasAudio == nil {
				layout.addLine(mode, distributorModelDiscountVideoPriceGeneral, row.Resolution, row.UsdAfterChannelDiscount)
				continue
			}
			audioRowsByMode[mode] = append(audioRowsByMode[mode], ratio_setting.VideoResolutionAudioPriceRule{
				Resolution: row.Resolution,
				HasAudio:   *row.HasAudio,
				Price:      row.UsdAfterChannelDiscount,
			})
		}
		for _, mode := range layout.modeLabels() {
			layout.addAudioMode(mode, audioRowsByMode[mode])
		}
		return strings.Join(layout.appendTo([]string{"视频生成（" + distributorModelDiscountVideoBillingLabel(hint.BillingMode) + "）"}, groupRatio, unit), "\n")
	}
	if p.VideoPrice != nil && *p.VideoPrice > 0 {
		channelPrice := *p.VideoPrice
		if ch.OptionVideoPrice != nil && *ch.OptionVideoPrice > 0 {
			channelPrice = *ch.OptionVideoPrice
		}
		return "视频生成：" + distributorModelDiscountPlatformUnitPrice(channelPrice, "条", groupRatio)
	}
	return ""
}

func distributorModelDiscountVideoLaneLabel(lane string) string {
	switch lane {
	case "text_to_video_legacy":
		return "文生视频"
	case "image_to_video_legacy":
		return "图生视频"
	case "video_to_video_input_legacy":
		return "视频生视频（输入）"
	case "video_to_video_output_legacy":
		return "视频生视频（输出）"
	case "image_to_video", "image_to_video_per_second", "image_to_video_per_token":
		return "图生视频"
	case "video_to_video", "video_to_video_per_second", "video_to_video_per_token":
		return "视频生视频"
	default:
		return "文生视频"
	}
}

func distributorModelDiscountVideoBillingLabel(mode string) string {
	switch mode {
	case "per_second":
		return "按秒"
	case "per_token":
		return "按 token"
	default:
		return "按条"
	}
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
	if item.OfficialASRPrice == nil {
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
	}
	if basePrice != "" && basePrice != "-" {
		sections = append(sections, basePrice)
	}
	if imagePricing := formatDistributorModelDiscountImagePricing(item, multiplier); imagePricing != "" {
		sections = append(sections, imagePricing)
	}
	if audioPricing := formatDistributorModelDiscountAudioPricing(item, multiplier); audioPricing != "" {
		sections = append(sections, audioPricing)
	}
	if videoPricing := formatDistributorModelDiscountVideoPricing(item, multiplier); videoPricing != "" {
		sections = append(sections, videoPricing)
	}
	if item.OfficialASRPrice != nil && *item.OfficialASRPrice > 0 {
		sections = append(sections, "语音识别："+
			formatDistributorModelDiscountUnitPrice(*item.OfficialASRPrice*multiplier, "秒"))
	}
	if len(sections) == 0 {
		return "-"
	}
	return strings.Join(sections, "\n")
}

func formatDistributorModelDiscountAudioPricing(item model.InviteeModelMarkupDiscountRateItem, multiplier float64) string {
	if item.OfficialAudioRatio == nil || *item.OfficialAudioRatio <= 0 {
		return ""
	}
	modelRatio := distributorModelDiscountGlobalModelRatio(item)
	if modelRatio <= 0 {
		return ""
	}
	input := modelRatio * ratio_setting.TierRatioBase * *item.OfficialAudioRatio * multiplier
	parts := []string{"音频输入 " + formatDistributorModelDiscountUnitPrice(input, "1M tokens")}
	if item.OfficialAudioCompletionRatio != nil && *item.OfficialAudioCompletionRatio > 0 {
		output := input * *item.OfficialAudioCompletionRatio
		parts = append(parts, "音频输出 "+formatDistributorModelDiscountUnitPrice(output, "1M tokens"))
	}
	return "音频按量\n" + strings.Join(parts, "；")
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
			layout := distributorModelDiscountVideoPriceLayout{}
			layout.addAudioMode("文生视频", rules.TextToVideoPerToken)
			layout.addAudioMode("图生视频", rules.ImageToVideoPerToken)
			layout.addAudioMode("视频生视频", rules.VideoToVideoPerToken)
			lines = layout.appendTo(lines, multiplier, "1M tokens")
			if len(lines) > 1 {
				return strings.Join(lines, "\n")
			}
		case ratio_setting.HasUsableVideoPerSecondRules(*rules):
			lines := []string{"视频生成（按秒）"}
			layout := distributorModelDiscountVideoPriceLayout{}
			layout.addAudioMode("文生视频", rules.TextToVideoPerSecond)
			layout.addAudioMode("图生视频", rules.ImageToVideoPerSecond)
			layout.addAudioMode("视频生视频", rules.VideoToVideoPerSecond)
			lines = layout.appendTo(lines, multiplier, "秒")
			if len(lines) > 1 {
				return strings.Join(lines, "\n")
			}
		case ratio_setting.HasUsableVideoPerVideoRules(*rules):
			lines := []string{"视频生成（按条）"}
			layout := distributorModelDiscountVideoPriceLayout{}
			layout.addPerVideoMode("文生视频", rules.TextToVideoPerVideo, rules.TextToVideoPerItem)
			layout.addPerVideoMode("图生视频", rules.ImageToVideoPerVideo, rules.ImageToVideoPerItem)
			if hasPositiveDistributorModelDiscountLegacyVideoRows(rules.VideoToVideoInputPerVideo) ||
				hasPositiveDistributorModelDiscountLegacyVideoRows(rules.VideoToVideoOutputPerVideo) {
				layout.addLegacyMode("视频生视频（输入）", rules.VideoToVideoInputPerVideo)
				layout.addLegacyMode("视频生视频（输出）", rules.VideoToVideoOutputPerVideo)
			} else {
				layout.addAudioMode("视频生视频", rules.VideoToVideoPerItem)
			}
			lines = layout.appendTo(lines, multiplier, "条")
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
	resolution string
	priceUSD   float64
}

type distributorModelDiscountVideoPriceMode struct {
	label  string
	groups [3][]distributorModelDiscountVideoPriceLine
}

// distributorModelDiscountVideoPriceLayout keeps the export hierarchy aligned
// across the agent and platform columns: generation mode first, then audio
// category, then naturally sorted resolution rows.
type distributorModelDiscountVideoPriceLayout struct {
	modes []distributorModelDiscountVideoPriceMode
}

type distributorModelDiscountVideoAudioPrices struct {
	silent *float64
	audio  *float64
}

func (layout *distributorModelDiscountVideoPriceLayout) ensureMode(label string) *distributorModelDiscountVideoPriceMode {
	for i := range layout.modes {
		if layout.modes[i].label == label {
			return &layout.modes[i]
		}
	}
	layout.modes = append(layout.modes, distributorModelDiscountVideoPriceMode{label: label})
	return &layout.modes[len(layout.modes)-1]
}

func (layout *distributorModelDiscountVideoPriceLayout) modeLabels() []string {
	labels := make([]string, 0, len(layout.modes))
	for _, mode := range layout.modes {
		labels = append(labels, mode.label)
	}
	return labels
}

func (layout *distributorModelDiscountVideoPriceLayout) addLine(mode string, category int, resolution string, priceUSD float64) {
	if !isFiniteInviteeModelDiscountFloat(priceUSD) || priceUSD <= 0 {
		return
	}
	group := layout.ensureMode(mode)
	group.groups[category] = append(group.groups[category], distributorModelDiscountVideoPriceLine{
		resolution: distributorModelDiscountResolutionLabel(resolution),
		priceUSD:   priceUSD,
	})
}

func (layout *distributorModelDiscountVideoPriceLayout) addAudioMode(mode string, rows []ratio_setting.VideoResolutionAudioPriceRule) {
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
			layout.addLine(mode, distributorModelDiscountVideoPriceGeneral, resolution, *prices.silent)
			continue
		}
		if prices.silent != nil {
			layout.addLine(mode, distributorModelDiscountVideoPriceSilent, resolution, *prices.silent)
		}
		if prices.audio != nil {
			layout.addLine(mode, distributorModelDiscountVideoPriceAudio, resolution, *prices.audio)
		}
	}
}

func (layout *distributorModelDiscountVideoPriceLayout) addPerVideoMode(
	mode string,
	legacyRows []ratio_setting.VideoResolutionPerVideoRule,
	itemRows []ratio_setting.VideoResolutionAudioPriceRule,
) {
	if hasPositiveDistributorModelDiscountLegacyVideoRows(legacyRows) {
		layout.addLegacyMode(mode, legacyRows)
		return
	}
	layout.addAudioMode(mode, itemRows)
}

func (layout *distributorModelDiscountVideoPriceLayout) addLegacyMode(mode string, rows []ratio_setting.VideoResolutionPerVideoRule) {
	sortedRows := append([]ratio_setting.VideoResolutionPerVideoRule(nil), rows...)
	sort.SliceStable(sortedRows, func(i, j int) bool {
		left := distributorModelDiscountResolutionLabel(sortedRows[i].Resolution)
		right := distributorModelDiscountResolutionLabel(sortedRows[j].Resolution)
		return distributorModelDiscountResolutionLess(left, right)
	})
	for _, row := range sortedRows {
		layout.addLine(mode, distributorModelDiscountVideoPriceGeneral, row.Resolution, row.VideoPrice)
	}
}

func (layout distributorModelDiscountVideoPriceLayout) appendTo(lines []string, multiplier float64, unit string) []string {
	labels := [...]string{"通用", "无声", "有声"}
	for _, mode := range layout.modes {
		categoryCount := 0
		for _, rows := range mode.groups {
			if len(rows) > 0 {
				categoryCount++
			}
		}
		if categoryCount == 0 {
			continue
		}
		lines = append(lines, mode.label)
		for category, rawRows := range mode.groups {
			if len(rawRows) == 0 {
				continue
			}
			rows := append([]distributorModelDiscountVideoPriceLine(nil), rawRows...)
			sort.SliceStable(rows, func(left, right int) bool {
				return distributorModelDiscountResolutionLess(rows[left].resolution, rows[right].resolution)
			})
			// A mode containing only unified prices needs no redundant "通用"
			// heading. Non-unified audio data keeps its category label so the
			// exported price remains unambiguous.
			if categoryCount > 1 || category != distributorModelDiscountVideoPriceGeneral {
				lines = append(lines, labels[category])
			}
			for _, row := range rows {
				lines = append(lines, fmt.Sprintf("%s：%s",
					row.resolution,
					formatDistributorModelDiscountUnitPrice(row.priceUSD*multiplier, unit),
				))
			}
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

func distributorModelDiscountExportRowHeight(values ...any) float64 {
	lineCount := 1
	for _, value := range values {
		count := strings.Count(fmt.Sprint(value), "\n") + 1
		if count > lineCount {
			lineCount = count
		}
	}
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
