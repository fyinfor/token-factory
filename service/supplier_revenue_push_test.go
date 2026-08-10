package service

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func testSupplierRevenuePrivateKey(t *testing.T) (*rsa.PrivateKey, string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	encoded := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	return key, string(encoded)
}

func TestSendEoraptorSupplierRevenueSignatureAndResponse(t *testing.T) {
	privateKey, privateKeyPEM := testSupplierRevenuePrivateKey(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.NoError(t, r.ParseMultipartForm(1<<20))
		number := r.FormValue("number")
		timestamp := r.FormValue("timestamp")
		signature, err := base64.StdEncoding.DecodeString(r.FormValue("sign"))
		require.NoError(t, err)
		source := fmt.Sprintf("number=%s&timestamp=%s", number, timestamp)
		digest := sha256.Sum256([]byte(source))
		require.NoError(t, rsa.VerifyPKCS1v15(&privateKey.PublicKey, crypto.SHA256, digest[:], signature))
		require.Equal(t, "100.000000", number)
		require.JSONEq(t, `{"callback_url":"https://example.com/callback"}`, r.FormValue("callback"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":200,"msg":"同步成功","result":{"number":"100"},"type":"success"}`))
	}))
	defer server.Close()

	config := DefaultSupplierRevenuePushConfig(1)
	config.Mode = model.SupplierRevenuePushModeEoraptor
	config.ContentType = "multipart/form-data"
	config.BodyTemplate = `{"number":"{{number}}","timestamp":"{{timestamp}}","sign":"{{sign}}","callback":{{callback_config}}}`
	config.CallbackConfigJSON = `{"callback_url":"https://example.com/callback"}`
	delivery := &model.SupplierRevenueDelivery{SupplierID: 1, Amount: "100.000000", Currency: "USDT"}
	result := sendEoraptorSupplierRevenue(context.Background(), config, delivery, privateKeyPEM, server.URL)
	require.Equal(t, model.SupplierRevenueAttemptSuccess, result.Outcome)
	require.Equal(t, http.StatusOK, result.HTTPStatus)
}

func TestSendEoraptorSupplierRevenueJSONFormat(t *testing.T) {
	privateKey, privateKeyPEM := testSupplierRevenuePrivateKey(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "application/json", r.Header.Get("Content-Type"))
		var payload map[string]string
		require.NoError(t, common.DecodeJson(r.Body, &payload))
		signature, err := base64.StdEncoding.DecodeString(payload["sign"])
		require.NoError(t, err)
		source := fmt.Sprintf("number=%s&timestamp=%s", payload["number"], payload["timestamp"])
		digest := sha256.Sum256([]byte(source))
		require.NoError(t, rsa.VerifyPKCS1v15(&privateKey.PublicKey, crypto.SHA256, digest[:], signature))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":200,"result":{"number":"1.250000"},"type":"success"}`))
	}))
	defer server.Close()

	config := DefaultSupplierRevenuePushConfig(1)
	config.Mode = model.SupplierRevenuePushModeEoraptor
	config.ContentType = "application/json"
	config.BodyTemplate = SupplierRevenueEoraptorDefaultBodyTemplate
	delivery := &model.SupplierRevenueDelivery{SupplierID: 1, Amount: "1.250000", Currency: "USDT"}
	result := sendEoraptorSupplierRevenue(context.Background(), config, delivery, privateKeyPEM, server.URL)
	require.Equal(t, model.SupplierRevenueAttemptSuccess, result.Outcome)
	require.NotContains(t, result.RequestBody, "PRIVATE KEY")
	require.Contains(t, result.RequestBody, `"sign":"[REDACTED]"`)
}

