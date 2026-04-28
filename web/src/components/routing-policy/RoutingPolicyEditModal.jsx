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
  ArrayField,
  Banner,
  Button,
  Form,
  Modal,
  Space,
  Typography,
} from '@douyinfe/semi-ui';
import { API, showError, showSuccess } from '../../helpers';

const { Text } = Typography;

const STRATEGY_OPTIONS = [
  { value: 'price', label: '低价优先（router-engine sort=price）' },
  { value: 'latency', label: '低时延优先（router-engine sort=latency）' },
  { value: 'throughput', label: '高吞吐优先（router-engine sort=throughput）' },
  { value: 'balanced', label: '均衡（不显式 sort，1/p² 加权随机）' },
  {
    value: 'custom',
    label: '自定义（直接透传 provider_overrides_json）',
  },
];

const FALLBACK_OPTIONS = [
  { value: '', label: '不兜底（候选耗尽即失败，严格闭环）' },
  { value: 'price', label: '价格兜底（候选外按全局最低价兜一次）' },
  { value: 'latency', label: '时延兜底（候选外按全局最快兜一次）' },
  { value: 'any', label: '任意兜底（router-engine 加权随机）' },
];

const TARGET_TYPE_OPTIONS = [
  { value: 'channel', label: '指定渠道（任意模型）' },
  { value: 'model', label: '指定模型（任意渠道）' },
  { value: 'channel_model', label: '渠道 × 模型（精准绑定）' },
];

// 默认表单值。与后端 toModelRoutingPolicy 的默认值规则保持一致：
//   - allow_fallbacks 默认 true（OpenRouter 兼容）；
//   - status 默认 1（启用）；
//   - is_default 默认 false。
const DEFAULT_FORM = {
  name: '',
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
};

// targetsToFormRows 把后端返回的 targets 列表转换成表单 ArrayField 行；保留 ID 字段
// 仅用于 React key，提交时 toModelRoutingPolicy 会忽略。
function targetsToFormRows(targets) {
  if (!Array.isArray(targets)) return [];
  return targets.map((t) => ({
    target_type: t.target_type || 'channel_model',
    channel_id: t.channel_id ?? 0,
    model_name: t.model_name || '',
  }));
}

// validateTargetRow 在前端做一次基础校验，与后端 validateRoutingTarget 对齐：
// channel 需要 channel_id>0；model 需要 model_name；channel_model 都要。
function validateTargetRow(row) {
  if (!row || !row.target_type) return '请选择候选条目类型';
  switch (row.target_type) {
    case 'channel':
      if (!row.channel_id || row.channel_id <= 0) return '渠道 ID 必须 > 0';
      return '';
    case 'model':
      if (!row.model_name) return '模型名不能为空';
      return '';
    case 'channel_model':
      if (!row.channel_id || row.channel_id <= 0) return '渠道 ID 必须 > 0';
      if (!row.model_name) return '模型名不能为空';
      return '';
    default:
      return '未知的条目类型';
  }
}

// channelsToOptionList 把后端返回的渠道清单转换成 Semi Select 的 optionList。
// label 含 id + 名称便于运维一眼对应；value 用 channel.id 这个数值（与后端 DTO 对齐）。
function channelsToOptionList(channels) {
  return (channels || []).map((ch) => ({
    value: ch.id,
    label: `[${ch.id}] ${ch.name || ''}`,
  }));
}

// channelModelsLookup 把渠道列表转成 channel_id → models[] 的 map，用于 channel_model
// 行的「先选渠道再联动模型」交互，避免用户在 100 个渠道下漫无目的输入。
function channelModelsLookup(channels) {
  const map = new Map();
  (channels || []).forEach((ch) => {
    map.set(ch.id, Array.isArray(ch.models) ? ch.models : []);
  });
  return map;
}

