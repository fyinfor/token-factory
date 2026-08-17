package model

import (
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
)

// AsrTask 阿里云 ASR 异步转写任务记录。
//
// 生命周期：提交（pending）→ 轮询（running）→ 成功（succeeded）/ 失败（failed）。
// 计费策略：提交时预扣 60 秒费用（Quota 初始为预扣额度），成功取结果后按 usage.duration
// 补差价并将 Quota 更新为实际额度；BilledAt 作为结算幂等键；
// 识别文本缓存于 ResultText，避免上游 transcription_url 过期后结果永久丢失。
type AsrTask struct {
	ID                int64   `json:"id" gorm:"primary_key;AUTO_INCREMENT"`
	TaskID            string  `json:"task_id" gorm:"type:varchar(64);uniqueIndex"`     // 对外公开任务 ID（网关生成）
	UpstreamTaskID    string  `json:"upstream_task_id" gorm:"type:varchar(128);index"` // 上游 DashScope task_id
	UserID            int     `json:"user_id" gorm:"index"`
	TokenID           int     `json:"token_id" gorm:"index"`
	ChannelID         int     `json:"channel_id" gorm:"index"`
	Model             string  `json:"model" gorm:"type:varchar(128)"`
	AudioURL          string  `json:"audio_url" gorm:"type:text"`
	Status            string  `json:"status" gorm:"type:varchar(16);index"` // dto.ASRTaskStatus*
	Quota             int     `json:"quota"`                                // 提交时=预扣额度；结算后=实际额度
	QuotaLogged       int     `json:"quota_logged"`                         // 已写入预扣消费日志的额度；0=旧任务未写预扣日志
	PriceDataSnapshot string  `json:"-" gorm:"type:text"`                   // 提交时价格快照，异步结算不得重新命中时段价格
	AudioSeconds      float64 `json:"audio_seconds"`
	ResultText        string  `json:"result_text" gorm:"type:text"`
	ResultTranscripts string  `json:"result_transcripts" gorm:"type:text"` // JSON: []dto.ASRTranscript
	FailReason        string  `json:"fail_reason" gorm:"type:text"`
	BilledAt          int64   `json:"billed_at"` // 结算时间戳，0 表示尚未按真实时长结算（幂等键）
	CreatedAt         int64   `json:"created_at" gorm:"index"`
	UpdatedAt         int64   `json:"updated_at"`
	FinishedAt        int64   `json:"finished_at"`
}

func NewAsrTaskID() string {
	return "asr_" + common.GetUUID()
}

func (t *AsrTask) Insert() error {
	now := time.Now().Unix()
	if t.CreatedAt == 0 {
		t.CreatedAt = now
	}
	t.UpdatedAt = now
	return DB.Create(t).Error
}

