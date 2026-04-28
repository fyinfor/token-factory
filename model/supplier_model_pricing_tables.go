package model

import (
	"time"
)

// SupplierModelPricing 供应商「全局模型定价」表记录（按 supplier_application_id + model_name 唯一）。
// 仅作用于该供应商名下渠道；优先级低于 SupplierChannelModelPricing，高于平台全局 Option。
type SupplierModelPricing struct {
	ID                    int       `json:"id" gorm:"primaryKey;comment:主键ID"`
	SupplierApplicationID int       `json:"supplier_application_id" gorm:"not null;uniqueIndex:uk_supplier_global_model;index;comment:供应商申请ID，关联 supplier_applications.id"`
	ModelName             string    `json:"model_name" gorm:"type:varchar(512);not null;uniqueIndex:uk_supplier_global_model;comment:模型名称（存库为 FormatMatching 规范化名）"`
	QuotaType             int8      `json:"quota_type" gorm:"not null;default:0;comment:计费类型 0按量倍率 1按次固定价"`
	ModelPrice            *float64  `json:"model_price,omitempty" gorm:"comment:按次固定价格（美元/次），QuotaType=1 时生效"`
	ModelRatio            *float64  `json:"model_ratio,omitempty" gorm:"comment:输入倍率（与平台 ModelRatio 语义一致）"`
	CompletionRatio       *float64  `json:"completion_ratio,omitempty" gorm:"comment:输出相对输入倍率"`
	CacheRatio            *float64  `json:"cache_ratio,omitempty" gorm:"comment:缓存命中相对输入倍率"`
	CreateCacheRatio      *float64  `json:"create_cache_ratio,omitempty" gorm:"comment:缓存写入相对输入倍率"`
	ImageRatio            *float64  `json:"image_ratio,omitempty" gorm:"comment:图像计费倍率"`
	AudioRatio            *float64  `json:"audio_ratio,omitempty" gorm:"comment:音频输入倍率"`
	AudioCompletionRatio  *float64  `json:"audio_completion_ratio,omitempty" gorm:"comment:音频输出相对输入倍率"`
	UpdatedByUserID       int       `json:"updated_by_user_id" gorm:"default:0;comment:最后更新人用户ID"`
	CreatedAt             time.Time `json:"created_at" gorm:"comment:创建时间"`
	UpdatedAt             time.Time `json:"updated_at" gorm:"comment:更新时间"`
}

// TableName 指定 GORM 表名 supplier_model_pricings（供应商全局模型定价表）。
func (SupplierModelPricing) TableName() string {
	return "supplier_model_pricings"
}

// SupplierChannelModelPricing 供应商「渠道模型定价」表记录（supplier_application_id + channel_id + model_name 唯一）。
// 优先级最高：计费与定价页展示均先于供应商全局与平台全局。
type SupplierChannelModelPricing struct {
	ID                    int       `json:"id" gorm:"primaryKey;comment:主键ID"`
	SupplierApplicationID int       `json:"supplier_application_id" gorm:"not null;uniqueIndex:uk_supplier_ch_model;index;comment:供应商申请ID，关联 supplier_applications.id"`
	ChannelID             int       `json:"channel_id" gorm:"not null;uniqueIndex:uk_supplier_ch_model;index;comment:渠道ID，关联 channels.id"`
	ModelName             string    `json:"model_name" gorm:"type:varchar(512);not null;uniqueIndex:uk_supplier_ch_model;comment:模型名称（存库为 FormatMatching 规范化名）"`
	QuotaType             int8      `json:"quota_type" gorm:"not null;default:0;comment:计费类型 0按量倍率 1按次固定价"`
	ModelPrice            *float64  `json:"model_price,omitempty" gorm:"comment:按次固定价格（美元/次），QuotaType=1 时生效"`
	ModelRatio            *float64  `json:"model_ratio,omitempty" gorm:"comment:输入倍率"`
	CompletionRatio       *float64  `json:"completion_ratio,omitempty" gorm:"comment:输出相对输入倍率"`
	CacheRatio            *float64  `json:"cache_ratio,omitempty" gorm:"comment:缓存命中相对输入倍率"`
	CreateCacheRatio      *float64  `json:"create_cache_ratio,omitempty" gorm:"comment:缓存写入相对输入倍率"`
	ImageRatio            *float64  `json:"image_ratio,omitempty" gorm:"comment:图像计费倍率"`
	AudioRatio            *float64  `json:"audio_ratio,omitempty" gorm:"comment:音频输入倍率"`
	AudioCompletionRatio  *float64  `json:"audio_completion_ratio,omitempty" gorm:"comment:音频输出相对输入倍率"`
	UpdatedByUserID       int       `json:"updated_by_user_id" gorm:"default:0;comment:最后更新人用户ID"`
	CreatedAt             time.Time `json:"created_at" gorm:"comment:创建时间"`
	UpdatedAt             time.Time `json:"updated_at" gorm:"comment:更新时间"`
}

// TableName 指定 GORM 表名 supplier_channel_model_pricings（供应商渠道模型定价表）。
func (SupplierChannelModelPricing) TableName() string {
	return "supplier_channel_model_pricings"
}
