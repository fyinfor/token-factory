package service

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestResolveUploadPurpose(t *testing.T) {
	tests := []struct {
		input     string
		purpose   string
		directory string
		temporary bool
	}{
		{input: "", purpose: UploadPurposeGeneral, directory: "permanent/general"},
		{input: "homepage", purpose: UploadPurposeHomepage, directory: "permanent/homepage"},
		{input: "icon", purpose: UploadPurposeIcons, directory: "permanent/icons"},
		{input: "suppliers", purpose: UploadPurposeSupplier, directory: "permanent/suppliers"},
		{input: "distributors", purpose: UploadPurposeDistributor, directory: "permanent/distributors"},
		{input: "channels", purpose: UploadPurposeChannel, directory: "permanent/channels"},
		{input: " PLAYGROUND ", purpose: UploadPurposePlayground, directory: "temporary/playground", temporary: true},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			purpose, spec, err := resolveUploadPurpose(test.input)
			require.NoError(t, err)
			require.Equal(t, test.purpose, purpose)
			require.Equal(t, test.directory, spec.directory)
			require.Equal(t, test.temporary, spec.temporary)
		})
	}

	_, _, err := resolveUploadPurpose("../playground")
	require.Error(t, err)
}

func TestPlaygroundUploadRetention(t *testing.T) {
	tests := []struct {
		hours    int
		expected time.Duration
	}{
		{hours: 0, expected: PlaygroundUploadRetention},
		{hours: 12, expected: 12 * time.Hour},
		{hours: 72, expected: 72 * time.Hour},
		{hours: 120, expected: 120 * time.Hour},
		{hours: 168, expected: 168 * time.Hour},
		{hours: maxPlaygroundUploadRetentionHours + 1, expected: PlaygroundUploadRetention},
	}
	for _, test := range tests {
		require.Equal(t, test.expected, playgroundUploadRetention(test.hours))
	}
}

func TestDeleteLocalStoredObject(t *testing.T) {
	root := t.TempDir()
	temporaryFile := filepath.Join(root, LocalUploadFolder, "tenant", "temporary", "playground", "2026", "07", "27", "file.png")
	permanentFile := filepath.Join(root, LocalUploadFolder, "tenant", "permanent", "homepage", "keep.png")
	require.NoError(t, os.MkdirAll(filepath.Dir(temporaryFile), 0755))
	require.NoError(t, os.MkdirAll(filepath.Dir(permanentFile), 0755))
	require.NoError(t, os.WriteFile(temporaryFile, []byte("temporary"), 0644))
	require.NoError(t, os.WriteFile(permanentFile, []byte("permanent"), 0644))

	objectKey := "tenant/temporary/playground/2026/07/27/file.png"
	require.NoError(t, deleteLocalStoredObject(root, objectKey))
	_, err := os.Stat(temporaryFile)
	require.True(t, os.IsNotExist(err))
	_, err = os.Stat(permanentFile)
	require.NoError(t, err)
	require.NoError(t, deleteLocalStoredObject(root, objectKey))

	outsideFile := filepath.Join(root, "outside.txt")
	require.NoError(t, os.WriteFile(outsideFile, []byte("keep"), 0644))
	require.Error(t, deleteLocalStoredObject(root, "../outside.txt"))
	_, err = os.Stat(outsideFile)
	require.NoError(t, err)
}

