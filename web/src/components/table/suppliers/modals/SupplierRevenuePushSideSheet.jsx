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
import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';
import timezone from 'dayjs/plugin/timezone';
import {
  Banner,
  Button,
  Card,
  Collapse,
  Form,
  Input,
  Modal,
  SideSheet,
  Space,
  Spin,
  Table,
  Tabs,
  Tag,
  TextArea,
  Tooltip,
  Typography,
} from '@douyinfe/semi-ui';
import { IconHelpCircle } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import {
  API,
  showError,
  showSuccess,
  timestamp2string,
} from '../../../../helpers';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';

const { Text, Title } = Typography;

dayjs.extend(utc);
dayjs.extend(timezone);

const EORAPTOR_PRODUCTION_ENDPOINT =
  'https://admin.eoraptor.org/api/open/syncCoin';
const EORAPTOR_MOCK_ENDPOINT =
  'https://mock.apipost.net/mock/4ba1b38f30f1000/api/open/syncCoin?apipost_id=301db04f0c005';
const EORAPTOR_BODY_TEMPLATE =
  '{\n  "number": "{{number}}",\n  "timestamp": "{{timestamp}}",\n  "sign": "{{sign}}"\n}';
const GENERIC_BODY_TEMPLATE =
  '{\n  "number": "{{number}}",\n  "timestamp": {{timestamp}},\n  "supplier_id": {{supplier_id}},\n  "batch_no": "{{batch_no}}",\n  "period_start": {{period_start}},\n  "period_end": {{period_end}},\n  "currency": "{{currency}}"\n}';
const EORAPTOR_SUCCESS_RESPONSE_EXAMPLE = `{
  "code": 200,
  "msg": "同步成功",
  "result": {
    "number": "100"
  },
  "type": "success"
}`;

const TIMEZONE_OPTIONS = [
  { label: 'UTC+08:00 · Asia/Shanghai', value: 'Asia/Shanghai' },
  { label: 'UTC+08:00 · Asia/Hong_Kong', value: 'Asia/Hong_Kong' },
  { label: 'UTC+08:00 · Asia/Singapore', value: 'Asia/Singapore' },
  { label: 'UTC+09:00 · Asia/Tokyo', value: 'Asia/Tokyo' },
  { label: 'UTC+00:00 · UTC', value: 'UTC' },
  { label: 'Europe/London', value: 'Europe/London' },
  { label: 'Europe/Berlin', value: 'Europe/Berlin' },
  { label: 'America/New_York', value: 'America/New_York' },
  { label: 'America/Los_Angeles', value: 'America/Los_Angeles' },
  { label: 'Australia/Sydney', value: 'Australia/Sydney' },
];

const DEFAULT_CONFIG = {
  enabled: false,
  mode: 'generic',
  schedule_type: 'daily',
  timezone: 'Asia/Shanghai',
  daily_time: '01:00',
  hourly_minute: 5,
  currency: 'USDT',
  negative_policy: 'hold',
  retry_count: 3,
  retry_interval_seconds: 300,
  retry_backoff: 'fixed',
  timeout_seconds: 15,
  environment: 'mock',
  endpoint: '',
  mock_endpoint: EORAPTOR_MOCK_ENDPOINT,
  http_method: 'POST',
  content_type: 'application/json',
  headers_json: '{}',
  body_template: GENERIC_BODY_TEMPLATE,
  success_http_status: 200,
  success_code_path: '',
  success_code_value: '',
  success_type_path: '',
  success_type_value: '',
  success_amount_path: '',
  callback_config_json: '{}',
};

const STATUS_COLORS = {
  created: 'grey',
  sending: 'blue',
  retrying: 'orange',
  success: 'green',
  failed: 'red',
  unknown: 'violet',
};

const nextRevenuePushAt = (
  { scheduleType, timezoneName, dailyTime, hourlyMinute },
  now,
) => {
  try {
    const current = dayjs(now).tz(timezoneName || 'Asia/Shanghai');
    if (scheduleType === 'hourly') {
      const minute = Number(hourlyMinute);
      if (!Number.isInteger(minute) || minute < 0 || minute > 59) return null;
      let target = current.minute(minute).second(0).millisecond(0);
      if (!target.isAfter(current)) target = target.add(1, 'hour');
      return target;
    }
    const match = /^(\d{1,2}):(\d{2})$/.exec(dailyTime || '');
    if (!match) return null;
    const hour = Number(match[1]);
    const minute = Number(match[2]);
    if (hour < 0 || hour > 23 || minute < 0 || minute > 59) return null;
    let target = current.hour(hour).minute(minute).second(0).millisecond(0);
    if (!target.isAfter(current)) target = target.add(1, 'day');
    return target;
  } catch {
    return null;
  }
};

