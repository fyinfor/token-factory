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

export const supportedLanguages = [
  'zh-CN',
  'zh-TW',
  'en',
  'fr',
  'ru',
  'ja',
  'vi',
  'id',
  'ms',
  'th',
  'sw',
];

export const normalizeLanguage = (language) => {
  if (!language) {
    return language;
  }

  const normalized = language.trim().replace(/_/g, '-');
  const lower = normalized.toLowerCase();

  if (
    lower === 'zh' ||
    lower === 'zh-cn' ||
    lower === 'zh-sg' ||
    lower.startsWith('zh-hans')
  ) {
    return 'zh-CN';
  }

  if (
    lower === 'zh-tw' ||
    lower === 'zh-hk' ||
    lower === 'zh-mo' ||
    lower.startsWith('zh-hant')
  ) {
    return 'zh-TW';
  }

  if (lower === 'id' || lower.startsWith('id-')) {
    return 'id';
  }

  if (lower === 'ms' || lower.startsWith('ms-')) {
    return 'ms';
  }

  if (lower === 'th' || lower.startsWith('th-')) {
    return 'th';
  }

  if (lower === 'sw' || lower.startsWith('sw-')) {
    return 'sw';
  }

  const matchedLanguage = supportedLanguages.find(
    (supportedLanguage) => supportedLanguage.toLowerCase() === lower,
  );

  return matchedLanguage || normalized;
};

/** 访客首次套用站点默认语言后写入 localStorage，避免每次覆盖 */
export const I18N_ANON_LANG_INITIALIZED_KEY = 'i18n_anonymous_lang_initialized_v1';

/** 浏览器语言提示条待展示（StrictMode 重挂载时从 session 恢复） */
export const I18N_BROWSER_LANG_BANNER_PENDING_KEY =
  'i18n_browser_lang_banner_pending_v1';

/** 浏览器语言与站点默认不一致时的提示，用户选择后写 done */
export const I18N_BROWSER_LANG_MISMATCH_PROMPT_KEY =
  'i18n_browser_lang_mismatch_prompt_v1';

export function isSupportedUiLanguage(code) {
  return Boolean(code && supportedLanguages.includes(code));
}

/**
 * 从 navigator.languages 中选取第一个本站支持的界面语言代码。
 * @returns {string} 受支持的语言代码，或空字符串
 */
export function pickPrimaryNavigatorLanguage() {
  if (typeof navigator === 'undefined') return '';
  const candidates =
    navigator.languages && navigator.languages.length > 0
      ? [...navigator.languages]
      : [navigator.language];
  for (const raw of candidates) {
    const n = normalizeLanguage(raw);
    if (n && supportedLanguages.includes(n)) {
      return n;
    }
  }
  return '';
}

export const LANGUAGE_NATIVE_LABELS = {
  'zh-CN': '简体中文',
  'zh-TW': '繁體中文',
  en: 'English',
  fr: 'Français',
  ru: 'Русский',
  ja: '日本語',
  vi: 'Tiếng Việt',
  id: 'Bahasa Indonesia',
  ms: 'Bahasa Melayu',
  th: 'ไทย',
  sw: 'Kiswahili',
};

export const languageSelectOptions = supportedLanguages.map((value) => ({
  value,
  label: LANGUAGE_NATIVE_LABELS[value] || value,
}));
