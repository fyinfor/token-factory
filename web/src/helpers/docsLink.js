/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published
by the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

/**
 * Locales for the docs app (token-factory-docs/src/lib/i18n.ts).
 * Paths are /{lang}/docs/...
 */
const DOCS_DEFAULT_LOCALE = 'en';
const DOCS_SUPPORTED_LOCALES = new Set(['en', 'zh', 'ja']);
/** Match /{lang}/docs or /{lang}/docs/... */
const DOCS_LOCALE_PATH_RE = /^\/([a-z]{2})(\/docs(?:\/.*)?)?$/i;
const DOCS_BARE_PATH_RE = /^\/docs(?:\/.*)?$/i;

/**
 * Map the web UI i18n language to a docs site locale.
 * Unsupported UI languages fall back to English.
 * @param {string | undefined} webLanguage
 * @returns {'en' | 'zh' | 'ja'}
 */
export function mapWebLanguageToDocsLocale(webLanguage) {
  if (!webLanguage) {
    return DOCS_DEFAULT_LOCALE;
  }
  const lower = webLanguage.trim().toLowerCase().replace(/_/g, '-');
  let mapped = DOCS_DEFAULT_LOCALE;
  if (lower === 'ja' || lower.startsWith('ja-')) {
    mapped = 'ja';
  } else if (
    lower === 'zh' ||
    lower.startsWith('zh-') ||
    lower.startsWith('zh-hans') ||
    lower.startsWith('zh-hant')
  ) {
    mapped = 'zh';
  }
  return DOCS_SUPPORTED_LOCALES.has(mapped) ? mapped : DOCS_DEFAULT_LOCALE;
}

/**
 * Rewrite a docs URL/path so the language segment matches docsLocale.
 * Recognizes /{lang}/docs..., /docs..., and site-root bases; leaves unrelated paths alone.
 * @param {string} href
 * @param {string} docsLocale
 * @param {string} [origin] - used to resolve relative URLs for same-origin checks
 * @returns {string}
 */
export function localizeDocsHref(href, docsLocale, origin = '') {
  const trimmed = (href || '').trim();
  if (!trimmed) {
    return `/${docsLocale}/docs`;
  }

  const base = origin || 'http://localhost';
  let url;
  try {
    url = new URL(trimmed, base);
  } catch {
    return trimmed;
  }

  const pathname = url.pathname.replace(/\/+$/, '') || '/';
  const localeMatch = pathname.match(DOCS_LOCALE_PATH_RE);
  if (localeMatch) {
    const rest = localeMatch[2] || '/docs';
    url.pathname = `/${docsLocale}${rest}`;
    return serializeDocsUrl(url, trimmed, origin);
  }

  if (DOCS_BARE_PATH_RE.test(pathname)) {
    url.pathname = `/${docsLocale}${pathname}`;
    return serializeDocsUrl(url, trimmed, origin);
  }

  if (pathname === '/') {
    url.pathname = `/${docsLocale}/docs`;
    return serializeDocsUrl(url, trimmed, origin);
  }

  return trimmed;
}

/**
 * @param {URL} url
 * @param {string} original
 * @param {string} origin
 */
function serializeDocsUrl(url, original, origin) {
  const looksAbsolute = /^[a-z][a-z0-9+.-]*:/i.test(original);
  if (looksAbsolute) {
    return url.toString().replace(/\/$/, original.endsWith('/') ? '/' : '');
  }
  // Relative input: keep path (+ search/hash), drop resolved origin
  if (origin && url.origin === new URL(origin).origin) {
    return `${url.pathname}${url.search}${url.hash}`;
  }
  return `${url.pathname}${url.search}${url.hash}`;
}

/**
 * @param {string} serverDocsLink - operation_setting.general docs_link; empty uses same-origin docs
 * @param {string} webLanguage - i18n.language
 * @returns {{ href: string, openInNewTab: boolean }}
 */
export function resolveDocsNav(serverDocsLink, webLanguage) {
  const trimmed = (serverDocsLink || '').trim();
  const origin = typeof window !== 'undefined' ? window.location.origin : '';
  const docsLocale = mapWebLanguageToDocsLocale(webLanguage);
  const path = `/${docsLocale}/docs`;

  if (trimmed) {
    let openInNewTab = true;
    try {
      const base = origin || 'http://localhost';
      const resolved = new URL(trimmed, base);
      if (origin && resolved.origin === new URL(origin).origin) {
        openInNewTab = false;
      }
    } catch {
      // keep openInNewTab true
    }
    return {
      href: localizeDocsHref(trimmed, docsLocale, origin),
      openInNewTab,
    };
  }

  const href = origin ? `${origin}${path}` : path;
  return { href, openInNewTab: false };
}
