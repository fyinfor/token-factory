package aliyunasr

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"path/filepath"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestUploadPlaygroundAudioFile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalDB := model.DB
	originalSetting := *operation_setting.GetOssSetting()
	originalServer := system_setting.ServerAddress
	t.Cleanup(func() {
		model.DB = originalDB
		*operation_setting.GetOssSetting() = originalSetting
		system_setting.ServerAddress = originalServer
	})

	db, err := gorm.Open(sqlite.Open("file:aliyunasr_upload?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.TemporaryUpload{}))
	model.DB = db

	root := t.TempDir()
	system_setting.ServerAddress = "https://example.test"
	cfg := operation_setting.GetOssSetting()
	*cfg = operation_setting.OssSetting{
		Enabled:                  true,
		StorageType:              operation_setting.StorageTypeLocal,
		LocalStoragePath:         root,
		LocalObjectKeyPrefix:     "tenant",
		LocalURLPrefix:           "/api",
		LocalMaxFileSizeMB:       20,
		MaxFileSizeMB:            20,
		PlaygroundRetentionHours: 24,
	}

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/audio/transcriptions", nil)
	common.SetContextKey(c, constant.ContextKeyUserId, 42)

	fileHeader := testAudioMultipartFile(t, "sample.mp3", "audio/mpeg", []byte("fake-mp3-bytes"))
	url, err := UploadPlaygroundAudioFile(c, fileHeader)
	require.NoError(t, err)
	require.Contains(t, url, "https://example.test/api/uploads/")
	require.Contains(t, url, "temporary/playground/")

	var record model.TemporaryUpload
	require.NoError(t, db.Where("url = ?", url).First(&record).Error)
	require.Equal(t, service.UploadPurposePlayground, record.Purpose)
	require.Equal(t, 42, record.UserID)
	require.Equal(t, "sample.mp3", record.OriginalName)

	localPath := filepath.Join(root, service.LocalUploadFolder, filepath.FromSlash(record.ObjectKey))
	_, err = os.Stat(localPath)
	require.NoError(t, err)
}

func TestUploadPlaygroundAudioFile_RequiresUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	_, err := UploadPlaygroundAudioFile(c, testAudioMultipartFile(t, "a.mp3", "audio/mpeg", []byte("x")))
	require.Error(t, err)
	require.Contains(t, err.Error(), "无法识别用户")
}

func testAudioMultipartFile(t *testing.T, filename, contentType string, data []byte) *multipart.FileHeader {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	header.Set("Content-Type", contentType)
	part, err := writer.CreatePart(header)
	require.NoError(t, err)
	_, err = part.Write(data)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, "/", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	require.NoError(t, req.ParseMultipartForm(1<<20))
	files := req.MultipartForm.File["file"]
	require.Len(t, files, 1)
	return files[0]
}
