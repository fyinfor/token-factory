package model

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

// SupplierCapability 供应商技术能力档案表（与 supplier_applications 一对一）。
type SupplierCapability struct {
	ID                        int    `json:"id" gorm:"primaryKey;comment:主键ID"`
	SupplierApplicationID     int    `json:"supplier_application_id" gorm:"uniqueIndex;not null;comment:供应商申请ID"`
	CoreServiceTypes          string `json:"core_service_types" gorm:"type:text;comment:核心服务类型(JSON数组)"`
	SupportedModels           string `json:"supported_models" gorm:"type:text;comment:支持的模型(JSON数组)"`
	SupportedModelNotes       string `json:"supported_model_notes" gorm:"type:text;comment:支持模型补充说明"`
	SupportedAPIEndpoints     string `json:"supported_api_endpoints" gorm:"type:text;comment:支持的API接口(JSON数组)"`
	SupportedAPIEndpointExtra string `json:"supported_api_endpoint_extra" gorm:"type:text;comment:API接口补充说明"`
	SupportedParams           string `json:"supported_params" gorm:"type:text;comment:支持参数配置(JSON数组)"`
	SupportedParamsExtra      string `json:"supported_params_extra" gorm:"type:text;comment:参数配置补充说明"`
	StreamingSupported        bool   `json:"streaming_supported" gorm:"type:boolean;default:false;comment:是否支持流式响应"`
	StreamingNotes            string `json:"streaming_notes" gorm:"type:text;comment:流式响应说明"`
	StructuredOutputSupported bool   `json:"structured_output_supported" gorm:"type:boolean;default:false;comment:是否支持结构化输出"`
	StructuredOutputNotes     string `json:"structured_output_notes" gorm:"type:text;comment:结构化输出说明"`
	MultimodalTypes           string `json:"multimodal_types" gorm:"type:text;comment:多模态支持类型(JSON数组)"`
	MultimodalExtra           string `json:"multimodal_extra" gorm:"type:text;comment:多模态补充说明"`
	PricingModes              string `json:"pricing_modes" gorm:"type:text;comment:定价模式(JSON数组)"`
	ReferenceInputPrice       string `json:"reference_input_price" gorm:"type:varchar(64);comment:参考输入单价(USD/1K Token)"`
	ReferenceOutputPrice      string `json:"reference_output_price" gorm:"type:varchar(64);comment:参考输出单价(USD/1K Token)"`
	FailureBillingMode        string `json:"failure_billing_mode" gorm:"type:varchar(32);comment:故障计费规则(bill/no_bill)"`
	FailureBillingNotes       string `json:"failure_billing_notes" gorm:"type:text;comment:故障计费说明"`
	APIBaseURLs               string `json:"api_base_urls" gorm:"type:text;comment:API接口地址(JSON数组)"`
	OpenAICompatible          bool   `json:"openai_compatible" gorm:"type:boolean;default:false;comment:是否兼容OpenAI规范"`
	TruthCommitmentConfirmed  bool   `json:"truth_commitment_confirmed" gorm:"type:boolean;default:false;comment:信息真实性承诺"`
	CreatedAt                 int64  `json:"created_at" gorm:"type:bigint;index;comment:创建时间戳"`
	UpdatedAt                 int64  `json:"updated_at" gorm:"type:bigint;comment:更新时间戳"`
}

