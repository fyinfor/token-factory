package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

// SupplierApplicationSubmitRequest 供应商提交申请请求体。
type SupplierApplicationSubmitRequest struct {
	ApplicantUserID     int    `json:"applicant_user_id"`
	SupplierAlias       string `json:"supplier_alias"`
	SupplierType        string `json:"supplier_type"`
	CompanyName         string `json:"company_name"`
	CreditCode          string `json:"credit_code"`
	BusinessLicenseURL  string `json:"business_license_url"`
	BusinessLicenseFile string `json:"business_license_file"`
	CompanyLogoURL      string `json:"company_logo_url"`
	LegalRepresentative string `json:"legal_representative"`
	CompanySize         string `json:"company_size"`
	ContactName         string `json:"contact_name"`
	ContactMobile       string `json:"contact_mobile"`
	ContactWechat       string `json:"contact_wechat"`
}

// getSupplierApplicationMissingFieldMessage 返回供应商申请缺失字段的中文提示。
func getSupplierApplicationMissingFieldMessage(req SupplierApplicationSubmitRequest) string {
	switch {
	case req.CompanyName == "":
		return "请填写企业/主体名称（company_name）"
	case req.CreditCode == "":
		return "请填写统一社会信用代码（credit_code）"
	case req.BusinessLicenseURL == "":
		return "请上传营业执照（business_license_url）"
	case req.CompanyLogoURL == "":
		return "请上传企业Logo（company_logo_url）"
	case req.LegalRepresentative == "":
		return "请填写法人/经营者姓名（legal_representative）"
	case req.ContactName == "":
		return "请填写对接人姓名（contact_name）"
	case req.ContactMobile == "":
		return "请填写对接人手机号（contact_mobile）"
	case req.ContactWechat == "":
		return "请填写对接人微信/企业微信（contact_wechat）"
	default:
		return ""
	}
}

// SupplierApplicationReviewRequest 管理员审核请求体。
type SupplierApplicationReviewRequest struct {
	Status        int    `json:"status"`
	Reason        string `json:"reason"`
	SupplierAlias string `json:"supplier_alias"`
	SupplierType  string `json:"supplier_type"`
}

var allowedSupplierTypes = map[string]struct{}{
	"公有云":   {},
	"AIDC":  {},
	"企业中转站": {},
	"个人中转站": {},
}

// isValidSupplierType 判断供应商类型是否在允许枚举内。
func isValidSupplierType(supplierType string) bool {
	_, ok := allowedSupplierTypes[supplierType]
	return ok
}

// SupplierDeactivateRequest 供应商注销请求体。
type SupplierDeactivateRequest struct {
	SupplierID int    `json:"supplier_id"`
	Reason     string `json:"reason"`
}

// PublishUserMessageRequest 管理员发布站内消息请求体。
type PublishUserMessageRequest struct {
	ReceiverUserID  int    `json:"receiver_user_id"`
	ReceiverMinRole int    `json:"receiver_min_role"`
	Type            string `json:"type"`
	Title           string `json:"title"`
	Content         string `json:"content"`
	BizType         string `json:"biz_type"`
	BizID           int    `json:"biz_id"`
}

// SupplierApplicationUpdateRequest 供应商修改申请请求体（必须带申请ID）。
type SupplierApplicationUpdateRequest struct {
	ID int `json:"id"`
	SupplierApplicationSubmitRequest
}

// SupplierCapabilityUpsertRequest 供应商技术能力档案写入请求体。
type SupplierCapabilityUpsertRequest struct {
	CoreServiceTypes          []string `json:"core_service_types"`
	SupportedModels           []string `json:"supported_models"`
	SupportedModelNotes       string   `json:"supported_model_notes"`
	SupportedAPIEndpoints     []string `json:"supported_api_endpoints"`
	SupportedAPIEndpointExtra string   `json:"supported_api_endpoint_extra"`
	SupportedParams           []string `json:"supported_params"`
	SupportedParamsExtra      string   `json:"supported_params_extra"`
	StreamingSupported        bool     `json:"streaming_supported"`
	StreamingNotes            string   `json:"streaming_notes"`
	StructuredOutputSupported bool     `json:"structured_output_supported"`
	StructuredOutputNotes     string   `json:"structured_output_notes"`
	MultimodalTypes           []string `json:"multimodal_types"`
	MultimodalExtra           string   `json:"multimodal_extra"`
	PricingModes              []string `json:"pricing_modes"`
	ReferenceInputPrice       string   `json:"reference_input_price"`
	ReferenceOutputPrice      string   `json:"reference_output_price"`
	FailureBillingMode        string   `json:"failure_billing_mode"`
	FailureBillingNotes       string   `json:"failure_billing_notes"`
	APIBaseURLs               []string `json:"api_base_urls"`
	OpenAICompatible          bool     `json:"openai_compatible"`
	TruthCommitmentConfirmed  bool     `json:"truth_commitment_confirmed"`
}

// requireApprovedSupplierApplication 要求当前用户为审核通过供应商。
func requireApprovedSupplierApplication(c *gin.Context) (*model.SupplierApplication, bool) {
	app, err := model.GetApprovedSupplierApplicationByApplicant(c.GetInt("id"))
	if err != nil {
		if model.IsSupplierApplicationNotFound(err) {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "当前用户未通过供应商审核，无法操作渠道或模型"})
			return nil, false
		}
		common.ApiError(c, err)
		return nil, false
	}
	return app, true
}

// readOptionalStatusQuery 读取可选 status 查询参数。
func readOptionalStatusQuery(c *gin.Context) (*int, error) {
	statusRaw := strings.TrimSpace(c.Query("status"))
	if statusRaw == "" {
		return nil, nil
	}
	status, err := strconv.Atoi(statusRaw)
	if err != nil {
		return nil, err
	}
	if status < model.SupplierApplicationStatusPending || status > model.SupplierApplicationStatusDeactivated {
		return nil, errors.New("invalid status")
	}
	return &status, nil
}

