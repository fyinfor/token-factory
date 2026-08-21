package model

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	commonRelay "github.com/QuantumNous/new-api/relay/common"
	"gorm.io/gorm"
)

type TaskStatus string

func (t TaskStatus) ToVideoStatus() string {
	var status string
	switch t {
	case TaskStatusNotStart:
		// 与 POST /v1/videos 提交后立即返回的 queued 对齐；库内落库可能仍为 NOT_START
		status = dto.VideoStatusQueued
	case TaskStatusQueued, TaskStatusSubmitted:
		status = dto.VideoStatusQueued
	case TaskStatusInProgress:
		status = dto.VideoStatusInProgress
	case TaskStatusSuccess:
		status = dto.VideoStatusCompleted
	case TaskStatusFailure:
		status = dto.VideoStatusFailed
	default:
		status = dto.VideoStatusUnknown // Default fallback
	}
	return status
}

const (
	TaskStatusNotStart   TaskStatus = "NOT_START"
	TaskStatusSubmitted             = "SUBMITTED"
	TaskStatusQueued                = "QUEUED"
	TaskStatusInProgress            = "IN_PROGRESS"
	TaskStatusFailure               = "FAILURE"
	TaskStatusSuccess               = "SUCCESS"
	TaskStatusUnknown               = "UNKNOWN"
)

type Task struct {
	ID         int64                 `json:"id" gorm:"primary_key;AUTO_INCREMENT"`
	CreatedAt  int64                 `json:"created_at" gorm:"index"`
	UpdatedAt  int64                 `json:"updated_at"`
	TaskID     string                `json:"task_id" gorm:"type:varchar(191);index"` // 第三方id，不一定有/ song id\ Task id
	Platform   constant.TaskPlatform `json:"platform" gorm:"type:varchar(30);index"` // 平台
	UserId     int                   `json:"user_id" gorm:"index"`
	Group      string                `json:"group" gorm:"type:varchar(50)"` // 修正计费用
	ChannelId  int                   `json:"channel_id" gorm:"index"`
	Quota      int                   `json:"quota"`
	Action     string                `json:"action" gorm:"type:varchar(40);index"` // 任务类型, song, lyrics, description-mode
	Status     TaskStatus            `json:"status" gorm:"type:varchar(20);index"` // 任务状态
	FailReason string                `json:"fail_reason"`
	SubmitTime int64                 `json:"submit_time" gorm:"index"`
	StartTime  int64                 `json:"start_time" gorm:"index"`
	FinishTime int64                 `json:"finish_time" gorm:"index"`
	Progress   string                `json:"progress" gorm:"type:varchar(20);index"`
	Properties Properties            `json:"properties" gorm:"type:json"`
	Username   string                `json:"username,omitempty" gorm:"-"`
	// 禁止返回给用户，内部可能包含key等隐私信息
	PrivateData TaskPrivateData `json:"-" gorm:"column:private_data;type:json"`
	Data        json.RawMessage `json:"data" gorm:"type:json"`
}

func (t *Task) SetData(data any) {
	b, _ := common.Marshal(data)
	t.Data = json.RawMessage(b)
}

func (t *Task) GetData(v any) error {
	return common.Unmarshal(t.Data, &v)
}

// GetOriginModelName 返回任务提交时的客户端模型名（计费与权限校验优先使用）。
func (t *Task) GetOriginModelName() string {
	if t == nil {
		return ""
	}
	if bc := t.PrivateData.BillingContext; bc != nil {
		if name := strings.TrimSpace(bc.OriginModelName); name != "" {
			return name
		}
	}
	return strings.TrimSpace(t.Properties.OriginModelName)
}

// GetFetchAccessModelName 查询结果鉴权用的模型名：优先 origin，其次任务 data.model。
func (t *Task) GetFetchAccessModelName() string {
	if name := t.GetOriginModelName(); name != "" {
		return name
	}
	if t == nil || len(t.Data) == 0 {
		return ""
	}
	var taskData map[string]interface{}
	if err := common.Unmarshal(t.Data, &taskData); err != nil {
		return ""
	}
	m, _ := taskData["model"].(string)
	return strings.TrimSpace(m)
}

