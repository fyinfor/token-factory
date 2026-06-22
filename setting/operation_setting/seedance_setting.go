package operation_setting

import (
	"strings"

	"github.com/QuantumNous/new-api/setting/config"
)

// 默认虚拟人像合规协议文案（勾选框文案）。
const (
	defaultPortraitAgreementZh = "我确认有权使用所提交的信息及内容，承诺上传的人像素材均为已授权虚拟人像，并已阅读且同意 《虚拟人像合规承诺函》"
	defaultPortraitAgreementEn = "I confirm that I have the right to use the submitted information and content, and undertake that the uploaded portrait materials are all authorized virtual portraits. I have read and agree to the 《Virtual Portrait Compliance Commitment Letter》"

	defaultPortraitAgreementDetailZh = "《虚拟人像合规承诺函》\n\n1. 本人承诺所上传的人像素材均已获得肖像权人的明确授权，可用于虚拟人视频合成等 AIGC 场景。\n2. 本人承诺不上传涉及违法、侵权、色情、暴力等违规内容。\n3. 因素材授权问题引发的一切法律责任由上传者本人承担。\n4. 平台仅提供技术服务，有权对违规素材进行下架处理。"
	defaultPortraitAgreementDetailEn = "《Virtual Portrait Compliance Commitment Letter》\n\n1. I undertake that all uploaded portrait materials have obtained explicit authorization from the portrait rights holders and can be used for AIGC scenarios such as virtual human video synthesis.\n2. I undertake not to upload any illegal, infringing, pornographic, violent or other non-compliant content.\n3. All legal liabilities arising from material authorization issues shall be borne by the uploader.\n4. The platform only provides technical services and reserves the right to remove non-compliant materials."
)

// SeedanceSetting Seedance2.0 素材库相关配置（在控制台 系统设置-其他设置-素材设置 中由超级管理员配置）。
type SeedanceSetting struct {
	// Enabled 是否启用素材库功能。
	Enabled bool `json:"enabled"`
	// APIBaseURL 素材库 API 基础地址，所有素材相关接口统一拼接该地址请求，如 https://ark.cn-beijing.volces.com。
	APIBaseURL string `json:"api_base_url"`
	// APIKey 素材库 API 鉴权密钥，作为 Authorization: Bearer 头发送（可选）。
	APIKey string `json:"api_key"`
	// MaxImageSizeMB 单张图片上传大小上限（MB）。
	MaxImageSizeMB int `json:"max_image_size_mb"`
	// AgreementZh 虚拟人像合规协议文案（中文，勾选框文案）。
	AgreementZh string `json:"agreement_zh"`
	// AgreementEn 虚拟人像合规协议文案（英文，勾选框文案）。
	AgreementEn string `json:"agreement_en"`
	// AgreementDetailZh 虚拟人像合规协议详情（中文，弹窗内容）。
	AgreementDetailZh string `json:"agreement_detail_zh"`
	// AgreementDetailEn 虚拟人像合规协议详情（英文，弹窗内容）。
	AgreementDetailEn string `json:"agreement_detail_en"`
}

var seedanceSetting = SeedanceSetting{
	Enabled:           false,
	APIBaseURL:        "",
	APIKey:            "",
	MaxImageSizeMB:    10,
	AgreementZh:       defaultPortraitAgreementZh,
	AgreementEn:       defaultPortraitAgreementEn,
	AgreementDetailZh: defaultPortraitAgreementDetailZh,
	AgreementDetailEn: defaultPortraitAgreementDetailEn,
}

func init() {
	config.GlobalConfig.Register("seedance_setting", &seedanceSetting)
}

// GetSeedanceSetting 返回 Seedance 素材库配置（运行时指针，勿并发写）。
func GetSeedanceSetting() *SeedanceSetting {
	return &seedanceSetting
}

// GetMaterialAPIBaseURL 返回去除尾部斜杠的素材库 API 基础地址。
func GetMaterialAPIBaseURL() string {
	return strings.TrimRight(strings.TrimSpace(seedanceSetting.APIBaseURL), "/")
}

// GetPortraitAgreement 根据语言返回合规协议勾选框文案，空配置时回退默认文案。
func GetPortraitAgreement(lang string) string {
	if strings.HasPrefix(strings.ToLower(lang), "en") {
		if strings.TrimSpace(seedanceSetting.AgreementEn) != "" {
			return seedanceSetting.AgreementEn
		}
		return defaultPortraitAgreementEn
	}
	if strings.TrimSpace(seedanceSetting.AgreementZh) != "" {
		return seedanceSetting.AgreementZh
	}
	return defaultPortraitAgreementZh
}

// IsSeedanceReady 判断素材库功能是否已启用且基础地址已配置。
func IsSeedanceReady() bool {
	return seedanceSetting.Enabled && GetMaterialAPIBaseURL() != ""
}
