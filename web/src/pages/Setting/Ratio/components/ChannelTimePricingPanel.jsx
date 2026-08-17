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
import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import {
  Button,
  Card,
  Checkbox,
  CheckboxGroup,
  Empty,
  Input,
  InputNumber,
  Modal,
  Select,
  Space,
  Spin,
  Switch,
  Table,
  Tag,
} from '@douyinfe/semi-ui';
import {
  IconDelete,
  IconEdit,
  IconPlus,
  IconRefresh,
} from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../../../helpers';
import ModelPricingEditor from './ModelPricingEditor';

const PRICING_KEYS = [
  'ModelPrice',
  'ModelRatio',
  'CompletionRatio',
  'CacheRatio',
  'CreateCacheRatio',
  'ImageRatio',
  'AudioRatio',
  'AudioCompletionRatio',
  'VideoRatio',
  'VideoCompletionRatio',
  'VideoPrice',
  'VideoPricingRules',
  'ImagePrice',
  'ImagePricingRules',
  'ASRPrice',
  'ModelRequestTierPricing',
];

const PLAN_MODE_PRICE = 'price';
const PLAN_MODE_RATE = 'rate';

const DEFAULT_CHANNEL_RATES = {
  price_discount_percent: 100,
  operating_cost_percent: 0,
  effective_cost_percent: 100,
  markup_discount_rate: 0,
};

const resolvePlanMode = (pricing) =>
  pricing?.Mode === PLAN_MODE_RATE ? PLAN_MODE_RATE : PLAN_MODE_PRICE;

const normalizeRateValue = (value, fallback = 0) => {
  const numeric = Number(value);
  if (!Number.isFinite(numeric)) return fallback;
  return Math.min(1000, Math.max(0, numeric));
};

const WEEKDAY_OPTIONS = [
  { value: 1, label: '周一' },
  { value: 2, label: '周二' },
  { value: 3, label: '周三' },
  { value: 4, label: '周四' },
  { value: 5, label: '周五' },
  { value: 6, label: '周六' },
  { value: 0, label: '周日' },
];

const DEFAULT_SCHEDULE_DRAFT = {
  id: 0,
  name: '工作日晚高峰',
  pricePlanId: 0,
  weekdays: [1, 2, 3, 4, 5],
  startTime: '18:00',
  endTime: '23:00',
  enabled: true,
};

const parseOptionMap = (raw) => {
  if (!raw || typeof raw !== 'string') return {};
  try {
    const value = JSON.parse(raw);
    return value && typeof value === 'object' && !Array.isArray(value)
      ? value
      : {};
  } catch {
    return {};
  }
};

const editorOptionsFromRegularPrice = (options, modelName) => {
  const next = {
    USDExchangeRate: options?.USDExchangeRate,
    'general_setting.custom_currency_exchange_rate':
      options?.['general_setting.custom_currency_exchange_rate'],
    'general_setting.quota_display_type':
      options?.['general_setting.quota_display_type'],
  };
  for (const key of PRICING_KEYS) {
    const source = parseOptionMap(options?.[key]);
    next[key] = JSON.stringify(
      Object.prototype.hasOwnProperty.call(source, modelName)
        ? { [modelName]: source[modelName] }
        : {},
    );
  }
  return next;
};

const editorOptionsFromPlan = (options, modelName, pricing) => {
  const next = editorOptionsFromRegularPrice(options, modelName);
  for (const key of PRICING_KEYS) {
    next[key] = JSON.stringify(
      Object.prototype.hasOwnProperty.call(pricing || {}, key)
        ? { [modelName]: pricing[key] }
        : {},
    );
  }
  return next;
};

const pricingPayloadFromEditorOutput = (output, modelName) => {
  const pricing = {};
  for (const key of PRICING_KEYS) {
    const value = output?.[key]?.[modelName];
    if (value !== undefined && value !== null) {
      pricing[key] = value;
    }
  }
  return pricing;
};

const timeToMinute = (value, allow24 = false) => {
  const matched = /^([01]\d|2[0-3]):([0-5]\d)$/.exec(
    String(value || '').trim(),
  );
  if (matched) {
    return Number(matched[1]) * 60 + Number(matched[2]);
  }
  if (allow24 && String(value || '').trim() === '24:00') return 1440;
  return null;
};

