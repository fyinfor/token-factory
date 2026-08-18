package service

import (
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
)

func TestSeedanceCalcToken_Example4x3_720p_15s(t *testing.T) {
	// 示例用例：4:3 720p 15s，入参 (1112, 834, 15)，预期 326947
	require.Equal(t, 326947, SeedanceCalcToken(1112, 834, 15))
}

func TestCalcSeedancePreConsumeTokens_Example4x3_720p_15s(t *testing.T) {
	got := CalcSeedancePreConsumeTokens("4:3", "720p", 15)
	require.Equal(t, 326947, got)
}

func TestCalcSeedancePreConsumeTokens_AdaptiveUses1x1(t *testing.T) {
	adaptive := CalcSeedancePreConsumeTokens("adaptive", "720p", 15)
	oneByOne := CalcSeedancePreConsumeTokens("1:1", "720p", 15)
	// adaptive 必须走 1:1 720p：960×960，15s → 960*960*(24*15+1)/1024 = 324900
	require.Equal(t, 324900, adaptive)
	require.Equal(t, oneByOne, adaptive)
	require.NotEqual(t, 326947, adaptive, "adaptive 不得误用 4:3 等其它比例")
}

func TestCalcSeedancePreConsumeTokens_AdaptiveCaseInsensitive(t *testing.T) {
	require.Equal(t,
		CalcSeedancePreConsumeTokens("1:1", "480p", 5),
		CalcSeedancePreConsumeTokens("ADAPTIVE", "480p", 5),
	)
}

func TestLookupSeedancePreConsumeSize_Table(t *testing.T) {
	cases := []struct {
		ratio, quality string
		w, h           int
	}{
		{"16:9", "480p", 864, 496},
		{"16:9", "720p", 1280, 720},
		{"16:9", "1080p", 1920, 1080},
		{"4:3", "720p", 1112, 834},
		{"1:1", "720p", 960, 960},
		{"3:4", "720p", 834, 1112},
		{"9:16", "720p", 720, 1280},
		{"21:9", "720p", 1470, 630},
		{"adaptive", "1080p", 1440, 1440},
	}
	for _, tt := range cases {
		w, h, ok := LookupSeedancePreConsumeSize(tt.ratio, tt.quality)
		require.True(t, ok, "%s %s", tt.ratio, tt.quality)
		require.Equal(t, tt.w, w, "%s %s width", tt.ratio, tt.quality)
		require.Equal(t, tt.h, h, "%s %s height", tt.ratio, tt.quality)
	}
}

func TestCalcSeedancePreConsumeTokens_UnknownQualityFallsBack(t *testing.T) {
	require.Equal(t, SeedanceTokenPreConsumeTokens, CalcSeedancePreConsumeTokens("16:9", "540p", 5))
}

func TestCalcSeedancePreConsumeTokensFromRequest_ExampleAndAdaptive(t *testing.T) {
	example := relaycommon.TaskSubmitReq{
		Duration: 15,
		Metadata: map[string]interface{}{
			"ratio":      "4:3",
			"resolution": "720p",
		},
	}
	require.Equal(t, 326947, CalcSeedancePreConsumeTokensFromRequest(example))

	adaptive := relaycommon.TaskSubmitReq{
		Ratio:      "adaptive",
		Resolution: "720p",
		Duration:   15,
	}
	require.Equal(t, 324900, CalcSeedancePreConsumeTokensFromRequest(adaptive))
}
