package tencentvod

import (
	"strings"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

// parseGenerateAudioFromMetadata 解析操练场/通用 metadata.generate_audio；缺省 true（生成音频）。
func parseGenerateAudioFromMetadata(meta map[string]interface{}) bool {
	if meta == nil {
		return true
	}
	for _, key := range []string{"generate_audio", "has_audio"} {
		v, ok := meta[key]
		if !ok {
			continue
		}
		switch x := v.(type) {
		case bool:
			return x
		case string:
			s := strings.TrimSpace(strings.ToLower(x))
			if s == "false" || s == "0" || s == "no" || s == "disabled" {
				return false
			}
			if s == "true" || s == "1" || s == "yes" || s == "enabled" {
				return true
			}
		}
	}
	return true
}

// ensureTencentOutputConfigAudio 将 generate_audio 映射为腾讯云 OutputConfig.AudioGeneration（Enabled/Disabled）。
// 原生请求已显式携带 AudioGeneration 时不覆盖。
func ensureTencentOutputConfigAudio(oc *AigcVideoOutputConfig, generateAudio bool) {
	if oc == nil {
		return
	}
	if strings.TrimSpace(oc.AudioGeneration) != "" {
		return
	}
	if generateAudio {
		oc.AudioGeneration = "Enabled"
	} else {
		oc.AudioGeneration = "Disabled"
	}
}

func applyGenerateAudioFromTaskSubmitReq(oc *AigcVideoOutputConfig, req relaycommon.TaskSubmitReq) {
	ensureTencentOutputConfigAudio(oc, parseGenerateAudioFromMetadata(req.Metadata))
}

// enrichNativeCreateRequestAudio 原生 CreateAigcVideoTask 请求：OutputConfig 未指定 AudioGeneration 时默认 Enabled。
func enrichNativeCreateRequestAudio(req *CreateAigcVideoTaskRequest, generateAudio bool) {
	if req == nil {
		return
	}
	if req.OutputConfig == nil {
		req.OutputConfig = &AigcVideoOutputConfig{StorageMode: "Temporary"}
	}
	ensureTencentOutputConfigAudio(req.OutputConfig, generateAudio)
}
