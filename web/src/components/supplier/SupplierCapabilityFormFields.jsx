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
import { Form, Row, Col, Divider, Typography } from '@douyinfe/semi-ui';

const { Text } = Typography;

/**
 * SupplierCapabilityFormFields 供应商技术能力表单字段区（两列 12/12 栅格：多模态后为参考单价，再为定价+API 地址成对，减少单列留白）。
 * @param {object} props
 * @param {function} props.t i18n 翻译函数
 * @param {function} [props.onCommitmentChange] 信息真实性承诺勾选变更回调
 */
const SupplierCapabilityFormFields = ({ t, onCommitmentChange }) => {
  return (
    <>
      <Divider margin='12px 12px'>
        <Text strong style={{ fontSize: '16px' }}>
          {t('技术能力信息')}
        </Text>
      </Divider>
      <Row gutter={12} type='flex' align='top'>
        <Col span={12}>
          <Form.Select
            field='core_service_types'
            label={<Text strong>{t('核心服务类型')}<Text type='danger'>*</Text></Text>}
            multiple
            placeholder={t('请选择核心服务类型')}
            rules={[{ required: true, message: t('请选择核心服务类型') }]}
            size='large'
            style={{ width: '100%' }}
            optionList={[
              { label: t('大语言模型'), value: 'llm' },
              { label: t('视觉模型'), value: 'vision' },
              { label: t('多模态模型'), value: 'multimodal' },
              { label: t('AI工具API'), value: 'tool_api' },
              { label: t('其他'), value: 'other' },
            ]}
          />
        </Col>
        <Col span={12}>
          <Form.TagInput
            field='supported_models'
            label={<Text strong>{t('支持的模型')}<Text type='danger'>*</Text></Text>}
            placeholder={t('输入后回车添加，如 GPT-4、文心一言4.0')}
            rules={[{ required: true, message: t('请填写支持的模型') }]}
            size='large'
            style={{ width: '100%' }}
          />
        </Col>
        <Col span={24}>
          <Form.Input field='supported_model_notes' label={<Text strong>{t('模型补充说明')}</Text>} showClear />
        </Col>
        <Col span={12}>
          <Form.Select
            field='supported_api_endpoints'
            label={<Text strong>{t('支持的API接口')}<Text type='danger'>*</Text></Text>}
            multiple
            placeholder={t('请选择支持的API接口')}
            rules={[{ required: true, message: t('请选择支持的API接口') }]}
            size='large'
            style={{ width: '100%' }}
            optionList={[
              { label: '/chat/completions', value: '/chat/completions' },
              { label: '/completions', value: '/completions' },
              { label: '/embeddings', value: '/embeddings' },
              { label: '/models', value: '/models' },
            ]}
          />
        </Col>
        <Col span={12}>
          <Form.Select
            field='supported_params'
            label={<Text strong>{t('支持的参数配置')}<Text type='danger'>*</Text></Text>}
            multiple
            placeholder={t('请选择支持的参数配置')}
            rules={[{ required: true, message: t('请选择支持的参数配置') }]}
            size='large'
            style={{ width: '100%' }}
            optionList={[
              { label: 'max_tokens', value: 'max_tokens' },
              { label: 'temperature', value: 'temperature' },
              { label: 'top_p', value: 'top_p' },
              { label: 'seed', value: 'seed' },
              { label: 'top_k', value: 'top_k' },
            ]}
          />
        </Col>
        <Col span={12}>
          <Form.Input field='supported_api_endpoint_extra' label={<Text strong>{t('API接口补充')}</Text>} showClear />
        </Col>
        <Col span={12}>
          <Form.Input field='supported_params_extra' label={<Text strong>{t('参数配置补充')}</Text>} showClear />
        </Col>
        <Col span={12}>
          <Form.RadioGroup
            field='streaming_supported'
            label={<Text strong>{t('流式响应支持')}<Text type='danger'>*</Text></Text>}
            type='button'
            options={[
              { label: t('是'), value: 'yes' },
              { label: t('否'), value: 'no' },
            ]}
          />
        </Col>
        <Col span={12}>
          <Form.Input field='streaming_notes' label={<Text strong>{t('流式响应说明')}</Text>} showClear />
        </Col>
        <Col span={12}>
          <Form.RadioGroup
            field='structured_output_supported'
            label={<Text strong>{t('结构化输出支持')}<Text type='danger'>*</Text></Text>}
            type='button'
            options={[
              { label: t('是'), value: 'yes' },
              { label: t('否'), value: 'no' },
            ]}
          />
        </Col>
        <Col span={12}>
          <Form.Input field='structured_output_notes' label={<Text strong>{t('结构化输出说明')}</Text>} showClear />
        </Col>
        <Col span={12}>
          <Form.Select
            field='multimodal_types'
            label={<Text strong>{t('多模态支持')}</Text>}
            multiple
            placeholder={t('请选择多模态支持')}
            size='large'
            style={{ width: '100%' }}
            optionList={[
              { label: t('图片'), value: 'image' },
              { label: t('音频'), value: 'audio' },
              { label: t('视频'), value: 'video' },
            ]}
          />
        </Col>
        <Col span={12}>
          <Form.Input field='multimodal_extra' label={<Text strong>{t('多模态补充')}</Text>} showClear />
        </Col>
        <Col span={12}>
          <Form.Input field='reference_input_price' label={<Text strong>{t('参考输入单价')}</Text>} showClear />
        </Col>
        <Col span={12}>
          <Form.Input field='reference_output_price' label={<Text strong>{t('参考输出单价')}</Text>} showClear />
        </Col>
        <Col span={12}>
          <Form.Select
            field='pricing_modes'
            label={<Text strong>{t('定价模式')}<Text type='danger'>*</Text></Text>}
            multiple
            placeholder={t('请选择定价模式')}
            rules={[{ required: true, message: t('请选择定价模式') }]}
            size='large'
            style={{ width: '100%' }}
            optionList={[
              { label: t('按Token计费'), value: 'token' },
              { label: t('按次计费'), value: 'times' },
              { label: t('包月'), value: 'monthly' },
              { label: t('定制化报价'), value: 'custom' },
            ]}
          />
        </Col>
        <Col span={12}>
          <Form.TagInput
            field='api_base_urls'
            label={<Text strong>{t('API接口地址')}<Text type='danger'>*</Text></Text>}
            placeholder={t('输入后回车添加，例如 https://api.example.com/v1')}
            rules={[{ required: true, message: t('请填写API接口地址') }]}
            size='large'
            style={{ width: '100%' }}
          />
        </Col>
        <Col span={12}>
          <Form.RadioGroup
            field='failure_billing_mode'
            label={<Text strong>{t('故障计费规则')}<Text type='danger'>*</Text></Text>}
            type='button'
            options={[
              { label: t('计费'), value: 'bill' },
              { label: t('不计费'), value: 'no_bill' },
            ]}
          />
        </Col>
        <Col span={12}>
          <Form.Input field='failure_billing_notes' label={<Text strong>{t('故障计费说明')}</Text>} showClear />
        </Col>
        <Col span={12}>
          <Form.RadioGroup
            field='openai_compatible'
            label={<Text strong>{t('兼容OpenAI规范')}</Text>}
            type='button'
            options={[
              { label: t('是'), value: 'yes' },
              { label: t('否'), value: 'no' },
            ]}
          />
        </Col>
        <Col span={12} style={{ display: 'flex', alignItems: 'flex-end' }}>
          <Form.Checkbox
            field='truth_commitment_confirmed'
            noLabel
            rules={[{ required: true, type: 'boolean', message: t('请勾选信息真实性承诺') }]}
            onChange={(checked) => onCommitmentChange?.(!!checked)}
          >
            {t('我承诺提交的信息真实有效，并同意平台审核')}
          </Form.Checkbox>
        </Col>
      </Row>
    </>
  );
};

export default SupplierCapabilityFormFields;
