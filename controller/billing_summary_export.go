package controller

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

const (
	billingSummaryGranularityPeriod = "period"
	billingSummaryGranularityDay    = "day"
	billingSummaryGranularityMonth  = "month"

	billingSummaryUnitMToken = "Mtoken"
	billingSummaryUnitSecond = "秒"
	billingSummaryUnitImage  = "张"
	billingSummaryUnitVideo  = "条"
	billingSummaryUnitCall   = "次"
)

type billingSummaryExportQuery struct {
	adminLogExportQuery
	Granularity string
}

type billingSummaryBucket struct {
	Key   string
	Label string
	Start int64
}

type billingSummaryTextRow struct {
	Bucket billingSummaryBucket
	Model  string
	Calls  int64

	InputUnitPrice        float64
	InputUsage            float64
	InputOfficialUSD      float64
	OutputUnitPrice       float64
	OutputUsage           float64
	OutputOfficialUSD     float64
	CacheWriteUnitPrice   float64
	CacheWriteUsage       float64
	CacheWriteOfficialUSD float64
	CacheReadUnitPrice    float64
	CacheReadUsage        float64
	CacheReadOfficialUSD  float64
	CallUnitPrice         float64
	CallUsage             float64
	CallOfficialUSD       float64
	OfficialTotalUSD      float64
	SettlementUSD         float64
}

type billingSummaryVideoRow struct {
	Bucket billingSummaryBucket
	Model  string
	Mode   string
	Spec   string
	Calls  int64

	TextInputUnitPrice   float64
	TextInputUsage       float64
	TextInputOfficialUSD float64
	VideoUnitPrice       float64
	VideoUnit            string
	VideoUsage           float64
	VideoOfficialUSD     float64
	OfficialTotalUSD     float64
	SettlementUSD        float64
}

type billingSummaryMediaRow struct {
	Bucket           billingSummaryBucket
	Model            string
	Mode             string
	Spec             string
	Calls            int64
	UnitPrice        float64
	Unit             string
	Usage            float64
	OfficialTotalUSD float64
	SettlementUSD    float64
}

type billingSummaryAudioRow struct {
	Bucket billingSummaryBucket
	Model  string
	Mode   string
	Spec   string
	Calls  int64

	TextInputUnitPrice     float64
	TextInputUsage         float64
	TextInputOfficialUSD   float64
	TextOutputUnitPrice    float64
	TextOutputUsage        float64
	TextOutputOfficialUSD  float64
	AudioInputUnitPrice    float64
	AudioInputUsage        float64
	AudioInputOfficialUSD  float64
	AudioOutputUnitPrice   float64
	AudioOutputUsage       float64
	AudioOutputOfficialUSD float64
	MediaUnitPrice         float64
	MediaUnit              string
	MediaUsage             float64
	MediaOfficialUSD       float64
	OfficialTotalUSD       float64
	SettlementUSD          float64
}

func (row billingSummaryAudioRow) BillingUnit(dict billingSummaryI18n) string {
	if strings.TrimSpace(row.MediaUnit) != "" {
		return row.MediaUnit
	}
	return dict.MToken
}

type billingSummaryData struct {
	Text  []*billingSummaryTextRow
	Video []*billingSummaryVideoRow
	Image []*billingSummaryMediaRow
	Audio []*billingSummaryAudioRow
}

type billingSummaryI18n struct {
	TextSheet, VideoSheet, ImageSheet, AudioSheet       string
	Period, Model, Mode, Spec, Input, Output            string
	CacheWrite, CacheRead, TextInput, TextOutput        string
	AudioInput, AudioOutput, Unit, Usage                string
	PerCall, CallCount, HeaderSeparator, TokenUsage     string
	OfficialTotalUSD, SettlementDiscount, SettlementUSD string
	SettlementTotal                                     string
	MToken, Second, Image, Video, Call                  string
	VideoPerSecond, VideoPerVideo, VideoToken           string
	ImagePerImage, ImageToken                           string
	AudioPerSecond, AudioToken                          string
	MultipleSpecs                                       string
}

var billingSummaryDictZH = billingSummaryI18n{
	TextSheet: "文本", VideoSheet: "视频", ImageSheet: "图片", AudioSheet: "音频",
	Period: "统计周期", Model: "渠道模型", Mode: "计费方式", Spec: "规格",
	Input: "输入", Output: "输出", CacheWrite: "缓存写入", CacheRead: "缓存读取",
	TextInput: "文本输入", TextOutput: "文本输出", AudioInput: "音频输入", AudioOutput: "音频输出",
	Unit: "单位", Usage: "用量", PerCall: "按次", CallCount: "调用次数",
	HeaderSeparator: " ", TokenUsage: "Token", MToken: billingSummaryUnitMToken,
	OfficialTotalUSD: "折扣前（USD）", SettlementDiscount: "折扣比例", SettlementUSD: "折扣后（USD）", SettlementTotal: "折扣后合计",
	Second: billingSummaryUnitSecond, Image: billingSummaryUnitImage, Video: billingSummaryUnitVideo,
	Call: billingSummaryUnitCall, VideoPerSecond: "视频按秒", VideoPerVideo: "视频按条", VideoToken: "视频 Token",
	ImagePerImage: "图片按张", ImageToken: "图片 Token", AudioPerSecond: "音频按秒", AudioToken: "音频 Token",
	MultipleSpecs: "多规格",
}

var billingSummaryDictEN = billingSummaryI18n{
	TextSheet: "Text", VideoSheet: "Video", ImageSheet: "Image", AudioSheet: "Audio",
	Period: "Reporting Period", Model: "Channel Model", Mode: "Billing Method", Spec: "Specification",
	Input: "Input", Output: "Output", CacheWrite: "Cache Write", CacheRead: "Cache Read",
	TextInput: "Text Input", TextOutput: "Text Output", AudioInput: "Audio Input", AudioOutput: "Audio Output",
	Unit: "Unit", Usage: "Usage", PerCall: "Per Call", CallCount: "Call Count",
	HeaderSeparator: " ", TokenUsage: "Tokens", MToken: billingSummaryUnitMToken,
	OfficialTotalUSD: "Before Discount (USD)", SettlementDiscount: "Discount Rate", SettlementUSD: "After Discount (USD)", SettlementTotal: "Total After Discount",
	Second: "second", Image: "image", Video: "video", Call: "call", MultipleSpecs: "Multiple specifications",
	VideoPerSecond: "Per Second", VideoPerVideo: "Per Video", VideoToken: "Video Token",
	ImagePerImage: "Per Image", ImageToken: "Image Token", AudioPerSecond: "Per Second", AudioToken: "Audio Token",
}

func (dict billingSummaryI18n) labeledHeader(prefix, suffix string) string {
	if strings.TrimSpace(prefix) == "" {
		return suffix
	}
	return prefix + dict.HeaderSeparator + suffix
}

func resolveBillingSummaryDict(lang string) billingSummaryI18n {
	if lang == "en" {
		return billingSummaryDictEN
	}
	return billingSummaryDictZH
}

func billingSummaryCurrencyUnit(lang string) string {
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		if lang == "en" {
			return "Quota"
		}
		return "额度"
	}
	if symbol := strings.TrimSpace(operation_setting.GetCurrencySymbol()); symbol != "" {
		return symbol
	}
	return "$"
}

