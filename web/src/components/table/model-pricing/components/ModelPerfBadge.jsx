/*
Copyright (C) 2025 QuantumNous
*/

import React, { memo } from 'react';
import { Tooltip } from '@douyinfe/semi-ui';
import {
  formatPerfLatency,
  formatPerfThroughput,
  formatPerfSuccessRate,
  formatPerfCacheHitRate,
  getSuccessRateColor,
} from '../../../../helpers/perfMetrics';

const ModelPerfBadge = memo(({ perf, t, compact = false }) => {
  if (!perf) return null;

  const latencyLabel = formatPerfLatency(perf.avg_latency_ms);
  const ttftLabel = formatPerfLatency(perf.avg_ttft_ms);
  const tpsLabel = formatPerfThroughput(perf.avg_tps);
  const successLabel = formatPerfSuccessRate(perf.success_rate);

  if (compact) {
    return (
      <Tooltip
        content={
          <div className='text-xs leading-5'>
            <div>
              {t('在线率')}: {successLabel}
            </div>
            <div>E2E: {latencyLabel}</div>
            <div>TTFT: {ttftLabel}</div>
            <div>TPS: {tpsLabel}</div>
            <div>Cache: {formatPerfCacheHitRate(perf.cache_hit_rate)}</div>
          </div>
        }
      >
        <span
          className='inline-flex items-center gap-1 text-xs text-gray-500 font-mono'
          onClick={(e) => e.stopPropagation()}
        >
          <span
            className='inline-block w-1.5 h-1.5 rounded-full'
            style={{ backgroundColor: getSuccessRateColor(perf.success_rate) }}
          />
          {latencyLabel}
        </span>
      </Tooltip>
    );
  }

  return null;
});

ModelPerfBadge.displayName = 'ModelPerfBadge';

export default ModelPerfBadge;
