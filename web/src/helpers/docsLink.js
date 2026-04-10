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

/**
 * Map the web UI i18n language to a docs site locale.
 * @param {string | undefined} webLanguage
 * @returns {'en' | 'zh' | 'ja'}
 */
export function mapWebLanguageToDocsLocale(webLanguage) {
  if (!webLanguage) {
    return DOCS_DEFAULT_LOCALE;
  }
  const lower = webLanguage.trim().toLowerCase().replace(/_/g, '-');
  if (lower === 'ja' || lower.startsWith('ja-')) {
    return 'ja';
  }
  if (
    lower === 'zh' ||
    lower.startsWith('zh-') ||
    lower.startsWith('zh-hans') ||
    lower.startsWith('zh-hant')
  ) {
    return 'zh';
  }
  return DOCS_DEFAULT_LOCALE;
}

/**
 * @param {string} serverDocsLink - operation_setting.general docs_link; empty uses same-origin docs
 * @param {string} webLanguage - i18n.language
 * @returns {{ href: string, openInNewTab: boolean }}
 */
export function resolveDocsNav(serverDocsLink, webLanguage) {
  const trimmed = (serverDocsLink || '').trim();
  const origin =
    typeof window !== 'undefined' ? window.location.origin : '';
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
    return { href: trimmed, openInNewTab };
  }

  const href = origin ? `${origin}${path}` : path;
  return { href, openInNewTab: false };
}
