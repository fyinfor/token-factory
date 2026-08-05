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

import {
  formatMatchingModelName,
  pickChannelScopedModelFloat,
} from './utils';

/** 从 status 读取阶梯基准货币汇率：1 USD = X */
function getTierCurrencyRates() {
  let usdExchangeRate = 7;
  let customExchangeRate = 1;
  try {
    const statusStr = localStorage.getItem('status');
    if (statusStr) {
      const s = JSON.parse(statusStr);
      const cny = Number(s?.usd_exchange_rate);
      const custom = Number(s?.custom_currency_exchange_rate);
      if (Number.isFinite(cny) && cny > 0) usdExchangeRate = cny;
      if (Number.isFinite(custom) && custom > 0) customExchangeRate = custom;
    }
  } catch {
    /* ignore */
  }
  return { USD: 1, CNY: usdExchangeRate, CUSTOM: customExchangeRate };
}

function normalizeTierCurrency(currency) {
  const key = String(currency || '').toUpperCase();
  return ['USD', 'CNY', 'CUSTOM'].includes(key) ? key : 'USD';
}

/** 基准货币单价 → USD；同币种不换算 */
function convertTierPriceToUSD(price, currency, rates = getTierCurrencyRates()) {
  const n = Number(price);
  if (!Number.isFinite(n) || n === 0) return 0;
  const from = normalizeTierCurrency(currency);
  if (from === 'USD') return n;
  const rate = rates[from] > 0 ? rates[from] : 1;
  return n / rate;
}

/** 从阶梯规则或 legacy segments 对象读取 tiers */
function readTierRule(entry) {
  if (!entry || typeof entry !== 'object') return null;
  if (Array.isArray(entry.tiers) && entry.tiers.length > 0) {
    return entry;
  }
  if (Array.isArray(entry.segments) && entry.segments.length > 0) {
    return {
      boundary: 'lt',
      currency: 'USD',
      tiers: entry.segments.map((row) => ({
        up_to: row.up_to,
        prices: {
          input: Number(row.ratio ?? 0) * 2,
          output: 0,
          cache_read: 0,
          cache_write: 0,
        },
      })),
    };
  }
  return null;
}

/** @deprecated 仍支持 legacy segments；优先 unified RequestTierPricing */
function readTierSegments(tierEntry) {
  const rule = readTierRule(tierEntry);
  if (rule?.tiers) {
    const rates = getTierCurrencyRates();
    const currency = normalizeTierCurrency(rule.currency);
    return rule.tiers.map((row) => ({
      up_to: row.up_to,
      ratio: convertTierPriceToUSD(Number(row.prices?.input ?? 0), currency, rates) / 2,
    }));
  }
  return Array.isArray(tierEntry?.segments) ? tierEntry.segments : [];
}

/** 按模型名从全局阶梯映射读取 RequestTierPricing 或 legacy segments */
export function pickGlobalModelTierSegments(globalTierMap, modelName) {
  if (!globalTierMap || modelName == null) return [];
  const formatted = formatMatchingModelName(modelName);
  for (const key of [modelName, formatted]) {
    const segments = readTierSegments(globalTierMap[key]);
    if (segments.length > 0) return segments;
  }
  return [];
}

/** 从渠道专属阶梯映射读取 segments（legacy 兼容层） */
export function pickChannelScopedModelTierSegments(
  channelTierMap,
  channelId,
  modelName,
) {
  if (!channelTierMap || channelId == null || modelName == null) return [];
  const byChannel = channelTierMap[String(channelId)];
  if (!byChannel || typeof byChannel !== 'object') return [];
  const formatted = formatMatchingModelName(modelName);
  for (const key of [modelName, formatted]) {
    const segments = readTierSegments(byChannel[key]);
    if (segments.length > 0) return segments;
  }
  return [];
}

/** 读取全局 unified RequestTierPricing */
export function pickGlobalModelRequestTierPricing(globalTierMap, modelName) {
  if (!globalTierMap || modelName == null) return null;
  const formatted = formatMatchingModelName(modelName);
  for (const key of [modelName, formatted]) {
    const rule = readTierRule(globalTierMap[key]);
    if (rule) return rule;
  }
  return null;
}

