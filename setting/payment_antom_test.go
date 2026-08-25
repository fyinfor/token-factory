package setting

import "testing"

func TestGetAntomPaymentMethodTypesForCurrency(t *testing.T) {
	AntomPaymentMethods = "ALIPAY_CN,ALIPAY_HK"
	if got := GetAntomPaymentMethodTypesForCurrency("CNY"); len(got) != 0 {
		t.Fatalf("legacy wallet pair should not restrict Hosted methods, got %v", got)
	}

	AntomPaymentMethods = "ALIPAY_CN"
	got := GetAntomPaymentMethodTypesForCurrency("CNY")
	if len(got) != 1 || got[0] != "ALIPAY_CN" {
		t.Fatalf("explicit ALIPAY_CN should keep filter, got %v", got)
	}

	AntomPaymentMethods = "ALIPAY_CN,CARD"
	got = GetAntomPaymentMethodTypesForCurrency("CNY")
	if len(got) != 2 {
		t.Fatalf("CNY should keep ALIPAY_CN and CARD, got %v", got)
	}
}
