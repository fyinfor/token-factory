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

import React from 'react';
import { Select, Input } from '@douyinfe/semi-ui';
import { IconPhone } from '@douyinfe/semi-icons';
import {
  COUNTRY_DIAL_CODES,
  findCountryByCode,
  toE164,
  splitE164Phone,
} from '../../../constants/countryDialCodes';

const EMOJI_FONT = '"Segoe UI Emoji","Apple Color Emoji","Noto Color Emoji","Twemoji Mozilla",sans-serif';

/**
 * 「国码 select + 本地号 input」复合控件。受控组件。
 *
 * Props:
 *   value:           string  当前完整手机号（国内 11 位 / E.164）
 *   onChange(v):     void    整串值变化
 *   onValidate(err): void    校验结果回调（err 为 string 或 null）
 *   intlEnabled:     boolean 是否启用国际号；不启用时下拉禁用，强制 CN
 *   disabled:        boolean
 *   placeholder:     string  本地号 placeholder
 *   showPhoneIcon:   boolean 是否在本地号 input 加 📞 prefix（默认 true）
 *   allowClear:      boolean
 *   width:           number  下拉宽度（默认 100）
 *   error:           string  外部传入的校验错误（用于服务端/异步错误）
 *   required:        boolean 是否必填
 *   rules:           Array   校验规则 [{ required, message, pattern, asyncValidator }]
 *   validateRef:     useRef  父级传 ref.current.runFullValidate(value) 跑完整校验
 */
