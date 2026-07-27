package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/bytedance/gopkg/util/gopool"
)

const (
	temporaryUploadCleanupInterval  = 10 * time.Minute
	temporaryUploadCleanupBatchSize = 100
)

var (
	temporaryUploadCleanupOnce    sync.Once
	temporaryUploadCleanupRunning atomic.Bool
)

func StartTemporaryUploadCleanupTask() {
	temporaryUploadCleanupOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}
		gopool.Go(func() {
			retention := playgroundUploadRetention(operation_setting.GetOssSetting().PlaygroundRetentionHours)
			logger.LogInfo(context.Background(), fmt.Sprintf(
				"temporary upload cleanup task started: tick=%s retention=%s",
				temporaryUploadCleanupInterval,
				retention,
			))
			runTemporaryUploadCleanupOnce()
			ticker := time.NewTicker(temporaryUploadCleanupInterval)
			defer ticker.Stop()
			for range ticker.C {
				runTemporaryUploadCleanupOnce()
			}
		})
	})
}

func runTemporaryUploadCleanupOnce() {
	if !temporaryUploadCleanupRunning.CompareAndSwap(false, true) {
		return
	}
	defer temporaryUploadCleanupRunning.Store(false)

	ctx := context.Background()
	uploads, err := model.ListExpiredUploadObjects(time.Now().Unix(), temporaryUploadCleanupBatchSize)
	if err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("temporary upload cleanup query failed: %v", err))
		return
	}
	deleted := 0
	for _, upload := range uploads {
		if err := deleteIndexedUploadStorage(upload); err != nil {
			_ = model.MarkUploadObjectCleanupFailure(upload.ID, err.Error())
			logger.LogWarn(ctx, fmt.Sprintf("temporary upload cleanup failed: id=%d storage=%s object=%s err=%v", upload.ID, upload.StorageType, upload.ObjectKey, err))
			continue
		}
		if err := model.DeleteUploadObject(upload.ID); err != nil {
			logger.LogWarn(ctx, fmt.Sprintf("temporary upload record delete failed: id=%d err=%v", upload.ID, err))
			continue
		}
		deleted++
	}
	if deleted > 0 {
		logger.LogInfo(ctx, fmt.Sprintf("temporary upload cleanup completed: deleted=%d", deleted))
	}
}

func DeleteStoredUpload(upload *UploadResult) error {
	if upload == nil {
		return nil
	}
	return deleteStoredObject(
		upload.StorageType,
		upload.ObjectKey,
		upload.StorageBase,
		upload.Endpoint,
		upload.Bucket,
	)
}

func deleteIndexedUploadStorage(upload model.TemporaryUpload) error {
	return deleteStoredObject(
		upload.StorageType,
		upload.ObjectKey,
		upload.StorageBase,
		upload.Endpoint,
		upload.Bucket,
	)
}

// DeleteIndexedUploadObject removes the stored object first and then its index row.
// Keeping the row when storage deletion fails makes the operation retryable.
func DeleteIndexedUploadObject(upload *model.TemporaryUpload) error {
	if upload == nil {
		return nil
	}
	if err := deleteIndexedUploadStorage(*upload); err != nil {
		return err
	}
	return model.DeleteUploadObject(upload.ID)
}

func deleteStoredObject(storageType, objectKey, storageBase, endpoint, bucket string) error {
	switch storageType {
	case operation_setting.StorageTypeLocal:
		return deleteLocalStoredObject(storageBase, objectKey)
	case operation_setting.StorageTypeOSS:
		return ossDeleteObject(endpoint, bucket, objectKey)
	default:
		return fmt.Errorf("未知的临时文件存储类型: %s", storageType)
	}
}

func deleteLocalStoredObject(storageBase, objectKey string) error {
	cleaned := path.Clean(strings.TrimSpace(objectKey))
	if cleaned == "." || path.IsAbs(cleaned) || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return fmt.Errorf("无效的本地临时文件对象键")
	}
	baseDir, err := filepath.Abs(LocalUploadBaseDir(storageBase))
	if err != nil {
		return err
	}
	target, err := filepath.Abs(filepath.Join(baseDir, filepath.FromSlash(cleaned)))
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(baseDir, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("本地临时文件路径越界")
	}
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		return err
	}
	removeEmptyUploadDirs(filepath.Dir(target), baseDir)
	return nil
}

func removeEmptyUploadDirs(dir, baseDir string) {
	for dir != baseDir {
		rel, err := filepath.Rel(baseDir, dir)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return
		}
		if err := os.Remove(dir); err != nil {
			return
		}
		dir = filepath.Dir(dir)
	}
}

func ossDeleteObject(endpoint, bucket, objectKey string) error {
	cfg := operation_setting.GetOssSetting()
	ak := strings.TrimSpace(cfg.AccessKeyID)
	sk := strings.TrimSpace(cfg.AccessKeySecret)
	if ak == "" || sk == "" {
		return fmt.Errorf("删除 OSS 临时文件需要有效的 AccessKey")
	}
	endpoint = strings.TrimSpace(endpoint)
	bucket = strings.TrimSpace(bucket)
	objectKey = strings.TrimLeft(strings.TrimSpace(objectKey), "/")
	if endpoint == "" || bucket == "" || objectKey == "" {
		return fmt.Errorf("OSS 临时文件记录不完整")
	}

	backoff := ossPutBackoffBase
	for attempt := 0; attempt < ossPutMaxAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(backoff)
			backoff *= 2
		}
		status, err := ossDeleteObjectOnce(endpoint, bucket, objectKey, ak, sk)
		if err == nil {
			return nil
		}
		if !ossRequestShouldRetry(status, err) || attempt == ossPutMaxAttempts-1 {
			return err
		}
	}
	return fmt.Errorf("OSS Delete: 内部错误（不应到达）")
}

func ossDeleteObjectOnce(endpoint, bucket, objectKey, accessKeyID, accessKeySecret string) (int, error) {
	return ossDeleteObjectOnceWithClient(GetOssHttpClient(), endpoint, bucket, objectKey, accessKeyID, accessKeySecret)
}

func ossDeleteObjectOnceWithClient(client *http.Client, endpoint, bucket, objectKey, accessKeyID, accessKeySecret string) (int, error) {
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "http://")
	endpoint = strings.TrimRight(endpoint, "/")
	canonicalResource := "/" + bucket + "/" + objectKey
	date := time.Now().UTC().Format(http.TimeFormat)
	stringToSign := fmt.Sprintf("DELETE\n\n\n%s\n%s", date, canonicalResource)
	mac := hmac.New(sha1.New, []byte(accessKeySecret))
	_, _ = mac.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	target := "https://" + bucket + "." + endpoint + "/" + objectKey
	req, err := http.NewRequest(http.MethodDelete, target, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Date", date)
	req.Header.Set("Authorization", "OSS "+accessKeyID+":"+signature)
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return resp.StatusCode, nil
	}
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return resp.StatusCode, fmt.Errorf("OSS 删除失败: HTTP %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return resp.StatusCode, nil
}
