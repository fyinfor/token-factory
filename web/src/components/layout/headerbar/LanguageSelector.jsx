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

import React, { useMemo } from 'react';
import { Button, Dropdown } from '@douyinfe/semi-ui';
import { Languages } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { normalizeLanguage } from '../../../i18n/language';

/** 与 supportedLanguages 顺序一致；展示名由 Intl 随界面语言变化 */
const LANGUAGE_CODES = [
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

const FALLBACK_LABEL = {
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

function useDisplayNamesForUi(i18nLang) {
  return useMemo(() => {
    const loc = normalizeLanguage(i18nLang || '') || 'zh-CN';
    try {
      return new Intl.DisplayNames([loc], { type: 'language' });
    } catch {
      return null;
    }
  }, [i18nLang]);
}

function labelForCode(code, displayNames) {
  try {
    if (displayNames) {
      const n = displayNames.of(code);
      if (n) return n;
    }
  } catch {
    /* Intl 不可用时走回退 */
  }
  return FALLBACK_LABEL[code] || code;
}

const itemClass = (active) =>
  `!px-3 !py-1.5 !text-sm !text-semi-color-text-0 dark:!text-gray-200 ${
    active
      ? '!bg-semi-color-primary-light-default dark:!bg-blue-600 !font-semibold'
      : 'hover:!bg-semi-color-fill-1 dark:hover:!bg-gray-600'
  }`;

const LanguageSelector = ({ currentLang, onLanguageChange }) => {
  const { t, i18n } = useTranslation();
  const displayNames = useDisplayNamesForUi(i18n.language);
  const normalized = normalizeLanguage(currentLang) || 'zh-CN';

  const currentLabel = useMemo(
    () => labelForCode(normalized, displayNames),
    [normalized, displayNames],
  );

  return (
    <Dropdown
      position='bottomRight'
      render={
        <Dropdown.Menu className='!bg-semi-color-bg-overlay !border-semi-color-border !shadow-lg !rounded-lg dark:!bg-gray-700 dark:!border-gray-600'>
          {LANGUAGE_CODES.map((code) => (
            <Dropdown.Item
              key={code}
              onClick={() => onLanguageChange(code)}
              className={itemClass(normalized === code)}
            >
              {labelForCode(code, displayNames)}
            </Dropdown.Item>
          ))}
        </Dropdown.Menu>
      }
    >
      <Button
        icon={<Languages size={18} />}
        aria-label={`${t('common.changeLanguage')}: ${currentLabel}`}
        theme='borderless'
        type='tertiary'
        className='!px-2 !py-1.5 !text-current focus:!bg-semi-color-fill-1 dark:focus:!bg-gray-700 !rounded-full !bg-semi-color-fill-0 dark:!bg-semi-color-fill-1 hover:!bg-semi-color-fill-1 dark:hover:!bg-semi-color-fill-2 !max-w-[11rem] sm:!max-w-[14rem]'
      >
        <span className='truncate text-sm font-medium min-w-0'>
          {currentLabel}
        </span>
      </Button>
    </Dropdown>
  );
};

export default LanguageSelector;
