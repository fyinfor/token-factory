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
import { Typography } from '@douyinfe/semi-ui';
import PrecisePriceText from './PrecisePriceText';

const { Text } = Typography;

const PRICE_GRID_COLUMNS = {
  image: '74px minmax(96px, 1fr) minmax(96px, 1fr) 52px',
  video: '70px 58px minmax(92px, 1fr) minmax(92px, 1fr) 52px',
  videoPerToken: 'minmax(108px, 1fr) minmax(92px, 1fr) minmax(92px, 1fr) 52px',
};

const discountStyle = (hasDiscount) => ({
  display: 'inline-flex',
  alignItems: 'center',
  justifyContent: 'center',
  minWidth: 42,
  height: 22,
  padding: '0 7px',
  borderRadius: 999,
  fontSize: 11,
  fontWeight: 600,
  color: hasDiscount ? '#E74C3C' : 'var(--semi-color-text-2)',
  backgroundColor: hasDiscount
    ? 'rgba(231, 76, 60, 0.11)'
    : 'rgba(142, 142, 147, 0.12)',
});

const priceTextStyle = {
  color: 'var(--semi-color-primary)',
  fontWeight: 700,
};

function TierPriceMatrix({
  title,
  count,
  countLabel,
  columns,
  rows,
  gridType = 'image',
  accent = 'blue',
  variant = 'default',
  t,
  zeroDiscountLabel = '0%',
}) {
  const gridTemplateColumns =
    typeof gridType === 'string'
      ? PRICE_GRID_COLUMNS[gridType] || PRICE_GRID_COLUMNS.image
      : gridType;
  const accentColor =
    accent === 'amber'
      ? 'rgb(255, 149, 0)'
      : accent === 'green'
        ? 'rgb(52, 199, 89)'
        : 'rgb(0, 122, 255)';
  const accentBg =
    accent === 'amber'
      ? 'rgba(255, 149, 0, 0.12)'
      : accent === 'green'
        ? 'rgba(52, 199, 89, 0.12)'
        : 'rgba(0, 122, 255, 0.12)';

  const isGlassTable = variant === 'glass';

  return (
    <div
      className={
        isGlassTable
          ? 'tier-price-glass-table overflow-hidden rounded-lg'
          : 'overflow-hidden rounded-2xl'
      }
      style={
        isGlassTable
          ? undefined
          : {
              backgroundColor: 'var(--semi-color-bg-1)',
              border: '1px solid var(--semi-color-border)',
              boxShadow: '0 12px 26px rgba(15, 23, 42, 0.06)',
              backdropFilter: 'saturate(180%) blur(18px)',
            }
      }
    >
      <div
        className={
          isGlassTable
            ? 'tier-price-glass-title flex items-center justify-between gap-2 px-3 py-2'
            : 'flex items-center justify-between gap-2 px-3 py-2.5'
        }
        style={
          isGlassTable
            ? undefined
            : { backgroundColor: 'var(--semi-color-fill-0)' }
        }
      >
        <Text strong size='small' className='min-w-0 truncate'>
          {title}
        </Text>
        {count != null ? (
          <span
            className={
              isGlassTable
                ? 'tier-price-glass-count shrink-0 text-[11px] font-medium'
                : 'shrink-0 text-[11px] font-semibold'
            }
            style={
              isGlassTable
                ? undefined
                : {
                    color: accentColor,
                    backgroundColor: accentBg,
                    borderRadius: 999,
                    padding: '2px 8px',
                  }
            }
          >
            {count} {countLabel || t('档')}
          </span>
        ) : null}
      </div>
      <div
        className={
          isGlassTable
            ? 'tier-price-glass-header grid items-center gap-2 px-3 py-2 text-[11px] font-semibold'
            : 'mx-2 mb-1 mt-1 grid items-center gap-2 rounded-full px-2 py-1.5 text-[11px] font-semibold'
        }
        style={{
          gridTemplateColumns,
          ...(isGlassTable
            ? { color: 'var(--semi-color-text-2)' }
            : {
                backgroundColor: 'var(--semi-color-fill-0)',
                color: 'var(--semi-color-text-2)',
              }),
        }}
      >
        {columns.map((column) => (
          <span
            key={column.key}
            className={column.align === 'right' ? 'text-right' : ''}
          >
            {column.label}
            {column.unitLabel ? (
              <span className='ml-0.5 font-normal'>/{column.unitLabel}</span>
            ) : null}
          </span>
        ))}
      </div>
      {rows.map((row, idx) => (
        <div
          key={row.key}
          className={
            isGlassTable
              ? 'tier-price-glass-row grid items-center gap-2 px-3 py-2.5 text-xs'
              : 'mx-2 grid items-center gap-2 px-2 py-2.5 text-xs'
          }
          style={{
            gridTemplateColumns,
            ...(isGlassTable || idx === 0
              ? undefined
              : { borderTop: '1px solid var(--semi-color-border)' }),
          }}
        >
          {columns.map((column) => {
            const value = row[column.key];
            if (column.key === 'discount') {
              return (
                <span key={column.key} className='flex justify-end'>
                  {row.discount != null ? (
                    <span style={discountStyle(row.hasDiscount)}>
                      {row.hasDiscount
                        ? `-${row.discount}%`
                        : zeroDiscountLabel}
                    </span>
                  ) : (
                    <span className='text-gray-400'>—</span>
                  )}
                </span>
              );
            }
            if (column.key === 'official') {
              return (
                <span
                  key={column.key}
                  className={`truncate ${row.hasDiscount ? 'line-through' : ''}`}
                  style={{
                    color: row.hasDiscount
                      ? 'var(--semi-color-text-2)'
                      : 'var(--semi-color-text-1)',
                  }}
                >
                  {value ? (
                    <PrecisePriceText exact={row[`${column.key}Exact`]}>
                      {value}
                    </PrecisePriceText>
                  ) : (
                    '—'
                  )}
                </span>
              );
            }
            return (
              <span
                key={column.key}
                className={`${column.strong ? 'font-semibold' : 'text-semi-color-text-1'} truncate`}
                style={column.strong ? priceTextStyle : undefined}
                title={typeof value === 'string' ? value : undefined}
              >
                {value ? (
                  <PrecisePriceText exact={row[`${column.key}Exact`]}>
                    {value}
                  </PrecisePriceText>
                ) : (
                  '—'
                )}
              </span>
            );
          })}
        </div>
      ))}
    </div>
  );
}

export default TierPriceMatrix;