// readSupplierStatusListQuery 读取供应商列表状态查询参数（支持逗号分隔）。
// 未传时默认返回“审核通过 + 已注销”。
func readSupplierStatusListQuery(c *gin.Context) ([]int, error) {
	statusRaw := strings.TrimSpace(c.Query("status"))
	if statusRaw == "" {
		return []int{model.SupplierApplicationStatusApproved, model.SupplierApplicationStatusDeactivated}, nil
	}
	statusParts := strings.Split(statusRaw, ",")
	statuses := make([]int, 0, len(statusParts))
	seen := make(map[int]struct{}, len(statusParts))
	for _, part := range statusParts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		status, err := strconv.Atoi(part)
		if err != nil {
			return nil, err
		}
		if status < model.SupplierApplicationStatusPending || status > model.SupplierApplicationStatusDeactivated {
			return nil, errors.New("invalid status")
		}
		if _, ok := seen[status]; ok {
			continue
		}
		seen[status] = struct{}{}
		statuses = append(statuses, status)
	}
	if len(statuses) == 0 {
		return nil, errors.New("empty status")
	}
	return statuses, nil
}

// trimAndFilterStrings 清理字符串数组中的空白项。
func trimAndFilterStrings(values []string) []string {
	cleaned := make([]string, 0, len(values))
	for _, item := range values {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		cleaned = append(cleaned, trimmed)
	}
	return cleaned
}

// marshalStringArray 将字符串数组编码为 JSON 字符串。
func marshalStringArray(values []string) (string, error) {
	bytes, err := common.Marshal(trimAndFilterStrings(values))
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// unmarshalStringArray 将 JSON 字符串解码为字符串数组。
func unmarshalStringArray(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{}
	}
	var items []string
	if err := common.UnmarshalJsonStr(raw, &items); err != nil {
		return []string{}
	}
	return trimAndFilterStrings(items)
}

// buildSupplierCapabilityModel 将请求体转换为模型对象。
func buildSupplierCapabilityModel(req *SupplierCapabilityUpsertRequest) (*model.SupplierCapability, error) {
	coreServiceTypes, err := marshalStringArray(req.CoreServiceTypes)
	if err != nil {
		return nil, err
	}
	supportedModels, err := marshalStringArray(req.SupportedModels)
	if err != nil {
		return nil, err
	}
	supportedAPIEndpoints, err := marshalStringArray(req.SupportedAPIEndpoints)
	if err != nil {
		return nil, err
	}
	supportedParams, err := marshalStringArray(req.SupportedParams)
	if err != nil {
		return nil, err
	}
	multimodalTypes, err := marshalStringArray(req.MultimodalTypes)
	if err != nil {
		return nil, err
	}
	pricingModes, err := marshalStringArray(req.PricingModes)
	if err != nil {
		return nil, err
	}
	apiBaseURLs, err := marshalStringArray(req.APIBaseURLs)
	if err != nil {
		return nil, err
	}
	return &model.SupplierCapability{
		CoreServiceTypes:          coreServiceTypes,
		SupportedModels:           supportedModels,
		SupportedModelNotes:       strings.TrimSpace(req.SupportedModelNotes),
		SupportedAPIEndpoints:     supportedAPIEndpoints,
		SupportedAPIEndpointExtra: strings.TrimSpace(req.SupportedAPIEndpointExtra),
		SupportedParams:           supportedParams,
		SupportedParamsExtra:      strings.TrimSpace(req.SupportedParamsExtra),
		StreamingSupported:        req.StreamingSupported,
		StreamingNotes:            strings.TrimSpace(req.StreamingNotes),
		StructuredOutputSupported: req.StructuredOutputSupported,
		StructuredOutputNotes:     strings.TrimSpace(req.StructuredOutputNotes),
		MultimodalTypes:           multimodalTypes,
		MultimodalExtra:           strings.TrimSpace(req.MultimodalExtra),
		PricingModes:              pricingModes,
		ReferenceInputPrice:       strings.TrimSpace(req.ReferenceInputPrice),
		ReferenceOutputPrice:      strings.TrimSpace(req.ReferenceOutputPrice),
		FailureBillingMode:        strings.TrimSpace(req.FailureBillingMode),
		FailureBillingNotes:       strings.TrimSpace(req.FailureBillingNotes),
		APIBaseURLs:               apiBaseURLs,
		OpenAICompatible:          req.OpenAICompatible,
		TruthCommitmentConfirmed:  req.TruthCommitmentConfirmed,
	}, nil
}

// buildSupplierCapabilityResponse 将模型对象转换为接口响应对象。
func buildSupplierCapabilityResponse(capability *model.SupplierCapability) gin.H {
	if capability == nil {
		return nil
	}
	return gin.H{
		"id":                           capability.ID,
		"supplier_application_id":      capability.SupplierApplicationID,
		"core_service_types":           unmarshalStringArray(capability.CoreServiceTypes),
		"supported_models":             unmarshalStringArray(capability.SupportedModels),
		"supported_model_notes":        capability.SupportedModelNotes,
		"supported_api_endpoints":      unmarshalStringArray(capability.SupportedAPIEndpoints),
		"supported_api_endpoint_extra": capability.SupportedAPIEndpointExtra,
		"supported_params":             unmarshalStringArray(capability.SupportedParams),
		"supported_params_extra":       capability.SupportedParamsExtra,
		"streaming_supported":          capability.StreamingSupported,
		"streaming_notes":              capability.StreamingNotes,
		"structured_output_supported":  capability.StructuredOutputSupported,
		"structured_output_notes":      capability.StructuredOutputNotes,
		"multimodal_types":             unmarshalStringArray(capability.MultimodalTypes),
		"multimodal_extra":             capability.MultimodalExtra,
		"pricing_modes":                unmarshalStringArray(capability.PricingModes),
		"reference_input_price":        capability.ReferenceInputPrice,
		"reference_output_price":       capability.ReferenceOutputPrice,
		"failure_billing_mode":         capability.FailureBillingMode,
		"failure_billing_notes":        capability.FailureBillingNotes,
		"api_base_urls":                unmarshalStringArray(capability.APIBaseURLs),
		"openai_compatible":            capability.OpenAICompatible,
		"truth_commitment_confirmed":   capability.TruthCommitmentConfirmed,
		"created_at":                   capability.CreatedAt,
		"updated_at":                   capability.UpdatedAt,
	}
}

// attachSupplierCapability 为供应商申请对象补充技术能力档案。
func attachSupplierCapability(app *model.SupplierApplication) error {
	if app == nil {
		return nil
	}
	capability, err := model.GetSupplierCapabilityByApplicationID(app.ID)
	if err != nil {
		if model.IsSupplierCapabilityNotFound(err) {
			app.SupplierCapability = nil
			return nil
		}
		return err
	}
	app.SupplierCapability = capability
	return nil
}

