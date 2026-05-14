package operation_setting

import (
	"strings"

	"github.com/QuantumNous/new-api/setting/config"
)

// 与前端 supportedLanguages 一致
var siteInterfaceLanguages = []string{
	"zh-CN", "zh-TW", "en", "fr", "ru", "ja", "vi", "id", "ms", "th", "sw",
}

// NormalizeDefaultSiteLanguage 将后台配置的默认界面语言规范为受支持的代码，非法则回退 zh-CN
func NormalizeDefaultSiteLanguage(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "zh-CN"
	}
	for _, a := range siteInterfaceLanguages {
		if strings.EqualFold(s, a) {
			return a
		}
	}
	return "zh-CN"
}

// 额度展示类型
const (
	QuotaDisplayTypeUSD    = "USD"
	QuotaDisplayTypeCNY    = "CNY"
	QuotaDisplayTypeTokens = "TOKENS"
	QuotaDisplayTypeCustom = "CUSTOM"
)

type GeneralSetting struct {
	DocsLink            string `json:"docs_link"`
	// DefaultSiteLanguage 未登录访客首次进入站点时使用的界面语言（BCP 47，如 zh-CN、en）
	DefaultSiteLanguage string `json:"default_site_language"`
	PingIntervalEnabled bool   `json:"ping_interval_enabled"`
	PingIntervalSeconds int    `json:"ping_interval_seconds"`
	// 当前站点额度展示类型：USD / CNY / TOKENS
	QuotaDisplayType string `json:"quota_display_type"`
	// 充值页金额展示币种：USD / CNY（仅用于钱包充值“实付金额”文案展示，不影响内部计价）
	RechargeDisplayCurrency string `json:"recharge_display_currency"`
	// 自定义货币符号，用于 CUSTOM 展示类型
	CustomCurrencySymbol string `json:"custom_currency_symbol"`
	// 自定义货币与美元汇率（1 USD = X Custom）
	CustomCurrencyExchangeRate float64 `json:"custom_currency_exchange_rate"`
}

// 默认配置
var generalSetting = GeneralSetting{
	DocsLink:                   "https://docs.newapi.pro",
	DefaultSiteLanguage:        "zh-CN",
	PingIntervalEnabled:        false,
	PingIntervalSeconds:        60,
	QuotaDisplayType:           QuotaDisplayTypeUSD,
	RechargeDisplayCurrency:    QuotaDisplayTypeUSD,
	CustomCurrencySymbol:       "¤",
	CustomCurrencyExchangeRate: 1.0,
}

func init() {
	// 注册到全局配置管理器
	config.GlobalConfig.Register("general_setting", &generalSetting)
}

func GetGeneralSetting() *GeneralSetting {
	return &generalSetting
}

// GetDefaultSiteLanguage 返回对外使用的默认界面语言（已校验）
func GetDefaultSiteLanguage() string {
	return NormalizeDefaultSiteLanguage(generalSetting.DefaultSiteLanguage)
}

// IsCurrencyDisplay 是否以货币形式展示（美元或人民币）
func IsCurrencyDisplay() bool {
	return generalSetting.QuotaDisplayType != QuotaDisplayTypeTokens
}

// IsCNYDisplay 是否以人民币展示
func IsCNYDisplay() bool {
	return generalSetting.QuotaDisplayType == QuotaDisplayTypeCNY
}

// GetQuotaDisplayType 返回额度展示类型
func GetQuotaDisplayType() string {
	return generalSetting.QuotaDisplayType
}

// GetCurrencySymbol 返回当前展示类型对应符号
func GetCurrencySymbol() string {
	switch generalSetting.QuotaDisplayType {
	case QuotaDisplayTypeUSD:
		return "$"
	case QuotaDisplayTypeCNY:
		return "¥"
	case QuotaDisplayTypeCustom:
		if generalSetting.CustomCurrencySymbol != "" {
			return generalSetting.CustomCurrencySymbol
		}
		return "¤"
	default:
		return ""
	}
}

// GetUsdToCurrencyRate 返回 1 USD = X <currency> 的 X（TOKENS 不适用）
func GetUsdToCurrencyRate(usdToCny float64) float64 {
	switch generalSetting.QuotaDisplayType {
	case QuotaDisplayTypeUSD:
		return 1
	case QuotaDisplayTypeCNY:
		return usdToCny
	case QuotaDisplayTypeCustom:
		if generalSetting.CustomCurrencyExchangeRate > 0 {
			return generalSetting.CustomCurrencyExchangeRate
		}
		return 1
	default:
		return 1
	}
}
