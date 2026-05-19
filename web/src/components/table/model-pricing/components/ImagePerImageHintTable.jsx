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
import { Table, Typography } from '@douyinfe/semi-ui';
import {
  formatVideoResolutionDisplayLabel,
  compareVideoResolutionAsc,
} from '../../../../helpers';
import {
  groupImagePerImageTiersByFamily,
  IMAGE_PER_IMAGE_FAMILY_TITLE_KEY,
} from '../constants/imagePerImageHintI18n';

const { Text } = Typography;

function sortTierRowsByResolutionAsc(rows) {
  return [...rows].sort((a, b) =>
    compareVideoResolutionAsc(a.resolution, b.resolution),
  );
}

function mapRowsToDataSource(rows, usedGroupRatio, displayPrice, unitLabel) {
  return rows.map((row, idx) => {
    const usd = Number(row.usd_after_channel_discount || 0) * usedGroupRatio;
    return {
      key: `img-${idx}-${row.lane}-${row.resolution}`,
      resolution:
        formatVideoResolutionDisplayLabel(row.resolution) || row.resolution || '—',
      price: `${displayPrice(usd)} / ${unitLabel}`,
    };
  });
}

function ImagePerImageHintTable({
  hint,
  usedGroupRatio = 1,
  displayPrice,
  t,
  blurPricing = false,
}) {
  const groups = useMemo(
    () => groupImagePerImageTiersByFamily(hint?.tiers),
    [hint?.tiers],
  );

  if (!hint || groups.length === 0) return null;

  const unitLabel = t('张');
  const tableCols = [
    { title: t('分辨率'), dataIndex: 'resolution', key: 'resolution' },
    {
      title: t('价格'),
      dataIndex: 'price',
      key: 'price',
      render: (text) => (
        <span className='font-semibold text-black'>{text}</span>
      ),
    },
  ];

  return (
    <>
      <style>{`
        .image-per-image-tier-tb.semi-table-wrapper {
          margin-top: 0 !important;
          margin-bottom: 0 !important;
        }
        .image-per-image-tier-tb .semi-table-thead .semi-table-row .semi-table-row-cell {
          padding-top: 2px !important;
          padding-bottom: 2px !important;
          line-height: 1.2 !important;
        }
        .image-per-image-tier-tb .semi-table-tbody .semi-table-row .semi-table-row-cell {
          padding-top: 3px !important;
          padding-bottom: 3px !important;
        }
      `}</style>
      <div className='mt-1 pt-1 border-t border-gray-100 flex flex-col gap-1'>
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
            const titleKey =
              IMAGE_PER_IMAGE_FAMILY_TITLE_KEY[family] || '文生图';
            const dataSource = mapRowsToDataSource(
              sortTierRowsByResolutionAsc(rows),
              usedGroupRatio,
              displayPrice,
              unitLabel,
            );
            return (
              <div
                key={family}
                className='rounded-md border border-[var(--semi-color-border)] bg-[var(--semi-color-fill-0)] px-2 py-1'
              >
                <Text
                  strong
                  size='small'
                  className='block text-gray-800 !leading-tight mb-1.5'
                >
                  {t(titleKey)}
                </Text>
                <Table
                  className='image-per-image-tier-tb'
                  size='small'
                  pagination={false}
                  columns={tableCols}
                  dataSource={dataSource}
                />
              </div>
            );
          })}
        </div>
      </div>
    </>
  );
}

export default ImagePerImageHintTable;
