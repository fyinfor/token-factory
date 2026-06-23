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

import React, { useEffect, useRef, useState } from 'react';
import {
  Banner,
  Button,
  Card,
  Col,
  Form,
  Input,
  Row,
  Spin,
} from '@douyinfe/semi-ui';
import { API, showError, showSuccess, showWarning } from '../../helpers';
import { useTranslation } from 'react-i18next';

// 素材设置涉及的配置项（api_key 为只写字段，后端出于安全不会回显）。
const DEFAULT_INPUTS = {
  'seedance_setting.enabled': false,
  'seedance_setting.api_base_url': '',
  'seedance_setting.max_image_size_mb': 10,
  'seedance_setting.agreement_zh': '',
  'seedance_setting.agreement_en': '',
  'seedance_setting.agreement_detail_zh': '',
  'seedance_setting.agreement_detail_en': '',
};

export default function SettingsSeedance() {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState(DEFAULT_INPUTS);
  const [inputsRow, setInputsRow] = useState(DEFAULT_INPUTS);
  // API Key 单独管理：后端不回显，留空表示保持不变。
  const [apiKey, setApiKey] = useState('');
  const refForm = useRef();

  const getOptions = async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/option/');
      const { success, message, data } = res.data;
      if (!success) {
        showError(message);
        return;
      }
      const next = { ...DEFAULT_INPUTS };
      data.forEach((item) => {
        if (!(item.key in DEFAULT_INPUTS)) return;
        if (item.key === 'seedance_setting.enabled') {
          next[item.key] = item.value === 'true' || item.value === true;
        } else if (item.key === 'seedance_setting.max_image_size_mb') {
          next[item.key] = Number(item.value) || 10;
        } else {
          next[item.key] = item.value ?? '';
        }
      });
      setInputs(next);
      setInputsRow(structuredClone(next));
      refForm.current?.setValues(next);
    } catch (e) {
      showError(t('加载素材设置失败'));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    getOptions();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleFieldChange = (field) => (value) => {
    setInputs((prev) => ({ ...prev, [field]: value }));
  };

  const onSubmit = async () => {
    const requestQueue = [];
    Object.keys(DEFAULT_INPUTS).forEach((key) => {
      if (String(inputs[key]) !== String(inputsRow[key])) {
        requestQueue.push(
          API.put('/api/option/', { key, value: String(inputs[key]) }),
        );
      }
    });
    // API Key 非空时才提交（留空保持原值）。
    if (apiKey.trim() !== '') {
      requestQueue.push(
        API.put('/api/option/', {
          key: 'seedance_setting.api_key',
          value: apiKey.trim(),
        }),
      );
    }
    if (requestQueue.length === 0) {
      return showWarning(t('你似乎并没有修改什么'));
    }
    setLoading(true);
    try {
      const results = await Promise.all(requestQueue);
      const failed = results.find((r) => r?.data && r.data.success === false);
      if (failed) {
        showError(failed.data.message || t('保存失败，请重试'));
      } else {
        showSuccess(t('保存成功'));
        setApiKey('');
        await getOptions();
      }
    } catch (e) {
      showError(t('保存失败，请重试'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <Form
      values={inputs}
      getFormApi={(formAPI) => (refForm.current = formAPI)}
    >
      <Card>
        <Spin spinning={loading}>
          <Form.Section text={t('素材设置')}>
            <Banner
              type='info'
              description={t(
                'Seedance2.0 合规素材库配置。启用后，用户可在「个人中心-素材管理」中上传虚拟人像素材。所有素材相关接口将统一拼接下方的 API 基础地址进行请求。',
              )}
              closeIcon={null}
              style={{ marginBottom: 16 }}
            />
            <Row gutter={16}>
              <Col xs={24} sm={12} md={8}>
                <Form.Switch
                  field={'seedance_setting.enabled'}
                  label={t('启用素材库功能')}
                  checkedText='｜'
                  uncheckedText='〇'
                  onChange={handleFieldChange('seedance_setting.enabled')}
                />
              </Col>
              <Col xs={24} sm={12} md={8}>
                <Form.InputNumber
                  field={'seedance_setting.max_image_size_mb'}
                  label={t('单图上传大小上限(MB)')}
                  min={1}
                  max={100}
                  style={{ width: '100%' }}
                  onChange={handleFieldChange(
                    'seedance_setting.max_image_size_mb',
                  )}
                />
              </Col>
            </Row>
            <Row gutter={16}>
              <Col xs={24} md={12}>
                <Form.Input
                  field={'seedance_setting.api_base_url'}
                  label={t('素材库API基础地址')}
                  placeholder='https://ark.cn-beijing.volces.com'
                  onChange={handleFieldChange('seedance_setting.api_base_url')}
                />
              </Col>
              <Col xs={24} md={12}>
                <Form.Slot label={t('素材库API密钥(留空保持不变)')}>
                  <Input
                    mode='password'
                    placeholder={t('如已配置则留空，不会回显')}
                    value={apiKey}
                    onChange={(v) => setApiKey(v)}
                  />
                </Form.Slot>
              </Col>
            </Row>

            <Form.Section text={t('虚拟人像合规协议')}>
              <Row gutter={16}>
                <Col xs={24} md={12}>
                  <Form.TextArea
                    field={'seedance_setting.agreement_zh'}
                    label={t('协议文案(中文)')}
                    autosize={{ minRows: 2, maxRows: 4 }}
                    placeholder={t('上传前勾选同意的协议文案')}
                    onChange={handleFieldChange('seedance_setting.agreement_zh')}
                  />
                </Col>
                <Col xs={24} md={12}>
                  <Form.TextArea
                    field={'seedance_setting.agreement_en'}
                    label={t('协议文案(英文)')}
                    autosize={{ minRows: 2, maxRows: 4 }}
                    placeholder={t('上传前勾选同意的协议文案')}
                    onChange={handleFieldChange('seedance_setting.agreement_en')}
                  />
                </Col>
              </Row>
              <Row gutter={16}>
                <Col xs={24} md={12}>
                  <Form.TextArea
                    field={'seedance_setting.agreement_detail_zh'}
                    label={t('协议详情(中文)')}
                    autosize={{ minRows: 4, maxRows: 8 }}
                    placeholder={t('点击协议时展示的详情内容')}
                    onChange={handleFieldChange(
                      'seedance_setting.agreement_detail_zh',
                    )}
                  />
                </Col>
                <Col xs={24} md={12}>
                  <Form.TextArea
                    field={'seedance_setting.agreement_detail_en'}
                    label={t('协议详情(英文)')}
                    autosize={{ minRows: 4, maxRows: 8 }}
                    placeholder={t('点击协议时展示的详情内容')}
                    onChange={handleFieldChange(
                      'seedance_setting.agreement_detail_en',
                    )}
                  />
                </Col>
              </Row>
            </Form.Section>

            <Row>
              <Button theme='solid' onClick={onSubmit}>
                {t('保存素材设置')}
              </Button>
            </Row>
          </Form.Section>
        </Spin>
      </Card>
    </Form>
  );
}
