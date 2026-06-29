/*
Copyright (C) 2025 QuantumNous
*/

import React from 'react';
import { Card, Typography } from '@douyinfe/semi-ui';
import ModelPerfHourlyChart from '../../components/ModelPerfHourlyChart';
import ModelPerfMetricsGrid from '../../components/ModelPerfMetricsGrid';

const { Text, Title } = Typography;

const ModelPerfPanel = ({ modelName, perfSummary, t }) => {
  if (!modelName) return null;

  const summary = perfSummary || null;

  if (!summary) {
    return (
      <Card className='!rounded-xl mb-6' bodyStyle={{ padding: 16 }}>
        <Text type='secondary'>{t('暂无该模型的性能数据')}</Text>
      </Card>
    );
  }

  const hourlySeries = summary.hourly_series || [];
  const hasHourlySeries = hourlySeries.some((p) => (p.request_count || 0) > 0);

  return (
    <Card
      className='!rounded-xl mb-6'
      title={
        <div>
          <Title heading={6}>{t('运行性能')}</Title>
          <Text type='tertiary' size='small'>
            {t('近24小时真实请求统计')}
          </Text>
        </div>
      }
      bodyStyle={{ paddingTop: 4 }}
    >
      {hasHourlySeries && (
        <div className='mb-4'>
          <ModelPerfHourlyChart series={hourlySeries} t={t} />
        </div>
      )}

      <div
        className='rounded-xl p-4'
        style={{
          backgroundColor: 'var(--semi-color-fill-0)',
          border: '1px solid var(--semi-color-border)',
        }}
      >
        <ModelPerfMetricsGrid perf={summary} t={t} />
      </div>
    </Card>
  );
};

export default ModelPerfPanel;