const minuteToTime = (minute) => {
  const value = Number(minute);
  if (value === 1440) return '24:00';
  if (!Number.isFinite(value) || value < 0 || value >= 1440) return '00:00';
  return `${String(Math.floor(value / 60)).padStart(2, '0')}:${String(value % 60).padStart(2, '0')}`;
};

const weekdaysToMask = (weekdays) =>
  (weekdays || []).reduce((mask, weekday) => mask | (1 << Number(weekday)), 0);

const maskToWeekdays = (mask) =>
  WEEKDAY_OPTIONS.map((item) => item.value).filter(
    (weekday) => (Number(mask) & (1 << weekday)) !== 0,
  );

const apiMessage = (error, fallback) =>
  error?.response?.data?.message || error?.message || fallback;

export default function ChannelTimePricingPanel({
  channelId,
  modelName,
  options,
}) {
  const { t } = useTranslation();
  const requestSequence = useRef(0);
  const [loading, setLoading] = useState(false);
  const [plans, setPlans] = useState([]);
  const [schedules, setSchedules] = useState([]);
  const [channelRates, setChannelRates] = useState(DEFAULT_CHANNEL_RATES);
  const [activePlanId, setActivePlanId] = useState(0);
  const [activeScheduleId, setActiveScheduleId] = useState(0);
  const [planEditor, setPlanEditor] = useState(null);
  const [planName, setPlanName] = useState('');
  const [rateDraft, setRateDraft] = useState(DEFAULT_CHANNEL_RATES);
  const [rateSaving, setRateSaving] = useState(false);
  const [scheduleVisible, setScheduleVisible] = useState(false);
  const [scheduleDraft, setScheduleDraft] = useState(DEFAULT_SCHEDULE_DRAFT);
  const [scheduleSaving, setScheduleSaving] = useState(false);

  const endpoint = useMemo(
    () => `/api/user/channel-pricing/${channelId}/time-pricing`,
    [channelId],
  );

  const loadTimePricing = useCallback(async () => {
    if (!channelId || !modelName) {
      setPlans([]);
      setSchedules([]);
      setActivePlanId(0);
      setActiveScheduleId(0);
      setChannelRates(DEFAULT_CHANNEL_RATES);
      return;
    }
    const sequence = ++requestSequence.current;
    setLoading(true);
    try {
      const response = await API.get(endpoint, {
        params: { model_name: modelName },
      });
      if (!response?.data?.success) {
        throw new Error(response?.data?.message || t('加载分时定价失败'));
      }
      if (sequence !== requestSequence.current) return;
      const data = response.data.data || {};
      setPlans(Array.isArray(data.plans) ? data.plans : []);
      setSchedules(Array.isArray(data.schedules) ? data.schedules : []);
      setActivePlanId(Number(data.active_plan_id || 0));
      setActiveScheduleId(Number(data.active_schedule_id || 0));
      setChannelRates({
        price_discount_percent: normalizeRateValue(
          data.channel_rates?.price_discount_percent,
          100,
        ),
        operating_cost_percent: normalizeRateValue(
          data.channel_rates?.operating_cost_percent,
          0,
        ),
        effective_cost_percent: normalizeRateValue(
          data.channel_rates?.effective_cost_percent,
          100,
        ),
        markup_discount_rate: normalizeRateValue(
          data.channel_rates?.markup_discount_rate,
          0,
        ),
      });
    } catch (error) {
      if (sequence === requestSequence.current) {
        showError(apiMessage(error, t('加载分时定价失败')));
      }
    } finally {
      if (sequence === requestSequence.current) setLoading(false);
    }
  }, [channelId, endpoint, modelName, t]);

  useEffect(() => {
    loadTimePricing();
  }, [loadTimePricing]);

  const planMap = useMemo(
    () => new Map(plans.map((plan) => [Number(plan.id), plan])),
    [plans],
  );
  const enabledPlans = useMemo(
    () => plans.filter((plan) => plan.enabled),
    [plans],
  );

  const openCreatePlan = useCallback(() => {
    setPlanName(`${modelName} ${t('高峰价格')}`);
    setPlanEditor({
      mode: 'create',
      planMode: PLAN_MODE_PRICE,
      plan: null,
      options: editorOptionsFromRegularPrice(options, modelName),
    });
  }, [modelName, options, t]);

  const openCreateRatePlan = useCallback(() => {
    setPlanName(`${modelName} ${t('高峰费率')}`);
    setRateDraft(channelRates);
    setPlanEditor({
      mode: 'create',
      planMode: PLAN_MODE_RATE,
      plan: null,
    });
  }, [channelRates, modelName, t]);

  const openEditPlan = useCallback(
    (plan) => {
      const planMode = resolvePlanMode(plan.pricing);
      setPlanName(plan.name || '');
      if (planMode === PLAN_MODE_RATE) {
        const priceDiscount = normalizeRateValue(
          plan.pricing?.PriceDiscountPercent,
          channelRates.price_discount_percent,
        );
        const operatingCost = normalizeRateValue(
          plan.pricing?.OperatingCostPercent,
          channelRates.operating_cost_percent,
        );
        setRateDraft({
          price_discount_percent: priceDiscount,
          operating_cost_percent: operatingCost,
          effective_cost_percent: Math.min(1000, priceDiscount + operatingCost),
          markup_discount_rate: normalizeRateValue(
            plan.pricing?.MarkupDiscountRate,
            channelRates.markup_discount_rate,
          ),
        });
      }
      setPlanEditor({
        mode: 'edit',
        planMode,
        plan,
        options:
          planMode === PLAN_MODE_PRICE
            ? editorOptionsFromPlan(options, modelName, plan.pricing || {})
            : null,
      });
    },
    [channelRates, modelName, options],
  );

  const savePlanOutput = useCallback(
    async (output) => {
      const trimmedName = planName.trim();
      if (!trimmedName) throw new Error(t('请输入价格方案名称'));
      const editorPricing = pricingPayloadFromEditorOutput(output, modelName);
      if (Object.keys(editorPricing).length === 0) {
        throw new Error(t('价格方案不能为空'));
      }
      const pricing = { Mode: PLAN_MODE_PRICE, ...editorPricing };
      const body = {
        model_name: modelName,
        name: trimmedName,
        enabled: planEditor?.plan?.enabled ?? true,
        pricing,
      };
      const response =
        planEditor?.mode === 'edit'
          ? await API.put(`${endpoint}/plans/${planEditor.plan.id}`, body)
          : await API.post(`${endpoint}/plans`, body);
      if (!response?.data?.success) {
        throw new Error(response?.data?.message || t('保存价格方案失败'));
      }
      setPlanEditor(null);
      await loadTimePricing();
      showSuccess(t('价格方案已保存'));
    },
    [endpoint, loadTimePricing, modelName, planEditor, planName, t],
  );

  const saveRatePlan = useCallback(async () => {
    const trimmedName = planName.trim();
    if (!trimmedName) {
      showError(t('请输入价格方案名称'));
      return;
    }
    const priceDiscount = normalizeRateValue(
      rateDraft.price_discount_percent,
      100,
    );
    const operatingCost = normalizeRateValue(
      rateDraft.operating_cost_percent,
      0,
    );
    const markupDiscount = normalizeRateValue(
      rateDraft.markup_discount_rate,
      0,
    );
    setRateSaving(true);
    try {
      const body = {
        model_name: modelName,
        name: trimmedName,
        enabled: planEditor?.plan?.enabled ?? true,
        pricing: {
          Mode: PLAN_MODE_RATE,
          PriceDiscountPercent: priceDiscount,
          OperatingCostPercent: operatingCost,
          MarkupDiscountRate: markupDiscount,
        },
      };
      const response =
        planEditor?.mode === 'edit'
          ? await API.put(`${endpoint}/plans/${planEditor.plan.id}`, body)
          : await API.post(`${endpoint}/plans`, body);
      if (!response?.data?.success) {
        throw new Error(response?.data?.message || t('保存费率方案失败'));
      }
      setPlanEditor(null);
      await loadTimePricing();
      showSuccess(t('费率方案已保存'));
    } catch (error) {
      showError(apiMessage(error, t('保存费率方案失败')));
    } finally {
      setRateSaving(false);
    }
  }, [
    endpoint,
    loadTimePricing,
    modelName,
    planEditor,
    planName,
    rateDraft,
    t,
  ]);

  const togglePlan = useCallback(
    async (plan) => {
      try {
        const response = await API.put(`${endpoint}/plans/${plan.id}`, {
          model_name: modelName,
          name: plan.name,
          enabled: !plan.enabled,
          pricing: plan.pricing || {},
        });
        if (!response?.data?.success) {
          throw new Error(response?.data?.message || t('更新价格方案失败'));
        }
        await loadTimePricing();
      } catch (error) {
        showError(apiMessage(error, t('更新价格方案失败')));
      }
    },
    [endpoint, loadTimePricing, modelName, t],
  );

  const deletePlan = useCallback(
    (plan) => {
      Modal.confirm({
        title: t('删除价格方案'),
        content: t('删除后不可恢复；已被时段引用的方案不能删除。'),
        okType: 'danger',
        onOk: async () => {
          try {
            const response = await API.delete(`${endpoint}/plans/${plan.id}`);
            if (!response?.data?.success) {
              throw new Error(response?.data?.message || t('删除价格方案失败'));
            }
            await loadTimePricing();
            showSuccess(t('价格方案已删除'));
          } catch (error) {
            showError(apiMessage(error, t('删除价格方案失败')));
            throw error;
          }
        },
      });
    },
    [endpoint, loadTimePricing, t],
  );

  const openCreateSchedule = useCallback(() => {
    setScheduleDraft({
      ...DEFAULT_SCHEDULE_DRAFT,
      pricePlanId: Number(enabledPlans[0]?.id || 0),
    });
    setScheduleVisible(true);
  }, [enabledPlans]);

  const openEditSchedule = useCallback((schedule) => {
    setScheduleDraft({
      id: Number(schedule.id),
      name: schedule.name || '',
      pricePlanId: Number(schedule.price_plan_id || 0),
      weekdays: maskToWeekdays(schedule.weekdays),
      startTime: minuteToTime(schedule.start_minute),
      endTime: minuteToTime(schedule.end_minute),
      enabled: Boolean(schedule.enabled),
    });
    setScheduleVisible(true);
  }, []);

  const saveSchedule = useCallback(async () => {
    const startMinute = timeToMinute(scheduleDraft.startTime);
    const endMinute = timeToMinute(scheduleDraft.endTime, true);
    if (!scheduleDraft.name.trim()) {
      showError(t('请输入时段名称'));
      return;
    }
    if (!scheduleDraft.pricePlanId) {
      showError(t('请选择价格方案'));
      return;
    }
    if (!scheduleDraft.weekdays.length) {
      showError(t('请至少选择一天'));
      return;
    }
    if (
      startMinute === null ||
      endMinute === null ||
      startMinute === endMinute
    ) {
      showError(t('请输入有效的开始和结束时间'));
      return;
    }
    setScheduleSaving(true);
    try {
      const body = {
        model_name: modelName,
        price_plan_id: scheduleDraft.pricePlanId,
        name: scheduleDraft.name.trim(),
        timezone: 'Asia/Shanghai',
        weekdays: weekdaysToMask(scheduleDraft.weekdays),
        start_minute: startMinute,
        end_minute: endMinute,
        enabled: scheduleDraft.enabled,
      };
      const response = scheduleDraft.id
        ? await API.put(`${endpoint}/schedules/${scheduleDraft.id}`, body)
        : await API.post(`${endpoint}/schedules`, body);
      if (!response?.data?.success) {
        throw new Error(response?.data?.message || t('保存时段失败'));
      }
      setScheduleVisible(false);
      await loadTimePricing();
      showSuccess(t('时段规则已保存'));
    } catch (error) {
      showError(apiMessage(error, t('保存时段失败')));
    } finally {
      setScheduleSaving(false);
    }
  }, [endpoint, loadTimePricing, modelName, scheduleDraft, t]);

  const toggleSchedule = useCallback(
    async (schedule) => {
      try {
        const response = await API.put(`${endpoint}/schedules/${schedule.id}`, {
          model_name: modelName,
          price_plan_id: schedule.price_plan_id,
          name: schedule.name,
          timezone: schedule.timezone || 'Asia/Shanghai',
          weekdays: schedule.weekdays,
          start_minute: schedule.start_minute,
          end_minute: schedule.end_minute,
          effective_from: schedule.effective_from || '',
          effective_to: schedule.effective_to || '',
          enabled: !schedule.enabled,
        });
        if (!response?.data?.success) {
          throw new Error(response?.data?.message || t('更新时段失败'));
        }
        await loadTimePricing();
      } catch (error) {
        showError(apiMessage(error, t('更新时段失败')));
      }
    },
    [endpoint, loadTimePricing, modelName, t],
  );

  const deleteSchedule = useCallback(
    (schedule) => {
      Modal.confirm({
        title: t('删除时段规则'),
        content: t('删除后该时间范围将恢复使用渠道常规价。'),
        okType: 'danger',
        onOk: async () => {
          try {
            const response = await API.delete(
              `${endpoint}/schedules/${schedule.id}`,
            );
            if (!response?.data?.success) {
              throw new Error(response?.data?.message || t('删除时段失败'));
            }
            await loadTimePricing();
            showSuccess(t('时段规则已删除'));
          } catch (error) {
            showError(apiMessage(error, t('删除时段失败')));
            throw error;
          }
        },
      });
    },
    [endpoint, loadTimePricing, t],
  );

  const weekdayLabel = useCallback(
    (mask) => {
      if (Number(mask) === 0x7f) return t('每天');
      if (Number(mask) === 0x3e) return t('工作日');
      return WEEKDAY_OPTIONS.filter(
        (item) => (Number(mask) & (1 << item.value)) !== 0,
      )
        .map((item) => t(item.label))
        .join('、');
    },
    [t],
  );

  const planColumns = useMemo(
    () => [
      {
        title: t('方案名称'),
        dataIndex: 'name',
        render: (name, plan) => (
          <Space>
            <span>{name}</span>
            {Number(plan.id) === activePlanId ? (
              <Tag color='green'>{t('当前生效')}</Tag>
            ) : null}
          </Space>
        ),
      },
      {
        title: t('版本'),
        dataIndex: 'version',
        width: 90,
        render: (version) => `v${version || 1}`,
      },
      {
        title: t('方案模式'),
        width: 130,
        render: (_, plan) =>
          resolvePlanMode(plan.pricing) === PLAN_MODE_RATE ? (
            <Tag color='orange'>{t('动态费率')}</Tag>
          ) : (
            <Tag color='blue'>{t('独立价格')}</Tag>
          ),
      },
      {
        title: t('状态'),
        width: 100,
        render: (_, plan) => (
          <Switch
            checked={Boolean(plan.enabled)}
            onChange={() => togglePlan(plan)}
          />
        ),
      },
      {
        title: t('操作'),
        width: 150,
        render: (_, plan) => (
          <Space>
            <Button icon={<IconEdit />} onClick={() => openEditPlan(plan)} />
            <Button
              type='danger'
              icon={<IconDelete />}
              onClick={() => deletePlan(plan)}
            />
          </Space>
        ),
      },
    ],
    [activePlanId, deletePlan, openEditPlan, t, togglePlan],
  );

  const scheduleColumns = useMemo(
    () => [
      {
        title: t('时段名称'),
        dataIndex: 'name',
        render: (name, schedule) => (
          <Space>
            <span>{name}</span>
            {Number(schedule.id) === activeScheduleId ? (
              <Tag color='green'>{t('当前生效')}</Tag>
            ) : null}
          </Space>
        ),
      },
      {
        title: t('生效时间'),
        render: (_, schedule) => (
          <div>
            <div>{weekdayLabel(schedule.weekdays)}</div>
            <div className='text-xs text-gray-500'>
              {minuteToTime(schedule.start_minute)}–
              {minuteToTime(schedule.end_minute)}
            </div>
          </div>
        ),
      },
      {
        title: t('价格方案'),
        render: (_, schedule) =>
          planMap.get(Number(schedule.price_plan_id))?.name || t('方案不存在'),
      },
      {
        title: t('状态'),
        width: 100,
        render: (_, schedule) => (
          <Switch
            checked={Boolean(schedule.enabled)}
            onChange={() => toggleSchedule(schedule)}
          />
        ),
      },
      {
        title: t('操作'),
        width: 150,
        render: (_, schedule) => (
          <Space>
            <Button
              icon={<IconEdit />}
              onClick={() => openEditSchedule(schedule)}
            />
            <Button
              type='danger'
              icon={<IconDelete />}
              onClick={() => deleteSchedule(schedule)}
            />
          </Space>
        ),
      },
    ],
    [
      activeScheduleId,
      deleteSchedule,
      openEditSchedule,
      planMap,
      t,
      toggleSchedule,
      weekdayLabel,
    ],
  );

  if (!channelId || !modelName) return null;

  return (
    <Card
      title={t('分时定价')}
      style={{ marginTop: 16 }}
      headerExtraContent={
        <Button
          icon={<IconRefresh />}
          onClick={loadTimePricing}
          loading={loading}
        >
          {t('刷新')}
        </Button>
      }
    >
      <div className='mb-4 text-sm text-gray-500'>
        {t(
          '分时方案支持独立价格和动态费率两种模式。动态费率模式保留渠道常规价，只临时覆盖成本折扣率、经营成本率和加价折扣率；加价部分仍以全局官方价为基数。时区固定为 Asia/Shanghai。',
        )}
      </div>
      <Spin spinning={loading}>
        <Card
          title={t('分时方案')}
          bodyStyle={{ padding: 12 }}
          headerExtraContent={
            <Space>
              <Button icon={<IconPlus />} onClick={openCreateRatePlan}>
                {t('新建动态费率方案')}
              </Button>
              <Button
                type='primary'
                icon={<IconPlus />}
                onClick={openCreatePlan}
              >
                {t('从渠道常规价复制')}
              </Button>
            </Space>
          }
        >
          {plans.length > 0 ? (
            <Table
              rowKey='id'
              columns={planColumns}
              dataSource={plans}
              pagination={false}
            />
          ) : (
            <Empty
              title={t('暂无价格方案')}
              description={t(
                '可以复制渠道常规价精确修改，也可以只设置三个动态费率。',
              )}
            />
          )}
        </Card>

        <Card
          title={t('生效时段')}
          bodyStyle={{ padding: 12 }}
          style={{ marginTop: 16 }}
          headerExtraContent={
            <Button
              type='primary'
              icon={<IconPlus />}
              disabled={enabledPlans.length === 0}
              onClick={openCreateSchedule}
            >
              {t('添加时间段')}
            </Button>
          }
        >
          {schedules.length > 0 ? (
            <Table
              rowKey='id'
              columns={scheduleColumns}
              dataSource={schedules}
              pagination={false}
            />
          ) : (
            <Empty title={t('暂无时段规则')} />
          )}
        </Card>
      </Spin>

      <Modal
        visible={Boolean(planEditor)}
        onCancel={() => setPlanEditor(null)}
        title={
          planEditor?.planMode === PLAN_MODE_RATE
            ? planEditor?.mode === 'edit'
              ? t('编辑动态费率方案')
              : t('新建动态费率方案')
            : planEditor?.mode === 'edit'
              ? t('编辑独立价格方案')
              : t('从渠道常规价复制价格方案')
        }
        footer={
          planEditor?.planMode === PLAN_MODE_RATE ? (
            <div className='flex justify-end gap-2'>
              <Button disabled={rateSaving} onClick={() => setPlanEditor(null)}>
                {t('取消')}
              </Button>
              <Button
                type='primary'
                loading={rateSaving}
                onClick={saveRatePlan}
              >
                {t('保存方案')}
              </Button>
            </div>
          ) : null
        }
        width={
          planEditor?.planMode === PLAN_MODE_RATE
            ? 'min(620px, 94vw)'
            : 'min(1240px, 96vw)'
        }
        bodyStyle={{
          maxHeight:
            planEditor?.planMode === PLAN_MODE_RATE
              ? 'calc(100vh - 240px)'
              : '82vh',
          overflowY: 'auto',
        }}
        closeOnEsc={false}
      >
        <div className='mb-4'>
          <div className='mb-2 font-medium'>{t('方案名称')}</div>
          <Input value={planName} onChange={setPlanName} maxLength={128} />
          <div className='mt-2 text-xs text-gray-500'>
            {planEditor?.planMode === PLAN_MODE_RATE
              ? t(
                  '命中时继续使用渠道常规价，只用这里的三个费率覆盖渠道默认值。',
                )
              : t('保存后成为独立快照，后续修改渠道常规价不会自动改变该方案。')}
          </div>
        </div>
        {planEditor?.planMode === PLAN_MODE_RATE ? (
          <div className='space-y-4'>
            <div className='overflow-hidden rounded-xl border border-blue-100 bg-gradient-to-br from-blue-50 via-white to-indigo-50 shadow-sm'>
              <div className='border-b border-blue-100 px-4 py-3'>
                <div className='text-sm font-semibold text-blue-900'>
                  {t('计费公式')}
                </div>
                <div className='mt-1 text-xs leading-5 text-blue-600'>
                  {t('渠道价格部分与官方加价部分分别计算后相加。')}
                </div>
              </div>
              <div className='space-y-2.5 px-4 py-3 text-sm'>
                <div className='flex items-start gap-3 rounded-lg bg-white/80 px-3 py-2.5'>
                  <span className='inline-flex h-6 min-w-6 items-center justify-center rounded-full bg-cyan-100 px-1.5 text-xs font-bold text-cyan-700'>
                    ①
                  </span>
                  <div className='min-w-0 leading-6 text-gray-700'>
                    <span className='font-semibold text-gray-900'>
                      {t('渠道常规价')}
                    </span>{' '}
                    ×（{t('成本折扣率')} + {t('经营成本率')}）
                  </div>
                </div>
                <div className='flex items-start gap-3 rounded-lg bg-white/80 px-3 py-2.5'>
                  <span className='inline-flex h-6 min-w-6 items-center justify-center rounded-full bg-violet-100 px-1.5 text-xs font-bold text-violet-700'>
                    ②
                  </span>
                  <div className='min-w-0 leading-6 text-gray-700'>
                    <span className='font-semibold text-gray-900'>
                      {t('全局官方价')}
                    </span>{' '}
                    × {t('加价折扣率')}
                  </div>
                </div>
                <div className='flex items-center justify-between gap-3 border-t border-blue-100 pt-3'>
                  <span className='text-sm font-semibold text-blue-900'>
                    {t('最终价格')}
                  </span>
                  <span className='rounded-full bg-blue-600 px-3 py-1 text-xs font-semibold text-white shadow-sm'>
                    ① + ②
                  </span>
                </div>
              </div>
            </div>
            <div>
              <div className='mb-2 font-medium'>{t('成本折扣率(%)')}</div>
              <InputNumber
                value={rateDraft.price_discount_percent}
                min={0}
                max={1000}
                precision={2}
                style={{ width: '100%' }}
                onChange={(value) =>
                  setRateDraft((current) => {
                    const priceDiscount = normalizeRateValue(value, 0);
                    return {
                      ...current,
                      price_discount_percent: priceDiscount,
                      effective_cost_percent: Math.min(
                        1000,
                        priceDiscount + current.operating_cost_percent,
                      ),
                    };
                  })
                }
              />
            </div>
            <div>
              <div className='mb-2 font-medium'>{t('经营成本率(%)')}</div>
              <InputNumber
                value={rateDraft.operating_cost_percent}
                min={0}
                max={1000}
                precision={2}
                style={{ width: '100%' }}
                onChange={(value) =>
                  setRateDraft((current) => {
                    const operatingCost = normalizeRateValue(value, 0);
                    return {
                      ...current,
                      operating_cost_percent: operatingCost,
                      effective_cost_percent: Math.min(
                        1000,
                        current.price_discount_percent + operatingCost,
                      ),
                    };
                  })
                }
              />
            </div>
            <div>
              <div className='mb-2 font-medium'>{t('加价折扣率(%)')}</div>
              <InputNumber
                value={rateDraft.markup_discount_rate}
                min={0}
                max={1000}
                precision={2}
                style={{ width: '100%' }}
                onChange={(value) =>
                  setRateDraft((current) => ({
                    ...current,
                    markup_discount_rate: normalizeRateValue(value, 0),
                  }))
                }
              />
            </div>
            <div className='rounded-lg bg-gray-50 p-3 text-sm text-gray-600'>
              {t('最终成本率')}：
              <span className='font-semibold text-gray-900'>
                {rateDraft.effective_cost_percent}%
              </span>
              <span className='ml-3 text-gray-400'>
                {t('渠道当前默认值')}：{channelRates.price_discount_percent}% +{' '}
                {channelRates.operating_cost_percent}% /{' '}
                {channelRates.markup_discount_rate}%
              </span>
            </div>
          </div>
        ) : planEditor ? (
          <ModelPricingEditor
            options={planEditor.options}
            refresh={async () => {}}
            candidateModelNames={[modelName]}
            forceCandidateModelNames
            includeAllCandidateModels
            filterMode='all'
            allowAddModel={false}
            allowDeleteModel={false}
            listDescription={t('当前仅编辑此价格方案，不会修改渠道常规价。')}
            onSaveOutput={savePlanOutput}
          />
        ) : null}
      </Modal>

      <Modal
        visible={scheduleVisible}
        onCancel={() => setScheduleVisible(false)}
        onOk={saveSchedule}
        confirmLoading={scheduleSaving}
        title={scheduleDraft.id ? t('编辑时间段') : t('添加时间段')}
        okText={t('保存')}
      >
        <div className='mb-4'>
          <div className='mb-2 font-medium'>{t('时段名称')}</div>
          <Input
            value={scheduleDraft.name}
            onChange={(value) =>
              setScheduleDraft((current) => ({ ...current, name: value }))
            }
            maxLength={128}
          />
        </div>
        <div className='mb-4'>
          <div className='mb-2 font-medium'>{t('使用价格方案')}</div>
          <Select
            value={scheduleDraft.pricePlanId || undefined}
            onChange={(value) =>
              setScheduleDraft((current) => ({
                ...current,
                pricePlanId: Number(value),
              }))
            }
            style={{ width: '100%' }}
            optionList={enabledPlans.map((plan) => ({
              label: `${plan.name} · ${
                resolvePlanMode(plan.pricing) === PLAN_MODE_RATE
                  ? t('动态费率')
                  : t('独立价格')
              } (v${plan.version})`,
              value: plan.id,
            }))}
          />
        </div>
        <div className='mb-4'>
          <div className='mb-2 font-medium'>{t('重复日期')}</div>
          <CheckboxGroup
            value={scheduleDraft.weekdays}
            onChange={(weekdays) =>
              setScheduleDraft((current) => ({ ...current, weekdays }))
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
        <div className='mb-4'>
          <div className='mb-2 font-medium'>{t('时间范围')}</div>
          <Space style={{ width: '100%' }}>
            <Input
              value={scheduleDraft.startTime}
              placeholder='18:00'
              onChange={(value) =>
                setScheduleDraft((current) => ({
                  ...current,
                  startTime: value,
                }))
              }
            />
            <span>–</span>
            <Input
              value={scheduleDraft.endTime}
              placeholder='23:00'
              onChange={(value) =>
                setScheduleDraft((current) => ({
                  ...current,
                  endTime: value,
                }))
              }
            />
          </Space>
          <div className='mt-2 text-xs text-gray-500'>
            {t('支持跨天，例如 22:00–02:00；结束时间不包含在时段内。')}
          </div>
        </div>
        <div className='flex items-center justify-between'>
          <span className='font-medium'>{t('启用规则')}</span>
          <Switch
            checked={scheduleDraft.enabled}
            onChange={(enabled) =>
              setScheduleDraft((current) => ({ ...current, enabled }))
            }
          />
        </div>
      </Modal>
    </Card>
  );
}
