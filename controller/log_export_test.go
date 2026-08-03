package controller

import (
	"bytes"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

// extractCacheReadTokens 优先级（与前端渲染完全一致）：
// 1) cache_tokens         （service/log_info_generate.go:73 标准键，所有前端展示都依赖它）
// 2) cache_read_tokens    （历史/扩展名）
// 3) cached_tokens        （OpenAI PromptTokensDetails.CachedTokens / Anthropic cache_read_input_tokens）
// 4) prompt_cache_hit_tokens
// 5) cache_creation_tokens（兜底）
// 6) cache_write_tokens   （兜底）
func TestExtractCacheReadTokens(t *testing.T) {
	cases := []struct {
		name   string
		other  string
		expect int
	}{
		{"空字符串返回 0", "", 0},
		{"非法 JSON 返回 0", "{not json", 0},
		{"无相关键返回 0", `{"foo": 1, "bar": "baz"}`, 0},
		{"cache_tokens 优先（与前端一致）", `{"cache_tokens": 1280, "cached_tokens": 999}`, 1280},
		{"只有 cache_tokens", `{"cache_tokens": 256}`, 256},
		{"cache_tokens 优先于 cache_creation", `{"cache_tokens": 30, "cache_creation_tokens": 1000}`, 30},
		{"cache_read_tokens 兼容", `{"cache_read_tokens": 100, "cached_tokens": 999}`, 100},
		{"只有 cached_tokens", `{"cached_tokens": 256}`, 256},
		{"只有 prompt_cache_hit_tokens", `{"prompt_cache_hit_tokens": 42}`, 42},
		{"cache_creation_tokens 兜底", `{"cache_creation_tokens": 50}`, 50},
		{"cache_write_tokens 兜底", `{"cache_write_tokens": 75}`, 75},
		{"数字支持 float64（GORM 反序列化）", `{"cache_tokens": 123.0}`, 123},
		{"数字支持 int", `{"cache_tokens": 88}`, 88},
		{"数字支持 int64", `{"cache_tokens": 188}`, 188},
		{"非数字类型忽略", `{"cache_tokens": "abc"}`, 0},
		{"0 值仍返回 0（不参与优先匹配）", `{"cache_tokens": 0, "cached_tokens": 5}`, 5},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := extractCacheReadTokens(c.other)
			if got != c.expect {
				t.Errorf("extractCacheReadTokens(%q) = %d, want %d", c.other, got, c.expect)
			}
		})
	}
}

// resolveStatementDict 应当：
// - 11 种 lang 全部命中、并各自返回正确语言的表头/类型标签；
// - 未知 lang/空 lang 一律回退到 zh-CN；
// - 表头列数恒为 15，类型标签 6 项全有。
func TestResolveStatementDict(t *testing.T) {
	expectLang := map[string]string{
		"zh-CN": "序号",
		"zh-TW": "序號",
		"en":    "No.",
		"fr":    "N°",
		"ru":    "№",
		"ja":    "No.",
		"vi":    "STT",
		"id":    "No.",
		"ms":    "No.",
		"th":    "ลำดับ",
		"sw":    "Nambari",
	}
	for lang, wantHeader := range expectLang {
		d := resolveStatementDict(lang)
		if len(d.Header) != 15 {
			t.Errorf("lang=%s header 列数=%d, want 15", lang, len(d.Header))
		}
		if d.Header[0] != wantHeader {
			t.Errorf("lang=%s header[0]=%q, want %q", lang, d.Header[0], wantHeader)
		}
		// 6 个 log type 标签都必须有
		for _, key := range []int{1, 2, 3, 4, 5, 6} {
			if _, ok := d.LogType[key]; !ok {
				t.Errorf("lang=%s 缺少 LogType[%d]", lang, key)
			}
		}
		if d.Meta1 == "" || d.Meta2 == "" || d.Meta3 == "" {
			t.Errorf("lang=%s 缺少 meta 模板", lang)
		}
	}

	// 未知 lang / 空 lang 全部回退 zh-CN
	if d := resolveStatementDict("xx-XX"); d.Header[0] != "序号" {
		t.Errorf("未知 lang 未回退 zh-CN, got header[0]=%q", d.Header[0])
	}
	if d := resolveStatementDict(""); d.Header[0] != "序号" {
		t.Errorf("空 lang 未回退 zh-CN, got header[0]=%q", d.Header[0])
	}
}

