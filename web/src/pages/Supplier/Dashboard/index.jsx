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
  useContext,
  useEffect,
  useMemo,
  useState,
} from 'react';
import {
  Button,
  Card,
  DatePicker,
  Empty,
  Modal,
  Space,
  Spin,
  Table,
  Typography,
} from '@douyinfe/semi-ui';
import {
  IllustrationNoAccess,
  IllustrationNoAccessDark,
} from '@douyinfe/semi-illustrations';
import { useTranslation } from 'react-i18next';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { IconDownload } from '@douyinfe/semi-icons';
import {
  Activity,
  BarChart3,
  Coins,
  Cpu,
  Gauge,
  RefreshCw,
  TrendingUp,
} from 'lucide-react';
import {
  API,
  isAdmin,
  isSupplier,
  showError,
  showSuccess,
  renderQuota,
  renderQuotaSum,
  renderQuotaWithPrompt,
} from '../../../helpers';
import dayjs from 'dayjs';
import { DATE_RANGE_PRESETS } from '../../../constants/console.constants';
import { normalizeLanguage } from '../../../i18n/language';
import { StatusContext } from '../../../context/Status';

const { Title, Text } = Typography;

/** 允许的最大统计区间（秒），防止单次查询过大。 */
const SUPPLIER_DASHBOARD_MAX_RANGE_SEC = 366 * 24 * 3600;

/**
 * 从模型统计（含 users 明细）收集额度列表，花费展示与详情弹窗按用户行级舍入后合计一致。
 */
const collectDisplayQuotasFromModelStats = (modelStats) => {
  const quotas = [];
  for (const row of modelStats || []) {
    if (Array.isArray(row.users) && row.users.length > 0) {
      for (const user of row.users) {
        quotas.push(user.quota ?? 0);
      }
    } else {
      quotas.push(row.quota ?? 0);
    }
  }
  return quotas;
};

/** 模型行花费：有 users 时按用户行舍入合计，否则按模型总额展示。 */
const renderModelUsageRowQuota = (row) => {
  if (Array.isArray(row?.users) && row.users.length > 0) {
    return renderQuotaSum(row.users.map((user) => user.quota ?? 0));
  }
  return renderQuota(row?.quota ?? 0);
};

const summarizeSupplierUserRows = (users) =>
  (users || []).reduce(
    (acc, user) => ({
      total_requests: acc.total_requests + (user.requests || 0),
      total_tokens: acc.total_tokens + (user.tokens ?? user.token_used ?? 0),
      total_quota: acc.total_quota + (user.quota || 0),
    }),
    { total_requests: 0, total_tokens: 0, total_quota: 0 },
  );
const ICON_BG_COLORS = {
  blue: '#3b82f6',
  emerald: '#10b981',
  violet: '#8b5cf6',
  amber: '#f59e0b',
  indigo: '#6366f1',
  sky: '#0ea5e9',
  lime: '#84cc16',
  fuchsia: '#d946ef',
  orange: '#f97316',
};

/**
 * getDefaultRange 返回最近24小时的看板时间范围。
 */
const getDefaultRange = () => {
  const endTimestamp = Math.floor(Date.now() / 1000);
  return {
    startTimestamp: endTimestamp - 24 * 3600,
    endTimestamp,
  };
};

/**
 * buildSupplierDashboardPresets 构造日期范围快捷选项（含默认近 24 小时与控制台通用预设）。
 * @param {function} t i18n t
 */
const buildSupplierDashboardPresets = (t) => {
  const last24h = {
    text: t('supplier_dashboard_preset_last_24h'),
    start: () => dayjs().subtract(24, 'hour').toDate(),
    end: () => dayjs().toDate(),
  };
  return [last24h, ...DATE_RANGE_PRESETS].map((preset) => ({
    text: preset.text,
    start: preset.start(),
    end: preset.end(),
  }));
};

/**
 * SupplierDashboardPage 供应商数据看板页。
 */
