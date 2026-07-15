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
import { useTranslation } from 'react-i18next';
import { Navigate } from 'react-router-dom';
import { VChart } from '@visactor/react-vchart';
import { initVChartSemiTheme } from '@visactor/vchart-semi-theme';
import {
  Button,
  Card,
  Empty,
  Form,
  Modal,
  Spin,
  Typography,
} from '@douyinfe/semi-ui';
import {
  Activity,
  BarChart3,
  Gauge,
  RefreshCw,
  Search,
  Timer,
  Zap,
} from 'lucide-react';
import { StatusContext } from '../../context/Status';
import { CHART_CONFIG } from '../../constants/dashboard.constants';
import {
  fetchPerfMetricsSummary,
  formatPerfLatency,
  formatPerfSuccessRate,
  formatPerfThroughput,
  getSuccessRateColor,
} from '../../helpers/perfMetrics';
import '../../components/dashboard/dashboard-glass.css';
import './performance.css';

const { Title, Text } = Typography;
const QUICK_RANGES = [
  { value: '24h', label: '近24小时', seconds: 24 * 3600 },
  { value: '7d', label: '近7天', seconds: 7 * 24 * 3600 },
  { value: '30d', label: '近30天', seconds: 30 * 24 * 3600 },
];
const GRANULARITY_OPTIONS = [
  { value: 'hour', label: '小时' },
  { value: 'day', label: '天' },
  { value: 'week', label: '周' },
];
const MODEL_CHARTS = [
  {
    key: 'requests',
    label: '请求量',
    field: 'request_count',
    color: '#3b82f6',
  },
  { key: 'success', label: '成功率', field: 'success_rate', color: '#10b981' },
  { key: 'latency', label: '延迟', field: 'avg_latency_ms', color: '#f59e0b' },
  { key: 'throughput', label: '吞吐', field: 'avg_tps', color: '#06b6d4' },
];
const numberFormatter = new Intl.NumberFormat();
const compactFormatter = new Intl.NumberFormat(undefined, {
  notation: 'compact',
  maximumFractionDigits: 1,
});

const hasMetric = (value) =>
  Number.isFinite(Number(value)) && Number(value) > 0;

function weightedAverage(models, field) {
  let totalWeight = 0;
  let total = 0;
  models.forEach((model) => {
    const value = Number(model[field]);
    const weight = Math.max(1, Number(model.request_count || 0));
    if (Number.isFinite(value) && value >= 0) {
      total += value * weight;
      totalWeight += weight;
    }
  });
  return totalWeight ? total / totalWeight : 0;
}

function getBucketStart(ts, granularity) {
  const date = new Date(ts * 1000);
  if (granularity === 'week') {
    const day = date.getDay() || 7;
    date.setDate(date.getDate() - day + 1);
    date.setHours(0, 0, 0, 0);
  } else if (granularity === 'day') {
    date.setHours(0, 0, 0, 0);
  } else {
    date.setMinutes(0, 0, 0);
  }
  return Math.floor(date.getTime() / 1000);
}

function formatTrendTime(ts, granularity) {
  const date = new Date(ts * 1000);
  if (granularity === 'week') {
    return date.toLocaleDateString(undefined, {
      month: '2-digit',
      day: '2-digit',
    });
  }
  if (granularity === 'day') {
    return date.toLocaleDateString(undefined, {
      month: '2-digit',
      day: '2-digit',
    });
  }
  return date.toLocaleString(undefined, {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    hour12: false,
  });
}

function buildTrend(models, granularity) {
  const buckets = new Map();
  models.forEach((model) => {
    (model.hourly_series || []).forEach((point) => {
      const requests = Number(point.request_count || 0);
      if (requests <= 0) return;
      const ts = getBucketStart(point.ts, granularity);
      const current = buckets.get(ts) || {
        ts,
        requests: 0,
        successWeighted: 0,
      };
      current.requests += requests;
      current.successWeighted += Number(point.success_rate || 0) * requests;
      buckets.set(ts, current);
    });
  });
  return [...buckets.values()]
    .sort((a, b) => a.ts - b.ts)
    .map((point) => ({
      ...point,
      time: formatTrendTime(point.ts, granularity),
      successRate: Number((point.successWeighted / point.requests).toFixed(2)),
    }));
}

function formatChartValue(chartKey, value) {
  if (chartKey === 'success') return formatPerfSuccessRate(value);
  if (chartKey === 'latency') return formatPerfLatency(value);
  if (chartKey === 'throughput') return formatPerfThroughput(value);
  return numberFormatter.format(Math.round(value || 0));
}

