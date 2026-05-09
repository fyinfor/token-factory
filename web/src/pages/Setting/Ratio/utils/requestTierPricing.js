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

export const TIER_CATEGORIES = [
  { key: 'input', label: '输入价格' },
  { key: 'output', label: '输出价格' },
  { key: 'cache_read', label: '缓存读取价格' },
  { key: 'cache_write', label: '缓存写入价格' },
];

export const emptyTierRule = () => ({
  mode: 'progressive',
  input: [],
  output: [],
  cache_read: [],
  cache_write: [],
});

export const parseJSONMap = (raw) => {
  if (!raw || String(raw).trim() === '') return {};
  try {
    const parsed = JSON.parse(raw);
    if (Array.isArray(parsed)) {
      return Object.fromEntries(
        parsed.map((item, index) => [`tpl_${index + 1}`, item]),
      );
    }
    return parsed && typeof parsed === 'object' && !Array.isArray(parsed)
      ? parsed
      : {};
  } catch {
    return {};
  }
};

export const normalizeTierRows = (rows) =>
  (Array.isArray(rows) ? rows : [])
    .map((row) => ({
      up_to: Number(row?.up_to || 0),
      ratio: Number(row?.ratio ?? 0),
    }))
    .filter((row) => Number.isFinite(row.up_to) && Number.isFinite(row.ratio));

export const normalizeTierRule = (rule) => {
  const src = rule && typeof rule === 'object' ? rule : {};
  return {
    mode: src.mode || 'progressive',
    input: normalizeTierRows(src.input),
    output: normalizeTierRows(src.output),
    cache_read: normalizeTierRows(src.cache_read),
    cache_write: normalizeTierRows(src.cache_write),
  };
};

export const ensureFinalInfinityTierRows = (rows) => {
  const normalized = normalizeTierRows(rows);
  if (!normalized.length) return normalized;
  if (normalized[normalized.length - 1].up_to === 0) return normalized;
  return [...normalized, { up_to: 0, ratio: 0 }];
};

export const hasTierRule = (rule) => {
  const normalized = normalizeTierRule(rule);
  return TIER_CATEGORIES.some(({ key }) => normalized[key].length > 0);
};

export const summarizeTierRule = (rule, t = (v) => v, visibleCategories = null) => {
  if (!hasTierRule(rule)) return t('未配置');
  const normalized = normalizeTierRule(rule);
  return TIER_CATEGORIES.filter(({ key }) => {
    // 如果提供了 visibleCategories，只显示开关打开的类别（或 input）
    if (visibleCategories !== null) {
      return key === 'input' || visibleCategories[key];
    }
    // 否则显示所有有数据的类别
    return normalized[key].length > 0;
  })
    .filter(({ key }) => normalized[key].length > 0)
    .map(({ key, label }) => `${t(label)} ${normalized[key].length}${t('档')}`)
    .join(' / ');
};

const formatTierStartBound = (value) => String(value);

const formatTierEndBound = (value) => {
  if (value === 0) return '∞';
  return String(value);
};

const formatTierPrice = (value) => {
  if (!Number.isFinite(value)) return '-';
  const fixed = Number(value.toFixed(6));
  return `$${fixed}`;
};

export const buildTierPriceDetails = (rule, basePrices = {}, t = (v) => v, visibleCategories = null) => {
  if (!hasTierRule(rule)) return [];
  const normalized = normalizeTierRule(rule);
  return TIER_CATEGORIES.filter(({ key }) => {
    // 如果提供了 visibleCategories，只显示开关打开的类别（或 input）
    if (visibleCategories !== null) {
      return key === 'input' || visibleCategories[key];
    }
    // 否则显示所有有数据的类别
    return normalized[key].length > 0;
  })
    .filter(({ key }) => normalized[key].length > 0)
    .map(({ key, label }) => {
      let previous = 0;
      const rows = ensureFinalInfinityTierRows(normalized[key]);
      const segments = rows.map((row) => {
        const from = previous;
        const to = row.up_to;
        previous = row.up_to || previous;
        // 最后一行（up_to === 0）使用上一档的 token 数量
        const tokenCount = row.up_to === 0
          ? (row === rows[0] ? 1 : rows[rows.indexOf(row) - 1]?.up_to || 1)
          : row.up_to - from;
        // 直接从 ratio 转换为价格，不使用 basePrices
        const price = formatTierPrice(
          row.ratio ? ratioToPrice(row.ratio, tokenCount, 1) : 0,
        );
        return {
          range: `${formatTierStartBound(from)}～${formatTierEndBound(to)}`,
          price,
          ratio: row.ratio,
        };
      });
      return {
        key,
        category: key,
        label: t(label),
        segments,
      };
    });
};

export const validateTierRule = (rule, t = (v) => v) => {
  const normalized = normalizeTierRule(rule);
  if (normalized.mode !== 'progressive') {
    return t('仅支持 progressive 阶梯计费模式');
  }
  for (const { key, label } of TIER_CATEGORIES) {
    let previous = 0;
    const rows = normalized[key];
    for (let i = 0; i < rows.length; i += 1) {
      const row = rows[i];
      if (row.ratio === 0 || row.ratio === null || row.ratio === undefined) return `${t(label)} ${t('价格不能为空')}`;
      if (row.ratio < 0) return `${t(label)} ${t('倍率不能小于 0')}`;
      if (row.up_to < 0) return `${t(label)} up_to ${t('不能小于 0')}`;
      if (row.up_to === 0 && i !== rows.length - 1) {
        return `${t(label)} ${t('只有最后一档 up_to 可以为 0')}`;
      }
      if (row.up_to !== 0 && row.up_to <= previous) {
        return `${t(label)} up_to ${t('必须递增')}`;
      }
      if (row.up_to !== 0) previous = row.up_to;
    }
  }
  return '';
};

export const serializeTierRule = (rule) => {
  const normalized = normalizeTierRule(rule);
  const out = { mode: normalized.mode || 'progressive' };
  TIER_CATEGORIES.forEach(({ key }) => {
    if (normalized[key].length > 0) out[key] = normalized[key];
  });
  return out;
};

// 价格转倍率：倍率 = 价格(USD) / (token数 × 汇率)
export const priceToRatio = (priceUSD, tokenCount, exchangeRate = 1) => {
  if (!priceUSD || !tokenCount || tokenCount <= 0) return 0;
  const ratio = priceUSD / (tokenCount * exchangeRate);
  return Math.max(0, ratio);
};

// 倍率转价格：价格 = token数 × 倍率 × 汇率
export const ratioToPrice = (ratio, tokenCount, exchangeRate = 1) => {
  if (!ratio || !tokenCount || tokenCount <= 0) return 0;
  return tokenCount * ratio * exchangeRate;
};
