package model

import (
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// quotaDataBackfillRow holds one aggregated (user, model, hour) bucket derived
// from the logs table — used to repair quota_data rows whose token_used is 0
// because they were written before the Seedance/Kling/Sora pre-charge
// instrumentation landed.
type quotaDataBackfillRow struct {
	UserID    int    `gorm:"column:user_id"`
	Username  string `gorm:"column:username"`
	ModelName string `gorm:"column:model_name"`
	Hour      int64  `gorm:"column:hour"`
	Count     int    `gorm:"column:count"`
	Quota     int    `gorm:"column:quota"`
}

// BackfillVideoQuotaDataTokenUsed 一次性回填脚本：
// 把「视频类模型」在 logs 表中的预扣/结算记录，按 (user_id, username, model_name, hour)
// 聚合后回写到 quota_data 表的 token_used 字段。
//
// 适用场景：升级前调用 Seedance/Kling/Sora 等异步视频任务时，pre_charge 走的是
// 旧路径（LogTaskConsumption 调用 LogQuotaData 时 token_used 传 0），
// 导致 /rankings 排行按 sum(token_used) HAVING > 0 过滤后这些模型不出现在 video 标签页。
//
// 幂等策略：
// - 跳过 prompt_tokens + completion_tokens > 0 的行（这些是带真实 token 的同步调用，
//   当前 token_used 已经正确，不需要回填）。
// - 若 quota_data 已存在同 (user_id, username, model_name, created_at) 的行：
//   - token_used == 0  → 用本次聚合的 token_used 覆盖（修复历史脏数据）。
//   - token_used > 0    → 跳过（升级后的新调用已经写过正确值，不能再叠加造成重复）。
// - 若不存在：插入新行。
//
// 通过环境变量 QUOTA_DATA_VIDEO_BACKFILL=1 触发，启动时执行一次后自动退出。
func BackfillVideoQuotaDataTokenUsed() (int, error) {
	if LOG_DB == nil {
		return 0, fmt.Errorf("logs db not initialized")
	}
	if DB == nil {
		return 0, fmt.Errorf("quota_data db not initialized")
	}

	videoKeywords := common.VideoGenerationModels
	if len(videoKeywords) == 0 {
		return 0, fmt.Errorf("video keyword list is empty")
	}

	// 1) 收集所有已知的视频模型名：直接查 logs 里出现过的、按 ModelCategory 判定为 video 的。
	//    这样不需要硬编码种子列表，能跟着 common.VideoGenerationModels 自动扩展。
	var allModelNames []string
	if err := LOG_DB.Table("logs").
		Select("DISTINCT model_name").
		Where("type = ?", LogTypeConsume).
		Where("model_name <> ''").
		Pluck("model_name", &allModelNames).Error; err != nil {
		return 0, fmt.Errorf("query distinct model_name from logs: %w", err)
	}
	videoModels := make([]string, 0, len(allModelNames))
	for _, m := range allModelNames {
		if common.IsVideoGenerationModel(m) {
			videoModels = append(videoModels, m)
		}
	}
	if len(videoModels) == 0 {
		common.SysLog("quota_data video backfill: no video model rows in logs, nothing to do")
		return 0, nil
	}

	// 2) 按 (user_id, username, model_name, hour) 聚合。
	//    限定为 prompt+completion tokens == 0 的行（旧路径写入的"假 0 token"任务）。
	//    token_used 直接用 quota 顶上（与 LogTaskConsumption 修复后的语义一致：
	//    异步视频任务没有真实 token，把消耗的额度作为 token_used 上报）。
	//
	//    排除 settlement_marker（结算展示日志，affects_balance=false）：
	//    它在 live 链路不写 quota_data（见 RecordTaskBillingLog 的 affects_balance 判断）。
	//    Kling 等按秒任务的 marker 是 Quota=0 会被 `quota > 0` 天然过滤掉，但 Seedance
	//    按 token 任务的 marker 记录的是 Quota=actualQuota（>0），若不排除会与 pre_charge
	//    在同一 (user,model,hour) 桶内重复累加，导致 Seedance 回填后的 token_used 翻倍。
	bucketExpr := backfillBucketExpr()
	bucketQuery := LOG_DB.Table("logs").
		Select(fmt.Sprintf(
			"user_id, username, model_name, %s as hour, COUNT(*) as count, SUM(quota) as quota",
			bucketExpr)).
		Where("type = ?", LogTypeConsume).
		Where("model_name IN ?", videoModels).
		Where("quota > 0").
		Where("prompt_tokens + completion_tokens = 0").
		Where("other NOT LIKE ?", settlementMarkerOtherPattern).
		Group(fmt.Sprintf("user_id, username, model_name, %s", bucketExpr))

	var rows []quotaDataBackfillRow
	if err := bucketQuery.Find(&rows).Error; err != nil {
		return 0, fmt.Errorf("aggregate logs by video buckets: %w", err)
	}
	if len(rows) == 0 {
		common.SysLog("quota_data video backfill: no eligible (token-less) video rows in logs")
		return 0, nil
	}

	common.SysLog(fmt.Sprintf("quota_data video backfill: %d (user,model,hour) buckets to process across %d video models",
		len(rows), len(videoModels)))

	// 3) 逐桶回写 quota_data，幂等合并。
	updated := 0
	inserted := 0
	skipped := 0
	for _, row := range rows {
		// token_used = quota（与新代码路径一致），count/log quota 走原值。
		tokenUsed := row.Quota

		existing := &QuotaData{}
		err := DB.Table("quota_data").Where("user_id = ? AND username = ? AND model_name = ? AND created_at = ?",
			row.UserID, row.Username, row.ModelName, row.Hour).
			First(existing).Error
		if err != nil && !isGormNotFound(err) {
			return updated + inserted + skipped, fmt.Errorf("query existing quota_data row: %w", err)
		}

		if existing.Id > 0 {
			// 已存在：只在 token_used == 0 时覆盖，避免破坏升级后写入的正确数据。
			if existing.TokenUsed > 0 {
				skipped++
				continue
			}
			if err := DB.Table("quota_data").Where("id = ?", existing.Id).Updates(map[string]interface{}{
				"token_used": tokenUsed,
				"count":      gorm.Expr("count + ?", row.Count),
				"quota":      gorm.Expr("quota + ?", row.Quota),
			}).Error; err != nil {
				return updated + inserted + skipped, fmt.Errorf("update quota_data id=%d: %w", existing.Id, err)
			}
			updated++
			continue
		}

		newRow := &QuotaData{
			UserID:    row.UserID,
			Username:  row.Username,
			ModelName: row.ModelName,
			CreatedAt: row.Hour,
			Count:     row.Count,
			Quota:     row.Quota,
			TokenUsed: tokenUsed,
		}
		if err := DB.Table("quota_data").Create(newRow).Error; err != nil {
			// 并发场景下 unique 索引可能冲突：忽略 duplicate-key 错误并视为 skipped。
			if isDuplicateKeyError(err) {
				skipped++
				continue
			}
			return updated + inserted + skipped, fmt.Errorf("insert quota_data row: %w", err)
		}
		inserted++
	}

	common.SysLog(fmt.Sprintf(
		"quota_data video backfill done: updated=%d, inserted=%d, skipped=%d (elapsed=%s)",
		updated, inserted, skipped, time.Since(time.Now()).Round(time.Millisecond),
	))
	return updated + inserted + skipped, nil
}

// settlementMarkerOtherPattern 匹配 logs.other 中由 SetBillingLogMetadata 写入的
// settlement_marker 标记（紧凑 JSON，无空格，见 common.MapToJsonStr）。
// 用于在回填聚合时排除「仅展示用、不影响余额」的结算日志，避免 Seedance 双重累加。
const settlementMarkerOtherPattern = `%"billing_phase":"settlement_marker"%`

func backfillBucketExpr() string {
	if common.UsingMySQL {
		return "FLOOR(created_at / 3600) * 3600"
	}
	return "(created_at / 3600) * 3600"
}

func isGormNotFound(err error) bool {
	if err == nil {
		return false
	}
	return err == gorm.ErrRecordNotFound || strings.Contains(err.Error(), "record not found")
}

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	// MySQL 1062 / SQLite 1555 / Postgres 23505 — 关键子串判断即可。
	return strings.Contains(msg, "Duplicate entry") ||
		strings.Contains(msg, "UNIQUE constraint failed") ||
		strings.Contains(msg, "duplicate key value")
}