// SubmitSupplierApplication godoc
// @Summary 提交供应商入驻申请
// @Description 普通用户提交供应商申请，提交后生成管理员待审核站内消息
// @Tags Supplier
// @Accept json
// @Produce json
// @Security CookieAuth
// @Security ApiUserID
// @Param request body SupplierApplicationSubmitRequest true "申请信息"
// @Success 200 {object} map[string]interface{} "success + data{id,status}"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Router /user/supplier/application [post]
func SubmitSupplierApplication(c *gin.Context) {
	var req SupplierApplicationSubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的参数"})
		return
	}
	req.CompanyName = strings.TrimSpace(req.CompanyName)
	req.CreditCode = strings.TrimSpace(req.CreditCode)
	req.BusinessLicenseURL = strings.TrimSpace(req.BusinessLicenseURL)
	req.BusinessLicenseFile = strings.TrimSpace(req.BusinessLicenseFile)
	req.CompanyLogoURL = strings.TrimSpace(req.CompanyLogoURL)
	req.SupplierType = strings.TrimSpace(req.SupplierType)
	req.LegalRepresentative = strings.TrimSpace(req.LegalRepresentative)
	req.CompanySize = strings.TrimSpace(req.CompanySize)
	req.ContactName = strings.TrimSpace(req.ContactName)
	req.ContactMobile = strings.TrimSpace(req.ContactMobile)
	req.ContactWechat = strings.TrimSpace(req.ContactWechat)
	if missingFieldMessage := getSupplierApplicationMissingFieldMessage(req); missingFieldMessage != "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": missingFieldMessage})
		return
	}
	if len(req.CreditCode) != 18 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "统一社会信用代码需为18位"})
		return
	}

	isAdminOrAbove := c.GetInt("role") >= common.RoleAdminUser
	applicantUserID := c.GetInt("id")
	if isAdminOrAbove {
		if req.SupplierType == "" {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "管理员代添加供应商时必须选择供应商类型"})
			return
		}
		if !isValidSupplierType(req.SupplierType) {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的供应商类型"})
			return
		}
		if req.ApplicantUserID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "管理员代添加供应商时必须提供有效的applicant_user_id"})
			return
		}
		if _, err := model.GetUserById(req.ApplicantUserID, false); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "指定的关联用户不存在"})
			return
		}
		applicantUserID = req.ApplicantUserID
	}
	app := &model.SupplierApplication{
		ApplicantUserID:     applicantUserID,
		CompanyName:         req.CompanyName,
		CreditCode:          req.CreditCode,
		BusinessLicenseURL:  req.BusinessLicenseURL,
		BusinessLicenseFile: req.BusinessLicenseFile,
		CompanyLogoURL:      req.CompanyLogoURL,
		SupplierType:        req.SupplierType,
		LegalRepresentative: req.LegalRepresentative,
		CompanySize:         req.CompanySize,
		ContactName:         req.ContactName,
		ContactMobile:       req.ContactMobile,
		ContactWechat:       req.ContactWechat,
	}
	var err error
	if isAdminOrAbove {
		err = model.CreateSupplierApplicationAutoApproved(app, c.GetInt("id"))
	} else {
		app.Status = model.SupplierApplicationStatusPending
		err = model.CreateSupplierApplication(app)
	}
	if err != nil {
		if model.IsSupplierCreditCodeDuplicateError(err) {
			common.ApiErrorMsg(c, "统一社会信用代码已存在，请核对后重试")
			return
		}
		common.ApiError(c, err)
		return
	}
	if !isAdminOrAbove {
		_ = model.CreateSupplierApplicationAudit(&model.SupplierApplicationAudit{
			ApplicationID:  app.ID,
			OperatorUserID: app.ApplicantUserID,
			Action:         model.SupplierApplicationAuditActionSubmit,
			FromStatus:     model.SupplierApplicationStatusPending,
			ToStatus:       model.SupplierApplicationStatusPending,
			Reason:         "",
		})
		_ = service.PublishUserMessage(&model.UserMessage{
			ReceiverUserID:  0,
			ReceiverMinRole: common.RoleAdminUser,
			Type:            model.UserMessageTypeSupplierSubmitted,
			Title:           "供应商入驻待审核",
			Content:         fmt.Sprintf("收到新的供应商申请：%s（统一社会信用代码：%s）", app.CompanyName, app.CreditCode),
			BizType:         model.UserMessageBizTypeSupplierApplication,
			BizID:           app.ID,
		})
	}
	common.ApiSuccess(c, gin.H{
		"id":     app.ID,
		"status": app.Status,
	})
}

// GetMySupplierApplication godoc
// @Summary 查询当前用户供应商申请
// @Tags Supplier
// @Produce json
// @Security CookieAuth
// @Security ApiUserID
// @Success 200 {object} map[string]interface{} "success + data{申请对象或null}"
// @Router /user/supplier/application/self [get]
func GetMySupplierApplication(c *gin.Context) {
	app, err := model.GetMySupplierApplication(c.GetInt("id"))
	if err != nil {
		if model.IsSupplierApplicationNotFound(err) {
			common.ApiSuccess(c, nil)
			return
		}
		common.ApiError(c, err)
		return
	}
	if err = attachSupplierCapability(app); err != nil {
		common.ApiError(c, err)
		return
	}
	response := gin.H{
		"id":                    app.ID,
		"applicant_user_id":     app.ApplicantUserID,
		"company_name":          app.CompanyName,
		"credit_code":           app.CreditCode,
		"business_license_url":  app.BusinessLicenseURL,
		"business_license_file": app.BusinessLicenseFile,
		"company_logo_url":      app.CompanyLogoURL,
		"supplier_type":         app.SupplierType,
		"legal_representative":  app.LegalRepresentative,
		"company_size":          app.CompanySize,
		"contact_name":          app.ContactName,
		"contact_mobile":        app.ContactMobile,
		"contact_wechat":        app.ContactWechat,
		"supplier_alias":        app.SupplierAlias,
		"status":                app.Status,
		"review_reason":         app.ReviewReason,
		"reviewed_by":           app.ReviewedBy,
		"reviewed_at":           app.ReviewedAt,
		"created_at":            app.CreatedAt,
		"updated_at":            app.UpdatedAt,
		"supplier_capability":   buildSupplierCapabilityResponse(app.SupplierCapability),
	}
	common.ApiSuccess(c, response)
}

