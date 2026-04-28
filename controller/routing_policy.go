// Package controller: routing_policy 暴露「路由偏好」页面所需的 CRUD API + dry-run 演练接口。
//
// 所有接口都挂在 selfRoute（已加 UserAuth 中间件），用 c.GetInt("id") 取登录用户。
// 写操作只允许操作自己的策略；查询接口同样按 user_id 严格过滤——避免越权。
//
// dry-run 接口不写库，仅用 PolicyToProviderJSON + buildAllowlists 的等价路径
// 模拟一次解析，让前端在保存前预览「这条策略最终会变成怎样的 provider JSON / 候选池」。
package controller

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

// routingPolicyTargetDTO 入参中的候选条目；与 model.RoutingPolicyTarget 字段对齐，
// 但不包含 ID / PolicyID / CreatedAt 这些后端补齐的字段。
type routingPolicyTargetDTO struct {
	TargetType string `json:"target_type"`
	ChannelID  int    `json:"channel_id"`
	ModelName  string `json:"model_name"`
}

// routingPolicyRequest 创建 / 更新策略的统一入参。
//
// 三个布尔/状态字段用指针接收：用 nil 区分「JSON 未传」与「显式 false/0」，避免
// "用户想关 fallback 却被默认值救回 true" 之类的歧义。
type routingPolicyRequest struct {
	Name                  string                   `json:"name"`
	Description           string                   `json:"description"`
	Strategy              string                   `json:"strategy"`
	AllowFallbacks        *bool                    `json:"allow_fallbacks"`
	FallbackStrategy      string                   `json:"fallback_strategy"`
	MaxPrice              float64                  `json:"max_price"`
	MaxLatencyMs          int                      `json:"max_latency_ms"`
	MinThroughputTps      float64                  `json:"min_throughput_tps"`
	ProviderOverridesJSON string                   `json:"provider_overrides_json"`
	Status                *int                     `json:"status"`
	IsDefault             *bool                    `json:"is_default"`
	Priority              int                      `json:"priority"`
	Targets               []routingPolicyTargetDTO `json:"targets"`
}

// dryRunRoutingPolicyRequest dry-run 入参：复用 routingPolicyRequest 主体并允许带请求侧 provider JSON。
type dryRunRoutingPolicyRequest struct {
	routingPolicyRequest
	RequestProviderJSON string `json:"request_provider_json"`
	RequestModel        string `json:"request_model"`
}

// dryRunResponse dry-run 输出：把 ResolvedRoutingPolicy 拍平成 JSON 友好的格式。
type dryRunResponse struct {
	PolicyID              int64    `json:"policy_id"`
	Strategy              string   `json:"strategy"`
	AllowFallbacks        bool     `json:"allow_fallbacks"`
	EffectiveProviderJSON string   `json:"effective_provider_json"`
	ChannelAllowlist      []int    `json:"channel_allowlist"`
	ModelAllowlist        []string `json:"model_allowlist"`
	ChannelModelAllowlist []string `json:"channel_model_allowlist"`
	FallbackStrategy      string   `json:"fallback_strategy"`
	Source                string   `json:"source"`
}

func bindRoutingPolicyRequest(c *gin.Context, req *routingPolicyRequest) error {
	if err := common.DecodeJson(c.Request.Body, req); err != nil {
		return err
	}
	return nil
}

// toModelRoutingPolicy 把请求 DTO 翻译为 model 实体。
//
// 默认值规则（用指针字段判断「未传」）：
//   - AllowFallbacks 未传 → true（OpenRouter 兼容默认）；
//   - Status 未传        → enabled；
//   - IsDefault 未传     → false。
//
// 显式传 false/0 时尊重原值，避免业务侧"想关 fallback 却被默认值救回"的歧义。
func toModelRoutingPolicy(req *routingPolicyRequest, userID int) *model.RoutingPolicy {
	p := &model.RoutingPolicy{
		UserID:                userID,
		Name:                  strings.TrimSpace(req.Name),
		Description:           req.Description,
		Strategy:              strings.TrimSpace(req.Strategy),
		FallbackStrategy:      strings.TrimSpace(req.FallbackStrategy),
		MaxPrice:              req.MaxPrice,
		MaxLatencyMs:          req.MaxLatencyMs,
		MinThroughputTps:      req.MinThroughputTps,
		ProviderOverridesJSON: strings.TrimSpace(req.ProviderOverridesJSON),
		Priority:              req.Priority,
	}
	if p.Strategy == "" {
		p.Strategy = model.RoutingStrategyBalanced
	}
	if req.AllowFallbacks != nil {
		p.AllowFallbacks = *req.AllowFallbacks
	} else {
		p.AllowFallbacks = true
	}
	if req.Status != nil {
		p.Status = *req.Status
	} else {
		p.Status = model.RoutingPolicyStatusEnabled
	}
	if req.IsDefault != nil {
		p.IsDefault = *req.IsDefault
	}
	for _, t := range req.Targets {
		p.Targets = append(p.Targets, model.RoutingPolicyTarget{
			TargetType: strings.TrimSpace(t.TargetType),
			ChannelID:  t.ChannelID,
			ModelName:  strings.TrimSpace(t.ModelName),
		})
	}
	return p
}