type Properties struct {
	Input             string `json:"input"`
	UpstreamModelName string `json:"upstream_model_name,omitempty"`
	OriginModelName   string `json:"origin_model_name,omitempty"`
}

func (m *Properties) Scan(val interface{}) error {
	bytesValue, _ := val.([]byte)
	if len(bytesValue) == 0 {
		*m = Properties{}
		return nil
	}
	return common.Unmarshal(bytesValue, m)
}

func (m Properties) Value() (driver.Value, error) {
	if m == (Properties{}) {
		return nil, nil
	}
	return common.Marshal(m)
}

type TaskPrivateData struct {
	Key            string `json:"key,omitempty"`
	UpstreamTaskID string `json:"upstream_task_id,omitempty"` // 上游真实 task ID
	ResultURL      string `json:"result_url,omitempty"`       // 任务成功后的结果 URL（视频地址等）
	TokenName      string `json:"token_name,omitempty"`       // 令牌名称（用于差额日志展示）
	// 计费上下文：用于异步退款/差额结算（轮询阶段读取）
	BillingSource   string              `json:"billing_source,omitempty"`    // "wallet" 或 "subscription"
	SubscriptionId  int                 `json:"subscription_id,omitempty"`   // 订阅 ID，用于订阅退款
	WalletPaidQuota int                 `json:"wallet_paid_quota,omitempty"` // 钱包预扣中来自可开票充值的额度
	TokenId         int                 `json:"token_id,omitempty"`          // 令牌 ID，用于令牌额度退款
	BillingContext  *TaskBillingContext `json:"billing_context,omitempty"`   // 计费参数快照（用于轮询阶段重新计算）
	// TfOpenVideoUpstreamStyle：TokenFactoryOpen(60) 视频上游路径风格，供轮询与提交一致。
	// 空或 "video_generations" => GET {base}/v1/video/generations/{id}；"openai_videos" => GET {base}/v1/videos/{id}。
	TfOpenVideoUpstreamStyle   string `json:"tf_open_video_upstream_style,omitempty"`
	AliyunVideoGuardrailTaskID string `json:"aliyun_video_guardrail_task_id,omitempty"`
	AliyunVideoGuardrailStatus string `json:"aliyun_video_guardrail_status,omitempty"`
	AliyunVideoGuardrailURL    string `json:"aliyun_video_guardrail_url,omitempty"`
	// VideoUpscale 视频超分上下文：命中渠道超分规则时写入，轮询阶段驱动 MPS 超分任务。
	VideoUpscale *TaskVideoUpscaleInfo `json:"video_upscale,omitempty"`
}

// 视频超分任务状态（TaskVideoUpscaleInfo.Status 取值）。
const (
	TaskVideoUpscaleStatusPending    = "pending"    // 已命中规则，等待视频生成完成
	TaskVideoUpscaleStatusProcessing = "processing" // MPS 超分任务已提交，处理中
	TaskVideoUpscaleStatusSuccess    = "success"    // 超分完成
	TaskVideoUpscaleStatusFailed     = "failed"     // 超分失败
)

// TaskVideoUpscaleInfo 视频超分上下文：记录命中的渠道规则与 MPS 任务进度。
type TaskVideoUpscaleInfo struct {
	SourceResolution string  `json:"source_resolution"`      // 生成分辨率（实际生成档位）
	TargetResolution string  `json:"target_resolution"`      // 超分分辨率（对外输出档位）
	TemplateId       uint64  `json:"template_id"`            // MPS 超分模版 ID
	MpsTaskId        string  `json:"mps_task_id,omitempty"`  // MPS 任务 ID（提交后回填）
	Status           string  `json:"status,omitempty"`       // pending/processing/success/failed
	OutputUrl        string  `json:"output_url,omitempty"`   // 超分后视频 URL（完成后回填）
	DurationSec      float64 `json:"duration_sec,omitempty"` // 超分后视频时长（秒，计费用）
	FailReason       string  `json:"fail_reason,omitempty"`  // 超分失败原因
}

