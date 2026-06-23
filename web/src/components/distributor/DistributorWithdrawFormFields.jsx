/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import React, {
  forwardRef,
  useEffect,
  useImperativeHandle,
  useRef,
  useState,
} from 'react';
import { Divider, Input, Tag, Typography } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import DistributorWithdrawDocUpload from './DistributorWithdrawDocUpload';
import {
  WD_ACCOUNT_PERSONAL,
  WD_ACCOUNT_ENTERPRISE,
} from './withdrawProfileUtils';

const { Text } = Typography;

/** 法币提现金额：输入过程不钳制，失焦后再校验并格式化 */
const WithdrawFiatAmountInput = forwardRef(function WithdrawFiatAmountInput(
  { value, onChange, min, max, placeholder },
  ref,
) {
  const [draft, setDraft] = useState('');
  const focusedRef = useRef(false);

  useEffect(() => {
    if (focusedRef.current) return;
    if (value == null || value === '') {
      setDraft('');
    } else {
      setDraft(String(value));
    }
  }, [value]);

  const commit = () => {
    const trimmed = String(draft).trim();
    if (trimmed === '') {
      onChange(undefined);
      setDraft('');
      return undefined;
    }
    let n = Math.round(parseFloat(trimmed) * 100) / 100;
    if (Number.isNaN(n) || n <= 0) {
      onChange(undefined);
      setDraft('');
      return undefined;
    }
    if (min != null && n < min) n = min;
    if (max != null && n > max) n = max;
    onChange(n);
    setDraft(String(n));
    return n;
  };

  useImperativeHandle(ref, () => ({ commit }), [draft, min, max, onChange]);

  return (
    <Input
      style={{ width: '100%' }}
      value={draft}
      onFocus={() => {
        focusedRef.current = true;
      }}
      onChange={(v) => setDraft(v == null ? '' : String(v))}
      onBlur={() => {
        focusedRef.current = false;
        commit();
      }}
      onEnterPress={commit}
      placeholder={placeholder}
    />
  );
});

function FieldLabel({ required, children }) {
  return (
    <Text size='small' className='block mb-1.5 text-[var(--semi-color-text-0)]'>
      {required ? (
        <span className='text-[var(--semi-color-danger)] mr-1' aria-hidden>
          *
        </span>
      ) : null}
      {children}
    </Text>
  );
}

function FormField({ children }) {
  return <div className='min-w-0'>{children}</div>;
}

function FormSection({ title, children }) {
  return (
    <>
      <Divider margin='16px' align='left'>
        {title}
      </Divider>
      <div className='space-y-4'>{children}</div>
    </>
  );
}

function TextField({ label, required, value, onChange, placeholder }) {
  return (
    <FormField>
      <FieldLabel required={required}>{label}</FieldLabel>
      <Input
        value={value}
        onChange={(v) => onChange(String(v ?? ''))}
        placeholder={placeholder}
      />
    </FormField>
  );
}

function DocUploadField({
  label,
  labelExtra,
  url,
  onUrlChange,
  onPreview,
  imagesOnly = false,
  allowPdf = false,
}) {
  return (
    <FormField>
      <DistributorWithdrawDocUpload
        label={label}
        labelExtra={labelExtra}
        required
        url={url}
        onUrlChange={onUrlChange}
        onPreview={onPreview}
        compact
        imagesOnly={imagesOnly}
        allowPdf={allowPdf}
        hint={imagesOnly ? '' : undefined}
      />
    </FormField>
  );
}

