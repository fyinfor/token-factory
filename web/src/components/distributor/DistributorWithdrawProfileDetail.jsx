/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import React from 'react';
import { Image, Typography } from '@douyinfe/semi-ui';
import { IconFile } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import {
  parseWithdrawRow,
  parseVoucherUrls,
  wdAccountTypeLabel,
  WD_ACCOUNT_ENTERPRISE,
} from './withdrawProfileUtils';

const { Text } = Typography;

function isPdfUrl(u) {
  return /\.pdf(\?|$)/i.test(u || '');
}

function fileNameFromUrl(url, fallback) {
  try {
    const path = new URL(url, window.location.origin).pathname;
    const base = path.split('/').filter(Boolean).pop();
    if (base) return decodeURIComponent(base);
  } catch {
    // ignore
  }
  return fallback;
}

/** 触发浏览器下载（跨域 OSS 先拉取 blob，失败则新窗口打开） */
async function downloadUrlInBrowser(url, filename) {
  const u = String(url || '').trim();
  if (!u) return;
  const name = filename || 'download';
  try {
    const res = await fetch(u, { mode: 'cors' });
    if (!res.ok) throw new Error('fetch failed');
    const blob = await res.blob();
    const blobUrl = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = blobUrl;
    a.download = name;
    document.body.appendChild(a);
    a.click();
    a.remove();
    URL.revokeObjectURL(blobUrl);
  } catch {
    const a = document.createElement('a');
    a.href = u;
    a.download = name;
    a.target = '_blank';
    a.rel = 'noopener noreferrer';
    document.body.appendChild(a);
    a.click();
    a.remove();
  }
}

function FieldRow({ label, value }) {
  const v = value != null ? String(value).trim() : '';
  if (!v) return null;
  return (
    <div className='flex flex-col gap-0.5 sm:flex-row sm:gap-2'>
      <Text type='tertiary' size='small' className='sm:w-36 flex-shrink-0'>
        {label}
      </Text>
      <Text size='small' className='break-all'>
        {v}
      </Text>
    </div>
  );
}

function DocLink({ label, url, onPreview }) {
  const u = String(url || '').trim();
  if (!u) return null;
  const isPdf = isPdfUrl(u);
  return (
    <div className='flex items-center gap-2'>
      <Text type='tertiary' size='small' className='w-36 flex-shrink-0'>
        {label}
      </Text>
      <button
        type='button'
        className='text-sm text-[var(--semi-color-link)] hover:underline'
        onClick={() => {
          if (isPdf) window.open(u, '_blank', 'noopener,noreferrer');
          else onPreview?.(u);
        }}
      >
        {isPdf ? 'PDF' : label}
      </button>
    </div>
  );
}

/** 发票：方形缩略图，点击浏览器下载 */
function DocInvoiceThumb({ label, url }) {
  const { t } = useTranslation();
  const u = String(url || '').trim();
  if (!u) return null;
  const pdf = isPdfUrl(u);
  const fileName = fileNameFromUrl(u, pdf ? 'invoice.pdf' : 'invoice');

  return (
    <div className='inline-flex w-28 flex-col gap-1.5'>
      <Text type='tertiary' size='small' className='leading-tight'>
        {label}
      </Text>
      <button
        type='button'
        className='relative h-28 w-28 shrink-0 cursor-pointer overflow-hidden rounded-lg border border-[var(--semi-color-border)] bg-[var(--semi-color-fill-0)] p-0 focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--semi-color-primary)]'
        onClick={() => downloadUrlInBrowser(u, fileName)}
        title={t('点击下载')}
      >
        {pdf ? (
          <div className='flex h-full w-full flex-col items-center justify-center'>
            <IconFile size='large' />
            <span className='mt-1 text-xs text-[var(--semi-color-text-2)]'>
              PDF
            </span>
          </div>
        ) : (
          <Image
            src={u}
            alt={label}
            preview={false}
            width='100%'
            height='100%'
            imgStyle={{ objectFit: 'cover' }}
          />
        )}
      </button>
    </div>
  );
}