const RevenuePushCountdown = React.memo(
  ({ enabled, scheduleType, timezoneName, dailyTime, hourlyMinute }) => {
    const { t } = useTranslation();
    const [now, setNow] = useState(() => Date.now());

    useEffect(() => {
      if (!enabled) return undefined;
      setNow(Date.now());
      const timer = window.setInterval(() => setNow(Date.now()), 1000);
      return () => window.clearInterval(timer);
    }, [enabled]);

    if (!enabled) return null;
    const nextAt = nextRevenuePushAt(
      { scheduleType, timezoneName, dailyTime, hourlyMinute },
      now,
    );
    if (!nextAt) {
      return (
        <Banner
          type='warning'
          className='mb-4'
          description={t('无法计算下一次推送时间，请检查时区和执行时间配置。')}
        />
      );
    }
    const remainingSeconds = Math.max(
      0,
      Math.floor((nextAt.valueOf() - now) / 1000),
    );
    const days = Math.floor(remainingSeconds / 86400);
    const hours = Math.floor((remainingSeconds % 86400) / 3600);
    const minutes = Math.floor((remainingSeconds % 3600) / 60);
    const seconds = remainingSeconds % 60;
    const clock = [hours, minutes, seconds]
      .map((value) => String(value).padStart(2, '0'))
      .join(':');

    return (
      <div className='mb-4 flex flex-wrap items-center justify-between gap-3 rounded-lg border border-solid border-[var(--semi-color-primary-light-default)] bg-[var(--semi-color-primary-light-default)] px-4 py-3'>
        <div>
          <Text strong>{t('距离下一次自动推送')}</Text>
          <div className='mt-1'>
            <Text type='tertiary' size='small'>
              {t('下次推送时间')}：{nextAt.format('YYYY-MM-DD HH:mm:ss')}（
              {timezoneName}）
            </Text>
          </div>
        </div>
        <Text
          strong
          className='font-mono text-lg text-[var(--semi-color-primary)]'
        >
          {days > 0 ? `${days}${t('天')} ` : ''}
          {clock}
        </Text>
      </div>
    );
  },
);

RevenuePushCountdown.displayName = 'RevenuePushCountdown';

const formatPeriod = (row) => {
  if (!row?.period_start || !row?.period_end) {
    return '-';
  }
  return `${timestamp2string(row.period_start)} ~ ${timestamp2string(row.period_end)}`;
};

