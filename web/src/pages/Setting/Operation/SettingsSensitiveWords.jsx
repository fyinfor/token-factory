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
import { Button, Col, Form, Row, Spin, Tag } from '@douyinfe/semi-ui';
import {
  compareObjects,
  API,
  showError,
  showSuccess,
  showWarning,
} from '../../../helpers';
import { useTranslation } from 'react-i18next';

const defaultInputs = {
  CheckSensitiveEnabled: false,
  CheckSensitiveOnPromptEnabled: false,
  SensitiveWords: '',
  TencentTMSModerationEnabled: false,
  TencentIMSModerationEnabled: false,
  TencentTMSOutputModerationEnabled: false,
  TencentIMSOutputModerationEnabled: false,
  TencentTMSSecretID: '',
  TencentTMSSecretKey: '',
  TencentTMSRegion: 'ap-guangzhou',
  TencentTMSBizType: 'TencentCloudDefault',
  TencentIMSBizType: 'TencentCloudDefault',
};

export default function SettingsSensitiveWords(props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState(defaultInputs);
  const refForm = useRef();
  const [inputsRow, setInputsRow] = useState(inputs);

  async function onSubmit() {
    if (
      (inputs.TencentTMSModerationEnabled ||
        inputs.TencentIMSModerationEnabled) &&
      (!inputs.TencentTMSSecretID ||
        !inputs.TencentTMSSecretKey ||
        !inputs.TencentTMSRegion)
    ) {
      return showWarning(t('启用腾讯云文本内容安全前请填写完整凭证和地域'));
    }
    const updateArray = compareObjects(inputs, inputsRow);
    if (!updateArray.length) return showWarning(t('你似乎并没有修改什么'));
    setLoading(true);
    try {
      const orderedUpdates = [...updateArray].sort((left, right) => {
        const delayedKeys = [
          'TencentTMSModerationEnabled',
          'TencentIMSModerationEnabled',
          'TencentTMSOutputModerationEnabled',
          'TencentIMSOutputModerationEnabled',
        ];
        if (delayedKeys.includes(left.key)) return 1;
        if (delayedKeys.includes(right.key)) return -1;
        return 0;
      });
      for (const item of orderedUpdates) {
        const value =
          typeof inputs[item.key] === 'boolean'
            ? String(inputs[item.key])
            : inputs[item.key];
        const response = await API.put('/api/option/', {
          key: item.key,
          value,
        });
        if (!response?.data?.success) {
          throw new Error(response?.data?.message || t('保存失败，请重试'));
        }
      }
      showSuccess(t('保存成功'));
      await props.refresh();
    } catch (error) {
      showError(error?.message || t('保存失败，请重试'));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    const currentInputs = { ...defaultInputs };
    for (let key in props.options) {
      if (Object.hasOwn(defaultInputs, key)) {
        currentInputs[key] = props.options[key];
      }
    }
    setInputs(currentInputs);
    setInputsRow(structuredClone(currentInputs));
    refForm.current.setValues(currentInputs);
  }, [props.options]);
  return (
    <>
      <Spin spinning={loading}>
        <Form
          values={inputs}
          getFormApi={(formAPI) => (refForm.current = formAPI)}
          style={{ marginBottom: 15 }}
        >
          <Form.Section text={t('屏蔽词过滤设置')}>
            <Row gutter={16}>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Switch
                  field={'CheckSensitiveEnabled'}
                  label={t('启用屏蔽词过滤功能')}
                  size='default'
                  checkedText='｜'
                  uncheckedText='〇'
                  onChange={(value) => {
                    setInputs({
                      ...inputs,
                      CheckSensitiveEnabled: value,
                    });
                  }}
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Switch
                  field={'CheckSensitiveOnPromptEnabled'}
                  label={t('启用 Prompt 检查')}
                  size='default'
                  checkedText='｜'
                  uncheckedText='〇'
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      CheckSensitiveOnPromptEnabled: value,
                    })
                  }
                />
              </Col>
            </Row>
            <Row>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.TextArea
                  label={t('屏蔽词列表')}
                  extraText={t('一行一个屏蔽词，不需要符号分割')}
                  placeholder={t('一行一个屏蔽词，不需要符号分割')}
                  field={'SensitiveWords'}
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      SensitiveWords: value,
                    })
                  }
                  style={{ fontFamily: 'JetBrains Mono, Consolas' }}
                  autosize={{ minRows: 6, maxRows: 12 }}
                />
              </Col>
            </Row>
            <Row gutter={16}>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Switch
                  field={'TencentTMSModerationEnabled'}
                  label={t('启用腾讯云文本内容安全')}
                  extraText={t('审核 Prompt，返回 Review 或 Block 时拒绝请求')}
                  size='default'
                  checkedText='｜'
                  uncheckedText='〇'
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      TencentTMSModerationEnabled: value,
                    })
                  }
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Switch
                  field={'TencentIMSModerationEnabled'}
                  label={t('启用腾讯云图片内容安全')}
                  extraText={t(
                    '审核请求中的图片，返回 Review 或 Block 时拒绝请求',
                  )}
                  size='default'
                  checkedText='｜'
                  uncheckedText='〇'
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      TencentIMSModerationEnabled: value,
                    })
                  }
                />
              </Col>
            </Row>
            <Row gutter={16}>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Switch
                  field={'TencentTMSOutputModerationEnabled'}
                  label={t('审核上游返回的文本')}
                  extraText={t('非流式完整审核，流式按段审核通过后再返回')}
                  size='default'
                  checkedText='｜'
                  uncheckedText='〇'
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      TencentTMSOutputModerationEnabled: value,
                    })
                  }
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Switch
                  field={'TencentIMSOutputModerationEnabled'}
                  label={t('审核上游生成的图片')}
                  extraText={t('取得图片 URL 或 Base64 后审核通过再返回')}
                  size='default'
                  checkedText='｜'
                  uncheckedText='〇'
                  onChange={(value) =>
                    setInputs({
                      ...inputs,
                      TencentIMSOutputModerationEnabled: value,
                    })
                  }
                />
              </Col>
            </Row>
            <Row gutter={16}>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Input
                  field={'TencentTMSSecretID'}
                  label={t('腾讯云 SecretId')}
                  placeholder={t('请输入腾讯云 SecretId')}
                  onChange={(value) =>
                    setInputs({ ...inputs, TencentTMSSecretID: value })
                  }
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Input
                  field={'TencentTMSSecretKey'}
                  label={t('腾讯云 SecretKey')}
                  placeholder={t('请输入腾讯云 SecretKey')}
                  mode='password'
                  onChange={(value) =>
                    setInputs({ ...inputs, TencentTMSSecretKey: value })
                  }
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Input
                  field={'TencentTMSRegion'}
                  label={t('腾讯云地域')}
                  placeholder='ap-guangzhou'
                  onChange={(value) =>
                    setInputs({ ...inputs, TencentTMSRegion: value })
                  }
                />
              </Col>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Input
                  field={'TencentIMSBizType'}
                  label={t('腾讯云图片 BizType')}
                  extraText={t('不使用图片自定义策略时保留默认值')}
                  placeholder='TencentCloudDefault'
                  onChange={(value) =>
                    setInputs({ ...inputs, TencentIMSBizType: value })
                  }
                />
              </Col>
            </Row>
            <Row gutter={16}>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Input
                  field={'TencentTMSBizType'}
                  label={t('腾讯云 BizType')}
                  extraText={t('不使用自定义策略时保留默认值')}
                  placeholder='TencentCloudDefault'
                  onChange={(value) =>
                    setInputs({ ...inputs, TencentTMSBizType: value })
                  }
                />
              </Col>
            </Row>
            <Row>
              <Button size='default' onClick={onSubmit}>
                {t('保存屏蔽词过滤设置')}
              </Button>
            </Row>
          </Form.Section>
        </Form>
      </Spin>
    </>
  );
}