// GetSupplierCapability godoc
// @Summary 查询供应商技术能力档案
// @Description 管理员可查任意申请；普通用户仅可查询自己的申请
// @Tags Supplier
// @Produce json
// @Security CookieAuth
// @Security ApiUserID
// @Param id path int true "供应商申请ID"
// @Success 200 {object} map[string]interface{} "success + data{供应商技术能力档案}"
// @Router /user/supplier/application/{id}/capability [get]
func GetSupplierCapability(c *gin.Context) {
	applicationID, err := strconv.Atoi(c.Param("id"))
	if err != nil || applicationID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的供应商申请ID"})
		return
	}
	app, err := model.GetSupplierByID(applicationID)
	if err != nil {
		if model.IsSupplierApplicationNotFound(err) {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "未找到供应商申请"})
			return
		}
		common.ApiError(c, err)
		return
	}
	if c.GetInt("role") < common.RoleAdminUser && app.ApplicantUserID != c.GetInt("id") {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "无权查看该供应商技术能力档案"})
		return
	}
	capability, err := model.GetSupplierCapabilityByApplicationID(applicationID)
	if err != nil {
		if model.IsSupplierCapabilityNotFound(err) {
			common.ApiSuccess(c, nil)
			return
		}
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, buildSupplierCapabilityResponse(capability))
}

// UpsertSupplierCapability godoc
// @Summary 保存供应商技术能力档案
// @Description 管理员可保存任意申请；普通用户仅可保存自己的申请
// @Tags Supplier
// @Accept json
// @Produce json
// @Security CookieAuth
// @Security ApiUserID
// @Param id path int true "供应商申请ID"
// @Param request body SupplierCapabilityUpsertRequest true "供应商技术能力档案"
// @Success 200 {object} map[string]interface{} "success + data{供应商技术能力档案}"
// @Router /user/supplier/application/{id}/capability [put]
func UpsertSupplierCapability(c *gin.Context) {
	applicationID, err := strconv.Atoi(c.Param("id"))
	if err != nil || applicationID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的供应商申请ID"})
		return
	}
	app, err := model.GetSupplierByID(applicationID)
	if err != nil {
		if model.IsSupplierApplicationNotFound(err) {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "未找到供应商申请"})
			return
		}
		common.ApiError(c, err)
		return
	}
	if c.GetInt("role") < common.RoleAdminUser && app.ApplicantUserID != c.GetInt("id") {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "无权修改该供应商技术能力档案"})
		return
	}
	var req SupplierCapabilityUpsertRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的参数"})
		return
	}
	capabilityModel, err := buildSupplierCapabilityModel(&req)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	updated, err := model.UpsertSupplierCapabilityByApplicationID(applicationID, capabilityModel)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, buildSupplierCapabilityResponse(updated))
}

// UpdateMySupplierApplication godoc
// @Summary 修改当前用户供应商申请并重新提交
// @Description 当前申请只要未审核通过都可修改，修改后状态重置为待审核(0)
// @Tags Supplier
// @Accept json
// @Produce json
// @Security CookieAuth
// @Security ApiUserID
// @Param request body SupplierApplicationUpdateRequest true "申请信息（含id）"
// @Success 200 {object} map[string]interface{} "success + data{id,status}"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Router /user/supplier/application/self [put]
func UpdateMySupplierApplication(c *gin.Context) {
	var req SupplierApplicationUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的参数"})
		return
	}
	if req.ID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "修改申请时必须提供有效的id"})
		return
	}
	req.CompanyName = strings.TrimSpace(req.CompanyName)
	req.CreditCode = strings.TrimSpace(req.CreditCode)
	req.BusinessLicenseURL = strings.TrimSpace(req.BusinessLicenseURL)
	req.BusinessLicenseFile = strings.TrimSpace(req.BusinessLicenseFile)
	req.CompanyLogoURL = strings.TrimSpace(req.CompanyLogoURL)
	req.SupplierType = strings.TrimSpace(req.SupplierType)
	req.LegalRepresentative = strings.TrimSpace(req.LegalRepresentative)
	req.CompanySize = strings.TrimSpace(req.CompanySize)
	req.ContactName = strings.TrimSpace(req.ContactName)
	req.ContactMobile = strings.TrimSpace(req.ContactMobile)
	req.ContactWechat = strings.TrimSpace(req.ContactWechat)
	if req.CompanyName == "" || req.CreditCode == "" || req.BusinessLicenseURL == "" || req.CompanyLogoURL == "" ||
		req.LegalRepresentative == "" || req.ContactName == "" || req.ContactMobile == "" || req.ContactWechat == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请填写完整的必填字段"})
		return
	}
	if len(req.CreditCode) != 18 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "统一社会信用代码需为18位"})
		return
	}
	app, err := model.UpdateMySupplierApplication(c.GetInt("id"), req.ID, &model.SupplierApplication{
		CompanyName:         req.CompanyName,
		CreditCode:          req.CreditCode,
		BusinessLicenseURL:  req.BusinessLicenseURL,
		BusinessLicenseFile: req.BusinessLicenseFile,
		CompanyLogoURL:      req.CompanyLogoURL,
		SupplierType:        req.SupplierType,
		LegalRepresentative: req.LegalRepresentative,
		CompanySize:         req.CompanySize,
		ContactName:         req.ContactName,
		ContactMobile:       req.ContactMobile,
		ContactWechat:       req.ContactWechat,
	})
	if err != nil {
		if model.IsSupplierCreditCodeDuplicateError(err) {
			common.ApiErrorMsg(c, "统一社会信用代码已存在，请核对后重试")
			return
		}
		if errors.Is(err, model.ErrSupplierApplicationStatusNotEditable) {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "当前申请状态不可修改"})
			return
		}
		if model.IsSupplierApplicationNotFound(err) {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "未找到可修改的供应商申请"})
			return
		}
		common.ApiError(c, err)
		return
	}
	_ = service.PublishUserMessage(&model.UserMessage{
		ReceiverUserID:  0,
		ReceiverMinRole: common.RoleAdminUser,
		Type:            model.UserMessageTypeSupplierSubmitted,
		Title:           "供应商入驻待审核",
		Content:         fmt.Sprintf("供应商申请已更新并重新提交：%s（统一社会信用代码：%s）", app.CompanyName, app.CreditCode),
		BizType:         model.UserMessageBizTypeSupplierApplication,
		BizID:           app.ID,
	})
	common.ApiSuccess(c, gin.H{
		"id":     app.ID,
		"status": app.Status,
	})
}

