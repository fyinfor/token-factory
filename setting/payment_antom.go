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
	return strings.ToUpper(strings.TrimSpace(AntomSettlementCurrency))
}

func GetAntomPaymentMethodTypes() []string {
	raw := strings.TrimSpace(AntomPaymentMethods)
	if raw == "" {
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

func paymentMethodMatchesCurrency(method, currency string) bool {
	switch method {
	case "ALIPAY_CN":
		return currency == "CNY"
	case "ALIPAY_HK":
		return currency == "HKD"
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
