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
import { Typography } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';

const PlaygroundGeneratedImageGallery = ({
  images = [],
  onPreview,
  maxWidth = 'min(100%, 780px)',
}) => {
  const { t } = useTranslation();
  const list = Array.isArray(images) ? images.filter(Boolean) : [];
  if (!list.length) return null;

  const gridClass =
    list.length === 1
      ? 'grid-cols-1'
      : 'grid-cols-1 sm:grid-cols-2';

  return (
    <div className='mb-3' style={{ maxWidth }}>
      <Typography.Text type='tertiary' size='small' className='block mb-2'>
        {t('已生成 {{count}} 张图片', { count: list.length })}
      </Typography.Text>
      <div className={`grid gap-2 sm:gap-3 ${gridClass}`}>
        {list.map((src, index) => (
          <button
            key={`${index}-${src.slice(0, 32)}`}
            type='button'
            className='block w-full p-0 border-0 bg-transparent rounded-lg overflow-hidden cursor-zoom-in focus:outline-none focus-visible:ring-2 focus-visible:ring-purple-400'
            onClick={() => onPreview?.(src)}
            aria-label={t('查看第 {{index}} 张图片', { index: index + 1 })}
          >
            <img
              src={src}
              alt={t('生成图片 {{index}}', { index: index + 1 })}
              className='w-full h-auto rounded-lg border border-gray-200 object-contain bg-gray-50'
              style={{ maxHeight: list.length > 1 ? '360px' : '520px' }}
              loading='lazy'
            />
          </button>
        ))}
      </div>
    </div>
  );
};

export default PlaygroundGeneratedImageGallery;
