package model

import (
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
)

func sqlUnixToYMDColumn(col string) string {
	switch {
	case common.UsingPostgreSQL:
		return fmt.Sprintf("TO_CHAR(TO_TIMESTAMP(%s), 'YYYY-MM-DD')", col)
	case common.UsingMySQL:
		return fmt.Sprintf("DATE_FORMAT(FROM_UNIXTIME(%s), '%%Y-%%m-%%d')", col)
	default:
		return fmt.Sprintf("strftime('%%Y-%%m-%%d', %s, 'unixepoch')", col)
	}
}

// DistributorAnalyticsDay 单日聚合（分销商看板与管理端序列共用形状）。
type DistributorAnalyticsDay struct {
	Date              string `json:"date"`
	ShortLinkClicks   int    `json:"short_link_clicks"`
	RegisterPageViews int    `json:"register_page_views"`
	NewRegistrations  int    `json:"new_registrations"`
	RewardQuota       int64  `json:"reward_quota"`
	InviteeQuotaAdded int64  `json:"invitee_quota_added"`
}

type aggCountRow struct {
	Day   string `gorm:"column:day"`
	Count int64  `gorm:"column:cnt"`
}

type aggSumRow struct {
	Day   string `gorm:"column:day"`
	Sum   int64  `gorm:"column:sum"`
	SumIn int64  `gorm:"column:sum_in"`
}

// InviteeTopAnalyticsRow 被邀请人排行（当前分销商维度）。
type InviteeTopAnalyticsRow struct {
	InviteeUserId       int    `json:"invitee_user_id"`
	Username            string `json:"username"`
	DisplayName         string `json:"display_name"`
	TotalRewardQuota    int64  `json:"total_reward_quota"`
	PeriodRewardQuota   int64  `json:"period_reward_quota"`
	TotalInviteeQuotaIn int64  `json:"total_invitee_quota_added"`
}

// DistributorAdminTopRow 管理端分销商排行一行。
type DistributorAdminTopRow struct {
	UserId           int    `json:"user_id"`
	Username         string `json:"username"`
	DisplayName      string `json:"display_name"`
	AffCode          string `json:"aff_code"`
	TotalRewardQuota int64  `json:"total_reward_quota"`
	PeriodRewardQuota int64 `json:"period_reward_quota"`
	InviteeCount      int64 `json:"invitee_count"`
}

func distributorUserJoinSQL(alias string) string {
	// 与 UserIsDistributor 一致：非管理员且（is_distributor=1 或历史 role=5）
	return fmt.Sprintf(`INNER JOIN users u ON u.id = %s.inviter_id AND u.role < %d AND (u.is_distributor = %d OR u.role = %d)`,
		alias, common.RoleAdminUser, common.DistributorFlagYes, common.RoleDistributorUser)
}

// BuildDistributorSelfAnalytics 合并漏斗表、注册关系、分成日志，生成连续日期序列。
func BuildDistributorSelfAnalytics(inviterId int, days int) ([]DistributorAnalyticsDay, error) {
	if inviterId <= 0 || days <= 0 {
		return []DistributorAnalyticsDay{}, nil
	}
	if days > 90 {
		days = 90
	}
	end := time.Now().UTC().Truncate(24 * time.Hour)
	start := end.AddDate(0, 0, -(days - 1))
	dateFrom := start.Format("2006-01-02")
	dateTo := end.Format("2006-01-02")
	startUnix := start.Unix()
	endUnix := end.AddDate(0, 0, 1).Unix() // [start, end] 闭区间按 created_at < 次日

	funnelRows, err := ListAffFunnelDailyForInviter(inviterId, dateFrom, dateTo)
	if err != nil {
		return nil, err
	}
	funnelMap := make(map[string]AffFunnelDaily, len(funnelRows))
	for _, r := range funnelRows {
		funnelMap[r.StatDate] = r
	}

	regMap, err := countAffRegistrationsByDay(inviterId, startUnix, endUnix)
	if err != nil {
		return nil, err
	}
	rewMap, inMap, err := sumAffCommissionByDay(inviterId, startUnix, endUnix)
	if err != nil {
		return nil, err
	}

	var dates []string
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		dates = append(dates, d.Format("2006-01-02"))
	}
	out := make([]DistributorAnalyticsDay, 0, len(dates))
	for _, ds := range dates {
		f := funnelMap[ds]
		out = append(out, DistributorAnalyticsDay{
			Date:              ds,
			ShortLinkClicks:   f.ShortLinkClicks,
			RegisterPageViews: f.RegisterPageViews,
			NewRegistrations:  int(regMap[ds]),
			RewardQuota:       rewMap[ds],
			InviteeQuotaAdded: inMap[ds],
		})
	}
	return out, nil
}

