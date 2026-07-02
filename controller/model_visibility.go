package controller

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

type modelVisibilitySetRequest struct {
	ID          int      `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	UserIDs     []int    `json:"user_ids"`
	UserTags    []string `json:"user_tags"`
	UserGroups  []string `json:"user_groups"`
}

type modelVisibilityBindingRequest struct {
	VisibilitySetIDs []int `json:"visibility_set_ids"`
}

func GetModelVisibilitySets(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	keyword := strings.TrimSpace(c.Query("keyword"))
	items, total, err := model.ListModelVisibilitySets(keyword, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

func GetModelVisibilitySet(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	detail, err := model.GetModelVisibilitySetDetail(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, detail)
}

func CreateModelVisibilitySet(c *gin.Context) {
	var req modelVisibilitySetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		common.ApiErrorMsg(c, "用户集名称不能为空")
		return
	}
	set := &model.ModelVisibilitySet{
		Name:        name,
		Description: strings.TrimSpace(req.Description),
	}
	if err := set.InsertWithMembers(req.UserIDs, req.UserTags, req.UserGroups); err != nil {
		common.ApiError(c, err)
		return
	}
	detail, err := model.GetModelVisibilitySetDetail(set.ID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	model.RefreshModelVisibilityCache()
	common.ApiSuccess(c, detail)
}

func UpdateModelVisibilitySet(c *gin.Context) {
	var req modelVisibilitySetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	if req.ID <= 0 {
		common.ApiErrorMsg(c, "缺少用户集 ID")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		common.ApiErrorMsg(c, "用户集名称不能为空")
		return
	}
	set := &model.ModelVisibilitySet{
		ID:          req.ID,
		Name:        name,
		Description: strings.TrimSpace(req.Description),
	}
	if err := model.UpdateModelVisibilitySetWithMembers(set, req.UserIDs, req.UserTags, req.UserGroups); err != nil {
		common.ApiError(c, err)
		return
	}
	detail, err := model.GetModelVisibilitySetDetail(set.ID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	model.RefreshModelVisibilityCache()
	common.ApiSuccess(c, detail)
}

func DeleteModelVisibilitySet(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if err := model.DeleteModelVisibilitySet(id); err != nil {
		common.ApiError(c, err)
		return
	}
	model.RefreshModelVisibilityCache()
	common.ApiSuccess(c, gin.H{"id": id})
}

func UpdateModelVisibilityBindings(c *gin.Context) {
	modelID, err := strconv.Atoi(c.Param("id"))
	if err != nil || modelID <= 0 {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	var req modelVisibilityBindingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	var count int64
	if err := model.DB.Model(&model.Model{}).Where("id = ?", modelID).Count(&count).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	if count == 0 {
		common.ApiErrorMsg(c, "模型不存在")
		return
	}
	if err := model.ReplaceModelVisibilityBindings(modelID, req.VisibilitySetIDs); err != nil {
		common.ApiError(c, err)
		return
	}
	ids, err := model.GetModelVisibilitySetIDs(modelID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	visibility := model.ModelVisibilityPublic
	if len(ids) > 0 {
		visibility = model.ModelVisibilitySets
	}
	model.RefreshModelVisibilityCache()
	common.ApiSuccess(c, gin.H{
		"id":                 modelID,
		"visibility":         visibility,
		"visibility_set_ids": ids,
	})
}

func SearchModelVisibilityUsers(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	keyword := strings.TrimSpace(c.Query("keyword"))
	group := strings.TrimSpace(c.Query("group"))
	tag := strings.TrimSpace(c.Query("tag"))
	users, err := model.QueryUsersForVisibilityPage(keyword, group, tag, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	total, err := model.CountUsersForVisibility(keyword, group, tag)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(users)
	common.ApiSuccess(c, pageInfo)
}

func PreviewModelVisibilityUsers(c *gin.Context) {
	var req struct {
		UserIDs    []int    `json:"user_ids"`
		UserTags   []string `json:"user_tags"`
		UserGroups []string `json:"user_groups"`
		Limit      int      `json:"limit"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	users, err := model.ListUsersByVisibilityRules(req.UserIDs, req.UserTags, req.UserGroups, req.Limit)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	total := len(users)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"items": users,
			"total": total,
		},
	})
}
