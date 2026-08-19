package model

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"gorm.io/gorm"
)

const DefaultChannelModelPricingTimezone = "Asia/Shanghai"

const (
	ChannelModelPricePlanModePrice = "price"
	ChannelModelPricePlanModeRate  = "rate"
)

var (
	ErrChannelModelPricePlanInUse   = errors.New("price plan is used by an enabled schedule")
	ErrChannelModelScheduleConflict = errors.New("pricing schedule overlaps another enabled schedule")
)

// ChannelModelScheduleConflictError carries the enabled schedule that conflicts
// with a proposed rule, so the API can return an actionable validation message.
type ChannelModelScheduleConflictError struct {
	ConflictingSchedule ChannelModelPriceSchedule
}

func (e *ChannelModelScheduleConflictError) Error() string {
	if e == nil {
		return ErrChannelModelScheduleConflict.Error()
	}
	return fmt.Sprintf("%s: %s", ErrChannelModelScheduleConflict, e.ConflictingSchedule.Name)
}

func (e *ChannelModelScheduleConflictError) Unwrap() error {
	return ErrChannelModelScheduleConflict
}

// ChannelModelPricePlanPayload is a complete, independent channel-model pricing snapshot.
// Pointer scalar fields preserve the difference between "unset" and an explicit zero.
type ChannelModelPricePlanPayload struct {
	Mode                    string                            `json:"Mode,omitempty"`
	ModelPrice              *float64                          `json:"ModelPrice,omitempty"`
	ModelRatio              *float64                          `json:"ModelRatio,omitempty"`
	CompletionRatio         *float64                          `json:"CompletionRatio,omitempty"`
	CacheRatio              *float64                          `json:"CacheRatio,omitempty"`
	CreateCacheRatio        *float64                          `json:"CreateCacheRatio,omitempty"`
	ImageRatio              *float64                          `json:"ImageRatio,omitempty"`
	AudioRatio              *float64                          `json:"AudioRatio,omitempty"`
	AudioCompletionRatio    *float64                          `json:"AudioCompletionRatio,omitempty"`
	VideoRatio              *float64                          `json:"VideoRatio,omitempty"`
	VideoCompletionRatio    *float64                          `json:"VideoCompletionRatio,omitempty"`
	VideoPrice              *float64                          `json:"VideoPrice,omitempty"`
	VideoPricingRules       *ratio_setting.VideoPricingRules  `json:"VideoPricingRules,omitempty"`
	ImagePrice              *float64                          `json:"ImagePrice,omitempty"`
	ImagePricingRules       *ratio_setting.ImagePricingRules  `json:"ImagePricingRules,omitempty"`
	ASRPrice                *float64                          `json:"ASRPrice,omitempty"`
	ModelRequestTierPricing *ratio_setting.RequestTierPricing `json:"ModelRequestTierPricing,omitempty"`
	PriceDiscountPercent    *float64                          `json:"PriceDiscountPercent,omitempty"`
	OperatingCostPercent    *float64                          `json:"OperatingCostPercent,omitempty"`
	MarkupDiscountRate      *float64                          `json:"MarkupDiscountRate,omitempty"`
}

func (p ChannelModelPricePlanPayload) ResolvedMode() string {
	if strings.EqualFold(strings.TrimSpace(p.Mode), ChannelModelPricePlanModeRate) {
		return ChannelModelPricePlanModeRate
	}
	return ChannelModelPricePlanModePrice
}

func (p ChannelModelPricePlanPayload) HasIndependentPricing() bool {
	return p.ModelPrice != nil || p.ModelRatio != nil || p.CompletionRatio != nil ||
		p.CacheRatio != nil || p.CreateCacheRatio != nil || p.ImageRatio != nil ||
		p.AudioRatio != nil || p.AudioCompletionRatio != nil || p.VideoRatio != nil ||
		p.VideoCompletionRatio != nil || p.VideoPrice != nil || p.VideoPricingRules != nil ||
		p.ImagePrice != nil || p.ImagePricingRules != nil || p.ASRPrice != nil ||
		p.ModelRequestTierPricing != nil
}

func (p ChannelModelPricePlanPayload) HasRateOverrides() bool {
	return p.PriceDiscountPercent != nil && p.OperatingCostPercent != nil && p.MarkupDiscountRate != nil
}

func (p ChannelModelPricePlanPayload) HasAnyPricing() bool {
	if p.ResolvedMode() == ChannelModelPricePlanModeRate {
		return p.HasRateOverrides()
	}
	return p.HasIndependentPricing()
}

func (p *ChannelModelPricePlanPayload) NormalizeAndValidate() error {
	if p == nil {
		return errors.New("price plan payload is required")
	}
	p.Mode = p.ResolvedMode()
	if p.Mode != ChannelModelPricePlanModeRate {
		return errors.New("only dynamic rate plans are supported")
	}
	if !p.HasRateOverrides() {
		return errors.New("rate plan must contain cost discount, operating cost and markup discount")
	}
	for label, value := range map[string]*float64{
		"price discount percent": p.PriceDiscountPercent,
		"operating cost percent": p.OperatingCostPercent,
		"markup discount rate":   p.MarkupDiscountRate,
	} {
		if value == nil || *value < 0 || *value > 1000 {
			return fmt.Errorf("%s must be between 0 and 1000", label)
		}
	}
	return nil
}

