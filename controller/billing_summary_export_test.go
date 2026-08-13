package controller

import (
	"bytes"
	"math"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

func billingSummaryTestQuery(granularity string) billingSummaryExportQuery {
	start := time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local).Unix()
	end := time.Date(2026, 6, 30, 23, 59, 59, 0, time.Local).Unix()
	return billingSummaryExportQuery{
		adminLogExportQuery: adminLogExportQuery{logExportQuery: logExportQuery{
			StartTs: start,
			EndTs:   end,
			Lang:    "zh-CN",
		}},
		Granularity: granularity,
	}
}

func TestParseBillingSummaryExportQueryAllowsCrossYearRange(t *testing.T) {
	gin.SetMode(gin.TestMode)
	request := httptest.NewRequest("GET", "/api/log/billing-summary/export?start_timestamp=1704038400&end_timestamp=1767225599&granularity=month", nil)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = request

	query, err := parseBillingSummaryExportQuery(context)
	if err != nil {
		t.Fatalf("cross-year billing summary range rejected: %v", err)
	}
	if query.StartTs != 1704038400 || query.EndTs != 1767225599 || query.Granularity != billingSummaryGranularityMonth {
		t.Fatalf("query=%+v", query)
	}
}

func TestStandardLogExportStillRejectsOverNinetyDays(t *testing.T) {
	gin.SetMode(gin.TestMode)
	request := httptest.NewRequest("GET", "/api/log/export?start_timestamp=1704038400&end_timestamp=1767225599", nil)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = request

	if _, err := parseLogExportQuery(context); err == nil {
		t.Fatal("standard log export unexpectedly accepted an over-90-day range")
	}
}

func TestBillingSummaryGVFastUsesVideoSeconds(t *testing.T) {
	oldQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 1
	t.Cleanup(func() { common.QuotaPerUnit = oldQuotaPerUnit })

	log := &model.Log{
		Id:        1,
		CreatedAt: time.Date(2026, 6, 15, 12, 0, 0, 0, time.Local).Unix(),
		Type:      model.LogTypeConsume,
		ModelName: "GV-3.1-fast",
		ChannelId: 1,
		Quota:     5,
		Other: `{
			"billing_mode":"video_per_second",
			"video_seconds":5,
			"global_video_price_per_second":2,
			"channel_price_discount_percent":50
		}`,
	}
	data := buildBillingSummaryData([]*model.Log{log}, billingSummaryTestQuery(billingSummaryGranularityPeriod))
	if len(data.Text) != 0 || len(data.Video) != 1 {
		t.Fatalf("text=%d video=%d, want video only", len(data.Text), len(data.Video))
	}
	row := data.Video[0]
	if row.Model != "GV-3.1-fast/u1" {
		t.Fatalf("model=%q, want routed model", row.Model)
	}
	if row.Mode != "视频按秒" || row.VideoUnit != "秒" {
		t.Fatalf("mode=%q unit=%q, want 视频按秒/秒", row.Mode, row.VideoUnit)
	}
	if row.VideoUsage != 5 || row.VideoUnitPrice != 2 || row.OfficialTotalUSD != 10 {
		t.Fatalf("usage=%v price=%v official=%v", row.VideoUsage, row.VideoUnitPrice, row.OfficialTotalUSD)
	}
	if math.Abs(row.SettlementUSD-5) > 1e-9 {
		t.Fatalf("settlement=%v, want 5", row.SettlementUSD)
	}
	workbook, err := buildBillingSummaryWorkbook(data, billingSummaryTestQuery(billingSummaryGranularityPeriod))
	if err != nil {
		t.Fatal(err)
	}
	defer workbook.Close()
	if value, _ := workbook.GetCellValue("视频", "G2"); value != "秒" {
		t.Fatalf("workbook video unit=%q, want 秒", value)
	}
	if value, _ := workbook.GetCellValue("视频", "F2", excelize.Options{RawCellValue: true}); value != "5" {
		t.Fatalf("workbook video usage=%q, want 5", value)
	}
}

