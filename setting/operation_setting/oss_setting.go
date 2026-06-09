package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

// StorageTypeLocal 本地存储
const StorageTypeLocal = "local"

// StorageTypeOSS 阿里云 OSS 存储
const StorageTypeOSS = "oss"

// OssSetting 文件上传配置（在控制台 运营设置 中由超级管理员配置）。
// StorageType 控制存储后端："local"（本地磁盘，默认）或 "oss"（阿里云 OSS）。
type OssSetting struct {
	Enabled         bool   `json:"enabled"`
	StorageType     string `json:"storage_type"` // "local" 或 "oss"，默认 "local"
	Endpoint        string `json:"endpoint"`     // 如 oss-cn-guangzhou.aliyuncs.com，不含协议（仅 OSS）
	Bucket          string `json:"bucket"`
	AccessKeyID     string `json:"access_key_id"`
	AccessKeySecret string `json:"access_key_secret"`
	// PublicBaseURL 对外访问基址，可填 CDN/自定义域名，如 https://img.example.com；为空则使用 https://{bucket}.{endpoint}/（仅 OSS）
	PublicBaseURL string `json:"public_base_url"`
	// ObjectKeyPrefix 对象键前缀，如 uploads/（OSS 与本地存储共用）
	ObjectKeyPrefix string `json:"object_key_prefix"`
	// MaxFileSizeMB 单文件大小上限（MB）
	MaxFileSizeMB int `json:"max_file_size_mb"`
	// LocalStoragePath 本地存储目录（相对于程序工作目录），默认 "uploads"
	LocalStoragePath string `json:"local_storage_path"`
}

var ossSetting = OssSetting{
	StorageType:      StorageTypeLocal,
	ObjectKeyPrefix:  "uploads/",
	MaxFileSizeMB:    20,
	LocalStoragePath: "uploads",
}

func init() {
	config.GlobalConfig.Register("oss_setting", &ossSetting)
}

// GetOssSetting 返回 OSS 配置（运行时指针，勿并发写）。
func GetOssSetting() *OssSetting {
	return &ossSetting
}

// IsUploadReady 是否已配置完整且启用上传（本地存储始终可用，OSS 需填写完整参数）。
func IsUploadReady() bool {
	s := &ossSetting
	if !s.Enabled {
		return false
	}
	if s.StorageType == StorageTypeLocal {
		return true
	}
	return IsOssUploadReady()
}

// IsOssUploadReady 是否已配置完整且启用 OSS 上传。
func IsOssUploadReady() bool {
	s := &ossSetting
	if !s.Enabled || s.Endpoint == "" || s.Bucket == "" || s.AccessKeyID == "" || s.AccessKeySecret == "" {
		return false
	}
	return true
}