// AdminListSupplierApplications godoc
// @Summary 管理员分页查询供应商申请
// @Tags SupplierAdmin
// @Produce json
// @Security CookieAuth
// @Security ApiUserID
// @Param p query int false "页码"
// @Param page_size query int false "每页数量"
// @Param status query int false "状态：0待审核 1审核通过 2审核驳回"
// @Success 200 {object} map[string]interface{} "分页结果"
// @Router /user/supplier/application [get]
func AdminListSupplierApplications(c *gin.Context) {
	status, err := readOptionalStatusQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的status参数"})
		return
	}
	pageInfo := common.GetPageQuery(c)
	items, total, err := model.ListSupplierApplications(status, pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

// AdminListSuppliers godoc
// @Summary 管理员分页查询供应商列表
// @Description 支持按供应商名称模糊查询，返回分页数据
// @Tags SupplierAdmin
// @Produce json
// @Security CookieAuth
// @Security ApiUserID
// @Param p query int false "页码"
// @Param page_size query int false "每页数量"
// @Param company_name query string false "供应商名称（模糊）"
// @Param status query string false "状态筛选，支持逗号分隔（如1,3）；默认查询1和3"
// @Success 200 {object} map[string]interface{} "分页结果"
// @Router /user/supplier/list [get]
func AdminListSuppliers(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	companyName := strings.TrimSpace(c.Query("company_name"))
	statuses, err := readSupplierStatusListQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的status参数"})
		return
	}
	items, total, err := model.ListSuppliersByCompanyName(companyName, statuses, pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

// AdminGetSupplierDetail godoc
// @Summary 管理员查询供应商详情
// @Description 根据供应商ID查询供应商详情，返回申请人用户名 applicant_username
// @Tags SupplierAdmin
// @Produce json
// @Security CookieAuth
// @Security ApiUserID
// @Param id path int true "供应商ID"
// @Success 200 {object} map[string]interface{} "供应商详情"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Router /user/supplier/{id} [get]
func AdminGetSupplierDetail(c *gin.Context) {
	supplierID, err := strconv.Atoi(c.Param("id"))
	if err != nil || supplierID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的供应商ID"})
		return
	}
	item, err := model.GetSupplierByID(supplierID)
	if err != nil {
		if model.IsSupplierApplicationNotFound(err) {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "未找到供应商信息"})
			return
		}
		common.ApiError(c, err)
		return
	}
	if err = attachSupplierCapability(item); err != nil {
		common.ApiError(c, err)
		return
	}
	response := gin.H{
		"id":                    item.ID,
		"applicant_user_id":     item.ApplicantUserID,
		"applicant_username":    item.ApplicantUsername,
		"company_name":          item.CompanyName,
		"credit_code":           item.CreditCode,
		"business_license_url":  item.BusinessLicenseURL,
		"business_license_file": item.BusinessLicenseFile,
		"company_logo_url":      item.CompanyLogoURL,
		"supplier_type":         item.SupplierType,
		"legal_representative":  item.LegalRepresentative,
		"company_size":          item.CompanySize,
		"contact_name":          item.ContactName,
		"contact_mobile":        item.ContactMobile,
		"contact_wechat":        item.ContactWechat,
		"supplier_alias":        item.SupplierAlias,
		"status":                item.Status,
		"review_reason":         item.ReviewReason,
		"reviewed_by":           item.ReviewedBy,
		"reviewed_at":           item.ReviewedAt,
		"created_at":            item.CreatedAt,
		"updated_at":            item.UpdatedAt,
		"supplier_capability":   buildSupplierCapabilityResponse(item.SupplierCapability),
	}
	common.ApiSuccess(c, response)
}

// AdminUpdateSupplierApplication godoc
// @Summary 管理员修改供应商申请资料
// @Description 管理员可修改任意供应商申请资料；审核通过(status=1)状态也允许修改，且修改后保持原状态
// @Tags SupplierAdmin
// @Accept json
// @Produce json
// @Security CookieAuth
// @Security ApiUserID
// @Param id path int true "供应商申请ID"
// @Param request body SupplierApplicationSubmitRequest true "申请信息"
// @Success 200 {object} map[string]interface{} "success + data{id,status}"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Router /user/supplier/application/{id} [put]
func AdminUpdateSupplierApplication(c *gin.Context) {
	applicationID, err := strconv.Atoi(c.Param("id"))
	if err != nil || applicationID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的供应商申请ID"})
		return
	}
	var req SupplierApplicationSubmitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的参数"})
		return
	}
	req.CompanyName = strings.TrimSpace(req.CompanyName)
	req.CreditCode = strings.TrimSpace(req.CreditCode)
	req.BusinessLicenseURL = strings.TrimSpace(req.BusinessLicenseURL)
	req.BusinessLicenseFile = strings.TrimSpace(req.BusinessLicenseFile)
	req.CompanyLogoURL = strings.TrimSpace(req.CompanyLogoURL)
	req.SupplierType = strings.TrimSpace(req.SupplierType)
	req.LegalRepresentative = strings.TrimSpace(req.LegalRepresentative)
	req.CompanySize = strings.TrimSpace(req.CompanySize)
	req.ContactName = strings.TrimSpace(req.ContactName)
	req.ContactMobile = strings.TrimSpace(req.ContactMobile)
	req.ContactWechat = strings.TrimSpace(req.ContactWechat)
	req.SupplierAlias = strings.TrimSpace(req.SupplierAlias)
	if req.CompanyName == "" || req.CreditCode == "" || req.BusinessLicenseURL == "" || req.CompanyLogoURL == "" ||
		req.LegalRepresentative == "" || req.ContactName == "" || req.ContactMobile == "" || req.ContactWechat == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请填写完整的必填字段"})
		return
	}
	if req.SupplierType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请填写供应商类型"})
		return
	}
	if !isValidSupplierType(req.SupplierType) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的供应商类型"})
		return
	}
	if len(req.CreditCode) != 18 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "统一社会信用代码需为18位"})
		return
	}
	aliasPtr := req.SupplierAlias
	app, err := model.AdminUpdateSupplierApplication(applicationID, &model.SupplierApplication{
		CompanyName:         req.CompanyName,
		CreditCode:          req.CreditCode,
		BusinessLicenseURL:  req.BusinessLicenseURL,
		BusinessLicenseFile: req.BusinessLicenseFile,
		CompanyLogoURL:      req.CompanyLogoURL,
		SupplierType:        req.SupplierType,
		LegalRepresentative: req.LegalRepresentative,
		CompanySize:         req.CompanySize,
		ContactName:         req.ContactName,
		ContactMobile:       req.ContactMobile,
		ContactWechat:       req.ContactWechat,
		SupplierAlias:       &aliasPtr,
	})
	if err != nil {
		if model.IsSupplierCreditCodeDuplicateError(err) {
			common.ApiErrorMsg(c, "统一社会信用代码已存在，请核对后重试")
			return
		}
		if model.IsSupplierApplicationNotFound(err) {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "未找到可修改的供应商申请"})
			return
		}
		if model.IsSupplierAliasDuplicateError(err) {
			common.ApiErrorMsg(c, "供应商别名已存在，请更换后重试")
			return
		}
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"id":     app.ID,
		"status": app.Status,
	})
}

