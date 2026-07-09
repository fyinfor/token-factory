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

import React, { useEffect, useMemo } from 'react';
import DOMPurify from 'dompurify';
import { Button, Typography } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import MarkdownRenderer from '../common/markdown/MarkdownRenderer';

const { Text } = Typography;

export const LEGAL_CONTENT_FORMATS = {
  auto: 'auto',
  markdown: 'markdown',
  html: 'html',
  richtext: 'richtext',
};

const HTML_FORMATS = new Set([
  LEGAL_CONTENT_FORMATS.html,
  LEGAL_CONTENT_FORMATS.richtext,
]);

export function normalizeLegalContentFormat(format) {
  const value = String(format || '')
    .trim()
    .toLowerCase();
  if (
    value === LEGAL_CONTENT_FORMATS.markdown ||
    value === LEGAL_CONTENT_FORMATS.html ||
    value === LEGAL_CONTENT_FORMATS.richtext
  ) {
    return value;
  }
  return LEGAL_CONTENT_FORMATS.auto;
}

export function looksLikeHtml(content) {
  if (!content || typeof content !== 'string') return false;
  return /<\/?[a-z][\s\S]*>/i.test(content);
}

export function isAbsoluteUrl(content) {
  if (!content || typeof content !== 'string') return false;
  try {
    const url = new URL(content.trim());
    return url.protocol === 'http:' || url.protocol === 'https:';
  } catch {
    return false;
  }
}

function sanitizeCss(css) {
  return String(css || '')
    .replace(/<\/?style[^>]*>/gi, '')
    .replace(/@import[^;]+;/gi, '')
    .replace(/expression\s*\(/gi, '')
    .replace(/javascript:/gi, '');
}

export function parseLegalHtml(html) {
  const raw = String(html || '');
  if (!raw) {
    return { content: '', styles: '' };
  }

  const doc = new DOMParser().parseFromString(raw, 'text/html');
  const styles = Array.from(doc.querySelectorAll('style'))
    .map((style) => style.textContent || '')
    .join('\n');

  doc.querySelectorAll('style').forEach((style) => style.remove());

  return {
    content: DOMPurify.sanitize(doc.body?.innerHTML || raw, {
      ADD_ATTR: ['class', 'rel', 'style', 'target'],
    }),
    styles: sanitizeCss(styles),
  };
}

function LegalHtmlContent({ content, styleId }) {
  const { content: htmlContent, styles } = useMemo(
    () => parseLegalHtml(content),
    [content],
  );

  useEffect(() => {
    if (!styleId) return undefined;

    const existing = document.getElementById(styleId);
    if (!styles) {
      if (existing) existing.remove();
      return undefined;
    }

    let styleEl = existing;
    if (!styleEl) {
      styleEl = document.createElement('style');
      styleEl.id = styleId;
      styleEl.type = 'text/css';
      document.head.appendChild(styleEl);
    }
    styleEl.textContent = styles;

    return () => {
      const el = document.getElementById(styleId);
      if (el) el.remove();
    };
  }, [styles, styleId]);

  return (
    <div
      className='legal-document-html prose prose-lg max-w-none'
      dangerouslySetInnerHTML={{ __html: htmlContent }}
    />
  );
}

function LegalExternalLink({ content, title }) {
  const { t } = useTranslation();
  const url = content.trim();

  return (
    <div className='text-center py-8'>
      <Text type='secondary'>
        {t('管理员设置了外部链接，点击下方按钮访问')}
      </Text>
      <div className='mt-4'>
        <Button
          theme='solid'
          type='primary'
          onClick={() => window.open(url, '_blank', 'noopener,noreferrer')}
        >
          {t('访问')}
          {title ? ` ${title}` : ''}
        </Button>
      </div>
    </div>
  );
}

export default function LegalContentRenderer({
  content,
  format = LEGAL_CONTENT_FORMATS.auto,
  styleId,
  title,
}) {
  const rawContent = String(content || '');
  const normalizedFormat = normalizeLegalContentFormat(format);
  const resolvedFormat =
    normalizedFormat === LEGAL_CONTENT_FORMATS.auto
      ? looksLikeHtml(rawContent)
        ? LEGAL_CONTENT_FORMATS.html
        : LEGAL_CONTENT_FORMATS.markdown
      : normalizedFormat;

  if (isAbsoluteUrl(rawContent)) {
    return <LegalExternalLink content={rawContent} title={title} />;
  }

  if (HTML_FORMATS.has(resolvedFormat)) {
    return <LegalHtmlContent content={rawContent} styleId={styleId} />;
  }

  return (
    <div className='prose prose-lg max-w-none'>
      <MarkdownRenderer content={rawContent} />
    </div>
  );
}
