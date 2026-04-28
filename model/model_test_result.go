package model

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

// 熔断状态机三态，写入 ModelTestResult.CircuitState；空字符串视为 CircuitStateClosed（兼容旧行）。
const (
	CircuitStateClosed   = "closed"
	CircuitStateOpen     = "open"
	CircuitStateHalfOpen = "half_open"
)

// 健康信号来源；用于区分一次写入是测试触发还是真实流量回写，便于排错与运营观测。
const (
	HealthSignalSourceManual    = "manual"
	HealthSignalSourceScheduled = "scheduled"
	HealthSignalSourceOnboard   = "onboard"
	HealthSignalSourceRelay     = "relay"
)

// 错误分类枚举：把上游返回的 HTTP 状态/错误对象归一化到少量类目，驱动差异化熔断与告警。
const (
	ErrorCategoryNone        = ""
	ErrorCategoryTimeout     = "timeout"
	ErrorCategoryRateLimit   = "rate_limit"
	ErrorCategoryAuth        = "auth"
	ErrorCategoryBadRequest  = "bad_request"
	ErrorCategoryUpstream5xx = "upstream_5xx"
	ErrorCategoryParse       = "parse_error"
	ErrorCategoryOther       = "other"
)

// ModelTestResult 记录“某渠道 + 某模型”的最近一次测试结果与累计统计。
// 约定：数据库表名为 model_test_results，布尔列 last_test_success 表示该 (channel_id, model_name) 行「最近一次单测是否成功」；与 Upsert、AutoMigrate 一致。
// 主键 (channel_id, model_name) 下每个渠道每个模型名至多一行。
type ModelTestResult struct {
	// ChannelId 渠道 ID（联合主键之一）。
	ChannelId int `json:"channel_id" gorm:"primaryKey;autoIncrement:false;index:idx_mtr_channel_model,priority:1;comment:渠道ID（联合主键）"`
	// ModelName 模型名称（联合主键之一；GORM 默认列 model_name）。
	ModelName string `json:"model_name" gorm:"primaryKey;autoIncrement:false;type:varchar(255);index:idx_mtr_channel_model,priority:2;comment:模型名称（联合主键）"`
	// LastTestSuccess 最新一次渠道单测是否成功。MySQL 通常映射为 TINYINT(1)，库内存 0/1（1=成功、0=失败），查询操练场用 Pluck+WHERE(=1) 与整型比较一致。
	LastTestSuccess bool `json:"last_test_success" gorm:"default:false;comment:最近一次测试是否成功"`
	// LastTestTime 最近一次测试时间（Unix 秒级时间戳）。
	LastTestTime int64 `json:"last_test_time" gorm:"bigint;default:0;comment:最近一次测试时间（Unix秒）"`
	// LastResponseTime 最近一次测试响应耗时（毫秒）。
	LastResponseTime int `json:"last_response_time" gorm:"default:0;comment:最近一次测试响应耗时（毫秒）"`
	// LastTestMessage 最近一次测试错误信息；成功时通常为空字符串。
	LastTestMessage string `json:"last_test_message" gorm:"type:text;comment:最近一次测试错误信息"`
	// TestCountSuccess 累计成功次数。
	TestCountSuccess int `json:"test_count_success" gorm:"default:0;comment:累计测试成功次数"`
	// TestCountFail 累计失败次数。
	TestCountFail int `json:"test_count_fail" gorm:"default:0;comment:累计测试失败次数"`
	// ManualDisplayResponseTime 运营手动覆盖的「展示用」响应时间（毫秒）；0 表示不覆盖展示耗时（仍用 LastResponseTime 与下列 ManualStabilityGrade 规则）。
	ManualDisplayResponseTime int `json:"manual_display_response_time" gorm:"default:0;comment:运营展示用响应耗时(毫秒) 0=不覆盖"`
	// ManualStabilityGrade 运营手动覆盖的稳定性等级 1-5，0 表示不覆盖（展示仍可按 LastResponseTime 分档）；与 ManualDisplayResponseTime 可同时使用，以手动为准参与 UI。
	ManualStabilityGrade int `json:"manual_stability_grade" gorm:"default:0;comment:运营展示用稳定性等级1-5 0=不覆盖"`

	// === 智能路由健康度字段（PR1 加入；GORM AutoMigrate 自动添加列） ===
	// 老行迁移后字段全部为 0 / 空串：路由消费方需把 0 / 空串视为「未估计」并退化到 LastResponseTime / 全局默认。

	// LastTtftMs 最近一次首 token 时延（毫秒）；非流式或未采样为 0。
	LastTtftMs int `json:"last_ttft_ms" gorm:"default:0;comment:最近一次首token时延(ms) 0=未采样"`
	// LastThroughputTps 最近一次实测吞吐（输出 token/秒）；未采样为 0。
	LastThroughputTps float64 `json:"last_throughput_tps" gorm:"default:0;comment:最近一次实测吞吐(tps) 0=未采样"`
	// LatencyEwmaMs 指数加权平均时延（毫秒），α 见 service.channelHealthEwmaAlpha；驱动路由 LatencyP50Seconds 估算。
	LatencyEwmaMs float64 `json:"latency_ewma_ms" gorm:"default:0;comment:EWMA时延(ms) 0=未估计"`
	// LatencyP50Ms / LatencyP95Ms 滚动百分位估算（PR2 写入；当前列预留）。
	LatencyP50Ms int `json:"latency_p50_ms" gorm:"default:0;comment:滚动P50时延(ms) 0=未估计"`
	LatencyP95Ms int `json:"latency_p95_ms" gorm:"default:0;comment:滚动P95时延(ms) 0=未估计"`
	// RecentSuccessRate 最近窗口内成功率 [0,1]；样本不足时为 0 且 RecentSampleCount<阈值，路由应忽略该信号。
	RecentSuccessRate float64 `json:"recent_success_rate" gorm:"default:0;comment:最近窗口成功率[0,1]"`
	// RecentSampleCount 最近窗口内参与统计的请求数（含成功+失败）；过小（<failureMinRequests）时不可信。
	RecentSampleCount int `json:"recent_sample_count" gorm:"default:0;comment:最近窗口样本数"`
	// ConsecutiveFailCount 连续失败次数；恢复一次成功即清零。
	ConsecutiveFailCount int `json:"consecutive_fail_count" gorm:"default:0;comment:连续失败次数"`
	// UnhealthyUntil 熔断到期 Unix 秒时间戳；now < UnhealthyUntil 时路由视为不健康，跳过候选。0 表示无熔断。
	UnhealthyUntil int64 `json:"unhealthy_until" gorm:"bigint;default:0;comment:熔断到期(Unix秒) 0=未熔断"`
	// CircuitState 熔断状态机：closed / open / half_open；空串等价 closed（兼容旧行）。
	CircuitState string `json:"circuit_state" gorm:"type:varchar(16);default:'';comment:熔断状态 closed|open|half_open"`
	// LastErrorCategory 最近一次失败的类目（timeout / rate_limit / auth / bad_request / upstream_5xx / parse_error / other）；成功时为空。
	LastErrorCategory string `json:"last_error_category" gorm:"type:varchar(32);default:'';comment:最近一次错误类目"`
	// LastHttpStatus 最近一次失败的 HTTP 状态码；成功或无 HTTP 上下文时为 0。
	LastHttpStatus int `json:"last_http_status" gorm:"default:0;comment:最近一次失败HTTP状态码 0=无"`
	// LastTestEndpointType 最近一次测试/调用使用的端点类型（chat / responses / embeddings 等）；用于区分同渠道多端点。
	LastTestEndpointType string `json:"last_test_endpoint_type" gorm:"type:varchar(64);default:'';comment:最近一次端点类型"`
	// LastSignalSource 最近一次健康信号来源（manual / scheduled / onboard / relay）；便于排错。
	LastSignalSource string `json:"last_signal_source" gorm:"type:varchar(16);default:'';comment:最近一次信号来源"`
}

