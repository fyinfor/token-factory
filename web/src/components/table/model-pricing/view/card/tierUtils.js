/**
 * 阶梯计费工具函数
 * 提供阶梯计费相关的常量、计算函数和数据处理逻辑
 */

import {
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
 * 按区间起点在 segments 中查找倍率；未匹配到区间返回 null
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
 * 检查类别是否有阶梯数据（模型全局或渠道内嵌）
 */
export const categoryHasTierData = (model, channel, cat) => {
  const field = TIER_FIELD_MAP[cat];
  return (
    getTierSegments(channel, field).length > 0 ||
    getTierSegments(model, field).length > 0
  );
};

/**
 * 解析阶梯 segments：
 *   globalSegments = 模型全局阶梯（官方价）
 *   channelSegments = 渠道阶梯（平台价基准）
 *   bandSegments    = 区间结构，优先渠道阶梯，否则全局阶梯
 */
export const resolveTierSegmentSources = ({ model, channel, cat }) => {
  const field = TIER_FIELD_MAP[cat];
  const globalSegments = getTierSegments(model, field);
  const channelSegments = getTierSegments(channel, field);
  const bandSegments =
    channelSegments.length > 0 ? channelSegments : globalSegments;
  return { globalSegments, channelSegments, bandSegments };
};

/**
 * 检查模型是否配置了阶梯倍率（渠道内嵌 / quota_type=3 / 模型全局内嵌）
 */
export const detectTokenTierPricing = (model) => {
  if (!model?.channel_list || model.channel_list.length === 0) return null;

  for (const ch of model.channel_list) {
    const flags = emptyTierFlags();
    let matched = ch.quota_type === 3;
    for (const { cat, flag } of TIER_CATEGORY_FLAGS) {
      if (categoryHasTierData(model, ch, cat)) {
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
 * 遍历 bandSegments 的所有区间，逐档计算：
 *   全局倍率 = 全局阶梯对应区间 ratio，无值则为 0
 *   渠道倍率 = 渠道阶梯对应区间 ratio，无值则回退全局倍率
 *   官方价   = 全局倍率 × 2
 *   平台价   = (渠道倍率 × 成本折扣率 + 全局倍率 × 加价折扣率) × 2 × 分组倍率
 *   折扣率   = 仅当官方价 > 0 时计算
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
) => {
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
      range: formatTierRange(previousUpTo, upTo, t),
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