// TaskBillingContext 记录任务提交时的计费参数，以便轮询阶段可以重新计算额度。
type TaskBillingContext struct {
	ModelPrice                  float64                `json:"model_price,omitempty"`                    // 模型单价
	GroupRatio                  float64                `json:"group_ratio,omitempty"`                    // 分组倍率
	ModelRatio                  float64                `json:"model_ratio,omitempty"`                    // 模型倍率
	OtherRatios                 map[string]float64     `json:"other_ratios,omitempty"`                   // 附加倍率（时长、分辨率等）
	OriginModelName             string                 `json:"origin_model_name,omitempty"`              // 模型名称，必须为OriginModelName
	PerCallBilling              bool                   `json:"per_call_billing,omitempty"`               // 按次计费：跳过轮询阶段的差额结算
	ChannelPriceDiscountPercent float64                `json:"channel_price_discount_percent,omitempty"` // 最终成本率（100=无折扣），与扣费时一致
	MarkupDiscountPercent       *float64               `json:"markup_discount_percent,omitempty"`        // 加价折扣率快照；指针用于保留显式 0%
	PriceDiscountPercent        *float64               `json:"price_discount_percent,omitempty"`         // 原始成本折扣率快照
	OperatingCostPercent        *float64               `json:"operating_cost_percent,omitempty"`         // 经营成本率快照
	EffectiveCostPercent        *float64               `json:"effective_cost_percent,omitempty"`         // 最终成本率快照；指针用于保留显式 0%
	VideoRuleUnit               string                 `json:"video_rule_unit,omitempty"`                // 视频规则计费单位，例如 per_video
	VideoBillingMode            string                 `json:"video_billing_mode,omitempty"`             // text_to_video / image_to_video / video_to_video
	VideoChannelRulePrice       float64                `json:"video_channel_rule_price,omitempty"`       // 提交时匹配到的渠道规则价（USD）
	VideoGlobalRulePrice        float64                `json:"video_global_rule_price,omitempty"`        // 提交时匹配到的全局规则价（USD）
	VideoRuleWidth              int                    `json:"video_rule_width,omitempty"`
	VideoRuleHeight             int                    `json:"video_rule_height,omitempty"`
	VideoRuleHasAudio           bool                   `json:"video_rule_has_audio,omitempty"`
	TimePricingScheduleID       int                    `json:"time_pricing_schedule_id,omitempty"`
	TimePricingPlanID           int                    `json:"time_pricing_plan_id,omitempty"`
	TimePricingPlanVersion      int                    `json:"time_pricing_plan_version,omitempty"`
	TimePricingScheduleName     string                 `json:"time_pricing_schedule_name,omitempty"`
	TimePricingPlanName         string                 `json:"time_pricing_plan_name,omitempty"`
	TimePricingTimezone         string                 `json:"time_pricing_timezone,omitempty"`
	TimePricingWeekdays         int                    `json:"time_pricing_weekdays,omitempty"`
	TimePricingStartMinute      int                    `json:"time_pricing_start_minute,omitempty"`
	TimePricingEndMinute        int                    `json:"time_pricing_end_minute,omitempty"`
	TimePricingEffectiveFrom    string                 `json:"time_pricing_effective_from,omitempty"`
	TimePricingEffectiveTo      string                 `json:"time_pricing_effective_to,omitempty"`
	TimePricingMatchedAt        int64                  `json:"time_pricing_matched_at,omitempty"`
	TimePricingPayload          string                 `json:"time_pricing_payload,omitempty"`
	UpstreamBillingOther        map[string]interface{} `json:"upstream_billing_other,omitempty"`
}

// GetUpstreamTaskID 获取上游真实 task ID（用于与 provider 通信）
// 旧数据没有 UpstreamTaskID 时，TaskID 本身就是上游 ID
func (t *Task) GetUpstreamTaskID() string {
	if t.PrivateData.UpstreamTaskID != "" {
		return t.PrivateData.UpstreamTaskID
	}
	return t.TaskID
}

// GetResultURL 获取任务结果 URL（视频地址等）
// 新数据存在 PrivateData.ResultURL 中；旧数据回退到 FailReason（历史兼容）
func (t *Task) GetResultURL() string {
	if t.PrivateData.ResultURL != "" {
		return t.PrivateData.ResultURL
	}
	return t.FailReason
}

