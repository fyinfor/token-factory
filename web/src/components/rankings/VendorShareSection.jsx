/*
Copyright (C) 2025 QuantumNous
*/

import React, { memo } from 'react';
import { Card, Empty, Progress, Spin } from '@douyinfe/semi-ui';
import { PieChart } from 'lucide-react';
import { formatRankingShare, formatRankingTokens } from '../../helpers/rankings';
import { getLobeHubIcon } from '../../helpers';

const VendorShareSection = memo(({ vendors = [], t, loading }) => (
  <Card
    className='!rounded-2xl shadow-sm h-full'
    title={
      <div className='flex items-center gap-2'>
        <PieChart size={16} />
        <span>{t('供应商份额')}</span>
      </div>
    }
  >
    <Spin spinning={loading}>
      {vendors.length === 0 ? (
        <Empty description={t('暂无排行数据')} />
      ) : (
        <div className='flex flex-col gap-3'>
          {vendors.map((item) => (
            <div key={item.vendor}>
              <div className='flex items-center justify-between gap-2 mb-1.5'>
                <div className='flex items-center gap-2 min-w-0'>
                  <span className='text-xs font-mono text-gray-400 w-5'>#{item.rank}</span>
                  <span className='w-6 h-6 flex items-center justify-center rounded-md bg-white border border-gray-100 shrink-0'>
                    {item.vendor_icon ? getLobeHubIcon(item.vendor_icon, 18) : null}
                  </span>
                  <span className='text-sm font-medium truncate'>{item.vendor}</span>
                </div>
                <span className='text-xs font-mono shrink-0' style={{ color: 'var(--semi-color-text-2)' }}>
                  {formatRankingShare(item.share)}
                </span>
              </div>
              <Progress
                percent={Math.min(100, Math.max(0, Number(item.share) || 0))}
                showInfo={false}
                stroke='var(--semi-color-primary)'
                size='small'
              />
              <div className='flex justify-between text-[10px] mt-1' style={{ color: 'var(--semi-color-text-2)' }}>
                <span>{formatRankingTokens(item.total_tokens)} tokens</span>
                <span>{t('代表模型')}: {item.top_model || '—'}</span>
              </div>
            </div>
          ))}
        </div>
      )}
    </Spin>
  </Card>
));

VendorShareSection.displayName = 'VendorShareSection';

export default VendorShareSection;
