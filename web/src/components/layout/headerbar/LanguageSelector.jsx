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

import React from 'react';
import { Button, Dropdown } from '@douyinfe/semi-ui';
import { Languages } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import {
  normalizeLanguage,
  supportedLanguages,
  LANGUAGE_NATIVE_LABELS,
} from '../../../i18n/language';

/** 语言在头部与下拉中一律显示其自称（如 English、简体中文），不随界面语言翻译 */
const nativeLabel = (code) => LANGUAGE_NATIVE_LABELS[code] || code;

const itemClass = (active) =>
  `!px-3 !py-1.5 !text-sm !text-semi-color-text-0 dark:!text-gray-200 ${
    active
      ? '!bg-semi-color-primary-light-default dark:!bg-blue-600 !font-semibold'
      : 'hover:!bg-semi-color-fill-1 dark:hover:!bg-gray-600'
  }`;

const LanguageSelector = ({ currentLang, onLanguageChange }) => {
  const { t } = useTranslation();
  const normalized = normalizeLanguage(currentLang) || 'zh-CN';
  const currentLabel = nativeLabel(normalized);

  return (
    <Dropdown
      position='bottomRight'
      render={
        <Dropdown.Menu className='!bg-semi-color-bg-overlay !border-semi-color-border !shadow-lg !rounded-lg dark:!bg-gray-700 dark:!border-gray-600'>
          {supportedLanguages.map((code) => (
            <Dropdown.Item
              key={code}
              onClick={() => onLanguageChange(code)}
              className={itemClass(normalized === code)}
            >
              {nativeLabel(code)}
            </Dropdown.Item>
          ))}
        </Dropdown.Menu>
      }
    >
      <Button
        icon={<Languages size={18} />}
        aria-label={`${t('语言')}: ${currentLabel}`}
        title={`${t('语言')}: ${currentLabel}`}
        theme='borderless'
        type='tertiary'
        className='header-language-button !rounded-full !bg-semi-color-fill-0 !text-current hover:!bg-semi-color-fill-1 focus-visible:!ring-2 focus-visible:!ring-semi-color-primary dark:!bg-semi-color-fill-1 dark:hover:!bg-semi-color-fill-2'
      >
        <span className='header-language-label min-w-0 truncate text-sm font-medium'>
          {currentLabel}
        </span>
      </Button>
    </Dropdown>
  );
};

export default LanguageSelector;
