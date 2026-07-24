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

/** 将“优惠百分比”转换为成交价占官方价的百分比。 */
export const formatPriceRatioFromDiscount = (discountPercent) => {
  const discount = Number(discountPercent);
  if (!Number.isFinite(discount) || discount <= 0) {
    return '-';
  }
  const ratio = Math.max(0, 100 - discount);
  const rounded = Math.round((ratio + Number.EPSILON) * 100) / 100;
  return `${Number.isInteger(rounded) ? rounded.toFixed(0) : rounded.toFixed(2)}%`;
};
