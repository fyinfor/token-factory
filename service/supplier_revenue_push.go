package service

import (
	"bytes"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/shopspring/decimal"
)

const (
	SupplierRevenueEoraptorProductionEndpoint  = "https://admin.eoraptor.org/api/open/syncCoin"
	SupplierRevenueEoraptorMockEndpoint        = "https://mock.apipost.net/mock/4ba1b38f30f1000/api/open/syncCoin?apipost_id=301db04f0c005"
	SupplierRevenueEoraptorDefaultBodyTemplate = `{"number":"{{number}}","timestamp":"{{timestamp}}","sign":"{{sign}}"}`
	SupplierRevenueGenericDefaultBodyTemplate  = `{"number":"{{number}}","timestamp":{{timestamp}},"supplier_id":{{supplier_id}},"batch_no":"{{batch_no}}","period_start":{{period_start}},"period_end":{{period_end}},"currency":"{{currency}}"}`

	supplierRevenueTaskInterval = time.Minute
	supplierRevenueMaxPeriods   = 400
	supplierRevenueResponseMax  = 64 << 10
)

var (
	supplierRevenueTaskOnce    sync.Once
	supplierRevenueTaskRunning atomic.Bool
)

type SupplierRevenueTestResult struct {
	DeliveryID int    `json:"delivery_id"`
	Outcome    string `json:"outcome"`
	HTTPStatus int    `json:"http_status"`
	Response   string `json:"response"`
	Message    string `json:"message"`
}

type supplierRevenueSendResult struct {
	Outcome      string
	Endpoint     string
	HTTPMethod   string
	RequestBody  string
	HTTPStatus   int
	ResponseBody string
	Message      string
	DurationMs   int64
}

func DefaultSupplierRevenuePushConfig(supplierID int) *model.SupplierRevenuePushConfig {
	return &model.SupplierRevenuePushConfig{
		SupplierID:           supplierID,
		Enabled:              false,
		Mode:                 model.SupplierRevenuePushModeGeneric,
		ScheduleType:         model.SupplierRevenueScheduleDaily,
		Timezone:             "Asia/Shanghai",
		DailyTime:            "01:00",
		HourlyMinute:         5,
		EffectiveAt:          time.Now().Unix(),
		Currency:             model.SupplierRevenueCurrencyUSDT,
		NegativePolicy:       model.SupplierRevenueNegativeHold,
		RetryCount:           3,
		RetryIntervalSeconds: 300,
		RetryBackoff:         "fixed",
		TimeoutSeconds:       15,
		Environment:          model.SupplierRevenueEnvironmentMock,
		Endpoint:             "",
		MockEndpoint:         SupplierRevenueEoraptorMockEndpoint,
		HTTPMethod:           http.MethodPost,
		ContentType:          "application/json",
		HeadersJSON:          "{}",
		BodyTemplate:         SupplierRevenueGenericDefaultBodyTemplate,
		CallbackConfigJSON:   "{}",
		SuccessHTTPStatus:    http.StatusOK,
		SuccessCodePath:      "",
		SuccessCodeValue:     "",
		SuccessTypePath:      "",
		SuccessTypeValue:     "",
		SuccessAmountPath:    "",
	}
}

func StartSupplierRevenuePushTask() {
	supplierRevenueTaskOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}
		gopool.Go(func() {
			logger.LogInfo(context.Background(), fmt.Sprintf("supplier revenue push task started: tick=%s", supplierRevenueTaskInterval))
			ticker := time.NewTicker(supplierRevenueTaskInterval)
			defer ticker.Stop()
			runSupplierRevenuePushTaskOnce()
			for range ticker.C {
				runSupplierRevenuePushTaskOnce()
			}
		})
	})
}

func runSupplierRevenuePushTaskOnce() {
	if !supplierRevenueTaskRunning.CompareAndSwap(false, true) {
		return
	}
	defer supplierRevenueTaskRunning.Store(false)

	ctx := context.Background()
	if err := model.FinalizeRetryingManualSupplierRevenueDeliveries(); err != nil {
		logger.LogError(ctx, fmt.Sprintf("finalize retrying manual supplier revenue deliveries failed: %v", err))
	}
	deliveries, err := model.ListRetryableSupplierRevenueDeliveries(time.Now().Unix(), 100)
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("load supplier revenue retries failed: %v", err))
	} else {
		for _, delivery := range deliveries {
			if err := attemptSupplierRevenueDelivery(ctx, delivery.ID); err != nil {
				logger.LogWarn(ctx, fmt.Sprintf("supplier revenue retry delivery=%d failed: %v", delivery.ID, err))
			}
		}
	}

	configs, err := model.ListEnabledSupplierRevenuePushConfigs()
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("load supplier revenue configs failed: %v", err))
		return
	}
	for _, config := range configs {
		if _, err := processSupplierRevenueConfig(ctx, config, false); err != nil {
			logger.LogWarn(ctx, fmt.Sprintf("supplier revenue process supplier=%d failed: %v", config.SupplierID, err))
		}
	}
}

func RunSupplierRevenuePushForSupplier(ctx context.Context, supplierID int) (*model.SupplierRevenueDelivery, error) {
	config, err := model.GetSupplierRevenuePushConfig(supplierID)
	if err != nil {
		return nil, err
	}
	if !config.Enabled {
		return nil, errors.New("收益推送尚未启用")
	}
	return processSupplierRevenueConfig(ctx, config, true)
}

