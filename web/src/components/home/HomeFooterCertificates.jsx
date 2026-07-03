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

import React, { useMemo, useState } from 'react';
import { Carousel, ImagePreview } from '@douyinfe/semi-ui';
import { IconExternalOpen } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';

const AUTO_SCROLL_MS = 4000;

function parseCertificates(raw) {
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

function normalizeCertificate(item) {
  return {
    title: String(item?.title || '').trim(),
    imageUrl: String(item?.image_url || '').trim(),
    linkUrl: String(item?.link_url || '').trim(),
  };
}

function isEnabled(value) {
  return value === true || value === 'true';
}

function CertificateCard({ certificate, index, t, onPreview }) {
  const title = certificate.title || `${t('证书')} #${index + 1}`;

  return (
    <div className='flex h-full w-full items-center justify-center px-10 pb-9'>
      <div className='relative flex w-[min(82vw,520px)] flex-col rounded-md border border-semi-color-border bg-semi-color-bg-1 p-1.5 shadow-sm'>
        <div className='flex min-h-9 items-center justify-center px-10 text-center text-base font-semibold leading-snug text-semi-color-text-0'>
          {title}
        </div>
        <button
          type='button'
          className='h-[240px] w-full cursor-zoom-in overflow-hidden rounded-sm border-0 bg-semi-color-fill-0 p-0 md:h-[300px]'
          aria-label={t('预览证书图片')}
          onClick={() => onPreview(certificate.imageUrl)}
        >
          <img
            src={certificate.imageUrl}
            alt={title}
            className='h-full w-full object-contain'
            loading='lazy'
            decoding='async'
          />
        </button>
        {certificate.linkUrl ? (
          <button
            type='button'
            className='absolute right-2 top-2 flex h-8 w-8 items-center justify-center rounded-md border-0 bg-transparent p-0 text-semi-color-text-2 transition-colors hover:bg-semi-color-fill-0 hover:text-semi-color-primary'
            title={t('打开证书链接')}
            aria-label={t('打开证书链接')}
            onClick={() =>
              window.open(certificate.linkUrl, '_blank', 'noopener,noreferrer')
            }
          >
            <IconExternalOpen aria-hidden />
          </button>
        ) : null}
      </div>
    </div>
  );
}

export default function HomeFooterCertificates({ enabled, rawCertificates }) {
  const { t } = useTranslation();
  const [previewUrl, setPreviewUrl] = useState('');
  const certificates = useMemo(
    () =>
      parseCertificates(rawCertificates)
        .map(normalizeCertificate)
        .filter((item) => item.imageUrl),
    [rawCertificates],
  );
  const multi = certificates.length > 1;

  if (!isEnabled(enabled) || certificates.length === 0) {
    return null;
  }

  return (
    <section className='w-full border-t border-semi-color-border bg-semi-color-bg-0 px-4 py-8'>
      <style>{`
        .home-cert-carousel {
          height: 350px;
        }
        @media (min-width: 768px) {
          .home-cert-carousel {
            height: 410px;
          }
        }
      `}</style>
      <div className='mx-auto max-w-6xl'>
        <div className='mx-auto mb-6 max-w-2xl text-center'>
          <div className='mb-2 text-xs font-semibold uppercase tracking-[0.18em] text-semi-color-primary'>
            Compliance
          </div>
          <h2 className='text-3xl font-semibold leading-tight text-semi-color-text-0 md:text-4xl'>
            {t('安全合规')}
          </h2>
          <p className='mt-3 text-sm leading-relaxed text-semi-color-text-2 md:text-base'>
            {t('资质证书公开展示，平台运营与服务保障更透明可信。')}
          </p>
        </div>
        <Carousel
          className='home-cert-carousel'
          animation='slide'
          arrowType='always'
          autoPlay={
            multi ? { interval: AUTO_SCROLL_MS, hoverToPause: true } : false
          }
          indicatorPosition='center'
          indicatorSize='small'
          indicatorType='dot'
          showArrow={multi}
          showIndicator={multi}
          slideDirection='left'
          speed={500}
          theme='primary'
          trigger='click'
        >
          {certificates.map((certificate, index) => (
            <div key={`${certificate.imageUrl}-${index}`}>
              <CertificateCard
                certificate={certificate}
                index={index}
                t={t}
                onPreview={setPreviewUrl}
              />
            </div>
          ))}
        </Carousel>
      </div>

      <ImagePreview
        src={previewUrl || ''}
        visible={Boolean(previewUrl)}
        onVisibleChange={(visible) => {
          if (!visible) {
            setPreviewUrl('');
          }
        }}
      />
    </section>
  );
}