/** 读取渠道 scoped unified RequestTierPricing */
export function pickChannelScopedModelRequestTierPricing(
  channelTierMap,
  channelId,
  modelName,
) {
  if (!channelTierMap || channelId == null || modelName == null) return null;
  const byChannel = channelTierMap[String(channelId)];
  if (!byChannel || typeof byChannel !== 'object') return null;
  const formatted = formatMatchingModelName(modelName);
  for (const key of [modelName, formatted]) {
    const rule = readTierRule(byChannel[key]);
    if (rule) return rule;
  }
  return null;
}

/** Claude 1h 缓存创建全局倍率相对 5m 的乘数（与 service/text_quota.go 一致） */
export const CLAUDE_CACHE_CREATE_1H_GLOBAL_MULT = 6 / 3.75;

/** 成本折扣率百分数 → 乘数（100=1.0） */
export function costDiscountMultiplier(percent) {
  const n = Number(percent);
  if (!Number.isFinite(n)) {
    return 1;
  }
  return Math.max(0, n) / 100;
}

/** 加价折扣率百分数 → 乘数 */
export function markupRateFromPercent(percent) {
  return (Number(percent) || 0) / 100;
}

/** 解析数值，缺省为 0（保留合法的 0） */
export function numOrZero(value) {
  if (value == null || value === '') {
    return 0;
  }
  const n = Number(value);
  return Number.isFinite(n) ? n : 0;
}

/** 判断倍率/价格字段是否可用于展示与计算 */
export function hasBillingRatioValue(value) {
  return (
    value !== undefined &&
    value !== null &&
    value !== '' &&
    Number.isFinite(Number(value))
  );
}

/**
 * 解析成本价单项数值：Option 渠道模型定价（channel 字段 / 映射）→ 全局模型定价。
 * 不使用 channel_list 已合并倍率，避免误用供应商或全局回退价。
 */
export function resolveCostOptionChannelScalar(
  optionChannelValue,
  channelMap,
  channelId,
  modelName,
  globalValue,
) {
  if (hasBillingRatioValue(optionChannelValue)) {
    return Number(optionChannelValue);
  }
  const fromChannelMap = pickChannelScopedModelFloat(
    channelMap,
    channelId,
    modelName,
  );
  if (hasBillingRatioValue(fromChannelMap)) {
    return Number(fromChannelMap);
  }
  if (hasBillingRatioValue(globalValue)) {
    return Number(globalValue);
  }
  return null;
}

/** 将存储倍率转为与定价编辑器一致的 $/1M tokens 展示价（倍率 × 2） */
export function modelRatioToDisplayUsdPerM(modelRatio) {
  return Number(modelRatio) * 2;
}

/**
 * 计算渠道成本价各项（仅 渠道/全局原价 × 成本折扣率，不含加价与分组倍率）。
 * 返回按量/按次计费下的展示用有效倍率或固定价（未乘 ×2，由调用方格式化）。
 */
