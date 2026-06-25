/*
Copyright (C) 2025 QuantumNous
*/

import React, { memo } from 'react';
import {
  formatPerfLatency,
  formatPerfThroughput,
  formatPerfSuccessRate,
  getSuccessRateColor,
} from '../../../../helpers/perfMetrics';
import PerfMetricLabel from './PerfMetricLabel';

const MetricItem = ({ label, value, valueColor, compact, showDivider }) => (
  <div
    className='flex flex-col items-center justify-center min-w-0 flex-1 text-center relative'
    style={
      showDivider
        ? { borderLeft: '1px solid var(--semi-color-border)' }
        : undefined
    }
  >
    <div
      className={`mb-1 truncate max-w-full px-1 ${compact ? 'text-[10px]' : 'text-xs'}`}
      style={{ color: 'var(--semi-color-text-2)' }}
    >
      {label}
    </div>
    <div
      className={`font-semibold font-mono tabular-nums truncate max-w-full px-1 ${
        compact ? 'text-[13px] leading-tight' : 'text-base'
      }`}
      style={{ color: valueColor || 'var(--semi-color-text-0)' }}
    >
      {value}
    </div>
  </div>
);

const METRICS = [
  {
    key: 'uptime',
    getLabel: (t) => t('在线率'),
    getValue: (perf) => formatPerfSuccessRate(perf.success_rate),
    getColor: (perf) => getSuccessRateColor(perf.success_rate),
  },
  {
    key: 'e2e',
    getLabel: (t) => (
      <PerfMetricLabel label='E2E' hint={t('E2E延迟说明')} className='text-inherit' />
    ),
    getValue: (perf) => formatPerfLatency(perf.avg_latency_ms),
  },
  {
    key: 'ttft',
    getLabel: (t) => (
      <PerfMetricLabel label='TTFT' hint={t('TTFT说明')} className='text-inherit' />
    ),
    getValue: (perf) => formatPerfLatency(perf.avg_ttft_ms),
  },
  {
    key: 'tps',
    getLabel: (t) => (
      <PerfMetricLabel label='TPS' hint={t('TPS说明')} className='text-inherit' />
    ),
    getValue: (perf) => formatPerfThroughput(perf.avg_tps),
  },
];

const ModelPerfMetricsGrid = memo(({ perf, t, compact = false }) => {
  if (!perf) return null;

  return (
    <div
      className='flex items-stretch'
      style={{
        paddingTop: compact ? 10 : 0,
        marginTop: compact ? 10 : 0,
        borderTop: compact ? '1px solid var(--semi-color-border)' : undefined,
      }}
    >
      {METRICS.map((metric, index) => (
        <MetricItem
          key={metric.key}
          compact={compact}
          showDivider={index > 0}
          label={metric.getLabel(t)}
          value={metric.getValue(perf)}
          valueColor={metric.getColor?.(perf)}
        />
      ))}
    </div>
  );
});

ModelPerfMetricsGrid.displayName = 'ModelPerfMetricsGrid';

export default ModelPerfMetricsGrid;
