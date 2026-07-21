package controller

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
)

type startRealNameVerificationRequest struct {
	MetaInfo json.RawMessage `json:"meta_info"`
	CertName string          `json:"cert_name"`
	CertNo   string          `json:"cert_no"`
}

func CreateRealNameVerification(c *gin.Context) {
	record, err := service.CreateRealNameVerification(c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"mobile_url": strings.TrimRight(system_setting.ServerAddress, "/") + "/real-name/start?token=" + record.LaunchToken, "expires_at": record.ExpiresAt.Unix()})
}

func GetRealNameVerificationStatus(c *gin.Context) {
	status, err := service.GetCurrentUserRealNameVerificationStatus(c.Request.Context(), c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, status)
}

func StartPublicRealNameVerification(c *gin.Context) {
	var request startRealNameVerificationRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		common.ApiErrorMsg(c, "请求参数无效")
		return
	}
	metaInfo, err := normalizeRealNameVerificationMetaInfo(request.MetaInfo)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	certName, certNo, err := normalizeRealNameVerificationCertificate(request.CertName, request.CertNo)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	certifyURL, err := service.StartAliyunRealNameVerification(c.Request.Context(), c.Query("token"), metaInfo, certName, certNo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"certify_url": certifyURL})
}

func GetPublicRealNameVerificationStatus(c *gin.Context) {
	status, err := service.GetAliyunRealNameVerificationStatus(c.Request.Context(), c.Query("token"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, status)
}

func normalizeRealNameVerificationMetaInfo(raw json.RawMessage) (string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", errors.New("请求参数无效")
	}
	var metaInfo string
	if err := common.Unmarshal(raw, &metaInfo); err == nil {
		if strings.TrimSpace(metaInfo) == "" {
			return "", errors.New("请求参数无效")
		}
		return metaInfo, nil
	}
	if common.GetJsonType(raw) != "object" {
		return "", errors.New("请求参数无效")
	}
	return string(raw), nil
}

func normalizeRealNameVerificationCertificate(certName, certNo string) (string, string, error) {
	certName = strings.TrimSpace(certName)
	certNo = strings.ToUpper(strings.TrimSpace(certNo))
	if certName == "" || len([]rune(certName)) > 64 {
		return "", "", errors.New("请输入有效的真实姓名")
	}
	if len(certNo) != 15 && len(certNo) != 18 {
		return "", "", errors.New("请输入有效的身份证号码")
	}
	for index, character := range certNo {
		isLastX := index == len(certNo)-1 && character == 'X'
		if !isLastX && (character < '0' || character > '9') {
			return "", "", errors.New("请输入有效的身份证号码")
		}
	}
	return certName, certNo, nil
}
