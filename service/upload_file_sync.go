package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

const ossListObjectsPageSize = 1000

type UploadSyncResult struct {
	StorageType string `json:"storage_type"`
	Scanned     int    `json:"scanned"`
	Synced      int    `json:"synced"`
}

var uploadFileSyncRunning atomic.Bool

// SyncExistingUploadObjects imports files that predate the upload index. It only
// scans the currently configured local prefix or OSS bucket/prefix.
func SyncExistingUploadObjects(ctx context.Context) (*UploadSyncResult, error) {
	if !uploadFileSyncRunning.CompareAndSwap(false, true) {
		return nil, fmt.Errorf("文件同步正在进行中，请稍后再试")
	}
	defer uploadFileSyncRunning.Store(false)

	cfg := *operation_setting.GetOssSetting()
	switch cfg.StorageType {
	case operation_setting.StorageTypeLocal:
		return syncLocalUploadObjects(ctx, &cfg)
	case operation_setting.StorageTypeOSS:
		return syncOssUploadObjects(ctx, &cfg)
	default:
		return nil, fmt.Errorf("不支持的文件存储类型: %s", cfg.StorageType)
	}
}

func syncLocalUploadObjects(ctx context.Context, cfg *operation_setting.OssSetting) (*UploadSyncResult, error) {
	prefix, err := NormalizeLocalUploadPrefix(cfg.LocalObjectKeyPrefix)
	if err != nil {
		return nil, err
	}
	baseDir, err := filepath.Abs(LocalUploadBaseDir(cfg.LocalStoragePath))
	if err != nil {
		return nil, err
	}
	scanDir := baseDir
	if prefix != "" {
		scanDir = filepath.Join(baseDir, filepath.FromSlash(prefix))
	}
	result := &UploadSyncResult{StorageType: operation_setting.StorageTypeLocal}
	retention := playgroundUploadRetention(cfg.PlaygroundRetentionHours)
	if _, err := os.Stat(scanDir); os.IsNotExist(err) {
		return result, nil
	} else if err != nil {
		return nil, fmt.Errorf("读取本地上传目录失败: %w", err)
	}

	err = filepath.WalkDir(scanDir, func(filePath string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		relativePath, err := filepath.Rel(baseDir, filePath)
		if err != nil {
			return err
		}
		objectKey := filepath.ToSlash(relativePath)
		indexed := indexedUploadObject(
			operation_setting.StorageTypeLocal,
			objectKey,
			prefix,
			cfg.LocalStoragePath,
			"",
			"",
			localObjectURL(cfg.LocalURLPrefix, path.Join(LocalUploadFolder, objectKey)),
			info.Size(),
			info.ModTime(),
			retention,
		)
		result.Scanned++
		if err := model.UpsertUploadObject(indexed); err != nil {
			return fmt.Errorf("同步本地文件 %s 失败: %w", objectKey, err)
		}
		result.Synced++
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func syncOssUploadObjects(ctx context.Context, cfg *operation_setting.OssSetting) (*UploadSyncResult, error) {
	if strings.TrimSpace(cfg.Endpoint) == "" || strings.TrimSpace(cfg.Bucket) == "" ||
		strings.TrimSpace(cfg.AccessKeyID) == "" || strings.TrimSpace(cfg.AccessKeySecret) == "" {
		return nil, ErrOssNotConfigured
	}
	prefix := strings.TrimLeft(strings.TrimSpace(cfg.ObjectKeyPrefix), "/")
	result := &UploadSyncResult{StorageType: operation_setting.StorageTypeOSS}
	retention := playgroundUploadRetention(cfg.PlaygroundRetentionHours)
	marker := ""
	for {
		pageResult, err := ossListObjectsPage(ctx, cfg, prefix, marker)
		if err != nil {
			return nil, err
		}
		for _, object := range pageResult.Contents {
			if strings.TrimSpace(object.Key) == "" || strings.HasSuffix(object.Key, "/") {
				continue
			}
			modifiedAt, _ := time.Parse(time.RFC3339, object.LastModified)
			indexed := indexedUploadObject(
				operation_setting.StorageTypeOSS,
				object.Key,
				prefix,
				"",
				cfg.Endpoint,
				cfg.Bucket,
				publicObjectURL(cfg, object.Key),
				object.Size,
				modifiedAt,
				retention,
			)
			result.Scanned++
			if err := model.UpsertUploadObject(indexed); err != nil {
				return nil, fmt.Errorf("同步 OSS 文件 %s 失败: %w", object.Key, err)
			}
			result.Synced++
		}
		if !pageResult.IsTruncated {
			break
		}
		nextMarker := strings.TrimSpace(pageResult.NextMarker)
		if nextMarker == "" && len(pageResult.Contents) > 0 {
			nextMarker = pageResult.Contents[len(pageResult.Contents)-1].Key
		}
		if nextMarker == "" || nextMarker == marker {
			return nil, fmt.Errorf("OSS 文件列表分页游标无效")
		}
		marker = nextMarker
	}
	return result, nil
}

func indexedUploadObject(storageType, objectKey, prefix, storageBase, endpoint, bucket, publicURL string, size int64, modifiedAt time.Time, retention time.Duration) *model.TemporaryUpload {
	purpose, temporary := inferUploadPurpose(objectKey, prefix)
	createdAt := modifiedAt.Unix()
	if createdAt <= 0 {
		createdAt = time.Now().Unix()
	}
	expiresAt := int64(0)
	if temporary {
		expiresAt = createdAt + int64(retention/time.Second)
	}
	hash := uploadStorageKeyHash(storageType, objectKey, storageBase, endpoint, bucket)
	return &model.TemporaryUpload{
		Purpose:        purpose,
		OriginalName:   path.Base(objectKey),
		MimeType:       uploadMimeTypeFromName(objectKey),
		Size:           size,
		URL:            publicURL,
		StorageType:    storageType,
		ObjectKey:      objectKey,
		StorageBase:    storageBase,
		Endpoint:       endpoint,
		Bucket:         bucket,
		StorageKeyHash: &hash,
		ExpiresAt:      expiresAt,
		CreatedAt:      createdAt,
		UpdatedAt:      createdAt,
	}
}

func inferUploadPurpose(objectKey, configuredPrefix string) (string, bool) {
	key := strings.TrimLeft(path.Clean(strings.TrimSpace(objectKey)), "/")
	prefix := strings.Trim(path.Clean(strings.TrimSpace(configuredPrefix)), "/")
	relative := key
	if prefix != "" && prefix != "." {
		if key == prefix {
			return UploadPurposeLegacy, false
		}
		if strings.HasPrefix(key, prefix+"/") {
			relative = strings.TrimPrefix(key, prefix+"/")
		}
	}
	for purpose, spec := range uploadPurposeSpecs {
		if relative == spec.directory || strings.HasPrefix(relative, spec.directory+"/") {
			return purpose, spec.temporary
		}
	}
	return UploadPurposeLegacy, false
}

func uploadMimeTypeFromName(filename string) string {
	mimeType := mime.TypeByExtension(path.Ext(filename))
	if mimeType == "" {
		return "application/octet-stream"
	}
	if len(mimeType) > 128 {
		return mimeType[:128]
	}
	return mimeType
}

func uploadStorageKeyHash(storageType, objectKey, storageBase, endpoint, bucket string) string {
	storageType = strings.ToLower(strings.TrimSpace(storageType))
	objectKey = strings.TrimLeft(path.Clean(strings.ReplaceAll(strings.TrimSpace(objectKey), "\\", "/")), "/")
	if storageType == operation_setting.StorageTypeLocal {
		if absoluteBase, err := filepath.Abs(LocalUploadBaseDir(storageBase)); err == nil {
			storageBase = filepath.Clean(absoluteBase)
		} else {
			storageBase = filepath.Clean(storageBase)
		}
		if runtime.GOOS == "windows" {
			storageBase = strings.ToLower(storageBase)
		}
	} else {
		storageBase = ""
	}
	endpoint = strings.ToLower(strings.TrimRight(strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(endpoint), "https://"), "http://"), "/"))
	bucket = strings.ToLower(strings.TrimSpace(bucket))
	sum := sha256.Sum256([]byte(strings.Join([]string{storageType, storageBase, endpoint, bucket, objectKey}, "\x00")))
	return hex.EncodeToString(sum[:])
}

type ossListBucketResult struct {
	IsTruncated bool              `xml:"IsTruncated"`
	NextMarker  string            `xml:"NextMarker"`
	Contents    []ossListedObject `xml:"Contents"`
}

type ossListedObject struct {
	Key          string `xml:"Key"`
	LastModified string `xml:"LastModified"`
	Size         int64  `xml:"Size"`
}

func ossListObjectsPage(ctx context.Context, cfg *operation_setting.OssSetting, prefix, marker string) (*ossListBucketResult, error) {
	return ossListObjectsPageWithClient(ctx, GetOssHttpClient(), cfg, prefix, marker)
}

func ossListObjectsPageWithClient(ctx context.Context, client *http.Client, cfg *operation_setting.OssSetting, prefix, marker string) (*ossListBucketResult, error) {
	endpoint := strings.TrimRight(strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(cfg.Endpoint), "https://"), "http://"), "/")
	bucket := strings.TrimSpace(cfg.Bucket)
	date := time.Now().UTC().Format(http.TimeFormat)
	canonicalResource := "/" + bucket + "/"
	stringToSign := fmt.Sprintf("GET\n\n\n%s\n%s", date, canonicalResource)
	mac := hmac.New(sha1.New, []byte(strings.TrimSpace(cfg.AccessKeySecret)))
	_, _ = mac.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	target := url.URL{Scheme: "https", Host: bucket + "." + endpoint, Path: "/"}
	query := target.Query()
	query.Set("max-keys", strconv.Itoa(ossListObjectsPageSize))
	if prefix != "" {
		query.Set("prefix", prefix)
	}
	if marker != "" {
		query.Set("marker", marker)
	}
	target.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Date", date)
	req.Header.Set("Authorization", "OSS "+strings.TrimSpace(cfg.AccessKeyID)+":"+signature)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("OSS 文件列表读取失败: HTTP %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var result ossListBucketResult
	if err := xml.NewDecoder(io.LimitReader(resp.Body, 32<<20)).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析 OSS 文件列表失败: %w", err)
	}
	return &result, nil
}