func ManualSupplierRevenuePush(ctx context.Context, supplierID int, amountText, remark string) (*model.SupplierRevenueDelivery, error) {
	config, err := model.GetSupplierRevenuePushConfig(supplierID)
	if err != nil {
		return nil, err
	}
	if !config.Enabled {
		return nil, errors.New("收益推送尚未启用")
	}
	if err = validateSupplierRevenueConfig(config, false); err != nil {
		return nil, err
	}
	active, err := model.HasActiveOrUnknownSupplierRevenueDelivery(supplierID)
	if err != nil {
		return nil, err
	}
	if active {
		return nil, errors.New("存在发送中、重试中或状态未知的批次，请先处理后再手动推送")
	}
	amountText = strings.TrimSpace(amountText)
	if amountText == "" || len(amountText) > 64 {
		return nil, errors.New("请输入有效的手动推送金额")
	}
	amount, err := decimal.NewFromString(amountText)
	if err != nil {
		return nil, errors.New("手动推送金额格式不正确")
	}
	if amount.IsNegative() && config.NegativePolicy != model.SupplierRevenueNegativeAllow {
		return nil, errors.New("当前负数处理策略不允许手动推送负数金额")
	}
	remark = strings.TrimSpace(remark)
	if len([]rune(remark)) > 500 {
		return nil, errors.New("手动推送备注不能超过500个字符")
	}
	snapshot := *config
	snapshot.PrivateKeyPEM = ""
	snapshotBytes, err := common.Marshal(&snapshot)
	if err != nil {
		return nil, err
	}
	endpoint, err := resolveSupplierRevenueEndpoint(config, false)
	if err != nil {
		return nil, err
	}
	now := time.Now().Unix()
	delivery := &model.SupplierRevenueDelivery{
		SupplierID:         supplierID,
		BatchNo:            fmt.Sprintf("SRP-MANUAL-%d-%s", supplierID, strings.ReplaceAll(common.GetUUID(), "-", "")),
		Kind:               model.SupplierRevenueDeliveryKindManual,
		PeriodEnd:          now,
		Amount:             roundSupplierRevenueAmount(amount).StringFixed(6),
		Currency:           config.Currency,
		Status:             model.SupplierRevenueDeliveryCreated,
		MaxAttempts:        1,
		Remark:             remark,
		EndpointSnapshot:   endpoint,
		ConfigSnapshotJSON: string(snapshotBytes),
	}
	if err = model.CreateSupplierRevenueDelivery(delivery, nil); err != nil {
		return nil, err
	}
	if err = attemptSupplierRevenueDelivery(ctx, delivery.ID); err != nil {
		return delivery, err
	}
	return model.GetSupplierRevenueDelivery(delivery.ID)
}

func processSupplierRevenueConfig(ctx context.Context, config *model.SupplierRevenuePushConfig, manual bool) (*model.SupplierRevenueDelivery, error) {
	if config == nil || config.SupplierID <= 0 {
		return nil, errors.New("invalid supplier revenue config")
	}
	if err := validateSupplierRevenueConfig(config, false); err != nil {
		return nil, err
	}
	created, err := createDueSupplierRevenuePeriods(config, time.Now())
	if err != nil {
		return nil, err
	}
	active, err := model.HasActiveOrUnknownSupplierRevenueDelivery(config.SupplierID)
	if err != nil {
		return nil, err
	}
	if active {
		return nil, nil
	}
	blocking, err := model.ListSupplierRevenuePeriodsByStatus(config.SupplierID, []string{model.SupplierRevenuePeriodHeld, model.SupplierRevenuePeriodUnknown})
	if err != nil {
		return nil, err
	}
	if len(blocking) > 0 {
		return nil, nil
	}
	periods, err := model.ListSupplierRevenuePeriodsByStatus(config.SupplierID, []string{model.SupplierRevenuePeriodPending})
	if err != nil {
		return nil, err
	}
	if len(periods) == 0 {
		return nil, nil
	}
	hasNewPeriod := false
	for _, period := range periods {
		if period.LastDeliveryID == 0 {
			hasNewPeriod = true
			break
		}
	}
	if !hasNewPeriod && !manual && created == 0 {
		return nil, nil
	}
	delivery, err := buildSupplierRevenueDelivery(config, periods, model.SupplierRevenueDeliveryKindScheduled)
	if err != nil || delivery == nil {
		return delivery, err
	}
	if err = model.CreateSupplierRevenueDelivery(delivery, periods); err != nil {
		return nil, err
	}
	if err = attemptSupplierRevenueDelivery(ctx, delivery.ID); err != nil {
		return delivery, err
	}
	return model.GetSupplierRevenueDelivery(delivery.ID)
}

