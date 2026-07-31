/*
Copyright (C) 2025 QuantumNous
*/

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Button,
  Checkbox,
  CheckboxGroup,
  Form,
  Radio,
  RadioGroup,
  Select,
  Spin,
  Tooltip,
  Typography,
} from '@douyinfe/semi-ui';
import {
  IconDownload,
  IconHelpCircle,
  IconRefresh,
} from '@douyinfe/semi-icons';
import { BarChart3, FileSpreadsheet } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import CardPro from '../../../components/common/ui/CardPro';
import { API, showError, showSuccess } from '../../../helpers';
import { getTodayStartTimestamp } from '../../../helpers/utils';
import { normalizeLanguage } from '../../../i18n/language';
import { DATE_RANGE_PRESETS } from '../../../constants/console.constants';
import dayjs from 'dayjs';

const SETTLEMENT_DATE_RANGE_PRESETS = [
  ...DATE_RANGE_PRESETS,
  {
    text: '近 3 个月',
    start: () => dayjs().subtract(89, 'day').startOf('day').toDate(),
    end: () => dayjs().endOf('day').toDate(),
  },
  {
    text: '近 1 年',
    start: () => dayjs().subtract(364, 'day').startOf('day').toDate(),
    end: () => dayjs().endOf('day').toDate(),
  },
];

const SETTLEMENT_CARD_CLASS = '!h-auto !max-h-none';

const { Text, Title } = Typography;

const HelpHint = ({ content }) => (
  <Tooltip content={content}>
    <IconHelpCircle
      size='small'
      className='text-gray-400 cursor-help'
      style={{ verticalAlign: 'middle' }}
    />
  </Tooltip>
);

const LabelWithHelp = ({ label, helpContent }) => (
  <span className='inline-flex items-center gap-1'>
    {label}
    {helpContent ? <HelpHint content={helpContent} /> : null}
  </span>
);

const COLUMN_GROUPS = [
  {
    label: '基础信息',
    keys: [
      { key: 'seq', label: '序号' },
      { key: 'request_id', label: '订单ID' },
      { key: 'time', label: '时间' },
    ],
  },
  {
    label: '主体信息',
    keys: [
      { key: 'channel', label: '渠道' },
      { key: 'model', label: '模型' },
      { key: 'agent', label: '代理商' },
      { key: 'user', label: '用户' },
    ],
  },
  {
    label: 'Token用量',
    keys: [
      { key: 'prompt_tokens', label: '输入 tokens' },
      { key: 'completion_tokens', label: '输出 tokens' },
      { key: 'cache_tokens', label: '缓存 tokens' },
    ],
  },
  {
    label: '官方价格',
    helpKey: '结算导出官方价格说明',
    keys: [
      { key: 'official_input_price', label: '官方输入价格' },
      { key: 'official_output_price', label: '官方输出价格' },
      { key: 'official_cache_price', label: '官方缓存价格' },
      {
        key: 'official_total',
        label: '官方总价',
        helpKey: '结算导出官方总价说明',
      },
    ],
  },
  {
    label: '折扣信息',
    helpKey: '结算导出折扣信息说明',
    keys: [
      { key: 'cost_discount', label: '成本折扣' },
      { key: 'operating_cost', label: '经营成本' },
      { key: 'markup_discount', label: '加价折扣' },
      { key: 'sales_discount', label: '销售折扣' },
    ],
  },
  {
    label: '结算价格',
    helpKey: '结算导出结算价格说明',
    keys: [
      { key: 'cost_price', label: '成本价' },
      { key: 'operating_price', label: '经营单价' },
      { key: 'sales_price', label: '销售单价' },
      { key: 'quota', label: '用户实付' },
    ],
  },
];

const ALL_KEYS = COLUMN_GROUPS.flatMap((g) => g.keys.map((k) => k.key));

const SUMMARY_METRICS = [
  { key: 'record_count', label: '请求笔数', isCount: true },
  { key: 'prompt_tokens', label: '输入 tokens', isCount: true },
  { key: 'completion_tokens', label: '输出 tokens', isCount: true },
  { key: 'cache_tokens', label: '缓存 tokens', isCount: true },
  { key: 'official_total', label: '官方总价' },
  { key: 'cost_price', label: '成本价' },
  { key: 'operating_price', label: '经营单价' },
  { key: 'sales_price', label: '销售单价' },
  { key: 'user_paid', label: '用户实付' },
];

