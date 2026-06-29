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
import { Typography, Tooltip } from '@douyinfe/semi-ui';
import { IconHelpCircle } from '@douyinfe/semi-icons';
import {
  formatVideoResolutionDisplayLabel,
  compareVideoResolutionAsc,
} from '../../../../helpers';
import { formatPreciseUsdPrice } from './PrecisePriceText';
import {
  VIDEO_FLAT_LANE_I18N_KEY,
  groupVideoFlatTiersByFamily,
  VIDEO_FLAT_FAMILY_TITLE_KEY,
} from '../constants/videoFlatClipLaneI18n';
import TierPriceMatrix from './TierPriceMatrix';

const { Text } = Typography;

function sortTierRowsByResolutionAsc(rows) {
  return [...rows].sort((a, b) => {
    let c = compareVideoResolutionAsc(a.resolution, b.resolution);
    if (c !== 0) return c;
    c = String(a.lane ?? '').localeCompare(String(b.lane ?? ''));
    if (c !== 0) return c;
    const ar = a.has_audio === true ? 1 : a.has_audio === false ? 0 : 2;
    const br = b.has_audio === true ? 1 : b.has_audio === false ? 0 : 2;
    return ar - br;
  });
}

function getAudioLabel(row, t) {
  if (row.has_audio === true) return t('有音轨');
  if (row.has_audio === false) return t('无音轨');
  return t('统一');
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

function mapRowsToItems(rows, usedGroupRatio, displayPrice, t) {
  return rows.map((row, idx) => {
    const laneKey = VIDEO_FLAT_LANE_I18N_KEY[row.lane];
    const currentUsd = Number(row.usd_after_channel_discount || 0);
    const platformUsd = currentUsd * usedGroupRatio;
    const discount = getDiscountPercent(platformUsd, row.usd_official);
    const resolutionLabel = formatVideoResolutionDisplayLabel(row.resolution) || '—';
    return {
      key: `v-${idx}-${row.lane}-${row.resolution}-${row.has_audio}`,
      lane: laneKey ? t(laneKey) : row.lane || '—',
      resolution: resolutionLabel,
      audio: getAudioLabel(row, t),
      price: formatTierPrice(currentUsd, usedGroupRatio, displayPrice),
      priceExact: currentUsd > 0 ? formatPreciseUsdPrice(platformUsd) : null,
      official: formatTierPrice(row.usd_official, 1, displayPrice),
      officialExact:
        Number(row.usd_official) > 0
          ? formatPreciseUsdPrice(row.usd_official)
          : null,
      discount,
      hasDiscount: discount > 0,
    };
  });
}

function VideoFlatClipHintTable({
  hint,
  usedGroupRatio = 1,
  displayPrice,
  t,
  blurPricing = false,
  isCostPrice = false,
}) {
  const groups = useMemo(
    () => groupVideoFlatTiersByFamily(hint?.tiers),
    [hint?.tiers],
  );

  if (!hint || groups.length === 0) return null;

  const billingMode = String(hint.billing_mode || '');
  const unitLabel =
    billingMode === 'per_second'
      ? t('秒')
      : billingMode === 'per_token'
        ? 'M token'
        : t('条');
  const columns =
    billingMode === 'per_token'
      ? [
          { key: 'resolution', label: t('分辨率') },
          { key: 'price', label: t('平台价'), unitLabel, strong: true },
          { key: 'official', label: t('官方价'), unitLabel },
          {
            key: 'discount',
            label: isCostPrice ? t('成本折扣') : t('折扣'),
            align: 'right',
          },
        ]
      : [
          { key: 'resolution', label: t('分辨率') },
          { key: 'audio', label: t('音轨') },
          { key: 'price', label: t('平台价'), unitLabel, strong: true },
          { key: 'official', label: t('官方价'), unitLabel },
          {
            key: 'discount',
            label: isCostPrice ? t('成本折扣') : t('折扣'),
            align: 'right',
          },
        ];

  return (
    <div className='mt-1 pt-2 border-t border-semi-color-border flex flex-col gap-2'>
      <div className='flex items-center justify-between gap-2'>
        <Text strong size='small'>
          {t('视频价格')}
        </Text>
        {billingMode !== 'per_token' ? (
          <Tooltip
            position='left'
            content={
              <div style={{ maxWidth: 220, whiteSpace: 'normal' }}>
                {t('音轨列说明')}
              </div>
            }
          >
            <span className='inline-flex items-center gap-1 text-xs text-gray-500 cursor-help'>
              <IconHelpCircle size='small' />
              {t('有无音轨价格')}
            </span>
          </Tooltip>
        ) : null}
      </div>
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
          const titleKey = VIDEO_FLAT_FAMILY_TITLE_KEY[family] || '其他';
          const items = mapRowsToItems(
            sortTierRowsByResolutionAsc(rows),
            usedGroupRatio,
            displayPrice,
            t,
          );

          return (
            <TierPriceMatrix
              key={family}
              title={t(titleKey)}
              count={items.length}
              columns={columns}
              rows={items}
              gridType={billingMode === 'per_token' ? 'videoPerToken' : 'video'}
              accent={family === 'image_to_video' ? 'amber' : 'blue'}
              t={t}
              zeroDiscountLabel={isCostPrice ? t('0折扣') : '0%'}
            />
          );
        })}
      </div>
    </div>
  );
}

export default VideoFlatClipHintTable;
