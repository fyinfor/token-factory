/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

For commercial licensing, please contact support@quantumnous.com
*/

import React from 'react';

const MAX_LEN = 280;

const MARKERS = [
  { open: '**', close: '**', type: 'bold' },
  { open: '^^', close: '^^', type: 'highlight' },
];

/** 预设色名 → 色值（仅允许白名单，禁止任意 CSS） */
const NAMED_COLORS = {
  red: '#ef4444',
  orange: '#f97316',
  amber: '#f59e0b',
  yellow: '#eab308',
  green: '#22c55e',
  emerald: '#10b981',
  blue: '#3b82f6',
  indigo: '#4f46e5',
  violet: '#7c3aed',
  purple: '#a855f7',
  pink: '#ec4899',
  rose: '#f43f5e',
  primary: '#4f46e5',
};

function sanitizeHexColor(input) {
  const m = /^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$/.exec(String(input).trim());
  if (!m) return null;
  let hex = m[1].toLowerCase();
  if (hex.length === 3) {
    hex = hex
      .split('')
      .map((c) => c + c)
      .join('');
  }
  return `#${hex}`;
}

/** ^^文字^^ 或 ^^#f97316:文字^^ / ^^orange:文字^^ */
function parseHighlightInner(inner) {
  const colonIdx = inner.indexOf(':');
  if (colonIdx <= 0) {
    return { text: inner, color: null };
  }

  const colorPart = inner.slice(0, colonIdx).trim();
  const text = inner.slice(colonIdx + 1);
  if (!text) {
    return { text: inner, color: null };
  }

  const hex = sanitizeHexColor(colorPart);
  if (hex) {
    return { text, color: hex };
  }

  const named = NAMED_COLORS[colorPart.toLowerCase()];
  if (named) {
    return { text, color: named };
  }

  return { text: inner, color: null };
}

/** 去除 HTML 标签，避免注入 */
export function sanitizeBannerPlainText(text) {
  if (text == null) return '';
  return String(text)
    .replace(/<[^>]*>/g, '')
    .replace(/[\u0000-\u0008\u000b\u000c\u000e-\u001f]/g, '')
    .slice(0, MAX_LEN);
}

function findNextMarker(str, from) {
  let best = null;
  for (const m of MARKERS) {
    const idx = str.indexOf(m.open, from);
    if (idx >= 0 && (best == null || idx < best.index)) {
      best = { index: idx, ...m };
    }
  }
  return best;
}

/**
 * 将副标题解析为 token；支持 **加粗**、^^高亮^^、^^#色值:文字^^、^^色名:文字^^，不解析 HTML。
 */
export function tokenizeBannerRichText(text) {
  const clean = sanitizeBannerPlainText(text);
  if (!clean) return [];

  const tokens = [];
  let pos = 0;

  while (pos < clean.length) {
    const m = findNextMarker(clean, pos);
    if (!m) {
      tokens.push({ type: 'text', value: clean.slice(pos) });
      break;
    }
    if (m.index > pos) {
      tokens.push({ type: 'text', value: clean.slice(pos, m.index) });
    }
    const contentStart = m.index + m.open.length;
    const closeIdx = clean.indexOf(m.close, contentStart);
    if (closeIdx < 0) {
      tokens.push({ type: 'text', value: clean.slice(m.index) });
      break;
    }
    const inner = clean.slice(contentStart, closeIdx);
    if (inner) {
      if (m.type === 'highlight') {
        tokens.push({ type: 'highlight', ...parseHighlightInner(inner) });
      } else {
        tokens.push({ type: m.type, value: inner });
      }
    }
    pos = closeIdx + m.close.length;
  }

  return tokens;
}

/**
 * @returns {React.ReactNode}
 */
export function renderBannerRichText(text) {
  const tokens = tokenizeBannerRichText(text);
  if (!tokens.length) return null;

  return tokens.map((tok, i) => {
    if (tok.type === 'text') {
      return tok.value ? (
        <React.Fragment key={i}>{tok.value}</React.Fragment>
      ) : null;
    }
    if (tok.type === 'bold') {
      return (
        <strong key={i} className='ad-rich-bold'>
          {tok.value}
        </strong>
      );
    }
    if (tok.type === 'highlight') {
      const style = tok.color ? { color: tok.color } : undefined;
      return (
        <span key={i} className='ad-subtitle-highlight' style={style}>
          {tok.text}
        </span>
      );
    }
    return null;
  });
}
