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
import { Table, Typography, Tag, Tooltip } from '@douyinfe/semi-ui';
import { IconHelpCircle } from '@douyinfe/semi-icons';
import { formatVideoResolutionDisplayLabel, compareVideoResolutionAsc } from '../../../../helpers';
import {
  VIDEO_FLAT_LANE_I18N_KEY,
  groupVideoFlatTiersByFamily,
  VIDEO_FLAT_FAMILY_TITLE_KEY,
} from '../constants/videoFlatClipLaneI18n';

const { Text } = Typography;

function AudioTrackColumnTitle({ t }) {
  const tip = t('音轨列说明');
  return (
    <span className='inline-flex items-center gap-0.5'>
      <span>{t('音轨')}</span>
      <Tooltip content={tip}>
        <IconHelpCircle
          className='cursor-help text-gray-400 hover:text-gray-600 align-middle'
          size='small'
          aria-label={tip}
        />
      </Tooltip>
    </span>
  );
}

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

function mapRowsToDataSource(rows, usedGroupRatio, displayPrice, unitLabel, t) {
  return rows.map((row, idx) => {
    const usd = Number(row.usd_after_channel_discount || 0) * usedGroupRatio;
    const laneKey = VIDEO_FLAT_LANE_I18N_KEY[row.lane];
    const audioKind =
      row.has_audio === true
        ? 'with'
        : row.has_audio === false
          ? 'without'
          : 'unified';
    return {
      key: `v-${idx}-${row.lane}-${row.resolution}`,
      laneLabel: laneKey ? t(laneKey) : row.lane || '—',
      resolution: formatVideoResolutionDisplayLabel(row.resolution) || '—',
      audioKind,
      price: `${displayPrice(usd)} / ${unitLabel}`,
    };
  });
}

/**
 * 分档视频（按条/按秒）：按文生/图生/视频生拆成独立小表；不展示总标题「视频按条价格表」。
 */
function VideoFlatClipHintTable({
  hint,
  usedGroupRatio = 1,
  displayPrice,
  t,
  blurPricing = false,
}) {
  const groups = useMemo(
    () => groupVideoFlatTiersByFamily(hint?.tiers),
    [hint?.tiers],
  );

  if (!hint || groups.length === 0) return null;

  const perSecond = hint.billing_mode === 'per_second';
  const unitLabel = perSecond ? t('秒') : t('条');

  const baseCols = [
    { title: t('分辨率'), dataIndex: 'resolution', key: 'resolution' },
    {
      title: <AudioTrackColumnTitle t={t} />,
      dataIndex: 'audioKind',
      key: 'audio',
      render: (kind) => {
        if (kind === 'with') {
          return (
            <Tag size='small' color='green' shape='circle'>
              {t('有音轨')}
            </Tag>
          );
        }
        if (kind === 'without') {
          return (
            <Tag size='small' color='orange' shape='circle'>
              {t('无音轨')}
            </Tag>
          );
        }
        return (
          <Tag size='small' color='grey' shape='circle'>
            {t('统一')}
          </Tag>
        );
      },
    },
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
        .video-flat-tier-tb.semi-table-wrapper {
          margin-top: 0 !important;
          margin-bottom: 0 !important;
        }
        .video-flat-tier-tb .semi-table-thead .semi-table-row .semi-table-row-cell {
          padding-top: 2px !important;
          padding-bottom: 2px !important;
          line-height: 1.2 !important;
        }
        .video-flat-tier-tb .semi-table-tbody .semi-table-row .semi-table-row-cell {
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
            const titleKey = VIDEO_FLAT_FAMILY_TITLE_KEY[family] || '其他';
            const sortedRows = sortTierRowsByResolutionAsc(rows);
            const dataSource = mapRowsToDataSource(
              sortedRows,
              usedGroupRatio,
              displayPrice,
              unitLabel,
              t,
            );
            const laneLabelSet = new Set(
              sortedRows.map((r) =>
                VIDEO_FLAT_LANE_I18N_KEY[r.lane]
                  ? t(VIDEO_FLAT_LANE_I18N_KEY[r.lane])
                  : String(r.lane || ''),
              ),
            );
            const showLaneCol = laneLabelSet.size > 1;
            const tableCols = showLaneCol
              ? [
                  {
                    title: t('子类型'),
                    dataIndex: 'laneLabel',
                    key: 'laneLabel',
                  },
                  ...baseCols,
                ]
              : baseCols;

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
                  className='video-flat-tier-tb'
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

export default VideoFlatClipHintTable;
