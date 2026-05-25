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
import { Modal, Table, Tooltip, Typography } from '@douyinfe/semi-ui';
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

export default function AffInviteeCommissionDetailModal({
  visible,
  onCancel,
  inviteeId,
  inviteeLabel,
  commissionMode = 'topup',
}) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [rows, setRows] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);

  const isProfitShare = commissionMode === 'profit_share';

  const load = useCallback(
    async (p, ps) => {
      if (!inviteeId) return;
      setLoading(true);
      try {
        const path = isProfitShare
          ? `/api/distributor/invitee/${inviteeId}/profit-shares?p=${p}&page_size=${ps}`
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
    [inviteeId, isProfitShare, t],
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
          title: t('渠道路由后缀'),
          dataIndex: 'route_slug',
          width: 120,
          render: (slug) => {
            const s = String(slug ?? '').trim();
            return s || '—';
          },
        },
        {
          title: t('用户消耗额度'),
          dataIndex: 'user_quota_charged',
          width: 130,
          render: (q) => renderProfitShareQuotaCell(q),
        },
        {
          title: (
            <Tooltip content={t('加价切片额度说明')}>
              <span className='cursor-help border-b border-dotted border-gray-400'>
                {t('加价切片额度')}
              </span>
            </Tooltip>
          ),
          dataIndex: 'markup_slice_quota',
          width: 130,
          render: (q) => renderProfitShareQuotaCell(q),
        },
        {
          title: t('当时分成比例'),
          dataIndex: 'commission_bps',
          width: 110,
          render: (bps) =>
            typeof bps === 'number' && bps > 0
              ? formatCommissionRatioPercent(bps)
              : '—',
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
        render: (bps) => formatCommissionRatioPercent(bps),
      },
      {
        title: t('收益额度'),
        dataIndex: 'reward_quota',
        render: (q) => renderQuota(q || 0),
      },
    ];
  }, [isProfitShare, t]);

  const titleText = isProfitShare ? t('利润分成明细') : t('分成明细');
  const hintText = isProfitShare
    ? t('利润分成明细说明')
    : t('每次被邀请用户充值入账后，按当时适用的分成比例计算一条记录。');

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
      width={isProfitShare ? 1180 : 880}
      bodyStyle={isProfitShare ? { overflow: 'visible' } : undefined}
    >
      <Text type='tertiary' size='small' className='block mb-3'>
        {hintText}
      </Text>
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