// GenerateTaskID 生成对外暴露的 task_xxxx 格式 ID
func GenerateTaskID() string {
	key, _ := common.GenerateRandomCharsKey(32)
	return "task_" + key
}

func (p *TaskPrivateData) Scan(val interface{}) error {
	bytesValue, _ := val.([]byte)
	if len(bytesValue) == 0 {
		return nil
	}
	return common.Unmarshal(bytesValue, p)
}

func (p TaskPrivateData) Value() (driver.Value, error) {
	if (p == TaskPrivateData{}) {
		return nil, nil
	}
	return common.Marshal(p)
}

// SyncTaskQueryParams 用于包含所有搜索条件的结构体，可以根据需求添加更多字段
type SyncTaskQueryParams struct {
	Platform        constant.TaskPlatform
	ChannelID       string
	TaskID          string
	ModelName       string
	UserID          string
	Action          string
	Status          string
	StartTimestamp  int64
	EndTimestamp    int64
	UserIDs         []int
	TaskIDs         []string
	VideoType       string // text_to_video / image_to_video / video_to_video
	VideoFailedOnly bool
}

var videoGenerateTaskActions = []string{
	constant.TaskActionGenerate,
	constant.TaskActionTextGenerate,
	constant.TaskActionFirstTailGenerate,
	constant.TaskActionReferenceGenerate,
	constant.TaskActionRemix,
}

var videoToVideoFallbackActions = []string{
	constant.TaskActionFirstTailGenerate,
	constant.TaskActionReferenceGenerate,
	constant.TaskActionRemix,
}

func applyVideoFailedOnlyFilter(query *gorm.DB, videoFailedOnly bool) *gorm.DB {
	if !videoFailedOnly {
		return query
	}
	return query.Where("status = ?", TaskStatusFailure).
		Where("action IN ?", videoGenerateTaskActions)
}

// taskModelNameFilterClause 按 properties 内 origin/upstream 模型名筛选（兼容 SQLite / MySQL / PostgreSQL）。
func taskModelNameFilterClause(modelName string) (string, []interface{}) {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return "", nil
	}
	pat := "%" + modelName + "%"
	if common.UsingMySQL {
		return "JSON_UNQUOTE(JSON_EXTRACT(properties, '$.origin_model_name')) LIKE ? OR JSON_UNQUOTE(JSON_EXTRACT(properties, '$.upstream_model_name')) LIKE ?",
			[]interface{}{pat, pat}
	}
	if common.UsingPostgreSQL {
		return "(properties::json->>'origin_model_name') LIKE ? OR (properties::json->>'upstream_model_name') LIKE ?",
			[]interface{}{pat, pat}
	}
	return "json_extract(properties, '$.origin_model_name') LIKE ? OR json_extract(properties, '$.upstream_model_name') LIKE ?",
		[]interface{}{pat, pat}
}

func normalizeTaskVideoType(videoType string) string {
	switch strings.TrimSpace(videoType) {
	case "text_to_video", constant.TaskActionTextGenerate:
		return "text_to_video"
	case "image_to_video", constant.TaskActionGenerate:
		return "image_to_video"
	case "video_to_video", constant.TaskActionRemix,
		constant.TaskActionFirstTailGenerate, constant.TaskActionReferenceGenerate:
		return "video_to_video"
	default:
		return strings.TrimSpace(videoType)
	}
}

func videoBillingModeEqualsClause(mode string) (string, []interface{}) {
	if common.UsingMySQL {
		return "JSON_UNQUOTE(JSON_EXTRACT(private_data, '$.billing_context.video_billing_mode')) = ?",
			[]interface{}{mode}
	}
	if common.UsingPostgreSQL {
		return "(private_data::json->'billing_context'->>'video_billing_mode') = ?",
			[]interface{}{mode}
	}
	return "json_extract(private_data, '$.billing_context.video_billing_mode') = ?",
		[]interface{}{mode}
}