func countAffRegistrationsByDay(inviterId int, startUnix, endUnix int64) (map[string]int64, error) {
	dayExpr := sqlUnixToYMDColumn("created_at")
	var rows []aggCountRow
	err := DB.Model(&AffInviteRelation{}).
		Select(dayExpr+" AS day, COUNT(*) AS cnt").
		Where("inviter_id = ? AND created_at >= ? AND created_at < ?", inviterId, startUnix, endUnix).
		Group(dayExpr).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	m := make(map[string]int64, len(rows))
	for _, r := range rows {
		m[r.Day] = r.Count
	}
	return m, nil
}

func sumAffCommissionByDay(inviterId int, startUnix, endUnix int64) (reward map[string]int64, inviteeIn map[string]int64, err error) {
	dayExpr := sqlUnixToYMDColumn("created_at")
	var rows []aggSumRow
	q := DB.Model(&AffInviteCommissionLog{}).
		Select(dayExpr+" AS day, COALESCE(SUM(reward_quota),0) AS sum, COALESCE(SUM(invitee_quota_added),0) AS sum_in").
		Where("inviter_id = ? AND created_at >= ? AND created_at < ?", inviterId, startUnix, endUnix).
		Group(dayExpr)
	err = q.Scan(&rows).Error
	if err != nil {
		return nil, nil, err
	}
	reward = make(map[string]int64, len(rows))
	inviteeIn = make(map[string]int64, len(rows))
	for _, r := range rows {
		reward[r.Day] = r.Sum
		inviteeIn[r.Day] = r.SumIn
	}
	return reward, inviteeIn, nil
}

// ListInviteeTopForDistributorAnalytics 当前分销商下被邀请人 TOP（按累计收益，附近 7 日收益）。
func ListInviteeTopForDistributorAnalytics(inviterId int, topN int) ([]InviteeTopAnalyticsRow, error) {
	if inviterId <= 0 {
		return []InviteeTopAnalyticsRow{}, nil
	}
	if topN <= 0 {
		topN = 10
	}
	if topN > 50 {
		topN = 50
	}
	now := time.Now().UTC()
	periodStart := now.AddDate(0, 0, -7).Unix()

	type sumRow struct {
		InviteeUserId     int   `gorm:"column:invitee_user_id"`
		TotalReward       int64 `gorm:"column:total_reward"`
		PeriodReward      int64 `gorm:"column:period_reward"`
		TotalInviteeQuota int64 `gorm:"column:total_invitee_quota"`
	}
	var sums []sumRow
	sel := fmt.Sprintf(`invitee_user_id,
			COALESCE(SUM(reward_quota),0) AS total_reward,
			COALESCE(SUM(CASE WHEN created_at >= %d THEN reward_quota ELSE 0 END),0) AS period_reward,
			COALESCE(SUM(invitee_quota_added),0) AS total_invitee_quota`, periodStart)
	err := DB.Model(&AffInviteCommissionLog{}).
		Select(sel).
		Where("inviter_id = ?", inviterId).
		Group("invitee_user_id").
		Order("total_reward DESC").
		Limit(topN).
		Scan(&sums).Error
	if err != nil {
		return nil, err
	}
	if len(sums) == 0 {
		return []InviteeTopAnalyticsRow{}, nil
	}
	ids := make([]int, 0, len(sums))
	for _, s := range sums {
		ids = append(ids, s.InviteeUserId)
	}
	var users []User
	_ = DB.Select("id", "username", "display_name").Where("id IN ?", ids).Find(&users).Error
	uMap := make(map[int]User, len(users))
	for _, u := range users {
		uMap[u.Id] = u
	}
	out := make([]InviteeTopAnalyticsRow, 0, len(sums))
	for _, s := range sums {
		u := uMap[s.InviteeUserId]
		out = append(out, InviteeTopAnalyticsRow{
			InviteeUserId:       s.InviteeUserId,
			Username:            u.Username,
			DisplayName:         u.DisplayName,
			TotalRewardQuota:    s.TotalReward,
			PeriodRewardQuota:   s.PeriodReward,
			TotalInviteeQuotaIn: s.TotalInviteeQuota,
		})
	}
	return out, nil
}

