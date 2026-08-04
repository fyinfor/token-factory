package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSetupContextForSelectedChannelTFOpenRouteOnlyForType60(t *testing.T) {
	gin.SetMode(gin.TestMode)

	otherInfo, err := common.Marshal(map[string]any{
		"source":              "tokenfactory_open",
		"upstream_route_slug": "u1Y",
	})
	require.NoError(t, err)

	t.Run("openai synced channel does not set route", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		base := "https://example.com"
		ch := &model.Channel{
			Id:        1016,
			Type:      constant.ChannelTypeOpenAI,
			Name:      "联通云",
			Key:       "sk-test",
			BaseURL:   &base,
			OtherInfo: string(otherInfo),
		}
		err := SetupContextForSelectedChannel(c, ch, "kimi-k2.5")
		require.Nil(t, err)
		require.Empty(t, common.GetContextKeyString(c, constant.ContextKeyTFOpenUpstreamChannelRoute))
	})

	t.Run("tokenfactory open channel sets route", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		base := "https://upstream-tf.example.com"
		ch := &model.Channel{
			Id:        2001,
			Type:      constant.ChannelTypeTokenFactoryOpen,
			Name:      "建站上游",
			Key:       "sk-test",
			BaseURL:   &base,
			OtherInfo: string(otherInfo),
		}
		err := SetupContextForSelectedChannel(c, ch, "kimi-k2.5")
		require.Nil(t, err)
		require.Equal(t, "u1Y", common.GetContextKeyString(c, constant.ContextKeyTFOpenUpstreamChannelRoute))
	})
}
