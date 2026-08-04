const test = require("node:test");
const assert = require("node:assert/strict");
const { buildViewModel, formatLogQuota, formatQuota } = require("../lib/quota");

function status(type, extra = {}) {
  return {
    success: true,
    data: { quota_display_type: type, quota_per_unit: 500000, usd_exchange_rate: 7.3, ...extra }
  };
}

test("formats all supported quota display types", () => {
  assert.equal(formatQuota(500000, status("USD")), "$1");
  assert.equal(formatQuota(500000, status("CNY")), "¥7.3");
  assert.equal(
    formatQuota(500000, status("CUSTOM", { custom_currency_symbol: "€", custom_currency_exchange_rate: 0.9 })),
    "€0.9"
  );
  assert.equal(formatQuota(1500000, status("TOKENS")), "1.5M");
});

test("keeps tiny non-zero currency values visible", () => {
  assert.equal(formatQuota(5, status("USD")), "$0.00001");
});

test("uses video quota-per-unit metadata for log amounts", () => {
  const value = formatLogQuota({ quota: 100, other: JSON.stringify({ video_quota_per_unit: 100 }) }, status("USD"));
  assert.equal(value, "$1");
});

test("builds renderer-safe view data", () => {
  const model = buildViewModel({
    status: status("USD"),
    usage: { success: true, data: { quota: 1000000, gift_quota: 3, used_quota: 4 } },
    logs: {
      success: true,
      data: { items: [{ id: 1, model_name: "gpt-test", quota: 500000, created_at: 1, prompt_tokens: 2, completion_tokens: 3, use_time: 1 }] }
    }
  });
  assert.equal(model.balance, "$2");
  assert.equal(model.logs.length, 1);
  assert.equal(model.logs[0].amount, "$1");
  assert.equal(model.logs[0].model, "gpt-test");
});