const blobToText = async (blob) => {
  if (typeof blob?.text === 'function') {
    return blob.text();
  }
  return '';
};

const dateRangeToTimestamps = (dateRange) => {
  const today = getTodayStartTimestamp();
  const defaultStart = today - 90 * 24 * 60 * 60;
  const defaultEnd = today + 24 * 60 * 60 - 1;
  if (
    !Array.isArray(dateRange) ||
    dateRange.length < 2 ||
    !dateRange[0] ||
    !dateRange[1]
  ) {
    return { startTs: defaultStart, endTs: defaultEnd, valid: false };
  }
  const startTs = Math.floor(new Date(dateRange[0]).getTime() / 1000);
  const endTs = Math.floor(new Date(dateRange[1]).getTime() / 1000);
  return { startTs, endTs, valid: true };
};

const formatUserLabel = (user) => {
  if (!user) return '';
  const name = user.username || `ID ${user.id}`;
  return user.display_name ? `${name} (${user.display_name})` : name;
};

const formatCount = (value) => {
  const num = Number(value);
  if (!Number.isFinite(num)) return '0';
  return num.toLocaleString();
};

const SummaryMetricRow = ({ label, value, isCount }) => (
  <div className='flex items-start justify-between gap-3 py-1.5 border-b border-gray-100 last:border-b-0'>
    <Text type='tertiary' size='small'>
      {label}
    </Text>
    <Text strong className='text-right break-all'>
      {isCount ? formatCount(value) : value || '-'}
    </Text>
  </div>
);

const SummaryAmountsBlock = ({ title, amounts, t, highlight = false }) => (
  <div
    className={`mb-4 last:mb-0 ${highlight ? 'rounded-xl border border-emerald-200 bg-emerald-50/60 dark:bg-emerald-950/20 dark:border-emerald-900 p-3' : ''}`}
  >
    {title ? (
      <Text strong className='block mb-2'>
        {title}
      </Text>
    ) : null}
    {SUMMARY_METRICS.map((metric) => (
      <SummaryMetricRow
        key={metric.key}
        label={t(metric.label)}
        value={amounts?.[metric.key]}
        isCount={metric.isCount}
      />
    ))}
  </div>
);

const SettlementSummaryPanel = ({
  t,
  loading,
  error,
  summary,
  scope,
  canFetch,
  onRefresh,
}) => {
  const groupTitle = useMemo(() => {
    switch (scope) {
      case 'user':
        return t('用户明细');
      case 'agent':
        return t('代理明细');
      default:
        return t('渠道明细');
    }
  }, [scope, t]);

  return (
    <CardPro
      type='type1'
      t={t}
      className={SETTLEMENT_CARD_CLASS}
      descriptionArea={
        <div className='flex items-center justify-between gap-2 w-full'>
          <div className='flex items-center text-emerald-600 min-w-0'>
            <BarChart3 size={16} className='mr-2 shrink-0' />
            <Text>{t('结算汇总')}</Text>
          </div>
          <Button
            icon={<IconRefresh />}
            size='small'
            theme='borderless'
            disabled={!canFetch || loading}
            onClick={onRefresh}
          />
        </div>
      }
    >
      <Text type='tertiary' size='small' className='block mb-4'>
        {t('结算汇总说明')}
      </Text>

      {!canFetch ? (
        <Text type='tertiary'>{t('完成筛选后查看汇总')}</Text>
      ) : loading ? (
        <div className='flex items-center justify-center py-10'>
          <Spin tip={t('汇总加载中')} />
        </div>
      ) : error ? (
        <Text type='danger'>{error}</Text>
      ) : !summary ? (
        <Text type='tertiary'>{t('请选择账期后查看汇总')}</Text>
      ) : (
        <>
          {summary.currency ? (
            <Text type='tertiary' size='small' className='block mb-3'>
              {summary.currency}
            </Text>
          ) : null}

          <SummaryAmountsBlock
            title={t('汇总合计')}
            amounts={summary.totals}
            t={t}
            highlight
          />

          {summary.groups?.length ? (
            <div className='mt-4 pt-4 border-t border-gray-200'>
              <Title heading={6} className='!mb-3'>
                {groupTitle}
              </Title>
              {summary.groups_truncated ? (
                <Text type='warning' size='small' className='block mb-3'>
                  {t('分组过多提示', { count: summary.groups.length })}
                </Text>
              ) : null}
              <div className='flex flex-col gap-4 max-h-[480px] overflow-y-auto pr-1'>
                {summary.groups.map((group) => (
                  <div
                    key={group.key}
                    className='rounded-lg border border-gray-200 bg-gray-50/80 p-3'
                  >
                    <Text strong className='block mb-2 break-all'>
                      {group.label}
                    </Text>
                    <SummaryAmountsBlock amounts={group} t={t} />
                  </div>
                ))}
              </div>
            </div>
          ) : null}
        </>
      )}
    </CardPro>
  );
};

