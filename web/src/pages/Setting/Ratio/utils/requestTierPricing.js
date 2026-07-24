/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

// ============================================================
// 阶梯计费 — 统一数据模型与工具函数（v3）
// 以【输入 Token 区间】划分档位；每档绑定四价；Rule 级 boundary=lt|lte
// ============================================================

export const TIER_MODE = 'progressive';
export const TIER_DIMENSION = 'input_tokens';
export const TIER_BOUNDARY_LT = 'lt';
export const TIER_BOUNDARY_LTE = 'lte';
export const TIER_RATIO_BASE = 2;

export const CURRENCY_OPTIONS = [
  { key: 'USD', label: 'USD ($)', symbol: '$' },
  { key: 'CNY', label: 'CNY (¥)', symbol: '¥' },
  { key: 'CUSTOM', label: '自定义', symbol: '¤' },
];

export const normalizeCurrency = (currency) => {
  const key = String(currency || '').toUpperCase();
  return ['USD', 'CNY', 'CUSTOM'].includes(key) ? key : 'USD';
};

export const getCurrencySymbol = (currency, customSymbol = '¤') => {
  const key = normalizeCurrency(currency);
  if (key === 'CUSTOM') return customSymbol || '¤';
  const opt = CURRENCY_OPTIONS.find((c) => c.key === key);
  return opt ? opt.symbol : '$';
};

/** 从 status / options 读取 1 USD = X 货币 的汇率表 */
export const buildCurrencyRates = ({
  usdExchangeRate,
  customExchangeRate,
} = {}) => {
  const cny = Number(usdExchangeRate);
  const custom = Number(customExchangeRate);
  return {
    USD: 1,
    CNY: Number.isFinite(cny) && cny > 0 ? cny : 1,
    CUSTOM: Number.isFinite(custom) && custom > 0 ? custom : 1,
  };
};

export const getCurrencyRatesFromStatus = () => {
  let usdExchangeRate = 7;
  let customExchangeRate = 1;
  try {
    const statusStr = localStorage.getItem('status');
    if (statusStr) {
      const s = JSON.parse(statusStr);
      usdExchangeRate = s?.usd_exchange_rate;
      customExchangeRate = s?.custom_currency_exchange_rate;
    }
  } catch {
    /* ignore */
  }
  return buildCurrencyRates({ usdExchangeRate, customExchangeRate });
};

export const getCurrencyRate = (currency, rates) => {
  const table = rates || getCurrencyRatesFromStatus();
  const key = normalizeCurrency(currency);
  return table[key] > 0 ? table[key] : 1;
};

/**
 * 基准货币价格 → 目标货币价格。
 * 基准货币与目标货币一致时不做换算。
 */
export const convertTierPriceBetweenCurrencies = (
  price,
  fromCurrency,
  toCurrency,
  rates,
) => {
  const n = Number(price);
  if (!Number.isFinite(n) || n === 0) return 0;
  const from = normalizeCurrency(fromCurrency);
  const to = normalizeCurrency(toCurrency);
  if (from === to) return n;
  const fromRate = getCurrencyRate(from, rates);
  const toRate = getCurrencyRate(to, rates);
  return (n / fromRate) * toRate;
};

export const convertTierPriceToUSD = (price, currency, rates) =>
  convertTierPriceBetweenCurrencies(price, currency, 'USD', rates);

export const emptyTierPricing = (currency = 'USD') => ({
  mode: TIER_MODE,
  dimension: TIER_DIMENSION,
  boundary: TIER_BOUNDARY_LT,
  currency: normalizeCurrency(currency),
  tiers: [],
});

export const normalizeBoundary = (boundary) =>
  String(boundary || '').toLowerCase() === TIER_BOUNDARY_LTE
    ? TIER_BOUNDARY_LTE
    : TIER_BOUNDARY_LT;

export const priceToTierRatio = (price) => {
  if (!Number.isFinite(Number(price))) return 0;
  return Math.max(0, Number(price) / TIER_RATIO_BASE);
};

export const tierRatioToPrice = (ratio) => {
  if (!Number.isFinite(Number(ratio))) return 0;
  return Math.max(0, Number(ratio) * TIER_RATIO_BASE);
};

export { priceToTierRatio as priceToRatio };
export { tierRatioToPrice as ratioToPrice };

export const normalizeTierRow = (row) => {
  const upTo = Number(row?.up_to ?? 0);
  if (!Number.isFinite(upTo) || upTo < 0) return null;
  const prices = row?.prices && typeof row.prices === 'object' ? row.prices : null;
  return {
    up_to: upTo,
    inputPrice: Math.max(
      0,
      Number(row?.inputPrice ?? row?.input_price ?? prices?.input ?? 0),
    ),
    outputPrice: Math.max(
      0,
      Number(row?.outputPrice ?? row?.output_price ?? prices?.output ?? 0),
    ),
    cacheReadPrice: Math.max(
      0,
      Number(
        row?.cacheReadPrice ?? row?.cache_read_price ?? prices?.cache_read ?? 0,
      ),
    ),
    cacheWritePrice: Math.max(
      0,
      Number(
        row?.cacheWritePrice ??
          row?.cache_write_price ??
          prices?.cache_write ??
          0,
      ),
    ),
  };
};