func TestSupplierRevenueAmountUsesSingleSixDigitCeiling(t *testing.T) {
	originalQuotaPerUnit := common.QuotaPerUnit
	common.QuotaPerUnit = 3
	t.Cleanup(func() { common.QuotaPerUnit = originalQuotaPerUnit })

	config := DefaultSupplierRevenuePushConfig(1)
	config.Currency = model.SupplierRevenueCurrencyUSD
	raw, rounded, rate := supplierRevenueAmount(1, config)
	require.True(t, raw.Equal(decimal.RequireFromString("0.3333333333333333")))
	require.Equal(t, "0.333334", rounded.StringFixed(6))
	require.Equal(t, "1", rate.String())

	_, negativeRounded, _ := supplierRevenueAmount(-1, config)
	require.Equal(t, "-0.333334", negativeRounded.StringFixed(6))
}

func TestNextCompleteSupplierRevenuePeriodStart(t *testing.T) {
	location, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	effective := time.Date(2026, 8, 7, 15, 23, 0, 0, location)

	daily := nextCompleteSupplierRevenuePeriodStart(effective, model.SupplierRevenueScheduleDaily)
	require.Equal(t, time.Date(2026, 8, 8, 0, 0, 0, 0, location), daily)
	hourly := nextCompleteSupplierRevenuePeriodStart(effective, model.SupplierRevenueScheduleHourly)
	require.Equal(t, time.Date(2026, 8, 7, 16, 0, 0, 0, location), hourly)
}

func TestValidateGenericSupplierRevenueResponseUsesDecimalAmountEquality(t *testing.T) {
	config := DefaultSupplierRevenuePushConfig(1)
	config.Mode = model.SupplierRevenuePushModeGeneric
	config.SuccessCodePath = "code"
	config.SuccessCodeValue = "200"
	config.SuccessTypePath = "type"
	config.SuccessTypeValue = "success"
	config.SuccessAmountPath = "result.number"
	err := validateGenericSupplierRevenueResponse(
		config,
		"100.000000",
		http.StatusOK,
		`{"code":200,"type":"success","result":{"number":"100"}}`,
	)
	require.NoError(t, err)
}

func TestValidateEoraptorSupplierRevenueConfigRequiresSignedTemplate(t *testing.T) {
	config := DefaultSupplierRevenuePushConfig(1)
	config.Mode = model.SupplierRevenuePushModeEoraptor
	config.Endpoint = SupplierRevenueEoraptorProductionEndpoint
	config.BodyTemplate = `{"number":"{{number}}","timestamp":"{{timestamp}}"}`
	err := validateSupplierRevenueConfig(config, true)
	require.ErrorContains(t, err, "sign")
}

func TestValidateEoraptorSupplierRevenueConfigAllowsHourlyAndCNY(t *testing.T) {
	config := DefaultSupplierRevenuePushConfig(1)
	config.Mode = model.SupplierRevenuePushModeEoraptor
	config.Endpoint = SupplierRevenueEoraptorProductionEndpoint
	config.BodyTemplate = SupplierRevenueEoraptorDefaultBodyTemplate
	config.ScheduleType = model.SupplierRevenueScheduleHourly
	config.Currency = model.SupplierRevenueCurrencyCNY
	require.NoError(t, validateSupplierRevenueConfig(config, true))
}

func TestResolveSupplierRevenueEndpointUsesConfiguredMockAddress(t *testing.T) {
	config := DefaultSupplierRevenuePushConfig(1)
	config.Mode = model.SupplierRevenuePushModeEoraptor
	config.Endpoint = SupplierRevenueEoraptorProductionEndpoint
	config.BodyTemplate = SupplierRevenueEoraptorDefaultBodyTemplate
	config.Environment = model.SupplierRevenueEnvironmentProduction
	config.MockEndpoint = "https://mock.example.com/supplier-revenue"

	endpoint, err := resolveSupplierRevenueEndpoint(config, true)
	require.NoError(t, err)
	require.Equal(t, config.MockEndpoint, endpoint)

	endpoint, err = resolveSupplierRevenueEndpoint(config, false)
	require.NoError(t, err)
	require.Equal(t, config.Endpoint, endpoint)
}