func createDueSupplierRevenuePeriods(config *model.SupplierRevenuePushConfig, now time.Time) (int, error) {
	location, err := time.LoadLocation(config.Timezone)
	if err != nil {
		return 0, fmt.Errorf("invalid timezone %q: %w", config.Timezone, err)
	}
	latestEnd, err := model.LatestSupplierRevenuePeriodEnd(config.SupplierID, config.ScheduleType)
	if err != nil {
		return 0, err
	}
	start := nextCompleteSupplierRevenuePeriodStart(time.Unix(config.EffectiveAt, 0).In(location), config.ScheduleType)
	if latestEnd > start.Unix() {
		start = time.Unix(latestEnd, 0).In(location)
	}
	channelIDs, err := model.GetSupplierRevenueChannelIDs(config.SupplierID)
	if err != nil {
		return 0, err
	}
	created := 0
	for i := 0; i < supplierRevenueMaxPeriods; i++ {
		end := nextSupplierRevenuePeriodEnd(start, config.ScheduleType)
		dueAt, dueErr := supplierRevenuePeriodDueAt(end, config)
		if dueErr != nil {
			return created, dueErr
		}
		if dueAt.After(now.In(location)) {
			break
		}
		rawQuota, sumErr := model.SumSupplierRevenueQuota(channelIDs, start.Unix(), end.Unix())
		if sumErr != nil {
			return created, sumErr
		}
		rawAmount, amount, rate := supplierRevenueAmount(rawQuota, config)
		status := model.SupplierRevenuePeriodPending
		if rawAmount.IsNegative() && config.NegativePolicy == model.SupplierRevenueNegativeHold {
			status = model.SupplierRevenuePeriodHeld
		}
		period := &model.SupplierRevenuePeriod{
			SupplierID: config.SupplierID, ScheduleType: config.ScheduleType,
			PeriodStart: start.Unix(), PeriodEnd: end.Unix(), RawQuota: rawQuota,
			RawAmount: rawAmount.String(), Amount: amount.StringFixed(6),
			Currency: config.Currency, ExchangeRate: rate.String(), Status: status,
		}
		wasCreated, createErr := model.CreateSupplierRevenuePeriod(period)
		if createErr != nil {
			return created, createErr
		}
		if wasCreated {
			created++
		}
		start = end
	}
	return created, nil
}

func nextCompleteSupplierRevenuePeriodStart(effective time.Time, scheduleType string) time.Time {
	if scheduleType == model.SupplierRevenueScheduleHourly {
		aligned := time.Date(effective.Year(), effective.Month(), effective.Day(), effective.Hour(), 0, 0, 0, effective.Location())
		if effective.Equal(aligned) {
			return aligned
		}
		return aligned.Add(time.Hour)
	}
	aligned := time.Date(effective.Year(), effective.Month(), effective.Day(), 0, 0, 0, 0, effective.Location())
	if effective.Equal(aligned) {
		return aligned
	}
	return aligned.AddDate(0, 0, 1)
}

func nextSupplierRevenuePeriodEnd(start time.Time, scheduleType string) time.Time {
	if scheduleType == model.SupplierRevenueScheduleHourly {
		return start.Add(time.Hour)
	}
	return start.AddDate(0, 0, 1)
}

func supplierRevenuePeriodDueAt(periodEnd time.Time, config *model.SupplierRevenuePushConfig) (time.Time, error) {
	if config.ScheduleType == model.SupplierRevenueScheduleHourly {
		return periodEnd.Add(time.Duration(config.HourlyMinute) * time.Minute), nil
	}
	parts := strings.Split(config.DailyTime, ":")
	if len(parts) != 2 {
		return time.Time{}, errors.New("daily_time must use HH:mm")
	}
	hour, err := strconv.Atoi(parts[0])
	if err != nil {
		return time.Time{}, errors.New("invalid daily hour")
	}
	minute, err := strconv.Atoi(parts[1])
	if err != nil {
		return time.Time{}, errors.New("invalid daily minute")
	}
	return time.Date(periodEnd.Year(), periodEnd.Month(), periodEnd.Day(), hour, minute, 0, 0, periodEnd.Location()), nil
}

func supplierRevenueAmount(rawQuota int64, config *model.SupplierRevenuePushConfig) (decimal.Decimal, decimal.Decimal, decimal.Decimal) {
	quotaUnit := decimal.NewFromFloat(common.QuotaPerUnit)
	if quotaUnit.IsZero() {
		quotaUnit = decimal.NewFromInt(1)
	}
	rate := decimal.NewFromInt(1)
	if config.Currency == model.SupplierRevenueCurrencyCNY {
		rate = decimal.NewFromFloat(operation_setting.USDExchangeRate)
		if !rate.IsPositive() {
			rate = decimal.NewFromInt(1)
		}
	}
	raw := decimal.NewFromInt(rawQuota).Div(quotaUnit).Mul(rate)
	return raw, roundSupplierRevenueAmount(raw), rate
}

func roundSupplierRevenueAmount(value decimal.Decimal) decimal.Decimal {
	if value.IsNegative() {
		return value.Abs().RoundCeil(6).Neg()
	}
	return value.RoundCeil(6)
}