// AdminReviewSupplierApplication godoc
// @Summary 管理员审核供应商申请
// @Description 任一管理员可审核一次，仅待审核状态允许处理
// @Tags SupplierAdmin
// @Accept json
// @Produce json
// @Security CookieAuth
// @Security ApiUserID
// @Param id path int true "申请ID"
// @Param request body SupplierApplicationReviewRequest true "审核信息"
// @Success 200 {object} map[string]interface{} "success + data{id,status}"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Router /user/supplier/application/{id}/review [post]
func AdminReviewSupplierApplication(c *gin.Context) {
	applicationID, err := strconv.Atoi(c.Param("id"))
	if err != nil || applicationID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的申请ID"})
		return
	}
	var req SupplierApplicationReviewRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的参数"})
		return
	}
	if req.Status != model.SupplierApplicationStatusApproved && req.Status != model.SupplierApplicationStatusRejected {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "审核状态仅支持通过或驳回"})
		return
	}
	req.Reason = strings.TrimSpace(req.Reason)
	req.SupplierType = strings.TrimSpace(req.SupplierType)
	if req.Status == model.SupplierApplicationStatusRejected && req.Reason == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "驳回时请填写原因"})
		return
	}
	if req.Status == model.SupplierApplicationStatusApproved {
		capability, capabilityErr := model.GetSupplierCapabilityByApplicationID(applicationID)
		if capabilityErr != nil {
			if model.IsSupplierCapabilityNotFound(capabilityErr) {
				c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "审批通过前请先完善供应商技术能力信息"})
				return
			}
			common.ApiError(c, capabilityErr)
			return
		}
		if !model.IsSupplierCapabilityComplete(capability) {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "审批通过前请先完善供应商技术能力必填字段"})
			return
		}
	}

	req.SupplierAlias = strings.TrimSpace(req.SupplierAlias)
	if req.Status == model.SupplierApplicationStatusApproved {
		if req.SupplierType == "" {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "审批通过时必须选择供应商类型"})
			return
		}
		if !isValidSupplierType(req.SupplierType) {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的供应商类型"})
			return
		}
	}
	app, err := model.ReviewSupplierApplication(applicationID, c.GetInt("id"), req.Status, req.Reason, req.SupplierAlias, req.SupplierType)
	if err != nil {
		if errors.Is(err, model.ErrSupplierApplicationAlreadyReviewed) {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "该申请已被其他管理员处理"})
			return
		}
		if model.IsSupplierAliasDuplicateError(err) {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "供应商别名已存在，请更换后重试"})
			return
		}
		common.ApiError(c, err)
		return
	}

	msgType := model.UserMessageTypeSupplierApproved
	msgTitle := "供应商入驻审核通过"
	msgContent := fmt.Sprintf("你的供应商申请“%s”已审核通过。", app.CompanyName)
	if app.Status == model.SupplierApplicationStatusRejected {
		msgType = model.UserMessageTypeSupplierRejected
		msgTitle = "供应商入驻审核驳回"
		msgContent = fmt.Sprintf("你的供应商申请“%s”已驳回，原因：%s", app.CompanyName, req.Reason)
	}
	_ = service.PublishUserMessage(&model.UserMessage{
		ReceiverUserID:  app.ApplicantUserID,
		ReceiverMinRole: 0,
		Type:            msgType,
		Title:           msgTitle,
		Content:         msgContent,
		BizType:         model.UserMessageBizTypeSupplierApplication,
		BizID:           app.ID,
	})
	common.ApiSuccess(c, gin.H{
		"id":     app.ID,
		"status": app.Status,
	})
}

// ListMyMessages godoc
// @Summary 查询当前用户站内消息
// @Tags Message
// @Produce json
// @Security CookieAuth
// @Security ApiUserID
// @Param p query int false "页码"
// @Param page_size query int false "每页数量"
// @Param title query string false "标题模糊查询"
// @Param read_status query string false "读取状态：all/read/unread，默认all"
// @Success 200 {object} map[string]interface{} "分页结果"
// @Router /user/messages/self [get]
func ListMyMessages(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	userID := c.GetInt("id")
	role := c.GetInt("role")
	titleKeyword := strings.TrimSpace(c.Query("title"))
	readStatus := strings.TrimSpace(c.Query("read_status"))
	if readStatus == "" {
		readStatus = "all"
	}
	if readStatus != "all" && readStatus != "read" && readStatus != "unread" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的read_status参数"})
		return
	}
	items, total, err := model.ListUserMessagesForUser(userID, role, pageInfo, titleKeyword, readStatus)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

// MarkMyMessageRead godoc
// @Summary 标记当前用户消息为已读
// @Tags Message
// @Produce json
// @Security CookieAuth
// @Security ApiUserID
// @Param id path int true "消息ID"
// @Success 200 {object} map[string]interface{} "success + data{updated}"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Router /user/messages/{id}/read [post]
func MarkMyMessageRead(c *gin.Context) {
	messageID, err := strconv.Atoi(c.Param("id"))
	if err != nil || messageID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的消息ID"})
		return
	}
	ok, err := model.MarkUserMessageAsRead(messageID, c.GetInt("id"), c.GetInt("role"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"updated": ok})
}