func (p ChannelModelPricePlanPayload) MarshalJSONString() (string, error) {
	b, err := common.Marshal(p)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func ParseChannelModelPricePlanPayload(raw string) (ChannelModelPricePlanPayload, error) {
	var payload ChannelModelPricePlanPayload
	if strings.TrimSpace(raw) == "" {
		return payload, nil
	}
	if err := common.UnmarshalJsonStr(raw, &payload); err != nil {
		return ChannelModelPricePlanPayload{}, err
	}
	return payload, nil
}

type ChannelModelPricePlan struct {
	ID                    int       `json:"id" gorm:"primaryKey"`
	ChannelID             int       `json:"channel_id" gorm:"not null;index:idx_channel_model_plan_lookup,priority:1"`
	SupplierApplicationID int       `json:"supplier_application_id" gorm:"not null;default:0;index"`
	ModelName             string    `json:"model_name" gorm:"type:varchar(512);not null;index:idx_channel_model_plan_lookup,priority:2"`
	Name                  string    `json:"name" gorm:"type:varchar(128);not null"`
	PricePayload          string    `json:"-" gorm:"type:text;not null"`
	Version               int       `json:"version" gorm:"not null;default:1"`
	Enabled               bool      `json:"enabled" gorm:"not null;default:true;index"`
	CreatedByUserID       int       `json:"created_by_user_id" gorm:"not null;default:0"`
	UpdatedByUserID       int       `json:"updated_by_user_id" gorm:"not null;default:0"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

func (ChannelModelPricePlan) TableName() string { return "channel_model_price_plans" }

type ChannelModelPriceSchedule struct {
	ID                    int       `json:"id" gorm:"primaryKey"`
	ChannelID             int       `json:"channel_id" gorm:"not null;index:idx_channel_model_schedule_lookup,priority:1"`
	SupplierApplicationID int       `json:"supplier_application_id" gorm:"not null;default:0;index"`
	ModelName             string    `json:"model_name" gorm:"type:varchar(512);not null;index:idx_channel_model_schedule_lookup,priority:2"`
	PricePlanID           int       `json:"price_plan_id" gorm:"not null;index"`
	Name                  string    `json:"name" gorm:"type:varchar(128);not null"`
	Timezone              string    `json:"timezone" gorm:"type:varchar(64);not null;default:'Asia/Shanghai'"`
	Weekdays              int       `json:"weekdays" gorm:"not null"` // bit 0=Sunday ... bit 6=Saturday
	StartMinute           int       `json:"start_minute" gorm:"not null"`
	EndMinute             int       `json:"end_minute" gorm:"not null"`
	EffectiveFrom         string    `json:"effective_from" gorm:"type:varchar(10);not null;default:''"`
	EffectiveTo           string    `json:"effective_to" gorm:"type:varchar(10);not null;default:''"`
	Enabled               bool      `json:"enabled" gorm:"not null;default:true;index"`
	CreatedByUserID       int       `json:"created_by_user_id" gorm:"not null;default:0"`
	UpdatedByUserID       int       `json:"updated_by_user_id" gorm:"not null;default:0"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

func (ChannelModelPriceSchedule) TableName() string { return "channel_model_price_schedules" }

type ActiveChannelModelPricePlan struct {
	Plan      ChannelModelPricePlan
	Schedule  ChannelModelPriceSchedule
	Payload   ChannelModelPricePlanPayload
	MatchedAt time.Time
}

// PricingTimePricingPrice is the resolved price snapshot exposed by the
// public pricing API. It uses the same fallback rules as relay billing: the
// independent plan wins, while omitted child ratios fall back to the official
// global model values instead of the channel's regular price.
type PricingTimePricingPrice struct {
	ModelPrice           float64                   `json:"model_price"`
	ModelRatio           float64                   `json:"model_ratio"`
	CompletionRatio      float64                   `json:"completion_ratio"`
	CacheRatio           float64                   `json:"cache_ratio"`
	CreateCacheRatio     float64                   `json:"create_cache_ratio"`
	RequestTierPricing   any                       `json:"request_tier_pricing,omitempty"`
	QuotaType            int                       `json:"quota_type"`
	ImageRatio           float64                   `json:"image_ratio,omitempty"`
	AudioRatio           float64                   `json:"audio_ratio,omitempty"`
	AudioCompletionRatio float64                   `json:"audio_completion_ratio,omitempty"`
	VideoRatio           float64                   `json:"video_ratio,omitempty"`
	VideoCompletionRatio float64                   `json:"video_completion_ratio,omitempty"`
	VideoPrice           float64                   `json:"video_price,omitempty"`
	ImagePrice           float64                   `json:"image_price,omitempty"`
	ASRPrice             float64                   `json:"asr_price,omitempty"`
	VideoFlatClipHint    *VideoFlatClipPricingHint `json:"video_flat_clip_hint,omitempty"`
	ImagePerImageHint    *ImagePerImagePricingHint `json:"image_per_image_hint,omitempty"`
}

type PricingTimePricingRates struct {
	PriceDiscountPercent float64 `json:"price_discount_percent"`
	OperatingCostPercent float64 `json:"operating_cost_percent"`
	EffectiveCostPercent float64 `json:"effective_cost_percent"`
	MarkupDiscountRate   float64 `json:"markup_discount_rate"`
}

type PricingTimePricingSchedule struct {
	ID            int                      `json:"id"`
	Name          string                   `json:"name"`
	Weekdays      int                      `json:"weekdays"`
	StartMinute   int                      `json:"start_minute"`
	EndMinute     int                      `json:"end_minute"`
	EffectiveFrom string                   `json:"effective_from,omitempty"`
	EffectiveTo   string                   `json:"effective_to,omitempty"`
	Active        bool                     `json:"active"`
	PlanID        int                      `json:"plan_id"`
	PlanName      string                   `json:"plan_name"`
	PlanVersion   int                      `json:"plan_version"`
	Mode          string                   `json:"mode"`
	Pricing       PricingTimePricingPrice  `json:"pricing"`
	Rates         *PricingTimePricingRates `json:"rates,omitempty"`
	payload       ChannelModelPricePlanPayload
}

type PricingTimePricingInfo struct {
	HasSchedules       bool                         `json:"has_schedules"`
	Active             bool                         `json:"active"`
	Timezone           string                       `json:"timezone"`
	RegularPricing     PricingTimePricingPrice      `json:"regular_pricing"`
	RegularRates       PricingTimePricingRates      `json:"regular_rates"`
	ActiveScheduleID   int                          `json:"active_schedule_id,omitempty"`
	ActiveScheduleName string                       `json:"active_schedule_name,omitempty"`
	ActivePlanID       int                          `json:"active_plan_id,omitempty"`
	ActivePlanName     string                       `json:"active_plan_name,omitempty"`
	ActivePlanVersion  int                          `json:"active_plan_version,omitempty"`
	Schedules          []PricingTimePricingSchedule `json:"schedules"`
}

// ChannelModelRateRule is the channel editor's combined view of one dynamic
// rate plan and its schedule. The database keeps the two normalized records,
// while the management UI treats them as one rule.
type ChannelModelRateRule struct {
	ScheduleID           int     `json:"schedule_id"`
	PlanID               int     `json:"plan_id"`
	PlanVersion          int     `json:"plan_version"`
	ModelName            string  `json:"model_name"`
	Name                 string  `json:"name"`
	PriceDiscountPercent float64 `json:"price_discount_percent"`
	OperatingCostPercent float64 `json:"operating_cost_percent"`
	EffectiveCostPercent float64 `json:"effective_cost_percent"`
	MarkupDiscountRate   float64 `json:"markup_discount_rate"`
	Timezone             string  `json:"timezone"`
	Weekdays             int     `json:"weekdays"`
	StartMinute          int     `json:"start_minute"`
	EndMinute            int     `json:"end_minute"`
	EffectiveFrom        string  `json:"effective_from,omitempty"`
	EffectiveTo          string  `json:"effective_to,omitempty"`
	Enabled              bool    `json:"enabled"`
	Active               bool    `json:"active"`
}

type ChannelModelRateRuleMutation struct {
	ChannelID             int
	SupplierApplicationID int
	ModelNames            []string
	Name                  string
	PriceDiscountPercent  float64
	OperatingCostPercent  float64
	MarkupDiscountRate    float64
	Timezone              string
	Weekdays              int
	StartMinute           int
	EndMinute             int
	EffectiveFrom         string
	EffectiveTo           string
	Enabled               bool
	UserID                int
}

type cachedChannelModelTimePricing struct {
	plans     map[int]ChannelModelPricePlan
	payloads  map[int]ChannelModelPricePlanPayload
	schedules []ChannelModelPriceSchedule
}

var channelModelTimePricingCache = struct {
	sync.RWMutex
	items map[string]cachedChannelModelTimePricing
}{items: make(map[string]cachedChannelModelTimePricing)}

func channelModelTimePricingKey(channelID int, modelName string) string {
	return fmt.Sprintf("%d:%s", channelID, ratio_setting.FormatMatchingModelName(modelName))
}

func InvalidateChannelModelTimePricingCache(channelID int, modelName string) {
	channelModelTimePricingCache.Lock()
	delete(channelModelTimePricingCache.items, channelModelTimePricingKey(channelID, modelName))
	channelModelTimePricingCache.Unlock()
}

func ClearChannelModelTimePricingCache() {
	channelModelTimePricingCache.Lock()
	channelModelTimePricingCache.items = make(map[string]cachedChannelModelTimePricing)
	channelModelTimePricingCache.Unlock()
}

func normalizePricingTimezone(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		value = DefaultChannelModelPricingTimezone
	}
	if value != DefaultChannelModelPricingTimezone {
		return "", fmt.Errorf("pricing timezone must be %s", DefaultChannelModelPricingTimezone)
	}
	if _, err := time.LoadLocation(value); err != nil {
		return "", fmt.Errorf("invalid timezone: %s", value)
	}
	return value, nil
}

func normalizeEffectiveDate(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if _, err := time.Parse("2006-01-02", value); err != nil {
		return "", fmt.Errorf("invalid effective date: %s", value)
	}
	return value, nil
}

func ValidateChannelModelPriceSchedule(schedule *ChannelModelPriceSchedule) error {
	if schedule == nil {
		return errors.New("schedule is required")
	}
	if strings.TrimSpace(schedule.Name) == "" {
		return errors.New("schedule name is required")
	}
	if schedule.Weekdays <= 0 || schedule.Weekdays > 0x7f {
		return errors.New("at least one weekday is required")
	}
	if schedule.StartMinute < 0 || schedule.StartMinute >= 1440 {
		return errors.New("start minute must be between 0 and 1439")
	}
	if schedule.EndMinute <= 0 || schedule.EndMinute > 1440 {
		return errors.New("end minute must be between 1 and 1440")
	}
	if schedule.StartMinute == schedule.EndMinute {
		return errors.New("start time and end time cannot be equal")
	}
	tz, err := normalizePricingTimezone(schedule.Timezone)
	if err != nil {
		return err
	}
	schedule.Timezone = tz
	from, err := normalizeEffectiveDate(schedule.EffectiveFrom)
	if err != nil {
		return err
	}
	to, err := normalizeEffectiveDate(schedule.EffectiveTo)
	if err != nil {
		return err
	}
	if from != "" && to != "" && from > to {
		return errors.New("effective_from must not be after effective_to")
	}
	schedule.EffectiveFrom = from
	schedule.EffectiveTo = to
	return nil
}

func scheduleMatchesAt(schedule ChannelModelPriceSchedule, at time.Time) bool {
	if !schedule.Enabled {
		return false
	}
	location, err := time.LoadLocation(schedule.Timezone)
	if err != nil {
		return false
	}
	local := at.In(location)
	date := local.Format("2006-01-02")
	minute := local.Hour()*60 + local.Minute()
	weekday := local.Weekday()
	dateForWindow := date
	weekdayForWindow := weekday
	matched := false
	if schedule.StartMinute < schedule.EndMinute {
		matched = minute >= schedule.StartMinute && minute < schedule.EndMinute
	} else {
		matched = minute >= schedule.StartMinute || minute < schedule.EndMinute
		if minute < schedule.EndMinute {
			previous := local.AddDate(0, 0, -1)
			weekdayForWindow = previous.Weekday()
			dateForWindow = previous.Format("2006-01-02")
		}
	}
	if !matched || schedule.Weekdays&(1<<int(weekdayForWindow)) == 0 {
		return false
	}
	if schedule.EffectiveFrom != "" && dateForWindow < schedule.EffectiveFrom {
		return false
	}
	if schedule.EffectiveTo != "" && dateForWindow > schedule.EffectiveTo {
		return false
	}
	return true
}

type weeklyInterval struct{ start, end int }

func scheduleWeeklyIntervals(schedule ChannelModelPriceSchedule) []weeklyInterval {
	intervals := make([]weeklyInterval, 0, 8)
	for weekday := 0; weekday < 7; weekday++ {
		if schedule.Weekdays&(1<<weekday) == 0 {
			continue
		}
		base := weekday * 1440
		if schedule.StartMinute < schedule.EndMinute {
			intervals = append(intervals, weeklyInterval{base + schedule.StartMinute, base + schedule.EndMinute})
			continue
		}
		start := base + schedule.StartMinute
		end := base + 1440 + schedule.EndMinute
		if end <= 10080 {
			intervals = append(intervals, weeklyInterval{start, end})
		} else {
			intervals = append(intervals, weeklyInterval{start, 10080}, weeklyInterval{0, end - 10080})
		}
	}
	return intervals
}

func effectiveDateRangesOverlap(a, b ChannelModelPriceSchedule) bool {
	aFrom, aTo := a.EffectiveFrom, a.EffectiveTo
	bFrom, bTo := b.EffectiveFrom, b.EffectiveTo
	if aFrom == "" {
		aFrom = "0000-01-01"
	}
	if bFrom == "" {
		bFrom = "0000-01-01"
	}
	if aTo == "" {
		aTo = "9999-12-31"
	}
	if bTo == "" {
		bTo = "9999-12-31"
	}
	return aFrom <= bTo && bFrom <= aTo
}

func schedulesOverlap(a, b ChannelModelPriceSchedule) bool {
	if !a.Enabled || !b.Enabled || a.Timezone != b.Timezone || !effectiveDateRangesOverlap(a, b) {
		return false
	}
	for _, ai := range scheduleWeeklyIntervals(a) {
		for _, bi := range scheduleWeeklyIntervals(b) {
			if ai.start < bi.end && bi.start < ai.end {
				return true
			}
		}
	}
	return false
}

func isUsableChannelModelRatePlan(plan ChannelModelPricePlan) bool {
	if !plan.Enabled {
		return false
	}
	payload, err := ParseChannelModelPricePlanPayload(plan.PricePayload)
	return err == nil && payload.ResolvedMode() == ChannelModelPricePlanModeRate && payload.HasRateOverrides()
}

func ValidateChannelModelScheduleConflict(tx *gorm.DB, schedule ChannelModelPriceSchedule) error {
	if !schedule.Enabled {
		return nil
	}
	if tx == nil {
		tx = DB
	}
	var existing []ChannelModelPriceSchedule
	query := tx.Where("channel_id = ? AND model_name = ? AND enabled = ?", schedule.ChannelID, schedule.ModelName, true)
	if schedule.ID > 0 {
		query = query.Where("id <> ?", schedule.ID)
	}
	if err := query.Find(&existing).Error; err != nil {
		return err
	}
	planIDs := make([]int, 0, len(existing))
	for _, item := range existing {
		planIDs = append(planIDs, item.PricePlanID)
	}
	usablePlanIDs := make(map[int]struct{}, len(planIDs))
	if len(planIDs) > 0 {
		var plans []ChannelModelPricePlan
		if err := tx.Where("id IN ?", planIDs).Find(&plans).Error; err != nil {
			return err
		}
		for _, plan := range plans {
			if isUsableChannelModelRatePlan(plan) {
				usablePlanIDs[plan.ID] = struct{}{}
			}
		}
	}
	for _, item := range existing {
		if _, ok := usablePlanIDs[item.PricePlanID]; !ok {
			continue
		}
		if schedulesOverlap(schedule, item) {
			return &ChannelModelScheduleConflictError{ConflictingSchedule: item}
		}
	}
	return nil
}

func loadChannelModelTimePricing(channelID int, modelName string) (cachedChannelModelTimePricing, error) {
	key := channelModelTimePricingKey(channelID, modelName)
	channelModelTimePricingCache.RLock()
	if cached, ok := channelModelTimePricingCache.items[key]; ok {
		channelModelTimePricingCache.RUnlock()
		return cached, nil
	}
	channelModelTimePricingCache.RUnlock()

	normalized := ratio_setting.FormatMatchingModelName(modelName)
	var plans []ChannelModelPricePlan
	if err := DB.Where("channel_id = ? AND model_name = ? AND enabled = ?", channelID, normalized, true).Find(&plans).Error; err != nil {
		return cachedChannelModelTimePricing{}, err
	}
	var schedules []ChannelModelPriceSchedule
	if err := DB.Where("channel_id = ? AND model_name = ? AND enabled = ?", channelID, normalized, true).
		Order("id ASC").Find(&schedules).Error; err != nil {
		return cachedChannelModelTimePricing{}, err
	}
	cached := cachedChannelModelTimePricing{
		plans:     make(map[int]ChannelModelPricePlan, len(plans)),
		payloads:  make(map[int]ChannelModelPricePlanPayload, len(plans)),
		schedules: schedules,
	}
	for _, plan := range plans {
		payload, err := ParseChannelModelPricePlanPayload(plan.PricePayload)
		if err != nil || payload.ResolvedMode() != ChannelModelPricePlanModeRate || !payload.HasRateOverrides() {
			continue
		}
		cached.plans[plan.ID] = plan
		cached.payloads[plan.ID] = payload
	}
	channelModelTimePricingCache.Lock()
	channelModelTimePricingCache.items[key] = cached
	channelModelTimePricingCache.Unlock()
	return cached, nil
}

func ResolveActiveChannelModelPricePlan(channelID int, modelName string, at time.Time) (*ActiveChannelModelPricePlan, bool) {
	if DB == nil || channelID <= 0 || strings.TrimSpace(modelName) == "" {
		return nil, false
	}
	cached, err := loadChannelModelTimePricing(channelID, modelName)
	if err != nil {
		return nil, false
	}
	for _, schedule := range cached.schedules {
		if !scheduleMatchesAt(schedule, at) {
			continue
		}
		plan, ok := cached.plans[schedule.PricePlanID]
		if !ok {
			continue
		}
		payload := cached.payloads[plan.ID]
		return &ActiveChannelModelPricePlan{Plan: plan, Schedule: schedule, Payload: payload, MatchedAt: at}, true
	}
	return nil, false
}

func resolveChannelModelPricePlanDisplayPricing(modelName string, payload ChannelModelPricePlanPayload) PricingTimePricingPrice {
	if payload.ResolvedMode() == ChannelModelPricePlanModeRate {
		return PricingTimePricingPrice{}
	}
	globalRatio, _, _ := ratio_setting.GetModelRatio(modelName)
	globalCompletionRatio := ratio_setting.GetCompletionRatio(modelName)
	globalCacheRatio, ok := ratio_setting.GetCacheRatio(modelName)
	if !ok {
		globalCacheRatio = 1
	}
	globalCreateCacheRatio, ok := ratio_setting.GetCreateCacheRatio(modelName)
	if !ok {
		globalCreateCacheRatio = 1.25
	}
	globalImageRatio, _ := ratio_setting.GetImageRatio(modelName)
	globalAudioRatio := ratio_setting.GetAudioRatio(modelName)
	globalAudioCompletionRatio := ratio_setting.GetAudioCompletionRatio(modelName)

	resolved := PricingTimePricingPrice{
		ModelRatio:           globalRatio,
		CompletionRatio:      globalCompletionRatio,
		CacheRatio:           globalCacheRatio,
		CreateCacheRatio:     globalCreateCacheRatio,
		ImageRatio:           globalImageRatio,
		AudioRatio:           globalAudioRatio,
		AudioCompletionRatio: globalAudioCompletionRatio,
		VideoRatio:           ratio_setting.GetVideoRatio(modelName),
		VideoCompletionRatio: ratio_setting.GetVideoCompletionRatio(modelName),
	}
	if payload.ModelPrice != nil {
		resolved.ModelPrice = *payload.ModelPrice
		resolved.QuotaType = 1
	} else if payload.ModelRatio != nil {
		resolved.ModelRatio = *payload.ModelRatio
	}
	if payload.CompletionRatio != nil {
		resolved.CompletionRatio = *payload.CompletionRatio
	}
	if payload.CacheRatio != nil {
		resolved.CacheRatio = *payload.CacheRatio
	}
	if payload.CreateCacheRatio != nil {
		resolved.CreateCacheRatio = *payload.CreateCacheRatio
	}
	if payload.ImageRatio != nil {
		resolved.ImageRatio = *payload.ImageRatio
	}
	if payload.AudioRatio != nil {
		resolved.AudioRatio = *payload.AudioRatio
	}
	if payload.AudioCompletionRatio != nil {
		resolved.AudioCompletionRatio = *payload.AudioCompletionRatio
	}
	if payload.VideoRatio != nil {
		resolved.VideoRatio = *payload.VideoRatio
	}
	if payload.VideoCompletionRatio != nil {
		resolved.VideoCompletionRatio = *payload.VideoCompletionRatio
	}
	if payload.VideoPrice != nil {
		resolved.VideoPrice = *payload.VideoPrice
	}
	if payload.ImagePrice != nil {
		resolved.ImagePrice = *payload.ImagePrice
	}
	if payload.ASRPrice != nil {
		resolved.ASRPrice = *payload.ASRPrice
	}
	if payload.ModelRequestTierPricing != nil {
		resolved.RequestTierPricing = *payload.ModelRequestTierPricing
		if resolved.QuotaType != 1 {
			resolved.QuotaType = 3
		}
	} else if globalTier, ok := ratio_setting.GetModelRequestTierPricing(modelName); ok && len(globalTier.Tiers) > 0 {
		resolved.RequestTierPricing = globalTier
		if resolved.QuotaType != 1 {
			resolved.QuotaType = 3
		}
	}
	return resolved
}

func channelModelPricePlanDisplayRates(payload ChannelModelPricePlanPayload) *PricingTimePricingRates {
	if payload.ResolvedMode() != ChannelModelPricePlanModeRate || !payload.HasRateOverrides() {
		return nil
	}
	raw := *payload.PriceDiscountPercent
	operating := *payload.OperatingCostPercent
	return &PricingTimePricingRates{
		PriceDiscountPercent: raw,
		OperatingCostPercent: operating,
		EffectiveCostPercent: EffectiveCostPercent(raw, operating),
		MarkupDiscountRate:   clampChannelMarkupDiscountRate(*payload.MarkupDiscountRate),
	}
}

// LoadChannelModelTimePricingDisplay loads all enabled plans and schedules for
// the visible channels in two queries. The result is keyed by the same
// normalized channel/model key used by relay billing.
func LoadChannelModelTimePricingDisplay(channelIDs []int, at time.Time) (map[string]PricingTimePricingInfo, error) {
	result := make(map[string]PricingTimePricingInfo)
	if DB == nil || len(channelIDs) == 0 {
		return result, nil
	}
	deduped := make([]int, 0, len(channelIDs))
	seen := make(map[int]struct{}, len(channelIDs))
	for _, channelID := range channelIDs {
		if channelID <= 0 {
			continue
		}
		if _, ok := seen[channelID]; ok {
			continue
		}
		seen[channelID] = struct{}{}
		deduped = append(deduped, channelID)
	}
	if len(deduped) == 0 {
		return result, nil
	}

	var plans []ChannelModelPricePlan
	if err := DB.Where("channel_id IN ? AND enabled = ?", deduped, true).Find(&plans).Error; err != nil {
		return nil, err
	}
	planByID := make(map[int]ChannelModelPricePlan, len(plans))
	payloadByID := make(map[int]ChannelModelPricePlanPayload, len(plans))
	for _, plan := range plans {
		payload, err := ParseChannelModelPricePlanPayload(plan.PricePayload)
		if err != nil || payload.ResolvedMode() != ChannelModelPricePlanModeRate || !payload.HasRateOverrides() {
			continue
		}
		planByID[plan.ID] = plan
		payloadByID[plan.ID] = payload
	}

	var schedules []ChannelModelPriceSchedule
	if err := DB.Where("channel_id IN ? AND enabled = ?", deduped, true).
		Order("start_minute ASC, id ASC").Find(&schedules).Error; err != nil {
		return nil, err
	}
	for _, schedule := range schedules {
		plan, ok := planByID[schedule.PricePlanID]
		if !ok || plan.ChannelID != schedule.ChannelID || plan.ModelName != schedule.ModelName {
			continue
		}
		payload := payloadByID[plan.ID]
		active := scheduleMatchesAt(schedule, at)
		key := channelModelTimePricingKey(schedule.ChannelID, schedule.ModelName)
		info := result[key]
		info.HasSchedules = true
		info.Timezone = schedule.Timezone
		info.Schedules = append(info.Schedules, PricingTimePricingSchedule{
			ID:            schedule.ID,
			Name:          schedule.Name,
			Weekdays:      schedule.Weekdays,
			StartMinute:   schedule.StartMinute,
			EndMinute:     schedule.EndMinute,
			EffectiveFrom: schedule.EffectiveFrom,
			EffectiveTo:   schedule.EffectiveTo,
			Active:        active,
			PlanID:        plan.ID,
			PlanName:      plan.Name,
			PlanVersion:   plan.Version,
			Mode:          payload.ResolvedMode(),
			Pricing:       resolveChannelModelPricePlanDisplayPricing(schedule.ModelName, payload),
			Rates:         channelModelPricePlanDisplayRates(payload),
			payload:       payload,
		})
		if active && !info.Active {
			info.Active = true
			info.ActiveScheduleID = schedule.ID
			info.ActiveScheduleName = schedule.Name
			info.ActivePlanID = plan.ID
			info.ActivePlanName = plan.Name
			info.ActivePlanVersion = plan.Version
		}
		result[key] = info
	}
	return result, nil
}

func ListChannelModelPricePlans(channelID int, modelName string) ([]ChannelModelPricePlan, error) {
	var rows []ChannelModelPricePlan
	err := DB.Where("channel_id = ? AND model_name = ?", channelID, ratio_setting.FormatMatchingModelName(modelName)).
		Order("updated_at DESC, id DESC").Find(&rows).Error
	return rows, err
}

func ListChannelModelPriceSchedules(channelID int, modelName string) ([]ChannelModelPriceSchedule, error) {
	var rows []ChannelModelPriceSchedule
	err := DB.Where("channel_id = ? AND model_name = ?", channelID, ratio_setting.FormatMatchingModelName(modelName)).
		Order("id ASC").Find(&rows).Error
	return rows, err
}

func GetChannelModelPricePlanByID(planID int) (*ChannelModelPricePlan, error) {
	if planID <= 0 {
		return nil, gorm.ErrRecordNotFound
	}
	var plan ChannelModelPricePlan
	if err := DB.First(&plan, planID).Error; err != nil {
		return nil, err
	}
	return &plan, nil
}

func GetChannelModelPriceScheduleByID(scheduleID int) (*ChannelModelPriceSchedule, error) {
	if scheduleID <= 0 {
		return nil, gorm.ErrRecordNotFound
	}
	var schedule ChannelModelPriceSchedule
	if err := DB.First(&schedule, scheduleID).Error; err != nil {
		return nil, err
	}
	return &schedule, nil
}

func CreateChannelModelPricePlan(plan *ChannelModelPricePlan, payload ChannelModelPricePlanPayload) error {
	if plan == nil {
		return errors.New("price plan is required")
	}
	plan.ModelName = ratio_setting.FormatMatchingModelName(strings.TrimSpace(plan.ModelName))
	plan.Name = strings.TrimSpace(plan.Name)
	if plan.ChannelID <= 0 || plan.ModelName == "" || plan.Name == "" {
		return errors.New("channel, model and plan name are required")
	}
	if err := payload.NormalizeAndValidate(); err != nil {
		return err
	}
	raw, err := payload.MarshalJSONString()
	if err != nil {
		return err
	}
	plan.PricePayload = raw
	plan.Version = 1
	if err := DB.Create(plan).Error; err != nil {
		return err
	}
	InvalidateChannelModelTimePricingCache(plan.ChannelID, plan.ModelName)
	return nil
}

func UpdateChannelModelPricePlan(plan *ChannelModelPricePlan, payload ChannelModelPricePlanPayload) error {
	if plan == nil || plan.ID <= 0 {
		return errors.New("price plan id is required")
	}
	plan.Name = strings.TrimSpace(plan.Name)
	if plan.Name == "" {
		return errors.New("plan name is required")
	}
	if err := payload.NormalizeAndValidate(); err != nil {
		return err
	}
	raw, err := payload.MarshalJSONString()
	if err != nil {
		return err
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var existing ChannelModelPricePlan
		if err := tx.First(&existing, plan.ID).Error; err != nil {
			return err
		}
		nextVersion := existing.Version + 1
		updates := map[string]interface{}{
			"name": plan.Name, "price_payload": raw, "enabled": plan.Enabled,
			"updated_by_user_id": plan.UpdatedByUserID, "version": nextVersion,
		}
		if err := tx.Model(&existing).Updates(updates).Error; err != nil {
			return err
		}
		plan.ChannelID = existing.ChannelID
		plan.ModelName = existing.ModelName
		plan.PricePayload = raw
		plan.Version = nextVersion
		InvalidateChannelModelTimePricingCache(existing.ChannelID, existing.ModelName)
		return nil
	})
}

