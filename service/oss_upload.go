package service

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/google/uuid"
)

// OssUploadMultipartFile 将表单文件上传到已配置的阿里云 OSS（REST PutObject + 签名版本 1），返回对外访问 URL。
// 需 Bucket/对象可读（公共读、CDN 或已授权访问）。
func OssUploadMultipartFile(file *multipart.FileHeader, userID int) (string, error) {
	if !operation_setting.IsOssUploadReady() {
		return "", fmt.Errorf("OSS 未启用或未配置完整")
	}
	cfg := operation_setting.GetOssSetting()
	maxBytes := int64(cfg.MaxFileSizeMB) * 1024 * 1024
	if cfg.MaxFileSizeMB <= 0 {
		maxBytes = 20 * 1024 * 1024
	}
	if file.Size > maxBytes {
		return "", fmt.Errorf("文件超过大小限制（最大 %d MB）", cfg.MaxFileSizeMB)
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
		return "", fmt.Errorf("文件超过大小限制（最大 %d MB）", cfg.MaxFileSizeMB)
	}

	orig := strings.TrimSpace(file.Filename)
	ext := path.Ext(orig)
	if ext != "" && len(ext) > 16 {
		ext = ""
	}
	ext = strings.ToLower(ext)
	objectKey := ossObjectKey(cfg.ObjectKeyPrefix, userID, ext)

	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	if err := ossPutObject(cfg, objectKey, contentType, data); err != nil {
		return "", err
	}
	return publicObjectURL(cfg, objectKey), nil
}

func ossObjectKey(prefix string, userID int, ext string) string {
	p := strings.Trim(prefix, "/")
	if p != "" {
		p += "/"
	}
	id := uuid.NewString()
	if ext != "" && !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	return fmt.Sprintf("%s%d/%s%s", p, userID, id, ext)
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

// ossPutObject 使用 OSS 兼容的 Authorization: OSS AccessKeyId:Signature（HMAC-SHA1）。
func ossPutObject(cfg *operation_setting.OssSetting, objectKey, contentType string, body []byte) error {
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
		return err
	}
	req.Header.Set("Date", date)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", auth)
	req.ContentLength = int64(len(body))

	resp, err := GetHttpClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("OSS 上传失败: HTTP %d %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return nil
}