func TestBillingSummaryMediaUnits(t *testing.T) {
	oldQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 1
	t.Cleanup(func() { common.QuotaPerUnit = oldQuotaPerUnit })

	created := time.Date(2026, 6, 4, 10, 0, 0, 0, time.Local).Unix()
	logs := []*model.Log{
		{Id: 1, CreatedAt: created, Type: model.LogTypeConsume, ModelName: "image-per-item", ChannelId: 1, Quota: 4, Other: `{"billing_mode":"image_per_image","image_count":2,"image_global_rule_usd":2}`},
		{Id: 2, CreatedAt: created + 1, Type: model.LogTypeConsume, ModelName: "image-token", ChannelId: 1, Quota: 2, Other: `{"image":true,"image_output":1000000,"global_model_ratio":1,"global_image_ratio":1}`},
		{Id: 3, CreatedAt: created + 2, Type: model.LogTypeConsume, ModelName: "image-call", ChannelId: 1, Quota: 3, Other: `{"image_generation_call":true,"global_model_price":3}`},
		{Id: 4, CreatedAt: created + 3, Type: model.LogTypeConsume, ModelName: "asr-model", ChannelId: 1, Quota: 2, Other: `{"asr":true,"audio_seconds":20,"global_model_price":0.1}`},
		{Id: 5, CreatedAt: created + 4, Type: model.LogTypeConsume, ModelName: "audio-token", ChannelId: 1, Quota: 2, PromptTokens: 1000000, Other: `{"audio":true,"audio_input":1000000,"global_model_ratio":1,"global_audio_ratio":1}`},
		{Id: 6, CreatedAt: created + 5, Type: model.LogTypeConsume, ModelName: "audio-call", ChannelId: 1, Quota: 3, Other: `{"audio":true,"global_model_price":3}`},
	}
	data := buildBillingSummaryData(logs, billingSummaryTestQuery(billingSummaryGranularityPeriod))
	imageUnits := map[string]bool{}
	for _, row := range data.Image {
		imageUnits[row.Unit] = true
	}
	for _, unit := range []string{"张", "Mtoken", "次"} {
		if !imageUnits[unit] {
			t.Fatalf("image units=%v, missing %s", imageUnits, unit)
		}
	}
	audioUnits := map[string]bool{}
	for _, row := range data.Audio {
		audioUnits[row.BillingUnit(resolveBillingSummaryDict("zh-CN"))] = true
	}
	for _, unit := range []string{"秒", "Mtoken", "次"} {
		if !audioUnits[unit] {
			t.Fatalf("audio units=%v, missing %s", audioUnits, unit)
		}
	}
}

func TestBillingSummaryAggregatesByBucketModelAndBillingUnit(t *testing.T) {
	oldQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 1
	t.Cleanup(func() { common.QuotaPerUnit = oldQuotaPerUnit })

	created := time.Date(2026, 6, 2, 9, 0, 0, 0, time.Local).Unix()
	logs := []*model.Log{
		{Id: 1, CreatedAt: created, Type: model.LogTypeConsume, ModelName: "video-model", ChannelId: 1, Quota: 4, Other: `{"billing_mode":"video_per_second","video_seconds":2,"global_video_price_per_second":2}`},
		{Id: 2, CreatedAt: created + 60, Type: model.LogTypeConsume, ModelName: "video-model", ChannelId: 1, Quota: 9, Other: `{"billing_mode":"video_per_second","video_seconds":3,"global_video_price_per_second":3}`},
		{Id: 3, CreatedAt: created + 120, Type: model.LogTypeConsume, ModelName: "video-model", ChannelId: 1, Quota: 7, Other: `{"billing_mode":"video_per_video","video_count":1,"global_video_price_per_video":7}`},
	}
	data := buildBillingSummaryData(logs, billingSummaryTestQuery(billingSummaryGranularityPeriod))
	if len(data.Video) != 2 {
		t.Fatalf("video rows=%d, want per-second and per-video rows", len(data.Video))
	}
	var perSecond *billingSummaryVideoRow
	for _, row := range data.Video {
		if row.VideoUnit == "秒" {
			perSecond = row
		}
	}
	if perSecond == nil {
		t.Fatal("missing per-second row")
	}
	if perSecond.Calls != 2 || perSecond.VideoUsage != 5 || math.Abs(perSecond.VideoUnitPrice-2.6) > 1e-9 {
		t.Fatalf("calls=%v usage=%v weighted price=%v, want 2/5/2.6", perSecond.Calls, perSecond.VideoUsage, perSecond.VideoUnitPrice)
	}
}

func TestBillingSummaryBucketsDayAndMonth(t *testing.T) {
	created := time.Date(2026, 6, 2, 9, 0, 0, 0, time.Local).Unix()
	log := &model.Log{Id: 1, CreatedAt: created, Type: model.LogTypeConsume, ModelName: "text-model", PromptTokens: 10, Other: `{"global_model_ratio":1}`}
	day := buildBillingSummaryData([]*model.Log{log}, billingSummaryTestQuery(billingSummaryGranularityDay))
	month := buildBillingSummaryData([]*model.Log{log}, billingSummaryTestQuery(billingSummaryGranularityMonth))
	if len(day.Text) != 1 || day.Text[0].Bucket.Key != "2026-06-02" {
		t.Fatalf("day bucket=%+v", day.Text)
	}
	if len(month.Text) != 1 || month.Text[0].Bucket.Key != "2026-06" {
		t.Fatalf("month bucket=%+v", month.Text)
	}
}

