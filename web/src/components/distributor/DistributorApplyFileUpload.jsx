/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import React, { useState } from 'react';
import { Button, Progress, Typography, Upload } from '@douyinfe/semi-ui';
import { IconClose, IconFile } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../helpers';
import { useSmoothUploadProgress } from './useSmoothUploadProgress';

const { Text } = Typography;

function ApplyRequiredLabel({ children, className = '' }) {
  return (
    <Text
      strong
      className={`text-[var(--semi-color-text-0)] ${className}`.trim()}
    >
      <span
        className='text-[var(--semi-color-danger)] mr-1 font-normal'
        aria-hidden
      >
        *
      </span>
      {children}
    </Text>
  );
}

function isPdfUrl(u) {
  return /\.pdf(\?|$)/i.test(u || '');
}

function isImageFile(fileInstance) {
  const type = fileInstance?.type || '';
  const name = (fileInstance?.name || '').toLowerCase();
  return (
    type === 'image/jpeg' ||
    type === 'image/png' ||
    /\.(jpe?g|png)$/.test(name)
  );
}

function isPdfFile(fileInstance) {
  const type = fileInstance?.type || '';
  const name = (fileInstance?.name || '').toLowerCase();
  return type === 'application/pdf' || name.endsWith('.pdf');
}

const thumbRemoveBtnClass =
  'absolute -right-1.5 -top-1.5 z-10 inline-flex h-[18px] w-[18px] shrink-0 cursor-pointer items-center justify-center rounded-full border-0 bg-[var(--semi-color-danger)] p-0 text-white opacity-100 shadow-[0_1px_4px_rgba(0,0,0,0.25)] transition-colors hover:brightness-90 focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--semi-color-danger)] [&_.semi-icon]:!m-0';

function UploadThumbRemoveButton({ onClick, label }) {
  return (
    <button
      type='button'
      aria-label={label}
      className={thumbRemoveBtnClass}
      onClick={onClick}
    >
      <IconClose size='small' className='!text-[10px]' />
    </button>
  );
}

/**
 * 与 /console/distributor/apply 资格证书相同的上传：进度条 + 缩略图 + 删除。
 * 单文件：传 url + onUrlChange + maxCount=1；多文件：urls + onUrlsChange。
 */
