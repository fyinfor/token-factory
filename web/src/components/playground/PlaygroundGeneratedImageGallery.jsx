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

import React, { useState } from 'react';
import { Typography } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';

const getConstrainedMediaSize = (dimensions, maxWidth, maxHeight) => {
  const width = Number(dimensions?.width || 0);
  const height = Number(dimensions?.height || 0);
  if (!width || !height) return {};
  const ratio = Math.min(maxWidth / width, maxHeight / height, 1);
  return {
    width: Math.round(width * ratio),
    height: Math.round(height * ratio),
  };
};

const PlaygroundGeneratedImageGallery = ({
  images = [],
  dimensions = {},
  onMediaLoad,
  onRevealProgress,
  onPreview,
  maxWidth = 'min(100%, 780px)',
  maxHeight = 520,
}) => {
  const { t } = useTranslation();
  const [loadedUrls, setLoadedUrls] = useState(() => new Set());
  const [loadedDimensions, setLoadedDimensions] = useState({});
  const list = Array.isArray(images) ? images.filter(Boolean) : [];
  if (!list.length) return null;
  const hasLoadedImages = list.some((src) => loadedUrls.has(src));

  const gridClass =
    list.length === 1 ? 'grid-cols-1' : 'grid-cols-1 sm:grid-cols-2';

  return (
    <div
      style={{
        maxWidth,
        marginBottom: hasLoadedImages ? '0.75rem' : 0,
        overflow: 'hidden',
        transition: 'margin-bottom 200ms ease',
      }}
    >
      {hasLoadedImages && (
        <Typography.Text type='tertiary' size='small' className='block mb-2'>
          {t('已生成 {{count}} 张图片', { count: list.length })}
        </Typography.Text>
      )}
      <div
        className={`grid ${gridClass}`}
        style={{
          gap: hasLoadedImages ? undefined : 0,
          overflow: 'hidden',
          transition: 'gap 200ms ease',
        }}
      >
        {list.map((src, index) => {
          const constrainedSize = getConstrainedMediaSize(
            loadedDimensions[src] || dimensions[src],
            780,
            list.length > 1 ? 360 : maxHeight,
          );
          return (
            <button
              key={`${index}-${src.slice(0, 32)}`}
              type='button'
              className='relative block p-0 border-0 bg-transparent rounded-lg overflow-hidden cursor-zoom-in focus:outline-none focus-visible:ring-2 focus-visible:ring-purple-400'
              style={{
                width: constrainedSize.width
                  ? loadedUrls.has(src)
                    ? `${constrainedSize.width}px`
                    : 0
                  : 0,
                height: constrainedSize.height
                  ? loadedUrls.has(src)
                    ? `${constrainedSize.height}px`
                    : 0
                  : 0,
                maxWidth: '100%',
                overflow: 'hidden',
                opacity: loadedUrls.has(src) ? 1 : 0,
                transition:
                  'width 200ms ease, height 200ms ease, opacity 200ms ease',
                transitionDelay: loadedUrls.has(src)
                  ? '0ms, 0ms, 200ms'
                  : '0ms',
              }}
              onClick={() => onPreview?.(src)}
              aria-label={t('查看第 {{index}} 张图片', { index: index + 1 })}
            >
              <img
                src={src}
                alt={t('生成图片 {{index}}', { index: index + 1 })}
                className='block rounded-lg border border-gray-200 object-contain bg-gray-50'
                style={{
                  width: constrainedSize.width
                    ? `${constrainedSize.width}px`
                    : '100%',
                  height: constrainedSize.height
                    ? `${constrainedSize.height}px`
                    : 'auto',
                  maxWidth: '100%',
                  maxHeight: list.length > 1 ? '360px' : `${maxHeight}px`,
                  opacity: loadedUrls.has(src) ? 1 : 0,
                  transition: 'opacity 200ms ease',
                  transitionDelay: loadedUrls.has(src) ? '200ms' : '0ms',
                }}
                loading='lazy'
                onLoad={(event) => {
                  const nextDimensions = {
                    width: event.currentTarget.naturalWidth,
                    height: event.currentTarget.naturalHeight,
                  };
                  setLoadedDimensions((prev) => ({
                    ...prev,
                    [src]: nextDimensions,
                  }));
                  setLoadedUrls((prev) => new Set(prev).add(src));
                  const progressTimer = window.setInterval(
                    () => onRevealProgress?.(),
                    16,
                  );
                  window.setTimeout(
                    () => window.clearInterval(progressTimer),
                    260,
                  );
                  onMediaLoad?.(src, nextDimensions);
                }}
              />
            </button>
          );
        })}
      </div>
    </div>
  );
};

export default PlaygroundGeneratedImageGallery;