export function computeChannelCostRates({
  channelId,
  modelName,
  optionModelRatio,
  optionCompletionRatio,
  optionCacheRatio,
  optionCreateCacheRatio,
  optionModelPrice,
  optionImageRatio,
  optionImagePrice,
  optionAudioRatio,
  optionAudioCompletionRatio,
  optionVideoRatio,
  optionVideoCompletionRatio,
  optionVideoPrice,
  channelModelRatioMap,
  channelCompletionRatioMap,
  channelCacheRatioMap,
  channelCreateCacheRatioMap,
  channelModelPriceMap,
  channelImageRatioMap,
  channelImagePriceMap,
  channelAudioRatioMap,
  channelAudioCompletionRatioMap,
  channelVideoRatioMap,
  channelVideoCompletionRatioMap,
  channelVideoPriceMap,
  priceDiscountPercent = 100,
  globalModelRatio = 0,
  globalModelPrice = 0,
  globalCompletionRatio = 0,
  globalCacheRatio = 0,
  globalCreateCacheRatio = 0,
  globalImageRatio,
  globalImagePrice = -1,
  globalAudioRatio,
  globalAudioCompletionRatio,
  globalVideoRatio,
  globalVideoCompletionRatio,
  globalVideoPrice = -1,
  skipImageTokenPricing = false,
  skipImageFlatSimple = false,
  skipVideoTokenPricing = false,
  skipVideoFlatSimple = false,
  quotaType = 0,
} = {}) {
  const costDisc = costDiscountMultiplier(priceDiscountPercent);
  // 0=按量计费, 1=按次计费, 3=阶梯计费（按量口径）
  const isPerToken = quotaType === 0 || quotaType === 3;
  const items = [];

  /** 追加按量计费成本项（$/1M tokens 展示价） */
  const pushTokenCostItem = (key, labelKey, usdPerM) => {
    if (
      usdPerM == null ||
      !Number.isFinite(usdPerM) ||
      usdPerM <= 0
    ) {
      return;
    }
    items.push({
      key,
      labelKey,
      displayUsdPerM: usdPerM,
      isFixedPrice: false,
    });
  };

  /** 追加固定单价成本项（按次/按张等） */
  const pushFixedCostItem = (key, labelKey, usdAmount, fixedUnitKey = '次') => {
    if (usdAmount == null || !Number.isFinite(usdAmount) || usdAmount < 0) {
      return;
    }
    items.push({
      key,
      labelKey,
      displayUsdPerM: usdAmount * costDisc,
      isFixedPrice: true,
      fixedUnitKey,
    });
  };

  if (!isPerToken) {
    const baseMp = resolveCostOptionChannelScalar(
      optionModelPrice,
      channelModelPriceMap,
      channelId,
      modelName,
      globalModelPrice,
    );
    if (baseMp != null && baseMp >= 0) {
      items.push({
        key: 'model_price',
        labelKey: '模型价格',
        displayUsdPerM: baseMp * costDisc,
        isFixedPrice: true,
      });
    }
    return items;
  }

  const baseMr = resolveCostOptionChannelScalar(
    optionModelRatio,
    channelModelRatioMap,
    channelId,
    modelName,
    globalModelRatio,
  );
  if (baseMr == null) {
    return items;
  }

  const inputUsdPerM = modelRatioToDisplayUsdPerM(baseMr) * costDisc;
  items.push({
    key: 'input',
    labelKey: '输入价格',
    displayUsdPerM: inputUsdPerM,
    isFixedPrice: false,
  });

  const completionRatio = resolveCostOptionChannelScalar(
    optionCompletionRatio,
    channelCompletionRatioMap,
    channelId,
    modelName,
    globalCompletionRatio,
  );
  if (
    completionRatio != null &&
    hasBillingRatioValue(globalCompletionRatio)
  ) {
    items.push({
      key: 'output',
      labelKey: '输出价格',
      displayUsdPerM:
        modelRatioToDisplayUsdPerM(baseMr) * completionRatio * costDisc,
      isFixedPrice: false,
    });
  }

  const cacheRatio = resolveCostOptionChannelScalar(
    optionCacheRatio,
    channelCacheRatioMap,
    channelId,
    modelName,
    globalCacheRatio,
  );
  if (cacheRatio != null && hasBillingRatioValue(globalCacheRatio)) {
    items.push({
      key: 'cache_read',
      labelKey: '缓存读取价格',
      displayUsdPerM:
        modelRatioToDisplayUsdPerM(baseMr) * cacheRatio * costDisc,
      isFixedPrice: false,
    });
  }

  const createCacheRatio = resolveCostOptionChannelScalar(
    optionCreateCacheRatio,
    channelCreateCacheRatioMap,
    channelId,
    modelName,
    globalCreateCacheRatio,
  );
  if (
    createCacheRatio != null &&
    hasBillingRatioValue(globalCreateCacheRatio)
  ) {
    pushTokenCostItem(
      'cache_create',
      '缓存创建价格',
      modelRatioToDisplayUsdPerM(baseMr) * createCacheRatio * costDisc,
    );
  }

  const globalImagePriceFallback =
    hasBillingRatioValue(globalImagePrice) && Number(globalImagePrice) >= 0
      ? Number(globalImagePrice)
      : null;
  const globalVideoPriceFallback =
    hasBillingRatioValue(globalVideoPrice) && Number(globalVideoPrice) >= 0
      ? Number(globalVideoPrice)
      : null;

  if (!skipImageTokenPricing && hasBillingRatioValue(globalImageRatio)) {
    const imageRatio = resolveCostOptionChannelScalar(
      optionImageRatio,
      channelImageRatioMap,
      channelId,
      modelName,
      globalImageRatio,
    );
    if (baseMr != null && imageRatio != null) {
      pushTokenCostItem(
        'image',
        '图片输入价格',
        modelRatioToDisplayUsdPerM(baseMr) * imageRatio * costDisc,
      );
    }
  }

  if (!skipImageFlatSimple) {
    const imageFlatUsd = resolveCostOptionChannelScalar(
      optionImagePrice,
      channelImagePriceMap,
      channelId,
      modelName,
      globalImagePriceFallback,
    );
    if (imageFlatUsd != null && imageFlatUsd > 0) {
      pushFixedCostItem('image_flat', '图片价格', imageFlatUsd, '张');
    }
  }

  if (hasBillingRatioValue(globalAudioRatio)) {
    const audioRatio = resolveCostOptionChannelScalar(
      optionAudioRatio,
      channelAudioRatioMap,
      channelId,
      modelName,
      globalAudioRatio,
    );
    if (baseMr != null && audioRatio != null) {
      pushTokenCostItem(
        'audio_input',
        '音频输入价格',
        modelRatioToDisplayUsdPerM(baseMr) * audioRatio * costDisc,
      );
      if (hasBillingRatioValue(globalAudioCompletionRatio)) {
        const audioCompletionRatio = resolveCostOptionChannelScalar(
          optionAudioCompletionRatio,
          channelAudioCompletionRatioMap,
          channelId,
          modelName,
          globalAudioCompletionRatio,
        );
        if (audioCompletionRatio != null) {
          pushTokenCostItem(
            'audio_output',
            '音频输出价格',
            modelRatioToDisplayUsdPerM(baseMr) *
              audioRatio *
              audioCompletionRatio *
              costDisc,
          );
        }
      }
    }
  }

  if (!skipVideoTokenPricing && hasBillingRatioValue(globalVideoRatio)) {
    const videoRatio = resolveCostOptionChannelScalar(
      optionVideoRatio,
      channelVideoRatioMap,
      channelId,
      modelName,
      globalVideoRatio,
    );
    if (baseMr != null && videoRatio != null) {
      pushTokenCostItem(
        'video_input',
        '视频输入价格',
        modelRatioToDisplayUsdPerM(baseMr) * videoRatio * costDisc,
      );
      if (hasBillingRatioValue(globalVideoCompletionRatio)) {
        const videoCompletionRatio = resolveCostOptionChannelScalar(
          optionVideoCompletionRatio,
          channelVideoCompletionRatioMap,
          channelId,
          modelName,
          globalVideoCompletionRatio,
        );
        if (videoCompletionRatio != null) {
          pushTokenCostItem(
            'video_output',
            '视频输出价格',
            modelRatioToDisplayUsdPerM(baseMr) *
              videoRatio *
              videoCompletionRatio *
              costDisc,
          );
        }
      }
    }
  }

  if (!skipVideoFlatSimple) {
    const videoFlatUsd = resolveCostOptionChannelScalar(
      optionVideoPrice,
      channelVideoPriceMap,
      channelId,
      modelName,
      globalVideoPriceFallback,
    );
    if (videoFlatUsd != null && videoFlatUsd > 0) {
      pushFixedCostItem('video_flat', '视频价格', videoFlatUsd, '次');
    }
  }

  return items;
}