// BuildPlatformAffiliateAnalytics 管理端：全平台按日序列 + 分销商排行。
func BuildPlatformAffiliateAnalytics(days int) (series []DistributorAnalyticsDay, topTotal, topPeriod, topInvite []DistributorAdminTopRow, err error) {
	if days <= 0 {
		days = 30
	}
	if days > 90 {
		days = 90
	}
	end := time.Now().UTC().Truncate(24 * time.Hour)
	start := end.AddDate(0, 0, -(days - 1))
	dateFrom := start.Format("2006-01-02")
	dateTo := end.Format("2006-01-02")
	startUnix := start.Unix()
	endUnix := end.AddDate(0, 0, 1).Unix()

	funnelPlat, err := SumAffFunnelDailyPlatform(dateFrom, dateTo)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	regPlat, err := countAllAffRegistrationsByDay(startUnix, endUnix)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	rewPlat, inPlat, err := sumAllAffCommissionByDay(startUnix, endUnix)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	var dates []string
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		dates = append(dates, d.Format("2006-01-02"))
	}
	series = make([]DistributorAnalyticsDay, 0, len(dates))
	for _, ds := range dates {
		f := funnelPlat[ds]
		series = append(series, DistributorAnalyticsDay{
			Date:              ds,
			ShortLinkClicks:   f.Clicks,
			RegisterPageViews: f.RegViews,
			NewRegistrations:  int(regPlat[ds]),
			RewardQuota:       rewPlat[ds],
			InviteeQuotaAdded: inPlat[ds],
		})
	}

	topTotal, err = listAdminTopDistributorsByReward(0, 20)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	periodStart := time.Now().UTC().AddDate(0, 0, -30).Unix()
	topPeriod, err = listAdminTopDistributorsByReward(periodStart, 20)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	topInvite, err = listAdminTopDistributorsByInviteeCount(20)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return series, topTotal, topPeriod, topInvite, nil
}

func countAllAffRegistrationsByDay(startUnix, endUnix int64) (map[string]int64, error) {
	dayExpr := sqlUnixToYMDColumn("created_at")
	var rows []aggCountRow
	err := DB.Model(&AffInviteRelation{}).
		Select(dayExpr+" AS day, COUNT(*) AS cnt").
		Where("created_at >= ? AND created_at < ?", startUnix, endUnix).
		Group(dayExpr).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	m := make(map[string]int64, len(rows))
	for _, r := range rows {
		m[r.Day] = r.Count
	}
	return m, nil
}

func sumAllAffCommissionByDay(startUnix, endUnix int64) (reward map[string]int64, inviteeIn map[string]int64, err error) {
	dayExpr := sqlUnixToYMDColumn("created_at")
	var rows []aggSumRow
	err = DB.Model(&AffInviteCommissionLog{}).
		Select(dayExpr+" AS day, COALESCE(SUM(reward_quota),0) AS sum, COALESCE(SUM(invitee_quota_added),0) AS sum_in").
		Where("created_at >= ? AND created_at < ?", startUnix, endUnix).
		Group(dayExpr).
		Scan(&rows).Error
	if err != nil {
		return nil, nil, err
	}
	reward = make(map[string]int64, len(rows))
	inviteeIn = make(map[string]int64, len(rows))
	for _, r := range rows {
		reward[r.Day] = r.Sum
		inviteeIn[r.Day] = r.SumIn
	}
	return reward, inviteeIn, nil
}

