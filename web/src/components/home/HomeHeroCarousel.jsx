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

import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { ChevronLeft, ChevronRight } from 'lucide-react';
import { useTranslation } from 'react-i18next';

const DEFAULT_INTERVAL_SEC = 5;
const MIN_INTERVAL_SEC = 2;
const MAX_INTERVAL_SEC = 60;
const MANUAL_PAUSE_MS = 30000;
const DEFAULT_OVERLAY_OPACITY = 0.15;

function parseSlides(raw) {
  if (Array.isArray(raw)) {
    return raw;
  }
  if (!raw || typeof raw !== 'string') {
    return [];
  }
  try {
    const value = JSON.parse(raw);
    return Array.isArray(value) ? value : [];
  } catch {
    return [];
  }
}

function clampIntervalSec(value) {
  const n = Number(value);
  if (!Number.isFinite(n)) {
    return DEFAULT_INTERVAL_SEC;
  }
  return Math.min(MAX_INTERVAL_SEC, Math.max(MIN_INTERVAL_SEC, Math.round(n)));
}

function clampOpacity(value) {
  const n = Number(value);
  if (!Number.isFinite(n)) {
    return DEFAULT_OVERLAY_OPACITY;
  }
  return Math.min(0.8, Math.max(0, n));
}

function normalizeContentAlign(value) {
  return ['left', 'center', 'right'].includes(value) ? value : 'left';
}

function contentAlignLayout(value) {
  const align = normalizeContentAlign(value);
  if (align === 'center') {
    return {
      containerClass: 'justify-center',
      contentClass: 'items-center text-center',
    };
  }
  if (align === 'right') {
    return {
      containerClass: 'justify-end',
      contentClass: 'items-end text-right',
    };
  }
  return {
    containerClass: 'justify-start',
    contentClass: 'items-start text-left',
  };
}

function normalizeSlide(slide, index) {
  const enabled =
    slide?.enabled === false ||
    slide?.enabled === 'false' ||
    slide?.status === 0 ||
    slide?.status === '0'
      ? false
      : true;
  const legacyImage = String(slide?.image_url || '').trim();

  return {
    id: String(slide?.id || index).trim(),
    enabled,
    sort: Number(slide?.sort) || index + 1,
    title: String(slide?.title || '').trim(),
    subtitle: String(slide?.subtitle || '').trim(),
    badgeText: String(slide?.badge_text || slide?.badge || '').trim(),
    buttonText: String(slide?.button_text || '').trim(),
    contentAlign: normalizeContentAlign(slide?.content_align),
    linkUrl: String(slide?.link_url || '').trim(),
    openMode: slide?.open_mode === 'blank' ? 'blank' : 'same',
    pcImage: String(slide?.img_pc || legacyImage).trim(),
    mobileImage: String(slide?.img_mobile || '').trim(),
    overlayOpacity: clampOpacity(slide?.overlay_opacity),
    textColor: String(slide?.text_color || '#ffffff').trim(),
    titleColor: String(
      slide?.title_color || slide?.text_color || '#ffffff',
    ).trim(),
    subtitleColor: String(
      slide?.subtitle_color || slide?.text_color || '#e5e7eb',
    ).trim(),
    buttonColor: String(slide?.button_color || '#ffffff').trim(),
    buttonTextColor: String(slide?.button_text_color || '#111827').trim(),
    backgroundColor: String(slide?.background_color || '#111827').trim(),
  };
}

function hasRenderableContent(slide) {
  return Boolean(
    slide.enabled &&
    (slide.pcImage ||
      slide.mobileImage ||
      slide.title ||
      slide.subtitle ||
      slide.badgeText ||
      slide.buttonText),
  );
}