func DeleteChannelModelPricePlan(planID int) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var plan ChannelModelPricePlan
		if err := tx.First(&plan, planID).Error; err != nil {
			return err
		}
		var count int64
		if err := tx.Model(&ChannelModelPriceSchedule{}).Where("price_plan_id = ?", planID).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrChannelModelPricePlanInUse
		}
		if err := tx.Delete(&plan).Error; err != nil {
			return err
		}
		InvalidateChannelModelTimePricingCache(plan.ChannelID, plan.ModelName)
		return nil
	})
}

func CreateChannelModelPriceSchedule(schedule *ChannelModelPriceSchedule) error {
	if err := ValidateChannelModelPriceSchedule(schedule); err != nil {
		return err
	}
	schedule.ModelName = ratio_setting.FormatMatchingModelName(schedule.ModelName)
	return DB.Transaction(func(tx *gorm.DB) error {
		var plan ChannelModelPricePlan
		if err := tx.First(&plan, schedule.PricePlanID).Error; err != nil {
			return err
		}
		if plan.ChannelID != schedule.ChannelID || plan.ModelName != schedule.ModelName || !isUsableChannelModelRatePlan(plan) {
			return errors.New("price plan does not belong to this channel model")
		}
		if err := ValidateChannelModelScheduleConflict(tx, *schedule); err != nil {
			return err
		}
		if err := tx.Create(schedule).Error; err != nil {
			return err
		}
		InvalidateChannelModelTimePricingCache(schedule.ChannelID, schedule.ModelName)
		return nil
	})
}

