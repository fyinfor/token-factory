package service

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRouteOrderedMaxRetries(t *testing.T) {
	orig := common.RetryTimes
	t.Cleanup(func() { common.RetryTimes = orig })

	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	common.RetryTimes = 0
	require.Equal(t, 0, RouteOrderedMaxRetries(c))

	common.SetContextKey(c, constant.ContextKeySmartRouteChannelOrder, []int{666, 667})
	require.Equal(t, 1, RouteOrderedMaxRetries(c))

	common.RetryTimes = 3
	require.Equal(t, 3, RouteOrderedMaxRetries(c))

	common.SetContextKey(c, constant.ContextKeySmartRouteChannelOrder, []int{1, 2, 3, 4, 5})
	require.Equal(t, 4, RouteOrderedMaxRetries(c))
}

func TestPickNextOrderedRouteChannelSkipsUsed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("use_channel", []string{"666"})

	order := []int{666, 667}
	_, picked := PickNextOrderedRouteChannel(c, order)
	require.False(t, picked, "should not pick without channel cache/db")
}