// MarkAllMyMessagesRead godoc
// @Summary 标记当前用户全部站内消息为已读
// @Tags Message
// @Produce json
// @Security CookieAuth
// @Security ApiUserID
// @Success 200 {object} map[string]interface{} "success + data{updated_count}"
// @Router /user/messages/read_all [post]
func MarkAllMyMessagesRead(c *gin.Context) {
	updatedCount, err := model.MarkAllUserMessagesAsRead(c.GetInt("id"), c.GetInt("role"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"updated_count": updatedCount})
}

// AdminPublishUserMessage godoc
// @Summary 管理员发布站内消息
// @Description 支持按指定用户或按最小角色发布站内消息，至少设置 receiver_user_id 或 receiver_min_role 之一
// @Tags MessageAdmin
// @Accept json
// @Produce json
// @Security CookieAuth
// @Security ApiUserID
// @Param request body PublishUserMessageRequest true "消息内容"
// @Success 200 {object} map[string]interface{} "success + data{published:true}"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Router /user/messages/publish [post]
func AdminPublishUserMessage(c *gin.Context) {
	var req PublishUserMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的参数"})
		return
	}
	if req.ReceiverUserID <= 0 && req.ReceiverMinRole <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请至少指定接收用户或角色门槛"})
		return
	}
	msg := &model.UserMessage{
		ReceiverUserID:  req.ReceiverUserID,
		ReceiverMinRole: req.ReceiverMinRole,
		Type:            req.Type,
		Title:           req.Title,
		Content:         req.Content,
		BizType:         req.BizType,
		BizID:           req.BizID,
	}
	if err := service.PublishUserMessage(msg); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	common.ApiSuccess(c, gin.H{"published": true})
}

// GetMyUnreadMessageCount godoc
// @Summary 获取当前用户未读站内消息数量
// @Tags Message
// @Produce json
// @Security CookieAuth
// @Security ApiUserID
// @Success 200 {object} map[string]interface{} "success + data{unread_count}"
// @Router /user/messages/unread_count [get]
func GetMyUnreadMessageCount(c *gin.Context) {
	total, err := model.CountUnreadUserMessages(c.GetInt("id"), c.GetInt("role"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"unread_count": total})
}

// DeactivateMySupplierApplication godoc
// @Summary 当前供应商注销
// @Description 仅审核通过状态可注销；注销后清空用户表 supplier_id 并将申请状态置为已注销
// @Tags Supplier
// @Accept json
// @Produce json
// @Security CookieAuth
// @Security ApiUserID
// @Param request body SupplierDeactivateRequest false "注销说明"
// @Success 200 {object} map[string]interface{} "success + data{id,status}"
// @Failure 400 {object} map[string]interface{} "参数错误"
// @Router /user/supplier/application/deactivate [post]
func DeactivateMySupplierApplication(c *gin.Context) {
	var req SupplierDeactivateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的参数"})
		return
	}
	if req.SupplierID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "注销供应商时必须提供有效的supplier_id"})
		return
	}
	reason := strings.TrimSpace(req.Reason)
	app, err := model.DeactivateSupplierApplication(c.GetInt("id"), c.GetInt("role"), req.SupplierID, reason)
	if err != nil {
		if errors.Is(err, model.ErrSupplierApplicationStatusNotApproved) {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "当前供应商状态不支持注销"})
			return
		}
		if model.IsSupplierApplicationNotFound(err) {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "未找到可注销的供应商申请"})
			return
		}
		common.ApiError(c, err)
		return
	}
	_ = service.PublishUserMessage(&model.UserMessage{
		ReceiverUserID:  0,
		ReceiverMinRole: common.RoleAdminUser,
		Type:            model.UserMessageTypeSupplierRejected,
		Title:           "供应商已注销",
		Content:         fmt.Sprintf("供应商“%s”已注销。", app.CompanyName),
		BizType:         model.UserMessageBizTypeSupplierApplication,
		BizID:           app.ID,
	})
	common.ApiSuccess(c, gin.H{
		"id":     app.ID,
		"status": app.Status,
	})
}

// CreateMySupplierChannel godoc
// @Summary 当前供应商新增渠道
// @Description 仅审核通过的供应商可新增，自动写入 owner_user_id 与 supplier_application_id
// @Tags Supplier
// @Accept json
// @Produce json
// @Security CookieAuth
// @Security ApiUserID
// @Param request body AddChannelRequest true "渠道创建参数"
// @Success 200 {object} map[string]interface{} "创建结果"
// @Router /user/supplier/channels [post]
func CreateMySupplierChannel(c *gin.Context) {
	supplierApp, ok := requireApprovedSupplierApplication(c)
	if !ok {
		return
	}
	addChannelRequest := AddChannelRequest{}
	if err := c.ShouldBindJSON(&addChannelRequest); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := validateChannel(addChannelRequest.Channel, true); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	addChannelRequest.Channel.CreatedTime = common.GetTimestamp()
	keys := make([]string, 0)
	switch addChannelRequest.Mode {
	case "multi_to_single":
		addChannelRequest.Channel.ChannelInfo.IsMultiKey = true
		addChannelRequest.Channel.ChannelInfo.MultiKeyMode = addChannelRequest.MultiKeyMode
		if addChannelRequest.Channel.Type == constant.ChannelTypeVertexAi && addChannelRequest.Channel.GetOtherSettings().VertexKeyType != dto.VertexKeyTypeAPIKey {
			array, err := getVertexArrayKeys(addChannelRequest.Channel.Key)
			if err != nil {
				c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
				return
			}
			addChannelRequest.Channel.ChannelInfo.MultiKeySize = len(array)
			addChannelRequest.Channel.Key = strings.Join(array, "\n")
		} else {
			cleanKeys := make([]string, 0)
			for _, key := range strings.Split(addChannelRequest.Channel.Key, "\n") {
				if key == "" {
					continue
				}
				key = strings.TrimSpace(key)
				cleanKeys = append(cleanKeys, key)
			}
			addChannelRequest.Channel.ChannelInfo.MultiKeySize = len(cleanKeys)
			addChannelRequest.Channel.Key = strings.Join(cleanKeys, "\n")
		}
		keys = []string{addChannelRequest.Channel.Key}
	case "batch":
		if addChannelRequest.Channel.Type == constant.ChannelTypeVertexAi && addChannelRequest.Channel.GetOtherSettings().VertexKeyType != dto.VertexKeyTypeAPIKey {
			array, err := getVertexArrayKeys(addChannelRequest.Channel.Key)
			if err != nil {
				c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
				return
			}
			keys = array
		} else {
			keys = strings.Split(addChannelRequest.Channel.Key, "\n")
		}
	case "single":
		keys = []string{addChannelRequest.Channel.Key}
	default:
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "不支持的添加模式"})
		return
	}
	channels := make([]model.Channel, 0, len(keys))
	for _, key := range keys {
		if key == "" {
			continue
		}
		localChannel := addChannelRequest.Channel
		localChannel.Key = key
		localChannel.OwnerUserID = c.GetInt("id")
		localChannel.SupplierApplicationID = supplierApp.ID
		if addChannelRequest.BatchAddSetKeyPrefix2Name && len(keys) > 1 {
			keyPrefix := localChannel.Key
			if len(localChannel.Key) > 8 {
				keyPrefix = localChannel.Key[:8]
			}
			localChannel.Name = fmt.Sprintf("%s %s", localChannel.Name, keyPrefix)
		}
		channels = append(channels, *localChannel)
	}
	if err := model.BatchInsertChannels(channels); err != nil {
		common.ApiError(c, err)
		return
	}
	service.ResetProxyClientCache()
	common.ApiSuccess(c, gin.H{"created": len(channels)})
}

