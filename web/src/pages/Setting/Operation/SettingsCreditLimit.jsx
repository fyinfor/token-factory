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

import React, { useEffect, useState, useRef } from 'react';
import { Button, Col, Form, Row, Spin } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import {
  compareObjects,
  API,
  showError,
  showSuccess,
  showWarning,
} from '../../../helpers';

/** 与后台 common/constants.go 默认 QuotaPerUnit 一致，仅当选项未加载时的兜底 */
const DEFAULT_QUOTA_PER_UNIT = 500 * 1000;

const CREDIT_LIMIT_FORM_KEYS = [
  'QuotaForNewUser',
  'PreConsumedQuota',
  'QuotaForInviter',
  'QuotaForInvitee',
  'quota_setting.enable_free_model_pre_consume',
];

/** 运营后台以美元填写、保存时换算为站内额度整数的选项（与 common.QuotaFromUSD 一致） */
const USD_QUOTA_OPTION_KEYS = [
  'QuotaForNewUser',
  'QuotaForInviter',
  'QuotaForInvitee',
];

function parseQuotaPerUnit(raw) {
  const n = parseFloat(raw);
  return Number.isFinite(n) && n > 0 ? n : DEFAULT_QUOTA_PER_UNIT;
}

/** 接口中的额度整数 → 表单展示的美元（与 LogQuota / 充值计价同一套 QuotaPerUnit） */
function internalQuotaToUsd(quotaRaw, quotaPerUnit) {
  const q = parseInt(String(quotaRaw ?? '0'), 10);
  if (!Number.isFinite(q) || q <= 0) return 0;
  return q / quotaPerUnit;
}

/** 美元 → 提交给后端的额度整数字符串（向零截断，与 common.QuotaFromUSD 一致） */
function usdToInternalQuotaString(usdVal, quotaPerUnit) {
  const u = Number(usdVal);
  if (!Number.isFinite(u) || u <= 0) return '0';
  return String(Math.trunc(u * quotaPerUnit));
}

export default function SettingsCreditLimit(props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState({
    QuotaForNewUser: '',
    PreConsumedQuota: '',
    QuotaForInviter: '',
    QuotaForInvitee: '',
    'quota_setting.enable_free_model_pre_consume': true,
  });
  const refForm = useRef();
  const [inputsRow, setInputsRow] = useState(inputs);

  function onSubmit() {
    const quotaPerUnit = parseQuotaPerUnit(props.options?.QuotaPerUnit);
    const updateArray = compareObjects(inputs, inputsRow);
    if (!updateArray.length) return showWarning(t('你似乎并没有修改什么'));
    const requestQueue = updateArray.map((item) => {
      let value = '';
      if (typeof inputs[item.key] === 'boolean') {
        value = String(inputs[item.key]);
      } else if (USD_QUOTA_OPTION_KEYS.includes(item.key)) {
        value = usdToInternalQuotaString(inputs[item.key], quotaPerUnit);
      } else {
        value = inputs[item.key];
      }
      return API.put('/api/option/', {
        key: item.key,
        value,
      });
    });
    setLoading(true);
    Promise.all(requestQueue)
      .then((res) => {
        if (requestQueue.length === 1) {
          if (res.includes(undefined)) return;
        } else if (requestQueue.length > 1) {
          if (res.includes(undefined))
            return showError(t('部分保存失败，请重试'));
        }
        showSuccess(t('保存成功'));
        props.refresh();
      })
      .catch(() => {
        showError(t('保存失败，请重试'));
      })
      .finally(() => {
        setLoading(false);
      });
  }

  useEffect(() => {
    const quotaPerUnit = parseQuotaPerUnit(props.options?.QuotaPerUnit);
    const currentInputs = {};
    for (const key of CREDIT_LIMIT_FORM_KEYS) {
      if (props.options && Object.prototype.hasOwnProperty.call(props.options, key)) {
        currentInputs[key] = props.options[key];
      }
    }
    for (const key of USD_QUOTA_OPTION_KEYS) {
      if (Object.prototype.hasOwnProperty.call(currentInputs, key)) {
        currentInputs[key] = internalQuotaToUsd(
          currentInputs[key],
          quotaPerUnit,
        );
      }
    }
    setInputs(currentInputs);
    setInputsRow(structuredClone(currentInputs));
    refForm.current?.setValues?.(currentInputs);
  }, [props.options]);

  const usdQuotaExtraText = t('额度美元配置说明');

  return (
    <>
      <Spin spinning={loading}>
        <Form
          values={inputs}
          getFormApi={(formAPI) => (refForm.current = formAPI)}
          style={{ marginBottom: 15 }}
        >
          <Form.Section text={t('额度设置')}>
            <Row gutter={16}>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.InputNumber
                  label={t('新用户初始额度')}
                  field={'QuotaForNewUser'}
                  step={0.01}
                  min={0}
                  precision={6}
                  suffix={'USD'}
                  extraText={usdQuotaExtraText}
                  placeholder={t('例如：0.01')}
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      QuotaForNewUser: value,
                    })
                  }
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.InputNumber
                  label={t('请求预扣费额度')}
                  field={'PreConsumedQuota'}
                  step={1}
                  min={0}
                  suffix={'Token'}
                  extraText={t('请求结束后多退少补')}
                  placeholder={''}
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      PreConsumedQuota: String(value),
                    })
                  }
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.InputNumber
                  label={t('邀请新用户奖励额度')}
                  field={'QuotaForInviter'}
                  step={0.01}
                  min={0}
                  precision={6}
                  suffix={'USD'}
                  extraText={usdQuotaExtraText}
                  placeholder={t('例如：0.01')}
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      QuotaForInviter: value,
                    })
                  }
                />
              </Col>
            </Row>
            <Row>
              <Col xs={24} sm={12} md={8} lg={8} xl={6}>
                <Form.InputNumber
                  label={t('新用户使用邀请码奖励额度')}
                  field={'QuotaForInvitee'}
                  step={0.01}
                  min={0}
                  precision={6}
                  suffix={'USD'}
                  extraText={usdQuotaExtraText}
                  placeholder={t('例如：0.01')}
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      QuotaForInvitee: value,
                    })
                  }
                />
              </Col>
            </Row>
            <Row>
              <Col>
                <Form.Switch
                  label={t('对免费模型启用预消耗')}
                  field={'quota_setting.enable_free_model_pre_consume'}
                  extraText={t(
                    '开启后，对免费模型（倍率为0，或者价格为0）的模型也会预消耗额度',
                  )}
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      'quota_setting.enable_free_model_pre_consume': value,
                    })
                  }
                />
              </Col>
            </Row>

            <Row>
              <Button size='default' onClick={onSubmit}>
                {t('保存额度设置')}
              </Button>
            </Row>
          </Form.Section>
        </Form>
      </Spin>
    </>
  );
}
