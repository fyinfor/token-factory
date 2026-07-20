package model

import "github.com/QuantumNous/new-api/common"

type AliyunGuardrailLog struct {
	Id        int    `json:"id" gorm:"primaryKey"`
	CreatedAt int64  `json:"created_at" gorm:"bigint;index"`
	UserId    int    `json:"user_id" gorm:"index"`
	Username  string `json:"username" gorm:"index;default:''"`
	ModelName string `json:"model_name" gorm:"index;default:''"`
	ChannelId int    `json:"channel" gorm:"index"`
	RequestId string `json:"request_id" gorm:"type:varchar(64);index;default:''"`
	RouteSlug string `json:"route_slug" gorm:"-"`
	Direction string `json:"direction" gorm:"type:varchar(16);index"`
	RiskLevel string `json:"risk_level" gorm:"type:varchar(16);index"`
	Service   string `json:"service" gorm:"type:varchar(64);default:''"`
	Content   string `json:"content" gorm:"type:text"`
	Detail    string `json:"detail" gorm:"type:text"`
}

type AliyunGuardrailLogFilter struct {
	Username       string
	StartTimestamp int64
	EndTimestamp   int64
}

func CreateAliyunGuardrailLog(log *AliyunGuardrailLog) error { return DB.Create(log).Error }

func GetAliyunGuardrailLogs(startIdx, num int, filter AliyunGuardrailLogFilter) ([]AliyunGuardrailLog, int64, error) {
	var logs []AliyunGuardrailLog
	var total int64
	query := DB.Model(&AliyunGuardrailLog{})
	if filter.Username != "" {
		query = query.Where("username = ?", filter.Username)
	}
	if filter.StartTimestamp > 0 {
		query = query.Where("created_at >= ?", filter.StartTimestamp)
	}
	if filter.EndTimestamp > 0 {
		query = query.Where("created_at <= ?", filter.EndTimestamp)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Order("id desc").Offset(startIdx).Limit(num).Find(&logs).Error
	attachAliyunGuardrailRouteSlugs(logs)
	return logs, total, err
}

func NewAliyunGuardrailLog() *AliyunGuardrailLog {
	return &AliyunGuardrailLog{CreatedAt: common.GetTimestamp()}
}

func attachAliyunGuardrailRouteSlugs(logs []AliyunGuardrailLog) {
	if len(logs) == 0 {
		return
	}
	channelIDs := make([]int, 0, len(logs))
	seen := make(map[int]struct{}, len(logs))
	for i := range logs {
		channelID := logs[i].ChannelId
		if channelID <= 0 {
			continue
		}
		if _, ok := seen[channelID]; ok {
			continue
		}
		seen[channelID] = struct{}{}
		channelIDs = append(channelIDs, channelID)
	}
	if len(channelIDs) == 0 {
		return
	}
	routeSlugMap := GetRouteSlugsByChannelIDs(channelIDs)
	for i := range logs {
		channelID := logs[i].ChannelId
		if channelID <= 0 {
			continue
		}
		routeSlug := routeSlugMap[channelID]
		if routeSlug == "" {
			routeSlug = DefaultRouteSlugFromChannelID(int64(channelID))
		}
		logs[i].RouteSlug = routeSlug
	}
}