/**
 * 渠道计费新公式（与首页 PricingCardView 一致）。
 * 展示价在有效倍率上乘以 2；固定价不含 ×2。
 */
export function computeChannelBillingRates({
  channelModelRatio = 0,
  channelCompletionRatio = 0,
  channelCacheRatio = 0,
  channelCreateCacheRatio = 0,
  channelCreateCacheRatio5m,
  channelCreateCacheRatio1h,
  channelModelPrice = -1,
  priceDiscountPercent = 100,
  markupDiscountPercent = 0,
  globalModelRatio = 0,
  globalModelPrice = 0,
  globalCompletionRatio = 0,
  globalCacheRatio = 0,
  globalCreateCacheRatio = 0,
} = {}) {
  const costDisc = costDiscountMultiplier(priceDiscountPercent);
  const markupRate = markupRateFromPercent(markupDiscountPercent);
  const chMr = numOrZero(channelModelRatio);
  const globalMr = numOrZero(globalModelRatio);
  const cr = numOrZero(channelCompletionRatio);
  const cacheR = numOrZero(channelCacheRatio);
  const createCacheR = numOrZero(channelCreateCacheRatio);
  const createCacheR5m =
    channelCreateCacheRatio5m != null
      ? numOrZero(channelCreateCacheRatio5m)
      : createCacheR;
  const createCacheR1h =
    channelCreateCacheRatio1h != null
      ? numOrZero(channelCreateCacheRatio1h)
      : createCacheR;
  const globalCR = numOrZero(globalCompletionRatio);
  const globalCacheR = numOrZero(globalCacheRatio);
  const globalCreateCacheR = numOrZero(globalCreateCacheRatio);
  const globalCreateCacheR1h = globalCreateCacheR * CLAUDE_CACHE_CREATE_1H_GLOBAL_MULT;

  const effInputRate = chMr * costDisc + globalMr * markupRate;
  const effOutputRate =
    chMr * cr * costDisc + globalMr * globalCR * markupRate;
  const effCacheReadRate =
    chMr * cacheR * costDisc + globalMr * globalCacheR * markupRate;
  const effCacheCreateRate =
    chMr * createCacheR * costDisc + globalMr * globalCreateCacheR * markupRate;
  const effCacheCreate5mRate =
    chMr * createCacheR5m * costDisc + globalMr * globalCreateCacheR * markupRate;
  const effCacheCreate1hRate =
    chMr * createCacheR1h * costDisc + globalMr * globalCreateCacheR1h * markupRate;

  const globalMp = numOrZero(globalModelPrice);
  const effModelPrice =
    channelModelPrice === -1 ||
    channelModelPrice === undefined ||
    channelModelPrice === null
      ? -1
      : numOrZero(channelModelPrice) * costDisc + globalMp * markupRate;

  return {
    costDisc,
    markupRate,
    effInputRate,
    effOutputRate,
    effCacheReadRate,
    effCacheCreateRate,
    effCacheCreate5mRate,
    effCacheCreate1hRate,
    effModelPrice,
    inputRatioPrice: effInputRate * 2,
    completionRatioPrice: effOutputRate * 2,
    cacheRatioPrice: effCacheReadRate * 2,
    cacheCreationRatioPrice: effCacheCreateRate * 2,
    cacheCreationRatioPrice5m: effCacheCreate5mRate * 2,
    cacheCreationRatioPrice1h: effCacheCreate1hRate * 2,
    mp: effModelPrice,
    mr: effInputRate,
  };
}

