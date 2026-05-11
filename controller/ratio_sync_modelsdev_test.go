package controller

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConvertModelsDevToRatioData_OnlyOfficialProviders(t *testing.T) {
	payload := `{
		"openai": {
			"id": "openai",
			"name": "OpenAI",
			"models": {
				"gpt-4o": {
					"cost": {"input": 5, "output": 15}
				}
			}
		},
		"moonshotai": {
			"id": "moonshotai",
			"api": "https://api.moonshot.ai/v1",
			"name": "Moonshot AI",
			"models": {
				"kimi-latest": {
					"cost": {"input": 12, "output": 36}
				}
			}
		},
		"moonshotai-cn": {
			"id": "moonshotai-cn",
			"api": "https://api.moonshot.cn/v1",
			"name": "Moonshot AI CN",
			"models": {
				"kimi-k2-0711-preview": {
					"cost": {"input": 10, "output": 20}
				}
			}
		},
		"openai-proxy": {
			"id": "openai-proxy",
			"api": "https://proxy.example.com/v1",
			"name": "OpenAI Proxy",
			"models": {
				"gpt-4.1-mini": {
					"cost": {"input": 1, "output": 2}
				}
			}
		}
	}`

	converted, err := convertModelsDevToRatioData(strings.NewReader(payload))
	require.NoError(t, err)

	modelRatios := converted["model_ratio"].(map[string]any)
	completionRatios := converted["completion_ratio"].(map[string]any)

	require.Contains(t, modelRatios, "gpt-4o")
	require.Contains(t, modelRatios, "kimi-latest")
	require.NotContains(t, modelRatios, "kimi-k2-0711-preview")
	require.NotContains(t, modelRatios, "gpt-4.1-mini")

	require.Equal(t, 2.5, modelRatios["gpt-4o"])
	require.Equal(t, 6.0, modelRatios["kimi-latest"])
	require.Equal(t, 3.0, completionRatios["gpt-4o"])
	require.Equal(t, 3.0, completionRatios["kimi-latest"])
}

func TestOldEffectiveForUpstream_ChannelDoesNotFallbackToGlobal(t *testing.T) {
	localData := map[string]any{
		"model_ratio": map[string]float64{
			"unconfigured-model": 9.9,
		},
	}

	require.Nil(t, oldEffectiveForUpstream(12345, "model_ratio", "unconfigured-model", localData))
	require.Equal(t, 9.9, oldEffectiveForUpstream(0, "model_ratio", "unconfigured-model", localData))
}

func TestOldChannelValueOrNil_ZeroDisplaysUnset(t *testing.T) {
	require.Nil(t, oldChannelValueOrNil(0))
	require.Nil(t, oldChannelValueOrNil(1e-10))
	require.Equal(t, 1.25, oldChannelValueOrNil(1.25))
}

func TestConvertChannelPricingItemsToRatioData_UsesMatchedChannelList(t *testing.T) {
	pricingItems := []pricingItem{
		{
			ModelName:        "gpt-test",
			QuotaType:        0,
			ModelRatio:       0,
			CompletionRatio:  0,
			CacheRatio:       0,
			CreateCacheRatio: 0,
			ModelPrice:       0,
			ChannelList: []pricingChannelItem{
				{
					ChannelID:        49,
					QuotaType:        0,
					ModelRatio:       9,
					CompletionRatio:  9,
					CacheRatio:       9,
					CreateCacheRatio: 9,
					ModelPrice:       9,
				},
				{
					ChannelID:        50,
					QuotaType:        0,
					ModelRatio:       2.125,
					CompletionRatio:  5,
					CacheRatio:       0.1,
					CreateCacheRatio: 1.25,
					ModelPrice:       0,
				},
			},
		},
	}

	converted := convertChannelPricingItemsToRatioData(pricingItems, 50)

	require.Equal(t, 2.125, converted["model_ratio"].(map[string]any)["gpt-test"])
	require.Equal(t, 5.0, converted["completion_ratio"].(map[string]any)["gpt-test"])
	require.Equal(t, 0.1, converted["cache_ratio"].(map[string]any)["gpt-test"])
	require.Equal(t, 1.25, converted["create_cache_ratio"].(map[string]any)["gpt-test"])
	require.Equal(t, 0.0, converted["model_price"].(map[string]any)["gpt-test"])
}

