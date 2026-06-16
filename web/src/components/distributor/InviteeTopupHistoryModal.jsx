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
import {
  Button,
  Input,
  Modal,
  Select,
  Space,
  Table,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { Search } from 'lucide-react';
import { API, showError, timestamp2string } from '../../helpers';
import { StatusContext } from '../../context/Status';

const { Text } = Typography;

const TOPUP_STATUS_ALL = '__all__';

const STATUS_META = {
  success: { color: 'green', label: '成功' },
  pending: { color: 'blue', label: '待支付' },
  failed: { color: 'red', label: '失败' },
  expired: { color: 'grey', label: '已过期' },
};

const PAYMENT_METHOD_MAP = {
  stripe: 'Stripe',
  creem: 'Creem',
  waffo: 'Waffo',
  alipay: '支付宝',
  wxpay: '微信',
  ALI_PC: '支付宝',
  WX_NATIVE: '微信',
};

function semiInputString(valOrEvt) {
  if (typeof valOrEvt === 'string') return valOrEvt;
  if (
    valOrEvt &&
    typeof valOrEvt === 'object' &&
    'target' in valOrEvt &&
    valOrEvt.target &&
    typeof valOrEvt.target.value === 'string'
  ) {
    return valOrEvt.target.value;
  }
  return '';
}

function formatTopupPayMoney(money, paymentMethod, usdExchangeRate) {
  const numericMoney = Number(money);
  const safeMoney = Number.isFinite(numericMoney) ? numericMoney : 0;
  const rate =
    Number.isFinite(usdExchangeRate) && usdExchangeRate > 0
      ? usdExchangeRate
      : 7.3;
  if ((paymentMethod || '').toLowerCase() === 'stripe') {
    return `￥${(safeMoney * rate).toFixed(2)}`;
  }
  return `￥${safeMoney.toFixed(2)}`;
}

export default function InviteeTopupHistoryModal({
  visible,
  onCancel,
  inviteeId,
  inviteeLabel,
  t,
}) {
  const [statusState] = useContext(StatusContext);
  const [loading, setLoading] = useState(false);
  const [rows, setRows] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [tradeNoFilter, setTradeNoFilter] = useState('');
  const [statusFilter, setStatusFilter] = useState(TOPUP_STATUS_ALL);

  const usdExchangeRate = useMemo(() => {
    const s = statusState?.status;
    const rate = Number(s?.usd_exchange_rate ?? s?.price);
    return Number.isFinite(rate) && rate > 0 ? rate : 7.3;
  }, [statusState?.status]);

  const statusOptions = useMemo(
    () => [
      { value: TOPUP_STATUS_ALL, label: t('全部') },
      { value: 'success', label: t('成功') },
      { value: 'pending', label: t('待支付') },
      { value: 'failed', label: t('失败') },
      { value: 'expired', label: t('已过期') },
    ],
    [t],
  );

  const load = useCallback(
    async (p, ps) => {
      if (!inviteeId) return;
      setLoading(true);
      try {
        const params = { p, page_size: ps };
        const tradeNo = tradeNoFilter.trim();
        if (tradeNo) {
          params.trade_no = tradeNo;
        }
        if (statusFilter && statusFilter !== TOPUP_STATUS_ALL) {
          params.status = statusFilter;
        }
        const res = await API.get(
          `/api/distributor/invitee/${inviteeId}/topups`,
          {
            params,
            disableDuplicate: true,
          },
        );
        const { success, message, data } = res.data || {};
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
    [inviteeId, statusFilter, t, tradeNoFilter],
  );

  useEffect(() => {
    if (!visible || !inviteeId) return;
    setPage(1);
    setPageSize(10);
    setTradeNoFilter('');
    setStatusFilter(TOPUP_STATUS_ALL);
  }, [visible, inviteeId]);

  useEffect(() => {
    if (!visible || !inviteeId) return;
    load(page, pageSize);
  }, [visible, inviteeId, page, pageSize, tradeNoFilter, statusFilter, load]);

  const renderStatus = useCallback(
    (status) => {
      const meta = STATUS_META[status] || {
        color: 'grey',
        label: status || '-',
      };
      return (
        <Tag color={meta.color} type='light' size='small'>
          {t(meta.label)}
        </Tag>
      );
    },
    [t],
  );

  const renderPaymentMethod = useCallback(
    (method) => {
      const label = PAYMENT_METHOD_MAP[method] || method || '-';
      return <Text>{label === '-' ? label : t(label)}</Text>;
    },
    [t],
  );

  const columns = useMemo(
    () => [
      {
        title: t('订单号'),
        dataIndex: 'trade_no',
        width: 150,
        render: (text) => <Text copyable>{text || '-'}</Text>,
      },
      {
        title: t('充值额度'),
        dataIndex: 'amount',
        width: 110,
        render: (amount, record) => {
          const tradeNo = String(record?.trade_no || '').toLowerCase();
          if (Number(amount || 0) === 0 && tradeNo.startsWith('sub')) {
            return (
              <Tag color='purple' type='light' size='small'>
                {t('订阅套餐')}
              </Tag>
            );
          }
          return <Text>{Number(amount || 0)}</Text>;
        },
      },
      {
        title: t('支付金额'),
        dataIndex: 'money',
        width: 110,
        render: (money, record) => (
          <Text type='danger'>
            {formatTopupPayMoney(
              money,
              record?.payment_method,
              usdExchangeRate,
            )}
          </Text>
        ),
      },
      {
        title: t('支付方式'),
        dataIndex: 'payment_method',
        width: 110,
        render: renderPaymentMethod,
      },
      {
        title: t('状态'),
        dataIndex: 'status',
        width: 96,
        render: renderStatus,
      },
      {
        title: t('创建时间'),
        dataIndex: 'create_time',
        width: 170,
        render: (time) => (time ? timestamp2string(time) : '-'),
      },
      {
        title: t('完成时间'),
        dataIndex: 'complete_time',
        width: 170,
        render: (time) => (time ? timestamp2string(time) : '-'),
      },
    ],
    [renderPaymentMethod, renderStatus, t, usdExchangeRate],
  );

  return (
    <Modal
      title={
        <span>
          {t('充值记录')}
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
      width='calc(100vw - 32px)'
      style={{ maxWidth: 1120 }}
    >
      <Space className='mb-3 w-full flex-wrap px-1' align='center'>
        <Input
          prefix={<Search size={15} className='ml-1' />}
          placeholder={t('订单号')}
          value={tradeNoFilter}
          onChange={(value) => {
            setTradeNoFilter(semiInputString(value));
            setPage(1);
          }}
          showClear
          style={{ width: 260 }}
        />
        <Select
          value={statusFilter}
          onChange={(value) => {
            setStatusFilter(value || TOPUP_STATUS_ALL);
            setPage(1);
          }}
          optionList={statusOptions}
          placeholder={t('状态')}
          allowClear
          style={{ width: 150 }}
        />
        <Button onClick={() => load(1, pageSize)}>{t('刷新')}</Button>
      </Space>
      <style>
        {`
          .invitee-topup-history-table,
          .invitee-topup-history-table .semi-table-container,
          .invitee-topup-history-table .semi-table-header,
          .invitee-topup-history-table .semi-table-body,
          .invitee-topup-history-table table {
            width: 100% !important;
          }
        `}
      </style>
      <Table
        className='invitee-topup-history-table w-full'
        style={{ width: '100%' }}
        loading={loading}
        rowKey='id'
        size='small'
        columns={columns}
        dataSource={rows}
        scroll={{ x: 990 }}
        pagination={{
          currentPage: page,
          pageSize,
          total,
          showSizeChanger: true,
          pageSizeOpts: [10, 20, 50, 100],
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
