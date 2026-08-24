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

import { renderQuota } from './render';

/** 支付方式 type / 历史订单 payment_method → i18n 中文 key */
const PAYMENT_METHOD_TYPE_LABEL_KEYS = {
  alipay: '支付宝',
  wxpay: '微信',
  stripe: 'Stripe',
  antom: 'Antom 收银台',
  waffo: 'Waffo (Global Payment)',
  paypal: 'PayPal',
  creem: 'Creem',
  ALI_PC: '支付宝',
  WX_NATIVE: '微信',
};

/** 后台默认或常见 name 字段 → i18n 中文 key */
const PAYMENT_METHOD_NAME_LABEL_KEYS = {
  支付宝: '支付宝',
  微信: '微信',
  自定义1: '自定义1',
  自定义2: '自定义2',
  自定义3: '自定义3',
  Stripe: 'Stripe',
  'Antom 收银台': 'Antom 收银台',
  'Waffo (Global Payment)': 'Waffo (Global Payment)',
  PayPal: 'PayPal',
  Creem: 'Creem',
  Card: 'Card',
  'Apple Pay': 'Apple Pay',
  'Google Pay': 'Google Pay',
};

/**
 * 解析充值支付方式展示名（支持 { type, name } 或历史订单 payment_method 字符串）。
 * @param {{ type?: string, name?: string } | string | null | undefined} payMethod
 * @param {(key: string) => string} t i18next 翻译函数
 * @returns {string}
 */
export function getPayMethodDisplayName(payMethod, t) {
  if (!payMethod) return '-';

  if (typeof payMethod === 'string') {
    const raw = payMethod.trim();
    if (!raw) return '-';
    const upper = raw.toUpperCase();
    if (upper.startsWith('PP_')) return t('PayPal');
    const typeKey =
      PAYMENT_METHOD_TYPE_LABEL_KEYS[raw] ||
      PAYMENT_METHOD_TYPE_LABEL_KEYS[raw.toLowerCase()];
    if (typeKey) return t(typeKey);
    const nameKey = PAYMENT_METHOD_NAME_LABEL_KEYS[raw];
    if (nameKey) return t(nameKey);
    return raw;
  }

  const type = String(payMethod.type || '').trim();
  const name = String(payMethod.name || '').trim();

  if (type) {
    const upper = type.toUpperCase();
    if (upper.startsWith('PP_')) return t('PayPal');
    const typeKey =
      PAYMENT_METHOD_TYPE_LABEL_KEYS[type] ||
      PAYMENT_METHOD_TYPE_LABEL_KEYS[type.toLowerCase()];
    if (typeKey) return t(typeKey);
  }

  if (name) {
    const nameKey = PAYMENT_METHOD_NAME_LABEL_KEYS[name];
    if (nameKey) return t(nameKey);
    return name;
  }

  return type || '-';
}

function normalizeTopupRate(usdExchangeRate) {
  const rate = Number(usdExchangeRate);
  return Number.isFinite(rate) && rate > 0 ? rate : 7.3;
}

function isLegacyConvertedUsdTopup(record, money) {
  const currency = String(record?.pay_currency || '').toUpperCase();
  const inputCurrency = String(record?.input_currency || '').toUpperCase();
  const paymentMethod = String(record?.payment_method || '').toLowerCase();
  const inputAmount = Number(record?.input_amount);
  if (
    currency !== 'USD' ||
    inputCurrency !== 'CNY' ||
    paymentMethod === 'stripe' ||
    !Number.isFinite(inputAmount) ||
    inputAmount <= 0
  ) {
    return false;
  }
  return Math.abs(money - inputAmount) <= 0.01;
}

/**
 * 格式化充值订单的实际支付金额。
 * 兼容旧 Yipay/Jeepay PayPal 订单：历史数据里 pay_currency=USD，但 money 仍存的是人民币金额。
 * @param {unknown} money 后端 TopUp.money
 * @param {object} record 后端 TopUp 行
 * @param {number} usdExchangeRate 美元兑人民币汇率
 * @returns {string}
 */
export function formatTopupPayMoney(money, record = {}, usdExchangeRate) {
  const numericMoney = Number(money);
  const safeMoney = Number.isFinite(numericMoney) ? numericMoney : 0;
  const rate = normalizeTopupRate(usdExchangeRate);
  const currency = String(record?.pay_currency || '').toUpperCase();

  if (String(record?.payment_method || '').toLowerCase() === 'ubcoin') {
    return `$${safeMoney.toFixed(2)}`;
  }

  if (currency === 'USD') {
    const displayMoney = isLegacyConvertedUsdTopup(record, safeMoney)
      ? safeMoney / rate
      : safeMoney;
    return `$${displayMoney.toFixed(2)} USD`;
  }
  if (currency === 'CNY') {
    return `¥${safeMoney.toFixed(2)}`;
  }

  if (String(record?.payment_method || '').toLowerCase() === 'stripe') {
    return `¥${(safeMoney * rate).toFixed(2)}`;
  }
  return `¥${safeMoney.toFixed(2)}`;
}

/**
 * 格式化充值账单「到账额度」展示。
 * 优先按实际入账 quota_to_add 展示（与钱包当前余额同一套 renderQuota 口径）；
 * 无额度字段时再回退到实付金额展示。
 * @param {unknown} money 后端 TopUp.money
 * @param {object} record 后端 TopUp 行
 * @param {number} usdExchangeRate 美元兑人民币汇率
 * @returns {string}
 */
export function formatTopupCreditedAmount(money, record = {}, usdExchangeRate) {
  const quotaToAdd = Number(record?.quota_to_add || 0);
  if (quotaToAdd > 0) {
    return renderQuota(quotaToAdd);
  }
  return formatTopupPayMoney(money, record, usdExchangeRate);
}

export function isHostedCheckoutPayMethod(type) {
  return type === 'stripe' || type === 'antom';
}

export function isPayMethodUnavailable(
  type,
  { enableOnlineTopUp, enableStripeTopUp, enableAntomTopUp },
) {
  if (type === 'stripe') return !enableStripeTopUp;
  if (type === 'antom') return !enableAntomTopUp;
  return !enableOnlineTopUp;
}
