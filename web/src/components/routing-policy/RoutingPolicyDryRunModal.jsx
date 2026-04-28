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

import React, { useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Banner,
  Button,
  Empty,
  Form,
  Modal,
  Space,
  Spin,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { API, showError } from '../../helpers';

const { Text, Paragraph } = Typography;

const STRATEGY_OPTIONS = [
  { value: 'price', label: '低价优先' },
  { value: 'latency', label: '低时延优先' },
  { value: 'throughput', label: '高吞吐优先' },
  { value: 'balanced', label: '均衡' },
  { value: 'custom', label: '自定义' },
];

const FALLBACK_OPTIONS = [
  { value: '', label: '不兜底' },
  { value: 'price', label: '价格兜底' },
  { value: 'latency', label: '时延兜底' },
  { value: 'any', label: '任意兜底' },
];

const SOURCE_LABEL = {
  none: '未生效',
  request: '请求体（最高优先级）',
  user: '用户默认策略',
  merged: '请求体 + 用户策略合并',
};

// dryRunSeedToFormValues 兼容三种入口：null（新建）、列表里的 row（不带 targets）、详情。
function dryRunSeedToFormValues(seed) {
  const base = {
    name: 'dry-run',
    description: '',
    strategy: 'balanced',
    allow_fallbacks: true,
    fallback_strategy: '',
    max_price: 0,
    max_latency_ms: 0,
    min_throughput_tps: 0,
    provider_overrides_json: '',
    status: 1,
    is_default: false,
    priority: 0,
    targets: [],
    request_provider_json: '',
    request_model: '',
  };
  if (!seed) return base;
  return {
    ...base,
    name: seed.name || 'dry-run',
    strategy: seed.strategy || 'balanced',
    allow_fallbacks: seed.allow_fallbacks ?? true,
    fallback_strategy: seed.fallback_strategy ?? '',
    max_price: seed.max_price ?? 0,
    max_latency_ms: seed.max_latency_ms ?? 0,
    min_throughput_tps: seed.min_throughput_tps ?? 0,
    provider_overrides_json: seed.provider_overrides_json || '',
    targets: Array.isArray(seed.targets)
      ? seed.targets.map((t) => ({
          target_type: t.target_type || 'channel_model',
          channel_id: t.channel_id ?? 0,
          model_name: t.model_name || '',
        }))
      : [],
  };
}

const RoutingPolicyDryRunModal = ({ visible, seedPolicy, onClose }) => {
  const { t } = useTranslation();
  const formApiRef = useRef(null);
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState(null);

  const initialValues = useMemo(
    () => dryRunSeedToFormValues(seedPolicy),
    [seedPolicy],
  );

  // visible 切换时重置；同时清空上一次预览结果，避免「换了 seed 还看到旧解析」。
  useEffect(() => {
    if (!visible) return;
    setResult(null);
    if (formApiRef.current) {
      formApiRef.current.setValues(initialValues, { isOverride: true });
    }
  }, [visible, initialValues]);

  const runDryRun = async (values) => {
    if (
      values.strategy === 'custom' &&
      values.provider_overrides_json &&
      values.provider_overrides_json.trim()
    ) {
      try {
        JSON.parse(values.provider_overrides_json);
      } catch (e) {
        showError(`${t('provider_overrides_json 不是合法 JSON')}: ${e.message}`);
        return;
      }
    }
    if (
      values.request_provider_json &&
      values.request_provider_json.trim()
    ) {
      try {
        JSON.parse(values.request_provider_json);
      } catch (e) {
        showError(`${t('请求侧 provider JSON 不是合法 JSON')}: ${e.message}`);
        return;
      }
    }

    const targets = (values.targets || []).map((row) => {
      const t = { target_type: row.target_type };
      if (row.target_type === 'channel') {
        t.channel_id = parseInt(row.channel_id, 10) || 0;
        t.model_name = '';
      } else if (row.target_type === 'model') {
        t.channel_id = 0;
        t.model_name = (row.model_name || '').trim();
      } else {
        t.channel_id = parseInt(row.channel_id, 10) || 0;
        t.model_name = (row.model_name || '').trim();
      }
      return t;
    });

    const payload = {
      name: (values.name || 'dry-run').trim(),
      description: '',
      strategy: values.strategy || 'balanced',
      allow_fallbacks: !!values.allow_fallbacks,
      fallback_strategy: values.fallback_strategy || '',
      max_price: Number(values.max_price) || 0,
      max_latency_ms: parseInt(values.max_latency_ms, 10) || 0,
      min_throughput_tps: Number(values.min_throughput_tps) || 0,
      provider_overrides_json: (values.provider_overrides_json || '').trim(),
      status: 1,
      is_default: false,
      priority: 0,
      targets,
      request_provider_json: (values.request_provider_json || '').trim(),
      request_model: (values.request_model || '').trim(),
    };

    setLoading(true);
    try {
      const res = await API.post(
        '/api/user/routing/policies/dry_run',
        payload,
      );
      const { success, message, data } = res.data;
      if (!success) {
        showError(message);
        return;
      }
      setResult(data || { source: 'none' });
    } catch (err) {
      showError(err?.message || String(err));
    } finally {
      setLoading(false);
    }
  };

  const renderResult = () => {
    if (!result)
      return (
        <Empty
          title={t('尚未演练')}
          description={t('点击「演练」按钮以查看 router-engine 实际生效的偏好。')}
        />
      );
    if (result.source === 'none' || !result.source) {
      return (
        <Banner
          type='warning'
          fullMode={false}
          closeIcon={null}
          description={t(
            '当前组合在 distributor 中不会生效（policy 被禁用 / 请求侧无 provider JSON / 候选池为空）。',
          )}
        />
      );
    }
    return (
      <div className='space-y-3'>
        <div>
          <Text type='tertiary' size='small'>
            {t('来源（distributor 实际取用的合并源）')}
          </Text>
          <div>
            <Tag color='violet'>{SOURCE_LABEL[result.source] || result.source}</Tag>
            {result.fallback_strategy && (
              <Tag color='amber' className='!ml-2'>
                fallback: {result.fallback_strategy}
              </Tag>
            )}
            <Tag
              color={result.allow_fallbacks ? 'green' : 'grey'}
              className='!ml-2'
            >
              allow_fallbacks: {String(result.allow_fallbacks)}
            </Tag>
          </div>
        </div>

        <div>
          <Text type='tertiary' size='small'>
            {t('effective_provider_json（router-engine 实际收到的 prefs）')}
          </Text>
          <Paragraph
            code
            copyable
            className='!my-1'
            style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-all' }}
          >
            {result.effective_provider_json || '(空：router-engine 走默认 1/p² 加权随机)'}
          </Paragraph>
        </div>

        <div>
          <Text type='tertiary' size='small'>
            {t('候选池展开（resolver 内部计算结果，distributor 用此过滤渠道）')}
          </Text>
          <div className='space-y-1 mt-1'>
            <div>
              <Text type='secondary'>{t('渠道白名单')}: </Text>
              {result.channel_allowlist && result.channel_allowlist.length > 0 ? (
                result.channel_allowlist.map((id) => (
                  <Tag key={`c-${id}`} className='!ml-1'>
                    #{id}
                  </Tag>
                ))
              ) : (
                <Text type='tertiary'>{t('不限')}</Text>
              )}
            </div>
            <div>
              <Text type='secondary'>{t('模型白名单')}: </Text>
              {result.model_allowlist && result.model_allowlist.length > 0 ? (
                result.model_allowlist.map((m) => (
                  <Tag key={`m-${m}`} className='!ml-1'>
                    {m}
                  </Tag>
                ))
              ) : (
                <Text type='tertiary'>{t('不限')}</Text>
              )}
            </div>
            <div>
              <Text type='secondary'>{t('精准绑定')}: </Text>
              {result.channel_model_allowlist &&
              result.channel_model_allowlist.length > 0 ? (
                result.channel_model_allowlist.map((cm) => (
                  <Tag key={`cm-${cm}`} className='!ml-1'>
                    {cm}
                  </Tag>
                ))
              ) : (
                <Text type='tertiary'>{t('不限')}</Text>
              )}
            </div>
          </div>
        </div>
      </div>
    );
  };

  return (
    <Modal
      title={t('路由策略演练')}
      visible={visible}
      onCancel={onClose}
      width={840}
      footer={
        <Space>
          <Button onClick={onClose}>{t('关闭')}</Button>
          <Button
            theme='solid'
            type='primary'
            loading={loading}
            onClick={() => formApiRef.current?.submitForm()}
          >
            {t('演练')}
          </Button>
        </Space>
      }
    >
      <Banner
        type='info'
        fullMode={false}
        closeIcon={null}
        description={t(
          'dry-run 不写库；用于在保存 / 客户端发请求前，预览这条策略经 distributor + resolver 处理后真实交给 router-engine 的偏好与候选过滤集合。',
        )}
        className='!mb-3'
      />
      <Form
        layout='vertical'
        initValues={initialValues}
        getFormApi={(api) => (formApiRef.current = api)}
        onSubmit={runDryRun}
      >
        {({ formState }) => (
          <>
            <Space style={{ width: '100%' }} spacing='loose'>
              <Form.Select
                field='strategy'
                label={t('策略类型')}
                optionList={STRATEGY_OPTIONS}
                style={{ width: '100%' }}
              />
              <Form.Select
                field='fallback_strategy'
                label={t('fallback 策略')}
                optionList={FALLBACK_OPTIONS}
                style={{ width: '100%' }}
              />
              <Form.Switch
                field='allow_fallbacks'
                label={t('router 内部 fallback')}
              />
            </Space>

            <Space style={{ width: '100%' }} spacing='loose'>
              <Form.InputNumber
                field='max_price'
                label='max_price'
                min={0}
                precision={6}
                hideButtons
                style={{ width: '100%' }}
              />
              <Form.InputNumber
                field='max_latency_ms'
                label='max_latency_ms'
                min={0}
                hideButtons
                style={{ width: '100%' }}
              />
              <Form.InputNumber
                field='min_throughput_tps'
                label='min_throughput_tps'
                min={0}
                precision={2}
                hideButtons
                style={{ width: '100%' }}
              />
            </Space>

            {formState?.values?.strategy === 'custom' && (
              <Form.TextArea
                field='provider_overrides_json'
                label='provider_overrides_json'
                placeholder='{"sort":"latency"}'
                rows={3}
              />
            )}

            <Form.TextArea
              field='request_provider_json'
              label={t('（可选）模拟请求体里的 provider JSON')}
              placeholder='{"sort":"price","ignore":["channel-x"]}'
              rows={2}
            />
            <Form.Input
              field='request_model'
              label={t('（可选）模拟请求模型名（仅展示用，resolver 不据此过滤）')}
              placeholder='gpt-4o'
              showClear
            />
          </>
        )}
      </Form>

      <div className='mt-4'>
        <Text strong>{t('解析结果')}</Text>
        <div className='mt-2'>
          {loading ? <Spin size='middle' /> : renderResult()}
        </div>
      </div>
    </Modal>
  );
};

export default RoutingPolicyDryRunModal;