func TestBillingSummaryTextCallCountIsActualAggregatedRequests(t *testing.T) {
	oldQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 1
	t.Cleanup(func() { common.QuotaPerUnit = oldQuotaPerUnit })

	created := time.Date(2026, 6, 2, 9, 0, 0, 0, time.Local).Unix()
	logs := []*model.Log{
		{Id: 1, CreatedAt: created, Type: model.LogTypeConsume, ModelName: "text-model", ChannelId: 1, Quota: 2, PromptTokens: 10, CompletionTokens: 5, Other: `{"global_model_ratio":1}`},
		{Id: 2, CreatedAt: created + 60, Type: model.LogTypeConsume, ModelName: "text-model", ChannelId: 1, Quota: 2, PromptTokens: 20, CompletionTokens: 8, Other: `{"global_model_ratio":1}`},
	}
	data := buildBillingSummaryData(logs, billingSummaryTestQuery(billingSummaryGranularityPeriod))
	if len(data.Text) != 1 {
		t.Fatalf("text rows=%d, want 1", len(data.Text))
	}
	row := data.Text[0]
	if row.Calls != 2 || row.InputUsage != 30 || row.OutputUsage != 13 {
		t.Fatalf("calls/input/output=%d/%v/%v, want 2/30/13", row.Calls, row.InputUsage, row.OutputUsage)
	}
}