func TestUploadMultipartFileByPurposeLocalLifecycle(t *testing.T) {
	originalDB := model.DB
	originalSetting := *operation_setting.GetOssSetting()
	t.Cleanup(func() {
		model.DB = originalDB
		*operation_setting.GetOssSetting() = originalSetting
	})

	db, err := gorm.Open(sqlite.Open("file:upload_lifecycle?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.TemporaryUpload{}))
	model.DB = db

	root := t.TempDir()
	cfg := operation_setting.GetOssSetting()
	*cfg = operation_setting.OssSetting{
		Enabled:                  true,
		StorageType:              operation_setting.StorageTypeLocal,
		LocalStoragePath:         root,
		LocalObjectKeyPrefix:     "tenant",
		LocalMaxFileSizeMB:       20,
		MaxFileSizeMB:            20,
		ObjectKeyPrefix:          "uploads/",
		OssMaxFileSizeMB:         20,
		PlaygroundRetentionHours: 120,
	}

	permanent, err := UploadMultipartFileByPurpose(testMultipartFile(t, "home.png", "image/png", []byte("home")), 7, UploadPurposeHomepage)
	require.NoError(t, err)
	require.Contains(t, permanent.ObjectKey, "tenant/permanent/homepage/")
	require.Zero(t, permanent.ExpiresAt)

	var count int64
	require.NoError(t, db.Model(&model.TemporaryUpload{}).Count(&count).Error)
	require.Equal(t, int64(1), count)
	var permanentRecord model.TemporaryUpload
	require.NoError(t, db.Where("object_key = ?", permanent.ObjectKey).First(&permanentRecord).Error)
	require.Equal(t, UploadPurposeHomepage, permanentRecord.Purpose)
	require.Equal(t, "home.png", permanentRecord.OriginalName)
	require.Zero(t, permanentRecord.ExpiresAt)
	require.NotNil(t, permanentRecord.StorageKeyHash)

	before := time.Now().Unix()
	temporary, err := UploadMultipartFileByPurpose(testMultipartFile(t, "input.png", "image/png", []byte("playground")), 7, UploadPurposePlayground)
	require.NoError(t, err)
	require.Contains(t, temporary.ObjectKey, "tenant/temporary/playground/")
	require.GreaterOrEqual(t, temporary.ExpiresAt, before+int64(120*time.Hour/time.Second)-1)

	var record model.TemporaryUpload
	require.NoError(t, db.Where("object_key = ?", temporary.ObjectKey).First(&record).Error)
	require.Equal(t, temporary.ObjectKey, record.ObjectKey)
	require.Equal(t, UploadPurposePlayground, record.Purpose)
	require.Equal(t, temporary.ExpiresAt, record.ExpiresAt)
	temporaryPath := filepath.Join(root, LocalUploadFolder, filepath.FromSlash(record.ObjectKey))
	_, err = os.Stat(temporaryPath)
	require.NoError(t, err)

	require.NoError(t, db.Model(&record).Update("expires_at", time.Now().Add(-time.Minute).Unix()).Error)
	runTemporaryUploadCleanupOnce()
	_, err = os.Stat(temporaryPath)
	require.True(t, os.IsNotExist(err))
	require.NoError(t, db.Model(&model.TemporaryUpload{}).Count(&count).Error)
	require.Equal(t, int64(1), count)

	permanentPath := filepath.Join(root, LocalUploadFolder, filepath.FromSlash(permanent.ObjectKey))
	_, err = os.Stat(permanentPath)
	require.NoError(t, err)
}

func testMultipartFile(t *testing.T, filename, contentType string, data []byte) *multipart.FileHeader {
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

	request := httptest.NewRequest("POST", "/upload", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	require.NoError(t, request.ParseMultipartForm(1<<20))
	file, _, err := request.FormFile("file")
	require.NoError(t, err)
	require.NoError(t, file.Close())
	return request.MultipartForm.File["file"][0]
}

func TestUploadPurposeDirectoriesUseSafeSegments(t *testing.T) {
	for _, spec := range uploadPurposeSpecs {
		require.NotContains(t, spec.directory, "\\")
		require.NotContains(t, spec.directory, "..")
		for _, segment := range strings.Split(spec.directory, "/") {
			require.Regexp(t, `^[a-z]+$`, segment)
		}
	}
}

func TestOssDeleteObjectOnceBuildsSignedDeleteRequest(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodDelete, request.Method)
		require.Equal(t, "bucket.oss-cn-test.aliyuncs.com", request.URL.Host)
		require.Equal(t, "/uploads/temporary/playground/file.png", request.URL.Path)
		require.NotEmpty(t, request.Header.Get("Date"))
		require.True(t, strings.HasPrefix(request.Header.Get("Authorization"), "OSS access-key-id:"))
		return &http.Response{
			StatusCode: http.StatusNoContent,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     make(http.Header),
		}, nil
	})}

	status, err := ossDeleteObjectOnceWithClient(
		client,
		"https://oss-cn-test.aliyuncs.com/",
		"bucket",
		"uploads/temporary/playground/file.png",
		"access-key-id",
		"access-key-secret",
	)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, status)
}

