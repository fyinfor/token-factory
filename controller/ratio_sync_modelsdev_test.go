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
