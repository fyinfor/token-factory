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

import { normalizeLanguage } from '../i18n/language';

const isChineseLanguage = (language) => {
  const normalized = normalizeLanguage(language);
  return normalized === 'zh-CN' || normalized === 'zh-TW';
};

/**
 * Select an announcement's display text for the active UI language.
 * English is intentionally the shared fallback for all non-Chinese locales.
 */
export const getLocalizedAnnouncement = (announcement, language) => {
  const content = String(
    announcement?.sourceContent ?? announcement?.content ?? '',
  );
  const extra = String(announcement?.sourceExtra ?? announcement?.extra ?? '');
  const contentEn = String(announcement?.content_en || '').trim();
  const extraEn = String(announcement?.extra_en || '').trim();
  const useEnglish = !isChineseLanguage(language) && Boolean(contentEn);

  return {
    ...announcement,
    content: useEnglish ? contentEn : content,
    extra: useEnglish && extraEn ? extraEn : extra,
    // Keep read-state and acknowledgement keys stable across language switches.
    sourceContent: content,
    sourceExtra: extra,
  };
};
