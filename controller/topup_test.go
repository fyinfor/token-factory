package controller

import (
	"math"
	"testing"

	"github.com/QuantumNous/new-api/common"
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

// truncatedCNYDisplay 模拟钱包「当前余额」前端口径：quota → USD → CNY 后向下截断到 2 位小数。
func truncatedCNYDisplay(quota int, rate float64) float64 {
	value := float64(quota) / common.QuotaPerUnit * rate
	return math.Floor(value*100) / 100
}

func useQuotaDisplayType(t *testing.T, displayType string) {
	t.Helper()
	gs := operation_setting.GetGeneralSetting()
	old := gs.QuotaDisplayType
	gs.QuotaDisplayType = displayType
	t.Cleanup(func() {
		gs.QuotaDisplayType = old
	})
}

func TestBuildTopupQuoteCNYDisplayMatchesPayAmount(t *testing.T) {
	useYipayTestRate(t, 7.3)
	useQuotaDisplayType(t, operation_setting.QuotaDisplayTypeCNY)

	quote, err := buildTopupQuote(500, "default", topupCurrencyCNY)
	if err != nil {
		t.Fatalf("buildTopupQuote: %v", err)
	}
	got := truncatedCNYDisplay(quote.QuotaToAdd, 7.3)
	if got < 500 {
		t.Fatalf("wallet CNY display after credit = %v (quota=%d); want >= 500", got, quote.QuotaToAdd)
	}
	// 不应为对齐而多补超过 0.01 的展示差额
	if got >= 500.01 {
		t.Fatalf("wallet CNY display after credit = %v; want < 500.01 (over-credit)", got)
	}
}

func TestBuildTopupQuoteRoundWouldShortenCNYDisplay(t *testing.T) {
	useYipayTestRate(t, 7.3)
	useQuotaDisplayType(t, operation_setting.QuotaDisplayTypeCNY)

	// 旧口径 Round(0)：500/7.3*QPU 四舍五入后反算截断会落到 499.99
	dRate := 7.3
	roundedQuota := int(math.Round(500 / dRate * common.QuotaPerUnit))
	if truncatedCNYDisplay(roundedQuota, dRate) >= 500 {
		t.Skip("current rate/QPU makes Round already align; Ceil still required for other rates")
	}

	quote, err := buildTopupQuote(500, "default", topupCurrencyCNY)
	if err != nil {
		t.Fatalf("buildTopupQuote: %v", err)
	}
	if quote.QuotaToAdd <= roundedQuota {
		t.Fatalf("aligned quota=%d; want > rounded=%d so truncated display recovers 500", quote.QuotaToAdd, roundedQuota)
	}
	if got := truncatedCNYDisplay(quote.QuotaToAdd, dRate); got < 500 {
		t.Fatalf("wallet CNY display = %v; want >= 500", got)
	}
}

func TestBuildTopupQuoteUSDDisplayMatchesPayAmount(t *testing.T) {
	useYipayTestRate(t, 7.3)
	useQuotaDisplayType(t, operation_setting.QuotaDisplayTypeUSD)

	quote, err := buildTopupQuote(10, "default", topupCurrencyUSD)
	if err != nil {
		t.Fatalf("buildTopupQuote: %v", err)
	}
	usd := float64(quote.QuotaToAdd) / common.QuotaPerUnit
	got := math.Floor(usd*100) / 100
	if got < 10 {
		t.Fatalf("wallet USD display after credit = %v (quota=%d); want >= 10", got, quote.QuotaToAdd)
	}
}
