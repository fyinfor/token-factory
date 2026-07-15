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

import React from 'react';
import { Card, Typography } from '@douyinfe/semi-ui';
import ModelPerfLineChart from '../../components/ModelPerfLineChart';
import ModelPerfMetricsGrid from '../../components/ModelPerfMetricsGrid';

const { Text, Title } = Typography;

const ModelPerfPanel = ({ modelName, perfSummary, t, flat = false }) => {
  if (!modelName) return null;

  const summary = perfSummary || null;
  if (!summary) return null;

  const hourlySeries = summary.hourly_series || [];
  const hasHourlySeries = hourlySeries.some(
    (point) => (point.request_count || 0) > 0,
  );
  const content = (
    <>
      <div className='mb-3'>
        <Title heading={6}>{t('运行性能')}</Title>
        <Text type='tertiary' size='small'>
          {t('近24小时真实请求统计')}
        </Text>
      </div>
      {hasHourlySeries ? (
        <div className='mb-3'>
          <ModelPerfLineChart series={hourlySeries} t={t} />
        </div>
      ) : null}
      <div className={flat ? 'pt-1' : 'rounded-xl border p-4'}>
        <ModelPerfMetricsGrid perf={summary} t={t} />
      </div>
    </>
  );

  if (flat) {
    return (
      <section className='mb-6 border-b border-semi-color-border pb-6'>
        {content}
      </section>
    );
  }

  return (
    <Card className='!rounded-xl mb-6' bodyStyle={{ paddingTop: 12 }}>
      {content}
    </Card>
  );
};

export default ModelPerfPanel;
