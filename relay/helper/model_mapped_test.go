package helper

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestApplyChannelModelMapping(t *testing.T) {
	require.Equal(t, "kk", ApplyChannelModelMapping("", "kk"))
	require.Equal(t, "kk", ApplyChannelModelMapping("{}", "kk"))
	require.Equal(t, "kimi-k2.5", ApplyChannelModelMapping(`{"kk":"kimi-k2.5"}`, "kk"))
	require.Equal(t, "kimi-k2.5", ApplyChannelModelMapping(`{"kk":"mid","mid":"kimi-k2.5"}`, "kk"))
	require.Equal(t, "other", ApplyChannelModelMapping(`{"kk":"kimi-k2.5"}`, "other"))
}

func TestModelMappedHelperAppliesMappingForTokenFactoryOpen(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(string(constant.ContextKeyChannelType), constant.ChannelTypeTokenFactoryOpen)
	c.Set("model_mapping", `{"kk":"kimi-k2.5"}`)
	c.Set(string(constant.ContextKeyTFOpenUpstreamChannelRoute), "uAb12")

	info := &relaycommon.RelayInfo{
		OriginModelName: "kk",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "kk",
		},
	}
	req := &dto.GeneralOpenAIRequest{Model: "kk"}

	err := ModelMappedHelper(c, info, req)
	require.NoError(t, err)
	require.Equal(t, "kk", info.OriginModelName)
	require.Equal(t, "kimi-k2.5/uAb12", info.UpstreamModelName)
	require.Equal(t, "kimi-k2.5/uAb12", req.Model)
	require.True(t, info.TFOpenUpstreamRouteApplied)
}

func TestModelMappedHelperOpenAIIgnoresTFOpenRouteSuffix(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// 同步过来的普通 OpenAI 渠道：即使上下文误带 route，也不应拼后缀
	c.Set(string(constant.ContextKeyChannelType), constant.ChannelTypeOpenAI)
	c.Set("model_mapping", `{"kk":"kimi-k2.5"}`)
	c.Set(string(constant.ContextKeyTFOpenUpstreamChannelRoute), "u1Y")

	info := &relaycommon.RelayInfo{
		OriginModelName: "kk",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "kk",
		},
	}
	req := &dto.GeneralOpenAIRequest{Model: "kk"}

	err := ModelMappedHelper(c, info, req)
	require.NoError(t, err)
	require.Equal(t, "kk", info.OriginModelName)
	require.Equal(t, "kimi-k2.5", info.UpstreamModelName)
	require.Equal(t, "kimi-k2.5", req.Model)
	require.False(t, info.TFOpenUpstreamRouteApplied)
}

func TestModelMappedHelperTokenFactoryOpenWithoutMappingKeepsOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set(string(constant.ContextKeyChannelType), constant.ChannelTypeTokenFactoryOpen)
	c.Set(string(constant.ContextKeyTFOpenUpstreamChannelRoute), "uAb12")

	info := &relaycommon.RelayInfo{
		OriginModelName: "kimi-k2.5",
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "kimi-k2.5",
		},
	}
	req := &dto.GeneralOpenAIRequest{Model: "kimi-k2.5"}

	err := ModelMappedHelper(c, info, req)
	require.NoError(t, err)
	require.Equal(t, "kimi-k2.5", info.OriginModelName)
	require.Equal(t, "kimi-k2.5/uAb12", info.UpstreamModelName)
}
