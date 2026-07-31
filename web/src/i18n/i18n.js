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

import { normalizeLanguage, supportedLanguages } from './language';

const localeLoaders = {
  en: () => import('./locales/en.json'),
  fr: () => import('./locales/fr.json'),
  'zh-CN': () => import('./locales/zh-CN.json'),
  'zh-TW': () => import('./locales/zh-TW.json'),
  ru: () => import('./locales/ru.json'),
  ja: () => import('./locales/ja.json'),
  vi: () => import('./locales/vi.json'),
  id: () => import('./locales/id.json'),
  ms: () => import('./locales/ms.json'),
  th: () => import('./locales/th.json'),
  sw: () => import('./locales/sw.json'),
};

const normalizeResource = (resource) => {
  const { translation = {}, ...rootTranslations } = resource || {};
  return {
    translation: {
      ...rootTranslations,
      ...translation,
    },
  };
};

const dynamicLocaleBackend = {
  type: 'backend',
  read(language, _namespace, callback) {
    const normalizedLanguage = normalizeLanguage(language);
    const loadLocale = localeLoaders[normalizedLanguage];
    if (!loadLocale) {
      callback(new Error(`Unsupported language: ${language}`), false);
      return;
    }
    loadLocale()
      .then((module) => {
        const resource = normalizeResource(module.default || module);
        callback(null, resource.translation);
      })
      .catch((error) => callback(error, false));
  },
};

export const i18nReady = i18n
  .use(LanguageDetector)
  .use(dynamicLocaleBackend)
  .use(initReactI18next)
  .init({
    detection: {
      convertDetectedLanguage: (lng) => normalizeLanguage(lng),
    },
    load: 'currentOnly',
    supportedLngs: supportedLanguages,
    fallbackLng: 'zh-CN',
    nsSeparator: false,
    interpolation: {
      escapeValue: false,
    },
  });

window.__i18n = i18n;

export default i18n;
