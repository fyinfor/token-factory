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

function parseSlides(raw) {
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

function normalizeSlide(slide) {
  return {
    imageUrl: String(slide?.image_url || '').trim(),
    linkUrl: String(slide?.link_url || '').trim(),
  };
}

export default function HomeHeroCarousel({ enabled, rawSlides, intervalSec }) {
  const { t } = useTranslation();
  const slides = useMemo(
    () =>
      parseSlides(rawSlides)
        .map(normalizeSlide)
        .filter((slide) => slide.imageUrl),
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
    window.location.assign(activeSlide.linkUrl);
  }, [activeSlide?.linkUrl]);

  if (!enabled || slides.length === 0) {
    return null;
  }

  return (
    <section
      className='relative -mt-6 mb-6 bg-black md:-mt-10'
      style={{
        width: '100vw',
        marginLeft: 'calc(50% - 50vw)',
        marginRight: 'calc(50% - 50vw)',
      }}
    >
      <div
        className={`relative w-full overflow-hidden bg-black ${
          activeSlide?.linkUrl ? 'cursor-pointer' : ''
        }`}
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
        <img
          src={activeSlide.imageUrl}
          alt=''
          className='invisible block h-auto w-full select-none'
          aria-hidden
          decoding='async'
        />

        {slides.map((slide, i) => (
          <img
            key={`${slide.imageUrl}-${i}`}
            src={slide.imageUrl}
            alt=''
            className='absolute inset-0 h-full w-full object-contain object-center transition-opacity duration-500'
            style={{
              opacity: i === index ? 1 : 0,
              pointerEvents: i === index ? 'auto' : 'none',
            }}
            aria-hidden={i !== index}
            decoding='async'
          />
        ))}
      </div>

      {multi ? (
        <>
          <button
            type='button'
            className='absolute left-3 top-1/2 z-[3] flex h-9 w-9 -translate-y-1/2 items-center justify-center rounded-full bg-black/28 text-white backdrop-blur-md transition hover:bg-black/40'
            aria-label={t('上一张')}
            onClick={(event) => {
              event.stopPropagation();
              pauseAfterManual();
              setIndex((i) => (i - 1 + slides.length) % slides.length);
            }}
          >
            <ChevronLeft size={20} aria-hidden />
          </button>
          <button
            type='button'
            className='absolute right-3 top-1/2 z-[3] flex h-9 w-9 -translate-y-1/2 items-center justify-center rounded-full bg-black/28 text-white backdrop-blur-md transition hover:bg-black/40'
            aria-label={t('下一张')}
            onClick={(event) => {
              event.stopPropagation();
              pauseAfterManual();
              setIndex((i) => (i + 1) % slides.length);
            }}
          >
            <ChevronRight size={20} aria-hidden />
          </button>
          <div className='absolute bottom-4 right-5 z-[3] flex gap-2'>
            {slides.map((_, i) => (
              <button
                key={i}
                type='button'
                className={`h-1.5 rounded-full transition-all ${
                  i === index ? 'w-6 bg-white' : 'w-2 bg-white/55'
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
    </section>
  );
}
