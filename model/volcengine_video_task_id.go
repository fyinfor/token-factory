package model

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/constant"
)

const (
	// volcEngineVideoTaskIDPrefix 火山方舟视频任务 ID 固定前缀。
	volcEngineVideoTaskIDPrefix = "cgt"
	// volcEngineVideoTaskIDTimeLayout 中间段时间戳，对应 yyyyMMddHHmmss。
	volcEngineVideoTaskIDTimeLayout = "20060102150405"
	// volcEngineVideoTaskIDRandLen 末尾随机段长度，对齐火山官方形态（如 c6r5j）。
	volcEngineVideoTaskIDRandLen = 5
	// volcEngineVideoTaskIDTimezone 时间戳按时区取任务创建时间，与火山控制台一致。
	volcEngineVideoTaskIDTimezone = "Asia/Shanghai"
	// volcEngineVideoTaskIDRandChars 末尾段字符集：小写字母 + 数字。
	volcEngineVideoTaskIDRandChars = "0123456789abcdefghijklmnopqrstuvwxyz"
)

// volcEngineVideoTaskIDPattern 校验已是火山标准格式：cgt-{14 位时间戳}-{小写字母或数字}。
var volcEngineVideoTaskIDPattern = regexp.MustCompile(`^cgt-\d{14}-[a-z0-9]+$`)

var (
	volcEngineTaskIDLocOnce sync.Once
	volcEngineTaskIDLoc     *time.Location
)

func volcEngineVideoTaskIDLocation() *time.Location {
	volcEngineTaskIDLocOnce.Do(func() {
		loc, err := time.LoadLocation(volcEngineVideoTaskIDTimezone)
		if err != nil {
			volcEngineTaskIDLoc = time.Local
			return
		}
		volcEngineTaskIDLoc = loc
	})
	return volcEngineTaskIDLoc
}

// IsVolcEngineVideoTaskID 判断是否已是火山标准视频任务 ID。
func IsVolcEngineVideoTaskID(taskID string) bool {
	return volcEngineVideoTaskIDPattern.MatchString(strings.TrimSpace(taskID))
}

// ConvertToVolcEngineVideoTaskID 将通用 task_xxxx 转为火山标准视频任务 ID。
//
// 格式：cgt-{yyyyMMddHHmmss}-{随机短串}
//  1. 固定前缀 cgt
//  2. 中间段：任务创建时间（Asia/Shanghai）格式化为 yyyyMMddHHmmss
//  3. 末尾段：小写字母 + 数字的随机短串
//
// 已是标准格式时原样返回（幂等，避免提交/查询重复改写）。
// createdAt 为零值时使用当前时间。genericTaskID 仅用于识别是否已转换，不硬编码任何样例 ID。
func ConvertToVolcEngineVideoTaskID(genericTaskID string, createdAt time.Time) string {
	id := strings.TrimSpace(genericTaskID)
	if IsVolcEngineVideoTaskID(id) {
		return id
	}
	return GenerateVolcEngineVideoTaskID(createdAt)
}

// GenerateVolcEngineVideoTaskID 按火山规则生成新的视频任务 ID（不依赖入参样例）。
func GenerateVolcEngineVideoTaskID(createdAt time.Time) string {
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	ts := createdAt.In(volcEngineVideoTaskIDLocation()).Format(volcEngineVideoTaskIDTimeLayout)
	return fmt.Sprintf("%s-%s-%s", volcEngineVideoTaskIDPrefix, ts, randomVolcEngineTaskIDSuffix(volcEngineVideoTaskIDRandLen))
}

// GeneratePublicTaskID 按渠道生成对外公开任务 ID。
// Seedance 2.0（火山方舟）使用 cgt- 标准格式，其余渠道仍为 task_xxxx。
func GeneratePublicTaskID(channelType int) string {
	genericID := GenerateTaskID()
	if channelType != constant.ChannelTypeSeedance {
		return genericID
	}
	return ConvertToVolcEngineVideoTaskID(genericID, time.Now())
}

func randomVolcEngineTaskIDSuffix(n int) string {
	if n <= 0 {
		n = volcEngineVideoTaskIDRandLen
	}
	chars := volcEngineVideoTaskIDRandChars
	max := big.NewInt(int64(len(chars)))
	b := make([]byte, n)
	for i := range b {
		idx, err := rand.Int(rand.Reader, max)
		if err != nil {
			// crypto/rand 失败时退化为纳秒取模，保证任务提交不中断。
			b[i] = chars[int(time.Now().UnixNano()+int64(i))%len(chars)]
			continue
		}
		b[i] = chars[idx.Int64()]
	}
	return string(b)
}
