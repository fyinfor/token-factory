package setting

import "strings"

var CheckSensitiveEnabled = true
var CheckSensitiveOnPromptEnabled = true

var TencentTMSModerationEnabled = false
var TencentIMSModerationEnabled = false
var TencentTMSOutputModerationEnabled = false
var TencentIMSOutputModerationEnabled = false
var TencentTMSSecretID = ""
var TencentTMSSecretKey = ""
var TencentTMSRegion = "ap-guangzhou"
var TencentTMSBizType = "TencentCloudDefault"
var TencentIMSBizType = "TencentCloudDefault"

//var CheckSensitiveOnCompletionEnabled = true

// StopOnSensitiveEnabled 如果检测到敏感词，是否立刻停止生成，否则替换敏感词
var StopOnSensitiveEnabled = true

// StreamCacheQueueLength 流模式缓存队列长度，0表示无缓存
var StreamCacheQueueLength = 0

// SensitiveWords 敏感词
// var SensitiveWords []string
var SensitiveWords = []string{
	"test_sensitive",
}

func SensitiveWordsToString() string {
	return strings.Join(SensitiveWords, "\n")
}

func SensitiveWordsFromString(s string) {
	SensitiveWords = []string{}
	sw := strings.Split(s, "\n")
	for _, w := range sw {
		w = strings.TrimSpace(w)
		if w != "" {
			SensitiveWords = append(SensitiveWords, w)
		}
	}
}

func ShouldCheckPromptSensitive() bool {
	return CheckSensitiveEnabled && CheckSensitiveOnPromptEnabled
}

func ShouldCheckPromptWithTencentTMS() bool {
	return TencentTMSModerationEnabled
}

func ShouldCheckImagesWithTencentIMS() bool {
	return TencentIMSModerationEnabled
}

func ShouldCheckOutputWithTencentTMS() bool {
	return TencentTMSOutputModerationEnabled
}

func ShouldCheckOutputImagesWithTencentIMS() bool {
	return TencentIMSOutputModerationEnabled
}

//func ShouldCheckCompletionSensitive() bool {
//	return CheckSensitiveEnabled && CheckSensitiveOnCompletionEnabled
//}
