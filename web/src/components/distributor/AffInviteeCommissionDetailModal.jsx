/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import React, { useCallback, useEffect, useState } from 'react';
import { Modal, Table, Typography } from '@douyinfe/semi-ui';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';
import {
  API,
  showError,
  formatCommissionRatioPercent,
  renderQuota,
} from '../../helpers';

const { Text } = Typography;

export default function AffInviteeCommissionDetailModal({
  visible,
  onCancel,
  inviteeId,
  inviteeLabel,
}) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [rows, setRows] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);

  const load = useCallback(
    async (p, ps) => {
      if (!inviteeId) return;
      setLoading(true);
      try {
        const res = await API.get(
          `/api/distributor/invitee/${inviteeId}/commissions?p=${p}&page_size=${ps}`,
        );
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
    [inviteeId, t],
  );

  useEffect(() => {
    if (!visible || !inviteeId) return;
    setPage(1);
    setPageSize(10);
    load(1, 10);
  }, [visible, inviteeId, load]);

  const columns = [
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

  return (
    <Modal
      title={
        <span>
          {t('分成明细')}
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
      width={880}
    >
      <Text type='tertiary' size='small' className='block mb-3'>
        {t('每次被邀请用户充值入账后，按当时适用的分成比例计算一条记录。')}
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
