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

import { API } from './api';

export function formatPerfLatency(ms) {
  if (!Number.isFinite(ms) || ms <= 0) return '—';
  if (ms >= 1000) return `${(ms / 1000).toFixed(ms >= 10000 ? 1 : 2)}s`;
  return `${Math.round(ms)}ms`;
}

export function formatPerfThroughput(tps) {
  if (!Number.isFinite(tps) || tps <= 0) return '—';
  if (tps >= 1000) return `${(tps / 1000).toFixed(1)}K t/s`;
  return `${tps.toFixed(tps < 10 ? 2 : 1)} t/s`;
}

export function formatPerfSuccessRate(rate) {
  if (!Number.isFinite(rate)) return '—';
  return `${rate.toFixed(1)}%`;
}

export function formatPerfCacheHitRate(rate) {
  if (!Number.isFinite(rate)) return '—';
  return `${rate.toFixed(1)}%`;
}

export function getCacheHitRateColor(rate) {
  if (!Number.isFinite(rate) || rate <= 0) return 'var(--semi-color-text-2)';
  if (rate >= 50) return '#3b82f6';
  return '#60a5fa';
}

export function getSuccessRateLevel(rate) {
  if (!Number.isFinite(rate)) return 'unknown';
  if (rate >= 100) return 'excellent';
  if (rate >= 90) return 'good';
  if (rate >= 70) return 'warning';
  return 'critical';
}

const SUCCESS_RATE_COLOR = {
  excellent: '#10b981',
  good: '#34d399',
  warning: '#f59e0b',
  critical: '#ef4444',
  unknown: '#9ca3af',
};

export function getSuccessRateColor(rate) {
  return SUCCESS_RATE_COLOR[getSuccessRateLevel(rate)];
}

export function getHourlyBarColor(point) {
  if (!point || (point.request_count || 0) <= 0) {
    return '#e5e7eb';
  }
  return getSuccessRateColor(point.success_rate);
}

export function formatPerfHourLabel(ts) {
  if (!ts) return '';
  const d = new Date(ts * 1000);
  const pad = (n) => String(n).padStart(2, '0');
  return `${d.getFullYear()}/${pad(d.getMonth() + 1)}/${pad(d.getDate())} ${pad(d.getHours())}:00`;
}

export function buildHourlySeriesFromSummary(perf) {
  if (perf?.hourly_series?.length) {
    return perf.hourly_series;
  }
  return [];
}

export async function fetchPerfMetricsSummary(hours = 24) {
  const res = await API.get('/api/perf_metrics/summary', { params: { hours } });
  const { success, data, message } = res.data || {};
  if (!success) {
    throw new Error(message || 'failed to load perf metrics summary');
  }
  const map = {};
  (data?.models || []).forEach((item) => {
    if (item?.model_name) {
      map[item.model_name] = item;
    }
  });
  return map;
}

export async function fetchPerfMetrics(
  modelName,
  hours = 24,
  group = '',
  channelId,
) {
  const params = { model: modelName, hours };
  if (group) params.group = group;
  if (channelId) params.channel_id = channelId;
  const res = await API.get('/api/perf_metrics', { params });
  const { success, data, message } = res.data || {};
  if (!success) {
    throw new Error(message || 'failed to load perf metrics');
  }
  return data;
}

export function perfQueryResultToSummary(result) {
  const groups = Array.isArray(result?.groups) ? result.groups : [];
  if (groups.length === 0) return null;
  const primary = groups[0];
  return {
    model_name: result.model_name,
    avg_latency_ms: primary.avg_latency_ms,
    avg_ttft_ms: primary.avg_ttft_ms,
    success_rate: primary.success_rate,
    avg_tps: primary.avg_tps,
    cache_hit_rate: primary.cache_hit_rate,
    hourly_series: primary.series || [],
  };
}