func buildSupplierRevenueDelivery(config *model.SupplierRevenuePushConfig, periods []*model.SupplierRevenuePeriod, kind string) (*model.SupplierRevenueDelivery, error) {
	if len(periods) == 0 {
		return nil, nil
	}
	rawAmount := decimal.Zero
	var rawQuota int64
	for _, period := range periods {
		amount, err := decimal.NewFromString(period.RawAmount)
		if err != nil {
			return nil, fmt.Errorf("invalid period raw amount: %w", err)
		}
		rawAmount = rawAmount.Add(amount)
		rawQuota += period.RawQuota
	}
	if rawAmount.IsNegative() {
		switch config.NegativePolicy {
		case model.SupplierRevenueNegativeAllow:
		case model.SupplierRevenueNegativeCarry:
			return nil, nil
		default:
			return nil, nil
		}
	}
	snapshot := *config
	snapshot.PrivateKeyPEM = ""
	snapshotBytes, err := common.Marshal(&snapshot)
	if err != nil {
		return nil, err
	}
	endpoint, err := resolveSupplierRevenueEndpoint(config, false)
	if err != nil {
		return nil, err
	}
	return &model.SupplierRevenueDelivery{
		SupplierID: config.SupplierID,
		BatchNo:    fmt.Sprintf("SRP-%d-%s", config.SupplierID, strings.ReplaceAll(common.GetUUID(), "-", "")),
		Kind:       kind, PeriodStart: periods[0].PeriodStart, PeriodEnd: periods[len(periods)-1].PeriodEnd,
		RawQuota: rawQuota, Amount: roundSupplierRevenueAmount(rawAmount).StringFixed(6), Currency: config.Currency,
		Status: model.SupplierRevenueDeliveryCreated, MaxAttempts: config.RetryCount + 1,
		EndpointSnapshot: endpoint, ConfigSnapshotJSON: string(snapshotBytes),
	}, nil
}

func attemptSupplierRevenueDelivery(ctx context.Context, deliveryID int) error {
	claimed, err := model.ClaimSupplierRevenueDelivery(deliveryID)
	if err != nil || !claimed {
		return err
	}
	delivery, err := model.GetSupplierRevenueDelivery(deliveryID)
	if err != nil {
		message := "读取已领取的推送批次失败: " + err.Error()
		if markErr := model.MarkSupplierRevenueDeliveryFailed(deliveryID, 0, message); markErr != nil {
			return errors.Join(err, markErr)
		}
		return err
	}
	currentConfig, err := model.GetSupplierRevenuePushConfig(delivery.SupplierID)
	if err != nil {
		message := "读取供应商推送配置失败: " + err.Error()
		if markErr := model.MarkSupplierRevenueDeliveryFailed(delivery.ID, delivery.AttemptCount, message); markErr != nil {
			return errors.Join(err, markErr)
		}
		return err
	}
	config := *currentConfig
	if strings.TrimSpace(delivery.ConfigSnapshotJSON) != "" {
		var snapshot model.SupplierRevenuePushConfig
		if err := common.UnmarshalJsonStr(delivery.ConfigSnapshotJSON, &snapshot); err == nil {
			snapshot.PrivateKeyPEM = currentConfig.PrivateKeyPEM
			config = snapshot
		}
	}
	privateKey := ""
	if config.Mode == model.SupplierRevenuePushModeEoraptor {
		privateKey = config.PrivateKeyPEM
	}
	result := sendSupplierRevenue(ctx, &config, delivery, privateKey, delivery.EndpointSnapshot)
	return finishSupplierRevenueDeliveryAttempt(delivery, result)
}

func finishSupplierRevenueDeliveryAttempt(delivery *model.SupplierRevenueDelivery, result *supplierRevenueSendResult) error {
	attemptNo := delivery.AttemptCount + 1
	attempt := &model.SupplierRevenueAttempt{
		DeliveryID: delivery.ID, AttemptNo: attemptNo, Endpoint: result.Endpoint, HTTPMethod: result.HTTPMethod,
		RequestBody: result.RequestBody, HTTPStatus: result.HTTPStatus, ResponseBody: result.ResponseBody,
		ErrorMessage: result.Message, Outcome: result.Outcome, DurationMs: result.DurationMs, RequestedAt: time.Now().Unix(),
	}
	if err := model.CreateSupplierRevenueAttempt(attempt); err != nil {
		logger.LogError(context.Background(), fmt.Sprintf("save supplier revenue attempt delivery=%d failed: %v", delivery.ID, err))
	}
	switch result.Outcome {
	case model.SupplierRevenueAttemptSuccess:
		return model.MarkSupplierRevenueDeliverySuccess(delivery.ID, attemptNo)
	case model.SupplierRevenueAttemptUnknown:
		return model.MarkSupplierRevenueDeliveryUnknown(delivery.ID, attemptNo, result.Message)
	default:
		if delivery.Kind != model.SupplierRevenueDeliveryKindManual && attemptNo < delivery.MaxAttempts {
			delay := supplierRevenueRetryDelay(delivery, attemptNo)
			return model.MarkSupplierRevenueDeliveryRetry(delivery.ID, attemptNo, time.Now().Add(delay).Unix(), result.Message)
		}
		return model.MarkSupplierRevenueDeliveryFailed(delivery.ID, attemptNo, result.Message)
	}
}

func supplierRevenueRetryDelay(delivery *model.SupplierRevenueDelivery, attemptNo int) time.Duration {
	base := 300
	var snapshot model.SupplierRevenuePushConfig
	if common.UnmarshalJsonStr(delivery.ConfigSnapshotJSON, &snapshot) == nil && snapshot.RetryIntervalSeconds > 0 {
		base = snapshot.RetryIntervalSeconds
	}
	delay := time.Duration(base) * time.Second
	if snapshot.RetryBackoff == "exponential" && attemptNo > 1 {
		shift := attemptNo - 1
		if shift > 8 {
			shift = 8
		}
		delay *= time.Duration(1 << shift)
	}
	if delay > 24*time.Hour {
		return 24 * time.Hour
	}
	return delay
}

