import React, { useEffect, useMemo, useState } from 'react';
import { Button, Card, Empty, Spin, Table, Typography } from '@douyinfe/semi-ui';
import {
  IllustrationNoAccess,
  IllustrationNoAccessDark,
} from '@douyinfe/semi-illustrations';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import {
  Activity,
  BarChart3,
  Coins,
  Cpu,
  Gauge,
  RefreshCw,
  TrendingUp,
} from 'lucide-react';
import { API, isAdmin, isSupplier, showError } from '../../../helpers';

const { Title, Text } = Typography;
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
 * SupplierDashboardPage 供应商数据看板页。
 */
export default function SupplierDashboardPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState(null);
  const [timeRange] = useState(getDefaultRange());

  /**
   * loadDashboardData 加载供应商看板数据（供应商=自己模型，管理员=全部供应商模型）。
   */
  const loadDashboardData = async () => {
    setLoading(true);
    try {
      const res = await API.get(
        `/api/user/supplier-dashboard?start_timestamp=${timeRange.startTimestamp}&end_timestamp=${timeRange.endTimestamp}`,
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
  };

  useEffect(() => {
    loadDashboardData();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const usageColumns = useMemo(
    () => [
      { title: t('模型'), dataIndex: 'model_name' },
      { title: t('请求次数'), dataIndex: 'requests' },
      { title: t('Token 消耗'), dataIndex: 'tokens' },
      { title: t('额度消耗'), dataIndex: 'quota' },
    ],
    [t],
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
            value: data?.model_data_analysis?.provided_model_count ?? data?.model_data_analysis?.model_count ?? 0,
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
            value: data?.model_data_analysis?.active_model_count ?? (data?.model_usage_stats || []).length,
            icon: <Cpu size={14} />,
            iconColor: ICON_BG_COLORS.amber,
          },
          {
            label: t('时间粒度'),
            value: t('小时'),
            icon: <BarChart3 size={14} />,
            iconColor: ICON_BG_COLORS.orange,
          },
        ],
      },
    ],
    [data, t],
  );

  const canAccess = isSupplier() || isAdmin();
  if (!canAccess) {
    return (
      <div className='mt-[60px] px-2'>
        <div className='flex items-center justify-center' style={{ minHeight: 'calc(100vh - 360px)' }}>
          <Empty
            image={<IllustrationNoAccess style={{ width: 200, height: 200 }} />}
            darkModeImage={<IllustrationNoAccessDark style={{ width: 200, height: 200 }} />}
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
        <div className='flex items-center justify-between mb-3'>
          <div className='flex items-center gap-2'>
            <div
              className='w-9 h-9 rounded-full text-white flex items-center justify-center'
              style={{ backgroundColor: ICON_BG_COLORS.indigo }}
            >
              <Activity size={18} />
            </div>
            <div>
              <Title heading={4} style={{ marginBottom: 0 }}>
                {t('供应商模型数据看板')}
              </Title>
              <Text type='tertiary'>
                {isAdmin()
                  ? t('当前展示全部供应商提供模型的统计数据')
                  : t('当前展示您提供模型的统计数据')}
              </Text>
            </div>
          </div>
          <Button
            type='tertiary'
            icon={<RefreshCw size={16} />}
            className='bg-blue-500 hover:bg-blue-600 text-white !rounded-full'
            onClick={loadDashboardData}
            loading={loading}
          />
        </div>

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
                  <div key={`${item.key}-${idx}`} className='flex items-center gap-2'>
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
      </Spin>
    </div>
  );
}
