package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type supplierRevenuePushConfigRequest struct {
	Enabled              bool   `json:"enabled"`
	Mode                 string `json:"mode"`
	ScheduleType         string `json:"schedule_type"`
	Timezone             string `json:"timezone"`
	DailyTime            string `json:"daily_time"`
	HourlyMinute         int    `json:"hourly_minute"`
	Currency             string `json:"currency"`
	NegativePolicy       string `json:"negative_policy"`
	RetryCount           int    `json:"retry_count"`
	RetryIntervalSeconds int    `json:"retry_interval_seconds"`
	RetryBackoff         string `json:"retry_backoff"`
	TimeoutSeconds       int    `json:"timeout_seconds"`
	Environment          string `json:"environment"`
	Endpoint             string `json:"endpoint"`
	MockEndpoint         string `json:"mock_endpoint"`
	PrivateKey           string `json:"private_key"`
	ClearPrivateKey      bool   `json:"clear_private_key"`
	HTTPMethod           string `json:"http_method"`
	ContentType          string `json:"content_type"`
	HeadersJSON          string `json:"headers_json"`
	BodyTemplate         string `json:"body_template"`
	SuccessHTTPStatus    int    `json:"success_http_status"`
	SuccessCodePath      string `json:"success_code_path"`
	SuccessCodeValue     string `json:"success_code_value"`
	SuccessTypePath      string `json:"success_type_path"`
	SuccessTypeValue     string `json:"success_type_value"`
	SuccessAmountPath    string `json:"success_amount_path"`
	CallbackConfigJSON   string `json:"callback_config_json"`
}

type supplierRevenueResolveRequest struct {
	Action string `json:"action"`
}

type supplierRevenueManualPushRequest struct {
	Amount string `json:"amount"`
	Remark string `json:"remark"`
}

func supplierRevenueSupplierID(c *gin.Context) (int, bool) {
	supplierID, err := strconv.Atoi(c.Param("id"))
	if err != nil || supplierID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的供应商ID"})
		return 0, false
	}
	if _, err = model.GetSupplierByID(supplierID); err != nil {
		if model.IsSupplierApplicationNotFound(err) || errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "供应商不存在"})
		} else {
			common.ApiError(c, err)
		}
		return 0, false
	}
	return supplierID, true
}

func supplierRevenueConfigResponse(config *model.SupplierRevenuePushConfig) gin.H {
	hasPrivateKey := false
	if strings.TrimSpace(config.PrivateKeyPEM) != "" {
		_, _, err := service.ParseSupplierRevenuePrivateKey(config.PrivateKeyPEM)
		hasPrivateKey = err == nil
	}
	return gin.H{
		"config":                  config,
		"has_private_key":         hasPrivateKey,
		"mock_endpoint":           service.SupplierRevenueEoraptorMockEndpoint,
		"production_endpoint":     service.SupplierRevenueEoraptorProductionEndpoint,
		"private_key_fingerprint": config.PrivateKeyFingerprint,
	}
}

func GetSupplierRevenuePushConfig(c *gin.Context) {
	supplierID, ok := supplierRevenueSupplierID(c)
	if !ok {
		return
	}
	config, err := model.GetSupplierRevenuePushConfig(supplierID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		config = service.DefaultSupplierRevenuePushConfig(supplierID)
	} else if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, supplierRevenueConfigResponse(config))
}

func PutSupplierRevenuePushConfig(c *gin.Context) {
	supplierID, ok := supplierRevenueSupplierID(c)
	if !ok {
		return
	}
	var request supplierRevenuePushConfigRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的推送配置"})
		return
	}
	config, err := buildSupplierRevenueConfig(supplierID, c.GetInt("id"), &request)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	if err = service.ValidateSupplierRevenuePushConfig(config, false); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	if err = model.SaveSupplierRevenuePushConfig(config); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, supplierRevenueConfigResponse(config))
}

