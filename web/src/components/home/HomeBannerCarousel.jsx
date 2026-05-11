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
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import HomeBannerIllustration from './HomeBannerIllustration';
import './model-ad-banner.css';

/** 自动轮播间隔（秒）— 与仪表盘说明一致可改为 4 */
const INTERVAL_MS = 4000;

const DEFAULT_BANNER_IMAGE = '/assets/banner-model.png';

function parseSlides(raw) {
  if (!raw || typeof raw !== 'string') return [];
  try {
    const v = JSON.parse(raw);
    return Array.isArray(v) ? v : [];
  } catch {
    return [];
  }
}

function oneLineTitle(raw) {
  if (!raw || typeof raw !== 'string') return '';
  return raw
    .split(/\n/)
    .map((s) => s.trim())
    .filter(Boolean)
    .join(' ');
}

function renderTitleNodes(line, highlight) {
  const hl = (highlight || '').trim();
  if (!hl || !line.includes(hl)) {
    return line;
  }
  const parts = line.split(hl);
  return parts.map((part, i) => (
    <React.Fragment key={i}>
      {part}
      {i < parts.length - 1 ? (
        <span className='ad-title-highlight'>{hl}</span>
      ) : null}
    </React.Fragment>
  ));
}

function SlideBanner({ slide, t, active, go }) {
  const titleLine = oneLineTitle(slide.title || t('首页广告默认标题'));
  const highlight = (slide.title_highlight || '').trim();
  const subtitle = (slide.subtitle || '').trim();
  const badge = (slide.badge ?? '').trim();
  const btnText = (slide.button_text || t('立即体验')).trim();
  const customImg = (slide.image_url || '').trim();
  const imgSrc = customImg || DEFAULT_BANNER_IMAGE;
  const [imgBroken, setImgBroken] = useState(false);

  useEffect(() => {
    setImgBroken(false);
  }, [imgSrc]);

  return (
    <section
      className='model-ad-banner model-ad-banner-slide'
      style={{
        opacity: active ? 1 : 0,
        pointerEvents: active ? 'auto' : 'none',
        zIndex: active ? 3 : 0,
      }}
      aria-hidden={!active}
      aria-label={`${titleLine}${subtitle ? `, ${subtitle}` : ''}`}
    >
      <div className='ad-banner-bg' aria-hidden>
        {!imgBroken ? (
          <img
            className='ad-banner-bg__img'
            src={imgSrc}
            alt=''
            onError={() => setImgBroken(true)}
            decoding='async'
          />
        ) : (
          <div className='ad-banner-bg-fallback'>
            <HomeBannerIllustration className='h-full max-h-[100px] w-[min(100%,220px)]' />
          </div>
        )}
      </div>
      <div className='ad-banner-scrim' aria-hidden />

      <div className='ad-content'>
        <div className='ad-copy'>
          <div
            className={`ad-head${badge ? ' ad-head--has-badge' : ' ad-head--no-badge'}`}
          >
            {badge ? <span className='ad-badge'>{badge}</span> : null}
            <h2 className='ad-title'>
              {renderTitleNodes(titleLine, highlight)}
            </h2>
            {subtitle ? (
              <p className='ad-subtitle'>{subtitle}</p>
            ) : null}
          </div>
        </div>

        <button
          className='ad-button'
          type='button'
          onClick={() => go(slide)}
        >
          {btnText}
          {' →'}
        </button>
      </div>
    </section>
  );
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

  return (
    <div className='model-ad-banner-root'>
      {slides.map((slide, i) => (
        <SlideBanner
          key={i}
          slide={slide}
          t={t}
          active={i === index}
          go={go}
        />
      ))}

      {slides.length > 1 ? (
        <div className='ad-dots' role='tablist' aria-label={t('首页广告轮播说明')}>
          {slides.map((_, i) => (
            <button
              key={i}
              type='button'
              className={`ad-dot${i === index ? ' active' : ''}`}
              aria-label={`${i + 1} / ${slides.length}`}
              aria-current={i === index ? 'true' : undefined}
              onClick={(e) => {
                e.stopPropagation();
                setIndex(i);
              }}
            />
          ))}
        </div>
      ) : null}
    </div>
  );
};

export default HomeBannerCarousel;