func applyBillingSummaryCurrency(dict billingSummaryI18n, lang string) billingSummaryI18n {
	unit := billingSummaryCurrencyUnit(lang)
	if lang == "en" {
		dict.OfficialTotalUSD = fmt.Sprintf("Before Discount (%s)", unit)
		dict.SettlementUSD = fmt.Sprintf("After Discount (%s)", unit)
		return dict
	}
	dict.OfficialTotalUSD = fmt.Sprintf("折扣前（%s）", unit)
	dict.SettlementUSD = fmt.Sprintf("折扣后（%s）", unit)
	return dict
}

func parseBillingSummaryExportQuery(c *gin.Context) (billingSummaryExportQuery, error) {
	// 计费汇总按时间桶聚合，允许跨年核对；数据量仍受日志导出的 10 万条安全上限约束。
	base, err := parseLogExportQueryWithoutMaxWindow(c)
	if err != nil {
		return billingSummaryExportQuery{}, err
	}
	q := billingSummaryExportQuery{
		adminLogExportQuery: adminLogExportQuery{logExportQuery: base},
		Granularity:         strings.TrimSpace(c.Query("granularity")),
	}
	q.Username = strings.TrimSpace(c.Query("username"))
	if raw := strings.TrimSpace(c.Query("channel")); raw != "" {
		channelID, parseErr := strconv.Atoi(raw)
		if parseErr != nil || channelID < 0 {
			return q, fmt.Errorf("channel 非法")
		}
		q.Channel = channelID
	}
	switch q.Granularity {
	case "", billingSummaryGranularityPeriod:
		q.Granularity = billingSummaryGranularityPeriod
	case billingSummaryGranularityDay, billingSummaryGranularityMonth:
	default:
		return q, fmt.Errorf("granularity 非法")
	}
	return q, nil
}

// ExportBillingSummary 导出按文本、视频、图片、音频拆分的累计计费汇总；现有对账单导出保持不变。
func ExportBillingSummary(c *gin.Context) {
	query, err := parseBillingSummaryExportQuery(c)
	if err != nil {
		c.JSON(400, gin.H{"success": false, "message": err.Error()})
		return
	}
	filter := query.toAdminModelFilter()
	filter.LogTypes = []int{model.LogTypeConsume, model.LogTypeRefund}
	logs, _, err := model.GetAllLogsForExport(filter)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	logs, err = appendBillingSummaryTerminalLogs(logs)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	workbook, err := buildBillingSummaryWorkbook(buildBillingSummaryData(logs, query), query)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	filename := fmt.Sprintf("billing-summary-%s-%s.xlsx",
		time.Unix(query.StartTs, 0).Format("20060102"),
		time.Unix(query.EndTs, 0).Format("20060102"),
	)
	writeUsageLogWorkbook(c, workbook, filename)
}

func billingSummaryTaskID(other map[string]interface{}) string {
	return strings.TrimSpace(logExportString(other, "task_id"))
}

func appendBillingSummaryTerminalLogs(logs []*model.Log) ([]*model.Log, error) {
	taskIDs := make([]string, 0)
	userIDs := make([]int, 0)
	taskSeen := make(map[string]struct{})
	userSeen := make(map[int]struct{})
	logSeen := make(map[int]struct{}, len(logs))
	var minCreatedAt int64
	for _, log := range logs {
		if log == nil {
			continue
		}
		if log.Id > 0 {
			logSeen[log.Id] = struct{}{}
		}
		if log.CreatedAt > 0 && (minCreatedAt == 0 || log.CreatedAt < minCreatedAt) {
			minCreatedAt = log.CreatedAt
		}
		other, _ := common.StrToMap(log.Other)
		taskID := billingSummaryTaskID(other)
		if taskID == "" {
			continue
		}
		if _, ok := taskSeen[taskID]; !ok {
			taskSeen[taskID] = struct{}{}
			taskIDs = append(taskIDs, taskID)
		}
		if log.UserId > 0 {
			if _, ok := userSeen[log.UserId]; !ok {
				userSeen[log.UserId] = struct{}{}
				userIDs = append(userIDs, log.UserId)
			}
		}
	}
	terminal, err := model.GetTaskBillingTerminalLogsForExport(taskIDs, userIDs, minCreatedAt)
	if err != nil {
		return nil, err
	}
	for _, log := range terminal {
		if log == nil {
			continue
		}
		other, _ := common.StrToMap(log.Other)
		if _, ok := taskSeen[billingSummaryTaskID(other)]; !ok {
			continue
		}
		if log.Id > 0 {
			if _, ok := logSeen[log.Id]; ok {
				continue
			}
			logSeen[log.Id] = struct{}{}
		}
		logs = append(logs, log)
	}
	return logs, nil
}

func billingSummaryPhasePriority(phase string) int {
	switch phase {
	case model.BillingPhaseDeltaCharge, model.BillingPhaseDeltaRefund:
		return 50
	case model.BillingPhaseSettlementMerged:
		return 40
	case model.BillingPhaseSettlementMarker:
		return 35
	case model.BillingPhasePreCharge:
		return 20
	case model.BillingPhaseNormal, "":
		return 10
	default:
		return 5
	}
}

// collapseBillingSummaryLogs 将异步任务预扣/结算/补差折叠为一次最终调用。
func collapseBillingSummaryLogs(logs []*model.Log) []*model.Log {
	type taskLog struct {
		log   *model.Log
		other map[string]interface{}
	}
	groups := make(map[string][]taskLog)
	standalone := make([]*model.Log, 0, len(logs))
	for _, log := range logs {
		if log == nil {
			continue
		}
		other, _ := common.StrToMap(log.Other)
		taskID := billingSummaryTaskID(other)
		if taskID == "" {
			if log.Type == model.LogTypeConsume {
				standalone = append(standalone, log)
			}
			continue
		}
		groups[taskID] = append(groups[taskID], taskLog{log: log, other: other})
	}
	out := append([]*model.Log{}, standalone...)
	for _, group := range groups {
		var base, selected *taskLog
		failedRefund := false
		for i := range group {
			item := &group[i]
			phase := logExportString(item.other, "billing_phase")
			if item.log.Type == model.LogTypeConsume && (base == nil || item.log.Id < base.log.Id) {
				base = item
			}
			if phase == model.BillingPhaseRefund && item.log.Type == model.LogTypeRefund {
				failedRefund = true
			}
			if selected == nil || billingSummaryPhasePriority(phase) > billingSummaryPhasePriority(logExportString(selected.other, "billing_phase")) ||
				(billingSummaryPhasePriority(phase) == billingSummaryPhasePriority(logExportString(selected.other, "billing_phase")) && item.log.Id > selected.log.Id) {
				selected = item
			}
		}
		if selected == nil || (failedRefund && logExportString(selected.other, "billing_phase") != model.BillingPhaseDeltaRefund) {
			continue
		}
		merged := make(map[string]interface{})
		if base != nil {
			for key, value := range base.other {
				merged[key] = value
			}
		}
		for key, value := range selected.other {
			merged[key] = value
		}
		clone := *selected.log
		if base != nil && base.log.CreatedAt > 0 {
			clone.CreatedAt = base.log.CreatedAt
		}
		clone.Other = common.MapToJsonStr(merged)
		out = append(out, &clone)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].CreatedAt != out[j].CreatedAt {
			return out[i].CreatedAt < out[j].CreatedAt
		}
		return out[i].Id < out[j].Id
	})
	return out
}

func billingSummaryFinalQuota(log *model.Log, other map[string]interface{}) int64 {
	if log == nil {
		return 0
	}
	phase := logExportString(other, "billing_phase")
	if phase == model.BillingPhaseDeltaCharge || phase == model.BillingPhaseDeltaRefund || phase == model.BillingPhaseSettlementMerged || phase == model.BillingPhaseSettlementMarker {
		if value, ok := logExportNumber(other, "video_final_quota", "actual_quota"); ok && value >= 0 {
			return int64(math.Round(value))
		}
	}
	return resolveUsageLogExportQuota(log, other)
}

