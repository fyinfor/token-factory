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

import React, { memo, useEffect, useMemo } from 'react';
import { VChart } from '@visactor/react-vchart';
import { initVChartSemiTheme } from '@visactor/vchart-semi-theme';
import {
  formatPerfCacheHitRate,
  formatPerfHourLabel,
  formatPerfLatency,
  formatPerfSuccessRate,
  formatPerfThroughput,
} from '../../../../helpers/perfMetrics';
import { CHART_CONFIG } from '../../../../constants/dashboard.constants';

const formatHour = (ts) => {
  if (!ts) return '';
  const date = new Date(ts * 1000);
  return `${String(date.getHours()).padStart(2, '0')}:00`;
};

const ModelPerfLineChart = memo(({ series = [], t }) => {
  useEffect(() => {
    initVChartSemiTheme({ isWatchingThemeSwitch: true });
  }, []);

  const values = useMemo(
    () =>
      series.map((point) => ({
        ...point,
        hour: formatHour(point.ts),
        requestCount: Math.max(0, Number(point.request_count || 0)),
      })),
    [series],
  );

  const spec = useMemo(() => {
    const maxValue = Math.max(1, ...values.map((item) => item.requestCount));
    return {
      type: 'line',
      animation: false,
      data: { values },
      xField: 'hour',
      yField: 'requestCount',
      background: 'transparent',
      padding: { top: 8, right: 8, bottom: 2, left: 2 },
      line: {
        style: {
          curveType: 'monotone',
          lineWidth: 2,
          stroke: '#64748b',
        },
      },
      area: { visible: false },
      point: {
        visible: false,
        state: {
          hover: {
            visible: true,
            size: 6,
            fill: '#64748b',
            stroke: 'rgba(255, 255, 255, 0.88)',
            lineWidth: 2,
          },
        },
      },
      axes: [
        {
          orient: 'bottom',
          sampling: false,
          tick: { visible: false },
          domainLine: { visible: false },
          label: {
            visible: true,
            flush: true,
            space: 6,
            style: { fontSize: 10 },
            formatMethod: (label, datum, index) => {
              if (index === 0 || index === values.length - 1) return label;
              return index % 6 === 0 ? label : '';
            },
          },
        },
        {
          orient: 'left',
          min: 0,
          max: Math.ceil(maxValue * 1.15),
          tick: { visible: false, tickCount: 4 },
          domainLine: { visible: false },
          grid: {
            visible: true,
            style: {
              lineDash: [4, 5],
              stroke: 'rgba(100, 116, 139, 0.16)',
            },
          },
          label: {
            style: { fontSize: 10 },
            formatMethod: (value) => String(Math.round(Number(value) || 0)),
          },
        },
      ],
      crosshair: {
        xField: {
          visible: true,
          line: {
            visible: true,
            type: 'line',
            style: {
              stroke: 'rgba(100, 116, 139, 0.42)',
              lineDash: [3, 4],
              opacity: 0.7,
            },
          },
        },
      },
      tooltip: {
        activeType: 'dimension',
        trigger: 'hover',
        dimension: {
          title: {
            visible: true,
            value: (datum) => formatPerfHourLabel(datum?.ts),
          },
          content: [
            {
              key: t('请求数'),
              value: (datum) => String(datum?.requestCount || 0),
            },
            {
              key: 'E2E',
              value: (datum) => formatPerfLatency(datum?.avg_latency_ms),
            },
            {
              key: 'TTFT',
              value: (datum) => formatPerfLatency(datum?.avg_ttft_ms),
            },
            {
              key: 'TPS',
              value: (datum) => formatPerfThroughput(datum?.avg_tps),
            },
            {
              key: 'Cache',
              value: (datum) => formatPerfCacheHitRate(datum?.cache_hit_rate),
            },
            {
              key: t('在线率'),
              value: (datum) => formatPerfSuccessRate(datum?.success_rate),
            },
          ],
        },
      },
    };
  }, [t, values]);

  if (!values.length) return null;

  return (
    <div
      className='overflow-hidden'
      style={{ height: 128, backgroundColor: 'transparent' }}
    >
      <VChart spec={spec} option={CHART_CONFIG} />
    </div>
  );
});

ModelPerfLineChart.displayName = 'ModelPerfLineChart';

export default ModelPerfLineChart;
