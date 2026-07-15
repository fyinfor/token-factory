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
import { Card } from '@douyinfe/semi-ui';
import { BarChart3, ChartPie, PieChart, TrendingUp } from 'lucide-react';
import { VChart } from '@visactor/react-vchart';

const ChartsPanel = ({
  spec_line,
  spec_model_line,
  spec_pie,
  spec_rank_bar,
  CARD_PROPS,
  CHART_CONFIG,
  t,
}) => {
  const charts = [
    {
      key: 'quota-distribution',
      title: t('消耗分布'),
      icon: <PieChart size={17} />,
      tone: 'blue',
      spec: spec_line,
    },
    {
      key: 'quota-trend',
      title: t('消耗趋势'),
      icon: <TrendingUp size={17} />,
      tone: 'green',
      spec: spec_model_line,
    },
    {
      key: 'request-distribution',
      title: t('调用次数分布'),
      icon: <ChartPie size={17} />,
      tone: 'cyan',
      spec: spec_pie,
    },
    {
      key: 'request-ranking',
      title: t('调用次数排行'),
      icon: <BarChart3 size={17} />,
      tone: 'amber',
      spec: spec_rank_bar,
    },
  ];

  return (
    <section className='dashboard-model-analysis'>
      <div className='dashboard-model-analysis__grid'>
        {charts.map((chart) => (
          <Card
            key={chart.key}
            {...CARD_PROPS}
            className='dashboard-glass-card dashboard-glass-card--chart dashboard-model-analysis__card'
            title={
              <div className='dashboard-model-analysis__card-title'>
                <span
                  className={
                    'dashboard-model-analysis__icon dashboard-model-analysis__icon--' +
                    chart.tone
                  }
                >
                  {chart.icon}
                </span>
                <span>{chart.title}</span>
              </div>
            }
            bodyStyle={{ padding: 0 }}
          >
            <div className='dashboard-chart-canvas dashboard-model-analysis__canvas'>
              <VChart spec={chart.spec} option={CHART_CONFIG} />
            </div>
          </Card>
        ))}
      </div>
    </section>
  );
};

export default ChartsPanel;