func billingSummaryQuotaPerUnit(other map[string]interface{}) float64 {
	if value, ok := logExportNumber(other, "video_quota_per_unit"); ok && value > 0 {
		return value
	}
	if common.QuotaPerUnit > 0 {
		return common.QuotaPerUnit
	}
	return 1
}

func billingSummarySettlementUSD(log *model.Log, other map[string]interface{}) float64 {
	quota := billingSummaryFinalQuota(log, other)
	if quota <= 0 {
		return 0
	}
	return float64(quota) / billingSummaryQuotaPerUnit(other)
}

func billingSummaryRoutedModel(log *model.Log) string {
	if log == nil {
		return ""
	}
	name := strings.TrimSpace(log.ModelName)
	route := strings.TrimSpace(log.RouteSlug)
	if route == "" && log.ChannelId > 0 {
		route = model.DefaultRouteSlugFromChannelID(int64(log.ChannelId))
	}
	if route == "" {
		return name
	}
	return name + "/" + route
}

func billingSummaryBucketFor(timestamp int64, query billingSummaryExportQuery) billingSummaryBucket {
	current := time.Unix(timestamp, 0).In(time.Local)
	switch query.Granularity {
	case billingSummaryGranularityDay:
		start := time.Date(current.Year(), current.Month(), current.Day(), 0, 0, 0, 0, time.Local)
		label := fmt.Sprintf("%d月%d日", start.Month(), start.Day())
		if query.Lang == "en" {
			label = start.Format("2006-01-02")
		}
		return billingSummaryBucket{Key: start.Format("2006-01-02"), Label: label, Start: start.Unix()}
	case billingSummaryGranularityMonth:
		start := time.Date(current.Year(), current.Month(), 1, 0, 0, 0, 0, time.Local)
		label := fmt.Sprintf("%d年%d月", start.Year(), start.Month())
		if query.Lang == "en" {
			label = start.Format("January 2006")
		}
		return billingSummaryBucket{Key: start.Format("2006-01"), Label: label, Start: start.Unix()}
	default:
		start := time.Unix(query.StartTs, 0).In(time.Local)
		end := time.Unix(query.EndTs, 0).In(time.Local)
		label := fmt.Sprintf("%d月%d日-%d月%d日", start.Month(), start.Day(), end.Month(), end.Day())
		if query.Lang == "en" {
			label = start.Format("2006-01-02") + " - " + end.Format("2006-01-02")
		}
		return billingSummaryBucket{Key: "period", Label: label, Start: query.StartTs}
	}
}

func billingSummaryPositiveNumber(other map[string]interface{}, keys ...string) float64 {
	if value, ok := logExportNumber(other, keys...); ok && value > 0 {
		return value
	}
	return 0
}

func billingSummaryFirstNumber(other map[string]interface{}, keys ...string) float64 {
	if value, ok := logExportNumber(other, keys...); ok {
		return value
	}
	return 0
}

func billingSummaryBool(other map[string]interface{}, key string) bool {
	v, ok := other[key]
	if !ok {
		return false
	}
	switch x := v.(type) {
	case bool:
		return x
	case string:
		return strings.EqualFold(strings.TrimSpace(x), "true") || strings.TrimSpace(x) == "1"
	case float64:
		return x != 0
	case int:
		return x != 0
	}
	return false
}

func billingSummaryCacheReadTokens(other map[string]interface{}) float64 {
	return billingSummaryPositiveNumber(other, "cache_tokens", "cache_read_tokens", "cached_tokens", "prompt_cache_hit_tokens")
}

func billingSummaryCacheWriteTokens(other map[string]interface{}) float64 {
	if value := billingSummaryPositiveNumber(other, "cache_write_tokens"); value > 0 {
		return value
	}
	total := billingSummaryPositiveNumber(other, "cache_creation_tokens")
	split := billingSummaryPositiveNumber(other, "cache_creation_tokens_5m") + billingSummaryPositiveNumber(other, "cache_creation_tokens_1h")
	if split > total {
		return split
	}
	return total
}

func billingSummaryFallbackOfficial(settlementUSD float64, other map[string]interface{}) float64 {
	if settlementUSD <= 0 {
		return 0
	}
	discount := model.ParseSettlementDiscountSnapshot(other).OperatingDiscountPercent
	if discount <= 0 {
		discount = 100
	}
	return settlementUSD * 100 / discount
}

func billingSummaryChannelSettlement(officialUSD, fallbackUSD float64, other map[string]interface{}) float64 {
	if officialUSD <= 0 {
		return fallbackUSD
	}
	discount := model.ParseSettlementDiscountSnapshot(other).OperatingDiscountPercent
	if discount < 0 {
		discount = 0
	}
	return officialUSD * discount / 100
}

func billingSummaryBasePrices(other map[string]interface{}) (float64, float64, float64, float64) {
	modelRatio := billingSummaryFirstNumber(other, "global_model_ratio", "model_ratio")
	completionRatio := billingSummaryFirstNumber(other, "global_completion_ratio", "completion_ratio")
	if completionRatio <= 0 {
		completionRatio = 1
	}
	cacheRatio := billingSummaryFirstNumber(other, "global_cache_ratio", "cache_ratio")
	if cacheRatio <= 0 {
		cacheRatio = 1
	}
	createRatio := billingSummaryFirstNumber(other, "global_create_cache_ratio", "cache_creation_ratio")
	if createRatio <= 0 {
		createRatio = 1
	}
	inputPrice := modelRatio * 2
	outputPrice := modelRatio * completionRatio * 2
	cacheReadPrice := modelRatio * cacheRatio * 2
	cacheWritePrice := modelRatio * createRatio * 2
	if billingSummaryBool(other, "request_tier_pricing") {
		if value := billingSummaryPositiveNumber(other, "tier_official_input_unit_price"); value > 0 {
			inputPrice = value
		}
		if value := billingSummaryPositiveNumber(other, "tier_official_output_unit_price"); value > 0 {
			outputPrice = value
		}
		if value := billingSummaryPositiveNumber(other, "tier_official_cache_read_unit_price"); value > 0 {
			cacheReadPrice = value
		}
		if value := billingSummaryPositiveNumber(other, "tier_official_cache_write_unit_price"); value > 0 {
			cacheWritePrice = value
		}
	}
	return inputPrice, outputPrice, cacheWritePrice, cacheReadPrice
}

func billingSummarySpec(other map[string]interface{}, image bool) string {
	parts := make([]string, 0, 4)
	appendPart := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		for _, existing := range parts {
			if existing == value {
				return
			}
		}
		parts = append(parts, value)
	}
	if image {
		appendPart(logExportString(other, "image_rule_tier"))
		appendPart(logExportString(other, "image_rule_resolution", "image_resolution"))
		appendPart(logExportString(other, "image_billing_mode"))
	} else {
		appendPart(logExportString(other, "video_billing_lane"))
		appendPart(logExportString(other, "video_resolution"))
		if len(parts) == 0 {
			width := billingSummaryPositiveNumber(other, "video_rule_width")
			height := billingSummaryPositiveNumber(other, "video_rule_height")
			if width > 0 && height > 0 {
				appendPart(fmt.Sprintf("%dx%d", int(width), int(height)))
			}
		}
		if value, ok := other["video_has_audio"].(bool); ok {
			if value {
				appendPart("有音轨")
			} else {
				appendPart("无音轨")
			}
		}
	}
	return strings.Join(parts, " · ")
}