/** 证件/照片：方形缩略图，点击放大 */
function DocImagePreview({ label, url, onPreview }) {
  const { t } = useTranslation();
  const u = String(url || '').trim();
  if (!u) return null;
  if (isPdfUrl(u)) {
    return <DocLink label={label} url={u} onPreview={onPreview} />;
  }
  return (
    <div className='inline-flex w-28 flex-col gap-1.5'>
      <Text type='tertiary' size='small' className='leading-tight'>
        {label}
      </Text>
      <button
        type='button'
        className='relative h-28 w-28 shrink-0 cursor-zoom-in overflow-hidden rounded-lg border border-[var(--semi-color-border)] bg-[var(--semi-color-fill-0)] p-0 focus:outline-none focus-visible:ring-2 focus-visible:ring-[var(--semi-color-primary)]'
        onClick={() => onPreview?.(u)}
        title={t('点击放大')}
      >
        <Image
          src={u}
          alt={label}
          preview={false}
          width='100%'
          height='100%'
          imgStyle={{ objectFit: 'cover' }}
        />
      </button>
    </div>
  );
}

export default function DistributorWithdrawProfileDetail({ row, onImagePreview }) {
  const { t } = useTranslation();
  const f = parseWithdrawRow(row);
  const legacyVouchers = parseVoucherUrls(row?.voucher_urls);
  const accountType = f.account_type;

  return (
    <div className='space-y-4 text-sm'>
      <Text strong>
        {t('账户类型')}：{wdAccountTypeLabel(t, accountType)}
      </Text>

      {accountType === WD_ACCOUNT_ENTERPRISE ? (
        <>
          <FieldRow label={t('企业名称')} value={f.real_name} />
          <FieldRow label={t('统一社会信用代码')} value={f.credit_code} />
          <FieldRow label={t('法人姓名')} value={f.legal_person_name} />
          <FieldRow label={t('法人手机号')} value={f.legal_person_phone} />
          <FieldRow label={t('对公卡号')} value={f.bank_account} />
          <FieldRow label={t('开户行')} value={f.bank_name} />
          <FieldRow label={t('联行号')} value={f.bank_branch_code} />
          <FieldRow label={t('联系人')} value={f.contact_person} />
          <div className='flex flex-wrap gap-4 pt-1'>
            <DocImagePreview
              label={t('营业执照（需加盖公章）')}
              url={f.business_license_url}
              onPreview={onImagePreview}
            />
            <DocImagePreview
              label={t('法人身份证（需加盖公章）')}
              url={f.legal_person_id_card_url}
              onPreview={onImagePreview}
            />
            <DocImagePreview
              label={t('对公账户证明（需加盖公章）')}
              url={f.corporate_account_proof_url}
              onPreview={onImagePreview}
            />
            <DocInvoiceThumb label={t('发票')} url={f.invoice_url} />
          </div>
        </>
      ) : (
        <>
          <FieldRow label={t('姓名')} value={f.real_name} />
          <FieldRow label={t('身份证号')} value={f.id_card_no} />
          <FieldRow label={t('身份证有效期')} value={f.id_card_expiry} />
          <FieldRow label={t('手机号')} value={f.mobile} />
          <FieldRow label={t('银行卡号')} value={f.bank_account} />
          <FieldRow label={t('开户行')} value={f.bank_name} />
          <FieldRow label={t('银行预留手机号')} value={f.bank_reserved_phone} />
          <div className='flex flex-wrap gap-4 pt-1'>
            <DocImagePreview
              label={t('身份证正面')}
              url={f.id_card_front_url}
              onPreview={onImagePreview}
            />
            <DocImagePreview
              label={t('身份证反面')}
              url={f.id_card_back_url}
              onPreview={onImagePreview}
            />
            <DocImagePreview
              label={t('银行卡照')}
              url={f.bank_card_photo_url}
              onPreview={onImagePreview}
            />
            <DocInvoiceThumb label={t('发票')} url={f.invoice_url} />
          </div>
        </>
      )}

      {legacyVouchers.length > 0 ? (
        <div>
          <Text type='tertiary' size='small' className='block mb-1'>
            {t('历史票据')}
          </Text>
          <div className='flex flex-wrap gap-2'>
            {legacyVouchers.map((u, i) => (
              <button
                key={`lv-${i}`}
                type='button'
                className='text-xs text-[var(--semi-color-link)]'
                onClick={() => {
                  if (isPdfUrl(u)) window.open(u, '_blank');
                  else onImagePreview?.(u);
                }}
              >
                {t('附件')} {i + 1}
              </button>
            ))}
          </div>
        </div>
      ) : null}
    </div>
  );
}
