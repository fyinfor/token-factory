/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Card, Select, Spin, Typography, Space } from '@douyinfe/semi-ui';
import { VChart } from '@visactor/react-vchart';
import { BarChart3, LineChart, TrendingUp } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import {
  API,
  showError,
  formatCommissionRatioPercent,
  getCurrencyConfig,
  getQuotaPerUnit,
} from '../../helpers';
import { CHART_CONFIG } from '../../constants/dashboard.constants';

const { Text, Title } = Typography;

/** 与控制台金额展示一致；TOKENS 展示模式下纵轴用美元等价（quota / quota_per_unit）。 */
function quotaToChartMoneyNumber(quota) {
  const q = Number(quota || 0);
  if (!Number.isFinite(q) || q <= 0) return 0;
  const { type, rate } = getCurrencyConfig();
  let per = getQuotaPerUnit();
  per = Number(per);
  if (!Number.isFinite(per) || per <= 0) {
    per = parseFloat(localStorage.getItem('quota_per_unit') || '500000');
  }
  if (!Number.isFinite(per) || per <= 0) return 0;
  const usd = q / per;
  if (type === 'TOKENS') return usd;
  if (type === 'USD') return usd;
  return usd * (rate || 1);
}

function formatChartMoneyTick(v) {
  const n = Number(v);
  if (!Number.isFinite(n)) return '';
  const { symbol, type } = getCurrencyConfig();
  if (type === 'TOKENS') return `$${n.toFixed(2)}`;
  return `${symbol}${n.toFixed(2)}`;
}

function getMoneyAxisTitle(t) {
  const { type, symbol } = getCurrencyConfig();
  if (type === 'TOKENS') return t('金额约美元');
  return `${t('金额')}(${symbol})`;
}

function buildLongSeriesIntCount(rows, fields) {
  const out = [];
  for (const row of rows || []) {
    for (const { key, label } of fields) {
      const raw = Number(row[key] ?? 0);
      const v = Number.isFinite(raw) ? Math.max(0, Math.round(raw)) : 0;
      out.push({
        date: row.date?.slice(5) || row.date,
        type: label,
        value: v,
      });
    }
  }
  return out;
}

function buildLongSeriesMoney(rows, fields) {
  const out = [];
  for (const row of rows || []) {
    for (const { key, label } of fields) {
      out.push({
        date: row.date?.slice(5) || row.date,
        type: label,
        value: quotaToChartMoneyNumber(row[key] ?? 0),
      });
    }
  }
  return out;
}

const intTick = (v) => {
  const n = Number(v);
  return Number.isFinite(n) ? String(Math.round(n)) : String(v);
};

/** 管理端代理 TOP 柱状图至少展示的档位数 */
const ADMIN_TOP_MIN_SLOTS = 5;

