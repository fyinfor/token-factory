package setting

import "strings"

var AliyunRealNameVerificationEnabled = false
var AliyunRealNameVerificationAccessKeyID = ``
var AliyunRealNameVerificationAccessKeySecret = ``
var AliyunRealNameVerificationRegionID = `cn-shanghai`
var AliyunRealNameVerificationProductCode = `ID_PRO`
var AliyunRealNameVerificationSceneID = ``
var AliyunRealNameVerificationModel = `SILENT_LIVENESS`
var AliyunRealNameVerificationCallbackURL = ``
var AliyunRealNameVerificationReturnURL = ``
var AliyunRealNameVerificationRewardEnabled = false
var AliyunRealNameVerificationRewardAmount = 0.0
var AliyunRealNameVerificationRequiredForTopUp = false

func AliyunRealNameVerificationConfigured() bool {
	return AliyunRealNameVerificationEnabled &&
		strings.TrimSpace(AliyunRealNameVerificationAccessKeyID) != `` &&
		strings.TrimSpace(AliyunRealNameVerificationAccessKeySecret) != `` &&
		strings.TrimSpace(AliyunRealNameVerificationSceneID) != `` &&
		strings.TrimSpace(AliyunRealNameVerificationModel) != ``
}