func TestBillingSummaryWorkbookHasFourMediaSheetsAndNumericCells(t *testing.T) {
	data := billingSummaryData{
		Video: []*billingSummaryVideoRow{{
			Bucket: billingSummaryBucket{Label: "6月1日-6月30日"}, Model: "GV-3.1-fast/u1",
			Calls: 1, Mode: "视频按秒", VideoUnit: "秒", VideoUsage: 5, VideoUnitPrice: 2,
			OfficialTotalUSD: 10, SettlementUSD: 5,
		}},
	}
	workbook, err := buildBillingSummaryWorkbook(data, billingSummaryTestQuery(billingSummaryGranularityPeriod))
	if err != nil {
		t.Fatal(err)
	}
	defer workbook.Close()
	var output bytes.Buffer
	if err := workbook.Write(&output); err != nil {
		t.Fatal(err)
	}
	opened, err := excelize.OpenReader(bytes.NewReader(output.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	wantSheets := []string{"文本", "视频", "图片", "音频"}
	gotSheets := opened.GetSheetList()
	if len(gotSheets) != len(wantSheets) {
		t.Fatalf("sheets=%v", gotSheets)
	}
	for index := range wantSheets {
		if gotSheets[index] != wantSheets[index] {
			t.Fatalf("sheets=%v, want %v", gotSheets, wantSheets)
		}
		merged, err := opened.GetMergeCells(wantSheets[index], true)
		if err != nil {
			t.Fatal(err)
		}
		if len(merged) != 0 {
			t.Fatalf("%s merged headers=%v, want single-row headers", wantSheets[index], merged)
		}
	}
	wantHeaders := map[string]map[string]string{
		"文本": {"B1": "渠道模型", "C1": "调用次数", "D1": "输入 Token", "E1": "输出 Token", "F1": "缓存写入 Token", "G1": "缓存读取 Token", "H1": "折扣前（$）", "I1": "折扣比例", "J1": "折扣后（$）"},
		"视频": {"B1": "渠道模型", "C1": "调用次数", "F1": "用量", "G1": "单位", "H1": "折扣前（$）", "I1": "折扣比例", "J1": "折扣后（$）"},
		"图片": {"B1": "渠道模型", "C1": "调用次数", "F1": "用量", "G1": "单位", "H1": "折扣前（$）", "I1": "折扣比例", "J1": "折扣后（$）"},
		"音频": {"B1": "渠道模型", "C1": "调用次数", "J1": "用量", "K1": "单位", "L1": "折扣前（$）", "M1": "折扣比例", "N1": "折扣后（$）"},
	}
	for sheet, headers := range wantHeaders {
		for cell, want := range headers {
			if value, _ := opened.GetCellValue(sheet, cell); value != want {
				t.Fatalf("%s!%s=%q, want %q", sheet, cell, value, want)
			}
		}
	}
	if value, _ := opened.GetCellValue("视频", "C2", excelize.Options{RawCellValue: true}); value != "1" {
		t.Fatalf("视频!C2=%q, want call count 1", value)
	}
	if value, _ := opened.GetCellValue("视频", "G2"); value != "秒" {
		t.Fatalf("视频!G2=%q, want 秒", value)
	}
	if cellType, err := opened.GetCellType("视频", "F2"); err != nil || (cellType != excelize.CellTypeNumber && cellType != excelize.CellTypeUnset) {
		t.Fatalf("视频!F2 type=%v err=%v, want numeric", cellType, err)
	}
	if value, _ := opened.GetCellValue("视频", "F2", excelize.Options{RawCellValue: true}); value != "5" {
		t.Fatalf("视频!F2 raw=%q, want numeric value 5", value)
	}
	if value, _ := opened.GetCellValue("视频", "I2", excelize.Options{RawCellValue: true}); value != "0.5" {
		t.Fatalf("视频!I2 raw=%q, want discount rate 0.5", value)
	}
	for sheet, cells := range map[string][2]string{
		"文本": {"I2", "J2"},
		"视频": {"I3", "J3"},
		"图片": {"I2", "J2"},
		"音频": {"M2", "N2"},
	} {
		if value, _ := opened.GetCellValue(sheet, cells[0]); value != "折扣后合计" {
			t.Fatalf("%s!%s=%q, want 折扣后合计", sheet, cells[0], value)
		}
	}
	if value, _ := opened.GetCellValue("视频", "J3", excelize.Options{RawCellValue: true}); value != "5" {
		t.Fatalf("视频!J3 raw=%q, want settlement total 5", value)
	}
	for sheet, cell := range map[string]string{"文本": "J2", "图片": "J2", "音频": "N2"} {
		if value, _ := opened.GetCellValue(sheet, cell, excelize.Options{RawCellValue: true}); value != "0" {
			t.Fatalf("%s!%s raw=%q, want empty-sheet total 0", sheet, cell, value)
		}
	}
}

func TestBillingSummaryWorkbookSumsSettlementTotals(t *testing.T) {
	data := billingSummaryData{Video: []*billingSummaryVideoRow{
		{Bucket: billingSummaryBucket{Label: "6月1日"}, Model: "video-a/u1", Calls: 1, SettlementUSD: 1.25},
		{Bucket: billingSummaryBucket{Label: "6月2日"}, Model: "video-b/u1", Calls: 1, SettlementUSD: 2.75},
	}}
	workbook, err := buildBillingSummaryWorkbook(data, billingSummaryTestQuery(billingSummaryGranularityDay))
	if err != nil {
		t.Fatal(err)
	}
	defer workbook.Close()
	if value, _ := workbook.GetCellValue("视频", "I4"); value != "折扣后合计" {
		t.Fatalf("视频!I4=%q, want 折扣后合计", value)
	}
	if value, _ := workbook.GetCellValue("视频", "J4", excelize.Options{RawCellValue: true}); value != "4" {
		t.Fatalf("视频!J4 raw=%q, want settlement total 4", value)
	}
}

func TestBillingSummaryWorkbookUsesConfiguredCurrency(t *testing.T) {
	setting := operation_setting.GetGeneralSetting()
	oldDisplayType := setting.QuotaDisplayType
	oldExchangeRate := operation_setting.USDExchangeRate
	setting.QuotaDisplayType = operation_setting.QuotaDisplayTypeCNY
	operation_setting.USDExchangeRate = 7.3
	t.Cleanup(func() {
		setting.QuotaDisplayType = oldDisplayType
		operation_setting.USDExchangeRate = oldExchangeRate
	})

	data := billingSummaryData{Video: []*billingSummaryVideoRow{{
		Bucket: billingSummaryBucket{Label: "6月1日-6月30日"}, Model: "GV-3.1-fast/u1",
		Calls: 1, Mode: "视频按秒", VideoUnit: "秒", VideoUsage: 5,
		OfficialTotalUSD: 10, SettlementUSD: 5,
	}}}
	workbook, err := buildBillingSummaryWorkbook(data, billingSummaryTestQuery(billingSummaryGranularityPeriod))
	if err != nil {
		t.Fatal(err)
	}
	defer workbook.Close()
	if value, _ := workbook.GetCellValue("视频", "H1"); value != "折扣前（¥）" {
		t.Fatalf("视频!H1=%q, want CNY symbol", value)
	}
	if value, _ := workbook.GetCellValue("视频", "J1"); value != "折扣后（¥）" {
		t.Fatalf("视频!J1=%q, want CNY symbol", value)
	}
	if value, _ := workbook.GetCellValue("视频", "H2", excelize.Options{RawCellValue: true}); value != "73" {
		t.Fatalf("视频!H2=%q, want converted amount 73", value)
	}
	if value, _ := workbook.GetCellValue("视频", "J2", excelize.Options{RawCellValue: true}); value != "36.5" {
		t.Fatalf("视频!J2=%q, want converted amount 36.5", value)
	}
	if value, _ := workbook.GetCellValue("视频", "J3", excelize.Options{RawCellValue: true}); value != "36.5" {
		t.Fatalf("视频!J3=%q, want converted settlement total 36.5", value)
	}
}