func UpdateChannelModelPriceSchedule(schedule *ChannelModelPriceSchedule) error {
	if schedule == nil || schedule.ID <= 0 {
		return errors.New("schedule id is required")
	}
	if err := ValidateChannelModelPriceSchedule(schedule); err != nil {
		return err
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var existing ChannelModelPriceSchedule
		if err := tx.First(&existing, schedule.ID).Error; err != nil {
			return err
		}
		var plan ChannelModelPricePlan
		if err := tx.First(&plan, schedule.PricePlanID).Error; err != nil {
			return err
		}
		if plan.ChannelID != existing.ChannelID || plan.ModelName != existing.ModelName || !isUsableChannelModelRatePlan(plan) {
			return errors.New("price plan does not belong to this channel model")
		}
		schedule.ChannelID = existing.ChannelID
		schedule.SupplierApplicationID = existing.SupplierApplicationID
		schedule.ModelName = existing.ModelName
		if err := ValidateChannelModelScheduleConflict(tx, *schedule); err != nil {
			return err
		}
		updates := map[string]interface{}{
			"price_plan_id": schedule.PricePlanID, "name": schedule.Name,
			"timezone": schedule.Timezone, "weekdays": schedule.Weekdays,
			"start_minute": schedule.StartMinute, "end_minute": schedule.EndMinute,
			"effective_from": schedule.EffectiveFrom, "effective_to": schedule.EffectiveTo,
			"enabled": schedule.Enabled, "updated_by_user_id": schedule.UpdatedByUserID,
		}
		if err := tx.Model(&existing).Updates(updates).Error; err != nil {
			return err
		}
		InvalidateChannelModelTimePricingCache(existing.ChannelID, existing.ModelName)
		return nil
	})
}

