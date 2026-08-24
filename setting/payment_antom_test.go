package setting

import "testing"

func TestGetAntomPaymentMethodTypesForCurrency(t *testing.T) {
	AntomPaymentMethods = "ALIPAY_CN,ALIPAY_HK"
	got := GetAntomPaymentMethodTypesForCurrency("CNY")
	if len(got) != 1 || got[0] != "ALIPAY_CN" {
		t.Fatalf("CNY should only keep ALIPAY_CN, got %v", got)
	}
	got = GetAntomPaymentMethodTypesForCurrency("HKD")
	if len(got) != 1 || got[0] != "ALIPAY_HK" {
		t.Fatalf("HKD should only keep ALIPAY_HK, got %v", got)
	}
}
