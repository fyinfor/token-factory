package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func ossUploadFail(c *gin.Context, message string) {
	c.JSON(http.StatusOK, gin.H{
		"success": false,
		"message": message,
	})
}

func ensureLocalUploadFileManager(c *gin.Context) bool {
	if operation_setting.GetOssSetting().StorageType != operation_setting.StorageTypeLocal {
		common.ApiErrorMsg(c, "轻量文件管理仅支持本地存储")
		return false
	}
	return true
}

// ListUploadFiles returns the persisted file index for the lightweight manager.
func ListUploadFiles(c *gin.Context) {
	if !ensureLocalUploadFileManager(c) {
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("p", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	files, total, err := model.ListUploadObjects(model.UploadObjectFilter{
		Keyword:     c.Query("keyword"),
		Purpose:     strings.TrimSpace(c.Query("purpose")),
		Lifecycle:   strings.TrimSpace(c.Query("lifecycle")),
		StorageType: operation_setting.StorageTypeLocal,
		Page:        page,
		PageSize:    pageSize,
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"items": files,
			"total": total,
		},
	})
}

// SyncUploadFiles discovers objects created before the file index existed.
func SyncUploadFiles(c *gin.Context) {
	if !ensureLocalUploadFileManager(c) {
		return
	}
	result, err := service.SyncExistingUploadObjects(c.Request.Context())
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	common.ApiSuccess(c, result)
}

// DeleteUploadFile deletes both the stored object and its index record.
func DeleteUploadFile(c *gin.Context) {
	if !ensureLocalUploadFileManager(c) {
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "无效的文件 ID")
		return
	}
	upload, err := model.GetUploadObjectByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		common.ApiErrorMsg(c, "文件记录不存在")
		return
	}
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if upload.StorageType != operation_setting.StorageTypeLocal {
		common.ApiErrorMsg(c, "轻量文件管理不能操作 OSS 文件")
		return
	}
	if err := service.DeleteIndexedUploadObject(upload); err != nil {
		common.ApiErrorMsg(c, "删除文件失败: "+err.Error())
		return
	}
	common.ApiSuccess(c, gin.H{"id": upload.ID})
}

type batchDeleteUploadFilesRequest struct {
	IDs []int64 `json:"ids"`
}

type batchDeleteUploadFileFailure struct {
	ID      int64  `json:"id"`
	Message string `json:"message"`
}

// BatchDeleteUploadFiles removes up to 100 local files and reports per-file failures.
func BatchDeleteUploadFiles(c *gin.Context) {
	if !ensureLocalUploadFileManager(c) {
		return
	}
	var request batchDeleteUploadFilesRequest
	if err := c.ShouldBindJSON(&request); err != nil || len(request.IDs) == 0 {
		common.ApiErrorMsg(c, "请选择要删除的文件")
		return
	}
	if len(request.IDs) > 100 {
		common.ApiErrorMsg(c, "单次最多删除 100 个文件")
		return
	}

	ids := make([]int64, 0, len(request.IDs))
	seen := make(map[int64]struct{}, len(request.IDs))
	for _, id := range request.IDs {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		common.ApiErrorMsg(c, "请选择有效的文件")
		return
	}

	uploads, err := model.GetUploadObjectsByIDs(ids)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	indexed := make(map[int64]*model.TemporaryUpload, len(uploads))
	for i := range uploads {
		indexed[uploads[i].ID] = &uploads[i]
	}

	deletedIDs := make([]int64, 0, len(ids))
	failures := make([]batchDeleteUploadFileFailure, 0)
	for _, id := range ids {
		upload, ok := indexed[id]
		if !ok {
			failures = append(failures, batchDeleteUploadFileFailure{ID: id, Message: "文件记录不存在"})
			continue
		}
		if upload.StorageType != operation_setting.StorageTypeLocal {
			failures = append(failures, batchDeleteUploadFileFailure{ID: id, Message: "不能操作 OSS 文件"})
			continue
		}
		if err := service.DeleteIndexedUploadObject(upload); err != nil {
			failures = append(failures, batchDeleteUploadFileFailure{ID: id, Message: err.Error()})
			continue
		}
		deletedIDs = append(deletedIDs, id)
	}
	common.ApiSuccess(c, gin.H{
		"deleted_ids": deletedIDs,
		"failures":    failures,
	})
}

type updateUploadFileExpirationRequest struct {
	ExpiresAt int64 `json:"expires_at"`
}

// UpdateUploadFileExpiration changes when an indexed local object is cleaned up.
func UpdateUploadFileExpiration(c *gin.Context) {
	if !ensureLocalUploadFileManager(c) {
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "无效的文件 ID")
		return
	}
	var request updateUploadFileExpirationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiErrorMsg(c, "无效的过期时间")
		return
	}
	if request.ExpiresAt < 0 || (request.ExpiresAt > 0 && request.ExpiresAt <= time.Now().Unix()) {
		common.ApiErrorMsg(c, "过期时间必须晚于当前时间")
		return
	}
	upload, err := model.GetUploadObjectByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		common.ApiErrorMsg(c, "文件记录不存在")
		return
	} else if err != nil {
		common.ApiError(c, err)
		return
	}
	if upload.StorageType != operation_setting.StorageTypeLocal {
		common.ApiErrorMsg(c, "轻量文件管理不能操作 OSS 文件")
		return
	}
	if err := model.UpdateUploadObjectExpiration(id, request.ExpiresAt); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"id": id, "expires_at": request.ExpiresAt})
}

// OssUpload 通用文件上传（需登录；需在运营设置中启用上传）。
// 根据 storage_type 分发到本地存储或阿里云 OSS。
func OssUpload(c *gin.Context) {
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

	upload, err := service.UploadMultipartFileByPurpose(file, id, c.PostForm("purpose"))
	if err != nil {
		ossUploadFail(c, err.Error())
		return
	}
	data := gin.H{"url": upload.URL}
	if upload.ExpiresAt > 0 {
		data["expires_at"] = upload.ExpiresAt
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    data,
	})
}
