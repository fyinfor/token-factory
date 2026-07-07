package controller

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/model"
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
