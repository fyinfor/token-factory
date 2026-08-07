package model

import (
	"errors"
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	SupplierRevenuePushModeEoraptor = "eoraptor"
	SupplierRevenuePushModeGeneric  = "generic"

	SupplierRevenueScheduleDaily  = "daily"
	SupplierRevenueScheduleHourly = "hourly"

	SupplierRevenueCurrencyUSD  = "USD"
	SupplierRevenueCurrencyCNY  = "CNY"
	SupplierRevenueCurrencyUSDT = "USDT"

	SupplierRevenueNegativeHold  = "hold"
	SupplierRevenueNegativeAllow = "allow"
	SupplierRevenueNegativeCarry = "carry"

	SupplierRevenueEnvironmentMock       = "mock"
	SupplierRevenueEnvironmentProduction = "production"

	SupplierRevenuePeriodPending = "pending"
	SupplierRevenuePeriodSettled = "settled"
	SupplierRevenuePeriodHeld    = "held"
	SupplierRevenuePeriodUnknown = "unknown"

	SupplierRevenueDeliveryCreated  = "created"
	SupplierRevenueDeliverySending  = "sending"
	SupplierRevenueDeliveryRetrying = "retrying"
	SupplierRevenueDeliverySuccess  = "success"
	SupplierRevenueDeliveryFailed   = "failed"
	SupplierRevenueDeliveryUnknown  = "unknown"

	SupplierRevenueDeliveryKindScheduled = "scheduled"
	SupplierRevenueDeliveryKindTest      = "test"
	SupplierRevenueDeliveryKindManual    = "manual"

	SupplierRevenueAttemptSuccess = "success"
	SupplierRevenueAttemptFailed  = "failed"
	SupplierRevenueAttemptUnknown = "unknown"
)

type SupplierRevenuePushConfig struct {
	ID                    int    `json:"id" gorm:"primaryKey;autoIncrement"`
	SupplierID            int    `json:"supplier_id" gorm:"not null;uniqueIndex"`
	Enabled               bool   `json:"enabled" gorm:"not null;default:false;index"`
	Mode                  string `json:"mode" gorm:"type:varchar(32);not null;default:eoraptor"`
	ScheduleType          string `json:"schedule_type" gorm:"type:varchar(16);not null;default:daily"`
	Timezone              string `json:"timezone" gorm:"type:varchar(64);not null;default:Asia/Shanghai"`
	DailyTime             string `json:"daily_time" gorm:"type:varchar(8);not null;default:01:00"`
	HourlyMinute          int    `json:"hourly_minute" gorm:"not null;default:5"`
	EffectiveAt           int64  `json:"effective_at" gorm:"type:bigint;not null;default:0"`
	Currency              string `json:"currency" gorm:"type:varchar(8);not null;default:USDT"`
	NegativePolicy        string `json:"negative_policy" gorm:"type:varchar(16);not null;default:hold"`
	RetryCount            int    `json:"retry_count" gorm:"not null;default:3"`
	RetryIntervalSeconds  int    `json:"retry_interval_seconds" gorm:"not null;default:300"`
	RetryBackoff          string `json:"retry_backoff" gorm:"type:varchar(16);not null;default:fixed"`
	TimeoutSeconds        int    `json:"timeout_seconds" gorm:"not null;default:15"`
	Environment           string `json:"environment" gorm:"type:varchar(16);not null;default:mock"`
	Endpoint              string `json:"endpoint" gorm:"type:varchar(1024);not null;default:''"`
	MockEndpoint          string `json:"mock_endpoint" gorm:"type:varchar(1024);not null;default:''"`
	PrivateKeyPEM         string `json:"-" gorm:"column:private_key_ciphertext;type:text"`
	PrivateKeyFingerprint string `json:"private_key_fingerprint" gorm:"type:varchar(128);not null;default:''"`
	HTTPMethod            string `json:"http_method" gorm:"type:varchar(16);not null;default:POST"`
	ContentType           string `json:"content_type" gorm:"type:varchar(64);not null;default:application/json"`
	HeadersJSON           string `json:"headers_json" gorm:"type:text"`
	BodyTemplate          string `json:"body_template" gorm:"type:text"`
	SuccessHTTPStatus     int    `json:"success_http_status" gorm:"not null;default:200"`
	SuccessCodePath       string `json:"success_code_path" gorm:"type:varchar(128);not null;default:''"`
	SuccessCodeValue      string `json:"success_code_value" gorm:"type:varchar(128);not null;default:''"`
	SuccessTypePath       string `json:"success_type_path" gorm:"type:varchar(128);not null;default:''"`
	SuccessTypeValue      string `json:"success_type_value" gorm:"type:varchar(128);not null;default:''"`
	SuccessAmountPath     string `json:"success_amount_path" gorm:"type:varchar(128);not null;default:''"`
	CallbackConfigJSON    string `json:"callback_config_json" gorm:"type:text"`
	CreatedBy             int    `json:"created_by" gorm:"not null;default:0"`
	UpdatedBy             int    `json:"updated_by" gorm:"not null;default:0"`
	CreatedAt             int64  `json:"created_at" gorm:"autoCreateTime;bigint;index"`
	UpdatedAt             int64  `json:"updated_at" gorm:"autoUpdateTime;bigint"`
}

