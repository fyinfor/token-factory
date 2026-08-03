function unwrap(payload) {
  return payload && payload.success === true ? payload.data : null;
}

function number(value, fallback = 0) {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : fallback;
}

function trimFixed(value, digits) {
  return Number(value).toFixed(digits).replace(/\.0+$|(?<=\.[0-9]*?)0+$/g, "");
}

function currencyConfig(statusPayload) {
  const status = unwrap(statusPayload) || statusPayload || {};
  const type = String(status.quota_display_type || "USD").toUpperCase();
  const quotaPerUnit = Math.max(number(status.quota_per_unit, 1), 1);

  if (type === "TOKENS") return { type, symbol: "", quotaPerUnit, rate: 1 };
  if (type === "CNY") {
    return { type, symbol: "¥", quotaPerUnit, rate: number(status.usd_exchange_rate, 1) || 1 };
  }
  if (type === "CUSTOM") {
    return {
      type,
      symbol: String(status.custom_currency_symbol || "¤"),
      quotaPerUnit,
      rate: number(status.custom_currency_exchange_rate, 1) || 1
    };
  }
  return { type: "USD", symbol: "$", quotaPerUnit, rate: 1 };
}

function compactTokens(value) {
  const absolute = Math.abs(value);
  const units = [
    [1e9, "B"],
    [1e6, "M"],
    [1e3, "K"]
  ];
  for (const [threshold, suffix] of units) {
    if (absolute >= threshold) return `${trimFixed(value / threshold, 2)}${suffix}`;
  }
  return Math.round(value).toLocaleString("en-US");
}

function formatQuota(rawQuota, statusPayload, preferredDigits = 2) {
  const quota = number(rawQuota);
  const config = currencyConfig(statusPayload);
  if (config.type === "TOKENS") return compactTokens(quota);

  const amount = (quota / config.quotaPerUnit) * config.rate;
  let digits = preferredDigits;
  if (amount !== 0 && Math.abs(amount) < Math.pow(10, -preferredDigits)) {
    digits = Math.min(6, Math.max(preferredDigits, Math.ceil(-Math.log10(Math.abs(amount))) + 1));
  }
  return `${config.symbol}${trimFixed(amount, digits)}`;
}

function parseOther(value) {
  if (!value) return {};
  if (typeof value === "object") return value;
  try {
    const parsed = JSON.parse(value);
    return parsed && typeof parsed === "object" ? parsed : {};
  } catch {
    return {};
  }
}

function formatLogQuota(log, statusPayload) {
  const other = parseOther(log?.other);
  const logQuotaPerUnit = number(other.video_quota_per_unit);
  if (logQuotaPerUnit > 0) {
    const status = { ...(unwrap(statusPayload) || statusPayload || {}), quota_per_unit: logQuotaPerUnit };
    return formatQuota(log?.quota, status, 6);
  }
  return formatQuota(log?.quota, statusPayload, 6);
}

function formatTime(epochSeconds) {
  const seconds = number(epochSeconds);
  if (seconds <= 0) return "--";
  return new Intl.DateTimeFormat("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hour12: false
  }).format(new Date(seconds * 1000));
}

function formatDuration(value) {
  const seconds = number(value);
  if (seconds <= 0) return "--";
  if (seconds < 1) return `${Math.round(seconds * 1000)}ms`;
  return `${trimFixed(seconds, seconds >= 10 ? 0 : 1)}s`;
}

function buildViewModel(result) {
  const status = result?.status;
  const usage = unwrap(result?.usage);
  const logsPage = unwrap(result?.logs);
  if (!status || !usage || !logsPage) throw new Error("查询数据不完整");

  const items = Array.isArray(logsPage.items) ? logsPage.items.slice(0, 10) : [];
  return {
    balance: formatQuota(usage.quota, status, 2),
    raw: {
      quota: number(usage.quota),
      giftQuota: number(usage.gift_quota),
      usedQuota: number(usage.used_quota)
    },
    logs: items.map((item) => ({
      id: number(item.id),
      model: String(item.model_name || "unknown"),
      amount: formatLogQuota(item, status),
      time: formatTime(item.created_at),
      promptTokens: number(item.prompt_tokens).toLocaleString("en-US"),
      completionTokens: number(item.completion_tokens).toLocaleString("en-US"),
      duration: formatDuration(item.use_time)
    }))
  };
}

module.exports = {
  unwrap,
  currencyConfig,
  compactTokens,
  formatQuota,
  parseOther,
  formatLogQuota,
  formatTime,
  formatDuration,
  buildViewModel
};
