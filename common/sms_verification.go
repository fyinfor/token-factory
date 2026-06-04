package common

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var mainlandChinaPhoneRegexp = regexp.MustCompile(`^1[3-9]\d{9}$`)
// internationalPhoneRegexp 校验 E.164 国际号（+ 国码 + 至少 4 位、最多 15 位）。不含 +86（已被 NormalizePhone 剥除为 11 位国内号）。
var internationalPhoneRegexp = regexp.MustCompile(`^\+[1-9]\d{4,14}$`)

// NormalizePhone 标准化手机号：
// 1. 去空格/分隔符；
// 2. 若以 +86 / 0086 / 86 开头（仅在 13 位时）剥除为 11 位国内号。
// 国际号（如 +14155552671）保持原样。
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

// IsValidLoginPhone 校验手机号格式：11 位国内号 或 E.164 国际号（+国码+4~15 位）。
func IsValidLoginPhone(phone string) bool {
	return ValidateMainlandChinaPhone(phone) || IsInternationalPhone(phone)
}

// IsMainlandChinaPhone 标准化后判断是否 11 位国内号。
func IsMainlandChinaPhone(phone string) bool {
	return mainlandChinaPhoneRegexp.MatchString(phone)
}

// IsInternationalPhone 标准化后判断是否 E.164 国际号（带 + 前缀）。
func IsInternationalPhone(phone string) bool {
	return internationalPhoneRegexp.MatchString(phone)
}

// SMSRegion 表示手机号所属地区：国内 / 国际。
type SMSRegion string

const (
	SMSRegionDomestic      SMSRegion = "domestic"
	SMSRegionInternational SMSRegion = "international"
)

// ClassifyPhoneRegion 标准化后判断手机号属于国内还是国际。
// - 11 位国内号 → domestic
// - 带 + 前缀的国际号 → international
// - 其它 → 空字符串（无效）
func ClassifyPhoneRegion(phone string) SMSRegion {
	if IsMainlandChinaPhone(phone) {
		return SMSRegionDomestic
	}
	if IsInternationalPhone(phone) {
		return SMSRegionInternational
	}
	return ""
}

// SMSVerificationCodeKey 返回短信验证码 Redis Key（通用：注册/重置/绑定）。
// 不同场景通过调用方传入不同前缀区分，避免登录验证码被其他流程误用。
func SMSVerificationCodeKey(purpose, phone string) string {
	return "sms:code:" + purpose + ":" + NormalizePhone(phone)
}

// SMSLoginVerificationCodeKey 登录短信验证码专用 key，避免与注册/重置/绑定流程的 key 互相冲掉。
func SMSLoginVerificationCodeKey(phone string) string {
	return SMSVerificationCodeKey("login", phone)
}

// SMSVerificationCooldownKey 返回注册/重置/绑定短信冷却 Redis Key。
func SMSVerificationCooldownKey(phone string) string {
	return "sms:cooldown:" + NormalizePhone(phone)
}

// SMSLoginCooldownKey 返回登录短信冷却 Redis Key。
func SMSLoginCooldownKey(phone string) string {
	return "sms:cooldown:login:" + NormalizePhone(phone)
}

// SMSVerificationDailyCountKey 返回注册/重置/绑定短信日计数 Redis Key。
func SMSVerificationDailyCountKey(phone string, now time.Time) string {
	return "sms:daily:" + NormalizePhone(phone) + ":" + now.Format("20060102")
}

