package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
)

func ossUploadFail(c *gin.Context, message string) {
	c.JSON(http.StatusOK, gin.H{
		"success": false,
		"message": message,
	})
}

// OssUpload 通用文件上传（需登录；需在运营设置中启用上传）。
// 根据 storage_type 分发到本地存储或阿里云 OSS。
func OssUpload(c *gin.Context) {
	cfg := operation_setting.GetOssSetting()
	if !cfg.Enabled {
		ossUploadFail(c, "文件上传未启用，请先在运营设置中启用")
		return
	}

	id := c.GetInt("id")
	if id == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "未授权",
		})
		return
	}
	user, err := model.GetUserById(id, false)
	if err != nil || user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "用户无效",
		})
		return
	}
	if user.Role < common.FileUploadPermission {
		ossUploadFail(c, "无上传权限")
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		ossUploadFail(c, "请选择文件字段 file")
		return
	}

	var publicURL string
	if cfg.StorageType == operation_setting.StorageTypeLocal {
		publicURL, err = service.LocalUploadMultipartFile(file, id)
	} else {
		if !operation_setting.IsOssUploadReady() {
			ossUploadFail(c, service.ErrOssNotConfigured.Error())
			return
		}
		publicURL, err = service.OssUploadMultipartFile(file, id)
	}

	if err != nil {
		ossUploadFail(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"url": publicURL,
		},
	})
}
