package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func TestTryDelegateTokenFactoryOpenChannelTestSendsUpstreamIdentity(t *testing.T) {
	var gotReq tfOpenSyncChannelTestRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/tf_open_sync/channel_test", r.URL.Path)
		require.Equal(t, "Bearer sk-test", r.Header.Get("Authorization"))
		require.Equal(t, "sk-test", r.Header.Get("X-TokenFactory-Open-Sync-Secret"))
		require.NoError(t, common.DecodeJson(r.Body, &gotReq))
		body, err := common.Marshal(tfOpenSyncChannelTestResponse{
			Success: true,
			Model:   gotReq.Model,
		})
		require.NoError(t, err)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	defer server.Close()

	otherInfo, err := common.Marshal(map[string]any{
		"upstream_channel_id":     158,
		"source":                  "tokenfactory_open",
		"upstream_route_slug":     "uAb12",
		"upstream_supplier_alias": "P0",
		"upstream_channel_no":     "c3",
	})
	require.NoError(t, err)
	channel := &model.Channel{
		Type:      constant.ChannelTypeTokenFactoryOpen,
		BaseURL:   &server.URL,
		Key:       "sk-test",
		OtherInfo: string(otherInfo),
	}

	result, handled := tryDelegateTokenFactoryOpenChannelTest(channel, "happyhorse-1.0-t2v", "image-generation", true)

	require.True(t, handled)
	require.NoError(t, result.localErr)
	require.Equal(t, "happyhorse-1.0-t2v", result.recordedModelName)
	require.Equal(t, "happyhorse-1.0-t2v", gotReq.Model)
	require.Equal(t, "image-generation", gotReq.EndpointType)
	require.True(t, gotReq.Stream)
	require.Equal(t, 158, gotReq.UpstreamChannelID)
	require.Equal(t, "uAb12", gotReq.UpstreamRouteSlug)
	require.Equal(t, "P0", gotReq.UpstreamSupplierAlias)
	require.Equal(t, "c3", gotReq.UpstreamChannelNo)
}

func TestTryDelegateTokenFactoryOpenChannelTestFallsBackWhenEndpointMissing(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	otherInfo, err := common.Marshal(map[string]any{
		"source":              "tokenfactory_open",
		"upstream_route_slug": "uAb12",
	})
	require.NoError(t, err)
	channel := &model.Channel{
		Type:      constant.ChannelTypeTokenFactoryOpen,
		BaseURL:   &server.URL,
		Key:       "sk-test",
		OtherInfo: string(otherInfo),
	}

	_, handled := tryDelegateTokenFactoryOpenChannelTest(channel, "gpt-4o", "", false)

	require.False(t, handled)
}

func TestResolveTFOpenSyncChannelTestChannelByRouteSlug(t *testing.T) {
	setupChannelImportTestDB(t)

	channel := &model.Channel{
		Name:      "upstream-video",
		Type:      constant.ChannelTypeOpenAIVideo,
		Status:    common.ChannelStatusEnabled,
		Models:    "happyhorse-1.0-t2v",
		RouteSlug: "uAb12",
	}
	require.NoError(t, model.DB.Create(channel).Error)

	resolved, err := resolveTFOpenSyncChannelTestChannel(tfOpenSyncChannelTestRequest{
		Model:             "happyhorse-1.0-t2v",
		UpstreamRouteSlug: "uAb12",
	})

	require.NoError(t, err)
	require.NotNil(t, resolved)
	require.Equal(t, channel.Id, resolved.Id)
	require.Equal(t, constant.ChannelTypeOpenAIVideo, resolved.Type)
}

func TestResolveTFOpenSyncChannelTestChannelByID(t *testing.T) {
	setupChannelImportTestDB(t)

	channel := &model.Channel{
		Name:   "upstream-kling",
		Type:   constant.ChannelTypeKling,
		Status: common.ChannelStatusEnabled,
		Models: "Kling-3.0",
	}
	require.NoError(t, model.DB.Create(channel).Error)

	resolved, err := resolveTFOpenSyncChannelTestChannel(tfOpenSyncChannelTestRequest{
		Model:             "Kling-3.0",
		UpstreamChannelID: channel.Id,
	})

	require.NoError(t, err)
	require.NotNil(t, resolved)
	require.Equal(t, channel.Id, resolved.Id)
	require.Equal(t, constant.ChannelTypeKling, resolved.Type)
}
