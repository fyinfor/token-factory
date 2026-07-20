package setting

import "strings"

const AliyunGuardrailBlockedReply = `我无法回答这个问题，你可以尝试提供其他话题，我会尽力为你解答。`

var AliyunGuardrailEnabled = false
var AliyunGuardrailInputEnabled = true
var AliyunGuardrailOutputEnabled = true
var AliyunGuardrailVideoEnabled = false
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
