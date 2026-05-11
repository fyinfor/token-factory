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

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Button } from '@douyinfe/semi-ui';
import { ChevronRight } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';

const INTERVAL_MS = 10000;

function parseSlides(raw) {
  if (!raw || typeof raw !== 'string') return [];
  try {
    const v = JSON.parse(raw);
    return Array.isArray(v) ? v : [];
  } catch {
    return [];
  }
}

const HomeBannerCarousel = ({ rawSlides }) => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const slides = useMemo(() => parseSlides(rawSlides), [rawSlides]);
  const [index, setIndex] = useState(0);

  useEffect(() => {
    setIndex(0);
  }, [slides.length]);

  useEffect(() => {
    if (slides.length <= 1) return undefined;
    const id = window.setInterval(() => {
      setIndex((i) => (i + 1) % slides.length);
    }, INTERVAL_MS);
    return () => window.clearInterval(id);
  }, [slides.length]);

  const go = useCallback(
    (slide) => {
      const model = (slide?.target_model || '').trim();
      if (model) {
        navigate(`/pricing?model=${encodeURIComponent(model)}`);
      } else {
        navigate('/pricing');
      }
    },
    [navigate],
  );

  if (!slides.length) return null;

  const slide = slides[Math.min(index, slides.length - 1)] || {};
  const hasImage = Boolean(slide.image_url && String(slide.image_url).trim());

  return (
    <div className='w-full max-w-5xl mx-auto mb-8 px-0'>
      <div
        role='button'
        tabIndex={0}
        onClick={() => go(slide)}
        onKeyDown={(e) => {
          if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault();
            go(slide);
          }
        }}
        className='group relative w-full overflow-hidden rounded-2xl border border-semi-color-border shadow-md cursor-pointer transition-shadow hover:shadow-lg focus:outline-none focus-visible:ring-2 focus-visible:ring-semi-color-primary'
        style={{ minHeight: hasImage ? undefined : '160px' }}
      >
        {hasImage ? (
          <img
            src={slide.image_url}
            alt=''
            className='absolute inset-0 h-full w-full object-cover transition-transform duration-500 group-hover:scale-[1.02]'
            decoding='async'
          />
        ) : (
          <div
            className='absolute inset-0'
            style={{
              background:
                'linear-gradient(120deg, #ede9fe 0%, #e0e7ff 38%, #fce7f3 100%)',
            }}
          />
        )}

        <div
          className={`relative flex flex-col gap-3 p-5 md:flex-row md:items-center md:justify-between md:p-8 ${hasImage ? 'bg-gradient-to-r from-black/55 via-black/35 to-transparent md:min-h-[200px]' : ''}`}
        >
          <div className='max-w-xl text-left'>
            {slide.badge ? (
              <span
                className={`mb-2 inline-block rounded-full px-2.5 py-0.5 text-xs font-semibold ${
                  hasImage
                    ? 'bg-blue-500 text-white'
                    : 'bg-indigo-600 text-white'
                }`}
              >
                {slide.badge}
              </span>
            ) : null}
            <h2
              className={`text-lg font-bold leading-snug md:text-2xl ${
                hasImage ? 'text-white drop-shadow' : 'text-semi-color-text-0'
              }`}
            >
              {slide.title || t('首页广告默认标题')}
            </h2>
            {slide.subtitle ? (
              <p
                className={`mt-2 text-sm md:text-base ${
                  hasImage
                    ? 'text-white/90 drop-shadow'
                    : 'text-semi-color-text-2'
                }`}
              >
                {slide.subtitle}
              </p>
            ) : null}
          </div>
          <div className='flex shrink-0 items-center gap-3 md:flex-col md:items-end'>
            <Button
              type='primary'
              theme='solid'
              icon={<ChevronRight size={16} />}
              iconPosition='right'
              className={hasImage ? '!bg-violet-600 hover:!bg-violet-500' : ''}
              onClick={(e) => {
                e.stopPropagation();
                go(slide);
              }}
            >
              {slide.button_text || t('立即体验')}
            </Button>
          </div>
        </div>
      </div>

      {slides.length > 1 ? (
        <div className='mt-3 flex justify-center gap-2'>
          {slides.map((_, i) => (
            <button
              type='button'
              key={i}
              className={`h-2 w-2 rounded-full transition-all ${
                i === index
                  ? 'w-6 bg-semi-color-primary'
                  : 'bg-semi-color-fill-2 hover:bg-semi-color-fill-1'
              }`}
              aria-label={`${i + 1}`}
              onClick={() => setIndex(i)}
            />
          ))}
        </div>
      ) : null}
    </div>
  );
};

export default HomeBannerCarousel;
