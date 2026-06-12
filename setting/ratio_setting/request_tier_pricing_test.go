package ratio_setting

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestTierPlatformUsdPerM(t *testing.T) {
	segments := []RequestTierSegment{
		{UpTo: 32000, Ratio: 6},
		{UpTo: 128000, Ratio: 12},
		{UpTo: 0, Ratio: 18},
	}
	// 命中第二档（fromToken=32000），无渠道阶梯时回退全局
	price := TierPlatformUsdPerM(nil, segments, 32000, 100, 0, 1)
	require.InDelta(t, 24, price, 1e-9) // 12 * 2 * 1

	channelSegments := []RequestTierSegment{
		{UpTo: 32000, Ratio: 5},
		{UpTo: 128000, Ratio: 10},
		{UpTo: 0, Ratio: 15},
	}
	// 渠道阶梯优先，命中第二档 channel ratio=10
	price = TierPlatformUsdPerM(channelSegments, segments, 32000, 90, 5, 1.2)
	// (10*0.9 + 12*0.05) * 2 * 1.2 = (9 + 0.6) * 2.4 = 23.04
	require.InDelta(t, 23.04, price, 1e-9)
}

func TestFindTierBandFromTokens(t *testing.T) {
	segments := []RequestTierSegment{
		{UpTo: 32000, Ratio: 1},
		{UpTo: 128000, Ratio: 2},
		{UpTo: 0, Ratio: 3},
	}
	require.Equal(t, int64(0), FindTierBandFromTokens(100, segments))
	require.Equal(t, int64(32000), FindTierBandFromTokens(50000, segments))
	require.Equal(t, int64(128000), FindTierBandFromTokens(200000, segments))
}

func TestApplyRequestTierPricingDecimalProgressive(t *testing.T) {
	rule := RequestTierPricingRule{
		Mode: RequestTierModeProgressive,
		Input: []RequestTierSegment{
			{UpTo: 1000, Ratio: 1},
			{UpTo: 2000, Ratio: 0.8},
			{UpTo: 0, Ratio: 0.5},
		},
		Output: []RequestTierSegment{
			{UpTo: 500, Ratio: 2},
			{UpTo: 0, Ratio: 1.5},
		},
		CacheRead: []RequestTierSegment{
			{UpTo: 0, Ratio: 0.1},
		},
		CacheWrite: []RequestTierSegment{
			{UpTo: 0, Ratio: 1.25},
		},
	}

	input, output, cacheRead, cacheWrite, breakdown := ApplyRequestTierPricingDecimal(
		rule,
		decimal.NewFromInt(2500),
		decimal.NewFromInt(800),
		decimal.NewFromInt(300),
		decimal.NewFromInt(400),
	)

	require.True(t, decimal.NewFromInt(2050).Equal(input))
	require.True(t, decimal.NewFromInt(1450).Equal(output))
	require.True(t, decimal.NewFromInt(30).Equal(cacheRead))
	require.True(t, decimal.NewFromInt(500).Equal(cacheWrite))
	require.Len(t, breakdown.Details["input"], 3)
	require.Len(t, breakdown.Details["output"], 2)
}

// TestApplyRequestTierPricingDecimalQwen3Max 测试 qwen3-max 三档阶梯定价
// 示例规则：
//
//	qwen3-max（Input<=32k）        输入 12、输出 48、缓存读取 2、 缓存写入 6
//	qwen3-max（32k<Input<=128k）   输入 24、输出 68、缓存读取 5、 缓存写入 10
//	qwen3-max（128k<Input<=256k）  输入 36、输出 88、缓存读取 8、 缓存写入 12
func TestApplyRequestTierPricingDecimalQwen3Max(t *testing.T) {
	// ratio 对应每 M tokens 的 USD 单价（直接用作计费乘数）
	rule := RequestTierPricingRule{
		Mode: RequestTierModeProgressive,
		Input: []RequestTierSegment{
			{UpTo: 32000, Ratio: 12},
			{UpTo: 128000, Ratio: 24},
			{UpTo: 256000, Ratio: 36},
			{UpTo: 0, Ratio: 0},
		},
		Output: []RequestTierSegment{
			{UpTo: 32000, Ratio: 48},
			{UpTo: 128000, Ratio: 68},
			{UpTo: 256000, Ratio: 88},
			{UpTo: 0, Ratio: 0},
		},
		CacheRead: []RequestTierSegment{
			{UpTo: 32000, Ratio: 2},
			{UpTo: 128000, Ratio: 5},
			{UpTo: 256000, Ratio: 8},
			{UpTo: 0, Ratio: 0},
		},
		CacheWrite: []RequestTierSegment{
			{UpTo: 32000, Ratio: 6},
			{UpTo: 128000, Ratio: 10},
			{UpTo: 256000, Ratio: 12},
			{UpTo: 0, Ratio: 0},
		},
	}

	// 模拟一次消费：输入 40000 tokens、输出 20000 tokens、缓存读取 5000 tokens、缓存写入 3000 tokens
	// 输入：32000×12 + 8000×24 = 384000 + 192000 = 576000
	input, output, cacheRead, cacheWrite, breakdown := ApplyRequestTierPricingDecimal(
		rule,
		decimal.NewFromInt(40000),
		decimal.NewFromInt(20000),
		decimal.NewFromInt(5000),
		decimal.NewFromInt(3000),
	)

	// 输入：40000 tokens，档位落在 32k<Input<=128k
	//   32000 × 12 = 384000，8000 × 24 = 192000，合计 576000
	require.True(t, decimal.NewFromInt(576000).Equal(input))
	// 输出：20000 tokens，档位落在 <=32k
	//   20000 × 48 = 960000
	require.True(t, decimal.NewFromInt(960000).Equal(output))
	// 缓存读取：5000 tokens，档位落在 <=32k
	//   5000 × 2 = 10000
	require.True(t, decimal.NewFromInt(10000).Equal(cacheRead))
	// 缓存写入：3000 tokens，档位落在 <=32k
	//   3000 × 6 = 18000
	require.True(t, decimal.NewFromInt(18000).Equal(cacheWrite))

	// 验证 breakdown details：每个类型都应有对应的分段明细
	require.Len(t, breakdown.Details["input"], 2)      // 32000 + 8000
	require.Len(t, breakdown.Details["output"], 1)     // 20000 全部在 <=32k
	require.Len(t, breakdown.Details["cache_read"], 1) // 5000 全部在 <=32k
	require.Len(t, breakdown.Details["cache_write"], 1)

	// 验证 breakdown 原始与实际值
	require.Equal(t, "40000", breakdown.InputBefore)
	require.Equal(t, "576000", breakdown.InputAfter)
	require.Equal(t, "20000", breakdown.OutputBefore)
	require.Equal(t, "960000", breakdown.OutputAfter)
	require.Equal(t, "5000", breakdown.CacheReadBefore)
	require.Equal(t, "10000", breakdown.CacheReadAfter)
	require.Equal(t, "3000", breakdown.CacheWriteBefore)
	require.Equal(t, "18000", breakdown.CacheWriteAfter)
}
