/**
 * 阶梯计费工具函数
 * 提供阶梯计费相关的常量、计算函数和数据处理逻辑
 */

import {
  costDiscountMultiplier,
  markupRateFromPercent,
} from '../../../../../helpers';
import {
  convertTierPriceToUSD,
  getCurrencyRatesFromStatus,
  normalizeCurrency,
} from '../../../../../pages/Setting/Ratio/utils/requestTierPricing';

/** 阶梯计费类别 → prices 字段名 */
export const TIER_PRICE_KEYS = {
  input: 'input',
  output: 'output',
  cache_read: 'cache_read',
  cache_write: 'cache_write',
};

/** @deprecated 旧 4 Key 字段映射，仅 legacy 回退 */
export const TIER_FIELD_MAP = {
  input: 'model_tier_ratio',
  output: 'completion_tier_ratio',
  cache_read: 'cache_tier_ratio',
  cache_write: 'create_cache_tier_ratio',
};

export const TIER_CATEGORY_FLAGS = [
  { cat: 'input', flag: 'hasModelTier' },
  { cat: 'output', flag: 'hasCompletionTier' },
  { cat: 'cache_read', flag: 'hasCacheTier' },
  { cat: 'cache_write', flag: 'hasCreateCacheTier' },
];

/**
 * 阶梯计费类别配置：颜色与标签
 */
export const TIER_CATEGORY_STYLES = {
  input: {
    labelKey: '输入',
    color: 'blue',
    backgroundColor: 'rgba(var(--semi-blue-0), .55)',
    rowBackgroundColor: 'rgba(var(--semi-blue-0), .22)',
    textColor: 'var(--semi-blue-7)',
    borderColor: 'var(--semi-blue-4)',
  },
  output: {
    labelKey: '输出',
    color: 'violet',
    backgroundColor: 'rgba(var(--semi-violet-0), .55)',
    rowBackgroundColor: 'rgba(var(--semi-violet-0), .22)',
    textColor: 'var(--semi-violet-7)',
    borderColor: 'var(--semi-violet-4)',
  },
  cache_read: {
    labelKey: '缓存读取',
    color: 'cyan',
    backgroundColor: 'rgba(var(--semi-cyan-0), .55)',
    rowBackgroundColor: 'rgba(var(--semi-cyan-0), .22)',
    textColor: 'var(--semi-cyan-7)',
    borderColor: 'var(--semi-cyan-4)',
  },
  cache_write: {
    labelKey: '缓存写入',
    color: 'amber',
    backgroundColor: 'rgba(var(--semi-amber-0), .55)',
    rowBackgroundColor: 'rgba(var(--semi-amber-0), .22)',
    textColor: 'var(--semi-amber-7)',
    borderColor: 'var(--semi-amber-4)',
  },
};

const normalizeBoundary = (boundary) =>
  String(boundary || '').toLowerCase() === 'lte' ? 'lte' : 'lt';

/** 读取统一 RequestTierPricing（model / channel 内嵌字段） */
export const getRequestTierPricing = (source) => {
  const rtp = source?.request_tier_pricing;
  if (!rtp || typeof rtp !== 'object') return null;
  if (!Array.isArray(rtp.tiers) || rtp.tiers.length === 0) return null;
  return rtp;
};

/**
 * 取阶梯倍率 segments（legacy：{ segments: [{ up_to, ratio }] }）
 */
export const getTierSegments = (source, field) =>
  Array.isArray(source?.[field]?.segments) ? source[field].segments : [];

/**
 * 返回空的阶梯标志对象
 */
export const emptyTierFlags = () => ({
  hasModelTier: false,
  hasCompletionTier: false,
  hasCacheTier: false,
  hasCreateCacheTier: false,
});

const tierCategoryHasPrice = (rule, cat) => {
  if (!rule?.tiers?.length) return false;
  const key = TIER_PRICE_KEYS[cat];
  return rule.tiers.some((row) => Number(row?.prices?.[key] ?? 0) > 0);
};

/**
 * 按区间起点在 tiers 中查找单价；未匹配到区间返回 null
 */
export const findTierPriceAtBand = (tiers, fromToken, priceKey, boundary = 'lt') => {
  if (!Array.isArray(tiers) || tiers.length === 0) return null;
  const b = normalizeBoundary(boundary);
  for (let i = 0; i < tiers.length; i += 1) {
    const row = tiers[i];
    const upTo = Number(row?.up_to) || 0;
    if (upTo === 0) {
      const price = Number(row?.prices?.[priceKey] ?? 0);
      return Number.isFinite(price) ? price : null;
    }
    if (b === 'lte') {
      if (fromToken <= upTo) {
        const price = Number(row?.prices?.[priceKey] ?? 0);
        return Number.isFinite(price) ? price : null;
      }
    } else if (fromToken < upTo) {
      const price = Number(row?.prices?.[priceKey] ?? 0);
      return Number.isFinite(price) ? price : null;
    }
  }
  return null;
};

