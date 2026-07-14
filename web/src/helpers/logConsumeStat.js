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

/**
 * /console/log 消费统计 — 金额计算工具
 *
 * 进位规则（全页统一）：
 * 1. 金额 / 费用计算精度最多保留小数点后 6 位（进一法）；
 * 2. 第 7 位及以后非零时，采用「进一法」向上取到 6 位小数（正数 Math.ceil，负数向绝对值增大）；
 * 3. 恰好落在 6 位小数上（无更多有效位）时保持原值，不额外进位；
 * 4. 展示时去掉末尾多余的 0（如 13.640000 → 13.64），不强制补零；
 * 5. 文本 / 图片 / 视频三类计费均纳入总消耗；各模型分项合计完成后，对该分项再做一次 6 位进一，
 *    总消耗为各计费类型进一后的求和，并对总和再固定 6 位。
 *
 * 注意：本模块刻意不依赖 render.jsx，避免与 helpers 入口形成循环引用。
 */

/** 日志页消费金额固定小数位 */
export const LOG_CONSUME_AMOUNT_DIGITS = 6;

/** 计费类型常量 */
export const BILLING_TYPE_TEXT = 'text';
export const BILLING_TYPE_IMAGE = 'image';
export const BILLING_TYPE_VIDEO = 'video';

const VIDEO_BILLING_MODES = new Set([
  'video_per_second',
  'video_token_output',
  'video_per_video',
  'video_token',
]);

function readQuotaPerUnit() {
  const raw = parseFloat(localStorage.getItem('quota_per_unit') || '1');
  return Number.isFinite(raw) && raw > 0 ? raw : 1;
}

function readCurrencyConfig() {
  const type = localStorage.getItem('quota_display_type') || 'USD';
  let symbol = '$';
  let rate = 1;
  if (type === 'CNY') {
    symbol = '¥';
    try {
      const statusStr = localStorage.getItem('status');
      if (statusStr) {
        const s = JSON.parse(statusStr);
        rate = s?.usd_exchange_rate || 1;
      }
    } catch (_) {
      /* ignore */
    }
  } else if (type === 'CUSTOM') {
    try {
      const statusStr = localStorage.getItem('status');
      if (statusStr) {
        const s = JSON.parse(statusStr);
        symbol = s?.custom_currency_symbol || '¤';
        rate = s?.custom_currency_exchange_rate || 1;
      }
    } catch (_) {
      symbol = '¤';
    }
  } else if (type === 'TOKENS') {
    symbol = '';
    rate = 1;
  }
  return { type, symbol, rate };
}

/**
 * 进一法保留指定位小数。
 * 例：1.2345671 → 1.234568；1.2345670 → 1.234567；-1.2345671 → -1.234568
 *
 * @param {number|string} value
 * @param {number} [digits=6]
 * @returns {number}
 */
export function ceilToFixedDecimals(value, digits = LOG_CONSUME_AMOUNT_DIGITS) {
  const n = Number(value);
  if (!Number.isFinite(n) || n === 0) {
    return 0;
  }
  const d = Number.isInteger(digits) && digits >= 0 ? digits : LOG_CONSUME_AMOUNT_DIGITS;
  const factor = 10 ** d;
  // 减去极小量，避免浮点误差被误判为需要再进一
  const eps = 1e-10;
  if (n > 0) {
    return Math.ceil(n * factor - eps) / factor;
  }
  return Math.floor(n * factor + eps) / factor;
}

/**
 * 将进一后的数值格式化为展示字符串：最多 digits 位，去掉末尾多余 0（如 13.640000 → 13.64）。
 *
 * @param {number|string} value
 * @param {number} [digits=6]
 * @returns {string}
 */
export function formatCeilFixedDecimals(value, digits = LOG_CONSUME_AMOUNT_DIGITS) {
  const fixed = ceilToFixedDecimals(value, digits);
  let s = fixed.toFixed(digits);
  if (s.includes('.')) {
    s = s.replace(/\.?0+$/, '');
  }
  return s || '0';
}

/**
 * 根据日志 other 字段判定计费大类：text / image / video。
 *
 * @param {object|null|undefined} other
 * @returns {'text'|'image'|'video'}
 */
export function classifyLogBillingType(other) {
  if (!other || typeof other !== 'object') {
    return BILLING_TYPE_TEXT;
  }
  const mode = String(other.billing_mode || '');
  if (mode === 'image_per_image') {
    return BILLING_TYPE_IMAGE;
  }
  if (VIDEO_BILLING_MODES.has(mode)) {
    return BILLING_TYPE_VIDEO;
  }
  if (other.video_billed_quota != null || other.video_quota_per_unit != null) {
    return BILLING_TYPE_VIDEO;
  }
  const path = String(other.request_path || '');
  if (path.includes('/videos')) {
    return BILLING_TYPE_VIDEO;
  }
  return BILLING_TYPE_TEXT;
}

/**
 * 内部额度 → 当前展示币种下的原始金额（未进位）。
 *
 * @param {number} quota
 * @param {{ quotaPerUnit?: number, rate?: number }=} opts
 * @returns {number}
 */
