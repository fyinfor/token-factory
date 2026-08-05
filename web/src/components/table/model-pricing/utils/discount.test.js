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

import { describe, expect, test } from 'bun:test';
import {
  calculatePriceDiscountPercent,
  getBestPriceDiscountPercent,
} from './discount';

describe('model pricing discounts', () => {
  test('calculates each GLM-5 cost item against its official price', () => {
    expect(calculatePriceDiscountPercent(2.172852, 2.728)).toBe(20);
    expect(calculatePriceDiscountPercent(3.240864, 18.0048)).toBe(82);
    expect(calculatePriceDiscountPercent(0.7604982, 0.9548)).toBe(20);
  });

  test('uses the lowest price ratio for the channel badge', () => {
    const discounts = [
      calculatePriceDiscountPercent(2.309252, 2.728),
      calculatePriceDiscountPercent(4.141104, 18.0048),
      calculatePriceDiscountPercent(0.8082382, 0.9548),
    ];

    expect(discounts).toEqual([15, 77, 15]);
    expect(getBestPriceDiscountPercent(discounts)).toBe(77);
  });

  test('ignores invalid or non-discounted values', () => {
    expect(calculatePriceDiscountPercent(10, 0)).toBeNull();
    expect(calculatePriceDiscountPercent(12, 10)).toBe(0);
    expect(getBestPriceDiscountPercent([null, 0, Number.NaN])).toBeNull();
  });
});