func videoBillingModeEmptyClause() string {
	if common.UsingMySQL {
		return "(JSON_EXTRACT(private_data, '$.billing_context.video_billing_mode') IS NULL OR JSON_UNQUOTE(JSON_EXTRACT(private_data, '$.billing_context.video_billing_mode')) IN ('', 'null'))"
	}
	if common.UsingPostgreSQL {
		return "(private_data::json->'billing_context'->>'video_billing_mode' IS NULL OR (private_data::json->'billing_context'->>'video_billing_mode') = '')"
	}
	return "(json_extract(private_data, '$.billing_context.video_billing_mode') IS NULL OR json_extract(private_data, '$.billing_context.video_billing_mode') = '')"
}

func applyVideoTypeFilter(query *gorm.DB, videoType string) *gorm.DB {
	mode := normalizeTaskVideoType(videoType)
	if mode == "" {
		return query
	}
	modeClause, modeArgs := videoBillingModeEqualsClause(mode)
	emptyClause := videoBillingModeEmptyClause()
	switch mode {
	case "text_to_video":
		args := append(append([]interface{}{}, modeArgs...), constant.TaskActionTextGenerate)
		return query.Where("("+modeClause+") OR (("+emptyClause+") AND action = ?)", args...)
	case "image_to_video":
		args := append(append([]interface{}{}, modeArgs...), constant.TaskActionGenerate)
		return query.Where("("+modeClause+") OR (("+emptyClause+") AND action = ?)", args...)
	case "video_to_video":
		args := append(append([]interface{}{}, modeArgs...), videoToVideoFallbackActions)
		return query.Where("("+modeClause+") OR (("+emptyClause+") AND action IN ?)", args...)
	default:
		return query
	}
}

func applySyncTaskQueryFilters(query *gorm.DB, queryParams SyncTaskQueryParams) *gorm.DB {
	if queryParams.ChannelID != "" {
		query = query.Where("channel_id = ?", queryParams.ChannelID)
	}
	if queryParams.Platform != "" {
		query = query.Where("platform = ?", queryParams.Platform)
	}
	if queryParams.UserID != "" {
		query = query.Where("user_id = ?", queryParams.UserID)
	}
	if len(queryParams.UserIDs) != 0 {
		query = query.Where("user_id in (?)", queryParams.UserIDs)
	}
	if queryParams.TaskID != "" {
		query = query.Where("task_id = ?", queryParams.TaskID)
	}
	if len(queryParams.TaskIDs) != 0 {
		query = query.Where("task_id IN ?", queryParams.TaskIDs)
	}
	if queryParams.Action != "" {
		query = query.Where("action = ?", queryParams.Action)
	}
	if queryParams.Status != "" {
		query = query.Where("status = ?", queryParams.Status)
	}
	if queryParams.StartTimestamp != 0 {
		query = query.Where("submit_time >= ?", queryParams.StartTimestamp)
	}
	if queryParams.EndTimestamp != 0 {
		query = query.Where("submit_time <= ?", queryParams.EndTimestamp)
	}
	if clause, args := taskModelNameFilterClause(queryParams.ModelName); clause != "" {
		query = query.Where(clause, args...)
	}
	query = applyVideoTypeFilter(query, queryParams.VideoType)
	query = applyVideoFailedOnlyFilter(query, queryParams.VideoFailedOnly)
	return query
}

func InitTask(platform constant.TaskPlatform, relayInfo *commonRelay.RelayInfo) *Task {
	properties := Properties{}
	privateData := TaskPrivateData{}
	if relayInfo != nil && relayInfo.ChannelMeta != nil {
		if relayInfo.ChannelMeta.ChannelType == constant.ChannelTypeGemini ||
			relayInfo.ChannelMeta.ChannelType == constant.ChannelTypeVertexAi {
			privateData.Key = relayInfo.ChannelMeta.ApiKey
		}
		if relayInfo.UpstreamModelName != "" {
			properties.UpstreamModelName = relayInfo.UpstreamModelName
		}
		if relayInfo.OriginModelName != "" {
			properties.OriginModelName = relayInfo.OriginModelName
		}
	}

	// 使用预生成的公开 ID（如果有），否则按渠道规则新生成。
	// Seedance 2.0：通用 task_xxxx 转为火山标准 cgt-{yyyyMMddHHmmss}-{rand}。
	taskID := ""
	channelType := 0
	if relayInfo != nil && relayInfo.ChannelMeta != nil {
		channelType = relayInfo.ChannelType
	}
	if relayInfo.TaskRelayInfo != nil && relayInfo.TaskRelayInfo.PublicTaskID != "" {
		taskID = relayInfo.TaskRelayInfo.PublicTaskID
		if channelType == constant.ChannelTypeSeedance {
			taskID = ConvertToVolcEngineVideoTaskID(taskID, time.Now())
		}
	} else {
		taskID = GeneratePublicTaskID(channelType)
	}
	if relayInfo.TaskRelayInfo != nil {
		relayInfo.TaskRelayInfo.PublicTaskID = taskID
	}

	t := &Task{
		TaskID:      taskID,
		UserId:      relayInfo.UserId,
		Group:       relayInfo.UsingGroup,
		SubmitTime:  time.Now().Unix(),
		Status:      TaskStatusNotStart,
		Progress:    "0%",
		ChannelId:   relayInfo.ChannelId,
		Platform:    platform,
		Properties:  properties,
		PrivateData: privateData,
	}
	return t
}