// TableName 显式表名，避免 GORM 命名与迁移/手工表名不一致导致查询为空。
func (ModelTestResult) TableName() string {
	return "model_test_results"
}

// HealthSignal 是 UpsertModelTestResultSignal 的入参，所有「快照型」字段用指针表示「nil=不更新该列」，
// 累计计数（TestCountSuccess/Fail）始终 +1（与旧 Upsert 行为一致）。
// 设计要点：
//   - LatencyP50/P95、CircuitState、UnhealthyUntil 等字段允许置空（指针指向零值），与「未提供」区分；
//   - 调用方一次构造一个 HealthSignal，避免在 service 层散落多个 update map。
type HealthSignal struct {
	ChannelID        int
	ModelName        string
	Success          bool
	ResponseTimeMs   int64
	Message          string
	Source           string

	// 以下为 PR1 新增的可选健康度快照字段；nil 表示该次不覆盖。
	TtftMs               *int
	ThroughputTps        *float64
	LatencyEwmaMs        *float64
	LatencyP50Ms         *int
	LatencyP95Ms         *int
	RecentSuccessRate    *float64
	RecentSampleCount    *int
	ConsecutiveFailCount *int
	UnhealthyUntil       *int64
	CircuitState         *string
	LastErrorCategory    *string
	LastHttpStatus       *int
	LastTestEndpointType *string
}

