package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type priceExportAPIResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    PriceExportData `json:"data"`
}

type priceImportAPIResponse struct {
	Success bool              `json:"success"`
	Message string            `json:"message"`
	Data    PriceImportResult `json:"data"`
}

func setupPriceExportImportTest(t *testing.T) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	previousDB := model.DB
	previousSQLite := common.UsingSQLite
	previousMySQL := common.UsingMySQL
	previousPostgreSQL := common.UsingPostgreSQL
	common.OptionMapRWMutex.RLock()
	previousOptionMap := common.OptionMap
	common.OptionMapRWMutex.RUnlock()

	t.Cleanup(func() {
		model.DB = previousDB
		common.UsingSQLite = previousSQLite
		common.UsingMySQL = previousMySQL
		common.UsingPostgreSQL = previousPostgreSQL
		common.OptionMapRWMutex.Lock()
		common.OptionMap = previousOptionMap
		common.OptionMapRWMutex.Unlock()
		_ = ratio_setting.UpdateASRPriceByJSONString("{}")
		_ = ratio_setting.UpdateVideoPricingRulesByJSONString("{}")
		_ = ratio_setting.UpdateChannelVideoPricingRulesByJSONString("{}")
	})

	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false

	db, err := gorm.Open(
		sqlite.Open(fmt.Sprintf("file:price_export_import_%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))),
		&gorm.Config{},
	)
	require.NoError(t, err)
	model.DB = db
	require.NoError(t, db.AutoMigrate(&model.Option{}, &model.Channel{}))

	common.OptionMapRWMutex.Lock()
	common.OptionMap = map[string]string{}
	common.OptionMapRWMutex.Unlock()
}

func setPriceOption(t *testing.T, key, value string) {
	t.Helper()
	require.NoError(t, model.UpdateOption(key, value))
}

func decodeVideoRules(t *testing.T, raw json.RawMessage) ratio_setting.VideoPricingRules {
	t.Helper()
	var rules ratio_setting.VideoPricingRules
	require.NoError(t, common.Unmarshal(raw, &rules))
	return rules
}

func sampleVideoTokenAndUpscaleRulesJSON() string {
	return `{
		"seedance-1.0": {
			"text_to_video_per_token": [{"resolution":"1280x720","has_audio":false,"price":0.15}],
			"image_to_video_per_token": [{"resolution":"1920x1080","has_audio":true,"price":0.31}],
			"video_upscale_per_second": [{"resolution":"720p","source_resolution":"480p","price":0.02}]
		}
	}`
}

func TestExportPricesIncludesVideoTokenUpscaleAndASR(t *testing.T) {
	setupPriceExportImportTest(t)

	setPriceOption(t, "ASRPrice", `{"paraformer-v2":0.00012}`)
	setPriceOption(t, "VideoPricingRules", sampleVideoTokenAndUpscaleRulesJSON())

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/admin/price/export", nil)
	ExportPrices(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	var resp priceExportAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &resp))
	require.True(t, resp.Success)
	require.InDelta(t, 0.00012, resp.Data.GlobalPrices.ASRPrice["paraformer-v2"], 1e-12)

	rules := decodeVideoRules(t, resp.Data.GlobalPrices.VideoPricingRules["seedance-1.0"])
	require.Len(t, rules.TextToVideoPerToken, 1)
	require.Equal(t, "1280x720", rules.TextToVideoPerToken[0].Resolution)
	require.InDelta(t, 0.15, rules.TextToVideoPerToken[0].Price, 1e-12)
	require.Len(t, rules.ImageToVideoPerToken, 1)
	require.True(t, rules.ImageToVideoPerToken[0].HasAudio)
	require.Len(t, rules.VideoUpscalePerSecond, 1)
	require.Equal(t, "720p", rules.VideoUpscalePerSecond[0].Resolution)
	require.Equal(t, "480p", rules.VideoUpscalePerSecond[0].SourceResolution)
	require.InDelta(t, 0.02, rules.VideoUpscalePerSecond[0].Price, 1e-12)
}