func TaskGetAllUserTask(userId int, startIdx int, num int, queryParams SyncTaskQueryParams) []*Task {
	var tasks []*Task
	var err error

	query := applySyncTaskQueryFilters(DB.Where("user_id = ?", userId), queryParams)

	err = query.Omit("channel_id").Order("id desc").Limit(num).Offset(startIdx).Find(&tasks).Error
	if err != nil {
		return nil
	}

	return tasks
}

func TaskGetAllTasks(startIdx int, num int, queryParams SyncTaskQueryParams) []*Task {
	var tasks []*Task
	var err error

	query := applySyncTaskQueryFilters(DB, queryParams)

	err = query.Order("id desc").Limit(num).Offset(startIdx).Find(&tasks).Error
	if err != nil {
		return nil
	}

	return tasks
}

func GetTimedOutUnfinishedTasks(cutoffUnix int64, limit int) []*Task {
	var tasks []*Task
	err := DB.Where("progress != ?", "100%").
		Where("status NOT IN ?", []string{TaskStatusFailure, TaskStatusSuccess}).
		Where("submit_time < ?", cutoffUnix).
		Order("submit_time").
		Limit(limit).
		Find(&tasks).Error
	if err != nil {
		return nil
	}
	return tasks
}

func GetAllUnFinishSyncTasks(limit int) []*Task {
	var tasks []*Task
	var err error
	// get all tasks progress is not 100%
	err = DB.Where("progress != ?", "100%").Where("status != ?", TaskStatusFailure).Where("status != ?", TaskStatusSuccess).Limit(limit).Order("id").Find(&tasks).Error
	if err != nil {
		return nil
	}
	return tasks
}

func GetByOnlyTaskId(taskId string) (*Task, bool, error) {
	if taskId == "" {
		return nil, false, nil
	}
	var task *Task
	var err error
	err = DB.Where("task_id = ?", taskId).First(&task).Error
	exist, err := RecordExist(err)
	if err != nil {
		return nil, false, err
	}
	return task, exist, err
}

func GetByTaskId(userId int, taskId string) (*Task, bool, error) {
	if taskId == "" {
		return nil, false, nil
	}
	var task *Task
	var err error
	err = DB.Where("user_id = ? and task_id = ?", userId, taskId).
		First(&task).Error
	exist, err := RecordExist(err)
	if err != nil {
		return nil, false, err
	}
	return task, exist, err
}

func GetByTaskIds(userId int, taskIds []any) ([]*Task, error) {
	if len(taskIds) == 0 {
		return nil, nil
	}
	var task []*Task
	var err error
	err = DB.Where("user_id = ? and task_id in (?)", userId, taskIds).
		Find(&task).Error
	if err != nil {
		return nil, err
	}
	return task, nil
}

// GetByTaskIdsOnly 按任务 ID 列表查询，不限制归属用户（管理员调试查询）。
func GetByTaskIdsOnly(taskIds []any) ([]*Task, error) {
	if len(taskIds) == 0 {
		return nil, nil
	}
	var task []*Task
	err := DB.Where("task_id in (?)", taskIds).Find(&task).Error
	if err != nil {
		return nil, err
	}
	return task, nil
}

func (Task *Task) Insert() error {
	var err error
	err = DB.Create(Task).Error
	return err
}

