package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	channelSyncKeyLen    = 6
	channelSyncKeyMaxLen = 64
	channelSyncKeyTries  = 16
)

// NewChannelSyncKey 生成约 6 位 base62 同步编号（字母数字）。
func NewChannelSyncKey() string {
	key, err := common.GenerateRandomCharsKey(channelSyncKeyLen)
	if err != nil || len(key) != channelSyncKeyLen {
		// 极罕见：退回截断 UUID 十六进制，仍保持 6 位
		u := common.GetUUID()
		if len(u) >= channelSyncKeyLen {
			return u[:channelSyncKeyLen]
		}
		return u
	}
	return key
}

// AllocateUniqueChannelSyncKey 生成库内唯一的 sync_key（排除 excludeChannelID）。
func AllocateUniqueChannelSyncKey(excludeChannelID int) (string, error) {
	for i := 0; i < channelSyncKeyTries; i++ {
		key := NewChannelSyncKey()
		if err := ValidateChannelSyncKeyUnique(excludeChannelID, key); err == nil {
			return key, nil
		}
	}
	return "", fmt.Errorf("无法生成唯一 sync_key，请重试")
}

// NormalizeChannelSyncKey 规范化 sync_key：去首尾空白；空串表示未设置。
func NormalizeChannelSyncKey(s string) string {
	return strings.TrimSpace(s)
}

// IsValidChannelSyncKey 校验手填/导入的 sync_key 格式（非空、长度、可打印 ASCII）。
func IsValidChannelSyncKey(s string) bool {
	s = NormalizeChannelSyncKey(s)
	if s == "" || len(s) > channelSyncKeyMaxLen {
		return false
	}
	for _, c := range s {
		if c < 33 || c > 126 {
			return false
		}
	}
	return true
}

// EnsureChannelSyncKey 若 sync_key 为空则生成唯一短码；非空则规范化。
func EnsureChannelSyncKey(ch *Channel) {
	if ch == nil {
		return
	}
	key := NormalizeChannelSyncKey(ch.SyncKey)
	if key != "" {
		ch.SyncKey = key
		return
	}
	if DB != nil {
		if allocated, err := AllocateUniqueChannelSyncKey(ch.Id); err == nil {
			ch.SyncKey = allocated
			return
		}
	}
	ch.SyncKey = NewChannelSyncKey()
}

// GetChannelBySyncKey 按 sync_key 查找渠道；未找到返回 (nil, nil)。
func GetChannelBySyncKey(syncKey string) (*Channel, error) {
	syncKey = NormalizeChannelSyncKey(syncKey)
	if syncKey == "" {
		return nil, nil
	}
	var channel Channel
	err := DB.Where("sync_key = ?", syncKey).First(&channel).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &channel, nil
}

// ValidateChannelSyncKeyUnique 校验 sync_key 未被其他渠道占用（空串不校验）。
func ValidateChannelSyncKeyUnique(excludeChannelID int, syncKey string) error {
	syncKey = NormalizeChannelSyncKey(syncKey)
	if syncKey == "" {
		return nil
	}
	if !IsValidChannelSyncKey(syncKey) {
		return fmt.Errorf("sync_key 格式无效（1～%d 位可打印 ASCII）", channelSyncKeyMaxLen)
	}
	if DB == nil {
		return nil
	}
	q := DB.Model(&Channel{}).Where("sync_key = ?", syncKey)
	if excludeChannelID > 0 {
		q = q.Where("id <> ?", excludeChannelID)
	}
	var cnt int64
	if err := q.Count(&cnt).Error; err != nil {
		return err
	}
	if cnt > 0 {
		return fmt.Errorf("sync_key 已被其他渠道占用")
	}
	return nil
}

// BackfillChannelSyncKeys 为缺少 sync_key 的渠道写入短码（幂等），并确保唯一索引。
func BackfillChannelSyncKeys() error {
	if DB == nil || DB.Migrator() == nil {
		return nil
	}
	if !DB.Migrator().HasColumn(&Channel{}, "sync_key") {
		return nil
	}
	var ids []int
	if err := DB.Model(&Channel{}).Where("sync_key IS NULL OR sync_key = ?", "").Pluck("id", &ids).Error; err != nil {
		return err
	}
	for _, id := range ids {
		key, err := AllocateUniqueChannelSyncKey(id)
		if err != nil {
			return fmt.Errorf("backfill sync_key channel_id=%d: %w", id, err)
		}
		if err := DB.Model(&Channel{}).Where("id = ? AND (sync_key IS NULL OR sync_key = ?)", id, "").
			Update("sync_key", key).Error; err != nil {
			return fmt.Errorf("backfill sync_key channel_id=%d: %w", id, err)
		}
	}
	return ensureSyncKeyUniqueIndex()
}

func ensureSyncKeyUniqueIndex() error {
	sql := "CREATE UNIQUE INDEX IF NOT EXISTS idx_channels_sync_key_unique ON channels (sync_key)"
	if common.UsingMySQL {
		sql = "CREATE UNIQUE INDEX idx_channels_sync_key_unique ON channels (sync_key)"
	}
	err := DB.Exec(sql).Error
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "duplicate") || strings.Contains(msg, "already exists") || strings.Contains(msg, "exist") {
		return nil
	}
	return fmt.Errorf("ensure sync_key unique index: %w", err)
}