func billingSummaryMergeSpec(current, next string, dict billingSummaryI18n) string {
	current = strings.TrimSpace(current)
	next = strings.TrimSpace(next)
	if current == next {
		return current
	}
	if current == "" {
		return next
	}
	if next == "" {
		return current
	}
	return dict.MultipleSpecs
}

func billingSummaryIsVideo(other map[string]interface{}) bool {
	mode := strings.ToLower(logExportString(other, "billing_mode"))
	path := strings.ToLower(logExportString(other, "request_path"))
	return strings.Contains(mode, "video") || strings.Contains(path, "/video") ||
		billingSummaryPositiveNumber(other, "video_billed_quota", "video_seconds", "video_count", "video_output_tokens") > 0 ||
		logExportString(other, "video_billing_lane", "video_resolution") != ""
}

func billingSummaryIsImage(other map[string]interface{}) bool {
	mode := strings.ToLower(logExportString(other, "billing_mode"))
	path := strings.ToLower(logExportString(other, "request_path"))
	return mode == "image_per_image" || strings.Contains(path, "/image") || billingSummaryBool(other, "image") ||
		billingSummaryBool(other, "image_generation_call") ||
		billingSummaryPositiveNumber(other, "image_count", "image_output") > 0
}

func billingSummaryIsAudio(other map[string]interface{}) bool {
	path := strings.ToLower(logExportString(other, "request_path"))
	return billingSummaryBool(other, "audio") || billingSummaryBool(other, "ws") || billingSummaryBool(other, "asr") ||
		strings.Contains(path, "/audio") || billingSummaryPositiveNumber(other,
		"audio_seconds", "audio_input", "audio_output", "audio_input_token_count") > 0
}

func billingSummaryTextComponent(log *model.Log, other map[string]interface{}, settlementUSD float64) billingSummaryTextRow {
	row := billingSummaryTextRow{}
	modelPrice := billingSummaryFirstNumber(other, "global_model_price", "model_price")
	if modelPrice > 0 {
		row.CallUsage = 1
		row.CallUnitPrice = modelPrice
		row.CallOfficialUSD = modelPrice
		row.OfficialTotalUSD = modelPrice
		row.SettlementUSD = billingSummaryChannelSettlement(row.OfficialTotalUSD, settlementUSD, other)
		return row
	}
	cacheRead := billingSummaryCacheReadTokens(other)
	cacheWrite := billingSummaryCacheWriteTokens(other)
	input := float64(log.PromptTokens)
	if value := billingSummaryPositiveNumber(other, "tier_billed_input_tokens", "text_input"); value > 0 {
		input = value
	} else if !strings.EqualFold(logExportString(other, "usage_semantic"), "anthropic") {
		input -= cacheRead + cacheWrite
		if input < 0 {
			input = 0
		}
	}
	output := float64(log.CompletionTokens)
	if value := billingSummaryPositiveNumber(other, "text_output"); value > 0 {
		output = value
	}
	row.InputUnitPrice, row.OutputUnitPrice, row.CacheWriteUnitPrice, row.CacheReadUnitPrice = billingSummaryBasePrices(other)
	row.InputUsage, row.OutputUsage, row.CacheWriteUsage, row.CacheReadUsage = input, output, cacheWrite, cacheRead
	row.InputOfficialUSD = input * row.InputUnitPrice / 1_000_000
	row.OutputOfficialUSD = output * row.OutputUnitPrice / 1_000_000
	row.CacheWriteOfficialUSD = cacheWrite * row.CacheWriteUnitPrice / 1_000_000
	row.CacheReadOfficialUSD = cacheRead * row.CacheReadUnitPrice / 1_000_000
	row.OfficialTotalUSD = row.InputOfficialUSD + row.OutputOfficialUSD + row.CacheWriteOfficialUSD + row.CacheReadOfficialUSD
	if row.OfficialTotalUSD <= 0 {
		row.OfficialTotalUSD = billingSummaryFallbackOfficial(settlementUSD, other)
		usage := input + output + cacheWrite + cacheRead
		if usage > 0 {
			row.InputOfficialUSD = row.OfficialTotalUSD * input / usage
			row.OutputOfficialUSD = row.OfficialTotalUSD * output / usage
			row.CacheWriteOfficialUSD = row.OfficialTotalUSD * cacheWrite / usage
			row.CacheReadOfficialUSD = row.OfficialTotalUSD * cacheRead / usage
		}
	}
	row.SettlementUSD = billingSummaryChannelSettlement(row.OfficialTotalUSD, settlementUSD, other)
	return row
}

func billingSummaryVideoComponent(other map[string]interface{}, settlementUSD float64, dict billingSummaryI18n) billingSummaryVideoRow {
	row := billingSummaryVideoRow{Spec: billingSummarySpec(other, false)}
	mode := strings.ToLower(logExportString(other, "billing_mode"))
	lane := strings.ToLower(logExportString(other, "video_billing_lane", "video_rule_unit"))
	inputPrice, _, _, _ := billingSummaryBasePrices(other)
	switch {
	case mode == "video_per_second" || strings.Contains(lane, "per_second") || billingSummaryPositiveNumber(other, "global_video_price_per_second", "video_price_per_second", "seconds") > 0:
		row.Mode = dict.VideoPerSecond
		row.VideoUnit = dict.Second
		row.VideoUsage = billingSummaryPositiveNumber(other, "video_seconds", "seconds", "video_duration")
		row.VideoUnitPrice = billingSummaryFirstNumber(other, "global_video_price_per_second", "video_price_per_second")
	case mode == "video_per_video" || strings.Contains(lane, "per_video") || strings.Contains(lane, "per_item") || billingSummaryPositiveNumber(other, "global_video_price_per_video", "video_price_per_video", "video_count") > 0:
		row.Mode = dict.VideoPerVideo
		row.VideoUnit = dict.Video
		row.VideoUsage = billingSummaryPositiveNumber(other, "video_count")
		if row.VideoUsage <= 0 {
			row.VideoUsage = 1
		}
		row.VideoUnitPrice = billingSummaryFirstNumber(other, "global_video_price_per_video", "video_price_per_video")
	case mode == "video_token" || mode == "video_token_output" || strings.Contains(lane, "per_token") || billingSummaryPositiveNumber(other, "video_output_tokens", "video_total_tokens") > 0:
		row.Mode = dict.VideoToken
		row.VideoUnit = dict.MToken
		row.VideoUsage = billingSummaryPositiveNumber(other, "video_output_tokens", "video_total_tokens")
		row.VideoUnitPrice = billingSummaryFirstNumber(other, "video_global_token_price")
		if row.VideoUnitPrice <= 0 {
			modelRatio := billingSummaryFirstNumber(other, "global_model_ratio", "model_ratio")
			videoRatio := billingSummaryFirstNumber(other, "global_video_ratio", "video_ratio")
			completionRatio := billingSummaryFirstNumber(other, "global_video_completion_ratio", "video_completion_ratio")
			if completionRatio <= 0 {
				completionRatio = 1
			}
			row.VideoUnitPrice = modelRatio * videoRatio * completionRatio * 2
		}
		row.TextInputUsage = billingSummaryPositiveNumber(other, "video_input_text_tokens")
		row.TextInputUnitPrice = inputPrice
		row.TextInputOfficialUSD = row.TextInputUsage * inputPrice / 1_000_000
	default:
		row.Mode = dict.PerCall
		row.VideoUnit = dict.Call
		row.VideoUsage = 1
		row.VideoUnitPrice = billingSummaryFirstNumber(other, "global_model_price", "model_price")
	}
	row.VideoOfficialUSD = row.VideoUnitPrice * row.VideoUsage
	if row.VideoUnit == dict.MToken {
		row.VideoOfficialUSD /= 1_000_000
	}
	row.OfficialTotalUSD = row.TextInputOfficialUSD + row.VideoOfficialUSD
	if row.OfficialTotalUSD <= 0 {
		row.OfficialTotalUSD = billingSummaryFallbackOfficial(settlementUSD, other)
		if row.VideoUsage > 0 {
			factor := 1.0
			if row.VideoUnit == dict.MToken {
				factor = 1_000_000
			}
			row.VideoUnitPrice = row.OfficialTotalUSD * factor / row.VideoUsage
			row.VideoOfficialUSD = row.OfficialTotalUSD
		}
	}
	row.SettlementUSD = billingSummaryChannelSettlement(row.OfficialTotalUSD, settlementUSD, other)
	return row
}

