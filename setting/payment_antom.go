package setting

import "strings"

var AntomEnabled = true
var AntomClientId = ""
var AntomMerchantPrivateKey = ""
var AntomPublicKey = ""
var AntomGatewayURL = "https://open-sea-global.alipay.com"
var AntomPayCurrency = "CNY"
var AntomSettlementCurrency = ""
var AntomPaymentMethods = ""
var AntomMinTopUp = 1

func AntomConfigured() bool {
	return strings.TrimSpace(AntomClientId) != "" &&
		strings.TrimSpace(AntomMerchantPrivateKey) != "" &&
		strings.TrimSpace(AntomPublicKey) != ""
}

// AntomReady 钱包入口与发起支付：开关打开且密钥已配。notify 仍只校验密钥，避免关开关后无法入账。
func AntomReady() bool {
	return AntomEnabled && AntomConfigured()
}

func GetAntomGatewayURL() string {
	url := strings.TrimRight(strings.TrimSpace(AntomGatewayURL), "/")
	if url == "" {
		return "https://open-sea-global.alipay.com"
	}
	return url
}

func GetAntomPayCurrency() string {
	cur := strings.ToUpper(strings.TrimSpace(AntomPayCurrency))
	switch cur {
	case "USD", "CNY":
		return cur
	default:
		return "CNY"
	}
}

func GetAntomSettlementCurrency() string {
	return strings.ToUpper(strings.TrimSpace(AntomSettlementCurrency))
}

func GetAntomPaymentMethodTypes() []string {
	raw := strings.TrimSpace(AntomPaymentMethods)
	if raw == "" || isHostedUnrestrictedMethodList(raw) {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, p := range parts {
		t := strings.ToUpper(strings.TrimSpace(p))
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out
}

// isHostedUnrestrictedMethodList 历史默认「支付宝+支付宝HK」会把 Visa 等卡挡掉。
// Hosted 收银台应交给 Antom 按已开通方式+币种展示，该组合视为不限制。
func isHostedUnrestrictedMethodList(raw string) bool {
	parts := strings.Split(raw, ",")
	seen := map[string]struct{}{}
	for _, p := range parts {
		t := strings.ToUpper(strings.TrimSpace(p))
		if t == "" {
			continue
		}
		if t != "ALIPAY_CN" && t != "ALIPAY_HK" {
			return false
		}
		seen[t] = struct{}{}
	}
	_, cn := seen["ALIPAY_CN"]
	_, hk := seen["ALIPAY_HK"]
	return cn && hk
}

func paymentMethodMatchesCurrency(method, currency string) bool {
	switch method {
	case "ALIPAY_CN":
		return currency == "CNY"
	case "ALIPAY_HK":
		return currency == "HKD"
	case "CARD", "VISA", "MASTERCARD", "JCB", "AMEX", "UNIONPAY":
		return true
	default:
		return true
	}
}

func GetAntomPaymentMethodTypesForCurrency(currency string) []string {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	all := GetAntomPaymentMethodTypes()
	if len(all) == 0 {
		return nil
	}
	out := make([]string, 0, len(all))
	for _, m := range all {
		if paymentMethodMatchesCurrency(m, currency) {
			out = append(out, m)
		}
	}
	return out
}
