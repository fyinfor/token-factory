package tencentvod

import (
	"math"
	"strings"
)

// AigcVideoOutputConfig 对应腾讯云点播 CreateAigcVideoTask / DescribeTaskDetail 的 OutputConfig。
// 计费核心字段：Resolution、Duration、AspectRatio。
type AigcVideoOutputConfig struct {
	StorageMode           string  `json:"StorageMode,omitempty"`
	MediaName             string  `json:"MediaName,omitempty"`
	ClassId               *int    `json:"ClassId,omitempty"`
	ExpireTime            string  `json:"ExpireTime,omitempty"`
	Duration              float64 `json:"Duration,omitempty"` // 官方类型 Float（秒）
	Resolution            string  `json:"Resolution,omitempty"`
	AspectRatio           string  `json:"AspectRatio,omitempty"`
	AudioGeneration       string  `json:"AudioGeneration,omitempty"`
	PersonGeneration      string  `json:"PersonGeneration,omitempty"`
	InputComplianceCheck  string  `json:"InputComplianceCheck,omitempty"`
	OutputComplianceCheck string  `json:"OutputComplianceCheck,omitempty"`
	EnhanceSwitch         string  `json:"EnhanceSwitch,omitempty"`
	OffPeak               string  `json:"OffPeak,omitempty"`
	FrameInterpolate      string  `json:"FrameInterpolate,omitempty"`
	LogoAdd               string  `json:"LogoAdd,omitempty"`
	EnableBGM             string  `json:"EnableBGM,omitempty"`
}

// AigcVideoFileInfo 对应 AigcVideoTaskInputFileInfo（兼容 Type=Url / Base64）。
type AigcVideoFileInfo struct {
	Type              string `json:"Type,omitempty"`
	Category          string `json:"Category,omitempty"`
	FileId            string `json:"FileId,omitempty"`
	Url               string `json:"Url,omitempty"`
	Base64            string `json:"Base64,omitempty"`
	ReferenceType     string `json:"ReferenceType,omitempty"`
	ObjectId          string `json:"ObjectId,omitempty"`
	VoiceId           string `json:"VoiceId,omitempty"`
	KeepOriginalSound string `json:"KeepOriginalSound,omitempty"`
	Usage             string `json:"Usage,omitempty"`
	Text              string `json:"Text,omitempty"`
}

// CreateAigcVideoTaskRequest 腾讯云 CreateAigcVideoTask 请求体（网关校验用）。
type CreateAigcVideoTaskRequest struct {
	SubAppId        uint64                 `json:"SubAppId"`
	ModelName       string                 `json:"ModelName"`
	ModelVersion    string                 `json:"ModelVersion"`
	FileInfos       []AigcVideoFileInfo    `json:"FileInfos,omitempty"`
	Prompt          string                 `json:"Prompt,omitempty"`
	NegativePrompt  string                 `json:"NegativePrompt,omitempty"`
	EnhancePrompt   string                 `json:"EnhancePrompt,omitempty"`
	OutputConfig    *AigcVideoOutputConfig `json:"OutputConfig,omitempty"`
	LastFrameFileId string                 `json:"LastFrameFileId,omitempty"`
	LastFrameUrl    string                 `json:"LastFrameUrl,omitempty"`
	ExtInfo         string                 `json:"ExtInfo,omitempty"`
}

// BillingOutputSpec 计费比对用的三大核心字段快照。
type BillingOutputSpec struct {
	Resolution  string
	Duration    int // 向上取整到秒，用于按秒计费
	AspectRatio string
}

// HasBillingCore 三大计费字段是否齐全。
func (s BillingOutputSpec) HasBillingCore() bool {
	return strings.TrimSpace(s.Resolution) != "" && s.Duration > 0 && strings.TrimSpace(s.AspectRatio) != ""
}

// ToBillingSpec 从 OutputConfig 提取计费核心字段。
func (c *AigcVideoOutputConfig) ToBillingSpec() BillingOutputSpec {
	if c == nil {
		return BillingOutputSpec{}
	}
	dur := int(math.Ceil(c.Duration))
	if dur <= 0 && c.Duration > 0 {
		dur = 1
	}
	return BillingOutputSpec{
		Resolution:  strings.TrimSpace(c.Resolution),
		Duration:    dur,
		AspectRatio: strings.TrimSpace(c.AspectRatio),
	}
}
