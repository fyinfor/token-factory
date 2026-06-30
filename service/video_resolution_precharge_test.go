package service

import (
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
)

func TestVideoResolutionParamFromRequest_TopLevel(t *testing.T) {
	req := relaycommon.TaskSubmitReq{Resolution: "1280x720"}
	require.Equal(t, "1280x720", videoResolutionParamFromRequest(req))
}

func TestVideoResolutionParamFromRequest_Metadata(t *testing.T) {
	req := relaycommon.TaskSubmitReq{
		Metadata: map[string]interface{}{"resolution": "480p"},
	}
	require.Equal(t, "480p", videoResolutionParamFromRequest(req))
}

func TestVideoResolutionParamFromRequest_SizeOnly(t *testing.T) {
	req := relaycommon.TaskSubmitReq{Size: "1280x720"}
	require.Equal(t, "", videoResolutionParamFromRequest(req))
}

func TestApplyPreChargeVideoResolution_UserInputPreserved(t *testing.T) {
	req := relaycommon.TaskSubmitReq{Resolution: "1280x720"}
	var resolution string
	var fromRequest bool
	applyPreChargeVideoResolution(req, &resolution, &fromRequest, "720p")
	require.True(t, fromRequest)
	require.Equal(t, "1280x720", resolution)
}

func TestApplyPreChargeVideoResolution_InferWhenMissing(t *testing.T) {
	req := relaycommon.TaskSubmitReq{Size: "1280x720"}
	var resolution string
	var fromRequest bool
	applyPreChargeVideoResolution(req, &resolution, &fromRequest, "1280x720")
	require.False(t, fromRequest)
	require.Equal(t, "720p", resolution)
}

func TestWriteVideoResolutionLogOther_FromInput(t *testing.T) {
	other := map[string]interface{}{}
	writeVideoResolutionLogOther(other, "1280x720", true, 1280, 720)
	require.Equal(t, "1280x720", other["video_resolution"])
	require.Equal(t, true, other["video_resolution_from_input"])
}