func TestFormatLogDetailForExportUsesTableSummary(t *testing.T) {
	cases := []struct {
		name string
		log  *model.Log
		want string
	}{
		{
			name: "异步任务失败退款",
			log: &model.Log{
				Type:  model.LogTypeRefund,
				Other: `{"billing_phase":"refund"}`,
			},
			want: "异步任务失败退款",
		},
		{
			name: "分辨率阶梯计费",
			log: &model.Log{
				Type:  model.LogTypeConsume,
				Other: `{"billing_mode":"video_per_second","model_price":0,"video_resolution":"720p"}`,
			},
			want: "分辨率阶梯计费",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := formatLogDetailForExport(c.log, "zh-CN")
			if !strings.Contains(got, c.want) {
				t.Fatalf("formatLogDetailForExport() = %q, want contains %q", got, c.want)
			}
		})
	}
}

func TestFormatUsageLogExportAmountMatchesConsoleCostPrecision(t *testing.T) {
	oldQuotaPerUnit := common.QuotaPerUnit
	oldDisplayType := operation_setting.GetGeneralSetting().QuotaDisplayType
	defer func() {
		common.QuotaPerUnit = oldQuotaPerUnit
		operation_setting.GetGeneralSetting().QuotaDisplayType = oldDisplayType
	}()

	common.QuotaPerUnit = 3
	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeUSD

	if got := formatUsageLogExportAmount(1, nil); got != "$0.333334\t" {
		t.Fatalf("normal amount = %q, want %q", got, "$0.333334\\t")
	}
	if got := formatUsageLogExportAmount(3, nil); got != "$1.000000\t" {
		t.Fatalf("whole amount = %q, want fixed six decimals", got)
	}
	videoOther := map[string]interface{}{
		"billing_mode":         "video_per_second",
		"video_quota_per_unit": float64(6),
	}
	if got := formatUsageLogExportAmount(1, videoOther); got != "$0.166667\t" {
		t.Fatalf("video amount = %q, want %q", got, "$0.166667\\t")
	}
}

