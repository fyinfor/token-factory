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

import React from 'react';
import { Tooltip } from '@douyinfe/semi-ui';
import { getBillingCurrencyConfig } from '../../../../helpers/billingFormula';
import {
  formatModelPriceNumber,
  MODEL_PRICE_MAX_DECIMALS,
} from '../utils/priceDisplay';

function trimNumberText(value) {
  return formatModelPriceNumber(value);
}

export function getDisplayCurrencyConfig() {
  return getBillingCurrencyConfig();
}

export function toDisplayCurrencyValue(usdAmount, { tokenUnit = 'M' } = {}) {
  const { rate } = getDisplayCurrencyConfig();
  const unitDivisor = tokenUnit === 'K' ? 1000 : 1;
  const value = ((Number(usdAmount) || 0) * rate) / unitDivisor;
  return Number.isFinite(value) ? value : 0;
}

export function formatCurrencyAmount(
  value,
  { precision = MODEL_PRICE_MAX_DECIMALS } = {},
) {
  const { symbol } = getDisplayCurrencyConfig();
  return `${symbol}${formatModelPriceNumber(value, precision)}`;
}

export function formatPreciseCurrencyValue(value) {
  const { symbol } = getDisplayCurrencyConfig();
  return `${symbol}${trimNumberText(value)}`;
}

export function formatPreciseUsdPrice(usdAmount, { tokenUnit = 'M' } = {}) {
  return formatPreciseCurrencyValue(
    toDisplayCurrencyValue(usdAmount, { tokenUnit }),
  );
}

function normalizeContent(content) {
  if (content == null || content === '') return null;
  return String(content);
}

function hasExactValue(exact) {
  const e = normalizeContent(exact);
  return Boolean(e);
}

function PrecisePriceText({
  children,
  display,
  exact,
  className = '',
  style,
  tooltipProps,
}) {
  const content = children ?? display;
  if (!hasExactValue(exact)) {
    return (
      <span className={className} style={style}>
        {content}
      </span>
    );
  }

  return (
    <Tooltip content={exact} {...tooltipProps}>
      <span
        className={`cursor-help border-b border-dotted border-gray-400 ${className}`}
        style={style}
      >
        {content}
      </span>
    </Tooltip>
  );
}

export default PrecisePriceText;
