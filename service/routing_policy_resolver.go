// Package service: routing_policy_resolver 把「用户配置的路由策略」翻译成 router-engine
// 可消费的 provider JSON 并附带候选池约束，供 PR3 在 distributor 接入时直接使用。
//
// 设计取舍：
//   - 请求侧 provider JSON（OpenRouter 兼容）始终优先：如果客户端在请求体里传了
//     {"provider": {...}}，那就以客户端意图为准，用户全局 policy 让位（仅候选池约束保留）。
//     这样对接 OpenRouter SDK 的客户端零迁移。
//   - 候选池约束（channel / model / channel_model 三种 target）独立于 provider JSON，
//     即使客户端传了完整 provider，也只能在用户许可的候选范围内挑——保护用户预算/合规。
//   - 翻译走纯函数：PolicyToProviderJSON 不依赖 DB / 上下文，方便单测。
//
// 调用链（PR3 接入后）：
//
//	distributor → ResolveRoutingPolicy(userID, model, reqProviderJSON)
//	            → buildRouterCandidatesFiltered(group, model, allowFn)
//	            → router.SelectProviders(req with EffectiveProviderJSON)
package service

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/model"
)

// ResolveSourceRequestOnly 表示该次解析没有用户默认策略，仅消费请求侧 provider JSON（或没有任何路由偏好）。
const (
	ResolveSourceNone        = "none"
	ResolveSourceRequestOnly = "request_only"
	ResolveSourceUserDefault = "user_default"
	ResolveSourceMerged      = "merged"
)

// ResolvedRoutingPolicy 是 resolver 的最终输出，distributor 直接消费即可。
//
// 字段含义：
//   - PolicyID：命中的策略 ID；0 表示该用户没有可用默认策略。
//   - Strategy：最终生效的策略名（price / latency / throughput / balanced / custom / external）。
//     external 表示请求侧自带 provider JSON 且没有用户策略。
//   - AllowFallbacks：是否允许 router-engine 在主候选失败时 fall back 到剩余候选。
//   - EffectiveProviderJSON：合成后喂给 router-engine 的 provider JSON 串（可能为空串）。
//   - ChannelAllowlist / ModelAllowlist / ChannelModelAllowlist：候选池约束；
//     三个集合合起来取并集（任一命中即允许）；全部为空时表示无候选池约束。
//   - FallbackStrategy：候选池整体不可用时的兜底策略；distributor 在主候选 0 命中时按此扩展。
//   - Source：本次解析的来源标识，便于日志排错。
type ResolvedRoutingPolicy struct {
	PolicyID              int64
	Strategy              string
	AllowFallbacks        bool
	EffectiveProviderJSON string
	ChannelAllowlist      map[int]struct{}
	ModelAllowlist        map[string]struct{}
	ChannelModelAllowlist map[string]struct{}
	FallbackStrategy      string
	Source                string
}

// HasCandidatePoolConstraint 是否存在候选池约束；用于 distributor 短路判断。
func (r *ResolvedRoutingPolicy) HasCandidatePoolConstraint() bool {
	if r == nil {
		return false
	}
	return len(r.ChannelAllowlist) > 0 || len(r.ModelAllowlist) > 0 || len(r.ChannelModelAllowlist) > 0
}

// IsCandidateAllowed 判断 (channelID, modelName) 是否落在候选池内。
//
// 三个 allowlist 任一命中即允许；全部为空时永远返回 true（无候选池约束）。
//
// 命中规则：
//   - ChannelAllowlist 命中 channelID → 允许（不限制 model）；
//   - ModelAllowlist   命中 modelName → 允许（不限制 channel）；
//   - ChannelModelAllowlist 命中 "channelID|modelName" → 允许（精准绑定）。
func (r *ResolvedRoutingPolicy) IsCandidateAllowed(channelID int, modelName string) bool {
	if r == nil || !r.HasCandidatePoolConstraint() {
		return true
	}
	if len(r.ChannelAllowlist) > 0 {
		if _, ok := r.ChannelAllowlist[channelID]; ok {
			return true
		}
	}
	if len(r.ModelAllowlist) > 0 {
		if _, ok := r.ModelAllowlist[modelName]; ok {
			return true
		}
	}
	if len(r.ChannelModelAllowlist) > 0 {
		key := strconv.Itoa(channelID) + "|" + modelName
		if _, ok := r.ChannelModelAllowlist[key]; ok {
			return true
		}
	}
	return false
}

