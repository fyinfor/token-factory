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

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Modal,
  Switch,
  Table,
  Tag,
  Tooltip,
  Typography,
} from '@douyinfe/semi-ui';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';
import {
  API,
  showError,
  formatCommissionRatioPercent,
  renderQuota,
} from '../../helpers';
import {
  ProfitShareRewardColumnTitle,
  renderProfitShareQuotaCell,
  renderProfitShareRewardCell,
} from './profitShareDisplay';

const { Text } = Typography;

const renderLoggedCommissionRatio = (bps) => {
  const n = Number(bps);
  return Number.isFinite(n) && n > 0 ? formatCommissionRatioPercent(n) : '-';
};

export default function AffInviteeCommissionDetailModal({
  visible,
  onCancel,
  inviteeId,
  inviteeLabel,
  commissionMode = 'topup',
  billingModeFilter = '',
}) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [rows, setRows] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [hideZeroReward, setHideZeroReward] = useState(false);

  const isProfitShare = commissionMode === 'profit_share';

  const load = useCallback(
    async (p, ps) => {
      if (!inviteeId) return;
      setLoading(true);
      try {
        const bm = String(billingModeFilter || '').trim();
        const bmParam = bm ? `&billing_mode=${encodeURIComponent(bm)}` : '';
        const rewardParam = hideZeroReward ? '&hide_zero_reward=true' : '';
        const path = isProfitShare
          ? `/api/distributor/invitee/${inviteeId}/profit-shares?p=${p}&page_size=${ps}${bmParam}${rewardParam}`
          : `/api/distributor/invitee/${inviteeId}/commissions?p=${p}&page_size=${ps}`;
        const res = await API.get(path);
        const { success, message, data } = res.data;
        if (!success) {
          showError(message || t('加载失败'));
          return;
        }
        setRows(data?.items || []);
        setTotal(data?.total ?? 0);
        setPage(p);
      } catch {
        showError(t('加载失败'));
      } finally {
        setLoading(false);
      }
    },
    [inviteeId, isProfitShare, billingModeFilter, hideZeroReward, t],
  );

  useEffect(() => {
    if (!visible || !inviteeId) return;
    setPage(1);
    setPageSize(10);
    load(1, 10);
  }, [visible, inviteeId, load]);

  const columns = useMemo(() => {
    if (isProfitShare) {
      return [
        {
          title: t('时间'),
          dataIndex: 'created_at',
          width: 170,
          render: (ts) =>
            ts ? dayjs.unix(Number(ts)).format('YYYY-MM-DD HH:mm:ss') : '—',
        },
        {
          title: t('模型'),
          dataIndex: 'model_name',
          width: 200,
          render: (m) => (m ? String(m) : '—'),
        },
        {
          title: t('计费分类'),
          dataIndex: 'billing_mode',
          width: 110,
          render: (bm) => {
            const mode = String(bm ?? '').trim();
            if (mode === 'video_token') {
              return (
                <Tag color='violet' type='light' size='small'>
                  {t('视频按Token')}
                </Tag>
              );
            }
            if (mode === 'video') {
              return (
                <Tag color='cyan' type='light' size='small'>
                  {t('视频其他')}
                </Tag>
              );
            }
            if (mode === 'text') {
              return (
                <Tag color='grey' type='light' size='small'>
                  {t('文本对话')}
                </Tag>
              );
            }
            return (
              <Tag type='light' size='small'>
                {t('未分类')}
              </Tag>
            );
          },
        },
        {
          title: t('Token消耗'),
          dataIndex: 'total_tokens',
          width: 120,
          render: (tk) => {
            const n = Number(tk);
            return n > 0 ? n.toLocaleString() : '—';
          },
        },
        {
          title: t('渠道路由后缀'),
          dataIndex: 'route_slug',
          width: 120,
          render: (slug) => {
            const s = String(slug ?? '').trim();
            return s || '—';
          },
        },
        {
          title: t('用户消耗金额'),
          dataIndex: 'user_quota_charged',
          width: 130,
          render: (q) => renderProfitShareQuotaCell(q),
        },
        {
          title: (
            <Tooltip content={t('利润金额说明')}>
              <span className='cursor-help border-b border-dotted border-gray-400'>
                {t('利润金额')}
              </span>
            </Tooltip>
          ),
          dataIndex: 'markup_slice_quota',
          width: 130,
          render: (q) => renderProfitShareQuotaCell(q),
        },
        {
          title: t('当时分销比例'),
          dataIndex: 'commission_bps',
          width: 120,
          render: (bps) => renderLoggedCommissionRatio(bps),
        },
        {
          title: <ProfitShareRewardColumnTitle t={t} />,
          dataIndex: 'reward_quota',
          width: 120,
          render: (_, row) => renderProfitShareRewardCell(row, t),
        },
      ];
    }
    return [
      {
        title: t('时间'),
        dataIndex: 'created_at',
        width: 170,
        render: (ts) =>
          ts ? dayjs.unix(Number(ts)).format('YYYY-MM-DD HH:mm:ss') : '—',
      },
      {
        title: t('充值入账额度'),
        dataIndex: 'invitee_quota_added',
        render: (q) => renderQuota(q || 0),
      },
      {
        title: t('当时分成比例'),
        dataIndex: 'commission_bps',
        width: 120,
        render: (bps) => renderLoggedCommissionRatio(bps),
      },
      {
        title: t('收益金额'),
        dataIndex: 'reward_quota',
        render: (q) => renderQuota(q || 0),
      },
    ];
  }, [isProfitShare, t]);

  const titleText = isProfitShare ? t('利润分成明细') : t('分成明细');

  return (
    <Modal
      title={
        <span>
          {titleText}
          {inviteeLabel ? (
            <Text type='tertiary' size='small' className='ml-2 font-normal'>
              {inviteeLabel}
            </Text>
          ) : null}
        </span>
      }
      visible={visible}
      onCancel={onCancel}
      footer={null}
      width={isProfitShare ? 1420 : 880}
      bodyStyle={isProfitShare ? { overflow: 'visible' } : undefined}
    >
      {isProfitShare ? (
        <div className='mb-3 flex items-center justify-end gap-2'>
          <Text type='secondary'>{t('隐藏零收益记录')}</Text>
          <Switch
            size='small'
            checked={hideZeroReward}
            disabled={loading}
            onChange={(checked) => setHideZeroReward(Boolean(checked))}
            aria-label={t('隐藏零收益记录')}
          />
        </div>
      ) : null}
      <Table
        loading={loading}
        rowKey='id'
        columns={columns}
        dataSource={rows}
        pagination={{
          currentPage: page,
          pageSize,
          total,
          onPageChange: (p) => load(p, pageSize),
          onPageSizeChange: (ps) => {
            setPageSize(ps);
            load(1, ps);
          },
        }}
      />
    </Modal>
  );
}
