package tencentvod

import (
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
)

func TestEnsureTencentOutputConfigAudio_DefaultEnabled(t *testing.T) {
	oc := &AigcVideoOutputConfig{StorageMode: "Temporary"}
	ensureTencentOutputConfigAudio(oc, true)
	require.Equal(t, "Enabled", oc.AudioGeneration)
}

func TestEnsureTencentOutputConfigAudio_Disabled(t *testing.T) {
	oc := &AigcVideoOutputConfig{}
	ensureTencentOutputConfigAudio(oc, false)
	require.Equal(t, "Disabled", oc.AudioGeneration)
}

func TestEnsureTencentOutputConfigAudio_PreserveExplicit(t *testing.T) {
	oc := &AigcVideoOutputConfig{AudioGeneration: "Disabled"}
	ensureTencentOutputConfigAudio(oc, true)
	require.Equal(t, "Disabled", oc.AudioGeneration)
}

func TestOutputConfigFromTaskSubmitReq_AudioFromMetadata(t *testing.T) {
	req := relaycommon.TaskSubmitReq{
		Duration:   5,
		Resolution: "1080P",
		Ratio:      "16:9",
		Metadata: map[string]interface{}{
			"generate_audio": false,
		},
	}
	oc, err := outputConfigFromTaskSubmitReq(req)
	require.NoError(t, err)
	require.Equal(t, "Disabled", oc.AudioGeneration)
}

func TestOutputConfigFromTaskSubmitReq_AudioDefaultTrue(t *testing.T) {
	req := relaycommon.TaskSubmitReq{
		Duration:   5,
		Resolution: "1080P",
		Ratio:      "16:9",
		Metadata:   map[string]interface{}{},
	}
	oc, err := outputConfigFromTaskSubmitReq(req)
	require.NoError(t, err)
	require.Equal(t, "Enabled", oc.AudioGeneration)
}