export default function DistributorAnalyticsBoard({ adminMode = false }) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(true);
  const [days, setDays] = useState(30);
  const [series, setSeries] = useState([]);
  const [topInvitees, setTopInvitees] = useState([]);
  const [topTotal, setTopTotal] = useState([]);
  const [topPeriod, setTopPeriod] = useState([]);
  const [topInviteCount, setTopInviteCount] = useState([]);
  const [effectiveBps, setEffectiveBps] = useState(0);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const url = adminMode
        ? `/api/distributor/admin/analytics?days=${days}`
        : `/api/distributor/analytics?days=${days}`;
      const res = await API.get(url);
      const { success, message, data } = res.data || {};
      if (!success) {
        showError(message || t('加载失败'));
        return;
      }
      setSeries(data?.series || []);
      if (adminMode) {
        setTopTotal(data?.top_total_reward || []);
        setTopPeriod(data?.top_period_reward || []);
        setTopInviteCount(data?.top_invitee_count || []);
        setTopInvitees([]);
      } else {
        setTopInvitees(data?.top_invitees || []);
        setEffectiveBps(data?.effective_commission_bps ?? 0);
        setTopTotal([]);
        setTopPeriod([]);
        setTopInviteCount([]);
      }
    } catch {
      showError(t('加载失败'));
    } finally {
      setLoading(false);
    }
  }, [adminMode, days, t]);

  useEffect(() => {
    load();
  }, [load]);

  const last7 = useMemo(() => {
    const s = series || [];
    return s.length <= 7 ? s : s.slice(s.length - 7);
  }, [series]);

  /** 流量图纵轴上限：至少 5 次，避免数据很小时刻度范围过小 */
  const funnelYMax = useMemo(() => {
    const keys = [
      'short_link_clicks',
      'register_page_views',
      'new_registrations',
    ];
    let m = 0;
    for (const row of last7 || []) {
      for (const k of keys) {
        m = Math.max(m, Math.round(Number(row[k] ?? 0)));
      }
    }
    return Math.max(5, m);
  }, [last7]);

  const funnelSpec = useMemo(() => {
    const fields = [
      { key: 'short_link_clicks', label: t('邀请链接点击') },
      { key: 'register_page_views', label: t('注册页浏览') },
      { key: 'new_registrations', label: t('新增邀请注册') },
    ];
    const values = buildLongSeriesIntCount(last7, fields);
    return {
      type: 'line',
      title: { visible: false },
      data: { values },
      xField: 'date',
      yField: 'value',
      seriesField: 'type',
      legends: { visible: true, orient: 'bottom' },
      point: { visible: true },
      axes: [
        { orient: 'bottom', label: { formatMethod: (v) => String(v) } },
        {
          orient: 'left',
          title: { visible: true, text: t('次数') },
          label: { formatMethod: intTick },
          min: 0,
          max: funnelYMax,
          tick: { tickMode: 'd3' },
        },
      ],
    };
  }, [last7, t, funnelYMax]);

  const revenueSpec = useMemo(() => {
    const fields = [
      { key: 'reward_quota', label: t('分销收益金额') },
      { key: 'invitee_quota_added', label: t('下级充值入账金额') },
    ];
    const values = buildLongSeriesMoney(series, fields);
    return {
      type: 'line',
      title: { visible: false },
      data: { values },
      xField: 'date',
      yField: 'value',
      seriesField: 'type',
      legends: { visible: true, orient: 'bottom' },
      point: { visible: true },
      axes: [
        { orient: 'bottom' },
        {
          orient: 'left',
          title: { visible: true, text: getMoneyAxisTitle(t) },
          label: { formatMethod: formatChartMoneyTick },
          min: 0,
        },
      ],
    };
  }, [series, t]);

  /** 固定 10 个槽位，不足补占位（图表与列表均占满 Top 10） */
  const top10Slots = useMemo(() => {
    const raw = (topInvitees || []).slice(0, 10);
    const slots = new Array(10).fill(null);
    for (let i = 0; i < raw.length; i++) {
      slots[i] = raw[i];
    }
    return slots;
  }, [topInvitees]);

  const inviteeBarSpec = useMemo(() => {
    const rank = (idx) => `Top ${idx + 1}`;
    const values = top10Slots.map((r, i) => {
      if (!r) {
        return {
          name: `${rank(i)} ${t('虚位以待')}`,
          value: 0,
          idx: i,
        };
      }
      const who = String(
        r.display_name || r.username || `#${r.invitee_user_id}`,
      ).slice(0, 12);
      return {
        name: `${rank(i)} ${who}`,
        value: quotaToChartMoneyNumber(r.total_reward_quota || 0),
        idx: i,
      };
    });
    return {
      type: 'bar',
      title: { visible: false },
      data: { values },
      xField: 'name',
      yField: 'value',
      axes: [
        { orient: 'bottom', label: { autoRotate: true } },
        {
          orient: 'left',
          title: { visible: true, text: getMoneyAxisTitle(t) },
          label: { formatMethod: formatChartMoneyTick },
          min: 0,
        },
      ],
    };
  }, [top10Slots, t]);

  const makeAdminBarSpec = useCallback(
    (rows, valueKey, money) => {
      const list = [...(rows || [])];
      while (list.length < ADMIN_TOP_MIN_SLOTS) {
        list.push(null);
      }
      const rank = (idx) => `Top ${idx + 1}`;
      const values = list.map((r, i) => {
        if (!r) {
          return {
            name: `${rank(i)} ${t('虚位以待')}`,
            value: 0,
            idx: i,
          };
        }
        const who = String(
          r.username || r.display_name || `#${r.user_id}`,
        ).slice(0, 10);
        return {
          name: `${rank(i)} ${who}`,
          value: money
            ? quotaToChartMoneyNumber(r[valueKey] ?? 0)
            : Math.max(0, Math.round(Number(r[valueKey] ?? 0))),
          idx: i,
        };
      });
      return {
        type: 'bar',
        title: { visible: false },
        data: { values },
        xField: 'name',
        yField: 'value',
        axes: [
          { orient: 'bottom', label: { autoRotate: true } },
          {
            orient: 'left',
            title: {
              visible: true,
              text: money ? getMoneyAxisTitle(t) : t('邀请人数'),
            },
            label: {
              formatMethod: money ? formatChartMoneyTick : intTick,
            },
            min: 0,
          },
        ],
      };
    },
    [t],
  );

  return (
    <div className='space-y-6 w-full'>
      <div className='flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3'>
        <div>
          <Title heading={5} className='!mb-1 flex items-center gap-2'>
            <TrendingUp size={18} />
            {adminMode ? t('分销数据大盘') : t('数据看板')}
          </Title>
          {!adminMode && effectiveBps > 0 ? (
            <Text type='tertiary' size='small'>
              {t('当前默认分销比例')}：
              {formatCommissionRatioPercent(effectiveBps)}
            </Text>
          ) : null}
        </div>
        <Space>
          <Text type='tertiary' size='small'>
            {t('统计区间')}
          </Text>
          <Select
            value={days}
            onChange={(v) => setDays(Number(v))}
            style={{ width: 120 }}
            optionList={[
              { label: t('近7日'), value: 7 },
              { label: t('近30日'), value: 30 },
              { label: t('近90日'), value: 90 },
            ]}
          />
        </Space>
      </div>

      <Spin spinning={loading}>
        <div className='flex flex-col gap-6'>
        <div className='grid grid-cols-1 xl:grid-cols-2 gap-6'>
          <Card
            title={
              <span className='flex items-center gap-2'>
                <LineChart size={16} />
                {t('近7日流量与注册')}
              </span>
            }
            className='!rounded-2xl'
            bodyStyle={{ paddingBottom: 8 }}
          >
            <div className='h-80'>
              <VChart spec={funnelSpec} option={CHART_CONFIG} />
            </div>
          </Card>
          <Card
            title={
              <span className='flex items-center gap-2'>
                <LineChart size={16} />
                {t('收益与充值带动')}
                <Text type='tertiary' size='small' className='!font-normal'>
                  ({t('所选区间')} · {t('金额为展示货币')})
                </Text>
              </span>
            }
            className='!rounded-2xl'
            bodyStyle={{ paddingBottom: 8 }}
          >
            <div className='h-80'>
              <VChart spec={revenueSpec} option={CHART_CONFIG} />
            </div>
          </Card>
        </div>

        {!adminMode ? (
          <Card
            title={
              <span className='flex items-center gap-2'>
                <BarChart3 size={16} />
                {t('被邀请人收益 TOP10')}
              </span>
            }
            className='!rounded-2xl'
            bodyStyle={{ paddingBottom: 8 }}
          >
            <Text type='tertiary' size='small' className='block mb-2'>
              {t('按累计分销收益排序；纵轴为展示货币金额。')}
            </Text>
            <div className='h-72'>
              <VChart spec={inviteeBarSpec} option={CHART_CONFIG} />
            </div>
          </Card>
        ) : (
          <div className='grid grid-cols-1 lg:grid-cols-3 gap-6'>
            <Card
              title={t('代理累计收益 TOP')}
              className='!rounded-2xl'
              bodyStyle={{ paddingBottom: 8 }}
            >
              <div className='h-64'>
                <VChart
                  spec={makeAdminBarSpec(topTotal, 'total_reward_quota', true)}
                  option={CHART_CONFIG}
                />
              </div>
            </Card>
            <Card
              title={t('代理近30日收益 TOP')}
              className='!rounded-2xl'
              bodyStyle={{ paddingBottom: 8 }}
            >
              <div className='h-64'>
                <VChart
                  spec={makeAdminBarSpec(
                    topPeriod,
                    'total_reward_quota',
                    true,
                  )}
                  option={CHART_CONFIG}
                />
              </div>
            </Card>
            <Card
              title={t('代理邀请人数 TOP')}
              className='!rounded-2xl'
              bodyStyle={{ paddingBottom: 8 }}
            >
              <div className='h-64'>
                <VChart
                  spec={makeAdminBarSpec(
                    topInviteCount,
                    'invitee_count',
                    false,
                  )}
                  option={CHART_CONFIG}
                />
              </div>
            </Card>
          </div>
        )}
        </div>
      </Spin>
    </div>
  );
}
