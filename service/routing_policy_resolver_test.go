package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 这些测试都是纯函数级别（PolicyToProviderJSON / buildAllowlists / SimulateResolveRoutingPolicy），
// 不读 DB，独立于 TestMain 设置；只校验「策略 → router-engine provider JSON / 候选池」的翻译契约。

func TestPolicyToProviderJSON_Balanced_NoSort(t *testing.T) {
	p := &model.RoutingPolicy{Strategy: model.RoutingStrategyBalanced, AllowFallbacks: true}
	js, err := PolicyToProviderJSON(p)
	require.NoError(t, err)
	assert.Empty(t, js, "balanced 策略 + 无阈值时不应序列化任何 provider JSON")
}

func TestPolicyToProviderJSON_PriceLatencyThroughput_SetsSort(t *testing.T) {
	cases := []struct {
		strategy string
		want     string
	}{
		{model.RoutingStrategyPrice, "price"},
		{model.RoutingStrategyLatency, "latency"},
		{model.RoutingStrategyThroughput, "throughput"},
	}
	for _, tc := range cases {
		t.Run(tc.strategy, func(t *testing.T) {
			p := &model.RoutingPolicy{Strategy: tc.strategy, AllowFallbacks: true}
			js, err := PolicyToProviderJSON(p)
			require.NoError(t, err)
			require.NotEmpty(t, js)
			var got map[string]any
			require.NoError(t, json.Unmarshal([]byte(js), &got))
			assert.Equal(t, tc.want, got["sort"])
			_, hasFallbacks := got["allow_fallbacks"]
			assert.False(t, hasFallbacks, "AllowFallbacks=true 时不应序列化该字段，让 router-engine 走默认 true")
		})
	}
}

func TestPolicyToProviderJSON_AllowFallbacksFalse_Emitted(t *testing.T) {
	p := &model.RoutingPolicy{Strategy: model.RoutingStrategyPrice, AllowFallbacks: false}
	js, err := PolicyToProviderJSON(p)
	require.NoError(t, err)
	assert.Contains(t, js, `"allow_fallbacks":false`)
}

func TestPolicyToProviderJSON_Thresholds_AreCarried(t *testing.T) {
	p := &model.RoutingPolicy{
		Strategy:         model.RoutingStrategyLatency,
		AllowFallbacks:   true,
		MaxPrice:         0.01,
		MaxLatencyMs:     1500,
		MinThroughputTps: 25.5,
	}
	js, err := PolicyToProviderJSON(p)
	require.NoError(t, err)
	require.NotEmpty(t, js)

	var got struct {
		Sort                   string             `json:"sort"`
		MaxPrice               map[string]float64 `json:"max_price"`
		PreferredMaxLatency    map[string]float64 `json:"preferred_max_latency"`
		PreferredMinThroughput float64            `json:"preferred_min_throughput"`
	}
	require.NoError(t, json.Unmarshal([]byte(js), &got))
	assert.Equal(t, "latency", got.Sort)
	assert.InDelta(t, 0.01, got.MaxPrice["completion"], 1e-9)
	assert.InDelta(t, 1.5, got.PreferredMaxLatency["p50"], 1e-9, "MaxLatencyMs=1500 应翻译为 1.5 秒")
	assert.InDelta(t, 25.5, got.PreferredMinThroughput, 1e-9)
}

func TestPolicyToProviderJSON_Custom_PassthroughOverrides(t *testing.T) {
	raw := `{"sort":"latency","max_price":{"completion":0.02},"allow_fallbacks":false}`
	p := &model.RoutingPolicy{
		Strategy:              model.RoutingStrategyCustom,
		ProviderOverridesJSON: "  " + raw + "  ",
		// 即使阈值字段有值，custom 策略下也不参与拼装：
		MaxPrice: 0.99,
	}
	js, err := PolicyToProviderJSON(p)
	require.NoError(t, err)
	assert.Equal(t, raw, js, "custom 策略应原样透传 ProviderOverridesJSON（仅 trim）")
}

func TestPolicyToProviderJSON_Custom_EmptyOverrides_ReturnsEmpty(t *testing.T) {
	p := &model.RoutingPolicy{Strategy: model.RoutingStrategyCustom, ProviderOverridesJSON: "   "}
	js, err := PolicyToProviderJSON(p)
	require.NoError(t, err)
	assert.Empty(t, js, "custom 策略 + 空 overrides 应等价于无偏好")
}

func TestBuildAllowlists_ThreeTargetTypes(t *testing.T) {
	out := &ResolvedRoutingPolicy{}
	buildAllowlists(out, []model.RoutingPolicyTarget{
		{TargetType: model.RoutingTargetTypeChannel, ChannelID: 11},
		{TargetType: model.RoutingTargetTypeChannel, ChannelID: 12},
		{TargetType: model.RoutingTargetTypeModel, ModelName: "gpt-4o"},
		{TargetType: model.RoutingTargetTypeModel, ModelName: " claude-3-5 "},
		{TargetType: model.RoutingTargetTypeChannelModel, ChannelID: 99, ModelName: "gpt-4o-mini"},
	})

	assert.Len(t, out.ChannelAllowlist, 2)
	assert.Contains(t, out.ChannelAllowlist, 11)
	assert.Contains(t, out.ChannelAllowlist, 12)

	assert.Len(t, out.ModelAllowlist, 2)
	assert.Contains(t, out.ModelAllowlist, "gpt-4o")
	assert.Contains(t, out.ModelAllowlist, "claude-3-5")

	assert.Len(t, out.ChannelModelAllowlist, 1)
	assert.Contains(t, out.ChannelModelAllowlist, "99|gpt-4o-mini")
}

