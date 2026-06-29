/*
Copyright (C) 2025 QuantumNous
*/

import { API } from './api';

const VALID_PERIODS = new Set(['today', 'week', 'month', 'year']);
const VALID_CATEGORIES = new Set(['all', 't2i', 't2v', 'seedance']);

export function normalizeRankingPeriod(period) {
  const value = String(period || 'week').trim().toLowerCase();
  return VALID_PERIODS.has(value) ? value : 'week';
}

export function normalizeRankingCategory(category) {
  const value = String(category || 'all').trim().toLowerCase();
  return VALID_CATEGORIES.has(value) ? value : 'all';
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

// 分类标签 i18n key 映射（用于在排行中按 category 显示徽章）。
export const RANKING_CATEGORY_LABEL_KEYS = {
  t2i: '文生图',
  t2v: '文生视频',
  seedance: 'Seedance',
  chat: '对话',
  all: '全部',
};

// 分类配色（与卡片风格保持一致，给前端徽章用）。
export const RANKING_CATEGORY_COLORS = {
  t2i: { color: '#0ea5e9', background: 'rgba(14, 165, 233, 0.12)' },
  t2v: { color: '#8b5cf6', background: 'rgba(139, 92, 246, 0.12)' },
  seedance: { color: '#f97316', background: 'rgba(249, 115, 22, 0.14)' },
  chat: { color: '#64748b', background: 'rgba(100, 116, 139, 0.12)' },
};

// 在「全部」Tab 下，按分类的展示优先级：seedance > t2v > t2i > chat
// 用于决定徽章显示的语义化标签。
export function getRankingCategoryLabel(category, t) {
  const key = RANKING_CATEGORY_LABEL_KEYS[category] || RANKING_CATEGORY_LABEL_KEYS.chat;
  return t ? t(key) : key;
}

export function getRankingCategoryStyle(category) {
  return RANKING_CATEGORY_COLORS[category] || RANKING_CATEGORY_COLORS.chat;
}

export async function fetchRankings(period = 'week', category = 'all') {
  const normalizedPeriod = normalizeRankingPeriod(period);
  const normalizedCategory = normalizeRankingCategory(category);
  const res = await API.get('/api/rankings', {
    params: { period: normalizedPeriod, category: normalizedCategory },
  });
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
