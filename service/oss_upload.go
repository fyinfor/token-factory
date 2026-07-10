package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/setting/operation_setting"
)

const (
	ossPutMaxAttempts = 3
	ossPutBackoffBase = 80 * time.Millisecond
)

// ErrOssNotConfigured OSS 未启用或必填项未配置完整。
var ErrOssNotConfigured = errors.New("未配置阿里云 OSS，请先在运营设置中启用并填写 Endpoint、Bucket、AccessKey 等参数")

// OssUploadMultipartFile 将表单文件上传到已配置的阿里云 OSS（REST PutObject + 签名版本 1），返回对外访问 URL。
// 需 Bucket/对象可读（公共读、CDN 或已授权访问）。
func OssUploadMultipartFile(file *multipart.FileHeader, userID int) (string, error) {
	cfg := operation_setting.GetOssSetting()
	return ossUploadMultipartFileWithPrefix(file, userID, cfg.ObjectKeyPrefix)
}

func ossUploadMultipartFileWithPrefix(file *multipart.FileHeader, userID int, objectPrefix string) (string, error) {
	_ = userID
	if !operation_setting.IsOssUploadReady() {
		return "", ErrOssNotConfigured
	}
	cfg := operation_setting.GetOssSetting()
	maxFileSizeMB := cfg.OssMaxFileSizeMB
	if maxFileSizeMB <= 0 {
		maxFileSizeMB = cfg.MaxFileSizeMB
	}
	if maxFileSizeMB <= 0 {
		maxFileSizeMB = 20
	}
	maxBytes := int64(maxFileSizeMB) * 1024 * 1024
	if file.Size > maxBytes {
		return "", fmt.Errorf("文件超过大小限制（最大 %d MB）", maxFileSizeMB)
	}

	f, err := file.Open()
	if err != nil {
		return "", err
	}
	defer f.Close()

	data, err := io.ReadAll(io.LimitReader(f, maxBytes+1))
	if err != nil {
		return "", err
	}
	if int64(len(data)) > maxBytes {
		return "", fmt.Errorf("文件超过大小限制（最大 %d MB）", maxFileSizeMB)
	}

	objectKey := BuildUploadObjectPath(objectPrefix, uploadFileExt(file.Filename))

	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	if err := ossPutObject(cfg, objectKey, contentType, data); err != nil {
		return "", err
	}
	return publicObjectURL(cfg, objectKey), nil
}

func publicObjectURL(cfg *operation_setting.OssSetting, objectKey string) string {
	base := strings.TrimSpace(cfg.PublicBaseURL)
	if base != "" {
		base = strings.TrimRight(base, "/")
		return base + "/" + strings.TrimLeft(objectKey, "/")
	}
	ep := strings.TrimSpace(cfg.Endpoint)
	ep = strings.TrimPrefix(ep, "https://")
	ep = strings.TrimPrefix(ep, "http://")
	bkt := strings.TrimSpace(cfg.Bucket)
	return fmt.Sprintf("https://%s.%s/%s", bkt, ep, strings.TrimLeft(objectKey, "/"))
}

// ossPutObject 使用 OSS 兼容的 Authorization: OSS AccessKeyId:Signature（HMAC-SHA1），带有限次指数退避重试。
func ossPutObject(cfg *operation_setting.OssSetting, objectKey, contentType string, body []byte) error {
	backoff := ossPutBackoffBase
	for attempt := 0; attempt < ossPutMaxAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(backoff)
			backoff *= 2
		}
		httpStatus, err := ossPutObjectOnce(cfg, objectKey, contentType, body)
		if err == nil {
			return nil
		}
		if !ossPutShouldRetry(httpStatus, err) || attempt == ossPutMaxAttempts-1 {
			return err
		}
	}
	return errors.New("OSS Put: 内部错误（不应到达）")
}

func ossPutShouldRetry(httpStatus int, err error) bool {
	if err == nil {
		return false
	}
	if httpStatus == http.StatusTooManyRequests {
		return true
	}
	if httpStatus >= 500 && httpStatus <= 599 {
		return true
	}
	if httpStatus != 0 {
		return false
	}
	return isTransientOssNetErr(err)
}

func isTransientOssNetErr(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "use of closed network connection") ||
		strings.Contains(msg, "connection reset by peer") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "unexpected eof")
}

// ossPutObjectOnce 单次 PUT；httpStatus 在传输失败时为 0。
func ossPutObjectOnce(cfg *operation_setting.OssSetting, objectKey, contentType string, body []byte) (httpStatus int, err error) {
	endpoint := strings.TrimSpace(cfg.Endpoint)
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "http://")
	bucket := strings.TrimSpace(cfg.Bucket)
	ak := strings.TrimSpace(cfg.AccessKeyID)
	sk := strings.TrimSpace(cfg.AccessKeySecret)

	objectKey = strings.TrimLeft(objectKey, "/")
	canonicalResource := "/" + bucket + "/" + objectKey
	date := time.Now().UTC().Format(http.TimeFormat)

	// 与 OSS 文档一致：Verb、Content-MD5(空)、Content-Type、Date、CanonicalizedResource；无 x-oss-* 头时不在 Date 与 Resource 之间插入额外行。
	stringToSign := fmt.Sprintf("PUT\n\n%s\n%s\n%s", contentType, date, canonicalResource)
	mac := hmac.New(sha1.New, []byte(sk))
	_, _ = mac.Write([]byte(stringToSign))
	sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	auth := "OSS " + ak + ":" + sig

	host := bucket + "." + endpoint
	target := "https://" + host + "/" + objectKey

	req, err := http.NewRequest(http.MethodPut, target, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Date", date)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", auth)
	req.ContentLength = int64(len(body))

	resp, err := GetOssHttpClient().Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return resp.StatusCode, fmt.Errorf("OSS 上传失败: HTTP %d %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return resp.StatusCode, nil
}
