/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import { API } from '../../../../helpers';

/**
 * 构建管理端「用户手机号」字段校验规则：大陆 11 位格式 + 异步占用检测。
 * checkPhoneUrl：管理员用默认 GET /api/user/check_phone；当前登录用户自助校验可用 GET /api/user/self/phone_available。
 *
 * @param {function} t i18n 翻译函数
 * @param {{ excludeUserId?: number|string|(() => number|string|undefined|null), checkPhoneUrl?: string }} options
 * @returns {Array<object>} Semi Form rules
 */
export function buildAdminUserPhoneFieldRules(
  t,
  { excludeUserId, checkPhoneUrl = '/api/user/check_phone' } = {},
) {
  /** @returns {number|string|undefined|null} */
  const resolveExclude = () => {
    if (typeof excludeUserId === 'function') {
      return excludeUserId();
    }
    return excludeUserId;
  };

  return [
    {
      validator: (rule, value) => {
        const v = (value || '').trim();
        if (!v) return true;
        if (!/^1[3-9]\d{9}$/.test(v)) {
          return new Error(t('请输入有效的手机号'));
        }
        return true;
      },
    },
    {
      asyncValidator: async (rule, value) => {
        const v = (value || '').trim();
        if (!v || !/^1[3-9]\d{9}$/.test(v)) return;
        const params = { phone: v };
        const selfCheck =
          typeof checkPhoneUrl === 'string' &&
          checkPhoneUrl.includes('phone_available');
        if (!selfCheck) {
          const ex = resolveExclude();
          if (ex !== undefined && ex !== null && ex !== '') {
            params.exclude_id = ex;
          }
        }
        try {
          const res = await API.get(checkPhoneUrl, {
            params,
            skipErrorHandler: true,
            disableDuplicate: true,
          });
          const payload = res.data;
          if (!payload.success) {
            throw new Error(payload.message || t('校验失败'));
          }
          const available = payload.data?.available !== false;
          if (!available) {
            throw new Error(t('手机号已被占用'));
          }
        } catch (e) {
          const msg = e?.response?.data?.message;
          if (msg) {
            throw new Error(msg);
          }
          throw e;
        }
      },
    },
  ];
}
