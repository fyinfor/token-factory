package controller

import "testing"

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
