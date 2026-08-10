package controller

import (
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

const maxComputePageHTMLBytes int64 = 20 * 1024 * 1024

type computePageEnabledRequest struct {
	Enabled bool `json:"enabled"`
}

type computePageJavaScriptRequest struct {
	AllowJavaScript bool `json:"allow_javascript"`
}

func computePageResponse(config service.ComputePageConfig) gin.H {
	return gin.H{
		"enabled":          config.Enabled && config.HasHTML,
		"allow_javascript": config.AllowJavaScript,
		"has_html":         config.HasHTML,
		"file_name":        config.FileName,
		"updated_at":       config.UpdatedAt,
	}
}

func GetComputePageStatus(c *gin.Context) {
	config, err := service.GetComputePageConfig()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"enabled":          config.Enabled && config.HasHTML,
			"allow_javascript": config.AllowJavaScript,
		},
	})
}

func GetComputePageContent(c *gin.Context) {
	content, config, err := service.ReadEnabledComputePageHTML()
	if err != nil {
		if errors.Is(err, service.ErrComputePageDisabled) {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": err.Error()})
			return
		}
		common.ApiError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Header("Content-Security-Policy", computePageContentSecurityPolicy(config.AllowJavaScript))
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, "text/html; charset=utf-8", content)
}

func computePageContentSecurityPolicy(allowJavaScript bool) string {
	sandbox := "sandbox allow-same-origin"
	scriptPolicy := ""
	if allowJavaScript {
		sandbox += " allow-scripts"
		scriptPolicy = "; script-src 'unsafe-inline' 'unsafe-eval' data: blob: https: http:; connect-src https: http: ws: wss:"
	}
	return sandbox + "; default-src 'none'" + scriptPolicy + "; style-src 'unsafe-inline' https:; img-src data: blob: https: http:; font-src data: https:; media-src data: blob: https: http:; frame-src https: http:; base-uri 'none'; form-action 'none'"
}

func AdminGetComputePageConfig(c *gin.Context) {
	config, err := service.GetComputePageConfig()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": computePageResponse(config)})
}

func AdminUpdateComputePageEnabled(c *gin.Context) {
	var request computePageEnabledRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的参数"})
		return
	}
	config, err := service.UpdateComputePageEnabled(request.Enabled)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": computePageResponse(config)})
}

func AdminUpdateComputePageJavaScript(c *gin.Context) {
	var request computePageJavaScriptRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的参数"})
		return
	}
	config, err := service.UpdateComputePageJavaScriptAllowed(request.AllowJavaScript)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": computePageResponse(config)})
}

func AdminUploadComputePageHTML(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxComputePageHTMLBytes+1024*1024)
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请选择 HTML 文件"})
		return
	}
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext != ".html" && ext != ".htm" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "仅支持 HTML 文件"})
		return
	}
	if fileHeader.Size > maxComputePageHTMLBytes {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "HTML 文件不能超过 20 MB"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, maxComputePageHTMLBytes+1))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if len(content) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "HTML 文件内容不能为空"})
		return
	}
	if int64(len(content)) > maxComputePageHTMLBytes {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "HTML 文件不能超过 20 MB"})
		return
	}
	if !utf8.Valid(content) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "HTML 文件必须使用 UTF-8 编码"})
		return
	}

	config, err := service.SaveComputePageHTML(fileHeader.Filename, content)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": computePageResponse(config)})
}