func DeleteChannelModelPriceSchedule(scheduleID int) error {
	var schedule ChannelModelPriceSchedule
	if err := DB.First(&schedule, scheduleID).Error; err != nil {
		return err
	}
	if err := DB.Delete(&schedule).Error; err != nil {
		return err
	}
	InvalidateChannelModelTimePricingCache(schedule.ChannelID, schedule.ModelName)
	return nil
}

func ListChannelModelRateRules(channelID int) ([]ChannelModelRateRule, error) {
	var plans []ChannelModelPricePlan
	if err := DB.Where("channel_id = ?", channelID).Find(&plans).Error; err != nil {
		return nil, err
	}
	planByID := make(map[int]ChannelModelPricePlan, len(plans))
	payloadByID := make(map[int]ChannelModelPricePlanPayload, len(plans))
	for _, plan := range plans {
		payload, err := ParseChannelModelPricePlanPayload(plan.PricePayload)
		if err != nil || payload.ResolvedMode() != ChannelModelPricePlanModeRate || !payload.HasRateOverrides() {
			continue
		}
		planByID[plan.ID] = plan
		payloadByID[plan.ID] = payload
	}

	var schedules []ChannelModelPriceSchedule
	if err := DB.Where("channel_id = ?", channelID).
		Order("model_name ASC, start_minute ASC, id ASC").Find(&schedules).Error; err != nil {
		return nil, err
	}
	rules := make([]ChannelModelRateRule, 0, len(schedules))
	for _, schedule := range schedules {
		plan, ok := planByID[schedule.PricePlanID]
		if !ok || plan.ModelName != schedule.ModelName {
			continue
		}
		payload := payloadByID[plan.ID]
		raw := *payload.PriceDiscountPercent
		operating := *payload.OperatingCostPercent
		rules = append(rules, ChannelModelRateRule{
			ScheduleID: schedule.ID, PlanID: plan.ID, PlanVersion: plan.Version,
			ModelName: schedule.ModelName, Name: schedule.Name,
			PriceDiscountPercent: raw, OperatingCostPercent: operating,
			EffectiveCostPercent: EffectiveCostPercent(raw, operating),
			MarkupDiscountRate:   clampChannelMarkupDiscountRate(*payload.MarkupDiscountRate),
			Timezone:             schedule.Timezone, Weekdays: schedule.Weekdays,
			StartMinute: schedule.StartMinute, EndMinute: schedule.EndMinute,
			EffectiveFrom: schedule.EffectiveFrom, EffectiveTo: schedule.EffectiveTo,
			Enabled: plan.Enabled && schedule.Enabled,
			Active:  plan.Enabled && scheduleMatchesAt(schedule, time.Now()),
		})
	}
	return rules, nil
}