// ResolveRoutingPolicy 是入口：按规则合成「最终给 router-engine 的 provider JSON + 候选池」。
//
// 行为：
//  1. 用户无可用默认策略：
//     - 请求侧 providerJSON 非空 → 返回 Strategy=external + 该 JSON，无候选池约束。
//     - 请求侧 providerJSON 也为空 → 返回 Source=none、PolicyID=0；distributor 走老逻辑。
//  2. 用户有默认策略：
//     - 请求侧 providerJSON 非空 → EffectiveProviderJSON 用请求侧，候选池仍取自策略，Source=merged。
//     - 请求侧 providerJSON 为空 → EffectiveProviderJSON 由 PolicyToProviderJSON 翻译，Source=user_default。
//
// 任何 DB 失败都不会向上抛错，而是返回 (nil, err) 让上层决定降级策略——本函数承诺：
// 即使返回 (nil, nil) 也是合法状态（视为「无策略」）。
func ResolveRoutingPolicy(userID int, modelName, requestProviderJSON string) (*ResolvedRoutingPolicy, error) {
	requestProviderJSON = strings.TrimSpace(requestProviderJSON)

	if userID <= 0 {
		return resolveWithoutPolicy(requestProviderJSON), nil
	}
	// 经 routing_policy_cache 包一层 TTL 缓存 + singleflight 合并并发回源；调用方零迁移。
	policy, err := LoadDefaultRoutingPolicyCached(userID)
	if err != nil {
		return nil, err
	}
	if policy == nil {
		return resolveWithoutPolicy(requestProviderJSON), nil
	}

	out := &ResolvedRoutingPolicy{
		PolicyID:         policy.ID,
		Strategy:         policy.Strategy,
		AllowFallbacks:   policy.AllowFallbacks,
		FallbackStrategy: policy.FallbackStrategy,
	}
	buildAllowlists(out, policy.Targets)

	if requestProviderJSON != "" {
		out.EffectiveProviderJSON = requestProviderJSON
		out.Source = ResolveSourceMerged
		return out, nil
	}
	js, err := PolicyToProviderJSON(policy)
	if err != nil {
		return nil, err
	}
	out.EffectiveProviderJSON = js
	out.Source = ResolveSourceUserDefault
	return out, nil
}

func resolveWithoutPolicy(requestProviderJSON string) *ResolvedRoutingPolicy {
	if requestProviderJSON == "" {
		return &ResolvedRoutingPolicy{
			PolicyID: 0,
			Source:   ResolveSourceNone,
		}
	}
	return &ResolvedRoutingPolicy{
		PolicyID:              0,
		Strategy:              "external",
		AllowFallbacks:        true,
		EffectiveProviderJSON: requestProviderJSON,
		Source:                ResolveSourceRequestOnly,
	}
}

// buildAllowlists 把 RoutingPolicyTarget 列表展开成三种 allowlist 集合。
// 抽出来便于单测；对空 / nil 输入安全。
func buildAllowlists(out *ResolvedRoutingPolicy, targets []model.RoutingPolicyTarget) {
	if len(targets) == 0 || out == nil {
		return
	}
	for _, t := range targets {
		switch t.TargetType {
		case model.RoutingTargetTypeChannel:
			if t.ChannelID > 0 {
				if out.ChannelAllowlist == nil {
					out.ChannelAllowlist = make(map[int]struct{})
				}
				out.ChannelAllowlist[t.ChannelID] = struct{}{}
			}
		case model.RoutingTargetTypeModel:
			if name := strings.TrimSpace(t.ModelName); name != "" {
				if out.ModelAllowlist == nil {
					out.ModelAllowlist = make(map[string]struct{})
				}
				out.ModelAllowlist[name] = struct{}{}
			}
		case model.RoutingTargetTypeChannelModel:
			name := strings.TrimSpace(t.ModelName)
			if t.ChannelID > 0 && name != "" {
				if out.ChannelModelAllowlist == nil {
					out.ChannelModelAllowlist = make(map[string]struct{})
				}
				key := strconv.Itoa(t.ChannelID) + "|" + name
				out.ChannelModelAllowlist[key] = struct{}{}
			}
		}
	}
}

// providerPrefsDTO 与 router-engine 的 rawPrefs 字段对齐（顺序无关 / 字段择一）；
// 因 router-engine 内部结构未导出，这里维护一份镜像用于序列化。
type providerPrefsDTO struct {
	Sort                   any  `json:"sort,omitempty"`
	AllowFallbacks         *bool `json:"allow_fallbacks,omitempty"`
	MaxPrice               any  `json:"max_price,omitempty"`
	PreferredMinThroughput any  `json:"preferred_min_throughput,omitempty"`
	PreferredMaxLatency    any  `json:"preferred_max_latency,omitempty"`
}

