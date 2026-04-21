package model

import (
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AffFunnelDaily 分销商邀请漏斗按日汇总：短链点击、带邀请码注册页浏览（UTC 日期维度）。
type AffFunnelDaily struct {
	Id                int    `json:"id" gorm:"primaryKey;autoIncrement"`
	InviterId         int    `json:"inviter_id" gorm:"not null;uniqueIndex:idx_aff_funnel_inv_day,priority:1;index:idx_aff_funnel_inv_date,priority:1"`
	StatDate          string `json:"stat_date" gorm:"type:varchar(10);not null;uniqueIndex:idx_aff_funnel_inv_day,priority:2;index:idx_aff_funnel_inv_date,priority:2;column:stat_date"` // YYYY-MM-DD UTC
	ShortLinkClicks   int    `json:"short_link_clicks" gorm:"not null;default:0"`
	RegisterPageViews int    `json:"register_page_views" gorm:"not null;default:0"`
}

func (AffFunnelDaily) TableName() string {
	return "aff_funnel_daily"
}

// UpsertAffFunnelIncrShortLink 短链 /r/:code 点击 +1（按 inviter 与 UTC 日期）。
func UpsertAffFunnelIncrShortLink(inviterId int, statDate string) error {
	if inviterId <= 0 {
		return nil
	}
	statDate = strings.TrimSpace(statDate)
	if statDate == "" {
		return nil
	}
	row := AffFunnelDaily{
		InviterId:         inviterId,
		StatDate:          statDate,
		ShortLinkClicks:   1,
		RegisterPageViews: 0,
	}
	return DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "inviter_id"}, {Name: "stat_date"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"short_link_clicks": gorm.Expr("short_link_clicks + ?", 1),
		}),
	}).Create(&row).Error
}

// UpsertAffFunnelIncrRegisterPageView 注册页带 aff 参数浏览 +1。
func UpsertAffFunnelIncrRegisterPageView(inviterId int, statDate string) error {
	if inviterId <= 0 {
		return nil
	}
	statDate = strings.TrimSpace(statDate)
	if statDate == "" {
		return nil
	}
	row := AffFunnelDaily{
		InviterId:         inviterId,
		StatDate:          statDate,
		ShortLinkClicks:   0,
		RegisterPageViews: 1,
	}
	return DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "inviter_id"}, {Name: "stat_date"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"register_page_views": gorm.Expr("register_page_views + ?", 1),
		}),
	}).Create(&row).Error
}

// ListAffFunnelDailyForInviter 返回 inviter 在 [dateFrom, dateTo]（含）内的漏斗日表行。
func ListAffFunnelDailyForInviter(inviterId int, dateFrom, dateTo string) ([]AffFunnelDaily, error) {
	if inviterId <= 0 {
		return []AffFunnelDaily{}, nil
	}
	var rows []AffFunnelDaily
	err := DB.Where("inviter_id = ? AND stat_date >= ? AND stat_date <= ?", inviterId, dateFrom, dateTo).
		Order("stat_date ASC").Find(&rows).Error
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []AffFunnelDaily{}
	}
	return rows, nil
}

// SumAffFunnelDailyPlatform 按日汇总全平台漏斗（管理端大盘）。
func SumAffFunnelDailyPlatform(dateFrom, dateTo string) (map[string]struct{ Clicks, RegViews int }, error) {
	out := make(map[string]struct{ Clicks, RegViews int })
	type row struct {
		StatDate          string
		ShortLinkClicks   int
		RegisterPageViews int
	}
	var list []row
	err := DB.Model(&AffFunnelDaily{}).
		Select("stat_date, COALESCE(SUM(short_link_clicks),0) AS short_link_clicks, COALESCE(SUM(register_page_views),0) AS register_page_views").
		Where("stat_date >= ? AND stat_date <= ?", dateFrom, dateTo).
		Group("stat_date").
		Order("stat_date ASC").
		Scan(&list).Error
	if err != nil {
		return nil, err
	}
	for _, r := range list {
		out[r.StatDate] = struct{ Clicks, RegViews int }{r.ShortLinkClicks, r.RegisterPageViews}
	}
	return out, nil
}
