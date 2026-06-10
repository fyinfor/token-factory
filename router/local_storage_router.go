package router

import (
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
)

// registerLocalStorageRoutes 注册本地文件存储的静态文件服务路由。
// 路由始终注册，确保已上传到本地的文件始终可访问。
// URL 路径 /api/uploads/*filepath 映射到磁盘 storeDir/uploads/filepath。
// 例如 URL /api/uploads/foo/2026/06/09/uuid.png -> 磁盘 storeDir/uploads/foo/2026/06/09/uuid.png。
func registerLocalStorageRoutes(router *gin.Engine) {
	router.GET("/api/uploads/*filepath", func(c *gin.Context) {
		serveLocalStorageFile(c, c.Param("filepath"))
	})
}

func tryServeLocalStorageByRequestPath(c *gin.Context) bool {
	requestPath := c.Request.URL.Path
	for _, route := range localStorageURLRoutes() {
		if !strings.HasPrefix(requestPath, route.URLPrefix) {
			continue
		}
		fileParam := path.Join(route.ObjectPrefix, strings.TrimPrefix(requestPath, route.URLPrefix))
		serveLocalStorageFile(c, fileParam)
		return true
	}
	return false
}

type localStorageRoute struct {
	URLPrefix    string
	ObjectPrefix string
}

func localStorageURLRoutes() []localStorageRoute {
	cfg := operation_setting.GetOssSetting()
	candidates := []localStorageRoute{
		buildLocalStorageRoute(cfg.LocalURLPrefix, cfg.LocalObjectKeyPrefix),
		buildLocalStorageRoute("/api", ""),
	}

	routes := make([]localStorageRoute, 0, len(candidates))
	seen := map[string]struct{}{}
	for _, route := range candidates {
		if route.URLPrefix == "" {
			continue
		}
		if _, ok := seen[route.URLPrefix]; ok {
			continue
		}
		seen[route.URLPrefix] = struct{}{}
		routes = append(routes, route)
	}
	return routes
}

func buildLocalStorageRoute(accessPrefix string, objectPrefix string) localStorageRoute {
	obj, err := service.NormalizeLocalUploadPrefix(objectPrefix)
	if err != nil {
		return localStorageRoute{}
	}
	base := normalizeLocalStorageAccessPrefix(accessPrefix)
	urlPrefix := "/" + strings.Trim(path.Join(strings.Trim(base, "/"), service.LocalUploadFolder, obj), "/") + "/"
	return localStorageRoute{URLPrefix: urlPrefix, ObjectPrefix: obj}
}

func normalizeLocalStorageAccessPrefix(raw string) string {
	prefix := strings.TrimSpace(raw)
	if prefix == "" {
		return "/api"
	}
	if u, err := url.Parse(prefix); err == nil && u.Path != "" && u.Scheme != "" {
		prefix = u.Path
	}
	prefix = strings.Trim(prefix, "/")
	if prefix == "" {
		return ""
	}
	return "/" + prefix
}

func serveLocalStorageFile(c *gin.Context, fileParam string) {
	cfg := operation_setting.GetOssSetting()
	storeDir := service.LocalUploadBaseDir(cfg.LocalStoragePath)

	fileParam = filepath.Clean("/" + fileParam)
	fileParam = strings.TrimPrefix(fileParam, "/")
	if fileParam == "." || fileParam == "" {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "文件未找到"})
		return
	}

	fullPath := filepath.Join(storeDir, filepath.FromSlash(fileParam))
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

	contentType := mime.TypeByExtension(filepath.Ext(absPath))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	c.Header("Content-Type", contentType)
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.File(absPath)
}
