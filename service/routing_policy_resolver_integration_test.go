package service

import (
	"testing"

	"github.com/QuantumNous/new-api/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 这些是集成测试：依赖 TestMain 起的内存 SQLite + AutoMigrate（routing_policies / routing_policy_targets）。
// 验证从 model 层 CRUD 到 service.ResolveRoutingPolicy 的端到端契约。

func resetRoutingTables(t *testing.T) {
	t.Helper()
	require.NoError(t, model.DB.Exec("DELETE FROM routing_policies").Error)
	require.NoError(t, model.DB.Exec("DELETE FROM routing_policy_targets").Error)
}

func makeBalancedPolicy(userID int, name string, isDefault bool) *model.RoutingPolicy {
	return &model.RoutingPolicy{
		UserID:         userID,
		Name:           name,
		Strategy:       model.RoutingStrategyBalanced,
		AllowFallbacks: true,
		Status:         model.RoutingPolicyStatusEnabled,
		IsDefault:      isDefault,
	}
}

func TestPR2_CreateRoutingPolicy_PersistsAndReadsBack(t *testing.T) {
	resetRoutingTables(t)

	p := &model.RoutingPolicy{
		UserID:           1001,
		Name:             "便宜优先",
		Description:      "走最低价",
		Strategy:         model.RoutingStrategyPrice,
		AllowFallbacks:   true,
		FallbackStrategy: model.RoutingFallbackAny,
		MaxPrice:         0.01,
		MaxLatencyMs:     2000,
		MinThroughputTps: 5.0,
		Status:           model.RoutingPolicyStatusEnabled,
		IsDefault:        true,
		Targets: []model.RoutingPolicyTarget{
			{TargetType: model.RoutingTargetTypeChannel, ChannelID: 11},
			{TargetType: model.RoutingTargetTypeModel, ModelName: "gpt-4o"},
			{TargetType: model.RoutingTargetTypeChannelModel, ChannelID: 22, ModelName: "claude-3-5"},
		},
	}
	require.NoError(t, model.CreateRoutingPolicy(p))
	require.NotZero(t, p.ID)

	got, err := model.GetRoutingPolicyByID(1001, p.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "便宜优先", got.Name)
	assert.Equal(t, model.RoutingStrategyPrice, got.Strategy)
	assert.True(t, got.IsDefault)
	assert.True(t, got.AllowFallbacks)
	assert.Equal(t, 0.01, got.MaxPrice)
	assert.Len(t, got.Targets, 3)
}

func TestPR2_ValidateRoutingPolicy_RejectsInvalidInputs(t *testing.T) {
	resetRoutingTables(t)

	// 空名字
	require.Error(t, model.CreateRoutingPolicy(&model.RoutingPolicy{
		UserID: 1, Name: "  ", Strategy: model.RoutingStrategyPrice,
	}))
	// 非法策略
	require.Error(t, model.CreateRoutingPolicy(&model.RoutingPolicy{
		UserID: 1, Name: "x", Strategy: "garbage",
	}))
	// 非法 fallback
	require.Error(t, model.CreateRoutingPolicy(&model.RoutingPolicy{
		UserID: 1, Name: "x", Strategy: model.RoutingStrategyPrice, FallbackStrategy: "garbage",
	}))
	// channel target 缺 channel_id
	require.Error(t, model.CreateRoutingPolicy(&model.RoutingPolicy{
		UserID: 1, Name: "x", Strategy: model.RoutingStrategyPrice,
		Targets: []model.RoutingPolicyTarget{{TargetType: model.RoutingTargetTypeChannel, ChannelID: 0}},
	}))
}

func TestPR2_SetDefaultRoutingPolicy_MutexAcrossSameUser(t *testing.T) {
	resetRoutingTables(t)

	a := makeBalancedPolicy(2002, "A", true)
	b := makeBalancedPolicy(2002, "B", false)
	c := makeBalancedPolicy(3003, "C-otherUser", true) // 同 user 互斥与本 user 无关

	require.NoError(t, model.CreateRoutingPolicy(a))
	require.NoError(t, model.CreateRoutingPolicy(b))
	require.NoError(t, model.CreateRoutingPolicy(c))

	// 翻转默认为 B：A 应被自动清掉。
	require.NoError(t, model.SetDefaultRoutingPolicy(2002, b.ID))

	gotA, err := model.GetRoutingPolicyByID(2002, a.ID)
	require.NoError(t, err)
	assert.False(t, gotA.IsDefault, "切换默认后旧默认应自动取消")

	gotB, err := model.GetRoutingPolicyByID(2002, b.ID)
	require.NoError(t, err)
	assert.True(t, gotB.IsDefault)

	gotC, err := model.GetRoutingPolicyByID(3003, c.ID)
	require.NoError(t, err)
	assert.True(t, gotC.IsDefault, "他人的默认策略不应受影响")
}

func TestPR2_SetDefault_DisabledPolicy_Rejected(t *testing.T) {
	resetRoutingTables(t)
	disabled := &model.RoutingPolicy{
		UserID:   4004,
		Name:     "off",
		Strategy: model.RoutingStrategyBalanced,
		Status:   model.RoutingPolicyStatusDisabled,
	}
	require.NoError(t, model.CreateRoutingPolicy(disabled))
	err := model.SetDefaultRoutingPolicy(4004, disabled.ID)
	assert.ErrorIs(t, err, model.ErrRoutingPolicyDisabledCannotDefault)
}

func TestPR2_UpdateRoutingPolicy_FullReplaceTargets(t *testing.T) {
	resetRoutingTables(t)
	p := &model.RoutingPolicy{
		UserID: 5005, Name: "v1",
		Strategy: model.RoutingStrategyBalanced, Status: model.RoutingPolicyStatusEnabled,
		Targets: []model.RoutingPolicyTarget{
			{TargetType: model.RoutingTargetTypeChannel, ChannelID: 1},
			{TargetType: model.RoutingTargetTypeChannel, ChannelID: 2},
		},
	}
	require.NoError(t, model.CreateRoutingPolicy(p))

	patch := &model.RoutingPolicy{
		Name: "v2", Strategy: model.RoutingStrategyLatency,
		Status: model.RoutingPolicyStatusEnabled, AllowFallbacks: false,
		Targets: []model.RoutingPolicyTarget{
			{TargetType: model.RoutingTargetTypeModel, ModelName: "gpt-4o"},
		},
	}
	require.NoError(t, model.UpdateRoutingPolicy(5005, p.ID, patch))

	got, err := model.GetRoutingPolicyByID(5005, p.ID)
	require.NoError(t, err)
	assert.Equal(t, "v2", got.Name)
	assert.Equal(t, model.RoutingStrategyLatency, got.Strategy)
	assert.False(t, got.AllowFallbacks)
	require.Len(t, got.Targets, 1, "全量替换：旧 channel targets 应被清空")
	assert.Equal(t, model.RoutingTargetTypeModel, got.Targets[0].TargetType)
	assert.Equal(t, "gpt-4o", got.Targets[0].ModelName)
}

func TestPR2_DeleteRoutingPolicy_AlsoCleansTargets(t *testing.T) {
	resetRoutingTables(t)
	p := &model.RoutingPolicy{
		UserID: 6006, Name: "to_delete",
		Strategy: model.RoutingStrategyBalanced, Status: model.RoutingPolicyStatusEnabled,
		Targets: []model.RoutingPolicyTarget{
			{TargetType: model.RoutingTargetTypeChannel, ChannelID: 7},
		},
	}
	require.NoError(t, model.CreateRoutingPolicy(p))

	require.NoError(t, model.DeleteRoutingPolicy(6006, p.ID))

	_, err := model.GetRoutingPolicyByID(6006, p.ID)
	assert.True(t, model.IsRoutingPolicyNotFound(err))

	var cnt int64
	require.NoError(t, model.DB.Model(&model.RoutingPolicyTarget{}).
		Where("policy_id = ?", p.ID).Count(&cnt).Error)
	assert.Equal(t, int64(0), cnt, "策略删除后候选池应一同清理")
}

func TestPR2_DeleteRoutingPolicy_OtherUserCannotDelete(t *testing.T) {
	resetRoutingTables(t)
	p := makeBalancedPolicy(7007, "mine", false)
	require.NoError(t, model.CreateRoutingPolicy(p))

	err := model.DeleteRoutingPolicy(8008, p.ID) // 不同 user
	assert.True(t, model.IsRoutingPolicyNotFound(err), "跨用户删除应被拒")

	// 原策略仍在
	got, err := model.GetRoutingPolicyByID(7007, p.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
}

func TestPR2_ResolveRoutingPolicy_NoUserDefault_NoRequest_ReturnsNone(t *testing.T) {
	resetRoutingTables(t)
	r, err := ResolveRoutingPolicy(9009, "gpt-4o", "")
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, ResolveSourceNone, r.Source)
	assert.Equal(t, int64(0), r.PolicyID)
}

func TestPR2_ResolveRoutingPolicy_LoadsUserDefault(t *testing.T) {
	resetRoutingTables(t)
	p := &model.RoutingPolicy{
		UserID: 10010, Name: "default", Strategy: model.RoutingStrategyPrice,
		AllowFallbacks: true, IsDefault: true, Status: model.RoutingPolicyStatusEnabled,
		MaxPrice: 0.02,
		Targets: []model.RoutingPolicyTarget{
			{TargetType: model.RoutingTargetTypeChannel, ChannelID: 50},
		},
	}
	require.NoError(t, model.CreateRoutingPolicy(p))

	r, err := ResolveRoutingPolicy(10010, "gpt-4o", "")
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, ResolveSourceUserDefault, r.Source)
	assert.Equal(t, p.ID, r.PolicyID)
	assert.Equal(t, model.RoutingStrategyPrice, r.Strategy)
	assert.NotEmpty(t, r.EffectiveProviderJSON)
	assert.Contains(t, r.ChannelAllowlist, 50, "策略候选池应反映在 allowlist 上")
}

