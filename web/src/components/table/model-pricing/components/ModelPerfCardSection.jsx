/*
Copyright (C) 2025 QuantumNous
*/

import React, { memo } from 'react';
import ModelPerfHourlyChart from './ModelPerfHourlyChart';
import ModelPerfMetricsGrid from './ModelPerfMetricsGrid';

const ModelPerfCardSection = memo(({ perf, t }) => {
  if (!perf) return null;

  const series = perf.hourly_series || [];
  const hasSeries = series.some((p) => (p.request_count || 0) > 0);
  if (!hasSeries && !perf.success_rate) return null;

  return (
    <div
      className='mt-3 rounded-xl px-3 py-2.5'
      style={{
        backgroundColor: 'var(--semi-color-fill-0)',
        border: '1px solid var(--semi-color-border)',
      }}
      onClick={(e) => e.stopPropagation()}
    >
      <div className='flex items-baseline justify-between gap-2 mb-2'>
        <span
          className='text-xs font-semibold'
          style={{ color: 'var(--semi-color-text-0)' }}
        >
          {t('运行性能')}
        </span>
        <span
          className='text-[10px] shrink-0'
          style={{ color: 'var(--semi-color-text-2)' }}
        >
          {t('近24小时真实请求统计')}
        </span>
      </div>

      {hasSeries && (
        <ModelPerfHourlyChart series={series} t={t} compact />
      )}

      <ModelPerfMetricsGrid perf={perf} t={t} compact />
    </div>
  );
});

ModelPerfCardSection.displayName = 'ModelPerfCardSection';

export default ModelPerfCardSection;
