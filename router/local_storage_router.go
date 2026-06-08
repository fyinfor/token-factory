package router

import (
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
)

// registerLocalStorageRoutes 注册本地文件存储的静态文件服务路由。
// 路由始终注册，确保已上传到本地的文件始终可访问。
// URL 路径 /uploads/*filepath 映射到磁盘 storeDir/filepath
// 例如 URL /uploads/1/uuid.png -> 磁盘 uploads/1/uuid.png
func registerLocalStorageRoutes(router *gin.Engine) {
	// 注册 /uploads/*filepath 路由
	// 运行时从配置中读取实际存储目录，而非注册时快照
	router.GET("/uploads/*filepath", func(c *gin.Context) {
		cfg := operation_setting.GetOssSetting()
		storeDir := strings.TrimSpace(cfg.LocalStoragePath)
		if storeDir == "" {
			storeDir = "uploads"
		}

		fileParam := c.Param("filepath")
		// 防止路径遍历
		fileParam = filepath.Clean("/" + fileParam)
		fileParam = strings.TrimPrefix(fileParam, "/")

		// 磁盘路径：storeDir/filepath
		// 例如 storeDir="uploads", fileParam="1/uuid.png"
		// -> fullPath="uploads/1/uuid.png"
		fullPath := filepath.Join(storeDir, fileParam)

		// 验证路径在存储目录内（防止路径遍历攻击）
		absPath, err := filepath.Abs(fullPath)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "文件未找到"})
			return
		}
		absDir, err := filepath.Abs(storeDir)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "文件未找到"})
			return
		}
		if !strings.HasPrefix(absPath, absDir+string(filepath.Separator)) && absPath != absDir {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "禁止访问"})
			return
		}
		info, err := os.Stat(absPath)
		if err != nil || info.IsDir() {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "文件未找到"})
			return
		}

		// 设置 Content-Type
		contentType := mime.TypeByExtension(filepath.Ext(absPath))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		c.Header("Content-Type", contentType)

		// 设置缓存头：上传文件内容不变，可以长期缓存
		c.Header("Cache-Control", "public, max-age=31536000, immutable")

		c.File(absPath)
	})
}