/**
 * 按区间起点在 segments 中查找倍率；未匹配到区间返回 null（legacy）
 */
export const findTierRatioAtBand = (segments, fromToken) => {
  if (!Array.isArray(segments) || segments.length === 0) return null;
  for (const seg of segments) {
    const upTo = Number(seg.up_to) || 0;
    if (upTo === 0 || fromToken < upTo) {
      const ratio = Number(seg.ratio);
      return Number.isFinite(ratio) ? ratio : null;
    }
  }
  return null;
};

/**
 * 检查类别是否有阶梯数据（统一 rule 或 legacy segments）
 */
export const categoryHasTierData = (model, channel, cat) => {
  const globalRule = getRequestTierPricing(model);
  const channelRule = getRequestTierPricing(channel);
  if (tierCategoryHasPrice(channelRule, cat) || tierCategoryHasPrice(globalRule, cat)) {
    return true;
  }
  const field = TIER_FIELD_MAP[cat];
  return (
    getTierSegments(channel, field).length > 0 ||
    getTierSegments(model, field).length > 0
  );
};

/**
 * 解析阶梯数据源：
 *   globalRule / channelRule = 统一 RequestTierPricing
 *   bandRule = 优先渠道，否则全局
 *   legacy segments 字段在 unified=false 时使用
 */
export const resolveTierSegmentSources = ({ model, channel, cat }) => {
  const globalRule = getRequestTierPricing(model);
  const channelRule = getRequestTierPricing(channel);
  const bandRule = channelRule || globalRule;

  if (bandRule) {
    return {
      unified: true,
      globalRule,
      channelRule,
      bandRule,
      boundary: normalizeBoundary(bandRule.boundary),
      globalSegments: [],
      channelSegments: [],
      bandSegments: bandRule.tiers || [],
    };
  }

  const field = TIER_FIELD_MAP[cat];
  const globalSegments = getTierSegments(model, field);
  const channelSegments = getTierSegments(channel, field);
  const bandSegments =
    channelSegments.length > 0 ? channelSegments : globalSegments;
  return {
    unified: false,
    globalRule: null,
    channelRule: null,
    bandRule: null,
    boundary: 'lt',
    globalSegments,
    channelSegments,
    bandSegments,
  };
};

/**
 * 检查模型是否配置了阶梯倍率（渠道内嵌 / quota_type=3 / 模型全局内嵌）
 */
export const detectTokenTierPricing = (model) => {
  if (!model?.channel_list || model.channel_list.length === 0) return null;

  for (const ch of model.channel_list) {
    const globalRule = getRequestTierPricing(model);
    const channelRule = getRequestTierPricing(ch);
    const rule = channelRule || globalRule;
    const flags = emptyTierFlags();
    let matched = ch.quota_type === 3;

    for (const { cat, flag } of TIER_CATEGORY_FLAGS) {
      if (categoryHasTierData(model, ch, cat)) {
        flags[flag] = true;
        matched = true;
      }
    }

    if (matched) {
      return {
        ...flags,
        channel: ch,
        boundary: normalizeBoundary(rule?.boundary),
      };
    }
  }
  return null;
};

/**
 * 格式化阶梯 Token 区间显示文本
 */
export const formatTierBound = (v) => {
  if (v >= 1000000) return `${(v / 1000000).toFixed(v % 1000000 === 0 ? 0 : 1)}M`;
  if (v >= 1000) return `${(v / 1000).toFixed(v % 1000 === 0 ? 0 : 1)}K`;
  return String(v);
};

export const formatTierRange = (from, to, t, boundary = 'lt') => {
  const b = normalizeBoundary(boundary);
  if (to === 0) return `${formatTierBound(from)}+`;
  if (b === 'lte') {
    return `${formatTierBound(from)}~${formatTierBound(to)} (≤)`;
  }
  return `${formatTierBound(from)}~${formatTierBound(to)} (<)`;
};