// ListMySupplierChannels godoc
// @Summary 查询当前供应商渠道列表
// @Description 供应商返回本人渠道；管理员返回所有供应商渠道
// @Tags Supplier
// @Produce json
// @Security CookieAuth
// @Security ApiUserID
// @Param p query int false "页码"
// @Param page_size query int false "每页数量"
// @Param channel_id query int false "渠道ID"
// @Param name query string false "渠道名称（模糊）"
// @Param key query string false "渠道密钥（精确或模糊）"
// @Param base_url query string false "API地址（模糊）"
// @Param model query string false "模型关键字（模糊）"
// @Param group query string false "分组"
// @Success 200 {object} map[string]interface{} "分页结果"
// @Router /user/supplier/channels [get]
func ListMySupplierChannels(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	channelID, parseErr := model.ParseSupplierChannelIDFilter(c.Query("channel_id"))
	if parseErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "渠道ID参数格式错误"})
		return
	}
	filter := model.SupplierChannelSearchFilter{
		ChannelID:    channelID,
		Name:         strings.TrimSpace(c.Query("name")),
		Key:          strings.TrimSpace(c.Query("key")),
		BaseURL:      strings.TrimSpace(c.Query("base_url")),
		ModelKeyword: strings.TrimSpace(c.Query("model")),
		Group:        strings.TrimSpace(c.Query("group")),
	}
	var (
		items []*model.Channel
		total int64
		err   error
	)
	if c.GetInt("role") >= common.RoleAdminUser {
		items, total, err = model.SearchSupplierChannels(nil, pageInfo.GetStartIdx(), pageInfo.GetPageSize(), filter)
	} else {
		ownerUserID := c.GetInt("id")
		items, total, err = model.SearchSupplierChannels(&ownerUserID, pageInfo.GetStartIdx(), pageInfo.GetPageSize(), filter)
	}
	if err != nil {
		common.ApiError(c, err)
		return
	}
	for _, item := range items {
		clearChannelInfo(item)
	}
	common.ApiSuccess(c, gin.H{
		"items":     items,
		"total":     total,
		"page":      pageInfo.GetPage(),
		"page_size": pageInfo.GetPageSize(),
	})
}

// CreateMySupplierModel godoc
// @Summary 当前供应商新增模型
// @Description 仅审核通过供应商可新增，自动写入 owner_user_id 与 supplier_application_id
// @Tags Supplier
// @Accept json
// @Produce json
// @Security CookieAuth
// @Security ApiUserID
// @Param request body model.Model true "模型创建参数"
// @Success 200 {object} map[string]interface{} "创建结果"
// @Router /user/supplier/models [post]
func CreateMySupplierModel(c *gin.Context) {
	supplierApp, ok := requireApprovedSupplierApplication(c)
	if !ok {
		return
	}
	var m model.Model
	if err := c.ShouldBindJSON(&m); err != nil {
		common.ApiError(c, err)
		return
	}
	if m.ModelName == "" {
		common.ApiErrorMsg(c, "模型名称不能为空")
		return
	}
	dup, err := model.IsModelNameDuplicated(0, m.ModelName)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if dup {
		common.ApiErrorMsg(c, "模型名称已存在")
		return
	}
	m.OwnerUserID = c.GetInt("id")
	m.SupplierApplicationID = supplierApp.ID
	if err := m.Insert(); err != nil {
		common.ApiError(c, err)
		return
	}
	model.RefreshPricing()
	common.ApiSuccess(c, &m)
}

// ListMySupplierModels godoc
// @Summary 查询当前供应商模型列表
// @Description 仅返回当前登录供应商创建的模型
// @Tags Supplier
// @Produce json
// @Security ApiKeyAuth
// @Security ApiUserID
// @Param p query int false "页码"
// @Param page_size query int false "每页数量"
// @Param model_name query string false "模型名称（模糊）"
// @Param model_type query string false "模型类型（映射 vendor，支持名称或ID）"
// @Success 200 {object} map[string]interface{} "分页结果"
// @Router /user/supplier/models [get]
func ListMySupplierModels(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	keyword := strings.TrimSpace(c.Query("model_name"))
	vendor := strings.TrimSpace(c.Query("model_type"))
	var (
		items []*model.Model
		total int64
		err   error
	)
	if c.GetInt("role") >= common.RoleAdminUser {
		items, total, err = model.SearchSupplierModels(nil, keyword, vendor, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	} else {
		ownerUserID := c.GetInt("id")
		items, total, err = model.SearchSupplierModels(&ownerUserID, keyword, vendor, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	}
	if err != nil {
		common.ApiError(c, err)
		return
	}
	enrichModels(items)
	common.ApiSuccess(c, gin.H{
		"items":     items,
		"total":     total,
		"page":      pageInfo.GetPage(),
		"page_size": pageInfo.GetPageSize(),
	})
}