const MetricCard = ({ icon, label, value, detail, tone }) => (
  <Card
    className='performance-metric-card dashboard-glass-card'
    bodyStyle={{ padding: 18 }}
  >
    <div className='performance-metric-card__head'>
      <span
        className={`performance-metric-card__icon performance-metric-card__icon--${tone}`}
      >
        {icon}
      </span>
      <Text type='tertiary'>{label}</Text>
    </div>
    <div className='performance-metric-card__value'>{value}</div>
    <Text type='tertiary' size='small'>
      {detail}
    </Text>
  </Card>
);

const RankingBars = ({
  title,
  caption,
  items,
  valueKey,
  formatValue,
  color,
  reverse = false,
}) => {
  const values = items.map((item) => Number(item[valueKey] || 0));
  const max = Math.max(...values, 1);
  const min = Math.min(...values, 0);
  return (
    <Card
      className='performance-ranking-card dashboard-glass-card'
      title={
        <div>
          <div className='performance-card-title'>{title}</div>
          <Text type='tertiary' size='small'>
            {caption}
          </Text>
        </div>
      }
    >
      <div className='performance-ranking-list'>
        {items.map((item, index) => {
          const value = Number(item[valueKey] || 0);
          const width = reverse
            ? 28 + ((max - value) / Math.max(max - min, 1)) * 72
            : 28 + (value / max) * 72;
          return (
            <div className='performance-ranking-row' key={item.model_name}>
              <div className='performance-ranking-row__label'>
                <span>{index + 1}</span>
                <strong title={item.model_name}>{item.model_name}</strong>
                <em>{formatValue(value)}</em>
              </div>
              <div className='performance-ranking-row__track'>
                <span
                  style={{
                    width: `${Math.min(100, width)}%`,
                    backgroundColor:
                      typeof color === 'function' ? color(item) : color,
                  }}
                />
              </div>
            </div>
          );
        })}
      </div>
    </Card>
  );
};