const SettingsSettlementExport = () => {
  const { t, i18n } = useTranslation();
  const today = getTodayStartTimestamp();
  const defaultDateRange = useMemo(
    () => [new Date((today - 90 * 24 * 60 * 60) * 1000), new Date()],
    [today],
  );
  const [scope, setScope] = useState('platform');
  const [dateRange, setDateRange] = useState(defaultDateRange);
  const [selectedColumns, setSelectedColumns] = useState(ALL_KEYS);
  const [selectedChannelIds, setSelectedChannelIds] = useState([]);
  const [selectedUserIds, setSelectedUserIds] = useState([]);
  const [selectedAgentIds, setSelectedAgentIds] = useState([]);
  const [channelOptions, setChannelOptions] = useState([]);
  const [userOptions, setUserOptions] = useState([]);
  const [agentOptions, setAgentOptions] = useState([]);
  const [optionsLoading, setOptionsLoading] = useState(false);
  const [userSearchLoading, setUserSearchLoading] = useState(false);
  const [exporting, setExporting] = useState(false);
  const [summaryLoading, setSummaryLoading] = useState(false);
  const [summaryData, setSummaryData] = useState(null);
  const [summaryError, setSummaryError] = useState('');
  const [summaryRefreshKey, setSummaryRefreshKey] = useState(0);

  const loadChannelOptions = useCallback(async () => {
    const res = await API.get('/api/channel/?p=1&page_size=500');
    if (!res.data?.success) return;
    const items = res.data.data?.items || [];
    setChannelOptions(
      items.map((ch) => ({
        value: ch.id,
        label: ch.name ? `#${ch.id} ${ch.name}` : `#${ch.id}`,
      })),
    );
  }, []);

  const loadAgentOptions = useCallback(async () => {
    const res = await API.get('/api/distributor/admin/distributors', {
      params: { p: 1, page_size: 500 },
    });
    if (!res.data?.success) return;
    const items = res.data.data?.items || [];
    setAgentOptions(
      items.map((row) => ({
        value: row.id,
        label: formatUserLabel(row),
      })),
    );
  }, []);

  const searchUsers = useCallback(async (keyword = '') => {
    setUserSearchLoading(true);
    try {
      const res = await API.get('/api/user/search', {
        params: { keyword: keyword || '', p: 1, page_size: 50 },
      });
      if (!res.data?.success) return;
      const items = res.data.data?.items || [];
      setUserOptions(
        items.map((user) => ({
          value: user.id,
          label: formatUserLabel(user),
        })),
      );
    } finally {
      setUserSearchLoading(false);
    }
  }, []);

  useEffect(() => {
    setOptionsLoading(true);
    Promise.all([
      loadChannelOptions(),
      loadAgentOptions(),
      searchUsers(''),
    ]).finally(() => {
      setOptionsLoading(false);
    });
  }, [loadChannelOptions, loadAgentOptions, searchUsers]);

  useEffect(() => {
    setSelectedChannelIds([]);
    setSelectedUserIds([]);
    setSelectedAgentIds([]);
  }, [scope]);

  const canFetchSummary = useMemo(() => {
    const { valid } = dateRangeToTimestamps(dateRange);
    if (!valid) return false;
    if (scope === 'channel' && !selectedChannelIds.length) return false;
    if (scope === 'user' && !selectedUserIds.length) return false;
    if (scope === 'agent' && !selectedAgentIds.length) return false;
    return true;
  }, [dateRange, scope, selectedChannelIds, selectedUserIds, selectedAgentIds]);

  const buildSettlementQueryParams = useCallback(
    (startTs, endTs) => {
      const params = {
        start_timestamp: String(startTs),
        end_timestamp: String(endTs),
        scope,
        lang: normalizeLanguage(i18n.language || '') || '',
      };
      if (scope === 'channel' && selectedChannelIds.length) {
        params.channel_ids = selectedChannelIds.join(',');
      }
      if (scope === 'user' && selectedUserIds.length) {
        params.user_ids = selectedUserIds.join(',');
      }
      if (scope === 'agent' && selectedAgentIds.length) {
        params.inviter_ids = selectedAgentIds.join(',');
      }
      return params;
    },
    [
      scope,
      i18n.language,
      selectedChannelIds,
      selectedUserIds,
      selectedAgentIds,
    ],
  );

  const fetchSummary = useCallback(async () => {
    if (!canFetchSummary) {
      setSummaryData(null);
      setSummaryError('');
      return;
    }
    const { startTs, endTs } = dateRangeToTimestamps(dateRange);
    setSummaryLoading(true);
    setSummaryError('');
    try {
      const res = await API.get('/api/log/settlement/summary', {
        params: buildSettlementQueryParams(startTs, endTs),
      });
      if (res.data?.success) {
        setSummaryData(res.data.data);
      } else {
        setSummaryData(null);
        setSummaryError(res.data?.message || t('汇总加载失败'));
      }
    } catch (e) {
      setSummaryData(null);
      setSummaryError(
        e?.response?.data?.message || e?.message || t('汇总加载失败'),
      );
    } finally {
      setSummaryLoading(false);
    }
  }, [buildSettlementQueryParams, canFetchSummary, dateRange, t]);

  useEffect(() => {
    if (!canFetchSummary) {
      setSummaryData(null);
      setSummaryError('');
      return undefined;
    }
    const timer = setTimeout(() => {
      fetchSummary();
    }, 500);
    return () => clearTimeout(timer);
  }, [
    canFetchSummary,
    dateRange,
    scope,
    selectedChannelIds,
    selectedUserIds,
    selectedAgentIds,
    summaryRefreshKey,
    fetchSummary,
  ]);

  const columnOptions = useMemo(
    () =>
      COLUMN_GROUPS.map((group) => (
        <div key={group.label} className='mb-3'>
          <div className='font-medium mb-1'>
            <LabelWithHelp
              label={t(group.label)}
              helpContent={group.helpKey ? t(group.helpKey) : null}
            />
          </div>
          <CheckboxGroup
            value={selectedColumns}
            onChange={setSelectedColumns}
            direction='horizontal'
          >
            {group.keys.map((item) => (
              <Checkbox key={item.key} value={item.key}>
                <LabelWithHelp
                  label={t(item.label)}
                  helpContent={item.helpKey ? t(item.helpKey) : null}
                />
              </Checkbox>
            ))}
          </CheckboxGroup>
        </div>
      )),
    [selectedColumns, t],
  );

  const handleExport = async (values) => {
    if (!selectedColumns.length) {
      showError(t('请至少选择一个导出字段'));
      return;
    }
    const range = values?.dateRange || dateRange;
    const { startTs, endTs, valid } = dateRangeToTimestamps(range);
    if (!valid) {
      showError(t('请选择账期'));
      return;
    }
    if (scope === 'channel' && !selectedChannelIds.length) {
      showError(t('请至少选择一个渠道'));
      return;
    }
    if (scope === 'user' && !selectedUserIds.length) {
      showError(t('请至少选择一个用户'));
      return;
    }
    if (scope === 'agent' && !selectedAgentIds.length) {
      showError(t('请至少选择一个代理商'));
      return;
    }

    setExporting(true);
    try {
      const params = new URLSearchParams({
        ...buildSettlementQueryParams(startTs, endTs),
        columns: selectedColumns.join(','),
      });
      const res = await API.get(
        `/api/log/settlement/export?${params.toString()}`,
        {
          responseType: 'blob',
        },
      );
      const blob = new Blob([res.data], { type: 'text/csv;charset=utf-8' });
      const objectUrl = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = objectUrl;
      a.download = `settlement-${new Date()
        .toISOString()
        .replace(/[-:T.Z]/g, '')
        .slice(0, 14)}.csv`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(objectUrl);
      showSuccess(t('结算单导出成功'));
    } catch (e) {
      const text = e?.response?.data
        ? await blobToText(e.response.data)
        : e?.message || String(e);
      showError(t('结算单导出失败') + ': ' + text);
    } finally {
      setExporting(false);
    }
  };

  return (
    <div className='grid grid-cols-1 lg:grid-cols-[minmax(0,1fr)_380px] gap-4 items-start'>
      <CardPro
        type='type1'
        t={t}
        className={`${SETTLEMENT_CARD_CLASS} order-2 lg:order-1`}
        descriptionArea={
          <div className='flex flex-col gap-1 w-full'>
            <div className='flex items-center text-emerald-600'>
              <FileSpreadsheet size={16} className='mr-2' />
              <Text>{t('结算单导出')}</Text>
            </div>
            <Text type='tertiary' size='small'>
              {t('结算单导出说明')}
            </Text>
          </div>
        }
      >
        <Form
          layout='vertical'
          onSubmit={handleExport}
          initValues={{ dateRange: defaultDateRange }}
        >
          <Form.DatePicker
            field='dateRange'
            className='w-full mb-4'
            label={t('账期')}
            type='dateTimeRange'
            rules={[{ required: true, message: t('请选择账期') }]}
            placeholder={[t('开始时间'), t('结束时间')]}
            presets={SETTLEMENT_DATE_RANGE_PRESETS.map((preset) => ({
              text: t(preset.text),
              start: preset.start(),
              end: preset.end(),
            }))}
            onChange={(value) => setDateRange(value)}
          />

          <div className='mb-4'>
            <div className='mb-2 font-medium'>{t('结算视角')}</div>
            <RadioGroup
              value={scope}
              onChange={(e) => setScope(e.target.value)}
            >
              <Radio value='platform'>{t('全平台')}</Radio>
              <Radio value='channel'>{t('按渠道')}</Radio>
              <Radio value='user'>{t('按用户')}</Radio>
              <Radio value='agent'>{t('按代理')}</Radio>
            </RadioGroup>
          </div>

          {scope === 'channel' ? (
            <div className='mb-4'>
              <div className='mb-1 text-sm text-gray-600'>{t('选择渠道')}</div>
              <Select
                multiple
                filter
                placeholder={t('选择渠道')}
                optionList={channelOptions}
                value={selectedChannelIds}
                onChange={setSelectedChannelIds}
                loading={optionsLoading}
                maxTagCount={3}
                style={{ width: '100%' }}
              />
            </div>
          ) : null}

          {scope === 'user' ? (
            <div className='mb-4'>
              <div className='mb-1 text-sm text-gray-600'>{t('选择用户')}</div>
              <Select
                multiple
                filter
                remote
                placeholder={t('选择用户')}
                optionList={userOptions}
                value={selectedUserIds}
                onChange={setSelectedUserIds}
                onSearch={searchUsers}
                loading={userSearchLoading}
                maxTagCount={3}
                style={{ width: '100%' }}
              />
            </div>
          ) : null}

          {scope === 'agent' ? (
            <div className='mb-4'>
              <div className='mb-1 text-sm text-gray-600'>
                {t('选择代理商')}
              </div>
              <Select
                multiple
                filter
                placeholder={t('选择代理商')}
                optionList={agentOptions}
                value={selectedAgentIds}
                onChange={setSelectedAgentIds}
                loading={optionsLoading}
                maxTagCount={3}
                style={{ width: '100%' }}
              />
            </div>
          ) : null}

          <div className='mb-4'>
            <div className='mb-2 font-medium'>{t('导出字段')}</div>
            {columnOptions}
          </div>

          <Button
            htmlType='submit'
            icon={<IconDownload />}
            theme='solid'
            type='primary'
            loading={exporting}
          >
            {t('导出结算单')}
          </Button>
        </Form>
      </CardPro>

      <div className='order-1 lg:order-2 lg:sticky lg:top-[76px]'>
        <SettlementSummaryPanel
          t={t}
          loading={summaryLoading}
          error={summaryError}
          summary={summaryData}
          scope={scope}
          canFetch={canFetchSummary}
          onRefresh={() => setSummaryRefreshKey((key) => key + 1)}
        />
      </div>
    </div>
  );
};

export default SettingsSettlementExport;