export function quotaToRawDisplayAmount(quota, opts = {}) {
  const q = Number(quota || 0);
  if (!Number.isFinite(q)) {
    return 0;
  }
  const { type, rate } = readCurrencyConfig();
  if (type === 'TOKENS') {
    return q;
  }
  const unit =
    Number(opts.quotaPerUnit) > 0
      ? Number(opts.quotaPerUnit)
      : readQuotaPerUnit();
  const exchange = Number(opts.rate) > 0 ? Number(opts.rate) : rate || 1;
  return (q / unit) * exchange;
}

/**
 * 单笔消费额度 → 进一法 6 位展示金额。
 *
 * @param {number} quota
 * @param {object=} options
 * @returns {number}
 */
export function quotaToCeilDisplayAmount(
  quota,
  options = { digits: LOG_CONSUME_AMOUNT_DIGITS },
) {
  const digits = options?.digits ?? LOG_CONSUME_AMOUNT_DIGITS;
  const raw = quotaToRawDisplayAmount(quota, options);
  const ceiled = ceilToFixedDecimals(raw, digits);
  if (ceiled === 0 && Number(quota) > 0 && raw > 0) {
    return 10 ** -digits;
  }
  return ceiled;
}

/**
 * 对一组分项金额求和后，再统一进一到 6 位。
 *
 * @param {Array<number>} amounts
 * @param {number} [digits=6]
 * @returns {number}
 */
export function sumCeilDisplayAmounts(amounts, digits = LOG_CONSUME_AMOUNT_DIGITS) {
  if (!Array.isArray(amounts) || amounts.length === 0) {
    return 0;
  }
  let sum = 0;
  for (const a of amounts) {
    sum += Number(a) || 0;
  }
  return ceilToFixedDecimals(sum, digits);
}

/**
 * 聚合消费日志：含文本 / 图片 / 视频总消耗，以及各模型分项。
 * 每行先按 6 位进一；模型分项合计后再进一；最终总消耗再进一收口。
 *
 * @param {Array<object>} logs
 * @param {(row: object) => object|null} [parseOther]
 * @returns {{ total: number, byType: object, byModel: Array<object>, digits: number }}
 */
export function aggregateLogConsumeStats(logs, parseOther) {
  const digits = LOG_CONSUME_AMOUNT_DIGITS;
  const byTypeRaw = {
    [BILLING_TYPE_TEXT]: 0,
    [BILLING_TYPE_IMAGE]: 0,
    [BILLING_TYPE_VIDEO]: 0,
  };
  const modelMap = new Map();

  const resolveOther = (row) => {
    if (typeof parseOther === 'function') {
      return parseOther(row) || {};
    }
    if (row?.other && typeof row.other === 'object') {
      return row.other;
    }
    return {};
  };

  for (const row of logs || []) {
    if (row?.type != null && Number(row.type) !== 2) {
      continue;
    }
    const quota = Number(row.quota || 0);
    const other = resolveOther(row);
    const billingType = classifyLogBillingType(other);
    const lineAmount = quotaToCeilDisplayAmount(quota, {
      digits,
      quotaPerUnit: other?.video_quota_per_unit,
    });
    byTypeRaw[billingType] = (byTypeRaw[billingType] || 0) + lineAmount;

    const modelName = String(row.model_name || row.modelName || '').trim() || '-';
    let bucket = modelMap.get(modelName);
    if (!bucket) {
      bucket = {
        modelName,
        quota: 0,
        text: 0,
        image: 0,
        video: 0,
      };
      modelMap.set(modelName, bucket);
    }
    bucket.quota += Number.isFinite(quota) ? quota : 0;
    bucket[billingType] += lineAmount;
  }

  const byType = {
    text: ceilToFixedDecimals(byTypeRaw.text, digits),
    image: ceilToFixedDecimals(byTypeRaw.image, digits),
    video: ceilToFixedDecimals(byTypeRaw.video, digits),
  };

  const byModel = [];
  for (const bucket of modelMap.values()) {
    const amount = sumCeilDisplayAmounts(
      [bucket.text, bucket.image, bucket.video],
      digits,
    );
    byModel.push({
      modelName: bucket.modelName,
      amount,
      quota: bucket.quota,
      text: ceilToFixedDecimals(bucket.text, digits),
      image: ceilToFixedDecimals(bucket.image, digits),
      video: ceilToFixedDecimals(bucket.video, digits),
    });
  }
  byModel.sort((a, b) => b.amount - a.amount);

  const total = sumCeilDisplayAmounts(
    [byType.text, byType.image, byType.video],
    digits,
  );

  return { total, byType, byModel, digits };
}

/**
 * 带货币符号的展示字符串（固定 6 位，进一后输出）。
 *
 * @param {number} amount
 * @param {number} [digits=6]
 * @returns {string}
 */
export function formatLogConsumeDisplayAmount(
  amount,
  digits = LOG_CONSUME_AMOUNT_DIGITS,
) {
  const { symbol, type } = readCurrencyConfig();
  const ceiled = ceilToFixedDecimals(amount, digits);
  if (type === 'TOKENS') {
    return formatCeilFixedDecimals(ceiled, Math.min(digits, 2));
  }
  return `${symbol}${formatCeilFixedDecimals(ceiled, digits)}`;
}