func (SupplierRevenuePushConfig) TableName() string { return "supplier_revenue_push_configs" }

type SupplierRevenuePeriod struct {
	ID                int    `json:"id" gorm:"primaryKey;autoIncrement"`
	SupplierID        int    `json:"supplier_id" gorm:"not null;uniqueIndex:idx_supplier_revenue_period,priority:1;index"`
	ScheduleType      string `json:"schedule_type" gorm:"type:varchar(16);not null;uniqueIndex:idx_supplier_revenue_period,priority:2"`
	PeriodStart       int64  `json:"period_start" gorm:"type:bigint;not null;uniqueIndex:idx_supplier_revenue_period,priority:3;index"`
	PeriodEnd         int64  `json:"period_end" gorm:"type:bigint;not null;uniqueIndex:idx_supplier_revenue_period,priority:4;index"`
	RawQuota          int64  `json:"raw_quota" gorm:"type:bigint;not null;default:0"`
	RawAmount         string `json:"raw_amount" gorm:"type:text;not null"`
	Amount            string `json:"amount" gorm:"type:varchar(64);not null"`
	Currency          string `json:"currency" gorm:"type:varchar(8);not null"`
	ExchangeRate      string `json:"exchange_rate" gorm:"type:varchar(64);not null"`
	Status            string `json:"status" gorm:"type:varchar(16);not null;index"`
	LastDeliveryID    int    `json:"last_delivery_id" gorm:"not null;default:0;index"`
	SettledDeliveryID int    `json:"settled_delivery_id" gorm:"not null;default:0;index"`
	CreatedAt         int64  `json:"created_at" gorm:"autoCreateTime;bigint;index"`
	UpdatedAt         int64  `json:"updated_at" gorm:"autoUpdateTime;bigint"`
}

func (SupplierRevenuePeriod) TableName() string { return "supplier_revenue_periods" }

type SupplierRevenueDelivery struct {
	ID                 int    `json:"id" gorm:"primaryKey;autoIncrement"`
	SupplierID         int    `json:"supplier_id" gorm:"not null;index"`
	BatchNo            string `json:"batch_no" gorm:"type:varchar(64);not null;uniqueIndex"`
	Kind               string `json:"kind" gorm:"type:varchar(16);not null;default:scheduled"`
	PeriodStart        int64  `json:"period_start" gorm:"type:bigint;not null;default:0;index"`
	PeriodEnd          int64  `json:"period_end" gorm:"type:bigint;not null;default:0;index"`
	PeriodIDsJSON      string `json:"period_ids_json" gorm:"type:text"`
	RawQuota           int64  `json:"raw_quota" gorm:"type:bigint;not null;default:0"`
	Amount             string `json:"amount" gorm:"type:varchar(64);not null"`
	Currency           string `json:"currency" gorm:"type:varchar(8);not null"`
	Status             string `json:"status" gorm:"type:varchar(16);not null;index"`
	AttemptCount       int    `json:"attempt_count" gorm:"not null;default:0"`
	MaxAttempts        int    `json:"max_attempts" gorm:"not null;default:1"`
	NextRetryAt        int64  `json:"next_retry_at" gorm:"type:bigint;not null;default:0;index"`
	LastError          string `json:"last_error" gorm:"type:text"`
	Remark             string `json:"remark" gorm:"type:varchar(500);not null;default:''"`
	EndpointSnapshot   string `json:"endpoint_snapshot" gorm:"type:varchar(1024);not null;default:''"`
	ConfigSnapshotJSON string `json:"config_snapshot_json" gorm:"type:text"`
	CompletedAt        int64  `json:"completed_at" gorm:"type:bigint;not null;default:0"`
	CreatedAt          int64  `json:"created_at" gorm:"autoCreateTime;bigint;index"`
	UpdatedAt          int64  `json:"updated_at" gorm:"autoUpdateTime;bigint"`
}

