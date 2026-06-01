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

function formatTierPrice(usd, usedGroupRatio, displayPrice, unitLabel) {
  const value = Number(usd || 0);
  if (!Number.isFinite(value) || value <= 0) return null;
  return `${displayPrice(value * usedGroupRatio)} / ${unitLabel}`;
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

function mapRowsToItems(rows, usedGroupRatio, displayPrice, unitLabel) {
  return rows.map((row, idx) => {
    const discount = getDiscountPercent(
      row.usd_after_channel_discount,
      row.usd_official,
    );
    return {
      key: `img-${idx}-${row.lane}-${row.resolution}`,
      resolution:
        formatVideoResolutionDisplayLabel(row.resolution) ||
        row.resolution ||
        '—',
      price: formatTierPrice(
        row.usd_after_channel_discount,
        usedGroupRatio,
        displayPrice,
        unitLabel,
      ),
      official: formatTierPrice(row.usd_official, 1, displayPrice, unitLabel),
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
}) {
  const groups = useMemo(
    () => groupImagePerImageTiersByFamily(hint?.tiers),
    [hint?.tiers],
  );

  if (!hint || groups.length === 0) return null;

  const unitLabel = t('张');
  const columns = [
    { key: 'resolution', label: t('分辨率') },
    { key: 'price', label: t('平台价'), strong: true },
    { key: 'official', label: t('官方价') },
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
              t={t}
              zeroDiscountLabel={isCostPrice ? t('0折扣') : '0%'}
            />
          );
        })}
      </div>
    </div>
  );
}

export default ImagePerImageHintTable;