// GetAsrTaskByTaskID 按对外任务 ID 查询，userID > 0 时强制限定归属用户（防止越权读取他人任务）。
func GetAsrTaskByTaskID(taskID string, userID int) (*AsrTask, error) {
	var task AsrTask
	query := DB.Where("task_id = ?", taskID)
	if userID > 0 {
		query = query.Where("user_id = ?", userID)
	}
	if err := query.First(&task).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// GetUnfinishedAsrTasks 获取未完成的异步 ASR 任务，供后台轮询（按创建时间升序）。
func GetUnfinishedAsrTasks(limit int) []*AsrTask {
	if limit <= 0 {
		limit = 100
	}
	var tasks []*AsrTask
	err := DB.Where("status IN ?", []string{dto.ASRTaskStatusPending, dto.ASRTaskStatusRunning}).
		Order("id asc").
		Limit(limit).
		Find(&tasks).Error
	if err != nil {
		common.SysLog("GetUnfinishedAsrTasks error: " + err.Error())
		return nil
	}
	return tasks
}

// GetTimedOutUnfinishedAsrTasks 获取超时仍未完成的异步 ASR 任务。
func GetTimedOutUnfinishedAsrTasks(cutoffUnix int64, limit int) []*AsrTask {
	if limit <= 0 {
		limit = 100
	}
	var tasks []*AsrTask
	err := DB.Where("status IN ? AND created_at <= ?",
		[]string{dto.ASRTaskStatusPending, dto.ASRTaskStatusRunning}, cutoffUnix).
		Order("id asc").
		Limit(limit).
		Find(&tasks).Error
	if err != nil {
		common.SysLog("GetTimedOutUnfinishedAsrTasks error: " + err.Error())
		return nil
	}
	return tasks
}

// MarkRunning 任务进入处理中（由 pending 迁移，幂等）。
func (t *AsrTask) MarkRunning() error {
	now := time.Now().Unix()
	return DB.Model(&AsrTask{}).
		Where("id = ? AND status = ?", t.ID, dto.ASRTaskStatusPending).
		Updates(map[string]any{"status": dto.ASRTaskStatusRunning, "updated_at": now}).Error
}

// MarkFailed 任务失败（终态）。返回 true 表示本次调用赢得状态迁移权（用于避免并发双重退款）。
func (t *AsrTask) MarkFailed(reason string) (bool, error) {
	now := time.Now().Unix()
	result := DB.Model(&AsrTask{}).
		Where("id = ? AND status IN ?", t.ID, []string{dto.ASRTaskStatusPending, dto.ASRTaskStatusRunning}).
		Updates(map[string]any{
			"status":      dto.ASRTaskStatusFailed,
			"fail_reason": reason,
			"updated_at":  now,
			"finished_at": now,
		})
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected > 0 {
		t.Status = dto.ASRTaskStatusFailed
		t.FailReason = reason
		t.FinishedAt = now
		return true, nil
	}
	return false, nil
}

// TryMarkSucceededAndBilled 任务成功并占位结算：仅当 billed_at = 0 时生效。
// 不覆盖 Quota（提交预扣额度需保留至差额结算成功后再更新为实际额度）。
// 返回 true 表示本次调用赢得了结算权，重复轮询/并发请求返回 false。
func (t *AsrTask) TryMarkSucceededAndBilled(resultText string, resultTranscripts string, seconds float64, _ int) (bool, error) {
	now := time.Now().Unix()
	result := DB.Model(&AsrTask{}).
		Where("id = ? AND billed_at = 0 AND status IN ?", t.ID, []string{dto.ASRTaskStatusPending, dto.ASRTaskStatusRunning}).
		Updates(map[string]any{
			"status":             dto.ASRTaskStatusSucceeded,
			"result_text":        resultText,
			"result_transcripts": resultTranscripts,
			"audio_seconds":      seconds,
			"billed_at":          now,
			"updated_at":         now,
			"finished_at":        now,
		})
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected > 0 {
		t.Status = dto.ASRTaskStatusSucceeded
		t.ResultText = resultText
		t.ResultTranscripts = resultTranscripts
		t.AudioSeconds = seconds
		t.BilledAt = now
		t.FinishedAt = now
		return true, nil
	}
	return false, nil
}

// UpdateSettledQuota 差额结算成功后，将 Quota 从预扣额度更新为实际额度。
func (t *AsrTask) UpdateSettledQuota(actualQuota int) error {
	t.Quota = actualQuota
	return DB.Model(&AsrTask{}).Where("id = ?", t.ID).
		Updates(map[string]any{"quota": actualQuota, "updated_at": time.Now().Unix()}).Error
}

// MarkSucceededFromCache 仅更新成功状态与结果（已计费过的补偿路径，不重复计费）。
func (t *AsrTask) MarkSucceededFromCache(resultText string, seconds float64) error {
	now := time.Now().Unix()
	return DB.Model(&AsrTask{}).
		Where("id = ? AND status != ?", t.ID, dto.ASRTaskStatusSucceeded).
		Updates(map[string]any{
			"status":        dto.ASRTaskStatusSucceeded,
			"result_text":   resultText,
			"audio_seconds": seconds,
			"updated_at":    now,
			"finished_at":   now,
		}).Error
}

// ResetBilledAt 结算占位后的失败回滚：差额结算失败时重置 billed_at，
// 使后续查询可重新进入结算流程。注意：保留 Quota（提交预扣额度），不得清零。
func (t *AsrTask) ResetBilledAt() error {
	preQuota := t.Quota
	err := DB.Model(&AsrTask{}).
		Where("id = ? AND billed_at = ?", t.ID, t.BilledAt).
		Updates(map[string]any{
			"status":      dto.ASRTaskStatusRunning,
			"billed_at":   0,
			"finished_at": 0,
			"updated_at":  time.Now().Unix(),
		}).Error
	if err == nil {
		t.BilledAt = 0
		t.FinishedAt = 0
		t.Status = dto.ASRTaskStatusRunning
		t.Quota = preQuota
	}
	return err
}
