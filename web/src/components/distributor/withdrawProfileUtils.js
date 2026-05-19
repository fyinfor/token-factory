/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

export const WD_ACCOUNT_PERSONAL = 1;
export const WD_ACCOUNT_ENTERPRISE = 2;

export const EMPTY_WD_FORM = {
  real_name: '',
  bank_name: '',
  bank_account: '',
  id_card_no: '',
  id_card_expiry: '',
  mobile: '',
  bank_reserved_phone: '',
  id_card_front_url: '',
  id_card_back_url: '',
  bank_card_photo_url: '',
  credit_code: '',
  legal_person_name: '',
  legal_person_phone: '',
  bank_branch_code: '',
  contact_person: '',
  business_license_url: '',
  corporate_account_proof_url: '',
  invoice_url: '',
};

/** 合并 API 返回的扁平字段与 profile_data */
export function parseWithdrawRow(row) {
  const r = row || {};
  const pd =
    r.profile_data && typeof r.profile_data === 'object' ? r.profile_data : {};
  return {
    account_type: Number(r.account_type) === WD_ACCOUNT_ENTERPRISE ? WD_ACCOUNT_ENTERPRISE : WD_ACCOUNT_PERSONAL,
    real_name: r.real_name || '',
    bank_name: r.bank_name || '',
    bank_account: r.bank_account || '',
    id_card_no: r.id_card_no || pd.id_card_no || '',
    id_card_expiry: r.id_card_expiry || pd.id_card_expiry || '',
    mobile: r.mobile || pd.mobile || '',
    bank_reserved_phone: r.bank_reserved_phone || pd.bank_reserved_phone || '',
    id_card_front_url: r.id_card_front_url || pd.id_card_front_url || '',
    id_card_back_url: r.id_card_back_url || pd.id_card_back_url || '',
    bank_card_photo_url: r.bank_card_photo_url || pd.bank_card_photo_url || '',
    credit_code: r.credit_code || pd.credit_code || '',
    legal_person_name: r.legal_person_name || pd.legal_person_name || '',
    legal_person_phone: r.legal_person_phone || pd.legal_person_phone || '',
    bank_branch_code: r.bank_branch_code || pd.bank_branch_code || '',
    contact_person: r.contact_person || pd.contact_person || '',
    business_license_url: r.business_license_url || pd.business_license_url || '',
    corporate_account_proof_url:
      r.corporate_account_proof_url || pd.corporate_account_proof_url || '',
    invoice_url: r.invoice_url || pd.invoice_url || '',
    voucher_urls: parseVoucherUrls(r.voucher_urls),
  };
}

export function parseVoucherUrls(raw) {
  if (raw == null) return [];
  if (Array.isArray(raw)) return raw.filter(Boolean);
  if (typeof raw === 'string') {
    const s = raw.trim();
    if (!s) return [];
    try {
      const j = JSON.parse(s);
      return Array.isArray(j) ? j.filter(Boolean) : [];
    } catch {
      return [];
    }
  }
  return [];
}

export function wdAccountTypeLabel(t, accountType) {
  return Number(accountType) === WD_ACCOUNT_ENTERPRISE ? t('企业') : t('个人');
}

/** 个人/企业表单校验，返回错误文案或 null */
export function validateWithdrawForm(t, accountType, f) {
  const name = String(f.real_name || '').trim();
  const bankName = String(f.bank_name || '').trim();
  const bankAccount = String(f.bank_account || '').trim();
  if (!name) {
    return accountType === WD_ACCOUNT_ENTERPRISE
      ? t('请填写企业名称')
      : t('请填写姓名');
  }
  if (!bankName) return t('请填写开户行');
  if (!bankAccount) {
    return accountType === WD_ACCOUNT_ENTERPRISE
      ? t('请填写对公卡号')
      : t('请填写银行卡号');
  }
  if (accountType === WD_ACCOUNT_PERSONAL) {
    if (!String(f.id_card_no || '').trim()) return t('请填写身份证号');
    if (!String(f.id_card_expiry || '').trim()) return t('请填写身份证有效期');
    if (!String(f.mobile || '').trim()) return t('请填写手机号');
    if (!String(f.bank_reserved_phone || '').trim()) {
      return t('请填写银行预留手机号');
    }
    if (!String(f.id_card_front_url || '').trim()) return t('请上传身份证正面');
    if (!String(f.id_card_back_url || '').trim()) return t('请上传身份证反面');
    if (!String(f.bank_card_photo_url || '').trim()) return t('请上传银行卡照');
    if (!String(f.invoice_url || '').trim()) return t('请上传发票');
  } else {
    if (!String(f.credit_code || '').trim()) return t('请填写统一社会信用代码');
    if (!String(f.legal_person_name || '').trim()) return t('请填写法人姓名');
    if (!String(f.legal_person_phone || '').trim()) return t('请填写法人手机号');
    if (!String(f.bank_branch_code || '').trim()) return t('请填写联行号');
    if (!String(f.contact_person || '').trim()) return t('请填写联系人');
    if (!String(f.business_license_url || '').trim()) return t('请上传营业执照');
    if (!String(f.corporate_account_proof_url || '').trim()) {
      return t('请上传对公账户证明');
    }
    if (!String(f.invoice_url || '').trim()) return t('请上传发票');
  }
  return null;
}

export function buildWithdrawSubmitBody(accountType, f, quotaAmount) {
  const base = {
    account_type: accountType,
    real_name: String(f.real_name || '').trim(),
    bank_name: String(f.bank_name || '').trim(),
    bank_account: String(f.bank_account || '').trim(),
    quota_amount: Math.round(Number(quotaAmount)),
  };
  if (accountType === WD_ACCOUNT_PERSONAL) {
    return {
      ...base,
      id_card_no: String(f.id_card_no || '').trim(),
      id_card_expiry: String(f.id_card_expiry || '').trim(),
      mobile: String(f.mobile || '').trim(),
      bank_reserved_phone: String(f.bank_reserved_phone || '').trim(),
      id_card_front_url: String(f.id_card_front_url || '').trim(),
      id_card_back_url: String(f.id_card_back_url || '').trim(),
      bank_card_photo_url: String(f.bank_card_photo_url || '').trim(),
      invoice_url: String(f.invoice_url || '').trim(),
    };
  }
  return {
    ...base,
    credit_code: String(f.credit_code || '').trim(),
    legal_person_name: String(f.legal_person_name || '').trim(),
    legal_person_phone: String(f.legal_person_phone || '').trim(),
    bank_branch_code: String(f.bank_branch_code || '').trim(),
    contact_person: String(f.contact_person || '').trim(),
    business_license_url: String(f.business_license_url || '').trim(),
    corporate_account_proof_url: String(f.corporate_account_proof_url || '').trim(),
    invoice_url: String(f.invoice_url || '').trim(),
  };
}