func TestSupplierRevenuePush(ctx context.Context, config *model.SupplierRevenuePushConfig, privateKeyOverride string) (*SupplierRevenueTestResult, error) {
	if err := validateSupplierRevenueConfig(config, true); err != nil {
		return nil, err
	}
	endpoint, err := resolveSupplierRevenueEndpoint(config, true)
	if err != nil {
		return nil, err
	}
	snapshot := *config
	snapshot.PrivateKeyPEM = ""
	snapshotBytes, err := common.Marshal(&snapshot)
	if err != nil {
		return nil, err
	}
	delivery := &model.SupplierRevenueDelivery{
		SupplierID: config.SupplierID, BatchNo: fmt.Sprintf("SRP-TEST-%d-%s", config.SupplierID, strings.ReplaceAll(common.GetUUID(), "-", "")),
		Kind: model.SupplierRevenueDeliveryKindTest, Amount: "100.000000", Currency: config.Currency,
		Status: model.SupplierRevenueDeliverySending, MaxAttempts: 1, EndpointSnapshot: endpoint,
		ConfigSnapshotJSON: string(snapshotBytes),
	}
	if err := model.CreateSupplierRevenueDelivery(delivery, nil); err != nil {
		return nil, err
	}
	privateKey := strings.TrimSpace(privateKeyOverride)
	if config.Mode == model.SupplierRevenuePushModeEoraptor && privateKey == "" {
		privateKey = config.PrivateKeyPEM
	}
	result := sendSupplierRevenue(ctx, config, delivery, privateKey, endpoint)
	if err := finishSupplierRevenueDeliveryAttempt(delivery, result); err != nil {
		return nil, err
	}
	return &SupplierRevenueTestResult{
		DeliveryID: delivery.ID, Outcome: result.Outcome, HTTPStatus: result.HTTPStatus,
		Response: result.ResponseBody, Message: result.Message,
	}, nil
}

func sendSupplierRevenue(ctx context.Context, config *model.SupplierRevenuePushConfig, delivery *model.SupplierRevenueDelivery, privateKey, endpoint string) *supplierRevenueSendResult {
	if config.Mode == model.SupplierRevenuePushModeEoraptor {
		return sendEoraptorSupplierRevenue(ctx, config, delivery, privateKey, endpoint)
	}
	return sendGenericSupplierRevenue(ctx, config, delivery, endpoint)
}

func sendEoraptorSupplierRevenue(ctx context.Context, config *model.SupplierRevenuePushConfig, delivery *model.SupplierRevenueDelivery, privateKeyPEM, endpoint string) *supplierRevenueSendResult {
	started := time.Now()
	result := &supplierRevenueSendResult{Outcome: model.SupplierRevenueAttemptFailed, Endpoint: endpoint, HTTPMethod: http.MethodPost}
	privateKey, _, err := ParseSupplierRevenuePrivateKey(privateKeyPEM)
	if err != nil {
		result.Message = err.Error()
		return result
	}
	timestamp := time.Now().Unix()
	signatureSource := fmt.Sprintf("number=%s&timestamp=%d", delivery.Amount, timestamp)
	digest := sha256.Sum256([]byte(signatureSource))
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, digest[:])
	if err != nil {
		result.Message = "RSA签名失败: " + err.Error()
		return result
	}
	signatureValue := base64.StdEncoding.EncodeToString(signature)
	template := strings.TrimSpace(config.BodyTemplate)
	if template == "" {
		template = SupplierRevenueEoraptorDefaultBodyTemplate
	}
	rendered := renderSupplierRevenueTemplate(template, supplierRevenueTemplateValues(config, delivery, timestamp, signatureValue))
	result.RequestBody = strings.ReplaceAll(rendered, signatureValue, "[REDACTED]")

	contentType := strings.ToLower(strings.TrimSpace(config.ContentType))
	var requestBody io.Reader
	var requestContentType string
	switch contentType {
	case "application/json":
		var payload map[string]any
		if err = common.Unmarshal([]byte(rendered), &payload); err != nil {
			result.Message = "发送参数模板不是有效的JSON对象: " + err.Error()
			return result
		}
		requestBody = strings.NewReader(rendered)
		requestContentType = "application/json"
	case "application/x-www-form-urlencoded":
		fields, parseErr := parseSupplierRevenueTemplateFields(rendered)
		if parseErr != nil {
			result.Message = parseErr.Error()
			return result
		}
		values := url.Values{}
		for key, value := range fields {
			fieldValue, valueErr := supplierRevenueFieldString(value)
			if valueErr != nil {
				result.Message = "编码表单字段失败: " + valueErr.Error()
				return result
			}
			values.Set(key, fieldValue)
		}
		requestBody = strings.NewReader(values.Encode())
		requestContentType = "application/x-www-form-urlencoded"
	default:
		fields, parseErr := parseSupplierRevenueTemplateFields(rendered)
		if parseErr != nil {
			result.Message = parseErr.Error()
			return result
		}
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		for key, value := range fields {
			fieldValue, valueErr := supplierRevenueFieldString(value)
			if valueErr != nil {
				result.Message = "编码multipart字段失败: " + valueErr.Error()
				return result
			}
			if writeErr := writer.WriteField(key, fieldValue); writeErr != nil {
				result.Message = "构造multipart请求失败: " + writeErr.Error()
				return result
			}
		}
		if err = writer.Close(); err != nil {
			result.Message = "构造multipart请求失败: " + err.Error()
			return result
		}
		requestBody = &body
		requestContentType = writer.FormDataContentType()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, requestBody)
	if err != nil {
		result.Message = "创建请求失败: " + err.Error()
		return result
	}
	req.Header.Set("Content-Type", requestContentType)
	if err = applySupplierRevenueHeaders(req, config.HeadersJSON); err != nil {
		result.Message = err.Error()
		return result
	}
	status, responseBody, requestErr, unknown := doSupplierRevenueRequest(req, config.TimeoutSeconds)
	result.DurationMs = time.Since(started).Milliseconds()
	result.HTTPStatus = status
	result.ResponseBody = responseBody
	if requestErr != nil {
		result.Message = requestErr.Error()
		if unknown {
			result.Outcome = model.SupplierRevenueAttemptUnknown
		}
		return result
	}
	if err = validateGenericSupplierRevenueResponse(config, delivery.Amount, status, responseBody); err != nil {
		result.Message = err.Error()
		return result
	}
	result.Outcome = model.SupplierRevenueAttemptSuccess
	return result
}

