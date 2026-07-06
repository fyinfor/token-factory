package service

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestTFRoutePickChannelPerUserIsolation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	groupKey := "test-isolation-" + t.Name()
	group := "default"
	strategy := "price"
	ordered := []int{1, 43}
	isEnabled := func(id int) bool { return true }

	cUserA, _ := gin.CreateTestContext(httptest.NewRecorder())
	cUserA.Set("id", 1001)
	id, ok := TFRoutePickChannel(cUserA, groupKey, group, strategy, ordered, isEnabled)
	require.True(t, ok)
	require.Equal(t, 1, id)

	RefreshTFRouteStickyChannel(cUserA, 43)
	RecordTFRouteResult(cUserA, 43, true)

	idA, ok := TFRoutePickChannel(cUserA, groupKey, group, strategy, ordered, isEnabled)
	require.True(t, ok)
	require.Equal(t, 43, idA, "user A should stick to channel 43 after failover success")

	cUserB, _ := gin.CreateTestContext(httptest.NewRecorder())
	cUserB.Set("id", 2002)
	idB, ok := TFRoutePickChannel(cUserB, groupKey, group, strategy, ordered, isEnabled)
	require.True(t, ok)
	require.Equal(t, 1, idB, "user B should pick cheapest channel, unaffected by user A sticky")
}

func TestTFRoutePickChannelModeSwitchImmediateEffect(t *testing.T) {
	gin.SetMode(gin.TestMode)

	groupKey := "test-mode-switch-" + t.Name()
	group := "default"
	isEnabled := func(id int) bool { return true }

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("id", 70)

	// 权重模式：渠道 43 排第一
	weightOrdered := []int{43, 1}
	id, ok := TFRoutePickChannel(c, groupKey, group, "weight", weightOrdered, isEnabled)
	require.True(t, ok)
	require.Equal(t, 43, id)

	RefreshTFRouteStickyChannel(c, 43)
	RecordTFRouteResult(c, 43, true)

	idWeight, ok := TFRoutePickChannel(c, groupKey, group, "weight", weightOrdered, isEnabled)
	require.True(t, ok)
	require.Equal(t, 43, idWeight, "same mode should reuse sticky channel")

	// 切换为性价比模式：渠道 1 排第一，应忽略权重模式的黏性
	priceOrdered := []int{1, 43}
	idPrice, ok := TFRoutePickChannel(c, groupKey, group, "price", priceOrdered, isEnabled)
	require.True(t, ok)
	require.Equal(t, 1, idPrice, "mode switch should pick first channel under new strategy immediately")
}

func TestTfStickyKeyIncludesUserIDAndStrategy(t *testing.T) {
	require.Equal(t, "glm-5.2#default#price#u70", tfStickyKey("glm-5.2", "default", "price", 70))
	require.NotEqual(t, tfStickyKey("glm-5.2", "default", "weight", 70), tfStickyKey("glm-5.2", "default", "price", 70))
	require.NotEqual(t, tfStickyKey("glm-5.2", "default", "price", 70), tfStickyKey("glm-5.2", "default", "price", 71))
}