func billingSummaryImageComponent(other map[string]interface{}, settlementUSD float64, dict billingSummaryI18n) billingSummaryMediaRow {
	row := billingSummaryMediaRow{Spec: billingSummarySpec(other, true)}
	mode := strings.ToLower(logExportString(other, "billing_mode"))
	switch {
	case mode == "image_per_image" || billingSummaryPositiveNumber(other, "image_count") > 0:
		row.Mode = dict.ImagePerImage
		row.Unit = dict.Image
		row.Usage = billingSummaryPositiveNumber(other, "image_count")
		if row.Usage <= 0 {
			row.Usage = 1
		}
		row.UnitPrice = billingSummaryFirstNumber(other, "image_global_rule_usd", "image_channel_rule_usd", "image_usd_per_image")
	case billingSummaryPositiveNumber(other, "image_output") > 0:
		row.Mode = dict.ImageToken
		row.Unit = dict.MToken
		row.Usage = billingSummaryPositiveNumber(other, "image_output")
		modelRatio := billingSummaryFirstNumber(other, "global_model_ratio", "model_ratio")
		imageRatio := billingSummaryFirstNumber(other, "global_image_ratio", "image_ratio")
		if imageRatio <= 0 {
			imageRatio = 1
		}
		row.UnitPrice = modelRatio * imageRatio * 2
	default:
		row.Mode = dict.PerCall
		row.Unit = dict.Call
		row.Usage = 1
		row.UnitPrice = billingSummaryFirstNumber(other, "global_model_price", "image_generation_call_price", "model_price")
	}
	row.OfficialTotalUSD = row.UnitPrice * row.Usage
	if row.Unit == dict.MToken {
		row.OfficialTotalUSD /= 1_000_000
	}
	if row.OfficialTotalUSD <= 0 {
		row.OfficialTotalUSD = billingSummaryFallbackOfficial(settlementUSD, other)
		if row.Usage > 0 {
			factor := 1.0
			if row.Unit == dict.MToken {
				factor = 1_000_000
			}
			row.UnitPrice = row.OfficialTotalUSD * factor / row.Usage
		}
	}
	row.SettlementUSD = billingSummaryChannelSettlement(row.OfficialTotalUSD, settlementUSD, other)
	return row
}

func billingSummaryAudioComponent(log *model.Log, other map[string]interface{}, settlementUSD float64, dict billingSummaryI18n) billingSummaryAudioRow {
	row := billingSummaryAudioRow{}
	inputPrice, outputPrice, _, _ := billingSummaryBasePrices(other)
	if billingSummaryBool(other, "asr") || billingSummaryPositiveNumber(other, "audio_seconds") > 0 {
		row.Mode = dict.AudioPerSecond
		row.MediaUnit = dict.Second
		row.MediaUsage = billingSummaryPositiveNumber(other, "audio_seconds", "seconds")
		row.MediaUnitPrice = billingSummaryFirstNumber(other, "global_asr_price", "global_model_price", "asr_unit_price", "model_price")
	} else if billingSummaryFirstNumber(other, "global_model_price", "model_price") > 0 {
		row.Mode = dict.PerCall
		row.MediaUnit = dict.Call
		row.MediaUsage = 1
		row.MediaUnitPrice = billingSummaryFirstNumber(other, "global_model_price", "model_price")
	} else {
		row.Mode = dict.AudioToken
		row.TextInputUsage = billingSummaryPositiveNumber(other, "text_input")
		row.TextOutputUsage = billingSummaryPositiveNumber(other, "text_output")
		row.AudioInputUsage = billingSummaryPositiveNumber(other, "audio_input", "audio_input_token_count")
		row.AudioOutputUsage = billingSummaryPositiveNumber(other, "audio_output")
		if row.TextInputUsage <= 0 && row.AudioInputUsage <= 0 {
			row.TextInputUsage = math.Max(0, float64(log.PromptTokens)-row.AudioInputUsage)
		}
		if row.TextOutputUsage <= 0 && row.AudioOutputUsage <= 0 {
			row.TextOutputUsage = math.Max(0, float64(log.CompletionTokens)-row.AudioOutputUsage)
		}
		row.TextInputUnitPrice = inputPrice
		row.TextOutputUnitPrice = outputPrice
		audioRatio := billingSummaryFirstNumber(other, "global_audio_ratio", "audio_ratio")
		if audioRatio <= 0 {
			audioRatio = 1
		}
		audioCompletionRatio := billingSummaryFirstNumber(other, "global_audio_completion_ratio", "audio_completion_ratio")
		if audioCompletionRatio <= 0 {
			audioCompletionRatio = 1
		}
		row.AudioInputUnitPrice = inputPrice * audioRatio
		if separate := billingSummaryPositiveNumber(other, "audio_input_price"); separate > 0 {
			row.AudioInputUnitPrice = separate
		}
		row.AudioOutputUnitPrice = inputPrice * audioRatio * audioCompletionRatio
		row.TextInputOfficialUSD = row.TextInputUsage * row.TextInputUnitPrice / 1_000_000
		row.TextOutputOfficialUSD = row.TextOutputUsage * row.TextOutputUnitPrice / 1_000_000
		row.AudioInputOfficialUSD = row.AudioInputUsage * row.AudioInputUnitPrice / 1_000_000
		row.AudioOutputOfficialUSD = row.AudioOutputUsage * row.AudioOutputUnitPrice / 1_000_000
	}
	row.MediaOfficialUSD = row.MediaUnitPrice * row.MediaUsage
	row.OfficialTotalUSD = row.TextInputOfficialUSD + row.TextOutputOfficialUSD + row.AudioInputOfficialUSD + row.AudioOutputOfficialUSD + row.MediaOfficialUSD
	if row.OfficialTotalUSD <= 0 {
		row.OfficialTotalUSD = billingSummaryFallbackOfficial(settlementUSD, other)
		if row.MediaUsage > 0 {
			row.MediaUnitPrice = row.OfficialTotalUSD / row.MediaUsage
			row.MediaOfficialUSD = row.OfficialTotalUSD
		}
	}
	row.SettlementUSD = billingSummaryChannelSettlement(row.OfficialTotalUSD, settlementUSD, other)
	return row
}