export default function HomeHeroCarousel({ enabled, rawSlides, intervalSec }) {
  const { t } = useTranslation();
  const slides = useMemo(
    () =>
      parseSlides(rawSlides)
        .map(normalizeSlide)
        .filter(hasRenderableContent)
        .sort((a, b) => a.sort - b.sort),
    [rawSlides],
  );
  const [index, setIndex] = useState(0);
  const pauseUntilRef = useRef(0);
  const multi = slides.length > 1;
  const intervalMs = clampIntervalSec(intervalSec) * 1000;
  const activeSlide = slides[index] || slides[0];

  useEffect(() => {
    setIndex(0);
    pauseUntilRef.current = 0;
  }, [slides.length]);

  useEffect(() => {
    if (!enabled || !multi) {
      return undefined;
    }
    const id = window.setInterval(() => {
      if (Date.now() < pauseUntilRef.current) {
        return;
      }
      setIndex((i) => (i + 1) % slides.length);
    }, intervalMs);
    return () => window.clearInterval(id);
  }, [enabled, multi, slides.length, intervalMs]);

  const pauseAfterManual = useCallback(() => {
    pauseUntilRef.current = Date.now() + MANUAL_PAUSE_MS;
  }, []);

  const openLink = useCallback(() => {
    if (!activeSlide?.linkUrl) {
      return;
    }
    if (activeSlide.openMode === 'blank') {
      window.open(activeSlide.linkUrl, '_blank', 'noopener,noreferrer');
      return;
    }
    window.location.assign(activeSlide.linkUrl);
  }, [activeSlide?.linkUrl, activeSlide?.openMode]);

  if (!enabled || slides.length === 0) {
    return null;
  }

  const overlayOpacity = activeSlide?.overlayOpacity ?? DEFAULT_OVERLAY_OPACITY;
  const alignLayout = contentAlignLayout(activeSlide?.contentAlign);

  return (
    <section
      className='home-hero-carousel relative z-0 -mt-20 mb-6 h-[360px] overflow-hidden md:-mt-24 md:h-[clamp(360px,31.25vw,600px)]'
      style={{
        width: '100vw',
        marginLeft: 'calc(50% - 50vw)',
        marginRight: 'calc(50% - 50vw)',
        backgroundColor: activeSlide?.backgroundColor || '#111827',
      }}
      aria-label={activeSlide?.title || t('首页主轮播')}
    >
      <div
        className={`group/hero absolute inset-0 ${activeSlide?.linkUrl ? 'cursor-pointer' : ''}`}
        role={activeSlide?.linkUrl ? 'link' : undefined}
        tabIndex={activeSlide?.linkUrl ? 0 : undefined}
        onClick={openLink}
        onKeyDown={(event) => {
          if (
            activeSlide?.linkUrl &&
            (event.key === 'Enter' || event.key === ' ')
          ) {
            event.preventDefault();
            openLink();
          }
        }}
      >
        {slides.map((slide, i) => {
          const pcImage = slide.pcImage || slide.mobileImage;
          const mobileImage = slide.mobileImage || slide.pcImage;
          return (
            <React.Fragment key={`${slide.id}-${i}`}>
              {pcImage ? (
                <img
                  src={pcImage}
                  alt=''
                  className='absolute inset-0 hidden h-full w-full object-cover object-center transition-opacity duration-500 md:block'
                  style={{
                    backgroundColor: slide.backgroundColor,
                    opacity: i === index ? 1 : 0,
                  }}
                  aria-hidden={i !== index}
                  loading={i === 0 ? 'eager' : 'lazy'}
                  decoding='async'
                  draggable={false}
                />
              ) : null}
              {mobileImage ? (
                <img
                  src={mobileImage}
                  alt=''
                  className='absolute inset-0 h-full w-full object-cover object-center transition-opacity duration-500 md:hidden'
                  style={{
                    backgroundColor: slide.backgroundColor,
                    opacity: i === index ? 1 : 0,
                  }}
                  aria-hidden={i !== index}
                  loading={i === 0 ? 'eager' : 'lazy'}
                  decoding='async'
                  draggable={false}
                />
              ) : null}
            </React.Fragment>
          );
        })}

        <div
          className='absolute inset-0 z-[1] transition-colors duration-500'
          style={{
            backgroundColor: `rgba(0,0,0,${overlayOpacity})`,
          }}
          aria-hidden
        />

        <div
          className={`relative z-[2] mx-auto flex h-full max-w-[1200px] items-center px-5 pb-16 pt-20 md:px-8 md:pb-20 md:pt-24 ${alignLayout.containerClass}`}
        >
          <div
            className={`flex max-w-[620px] -translate-y-3 flex-col md:-translate-y-5 ${alignLayout.contentClass}`}
            style={{
              color:
                activeSlide.titleColor || activeSlide.textColor || '#ffffff',
            }}
          >
            {activeSlide.badgeText ? (
              <div className='mb-4 inline-flex rounded-full border border-white/35 bg-white/10 px-3.5 py-1 text-xs font-semibold backdrop-blur-sm md:text-sm'>
                {activeSlide.badgeText}
              </div>
            ) : null}
            {activeSlide.title ? (
              <h1
                className='m-0 text-[2rem] font-semibold leading-[1.08] md:text-[3.25rem]'
                style={{
                  color:
                    activeSlide.titleColor ||
                    activeSlide.textColor ||
                    '#ffffff',
                  textShadow: '0 2px 18px rgba(0,0,0,0.35)',
                }}
              >
                {activeSlide.title}
              </h1>
            ) : null}
            {activeSlide.subtitle ? (
              <p
                className='mt-3 max-w-[560px] text-[0.95rem] leading-relaxed opacity-90 md:mt-5 md:text-lg'
                style={{
                  color:
                    activeSlide.subtitleColor ||
                    activeSlide.textColor ||
                    '#e5e7eb',
                  textShadow: '0 1px 12px rgba(0,0,0,0.28)',
                }}
              >
                {activeSlide.subtitle}
              </p>
            ) : null}
            {activeSlide.buttonText && activeSlide.linkUrl ? (
              <button
                type='button'
                className='mt-5 inline-flex min-h-10 items-center justify-center rounded-md px-5 text-sm font-semibold shadow-[0_10px_30px_rgba(0,0,0,0.18)] transition hover:-translate-y-0.5 hover:brightness-95 md:mt-7 md:min-h-11 md:px-6 md:text-base'
                style={{
                  backgroundColor: activeSlide.buttonColor || '#ffffff',
                  color: activeSlide.buttonTextColor || '#111827',
                }}
                onClick={(event) => {
                  event.stopPropagation();
                  openLink();
                }}
              >
                {activeSlide.buttonText}
              </button>
            ) : null}
          </div>
        </div>

        {multi ? (
          <>
            <button
              type='button'
              className='absolute left-4 top-1/2 z-[5] flex h-11 w-11 -translate-y-1/2 items-center justify-center rounded-full bg-[#f6f7f9] text-[#263040] opacity-95 shadow-[0_12px_30px_rgba(15,23,42,0.22)] transition hover:scale-105 hover:bg-white hover:shadow-[0_16px_36px_rgba(15,23,42,0.26)] md:opacity-0 md:group-hover/hero:opacity-100 md:focus-visible:opacity-100'
              aria-label={t('上一张')}
              onClick={(event) => {
                event.stopPropagation();
                pauseAfterManual();
                setIndex((i) => (i - 1 + slides.length) % slides.length);
              }}
            >
              <ChevronLeft
                size={26}
                strokeWidth={2.4}
                className='text-current'
                aria-hidden
              />
            </button>
            <button
              type='button'
              className='absolute right-4 top-1/2 z-[5] flex h-11 w-11 -translate-y-1/2 items-center justify-center rounded-full bg-[#f6f7f9] text-[#263040] opacity-95 shadow-[0_12px_30px_rgba(15,23,42,0.22)] transition hover:scale-105 hover:bg-white hover:shadow-[0_16px_36px_rgba(15,23,42,0.26)] md:opacity-0 md:group-hover/hero:opacity-100 md:focus-visible:opacity-100'
              aria-label={t('下一张')}
              onClick={(event) => {
                event.stopPropagation();
                pauseAfterManual();
                setIndex((i) => (i + 1) % slides.length);
              }}
            >
              <ChevronRight
                size={26}
                strokeWidth={2.4}
                className='text-current'
                aria-hidden
              />
            </button>
            <div
              className='absolute bottom-5 left-1/2 z-[6] flex -translate-x-1/2 items-center gap-2.5 rounded-full px-3.5 py-2.5'
              style={{
                backgroundColor: 'rgba(255, 255, 255, 0.62)',
                boxShadow: '0 8px 22px rgba(15, 23, 42, 0.14)',
                backdropFilter: 'blur(14px)',
                WebkitBackdropFilter: 'blur(14px)',
              }}
              onClick={(event) => event.stopPropagation()}
            >
              {slides.map((_, i) => (
                <button
                  key={i}
                  type='button'
                  className={`h-3 rounded-full transition-all ${
                    i === index
                      ? 'w-8 bg-[#f59e0b] shadow-[0_2px_8px_rgba(245,158,11,0.45)] hover:bg-[#fbbf24]'
                      : 'w-3 bg-white/40 shadow-[inset_0_0_0_1px_rgba(255,255,255,0.42),0_2px_7px_rgba(15,23,42,0.12)] backdrop-blur-sm hover:bg-white/60'
                  }`}
                  aria-label={`${i + 1} / ${slides.length}`}
                  onClick={(event) => {
                    event.stopPropagation();
                    pauseAfterManual();
                    setIndex(i);
                  }}
                />
              ))}
            </div>
          </>
        ) : null}
      </div>
    </section>
  );
}