// PolicyToProviderJSON 把 RoutingPolicy 翻译成 router-engine 可消费的 provider JSON。
//
// 翻译规则：
//   - Strategy=custom 时直接返回 ProviderOverridesJSON（去 trim），不再做字段拼装；
//     允许是空串，视为没有任何 provider 偏好（router-engine 走默认 default_price_lb）。
//   - Strategy=price/latency/throughput → 设 sort 字段。
//   - Strategy=balanced → 不设 sort，让 router-engine 走默认加权随机。
//   - AllowFallbacks=false → 设 allow_fallbacks=false（router-engine 会只取第一名）。
//   - MaxPrice>0 → max_price.completion。
//   - MaxLatencyMs>0 → preferred_max_latency.p50（秒），与 router-engine 单位一致。
//   - MinThroughputTps>0 → preferred_min_throughput（标量）。
//
// 返回的 JSON 串可直接喂给 router.ParsePrefs；空策略时返回空串（不是 "{}"），
// router-engine 对空串会构造零值 Prefs，与「无偏好」一致。
func PolicyToProviderJSON(p *model.RoutingPolicy) (string, error) {
	if p == nil {
		return "", nil
	}
	if p.Strategy == model.RoutingStrategyCustom {
		return strings.TrimSpace(p.ProviderOverridesJSON), nil
	}
	dto := providerPrefsDTO{}
	switch p.Strategy {
	case model.RoutingStrategyPrice, model.RoutingStrategyLatency, model.RoutingStrategyThroughput:
		dto.Sort = p.Strategy
	case model.RoutingStrategyBalanced, "":
		// 不设 sort，走默认加权随机。
	}
	if !p.AllowFallbacks {
		f := false
		dto.AllowFallbacks = &f
	}
	if p.MaxPrice > 0 {
		dto.MaxPrice = map[string]float64{"completion": p.MaxPrice}
	}
	if p.MaxLatencyMs > 0 {
		dto.PreferredMaxLatency = map[string]float64{"p50": float64(p.MaxLatencyMs) / 1000.0}
	}
	if p.MinThroughputTps > 0 {
		dto.PreferredMinThroughput = p.MinThroughputTps
	}
	if isEmptyPrefs(dto) {
		return "", nil
	}
	buf, err := json.Marshal(dto)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

// isEmptyPrefs 判断 DTO 是否所有字段都没设值；用于避免 marshal 出空对象 "{}"。
func isEmptyPrefs(d providerPrefsDTO) bool {
	return d.Sort == nil &&
		d.AllowFallbacks == nil &&
		d.MaxPrice == nil &&
		d.PreferredMinThroughput == nil &&
		d.PreferredMaxLatency == nil
}

// SimulateResolveRoutingPolicy 是 dry-run 专用入口：与 ResolveRoutingPolicy 逻辑等价，
// 但 policy 由调用方在内存中提供（不读 DB），便于「保存前预览」场景。
//
// 与 ResolveRoutingPolicy 的差异：
//   - 不查 DB，policy 取自入参；nil 表示用户没有策略；
//   - 永远不返回 error（PolicyToProviderJSON 失败时把 EffectiveProviderJSON 留空，
//     Strategy 改为 "policy_translate_failed" 让前端能定位问题）。
//
// PR3 不会用到这个函数，仅前端 dry-run / 单测使用。
func SimulateResolveRoutingPolicy(policy *model.RoutingPolicy, requestProviderJSON string) *ResolvedRoutingPolicy {
	requestProviderJSON = strings.TrimSpace(requestProviderJSON)
	if policy == nil {
		return resolveWithoutPolicy(requestProviderJSON)
	}
	out := &ResolvedRoutingPolicy{
		PolicyID:         policy.ID,
		Strategy:         policy.Strategy,
		AllowFallbacks:   policy.AllowFallbacks,
		FallbackStrategy: policy.FallbackStrategy,
	}
	buildAllowlists(out, policy.Targets)
	if requestProviderJSON != "" {
		out.EffectiveProviderJSON = requestProviderJSON
		out.Source = ResolveSourceMerged
		return out
	}
	if js, err := PolicyToProviderJSON(policy); err == nil {
		out.EffectiveProviderJSON = js
	} else {
		out.Strategy = "policy_translate_failed"
	}
	out.Source = ResolveSourceUserDefault
	return out
}
