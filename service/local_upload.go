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
	"github.com/google/uuid"
)

// ErrLocalStorageNotConfigured 本地存储目录不可写。
var ErrLocalStorageNotConfigured = fmt.Errorf("本地存储未正确配置，请检查存储目录是否可写")

// LocalUploadMultipartFile 将表单文件保存到本地磁盘，返回对外访问 URL。
// 文件存储路径为 local_storage_path/userID/uuid.ext
// 对外访问 URL 为 {ServerAddress}/{object_key_prefix}userID/uuid.ext
func LocalUploadMultipartFile(file *multipart.FileHeader, userID int) (string, error) {
	cfg := operation_setting.GetOssSetting()

	maxBytes := int64(cfg.MaxFileSizeMB) * 1024 * 1024
	if cfg.MaxFileSizeMB <= 0 {
		maxBytes = 20 * 1024 * 1024
	}
	if file.Size > maxBytes {
		return "", fmt.Errorf("文件超过大小限制（最大 %d MB）", cfg.MaxFileSizeMB)
	}

	// 确定存储根目录
	storeDir := strings.TrimSpace(cfg.LocalStoragePath)
	if storeDir == "" {
		storeDir = "uploads"
	}

	// 生成文件相对路径：userID/uuid.ext
	orig := strings.TrimSpace(file.Filename)
	ext := path.Ext(orig)
	if ext != "" && len(ext) > 16 {
		ext = ""
	}
	ext = strings.ToLower(ext)
	relPath := fmt.Sprintf("%d/%s%s", userID, uuid.NewString(), ext)

	// 完整文件路径
	fullPath := filepath.Join(storeDir, relPath)

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
		return "", fmt.Errorf("文件超过大小限制（最大 %d MB）", cfg.MaxFileSizeMB)
	}

	// 构建对外访问 URL：使用 ObjectKeyPrefix 作为 URL 路径前缀
	urlPath := path.Join(strings.Trim(cfg.ObjectKeyPrefix, "/"), relPath)
	return localObjectURL(urlPath), nil
}

// localObjectURL 根据系统 ServerAddress 和路径生成对外访问 URL。
func localObjectURL(urlPath string) string {
	base := strings.TrimRight(system_setting.ServerAddress, "/")
	if base == "" {
		base = "http://localhost:3000"
	}
	return base + "/" + urlPath
}

// EnsureLocalStorageDir 确保本地存储目录存在且可写。
func EnsureLocalStorageDir() error {
	cfg := operation_setting.GetOssSetting()
	if cfg.StorageType != operation_setting.StorageTypeLocal {
		return nil
	}
	storeDir := strings.TrimSpace(cfg.LocalStoragePath)
	if storeDir == "" {
		storeDir = "uploads"
	}
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
