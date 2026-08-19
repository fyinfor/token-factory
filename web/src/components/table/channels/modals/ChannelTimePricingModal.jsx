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

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Banner,
  Button,
  Checkbox,
  CheckboxGroup,
  DatePicker,
  Input,
  InputNumber,
  Modal,
  Select,
  Space,
  Spin,
  Switch,
  Table,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import {
  IconCalendarClock,
  IconDelete,
  IconEdit,
  IconPlus,
  IconRefresh,
} from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../../../helpers';

const { Text } = Typography;

const WEEKDAY_OPTIONS = [
  { value: 1, label: '周一' },
  { value: 2, label: '周二' },
  { value: 3, label: '周三' },
  { value: 4, label: '周四' },
  { value: 5, label: '周五' },
  { value: 6, label: '周六' },
  { value: 0, label: '周日' },
];

const DEFAULT_RATES = {
  price_discount_percent: 100,
  operating_cost_percent: 0,
  markup_discount_rate: 0,
};

const normalizeRate = (value, fallback = 0) => {
  const numeric = Number(value);
  if (!Number.isFinite(numeric)) return fallback;
  return Math.min(1000, Math.max(0, numeric));
};

const timeToMinute = (value, allow24 = false) => {
  const text = String(value || '').trim();
  const matched = /^([01]\d|2[0-3]):([0-5]\d)$/.exec(text);
  if (matched) return Number(matched[1]) * 60 + Number(matched[2]);
  if (allow24 && text === '24:00') return 1440;
  return null;
};

const minuteToTime = (minute) => {
  const numeric = Number(minute);
  if (numeric === 1440) return '24:00';
  if (!Number.isFinite(numeric) || numeric < 0 || numeric >= 1440) {
    return '00:00';
  }
  return `${String(Math.floor(numeric / 60)).padStart(2, '0')}:${String(numeric % 60).padStart(2, '0')}`;
};

const weekdaysToMask = (weekdays) =>
  (weekdays || []).reduce((mask, weekday) => mask | (1 << Number(weekday)), 0);

const maskToWeekdays = (mask) =>
  WEEKDAY_OPTIONS.map((item) => item.value).filter(
    (weekday) => (Number(mask) & (1 << weekday)) !== 0,
  );

const weekdayLabel = (mask, t) => {
  if (Number(mask) === 0x7f) return t('每天');
  if (Number(mask) === 0x3e) return t('工作日');
  return WEEKDAY_OPTIONS.filter(
    (item) => (Number(mask) & (1 << item.value)) !== 0,
  )
    .map((item) => t(item.label))
    .join('、');
};

const apiMessage = (error, fallback) =>
  error?.response?.data?.message || error?.message || fallback;

const createDraft = (models, rates) => ({
  scheduleId: 0,
  modelNames: models.length > 0 ? [models[0]] : [],
  name: '',
  priceDiscountPercent: normalizeRate(
    rates?.price_discount_percent,
    DEFAULT_RATES.price_discount_percent,
  ),
  operatingCostPercent: normalizeRate(
    rates?.operating_cost_percent,
    DEFAULT_RATES.operating_cost_percent,
  ),
  markupDiscountRate: normalizeRate(
    rates?.markup_discount_rate,
    DEFAULT_RATES.markup_discount_rate,
  ),
  weekdays: [1, 2, 3, 4, 5],
  startTime: '18:00',
  endTime: '23:00',
  effectiveFrom: '',
  effectiveTo: '',
  enabled: true,
});