/**
 * 消费日志：从 other + 调用参数解析有效倍率（与首页卡片同一套 computeChannelBillingRates）。
 */
export function resolveConsumeLogBillingRates({
  modelRatio = 0,
  completionRatio = 0,
  cacheRatio = 0,
  cacheCreationRatio = 0,
  cacheCreationRatio5m,
  cacheCreationRatio1h,
  modelPrice = -1,
  channelPriceDiscountPercent = 100,
  billingMeta = null,
} = {}) {
  const meta =
    billingMeta && typeof billingMeta === 'object' ? billingMeta : {};
  return computeChannelBillingRates({
    channelModelRatio:
      meta.model_ratio != null ? meta.model_ratio : modelRatio,
    channelCompletionRatio:
      meta.completion_ratio != null ? meta.completion_ratio : completionRatio,
    channelCacheRatio: meta.cache_ratio != null ? meta.cache_ratio : cacheRatio,
    channelCreateCacheRatio:
      meta.cache_creation_ratio != null
        ? meta.cache_creation_ratio
        : cacheCreationRatio,
    channelCreateCacheRatio5m:
      meta.cache_creation_ratio_5m != null
        ? meta.cache_creation_ratio_5m
        : cacheCreationRatio5m,
    channelCreateCacheRatio1h:
      meta.cache_creation_ratio_1h != null
        ? meta.cache_creation_ratio_1h
        : cacheCreationRatio1h,
    channelModelPrice:
      meta.model_price != null && meta.model_price !== -1
        ? meta.model_price
        : modelPrice,
    priceDiscountPercent:
      meta.channel_price_discount_percent ?? channelPriceDiscountPercent ?? 100,
    markupDiscountPercent: meta.markup_discount_rate,
    globalModelRatio: meta.global_model_ratio,
    globalModelPrice: meta.global_model_price,
    globalCompletionRatio: meta.global_completion_ratio,
    globalCacheRatio: meta.global_cache_ratio,
    globalCreateCacheRatio: meta.global_create_cache_ratio,
  });
}