const SupplierRevenuePushSideSheet = ({ visible, supplier, onClose }) => {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const privateKeyInputRef = useRef(null);
  const lastModeRef = useRef(DEFAULT_CONFIG.mode);
  const [formApi, setFormApi] = useState(null);
  const [formValues, setFormValues] = useState(DEFAULT_CONFIG);
  const [metadata, setMetadata] = useState({});
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [running, setRunning] = useState(false);
  const [manualVisible, setManualVisible] = useState(false);
  const [manualSending, setManualSending] = useState(false);
  const [manualAmount, setManualAmount] = useState('');
  const [manualRemark, setManualRemark] = useState('');
  const [privateKey, setPrivateKey] = useState('');
  const [privateKeyName, setPrivateKeyName] = useState('');
  const [deliveries, setDeliveries] = useState([]);
  const [deliveryLoading, setDeliveryLoading] = useState(false);
  const [deliveryPage, setDeliveryPage] = useState(1);
  const [deliveryTotal, setDeliveryTotal] = useState(0);
  const [attempts, setAttempts] = useState([]);
  const [attemptLoading, setAttemptLoading] = useState(false);
  const [attemptVisible, setAttemptVisible] = useState(false);
  const [selectedDelivery, setSelectedDelivery] = useState(null);

  const supplierId = supplier?.id;
  const mode = formValues?.mode || 'generic';
  const scheduleType = formValues?.schedule_type || 'daily';

  const loadDeliveries = useCallback(
    async (page = 1) => {
      if (!supplierId) return;
      setDeliveryLoading(true);
      try {
        const response = await API.get(
          `/api/user/supplier/${supplierId}/revenue-push/deliveries?p=${page}&page_size=10`,
        );
        if (!response.data.success) {
          showError(response.data.message || t('加载推送日志失败'));
          return;
        }
        setDeliveries(response.data.data?.items || []);
        setDeliveryTotal(response.data.data?.total || 0);
        setDeliveryPage(response.data.data?.page || page);
      } catch (error) {
        showError(error.response?.data?.message || t('加载推送日志失败'));
      } finally {
        setDeliveryLoading(false);
      }
    },
    [supplierId, t],
  );

  const loadConfig = useCallback(async () => {
    if (!supplierId) return;
    setLoading(true);
    try {
      const response = await API.get(
        `/api/user/supplier/${supplierId}/revenue-push/config`,
      );
      if (!response.data.success) {
        showError(response.data.message || t('加载收益推送配置失败'));
        return;
      }
      const data = response.data.data || {};
      const config = data.config || {};
      const nextValues = {
        ...DEFAULT_CONFIG,
        ...config,
        mock_endpoint:
          config.mock_endpoint || data.mock_endpoint || EORAPTOR_MOCK_ENDPOINT,
      };
      lastModeRef.current = nextValues.mode;
      setFormValues(nextValues);
      setMetadata(data);
      formApi?.setValues(nextValues);
    } catch (error) {
      showError(error.response?.data?.message || t('加载收益推送配置失败'));
    } finally {
      setLoading(false);
    }
  }, [formApi, supplierId, t]);

  useEffect(() => {
    if (!visible || !supplierId) return;
    setPrivateKey('');
    setPrivateKeyName('');
    void Promise.all([loadConfig(), loadDeliveries(1)]);
  }, [loadConfig, loadDeliveries, supplierId, visible]);

  const buildPayload = useCallback(() => {
    const values = formApi?.getValues() || formValues;
    const payload = {
      ...DEFAULT_CONFIG,
      ...values,
      private_key: privateKey,
    };
    if (payload.mode === 'eoraptor') {
      payload.http_method = 'POST';
    }
    return payload;
  }, [formApi, formValues, privateKey]);

  const validatePayload = useCallback(
    (payload) => {
      if (!payload.timezone || !payload.daily_time) {
        showError(t('请完整填写调度配置'));
        return false;
      }
      if (payload.mode === 'generic' && !payload.endpoint) {
        showError(t('请填写通用推送地址'));
        return false;
      }
      if (payload.mode === 'eoraptor' && !payload.endpoint) {
        showError(t('请填写定制版推送地址'));
        return false;
      }
      if (payload.mode === 'eoraptor' && !payload.mock_endpoint) {
        showError(t('请填写Mock测试地址'));
        return false;
      }
      if (
        payload.mode === 'eoraptor' &&
        payload.enabled &&
        !privateKey &&
        !metadata.has_private_key
      ) {
        showError(t('启用前请上传RSA私钥'));
        return false;
      }
      for (const field of ['headers_json', 'callback_config_json']) {
        try {
          JSON.parse(payload[field] || '{}');
        } catch {
          showError(t('JSON配置格式不正确'));
          return false;
        }
      }
      if (payload.mode === 'eoraptor') {
        const requiredVariables = ['{{number}}', '{{timestamp}}', '{{sign}}'];
        if (
          requiredVariables.some(
            (variable) => !payload.body_template?.includes(variable),
          )
        ) {
          showError(t('定制版发送参数模板必须包含number、timestamp和sign变量'));
          return false;
        }
      }
      return true;
    },
    [metadata.has_private_key, privateKey, t],
  );

  const handleModeChange = useCallback(
    (nextMode) => {
      const values = formApi?.getValues() || formValues;
      if (lastModeRef.current === nextMode) {
        setFormValues((current) => ({ ...current, mode: nextMode }));
        return;
      }
      lastModeRef.current = nextMode;
      const nextValues =
        nextMode === 'eoraptor'
          ? {
              ...values,
              mode: nextMode,
              schedule_type: values.schedule_type || 'daily',
              currency: values.currency || 'USDT',
              endpoint:
                metadata.production_endpoint || EORAPTOR_PRODUCTION_ENDPOINT,
              mock_endpoint:
                values.mock_endpoint ||
                metadata.mock_endpoint ||
                EORAPTOR_MOCK_ENDPOINT,
              http_method: 'POST',
              content_type: 'multipart/form-data',
              body_template: EORAPTOR_BODY_TEMPLATE,
              success_http_status: 200,
              success_code_path: 'code',
              success_code_value: '200',
              success_type_path: 'type',
              success_type_value: 'success',
              success_amount_path: 'result.number',
            }
          : {
              ...values,
              mode: nextMode,
              endpoint: '',
              http_method: 'POST',
              content_type: 'application/json',
              body_template: GENERIC_BODY_TEMPLATE,
              success_http_status: 200,
              success_code_path: '',
              success_code_value: '',
              success_type_path: '',
              success_type_value: '',
              success_amount_path: '',
            };
      setFormValues(nextValues);
      formApi?.setValues(nextValues);
    },
    [formApi, formValues, metadata.mock_endpoint, metadata.production_endpoint],
  );

  const applyResponseRulePreset = useCallback(
    (preset) => {
      const values =
        preset === 'http_only'
          ? {
              success_http_status: 200,
              success_code_path: '',
              success_code_value: '',
              success_type_path: '',
              success_type_value: '',
              success_amount_path: '',
            }
          : {
              success_http_status: 200,
              success_code_path: 'code',
              success_code_value: '200',
              success_type_path: 'type',
              success_type_value: 'success',
              success_amount_path: 'result.number',
            };
      const currentValues = formApi?.getValues() || formValues;
      formApi?.setValues({ ...currentValues, ...values });
      setFormValues((current) => ({ ...current, ...values }));
    },
    [formApi, formValues],
  );

  const saveConfig = async () => {
    const payload = buildPayload();
    if (!validatePayload(payload)) return;
    setSaving(true);
    try {
      const response = await API.put(
        `/api/user/supplier/${supplierId}/revenue-push/config`,
        payload,
      );
      if (!response.data.success) {
        showError(response.data.message || t('保存失败'));
        return;
      }
      const data = response.data.data || {};
      setMetadata(data);
      setPrivateKey('');
      setPrivateKeyName('');
      showSuccess(t('收益推送配置已保存'));
      await loadConfig();
    } catch (error) {
      showError(error.response?.data?.message || t('保存失败'));
    } finally {
      setSaving(false);
    }
  };

  const testPush = async () => {
    const payload = buildPayload();
    if (!validatePayload(payload)) return;
    if (
      payload.mode === 'eoraptor' &&
      !privateKey &&
      !metadata.has_private_key
    ) {
      showError(t('测试前请上传RSA私钥'));
      return;
    }
    setTesting(true);
    try {
      const response = await API.post(
        `/api/user/supplier/${supplierId}/revenue-push/test`,
        payload,
      );
      if (!response.data.success) {
        showError(response.data.message || t('测试推送失败'));
        return;
      }
      const result = response.data.data || {};
      if (result.outcome === 'success') {
        showSuccess(t('Mock测试推送成功'));
      } else {
        showError(result.message || t('测试推送失败'));
      }
      await loadDeliveries(1);
    } catch (error) {
      showError(error.response?.data?.message || t('测试推送失败'));
    } finally {
      setTesting(false);
    }
  };

  const runNow = async () => {
    setRunning(true);
    try {
      const response = await API.post(
        `/api/user/supplier/${supplierId}/revenue-push/run`,
      );
      if (!response.data.success) {
        showError(response.data.message || t('执行失败'));
        return;
      }
      showSuccess(response.data.data?.message || t('已执行到期账期检查'));
      await loadDeliveries(1);
    } catch (error) {
      showError(error.response?.data?.message || t('执行失败'));
    } finally {
      setRunning(false);
    }
  };

  const openManualPush = () => {
    setManualAmount('');
    setManualRemark('');
    setManualVisible(true);
  };

  const submitManualPush = async () => {
    const amount = manualAmount.trim();
    if (!/^-?\d+(?:\.\d+)?$/.test(amount)) {
      showError(t('请输入有效的手动推送金额'));
      return;
    }
    setManualSending(true);
    try {
      const response = await API.post(
        `/api/user/supplier/${supplierId}/revenue-push/manual`,
        { amount, remark: manualRemark.trim() },
      );
      if (!response.data.success) {
        showError(response.data.message || t('手动推送失败'));
        return;
      }
      const delivery = response.data.data || {};
      if (delivery.status === 'success') {
        showSuccess(t('手动推送成功'));
        setManualVisible(false);
      } else if (delivery.status === 'retrying') {
        showSuccess(t('手动推送已创建，首次发送失败，将按重试策略继续处理'));
        setManualVisible(false);
      } else {
        showError(delivery.last_error || t('手动推送失败'));
      }
      await loadDeliveries(1);
    } catch (error) {
      if (error.response?.status === 404) {
        showError(t('手动推送接口未加载，请重启后端服务'));
      } else {
        showError(error.response?.data?.message || t('手动推送失败'));
      }
    } finally {
      setManualSending(false);
    }
  };

  const handlePrivateKeyFile = async (event) => {
    const file = event.target.files?.[0];
    event.target.value = '';
    if (!file) return;
    if (file.size > 64 * 1024) {
      showError(t('私钥文件不能超过64KB'));
      return;
    }
    const content = await file.text();
    if (!content.includes('PRIVATE KEY')) {
      showError(t('请选择有效的PEM私钥文件'));
      return;
    }
    setPrivateKey(content);
    setPrivateKeyName(file.name);
  };

  const showAttempts = async (delivery) => {
    setSelectedDelivery(delivery);
    setAttempts([]);
    setAttemptVisible(true);
    setAttemptLoading(true);
    try {
      const response = await API.get(
        `/api/user/supplier/${supplierId}/revenue-push/deliveries/${delivery.id}/attempts`,
      );
      if (!response.data.success) {
        showError(response.data.message || t('加载请求明细失败'));
        return;
      }
      setAttempts(response.data.data || []);
    } catch (error) {
      showError(error.response?.data?.message || t('加载请求明细失败'));
    } finally {
      setAttemptLoading(false);
    }
  };

  const resolveUnknown = (delivery, action) => {
    Modal.confirm({
      title:
        action === 'settled'
          ? t('确认远端已入账？')
          : t('确认远端未入账并重新推送？'),
      content:
        action === 'settled'
          ? t('确认后该批次账期将标记为已结算，不再自动推送。')
          : t('重新推送存在重复入账风险，请先在远端完成核对。'),
      onOk: async () => {
        try {
          const response = await API.post(
            `/api/user/supplier/${supplierId}/revenue-push/deliveries/${delivery.id}/resolve`,
            { action },
          );
          if (!response.data.success) {
            showError(response.data.message || t('处理失败'));
            return;
          }
          showSuccess(t('处理成功'));
          await loadDeliveries(1);
        } catch (error) {
          showError(error.response?.data?.message || t('处理失败'));
        }
      },
    });
  };

  const deliveryColumns = useMemo(
    () => [
      {
        title: t('批次号'),
        dataIndex: 'batch_no',
        width: 230,
        render: (value) => <Text copyable>{value}</Text>,
      },
      {
        title: t('类型'),
        dataIndex: 'kind',
        width: 90,
        render: (value) => <Tag>{value}</Tag>,
      },
      {
        title: t('账期'),
        width: 280,
        render: (_, row) => formatPeriod(row),
      },
      {
        title: t('金额'),
        width: 150,
        render: (_, row) => `${row.amount} ${row.currency}`,
      },
      {
        title: t('状态'),
        dataIndex: 'status',
        width: 100,
        render: (value) => (
          <Tag color={STATUS_COLORS[value] || 'grey'}>{value}</Tag>
        ),
      },
      {
        title: t('尝试次数'),
        width: 100,
        render: (_, row) => `${row.attempt_count}/${row.max_attempts}`,
      },
      {
        title: t('创建时间'),
        dataIndex: 'created_at',
        width: 170,
        render: timestamp2string,
      },
      {
        title: t('备注'),
        dataIndex: 'remark',
        width: 180,
        render: (value) => value || '-',
      },
      {
        title: t('操作'),
        width: 250,
        fixed: 'right',
        render: (_, row) => (
          <Space>
            <Button size='small' onClick={() => showAttempts(row)}>
              {t('请求明细')}
            </Button>
            {row.status === 'unknown' ? (
              <>
                <Button
                  size='small'
                  type='primary'
                  onClick={() => resolveUnknown(row, 'settled')}
                >
                  {t('确认已入账')}
                </Button>
                <Button
                  size='small'
                  type='danger'
                  onClick={() => resolveUnknown(row, 'retry')}
                >
                  {t('确认未入账')}
                </Button>
              </>
            ) : null}
          </Space>
        ),
      },
    ],
    [loadDeliveries, supplierId, t],
  );

  const attemptColumns = useMemo(
    () => [
      { title: t('次数'), dataIndex: 'attempt_no', width: 70 },
      {
        title: t('结果'),
        dataIndex: 'outcome',
        width: 100,
        render: (value) => (
          <Tag color={STATUS_COLORS[value] || 'grey'}>{value}</Tag>
        ),
      },
      { title: 'HTTP', dataIndex: 'http_status', width: 80 },
      { title: t('耗时'), dataIndex: 'duration_ms', render: (v) => `${v}ms` },
      {
        title: t('请求时间'),
        dataIndex: 'requested_at',
        width: 170,
        render: timestamp2string,
      },
      {
        title: t('错误信息'),
        dataIndex: 'error_message',
        width: 260,
        render: (value) => value || '-',
      },
    ],
    [t],
  );

  return (
    <>
      <SideSheet
        visible={visible}
        placement='right'
        width={isMobile ? '100%' : 980}
        onCancel={onClose}
        bodyStyle={{ padding: 0 }}
        title={
          <Space>
            <Title heading={4} className='m-0'>
              {t('供应商收益推送')}
            </Title>
            <Text type='tertiary'>
              {supplier?.supplier_alias || supplier?.company_name} (ID:{' '}
              {supplierId})
            </Text>
          </Space>
        }
      >
        <Spin spinning={loading}>
          <div className='p-4'>
            <Tabs type='line'>
              <Tabs.TabPane tab={t('推送配置')} itemKey='config'>
                <div className='flex flex-wrap justify-end gap-2 mt-5 mb-4'>
                  <Button loading={testing} onClick={testPush}>
                    {mode === 'eoraptor' ? t('Mock测试推送') : t('测试推送')}
                  </Button>
                  <Button loading={running} onClick={runNow}>
                    {t('执行到期账期')}
                  </Button>
                  <Button onClick={openManualPush}>{t('手动推送')}</Button>
                  <Button
                    type='primary'
                    theme='solid'
                    loading={saving}
                    onClick={saveConfig}
                  >
                    {t('保存配置')}
                  </Button>
                </div>

                <Form
                  key={`${supplierId}-${metadata.config?.updated_at || 0}`}
                  initValues={formValues}
                  getFormApi={setFormApi}
                  layout='vertical'
                >
                  <Card title={t('基本配置')} className='mb-4'>
                    <div className='grid grid-cols-1 md:grid-cols-3 gap-x-4'>
                      <Form.Switch
                        field='enabled'
                        label={t('启用收益推送')}
                        onChange={(value) =>
                          setFormValues((current) => ({
                            ...current,
                            enabled: value,
                          }))
                        }
                      />
                      <Form.Select
                        field='mode'
                        label={t('推送类型')}
                        onChange={handleModeChange}
                        optionList={[
                          { label: t('定制版'), value: 'eoraptor' },
                          { label: t('通用Webhook'), value: 'generic' },
                        ]}
                      />
                      <Form.Select
                        field='negative_policy'
                        label={t('负数处理')}
                        optionList={[
                          { label: t('暂停并人工处理'), value: 'hold' },
                          { label: t('允许负数推送'), value: 'allow' },
                          { label: t('抵扣后续收入'), value: 'carry' },
                        ]}
                      />
                    </div>
                  </Card>

                  <Card title={t('调度与重试')} className='mb-4'>
                    <RevenuePushCountdown
                      enabled={Boolean(formValues.enabled)}
                      scheduleType={scheduleType}
                      timezoneName={formValues.timezone}
                      dailyTime={formValues.daily_time}
                      hourlyMinute={formValues.hourly_minute}
                    />
                    <div className='grid grid-cols-1 md:grid-cols-3 gap-x-4'>
                      <Form.Select
                        field='schedule_type'
                        label={t('推送频率')}
                        onChange={(value) =>
                          setFormValues((current) => ({
                            ...current,
                            schedule_type: value,
                          }))
                        }
                        optionList={[
                          { label: t('每日T+1'), value: 'daily' },
                          { label: t('每小时'), value: 'hourly' },
                        ]}
                      />
                      <Form.Select
                        field='timezone'
                        label={t('账期时区')}
                        onChange={(value) =>
                          setFormValues((current) => ({
                            ...current,
                            timezone: value,
                          }))
                        }
                        optionList={TIMEZONE_OPTIONS}
                        filter
                        showClear={false}
                      />
                      {scheduleType === 'daily' ? (
                        <Form.Input
                          field='daily_time'
                          label={t('每日执行时间')}
                          placeholder='01:00'
                          onChange={(value) =>
                            setFormValues((current) => ({
                              ...current,
                              daily_time: value,
                            }))
                          }
                        />
                      ) : (
                        <Form.InputNumber
                          field='hourly_minute'
                          label={t('每小时第几分钟执行')}
                          min={0}
                          max={59}
                          onChange={(value) =>
                            setFormValues((current) => ({
                              ...current,
                              hourly_minute: value,
                            }))
                          }
                        />
                      )}
                      <Form.InputNumber
                        field='retry_count'
                        label={t('失败重试次数')}
                        min={0}
                        max={10}
                        extraText={t('默认3次，加首次请求最多4次')}
                      />
                      <Form.InputNumber
                        field='retry_interval_seconds'
                        label={t('重试间隔（秒）')}
                        min={1}
                        max={86400}
                      />
                      <Form.Select
                        field='retry_backoff'
                        label={
                          <span className='inline-flex items-center gap-1'>
                            <span>{t('重试策略')}</span>
                            <Tooltip
                              content={t(
                                '固定间隔：每次按相同秒数重试；指数退避：每次失败后等待时间逐步翻倍，可降低远端故障时的请求压力。',
                              )}
                            >
                              <IconHelpCircle
                                size='small'
                                className='cursor-help text-[var(--semi-color-text-2)]'
                                aria-label={t('重试策略说明')}
                              />
                            </Tooltip>
                          </span>
                        }
                        optionList={[
                          { label: t('固定间隔'), value: 'fixed' },
                          { label: t('指数退避'), value: 'exponential' },
                        ]}
                      />
                      <Form.InputNumber
                        field='timeout_seconds'
                        label={t('请求超时（秒）')}
                        min={1}
                        max={120}
                      />
                      <Form.Select
                        field='currency'
                        label={t('推送币种')}
                        extraText={t(
                          '选择CNY时按系统美元汇率换算后推送；USD与USDT当前按1:1处理。',
                        )}
                        optionList={[
                          { label: 'USD', value: 'USD' },
                          { label: 'CNY', value: 'CNY' },
                          { label: 'USDT', value: 'USDT' },
                        ]}
                      />
                    </div>
                  </Card>

                  {mode === 'eoraptor' ? (
                    <Card title={t('定制版')} className='mb-4'>
                      <Banner
                        type='warning'
                        className='mb-4'
                        description={t(
                          '定制版支持每日或每小时推送；USD与USDT按1:1，CNY按系统美元汇率换算。测试按钮使用单独配置的Mock地址。',
                        )}
                      />
                      <div className='grid grid-cols-1 md:grid-cols-3 gap-x-4'>
                        <Form.Select
                          field='environment'
                          label={t('运行环境')}
                          onChange={(value) =>
                            setFormValues((current) => ({
                              ...current,
                              environment: value,
                            }))
                          }
                          optionList={[
                            { label: t('Mock测试环境'), value: 'mock' },
                            { label: t('正式环境'), value: 'production' },
                          ]}
                        />
                        <Form.Input
                          field='endpoint'
                          label={t('正式推送URL地址')}
                          className='md:col-span-2'
                          placeholder={EORAPTOR_PRODUCTION_ENDPOINT}
                          extraText={t('正式环境执行到期账期时使用此地址。')}
                        />
                        <Form.Input
                          field='mock_endpoint'
                          label={t('Mock测试URL地址')}
                          className='md:col-span-3'
                          placeholder={EORAPTOR_MOCK_ENDPOINT}
                          extraText={t(
                            '点击Mock测试推送或选择Mock测试环境时使用此地址，可替换为自建Mock服务。',
                          )}
                        />
                        <Form.Select
                          field='content_type'
                          label={t('发送格式')}
                          optionList={[
                            {
                              label: t('Multipart表单'),
                              value: 'multipart/form-data',
                            },
                            { label: 'JSON', value: 'application/json' },
                            {
                              label: t('URL编码表单'),
                              value: 'application/x-www-form-urlencoded',
                            },
                          ]}
                        />
                        <Form.TextArea
                          field='headers_json'
                          label={t('请求头JSON')}
                          autosize={{ minRows: 3, maxRows: 8 }}
                          className='md:col-span-2'
                        />
                        <Form.TextArea
                          field='body_template'
                          label={t('发送参数模板JSON')}
                          autosize={{ minRows: 6, maxRows: 14 }}
                          className='md:col-span-3'
                          extraText={t(
                            '必须保留number、timestamp、sign变量；签名原文固定为number={number}&timestamp={timestamp}。',
                          )}
                        />
                        <Form.TextArea
                          field='callback_config_json'
                          label={t('回调参数JSON')}
                          autosize={{ minRows: 3, maxRows: 8 }}
                          className='md:col-span-3'
                          extraText={t(
                            '可填写callback_url等回调参数，并通过callback_config模板变量注入发送参数模板。',
                          )}
                        />
                      </div>
                      <input
                        ref={privateKeyInputRef}
                        type='file'
                        accept='.pem,.key,text/plain'
                        style={{ display: 'none' }}
                        onChange={handlePrivateKeyFile}
                      />
                      <Space wrap>
                        <Button
                          onClick={() => privateKeyInputRef.current?.click()}
                        >
                          {t('上传RSA私钥')}
                        </Button>
                        <Text>
                          {privateKeyName ||
                            (metadata.has_private_key
                              ? t('已保存私钥')
                              : t('尚未上传私钥'))}
                        </Text>
                      </Space>
                      {metadata.private_key_fingerprint ? (
                        <div className='mt-3 break-all'>
                          <Text type='tertiary'>
                            SHA-256: {metadata.private_key_fingerprint}
                          </Text>
                        </div>
                      ) : null}
                    </Card>
                  ) : (
                    <Card title={t('通用请求配置')} className='mb-4'>
                      <div className='grid grid-cols-1 md:grid-cols-3 gap-x-4'>
                        <Form.Input
                          field='endpoint'
                          label={t('推送地址')}
                          className='md:col-span-2'
                        />
                        <Form.Select
                          field='http_method'
                          label={t('请求方法')}
                          optionList={['POST', 'PUT', 'PATCH', 'GET'].map(
                            (value) => ({ label: value, value }),
                          )}
                        />
                        <Form.Input
                          field='content_type'
                          label='Content-Type'
                          className='md:col-span-2'
                        />
                        <Form.TextArea
                          field='headers_json'
                          label={t('请求头JSON')}
                          autosize={{ minRows: 3, maxRows: 8 }}
                          className='md:col-span-3'
                        />
                        <Form.TextArea
                          field='body_template'
                          label={t('请求体模板')}
                          autosize={{ minRows: 8, maxRows: 16 }}
                          className='md:col-span-3'
                          extraText={t(
                            '支持number、timestamp、supplier_id、batch_no、period_start、period_end、currency等模板变量。',
                          )}
                        />
                        <Form.TextArea
                          field='callback_config_json'
                          label={t('回调参数JSON')}
                          autosize={{ minRows: 3, maxRows: 8 }}
                          className='md:col-span-3'
                          extraText={t(
                            '可通过callback_config模板变量注入请求体；异步回调验签需按渠道协议单独适配。',
                          )}
                        />
                      </div>
                    </Card>
                  )}
                  <Card title={t('同步返回成功判定')} className='mb-4'>
                    <Banner
                      type='info'
                      className='mb-4'
                      description={t(
                        '这里配置的是推送请求收到的即时响应，不是异步回调地址。默认规则已经适配下方示例，一般无需修改。',
                      )}
                    />
                    <div className='grid grid-cols-1 md:grid-cols-2 gap-4 mb-4'>
                      <div className='rounded-lg border border-solid border-[var(--semi-color-border)] bg-[var(--semi-color-fill-0)] p-3'>
                        <Text strong>{t('成功响应示例')}</Text>
                        <pre className='mt-2 mb-0 whitespace-pre-wrap break-all text-xs'>
                          {EORAPTOR_SUCCESS_RESPONSE_EXAMPLE}
                        </pre>
                      </div>
                      <div className='rounded-lg border border-solid border-[var(--semi-color-border)] p-3'>
                        <Text strong>{t('快速配置')}</Text>
                        <div className='mt-3 flex flex-wrap gap-2'>
                          <Button
                            onClick={() => applyResponseRulePreset('eoraptor')}
                          >
                            {t('使用模板')}
                          </Button>
                          <Button
                            onClick={() => applyResponseRulePreset('http_only')}
                          >
                            {t('仅判断HTTP 200')}
                          </Button>
                        </div>
                        <div className='mt-3'>
                          <Text type='tertiary' size='small'>
                            {t(
                              '标准规则会检查HTTP 200、code=200、type=success，并确认返回金额与推送金额一致。仅HTTP模式适合不返回JSON的Webhook。',
                            )}
                          </Text>
                        </div>
                      </div>
                    </div>
                    <Collapse keepDOM>
                      <Collapse.Panel
                        header={t('高级规则（按需修改）')}
                        itemKey='response-rules'
                      >
                        <Text type='tertiary' size='small'>
                          {t(
                            '字段路径使用点号读取嵌套JSON，例如result.number表示result对象中的number；路径留空即不检查该项。',
                          )}
                        </Text>
                        <div className='grid grid-cols-1 md:grid-cols-3 gap-x-4 mt-3'>
                          <Form.InputNumber
                            field='success_http_status'
                            label={t('成功HTTP状态')}
                            placeholder='200'
                          />
                          <Form.Input
                            field='success_code_path'
                            label={t('业务码字段路径')}
                            placeholder='code'
                          />
                          <Form.Input
                            field='success_code_value'
                            label={t('业务码成功值')}
                            placeholder='200'
                          />
                          <Form.Input
                            field='success_type_path'
                            label={t('状态字段路径')}
                            placeholder='type'
                          />
                          <Form.Input
                            field='success_type_value'
                            label={t('状态成功值')}
                            placeholder='success'
                          />
                          <Form.Input
                            field='success_amount_path'
                            label={t('返回金额字段路径')}
                            placeholder='result.number'
                          />
                        </div>
                      </Collapse.Panel>
                    </Collapse>
                  </Card>
                </Form>
              </Tabs.TabPane>

              <Tabs.TabPane tab={t('推送日志')} itemKey='logs'>
                <div className='flex justify-end mt-5 mb-3'>
                  <Button onClick={() => loadDeliveries(deliveryPage)}>
                    {t('刷新')}
                  </Button>
                </div>
                <Table
                  rowKey='id'
                  columns={deliveryColumns}
                  dataSource={deliveries}
                  loading={deliveryLoading}
                  scroll={{ x: 1630 }}
                  pagination={{
                    currentPage: deliveryPage,
                    pageSize: 10,
                    total: deliveryTotal,
                    showSizeChanger: false,
                    onPageChange: loadDeliveries,
                  }}
                />
              </Tabs.TabPane>
            </Tabs>
          </div>
        </Spin>
      </SideSheet>

      <Modal
        visible={manualVisible}
        title={t('手动推送指定金额')}
        okText={t('立即推送')}
        cancelText={t('取消')}
        confirmLoading={manualSending}
        onOk={submitManualPush}
        onCancel={() => setManualVisible(false)}
        maskClosable={!manualSending}
      >
        <Banner
          type='warning'
          className='mb-4'
          description={t(
            '手动推送不会修改自动账期统计，也不会将任何账期标记为已结算。请确认该金额尚未通过其他批次发送，避免重复入账。',
          )}
        />
        <div className='mb-4'>
          <div className='mb-2'>
            <Text strong>
              {t('推送金额')}（
              {formApi?.getValues()?.currency || formValues.currency}）
            </Text>
          </div>
          <Input
            value={manualAmount}
            onChange={setManualAmount}
            placeholder='0.000000'
            disabled={manualSending}
          />
          <div className='mt-2'>
            <Text type='tertiary' size='small'>
              {t('金额将按统一规则处理为六位小数，0金额也允许推送。')}
            </Text>
          </div>
        </div>
        <div>
          <div className='mb-2'>
            <Text strong>{t('备注（可选）')}</Text>
          </div>
          <TextArea
            value={manualRemark}
            onChange={setManualRemark}
            maxCount={500}
            autosize={{ minRows: 3, maxRows: 6 }}
            placeholder={t('例如：补发历史结算金额、财务人工调整')}
            disabled={manualSending}
          />
        </div>
      </Modal>

      <Modal
        visible={attemptVisible}
        title={`${t('请求明细')} · ${selectedDelivery?.batch_no || ''}`}
        width={900}
        footer={null}
        onCancel={() => setAttemptVisible(false)}
      >
        <Table
          rowKey='id'
          columns={attemptColumns}
          dataSource={attempts}
          loading={attemptLoading}
          pagination={false}
          scroll={{ x: 900 }}
          expandedRowRender={(row) => (
            <div className='space-y-3'>
              <div>
                <Text strong>{t('请求参数')}</Text>
                <pre className='mt-1 whitespace-pre-wrap break-all'>
                  {row.request_body || '-'}
                </pre>
              </div>
              <div>
                <Text strong>{t('响应内容')}</Text>
                <pre className='mt-1 whitespace-pre-wrap break-all'>
                  {row.response_body || '-'}
                </pre>
              </div>
            </div>
          )}
        />
      </Modal>
    </>
  );
};

export default SupplierRevenuePushSideSheet;
