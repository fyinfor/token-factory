/*
Copyright (C) 2025 QuantumNous
*/

import { API } from './api';

const VALID_PERIODS = new Set(['today', 'week', 'month', 'year']);

export function normalizeRankingPeriod(period) {
  const value = String(period || 'week').trim().toLowerCase();
  return VALID_PERIODS.has(value) ? value : 'week';
}

export function formatRankingTokens(value) {
  const n = Number(value);
  if (!Number.isFinite(n) || n <= 0) return '0';
  if (n >= 1_000_000_000) return `${(n / 1_000_000_000).toFixed(2)}B`;
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(2)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`;
  return String(Math.round(n));
}

export function formatRankingShare(share) {
  const n = Number(share);
  if (!Number.isFinite(n)) return '—';
  if (n < 0.01 && n > 0) return '<0.01%';
  return `${n.toFixed(n >= 10 ? 1 : 2)}%`;
}

export function formatRankingGrowth(growth) {
  const n = Number(growth);
  if (!Number.isFinite(n)) return '—';
  const sign = n > 0 ? '+' : '';
  return `${sign}${n.toFixed(1)}%`;
}

export function getRankingGrowthColor(growth) {
  const n = Number(growth);
  if (!Number.isFinite(n) || n === 0) return 'var(--semi-color-text-2)';
  return n > 0 ? '#10b981' : '#ef4444';
}

export async function fetchRankings(period = 'week') {
  const normalized = normalizeRankingPeriod(period);
  const res = await API.get('/api/rankings', { params: { period: normalized } });
  const { success, data, message } = res.data || {};
  if (!success) {
    throw new Error(message || 'failed to load rankings');
  }
  return data || {};
}

export function pickPerfLeaders(models = [], limit = 6) {
  const withTraffic = models.filter(
    (m) =>
      (m.avg_latency_ms || 0) > 0 ||
      (m.success_rate || 0) > 0 ||
      (m.avg_tps || 0) > 0,
  );
  const source = withTraffic.length ? withTraffic : models;

  const byUptime = [...source]
    .filter((m) => Number.isFinite(m.success_rate))
    .sort((a, b) => b.success_rate - a.success_rate || a.avg_latency_ms - b.avg_latency_ms)
    .slice(0, limit);

  const byLatency = [...source]
    .filter((m) => (m.avg_latency_ms || 0) > 0)
    .sort((a, b) => a.avg_latency_ms - b.avg_latency_ms)
    .slice(0, limit);

  const byTps = [...source]
    .filter((m) => (m.avg_tps || 0) > 0)
    .sort((a, b) => b.avg_tps - a.avg_tps)
    .slice(0, limit);

  return { byUptime, byLatency, byTps };
}
