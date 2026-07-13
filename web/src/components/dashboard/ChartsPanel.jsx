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
import { Button, Card } from '@douyinfe/semi-ui';
import { PieChart } from 'lucide-react';
import { VChart } from '@visactor/react-vchart';

const CHART_TABS = [
  { key: '1', label: '消耗分布' },
  { key: '2', label: '消耗趋势' },
  { key: '3', label: '调用次数分布' },
  { key: '4', label: '调用次数排行' },
];

const ChartsPanel = ({
  activeChartTab,
  setActiveChartTab,
  spec_line,
  spec_model_line,
  spec_pie,
  spec_rank_bar,
  CARD_PROPS,
  CHART_CONFIG,
  FLEX_CENTER_GAP2,
  hasApiInfoPanel,
  t,
}) => {
  return (
    <Card
      {...CARD_PROPS}
      className={`dashboard-glass-card dashboard-glass-card--chart !rounded-3xl ${hasApiInfoPanel ? 'lg:col-span-3' : ''}`}
      title={
        <div className='flex flex-col lg:flex-row lg:items-center lg:justify-between w-full gap-3'>
          <div className={FLEX_CENTER_GAP2}>
            <PieChart size={16} />
            {t('模型数据分析')}
          </div>
          <div className='dashboard-chart-switcher' role='tablist'>
            {CHART_TABS.map((tab) => (
              <Button
                key={tab.key}
                role='tab'
                aria-selected={activeChartTab === tab.key}
                size='small'
                theme={activeChartTab === tab.key ? 'solid' : 'borderless'}
                type={activeChartTab === tab.key ? 'primary' : 'tertiary'}
                className='dashboard-chart-switcher__button'
                onClick={() => setActiveChartTab(tab.key)}
              >
                {t(tab.label)}
              </Button>
            ))}
          </div>
        </div>
      }
      bodyStyle={{ padding: 0 }}
    >
      <div className='dashboard-chart-canvas h-96 p-2'>
        {activeChartTab === '1' && (
          <VChart spec={spec_line} option={CHART_CONFIG} />
        )}
        {activeChartTab === '2' && (
          <VChart spec={spec_model_line} option={CHART_CONFIG} />
        )}
        {activeChartTab === '3' && (
          <VChart spec={spec_pie} option={CHART_CONFIG} />
        )}
        {activeChartTab === '4' && (
          <VChart spec={spec_rank_bar} option={CHART_CONFIG} />
        )}
      </div>
    </Card>
  );
};

export default ChartsPanel;