// UpsertModelTestResult 旧签名兼容入口：仅写入「最近一次测试」基础字段与累计计数，不影响新增的健康度列。
// 行为与 PR1 之前完全一致（包括不写 last_signal_source），便于 controller 层零改动；新代码请直接调用 UpsertModelTestResultSignal。
func UpsertModelTestResult(channelId int, modelName string, success bool, responseTime int64, message string) error {
	return UpsertModelTestResultSignal(HealthSignal{
		ChannelID:      channelId,
		ModelName:      modelName,
		Success:        success,
		ResponseTimeMs: responseTime,
		Message:        message,
	})
}

// UpsertModelTestResultSignal 按 (channel_id, model_name) 更新模型测试结果；不存在则插入。
//   - 始终更新「最近一次」类字段（last_test_success / last_test_time / last_response_time / last_test_message）；
//   - 累计计数 test_count_success / test_count_fail 始终 +1；
//   - HealthSignal 中非 nil 的指针字段才会写入对应列（避免误覆盖正在估算中的滚动指标）；
//   - last_signal_source 在 Source 非空时写入，否则保持原值。
func UpsertModelTestResultSignal(s HealthSignal) error {
	modelName := strings.TrimSpace(s.ModelName)
	if s.ChannelID <= 0 || modelName == "" {
		return nil
	}
	now := common.GetTimestamp()

	// 初始行（FirstOrCreate 命中插入分支时使用）：把可选字段也填上，避免新行落空值。
	result := &ModelTestResult{
		ChannelId:        s.ChannelID,
		ModelName:        modelName,
		LastTestSuccess:  s.Success,
		LastTestTime:     now,
		LastResponseTime: int(s.ResponseTimeMs),
		LastTestMessage:  s.Message,
		LastSignalSource: s.Source,
	}
	if s.Success {
		result.TestCountSuccess = 1
	} else {
		result.TestCountFail = 1
	}
	applySignalToInsertRow(result, s)

	// Update 分支：基础字段 + 累计计数 + 可选快照字段。
	update := map[string]interface{}{
		"last_test_success":  s.Success,
		"last_test_time":     now,
		"last_response_time": int(s.ResponseTimeMs),
		"last_test_message":  s.Message,
	}
	if s.Source != "" {
		update["last_signal_source"] = s.Source
	}
	if s.Success {
		update["test_count_success"] = DB.Raw("test_count_success + 1")
	} else {
		update["test_count_fail"] = DB.Raw("test_count_fail + 1")
	}
	applySignalToUpdateMap(update, s)

	return DB.Where("channel_id = ? AND model_name = ?", s.ChannelID, modelName).
		Assign(update).
		FirstOrCreate(result).Error
}

