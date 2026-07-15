package tencentvod

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
)

func TestValidateCreateAigcVideoTaskRequest_OK(t *testing.T) {
	req := &CreateAigcVideoTaskRequest{
		SubAppId:     1500044236,
		ModelName:    "GV",
		ModelVersion: "3.1",
		FileInfos: []AigcVideoFileInfo{
			{Type: "Url", Url: "https://example.com/a.jpeg"},
			{Type: "Base64", Base64: "xxxx"},
		},
		Prompt: "微笑的向我走来",
		OutputConfig: &AigcVideoOutputConfig{
			StorageMode: "Temporary",
			Resolution:  "1080P",
			Duration:    5,
			AspectRatio: "16:9",
		},
	}
	require.NoError(t, validateCreateAigcVideoTaskRequest(req, 1500044236))
}

func TestValidateCreateAigcVideoTaskRequest_MissingOutputConfig(t *testing.T) {
	req := &CreateAigcVideoTaskRequest{
		SubAppId:     1,
		ModelName:    "GV",
		ModelVersion: "3.1",
	}
	err := validateCreateAigcVideoTaskRequest(req, 1)
	require.Error(t, err)
	require.Contains(t, err.Error(), "OutputConfig")
}

func TestValidateFileInfos_UrlBase64(t *testing.T) {
	require.NoError(t, validateFileInfos([]AigcVideoFileInfo{
		{Type: "Url", Url: "https://a.com/1.png"},
		{Type: "Base64", Base64: "abc"},
	}))
	require.Error(t, validateFileInfos([]AigcVideoFileInfo{
		{Type: "Url", Url: ""},
	}))
}

func TestParseInputOutputConfigFromDescribeBody(t *testing.T) {
	raw := []byte(`{
	  "Response": {
	    "Status": "FINISH",
	    "AigcVideoTask": {
	      "Status": "FINISH",
	      "ErrCode": 0,
	      "Input": {
	        "Prompt": "一只可爱的橘猫",
	        "ModelName": "Kling",
	        "ModelVersion": "3.0",
	        "OutputConfig": {
	          "StorageMode": "Temporary",
	          "Duration": 15,
	          "Resolution": "720P",
	          "AspectRatio": "16:9"
	        }
	      },
	      "Output": {}
	    }
	  }
	}`)
	oc := ParseInputOutputConfigFromDescribeBody(raw)
	require.NotNil(t, oc)
	spec := oc.ToBillingSpec()
	require.Equal(t, "720P", spec.Resolution)
	require.Equal(t, 15, spec.Duration)
	require.Equal(t, "16:9", spec.AspectRatio)
}

func TestBillingSpecMismatch(t *testing.T) {
	submitted := BillingOutputSpec{Resolution: "1080P", Duration: 5, AspectRatio: "16:9"}
	actual := BillingOutputSpec{Resolution: "720P", Duration: 15, AspectRatio: "16:9"}
	mismatched, reasons := billingSpecMismatch(submitted, actual, true)
	require.True(t, mismatched)
	require.Contains(t, reasons, "resolution")
	require.Contains(t, reasons, "duration")
}

func TestAdjustBillingOnComplete_DurationMismatch(t *testing.T) {
	a := &TaskAdaptor{}
	input, _ := common.Marshal(map[string]any{
		"prompt":     "x",
		"model":      "GV-3.1",
		"duration":   5,
		"resolution": "1080P",
		"ratio":      "16:9",
	})
	task := &model.Task{
		TaskID: "t1",
		Status: model.TaskStatusSuccess,
		Quota:  1000,
		Properties: model.Properties{
			Input: string(input),
		},
	}
	ti := &relaycommon.TaskInfo{
		Resolution: "720P",
		Duration:   15,
		Ratio:      "16:9",
	}
	actual := a.AdjustBillingOnComplete(task, ti)
	require.Equal(t, 3000, actual) // 1000 * 15/5
}

func TestParseTaskResult_FillsBillingSpec(t *testing.T) {
	a := &TaskAdaptor{}
	raw := []byte(`{
	  "Response": {
	    "Status": "FINISH",
	    "AigcVideoTask": {
	      "Input": {
	        "OutputConfig": {
	          "Duration": 15,
	          "Resolution": "720P",
	          "AspectRatio": "16:9"
	        }
	      },
	      "Output": {
	        "FileInfos": [{"FileUrl": "https://example.com/out.mp4"}]
	      }
	    }
	  }
	}`)
	ti, err := a.ParseTaskResult(raw)
	require.NoError(t, err)
	require.Equal(t, string(model.TaskStatusSuccess), ti.Status)
	require.Equal(t, "720P", ti.Resolution)
	require.Equal(t, 15, ti.Duration)
	require.Equal(t, "16:9", ti.Ratio)
	require.Equal(t, "https://example.com/out.mp4", ti.Url)
}