func TestConvertChannelPricingItemsToRatioData_ExtractsAllFieldsWithoutQuotaTypeFiltering(t *testing.T) {
	pricingItems := []pricingItem{
		{
			ModelName:  "image-test",
			QuotaType:  0,
			ModelPrice: 0,
			ChannelList: []pricingChannelItem{
				{
					ChannelID:        50,
					QuotaType:        0,
					ModelRatio:       3,
					ModelPrice:       0.25,
					CompletionRatio:  4,
					CacheRatio:       0.2,
					CreateCacheRatio: 1.5,
				},
			},
		},
	}

	converted := convertChannelPricingItemsToRatioData(pricingItems, 50)

	require.Equal(t, 0.25, converted["model_price"].(map[string]any)["image-test"])
	require.Equal(t, 3.0, converted["model_ratio"].(map[string]any)["image-test"])
	require.Equal(t, 4.0, converted["completion_ratio"].(map[string]any)["image-test"])
	require.Equal(t, 0.2, converted["cache_ratio"].(map[string]any)["image-test"])
	require.Equal(t, 1.5, converted["create_cache_ratio"].(map[string]any)["image-test"])
}

func TestConvertChannelPricingItemsToRatioData_DuplicateModelKeepsNonZeroValues(t *testing.T) {
	pricingItems := []pricingItem{
		{
			ModelName: "gpt-dup",
			ChannelList: []pricingChannelItem{
				{
					ChannelID:        50,
					ModelRatio:       2.125,
					CompletionRatio:  5,
					CacheRatio:       0.1,
					CreateCacheRatio: 1.25,
					ModelPrice:       0.3,
				},
			},
		},
		{
			ModelName: "gpt-dup",
			ChannelList: []pricingChannelItem{
				{
					ChannelID:        50,
					ModelRatio:       0,
					CompletionRatio:  0,
					CacheRatio:       0,
					CreateCacheRatio: 0,
					ModelPrice:       0,
				},
			},
		},
	}

	converted := convertChannelPricingItemsToRatioData(pricingItems, 50)

	require.Equal(t, 2.125, converted["model_ratio"].(map[string]any)["gpt-dup"])
	require.Equal(t, 5.0, converted["completion_ratio"].(map[string]any)["gpt-dup"])
	require.Equal(t, 0.1, converted["cache_ratio"].(map[string]any)["gpt-dup"])
	require.Equal(t, 1.25, converted["create_cache_ratio"].(map[string]any)["gpt-dup"])
	require.Equal(t, 0.3, converted["model_price"].(map[string]any)["gpt-dup"])
}

func TestConvertChannelPricingItemsToRatioData_DuplicateModelCanFillMissingNonZeroFields(t *testing.T) {
	pricingItems := []pricingItem{
		{
			ModelName: "gpt-dup",
			ChannelList: []pricingChannelItem{
				{
					ChannelID:       50,
					ModelRatio:      2.125,
					CompletionRatio: 0,
				},
			},
		},
		{
			ModelName: "gpt-dup",
			ChannelList: []pricingChannelItem{
				{
					ChannelID:        50,
					ModelRatio:       0,
					CompletionRatio:  5,
					CacheRatio:       0.1,
					CreateCacheRatio: 1.25,
				},
			},
		},
	}

	converted := convertChannelPricingItemsToRatioData(pricingItems, 50)

	require.Equal(t, 2.125, converted["model_ratio"].(map[string]any)["gpt-dup"])
	require.Equal(t, 5.0, converted["completion_ratio"].(map[string]any)["gpt-dup"])
	require.Equal(t, 0.1, converted["cache_ratio"].(map[string]any)["gpt-dup"])
	require.Equal(t, 1.25, converted["create_cache_ratio"].(map[string]any)["gpt-dup"])
}