func buildBillingSummaryData(logs []*model.Log, query billingSummaryExportQuery) billingSummaryData {
	dict := resolveBillingSummaryDict(query.Lang)
	textMap := make(map[string]*billingSummaryTextRow)
	videoMap := make(map[string]*billingSummaryVideoRow)
	imageMap := make(map[string]*billingSummaryMediaRow)
	audioMap := make(map[string]*billingSummaryAudioRow)
	for _, log := range collapseBillingSummaryLogs(logs) {
		if log == nil {
			continue
		}
		other, _ := common.StrToMap(log.Other)
		settlementUSD := billingSummarySettlementUSD(log, other)
		if settlementUSD <= 0 && log.PromptTokens <= 0 && log.CompletionTokens <= 0 {
			continue
		}
		bucket := billingSummaryBucketFor(log.CreatedAt, query)
		modelName := billingSummaryRoutedModel(log)
		baseKey := strings.Join([]string{bucket.Key, modelName}, "\x00")
		switch {
		case billingSummaryIsVideo(other):
			component := billingSummaryVideoComponent(other, settlementUSD, dict)
			component.Calls = 1
			key := strings.Join([]string{baseKey, component.Mode, component.VideoUnit}, "\x00")
			row := videoMap[key]
			if row == nil {
				component.Bucket, component.Model = bucket, modelName
				videoMap[key] = &component
			} else {
				row.Calls += component.Calls
				row.TextInputUsage += component.TextInputUsage
				row.TextInputOfficialUSD += component.TextInputOfficialUSD
				row.VideoUsage += component.VideoUsage
				row.VideoOfficialUSD += component.VideoOfficialUSD
				row.OfficialTotalUSD += component.OfficialTotalUSD
				row.SettlementUSD += component.SettlementUSD
				row.Spec = billingSummaryMergeSpec(row.Spec, component.Spec, dict)
			}
		case billingSummaryIsImage(other):
			component := billingSummaryImageComponent(other, settlementUSD, dict)
			component.Calls = 1
			key := strings.Join([]string{baseKey, component.Mode, component.Unit}, "\x00")
			row := imageMap[key]
			if row == nil {
				component.Bucket, component.Model = bucket, modelName
				imageMap[key] = &component
			} else {
				row.Calls += component.Calls
				row.Usage += component.Usage
				row.OfficialTotalUSD += component.OfficialTotalUSD
				row.SettlementUSD += component.SettlementUSD
				row.Spec = billingSummaryMergeSpec(row.Spec, component.Spec, dict)
			}
		case billingSummaryIsAudio(other):
			component := billingSummaryAudioComponent(log, other, settlementUSD, dict)
			component.Calls = 1
			key := strings.Join([]string{baseKey, component.Mode, component.BillingUnit(dict)}, "\x00")
			row := audioMap[key]
			if row == nil {
				component.Bucket, component.Model = bucket, modelName
				audioMap[key] = &component
			} else {
				row.Calls += component.Calls
				row.TextInputUsage += component.TextInputUsage
				row.TextInputOfficialUSD += component.TextInputOfficialUSD
				row.TextOutputUsage += component.TextOutputUsage
				row.TextOutputOfficialUSD += component.TextOutputOfficialUSD
				row.AudioInputUsage += component.AudioInputUsage
				row.AudioInputOfficialUSD += component.AudioInputOfficialUSD
				row.AudioOutputUsage += component.AudioOutputUsage
				row.AudioOutputOfficialUSD += component.AudioOutputOfficialUSD
				row.MediaUsage += component.MediaUsage
				row.MediaOfficialUSD += component.MediaOfficialUSD
				row.OfficialTotalUSD += component.OfficialTotalUSD
				row.SettlementUSD += component.SettlementUSD
			}
		default:
			component := billingSummaryTextComponent(log, other, settlementUSD)
			component.Calls = 1
			row := textMap[baseKey]
			if row == nil {
				component.Bucket, component.Model = bucket, modelName
				textMap[baseKey] = &component
			} else {
				row.Calls += component.Calls
				row.InputUsage += component.InputUsage
				row.InputOfficialUSD += component.InputOfficialUSD
				row.OutputUsage += component.OutputUsage
				row.OutputOfficialUSD += component.OutputOfficialUSD
				row.CacheWriteUsage += component.CacheWriteUsage
				row.CacheWriteOfficialUSD += component.CacheWriteOfficialUSD
				row.CacheReadUsage += component.CacheReadUsage
				row.CacheReadOfficialUSD += component.CacheReadOfficialUSD
				row.CallUsage += component.CallUsage
				row.CallOfficialUSD += component.CallOfficialUSD
				row.OfficialTotalUSD += component.OfficialTotalUSD
				row.SettlementUSD += component.SettlementUSD
			}
		}
	}
	data := billingSummaryData{}
	for _, row := range textMap {
		if row.InputUsage > 0 {
			row.InputUnitPrice = row.InputOfficialUSD * 1_000_000 / row.InputUsage
		}
		if row.OutputUsage > 0 {
			row.OutputUnitPrice = row.OutputOfficialUSD * 1_000_000 / row.OutputUsage
		}
		if row.CacheWriteUsage > 0 {
			row.CacheWriteUnitPrice = row.CacheWriteOfficialUSD * 1_000_000 / row.CacheWriteUsage
		}
		if row.CacheReadUsage > 0 {
			row.CacheReadUnitPrice = row.CacheReadOfficialUSD * 1_000_000 / row.CacheReadUsage
		}
		if row.CallUsage > 0 {
			row.CallUnitPrice = row.CallOfficialUSD / row.CallUsage
		}
		data.Text = append(data.Text, row)
	}
	for _, row := range videoMap {
		if row.TextInputUsage > 0 {
			row.TextInputUnitPrice = row.TextInputOfficialUSD * 1_000_000 / row.TextInputUsage
		}
		if row.VideoUsage > 0 {
			factor := 1.0
			if row.VideoUnit == dict.MToken {
				factor = 1_000_000
			}
			row.VideoUnitPrice = row.VideoOfficialUSD * factor / row.VideoUsage
		}
		data.Video = append(data.Video, row)
	}
	for _, row := range imageMap {
		if row.Usage > 0 {
			factor := 1.0
			if row.Unit == dict.MToken {
				factor = 1_000_000
			}
			row.UnitPrice = row.OfficialTotalUSD * factor / row.Usage
		}
		data.Image = append(data.Image, row)
	}
	for _, row := range audioMap {
		if row.TextInputUsage > 0 {
			row.TextInputUnitPrice = row.TextInputOfficialUSD * 1_000_000 / row.TextInputUsage
		}
		if row.TextOutputUsage > 0 {
			row.TextOutputUnitPrice = row.TextOutputOfficialUSD * 1_000_000 / row.TextOutputUsage
		}
		if row.AudioInputUsage > 0 {
			row.AudioInputUnitPrice = row.AudioInputOfficialUSD * 1_000_000 / row.AudioInputUsage
		}
		if row.AudioOutputUsage > 0 {
			row.AudioOutputUnitPrice = row.AudioOutputOfficialUSD * 1_000_000 / row.AudioOutputUsage
		}
		if row.MediaUsage > 0 {
			row.MediaUnitPrice = row.MediaOfficialUSD / row.MediaUsage
		}
		data.Audio = append(data.Audio, row)
	}
	sortBillingSummaryData(&data)
	return data
}

