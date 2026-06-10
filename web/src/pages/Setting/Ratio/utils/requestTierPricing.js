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
// 阶梯计费 — 统一数据模型与工具函数（v2 重构版）
// ============================================================
// 新模型：以【输入Token数量区间】作为唯一阶梯划分条件，
// 每个档位统一绑定 4 项计费单价：输入 / 输出 / 缓存读取 / 缓存写入。
// 后端仍存储 4 个独立 Option Key，前端在序列化时自动转换。
// ============================================================

// ---- 常量定义 -------------------------------------------------

/** 阶梯计费模式（固定 progressive） */
export const TIER_MODE = 'progressive';

/** 内部 ratio 换算基准倍率（与后端 model_ratio 默认 2 对齐） */
export const TIER_RATIO_BASE = 2;

/** 货币选项 */
export const CURRENCY_OPTIONS = [
  {
    key: 'USD',
    label: 'USD ($)',
    symbol: '$',
    rate: 1,
  },
  {
    key: 'CNY',
    label: 'CNY (¥)',
    symbol: '¥',
    rate: 7.2, // 默认汇率，可从站点配置覆盖
  },
  {
    key: 'CUSTOM',
    label: '自定义 ($)',
    symbol: '$',
    rate: 1,
  },
];

/** 根据 currency key 获取显示符号 */
export const getCurrencySymbol = (currency) => {
  const opt = CURRENCY_OPTIONS.find((c) => c.key === currency);
  return opt ? opt.symbol : '$';
};

/** 根据 currency key 获取汇率 */
export const getCurrencyRate = (currency) => {
  const opt = CURRENCY_OPTIONS.find((c) => c.key === currency);
  return opt ? opt.rate : 1;
};

// ---- 数据结构定义 ----------------------------------------------

/**
 * 单条阶梯档位（前端编辑模型）
 * @typedef {Object} TierRow
 * @property {number} up_to — 输入Token上限 (0=无限，仅最后一行)
 * @property {number} inputPrice — 输入价格 (用户输入货币单位/1M tokens)
 * @property {number} outputPrice — 输出价格
 * @property {number} cacheReadPrice — 缓存读取价格
 * @property {number} cacheWritePrice — 缓存写入价格
 */

/**
 * 统一阶梯定价数据
 * @typedef {Object} TierPricing
 * @property {'progressive'} mode
 * @property {'USD'|'CNY'|'CUSTOM'} currency
 * @property {TierRow[]} tiers
 */

/** 创建空的阶梯定价对象 */
export const emptyTierPricing = () => ({
  mode: TIER_MODE,
  currency: 'USD',
  tiers: [],
});

/** 创建空模板 */
export const emptyTierTemplate = () => ({
  name: '',
  ...emptyTierPricing(),
});

// ---- 货币无关的价格 ↔ ratio 转换 --------------------------------

/**
 * 价格 → 后端 ratio
 * 公式：ratio = price / TIER_RATIO_BASE
 * 含义：ratio * model_ratio(2) = price (per 1M tokens)
 */
export const priceToTierRatio = (price) => {
  if (!Number.isFinite(Number(price))) return 0;
  return Math.max(0, Number(price) / TIER_RATIO_BASE);
};

/**
 * 后端 ratio → 显示价格
 * 公式：price = ratio * TIER_RATIO_BASE
 */
export const tierRatioToPrice = (ratio) => {
  if (!Number.isFinite(Number(ratio))) return 0;
  return Math.max(0, Number(ratio) * TIER_RATIO_BASE);
};

// 保留旧版函数名以兼容现有代码中的直接调用
export { priceToTierRatio as priceToRatio };
export { tierRatioToPrice as ratioToPrice };

// ---- 规范化 & 校验 ---------------------------------------------