// GetSupplierCapabilityByApplicationID 根据申请ID查询供应商技术能力档案。
func GetSupplierCapabilityByApplicationID(applicationID int) (*SupplierCapability, error) {
	var item SupplierCapability
	if err := DB.Where("supplier_application_id = ?", applicationID).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

// UpsertSupplierCapabilityByApplicationID 按申请ID新增或更新供应商技术能力档案。
func UpsertSupplierCapabilityByApplicationID(applicationID int, capability *SupplierCapability) (*SupplierCapability, error) {
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	now := time.Now().Unix()
	var existing SupplierCapability
	if err := tx.Where("supplier_application_id = ?", applicationID).First(&existing).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			capability.SupplierApplicationID = applicationID
			capability.CreatedAt = now
			capability.UpdatedAt = now
			if err = tx.Create(capability).Error; err != nil {
				tx.Rollback()
				return nil, err
			}
			if err = tx.Commit().Error; err != nil {
				return nil, err
			}
			return capability, nil
		}
		tx.Rollback()
		return nil, err
	}
	updates := map[string]any{
		"core_service_types":           capability.CoreServiceTypes,
		"supported_models":             capability.SupportedModels,
		"supported_model_notes":        capability.SupportedModelNotes,
		"supported_api_endpoints":      capability.SupportedAPIEndpoints,
		"supported_api_endpoint_extra": capability.SupportedAPIEndpointExtra,
		"supported_params":             capability.SupportedParams,
		"supported_params_extra":       capability.SupportedParamsExtra,
		"streaming_supported":          capability.StreamingSupported,
		"streaming_notes":              capability.StreamingNotes,
		"structured_output_supported":  capability.StructuredOutputSupported,
		"structured_output_notes":      capability.StructuredOutputNotes,
		"multimodal_types":             capability.MultimodalTypes,
		"multimodal_extra":             capability.MultimodalExtra,
		"pricing_modes":                capability.PricingModes,
		"reference_input_price":        capability.ReferenceInputPrice,
		"reference_output_price":       capability.ReferenceOutputPrice,
		"failure_billing_mode":         capability.FailureBillingMode,
		"failure_billing_notes":        capability.FailureBillingNotes,
		"api_base_urls":                capability.APIBaseURLs,
		"openai_compatible":            capability.OpenAICompatible,
		"truth_commitment_confirmed":   capability.TruthCommitmentConfirmed,
		"updated_at":                   now,
	}
	if err := tx.Model(&SupplierCapability{}).Where("supplier_application_id = ?", applicationID).Updates(updates).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	existing.CoreServiceTypes = capability.CoreServiceTypes
	existing.SupportedModels = capability.SupportedModels
	existing.SupportedModelNotes = capability.SupportedModelNotes
	existing.SupportedAPIEndpoints = capability.SupportedAPIEndpoints
	existing.SupportedAPIEndpointExtra = capability.SupportedAPIEndpointExtra
	existing.SupportedParams = capability.SupportedParams
	existing.SupportedParamsExtra = capability.SupportedParamsExtra
	existing.StreamingSupported = capability.StreamingSupported
	existing.StreamingNotes = capability.StreamingNotes
	existing.StructuredOutputSupported = capability.StructuredOutputSupported
	existing.StructuredOutputNotes = capability.StructuredOutputNotes
	existing.MultimodalTypes = capability.MultimodalTypes
	existing.MultimodalExtra = capability.MultimodalExtra
	existing.PricingModes = capability.PricingModes
	existing.ReferenceInputPrice = capability.ReferenceInputPrice
	existing.ReferenceOutputPrice = capability.ReferenceOutputPrice
	existing.FailureBillingMode = capability.FailureBillingMode
	existing.FailureBillingNotes = capability.FailureBillingNotes
	existing.APIBaseURLs = capability.APIBaseURLs
	existing.OpenAICompatible = capability.OpenAICompatible
	existing.TruthCommitmentConfirmed = capability.TruthCommitmentConfirmed
	existing.UpdatedAt = now
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}
	return &existing, nil
}

// IsSupplierCapabilityNotFound 判断是否未找到供应商技术能力档案。
func IsSupplierCapabilityNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

// IsSupplierCapabilityComplete 判断供应商技术能力档案是否满足审批通过最低必填条件。
func IsSupplierCapabilityComplete(capability *SupplierCapability) bool {
	if capability == nil {
		return false
	}
	if strings.TrimSpace(capability.CoreServiceTypes) == "" {
		return false
	}
	if strings.TrimSpace(capability.SupportedModels) == "" {
		return false
	}
	if strings.TrimSpace(capability.SupportedAPIEndpoints) == "" {
		return false
	}
	if strings.TrimSpace(capability.SupportedParams) == "" {
		return false
	}
	if strings.TrimSpace(capability.PricingModes) == "" {
		return false
	}
	if strings.TrimSpace(capability.FailureBillingMode) == "" {
		return false
	}
	if strings.TrimSpace(capability.APIBaseURLs) == "" {
		return false
	}
	return capability.TruthCommitmentConfirmed
}