func TestImportPricesMergesVideoTokenUpscaleAndASR(t *testing.T) {
	setupPriceExportImportTest(t)

	setPriceOption(t, "ASRPrice", `{"paraformer-v2":0.0001,"keep-asr":0.002}`)
	setPriceOption(t, "VideoPricingRules", `{
		"keep-video": {"text_to_video_per_second":[{"resolution":"720p","has_audio":false,"price":0.01}]}
	}`)

	tokenRules, err := common.Marshal(map[string]any{
		"text_to_video_per_token": []map[string]any{
			{"resolution": "1280x720", "has_audio": false, "price": 0.22},
		},
		"video_upscale_per_second": []map[string]any{
			{"resolution": "1080p", "source_resolution": "720p", "price": 0.08},
		},
	})
	require.NoError(t, err)

	payload := PriceExportData{
		GlobalPrices: PriceExportModelMaps{
			ASRPrice: map[string]float64{"paraformer-v2": 0.0005, "new-asr": 0.003},
			VideoPricingRules: map[string]json.RawMessage{
				"seedance-1.0": tokenRules,
			},
		},
	}
	body, err := common.Marshal(payload)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/admin/price/import", bytes.NewReader(body))
	ImportPrices(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	var resp priceImportAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &resp))
	require.True(t, resp.Success)
	require.Greater(t, resp.Data.GlobalAdded+resp.Data.GlobalUpdated, 0)

	asrPrice, ok := ratio_setting.GetASRPrice("paraformer-v2")
	require.True(t, ok)
	require.InDelta(t, 0.0005, asrPrice, 1e-12)
	keepASR, ok := ratio_setting.GetASRPrice("keep-asr")
	require.True(t, ok)
	require.InDelta(t, 0.002, keepASR, 1e-12)
	newASR, ok := ratio_setting.GetASRPrice("new-asr")
	require.True(t, ok)
	require.InDelta(t, 0.003, newASR, 1e-12)

	rules, ok := ratio_setting.GetVideoPricingRules("seedance-1.0")
	require.True(t, ok)
	require.True(t, ratio_setting.HasUsableVideoPerTokenRules(rules))
	require.True(t, ratio_setting.HasUsableVideoUpscaleRules(rules))
	require.InDelta(t, 0.22, rules.TextToVideoPerToken[0].Price, 1e-12)
	require.Equal(t, "1080p", rules.VideoUpscalePerSecond[0].Resolution)

	keepRules, ok := ratio_setting.GetVideoPricingRules("keep-video")
	require.True(t, ok)
	require.Len(t, keepRules.TextToVideoPerSecond, 1)
}

func TestImportPricesASROnlyIsNotRejected(t *testing.T) {
	setupPriceExportImportTest(t)

	payload := PriceExportData{
		GlobalPrices: PriceExportModelMaps{
			ASRPrice: map[string]float64{"paraformer-v2": 0.0008},
		},
	}
	body, err := common.Marshal(payload)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/admin/price/import", bytes.NewReader(body))
	ImportPrices(c)

	require.Equal(t, http.StatusOK, recorder.Code)
	var resp priceImportAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &resp))
	require.True(t, resp.Success)
	price, ok := ratio_setting.GetASRPrice("paraformer-v2")
	require.True(t, ok)
	require.InDelta(t, 0.0008, price, 1e-12)
}

func TestExportImportChannelVideoTokenAndUpscale(t *testing.T) {
	setupPriceExportImportTest(t)

	channel := &model.Channel{Name: "video-channel", Type: 1, Status: common.ChannelStatusEnabled}
	require.NoError(t, model.DB.Create(channel).Error)
	idStr := fmt.Sprintf("%d", channel.Id)

	channelRules := fmt.Sprintf(`{
		"%s": {
			"seedance-1.0": {
				"text_to_video_per_token": [{"resolution":"1280x720","has_audio":false,"price":0.18}],
				"video_upscale_per_second": [{"resolution":"1080p","source_resolution":"720p","price":0.05}]
			}
		}
	}`, idStr)
	setPriceOption(t, "ChannelVideoPricingRules", channelRules)

	exportRecorder := httptest.NewRecorder()
	exportCtx, _ := gin.CreateTestContext(exportRecorder)
	exportCtx.Request = httptest.NewRequest(http.MethodGet, "/api/admin/price/export", nil)
	ExportPrices(exportCtx)
	require.Equal(t, http.StatusOK, exportRecorder.Code)

	var exportResp priceExportAPIResponse
	require.NoError(t, common.Unmarshal(exportRecorder.Body.Bytes(), &exportResp))
	require.True(t, exportResp.Success)
	require.Len(t, exportResp.Data.Channels, 1)
	require.Equal(t, "video-channel", exportResp.Data.Channels[0].ChannelName)
	chRules := decodeVideoRules(t, exportResp.Data.Channels[0].Models.VideoPricingRules["seedance-1.0"])
	require.True(t, ratio_setting.HasUsableVideoPerTokenRules(chRules))
	require.True(t, ratio_setting.HasUsableVideoUpscaleRules(chRules))

	updatedRules, err := common.Marshal(map[string]any{
		"text_to_video_per_token": []map[string]any{
			{"resolution": "1280x720", "has_audio": false, "price": 0.4},
		},
		"video_upscale_per_second": []map[string]any{
			{"resolution": "1080p", "source_resolution": "720p", "price": 0.09},
		},
	})
	require.NoError(t, err)
	importPayload := PriceExportData{
		Channels: []PriceExportChannelEntry{{
			ChannelName: "video-channel",
			Models: PriceExportModelMaps{
				VideoPricingRules: map[string]json.RawMessage{
					"seedance-1.0": updatedRules,
				},
			},
		}},
	}
	body, err := common.Marshal(importPayload)
	require.NoError(t, err)

	importRecorder := httptest.NewRecorder()
	importCtx, _ := gin.CreateTestContext(importRecorder)
	importCtx.Request = httptest.NewRequest(http.MethodPost, "/api/admin/price/import", bytes.NewReader(body))
	ImportPrices(importCtx)
	require.Equal(t, http.StatusOK, importRecorder.Code)

	imported, ok := ratio_setting.GetChannelVideoPricingRules(channel.Id, "seedance-1.0")
	require.True(t, ok)
	require.InDelta(t, 0.4, imported.TextToVideoPerToken[0].Price, 1e-12)
	require.InDelta(t, 0.09, imported.VideoUpscalePerSecond[0].Price, 1e-12)
}
