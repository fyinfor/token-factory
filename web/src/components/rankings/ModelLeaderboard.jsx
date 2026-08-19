/*
Copyright (C) 2025 QuantumNous
*/

import React, { memo } from 'react';
import { Card, Empty, Spin, Tag } from '@douyinfe/semi-ui';
import { Trophy } from 'lucide-react';
import {
  formatRankingCallVolume,
  formatRankingGrowth,
  formatRankingShare,
  getRankingGrowthColor,
} from '../../helpers/rankings';
import { getLobeHubIcon } from '../../helpers';
import { getVendorLocalizedName } from '../../helpers/modelPricing';
import CategoryBadge from './CategoryBadge';

const RankBadge = ({ rank }) => {
  if (rank === 1) return <span className='text-amber-500 font-bold'>#{rank}</span>;
  if (rank === 2) return <span className='text-slate-400 font-bold'>#{rank}</span>;
  if (rank === 3) return <span className='text-orange-400 font-bold'>#{rank}</span>;
  return <span className='text-gray-400 font-mono text-xs'>#{rank}</span>;
};

const ModelLeaderboard = memo(({ models = [], t, loading, language }) => (
  <Card
    className='!rounded-2xl shadow-sm h-full'
    title={
      <div className='flex items-center gap-2'>
        <Trophy size={16} />
        <span>{t('热门模型')}</span>
      </div>
    }
  >
    <Spin spinning={loading}>
      {models.length === 0 ? (
        <Empty description={t('暂无排行数据')} />
      ) : (
        <div className='flex flex-col gap-2'>
          {models.map((item) => (
            <div
              key={item.model_name}
              className='flex items-center gap-3 rounded-xl px-3 py-2.5'
              style={{ backgroundColor: 'var(--semi-color-fill-0)' }}
            >
              <div className='w-8 shrink-0 text-center'>
                <RankBadge rank={item.rank} />
              </div>
              <div className='w-8 h-8 shrink-0 flex items-center justify-center rounded-lg bg-white border border-gray-100'>
                {item.vendor_icon ? getLobeHubIcon(item.vendor_icon, 20) : null}
              </div>
              <div className='min-w-0 flex-1'>
                <div className='font-medium text-sm truncate flex items-center gap-1'>
                  <span className='truncate'>{item.model_name}</span>
                  <CategoryBadge category={item.category} />
                </div>
                <div className='text-xs truncate' style={{ color: 'var(--semi-color-text-2)' }}>
                  {item.vendor
                    ? getVendorLocalizedName(
                        { name: item.vendor, name_en: item.vendor_name_en },
                        language,
                      )
                    : t('未知供应商')}
                </div>
              </div>
              <div className='text-right shrink-0'>
                <div className='font-mono text-sm font-semibold'>
                  {formatRankingCallVolume(item.total_tokens)}
                </div>
                <div className='text-[10px]' style={{ color: 'var(--semi-color-text-2)' }}>
                  {formatRankingShare(item.share)}
                </div>
              </div>
              {Number.isFinite(item.growth_pct) && item.growth_pct !== 0 ? (
                <Tag
                  size='small'
                  style={{
                    color: getRankingGrowthColor(item.growth_pct),
                    backgroundColor: 'transparent',
                  }}
                >
                  {formatRankingGrowth(item.growth_pct)}
                </Tag>
              ) : null}
            </div>
          ))}
        </div>
      )}
    </Spin>
  </Card>
));

ModelLeaderboard.displayName = 'ModelLeaderboard';

export default ModelLeaderboard;
