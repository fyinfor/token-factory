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

const USER_AVATAR_COLORS = [
  'amber',
  'blue',
  'cyan',
  'green',
  'grey',
  'indigo',
  'light-blue',
  'lime',
  'orange',
  'pink',
  'purple',
  'red',
  'teal',
  'violet',
  'yellow',
];

export function stringToColor(value) {
  const text = String(value || '');
  let sum = 0;
  for (let i = 0; i < text.length; i++) {
    sum += text.charCodeAt(i);
  }
  return USER_AVATAR_COLORS[sum % USER_AVATAR_COLORS.length];
}

function trimFixedDecimalDisplay(value, fractionDigits = 2) {
  const number = Number(value);
  if (!Number.isFinite(number)) return String(value ?? '');
  const factor = 10 ** fractionDigits;
  const scaled = number * factor;
  const epsilon = Number.EPSILON * Math.max(1, Math.abs(scaled));
  const truncated =
    (number >= 0 ? Math.floor(scaled + epsilon) : Math.ceil(scaled - epsilon)) /
    factor;
  return truncated.toFixed(fractionDigits).replace(/\.?0+$/, '');
}

function renderNumber(number) {
  if (number >= 1000000000) return (number / 1000000000).toFixed(1) + 'B';
  if (number >= 1000000) return (number / 1000000).toFixed(1) + 'M';
  if (number >= 10000) return (number / 1000).toFixed(1) + 'k';
  return number;
}

export function renderQuota(quota, digits = 2) {
  const quotaPerUnit = Number(localStorage.getItem('quota_per_unit'));
  const quotaDisplayType = localStorage.getItem('quota_display_type') || 'USD';
  if (quotaDisplayType === 'TOKENS') {
    return renderNumber(Number(quota || 0));
  }

  const resultUSD = Number(quota || 0) / quotaPerUnit;
  let symbol = '$';
  let value = resultUSD;
  let status = {};
  try {
    status = JSON.parse(localStorage.getItem('status') || '{}') || {};
  } catch {
    status = {};
  }

  if (quotaDisplayType === 'CNY') {
    symbol = '¥';
    value *= Number(status.usd_exchange_rate) || 1;
  } else if (quotaDisplayType === 'CUSTOM') {
    symbol = status.custom_currency_symbol || '¤';
    value *= Number(status.custom_currency_exchange_rate) || 1;
  }

  const fixedResult = Number(trimFixedDecimalDisplay(value, digits));
  const displayValue =
    fixedResult === 0 && quota > 0 && value > 0 ? 10 ** -digits : fixedResult;
  return symbol + trimFixedDecimalDisplay(displayValue, digits);
}
