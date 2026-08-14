package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

// VideoUpscaleTemplate 视频超分模版（腾讯云 MPS 转码模版），供渠道超分规则下拉选择。
type VideoUpscaleTemplate struct {
	Id   uint64 `json:"id"`   // MPS 转码模版 Definition ID
	Name string `json:"name"` // 展示名称
}

// VideoUpscaleSetting 视频超分全局配置（控制台 系统设置 中由超级管理员维护）。
// SecretId/SecretKey/OutputPath 任一为空视为未启用，业务完全走原有流程。
type VideoUpscaleSetting struct {
	SecretId  string `json:"secret_id"`  // 腾讯云 SecretId（敏感，接口返回时脱敏）
	SecretKey string `json:"secret_key"` // 腾讯云 SecretKey（敏感，接口返回时脱敏）
	// OutputPath COS 输出路径，格式：https://<bucket>.cos.<region>.myqcloud.com/<prefix>/
	// 超分输出对象写入该路径下，最终对外 URL 按同一基址重组。
	OutputPath string                 `json:"output_path"`
	Templates  []VideoUpscaleTemplate `json:"templates"`
}

var videoUpscaleSetting = VideoUpscaleSetting{
	Templates: []VideoUpscaleTemplate{},
}

func init() {
	config.GlobalConfig.Register("video_upscale_setting", &videoUpscaleSetting)
}

// GetVideoUpscaleSetting 返回视频超分全局配置（运行时指针，勿并发写）。
func GetVideoUpscaleSetting() *VideoUpscaleSetting {
	return &videoUpscaleSetting
}

// IsVideoUpscaleReady 全局配置是否完整；不完整时渠道规则命中也不启用超分。
func IsVideoUpscaleReady() bool {
	s := &videoUpscaleSetting
	return s.SecretId != "" && s.SecretKey != "" && s.OutputPath != ""
}
