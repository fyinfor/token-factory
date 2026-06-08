package controller

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

type changelogRequest struct {
	Date    string `json:"date"`
	Content string `json:"content"`
}

func changelogPageParams(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("p", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func validateChangelogRequest(req changelogRequest) (model.Changelog, bool) {
	changelog := model.Changelog{
		Date:    strings.TrimSpace(req.Date),
		Content: strings.TrimSpace(req.Content),
	}
	return changelog, changelog.Date != "" && changelog.Content != ""
}

func ListPublicChangelogs(c *gin.Context) {
	changelogs, err := model.ListAllChangelogs()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    changelogs,
	})
}

func AdminListChangelogs(c *gin.Context) {
	page, pageSize := changelogPageParams(c)
	changelogs, total, err := model.ListChangelogs((page-1)*pageSize, pageSize)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"items": changelogs,
			"total": total,
		},
	})
}

func AdminCreateChangelog(c *gin.Context) {
	var req changelogRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的参数"})
		return
	}
	changelog, ok := validateChangelogRequest(req)
	if !ok {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请完整填写更新日志日期和内容"})
		return
	}
	if err := model.CreateChangelog(&changelog); err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": changelog})
}

func AdminUpdateChangelog(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的参数"})
		return
	}
	var req changelogRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的参数"})
		return
	}
	changelog, ok := validateChangelogRequest(req)
	if !ok {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请完整填写更新日志日期和内容"})
		return
	}
	changelog.Id = id
	if err := model.UpdateChangelog(&changelog); err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": changelog})
}

func AdminDeleteChangelog(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的参数"})
		return
	}
	if err := model.DeleteChangelog(id); err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}