func TestAdminLogXLSXUsesNumericSixDecimalAmountAndKeepsOrder(t *testing.T) {
	oldQuotaPerUnit := common.QuotaPerUnit
	oldDisplayType := operation_setting.GetGeneralSetting().QuotaDisplayType
	defer func() {
		common.QuotaPerUnit = oldQuotaPerUnit
		operation_setting.GetGeneralSetting().QuotaDisplayType = oldDisplayType
	}()

	common.QuotaPerUnit = 3
	operation_setting.GetGeneralSetting().QuotaDisplayType = operation_setting.QuotaDisplayTypeUSD
	logs := []*model.Log{
		{Id: 2, CreatedAt: 200, Type: model.LogTypeConsume, Quota: 3, RequestId: "newest"},
		{Id: 1, CreatedAt: 100, Type: model.LogTypeConsume, Quota: 1, RequestId: "oldest"},
	}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	streamAdminLogsXLSX(c, logs, logExportQuery{StartTs: 1, EndTs: 300, Lang: "en"}, "logs.xlsx", resolveAdminLogExportDict("en"))

	if got := recorder.Header().Get("Content-Type"); got != logExportXLSXContentType {
		t.Fatalf("content type = %q, want %q", got, logExportXLSXContentType)
	}
	f, err := excelize.OpenReader(bytes.NewReader(recorder.Body.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if got, err := f.GetCellValue("Usage Logs", "I4"); err != nil || got != "newest" {
		t.Fatalf("first request id = %q, err=%v, want newest", got, err)
	}
	if got, err := f.GetCellValue("Usage Logs", "I5"); err != nil || got != "oldest" {
		t.Fatalf("second request id = %q, err=%v, want oldest", got, err)
	}
	if typ, err := f.GetCellType("Usage Logs", "O5"); err != nil || (typ != excelize.CellTypeUnset && typ != excelize.CellTypeNumber) {
		t.Fatalf("amount cell type = %v, err=%v, want numeric/unset", typ, err)
	}
	if raw, err := f.GetCellValue("Usage Logs", "O5", excelize.Options{RawCellValue: true}); err != nil || raw != "0.333334" {
		t.Fatalf("raw amount = %q, err=%v, want numeric 0.333334", raw, err)
	}
	if got, err := f.GetCellValue("Usage Logs", "O4"); err != nil || got != "$1.000000" {
		t.Fatalf("whole amount = %q, err=%v, want $1.000000", got, err)
	}
	if got, err := f.GetCellValue("Usage Logs", "O5"); err != nil || got != "$0.333334" {
		t.Fatalf("fractional amount = %q, err=%v, want $0.333334", got, err)
	}
	styleID, err := f.GetCellStyle("Usage Logs", "O5")
	if err != nil {
		t.Fatal(err)
	}
	style, err := f.GetStyle(styleID)
	if err != nil {
		t.Fatal(err)
	}
	if style.CustomNumFmt == nil || !strings.Contains(*style.CustomNumFmt, "0.000000") {
		t.Fatalf("amount number format = %v, want six decimals", style.CustomNumFmt)
	}
}

func TestResolveUsageLogExportQuotaUsesSettledAmount(t *testing.T) {
	log := &model.Log{Quota: 10}
	other := map[string]interface{}{
		"actual_quota":       float64(7),
		"video_billed_quota": float64(10),
	}
	if got := resolveUsageLogExportQuota(log, other); got != 7 {
		t.Fatalf("resolved quota = %d, want 7", got)
	}
	log.Type = model.LogTypeConsume
	if got := resolveUsageLogExportSignedQuota(log, other); got != -7 {
		t.Fatalf("resolved signed quota = %d, want -7", got)
	}
}

// 差额结算行必须导出这一笔差额，而不是任务总额；否则与同一任务的预扣行相加会翻倍。
func TestResolveUsageLogExportQuotaUsesRowQuotaForDeltaLogs(t *testing.T) {
	deltaCharge := &model.Log{Type: model.LogTypeConsume, Quota: 181458}
	chargeOther := map[string]interface{}{
		"billing_phase":      model.BillingPhaseDeltaCharge,
		"pre_consumed_quota": float64(154039),
		"actual_quota":       float64(335497),
		"video_final_quota":  float64(335497),
		"video_billed_quota": float64(335497),
	}
	if got := resolveUsageLogExportQuota(deltaCharge, chargeOther); got != 181458 {
		t.Fatalf("delta charge quota = %d, want 181458", got)
	}
	if got := resolveUsageLogExportSignedQuota(deltaCharge, chargeOther); got != -181458 {
		t.Fatalf("delta charge signed quota = %d, want -181458", got)
	}

	deltaRefund := &model.Log{Type: model.LogTypeRefund, Quota: 200000}
	refundOther := map[string]interface{}{
		"billing_phase":      model.BillingPhaseDeltaRefund,
		"pre_consumed_quota": float64(750000),
		"actual_quota":       float64(550000),
		"video_final_quota":  float64(550000),
	}
	if got := resolveUsageLogExportQuota(deltaRefund, refundOther); got != 200000 {
		t.Fatalf("delta refund quota = %d, want 200000", got)
	}
	if got := resolveUsageLogExportSignedQuota(deltaRefund, refundOther); got != 200000 {
		t.Fatalf("delta refund signed quota = %d, want 200000", got)
	}
}