func normalizeChannelModelRateRuleMutation(mutation *ChannelModelRateRuleMutation) (ChannelModelPricePlanPayload, error) {
	if mutation == nil {
		return ChannelModelPricePlanPayload{}, errors.New("dynamic rate rule is required")
	}
	mutation.Name = strings.TrimSpace(mutation.Name)
	if mutation.ChannelID <= 0 || mutation.Name == "" || len(mutation.ModelNames) == 0 {
		return ChannelModelPricePlanPayload{}, errors.New("channel, models and rule name are required")
	}
	modelNames := make([]string, 0, len(mutation.ModelNames))
	seen := make(map[string]struct{}, len(mutation.ModelNames))
	for _, rawName := range mutation.ModelNames {
		name := ratio_setting.FormatMatchingModelName(strings.TrimSpace(rawName))
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		modelNames = append(modelNames, name)
	}
	if len(modelNames) == 0 {
		return ChannelModelPricePlanPayload{}, errors.New("at least one model is required")
	}
	mutation.ModelNames = modelNames
	payload := ChannelModelPricePlanPayload{
		Mode:                 ChannelModelPricePlanModeRate,
		PriceDiscountPercent: &mutation.PriceDiscountPercent,
		OperatingCostPercent: &mutation.OperatingCostPercent,
		MarkupDiscountRate:   &mutation.MarkupDiscountRate,
	}
	if err := payload.NormalizeAndValidate(); err != nil {
		return ChannelModelPricePlanPayload{}, err
	}
	return payload, nil
}

