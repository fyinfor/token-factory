/**
 * 阶梯计费工具函数
 * 提供阶梯计费相关的常量、计算函数和数据处理逻辑
 */

import {
  pickChannelScopedModelTierSegments,
  pickGlobalModelTierSegments,
  costDiscountMultiplier,
  markupRateFromPercent,
} from '../../../../../helpers';

/** 阶梯计费类别 -> 阶梯倍率字段名（渠道与模型全局同名） */
export const TIER_FIELD_MAP = {
  input: 'model_tier_ratio',
  output: 'completion_tier_ratio',
  cache_read: 'cache_tier_ratio',
  cache_write: 'create_cache_tier_ratio',
};

/** 阶梯类别 -> 定价接口全局/渠道阶梯倍率映射字段 */
export const TIER_GLOBAL_MAP_KEYS = {
  input: 'globalModelTierRatio',
  output: 'globalCompletionTierRatio',
  cache_read: 'globalCacheTierRatio',
  cache_write: 'globalCreateCacheTierRatio',
};

export const TIER_CHANNEL_MAP_KEYS = {
  input: 'channelModelTierRatio',
  output: 'channelCompletionTierRatio',
  cache_read: 'channelCacheTierRatio',
  cache_write: 'channelCreateCacheTierRatio',
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

/**
 * 取阶梯倍率 segments（结构：{ segments: [{ up_to, ratio }] }，up_to=0 表示无上限）
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

/**
 * 检查类别是否有阶梯数据
 */
export const categoryHasTierData = (model, channel, cat, tierRatioMaps = {}) => {
  const field = TIER_FIELD_MAP[cat];
  if (getTierSegments(channel, field).length > 0) return true;
  if (getTierSegments(model, field).length > 0) return true;
  const globalMap = tierRatioMaps[TIER_GLOBAL_MAP_KEYS[cat]];
  if (pickGlobalModelTierSegments(globalMap, model.model_name).length > 0) {
    return true;
  }
  const channelMap = tierRatioMaps[TIER_CHANNEL_MAP_KEYS[cat]];
  if (
    pickChannelScopedModelTierSegments(
      channelMap,
      channel?.channel_id,
      model.model_name,
    ).length > 0
  ) {
    return true;
  }
  return false;
};

/**
 * 解析阶梯 segments：官方价用 globalSegments，平台价用 baseSegments
 * 优先级：
 *   globalSegments = 全局映射 → 模型内嵌 → 渠道内嵌（仅无渠道专属覆盖时）
 *   baseSegments   = 渠道专属映射 → globalSegments → 渠道内嵌
 */
export const resolveTierSegmentSources = ({
  model,
  channel,
  channelId,
  cat,
  tierRatioMaps,
}) => {
  const field = TIER_FIELD_MAP[cat];
  const globalMap = tierRatioMaps[TIER_GLOBAL_MAP_KEYS[cat]];
  const channelMap = tierRatioMaps[TIER_CHANNEL_MAP_KEYS[cat]];
  const channelEmbedSegments = getTierSegments(channel, field);
  const channelOnlySegments = pickChannelScopedModelTierSegments(
    channelMap,
    channelId,
    model.model_name,
  );

  let globalSegments = pickGlobalModelTierSegments(globalMap, model.model_name);
  if (globalSegments.length === 0) {
    globalSegments = getTierSegments(model, field);
  }
  if (globalSegments.length === 0 && channelOnlySegments.length === 0) {
    globalSegments = channelEmbedSegments;
  }

  let baseSegments =
    channelOnlySegments.length > 0 ? channelOnlySegments : globalSegments;
  if (baseSegments.length === 0) {
    baseSegments = channelEmbedSegments;
  }

  return { globalSegments, baseSegments, channelOnlySegments };
};

/**
 * 检查模型是否配置了阶梯倍率（渠道内嵌 / quota_type=3 / 全局映射 / 模型内嵌）
 */
export const detectTokenTierPricing = (model, tierRatioMaps = {}) => {
  if (!model?.channel_list || model.channel_list.length === 0) return null;

  for (const ch of model.channel_list) {
    const flags = emptyTierFlags();
    let matched = ch.quota_type === 3;
    for (const { cat, flag } of TIER_CATEGORY_FLAGS) {
      if (categoryHasTierData(model, ch, cat, tierRatioMaps)) {
        flags[flag] = true;
        matched = true;
      }
    }
    if (matched) {
      return { ...flags, channel: ch };
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

export const formatTierRange = (from, to, t) => {
  if (to === 0) return `${formatTierBound(from)}+`;
  return `${formatTierBound(from)}~${formatTierBound(to)}`;
};

/**
 * 构建阶梯计费展示行数据
 * 遍历基准阶梯（渠道阶梯优先，渠道无配置时回退全局阶梯）的所有区间，逐档计算：
 *   官方价     = 全局阶梯对应区间 ratio × 2
 *   阶梯基准原价 = 基准阶梯对应区间 ratio × 2
 *   平台价     = (基准阶梯 ratio × 成本折扣率 + 全局阶梯 ratio × 加价折扣率) × 2 × 分组倍率
 *   折扣率     = round((1 - 平台价/官方价) × 100)%
 */
export const buildTokenTierPreviewItems = (
  baseSegments,
  globalSegments,
  channel,
  tierType,
  usedGroupRatio,
  displayPrice,
  t,
) => {
  if (!Array.isArray(baseSegments) || baseSegments.length === 0) return [];

  const priceDiscountPercent =
    channel.price_discount_percent != null ? channel.price_discount_percent : 100;
  const costDisc = costDiscountMultiplier(priceDiscountPercent);
  const markupRate = markupRateFromPercent(channel.markup_discount_rate || 0);

  // 按区间起点匹配全局阶梯对应档位（up_to=0 代表大于上一区间的所有范围）
  const findGlobalRatio = (from, fallbackRatio) => {
    if (!Array.isArray(globalSegments)) return fallbackRatio;
    for (const seg of globalSegments) {
      const upTo = Number(seg.up_to) || 0;
      if (upTo === 0 || from < upTo) return Number(seg.ratio) || 0;
    }
    return fallbackRatio;
  };

  const rows = [];
  let previousUpTo = 0;
  for (const seg of baseSegments) {
    const baseRatio = Number(seg.ratio) || 0;
    const upTo = Number(seg.up_to) || 0;
    // 全局阶梯无对应区间时回退基准阶梯 ratio
    const globalRatio = findGlobalRatio(previousUpTo, baseRatio);

    // 官方价 = 全局阶梯对应区间 ratio × 2
    const officialUsdPerM = globalRatio * 2;
    // 平台价 = (基准阶梯 ratio × 成本折扣率 + 全局阶梯 ratio × 加价折扣率) × 2 × 分组倍率
    const platformUsdPerM =
      (baseRatio * costDisc + globalRatio * markupRate) * 2 * usedGroupRatio;

    if (!Number.isFinite(platformUsdPerM) || platformUsdPerM <= 0) {
      previousUpTo = upTo || previousUpTo;
      continue;
    }

    const discount =
      Number.isFinite(officialUsdPerM) &&
      officialUsdPerM > 0 &&
      officialUsdPerM > platformUsdPerM
        ? Math.round((1 - platformUsdPerM / officialUsdPerM) * 100)
        : null;

    rows.push({
      key: `${tierType}-${upTo}-${baseRatio}`,
      range: formatTierRange(previousUpTo, upTo, t),
      fromToken: previousUpTo,
      upTo,
      platformPrice: displayPrice(platformUsdPerM),
      platformPriceUsd: platformUsdPerM,
      officialPrice:
        Number.isFinite(officialUsdPerM) && officialUsdPerM > 0
          ? displayPrice(officialUsdPerM)
          : '-',
      officialPriceUsd: officialUsdPerM,
      discount,
      tierType,
    });

    previousUpTo = upTo || previousUpTo;
  }

  return rows;
};