const buildUnifiedTierPreviewItems = (
  bandTiers,
  globalRule,
  channelRule,
  boundary,
  channel,
  tierType,
  usedGroupRatio,
  displayPrice,
  t,
) => {
  if (!Array.isArray(bandTiers) || bandTiers.length === 0) return [];
  const priceKey = TIER_PRICE_KEYS[tierType];
  const globalTiers = globalRule?.tiers || [];
  const channelTiers = channelRule?.tiers || [];
  const currencyRates = getCurrencyRatesFromStatus();
  const globalCurrency = normalizeCurrency(globalRule?.currency);
  const channelCurrency = normalizeCurrency(
    channelRule?.currency || globalRule?.currency,
  );
  const priceDiscountPercent =
    channel.price_discount_percent != null ? channel.price_discount_percent : 100;
  const costDisc = costDiscountMultiplier(priceDiscountPercent);
  const markupRate = markupRateFromPercent(channel.markup_discount_rate || 0);

  const rows = [];
  let previousUpTo = 0;
  for (const seg of bandTiers) {
    const upTo = Number(seg.up_to) || 0;
    // 按区间起点取价时固定用 lt 语义（与旧 FindTierRatioAtBand 一致），boundary 只影响区间文案与实扣命中
    const globalPriceRaw =
      findTierPriceAtBand(globalTiers, previousUpTo, priceKey, 'lt') ?? 0;
    const channelPriceRaw =
      channelTiers.length > 0
        ? findTierPriceAtBand(channelTiers, previousUpTo, priceKey, 'lt')
        : null;
    const globalPrice = convertTierPriceToUSD(
      globalPriceRaw,
      globalCurrency,
      currencyRates,
    );
    const channelPrice =
      channelPriceRaw != null
        ? convertTierPriceToUSD(channelPriceRaw, channelCurrency, currencyRates)
        : null;
    const effectiveChannelPrice =
      channelTiers.length > 0 && channelPrice != null ? channelPrice : globalPrice;

    const officialUsdPerM = globalPrice;
    const platformUsdPerM =
      (effectiveChannelPrice * costDisc + globalPrice * markupRate) *
      usedGroupRatio;

    if (!Number.isFinite(platformUsdPerM) || platformUsdPerM <= 0) {
      previousUpTo = upTo || previousUpTo;
      continue;
    }
    const discount =
      officialUsdPerM > 0 && officialUsdPerM > platformUsdPerM
        ? Math.round((1 - platformUsdPerM / officialUsdPerM) * 100)
        : null;

    rows.push({
      key: `${tierType}-${upTo}-${effectiveChannelPrice}`,
      range: formatTierRange(previousUpTo, upTo, t, boundary),
      fromToken: previousUpTo,
      upTo,
      platformPrice: displayPrice(platformUsdPerM),
      platformPriceUsd: platformUsdPerM,
      officialPrice: officialUsdPerM > 0 ? displayPrice(officialUsdPerM) : '-',
      officialPriceUsd: officialUsdPerM,
      discount,
      tierType,
    });
    previousUpTo = upTo || previousUpTo;
  }
  return rows;
};

/**
 * 构建阶梯计费展示行数据
 */
export const buildTokenTierPreviewItems = (
  bandSegments,
  globalSegments,
  channelSegments,
  channel,
  tierType,
  usedGroupRatio,
  displayPrice,
  t,
  segmentSources = null,
) => {
  if (segmentSources?.unified) {
    return buildUnifiedTierPreviewItems(
      segmentSources.bandSegments || bandSegments,
      segmentSources.globalRule,
      segmentSources.channelRule,
      segmentSources.boundary || 'lt',
      channel,
      tierType,
      usedGroupRatio,
      displayPrice,
      t,
    );
  }

  if (!Array.isArray(bandSegments) || bandSegments.length === 0) return [];
  const priceDiscountPercent =
    channel.price_discount_percent != null ? channel.price_discount_percent : 100;
  const costDisc = costDiscountMultiplier(priceDiscountPercent);
  const markupRate = markupRateFromPercent(channel.markup_discount_rate || 0);

  const rows = [];
  let previousUpTo = 0;
  for (const seg of bandSegments) {
    const upTo = Number(seg.up_to) || 0;
    const globalRatio = findTierRatioAtBand(globalSegments, previousUpTo) ?? 0;
    const channelRatio = findTierRatioAtBand(channelSegments, previousUpTo);
    const baseRatio =
      channelSegments.length > 0
        ? channelRatio != null
          ? channelRatio
          : globalRatio
        : globalRatio;

    const officialUsdPerM = globalRatio * 2;
    const platformUsdPerM =
      (baseRatio * costDisc + globalRatio * markupRate) * 2 * usedGroupRatio;

    if (!Number.isFinite(platformUsdPerM) || platformUsdPerM <= 0) {
      previousUpTo = upTo || previousUpTo;
      continue;
    }
    const discount =
      officialUsdPerM > 0 && officialUsdPerM > platformUsdPerM
        ? Math.round((1 - platformUsdPerM / officialUsdPerM) * 100)
        : null;

    rows.push({
      key: `${tierType}-${upTo}-${baseRatio}`,
      range: formatTierRange(previousUpTo, upTo, t, 'lt'),
      fromToken: previousUpTo,
      upTo,
      platformPrice: displayPrice(platformUsdPerM),
      platformPriceUsd: platformUsdPerM,
      officialPrice: officialUsdPerM > 0 ? displayPrice(officialUsdPerM) : '-',
      officialPriceUsd: officialUsdPerM,
      discount,
      tierType,
    });
    previousUpTo = upTo || previousUpTo;
  }
  return rows;
};