func sortBillingSummaryData(data *billingSummaryData) {
	if data == nil {
		return
	}
	sort.Slice(data.Text, func(i, j int) bool {
		if data.Text[i].Bucket.Start != data.Text[j].Bucket.Start {
			return data.Text[i].Bucket.Start < data.Text[j].Bucket.Start
		}
		return data.Text[i].Model < data.Text[j].Model
	})
	sort.Slice(data.Video, func(i, j int) bool {
		if data.Video[i].Bucket.Start != data.Video[j].Bucket.Start {
			return data.Video[i].Bucket.Start < data.Video[j].Bucket.Start
		}
		if data.Video[i].Model != data.Video[j].Model {
			return data.Video[i].Model < data.Video[j].Model
		}
		return data.Video[i].Mode < data.Video[j].Mode
	})
	sort.Slice(data.Image, func(i, j int) bool {
		if data.Image[i].Bucket.Start != data.Image[j].Bucket.Start {
			return data.Image[i].Bucket.Start < data.Image[j].Bucket.Start
		}
		if data.Image[i].Model != data.Image[j].Model {
			return data.Image[i].Model < data.Image[j].Model
		}
		return data.Image[i].Mode < data.Image[j].Mode
	})
	sort.Slice(data.Audio, func(i, j int) bool {
		if data.Audio[i].Bucket.Start != data.Audio[j].Bucket.Start {
			return data.Audio[i].Bucket.Start < data.Audio[j].Bucket.Start
		}
		if data.Audio[i].Model != data.Audio[j].Model {
			return data.Audio[i].Model < data.Audio[j].Model
		}
		return data.Audio[i].Mode < data.Audio[j].Mode
	})
}

type billingSummaryWorkbookStyles struct {
	Header, Text, Money, Percent, Usage, Count, TotalLabel, TotalMoney int
}

func billingSummaryCurrencyFormat() string {
	return usageLogAmountNumberFormat()
}

func billingSummaryDisplayAmount(usd float64) float64 {
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		if common.QuotaPerUnit <= 0 {
			return usd
		}
		return usd * common.QuotaPerUnit
	}
	rate := operation_setting.GetUsdToCurrencyRate(operation_setting.USDExchangeRate)
	if rate <= 0 {
		rate = 1
	}
	return usd * rate
}

func billingSummaryDiscount(officialUSD, settlementUSD float64) float64 {
	if officialUSD <= 0 {
		return 0
	}
	return settlementUSD / officialUSD
}