export default function DistributorApplyFileUpload({
  label,
  labelExtra,
  required = false,
  url = '',
  onUrlChange,
  urls: urlsProp,
  onUrlsChange,
  maxCount = 1,
  multiple,
  hint,
  onPreview,
  disabled: disabledProp,
  compact = false,
  imagesOnly = false,
  allowPdf = false,
}) {
  const { t } = useTranslation();
  const [uploadPct, setUploadPct] = useState(null);
  const uploadDisplayPct = useSmoothUploadProgress(uploadPct);

  const isSingle = typeof onUrlChange === 'function';
  const urls = isSingle
    ? String(url || '').trim()
      ? [String(url).trim()]
      : []
    : urlsProp || [];

  const setUrls = (next) => {
    if (isSingle) {
      onUrlChange(next[0] || '');
    } else {
      onUrlsChange?.(next);
    }
  };

  const limit = maxCount;
  const isMultiple = multiple ?? limit > 1;
  const atLimit = urls.length >= limit;
  const disabled = disabledProp ?? atLimit;

  const defaultHint = imagesOnly
    ? null
    : allowPdf
      ? t('支持 JPG/PNG 或 PDF')
      : isMultiple
        ? t('支持图片或 PDF，最多 {{n}} 个；点击图片可大图预览', { n: limit })
        : t('支持图片或 PDF；删除后可重新上传');
  const hintText = hint !== undefined ? hint : defaultHint;
  const acceptAttr = imagesOnly
    ? '.jpg,.jpeg,.png'
    : allowPdf
      ? '.jpg,.jpeg,.png,.pdf'
      : 'image/*,.pdf';

  const customRequest = async ({ file, onSuccess, onError, onProgress }) => {
    const inst = file.fileInstance || file;
    if (imagesOnly) {
      if (!isImageFile(inst)) {
        showError(t('只支持 JPG/PNG 格式的图片'));
        onError();
        return;
      }
    } else if (allowPdf) {
      if (!isImageFile(inst) && !isPdfFile(inst)) {
        showError(t('只支持 JPG/PNG/PDF 格式的文件'));
        onError();
        return;
      }
    }

    const fd = new FormData();
    fd.append('file', inst);
    fd.append('purpose', 'distributor');
    setUploadPct(0);
    try {
      const res = await API.post('/api/oss/upload', fd, {
        skipErrorHandler: true,
        onUploadProgress: (ev) => {
          const total = ev.total || ev.loaded || 1;
          const raw = Math.round((ev.loaded * 100) / total);
          const pct = Math.min(99, raw);
          setUploadPct(pct);
          if (typeof onProgress === 'function') {
            onProgress({ total, loaded: ev.loaded });
          }
        },
      });
      const { success, message, data } = res.data || {};
      if (!success || !data?.url) {
        onError(new Error(message || 'upload'));
        showError(message || t('上传失败'));
        return;
      }
      setUploadPct(100);
      const uploaded = String(data.url).trim();
      setUrls(isMultiple ? [...urls, uploaded] : [uploaded]);
      onSuccess(data);
      showSuccess(t('已上传'));
    } catch (e) {
      onError(e);
      showError(e?.response?.data?.message || t('上传失败'));
    } finally {
      setUploadPct(null);
    }
  };

  return (
    <div className={compact ? 'mb-0' : 'mb-4'}>
      {label ? (
        required ? (
          <div className='mb-2 flex flex-wrap items-center gap-2'>
            <ApplyRequiredLabel>{label}</ApplyRequiredLabel>
            {labelExtra ? (
              <Text type='warning' size='small'>
                {labelExtra}
              </Text>
            ) : null}
          </div>
        ) : (
          <div className='mb-2 flex flex-wrap items-center gap-2'>
            <Text strong className='text-[var(--semi-color-text-0)]'>
              {label}
            </Text>
            {labelExtra ? (
              <Text type='warning' size='small'>
                {labelExtra}
              </Text>
            ) : null}
          </div>
        )
      ) : null}
      <Upload
        action=''
        accept={acceptAttr}
        showUploadList={false}
        customRequest={customRequest}
        limit={limit}
        multiple={isMultiple}
        disabled={disabled}
      >
        <Button disabled={disabled}>{t('上传文件')}</Button>
      </Upload>
      {uploadPct != null ? (
        <Progress
          percent={uploadDisplayPct ?? uploadPct}
          showInfo
          className='mt-2'
        />
      ) : null}
      {hintText ? (
        <Text type='tertiary' size='small' className='block mt-1'>
          {hintText}
        </Text>
      ) : null}
      {urls.length > 0 ? (
        <div className='mt-3 flex flex-wrap gap-3'>
          {urls.map((u, idx) =>
            isPdfUrl(u) ? (
              <div
                key={`${u}-${idx}`}
                className='relative flex h-24 w-24 flex-col items-center justify-center rounded-lg border border-[var(--semi-color-border)] bg-[var(--semi-color-fill-0)]'
              >
                <IconFile size='large' />
                <span className='mt-1 text-xs text-[var(--semi-color-text-2)]'>
                  PDF
                </span>
                <button
                  type='button'
                  className='absolute inset-0 rounded-lg focus:outline-none focus-visible:ring-2 focus-visible:ring-primary'
                  title={t('在新窗口打开')}
                  onClick={() =>
                    window.open(u, '_blank', 'noopener,noreferrer')
                  }
                />
                <UploadThumbRemoveButton
                  label={t('删除')}
                  onClick={(e) => {
                    e.stopPropagation();
                    setUrls(urls.filter((_, i) => i !== idx));
                  }}
                />
              </div>
            ) : (
              <div key={`${u}-${idx}`} className='relative h-24 w-24'>
                <button
                  type='button'
                  className='block h-full w-full cursor-zoom-in overflow-hidden rounded-lg border border-[var(--semi-color-border)] bg-[var(--semi-color-fill-0)] p-0'
                  onClick={() => {
                    if (onPreview) onPreview(u);
                    else window.open(u, '_blank', 'noopener,noreferrer');
                  }}
                >
                  <img src={u} alt='' className='h-full w-full object-cover' />
                </button>
                <UploadThumbRemoveButton
                  label={t('删除')}
                  onClick={(e) => {
                    e.stopPropagation();
                    setUrls(urls.filter((_, i) => i !== idx));
                  }}
                />
              </div>
            ),
          )}
        </div>
      ) : null}
    </div>
  );
}