func CreateChannelModelRateRules(mutation *ChannelModelRateRuleMutation) error {
	payload, err := normalizeChannelModelRateRuleMutation(mutation)
	if err != nil {
		return err
	}
	raw, err := payload.MarshalJSONString()
	if err != nil {
		return err
	}
	err = DB.Transaction(func(tx *gorm.DB) error {
		for _, modelName := range mutation.ModelNames {
			plan := ChannelModelPricePlan{
				ChannelID: mutation.ChannelID, SupplierApplicationID: mutation.SupplierApplicationID,
				ModelName: modelName, Name: mutation.Name, PricePayload: raw, Version: 1,
				Enabled: mutation.Enabled, CreatedByUserID: mutation.UserID, UpdatedByUserID: mutation.UserID,
			}
			if err := tx.Create(&plan).Error; err != nil {
				return err
			}
			if !mutation.Enabled {
				if err := tx.Model(&plan).Update("enabled", false).Error; err != nil {
					return err
				}
			}
			schedule := ChannelModelPriceSchedule{
				ChannelID: mutation.ChannelID, SupplierApplicationID: mutation.SupplierApplicationID,
				ModelName: modelName, PricePlanID: plan.ID, Name: mutation.Name,
				Timezone: mutation.Timezone, Weekdays: mutation.Weekdays,
				StartMinute: mutation.StartMinute, EndMinute: mutation.EndMinute,
				EffectiveFrom: mutation.EffectiveFrom, EffectiveTo: mutation.EffectiveTo,
				Enabled: mutation.Enabled, CreatedByUserID: mutation.UserID, UpdatedByUserID: mutation.UserID,
			}
			if err := ValidateChannelModelPriceSchedule(&schedule); err != nil {
				return err
			}
			if err := ValidateChannelModelScheduleConflict(tx, schedule); err != nil {
				return err
			}
			if err := tx.Create(&schedule).Error; err != nil {
				return err
			}
			if !mutation.Enabled {
				if err := tx.Model(&schedule).Update("enabled", false).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, modelName := range mutation.ModelNames {
		InvalidateChannelModelTimePricingCache(mutation.ChannelID, modelName)
	}
	return nil
}

func UpdateChannelModelRateRule(scheduleID int, mutation *ChannelModelRateRuleMutation) error {
	if scheduleID <= 0 {
		return errors.New("schedule id is required")
	}
	if mutation != nil && len(mutation.ModelNames) > 1 {
		return errors.New("only one model can be updated at a time")
	}
	payload, err := normalizeChannelModelRateRuleMutation(mutation)
	if err != nil {
		return err
	}
	raw, err := payload.MarshalJSONString()
	if err != nil {
		return err
	}
	var invalidatedModel string
	err = DB.Transaction(func(tx *gorm.DB) error {
		var schedule ChannelModelPriceSchedule
		if err := tx.First(&schedule, scheduleID).Error; err != nil {
			return err
		}
		if schedule.ChannelID != mutation.ChannelID || schedule.ModelName != mutation.ModelNames[0] {
			return gorm.ErrRecordNotFound
		}
		var plan ChannelModelPricePlan
		if err := tx.First(&plan, schedule.PricePlanID).Error; err != nil {
			return err
		}
		oldPayload, err := ParseChannelModelPricePlanPayload(plan.PricePayload)
		if err != nil || oldPayload.ResolvedMode() != ChannelModelPricePlanModeRate || !oldPayload.HasRateOverrides() {
			return errors.New("only dynamic rate rules can be updated")
		}
		if plan.ChannelID != schedule.ChannelID || plan.ModelName != schedule.ModelName {
			return errors.New("dynamic rate plan does not belong to this schedule")
		}
		schedule.Name = mutation.Name
		schedule.Timezone = mutation.Timezone
		schedule.Weekdays = mutation.Weekdays
		schedule.StartMinute = mutation.StartMinute
		schedule.EndMinute = mutation.EndMinute
		schedule.EffectiveFrom = mutation.EffectiveFrom
		schedule.EffectiveTo = mutation.EffectiveTo
		schedule.Enabled = mutation.Enabled
		schedule.UpdatedByUserID = mutation.UserID
		if err := ValidateChannelModelPriceSchedule(&schedule); err != nil {
			return err
		}
		if err := ValidateChannelModelScheduleConflict(tx, schedule); err != nil {
			return err
		}
		var planScheduleCount int64
		if err := tx.Model(&ChannelModelPriceSchedule{}).
			Where("price_plan_id = ?", plan.ID).Count(&planScheduleCount).Error; err != nil {
			return err
		}
		if planScheduleCount > 1 {
			detachedPlan := ChannelModelPricePlan{
				ChannelID: schedule.ChannelID, SupplierApplicationID: schedule.SupplierApplicationID,
				ModelName: schedule.ModelName, Name: mutation.Name, PricePayload: raw,
				Version: plan.Version + 1, Enabled: mutation.Enabled,
				CreatedByUserID: mutation.UserID, UpdatedByUserID: mutation.UserID,
			}
			if err := tx.Create(&detachedPlan).Error; err != nil {
				return err
			}
			if !mutation.Enabled {
				if err := tx.Model(&detachedPlan).Update("enabled", false).Error; err != nil {
					return err
				}
			}
			schedule.PricePlanID = detachedPlan.ID
		}
		if err := tx.Model(&schedule).Updates(map[string]interface{}{
			"price_plan_id": schedule.PricePlanID,
			"name":          schedule.Name, "timezone": schedule.Timezone, "weekdays": schedule.Weekdays,
			"start_minute": schedule.StartMinute, "end_minute": schedule.EndMinute,
			"effective_from": schedule.EffectiveFrom, "effective_to": schedule.EffectiveTo,
			"enabled": schedule.Enabled, "updated_by_user_id": schedule.UpdatedByUserID,
		}).Error; err != nil {
			return err
		}
		if planScheduleCount == 1 {
			if err := tx.Model(&plan).Updates(map[string]interface{}{
				"name": mutation.Name, "price_payload": raw, "enabled": mutation.Enabled,
				"updated_by_user_id": mutation.UserID, "version": plan.Version + 1,
			}).Error; err != nil {
				return err
			}
		}
		invalidatedModel = schedule.ModelName
		return nil
	})
	if err == nil {
		InvalidateChannelModelTimePricingCache(mutation.ChannelID, invalidatedModel)
	}
	return err
}

func DeleteChannelModelRateRule(channelID, scheduleID int) error {
	var invalidatedModel string
	err := DB.Transaction(func(tx *gorm.DB) error {
		var schedule ChannelModelPriceSchedule
		if err := tx.First(&schedule, scheduleID).Error; err != nil {
			return err
		}
		if schedule.ChannelID != channelID {
			return gorm.ErrRecordNotFound
		}
		var plan ChannelModelPricePlan
		if err := tx.First(&plan, schedule.PricePlanID).Error; err != nil {
			return err
		}
		payload, err := ParseChannelModelPricePlanPayload(plan.PricePayload)
		if err != nil || payload.ResolvedMode() != ChannelModelPricePlanModeRate || !payload.HasRateOverrides() {
			return errors.New("only dynamic rate rules can be deleted")
		}
		if plan.ChannelID != schedule.ChannelID || plan.ModelName != schedule.ModelName {
			return errors.New("dynamic rate plan does not belong to this schedule")
		}
		if err := tx.Delete(&schedule).Error; err != nil {
			return err
		}
		var count int64
		if err := tx.Model(&ChannelModelPriceSchedule{}).Where("price_plan_id = ?", plan.ID).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if err := tx.Delete(&plan).Error; err != nil {
				return err
			}
		}
		invalidatedModel = schedule.ModelName
		return nil
	})
	if err == nil {
		InvalidateChannelModelTimePricingCache(channelID, invalidatedModel)
	}
	return err
}

func SortChannelModelSchedulesForDisplay(schedules []ChannelModelPriceSchedule) {
	sort.SliceStable(schedules, func(i, j int) bool {
		if schedules[i].StartMinute != schedules[j].StartMinute {
			return schedules[i].StartMinute < schedules[j].StartMinute
		}
		return schedules[i].ID < schedules[j].ID
	})
}
