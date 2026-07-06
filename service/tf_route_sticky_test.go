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
	ordered := []int{1, 43}
	isEnabled := func(id int) bool { return true }

	cUserA, _ := gin.CreateTestContext(httptest.NewRecorder())
	cUserA.Set("id", 1001)
	id, ok := TFRoutePickChannel(cUserA, groupKey, group, ordered, isEnabled)
	require.True(t, ok)
	require.Equal(t, 1, id)

	RefreshTFRouteStickyChannel(cUserA, 43)
	RecordTFRouteResult(cUserA, 43, true)

	idA, ok := TFRoutePickChannel(cUserA, groupKey, group, ordered, isEnabled)
	require.True(t, ok)
	require.Equal(t, 43, idA, "user A should stick to channel 43 after failover success")

	cUserB, _ := gin.CreateTestContext(httptest.NewRecorder())
	cUserB.Set("id", 2002)
	idB, ok := TFRoutePickChannel(cUserB, groupKey, group, ordered, isEnabled)
	require.True(t, ok)
	require.Equal(t, 1, idB, "user B should pick cheapest channel, unaffected by user A sticky")
}

func TestTfStickyKeyIncludesUserID(t *testing.T) {
	require.Equal(t, "glm-5.2#default#u70", tfStickyKey("glm-5.2", "default", 70))
	require.NotEqual(t, tfStickyKey("glm-5.2", "default", 70), tfStickyKey("glm-5.2", "default", 71))
}
