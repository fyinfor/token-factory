// PR7 集成测试：模型层 GetUserAccessibleChannelsWithModels。
//
// 验证「按 group 过滤 + 按 channel 聚合 + 隐藏 disabled」三个核心契约：
//
//   - 同一渠道下多个模型聚合到 models[]；
//   - abilities.enabled=false 不返；
//   - 跨 group 隔离：当前 group 看不到其它 group 的渠道；
//   - 输入 group 为空时返回 (nil, nil)，不打 DB；
//   - 没有任何 ability 命中时返回空切片（而非 nil），便于 controller 直接 JSON 序列化。
//
// 集成测试依赖 task_billing_test.go::TestMain 起的内存 SQLite。
package service

import (
	"testing"

	"github.com/QuantumNous/new-api/model"

	"github.com/stretchr/testify/require"
)

// seedAbility 直接 Create 一条 ability + channel 元数据；测试用例独立的 truncate 由 t.Cleanup
// 在 truncate(t) 帮忙完成（清 abilities + channels）。
func seedAbility(t *testing.T, group string, modelName string, channelID int, enabled bool) {
	t.Helper()
	require.NoError(t, model.DB.Create(&model.Ability{
		Group:     group,
		Model:     modelName,
		ChannelId: channelID,
		Enabled:   enabled,
	}).Error)
}

// seedChannelMeta 写一条 channels 行作 join 源（仅设 ID/Name/Type，其它字段默认零值即可）。
func seedChannelMeta(t *testing.T, id int, name string, channelType int) {
	t.Helper()
	require.NoError(t, model.DB.Create(&model.Channel{
		Id:   id,
		Name: name,
		Type: channelType,
	}).Error)
}

func TestPR7_GetUserAccessibleChannelsWithModels_BasicGrouping(t *testing.T) {
	truncate(t)

	// 渠道 101 在 group=default 下能服务两个模型；渠道 102 只能服务一个。
	seedChannelMeta(t, 101, "openai-prod", 1)
	seedChannelMeta(t, 102, "azure-east", 3)
	seedAbility(t, "default", "gpt-4o", 101, true)
	seedAbility(t, "default", "gpt-4o-mini", 101, true)
	seedAbility(t, "default", "gpt-4o", 102, true)

	got, err := model.GetUserAccessibleChannelsWithModels("default")
	require.NoError(t, err)
	require.Len(t, got, 2, "应聚合出 2 个渠道")

	// 第一个渠道 ID=101，应携带 2 个模型，按字母序：gpt-4o, gpt-4o-mini。
	require.Equal(t, 101, got[0].ID)
	require.Equal(t, "openai-prod", got[0].Name)
	require.Equal(t, 1, got[0].Type)
	require.Equal(t, []string{"gpt-4o", "gpt-4o-mini"}, got[0].Models)

	require.Equal(t, 102, got[1].ID)
	require.Equal(t, "azure-east", got[1].Name)
	require.Equal(t, []string{"gpt-4o"}, got[1].Models)
}

func TestPR7_GetUserAccessibleChannelsWithModels_HidesDisabled(t *testing.T) {
	truncate(t)

	seedChannelMeta(t, 200, "ch-disabled", 1)
	seedChannelMeta(t, 201, "ch-active", 1)
	// channel 200 名下两个模型一个 enabled 一个 disabled —— 应只返回 enabled 的。
	seedAbility(t, "default", "gpt-4o", 200, true)
	seedAbility(t, "default", "gpt-4o-mini", 200, false)
	// channel 201 整个被禁 —— 应根本不出现在结果中。
	seedAbility(t, "default", "claude-3.5", 201, false)

	got, err := model.GetUserAccessibleChannelsWithModels("default")
	require.NoError(t, err)
	require.Len(t, got, 1, "201 全 disabled，不应出现")
	require.Equal(t, 200, got[0].ID)
	require.Equal(t, []string{"gpt-4o"}, got[0].Models, "disabled 模型应被过滤")
}

func TestPR7_GetUserAccessibleChannelsWithModels_IsolatesByGroup(t *testing.T) {
	truncate(t)

	seedChannelMeta(t, 301, "premium-only", 1)
	seedChannelMeta(t, 302, "shared", 2)
	seedAbility(t, "premium", "gpt-4o", 301, true)
	seedAbility(t, "premium", "gpt-4o", 302, true)
	seedAbility(t, "default", "gpt-3.5", 302, true)

	premium, err := model.GetUserAccessibleChannelsWithModels("premium")
	require.NoError(t, err)
	require.Len(t, premium, 2, "premium 组应能同时看到独享 + 共享渠道")

	def, err := model.GetUserAccessibleChannelsWithModels("default")
	require.NoError(t, err)
	require.Len(t, def, 1, "default 组只能看到 302")
	require.Equal(t, 302, def[0].ID)
	require.Equal(t, []string{"gpt-3.5"}, def[0].Models, "default 组下 302 仅暴露 gpt-3.5")
}

func TestPR7_GetUserAccessibleChannelsWithModels_EmptyGroupShortCircuits(t *testing.T) {
	truncate(t)
	// 即使 DB 里有数据，传空 group 也不应触发 SQL 扫描；返回 nil 由 controller 兜底成 [].
	seedChannelMeta(t, 401, "any", 1)
	seedAbility(t, "default", "gpt-4o", 401, true)

	got, err := model.GetUserAccessibleChannelsWithModels("")
	require.NoError(t, err)
	require.Nil(t, got, "空 group 应返回 nil 短路")

	// 空白 group 同样短路（不是「空字符串」一种）。
	got, err = model.GetUserAccessibleChannelsWithModels("   ")
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestPR7_GetUserAccessibleChannelsWithModels_NoMatchReturnsEmptySlice(t *testing.T) {
	truncate(t)
	seedChannelMeta(t, 501, "premium", 1)
	seedAbility(t, "premium", "gpt-4o", 501, true)

	// 用一个无人使用的 group 查询：应返回空切片而非 nil，保证 controller 输出稳定的 [].
	got, err := model.GetUserAccessibleChannelsWithModels("not_in_use")
	require.NoError(t, err)
	require.NotNil(t, got, "controller 直接 JSON marshal，nil 会变成 null；应返回空切片")
	require.Empty(t, got)
}