// CreateMyRoutingPolicy POST /api/user/routing/policies
func CreateMyRoutingPolicy(c *gin.Context) {
	userID := c.GetInt("id")
	if userID <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "未登录"})
		return
	}
	var req routingPolicyRequest
	if err := bindRoutingPolicyRequest(c, &req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的请求"})
		return
	}
	p := toModelRoutingPolicy(&req, userID)
	if err := model.CreateRoutingPolicy(p); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	// 写后立即清缓存：哪怕 is_default=false，新建本身也可能影响后续 SetDefault 时的快照。
	service.InvalidateRoutingPolicyCache(userID)
	full, err := model.GetRoutingPolicyByID(userID, p.ID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": full})
}

// UpdateMyRoutingPolicy PUT /api/user/routing/policies/:id
func UpdateMyRoutingPolicy(c *gin.Context) {
	userID := c.GetInt("id")
	policyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || policyID <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的策略 ID"})
		return
	}
	var req routingPolicyRequest
	if err := bindRoutingPolicyRequest(c, &req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的请求"})
		return
	}
	patch := toModelRoutingPolicy(&req, userID)
	if err := model.UpdateRoutingPolicy(userID, policyID, patch); err != nil {
		if model.IsRoutingPolicyNotFound(err) {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "策略不存在或无权访问"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	service.InvalidateRoutingPolicyCache(userID)
	full, err := model.GetRoutingPolicyByID(userID, policyID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": full})
}

// DeleteMyRoutingPolicy DELETE /api/user/routing/policies/:id
func DeleteMyRoutingPolicy(c *gin.Context) {
	userID := c.GetInt("id")
	policyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || policyID <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的策略 ID"})
		return
	}
	if err := model.DeleteRoutingPolicy(userID, policyID); err != nil {
		if model.IsRoutingPolicyNotFound(err) {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "策略不存在或无权访问"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	service.InvalidateRoutingPolicyCache(userID)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}

// ListMyRoutingPolicies GET /api/user/routing/policies?p=1&page_size=20
func ListMyRoutingPolicies(c *gin.Context) {
	userID := c.GetInt("id")
	pageInfo := common.GetPageQuery(c)
	items, total, err := model.ListRoutingPoliciesByUser(userID, pageInfo)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": pageInfo})
}

// GetMyRoutingPolicy GET /api/user/routing/policies/:id
func GetMyRoutingPolicy(c *gin.Context) {
	userID := c.GetInt("id")
	policyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || policyID <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的策略 ID"})
		return
	}
	p, err := model.GetRoutingPolicyByID(userID, policyID)
	if err != nil {
		if model.IsRoutingPolicyNotFound(err) {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "策略不存在或无权访问"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": p})
}

// SetMyDefaultRoutingPolicy POST /api/user/routing/policies/:id/default
func SetMyDefaultRoutingPolicy(c *gin.Context) {
	userID := c.GetInt("id")
	policyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || policyID <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的策略 ID"})
		return
	}
	if err := model.SetDefaultRoutingPolicy(userID, policyID); err != nil {
		if model.IsRoutingPolicyNotFound(err) {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "策略不存在或无权访问"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	service.InvalidateRoutingPolicyCache(userID)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}

// ClearMyDefaultRoutingPolicy DELETE /api/user/routing/policies/default
func ClearMyDefaultRoutingPolicy(c *gin.Context) {
	userID := c.GetInt("id")
	if err := model.ClearDefaultRoutingPolicy(userID); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	service.InvalidateRoutingPolicyCache(userID)
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}

// GetMyDefaultRoutingPolicy GET /api/user/routing/policies/default
func GetMyDefaultRoutingPolicy(c *gin.Context) {
	userID := c.GetInt("id")
	p, err := model.GetDefaultRoutingPolicyByUser(userID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": p})
}

// ListMyRoutingChannels GET /api/user/routing/channels
//
// 返回当前用户能用的渠道清单 + 每个渠道下可用模型集合，仅暴露 id/name/type/models 公开字段，
// 给前端「路由偏好」页面候选池配置做下拉源——避免用户手填 channel_id 写错。
//
// 数据源是 abilities 表（与 distributor 选渠道一致）：
//   - 用 user.Group 过滤，让用户看到的渠道集合 = 实际能调到的集合；
//   - abilities.enabled=true 才返回，自动过滤掉 channel.status≠1 的禁用渠道。
//
// 不带 key、base_url、setting 等内部字段，权限上等同 GET /api/user/models。
func ListMyRoutingChannels(c *gin.Context) {
	userID := c.GetInt("id")
	if userID <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "未登录"})
		return
	}
	// 优先用 ctx 已写入的 group（middleware/auth 中由 user_cache 填充）；缺失则按 ID 查一次 DB。
	// 避免在请求路径上为「拉一次下拉选项」多打一次 DB。
	group := strings.TrimSpace(c.GetString("group"))
	if group == "" {
		g, err := model.GetUserGroup(userID, false)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
			return
		}
		group = strings.TrimSpace(g)
	}
	if group == "" {
		// 用户没有 group：等价于「无可用渠道」；返回空数组而非 nil 让前端拿到稳定结构。
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": []model.UserAccessibleChannel{}})
		return
	}
	channels, err := model.GetUserAccessibleChannelsWithModels(group)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if channels == nil {
		channels = []model.UserAccessibleChannel{}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": channels})
}

// GetRoutingPolicyMetrics GET /api/log/routing_policy_metrics（admin only）
//
// 把 service.GetRoutingPolicyMetricsSnapshot 输出为 JSON；运维直接 curl 查命中率分布。
// 路由挂在 logRoute（已强制 AdminAuth），与现有 channel_affinity_usage_cache 对齐。
func GetRoutingPolicyMetrics(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    service.GetRoutingPolicyMetricsSnapshot(),
	})
}

// ResetRoutingPolicyMetrics POST /api/log/routing_policy_metrics/reset（admin only）
//
// 把所有运行期累计计数清零；常用于策略 A/B 切换后想从干净基线开始观察。
// 不重置 cache stats（cache 命中是缓存自身的健康指标，需要时单独清）。
func ResetRoutingPolicyMetrics(c *gin.Context) {
	service.ResetRoutingPolicyMetrics()
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}

// DryRunRoutingPolicy POST /api/user/routing/policies/dry_run
//
// 演练入参（routingPolicyRequest + 可选请求侧 provider JSON / 模型名），不写库；
// 返回 PolicyToProviderJSON 翻译结果 + 候选池展开 + 合成来源标识，便于前端预览。
func DryRunRoutingPolicy(c *gin.Context) {
	userID := c.GetInt("id")
	var req dryRunRoutingPolicyRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的请求"})
		return
	}
	p := toModelRoutingPolicy(&req.routingPolicyRequest, userID)
	if err := model.ValidateRoutingPolicy(p); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	resolved := service.SimulateResolveRoutingPolicy(p, strings.TrimSpace(req.RequestProviderJSON))
	if resolved == nil {
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": dryRunResponse{Source: service.ResolveSourceNone}})
		return
	}
	resp := dryRunResponse{
		PolicyID:              resolved.PolicyID,
		Strategy:              resolved.Strategy,
		AllowFallbacks:        resolved.AllowFallbacks,
		EffectiveProviderJSON: resolved.EffectiveProviderJSON,
		FallbackStrategy:      resolved.FallbackStrategy,
		Source:                resolved.Source,
	}
	for id := range resolved.ChannelAllowlist {
		resp.ChannelAllowlist = append(resp.ChannelAllowlist, id)
	}
	for name := range resolved.ModelAllowlist {
		resp.ModelAllowlist = append(resp.ModelAllowlist, name)
	}
	for key := range resolved.ChannelModelAllowlist {
		resp.ChannelModelAllowlist = append(resp.ChannelModelAllowlist, key)
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": resp})
}