// applySignalToInsertRow 把 HealthSignal 中非 nil 的字段拷贝到新行结构体（仅 FirstOrCreate 插入分支生效）。
func applySignalToInsertRow(row *ModelTestResult, s HealthSignal) {
	if s.TtftMs != nil {
		row.LastTtftMs = *s.TtftMs
	}
	if s.ThroughputTps != nil {
		row.LastThroughputTps = *s.ThroughputTps
	}
	if s.LatencyEwmaMs != nil {
		row.LatencyEwmaMs = *s.LatencyEwmaMs
	}
	if s.LatencyP50Ms != nil {
		row.LatencyP50Ms = *s.LatencyP50Ms
	}
	if s.LatencyP95Ms != nil {
		row.LatencyP95Ms = *s.LatencyP95Ms
	}
	if s.RecentSuccessRate != nil {
		row.RecentSuccessRate = *s.RecentSuccessRate
	}
	if s.RecentSampleCount != nil {
		row.RecentSampleCount = *s.RecentSampleCount
	}
	if s.ConsecutiveFailCount != nil {
		row.ConsecutiveFailCount = *s.ConsecutiveFailCount
	}
	if s.UnhealthyUntil != nil {
		row.UnhealthyUntil = *s.UnhealthyUntil
	}
	if s.CircuitState != nil {
		row.CircuitState = *s.CircuitState
	}
	if s.LastErrorCategory != nil {
		row.LastErrorCategory = *s.LastErrorCategory
	}
	if s.LastHttpStatus != nil {
		row.LastHttpStatus = *s.LastHttpStatus
	}
	if s.LastTestEndpointType != nil {
		row.LastTestEndpointType = *s.LastTestEndpointType
	}
}

// applySignalToUpdateMap 把 HealthSignal 中非 nil 的字段填入 GORM Assign map。
func applySignalToUpdateMap(update map[string]interface{}, s HealthSignal) {
	if s.TtftMs != nil {
		update["last_ttft_ms"] = *s.TtftMs
	}
	if s.ThroughputTps != nil {
		update["last_throughput_tps"] = *s.ThroughputTps
	}
	if s.LatencyEwmaMs != nil {
		update["latency_ewma_ms"] = *s.LatencyEwmaMs
	}
	if s.LatencyP50Ms != nil {
		update["latency_p50_ms"] = *s.LatencyP50Ms
	}
	if s.LatencyP95Ms != nil {
		update["latency_p95_ms"] = *s.LatencyP95Ms
	}
	if s.RecentSuccessRate != nil {
		update["recent_success_rate"] = *s.RecentSuccessRate
	}
	if s.RecentSampleCount != nil {
		update["recent_sample_count"] = *s.RecentSampleCount
	}
	if s.ConsecutiveFailCount != nil {
		update["consecutive_fail_count"] = *s.ConsecutiveFailCount
	}
	if s.UnhealthyUntil != nil {
		update["unhealthy_until"] = *s.UnhealthyUntil
	}
	if s.CircuitState != nil {
		update["circuit_state"] = *s.CircuitState
	}
	if s.LastErrorCategory != nil {
		update["last_error_category"] = *s.LastErrorCategory
	}
	if s.LastHttpStatus != nil {
		update["last_http_status"] = *s.LastHttpStatus
	}
	if s.LastTestEndpointType != nil {
		update["last_test_endpoint_type"] = *s.LastTestEndpointType
	}
}