// SMSLoginDailyCountKey 返回登录短信日计数 Redis Key。
func SMSLoginDailyCountKey(phone string, now time.Time) string {
	return "sms:daily:login:" + NormalizePhone(phone) + ":" + now.Format("20060102")
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

// CheckSMSLoginCanSend 登录场景冷却/日限检查（更严格：登录涉及账号安全）。
func CheckSMSLoginCanSend(phone string) error {
	if err := EnsureRedisEnabledForSMS(); err != nil {
		return err
	}
	phone = NormalizePhone(phone)
	ctx := context.Background()

	cooldownKey := SMSLoginCooldownKey(phone)
	exists, err := RDB.Exists(ctx, cooldownKey).Result()
	if err != nil {
		return fmt.Errorf("读取登录短信冷却状态失败: %w", err)
	}
	if exists > 0 {
		return fmt.Errorf("发送过于频繁，请 %d 秒后再试", SMSLoginCooldownSeconds)
	}

	dailyKey := SMSLoginDailyCountKey(phone, time.Now())
	countStr, err := RDB.Get(ctx, dailyKey).Result()
	if err == nil {
		count, parseErr := strconv.Atoi(countStr)
		if parseErr == nil && count >= SMSLoginDailyLimit {
			return fmt.Errorf("该手机号今日登录验证码发送次数已达上限（%d 次）", SMSLoginDailyLimit)
		}
	}
	return nil
}

// RecordSMSLoginSend 记录登录短信冷却与当日计数。
func RecordSMSLoginSend(phone string) error {
	if err := EnsureRedisEnabledForSMS(); err != nil {
		return err
	}
	phone = NormalizePhone(phone)
	ctx := context.Background()

	cooldown := SMSLoginCooldownSeconds
	if cooldown <= 0 {
		cooldown = 60
	}
	cooldownKey := SMSLoginCooldownKey(phone)
	if err := RDB.Set(ctx, cooldownKey, "1", time.Duration(cooldown)*time.Second).Err(); err != nil {
		return fmt.Errorf("写入登录短信冷却状态失败: %w", err)
	}

	now := time.Now()
	dailyKey := SMSLoginDailyCountKey(phone, now)
	count, err := RDB.Incr(ctx, dailyKey).Result()
	if err != nil {
		return fmt.Errorf("更新登录短信日计数失败: %w", err)
	}
	if count == 1 {
		nextDay := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		expire := time.Until(nextDay)
		if expire <= 0 {
			expire = 24 * time.Hour
		}
		if err := RDB.Expire(ctx, dailyKey, expire).Err(); err != nil {
			return fmt.Errorf("设置登录短信日计数过期失败: %w", err)
		}
	}
	return nil
}

// CheckSMSCanSend 注册/重置/绑定短信冷却/日限检查。
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

// RecordSMSSend 成功发送注册/重置/绑定短信后，记录冷却与当日计数。
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

// StoreSMSVerificationCode 保存短信验证码（注册/重置/绑定通用），默认 5 分钟过期。
func StoreSMSVerificationCode(phone, code string) error {
	return StoreSMSVerificationCodeWithKey(SMSVerificationCodeKey("common", phone), code, time.Duration(SMSCodeValidMinutes)*time.Minute)
}

// StoreSMSLoginCode 保存登录专用短信验证码。
func StoreSMSLoginCode(phone, code string) error {
	return StoreSMSVerificationCodeWithKey(SMSLoginVerificationCodeKey(phone), code, time.Duration(SMSCodeValidMinutes)*time.Minute)
}

// StoreSMSVerificationCodeWithKey 内部：直接以指定 key 写入 Redis。
func StoreSMSVerificationCodeWithKey(key, code string, expire time.Duration) error {
	if err := EnsureRedisEnabledForSMS(); err != nil {
		return err
	}
	if expire <= 0 {
		expire = 5 * time.Minute
	}
	return RDB.Set(context.Background(), key, code, expire).Err()
}

// VerifyAndConsumeSMSCode 校验注册/重置/绑定通用短信验证码，成功后删除。
func VerifyAndConsumeSMSCode(phone, code string) bool {
	return verifyAndConsumeKey(SMSVerificationCodeKey("common", phone), code)
}

// VerifyAndConsumeSMSLoginCode 校验登录短信验证码，成功后删除。
func VerifyAndConsumeSMSLoginCode(phone, code string) bool {
	return verifyAndConsumeKey(SMSLoginVerificationCodeKey(phone), code)
}

func verifyAndConsumeKey(key, code string) bool {
	if err := EnsureRedisEnabledForSMS(); err != nil {
		return false
	}
	ctx := context.Background()
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

// GenerateSMSCode 生成 6 位数字短信验证码。
func GenerateSMSCode() string {
	max := big.NewInt(1000000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		// 极端情况下退化为时间戳后 6 位
		return strconv.FormatInt(time.Now().UnixNano()%1000000, 10)
	}
	return fmt.Sprintf("%06d", n.Int64())
}
