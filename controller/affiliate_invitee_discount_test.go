package controller

import (
	"bytes"
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/xuri/excelize/v2"
)

func TestBuildDistributorModelDiscountTemplateExportWorkbook(t *testing.T) {
	data, err := buildDistributorModelDiscountTemplateExportWorkbook(
		[]model.InviteeModelMarkupDiscountRateItem{
			{
				ModelName:                   "gpt-4.1",
				ChannelPath:                 "gpt-4.1/primary",
				ChannelPriceDiscountPercent: 72.34,
			},
		},
	)
	if err != nil {
		t.Fatalf("build workbook: %v", err)
	}

	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("open workbook: %v", err)
	}
	defer f.Close()

	const sheet = "调用折扣"
	if value, err := f.GetCellValue(sheet, "A1"); err != nil || value != "模型 / 通道路径" {
		t.Fatalf("A1 = %q, %v", value, err)
	}
	if value, err := f.GetCellValue(sheet, "B1"); err != nil || value != "调用折扣" {
		t.Fatalf("B1 = %q, %v", value, err)
	}
	if value, err := f.GetCellValue(sheet, "A2"); err != nil || value != "gpt-4.1\ngpt-4.1/primary" {
		t.Fatalf("A2 = %q, %v", value, err)
	}
	if value, err := f.GetCellValue(sheet, "B2"); err != nil || value != "72.3%" {
		t.Fatalf("B2 = %q, %v", value, err)
	}
}
