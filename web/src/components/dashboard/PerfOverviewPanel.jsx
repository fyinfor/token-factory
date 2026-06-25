/*
Copyright (C) 2025 QuantumNous
*/

import React, { useContext, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Card, Button, Spin, Typography } from '@douyinfe/semi-ui';
import { Activity, RefreshCw } from 'lucide-react';
import { Link } from 'react-router-dom';
import { StatusContext } from '../../context/Status';
import {
  fetchPerfMetricsSummary,
  formatPerfLatency,
  formatPerfThroughput,
  formatPerfSuccessRate,
  getSuccessRateColor,
} from '../../helpers/perfMetrics';
import { pickPerfLeaders } from '../../helpers/rankings';
import PerfMetricLabel from '../table/model-pricing/components/PerfMetricLabel';

const { Text, Title } = Typography;

const LeaderColumn = ({ title, rows, renderValue, valueColor }) => (
  <div className='min-w-0 flex-1'>
    <div
      className='text-xs font-medium mb-2 pb-2 border-b'
      style={{
        color: 'var(--semi-color-text-2)',
        borderColor: 'var(--semi-color-border)',
      }}
    >
      {title}
    </div>
    {rows.length === 0 ? (
      <Text type='tertiary' size='small'>
        —
      </Text>
    ) : (
      <div className='flex flex-col gap-2'>
        {rows.map((item, index) => (
          <div key={item.model_name} className='flex items-center justify-between gap-2'>
            <div className='flex items-center gap-2 min-w-0'>
              <span className='text-[10px] font-mono text-gray-400 w-4'>{index + 1}</span>
              <span className='text-xs truncate'>{item.model_name}</span>
            </div>
            <span
              className='text-xs font-mono font-semibold shrink-0'
              style={{ color: valueColor?.(item) || 'var(--semi-color-text-0)' }}
            >
              {renderValue(item)}
            </span>
          </div>
        ))}
      </div>
    )}
  </div>
);

const PerfOverviewPanel = ({ CARD_PROPS, t: propT }) => {
  const { t: i18nT } = useTranslation();
  const t = propT || i18nT;
  const [statusState] = useContext(StatusContext);
  const [loading, setLoading] = useState(true);
  const [models, setModels] = useState([]);
  const [error, setError] = useState('');

  const rankingsEnabled = useMemo(() => {
    try {
      const config = statusState?.status?.HeaderNavModules;
      if (!config) return true;
      const modules = JSON.parse(config);
      return modules.rankings !== false;
    } catch {
      return true;
    }
  }, [statusState?.status?.HeaderNavModules]);

  const load = async () => {
    setLoading(true);
    setError('');
    try {
      const map = await fetchPerfMetricsSummary(24);
      setModels(Object.values(map || {}));
    } catch (err) {
      setModels([]);
      setError(err?.message || String(err));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const leaders = useMemo(() => pickPerfLeaders(models, 5), [models]);

  if (!loading && models.length === 0 && !error) {
    return null;
  }

  return (
    <Card
      {...CARD_PROPS}
      className='shadow-sm !rounded-2xl'
      title={
        <div className='flex items-center justify-between gap-3 w-full'>
          <div className='flex items-center gap-2 min-w-0'>
            <Activity size={16} className='shrink-0' />
            <div className='min-w-0'>
              <Title heading={6} style={{ margin: 0 }}>
                {t('模型运行性能概览')}
              </Title>
              <Text type='tertiary' size='small'>
                {t('近24小时真实请求统计')}
              </Text>
            </div>
          </div>
          <div className='flex items-center gap-2 shrink-0'>
            <Button
              size='small'
              theme='borderless'
              icon={<RefreshCw size={14} />}
              loading={loading}
              onClick={load}
            />
            {rankingsEnabled ? (
              <Link to='/rankings'>
                <Button size='small' theme='borderless'>
                  {t('模型排行榜')}
                </Button>
              </Link>
            ) : null}
            <Link to='/pricing'>
              <Button size='small' theme='light'>
                {t('模型广场')}
              </Button>
            </Link>
          </div>
        </div>
      }
    >
      <Spin spinning={loading}>
        {error ? (
          <Text type='danger' size='small'>
            {error}
          </Text>
        ) : (
          <div className='flex flex-col lg:flex-row gap-6'>
            <LeaderColumn
              title={t('在线率 TOP')}
              rows={leaders.byUptime}
              renderValue={(item) => formatPerfSuccessRate(item.success_rate)}
              valueColor={(item) => getSuccessRateColor(item.success_rate)}
            />
            <LeaderColumn
              title={
                <PerfMetricLabel label='E2E' hint={t('E2E延迟说明')} className='text-xs' />
              }
              rows={leaders.byLatency}
              renderValue={(item) => formatPerfLatency(item.avg_latency_ms)}
            />
            <LeaderColumn
              title={
                <PerfMetricLabel label='TPS' hint={t('TPS说明')} className='text-xs' />
              }
              rows={leaders.byTps}
              renderValue={(item) => formatPerfThroughput(item.avg_tps)}
            />
          </div>
        )}
      </Spin>
    </Card>
  );
};

export default PerfOverviewPanel;
