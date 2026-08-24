package setting

import "strings"

var AntomClientId = ""
var AntomMerchantPrivateKey = ""
var AntomPublicKey = ""
var AntomGatewayURL = "https://open-sea-global.alipay.com"
var AntomPayCurrency = "CNY"
var AntomSettlementCurrency = ""
var AntomPaymentMethods = "ALIPAY_CN,ALIPAY_HK"
var AntomMinTopUp = 1

func AntomConfigured() bool {
	return strings.TrimSpace(AntomClientId) != "" &&
		strings.TrimSpace(AntomMerchantPrivateKey) != "" &&
		strings.TrimSpace(AntomPublicKey) != ""
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
	cur := strings.ToUpper(strings.TrimSpace(AntomSettlementCurrency))
	if cur == "" {
		return GetAntomPayCurrency()
	}
	return cur
}

func GetAntomPaymentMethodTypes() []string {
	raw := strings.TrimSpace(AntomPaymentMethods)
	if raw == "" {
		return []string{"ALIPAY_CN", "ALIPAY_HK"}
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
	if len(out) == 0 {
		return []string{"ALIPAY_CN", "ALIPAY_HK"}
	}
	return out
}
