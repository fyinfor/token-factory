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

const formatDiscountNumber = (value) => {
  const rounded = Math.round((value + Number.EPSILON) * 100) / 100;
  return String(rounded);
};

/** 中文显示成交折数，其他语言显示减免百分比。 */
export const formatPriceRatioFromDiscount = (discountPercent, t) => {
  const discount = Number(discountPercent);
  if (!Number.isFinite(discount) || discount <= 0) {
    return '-';
  }
  const normalizedDiscount = Math.min(100, discount);
  const ratio = (100 - normalizedDiscount) / 10;
  const values = {
    ratio: formatDiscountNumber(ratio),
    discount: formatDiscountNumber(normalizedDiscount),
  };
  return typeof t === 'function'
    ? t('{{ratio}}折', values)
    : `${values.discount}% off`;
};
