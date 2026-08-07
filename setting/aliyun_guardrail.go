package setting

import "strings"

const AliyunGuardrailBlockedReply = `抱歉，我无法协助处理这个请求。你可以换个安全、合规的方向提问，我会尽力帮助。`

var AliyunGuardrailEnabled = false
var AliyunGuardrailInputEnabled = true
var AliyunGuardrailOutputEnabled = true
var AliyunGuardrailVideoEnabled = false
var AliyunGuardrailHidePlaygroundMediaTabs = false
var AliyunGuardrailAccessKeyID = ``
var AliyunGuardrailAccessKeySecret = ``
var AliyunGuardrailRegionID = `cn-shanghai`

func AliyunGuardrailConfigured() bool {
	return strings.TrimSpace(AliyunGuardrailAccessKeyID) != `` && strings.TrimSpace(AliyunGuardrailAccessKeySecret) != ``
}

func ShouldCheckAliyunGuardrailInput() bool {
	return AliyunGuardrailEnabled && AliyunGuardrailInputEnabled && AliyunGuardrailConfigured()
}

func ShouldCheckAliyunGuardrailOutput() bool {
	return AliyunGuardrailEnabled && AliyunGuardrailOutputEnabled && AliyunGuardrailConfigured()
}

func ShouldCheckAliyunGuardrailVideo() bool {
	return ShouldCheckAliyunGuardrailOutput() && AliyunGuardrailVideoEnabled
}