const Performance = () => {
  const { t } = useTranslation();
  const [statusState] = useContext(StatusContext);
  const [range, setRange] = useState(() => {
    const end = new Date();
    return [new Date(end.getTime() - 7 * 24 * 3600 * 1000), end];
  });
  const [appliedRange, setAppliedRange] = useState(range);
  const [granularity, setGranularity] = useState(
    () => localStorage.getItem('data_export_default_time') || 'day',
  );
  const [activeQuickRange, setActiveQuickRange] = useState('7d');
  const [searchModalVisible, setSearchModalVisible] = useState(false);
  const [models, setModels] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [activeModelChart, setActiveModelChart] = useState('requests');

  useEffect(() => {
    initVChartSemiTheme({ isWatchingThemeSwitch: true });
  }, []);

  const load = useCallback(
    async (selectedRange = appliedRange) => {
      const [start, end] = selectedRange || [];
      if (!start || !end) return;
      setLoading(true);
      setError('');
      try {
        const map = await fetchPerfMetricsSummary({
          startTimestamp: Math.floor(new Date(start).getTime() / 1000),
          endTimestamp: Math.floor(new Date(end).getTime() / 1000),
        });
        setModels(Object.values(map || {}));
        setAppliedRange(selectedRange);
      } catch (err) {
        setModels([]);
        setError(err?.message || t('获取性能数据失败'));
      } finally {
        setLoading(false);
      }
    },
    [appliedRange, t],
  );

  useEffect(() => {
    load(appliedRange);
  }, []);

  const applyQuickRange = useCallback(
    (preset) => {
      const end = new Date();
      const nextRange = [new Date(end.getTime() - preset.seconds * 1000), end];
      setRange(nextRange);
      setActiveQuickRange(preset.value);
      load(nextRange);
    },
    [load],
  );

  const applyRange = useCallback(() => {
    setActiveQuickRange('');
    load(range);
  }, [load, range]);

  const handleGranularityChange = useCallback((value) => {
    setGranularity(value);
    localStorage.setItem('data_export_default_time', value);
  }, []);

  const enabled = statusState?.status?.perf_overview_enabled ?? true;
  const totalRequests = useMemo(
    () =>
      models.reduce((sum, model) => sum + Number(model.request_count || 0), 0),
    [models],
  );
  const avgSuccess = useMemo(
    () => weightedAverage(models, 'success_rate'),
    [models],
  );
  const avgLatency = useMemo(
    () => weightedAverage(models, 'avg_latency_ms'),
    [models],
  );
  const avgTps = useMemo(() => weightedAverage(models, 'avg_tps'), [models]);
  const trend = useMemo(
    () => buildTrend(models, granularity),
    [granularity, models],
  );

  const requestTrendSpec = useMemo(
    () => ({
      type: 'bar',
      animation: false,
      data: { values: trend },
      xField: 'time',
      yField: 'requests',
      background: 'transparent',
      padding: { top: 20, right: 12, bottom: 8, left: 8 },
      bar: {
        style: {
          fill: '#3b82f6',
          cornerRadius: [5, 5, 0, 0],
        },
      },
      axes: [
        {
          orient: 'bottom',
          tick: { visible: false },
          label: {
            visible: true,
            autoRotate: false,
            autoHide: true,
            style: { fontSize: 10 },
          },
        },
        {
          orient: 'left',
          min: 0,
          grid: {
            visible: true,
            style: { lineDash: [4, 4], stroke: 'rgba(100,116,139,.16)' },
          },
          label: {
            formatMethod: (value) => compactFormatter.format(Number(value)),
          },
        },
      ],
      tooltip: {
        mark: {
          content: [
            {
              key: t('请求量'),
              value: (datum) => numberFormatter.format(datum?.requests || 0),
            },
          ],
        },
      },
    }),
    [t, trend],
  );

  const successTrendSpec = useMemo(() => {
    const rates = trend.map((item) => Number(item.successRate || 0));
    const minRate = rates.length
      ? Math.max(0, Math.floor(Math.min(...rates) - 1))
      : 0;
    return {
      type: 'line',
      animation: false,
      data: { values: trend },
      xField: 'time',
      yField: 'successRate',
      background: 'transparent',
      padding: { top: 20, right: 12, bottom: 8, left: 8 },
      line: {
        style: { stroke: '#10b981', lineWidth: 3, curveType: 'monotone' },
      },
      area: {
        visible: true,
        style: {
          fill: {
            gradient: 'linear',
            x0: 0,
            y0: 0,
            x1: 0,
            y1: 1,
            stops: [
              { offset: 0, color: 'rgba(16,185,129,0.32)' },
              { offset: 1, color: 'rgba(16,185,129,0.02)' },
            ],
          },
        },
      },
      point: {
        visible: false,
        state: { hover: { visible: true, size: 7, fill: '#10b981' } },
      },
      axes: [
        {
          orient: 'bottom',
          tick: { visible: false },
          label: {
            visible: true,
            autoRotate: false,
            autoHide: true,
            style: { fontSize: 10 },
          },
        },
        {
          orient: 'left',
          min: minRate,
          max: 100,
          grid: {
            visible: true,
            style: { lineDash: [4, 4], stroke: 'rgba(100,116,139,.16)' },
          },
          label: { formatMethod: (value) => `${Number(value).toFixed(0)}%` },
        },
      ],
      tooltip: {
        mark: {
          content: [
            {
              key: t('成功率'),
              value: (datum) => formatPerfSuccessRate(datum?.successRate),
            },
          ],
        },
      },
    };
  }, [t, trend]);

  const activeChart =
    MODEL_CHARTS.find((item) => item.key === activeModelChart) ||
    MODEL_CHARTS[0];
  const modelChartValues = useMemo(() => {
    const values = models
      .map((model) => ({
        model: model.model_name,
        value: Number(model[activeChart.field] || 0),
      }))
      .filter((item) => Number.isFinite(item.value));
    values.sort((a, b) =>
      activeChart.key === 'latency' ? a.value - b.value : b.value - a.value,
    );
    return values.slice(0, 12);
  }, [activeChart.field, activeChart.key, models]);

  const modelChartSpec = useMemo(
    () => ({
      type: 'bar',
      direction: 'horizontal',
      animation: false,
      data: { values: modelChartValues },
      xField: 'value',
      yField: 'model',
      background: 'transparent',
      padding: { top: 8, right: 76, bottom: 18, left: 8 },
      bar: {
        style: {
          fill: activeChart.color,
          cornerRadius: [0, 6, 6, 0],
        },
      },
      label: {
        visible: true,
        position: 'right',
        overlap: false,
        style: {
          fill: 'var(--semi-color-text-0)',
          fontSize: 11,
          fontWeight: 600,
        },
        formatMethod: (text, datum) =>
          formatChartValue(activeChart.key, Number(datum?.value ?? text)),
      },
      axes: [
        {
          orient: 'left',
          tick: { visible: false },
          domainLine: { visible: false },
          label: {
            visible: true,
            style: { fontSize: 11, fontWeight: 500 },
          },
        },
        {
          orient: 'bottom',
          min: 0,
          grid: {
            visible: true,
            style: { lineDash: [4, 5], stroke: 'rgba(100,116,139,.16)' },
          },
          label: {
            formatMethod: (value) =>
              formatChartValue(activeChart.key, Number(value)),
          },
        },
      ],
      tooltip: {
        mark: {
          content: [
            {
              key: t(activeChart.label),
              value: (datum) => formatChartValue(activeChart.key, datum?.value),
            },
          ],
        },
      },
    }),
    [activeChart, modelChartValues, t],
  );

  const validLatency = useMemo(
    () =>
      models
        .filter((item) => hasMetric(item.avg_latency_ms))
        .sort((a, b) => a.avg_latency_ms - b.avg_latency_ms)
        .slice(0, 10),
    [models],
  );
  const validTps = useMemo(
    () =>
      models
        .filter((item) => hasMetric(item.avg_tps))
        .sort((a, b) => b.avg_tps - a.avg_tps)
        .slice(0, 10),
    [models],
  );
  const validSuccess = useMemo(
    () =>
      models
        .filter((item) => Number.isFinite(Number(item.success_rate)))
        .sort(
          (a, b) =>
            b.success_rate - a.success_rate ||
            b.request_count - a.request_count,
        )
        .slice(0, 10),
    [models],
  );

  if (!enabled) return <Navigate to='/console' replace />;

  return (
    <div className='performance-page'>
      <div className='performance-page__inner dashboard-glass'>
        <header className='performance-header dashboard-glass-header'>
          <div>
            <div className='performance-header__title dashboard-glass-header__title'>
              <Gauge size={24} />
              <Title heading={3} style={{ margin: 0 }}>
                {t('性能概览')}
              </Title>
            </div>
            <Text type='tertiary'>{t('模型运行性能概览')}</Text>
          </div>
          <div className='performance-header__actions dashboard-glass-header__actions'>
            <div className='performance-header__icon-actions'>
              <Button
                theme='borderless'
                type='tertiary'
                aria-label={t('搜索条件')}
                icon={<Search size={17} />}
                onClick={() => setSearchModalVisible(true)}
                className='dashboard-glass-header__action dashboard-glass-header__action--search'
              />
              <Button
                theme='borderless'
                type='tertiary'
                aria-label={t('刷新')}
                icon={<RefreshCw size={17} />}
                loading={loading}
                onClick={() => load(appliedRange)}
                className='dashboard-glass-header__action dashboard-glass-header__action--refresh'
              />
            </div>
            <div
              className='performance-periods dashboard-chart-switcher'
              role='group'
              aria-label={t('快捷时间范围')}
            >
              {QUICK_RANGES.map((preset) => (
                <Button
                  key={preset.value}
                  className='dashboard-chart-switcher__button'
                  size='small'
                  theme={
                    activeQuickRange === preset.value ? 'solid' : 'borderless'
                  }
                  type={
                    activeQuickRange === preset.value ? 'primary' : 'tertiary'
                  }
                  onClick={() => applyQuickRange(preset)}
                >
                  {t(preset.label)}
                </Button>
              ))}
            </div>
          </div>
        </header>

        <Modal
          title={t('搜索条件')}
          visible={searchModalVisible}
          onOk={() => {
            applyRange();
            setSearchModalVisible(false);
          }}
          onCancel={() => setSearchModalVisible(false)}
          closeOnEsc
          centered
          size='small'
        >
          <Form layout='vertical' className='w-full'>
            <Form.DatePicker
              field='start_timestamp'
              label={t('起始时间')}
              type='dateTime'
              initValue={range?.[0]}
              value={range?.[0]}
              onChange={(value) => {
                setRange((current) => [value, current?.[1]]);
                setActiveQuickRange('');
              }}
              className='w-full mb-2'
            />
            <Form.DatePicker
              field='end_timestamp'
              label={t('结束时间')}
              type='dateTime'
              initValue={range?.[1]}
              value={range?.[1]}
              onChange={(value) => {
                setRange((current) => [current?.[0], value]);
                setActiveQuickRange('');
              }}
              className='w-full mb-2'
            />
            <Form.Select
              field='data_export_default_time'
              label={t('时间粒度')}
              initValue={granularity}
              value={granularity}
              optionList={GRANULARITY_OPTIONS.map((option) => ({
                ...option,
                label: t(option.label),
              }))}
              onChange={handleGranularityChange}
              className='w-full mb-2'
            />
          </Form>
        </Modal>

        {error ? <div className='performance-error'>{error}</div> : null}
        <Spin spinning={loading && models.length === 0}>
          <section className='performance-metrics'>
            <MetricCard
              icon={<BarChart3 size={18} />}
              label={t('请求总量')}
              value={compactFormatter.format(totalRequests)}
              detail={t('{{count}} 个模型有数据', { count: models.length })}
              tone='blue'
            />
            <MetricCard
              icon={<Activity size={18} />}
              label={t('平均成功率')}
              value={formatPerfSuccessRate(avgSuccess)}
              detail={t('按请求量加权')}
              tone='green'
            />
            <MetricCard
              icon={<Timer size={18} />}
              label={t('平均延迟')}
              value={formatPerfLatency(avgLatency)}
              detail='E2E'
              tone='amber'
            />
            <MetricCard
              icon={<Zap size={18} />}
              label={t('平均吞吐')}
              value={formatPerfThroughput(avgTps)}
              detail='Token / s'
              tone='cyan'
            />
          </section>

          <section className='performance-trend-grid'>
            <Card
              className='performance-trend-card dashboard-glass-card dashboard-glass-card--chart'
              title={
                <div className='performance-chart-heading'>
                  <span className='performance-chart-heading__icon performance-chart-heading__icon--requests'>
                    <BarChart3 size={16} />
                  </span>
                  <div>
                    <div className='performance-card-title'>
                      {t('请求量趋势')}
                    </div>
                    <Text type='tertiary' size='small'>
                      {t('每个时间段收到的请求数量')}
                    </Text>
                  </div>
                </div>
              }
            >
              {trend.length ? (
                <div className='performance-trend-chart dashboard-chart-canvas'>
                  <VChart spec={requestTrendSpec} option={CHART_CONFIG} />
                </div>
              ) : (
                <Empty description={t('暂无性能数据')} />
              )}
            </Card>
            <Card
              className='performance-trend-card dashboard-glass-card dashboard-glass-card--chart'
              title={
                <div className='performance-chart-heading'>
                  <span className='performance-chart-heading__icon performance-chart-heading__icon--success'>
                    <Activity size={16} />
                  </span>
                  <div>
                    <div className='performance-card-title'>
                      {t('成功率趋势')}
                    </div>
                    <Text type='tertiary' size='small'>
                      {t('请求成功比例随时间的变化')}
                    </Text>
                  </div>
                </div>
              }
            >
              {trend.length ? (
                <div className='performance-trend-chart dashboard-chart-canvas'>
                  <VChart spec={successTrendSpec} option={CHART_CONFIG} />
                </div>
              ) : (
                <Empty description={t('暂无性能数据')} />
              )}
            </Card>
          </section>

          <section className='performance-rankings'>
            <RankingBars
              title={t('稳定模型')}
              caption={t('成功率越高越好')}
              items={validSuccess}
              valueKey='success_rate'
              formatValue={formatPerfSuccessRate}
              color={(item) => getSuccessRateColor(item.success_rate)}
            />
            <RankingBars
              title={t('低延迟模型')}
              caption={t('E2E 延迟越低越好')}
              items={validLatency}
              valueKey='avg_latency_ms'
              formatValue={formatPerfLatency}
              color='#f59e0b'
              reverse
            />
            <RankingBars
              title={t('高吞吐模型')}
              caption={t('每秒输出 Token 越高越好')}
              items={validTps}
              valueKey='avg_tps'
              formatValue={formatPerfThroughput}
              color='#06b6d4'
            />
          </section>

          <Card
            className='performance-model-chart dashboard-glass-card dashboard-glass-card--chart'
            title={
              <div>
                <div className='performance-card-title'>{t('模型表现')}</div>
                <Text type='tertiary' size='small'>
                  {t('按指标排序，最佳模型位于顶部')}
                </Text>
              </div>
            }
            headerExtraContent={
              <div
                className='performance-model-switcher dashboard-chart-switcher'
                role='tablist'
              >
                {MODEL_CHARTS.map((chart) => (
                  <Button
                    key={chart.key}
                    className='dashboard-chart-switcher__button'
                    size='small'
                    theme={
                      activeModelChart === chart.key ? 'solid' : 'borderless'
                    }
                    type={
                      activeModelChart === chart.key ? 'primary' : 'tertiary'
                    }
                    onClick={() => setActiveModelChart(chart.key)}
                  >
                    {t(chart.label)}
                  </Button>
                ))}
              </div>
            }
          >
            <div className='performance-model-chart__canvas dashboard-chart-canvas'>
              <VChart spec={modelChartSpec} option={CHART_CONFIG} />
            </div>
          </Card>
        </Spin>
      </div>
    </div>
  );
};

export default Performance;
