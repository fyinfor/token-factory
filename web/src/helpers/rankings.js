/*
Copyright (C) 2025 QuantumNous
*/

import { API } from './api';

const VALID_PERIODS = new Set(['today', 'week', 'month', 'year']);
// 分类取值与 system 其他位置（playground display_mode、供应商能力多模态选项）保持一致：
// 'all' 不过滤；'text' 文本对话；'image' 文生图；'video' 文生视频（含 Seedance 等专用视频模型）。
const VALID_CATEGORIES = new Set(['all', 'text', 'image', 'video']);

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

// 分类标签 i18n key 映射。
// 保持与 system 内其他模块（playground/SettingsPanel、SupplierCapabilityFormFields）一致：
// 文本/图片/视频。Seedance 仍归入视频 tab。
export const RANKING_CATEGORY_LABEL_KEYS = {
  text: '文本',
  image: '图片',
  video: '视频',
  all: '全部',
};

// 分类配色（与卡片风格保持一致，给前端徽章用）。
export const RANKING_CATEGORY_COLORS = {
  text: { color: '#0ea5e9', background: 'rgba(14, 165, 233, 0.12)' },
  image: { color: '#f97316', background: 'rgba(249, 115, 22, 0.14)' },
  video: { color: '#8b5cf6', background: 'rgba(139, 92, 246, 0.14)' },
};

export function getRankingCategoryLabel(category, t) {
  const key = RANKING_CATEGORY_LABEL_KEYS[category] || RANKING_CATEGORY_LABEL_KEYS.text;
  return t ? t(key) : key;
}

export function getRankingCategoryStyle(category) {
  return RANKING_CATEGORY_COLORS[category] || RANKING_CATEGORY_COLORS.text;
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