export default function SupplierDashboardPage() {
  const { t, i18n } = useTranslation();
  const [statusState] = useContext(StatusContext);
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const selectedSupplierId = searchParams.get('supplier_id');
  const selectedSupplierName = (searchParams.get('supplier_name') || '').trim();
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState(null);
  const [timeRange, setTimeRange] = useState(() => getDefaultRange());
  const [detailVisible, setDetailVisible] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailModelName, setDetailModelName] = useState('');
  const [detailUsers, setDetailUsers] = useState([]);
  const [detailSummary, setDetailSummary] = useState(null);
  const [exporting, setExporting] = useState(false);
  const status = statusState?.status || {};
  const quotaDisplayKey = [
    status.quota_display_type ||
      localStorage.getItem('quota_display_type') ||
      'USD',
    status.quota_per_unit || localStorage.getItem('quota_per_unit') || '',
    status.usd_exchange_rate || '',
    status.custom_currency_symbol || '',
    status.custom_currency_exchange_rate || '',
  ].join('|');

  /** DatePicker 受控展示用的起止时间（由 timeRange 派生）。 */
  const rangePickerValue = useMemo(
    () => [
      new Date(timeRange.startTimestamp * 1000),
      new Date(timeRange.endTimestamp * 1000),
    ],
    [timeRange.startTimestamp, timeRange.endTimestamp],
  );

  /**
   * loadDashboardData 加载供应商看板数据（供应商=自己模型，管理员=全部供应商模型）。
   */
  const loadDashboardData = useCallback(async () => {
    setLoading(true);
    try {
      const supplierQuery = selectedSupplierId
        ? `&supplier_id=${encodeURIComponent(selectedSupplierId)}`
        : '';
      const res = await API.get(
        `/api/user/supplier-dashboard?start_timestamp=${timeRange.startTimestamp}&end_timestamp=${timeRange.endTimestamp}${supplierQuery}`,
      );
      if (res?.data?.success) {
        setData(res.data.data || null);
      } else {
        showError(res?.data?.message || t('获取供应商数据看板失败'));
      }
    } catch (error) {
      showError(error?.response?.data?.message || t('获取供应商数据看板失败'));
    } finally {
      setLoading(false);
    }
  }, [selectedSupplierId, timeRange.startTimestamp, timeRange.endTimestamp, t]);

  useEffect(() => {
    loadDashboardData();
  }, [loadDashboardData]);

  /**
   * handleStatsRangeChange 用户在选择器中变更统计时间范围后更新状态并触发重新拉数。
   * @param {Date[]|null|undefined} dates Semi DatePicker 返回的起止日期
   */
  const handleStatsRangeChange = (dates) => {
    if (!dates || !dates[0] || !dates[1]) {
      setTimeRange(getDefaultRange());
      return;
    }
    const startTimestamp = Math.floor(dates[0].getTime() / 1000);
    const endTimestamp = Math.floor(dates[1].getTime() / 1000);
    if (startTimestamp >= endTimestamp) {
      showError(t('supplier_dashboard_invalid_range'));
      return;
    }
    if (endTimestamp - startTimestamp > SUPPLIER_DASHBOARD_MAX_RANGE_SEC) {
      showError(t('supplier_dashboard_range_too_long'));
      return;
    }
    setTimeRange({ startTimestamp, endTimestamp });
  };

  /** resetStatsRangeToLast24h 将统计区间重置为最近 24 小时（与默认一致）。 */
  const resetStatsRangeToLast24h = () => {
    setTimeRange(getDefaultRange());
  };

  /** handleExportUsageDetail 按当前统计时间范围导出使用详情账单（模型汇总 + 逐条明细）。 */
  const handleExportUsageDetail = async () => {
    setExporting(true);
    try {
      const lang = normalizeLanguage(i18n.language || '');
      const params = new URLSearchParams({
        start_timestamp: String(timeRange.startTimestamp),
        end_timestamp: String(timeRange.endTimestamp),
        lang: lang || '',
      });
      if (selectedSupplierId) {
        params.set('supplier_id', selectedSupplierId);
      }
      if (selectedSupplierName) {
        params.set('supplier_name', selectedSupplierName);
      }
      const res = await API.get(
        `/api/user/supplier-dashboard/export?${params.toString()}`,
        { responseType: 'blob' },
      );
      const blob = new Blob([res.data], { type: 'text/csv;charset=utf-8' });
      const objectUrl = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = objectUrl;
      a.download = `supplier-usage-detail-${new Date()
        .toISOString()
        .replace(/[-:T.Z]/g, '')
        .slice(0, 14)}.csv`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(objectUrl);
      showSuccess(t('supplier_dashboard_export_success'));
    } catch (error) {
      let message = error?.message || String(error);
      if (error?.response?.status === 429) {
        message = t('supplier_dashboard_export_rate_limited');
      } else if (error?.response?.data) {
        try {
          message = await error.response.data.text();
        } catch (_) {
          // ignore
        }
      }
      showError(`${t('supplier_dashboard_export_failed')}: ${message}`);
    } finally {
      setExporting(false);
    }
  };

  /**
   * openModelUsageDetail 打开指定模型的按用户消耗明细弹窗。
   * @param {string} modelName 模型名
   */
  const openModelUsageDetail = useCallback(
    async (modelName) => {
      if (!modelName) {
        return;
      }
      setDetailModelName(modelName);
      setDetailVisible(true);
      setDetailUsers([]);
      setDetailSummary(null);

      const cachedRow = (data?.model_usage_stats || []).find(
        (row) => row.model_name === modelName,
      );
      if (cachedRow?.users?.length) {
        setDetailUsers(cachedRow.users);
        setDetailSummary(summarizeSupplierUserRows(cachedRow.users));
        return;
      }

      setDetailLoading(true);
      try {
        const supplierQuery = selectedSupplierId
          ? `&supplier_id=${encodeURIComponent(selectedSupplierId)}`
          : '';
        const res = await API.get(
          `/api/user/supplier-dashboard/model-users?model_name=${encodeURIComponent(modelName)}&start_timestamp=${timeRange.startTimestamp}&end_timestamp=${timeRange.endTimestamp}${supplierQuery}`,
        );
        if (res?.data?.success) {
          const payload = res.data.data || {};
          setDetailUsers(payload.users || []);
          setDetailSummary(payload.summary || null);
        } else {
          showError(res?.data?.message || t('获取供应商模型用户明细失败'));
          setDetailVisible(false);
        }
      } catch (error) {
        showError(
          error?.response?.data?.message || t('获取供应商模型用户明细失败'),
        );
        setDetailVisible(false);
      } finally {
        setDetailLoading(false);
      }
    },
    [
      data?.model_usage_stats,
      selectedSupplierId,
      timeRange.startTimestamp,
      timeRange.endTimestamp,
      t,
    ],
  );

  const dashboardDisplayQuotas = useMemo(
    () => collectDisplayQuotasFromModelStats(data?.model_usage_stats),
    [data?.model_usage_stats],
  );

  const dashboardSpendDisplay = useMemo(
    () => renderQuotaSum(dashboardDisplayQuotas),
    [dashboardDisplayQuotas, quotaDisplayKey],
  );

  const usageColumns = useMemo(
    () => [
      { title: t('模型'), dataIndex: 'model_name' },
      { title: t('请求次数'), dataIndex: 'requests' },
      { title: t('Token 消耗'), dataIndex: 'tokens' },
      { title: t('额度消耗'), dataIndex: 'quota' },
      {
        title: t('花费'),
        dataIndex: 'quota',
        render: (_, row) => renderModelUsageRowQuota(row),
      },
      {
        title: t('操作'),
        dataIndex: 'model_name',
        width: 100,
        render: (modelName) => (
          <Button
            theme='borderless'
            type='primary'
            size='small'
            onClick={() => openModelUsageDetail(modelName)}
          >
            {t('查看详情')}
          </Button>
        ),
      },
    ],
    [t, openModelUsageDetail, quotaDisplayKey],
  );

  const modelUserDetailColumns = useMemo(
    () => [
      {
        title: t('用户'),
        dataIndex: 'username',
        render: (name, row) => name || `#${row.user_id}`,
      },
      { title: t('请求次数'), dataIndex: 'requests' },
      { title: t('Token 消耗'), dataIndex: 'tokens' },
      {
        title: t('花费'),
        dataIndex: 'quota',
        render: (q) => renderQuota(q ?? 0),
      },
    ],
    [t, quotaDisplayKey],
  );

  /**
   * summaryCards 看板指标卡配置（含图标与颜色）。
   */
  const summaryCards = useMemo(
    () => [
      {
        key: 'model_usage',
        title: t('模型使用统计'),
        icon: <BarChart3 size={18} />,
        iconColor: ICON_BG_COLORS.blue,
        lines: [
          {
            label: t('供应商提供模型数'),
            value:
              data?.model_data_analysis?.provided_model_count ??
              data?.model_data_analysis?.model_count ??
              0,
            icon: <Cpu size={14} />,
            iconColor: ICON_BG_COLORS.indigo,
          },
          {
            label: t('总请求次数'),
            value: data?.resource_consumption?.total_requests || 0,
            icon: <TrendingUp size={14} />,
            iconColor: ICON_BG_COLORS.sky,
          },
        ],
      },
      {
        key: 'resource',
        title: t('资源消耗'),
        icon: <Coins size={18} />,
        iconColor: ICON_BG_COLORS.emerald,
        lines: [
          {
            label: t('总 Token'),
            value: data?.resource_consumption?.total_tokens || 0,
            icon: <BarChart3 size={14} />,
            iconColor: ICON_BG_COLORS.emerald,
          },
          {
            label: t('总额度'),
            value: data?.resource_consumption?.total_quota || 0,
            icon: <Coins size={14} />,
            iconColor: ICON_BG_COLORS.lime,
          },
          {
            label: t('花费'),
            value: dashboardSpendDisplay,
            icon: <Coins size={14} />,
            iconColor: ICON_BG_COLORS.orange,
          },
        ],
      },
      {
        key: 'performance',
        title: t('性能指标'),
        icon: <Gauge size={18} />,
        iconColor: ICON_BG_COLORS.violet,
        lines: [
          {
            label: t('最近1分钟 RPM'),
            value: data?.performance_metrics?.rpm || 0,
            icon: <Activity size={14} />,
            iconColor: ICON_BG_COLORS.violet,
          },
          {
            label: t('最近1分钟 TPM'),
            value: data?.performance_metrics?.tpm || 0,
            icon: <Gauge size={14} />,
            iconColor: ICON_BG_COLORS.fuchsia,
          },
        ],
      },
      {
        key: 'analysis',
        title: t('模型数据分析'),
        icon: <Cpu size={18} />,
        iconColor: ICON_BG_COLORS.amber,
        lines: [
          {
            label: t('有数据模型数'),
            value:
              data?.model_data_analysis?.active_model_count ??
              (data?.model_usage_stats || []).length,
            icon: <Cpu size={14} />,
            iconColor: ICON_BG_COLORS.amber,
          },
          {
            label: t('明细聚合'),
            value: t('按小时桶'),
            icon: <BarChart3 size={14} />,
            iconColor: ICON_BG_COLORS.orange,
          },
        ],
      },
    ],
    [data, dashboardSpendDisplay, t],
  );

  const canAccess = isSupplier() || isAdmin();
  /** 管理员查看「全部供应商汇总」时不展示单账户；指定 supplier_id 或与供应商本人一致时再展示。 */
  const showAccountSection =
    !isAdmin() ||
    Boolean(selectedSupplierId && String(selectedSupplierId).trim());

  if (!canAccess) {
    return (
      <div className='mt-[60px] px-2'>
        <div
          className='flex items-center justify-center'
          style={{ minHeight: 'calc(100vh - 360px)' }}
        >
          <Empty
            image={<IllustrationNoAccess style={{ width: 200, height: 200 }} />}
            darkModeImage={
              <IllustrationNoAccessDark style={{ width: 200, height: 200 }} />
            }
            layout='horizontal'
            title={t('需要供应商权限')}
            description={t('您需要先成为供应商才能访问此页面。')}
          >
            <Button
              theme='solid'
              type='primary'
              size='large'
              className='!rounded-md mt-4'
              onClick={() => navigate('/console/supplier/apply')}
            >
              {t('前往申请')}
            </Button>
          </Empty>
        </div>
      </div>
    );
  }

  return (
    <div className='mt-[60px] w-full max-w-[1600px] mx-auto px-3 sm:px-4 lg:px-6 pb-6'>
      <Spin spinning={loading} size='large'>
        <div className='flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between mb-3'>
          <div className='flex items-center gap-2 min-w-0'>
            <div
              className='w-9 h-9 rounded-full text-white flex items-center justify-center shrink-0'
              style={{ backgroundColor: ICON_BG_COLORS.indigo }}
            >
              <Activity size={18} />
            </div>
            <div className='min-w-0'>
              <Title heading={4} style={{ marginBottom: 0 }}>
                {selectedSupplierName
                  ? t('{{name}} 模型数据看板', { name: selectedSupplierName })
                  : t('供应商模型数据看板')}
              </Title>
              <Text type='tertiary'>
                {isAdmin()
                  ? selectedSupplierId
                    ? selectedSupplierName
                      ? t('当前展示指定供应商「{{name}}」的数据看板', {
                          name: selectedSupplierName,
                        })
                      : t('当前展示指定供应商的数据看板')
                    : t('当前展示全部供应商提供模型的统计数据')
                  : t('当前展示您提供模型的统计数据')}
              </Text>
            </div>
          </div>
          <Space wrap className='w-full lg:w-auto lg:justify-end' align='end'>
            <div className='flex flex-col gap-1 min-w-[280px] flex-1 lg:flex-initial'>
              <Text type='tertiary' size='small'>
                {t('supplier_dashboard_time_range')}
              </Text>
              <DatePicker
                type='dateTimeRange'
                value={rangePickerValue}
                onChange={(dates) => handleStatsRangeChange(dates)}
                presets={buildSupplierDashboardPresets(t)}
                placeholder={[t('开始时间'), t('结束时间')]}
                showClear
                size='small'
                className='w-full'
              />
            </div>
            <Button type='tertiary' onClick={resetStatsRangeToLast24h}>
              {t('supplier_dashboard_reset_24h')}
            </Button>
            <Button
              type='tertiary'
              icon={<IconDownload />}
              loading={exporting}
              onClick={handleExportUsageDetail}
            >
              {t('supplier_dashboard_export_usage')}
            </Button>
            <Button
              type='tertiary'
              icon={<RefreshCw size={16} />}
              className='bg-blue-500 hover:bg-blue-600 text-white !rounded-full'
              onClick={loadDashboardData}
              loading={loading}
            >
              {t('刷新')}
            </Button>
          </Space>
        </div>

        {showAccountSection ? (
          <Card title={t('supplier_dashboard_account_title')} className='mt-3'>
            <div className='grid grid-cols-1 md:grid-cols-2 gap-4'>
              <div>
                <Text type='secondary'>{t('剩余额度')}</Text>
                <div className='text-lg font-semibold'>
                  {data?.account?.quota ?? 0}
                </div>
                <Text type='tertiary' size='small'>
                  {renderQuotaWithPrompt(data?.account?.quota ?? 0)}
                </Text>
              </div>
              <div>
                <Text type='secondary'>{t('历史消耗')}</Text>
                <div className='text-lg font-semibold'>
                  {data?.account?.used_quota ?? 0}
                </div>
                <Text type='tertiary' size='small'>
                  {renderQuotaWithPrompt(data?.account?.used_quota ?? 0)}
                </Text>
              </div>
            </div>
          </Card>
        ) : (
          isAdmin() &&
          !selectedSupplierId && (
            <Text type='tertiary' className='block mt-2'>
              {t('supplier_dashboard_admin_account_hint')}
            </Text>
          )
        )}

        <div className='grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-3 mt-3'>
          {summaryCards.map((item) => (
            <Card
              key={item.key}
              title={
                <div className='flex items-center gap-2'>
                  <span
                    className='w-7 h-7 rounded-full text-white flex items-center justify-center'
                    style={{ backgroundColor: item.iconColor }}
                  >
                    {item.icon}
                  </span>
                  <span>{item.title}</span>
                </div>
              }
            >
              <div className='space-y-2'>
                {item.lines.map((line, idx) => (
                  <div
                    key={`${item.key}-${idx}`}
                    className='flex items-center gap-2'
                  >
                    <span
                      className='w-5 h-5 rounded-full text-white flex items-center justify-center'
                      style={{ backgroundColor: line.iconColor }}
                    >
                      {line.icon}
                    </span>
                    <Text>
                      {line.label}: {line.value}
                    </Text>
                  </div>
                ))}
              </div>
            </Card>
          ))}
        </div>

        <Card title={t('模型使用统计明细')} className='mt-3'>
          <Table
            rowKey='model_name'
            columns={usageColumns}
            dataSource={data?.model_usage_stats || []}
            pagination={{ pageSize: 15 }}
            empty={t('暂无统计数据')}
          />
        </Card>

        <Modal
          title={`${t('模型')}：${detailModelName}`}
          visible={detailVisible}
          onCancel={() => setDetailVisible(false)}
          footer={null}
          width={720}
          bodyStyle={{ paddingTop: 8 }}
        >
          <Spin spinning={detailLoading}>
            <Text type='tertiary' className='block mb-3'>
              {t('供应商模型用户消耗明细说明')}
            </Text>
            {detailSummary ? (
              <div className='flex flex-wrap gap-4 mb-3'>
                <Text>
                  {t('请求次数')}: {detailSummary.total_requests ?? 0}
                </Text>
                <Text>
                  {t('Token 消耗')}: {detailSummary.total_tokens ?? 0}
                </Text>
                <Text>
                  {t('花费')}:{' '}
                  {renderQuotaSum(
                    (detailUsers || []).map((row) => row.quota ?? 0),
                  )}
                </Text>
              </div>
            ) : null}
            <Table
              rowKey={(row) => `${row.user_id}-${row.username}`}
              columns={modelUserDetailColumns}
              dataSource={detailUsers}
              pagination={{ pageSize: 10 }}
              empty={t('暂无统计数据')}
            />
          </Spin>
        </Modal>
      </Spin>
    </div>
  );
}