/** 规范化单条档位：补齐默认值，过滤非法数据 */
export const normalizeTierRow = (row) => {
  const upTo = Number(row?.up_to ?? 0);
  if (!Number.isFinite(upTo) || upTo < 0) return null;
  return {
    up_to: upTo,
    inputPrice: Math.max(0, Number(row?.inputPrice ?? row?.input_price ?? 0)),
    outputPrice: Math.max(0, Number(row?.outputPrice ?? row?.output_price ?? 0)),
    cacheReadPrice: Math.max(
      0,
      Number(row?.cacheReadPrice ?? row?.cache_read_price ?? row?.cacheReadPrice ?? row?.cache_read_price ?? 0),
    ),
    cacheWritePrice: Math.max(
      0,
      Number(row?.cacheWritePrice ?? row?.cache_write_price ?? row?.cacheWritePrice ?? row?.cache_write_price ?? 0),
    ),
  };
};

/** 规范化档位列表：排序 + 保证最后一档无限 + 去重 */
export const normalizeTierRows = (rows) => {
  if (!Array.isArray(rows)) return [];
  // 过滤 + 规范化
  const normalized = rows
    .map(normalizeTierRow)
    .filter(Boolean);

  if (normalized.length === 0) return [];

  // 按 up_to 升序排列（0=无限 放最后）
  const sorted = [...normalized].sort((a, b) => {
    if (a.up_to === 0) return 1;
    if (b.up_to === 0) return -1;
    return a.up_to - b.up_to;
  });

  // 去重：相同 up_to 只保留第一个
  const deduped = [];
  const seen = new Set();
  for (const row of sorted) {
    if (seen.has(row.up_to)) continue;
    seen.add(row.up_to);
    deduped.push(row);
  }

  // 确保最后一行 up_to = 0
  if (deduped.length > 0 && deduped[deduped.length - 1].up_to !== 0) {
    deduped.push({ ...deduped[deduped.length - 1], up_to: 0, inputPrice: 0, outputPrice: 0, cacheReadPrice: 0, cacheWritePrice: 0 });
  }

  return deduped;
};

/** 规范化完整阶梯定价对象 */
export const normalizeTierPricing = (tp) => {
  const src = tp && typeof tp === 'object' ? tp : {};
  return {
    mode: src.mode || TIER_MODE,
    currency: ['USD', 'CNY', 'CUSTOM'].includes(src.currency) ? src.currency : 'USD',
    tiers: normalizeTierRows(src.tiers),
  };
};

/** 检查是否有阶梯定价数据 */
export const hasTierPricing = (tp) => {
  const normalized = normalizeTierPricing(tp);
  return normalized.tiers.length > 0;
};

/** 兼容旧接口：检查是否有 tier 数据 */
export const hasTierRule = (tp) => hasTierPricing(tp);

/** 兼容旧接口：检查是否有 tier segments */
export const hasTierSegments = (tier) => {
  if (!tier) return false;
  if (Array.isArray(tier?.segments)) {
    return tier.segments.length > 0;
  }
  // 新格式
  if (Array.isArray(tier?.tiers)) {
    return tier.tiers.length > 0;
  }
  return false;
};

/**
 * 校验阶梯定价数据
 * @returns {string} 错误信息，空字符串表示无错误
 */
export const validateTierPricing = (tp, t = (v) => v) => {
  const { tiers } = normalizeTierPricing(tp);
  if (tiers.length === 0) return '';

  let previous = 0;
  for (let i = 0; i < tiers.length; i += 1) {
    const row = tiers[i];

    // 检查 up_to 递增
    if (row.up_to === 0) {
      if (i !== tiers.length - 1) {
        return t('只有最后一档上限可以为 0（无限）');
      }
      continue;
    }
    if (row.up_to <= previous) {
      return t('输入Token区间上限必须递增，第 {{index}} 档异常', { index: i + 1 });
    }
    previous = row.up_to;

    // 检查价格非负数（至少输入价格必填）
    if (row.inputPrice < 0 || row.outputPrice < 0 || row.cacheReadPrice < 0 || row.cacheWritePrice < 0) {
      return t('价格不能为负数，第 {{index}} 档异常', { index: i + 1 });
    }
  }

  // 检查至少有一档有有效输入价格
  if (!tiers.some((row) => row.inputPrice > 0)) {
    return t('至少需要一档输入价格大于 0');
  }

  return '';
};