// GetModelTestResult 读取单行 (channel_id, model_name) 记录；不存在返回 (nil, nil)。
// 供 service.RecordChannelHealthSignal 等读后写场景使用。
func GetModelTestResult(channelID int, modelName string) (*ModelTestResult, error) {
	modelName = strings.TrimSpace(modelName)
	if channelID <= 0 || modelName == "" {
		return nil, nil
	}
	var row ModelTestResult
	err := DB.Where("channel_id = ? AND model_name = ?", channelID, modelName).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

// mtrResultTableNames 以正式表名 model_test_results 为首；少数旧环境若仅有 model_test_result 会第二顺位尝试读。
var mtrResultTableNames = []string{"model_test_results", "model_test_result"}

// pluckMTRLastSuccessModelNames 在 SQL 的 WHERE 中筛出最近一次成功，只 Pluck model_name。MySQL 下成功存为 1，必须用 1 比较，否则与库内整型/BOOL 对拍失败会导致全空。
// 对 Find 只取少数字段时 bool 解包异常，这里不用 Find，只用 Pluck+字符串列。
func pluckMTRLastSuccessModelNames(t string) ([]string, error) {
	var names []string
	if common.UsingPostgreSQL {
		if err := DB.Table(t).Select("model_name").Where("last_test_success = ?", true).Pluck("model_name", &names).Error; err != nil {
			return nil, err
		}
		return names, nil
	}
	// MySQL 常见 TINYINT(1)/BIT：成功为 1。SQLite 等亦多为 0/1，与 ? 传 1 对拍。
	if common.UsingMySQL || common.UsingSQLite {
		if err := DB.Table(t).Select("model_name").Where("last_test_success = ?", 1).Pluck("model_name", &names).Error; err != nil {
			return nil, err
		}
		return names, nil
	}
	if err := DB.Table(t).Select("model_name").Where("last_test_success = ?", 1).Pluck("model_name", &names).Error; err != nil {
		return nil, err
	}
	if len(names) == 0 {
		var names2 []string
		if err2 := DB.Table(t).Select("model_name").Where("last_test_success = ?", true).Pluck("model_name", &names2).Error; err2 == nil {
			return names2, nil
		}
	}
	return names, nil
}

// loadMTRAllLastSuccessModelNames 合并多表名尝试后的、Trim 去重后的 model_name 列表（均为最近一次为成功的行）。
func loadMTRAllLastSuccessModelNames() ([]string, error) {
	if DB == nil {
		return nil, nil
	}
	mg := DB.Migrator()
	seen := make(map[string]struct{})
	out := make([]string, 0, 32)
	for _, t := range mtrResultTableNames {
		if !mg.HasTable(t) {
			continue
		}
		names, err := pluckMTRLastSuccessModelNames(t)
		if err != nil {
			return nil, err
		}
		for i := range names {
			k := strings.TrimSpace(names[i])
			if k == "" {
				continue
			}
			if _, ok := seen[k]; ok {
				continue
			}
			seen[k] = struct{}{}
			out = append(out, k)
		}
	}
	return out, nil
}

// lastPathSeg 取路径中最后一段（以 / 分隔，常见于 供应商/模型 与短名 对照）。
func lastPathSeg(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	if i := strings.LastIndex(s, "/"); i >= 0 && i+1 < len(s) {
		return strings.TrimSpace(s[i+1:])
	}
	return s
}

// stripGeminiModelsPrefix 若形如 models/xxx（Gemini API 常带此前缀），与后台短名对拍时去掉再比。
func stripGeminiModelsPrefix(s string) string {
	s = strings.TrimSpace(s)
	low := strings.ToLower(s)
	if strings.HasPrefix(low, "models/") {
		if len(s) < len("models/")+1 {
			return s
		}
		return s[len("models/"):]
	}
	return s
}

// mtrNameMatchesForPlayground 判断 model_test_results 中记录的名称与 models.model_name 是否可视为同一条目（全串 Trim+大小写、models/ 前缀、路径最后一段对拍）。
func mtrNameMatchesForPlayground(mtrName, modelMetaName string) bool {
	a := strings.TrimSpace(mtrName)
	b := strings.TrimSpace(modelMetaName)
	if a == "" || b == "" {
		return false
	}
	a, b = stripGeminiModelsPrefix(a), stripGeminiModelsPrefix(b)
	if a == "" || b == "" {
		return false
	}
	if strings.EqualFold(a, b) {
		return true
	}
	aLast, bLast := lastPathSeg(a), lastPathSeg(b)
	if strings.EqualFold(aLast, b) || strings.EqualFold(bLast, a) {
		return true
	}
	if aLast != a && bLast != b && strings.EqualFold(aLast, bLast) {
		return true
	}
	return false
}

// GetPlaygroundTestSuccessByModelNames 对来自 models 元数据的一批 model_name，标出是否在 model_test_results 中存在可对应的「最近一次成功」条（多策略对名）。
func GetPlaygroundTestSuccessByModelNames(candidates []string) (map[string]bool, error) {
	out := make(map[string]bool, len(candidates))
	if len(candidates) == 0 {
		return out, nil
	}
	mtrList, err := loadMTRAllLastSuccessModelNames()
	if err != nil {
		return nil, err
	}
	if len(mtrList) == 0 {
		for _, c := range candidates {
			out[c] = false
		}
		return out, nil
	}
	for _, c := range candidates {
		ok := false
		for i := range mtrList {
			if mtrNameMatchesForPlayground(mtrList[i], c) {
				ok = true
				break
			}
		}
		// 同一 model_name 在 candidate 中重复时结果相同，以最后一次覆盖即可
		out[c] = ok
	}
	return out, nil
}

// GetLatestSuccessfulModelNames 返回「在任意 (channel,model) 上最近一次测试成功」的 model_name 去重集合（键为 Trim 后；供其它逻辑复用）。
func GetLatestSuccessfulModelNames() (map[string]bool, error) {
	list, err := loadMTRAllLastSuccessModelNames()
	if err != nil {
		return nil, err
	}
	result := make(map[string]bool, len(list))
	for i := range list {
		result[list[i]] = true
	}
	return result, nil
}

type channelModelTestRow struct {
	ChannelId int    `gorm:"column:channel_id"`
	ModelName string `gorm:"column:model_name"`
}

func loadMTRPricingSuccessRows(table string) ([]channelModelTestRow, error) {
	var rows []channelModelTestRow
	q := DB.Table(table).Select("channel_id", "model_name")
	if common.UsingPostgreSQL {
		if err := q.Where("last_test_success = ?", true).Find(&rows).Error; err != nil {
			return nil, err
		}
		return rows, nil
	}
	if err := q.Where("last_test_success = ?", 1).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// LoadChannelPricingTestSuccessIndex 返回 channel_id -> 该渠道下最近一次单测成功的模型名列表（去重，保留插入顺序）。
// 会依次尝试 model_test_results / model_test_result 等与 Upsert 一致的表名。
func LoadChannelPricingTestSuccessIndex() (map[int][]string, error) {
	if DB == nil {
		return map[int][]string{}, nil
	}
	mg := DB.Migrator()
	out := make(map[int][]string)
	seen := make(map[int]map[string]struct{})
	for _, t := range mtrResultTableNames {
		if !mg.HasTable(t) {
			continue
		}
		rows, err := loadMTRPricingSuccessRows(t)
		if err != nil {
			return nil, err
		}
		for i := range rows {
			cid := rows[i].ChannelId
			name := strings.TrimSpace(rows[i].ModelName)
			if cid <= 0 || name == "" {
				continue
			}
			if seen[cid] == nil {
				seen[cid] = make(map[string]struct{})
			}
			if _, ok := seen[cid][name]; ok {
				continue
			}
			seen[cid][name] = struct{}{}
			out[cid] = append(out[cid], name)
		}
	}
	return out, nil
}

// ChannelPricingRowMatchesLastTestSuccess 判断 (channelID, pricingModelName) 是否在单测结果表中存在可匹配的成功记录。
func ChannelPricingRowMatchesLastTestSuccess(byChannel map[int][]string, channelID int, pricingModelName string) bool {
	if byChannel == nil || channelID <= 0 {
		return false
	}
	names, ok := byChannel[channelID]
	if !ok || len(names) == 0 {
		return false
	}
	for i := range names {
		if mtrNameMatchesForPlayground(names[i], pricingModelName) {
			return true
		}
	}
	return false
}

// GetModelTestResultsByModelNameAndChannelIDs 按定价/元数据中的 model_name 与渠道 ID 列表查询 model_test_results（行键与 Upsert 写入的 model_name 一致）。
// channelIds 为空时返回空切片，不查库。
func GetModelTestResultsByModelNameAndChannelIDs(modelName string, channelIds []int) ([]ModelTestResult, error) {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" || len(channelIds) == 0 {
		return nil, nil
	}
	var out []ModelTestResult
	err := DB.Model(&ModelTestResult{}).
		Where("model_name = ? AND channel_id IN ?", modelName, channelIds).
		Find(&out).Error
	if err != nil {
		return nil, err
	}
	return out, nil
}

// SetModelTestResultManualDisplay 更新 (channel_id, model_name) 的展示用运营字段；manualMs、manualGrade 均为 0 表示取消覆盖。
// manualGrade 允许 0-5：0=不展示等级覆盖；1-5 为有效等级。manualMs 为毫秒，>0 表示用该值参与模型广场侧响应时间/颜色展示。
func SetModelTestResultManualDisplay(channelId int, modelName string, manualMs int, manualGrade int) error {
	modelName = strings.TrimSpace(modelName)
	if channelId <= 0 || modelName == "" {
		return errors.New("invalid channel_id or model_name")
	}
	if manualMs < 0 {
		manualMs = 0
	}
	if manualGrade < 0 {
		manualGrade = 0
	}
	if manualGrade > 5 {
		manualGrade = 5
	}
	updates := map[string]interface{}{
		"manual_display_response_time": manualMs,
		"manual_stability_grade":       manualGrade,
	}
	// 使用 Assign + FirstOrCreate 做幂等写入，避免「字段值未变化导致 RowsAffected=0」时误判为不存在而重复插入主键。
	return DB.Where("channel_id = ? AND model_name = ?", channelId, modelName).
		Assign(updates).
		FirstOrCreate(&ModelTestResult{
			ChannelId:                 channelId,
			ModelName:                 modelName,
			ManualDisplayResponseTime: manualMs,
			ManualStabilityGrade:      manualGrade,
		}).Error
}

// GetModelTestResultsByChannelIDAndModelNames 渠道测试弹窗用：单渠道 + 多模型名，一次查出已有单测/运营行。
func GetModelTestResultsByChannelIDAndModelNames(channelId int, modelNames []string) ([]ModelTestResult, error) {
	if channelId <= 0 || len(modelNames) == 0 {
		return nil, nil
	}
	var out []ModelTestResult
	err := DB.Model(&ModelTestResult{}).
		Where("channel_id = ? AND model_name IN ?", channelId, modelNames).
		Find(&out).Error
	if err != nil {
		return nil, err
	}
	return out, nil
}

// GetAllModelTestResultsByChannelID 返回某渠道在 model_test_results 中的全部行（弹窗内避免 URL 携带超长 model_names）。
func GetAllModelTestResultsByChannelID(channelId int) ([]ModelTestResult, error) {
	if channelId <= 0 {
		return nil, nil
	}
	var out []ModelTestResult
	err := DB.Model(&ModelTestResult{}).Where("channel_id = ?", channelId).Find(&out).Error
	if err != nil {
		return nil, err
	}
	return out, nil
}
