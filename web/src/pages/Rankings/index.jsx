/*
Copyright (C) 2025 QuantumNous
*/

import React, { useEffect, useMemo, useState, useContext } from 'react';
import { useTranslation } from 'react-i18next';
import { Navigate } from 'react-router-dom';
import { Button, Spin, TabPane, Tabs, Typography } from '@douyinfe/semi-ui';
import { BarChart3, RefreshCw } from 'lucide-react';
import { StatusContext } from '../../context/Status';
import { useRankingsData } from '../../hooks/rankings/useRankingsData';
import ModelLeaderboard from '../../components/rankings/ModelLeaderboard';
import VendorShareSection from '../../components/rankings/VendorShareSection';
import PulseSection from '../../components/rankings/PulseSection';

const { Title, Text } = Typography;

const PERIOD_TABS = [
  { key: 'today', labelKey: '今日' },
  { key: 'week', labelKey: '本周' },
  { key: 'month', labelKey: '本月' },
  { key: 'year', labelKey: '本年' },
];

const RankingsPage = () => {
  const { t } = useTranslation();
  const [statusState] = useContext(StatusContext);
  const { period, changePeriod, snapshot, loading, error, reload } = useRankingsData('week');
  const [activeTab, setActiveTab] = useState('week');

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

  useEffect(() => {
    setActiveTab(period);
  }, [period]);

  if (!rankingsEnabled) {
    return <Navigate to='/' replace />;
  }

  const models = useMemo(() => snapshot?.models || [], [snapshot]);
  const vendors = useMemo(() => snapshot?.vendors || [], [snapshot]);

  return (
    <div className='rankings-page'>
      <div className='rankings-page__inner w-full max-w-6xl mx-auto px-4 py-6 md:py-8'>
      <div className='flex flex-col md:flex-row md:items-end md:justify-between gap-4 mb-6'>
        <div>
          <div className='flex items-center gap-2 mb-2'>
            <BarChart3 size={22} />
            <Title heading={3} style={{ margin: 0 }}>
              {t('模型排行榜')}
            </Title>
          </div>
          <Text type='tertiary'>{t('基于真实调用 Token 用量的模型与供应商排行')}</Text>
        </div>
        <Button
          icon={<RefreshCw size={14} />}
          onClick={reload}
          loading={loading}
          theme='light'
        >
          {t('刷新')}
        </Button>
      </div>

      <Tabs
        type='button'
        activeKey={activeTab}
        onChange={(key) => {
          setActiveTab(key);
          changePeriod(key);
        }}
        className='mb-6'
      >
        {PERIOD_TABS.map((tab) => (
          <TabPane tab={t(tab.labelKey)} itemKey={tab.key} key={tab.key} />
        ))}
      </Tabs>

      {error ? (
        <div
          className='rounded-xl px-4 py-3 mb-6 text-sm'
          style={{
            color: 'var(--semi-color-danger)',
            backgroundColor: 'rgba(var(--semi-red-0), 0.35)',
          }}
        >
          {error}
        </div>
      ) : null}

      <Spin spinning={loading && !snapshot}>
        <div className='grid lg:grid-cols-2 gap-6 mb-6'>
          <ModelLeaderboard models={models} t={t} loading={loading} />
          <VendorShareSection vendors={vendors} t={t} loading={loading} />
        </div>
        <PulseSection
          topMovers={snapshot?.top_movers || []}
          topDroppers={snapshot?.top_droppers || []}
          t={t}
          loading={loading}
        />
      </Spin>
      </div>
    </div>
  );
};

export default RankingsPage;
