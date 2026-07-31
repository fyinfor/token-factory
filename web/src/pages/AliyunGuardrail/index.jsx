import React, { useEffect, useState } from 'react';
import {
  Button,
  Form,
  Modal,
  Table,
  Tag,
  Toast,
  Tooltip,
} from '@douyinfe/semi-ui';
import { IconCopy, IconSearch } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { API, copy, toUnixTimestamp } from '../../helpers';
import { DATE_RANGE_PRESETS } from '../../constants/console.constants';

const riskStyles = {
  high: { color: 'red', label: '安全护栏风险：高风险' },
  medium: { color: 'orange', label: '安全护栏风险：中风险' },
  low: { color: 'green', label: '安全护栏风险：低风险' },
  none: { color: 'green', label: '安全护栏风险：无风险' },
  block: { color: 'red', label: '安全护栏风险：已拦截' },
  pass: { color: 'green', label: '安全护栏风险：通过' },
  error: { color: 'red', label: '安全护栏风险：调用失败' },
};

function RiskTag({ value, t }) {
  const normalized = String(value || '').toLowerCase();
  const config = riskStyles[normalized];
  return (
    <Tag color={config?.color || 'grey'} size='small'>
      {config ? t(config.label) : value || t('安全护栏风险：未知')}
    </Tag>
  );
}

function parseModerationDetail(value, fallbackRisk, t) {
  try {
    const parsed = JSON.parse(value);
    const descriptions = [
      ...(parsed.Result || []),
      ...(parsed.AttackResult || []),
    ]
      .map((item) => item?.Description)
      .filter(Boolean);
    return {
      riskLevel: parsed.RiskLevel || parsed.AttackLevel || fallbackRisk,
      description: [...new Set(descriptions)].join('；') || t('未返回风险说明'),
      formatted: JSON.stringify(parsed, null, 2),
    };
  } catch {
    return {
      riskLevel: fallbackRisk,
      description: value || '-',
      formatted: value || '-',
    };
  }
}

function PreviewBlock({ value, onView, actionText, accent }) {
  if (!value) return '-';
  return (
    <div
      style={{
        width: 260,
        height: 74,
        padding: '8px 10px',
        overflow: 'hidden',
        border: '1px solid var(--semi-color-border)',
        borderRadius: 6,
        background: 'var(--semi-color-fill-0)',
        boxSizing: 'border-box',
      }}
    >
      <div
        style={{
          display: '-webkit-box',
          height: 36,
          overflow: 'hidden',
          WebkitBoxOrient: 'vertical',
          WebkitLineClamp: 2,
          wordBreak: 'break-all',
          overflowWrap: 'anywhere',
          fontSize: 12,
          lineHeight: '18px',
          color: accent || 'var(--semi-color-text-1)',
        }}
      >
        {value}
      </div>
      <Button
        theme='borderless'
        type='tertiary'
        size='small'
        onClick={onView}
        style={{
          marginTop: 4,
          padding: 0,
          height: 18,
          lineHeight: '18px',
          fontSize: 12,
        }}
      >
        {actionText}
      </Button>
    </div>
  );
}

function DetailPreview({ value, riskLevel, onView, t }) {
  const detail = parseModerationDetail(value, riskLevel, t);
  return (
    <div
      style={{
        width: 280,
        height: 74,
        padding: '8px 10px',
        overflow: 'hidden',
        border: '1px solid var(--semi-color-border)',
        borderRadius: 6,
        background: 'var(--semi-color-fill-0)',
        boxSizing: 'border-box',
      }}
    >
      <div className='flex items-start gap-2'>
        <div style={{ flex: '0 0 auto' }}>
          <RiskTag value={detail.riskLevel} t={t} />
        </div>
        <div
          style={{
            display: '-webkit-box',
            flex: 1,
            height: 36,
            overflow: 'hidden',
            WebkitBoxOrient: 'vertical',
            WebkitLineClamp: 2,
            wordBreak: 'break-all',
            overflowWrap: 'anywhere',
            fontSize: 12,
            lineHeight: '18px',
            color: 'var(--semi-color-text-1)',
          }}
        >
          {detail.description}
        </div>
      </div>
      <Button
        theme='borderless'
        type='tertiary'
        size='small'
        onClick={() => onView(detail.formatted)}
        style={{
          marginTop: 4,
          padding: 0,
          height: 18,
          lineHeight: '18px',
          fontSize: 12,
        }}
      >
        {t('查看完整详情')}
      </Button>
    </div>
  );
}

function ChannelSuffix({ record, t }) {
  const suffix = String(record?.route_slug || '').trim();
  if (!suffix) return '-';
  return (
    <Tooltip content={`${t('渠道路由后缀')}：${suffix}`}>
      <Tag color='blue' size='small' style={{ maxWidth: 120 }}>
        <span
          style={{
            display: 'inline-block',
            maxWidth: 96,
            overflow: 'hidden',
            textOverflow: 'ellipsis',
            verticalAlign: 'bottom',
            whiteSpace: 'nowrap',
          }}
        >
          {suffix}
        </span>
      </Tag>
    </Tooltip>
  );
}

function shortRequestId(value) {
  const requestId = String(value || '').trim();
  if (!requestId) return '-';
  if (requestId.length <= 10) return requestId;
  return `…${requestId.slice(-8)}`;
}