/** 兼容旧接口 */
export const validateTierRule = (tp, t) => validateTierPricing(tp, t);

// ---- 摘要 & 明细 -----------------------------------------------

/** 生成阶梯定价摘要文本 */
export const summarizeTierPricing = (tp, t = (v) => v) => {
  const { currency, tiers } = normalizeTierPricing(tp);
  if (tiers.length === 0) return t('未配置');
  const symbol = getCurrencySymbol(currency);
  return `${tiers.length}${t('档')} / ${t('输入')} ${tiers.map((r) => `${symbol}${Number(r.inputPrice.toFixed(2))}`).join(' → ')}`;
};

/** 兼容旧接口 */
export const summarizeTierRule = (tp, t) => summarizeTierPricing(tp, t);

/**
 * 构建阶梯定价明细（用于 Tooltip / 预览）
 * @returns {{ key: string, label: string, symbol: string, segments: { range: string, price: string }[] }[]}
 */
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
        range: `${from}～${to}`,
        price: price > 0 ? `${symbol}${Number(price.toFixed(6))}` : '-',
        rawPrice: price,
      };
    });
    return { key, label, symbol, segments };
  });
};

// 兼容旧接口：保留 buildTierPriceDetails 的兼容别名
// buildTierPriceDetails 的参数签名已变更，旧的调用者会收到更新后的返回格式

// ---- 序列化 / 反序列化 -----------------------------------------

/**
 * 将统一阶梯定价拆分为 4 个独立 { segments: [{ up_to, ratio }] } 对象
 * 适配后端 4 个 Option Key：ModelTierRatio / CompletionTierRatio / CacheTierRatio / CreateCacheTierRatio
 */
export const unifiedToLegacy = (tp) => {
  const { tiers } = normalizeTierPricing(tp);
  if (tiers.length === 0) {
    return {
      modelTierRatio: null,
      completionTierRatio: null,
      cacheTierRatio: null,
      createCacheTierRatio: null,
    };
  }

  const toSegments = (priceKey) => ({
    segments: tiers.map((row) => ({
      up_to: row.up_to,
      ratio: priceToTierRatio(row[priceKey]),
    })),
  });

  return {
    modelTierRatio: toSegments('inputPrice'),
    completionTierRatio: toSegments('outputPrice'),
    cacheTierRatio: toSegments('cacheReadPrice'),
    createCacheTierRatio: toSegments('cacheWritePrice'),
  };
};

/**
 * 从 4 个独立 legacy 对象合并回统一阶梯定价
 * @param {Object} modelTierRatio — { segments: [{ up_to, ratio }] }
 * @param {Object} completionTierRatio
 * @param {Object} cacheTierRatio
 * @param {Object} createCacheTierRatio
 * @param {string} currency — 货币类型
 * @returns {TierPricing}
 */
export const legacyToUnified = (
  modelTierRatio,
  completionTierRatio,
  cacheTierRatio,
  createCacheTierRatio,
  currency = 'USD',
) => {
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

  if (inputSegments.length === 0) {
    // 尝试从其他三类推断 up_to（兼容旧数据遗留）
    const allSegments = [outputSegments, cacheReadSegments, cacheWriteSegments]
      .find((s) => s.length > 0);
    if (!allSegments) return emptyTierPricing();
  }

  // 以 input 的 up_to 为基准构建 tiers
  // 如果 input 为空但其他非空，使用其他类的 up_to（向后兼容）
  const baseSegments = inputSegments.length > 0 ? inputSegments
    : (outputSegments.length > 0 ? outputSegments
      : (cacheReadSegments.length > 0 ? cacheReadSegments : cacheWriteSegments));

  if (baseSegments.length === 0) return emptyTierPricing();

  // 建立 up_to → ratio 映射
  const outputMap = new Map(outputSegments.map((s) => [s.up_to, s.ratio]));
  const cacheReadMap = new Map(cacheReadSegments.map((s) => [s.up_to, s.ratio]));
  const cacheWriteMap = new Map(cacheWriteSegments.map((s) => [s.up_to, s.ratio]));
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

  return {
    mode: TIER_MODE,
    currency: ['USD', 'CNY', 'CUSTOM'].includes(currency) ? currency : 'USD',
    tiers: normalizeTierRows(tiers),
  };
};