func sendGenericSupplierRevenue(ctx context.Context, config *model.SupplierRevenuePushConfig, delivery *model.SupplierRevenueDelivery, endpoint string) *supplierRevenueSendResult {
	started := time.Now()
	method := strings.ToUpper(strings.TrimSpace(config.HTTPMethod))
	result := &supplierRevenueSendResult{Outcome: model.SupplierRevenueAttemptFailed, Endpoint: endpoint, HTTPMethod: method}
	timestamp := time.Now().Unix()
	template := strings.TrimSpace(config.BodyTemplate)
	if template == "" {
		template = SupplierRevenueGenericDefaultBodyTemplate
	}
	template = renderSupplierRevenueTemplate(template, supplierRevenueTemplateValues(config, delivery, timestamp, ""))
	result.RequestBody = template
	var body io.Reader = strings.NewReader(template)
	if method == http.MethodGet {
		body = nil
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		result.Message = "创建请求失败: " + err.Error()
		return result
	}
	contentType := strings.TrimSpace(config.ContentType)
	if contentType == "" {
		contentType = "application/json"
	}
	req.Header.Set("Content-Type", contentType)
	if err := applySupplierRevenueHeaders(req, config.HeadersJSON); err != nil {
		result.Message = err.Error()
		return result
	}
	status, responseBody, requestErr, unknown := doSupplierRevenueRequest(req, config.TimeoutSeconds)
	result.DurationMs = time.Since(started).Milliseconds()
	result.HTTPStatus = status
	result.ResponseBody = responseBody
	if requestErr != nil {
		result.Message = requestErr.Error()
		if unknown {
			result.Outcome = model.SupplierRevenueAttemptUnknown
		}
		return result
	}
	if err := validateGenericSupplierRevenueResponse(config, delivery.Amount, status, responseBody); err != nil {
		result.Message = err.Error()
		return result
	}
	result.Outcome = model.SupplierRevenueAttemptSuccess
	return result
}

func supplierRevenueTemplateValues(config *model.SupplierRevenuePushConfig, delivery *model.SupplierRevenueDelivery, timestamp int64, signature string) map[string]string {
	callbackConfig := strings.TrimSpace(config.CallbackConfigJSON)
	if callbackConfig == "" {
		callbackConfig = "{}"
	}
	return map[string]string{
		"{{number}}": delivery.Amount, "{{timestamp}}": strconv.FormatInt(timestamp, 10),
		"{{supplier_id}}": strconv.Itoa(delivery.SupplierID), "{{batch_no}}": delivery.BatchNo,
		"{{period_start}}": strconv.FormatInt(delivery.PeriodStart, 10), "{{period_end}}": strconv.FormatInt(delivery.PeriodEnd, 10),
		"{{currency}}": delivery.Currency, "{{callback_config}}": callbackConfig, "{{sign}}": signature,
	}
}

func renderSupplierRevenueTemplate(template string, replacements map[string]string) string {
	for key, value := range replacements {
		template = strings.ReplaceAll(template, key, value)
	}
	return template
}

func parseSupplierRevenueTemplateFields(rendered string) (map[string]any, error) {
	var fields map[string]any
	if err := common.Unmarshal([]byte(rendered), &fields); err != nil {
		return nil, fmt.Errorf("发送参数模板不是有效的JSON对象: %w", err)
	}
	if len(fields) == 0 {
		return nil, errors.New("发送参数模板不能为空")
	}
	return fields, nil
}

func supplierRevenueFieldString(value any) (string, error) {
	switch typed := value.(type) {
	case string:
		return typed, nil
	case nil:
		return "", nil
	case map[string]any, []any:
		encoded, err := common.Marshal(typed)
		return string(encoded), err
	default:
		return fmt.Sprint(typed), nil
	}
}