function RequestIdCell({ value, t }) {
  const requestId = String(value || '').trim();
  if (!requestId) return '-';
  const handleCopy = async () => {
    if (await copy(requestId)) {
      Toast.success({ content: t('请求 ID 已复制') });
    }
  };
  return (
    <div className='flex items-center gap-1'>
      <span
        style={{
          maxWidth: 92,
          overflow: 'hidden',
          textOverflow: 'ellipsis',
          whiteSpace: 'nowrap',
          color: 'var(--semi-color-text-2)',
          fontFamily:
            'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
          fontSize: 12,
        }}
      >
        {shortRequestId(requestId)}
      </span>
      <Tooltip content={t('复制请求 ID')}>
        <Button
          icon={<IconCopy />}
          theme='borderless'
          type='tertiary'
          size='small'
          onClick={handleCopy}
          style={{ padding: 0, width: 24, height: 24 }}
        />
      </Tooltip>
    </div>
  );
}

export default function AliyunGuardrail() {
  const { t } = useTranslation();
  const [data, setData] = useState([]);
  const [loading, setLoading] = useState(false);
  const [formApi, setFormApi] = useState(null);
  const [detailModal, setDetailModal] = useState({
    visible: false,
    title: '',
    value: '',
  });

  const showDetail = (title, value) => {
    setDetailModal({ visible: true, title, value: value || '-' });
  };

  const fetchLogs = (formValues = {}) => {
    const params = new URLSearchParams({ p: '1', page_size: '50' });
    const username = String(formValues.username || '').trim();
    if (username) {
      params.set('username', username);
    }
    if (
      Array.isArray(formValues.dateRange) &&
      formValues.dateRange.length === 2
    ) {
      const startTimestamp = toUnixTimestamp(formValues.dateRange[0]);
      const endTimestamp = toUnixTimestamp(formValues.dateRange[1]);
      if (startTimestamp > 0) {
        params.set('start_timestamp', String(startTimestamp));
      }
      if (endTimestamp > 0) {
        params.set('end_timestamp', String(endTimestamp));
      }
    }
    setLoading(true);
    API.get(`/api/log/aliyun-guardrail?${params.toString()}`)
      .then((res) => {
        if (res.data.success) setData(res.data.data.items || []);
        else Toast.error(res.data.message || t('加载安全护栏记录失败'));
      })
      .catch((error) => {
        const status = error?.response?.status;
        const message = error?.response?.data?.message || error?.message;
        Toast.error(
          t('加载安全护栏记录失败') +
            (status ? `（HTTP ${status}）` : '') +
            (message ? `：${message}` : ''),
        );
      })
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    fetchLogs();
  }, []);

  return (
    <div className='mt-[60px] px-2'>
      <Form
        getFormApi={setFormApi}
        onSubmit={fetchLogs}
        allowEmpty
        layout='vertical'
        className='mb-3'
      >
        <div className='grid grid-cols-1 gap-2 md:grid-cols-2 lg:grid-cols-4'>
          <div className='lg:col-span-2'>
            <Form.DatePicker
              field='dateRange'
              className='w-full'
              type='dateTimeRange'
              placeholder={[t('开始时间'), t('结束时间')]}
              showClear
              pure
              size='small'
              presets={DATE_RANGE_PRESETS.map((preset) => ({
                text: t(preset.text),
                start: preset.start(),
                end: preset.end(),
              }))}
            />
          </div>
          <Form.Input
            field='username'
            prefix={<IconSearch />}
            placeholder={t('用户名称')}
            showClear
            pure
            size='small'
          />
          <div className='flex items-center gap-2'>
            <Button
              htmlType='submit'
              type='primary'
              size='small'
              loading={loading}
            >
              {t('搜索')}
            </Button>
            <Button
              size='small'
              onClick={() => {
                formApi?.reset();
                fetchLogs();
              }}
            >
              {t('重置')}
            </Button>
          </div>
        </div>
      </Form>
      <Table
        loading={loading}
        dataSource={data}
        rowKey='id'
        pagination={{ pageSize: 20 }}
        columns={[
          {
            title: t('时间'),
            dataIndex: 'created_at',
            render: (value) => new Date(value * 1000).toLocaleString(),
          },
          { title: t('用户'), dataIndex: 'username' },
          { title: t('方向'), dataIndex: 'direction' },
          {
            title: t('风险等级'),
            dataIndex: 'risk_level',
            render: (value) => <RiskTag value={value} t={t} />,
          },
          { title: t('服务'), dataIndex: 'service' },
          { title: t('模型'), dataIndex: 'model_name', width: 150 },
          {
            title: t('渠道后缀'),
            dataIndex: 'channel',
            width: 130,
            render: (value, record) => <ChannelSuffix record={record} t={t} />,
          },
          {
            title: t('请求 ID'),
            dataIndex: 'request_id',
            width: 130,
            render: (value) => <RequestIdCell value={value} t={t} />,
          },
          {
            title: t('内容'),
            dataIndex: 'content',
            width: 280,
            render: (value) => (
              <PreviewBlock
                value={value}
                actionText={t('查看完整内容')}
                onView={() => showDetail(t('审核内容'), value)}
              />
            ),
          },
          {
            title: t('详情'),
            dataIndex: 'detail',
            width: 300,
            render: (value, record) => (
              <DetailPreview
                value={value}
                riskLevel={record.risk_level}
                onView={(formatted) => showDetail(t('审核详情'), formatted)}
                t={t}
              />
            ),
          },
        ]}
        scroll={{ x: 1540 }}
      />
      <Modal
        title={detailModal.title}
        visible={detailModal.visible}
        footer={null}
        width={720}
        onCancel={() =>
          setDetailModal((current) => ({ ...current, visible: false }))
        }
      >
        <pre
          style={{
            maxHeight: '60vh',
            margin: 0,
            padding: 12,
            overflow: 'auto',
            whiteSpace: 'pre-wrap',
            overflowWrap: 'anywhere',
            borderRadius: 6,
            background: 'var(--semi-color-fill-0)',
            fontSize: 12,
            lineHeight: 1.6,
          }}
        >
          {detailModal.value}
        </pre>
      </Modal>
    </div>
  );
}
