package model

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

var cnyAmountInLogRe = regexp.MustCompile(`¥\s*([0-9]+(?:\.[0-9]+)?)`)

// creditQuotaFromCNYDisplay 把日志里的「¥xx」展示金额还原为内部额度。
func creditQuotaFromCNYDisplay(cny float64) int64 {
	if cny <= 0 || common.QuotaPerUnit <= 0 {
		return 0
	}
	rate := operation_setting.USDExchangeRate
	if rate <= 0 {
		rate = 1
	}
	return int64(math.Round(cny / rate * common.QuotaPerUnit))
}

func parseCreditQuotaFromLogContent(content string) int64 {
	content = strings.TrimSpace(content)
	if content == "" {
		return 0
	}
	m := cnyAmountInLogRe.FindStringSubmatch(content)
	if len(m) < 2 {
		return 0
	}
	cny, err := strconv.ParseFloat(m[1], 64)
	if err != nil || cny <= 0 {
		return 0
	}
	return creditQuotaFromCNYDisplay(cny)
}

// SumUserExternalCreditQuota 统计用户外部到账额度（充值 + 赠送/系统入账等），不含消费退款循环。
// 用于保证展示恒等式：剩余 + 已用 ≈ 累计到账。
func SumUserExternalCreditQuota(userID int) (int64, error) {
	if userID <= 0 {
		return 0, fmt.Errorf("invalid user id")
	}
	var topupSum int64
	if err := DB.Model(&TopUp{}).
		Where("user_id = ? AND status = ?", userID, common.TopUpStatusSuccess).
		Select("COALESCE(SUM(quota_to_add),0)").Scan(&topupSum).Error; err != nil {
		return 0, err
	}

	// 管理调整：若日志带正数 quota，直接计入到账。
	var manageSum int64
	if err := LOG_DB.Model(&Log{}).
		Where("user_id = ? AND type = ? AND quota > 0", userID, LogTypeManage).
		Select("COALESCE(SUM(quota),0)").Scan(&manageSum).Error; err != nil {
		return 0, err
	}

	// 系统/赠送日志：quota 字段常为 0，从 content 的 ¥ 金额还原。
	var sysLogs []Log
	if err := LOG_DB.Model(&Log{}).
		Select("content", "quota").
		Where("user_id = ? AND type = ?", userID, LogTypeSystem).
		Find(&sysLogs).Error; err != nil {
		return 0, err
	}
	var giftSum int64
	for _, l := range sysLogs {
		if l.Quota > 0 {
			giftSum += int64(l.Quota)
			continue
		}
		giftSum += parseCreditQuotaFromLogContent(l.Content)
	}

	return topupSum + manageSum + giftSum, nil
}

// ReconcileUserUsedQuotaFromLogs 按「累计到账 − 当前剩余」重算累积已用。
// used = max(0, external_credits - remain)
// 这样剩余 + 已用 = 累计到账，不会出现充值 300 却「剩余+已用=275」的错账观感。
func ReconcileUserUsedQuotaFromLogs(userID int) (oldUsed, newUsed int, err error) {
	if userID <= 0 {
		return 0, 0, fmt.Errorf("invalid user id")
	}
	var user User
	if err = DB.Select("id", "quota", "used_quota").Where("id = ?", userID).First(&user).Error; err != nil {
		return 0, 0, err
	}
	oldUsed = user.UsedQuota

	credits, err := SumUserExternalCreditQuota(userID)
	if err != nil {
		return oldUsed, 0, err
	}
	newUsed = int(credits) - user.Quota
	if newUsed < 0 {
		newUsed = 0
	}
	if newUsed == oldUsed {
		return oldUsed, newUsed, nil
	}
	if err = DB.Model(&User{}).Where("id = ?", userID).Update("used_quota", newUsed).Error; err != nil {
		return oldUsed, newUsed, err
	}
	common.SysLog(fmt.Sprintf(
		"reconciled user used_quota: userId=%d old=%d new=%d credits=%d remain=%d",
		userID, oldUsed, newUsed, credits, user.Quota,
	))
	return oldUsed, newUsed, nil
}

// ReconcileAllUsersUsedQuotaFromLogs 批量按到账−剩余重算所有用户 used_quota。
func ReconcileAllUsersUsedQuotaFromLogs() (updated int, err error) {
	var ids []int
	if err = DB.Model(&User{}).Select("id").Order("id").Pluck("id", &ids).Error; err != nil {
		return 0, err
	}
	for _, id := range ids {
		oldUsed, newUsed, recErr := ReconcileUserUsedQuotaFromLogs(id)
		if recErr != nil {
			common.SysLog(fmt.Sprintf("reconcile used_quota failed userId=%d: %v", id, recErr))
			continue
		}
		if oldUsed != newUsed {
			updated++
		}
	}
	return updated, nil
}