// ---- JSON 解析工具 ---------------------------------------------

/** 安全解析 JSON 字符串为对象 */
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

// ---- 兼容旧版导出（便于逐步迁移）--------------------------------

/** 兼容旧版 TIER_CATEGORIES 引用 */
export const TIER_CATEGORIES = [
  { key: 'input', label: '输入价格' },
  { key: 'output', label: '输出价格' },
  { key: 'cache_read', label: '缓存读取价格' },
  { key: 'cache_write', label: '缓存写入价格' },
];

/** 兼容旧版 normalizeTierRule */
export const normalizeTierRule = (rule) => {
  const src = rule && typeof rule === 'object' ? rule : {};
  // 新格式
  if (Array.isArray(src.tiers)) return normalizeTierPricing(src);
  // 旧格式：4 个独立类别
  return {
    mode: src.mode || TIER_MODE,
    currency: 'USD',
    tiers: normalizeTierRows(
      (Array.isArray(src.input) ? src.input : []).map((row, i) => {
        const outputRow = Array.isArray(src.output) ? src.output[i] : null;
        const cacheReadRow = Array.isArray(src.cache_read) ? src.cache_read[i] : null;
        const cacheWriteRow = Array.isArray(src.cache_write) ? src.cache_write[i] : null;
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

/** 兼容旧版 serializeTierRule */
export const serializeTierRule = (tp) => {
  const inherited = unifiedToLegacy(tp);
  const out = { mode: tp?.mode || TIER_MODE };
  if (inherited.modelTierRatio?.segments?.length > 0) {
    out.input = inherited.modelTierRatio.segments;
  }
  if (inherited.completionTierRatio?.segments?.length > 0) {
    out.output = inherited.completionTierRatio.segments;
  }
  if (inherited.cacheTierRatio?.segments?.length > 0) {
    out.cache_read = inherited.cacheTierRatio.segments;
  }
  if (inherited.createCacheTierRatio?.segments?.length > 0) {
    out.cache_write = inherited.createCacheTierRatio.segments;
  }
  return out;
};

/** 兼容旧版 emptyTierRule */
export const emptyTierRule = emptyTierPricing;

/** 兼容旧版 normalizeTierSegments */
export const normalizeTierSegments = (tier) => {
  const src = tier && typeof tier === 'object' ? tier : {};
  // 新格式：{ tiers: [...] }
  if (Array.isArray(src.tiers)) {
    return { segments: src.tiers.map((r) => ({ up_to: r.up_to, ratio: priceToTierRatio(r.inputPrice) })) };
  }
  // 旧格式：{ segments: [...] }
  return {
    segments: (Array.isArray(src.segments) ? src.segments : [])
      .map((row) => ({
        up_to: Number(row?.up_to || 0),
        ratio: Number(row?.ratio ?? 0),
      }))
      .filter((row) => Number.isFinite(row.up_to) && Number.isFinite(row.ratio)),
  };
};

/** 兼容旧版 ensureFinalInfinityTierSegments */
export const ensureFinalInfinityTierSegments = (segments) => {
  const normalized = Array.isArray(segments) ? segments : [];
  if (!normalized.length) return normalized;
  if (normalized[normalized.length - 1].up_to === 0) return normalized;
  return [...normalized, { up_to: 0, ratio: 0 }];
};