export default function DistributorWithdrawFormFields({
  accountType,
  form,
  onFieldChange,
  onPreview,
  fiatCommitRef,
  isQuotaTokensMode,
  quotaInput,
  onQuotaInputChange,
  fiatAmount,
  onFiatAmountChange,
  fiatMin,
  fiatMax,
  affQuotaFloor,
  minQInternal,
  affQuota,
  renderQuota,
}) {
  const { t } = useTranslation();
  const set = onFieldChange;

  return (
    <div className='space-y-5'>
      <div className='flex flex-wrap items-center gap-2'>
        <Text size='small' className='text-[var(--semi-color-text-0)]'>
          {t('账户类型')}
        </Text>
        <Tag
          color={accountType === WD_ACCOUNT_ENTERPRISE ? 'blue' : 'green'}
          type='solid'
          size='large'
        >
          {accountType === WD_ACCOUNT_ENTERPRISE ? t('企业') : t('个人')}
        </Tag>
      </div>

      {accountType === WD_ACCOUNT_PERSONAL ? (
        <>
          <FormSection title={t('基本信息')}>
            <TextField
              label={t('姓名')}
              required
              value={form.real_name}
              onChange={(v) => set('real_name', v)}
              placeholder={t('姓名（必填）')}
            />
            <TextField
              label={t('身份证号')}
              required
              value={form.id_card_no}
              onChange={(v) => set('id_card_no', v)}
              placeholder={t('身份证号（必填）')}
            />
            <TextField
              label={t('身份证有效期')}
              required
              value={form.id_card_expiry}
              onChange={(v) => set('id_card_expiry', v)}
              placeholder={t('如 2030-12-31 或 长期')}
            />
            <TextField
              label={t('手机号')}
              required
              value={form.mobile}
              onChange={(v) => set('mobile', v)}
              placeholder={t('手机号（必填）')}
            />
          </FormSection>
          <FormSection title={t('银行信息')}>
            <TextField
              label={t('银行卡号')}
              required
              value={form.bank_account}
              onChange={(v) => set('bank_account', v)}
              placeholder={t('银行卡号（必填）')}
            />
            <TextField
              label={t('开户行')}
              required
              value={form.bank_name}
              onChange={(v) => set('bank_name', v)}
              placeholder={t('开户行（必填）')}
            />
            <TextField
              label={t('银行预留手机号')}
              required
              value={form.bank_reserved_phone}
              onChange={(v) => set('bank_reserved_phone', v)}
              placeholder={t('银行预留手机号（必填）')}
            />
          </FormSection>
          <FormSection title={t('证件照片')}>
            <DocUploadField
              label={t('身份证正面')}
              url={form.id_card_front_url}
              onUrlChange={(u) => set('id_card_front_url', u)}
              onPreview={onPreview}
              imagesOnly
            />
            <DocUploadField
              label={t('身份证反面')}
              url={form.id_card_back_url}
              onUrlChange={(u) => set('id_card_back_url', u)}
              onPreview={onPreview}
              imagesOnly
            />
            <DocUploadField
              label={t('银行卡照')}
              url={form.bank_card_photo_url}
              onUrlChange={(u) => set('bank_card_photo_url', u)}
              onPreview={onPreview}
              imagesOnly
            />
          </FormSection>
          <FormSection title={t('发票')}>
            <DocUploadField
              label={t('发票')}
              url={form.invoice_url}
              onUrlChange={(u) => set('invoice_url', u)}
              onPreview={onPreview}
              allowPdf
            />
          </FormSection>
        </>
      ) : (
        <>
          <FormSection title={t('企业信息')}>
            <TextField
              label={t('企业名称')}
              required
              value={form.real_name}
              onChange={(v) => set('real_name', v)}
              placeholder={t('企业名称（必填）')}
            />
            <TextField
              label={t('统一社会信用代码')}
              required
              value={form.credit_code}
              onChange={(v) => set('credit_code', v)}
              placeholder={t('统一社会信用代码（必填）')}
            />
            <TextField
              label={t('法人姓名')}
              required
              value={form.legal_person_name}
              onChange={(v) => set('legal_person_name', v)}
              placeholder={t('法人姓名（必填）')}
            />
            <TextField
              label={t('法人手机号')}
              required
              value={form.legal_person_phone}
              onChange={(v) => set('legal_person_phone', v)}
              placeholder={t('法人手机号（必填）')}
            />
            <TextField
              label={t('对公卡号')}
              required
              value={form.bank_account}
              onChange={(v) => set('bank_account', v)}
              placeholder={t('对公卡号（必填）')}
            />
            <TextField
              label={t('开户行')}
              required
              value={form.bank_name}
              onChange={(v) => set('bank_name', v)}
              placeholder={t('开户行（必填）')}
            />
            <TextField
              label={t('联行号')}
              required
              value={form.bank_branch_code}
              onChange={(v) => set('bank_branch_code', v)}
              placeholder={t('联行号（必填）')}
            />
            <TextField
              label={t('联系人')}
              required
              value={form.contact_person}
              onChange={(v) => set('contact_person', v)}
              placeholder={t('联系人（必填）')}
            />
          </FormSection>
          <FormSection title={t('企业附件')}>
            <DocUploadField
              label={t('营业执照')}
              labelExtra={t('需加盖公章')}
              url={form.business_license_url}
              onUrlChange={(u) => set('business_license_url', u)}
              onPreview={onPreview}
              imagesOnly
            />
            <DocUploadField
              label={t('法人身份证')}
              labelExtra={t('需加盖公章')}
              url={form.legal_person_id_card_url}
              onUrlChange={(u) => set('legal_person_id_card_url', u)}
              onPreview={onPreview}
              imagesOnly
            />
            <DocUploadField
              label={t('对公账户证明')}
              labelExtra={t('需加盖公章')}
              url={form.corporate_account_proof_url}
              onUrlChange={(u) => set('corporate_account_proof_url', u)}
              onPreview={onPreview}
              imagesOnly
            />
          </FormSection>
          <FormSection title={t('发票')}>
            <DocUploadField
              label={t('发票')}
              url={form.invoice_url}
              onUrlChange={(u) => set('invoice_url', u)}
              onPreview={onPreview}
              allowPdf
            />
          </FormSection>
        </>
      )}

      <FormSection title={t('提现金额')}>
        <FieldLabel required>{t('提现余额')}</FieldLabel>
        {isQuotaTokensMode ? (
          <>
            <Text type='tertiary' size='small' className='block mb-2'>
              {t(
                'TOKENS 模式：与上方「待使用收益」数字一致，填写系统内部点数（正整数）',
              )}
            </Text>
            <Input
              value={quotaInput}
              onChange={(v) => onQuotaInputChange(v == null ? '' : String(v))}
              placeholder={t('请输入内部点数')}
            />
          </>
        ) : (
          <WithdrawFiatAmountInput
            ref={fiatCommitRef}
            value={fiatAmount}
            onChange={onFiatAmountChange}
            min={fiatMin}
            max={fiatMax}
            placeholder={t('填写收益金额')}
          />
        )}
        <Text type='tertiary' size='small' className='block mt-1'>
          {affQuotaFloor >= minQInternal ? (
            <>
              {t('单笔最低')}: {renderQuota(minQInternal)} · {t('当前待使用余额')}:{' '}
              {renderQuota(affQuota || 0)}
            </>
          ) : (
            <>
              {t('单笔最低')}: {renderQuota(minQInternal)} ·{' '}
              {affQuotaFloor > 0 ? (
                <>
                  {t('当前余额低于系统最低门槛时，可提范围')}: {renderQuota(1)}～
                  {renderQuota(affQuotaFloor)} ·{' '}
                </>
              ) : null}
              {t('当前待使用余额')}: {renderQuota(affQuota || 0)}
            </>
          )}
        </Text>
      </FormSection>
    </div>
  );
}
