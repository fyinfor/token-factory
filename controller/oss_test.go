package controller

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupUploadFileControllerTest(t *testing.T) *gorm.DB {
	t.Helper()
	originalDB := model.DB
	originalSetting := *operation_setting.GetOssSetting()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.TemporaryUpload{}))
	model.DB = db
	operation_setting.GetOssSetting().StorageType = operation_setting.StorageTypeLocal
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() {
		model.DB = originalDB
		*operation_setting.GetOssSetting() = originalSetting
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func uploadFileTestContext(t *testing.T, method, target, id string, body any) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	var payload []byte
	if body != nil {
		var err error
		payload, err = common.Marshal(body)
		require.NoError(t, err)
	}
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "id", Value: id}}
	ctx.Request = httptest.NewRequest(method, target, bytes.NewReader(payload))
	if body != nil {
		ctx.Request.Header.Set("Content-Type", "application/json")
	}
	return ctx, recorder
}

func TestDeleteUploadFileDoesNotRequireConfirmation(t *testing.T) {
	db := setupUploadFileControllerTest(t)
	root := t.TempDir()
	objectKey := "tenant/permanent/homepage/home.png"
	filePath := filepath.Join(root, service.LocalUploadFolder, filepath.FromSlash(objectKey))
	require.NoError(t, os.MkdirAll(filepath.Dir(filePath), 0755))
	require.NoError(t, os.WriteFile(filePath, []byte("home"), 0644))
	upload := model.TemporaryUpload{
		Purpose:      service.UploadPurposeHomepage,
		OriginalName: "home.png",
		StorageType:  "local",
		ObjectKey:    objectKey,
		StorageBase:  root,
		ExpiresAt:    0,
	}
	require.NoError(t, db.Create(&upload).Error)

	ctx, recorder := uploadFileTestContext(t, http.MethodDelete, "/api/oss/files/1", fmt.Sprint(upload.ID), nil)
	DeleteUploadFile(ctx)

	var response map[string]any
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, true, response["success"])
	_, err := os.Stat(filePath)
	require.True(t, os.IsNotExist(err))
	var count int64
	require.NoError(t, db.Model(&model.TemporaryUpload{}).Count(&count).Error)
	require.Zero(t, count)
}

func TestBatchDeleteUploadFilesReturnsPartialFailures(t *testing.T) {
	db := setupUploadFileControllerTest(t)
	root := t.TempDir()
	localUploads := make([]model.TemporaryUpload, 0, 2)
	for _, name := range []string{"first.png", "second.png"} {
		objectKey := "tenant/permanent/homepage/" + name
		filePath := filepath.Join(root, service.LocalUploadFolder, filepath.FromSlash(objectKey))
		require.NoError(t, os.MkdirAll(filepath.Dir(filePath), 0755))
		require.NoError(t, os.WriteFile(filePath, []byte(name), 0644))
		upload := model.TemporaryUpload{
			Purpose:      service.UploadPurposeHomepage,
			OriginalName: name,
			StorageType:  operation_setting.StorageTypeLocal,
			ObjectKey:    objectKey,
			StorageBase:  root,
		}
		require.NoError(t, db.Create(&upload).Error)
		localUploads = append(localUploads, upload)
	}
	ossUpload := model.TemporaryUpload{
		Purpose:      service.UploadPurposeHomepage,
		OriginalName: "oss.png",
		StorageType:  operation_setting.StorageTypeOSS,
		ObjectKey:    "uploads/permanent/homepage/oss.png",
		Endpoint:     "oss-cn-test.aliyuncs.com",
		Bucket:       "bucket",
	}
	require.NoError(t, db.Create(&ossUpload).Error)

	ids := []int64{localUploads[0].ID, localUploads[1].ID, ossUpload.ID, 999999, localUploads[0].ID}
	ctx, recorder := uploadFileTestContext(t, http.MethodPost, "/api/oss/files/batch-delete", "", map[string]any{"ids": ids})
	BatchDeleteUploadFiles(ctx)

	var response map[string]any
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, true, response["success"])
	data, ok := response["data"].(map[string]any)
	require.True(t, ok)
	require.Len(t, data["deleted_ids"], 2)
	require.Len(t, data["failures"], 2)
	for _, upload := range localUploads {
		filePath := filepath.Join(root, service.LocalUploadFolder, filepath.FromSlash(upload.ObjectKey))
		_, err := os.Stat(filePath)
		require.True(t, os.IsNotExist(err))
	}
	require.NoError(t, db.First(&ossUpload, ossUpload.ID).Error)
}

func TestUpdateUploadFileExpirationAcceptsFutureTimeAndPermanent(t *testing.T) {
	db := setupUploadFileControllerTest(t)
	upload := model.TemporaryUpload{
		Purpose:      service.UploadPurposePlayground,
		OriginalName: "input.png",
		StorageType:  "local",
		ObjectKey:    "tenant/temporary/playground/input.png",
		ExpiresAt:    time.Now().Add(24 * time.Hour).Unix(),
	}
	require.NoError(t, db.Create(&upload).Error)

	future := time.Now().Add(7 * 24 * time.Hour).Unix()
	ctx, recorder := uploadFileTestContext(t, http.MethodPut, "/api/oss/files/1/expiration", fmt.Sprint(upload.ID), map[string]any{"expires_at": future})
	UpdateUploadFileExpiration(ctx)
	var response map[string]any
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, true, response["success"])
	require.NoError(t, db.First(&upload, upload.ID).Error)
	require.Equal(t, future, upload.ExpiresAt)

	ctx, recorder = uploadFileTestContext(t, http.MethodPut, "/api/oss/files/1/expiration", fmt.Sprint(upload.ID), map[string]any{"expires_at": 0})
	UpdateUploadFileExpiration(ctx)
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, true, response["success"])
	require.NoError(t, db.First(&upload, upload.ID).Error)
	require.Zero(t, upload.ExpiresAt)
}

func TestUploadFileManagerRejectsOssFiles(t *testing.T) {
	db := setupUploadFileControllerTest(t)
	upload := model.TemporaryUpload{
		Purpose:      service.UploadPurposeHomepage,
		OriginalName: "home.png",
		StorageType:  operation_setting.StorageTypeOSS,
		ObjectKey:    "uploads/permanent/homepage/home.png",
		Endpoint:     "oss-cn-test.aliyuncs.com",
		Bucket:       "bucket",
	}
	require.NoError(t, db.Create(&upload).Error)

	ctx, recorder := uploadFileTestContext(t, http.MethodDelete, "/api/oss/files/1", fmt.Sprint(upload.ID), nil)
	DeleteUploadFile(ctx)
	var response map[string]any
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, false, response["success"])
	require.NoError(t, db.First(&upload, upload.ID).Error)

	operation_setting.GetOssSetting().StorageType = operation_setting.StorageTypeOSS
	ctx, recorder = uploadFileTestContext(t, http.MethodGet, "/api/oss/files", "", nil)
	ListUploadFiles(ctx)
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.Equal(t, false, response["success"])
}
