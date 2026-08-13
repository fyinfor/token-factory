package controller

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setChangelogEnabledForTest(t *testing.T, enabled bool) {
	t.Helper()
	common.OptionMapRWMutex.Lock()
	previous := common.OptionMap
	value := "false"
	if enabled {
		value = "true"
	}
	common.OptionMap = map[string]string{"ChangelogEnabled": value}
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = previous
		common.OptionMapRWMutex.Unlock()
	})
}

func callPublicChangelogHandler() *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ListPublicChangelogs(ctx)
	return recorder
}

func TestListPublicChangelogsRespectsEnabledOption(t *testing.T) {
	t.Run("disabled returns no entries", func(t *testing.T) {
		setChangelogEnabledForTest(t, false)
		recorder := callPublicChangelogHandler()

		require.Equal(t, 200, recorder.Code)
		var response struct {
			Success bool               `json:"success"`
			Data    []*model.Changelog `json:"data"`
		}
		require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
		require.True(t, response.Success)
		require.Empty(t, response.Data)
	})

	t.Run("enabled returns saved entries", func(t *testing.T) {
		setChangelogEnabledForTest(t, true)
		previousDB := model.DB
		db, err := gorm.Open(sqlite.Open("file:changelog_test?mode=memory&cache=shared"), &gorm.Config{})
		require.NoError(t, err)
		model.DB = db
		t.Cleanup(func() { model.DB = previousDB })
		require.NoError(t, db.AutoMigrate(&model.Changelog{}))
		require.NoError(t, model.CreateChangelog(&model.Changelog{
			Date:    "2026-08-13",
			Content: "Visible entry",
		}))

		recorder := callPublicChangelogHandler()

		require.Equal(t, 200, recorder.Code)
		var response struct {
			Success bool               `json:"success"`
			Data    []*model.Changelog `json:"data"`
		}
		require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
		require.True(t, response.Success)
		require.Len(t, response.Data, 1)
		require.Equal(t, "Visible entry", response.Data[0].Content)
	})
}