func (SupplierRevenueDelivery) TableName() string { return "supplier_revenue_deliveries" }

type SupplierRevenueAttempt struct {
	ID           int    `json:"id" gorm:"primaryKey;autoIncrement"`
	DeliveryID   int    `json:"delivery_id" gorm:"not null;index"`
	AttemptNo    int    `json:"attempt_no" gorm:"not null"`
	Endpoint     string `json:"endpoint" gorm:"type:varchar(1024);not null"`
	HTTPMethod   string `json:"http_method" gorm:"type:varchar(16);not null"`
	RequestBody  string `json:"request_body" gorm:"type:text"`
	HTTPStatus   int    `json:"http_status" gorm:"not null;default:0"`
	ResponseBody string `json:"response_body" gorm:"type:text"`
	ErrorMessage string `json:"error_message" gorm:"type:text"`
	Outcome      string `json:"outcome" gorm:"type:varchar(16);not null;index"`
	DurationMs   int64  `json:"duration_ms" gorm:"type:bigint;not null;default:0"`
	RequestedAt  int64  `json:"requested_at" gorm:"type:bigint;not null;index"`
	CreatedAt    int64  `json:"created_at" gorm:"autoCreateTime;bigint"`
}

func (SupplierRevenueAttempt) TableName() string { return "supplier_revenue_attempt_logs" }

func GetSupplierRevenuePushConfig(supplierID int) (*SupplierRevenuePushConfig, error) {
	var config SupplierRevenuePushConfig
	err := DB.Where("supplier_id = ?", supplierID).First(&config).Error
	return &config, err
}

func SaveSupplierRevenuePushConfig(config *SupplierRevenuePushConfig) error {
	if config == nil || config.SupplierID <= 0 {
		return errors.New("invalid supplier revenue push config")
	}
	var existing SupplierRevenuePushConfig
	err := DB.Where("supplier_id = ?", config.SupplierID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return DB.Create(config).Error
	}
	if err != nil {
		return err
	}
	config.ID = existing.ID
	config.CreatedAt = existing.CreatedAt
	return DB.Save(config).Error
}

func ListEnabledSupplierRevenuePushConfigs() ([]*SupplierRevenuePushConfig, error) {
	var configs []*SupplierRevenuePushConfig
	err := DB.Where("enabled = ?", true).Order("id asc").Find(&configs).Error
	return configs, err
}

func GetSupplierRevenueChannelIDs(supplierID int) ([]int, error) {
	app, err := GetSupplierByID(supplierID)
	if err != nil {
		return nil, err
	}
	var ids []int
	err = DB.Model(&Channel{}).
		Distinct("id").
		Where("owner_user_id = ? OR supplier_application_id = ?", app.ApplicantUserID, supplierID).
		Pluck("id", &ids).Error
	return ids, err
}

// SumSupplierRevenueQuota returns supplier revenue movement for a period. Consume
// is positive revenue and refund is a negative adjustment on the day it occurs.
func SumSupplierRevenueQuota(channelIDs []int, periodStart, periodEnd int64) (int64, error) {
	if len(channelIDs) == 0 {
		return 0, nil
	}
	var row struct {
		RawQuota int64 `gorm:"column:raw_quota"`
	}
	err := LOG_DB.Table("logs").
		Select("COALESCE(SUM(CASE WHEN type = ? THEN quota WHEN type = ? THEN -quota ELSE 0 END), 0) AS raw_quota", LogTypeConsume, LogTypeRefund).
		Where("channel_id IN ?", channelIDs).
		Where("type IN ?", []int{LogTypeConsume, LogTypeRefund}).
		Where("created_at >= ? AND created_at < ?", periodStart, periodEnd).
		Scan(&row).Error
	return row.RawQuota, err
}

func LatestSupplierRevenuePeriodEnd(supplierID int, scheduleType string) (int64, error) {
	var row struct {
		PeriodEnd int64 `gorm:"column:period_end"`
	}
	err := DB.Model(&SupplierRevenuePeriod{}).
		Select("COALESCE(MAX(period_end), 0) AS period_end").
		Where("supplier_id = ? AND schedule_type = ?", supplierID, scheduleType).
		Scan(&row).Error
	return row.PeriodEnd, err
}

