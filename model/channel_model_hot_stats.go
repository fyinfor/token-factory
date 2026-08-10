package model

import (
	"sync"
	"time"
)

const channelModelHotStatsCacheTTL = time.Minute

type channelModelHotStatsCacheEntry struct {
	expiresAt time.Time
	stats     []ChannelModelRequestStats
}

var channelModelHotStatsCache = struct {
	sync.Mutex
	entries map[string]channelModelHotStatsCacheEntry
}{entries: make(map[string]channelModelHotStatsCacheEntry)}

// GetCachedChannelModelRequestStatsByPeriod returns global channel-model
// request counts. The short cache avoids scanning the logs table for every
// public pricing request while keeping homepage popularity reasonably fresh.
func GetCachedChannelModelRequestStatsByPeriod(period string) ([]ChannelModelRequestStats, error) {
	channelModelHotStatsCache.Lock()
	defer channelModelHotStatsCache.Unlock()

	now := time.Now()
	if entry, ok := channelModelHotStatsCache.entries[period]; ok && now.Before(entry.expiresAt) {
		return append([]ChannelModelRequestStats(nil), entry.stats...), nil
	}

	startTime := int64(0)
	switch period {
	case HeatStatPeriod30d:
		startTime = now.AddDate(0, 0, -30).Unix()
	case HeatStatPeriodAll:
		startTime = 0
	default:
		startTime = now.AddDate(0, 0, -7).Unix()
	}

	db := LOG_DB
	if db == nil {
		db = DB
	}
	var stats []ChannelModelRequestStats
	query := db.Model(&Log{}).
		Select("channel_id, model_name, COUNT(*) AS req_count_7d, COUNT(*) AS req_count_30d").
		Where("type = ? AND model_name <> '' AND channel_id > 0", LogTypeConsume)
	if startTime > 0 {
		query = query.Where("created_at >= ?", startTime)
	}
	if err := query.Group("channel_id, model_name").Scan(&stats).Error; err != nil {
		return nil, err
	}

	channelModelHotStatsCache.entries[period] = channelModelHotStatsCacheEntry{
		expiresAt: now.Add(channelModelHotStatsCacheTTL),
		stats:     append([]ChannelModelRequestStats(nil), stats...),
	}
	return stats, nil
}

func InvalidateChannelModelHotStatsCache() {
	channelModelHotStatsCache.Lock()
	channelModelHotStatsCache.entries = make(map[string]channelModelHotStatsCacheEntry)
	channelModelHotStatsCache.Unlock()
}
