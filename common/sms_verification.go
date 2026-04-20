package common

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var mainlandChinaPhoneRegexp = regexp.MustCompile(`^1[3-9]\d{9}$`)

// NormalizePhone 标准化手机号（去空格）。
func NormalizePhone(phone string) string {
	normalized := strings.TrimSpace(phone)
	normalized = strings.ReplaceAll(normalized, " ", "")
	normalized = strings.ReplaceAll(normalized, "-", "")
	normalized = strings.ReplaceAll(normalized, "(", "")
	normalized = strings.ReplaceAll(normalized, ")", "")
	if strings.HasPrefix(normalized, "+86") {
		normalized = strings.TrimPrefix(normalized, "+86")
	} else if strings.HasPrefix(normalized, "0086") {
		normalized = strings.TrimPrefix(normalized, "0086")
	} else if len(normalized) == 13 && strings.HasPrefix(normalized, "86") {
		normalized = strings.TrimPrefix(normalized, "86")
	}
	return normalized
}

// ValidateMainlandChinaPhone 校验中国大陆 11 位手机号格式。
func ValidateMainlandChinaPhone(phone string) bool {
	return mainlandChinaPhoneRegexp.MatchString(NormalizePhone(phone))
}

// SMSVerificationCodeKey 返回短信验证码 Redis Key。
func SMSVerificationCodeKey(phone string) string {
	return "sms:code:" + NormalizePhone(phone)
}

// SMSVerificationCooldownKey 返回短信冷却 Redis Key。
func SMSVerificationCooldownKey(phone string) string {
	return "sms:cooldown:" + NormalizePhone(phone)
}

// SMSVerificationDailyCountKey 返回短信日计数 Redis Key。
func SMSVerificationDailyCountKey(phone string, now time.Time) string {
	return "sms:daily:" + NormalizePhone(phone) + ":" + now.Format("20060102")
}

// EnsureRedisEnabledForSMS 短信验证码依赖 Redis；未启用时返回错误。
func EnsureRedisEnabledForSMS() error {
	if !RedisEnabled || RDB == nil {
		return fmt.Errorf("短信验证码服务未启用，请先配置 Redis")
	}
	return nil
}

// IsSMSPhoneBlacklisted 判断手机号是否在短信黑名单中。
func IsSMSPhoneBlacklisted(phone string) bool {
	phone = NormalizePhone(phone)
	for _, blocked := range SMSPhoneBlacklist {
		if NormalizePhone(blocked) == phone {
			return true
		}
	}
	return false
}

// CheckSMSCanSend 校验手机号是否满足发送频率限制。
func CheckSMSCanSend(phone string) error {
	if err := EnsureRedisEnabledForSMS(); err != nil {
		return err
	}
	phone = NormalizePhone(phone)
	ctx := context.Background()

	cooldownKey := SMSVerificationCooldownKey(phone)
	exists, err := RDB.Exists(ctx, cooldownKey).Result()
	if err != nil {
		return fmt.Errorf("读取短信冷却状态失败: %w", err)
	}
	if exists > 0 {
		return fmt.Errorf("发送过于频繁，请 %d 分钟后再试", SMSCodeCooldownMinutes)
	}

	dailyKey := SMSVerificationDailyCountKey(phone, time.Now())
	countStr, err := RDB.Get(ctx, dailyKey).Result()
	if err == nil {
		count, parseErr := strconv.Atoi(countStr)
		if parseErr == nil && count >= SMSCodeDailyLimit {
			return fmt.Errorf("该手机号今日发送次数已达上限（%d 次）", SMSCodeDailyLimit)
		}
	}
	return nil
}

// RecordSMSSend 成功发送短信后，记录冷却与当日计数。
func RecordSMSSend(phone string) error {
	if err := EnsureRedisEnabledForSMS(); err != nil {
		return err
	}
	phone = NormalizePhone(phone)
	ctx := context.Background()

	cooldownKey := SMSVerificationCooldownKey(phone)
	if err := RDB.Set(ctx, cooldownKey, "1", time.Duration(SMSCodeCooldownMinutes)*time.Minute).Err(); err != nil {
		return fmt.Errorf("写入短信冷却状态失败: %w", err)
	}

	now := time.Now()
	dailyKey := SMSVerificationDailyCountKey(phone, now)
	count, err := RDB.Incr(ctx, dailyKey).Result()
	if err != nil {
		return fmt.Errorf("更新短信日计数失败: %w", err)
	}
	if count == 1 {
		nextDay := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		expire := time.Until(nextDay)
		if expire <= 0 {
			expire = 24 * time.Hour
		}
		if err := RDB.Expire(ctx, dailyKey, expire).Err(); err != nil {
			return fmt.Errorf("设置短信日计数过期失败: %w", err)
		}
	}
	return nil
}

// StoreSMSVerificationCode 保存短信验证码，默认 5 分钟过期。
func StoreSMSVerificationCode(phone, code string) error {
	if err := EnsureRedisEnabledForSMS(); err != nil {
		return err
	}
	ctx := context.Background()
	key := SMSVerificationCodeKey(phone)
	return RDB.Set(ctx, key, code, time.Duration(SMSCodeValidMinutes)*time.Minute).Err()
}

// VerifyAndConsumeSMSCode 校验短信验证码并在成功后删除，避免重复使用。
func VerifyAndConsumeSMSCode(phone, code string) bool {
	if err := EnsureRedisEnabledForSMS(); err != nil {
		return false
	}
	ctx := context.Background()
	key := SMSVerificationCodeKey(phone)
	val, err := RDB.Get(ctx, key).Result()
	if err != nil || strings.TrimSpace(val) == "" {
		return false
	}
	if strings.TrimSpace(val) != strings.TrimSpace(code) {
		return false
	}
	_ = RDB.Del(ctx, key).Err()
	return true
}
