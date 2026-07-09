package controller

import (
	"math"
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
)

func useYipayTestRate(t *testing.T, rate float64) {
	t.Helper()
	oldUSDExchangeRate := operation_setting.USDExchangeRate
	oldPrice := operation_setting.Price
	operation_setting.USDExchangeRate = rate
	operation_setting.Price = rate
	t.Cleanup(func() {
		operation_setting.USDExchangeRate = oldUSDExchangeRate
		operation_setting.Price = oldPrice
	})
}

func TestYipayJeepayStoredPayMoneyUsesOrderCurrency(t *testing.T) {
	useYipayTestRate(t, 7.3)

	amountMinor, currency := yipayJeepayAmountMinorAndCurrency(7.3, "PP_PC")
	if amountMinor != 100 || currency != "usd" {
		t.Fatalf("amountMinor, currency = %d, %q; want 100, usd", amountMinor, currency)
	}

	got := yipayJeepayStoredPayMoney(7.3, amountMinor, currency)
	if math.Abs(got-1.0) > 0.000001 {
		t.Fatalf("stored pay money = %v; want 1.0", got)
	}
}

func TestValidateYipayTopupPaidAmountSupportsNewAndLegacyUSDOrders(t *testing.T) {
	useYipayTestRate(t, 7.3)

	newOrder := &model.TopUp{
		Money:         1.0,
		PayCurrency:   "USD",
		PaymentMethod: "PP_PC",
	}
	if err := validateYipayTopupPaidAmount(map[string]string{"amount": "100", "currency": "usd"}, newOrder); err != nil {
		t.Fatalf("new USD order validation failed: %v", err)
	}
	if err := validateYipayTopupPaidAmount(map[string]string{"amount": "730", "currency": "usd"}, newOrder); err == nil {
		t.Fatalf("new USD order accepted wrong amount")
	}

	legacyOrder := &model.TopUp{
		Money:         7.3,
		InputAmount:   7.3,
		InputCurrency: "CNY",
		PayCurrency:   "USD",
		PaymentMethod: "PP_PC",
	}
	if err := validateYipayTopupPaidAmount(map[string]string{"amount": "100", "currency": "usd"}, legacyOrder); err != nil {
		t.Fatalf("legacy USD order validation failed: %v", err)
	}
	if err := validateYipayTopupPaidAmount(map[string]string{"amount": "730", "currency": "usd"}, legacyOrder); err == nil {
		t.Fatalf("legacy USD order accepted wrong amount")
	}
}