export const normalizeTierRows = (rows) => {
  if (!Array.isArray(rows)) return [];
  const normalized = rows.map(normalizeTierRow).filter(Boolean);
  if (normalized.length === 0) return [];

  const sorted = [...normalized].sort((a, b) => {
    if (a.up_to === 0) return 1;
    if (b.up_to === 0) return -1;
    return a.up_to - b.up_to;
  });

  const deduped = [];
  const seen = new Set();
  for (const row of sorted) {
    if (seen.has(row.up_to)) continue;
    seen.add(row.up_to);
    deduped.push(row);
  }

  if (deduped.length > 0 && deduped[deduped.length - 1].up_to !== 0) {
    deduped.push({
      ...deduped[deduped.length - 1],
      up_to: 0,
      inputPrice: 0,
      outputPrice: 0,
      cacheReadPrice: 0,
      cacheWritePrice: 0,
    });
  }

  return deduped;
};

export const normalizeTierPricing = (tp) => {
  const src = tp && typeof tp === 'object' ? tp : {};
  return {
    mode: src.mode || TIER_MODE,
    dimension: src.dimension || TIER_DIMENSION,
    boundary: normalizeBoundary(src.boundary),
    currency: normalizeCurrency(src.currency),
    tiers: normalizeTierRows(src.tiers),
  };
};

export const hasTierPricing = (tp) => normalizeTierPricing(tp).tiers.length > 0;
export const hasTierRule = (tp) => hasTierPricing(tp);

export const hasTierSegments = (tier) => {
  if (!tier) return false;
  if (Array.isArray(tier?.tiers)) return tier.tiers.length > 0;
  if (Array.isArray(tier?.segments)) return tier.segments.length > 0;
  return false;
};

export const validateTierPricing = (tp, t = (v) => v) => {
  const { tiers, boundary, dimension } = normalizeTierPricing(tp);
  if (tiers.length === 0) return '';
  if (dimension !== TIER_DIMENSION) {
    return t('当前仅支持按输入Token区间阶梯计费');
  }
  if (boundary !== TIER_BOUNDARY_LT && boundary !== TIER_BOUNDARY_LTE) {
    return t('边界配置无效');
  }

  let previous = 0;
  for (let i = 0; i < tiers.length; i += 1) {
    const row = tiers[i];
    if (row.up_to === 0) {
      if (i !== tiers.length - 1) {
        return t('只有最后一档上限可以为 0（无限）');
      }
      continue;
    }
    if (row.up_to <= previous) {
      return t('输入Token区间上限必须递增，第 {{index}} 档异常', {
        index: i + 1,
      });
    }
    previous = row.up_to;
    if (
      row.inputPrice < 0 ||
      row.outputPrice < 0 ||
      row.cacheReadPrice < 0 ||
      row.cacheWritePrice < 0
    ) {
      return t('价格不能为负数，第 {{index}} 档异常', { index: i + 1 });
    }
  }

  if (!tiers.some((row) => row.inputPrice > 0)) {
    return t('至少需要一档输入价格大于 0');
  }
  return '';
};

export const validateTierRule = (tp, t) => validateTierPricing(tp, t);

export const summarizeTierPricing = (tp, t = (v) => v) => {
  const { currency, tiers, boundary } = normalizeTierPricing(tp);
  if (tiers.length === 0) return t('未配置');
  const symbol = getCurrencySymbol(currency);
  const boundLabel = boundary === TIER_BOUNDARY_LTE ? '≤' : '<';
  return `${tiers.length}${t('档')} / ${boundLabel} / ${t('输入')} ${tiers
    .map((r) => `${symbol}${Number(r.inputPrice.toFixed(2))}`)
    .join(' → ')}`;
};

export const summarizeTierRule = (tp, t) => summarizeTierPricing(tp, t);

export const formatTierRangeLabel = (from, to, boundary) => {
  const b = normalizeBoundary(boundary);
  if (!to) {
    return from > 0 ? `${from}+` : '0+';
  }
  if (b === TIER_BOUNDARY_LTE) {
    return `${from}～${to}（≤）`;
  }
  return `${from}～${to}（<）`;
};