const CountryPhoneInput = ({
  value = '',
  onChange,
  onValidate,
  intlEnabled = false,
  disabled = false,
  placeholder,
  showPhoneIcon = true,
  allowClear = false,
  width = 100,
  error: externalError,
  required = false,
  rules = [],
  validateRef,
}) => {
  const split = React.useMemo(() => splitE164Phone(value), [value]);
  const [countryCode, setCountryCode] = React.useState(split.countryCode);
  const [localNumber, setLocalNumber] = React.useState(split.localNumber);
  const [internalError, setInternalError] = React.useState('');

  React.useEffect(() => {
    const s = splitE164Phone(value);
    setCountryCode(s.countryCode);
    setLocalNumber(s.localNumber);
  }, [value]);

  // 同步执行校验
  const validate = React.useCallback(
    (val) => {
      const v = String(val || '').trim();
      // 必填检查
      if (required && !v) {
        const msg = (rules.find((r) => r.required) || {}).message || '此项为必填';
        return msg;
      }
      // 同步规则
      for (const rule of rules) {
        if (rule.required && !v) {
          return rule.message || '此项为必填';
        }
        if (rule.pattern && v && !rule.pattern.test(v)) {
          return rule.message || '格式不正确';
        }
        // 同步 validator
        if (typeof rule.validator === 'function' && v) {
          let result;
          try {
            result = rule.validator(rule, v);
          } catch (e) {
            return e?.message || '校验失败';
          }
          if (result && typeof result === 'object' && 'message' in result) {
            return result.message;
          }
          if (result instanceof Error) {
            return result.message;
          }
          if (result === false) {
            return rule.message || '校验失败';
          }
        }
      }
      return '';
    },
    [rules, required],
  );

  // value 变化时跑同步校验
  React.useEffect(() => {
    if (!onValidate) return;
    const err = validate(value);
    setInternalError(err);
    onValidate(err || null);
  }, [value, validate, onValidate]);

  /**
   * 暴露给父级调用：跑全部 rules（含 asyncValidator），返回首个错误或 null。
   * 用于在 form submit 时阻塞提交。
   */
  const runFullValidate = React.useCallback(
    async (val) => {
      const v = String(val || '').trim();
      if (required && !v) {
        const msg = (rules.find((r) => r.required) || {}).message || '此项为必填';
        setInternalError(msg);
        return msg;
      }
      for (const rule of rules) {
        if (rule.required && !v) {
          const msg = rule.message || '此项为必填';
          setInternalError(msg);
          return msg;
        }
        if (rule.pattern && v && !rule.pattern.test(v)) {
          const msg = rule.message || '格式不正确';
          setInternalError(msg);
          return msg;
        }
        if (typeof rule.validator === 'function' && v) {
          try {
            const r = rule.validator(rule, v);
            if (r && typeof r === 'object' && 'message' in r) {
              setInternalError(r.message);
              return r.message;
            }
            if (r instanceof Error) {
              setInternalError(r.message);
              return r.message;
            }
            if (r === false) {
              const msg = rule.message || '校验失败';
              setInternalError(msg);
              return msg;
            }
          } catch (e) {
            const msg = e?.message || '校验失败';
            setInternalError(msg);
            return msg;
          }
        }
        if (typeof rule.asyncValidator === 'function' && v) {
          try {
            await rule.asyncValidator(rule, v);
          } catch (e) {
            const msg = e?.message || '校验失败';
            setInternalError(msg);
            return msg;
          }
        }
      }
      setInternalError('');
      return null;
    },
    [rules, required],
  );

  /**
   * 暴露给父级调用：跑全部 rules（含 asyncValidator），返回首个错误或 null。
   * 父级通过给本组件传 `validateRef` 拿到 runFullValidate 句柄。
   */
  const runFullValidateRef = React.useRef(null);
  runFullValidateRef.current = runFullValidate;
  React.useEffect(() => {
    if (validateRef) {
      validateRef.current = {
        runFullValidate: (v) => runFullValidateRef.current(v),
      };
    }
    return () => {
      if (validateRef) validateRef.current = null;
    };
  }, [validateRef, runFullValidate]);

  const emit = (nextCountry, nextLocal) => {
    const country = findCountryByCode(nextCountry) || COUNTRY_DIAL_CODES[0];
    const local = String(nextLocal || '').replace(/[^\d]/g, '');
    let nextValue;
    if (!intlEnabled || country.code === 'CN') {
      nextValue = local;
    } else {
      nextValue = toE164(country, local);
    }
    onChange?.(nextValue);
  };

  const renderCountryOption = (c) => (
    <span
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: 6,
        minWidth: 0,
      }}
    >
      <span
        style={{
          fontFamily: EMOJI_FONT,
          fontSize: 16,
          lineHeight: 1,
        }}
      >
        {c.flag}
      </span>
      <span style={{ flexShrink: 0, fontSize: 12 }}>+{c.dial}</span>
    </span>
  );

  const renderSelectedCountry = (c) => (
    <span
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: 4,
        lineHeight: 1,
      }}
    >
      <span
        style={{
          fontFamily: EMOJI_FONT,
          fontSize: 16,
          lineHeight: 1,
        }}
      >
        {c.flag}
      </span>
      <span style={{ fontSize: 12 }}>+{c.dial}</span>
    </span>
  );

  const showError = externalError || internalError;

  return (
    <div>
      <div style={{ display: 'flex', gap: 8, alignItems: 'stretch', width: '100%' }}>
        <div style={{ width, flexShrink: 0 }}>
          <Select
            value={countryCode}
            disabled={disabled || !intlEnabled}
            onChange={(val) => {
              setCountryCode(val);
              emit(val, localNumber);
            }}
            filter
            searchPosition='dropdown'
            renderSelectedItem={(option) => {
              const c = findCountryByCode(option.value) || COUNTRY_DIAL_CODES[0];
              return renderSelectedCountry(c);
            }}
            optionList={COUNTRY_DIAL_CODES.map((c) => ({
              value: c.code,
              label: renderCountryOption(c),
              text: `${c.name} +${c.dial} ${c.code}`,
            }))}
            style={{ width: '100%' }}
          />
        </div>
        <div style={{ flex: 1, minWidth: 0 }}>
          <Input
            disabled={disabled}
            placeholder={placeholder || '请输入手机号'}
            value={localNumber}
            allowClear={allowClear}
            onChange={(v) => {
              const cleaned = String(v || '').replace(/[^\d]/g, '');
              setLocalNumber(cleaned);
              emit(countryCode, cleaned);
            }}
            prefix={showPhoneIcon ? <IconPhone /> : null}
            validateStatus={showError ? 'error' : undefined}
          />
        </div>
      </div>
      {showError && (
        <div
          style={{
            color: 'var(--semi-color-danger)',
            fontSize: 12,
            marginTop: 4,
          }}
        >
          {showError}
        </div>
      )}
    </div>
  );
};

export default CountryPhoneInput;