func TestManualSupplierRevenuePushCreatesIndependentDelivery(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:supplier_revenue_manual_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.SupplierRevenuePushConfig{}, &model.SupplierRevenueDelivery{}, &model.SupplierRevenuePeriod{}, &model.SupplierRevenueAttempt{}))
	originalDB := model.DB
	model.DB = db
	t.Cleanup(func() { model.DB = originalDB })

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := DefaultSupplierRevenuePushConfig(7)
	config.Enabled = true
	config.Mode = model.SupplierRevenuePushModeGeneric
	config.Currency = model.SupplierRevenueCurrencyUSD
	config.Endpoint = server.URL
	config.ContentType = "application/json"
	config.BodyTemplate = `{"number":"{{number}}"}`
	config.SuccessCodePath = ""
	config.SuccessCodeValue = ""
	config.SuccessTypePath = ""
	config.SuccessTypeValue = ""
	config.SuccessAmountPath = ""
	require.NoError(t, model.SaveSupplierRevenuePushConfig(config))

	delivery, err := ManualSupplierRevenuePush(context.Background(), config.SupplierID, "12.3456789", "manual finance adjustment")
	require.NoError(t, err)
	require.Equal(t, model.SupplierRevenueDeliveryKindManual, delivery.Kind)
	require.Equal(t, model.SupplierRevenueDeliverySuccess, delivery.Status, delivery.LastError)
	require.Equal(t, 1, delivery.MaxAttempts)
	require.Equal(t, "12.345679", delivery.Amount)
	require.Equal(t, "manual finance adjustment", delivery.Remark)
	require.Equal(t, "[]", delivery.PeriodIDsJSON)
}

func TestManualSupplierRevenuePushFailureDoesNotRetry(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:supplier_revenue_manual_failure_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.SupplierRevenuePushConfig{}, &model.SupplierRevenueDelivery{}, &model.SupplierRevenuePeriod{}, &model.SupplierRevenueAttempt{}))
	originalDB := model.DB
	model.DB = db
	t.Cleanup(func() { model.DB = originalDB })

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	config := DefaultSupplierRevenuePushConfig(8)
	config.Enabled = true
	config.Mode = model.SupplierRevenuePushModeGeneric
	config.Endpoint = server.URL
	config.RetryCount = 3
	require.NoError(t, model.SaveSupplierRevenuePushConfig(config))

	delivery, err := ManualSupplierRevenuePush(context.Background(), config.SupplierID, "1", "no retry")
	require.NoError(t, err)
	require.Equal(t, model.SupplierRevenueDeliveryFailed, delivery.Status)
	require.Equal(t, 1, delivery.AttemptCount)
	require.Equal(t, 1, delivery.MaxAttempts)
	require.Zero(t, delivery.NextRetryAt)
}

func TestAttemptSupplierRevenueDeliveryMarksFailedWhenConfigIsMissing(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:supplier_revenue_service_%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.SupplierRevenuePushConfig{}, &model.SupplierRevenueDelivery{}, &model.SupplierRevenuePeriod{}, &model.SupplierRevenueAttempt{}))
	originalDB := model.DB
	model.DB = db
	t.Cleanup(func() { model.DB = originalDB })

	delivery := &model.SupplierRevenueDelivery{
		SupplierID: 99, BatchNo: "SRP-MISSING-CONFIG", Amount: "1.000000", Currency: "USD",
		Status: model.SupplierRevenueDeliveryCreated, MaxAttempts: 4,
	}
	require.NoError(t, model.DB.Create(delivery).Error)

	require.Error(t, attemptSupplierRevenueDelivery(context.Background(), delivery.ID))
	stored, err := model.GetSupplierRevenueDelivery(delivery.ID)
	require.NoError(t, err)
	require.Equal(t, model.SupplierRevenueDeliveryFailed, stored.Status)
	require.Contains(t, stored.LastError, "读取供应商推送配置失败")
}
