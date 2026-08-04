const test = require("node:test");
const assert = require("node:assert/strict");
const { buildEndpoint, normalizeBaseUrl, validateApiKey } = require("../lib/client");

test("normalizes supported base URLs", () => {
  assert.equal(normalizeBaseUrl("https://tokease.com"), "https://tokease.com");
  assert.equal(normalizeBaseUrl("https://tokease.cn/"), "https://tokease.cn");
  assert.equal(normalizeBaseUrl("http://127.0.0.1:3000/"), "http://127.0.0.1:3000");
  assert.equal(normalizeBaseUrl("https://example.com/prefix/"), "https://example.com/prefix");
});

test("rejects unsafe base URLs", () => {
  assert.throws(() => normalizeBaseUrl("file:///tmp/test"), /HTTP/);
  assert.throws(() => normalizeBaseUrl("https://user:pass@example.com"), /不能包含/);
  assert.throws(() => normalizeBaseUrl("https://example.com?a=1"), /不能包含/);
  assert.throws(() => normalizeBaseUrl("https://example.com/#x"), /不能包含/);
});

test("builds fixed API endpoints", () => {
  assert.equal(
    buildEndpoint("https://example.com/base/", "/api/status"),
    "https://example.com/base/api/status"
  );
});

test("validates API keys without changing them", () => {
  assert.equal(validateApiKey(" sk-example-123 "), "sk-example-123");
  assert.throws(() => validateApiKey("short"), /有效/);
});