func TestSyncLocalUploadObjectsImportsLegacyAndPurposeFiles(t *testing.T) {
	originalDB := model.DB
	t.Cleanup(func() { model.DB = originalDB })
	db, err := gorm.Open(sqlite.Open("file:upload_sync?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.TemporaryUpload{}))
	model.DB = db

	root := t.TempDir()
	files := map[string]string{
		"tenant/permanent/homepage/home.png":    "homepage",
		"tenant/temporary/playground/input.mp4": "playground",
		"tenant/2025/12/31/old-document.pdf":    "legacy",
	}
	modifiedAt := time.Now().Add(-2 * time.Hour).Truncate(time.Second)
	for objectKey := range files {
		filePath := filepath.Join(root, LocalUploadFolder, filepath.FromSlash(objectKey))
		require.NoError(t, os.MkdirAll(filepath.Dir(filePath), 0755))
		require.NoError(t, os.WriteFile(filePath, []byte(objectKey), 0644))
		require.NoError(t, os.Chtimes(filePath, modifiedAt, modifiedAt))
	}

	result, err := syncLocalUploadObjects(context.Background(), &operation_setting.OssSetting{
		StorageType:              operation_setting.StorageTypeLocal,
		LocalStoragePath:         root,
		LocalObjectKeyPrefix:     "tenant",
		PlaygroundRetentionHours: 72,
	})
	require.NoError(t, err)
	require.Equal(t, 3, result.Scanned)
	require.Equal(t, 3, result.Synced)

	var records []model.TemporaryUpload
	require.NoError(t, db.Order("object_key ASC").Find(&records).Error)
	require.Len(t, records, 3)
	for _, record := range records {
		require.Equal(t, files[record.ObjectKey], record.Purpose)
		require.NotEmpty(t, record.URL)
		require.NotNil(t, record.StorageKeyHash)
		if record.Purpose == UploadPurposePlayground {
			require.Equal(t, modifiedAt.Add(72*time.Hour).Unix(), record.ExpiresAt)
		} else {
			require.Zero(t, record.ExpiresAt)
		}
	}
	var playgroundRecord model.TemporaryUpload
	require.NoError(t, db.Where("purpose = ?", UploadPurposePlayground).First(&playgroundRecord).Error)
	customExpiration := time.Now().Add(7 * 24 * time.Hour).Unix()
	require.NoError(t, model.UpdateUploadObjectExpiration(playgroundRecord.ID, customExpiration))

	_, err = syncLocalUploadObjects(context.Background(), &operation_setting.OssSetting{
		StorageType:              operation_setting.StorageTypeLocal,
		LocalStoragePath:         root,
		LocalObjectKeyPrefix:     "tenant",
		PlaygroundRetentionHours: 72,
	})
	require.NoError(t, err)
	var count int64
	require.NoError(t, db.Model(&model.TemporaryUpload{}).Count(&count).Error)
	require.Equal(t, int64(3), count)
	require.NoError(t, db.First(&playgroundRecord, playgroundRecord.ID).Error)
	require.Equal(t, customExpiration, playgroundRecord.ExpiresAt)
}

func TestOssListObjectsPageBuildsSignedListRequest(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodGet, request.Method)
		require.Equal(t, "bucket.oss-cn-test.aliyuncs.com", request.URL.Host)
		require.Equal(t, "/", request.URL.Path)
		require.Equal(t, "uploads/", request.URL.Query().Get("prefix"))
		require.Equal(t, "uploads/old.png", request.URL.Query().Get("marker"))
		require.Equal(t, "1000", request.URL.Query().Get("max-keys"))
		require.True(t, strings.HasPrefix(request.Header.Get("Authorization"), "OSS access-key-id:"))
		body := `<ListBucketResult><IsTruncated>false</IsTruncated><Contents><Key>uploads/permanent/icons/logo.png</Key><LastModified>2026-07-27T08:00:00.000Z</LastModified><Size>42</Size></Contents></ListBucketResult>`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		}, nil
	})}
	cfg := &operation_setting.OssSetting{
		Endpoint:        "https://oss-cn-test.aliyuncs.com/",
		Bucket:          "bucket",
		AccessKeyID:     "access-key-id",
		AccessKeySecret: "access-key-secret",
	}
	result, err := ossListObjectsPageWithClient(context.Background(), client, cfg, "uploads/", "uploads/old.png")
	require.NoError(t, err)
	require.Len(t, result.Contents, 1)
	require.Equal(t, "uploads/permanent/icons/logo.png", result.Contents[0].Key)
	require.Equal(t, int64(42), result.Contents[0].Size)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}