/** 读取与 render.getCurrencyConfig 一致的展示货币配置 */
export function getBillingCurrencyConfig() {
  const quotaDisplayType = localStorage.getItem('quota_display_type') || 'USD';
  const statusStr = localStorage.getItem('status');
  let symbol = '$';
  let rate = 1;
  if (quotaDisplayType === 'CNY') {
    symbol = '¥';
    try {
      if (statusStr) {
        const s = JSON.parse(statusStr);
        rate = s?.usd_exchange_rate || 7;
      }
    } catch (e) {
      /* ignore */
    }
  } else if (quotaDisplayType === 'CUSTOM') {
    try {
      if (statusStr) {
        const s = JSON.parse(statusStr);
        symbol = s?.custom_currency_symbol || '¤';
        rate = s?.custom_currency_exchange_rate || 1;
      }
    } catch (e) {
      /* ignore */
    }
  }
  return { symbol, rate, type: quotaDisplayType };
}

/**
 * ASR 用户实付每秒单价（USD）：成本/加价折扣后的有效价 × 分组/专属倍率。
 * 优先使用后端写入的 other.asr_unit_price（与实扣一致）。
 */
export function resolveASRUserPerSecondUsd(other = {}) {
  const explicit = Number(other?.asr_unit_price);
  if (Number.isFinite(explicit) && explicit > 0) {
    return explicit;
  }
  const chPct = Number(other?.channel_price_discount_percent ?? 100);
  const { effModelPrice } = resolveConsumeLogBillingRates({
    modelPrice: other?.model_price,
    channelPriceDiscountPercent: chPct,
    billingMeta: other,
  });
  const userGroupRatio = Number(other?.user_group_ratio);
  const groupRatio = Number(other?.group_ratio);
  const useUserGroupRatio = Number.isFinite(userGroupRatio) && userGroupRatio !== -1;
  const effectiveGroupRatio = useUserGroupRatio
    ? userGroupRatio
    : Number.isFinite(groupRatio) && groupRatio !== -1
      ? groupRatio
      : 1;
  const unit = Number(effModelPrice);
  if (!Number.isFinite(unit) || unit <= 0) {
    return 0;
  }
  return unit * (Number(effectiveGroupRatio) || 1);
}

/**
 * ASR 每秒单价展示（含货币符号），与使用日志「每秒价格」口径一致。
 */
export function formatASRUserPerSecondPrice(other = {}) {
  const { symbol, rate } = getBillingCurrencyConfig();
  const usd = resolveASRUserPerSecondUsd(other);
  return {
    symbol,
    price: formatTierUsdPrice(usd * (Number(rate) || 1)),
    usd,
  };
}

/**
 * 阶梯/日志单价：最多 6 位小数，去除尾随零（7.500000 → 7.5，0.06666666 → 0.066667）
 */
export function formatTierUsdPrice(usdAmount) {
  const n = Number(usdAmount);
  if (!Number.isFinite(n)) {
    return '0';
  }
  let s = n.toFixed(6);
  if (s.includes('.')) {
    s = s.replace(/0+$/, '').replace(/\.$/, '');
  }
  return s;
}

export function formatBillingUsdDisplay(usdAmount, { tokenUnit = 'M' } = {}) {
  const { symbol, rate } = getBillingCurrencyConfig();
  const unitDivisor = tokenUnit === 'K' ? 1000 : 1;
  const raw = (Number(usdAmount) || 0) * rate;
  if (!Number.isFinite(raw)) {
    return `${symbol}0`;
  }
  const numeric = parseFloat((raw / unitDivisor).toFixed(2));
  return `${symbol}${numeric}`;
}
