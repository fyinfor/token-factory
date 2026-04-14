package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

// OssSetting 阿里云 OSS 通用上传配置（在控制台 运营设置 中由超级管理员配置）。
type OssSetting struct {
	Enabled         bool   `json:"enabled"`
	Endpoint        string `json:"endpoint"` // 如 oss-cn-guangzhou.aliyuncs.com，不含协议
	Bucket          string `json:"bucket"`
	AccessKeyID     string `json:"access_key_id"`
	AccessKeySecret string `json:"access_key_secret"`
	// PublicBaseURL 对外访问基址，可填 CDN/自定义域名，如 https://img.example.com；为空则使用 https://{bucket}.{endpoint}/
	PublicBaseURL string `json:"public_base_url"`
	// ObjectKeyPrefix 对象键前缀，如 uploads/
	ObjectKeyPrefix string `json:"object_key_prefix"`
	// MaxFileSizeMB 单文件大小上限（MB）
	MaxFileSizeMB int `json:"max_file_size_mb"`
}

var ossSetting = OssSetting{
	ObjectKeyPrefix: "uploads/",
	MaxFileSizeMB:   20,
}

func init() {
	config.GlobalConfig.Register("oss_setting", &ossSetting)
}

// GetOssSetting 返回 OSS 配置（运行时指针，勿并发写）。
func GetOssSetting() *OssSetting {
	return &ossSetting
}

// IsOssUploadReady 是否已配置完整且启用上传。
func IsOssUploadReady() bool {
	s := &ossSetting
	if !s.Enabled || s.Endpoint == "" || s.Bucket == "" || s.AccessKeyID == "" || s.AccessKeySecret == "" {
		return false
	}
	return true
}
