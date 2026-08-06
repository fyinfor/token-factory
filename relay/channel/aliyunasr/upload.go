package aliyunasr

import (
	"fmt"
	"mime/multipart"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

// UploadPlaygroundAudioFile 将音频文件上传到与操练场「视频→媒体」相同的附件库
//（service.UploadMultipartFileByPurpose, purpose=playground），返回可访问 URL，供上游 ASR 拉取。
func UploadPlaygroundAudioFile(c *gin.Context, fileHeader *multipart.FileHeader) (string, error) {
	if c == nil {
		return "", fmt.Errorf("缺少请求上下文，无法上传音频")
	}
	if fileHeader == nil {
		return "", fmt.Errorf("未提供音频文件")
	}
	userID := common.GetContextKeyInt(c, constant.ContextKeyUserId)
	if userID <= 0 {
		userID = c.GetInt("id")
	}
	if userID <= 0 {
		return "", fmt.Errorf("无法识别用户，拒绝上传音频附件")
	}

	result, err := service.UploadMultipartFileByPurpose(fileHeader, userID, service.UploadPurposePlayground)
	if err != nil {
		return "", fmt.Errorf("上传音频到附件库失败: %w", err)
	}
	url := strings.TrimSpace(result.URL)
	if url == "" {
		return "", fmt.Errorf("上传音频到附件库成功但未返回可访问地址")
	}
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return "", fmt.Errorf("附件库返回的音频地址不是 http/https（当前: %s）。请配置系统 ServerAddress，或改用 OSS 存储", url)
	}
	return url, nil
}