func TestPR2_ResolveRoutingPolicy_RequestJSON_Wins(t *testing.T) {
	resetRoutingTables(t)
	p := &model.RoutingPolicy{
		UserID: 11011, Name: "default", Strategy: model.RoutingStrategyPrice,
		AllowFallbacks: true, IsDefault: true, Status: model.RoutingPolicyStatusEnabled,
		Targets: []model.RoutingPolicyTarget{
			{TargetType: model.RoutingTargetTypeModel, ModelName: "claude-3-5"},
		},
	}
	require.NoError(t, model.CreateRoutingPolicy(p))

	req := `{"sort":"throughput"}`
	r, err := ResolveRoutingPolicy(11011, "claude-3-5", req)
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, ResolveSourceMerged, r.Source)
	assert.Equal(t, req, r.EffectiveProviderJSON, "请求侧 JSON 永远胜出")
	assert.Contains(t, r.ModelAllowlist, "claude-3-5", "候选池仍来自策略")
}

func TestPR2_ResolveRoutingPolicy_DisabledPolicyIgnored(t *testing.T) {
	resetRoutingTables(t)
	p := &model.RoutingPolicy{
		UserID: 12012, Name: "off", Strategy: model.RoutingStrategyPrice,
		Status: model.RoutingPolicyStatusDisabled, IsDefault: true,
	}
	// 直接落库（绕过 CreateRoutingPolicy 对 IsDefault+Disabled 的纠偏，模拟脏数据）。
	require.NoError(t, model.DB.Create(p).Error)

	r, err := ResolveRoutingPolicy(12012, "gpt-4o", "")
	require.NoError(t, err)
	require.NotNil(t, r)
	assert.Equal(t, ResolveSourceNone, r.Source, "禁用策略不应被 resolver 选中")
}