func listAdminTopDistributorsByReward(periodStartUnix int64, limit int) ([]DistributorAdminTopRow, error) {
	type row struct {
		InviterId int   `gorm:"column:inviter_id"`
		Sum       int64 `gorm:"column:sum_reward"`
	}
	var sums []row
	q := DB.Model(&AffInviteCommissionLog{}).Table("aff_invite_commission_logs AS l").
		Select("l.inviter_id, COALESCE(SUM(l.reward_quota),0) AS sum_reward").
		Joins(distributorUserJoinSQL("l")).
		Group("l.inviter_id")
	if periodStartUnix > 0 {
		q = q.Where("l.created_at >= ?", periodStartUnix)
	}
	err := q.Order("sum_reward DESC").Limit(limit).Scan(&sums).Error
	if err != nil {
		return nil, err
	}
	out := make([]DistributorAdminTopRow, 0, len(sums))
	for _, s := range sums {
		out = append(out, DistributorAdminTopRow{
			UserId:           s.InviterId,
			TotalRewardQuota: s.Sum,
		})
	}
	if len(out) == 0 {
		return out, nil
	}
	ids := make([]int, len(out))
	for i := range out {
		ids[i] = out[i].UserId
	}
	var users []User
	_ = DB.Select("id", "username", "display_name", "aff_code").Where("id IN ?", ids).Find(&users).Error
	uMap := make(map[int]User, len(users))
	for _, u := range users {
		uMap[u.Id] = u
	}
	type ic struct {
		InviterId int   `gorm:"column:inviter_id"`
		Cnt       int64 `gorm:"column:cnt"`
	}
	var icRows []ic
	_ = DB.Model(&User{}).Select("inviter_id, COUNT(*) AS cnt").Where("inviter_id IN ?", ids).Group("inviter_id").Scan(&icRows).Error
	icMap := make(map[int]int64, len(icRows))
	for _, r := range icRows {
		icMap[r.InviterId] = r.Cnt
	}
	for i := range out {
		u := uMap[out[i].UserId]
		out[i].Username = u.Username
		out[i].DisplayName = u.DisplayName
		out[i].AffCode = u.AffCode
		out[i].InviteeCount = icMap[out[i].UserId]
	}
	return out, nil
}

func listAdminTopDistributorsByInviteeCount(limit int) ([]DistributorAdminTopRow, error) {
	type row struct {
		InviterId int   `gorm:"column:inviter_id"`
		Cnt       int64 `gorm:"column:cnt"`
	}
	var sums []row
	err := DB.Model(&User{}).Table("users AS u").
		Select("u.id AS inviter_id, COUNT(c.id) AS cnt").
		Joins("LEFT JOIN users c ON c.inviter_id = u.id").
		Where("u.role < ? AND (u.is_distributor = ? OR u.role = ?)", common.RoleAdminUser, common.DistributorFlagYes, common.RoleDistributorUser).
		Group("u.id").
		Order("cnt DESC").
		Limit(limit).
		Scan(&sums).Error
	if err != nil {
		return nil, err
	}
	out := make([]DistributorAdminTopRow, 0, len(sums))
	for _, s := range sums {
		out = append(out, DistributorAdminTopRow{
			UserId:         s.InviterId,
			InviteeCount: s.Cnt,
		})
	}
	if len(out) == 0 {
		return out, nil
	}
	ids := make([]int, len(out))
	for i := range out {
		ids[i] = out[i].UserId
	}
	var users []User
	_ = DB.Select("id", "username", "display_name", "aff_code").Where("id IN ?", ids).Find(&users).Error
	uMap := make(map[int]User, len(users))
	for _, u := range users {
		uMap[u.Id] = u
	}
	for i := range out {
		u := uMap[out[i].UserId]
		out[i].Username = u.Username
		out[i].DisplayName = u.DisplayName
		out[i].AffCode = u.AffCode
	}
	return out, nil
}