func TestSupplierRevenuePush(c *gin.Context) {
	supplierID, ok := supplierRevenueSupplierID(c)
	if !ok {
		return
	}
	var request supplierRevenuePushConfigRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的测试配置"})
		return
	}
	config, err := buildSupplierRevenueConfig(supplierID, c.GetInt("id"), &request)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	privateKey := strings.TrimSpace(request.PrivateKey)
	if config.Mode == model.SupplierRevenuePushModeEoraptor && privateKey == "" && config.PrivateKeyPEM == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "请先上传RSA私钥"})
		return
	}
	result, err := service.TestSupplierRevenuePush(c.Request.Context(), config, privateKey)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func RunSupplierRevenuePush(c *gin.Context) {
	supplierID, ok := supplierRevenueSupplierID(c)
	if !ok {
		return
	}
	delivery, err := service.RunSupplierRevenuePushForSupplier(c.Request.Context(), supplierID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	if delivery == nil {
		common.ApiSuccess(c, gin.H{"message": "未创建推送：账期可能尚未到执行时间，或已有发送中、重试中、状态未知的批次；如需发送指定金额可使用手动推送"})
		return
	}
	common.ApiSuccess(c, delivery)
}

func ManualSupplierRevenuePush(c *gin.Context) {
	supplierID, ok := supplierRevenueSupplierID(c)
	if !ok {
		return
	}
	var request supplierRevenueManualPushRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的手动推送参数"})
		return
	}
	delivery, err := service.ManualSupplierRevenuePush(c.Request.Context(), supplierID, request.Amount, request.Remark)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	common.ApiSuccess(c, delivery)
}

func ListSupplierRevenuePushDeliveries(c *gin.Context) {
	supplierID, ok := supplierRevenueSupplierID(c)
	if !ok {
		return
	}
	pageInfo := common.GetPageQuery(c)
	items, total, err := model.ListSupplierRevenueDeliveries(supplierID, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

func ListSupplierRevenuePushAttempts(c *gin.Context) {
	supplierID, ok := supplierRevenueSupplierID(c)
	if !ok {
		return
	}
	deliveryID, err := strconv.Atoi(c.Param("delivery_id"))
	if err != nil || deliveryID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的推送批次ID"})
		return
	}
	delivery, err := model.GetSupplierRevenueDelivery(deliveryID)
	if err != nil || delivery.SupplierID != supplierID {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "推送批次不存在"})
		return
	}
	attempts, err := model.ListSupplierRevenueAttempts(deliveryID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, attempts)
}

func ResolveSupplierRevenuePushDelivery(c *gin.Context) {
	supplierID, ok := supplierRevenueSupplierID(c)
	if !ok {
		return
	}
	deliveryID, err := strconv.Atoi(c.Param("delivery_id"))
	if err != nil || deliveryID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的推送批次ID"})
		return
	}
	delivery, err := model.GetSupplierRevenueDelivery(deliveryID)
	if err != nil || delivery.SupplierID != supplierID {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "推送批次不存在"})
		return
	}
	var request supplierRevenueResolveRequest
	if err = c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "无效的处理参数"})
		return
	}
	switch strings.TrimSpace(request.Action) {
	case "settled":
		err = model.ResolveUnknownSupplierRevenueDelivery(deliveryID, true)
	case "retry":
		err = model.ResolveUnknownSupplierRevenueDelivery(deliveryID, false)
		if err == nil {
			_, err = service.RunSupplierRevenuePushForSupplier(c.Request.Context(), supplierID)
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "action仅支持settled或retry"})
		return
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": err.Error()})
		return
	}
	common.ApiSuccess(c, gin.H{"delivery_id": deliveryID})
}

