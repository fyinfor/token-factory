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
  costDiscountMultiplier,
  markupRateFromPercent,
} from '../../../../helpers';
import { formatPreciseUsdPrice } from './PrecisePriceText';
import {
  groupImagePerImageTiersByFamily,
  IMAGE_PER_IMAGE_FAMILY_TITLE_KEY,
} from '../constants/imagePerImageHintI18n';
import TierPriceMatrix from './TierPriceMatrix';

const { Text } = Typography;

function sortTierRowsByResolutionAsc(rows) {
  return [...rows].sort((a, b) =>
    compareVideoResolutionAsc(a.resolution, b.resolution),
  );
}

function formatTierPrice(usd, usedGroupRatio, displayPrice) {
  const value = Number(usd || 0);
  if (!Number.isFinite(value) || value <= 0) return null;
  return displayPrice(value * usedGroupRatio);
}

function getDiscountPercent(currentUsd, officialUsd) {
  const current = Number(currentUsd || 0);
  const official = Number(officialUsd || 0);
  if (
    !Number.isFinite(current) ||
    !Number.isFinite(official) ||
    official <= 0
  ) {
    return null;
  }
  return official > current ? Math.round((1 - current / official) * 100) : 0;
}

function mapRowsToItems(
  rows,
  usedGroupRatio,
  displayPrice,
  unitLabel,
  isCostPrice,
  priceDiscountPercent,
  markupDiscountRate,
) {
  return rows.map((row, idx) => {
    const effectiveUsd = Number(row.usd_after_channel_discount || 0);
    const channelRawUsd = Number(row.usd_channel_raw);
    const officialUsd = Number(row.usd_official || 0);
    const costMultiplier = costDiscountMultiplier(priceDiscountPercent);
    const markupRate = markupRateFromPercent(markupDiscountRate);
    const fallbackCostUsd =
      officialUsd > 0
        ? Math.max(0, effectiveUsd - officialUsd * markupRate)
        : costMultiplier + markupRate > 0
          ? effectiveUsd * (costMultiplier / (costMultiplier + markupRate))
          : 0;
    const currentUsd = isCostPrice
      ? Number.isFinite(channelRawUsd)
        ? channelRawUsd * costMultiplier
        : fallbackCostUsd
      : effectiveUsd;
    const platformUsd = currentUsd * usedGroupRatio;
    const costPercent = Number(priceDiscountPercent);
    const discount = isCostPrice
      ? Number.isFinite(costPercent) && costPercent >= 0 && costPercent < 100
        ? 100 - costPercent
        : 0
      : getDiscountPercent(platformUsd, officialUsd);
    return {
      key: `img-${idx}-${row.lane}-${row.resolution}`,
      resolution:
        formatVideoResolutionDisplayLabel(row.resolution) ||
        row.resolution ||
        '—',
      price: formatTierPrice(currentUsd, usedGroupRatio, displayPrice),
      priceExact: currentUsd > 0 ? formatPreciseUsdPrice(platformUsd) : null,
      official: formatTierPrice(officialUsd, 1, displayPrice),
      officialExact:
        Number(row.usd_official) > 0
          ? formatPreciseUsdPrice(row.usd_official)
          : null,
      discount,
      hasDiscount: discount > 0,
    };
  });
}

function ImagePerImageHintTable({
  hint,
  usedGroupRatio = 1,
  displayPrice,
  t,
  blurPricing = false,
  isCostPrice = false,
  priceDiscountPercent = 100,
  markupDiscountRate = 0,
}) {
  const groups = useMemo(
    () => groupImagePerImageTiersByFamily(hint?.tiers),
    [hint?.tiers],
  );

  if (!hint || groups.length === 0) return null;

  const unitLabel = t('张');
  const columns = [
    { key: 'resolution', label: t('分辨率') },
    {
      key: 'price',
      label: isCostPrice ? t('成本价') : t('平台价'),
      unitLabel,
      strong: true,
    },
    { key: 'official', label: t('官方价'), unitLabel },
    {
      key: 'discount',
      label: isCostPrice ? t('成本折扣') : t('折扣'),
      align: 'right',
    },
  ];

  return (
    <div className='mt-1 pt-2 border-t border-semi-color-border flex flex-col gap-2'>
      <Text strong size='small'>
        {t('图片价格')}
      </Text>
      <div
        style={
          blurPricing
            ? {
                filter: 'blur(8px)',
                userSelect: 'none',
                pointerEvents: 'none',
              }
            : undefined
        }
        className='flex flex-col gap-2'
      >
        {groups.map(({ family, rows }) => {
          const titleKey = IMAGE_PER_IMAGE_FAMILY_TITLE_KEY[family] || '文生图';
          const items = mapRowsToItems(
            sortTierRowsByResolutionAsc(rows),
            usedGroupRatio,
            displayPrice,
            unitLabel,
            isCostPrice,
            priceDiscountPercent,
            markupDiscountRate,
          );

          return (
            <TierPriceMatrix
              key={family}
              title={t(titleKey)}
              count={items.length}
              columns={columns}
              rows={items}
              gridType='image'
              accent={family === 'image_to_image' ? 'green' : 'blue'}
              variant='glass'
              t={t}
            />
          );
        })}
      </div>
    </div>
  );
}

export default ImagePerImageHintTable;
