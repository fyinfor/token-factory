/*
Copyright (C) 2025 QuantumNous
*/

import React, { memo } from 'react';
import { Tooltip } from '@douyinfe/semi-ui';
import {
  formatPerfLatency,
  formatPerfThroughput,
  formatPerfSuccessRate,
  formatPerfHourLabel,
  getHourlyBarColor,
} from '../../../../helpers/perfMetrics';

const TOOLTIP_STYLE = {
  title: 'rgba(255, 255, 255, 0.95)',
  label: 'rgba(255, 255, 255, 0.72)',
  value: 'rgba(255, 255, 255, 0.95)',
  valueWarm: '#fdba74',
  border: 'rgba(255, 255, 255, 0.18)',
};

const buildHourlyTooltip = (point, t) => (
  <div className='text-xs leading-[1.6] min-w-[148px]'>
    <div
      className='font-medium mb-1.5 pb-1'
      style={{
        color: TOOLTIP_STYLE.title,
        borderBottom: `1px solid ${TOOLTIP_STYLE.border}`,
      }}
    >
      {formatPerfHourLabel(point.ts)}
    </div>
    <div className='flex justify-between gap-4'>
      <span style={{ color: TOOLTIP_STYLE.label }}>E2E</span>
      <span
        className='font-mono font-medium'
        style={{ color: TOOLTIP_STYLE.valueWarm }}
      >
        {formatPerfLatency(point.avg_latency_ms)}
      </span>
    </div>
    <div className='flex justify-between gap-4'>
      <span style={{ color: TOOLTIP_STYLE.label }}>TTFT</span>
      <span
        className='font-mono font-medium'
        style={{ color: TOOLTIP_STYLE.valueWarm }}
      >
        {formatPerfLatency(point.avg_ttft_ms)}
      </span>
    </div>
    <div className='flex justify-between gap-4'>
      <span style={{ color: TOOLTIP_STYLE.label }}>TPS</span>
      <span className='font-mono font-medium' style={{ color: TOOLTIP_STYLE.value }}>
        {formatPerfThroughput(point.avg_tps)}
      </span>
    </div>
    <div className='flex justify-between gap-4 mt-0.5'>
      <span style={{ color: TOOLTIP_STYLE.label }}>{t('在线率')}</span>
      <span
        className='font-mono font-medium'
        style={{ color: getHourlyBarColor(point) }}
      >
        {formatPerfSuccessRate(point.success_rate)}
      </span>
    </div>
  </div>
);

const buildEmptyHourlyTooltip = (point, t) => (
  <div className='text-xs leading-[1.6]' style={{ color: TOOLTIP_STYLE.label }}>
    <div style={{ color: TOOLTIP_STYLE.title }}>{formatPerfHourLabel(point.ts)}</div>
    <div className='mt-0.5'>{t('暂无数据')}</div>
  </div>
);

const ModelPerfHourlyChart = memo(({ series = [], t, compact = false }) => {
  if (!series.length) return null;

  const barHeight = compact ? 20 : 24;
  const gap = compact ? 2 : 3;

  return (
    <div
      className='rounded-lg px-2 py-2'
      style={{
        backgroundColor: 'var(--semi-color-fill-0)',
        border: '1px solid var(--semi-color-border)',
      }}
      onClick={(e) => e.stopPropagation()}
    >
      <div
        className='flex items-stretch w-full'
        style={{ height: barHeight, gap }}
      >
        {series.map((point) => {
          const hasData = (point.request_count || 0) > 0;
          const tooltip = hasData
            ? buildHourlyTooltip(point, t)
            : buildEmptyHourlyTooltip(point, t);

          return (
            <Tooltip key={point.ts} content={tooltip} position='top'>
              <span
                className='flex-1 min-w-0 rounded-[3px] cursor-default transition-opacity hover:opacity-80'
                style={{
                  height: '100%',
                  backgroundColor: hasData
                    ? getHourlyBarColor(point)
                    : 'var(--semi-color-fill-2)',
                  opacity: hasData ? 1 : 0.65,
                }}
              />
            </Tooltip>
          );
        })}
      </div>
    </div>
  );
});

ModelPerfHourlyChart.displayName = 'ModelPerfHourlyChart';

export default ModelPerfHourlyChart;
