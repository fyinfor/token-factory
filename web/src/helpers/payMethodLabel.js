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

/** 支付方式 type / 历史订单 payment_method → i18n 中文 key */
const PAYMENT_METHOD_TYPE_LABEL_KEYS = {
  alipay: '支付宝',
  wxpay: '微信',
  stripe: 'Stripe',
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
