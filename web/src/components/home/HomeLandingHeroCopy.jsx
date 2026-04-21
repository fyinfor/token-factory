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
import { useTranslation } from 'react-i18next';

/** 首页 Banner 主标题 + 副标题（模型广场等页复用，与首页文案与样式一致） */
const HomeLandingHeroCopy = ({ className = '' }) => {
  const { t, i18n } = useTranslation();
  const isChinese = i18n.language.startsWith('zh');

  return (
    <div
      className={`flex flex-col items-center justify-center text-center max-w-3xl mx-auto ${className}`.trim()}
    >
      <h1
        className={`text-4xl md:text-5xl lg:text-6xl font-medium text-semi-color-text-0 leading-tight mb-4 ${isChinese ? 'tracking-wide' : ''}`}
      >
        {t('一站式大模型服务统一入口')}
      </h1>
      <p className='text-sm md:text-base text-semi-color-text-2 mb-8'>
        {t('按需定制，更优')}
        <span className='font-semibold text-semi-color-text-0'>{t('价格')}</span>
        {t('，更稳的')}
        <span className='font-semibold text-semi-color-text-0'>{t('可靠')}</span>
        {t('，开箱即用')}
      </p>
    </div>
  );
};

export default HomeLandingHeroCopy;