const RoutingPolicyEditModal = ({
  visible,
  editingPolicy,
  onClose,
  onSaved,
}) => {
  const { t } = useTranslation();
  const formApiRef = useRef(null);
  const [submitting, setSubmitting] = useState(false);
  const [userModels, setUserModels] = useState([]);
  const [userChannels, setUserChannels] = useState([]);

  const isEdit = Boolean(editingPolicy?.id);

  // 渠道下拉的 optionList + (channel_id → models[]) 联动表；channels 数据源稳定时
  // useMemo 复用引用，避免每次渲染都新建数组让 Select 频繁刷新。
  const channelOptions = useMemo(
    () => channelsToOptionList(userChannels),
    [userChannels],
  );
  const channelModelsMap = useMemo(
    () => channelModelsLookup(userChannels),
    [userChannels],
  );

  // 一次性并发拉「可用模型」+「可用渠道」；二者都用于候选池下拉，互不依赖。
  // Promise.allSettled 容忍其中一个接口失败：拉不到就退化为输入框，不阻塞表单。
  useEffect(() => {
    if (!visible) return;
    let aborted = false;
    (async () => {
      const results = await Promise.allSettled([
        API.get('/api/user/models'),
        API.get('/api/user/routing/channels'),
      ]);
      if (aborted) return;
      const [modelsRes, channelsRes] = results;
      if (modelsRes.status === 'fulfilled') {
        const list = Array.isArray(modelsRes.value?.data?.data)
          ? modelsRes.value.data.data
          : [];
        setUserModels(list);
      }
      if (channelsRes.status === 'fulfilled') {
        const list = Array.isArray(channelsRes.value?.data?.data)
          ? channelsRes.value.data.data
          : [];
        setUserChannels(list);
      }
    })();
    return () => {
      aborted = true;
    };
  }, [visible]);

  // 把后端实体翻译成表单初值；不存在则用默认值。
  const initialValues = useMemo(() => {
    if (!editingPolicy) return DEFAULT_FORM;
    return {
      ...DEFAULT_FORM,
      name: editingPolicy.name || '',
      description: editingPolicy.description || '',
      strategy: editingPolicy.strategy || 'balanced',
      allow_fallbacks: editingPolicy.allow_fallbacks ?? true,
      fallback_strategy: editingPolicy.fallback_strategy ?? '',
      max_price: editingPolicy.max_price ?? 0,
      max_latency_ms: editingPolicy.max_latency_ms ?? 0,
      min_throughput_tps: editingPolicy.min_throughput_tps ?? 0,
      provider_overrides_json: editingPolicy.provider_overrides_json || '',
      status: editingPolicy.status ?? 1,
      is_default: editingPolicy.is_default ?? false,
      priority: editingPolicy.priority ?? 0,
      targets: targetsToFormRows(editingPolicy.targets),
    };
  }, [editingPolicy]);

  // visible 切换或编辑对象切换时强制重置表单，避免残留上次填的值。
  useEffect(() => {
    if (!visible) return;
    if (formApiRef.current) {
      formApiRef.current.setValues(initialValues, { isOverride: true });
    }
  }, [visible, initialValues]);

  const submit = async (values) => {
    if (!values.name || !values.name.trim()) {
      showError(t('请填写策略名'));
      return;
    }
    if (Array.isArray(values.targets)) {
      for (let i = 0; i < values.targets.length; i++) {
        const err = validateTargetRow(values.targets[i]);
        if (err) {
          showError(`${t('候选条目第')} ${i + 1} ${t('行')}：${err}`);
          return;
        }
      }
    }
    if (
      values.strategy === 'custom' &&
      values.provider_overrides_json &&
      values.provider_overrides_json.trim()
    ) {
      try {
        JSON.parse(values.provider_overrides_json);
      } catch (e) {
        showError(
          `${t('provider_overrides_json 不是合法 JSON')}: ${e.message}`,
        );
        return;
      }
    }

    // 构造提交体：targets 中的 channel_id / model_name 按 target_type 规整。
    // 与后端 validateRoutingTarget 对齐——避免传错 0 / '' 引发 ErrRoutingPolicyInvalidTarget。
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
      name: (values.name || '').trim(),
      description: values.description || '',
      strategy: values.strategy || 'balanced',
      allow_fallbacks: !!values.allow_fallbacks,
      fallback_strategy: values.fallback_strategy || '',
      max_price: Number(values.max_price) || 0,
      max_latency_ms: parseInt(values.max_latency_ms, 10) || 0,
      min_throughput_tps: Number(values.min_throughput_tps) || 0,
      provider_overrides_json: (values.provider_overrides_json || '').trim(),
      status: parseInt(values.status, 10) === 0 ? 0 : 1,
      is_default: !!values.is_default,
      priority: parseInt(values.priority, 10) || 0,
      targets,
    };

    setSubmitting(true);
    try {
      const res = isEdit
        ? await API.put(
            `/api/user/routing/policies/${editingPolicy.id}`,
            payload,
          )
        : await API.post('/api/user/routing/policies', payload);
      const { success, message } = res.data;
      if (!success) {
        showError(message);
        return;
      }
      showSuccess(isEdit ? t('已更新') : t('已创建'));
      onSaved && onSaved(res.data?.data);
    } catch (err) {
      showError(err?.message || String(err));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Modal
      title={isEdit ? t('编辑路由策略') : t('新建路由策略')}
      visible={visible}
      onCancel={onClose}
      width={760}
      footer={
        <Space>
          <Button onClick={onClose}>{t('取消')}</Button>
          <Button
            theme='solid'
            type='primary'
            loading={submitting}
            onClick={() => formApiRef.current?.submitForm()}
          >
            {t('保存')}
          </Button>
        </Space>
      }
    >
      <Form
        layout='vertical'
        initValues={initialValues}
        getFormApi={(api) => (formApiRef.current = api)}
        onSubmit={submit}
      >
        {({ formState }) => (
          <>
            <Form.Input
              field='name'
              label={t('策略名')}
              placeholder={t('如：低价优先 / Claude 3.5 专用')}
              maxLength={128}
              showClear
              rules={[{ required: true, message: t('请填写策略名') }]}
            />

            <Form.TextArea
              field='description'
              label={t('描述')}
              placeholder={t('可选：让自己半年后还看得懂这条策略')}
              rows={2}
              maxCount={500}
            />

            <Form.Select
              field='strategy'
              label={t('策略类型')}
              optionList={STRATEGY_OPTIONS}
              style={{ width: '100%' }}
            />

            {formState?.values?.strategy === 'custom' && (
              <Form.TextArea
                field='provider_overrides_json'
                label={t(
                  'provider_overrides_json（合法 JSON，自定义策略时直接透传）',
                )}
                placeholder='{"sort":"latency","ignore":["channel-foo"]}'
                rows={4}
              />
            )}

            <Form.Switch
              field='allow_fallbacks'
              label={t('router 内部 fallback（候选间失败时换下一个）')}
            />

            <Form.Select
              field='fallback_strategy'
              label={t('候选池整体不可用时的兜底策略')}
              optionList={FALLBACK_OPTIONS}
              style={{ width: '100%' }}
            />

            <Banner
              type='info'
              fullMode={false}
              closeIcon={null}
              description={t(
                '阈值为 0 表示不启用该项过滤；启用后会在 router-engine 之前剔除不达标渠道。',
              )}
              className='!my-2'
            />

            <Space spacing='loose' style={{ width: '100%' }}>
              <Form.InputNumber
                field='max_price'
                label={t('max_price（按补全单价过滤）')}
                min={0}
                step={0.000001}
                precision={6}
                hideButtons
                style={{ width: '100%' }}
              />
              <Form.InputNumber
                field='max_latency_ms'
                label={t('max_latency_ms')}
                min={0}
                step={50}
                hideButtons
                style={{ width: '100%' }}
              />
              <Form.InputNumber
                field='min_throughput_tps'
                label={t('min_throughput_tps')}
                min={0}
                step={1}
                precision={2}
                hideButtons
                style={{ width: '100%' }}
              />
            </Space>

            <Form.RadioGroup
              field='status'
              label={t('状态')}
              type='button'
              options={[
                { label: t('启用'), value: 1 },
                { label: t('禁用'), value: 0 },
              ]}
            />

            <Form.Switch
              field='is_default'
              label={t('设为默认（同账号同时仅有一条默认策略）')}
            />

            <div className='mt-2'>
              <Text strong>{t('候选池（不填表示不限制）')}</Text>
              <div>
                <Text type='tertiary' size='small'>
                  {t(
                    '三种条目可混合：channel 锁渠道、model 锁模型、channel_model 精准绑定。条目之间是「或」的关系。',
                  )}
                </Text>
              </div>
            </div>

            <ArrayField field='targets'>
              {({ add, arrayFields }) => (
                <div className='space-y-2'>
                  {arrayFields.map(({ field, key, remove }, idx) => {
                    // 当前行字段值——用 formState 实时读，确保 channel 改完模型联动跟着变。
                    const row = formState?.values?.targets?.[idx] || {};
                    const targetType = row.target_type || 'channel_model';
                    const showChannel =
                      targetType === 'channel' ||
                      targetType === 'channel_model';
                    const showModel =
                      targetType === 'model' || targetType === 'channel_model';
                    // model 模式下用全局可用模型集合；channel_model 模式下按所选 channel 过滤
                    // ——大幅减少误填，比如「在不支持 GPT-4o 的渠道里选了 gpt-4o」这种事故。
                    let modelData = userModels;
                    if (targetType === 'channel_model' && row.channel_id) {
                      modelData = channelModelsMap.get(row.channel_id) || [];
                    }
                    return (
                      <div
                        key={key}
                        className='border border-[var(--semi-color-border)] rounded-lg p-3'
                      >
                        <Space style={{ width: '100%' }} align='end' wrap>
                          <Form.Select
                            field={`${field}.target_type`}
                            label={idx === 0 ? t('类型') : ''}
                            optionList={TARGET_TYPE_OPTIONS}
                            style={{ width: 200 }}
                          />
                          {showChannel && (
                            <Form.Select
                              field={`${field}.channel_id`}
                              label={idx === 0 ? t('渠道') : ''}
                              optionList={channelOptions}
                              placeholder={t('选择渠道（按 group 过滤）')}
                              filter
                              showClear
                              emptyContent={t(
                                '当前用户组下暂无可用渠道，请联系管理员',
                              )}
                              style={{ width: 240 }}
                            />
                          )}
                          {showModel && (
                            <Form.AutoComplete
                              field={`${field}.model_name`}
                              label={idx === 0 ? t('模型名') : ''}
                              data={modelData}
                              placeholder={
                                targetType === 'channel_model' && row.channel_id
                                  ? t('该渠道下可选模型')
                                  : t('如 gpt-4o')
                              }
                              style={{ width: 280 }}
                              showClear
                            />
                          )}
                          <Button
                            type='danger'
                            theme='light'
                            onClick={() => remove()}
                          >
                            {t('删除')}
                          </Button>
                        </Space>
                      </div>
                    );
                  })}
                  <Button
                    onClick={() =>
                      add({
                        target_type: 'channel_model',
                        channel_id: 0,
                        model_name: '',
                      })
                    }
                  >
                    {t('+ 新增候选条目')}
                  </Button>
                </div>
              )}
            </ArrayField>
          </>
        )}
      </Form>
    </Modal>
  );
};

export default RoutingPolicyEditModal;
