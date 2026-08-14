package controller

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

const userModelPricingExportContentType =
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"

type userModelPricingExportColumn struct {
	Key       string
	Header    string
	Width     float64
	CellValue func(row model.UserModelPricingOverride, username string) any
}

var userModelPricingExportColumns = []userModelPricingExportColumn{
	{
		Key:    "model_name",
		Header: "模型",
		Width:  36,
		CellValue: func(row model.UserModelPricingOverride, _ string) any {
			return row.ModelName
		},
	},
	{
		Key:    "mode",
		Header: "模式",
		Width:  14,
		CellValue: func(row model.UserModelPricingOverride, _ string) any {
			if row.NormalizedMode() == model.UserPricingModeChannelList {
				return "渠道清单"
			}
			return "价格上限"
		},
	},
	{
		Key:    "price_discount_percent",
		Header: "成本折扣(%)",
		Width:  14,
		CellValue: func(row model.UserModelPricingOverride, _ string) any {
			return row.PriceDiscountPercent
		},
	},
	{
		Key:    "operating_cost_percent",
		Header: "经营成本(%)",
		Width:  14,
		CellValue: func(row model.UserModelPricingOverride, _ string) any {
			return row.OperatingCostPercent
		},
	},
	{
		Key:    "markup_discount_rate",
		Header: "加价折扣(%)",
		Width:  14,
		CellValue: func(row model.UserModelPricingOverride, _ string) any {
			return row.MarkupDiscountRate
		},
	},
	{
		Key:    "total_percent",
		Header: "总折扣(%)",
		Width:  14,
		CellValue: func(row model.UserModelPricingOverride, _ string) any {
			return row.TotalPercent()
		},
	},
	{
		Key:    "enabled",
		Header: "启用",
		Width:  10,
		CellValue: func(row model.UserModelPricingOverride, _ string) any {
			if row.Enabled {
				return "启用"
			}
			return "禁用"
		},
	},
	{
		Key:    "updated_time",
		Header: "更新时间",
		Width:  20,
		CellValue: func(row model.UserModelPricingOverride, _ string) any {
			if row.UpdatedTime <= 0 {
				return ""
			}
			return time.Unix(row.UpdatedTime, 0).Format("2006-01-02 15:04:05")
		},
	},
	{
		Key:    "username",
		Header: "用户名",
		Width:  18,
		CellValue: func(_ model.UserModelPricingOverride, username string) any {
			return username
		},
	},
	{
		Key:    "user_id",
		Header: "用户ID",
		Width:  12,
		CellValue: func(row model.UserModelPricingOverride, _ string) any {
			return row.UserId
		},
	},
}

var userModelPricingExportColumnByKey = func() map[string]userModelPricingExportColumn {
	m := make(map[string]userModelPricingExportColumn, len(userModelPricingExportColumns))
	for _, col := range userModelPricingExportColumns {
		m[col.Key] = col
	}
	return m
}()

var userModelPricingExportDefaultFields = []string{
	"model_name",
	"price_discount_percent",
	"operating_cost_percent",
	"markup_discount_rate",
	"total_percent",
	"enabled",
	"updated_time",
}

// ExportUserModelPricing GET /api/user_model_pricing/export?user_id=&fields=&model_name=
// 导出指定用户的模型折扣配置为 xlsx；fields 为逗号分隔字段 key，可仅导出部分列。
func ExportUserModelPricing(c *gin.Context) {
	userId, _ := strconv.Atoi(c.Query("user_id"))
	if userId <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "user_id 不能为空"})
		return
	}
	user, err := model.GetUserById(userId, false)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "用户不存在"})
		return
	}

	columns, errMsg := parseUserModelPricingExportFields(c.Query("fields"))
	if errMsg != "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": errMsg})
		return
	}

	modelName := strings.TrimSpace(c.Query("model_name"))
	rows, err := model.ListUserModelPricingOverrides(userId, modelName)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if len(rows) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "该用户暂无指定价配置可导出"})
		return
	}

	username := strings.TrimSpace(user.Username)
	data, err := buildUserModelPricingExportWorkbook(rows, username, columns)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}

	safeName := sanitizeExportFilenamePart(username)
	if safeName == "" {
		safeName = strconv.Itoa(userId)
	}
	filename := fmt.Sprintf(
		"user-model-pricing-%s-%d-%s.xlsx",
		safeName,
		userId,
		time.Now().Format("20060102-150405"),
	)
	c.Header("Content-Type", userModelPricingExportContentType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	c.Header("Content-Length", strconv.Itoa(len(data)))
	c.Data(http.StatusOK, userModelPricingExportContentType, data)
}

func parseUserModelPricingExportFields(raw string) ([]userModelPricingExportColumn, string) {
	raw = strings.TrimSpace(raw)
	keys := userModelPricingExportDefaultFields
	if raw != "" {
		parts := strings.Split(raw, ",")
		keys = make([]string, 0, len(parts))
		seen := make(map[string]struct{}, len(parts))
		for _, part := range parts {
			key := strings.TrimSpace(part)
			if key == "" {
				continue
			}
			if _, ok := userModelPricingExportColumnByKey[key]; !ok {
				return nil, fmt.Sprintf("不支持的导出字段: %s", key)
			}
			if _, dup := seen[key]; dup {
				continue
			}
			seen[key] = struct{}{}
			keys = append(keys, key)
		}
	}
	if len(keys) == 0 {
		return nil, "请至少选择一个导出字段"
	}
	cols := make([]userModelPricingExportColumn, 0, len(keys))
	for _, key := range keys {
		cols = append(cols, userModelPricingExportColumnByKey[key])
	}
	return cols, ""
}

func buildUserModelPricingExportWorkbook(
	rows []model.UserModelPricingOverride,
	username string,
	columns []userModelPricingExportColumn,
) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "用户指定价"
	defaultSheet := f.GetSheetName(0)
	if err := f.SetSheetName(defaultSheet, sheet); err != nil {
		return nil, err
	}

	for i, col := range columns {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err := f.SetCellValue(sheet, cell, col.Header); err != nil {
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
		},
	})

	lastCol, _ := excelize.CoordinatesToCellName(len(columns), 1)
	_ = f.SetCellStyle(sheet, "A1", lastCol, headerStyle)
	_ = f.SetRowHeight(sheet, 1, 28)
	for i, col := range columns {
		colName, _ := excelize.ColumnNumberToName(i + 1)
		_ = f.SetColWidth(sheet, colName, colName, col.Width)
	}
	_ = f.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})

	for idx, row := range rows {
		excelRow := idx + 2
		for colIdx, col := range columns {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, excelRow)
			if err := f.SetCellValue(sheet, cell, col.CellValue(row, username)); err != nil {
				return nil, err
			}
		}
		_ = f.SetRowHeight(sheet, excelRow, 22)
		startCell, _ := excelize.CoordinatesToCellName(1, excelRow)
		endCell, _ := excelize.CoordinatesToCellName(len(columns), excelRow)
		style := bodyStyle
		if columns[0].Key != "model_name" {
			style = centerStyle
		}
		_ = f.SetCellStyle(sheet, startCell, endCell, style)
		// 模型名左对齐，其余列居中更易扫读
		for colIdx, col := range columns {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, excelRow)
			if col.Key == "model_name" || col.Key == "username" {
				_ = f.SetCellStyle(sheet, cell, cell, bodyStyle)
			} else {
				_ = f.SetCellStyle(sheet, cell, cell, centerStyle)
			}
		}
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func sanitizeExportFilenamePart(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		" ", "_",
	)
	return replacer.Replace(s)
}
