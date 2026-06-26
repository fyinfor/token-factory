/*
Copyright (C) 2025 QuantumNous
*/

import React, { memo } from 'react';
import { Card, Empty, Spin, Tag } from '@douyinfe/semi-ui';
import { TrendingDown, TrendingUp } from 'lucide-react';
import { formatRankingGrowth, getRankingGrowthColor } from '../../helpers/rankings';

const MoverList = ({ title, icon, items, emptyText, positive }) => (
  <div>
    <div className='flex items-center gap-2 mb-3 text-sm font-semibold'>
      {icon}
      <span>{title}</span>
    </div>
    {items.length === 0 ? (
      <Empty description={emptyText} image={<div />} />
    ) : (
      <div className='flex flex-col gap-2'>
        {items.map((item) => (
          <div
            key={item.model_name}
            className='flex items-center justify-between gap-2 rounded-lg px-3 py-2'
            style={{ backgroundColor: 'var(--semi-color-fill-0)' }}
          >
            <div className='min-w-0'>
              <div className='text-sm font-medium truncate'>{item.model_name}</div>
              <div className='text-xs truncate' style={{ color: 'var(--semi-color-text-2)' }}>
                {item.vendor}
              </div>
            </div>
            <div className='shrink-0 text-right'>
              <Tag
                size='small'
                style={{
                  color: getRankingGrowthColor(positive ? item.growth_pct : -Math.abs(item.growth_pct || 0)),
                  backgroundColor: 'transparent',
                }}
              >
                {positive ? `↑${item.rank_delta || 0}` : `↓${Math.abs(item.rank_delta || 0)}`}
              </Tag>
              <div
                className='text-[10px] font-mono mt-0.5'
                style={{ color: getRankingGrowthColor(item.growth_pct) }}
              >
                {formatRankingGrowth(item.growth_pct)}
              </div>
            </div>
          </div>
        ))}
      </div>
    )}
  </div>
);

const PulseSection = memo(({ topMovers = [], topDroppers = [], t, loading }) => (
  <Card
    className='!rounded-2xl shadow-sm'
    title={t('排名变化')}
  >
    <Spin spinning={loading}>
      <div className='grid md:grid-cols-2 gap-6'>
        <MoverList
          title={t('排名上升')}
          icon={<TrendingUp size={16} className='text-emerald-500' />}
          items={topMovers}
          emptyText={t('暂无上升模型')}
          positive
        />
        <MoverList
          title={t('排名下降')}
          icon={<TrendingDown size={16} className='text-red-500' />}
          items={topDroppers}
          emptyText={t('暂无下降模型')}
          positive={false}
        />
      </div>
    </Spin>
  </Card>
));

PulseSection.displayName = 'PulseSection';

export default PulseSection;