func CreateSupplierRevenuePeriod(period *SupplierRevenuePeriod) (bool, error) {
	if period == nil {
		return false, errors.New("nil supplier revenue period")
	}
	result := DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "supplier_id"}, {Name: "schedule_type"}, {Name: "period_start"}, {Name: "period_end"}},
		DoNothing: true,
	}).Create(period)
	return result.RowsAffected > 0, result.Error
}

func ListSupplierRevenuePeriodsByStatus(supplierID int, statuses []string) ([]*SupplierRevenuePeriod, error) {
	var periods []*SupplierRevenuePeriod
	query := DB.Where("supplier_id = ?", supplierID)
	if len(statuses) > 0 {
		query = query.Where("status IN ?", statuses)
	}
	err := query.Order("period_start asc, id asc").Find(&periods).Error
	return periods, err
}

func CreateSupplierRevenueDelivery(delivery *SupplierRevenueDelivery, periods []*SupplierRevenuePeriod) error {
	if delivery == nil {
		return errors.New("nil supplier revenue delivery")
	}
	periodIDs := make([]int, 0, len(periods))
	for _, period := range periods {
		if period != nil && period.ID > 0 {
			periodIDs = append(periodIDs, period.ID)
		}
	}
	encoded, err := common.Marshal(periodIDs)
	if err != nil {
		return err
	}
	delivery.PeriodIDsJSON = string(encoded)
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(delivery).Error; err != nil {
			return err
		}
		if len(periodIDs) == 0 {
			return nil
		}
		return tx.Model(&SupplierRevenuePeriod{}).
			Where("id IN ?", periodIDs).
			Updates(map[string]any{"last_delivery_id": delivery.ID, "updated_at": time.Now().Unix()}).Error
	})
}

func GetSupplierRevenueDelivery(deliveryID int) (*SupplierRevenueDelivery, error) {
	var delivery SupplierRevenueDelivery
	err := DB.First(&delivery, deliveryID).Error
	return &delivery, err
}

func ClaimSupplierRevenueDelivery(deliveryID int) (bool, error) {
	result := DB.Model(&SupplierRevenueDelivery{}).
		Where("id = ? AND status IN ?", deliveryID, []string{SupplierRevenueDeliveryCreated, SupplierRevenueDeliveryRetrying}).
		Updates(map[string]any{"status": SupplierRevenueDeliverySending, "updated_at": time.Now().Unix()})
	return result.RowsAffected > 0, result.Error
}

func CreateSupplierRevenueAttempt(attempt *SupplierRevenueAttempt) error {
	return DB.Create(attempt).Error
}

func MarkSupplierRevenueDeliverySuccess(deliveryID int, attemptCount int) error {
	now := time.Now().Unix()
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&SupplierRevenueDelivery{}).Where("id = ?", deliveryID).Updates(map[string]any{
			"status": SupplierRevenueDeliverySuccess, "attempt_count": attemptCount,
			"next_retry_at": 0, "last_error": "", "completed_at": now, "updated_at": now,
		}).Error; err != nil {
			return err
		}
		return tx.Model(&SupplierRevenuePeriod{}).
			Where("last_delivery_id = ? AND status IN ?", deliveryID, []string{SupplierRevenuePeriodPending, SupplierRevenuePeriodUnknown}).
			Updates(map[string]any{
				"status": SupplierRevenuePeriodSettled, "settled_delivery_id": deliveryID, "updated_at": now,
			}).Error
	})
}

func MarkSupplierRevenueDeliveryRetry(deliveryID int, attemptCount int, nextRetryAt int64, message string) error {
	return DB.Model(&SupplierRevenueDelivery{}).Where("id = ?", deliveryID).Updates(map[string]any{
		"status": SupplierRevenueDeliveryRetrying, "attempt_count": attemptCount,
		"next_retry_at": nextRetryAt, "last_error": message, "updated_at": time.Now().Unix(),
	}).Error
}

func MarkSupplierRevenueDeliveryFailed(deliveryID int, attemptCount int, message string) error {
	now := time.Now().Unix()
	return DB.Model(&SupplierRevenueDelivery{}).Where("id = ?", deliveryID).Updates(map[string]any{
		"status": SupplierRevenueDeliveryFailed, "attempt_count": attemptCount,
		"next_retry_at": 0, "last_error": message, "completed_at": now, "updated_at": now,
	}).Error
}