func newBillingSummaryWorkbookStyles(file *excelize.File) (billingSummaryWorkbookStyles, error) {
	styles := billingSummaryWorkbookStyles{}
	var err error
	styles.Header, err = file.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"}, Fill: excelize.Fill{Type: "pattern", Color: []string{"4472C4"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border:    []excelize.Border{{Type: "left", Color: "D9E2F3", Style: 1}, {Type: "right", Color: "D9E2F3", Style: 1}, {Type: "top", Color: "D9E2F3", Style: 1}, {Type: "bottom", Color: "D9E2F3", Style: 1}},
	})
	if err != nil {
		return styles, err
	}
	styles.Text, err = file.NewStyle(&excelize.Style{Alignment: &excelize.Alignment{Vertical: "center", WrapText: true}})
	if err != nil {
		return styles, err
	}
	moneyFormat := billingSummaryCurrencyFormat()
	styles.Money, err = file.NewStyle(&excelize.Style{CustomNumFmt: &moneyFormat, Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"}})
	if err != nil {
		return styles, err
	}
	percentFormat := "0.00%"
	styles.Percent, err = file.NewStyle(&excelize.Style{CustomNumFmt: &percentFormat, Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"}})
	if err != nil {
		return styles, err
	}
	usageFormat := "#,##0.00"
	styles.Usage, err = file.NewStyle(&excelize.Style{CustomNumFmt: &usageFormat, Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"}})
	if err != nil {
		return styles, err
	}
	countFormat := "#,##0"
	styles.Count, err = file.NewStyle(&excelize.Style{CustomNumFmt: &countFormat, Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"}})
	if err != nil {
		return styles, err
	}
	totalBorder := []excelize.Border{
		{Type: "top", Color: "4472C4", Style: 2},
		{Type: "bottom", Color: "4472C4", Style: 1},
	}
	styles.TotalLabel, err = file.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "1F4E78"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"D9EAF7"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		Border:    totalBorder,
	})
	if err != nil {
		return styles, err
	}
	styles.TotalMoney, err = file.NewStyle(&excelize.Style{
		Font:         &excelize.Font{Bold: true, Color: "1F4E78"},
		Fill:         excelize.Fill{Type: "pattern", Color: []string{"D9EAF7"}, Pattern: 1},
		CustomNumFmt: &moneyFormat,
		Alignment:    &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		Border:       totalBorder,
	})
	return styles, err
}

func setBillingSummarySheetLayout(file *excelize.File, sheet string, widths []float64, freezeRows int) error {
	for index, width := range widths {
		column, _ := excelize.ColumnNumberToName(index + 1)
		if err := file.SetColWidth(sheet, column, column, width); err != nil {
			return err
		}
	}
	if err := file.SetRowHeight(sheet, 1, 34); err != nil {
		return err
	}
	return file.SetPanes(sheet, &excelize.Panes{Freeze: true, YSplit: freezeRows, TopLeftCell: fmt.Sprintf("A%d", freezeRows+1), ActivePane: "bottomLeft"})
}

func setBillingSummaryCell(file *excelize.File, sheet string, column, row int, value interface{}, style int) error {
	cell, _ := excelize.CoordinatesToCellName(column, row)
	if err := file.SetCellValue(sheet, cell, value); err != nil {
		return err
	}
	if style > 0 {
		return file.SetCellStyle(sheet, cell, cell, style)
	}
	return nil
}

func writeBillingSummarySettlementTotal(file *excelize.File, sheet string, row, labelColumn, valueColumn int, totalUSD float64, styles billingSummaryWorkbookStyles, dict billingSummaryI18n) error {
	if err := setBillingSummaryCell(file, sheet, labelColumn, row, dict.SettlementTotal, styles.TotalLabel); err != nil {
		return err
	}
	return setBillingSummaryCell(file, sheet, valueColumn, row, billingSummaryDisplayAmount(totalUSD), styles.TotalMoney)
}

func writeBillingSummaryTextSheet(file *excelize.File, sheet string, rows []*billingSummaryTextRow, styles billingSummaryWorkbookStyles, dict billingSummaryI18n) error {
	if err := setBillingSummarySheetLayout(file, sheet, []float64{18, 38, 14, 19, 19, 20, 20, 20, 14, 20}, 1); err != nil {
		return err
	}
	headers := []string{
		dict.Period, dict.Model, dict.CallCount,
		dict.labeledHeader(dict.Input, dict.TokenUsage), dict.labeledHeader(dict.Output, dict.TokenUsage),
		dict.labeledHeader(dict.CacheWrite, dict.TokenUsage), dict.labeledHeader(dict.CacheRead, dict.TokenUsage),
		dict.OfficialTotalUSD, dict.SettlementDiscount, dict.SettlementUSD,
	}
	for col, value := range headers {
		if err := setBillingSummaryCell(file, sheet, col+1, 1, value, styles.Header); err != nil {
			return err
		}
	}
	settlementTotalUSD := 0.0
	for index, item := range rows {
		settlementTotalUSD += item.SettlementUSD
		row := index + 2
		values := []struct {
			value interface{}
			style int
		}{
			{item.Bucket.Label, styles.Text}, {item.Model, styles.Text}, {item.Calls, styles.Count},
			{item.InputUsage, styles.Usage}, {item.OutputUsage, styles.Usage},
			{item.CacheWriteUsage, styles.Usage}, {item.CacheReadUsage, styles.Usage},
			{billingSummaryDisplayAmount(item.OfficialTotalUSD), styles.Money}, {billingSummaryDiscount(item.OfficialTotalUSD, item.SettlementUSD), styles.Percent}, {billingSummaryDisplayAmount(item.SettlementUSD), styles.Money},
		}
		for col, value := range values {
			if err := setBillingSummaryCell(file, sheet, col+1, row, value.value, value.style); err != nil {
				return err
			}
		}
	}
	return writeBillingSummarySettlementTotal(file, sheet, len(rows)+2, 9, 10, settlementTotalUSD, styles, dict)
}

func writeBillingSummaryVideoSheet(file *excelize.File, sheet string, rows []*billingSummaryVideoRow, styles billingSummaryWorkbookStyles, dict billingSummaryI18n) error {
	if err := setBillingSummarySheetLayout(file, sheet, []float64{18, 38, 14, 20, 30, 18, 14, 20, 14, 20}, 1); err != nil {
		return err
	}
	headers := []string{
		dict.Period, dict.Model, dict.CallCount, dict.Mode, dict.Spec, dict.Usage, dict.Unit,
		dict.OfficialTotalUSD, dict.SettlementDiscount, dict.SettlementUSD,
	}
	for col, value := range headers {
		if err := setBillingSummaryCell(file, sheet, col+1, 1, value, styles.Header); err != nil {
			return err
		}
	}
	settlementTotalUSD := 0.0
	for index, item := range rows {
		settlementTotalUSD += item.SettlementUSD
		row := index + 2
		values := []struct {
			value interface{}
			style int
		}{
			{item.Bucket.Label, styles.Text}, {item.Model, styles.Text}, {item.Calls, styles.Count},
			{item.Mode, styles.Text}, {item.Spec, styles.Text}, {item.VideoUsage, styles.Usage}, {item.VideoUnit, styles.Text},
			{billingSummaryDisplayAmount(item.OfficialTotalUSD), styles.Money}, {billingSummaryDiscount(item.OfficialTotalUSD, item.SettlementUSD), styles.Percent}, {billingSummaryDisplayAmount(item.SettlementUSD), styles.Money},
		}
		for col, value := range values {
			if err := setBillingSummaryCell(file, sheet, col+1, row, value.value, value.style); err != nil {
				return err
			}
		}
	}
	return writeBillingSummarySettlementTotal(file, sheet, len(rows)+2, 9, 10, settlementTotalUSD, styles, dict)
}

func writeBillingSummaryImageSheet(file *excelize.File, sheet string, rows []*billingSummaryMediaRow, styles billingSummaryWorkbookStyles, dict billingSummaryI18n) error {
	if err := setBillingSummarySheetLayout(file, sheet, []float64{18, 38, 14, 20, 30, 18, 14, 20, 14, 20}, 1); err != nil {
		return err
	}
	headers := []string{
		dict.Period, dict.Model, dict.CallCount, dict.Mode, dict.Spec, dict.Usage, dict.Unit,
		dict.OfficialTotalUSD, dict.SettlementDiscount, dict.SettlementUSD,
	}
	for col, value := range headers {
		if err := setBillingSummaryCell(file, sheet, col+1, 1, value, styles.Header); err != nil {
			return err
		}
	}
	settlementTotalUSD := 0.0
	for index, item := range rows {
		settlementTotalUSD += item.SettlementUSD
		row := index + 2
		values := []struct {
			value interface{}
			style int
		}{
			{item.Bucket.Label, styles.Text}, {item.Model, styles.Text}, {item.Calls, styles.Count},
			{item.Mode, styles.Text}, {item.Spec, styles.Text}, {item.Usage, styles.Usage}, {item.Unit, styles.Text},
			{billingSummaryDisplayAmount(item.OfficialTotalUSD), styles.Money}, {billingSummaryDiscount(item.OfficialTotalUSD, item.SettlementUSD), styles.Percent}, {billingSummaryDisplayAmount(item.SettlementUSD), styles.Money},
		}
		for col, value := range values {
			if err := setBillingSummaryCell(file, sheet, col+1, row, value.value, value.style); err != nil {
				return err
			}
		}
	}
	return writeBillingSummarySettlementTotal(file, sheet, len(rows)+2, 9, 10, settlementTotalUSD, styles, dict)
}

func writeBillingSummaryAudioSheet(file *excelize.File, sheet string, rows []*billingSummaryAudioRow, styles billingSummaryWorkbookStyles, dict billingSummaryI18n) error {
	if err := setBillingSummarySheetLayout(file, sheet, []float64{18, 38, 14, 20, 26, 19, 19, 19, 19, 18, 14, 20, 14, 20}, 1); err != nil {
		return err
	}
	headers := []string{
		dict.Period, dict.Model, dict.CallCount, dict.Mode, dict.Spec,
		dict.labeledHeader(dict.TextInput, dict.TokenUsage), dict.labeledHeader(dict.TextOutput, dict.TokenUsage),
		dict.labeledHeader(dict.AudioInput, dict.TokenUsage), dict.labeledHeader(dict.AudioOutput, dict.TokenUsage),
		dict.Usage, dict.Unit,
		dict.OfficialTotalUSD, dict.SettlementDiscount, dict.SettlementUSD,
	}
	for col, value := range headers {
		if err := setBillingSummaryCell(file, sheet, col+1, 1, value, styles.Header); err != nil {
			return err
		}
	}
	settlementTotalUSD := 0.0
	for index, item := range rows {
		settlementTotalUSD += item.SettlementUSD
		row := index + 2
		values := []struct {
			value interface{}
			style int
		}{
			{item.Bucket.Label, styles.Text}, {item.Model, styles.Text}, {item.Calls, styles.Count},
			{item.Mode, styles.Text}, {item.Spec, styles.Text},
			{item.TextInputUsage, styles.Usage}, {item.TextOutputUsage, styles.Usage},
			{item.AudioInputUsage, styles.Usage}, {item.AudioOutputUsage, styles.Usage},
			{item.MediaUsage, styles.Usage}, {item.BillingUnit(dict), styles.Text},
			{billingSummaryDisplayAmount(item.OfficialTotalUSD), styles.Money}, {billingSummaryDiscount(item.OfficialTotalUSD, item.SettlementUSD), styles.Percent}, {billingSummaryDisplayAmount(item.SettlementUSD), styles.Money},
		}
		for col, value := range values {
			if err := setBillingSummaryCell(file, sheet, col+1, row, value.value, value.style); err != nil {
				return err
			}
		}
	}
	return writeBillingSummarySettlementTotal(file, sheet, len(rows)+2, 13, 14, settlementTotalUSD, styles, dict)
}

func buildBillingSummaryWorkbook(data billingSummaryData, query billingSummaryExportQuery) (*excelize.File, error) {
	file := excelize.NewFile()
	dict := applyBillingSummaryCurrency(resolveBillingSummaryDict(query.Lang), query.Lang)
	defaultSheet := file.GetSheetName(0)
	if err := file.SetSheetName(defaultSheet, dict.TextSheet); err != nil {
		_ = file.Close()
		return nil, err
	}
	for _, sheet := range []string{dict.VideoSheet, dict.ImageSheet, dict.AudioSheet} {
		if _, err := file.NewSheet(sheet); err != nil {
			_ = file.Close()
			return nil, err
		}
	}
	styles, err := newBillingSummaryWorkbookStyles(file)
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	if err := writeBillingSummaryTextSheet(file, dict.TextSheet, data.Text, styles, dict); err != nil {
		_ = file.Close()
		return nil, err
	}
	if err := writeBillingSummaryVideoSheet(file, dict.VideoSheet, data.Video, styles, dict); err != nil {
		_ = file.Close()
		return nil, err
	}
	if err := writeBillingSummaryImageSheet(file, dict.ImageSheet, data.Image, styles, dict); err != nil {
		_ = file.Close()
		return nil, err
	}
	if err := writeBillingSummaryAudioSheet(file, dict.AudioSheet, data.Audio, styles, dict); err != nil {
		_ = file.Close()
		return nil, err
	}
	file.SetActiveSheet(0)
	return file, nil
}