export const buildTierPriceDetails = (tp, t = (v) => v) => {
  const normalized = normalizeTierPricing(tp);
  if (normalized.tiers.length === 0) return [];
  const symbol = getCurrencySymbol(normalized.currency);
  const categories = [
    { key: 'input', label: t('输入价格'), priceKey: 'inputPrice' },
    { key: 'output', label: t('输出价格'), priceKey: 'outputPrice' },
    { key: 'cache_read', label: t('缓存读取价格'), priceKey: 'cacheReadPrice' },
    { key: 'cache_write', label: t('缓存写入价格'), priceKey: 'cacheWritePrice' },
  ];

  return categories.map(({ key, label, priceKey }) => {
    let prev = 0;
    const segments = normalized.tiers.map((row) => {
      const from = prev;
      const to = row.up_to || '∞';
      prev = row.up_to || prev;
      const price = row[priceKey];
      return {
        range: formatTierRangeLabel(from, row.up_to || 0, normalized.boundary),
        rangeRaw: `${from}～${to}`,
        price: price > 0 ? `${symbol}${Number(price.toFixed(6))}` : '-',
        rawPrice: price,
      };
    });
    return { key, label, symbol, segments };
  });
};

/** 前端编辑模型 → 后端 RequestTierPricing（价格按基准货币原样存储） */
export const serializeRequestTierPricing = (tp) => {
  const normalized = normalizeTierPricing(tp);
  if (normalized.tiers.length === 0) return null;
  return {
    mode: TIER_MODE,
    dimension: TIER_DIMENSION,
    boundary: normalized.boundary,
    currency: normalized.currency,
    tiers: normalized.tiers.map((row) => ({
      up_to: row.up_to,
      prices: {
        input: row.inputPrice,
        output: row.outputPrice,
        cache_read: row.cacheReadPrice,
        cache_write: row.cacheWritePrice,
      },
    })),
  };
};

/** 后端 RequestTierPricing → 前端编辑模型 */
export const parseRequestTierPricing = (rule, fallbackCurrency = 'USD') => {
  if (!rule || typeof rule !== 'object') {
    return emptyTierPricing(fallbackCurrency);
  }
  const currency = normalizeCurrency(rule.currency || fallbackCurrency);
  if (Array.isArray(rule.tiers) && rule.tiers.length > 0) {
    return normalizeTierPricing({
      mode: rule.mode,
      dimension: rule.dimension,
      boundary: rule.boundary,
      currency,
      tiers: rule.tiers,
    });
  }
  if (Array.isArray(rule.input) || Array.isArray(rule.output)) {
    return normalizeTierRule(rule, currency);
  }
  return emptyTierPricing(currency);
};

/** @deprecated */
export const unifiedToLegacy = (tp) => {
  const rule = serializeRequestTierPricing(tp);
  if (!rule) {
    return {
      modelTierRatio: null,
      completionTierRatio: null,
      cacheTierRatio: null,
      createCacheTierRatio: null,
      requestTierPricing: null,
    };
  }
  const toSegments = (priceKey) => ({
    segments: rule.tiers.map((row) => ({
      up_to: row.up_to,
      ratio: priceToTierRatio(row.prices[priceKey]),
    })),
  });
  return {
    requestTierPricing: rule,
    modelTierRatio: toSegments('input'),
    completionTierRatio: toSegments('output'),
    cacheTierRatio: toSegments('cache_read'),
    createCacheTierRatio: toSegments('cache_write'),
  };
};

/** @deprecated 仍支持旧 4 Key 合并；首参若已是统一结构则直接 parse */
export const legacyToUnified = (
  modelTierRatio,
  completionTierRatio,
  cacheTierRatio,
  createCacheTierRatio,
  currency = 'USD',
  boundary = TIER_BOUNDARY_LT,
) => {
  if (
    modelTierRatio &&
    typeof modelTierRatio === 'object' &&
    Array.isArray(modelTierRatio.tiers)
  ) {
    return parseRequestTierPricing(modelTierRatio, currency);
  }

  const inputSegments = Array.isArray(modelTierRatio?.segments)
    ? modelTierRatio.segments
    : [];
  const outputSegments = Array.isArray(completionTierRatio?.segments)
    ? completionTierRatio.segments
    : [];
  const cacheReadSegments = Array.isArray(cacheTierRatio?.segments)
    ? cacheTierRatio.segments
    : [];
  const cacheWriteSegments = Array.isArray(createCacheTierRatio?.segments)
    ? createCacheTierRatio.segments
    : [];

  const baseSegments =
    inputSegments.length > 0
      ? inputSegments
      : outputSegments.length > 0
        ? outputSegments
        : cacheReadSegments.length > 0
          ? cacheReadSegments
          : cacheWriteSegments;

  if (baseSegments.length === 0) return emptyTierPricing();

  const outputMap = new Map(outputSegments.map((s) => [s.up_to, s.ratio]));
  const cacheReadMap = new Map(cacheReadSegments.map((s) => [s.up_to, s.ratio]));
  const cacheWriteMap = new Map(
    cacheWriteSegments.map((s) => [s.up_to, s.ratio]),
  );
  const inputMap = new Map(inputSegments.map((s) => [s.up_to, s.ratio]));

  const tiers = baseSegments
    .filter((s) => Number.isFinite(s.up_to) && s.up_to >= 0)
    .map((s) => ({
      up_to: s.up_to,
      inputPrice: tierRatioToPrice(inputMap.get(s.up_to) ?? 0),
      outputPrice: tierRatioToPrice(outputMap.get(s.up_to) ?? 0),
      cacheReadPrice: tierRatioToPrice(cacheReadMap.get(s.up_to) ?? 0),
      cacheWritePrice: tierRatioToPrice(cacheWriteMap.get(s.up_to) ?? 0),
    }));

  return normalizeTierPricing({
    mode: TIER_MODE,
    dimension: TIER_DIMENSION,
    boundary,
    currency,
    tiers,
  });
};

