package controller

import (
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
)

// OssUpload 通用 OSS 上传（需登录；需在运营设置中启用并填写 OSS 参数）。
func OssUpload(c *gin.Context) {
	if !operation_setting.IsOssUploadReady() {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "未启用阿里云 OSS 或配置不完整",
		})
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
		c.JSON(http.StatusForbidden, gin.H{
			"success": false,
			"message": "无上传权限",
		})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请选择文件字段 file",
		})
		return
	}

	publicURL, err := service.OssUploadMultipartFile(file, id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
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
