package setting

import "testing"

func TestAntomReady(t *testing.T) {
	origEnabled := AntomEnabled
	origID, origPriv, origPub := AntomClientId, AntomMerchantPrivateKey, AntomPublicKey
	t.Cleanup(func() {
		AntomEnabled = origEnabled
		AntomClientId, AntomMerchantPrivateKey, AntomPublicKey = origID, origPriv, origPub
	})

	AntomClientId, AntomMerchantPrivateKey, AntomPublicKey = "cid", "priv", "pub"
	AntomEnabled = true
	if !AntomReady() {
		t.Fatal("expected ready when enabled and configured")
	}
	AntomEnabled = false
	if AntomReady() {
		t.Fatal("expected not ready when switch off")
	}
	if !AntomConfigured() {
		t.Fatal("configured should ignore switch")
	}
}

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