func buildSupplierRevenueConfig(supplierID, operatorID int, request *supplierRevenuePushConfigRequest) (*model.SupplierRevenuePushConfig, error) {
	config, err := model.GetSupplierRevenuePushConfig(supplierID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		config = service.DefaultSupplierRevenuePushConfig(supplierID)
	} else if err != nil {
		return nil, err
	}
	config.Enabled = request.Enabled
	config.Mode = strings.ToLower(strings.TrimSpace(request.Mode))
	config.ScheduleType = strings.ToLower(strings.TrimSpace(request.ScheduleType))
	config.Timezone = strings.TrimSpace(request.Timezone)
	config.DailyTime = strings.TrimSpace(request.DailyTime)
	config.HourlyMinute = request.HourlyMinute
	config.Currency = strings.ToUpper(strings.TrimSpace(request.Currency))
	config.NegativePolicy = strings.ToLower(strings.TrimSpace(request.NegativePolicy))
	config.RetryCount = request.RetryCount
	config.RetryIntervalSeconds = request.RetryIntervalSeconds
	config.RetryBackoff = strings.ToLower(strings.TrimSpace(request.RetryBackoff))
	config.TimeoutSeconds = request.TimeoutSeconds
	config.Environment = strings.ToLower(strings.TrimSpace(request.Environment))
	config.Endpoint = strings.TrimSpace(request.Endpoint)
	config.MockEndpoint = strings.TrimSpace(request.MockEndpoint)
	config.HTTPMethod = strings.ToUpper(strings.TrimSpace(request.HTTPMethod))
	config.ContentType = strings.TrimSpace(request.ContentType)
	config.HeadersJSON = strings.TrimSpace(request.HeadersJSON)
	config.BodyTemplate = strings.TrimSpace(request.BodyTemplate)
	config.SuccessHTTPStatus = request.SuccessHTTPStatus
	config.SuccessCodePath = strings.TrimSpace(request.SuccessCodePath)
	config.SuccessCodeValue = strings.TrimSpace(request.SuccessCodeValue)
	config.SuccessTypePath = strings.TrimSpace(request.SuccessTypePath)
	config.SuccessTypeValue = strings.TrimSpace(request.SuccessTypeValue)
	config.SuccessAmountPath = strings.TrimSpace(request.SuccessAmountPath)
	config.CallbackConfigJSON = strings.TrimSpace(request.CallbackConfigJSON)
	if config.EffectiveAt <= 0 {
		config.EffectiveAt = time.Now().Unix()
	}
	if config.CreatedBy == 0 {
		config.CreatedBy = operatorID
	}
	config.UpdatedBy = operatorID

	if config.Timezone == "" {
		config.Timezone = "Asia/Shanghai"
	}
	if config.DailyTime == "" {
		config.DailyTime = "01:00"
	}
	if config.RetryIntervalSeconds == 0 {
		config.RetryIntervalSeconds = 300
	}
	if config.TimeoutSeconds == 0 {
		config.TimeoutSeconds = 15
	}
	if config.RetryBackoff == "" {
		config.RetryBackoff = "fixed"
	}
	if config.NegativePolicy == "" {
		config.NegativePolicy = model.SupplierRevenueNegativeHold
	}
	if config.Environment == "" {
		config.Environment = model.SupplierRevenueEnvironmentMock
	}
	if config.HTTPMethod == "" {
		config.HTTPMethod = http.MethodPost
	}
	if config.ContentType == "" {
		config.ContentType = "application/json"
	}
	if config.SuccessHTTPStatus == 0 {
		config.SuccessHTTPStatus = http.StatusOK
	}

	if request.ClearPrivateKey {
		config.PrivateKeyPEM = ""
		config.PrivateKeyFingerprint = ""
	}
	if privateKey := strings.TrimSpace(request.PrivateKey); privateKey != "" {
		_, fingerprint, parseErr := service.ParseSupplierRevenuePrivateKey(privateKey)
		if parseErr != nil {
			return nil, parseErr
		}
		config.PrivateKeyPEM = privateKey
		config.PrivateKeyFingerprint = fingerprint
	}
	if config.Mode == model.SupplierRevenuePushModeEoraptor {
		config.HTTPMethod = http.MethodPost
		if config.Endpoint == "" {
			config.Endpoint = service.SupplierRevenueEoraptorProductionEndpoint
		}
		if config.MockEndpoint == "" {
			config.MockEndpoint = service.SupplierRevenueEoraptorMockEndpoint
		}
		if config.ContentType == "" {
			config.ContentType = "multipart/form-data"
		}
		if config.BodyTemplate == "" {
			config.BodyTemplate = service.SupplierRevenueEoraptorDefaultBodyTemplate
		}
	}
	return config, nil
}
