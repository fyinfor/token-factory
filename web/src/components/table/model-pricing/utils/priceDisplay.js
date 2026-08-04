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

export const MODEL_PRICE_MAX_DECIMALS = 6;
export const MODEL_CARD_PRICE_MAX_DECIMALS = 2;

const normalizePrecision = (precision) => {
  const parsed = Number(precision);
  if (!Number.isFinite(parsed)) return MODEL_PRICE_MAX_DECIMALS;
  return Math.max(0, Math.min(MODEL_PRICE_MAX_DECIMALS, Math.trunc(parsed)));
};

export const truncateModelPriceValue = (
  value,
  precision = MODEL_PRICE_MAX_DECIMALS,
) => {
  const numeric = Number(value);
  if (!Number.isFinite(numeric)) return 0;
  const digits = normalizePrecision(precision);
  const factor = 10 ** digits;
  const truncated = Math.trunc(numeric * factor) / factor;
  return Object.is(truncated, -0) ? 0 : truncated;
};

export const formatModelPriceNumber = (
  value,
  precision = MODEL_PRICE_MAX_DECIMALS,
) => {
  const digits = normalizePrecision(precision);
  const fixed = truncateModelPriceValue(value, digits).toFixed(digits);
  return fixed.includes('.')
    ? fixed.replace(/0+$/, '').replace(/\.$/, '')
    : fixed;
};