func applySupplierRevenueHeaders(req *http.Request, headersJSON string) error {
	if strings.TrimSpace(headersJSON) == "" {
		return nil
	}
	var headers map[string]string
	if err := common.UnmarshalJsonStr(headersJSON, &headers); err != nil {
		return fmt.Errorf("请求头JSON无效: %w", err)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	return nil
}

func doSupplierRevenueRequest(req *http.Request, timeoutSeconds int) (int, string, error, bool) {
	if timeoutSeconds <= 0 {
		timeoutSeconds = 15
	}
	client := &http.Client{
		Timeout:       time.Duration(timeoutSeconds) * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("推送请求失败: %w", err), !supplierRevenueRequestDefinitelyUnsent(err)
	}
	defer resp.Body.Close()
	body, readErr := io.ReadAll(io.LimitReader(resp.Body, supplierRevenueResponseMax+1))
	if readErr != nil {
		return resp.StatusCode, "", fmt.Errorf("读取响应失败: %w", readErr), true
	}
	if len(body) > supplierRevenueResponseMax {
		return resp.StatusCode, string(body[:supplierRevenueResponseMax]), errors.New("远端响应超过64KB限制"), false
	}
	return resp.StatusCode, string(body), nil, false
}

func supplierRevenueRequestDefinitelyUnsent(err error) bool {
	var urlErr *url.Error
	if !errors.As(err, &urlErr) {
		return false
	}
	if urlErr.Timeout() {
		return false
	}
	var opErr *net.OpError
	if errors.As(urlErr.Err, &opErr) {
		return opErr.Op == "dial"
	}
	return false
}

func validateGenericSupplierRevenueResponse(config *model.SupplierRevenuePushConfig, amount string, status int, body string) error {
	expectedStatus := config.SuccessHTTPStatus
	if expectedStatus == 0 {
		expectedStatus = http.StatusOK
	}
	if status != expectedStatus {
		return fmt.Errorf("HTTP状态不符合成功规则: expected=%d actual=%d", expectedStatus, status)
	}
	pathsConfigured := strings.TrimSpace(config.SuccessCodePath) != "" || strings.TrimSpace(config.SuccessTypePath) != "" || strings.TrimSpace(config.SuccessAmountPath) != ""
	if !pathsConfigured {
		return nil
	}
	var payload any
	if err := common.Unmarshal([]byte(body), &payload); err != nil {
		return fmt.Errorf("解析响应JSON失败: %w", err)
	}
	checks := []struct{ path, expected, label string }{
		{config.SuccessCodePath, config.SuccessCodeValue, "code"},
		{config.SuccessTypePath, config.SuccessTypeValue, "type"},
	}
	for _, check := range checks {
		if strings.TrimSpace(check.path) == "" {
			continue
		}
		value, ok := supplierRevenueJSONPath(payload, check.path)
		if !ok || fmt.Sprint(value) != check.expected {
			return fmt.Errorf("响应%s不符合成功规则: path=%s expected=%s actual=%v", check.label, check.path, check.expected, value)
		}
	}
	if strings.TrimSpace(config.SuccessAmountPath) != "" {
		value, ok := supplierRevenueJSONPath(payload, config.SuccessAmountPath)
		if !ok {
			return fmt.Errorf("响应缺少金额字段: %s", config.SuccessAmountPath)
		}
		want, wantErr := decimal.NewFromString(amount)
		got, gotErr := decimal.NewFromString(fmt.Sprint(value))
		if wantErr != nil || gotErr != nil || !want.Equal(got) {
			return fmt.Errorf("响应金额不一致: sent=%s returned=%v", amount, value)
		}
	}
	return nil
}

func supplierRevenueJSONPath(value any, path string) (any, bool) {
	current := value
	for _, part := range strings.Split(path, ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = object[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func ParseSupplierRevenuePrivateKey(privateKeyPEM string) (*rsa.PrivateKey, string, error) {
	block, _ := pem.Decode([]byte(strings.TrimSpace(privateKeyPEM)))
	if block == nil {
		return nil, "", errors.New("私钥不是有效的PEM格式")
	}
	if x509.IsEncryptedPEMBlock(block) {
		return nil, "", errors.New("暂不支持带密码的PEM私钥")
	}
	var privateKey *rsa.PrivateKey
	var err error
	switch block.Type {
	case "RSA PRIVATE KEY":
		privateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	case "PRIVATE KEY":
		var parsed any
		parsed, err = x509.ParsePKCS8PrivateKey(block.Bytes)
		if err == nil {
			var ok bool
			privateKey, ok = parsed.(*rsa.PrivateKey)
			if !ok {
				return nil, "", errors.New("PKCS#8私钥不是RSA类型")
			}
		}
	default:
		return nil, "", fmt.Errorf("不支持的私钥类型: %s", block.Type)
	}
	if err != nil {
		return nil, "", fmt.Errorf("解析RSA私钥失败: %w", err)
	}
	if err = privateKey.Validate(); err != nil {
		return nil, "", fmt.Errorf("RSA私钥校验失败: %w", err)
	}
	publicDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, "", err
	}
	fingerprintBytes := sha256.Sum256(publicDER)
	fingerprint := strings.ToUpper(hex.EncodeToString(fingerprintBytes[:]))
	return privateKey, fingerprint, nil
}

func resolveSupplierRevenueEndpoint(config *model.SupplierRevenuePushConfig, forceMock bool) (string, error) {
	if config.Mode == model.SupplierRevenuePushModeEoraptor {
		if forceMock || config.Environment == model.SupplierRevenueEnvironmentMock {
			mockEndpoint := strings.TrimSpace(config.MockEndpoint)
			if mockEndpoint == "" {
				mockEndpoint = SupplierRevenueEoraptorMockEndpoint
			}
			return validateSupplierRevenueEndpoint(mockEndpoint, "Mock测试地址")
		}
	}
	return validateSupplierRevenueEndpoint(config.Endpoint, "推送地址")
}

func validateSupplierRevenueEndpoint(endpoint, label string) (string, error) {
	endpoint = strings.TrimSpace(endpoint)
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", fmt.Errorf("%s必须是有效的HTTP或HTTPS URL", label)
	}
	return endpoint, nil
}

func validateSupplierRevenueConfig(config *model.SupplierRevenuePushConfig, testing bool) error {
	if config == nil || config.SupplierID <= 0 {
		return errors.New("无效的供应商推送配置")
	}
	if config.Mode != model.SupplierRevenuePushModeEoraptor && config.Mode != model.SupplierRevenuePushModeGeneric {
		return errors.New("推送类型仅支持 eoraptor 或 generic")
	}
	if config.Mode == model.SupplierRevenuePushModeEoraptor {
		if !testing && config.Enabled {
			if strings.TrimSpace(config.PrivateKeyPEM) == "" {
				return errors.New("请上传供应商RSA私钥")
			}
			if _, _, err := ParseSupplierRevenuePrivateKey(config.PrivateKeyPEM); err != nil {
				return errors.New("已保存的RSA私钥无效，请重新上传private.pem")
			}
		}
		endpoint := strings.TrimSpace(config.Endpoint)
		parsed, err := url.Parse(endpoint)
		if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return errors.New("eoraptor定制推送地址必须是有效的HTTP或HTTPS URL")
		}
		mockEndpoint := strings.TrimSpace(config.MockEndpoint)
		if mockEndpoint == "" {
			mockEndpoint = SupplierRevenueEoraptorMockEndpoint
		}
		if _, err = validateSupplierRevenueEndpoint(mockEndpoint, "Mock测试地址"); err != nil {
			return err
		}
		contentType := strings.ToLower(strings.TrimSpace(config.ContentType))
		if contentType != "multipart/form-data" && contentType != "application/json" && contentType != "application/x-www-form-urlencoded" {
			return errors.New("eoraptor发送格式仅支持multipart、JSON或表单编码")
		}
		template := strings.TrimSpace(config.BodyTemplate)
		if !strings.Contains(template, "{{number}}") || !strings.Contains(template, "{{timestamp}}") || !strings.Contains(template, "{{sign}}") {
			return errors.New("eoraptor发送参数模板必须包含number、timestamp和sign模板变量")
		}
		testRendered := renderSupplierRevenueTemplate(template, map[string]string{
			"{{number}}": "1.000000", "{{timestamp}}": "1", "{{sign}}": "signature",
			"{{supplier_id}}": "1", "{{batch_no}}": "test", "{{period_start}}": "1",
			"{{period_end}}": "2", "{{currency}}": "USDT", "{{callback_config}}": "{}",
		})
		if _, err = parseSupplierRevenueTemplateFields(testRendered); err != nil {
			return err
		}
	}
	if config.ScheduleType != model.SupplierRevenueScheduleDaily && config.ScheduleType != model.SupplierRevenueScheduleHourly {
		return errors.New("推送频率仅支持daily或hourly")
	}
	if config.Currency != model.SupplierRevenueCurrencyUSD && config.Currency != model.SupplierRevenueCurrencyCNY && config.Currency != model.SupplierRevenueCurrencyUSDT {
		return errors.New("推送币种仅支持USD、CNY或USDT")
	}
	if _, err := time.LoadLocation(config.Timezone); err != nil {
		return errors.New("无效的时区")
	}
	if config.HourlyMinute < 0 || config.HourlyMinute > 59 {
		return errors.New("每小时推送分钟必须在0到59之间")
	}
	if config.RetryCount < 0 || config.RetryCount > 10 {
		return errors.New("重试次数必须在0到10之间")
	}
	if config.RetryIntervalSeconds < 1 || config.RetryIntervalSeconds > 86400 {
		return errors.New("重试间隔必须在1到86400秒之间")
	}
	if config.TimeoutSeconds < 1 || config.TimeoutSeconds > 120 {
		return errors.New("请求超时必须在1到120秒之间")
	}
	if config.NegativePolicy != model.SupplierRevenueNegativeHold && config.NegativePolicy != model.SupplierRevenueNegativeAllow && config.NegativePolicy != model.SupplierRevenueNegativeCarry {
		return errors.New("无效的负数处理策略")
	}
	if config.RetryBackoff != "fixed" && config.RetryBackoff != "exponential" {
		return errors.New("重试策略仅支持fixed或exponential")
	}
	for label, value := range map[string]string{"请求头JSON": config.HeadersJSON, "回调参数JSON": config.CallbackConfigJSON} {
		if strings.TrimSpace(value) == "" {
			continue
		}
		var object map[string]any
		if err := common.UnmarshalJsonStr(value, &object); err != nil {
			return fmt.Errorf("%s无效: %w", label, err)
		}
	}
	if _, err := resolveSupplierRevenueEndpoint(config, testing && config.Mode == model.SupplierRevenuePushModeEoraptor); err != nil {
		return err
	}
	return nil
}

func ValidateSupplierRevenuePushConfig(config *model.SupplierRevenuePushConfig, testing bool) error {
	return validateSupplierRevenueConfig(config, testing)
}