func TestBuildAllowlists_IgnoresInvalidEntries(t *testing.T) {
	out := &ResolvedRoutingPolicy{}
	buildAllowlists(out, []model.RoutingPolicyTarget{
		{TargetType: model.RoutingTargetTypeChannel, ChannelID: 0},     // 非法：channel_id<=0
		{TargetType: model.RoutingTargetTypeModel, ModelName: ""},      // 非法：model_name 空
		{TargetType: model.RoutingTargetTypeChannelModel, ChannelID: 1}, // 非法：缺 model
		{TargetType: "garbage", ChannelID: 5, ModelName: "x"},          // 非法 type
	})
	assert.Empty(t, out.ChannelAllowlist)
	assert.Empty(t, out.ModelAllowlist)
	assert.Empty(t, out.ChannelModelAllowlist)
}

func TestIsCandidateAllowed_NoConstraint_AllowsAll(t *testing.T) {
	r := &ResolvedRoutingPolicy{}
	assert.True(t, r.IsCandidateAllowed(7, "any-model"))
}

func TestIsCandidateAllowed_UnionSemantics(t *testing.T) {
	r := &ResolvedRoutingPolicy{
		ChannelAllowlist: map[int]struct{}{10: {}},
		ModelAllowlist:   map[string]struct{}{"gpt-4o": {}},
		ChannelModelAllowlist: map[string]struct{}{
			"42|claude-3-5": {},
		},
	}
	assert.True(t, r.IsCandidateAllowed(10, "any-model"), "channel 命中即放行")
	assert.True(t, r.IsCandidateAllowed(99, "gpt-4o"), "model 命中即放行")
	assert.True(t, r.IsCandidateAllowed(42, "claude-3-5"), "channel_model 精准命中")
	assert.False(t, r.IsCandidateAllowed(99, "claude-3-5"), "三个 list 都不命中应拦截")
}

func TestSimulateResolve_NoPolicy_NoRequestJSON_ReturnsNone(t *testing.T) {
	r := SimulateResolveRoutingPolicy(nil, "")
	require.NotNil(t, r)
	assert.Equal(t, ResolveSourceNone, r.Source)
	assert.Equal(t, int64(0), r.PolicyID)
	assert.Empty(t, r.EffectiveProviderJSON)
}

func TestSimulateResolve_NoPolicy_RequestJSONOnly(t *testing.T) {
	req := `{"sort":"throughput"}`
	r := SimulateResolveRoutingPolicy(nil, "  "+req+"  ")
	require.NotNil(t, r)
	assert.Equal(t, ResolveSourceRequestOnly, r.Source)
	assert.Equal(t, "external", r.Strategy)
	assert.Equal(t, req, r.EffectiveProviderJSON, "应去除前后空白后透传")
	assert.False(t, r.HasCandidatePoolConstraint())
}

func TestSimulateResolve_RequestJSONWinsOverPolicyJSON(t *testing.T) {
	policy := &model.RoutingPolicy{
		ID:             7,
		Strategy:       model.RoutingStrategyPrice,
		AllowFallbacks: true,
		MaxPrice:       0.005,
		Targets: []model.RoutingPolicyTarget{
			{TargetType: model.RoutingTargetTypeChannel, ChannelID: 1},
		},
	}
	req := `{"sort":"latency","allow_fallbacks":false}`
	r := SimulateResolveRoutingPolicy(policy, req)
	require.NotNil(t, r)
	assert.Equal(t, ResolveSourceMerged, r.Source, "请求 JSON 与策略并存时应标记 merged")
	assert.Equal(t, req, r.EffectiveProviderJSON, "请求 JSON 永远覆盖策略翻译结果")
	assert.True(t, r.HasCandidatePoolConstraint(), "候选池约束不受请求 JSON 影响")
	assert.Contains(t, r.ChannelAllowlist, 1)
	assert.Equal(t, model.RoutingStrategyPrice, r.Strategy, "Strategy 字段反映用户策略名，便于日志排错")
}

func TestSimulateResolve_NoRequest_PolicyTranslationDrives(t *testing.T) {
	policy := &model.RoutingPolicy{
		ID:               9,
		Strategy:         model.RoutingStrategyLatency,
		AllowFallbacks:   true,
		MaxLatencyMs:     2000,
		FallbackStrategy: model.RoutingFallbackPrice,
	}
	r := SimulateResolveRoutingPolicy(policy, "")
	require.NotNil(t, r)
	assert.Equal(t, ResolveSourceUserDefault, r.Source)
	assert.NotEmpty(t, r.EffectiveProviderJSON)
	assert.True(t, strings.Contains(r.EffectiveProviderJSON, `"sort":"latency"`))
	assert.True(t, strings.Contains(r.EffectiveProviderJSON, `"preferred_max_latency"`))
	assert.Equal(t, model.RoutingFallbackPrice, r.FallbackStrategy)
}
