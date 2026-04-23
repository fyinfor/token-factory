package model

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
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
}

// TableName 显式表名，避免 GORM 命名与迁移/手工表名不一致导致查询为空。
func (ModelTestResult) TableName() string {
	return "model_test_results"
}

// UpsertModelTestResult 按 (channel_id, model_name) 更新模型测试结果；不存在则插入。
func UpsertModelTestResult(channelId int, modelName string, success bool, responseTime int64, message string) error {
	modelName = strings.TrimSpace(modelName)
	if channelId <= 0 || modelName == "" {
		return nil
	}
	now := common.GetTimestamp()
	result := &ModelTestResult{
		ChannelId:        channelId,
		ModelName:        modelName,
		LastTestSuccess:  success,
		LastTestTime:     now,
		LastResponseTime: int(responseTime),
		LastTestMessage:  message,
	}
	if success {
		result.TestCountSuccess = 1
	} else {
		result.TestCountFail = 1
	}
	update := map[string]interface{}{
		"last_test_success":  success,
		"last_test_time":     now,
		"last_response_time": int(responseTime),
		"last_test_message":  message,
	}
	if success {
		update["test_count_success"] = DB.Raw("test_count_success + 1")
	} else {
		update["test_count_fail"] = DB.Raw("test_count_fail + 1")
	}
	return DB.Where("channel_id = ? AND model_name = ?", channelId, modelName).Assign(update).FirstOrCreate(result).Error
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
