package service

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"
)

// ErrLocalStorageNotConfigured 本地存储目录不可写。
var ErrLocalStorageNotConfigured = fmt.Errorf("本地存储未正确配置，请检查存储目录是否可写")

// LocalUploadMultipartFile 将表单文件保存到本地磁盘，返回对外访问 URL。
// 文件存储路径为 local_storage_path/uploads/local_object_key_prefix/yyyy/mm/dd/uuid.ext
// 对外访问 URL 为 {local_url_prefix}/uploads/local_object_key_prefix/yyyy/mm/dd/uuid.ext
func LocalUploadMultipartFile(file *multipart.FileHeader, userID int) (string, error) {
	_ = userID
	cfg := operation_setting.GetOssSetting()

	maxFileSizeMB := cfg.LocalMaxFileSizeMB
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

	// 确定存储根目录：本地上传固定写入 uploads 目录，配置的文件夹前缀只作为 uploads 下的子目录。
	storeDir := LocalUploadBaseDir(cfg.LocalStoragePath)

	// 生成文件相对路径：local_object_key_prefix/yyyy/mm/dd/uuid.ext
	relPath, err := BuildLocalUploadObjectPath(cfg.LocalObjectKeyPrefix, uploadFileExt(file.Filename))
	if err != nil {
		return "", err
	}

	// 完整文件路径
	fullPath := filepath.Join(storeDir, filepath.FromSlash(relPath))

	// 确保目录存在
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("创建本地存储目录失败: %w", err)
	}

	// 打开上传文件
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("读取上传文件失败: %w", err)
	}
	defer src.Close()

	// 创建目标文件
	dst, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("创建本地文件失败: %w", err)
	}
	defer dst.Close()

	// 复制文件内容（限制大小）
	written, err := io.Copy(dst, io.LimitReader(src, maxBytes+1))
	if err != nil {
		return "", fmt.Errorf("写入本地文件失败: %w", err)
	}
	if written > maxBytes {
		os.Remove(fullPath)
		return "", fmt.Errorf("文件超过大小限制（最大 %d MB）", maxFileSizeMB)
	}

	return localObjectURL(cfg.LocalURLPrefix, path.Join(LocalUploadFolder, relPath)), nil
}

// localObjectURL 根据系统 ServerAddress 和路径生成对外访问 URL。
func localObjectURL(urlPrefix string, objectPath string) string {
	urlPrefix = strings.TrimSpace(urlPrefix)
	if urlPrefix == "" {
		urlPrefix = "/api"
	}
	if strings.HasPrefix(urlPrefix, "http://") || strings.HasPrefix(urlPrefix, "https://") {
		return strings.TrimRight(urlPrefix, "/") + "/" + strings.TrimLeft(objectPath, "/")
	}
	base := strings.TrimRight(system_setting.ServerAddress, "/")
	if base == "" {
		return "/" + path.Join(strings.Trim(urlPrefix, "/"), strings.Trim(objectPath, "/"))
	}
	urlPath := path.Join(strings.Trim(urlPrefix, "/"), strings.Trim(objectPath, "/"))
	return base + "/" + urlPath
}

// EnsureLocalStorageDir 确保本地存储目录存在且可写。
func EnsureLocalStorageDir() error {
	cfg := operation_setting.GetOssSetting()
	if cfg.StorageType != operation_setting.StorageTypeLocal {
		return nil
	}
	if _, err := NormalizeLocalUploadPrefix(cfg.LocalObjectKeyPrefix); err != nil {
		return err
	}
	storeDir := LocalUploadBaseDir(cfg.LocalStoragePath)
	if err := os.MkdirAll(storeDir, 0755); err != nil {
		return fmt.Errorf("无法创建本地存储目录 %s: %w", storeDir, err)
	}
	// 检查可写
	testFile := filepath.Join(storeDir, ".write_test")
	if err := os.WriteFile(testFile, []byte("ok"), 0644); err != nil {
		return fmt.Errorf("本地存储目录 %s 不可写: %w", storeDir, err)
	}
	os.Remove(testFile)
	return nil
}