export default function ChannelTimePricingModal({
  visible,
  channelId,
  channelName,
  models = [],
  onCancel,
}) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [rules, setRules] = useState([]);
  const [channelRates, setChannelRates] = useState(DEFAULT_RATES);
  const [editorVisible, setEditorVisible] = useState(false);
  const [draft, setDraft] = useState(() => createDraft([], DEFAULT_RATES));

  const normalizedModels = useMemo(
    () =>
      Array.from(
        new Set(
          (models || [])
            .map((model) => String(model || '').trim())
            .filter(Boolean),
        ),
      ),
    [models],
  );

  const endpoint = useMemo(
    () => `/api/user/channel-pricing/${channelId}/time-pricing/rules`,
    [channelId],
  );

  const loadRules = useCallback(async () => {
    if (!visible || !channelId) return;
    setLoading(true);
    try {
      const response = await API.get(endpoint);
      if (!response?.data?.success) {
        throw new Error(response?.data?.message || t('加载动态费率失败'));
      }
      const data = response.data.data || {};
      setRules(Array.isArray(data.rules) ? data.rules : []);
      setChannelRates({
        price_discount_percent: normalizeRate(
          data.channel_rates?.price_discount_percent,
          100,
        ),
        operating_cost_percent: normalizeRate(
          data.channel_rates?.operating_cost_percent,
          0,
        ),
        markup_discount_rate: normalizeRate(
          data.channel_rates?.markup_discount_rate,
          0,
        ),
      });
    } catch (error) {
      showError(apiMessage(error, t('加载动态费率失败')));
    } finally {
      setLoading(false);
    }
  }, [channelId, endpoint, t, visible]);

  useEffect(() => {
    if (visible) loadRules();
  }, [loadRules, visible]);

  const openCreate = useCallback(() => {
    if (normalizedModels.length === 0) {
      showError(t('该渠道还没有已保存的模型'));
      return;
    }
    setDraft(createDraft(normalizedModels, channelRates));
    setEditorVisible(true);
  }, [channelRates, normalizedModels, t]);

  const openEdit = useCallback((rule) => {
    setDraft({
      scheduleId: Number(rule.schedule_id),
      modelNames: [rule.model_name],
      name: rule.name || '',
      priceDiscountPercent: normalizeRate(rule.price_discount_percent, 100),
      operatingCostPercent: normalizeRate(rule.operating_cost_percent, 0),
      markupDiscountRate: normalizeRate(rule.markup_discount_rate, 0),
      weekdays: maskToWeekdays(rule.weekdays),
      startTime: minuteToTime(rule.start_minute),
      endTime: minuteToTime(rule.end_minute),
      effectiveFrom: rule.effective_from || '',
      effectiveTo: rule.effective_to || '',
      enabled: Boolean(rule.enabled),
    });
    setEditorVisible(true);
  }, []);

  const buildBody = useCallback(
    (sourceDraft, enabled = sourceDraft.enabled) => ({
      model_names: sourceDraft.modelNames,
      name: sourceDraft.name.trim(),
      price_discount_percent: normalizeRate(
        sourceDraft.priceDiscountPercent,
        100,
      ),
      operating_cost_percent: normalizeRate(
        sourceDraft.operatingCostPercent,
        0,
      ),
      markup_discount_rate: normalizeRate(sourceDraft.markupDiscountRate, 0),
      timezone: 'Asia/Shanghai',
      weekdays: weekdaysToMask(sourceDraft.weekdays),
      start_minute: timeToMinute(sourceDraft.startTime),
      end_minute: timeToMinute(sourceDraft.endTime, true),
      effective_from: sourceDraft.effectiveFrom.trim(),
      effective_to: sourceDraft.effectiveTo.trim(),
      enabled,
    }),
    [],
  );

  const saveRule = useCallback(async () => {
    const body = buildBody(draft);
    if (body.model_names.length === 0) {
      showError(t('请至少选择一个模型'));
      return;
    }
    if (!body.name) {
      showError(t('请输入规则名称'));
      return;
    }
    if (!draft.weekdays.length) {
      showError(t('请至少选择一天'));
      return;
    }
    if (
      body.start_minute === null ||
      body.end_minute === null ||
      body.start_minute === body.end_minute
    ) {
      showError(t('请输入有效的开始和结束时间'));
      return;
    }
    setSaving(true);
    try {
      const response = draft.scheduleId
        ? await API.put(`${endpoint}/${draft.scheduleId}`, body)
        : await API.post(endpoint, body);
      if (!response?.data?.success) {
        throw new Error(response?.data?.message || t('保存动态费率失败'));
      }
      setEditorVisible(false);
      await loadRules();
      showSuccess(t('动态费率已保存'));
    } catch (error) {
      showError(apiMessage(error, t('保存动态费率失败')));
    } finally {
      setSaving(false);
    }
  }, [buildBody, draft, endpoint, loadRules, t]);

  const toggleRule = useCallback(
    async (rule) => {
      const sourceDraft = {
        modelNames: [rule.model_name],
        name: rule.name || '',
        priceDiscountPercent: rule.price_discount_percent,
        operatingCostPercent: rule.operating_cost_percent,
        markupDiscountRate: rule.markup_discount_rate,
        weekdays: maskToWeekdays(rule.weekdays),
        startTime: minuteToTime(rule.start_minute),
        endTime: minuteToTime(rule.end_minute),
        effectiveFrom: rule.effective_from || '',
        effectiveTo: rule.effective_to || '',
        enabled: Boolean(rule.enabled),
      };
      try {
        const response = await API.put(
          `${endpoint}/${rule.schedule_id}`,
          buildBody(sourceDraft, !rule.enabled),
        );
        if (!response?.data?.success) {
          throw new Error(response?.data?.message || t('更新动态费率失败'));
        }
        await loadRules();
      } catch (error) {
        showError(apiMessage(error, t('更新动态费率失败')));
      }
    },
    [buildBody, endpoint, loadRules, t],
  );

  const deleteRule = useCallback(
    (rule) => {
      Modal.confirm({
        title: t('删除动态费率'),
        content: t('删除后，该模型在此时段将恢复使用渠道常规费率。'),
        okType: 'danger',
        onOk: async () => {
          try {
            const response = await API.delete(
              `${endpoint}/${rule.schedule_id}`,
            );
            if (!response?.data?.success) {
              throw new Error(response?.data?.message || t('删除动态费率失败'));
            }
            await loadRules();
            showSuccess(t('动态费率已删除'));
          } catch (error) {
            showError(apiMessage(error, t('删除动态费率失败')));
            throw error;
          }
        },
      });
    },
    [endpoint, loadRules, t],
  );

  const columns = useMemo(
    () => [
      {
        title: t('模型'),
        dataIndex: 'model_name',
        width: 220,
        render: (value) => <Text code>{value}</Text>,
      },
      {
        title: t('规则'),
        dataIndex: 'name',
        render: (value, rule) => (
          <Space wrap>
            <span>{value}</span>
            {rule.active ? <Tag color='green'>{t('当前生效')}</Tag> : null}
            <Tag color='blue'>v{rule.plan_version || 1}</Tag>
          </Space>
        ),
      },
      {
        title: t('生效时段'),
        width: 230,
        render: (_, rule) => (
          <div>
            <div>{weekdayLabel(rule.weekdays, t)}</div>
            <div className='text-xs text-gray-500'>
              {minuteToTime(rule.start_minute)}–{minuteToTime(rule.end_minute)}
            </div>
            {rule.effective_from || rule.effective_to ? (
              <div className='text-xs text-gray-500'>
                {rule.effective_from || '不限'} ~ {rule.effective_to || '不限'}
              </div>
            ) : null}
          </div>
        ),
      },
      {
        title: t('动态费率'),
        width: 210,
        render: (_, rule) => (
          <div className='text-xs leading-5'>
            <div>
              {t('成本')} {rule.price_discount_percent}% + {t('经营')}{' '}
              {rule.operating_cost_percent}%
            </div>
            <div>
              {t('最终成本率')} {rule.effective_cost_percent}% · {t('加价')}{' '}
              {rule.markup_discount_rate}%
            </div>
          </div>
        ),
      },
      {
        title: t('状态'),
        width: 80,
        render: (_, rule) => (
          <Switch
            checked={Boolean(rule.enabled)}
            onChange={() => toggleRule(rule)}
          />
        ),
      },
      {
        title: t('操作'),
        width: 110,
        render: (_, rule) => (
          <Space spacing='tight'>
            <Button icon={<IconEdit />} onClick={() => openEdit(rule)} />
            <Button
              type='danger'
              icon={<IconDelete />}
              onClick={() => deleteRule(rule)}
            />
          </Space>
        ),
      },
    ],
    [deleteRule, openEdit, t, toggleRule],
  );

  const effectiveCost = Math.min(
    1000,
    normalizeRate(draft.priceDiscountPercent, 0) +
      normalizeRate(draft.operatingCostPercent, 0),
  );

  return (
    <>
      <Modal
        visible={visible}
        onCancel={onCancel}
        title={
          <Space>
            <IconCalendarClock />
            <span>{t('动态费率')}</span>
            {channelName ? <Tag color='blue'>{channelName}</Tag> : null}
          </Space>
        }
        footer={
          <Button type='primary' onClick={onCancel}>
            {t('完成')}
          </Button>
        }
        width='min(1180px, 96vw)'
        bodyStyle={{ maxHeight: '76vh', overflowY: 'auto' }}
        closeOnEsc={false}
      >
        <Banner
          type='info'
          description={t(
            '动态费率按渠道下的指定模型生效。命中时继续使用模型的渠道常规价，只临时覆盖成本折扣率、经营成本率和加价折扣率；时区固定为 Asia/Shanghai。',
          )}
          style={{ marginBottom: 16 }}
        />
        <div className='mb-4 flex flex-wrap items-center justify-between gap-3'>
          <div className='text-sm text-gray-500'>
            {t('渠道当前默认值')}：{channelRates.price_discount_percent}% +{' '}
            {channelRates.operating_cost_percent}% /{' '}
            {channelRates.markup_discount_rate}%
          </div>
          <Space>
            <Button
              icon={<IconRefresh />}
              loading={loading}
              onClick={loadRules}
            >
              {t('刷新')}
            </Button>
            <Button type='primary' icon={<IconPlus />} onClick={openCreate}>
              {t('添加动态费率')}
            </Button>
          </Space>
        </div>
        <Spin spinning={loading} style={{ width: '100%' }}>
          <Table
            rowKey='schedule_id'
            columns={columns}
            dataSource={rules}
            pagination={false}
            style={{ width: '100%' }}
            scroll={{ x: '100%' }}
            empty={t('暂无动态费率规则')}
          />
        </Spin>
      </Modal>

      <Modal
        visible={editorVisible}
        onCancel={() => setEditorVisible(false)}
        onOk={saveRule}
        confirmLoading={saving}
        title={draft.scheduleId ? t('编辑动态费率') : t('添加动态费率')}
        okText={t('保存')}
        width={720}
        style={{ maxWidth: 'calc(100vw - 32px)' }}
        closeOnEsc={false}
      >
        <div className='min-w-0 space-y-3'>
          <div>
            <div className='mb-2 font-medium'>{t('生效模型')}</div>
            <Select
              multiple={!draft.scheduleId}
              value={draft.scheduleId ? draft.modelNames[0] : draft.modelNames}
              disabled={Boolean(draft.scheduleId)}
              optionList={normalizedModels.map((model) => ({
                label: model,
                value: model,
              }))}
              filter
              style={{ width: '100%' }}
              placeholder={t('请选择模型')}
              onChange={(value) =>
                setDraft((current) => ({
                  ...current,
                  modelNames: Array.isArray(value) ? value : [value],
                }))
              }
            />
            {!draft.scheduleId ? (
              <div className='mt-1 text-xs text-gray-500'>
                {t('可同时选择多个模型，系统会为每个模型创建相同规则。')}
              </div>
            ) : null}
          </div>
          <div>
            <div className='mb-2 font-medium'>{t('规则名称')}</div>
            <Input
              value={draft.name}
              maxLength={128}
              onChange={(value) =>
                setDraft((current) => ({ ...current, name: value }))
              }
            />
          </div>
          <div className='grid grid-cols-1 gap-4 md:grid-cols-3'>
            <div>
              <div className='mb-2 font-medium'>{t('成本折扣率(%)')}</div>
              <InputNumber
                value={draft.priceDiscountPercent}
                min={0}
                max={1000}
                precision={2}
                style={{ width: '100%' }}
                onChange={(value) =>
                  setDraft((current) => ({
                    ...current,
                    priceDiscountPercent: normalizeRate(value, 0),
                  }))
                }
              />
            </div>
            <div>
              <div className='mb-2 font-medium'>{t('经营成本率(%)')}</div>
              <InputNumber
                value={draft.operatingCostPercent}
                min={0}
                max={1000}
                precision={2}
                style={{ width: '100%' }}
                onChange={(value) =>
                  setDraft((current) => ({
                    ...current,
                    operatingCostPercent: normalizeRate(value, 0),
                  }))
                }
              />
            </div>
            <div>
              <div className='mb-2 font-medium'>{t('加价折扣率(%)')}</div>
              <InputNumber
                value={draft.markupDiscountRate}
                min={0}
                max={1000}
                precision={2}
                style={{ width: '100%' }}
                onChange={(value) =>
                  setDraft((current) => ({
                    ...current,
                    markupDiscountRate: normalizeRate(value, 0),
                  }))
                }
              />
            </div>
          </div>
          <div className='rounded-lg bg-gray-50 p-3 text-sm text-gray-600 dark:bg-gray-800'>
            {t('最终成本率')}：<strong>{effectiveCost}%</strong>
            <span className='ml-3'>
              {t('最终价格')} = {t('渠道常规价')} × {effectiveCost}% +{' '}
              {t('全局官方价')} × {normalizeRate(draft.markupDiscountRate, 0)}%
            </span>
          </div>
          <div>
            <div className='mb-2 font-medium'>{t('重复日期')}</div>
            <CheckboxGroup
              value={draft.weekdays}
              onChange={(weekdays) =>
                setDraft((current) => ({ ...current, weekdays }))
              }
              direction='horizontal'
            >
              {WEEKDAY_OPTIONS.map((item) => (
                <Checkbox key={item.value} value={item.value}>
                  {t(item.label)}
                </Checkbox>
              ))}
            </CheckboxGroup>
          </div>
          <div>
            <div className='mb-2 font-medium'>{t('时间范围')}</div>
            <div className='grid grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)] items-center gap-2'>
              <Input
                value={draft.startTime}
                placeholder='18:00'
                style={{ width: '100%', minWidth: 0 }}
                onChange={(value) =>
                  setDraft((current) => ({ ...current, startTime: value }))
                }
              />
              <span>–</span>
              <Input
                value={draft.endTime}
                placeholder='23:00'
                style={{ width: '100%', minWidth: 0 }}
                onChange={(value) =>
                  setDraft((current) => ({ ...current, endTime: value }))
                }
              />
            </div>
            <div className='mt-1 text-xs text-gray-500'>
              {t('支持跨天，例如 22:00–02:00；结束时间不包含在时段内。')}
            </div>
          </div>
          <div>
            <div className='mb-2 font-medium'>{t('生效日期（可选）')}</div>
            <div className='grid grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)] items-center gap-2'>
              <DatePicker
                type='date'
                value={draft.effectiveFrom || undefined}
                format='yyyy-MM-dd'
                placeholder={t('请选择日期')}
                showClear
                inputReadOnly
                style={{ width: '100%', minWidth: 0 }}
                onChange={(_, value) =>
                  setDraft((current) => ({
                    ...current,
                    effectiveFrom: typeof value === 'string' ? value : '',
                  }))
                }
              />
              <span>~</span>
              <DatePicker
                type='date'
                value={draft.effectiveTo || undefined}
                format='yyyy-MM-dd'
                placeholder={t('请选择日期')}
                showClear
                inputReadOnly
                style={{ width: '100%', minWidth: 0 }}
                onChange={(_, value) =>
                  setDraft((current) => ({
                    ...current,
                    effectiveTo: typeof value === 'string' ? value : '',
                  }))
                }
              />
            </div>
          </div>
          <div className='flex items-center justify-between'>
            <span className='font-medium'>{t('启用规则')}</span>
            <Switch
              checked={draft.enabled}
              onChange={(enabled) =>
                setDraft((current) => ({ ...current, enabled }))
              }
            />
          </div>
        </div>
      </Modal>
    </>
  );
}
