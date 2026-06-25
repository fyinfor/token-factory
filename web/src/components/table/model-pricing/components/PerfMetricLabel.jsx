/*
Copyright (C) 2025 QuantumNous
*/

import React, { memo } from 'react';
import { Tooltip } from '@douyinfe/semi-ui';

const PerfMetricLabel = memo(({ label, hint, className = '' }) => (
  <Tooltip content={hint} position='top'>
    <span
      className={`inline-flex items-center gap-1 cursor-help select-none ${className}`}
      onClick={(e) => e.stopPropagation()}
    >
      <span className='font-medium tracking-wide'>{label}</span>
      <span
        className='inline-flex h-3.5 w-3.5 shrink-0 items-center justify-center rounded-full text-[9px] font-semibold leading-none'
        style={{
          color: 'var(--semi-color-text-2)',
          backgroundColor: 'var(--semi-color-fill-1)',
        }}
      >
        ?
      </span>
    </span>
  </Tooltip>
));

PerfMetricLabel.displayName = 'PerfMetricLabel';

export default PerfMetricLabel;
