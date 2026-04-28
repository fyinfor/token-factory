package middleware

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// buildFallbackProviderJSON 是一段纯字符串生成器，没有外部依赖，专门做单测最稳。
// 这里把策略 → JSON 的映射全部覆盖到，避免后续把字符串改坏。
func TestBuildFallbackProviderJSON_Mapping(t *testing.T) {
	cases := []struct {
		name      string
		strategy  string
		wantSort  any // nil 表示不应包含 sort
		wantAllow bool
		wantEmpty bool
	}{
		{name: "price", strategy: "price", wantSort: "price", wantAllow: true},
		{name: "latency", strategy: "latency", wantSort: "latency", wantAllow: true},
		{name: "throughput", strategy: "throughput", wantSort: "throughput", wantAllow: true},
		{name: "any: only allow_fallbacks", strategy: "any", wantSort: nil, wantAllow: true},
		{name: "none → empty", strategy: "", wantEmpty: true},
		{name: "garbage → empty", strategy: "garbage", wantEmpty: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildFallbackProviderJSON(tc.strategy, "")
			if tc.wantEmpty {
				assert.Empty(t, got)
				return
			}
			var parsed map[string]any
			if err := json.Unmarshal([]byte(got), &parsed); err != nil {
				t.Fatalf("unmarshal failed: %v (raw=%q)", err, got)
			}
			if tc.wantSort == nil {
				_, has := parsed["sort"]
				assert.False(t, has, "any 兜底不应设 sort")
			} else {
				assert.Equal(t, tc.wantSort, parsed["sort"])
			}
			assert.Equal(t, tc.wantAllow, parsed["allow_fallbacks"], "兜底永远 allow_fallbacks=true")
		})
	}
}
