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

import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import LanguageDetector from 'i18next-browser-languagedetector';

import enTranslation from './locales/en.json';
import frTranslation from './locales/fr.json';
import zhCNTranslation from './locales/zh-CN.json';
import zhTWTranslation from './locales/zh-TW.json';
import ruTranslation from './locales/ru.json';
import jaTranslation from './locales/ja.json';
import viTranslation from './locales/vi.json';
import idTranslation from './locales/id.json';
import msTranslation from './locales/ms.json';
import thTranslation from './locales/th.json';
import swTranslation from './locales/sw.json';
import { normalizeLanguage, supportedLanguages } from './language';

const normalizeResource = (resource) => {
  const { translation = {}, ...rootTranslations } = resource || {};
  return {
    translation: {
      ...rootTranslations,
      ...translation,
    },
  };
};

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    detection: {
      convertDetectedLanguage: (lng) => normalizeLanguage(lng),
    },
    load: 'currentOnly',
    supportedLngs: supportedLanguages,
    resources: {
      en: normalizeResource(enTranslation),
      'zh-CN': normalizeResource(zhCNTranslation),
      'zh-TW': normalizeResource(zhTWTranslation),
      fr: normalizeResource(frTranslation),
      ru: normalizeResource(ruTranslation),
      ja: normalizeResource(jaTranslation),
      vi: normalizeResource(viTranslation),
      id: normalizeResource(idTranslation),
      ms: normalizeResource(msTranslation),
      th: normalizeResource(thTranslation),
      sw: normalizeResource(swTranslation),
    },
    fallbackLng: 'zh-CN',
    nsSeparator: false,
    interpolation: {
      escapeValue: false,
    },
  });

window.__i18n = i18n;

export default i18n;