export const parseJSONMap = (raw) => {
  if (!raw || String(raw).trim() === '') return {};
  try {
    const parsed = JSON.parse(raw);
    if (Array.isArray(parsed)) {
      return Object.fromEntries(
        parsed.map((item, index) => [`tpl_${index + 1}`, item]),
      );
    }
    return parsed && typeof parsed === 'object' && !Array.isArray(parsed)
      ? parsed
      : {};
  } catch {
    return {};
  }
};

export const TIER_CATEGORIES = [
  { key: 'input', label: '输入价格' },
  { key: 'output', label: '输出价格' },
  { key: 'cache_read', label: '缓存读取价格' },
  { key: 'cache_write', label: '缓存写入价格' },
];

export const normalizeTierRule = (rule, fallbackCurrency = 'USD') => {
  const src = rule && typeof rule === 'object' ? rule : {};
  if (Array.isArray(src.tiers)) {
    return normalizeTierPricing({
      ...src,
      currency: src.currency || fallbackCurrency,
    });
  }
  return {
    mode: src.mode || TIER_MODE,
    dimension: TIER_DIMENSION,
    boundary: normalizeBoundary(src.boundary),
    currency: normalizeCurrency(src.currency || fallbackCurrency),
    tiers: normalizeTierRows(
      (Array.isArray(src.input) ? src.input : []).map((row, i) => {
        const outputRow = Array.isArray(src.output) ? src.output[i] : null;
        const cacheReadRow = Array.isArray(src.cache_read)
          ? src.cache_read[i]
          : null;
        const cacheWriteRow = Array.isArray(src.cache_write)
          ? src.cache_write[i]
          : null;
        return {
          up_to: row?.up_to ?? 0,
          inputPrice: tierRatioToPrice(row?.ratio ?? 0),
          outputPrice: tierRatioToPrice(outputRow?.ratio ?? 0),
          cacheReadPrice: tierRatioToPrice(cacheReadRow?.ratio ?? 0),
          cacheWritePrice: tierRatioToPrice(cacheWriteRow?.ratio ?? 0),
        };
      }),
    ),
  };
};

export const serializeTierRule = (tp) => serializeRequestTierPricing(tp);
export const emptyTierRule = emptyTierPricing;

export const normalizeTierSegments = (tier) => {
  const src = tier && typeof tier === 'object' ? tier : {};
  if (Array.isArray(src.tiers)) {
    return {
      segments: src.tiers.map((r) => ({
        up_to: r.up_to,
        ratio: priceToTierRatio(r.inputPrice ?? r.prices?.input ?? 0),
      })),
    };
  }
  return {
    segments: (Array.isArray(src.segments) ? src.segments : [])
      .map((row) => ({
        up_to: Number(row?.up_to || 0),
        ratio: Number(row?.ratio ?? 0),
      }))
      .filter((row) => Number.isFinite(row.up_to) && Number.isFinite(row.ratio)),
  };
};

export const ensureFinalInfinityTierSegments = (segments) => {
  const normalized = Array.isArray(segments) ? segments : [];
  if (!normalized.length) return normalized;
  if (normalized[normalized.length - 1].up_to === 0) return normalized;
  return [...normalized, { up_to: 0, ratio: 0 }];
};

export const findTierBandIndex = (
  tokens,
  tiers,
  boundary = TIER_BOUNDARY_LT,
) => {
  const rows = normalizeTierRows(tiers);
  if (rows.length === 0) return -1;
  const b = normalizeBoundary(boundary);
  for (let i = 0; i < rows.length; i += 1) {
    const upTo = rows[i].up_to;
    if (upTo === 0) return i;
    if (b === TIER_BOUNDARY_LTE) {
      if (tokens <= upTo) return i;
    } else if (tokens < upTo) {
      return i;
    }
  }
  return rows.length - 1;
};