type taskSnapshot struct {
	Status     TaskStatus
	Progress   string
	StartTime  int64
	FinishTime int64
	FailReason string
	ResultURL  string
	Data       json.RawMessage
}

func (s taskSnapshot) Equal(other taskSnapshot) bool {
	return s.Status == other.Status &&
		s.Progress == other.Progress &&
		s.StartTime == other.StartTime &&
		s.FinishTime == other.FinishTime &&
		s.FailReason == other.FailReason &&
		s.ResultURL == other.ResultURL &&
		bytes.Equal(s.Data, other.Data)
}

func (t *Task) Snapshot() taskSnapshot {
	return taskSnapshot{
		Status:     t.Status,
		Progress:   t.Progress,
		StartTime:  t.StartTime,
		FinishTime: t.FinishTime,
		FailReason: t.FailReason,
		ResultURL:  t.PrivateData.ResultURL,
		Data:       t.Data,
	}
}

func (Task *Task) Update() error {
	var err error
	err = DB.Save(Task).Error
	return err
}

// UpdateWithStatus performs a conditional UPDATE guarded by fromStatus (CAS).
// Returns (true, nil) if this caller won the update, (false, nil) if
// another process already moved the task out of fromStatus.
//
// Uses Model().Select("*").Updates() instead of Save() because GORM's Save
// falls back to INSERT ON CONFLICT when the WHERE-guarded UPDATE matches
// zero rows, which silently bypasses the CAS guard.
func (t *Task) UpdateWithStatus(fromStatus TaskStatus) (bool, error) {
	result := DB.Model(t).Where("status = ?", fromStatus).Select("*").Updates(t)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

// TaskBulkUpdateByID performs an unconditional bulk UPDATE by primary key IDs.
// WARNING: This function has NO CAS (Compare-And-Swap) guard — it will overwrite
// any concurrent status changes. DO NOT use in billing/quota lifecycle flows
// (e.g., timeout, success, failure transitions that trigger refunds or settlements).
// For status transitions that involve billing, use Task.UpdateWithStatus() instead.
func TaskBulkUpdateByID(ids []int64, params map[string]any) error {
	if len(ids) == 0 {
		return nil
	}
	return DB.Model(&Task{}).
		Where("id in (?)", ids).
		Updates(params).Error
}

type TaskQuotaUsage struct {
	Mode  string  `json:"mode"`
	Count float64 `json:"count"`
}

// TaskCountAllTasks returns total tasks that match the given query params (admin usage)
func TaskCountAllTasks(queryParams SyncTaskQueryParams) int64 {
	var total int64
	query := applySyncTaskQueryFilters(DB.Model(&Task{}), queryParams)
	_ = query.Count(&total).Error
	return total
}

// TaskCountAllUserTask returns total tasks for given user
func TaskCountAllUserTask(userId int, queryParams SyncTaskQueryParams) int64 {
	var total int64
	query := applySyncTaskQueryFilters(DB.Model(&Task{}).Where("user_id = ?", userId), queryParams)
	_ = query.Count(&total).Error
	return total
}
func (t *Task) ToOpenAIVideo() *dto.OpenAIVideo {
	openAIVideo := dto.NewOpenAIVideo()
	openAIVideo.ID = t.TaskID
	openAIVideo.Status = t.Status.ToVideoStatus()
	openAIVideo.Model = t.Properties.OriginModelName
	openAIVideo.SetProgressStr(t.Progress)
	openAIVideo.CreatedAt = dto.FormatTimeUnixRFC3339(t.CreatedAt)
	if t.FinishTime > 0 {
		openAIVideo.CompletedAt = dto.FormatTimeUnixRFC3339(t.FinishTime)
	}
	if u := t.ResultURLForResponse(); u != "" {
		openAIVideo.SetOutputVideoURL(u)
	}
	return openAIVideo
}

func (t *Task) ResultURLForResponse() string {
	if t == nil || t.Status == TaskStatusFailure {
		return ""
	}
	u := strings.TrimSpace(t.GetResultURL())
	if strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") || strings.HasPrefix(u, "data:") || strings.HasPrefix(u, "/") {
		return u
	}
	return ""
}
