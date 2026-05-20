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

/** Claude 1h 缓存创建全局倍率相对 5m 的乘数（与 service/text_quota.go 一致） */
export const CLAUDE_CACHE_CREATE_1H_GLOBAL_MULT = 6 / 3.75;

/** 成本折扣率百分数 → 乘数（100=1.0） */
export function costDiscountMultiplier(percent) {
  const n = Number(percent);
  if (!Number.isFinite(n) || n <= 0) {
    return 1;
  }
  return n / 100;
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
 * 将 USD 单价格式化为与首页卡片 formatPrice 一致的展示字符串（默认 2 位小数）。
 */
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