func MarkSupplierRevenueDeliveryUnknown(deliveryID int, attemptCount int, message string) error {
	now := time.Now().Unix()
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&SupplierRevenueDelivery{}).Where("id = ?", deliveryID).Updates(map[string]any{
			"status": SupplierRevenueDeliveryUnknown, "attempt_count": attemptCount,
			"next_retry_at": 0, "last_error": message, "completed_at": now, "updated_at": now,
		}).Error; err != nil {
			return err
		}
		return tx.Model(&SupplierRevenuePeriod{}).
			Where("last_delivery_id = ? AND status = ?", deliveryID, SupplierRevenuePeriodPending).
			Updates(map[string]any{"status": SupplierRevenuePeriodUnknown, "updated_at": now}).Error
	})
}

func ListRetryableSupplierRevenueDeliveries(now int64, limit int) ([]*SupplierRevenueDelivery, error) {
	if limit <= 0 {
		limit = 100
	}
	var deliveries []*SupplierRevenueDelivery
	enabledSuppliers := DB.Model(&SupplierRevenuePushConfig{}).
		Select("supplier_id").
		Where("enabled = ?", true)
	err := DB.Where("status = ? AND kind != ? AND next_retry_at <= ? AND supplier_id IN (?)", SupplierRevenueDeliveryRetrying, SupplierRevenueDeliveryKindManual, now, enabledSuppliers).
		Order("next_retry_at asc, id asc").Limit(limit).Find(&deliveries).Error
	return deliveries, err
}

func FinalizeRetryingManualSupplierRevenueDeliveries() error {
	now := time.Now().Unix()
	return DB.Model(&SupplierRevenueDelivery{}).
		Where("kind = ? AND status = ?", SupplierRevenueDeliveryKindManual, SupplierRevenueDeliveryRetrying).
		Updates(map[string]any{
			"status":        SupplierRevenueDeliveryFailed,
			"next_retry_at": 0,
			"completed_at":  now,
			"updated_at":    now,
		}).Error
}

func HasActiveOrUnknownSupplierRevenueDelivery(supplierID int) (bool, error) {
	var count int64
	err := DB.Model(&SupplierRevenueDelivery{}).
		Where("supplier_id = ? AND kind != ? AND status IN ?", supplierID, SupplierRevenueDeliveryKindTest,
			[]string{SupplierRevenueDeliveryCreated, SupplierRevenueDeliverySending, SupplierRevenueDeliveryRetrying, SupplierRevenueDeliveryUnknown}).
		Count(&count).Error
	return count > 0, err
}

func ListSupplierRevenueDeliveries(supplierID, startIdx, num int) ([]*SupplierRevenueDelivery, int64, error) {
	if num <= 0 {
		num = 20
	}
	var deliveries []*SupplierRevenueDelivery
	var total int64
	query := DB.Model(&SupplierRevenueDelivery{}).Where("supplier_id = ?", supplierID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Order("id desc").Limit(num).Offset(startIdx).Find(&deliveries).Error
	return deliveries, total, err
}

func ListSupplierRevenueAttempts(deliveryID int) ([]*SupplierRevenueAttempt, error) {
	var attempts []*SupplierRevenueAttempt
	err := DB.Where("delivery_id = ?", deliveryID).Order("attempt_no asc, id asc").Find(&attempts).Error
	return attempts, err
}

func ResolveUnknownSupplierRevenueDelivery(deliveryID int, settled bool) error {
	delivery, err := GetSupplierRevenueDelivery(deliveryID)
	if err != nil {
		return err
	}
	if delivery.Status != SupplierRevenueDeliveryUnknown {
		return fmt.Errorf("delivery status is %s, not unknown", delivery.Status)
	}
	if settled {
		return MarkSupplierRevenueDeliverySuccess(deliveryID, delivery.AttemptCount)
	}
	now := time.Now().Unix()
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&SupplierRevenueDelivery{}).Where("id = ?", deliveryID).Updates(map[string]any{
			"status": SupplierRevenueDeliveryFailed, "last_error": "管理员确认远端未入账", "completed_at": now, "updated_at": now,
		}).Error; err != nil {
			return err
		}
		return tx.Model(&SupplierRevenuePeriod{}).
			Where("last_delivery_id = ? AND status = ?", deliveryID, SupplierRevenuePeriodUnknown).
			Updates(map[string]any{"status": SupplierRevenuePeriodPending, "last_delivery_id": 0, "updated_at": now}).Error
	})
}
