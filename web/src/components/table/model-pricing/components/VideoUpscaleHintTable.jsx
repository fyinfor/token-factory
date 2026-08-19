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

import React, { useMemo } from 'react';
import { Typography } from '@douyinfe/semi-ui';
import {
  formatVideoResolutionDisplayLabel,
  compareVideoResolutionAsc,
} from '../../../../helpers';
import PrecisePriceText, { formatPreciseUsdPrice } from './PrecisePriceText';
import { hasVideoUpscaleTierTable } from '../constants/videoFlatClipLaneI18n';

const { Text } = Typography;

const priceTextStyle = {
  color: 'var(--semi-color-primary)',
  fontWeight: 700,
};

function sortUpscaleTiers(tiers) {
  return [...tiers].sort((a, b) =>
    compareVideoResolutionAsc(a.resolution, b.resolution),
  );
}

function VideoUpscaleHintTable({
  hint,
  usedGroupRatio = 1,
  displayPrice,
  t,
  blurPricing = false,
}) {
  const columns = useMemo(() => {
    const tiers = sortUpscaleTiers(
      Array.isArray(hint?.upscale_tiers) ? hint.upscale_tiers : [],
    ).filter((row) => Number(row?.usd_after_channel_discount) > 0);
    return tiers.map((row) => {
      const usd = Number(row.usd_after_channel_discount) * usedGroupRatio;
      return {
        key: String(row.resolution || ''),
        label: formatVideoResolutionDisplayLabel(row.resolution) || '—',
        price: displayPrice(usd),
        priceExact: formatPreciseUsdPrice(usd),
      };
    });
  }, [hint?.upscale_tiers, usedGroupRatio, displayPrice]);

  if (!hasVideoUpscaleTierTable(hint) || columns.length === 0) {
    return null;
  }

  const gridTemplateColumns = `minmax(96px, max-content) repeat(${columns.length}, minmax(72px, 1fr))`;

  return (
    <div className='video-price-section mt-1 flex flex-col gap-2 pt-2'>
      <div className='flex items-center justify-between gap-2'>
        <Text strong size='small'>
          {t('超分价格')}
        </Text>
        <span className='tier-price-glass-count shrink-0 text-[11px] font-medium'>
          {columns.length} {t('档')}
        </span>
      </div>
      <div
        className='overflow-x-auto'
        style={
          blurPricing
            ? {
                filter: 'blur(8px)',
                userSelect: 'none',
                pointerEvents: 'none',
              }
            : undefined
        }
      >
        <div className='tier-price-glass-table overflow-hidden rounded-lg min-w-full'>
          <div
            className='tier-price-glass-header grid items-center gap-2 px-3 py-2 text-[11px] font-semibold'
            style={{
              gridTemplateColumns,
              color: 'var(--semi-color-text-2)',
            }}
          >
            <span>{t('分辨率')}</span>
            {columns.map((column) => (
              <span key={column.key} className='truncate' title={column.label}>
                {column.label}
              </span>
            ))}
          </div>
          <div
            className='tier-price-glass-row grid items-center gap-2 px-3 py-2.5 text-xs'
            style={{ gridTemplateColumns }}
          >
            <span className='text-semi-color-text-1 truncate whitespace-nowrap'>
              {t('超分/秒')}
            </span>
            {columns.map((column) => (
              <span
                key={column.key}
                className='font-semibold truncate'
                style={priceTextStyle}
                title={column.priceExact || undefined}
              >
                <PrecisePriceText exact={column.priceExact}>
                  {column.price}
                </PrecisePriceText>
              </span>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

export default VideoUpscaleHintTable;
