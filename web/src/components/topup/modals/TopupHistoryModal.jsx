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
import React, { useState, useEffect, useMemo } from 'react';
import {
  Modal,
  Table,
  Badge,
  Typography,
  Toast,
  Empty,
  Button,
  Input,
  Tag,
  Select,
} from '@douyinfe/semi-ui';
import {
  IllustrationNoResult,
  IllustrationNoResultDark,
} from '@douyinfe/semi-illustrations';
import { Coins } from 'lucide-react';
import { IconSearch } from '@douyinfe/semi-icons';
import { API, timestamp2string } from '../../../helpers';
import { isAdmin } from '../../../helpers/utils';
import { useIsMobile } from '../../../hooks/common/useIsMobile';
const { Text } = Typography;

// 状态映射配置
const STATUS_CONFIG = {
  success: { type: 'success', key: '成功' },
  pending: { type: 'warning', key: '待支付' },
  failed: { type: 'danger', key: '失败' },
  expired: { type: 'danger', key: '已过期' },
};

// 支付方式映射
const PAYMENT_METHOD_MAP = {
  stripe: 'Stripe',
  creem: 'Creem',
  waffo: 'Waffo',
  alipay: '支付宝',
  wxpay: '微信',
  ALI_PC: '支付宝',
  WX_NATIVE: '微信',
};

/** 支付状态筛选项「不限」占位值（Semi Select 对 value="" 不稳定，勿用空串） */
const TOPUP_STATUS_ALL = '__all__';

/**
 * 从 Semi Input onChange 首参解析字符串（兼容部分环境下传入合成事件的情况）。
 * @param {unknown} valOrEvt onChange 第一个参数
 * @returns {string}
 */
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

/**
 * 钱包管理「充值账单」弹窗：分页列表；支持用户名（管理员）、订单号、支付状态筛选。
 * @param {{ visible: boolean, onCancel: () => void, t: (key: string) => string }} props 弹窗显隐与文案函数
 */
const TopupHistoryModal = ({ visible, onCancel, t }) => {
  const [loading, setLoading] = useState(false);
  const [topups, setTopups] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  /** 订单号模糊筛选（trade_no） */
  const [tradeNoFilter, setTradeNoFilter] = useState('');
  /** 管理员按用户名模糊筛选 */
  const [usernameFilter, setUsernameFilter] = useState('');
  /** 支付状态：TOPUP_STATUS_ALL 表示不限；其它与后端 TopUp.status 一致 */
  const [statusFilter, setStatusFilter] = useState(TOPUP_STATUS_ALL);
  const isMobile = useIsMobile();

  /** 每次渲染读取管理员身份（避免 useMemo 空依赖导致角色变更后仍走 self 接口） */
  const userIsAdmin = isAdmin();

  /** 支付状态下拉选项（与后端 TopUp.status / STATUS_CONFIG 一致） */
  const statusFilterOptions = useMemo(
    () => [
      { value: TOPUP_STATUS_ALL, label: t('全部') },
      { value: 'success', label: t('成功') },
      { value: 'pending', label: t('待支付') },
      { value: 'failed', label: t('失败') },
      { value: 'expired', label: t('已过期') },
    ],
    [t],
  );

  /** 请求充值账单分页数据 */
  const loadTopups = async (currentPage, currentPageSize) => {
    setLoading(true);
    try {
      const adminNow = isAdmin();
      const base = adminNow ? '/api/user/topup' : '/api/user/topup/self';
      const tn = tradeNoFilter.trim();
      const un = usernameFilter.trim();
      /** @type {Record<string, string | number>} */
      const params = {
        p: currentPage,
        page_size: currentPageSize,
      };
      if (tn) params.trade_no = tn;
      if (adminNow && un) params.username = un;
      if (statusFilter && statusFilter !== TOPUP_STATUS_ALL) {
        params.status = statusFilter;
      }
      const res = await API.get(base, {
        params,
        disableDuplicate: true,
      });
      const { success, message, data } = res.data;
      if (success) {
        setTopups(data.items || []);
        setTotal(data.total || 0);
      } else {
        Toast.error({ content: message || t('加载失败') });
      }
    } catch (error) {
      Toast.error({ content: t('加载账单失败') });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (visible) {
      loadTopups(page, pageSize);
    }
  }, [visible, page, pageSize, tradeNoFilter, usernameFilter, statusFilter]);

  const handlePageChange = (currentPage) => {
    setPage(currentPage);
  };

  const handlePageSizeChange = (currentPageSize) => {
    setPageSize(currentPageSize);
    setPage(1);
  };

  /** 订单号输入变更 */
  const handleTradeNoChange = (value) => {
    setTradeNoFilter(semiInputString(value));
    setPage(1);
  };

  /** 用户名筛选变更（管理员） */
  const handleUsernameChange = (value) => {
    setUsernameFilter(semiInputString(value));
    setPage(1);
  };

  /** 支付状态变更（含清空回到「全部」） */
  const handleStatusFilterChange = (value) => {
    const v = value == null || value === '' ? TOPUP_STATUS_ALL : value;
    setStatusFilter(v);
    setPage(1);
  };

  // 管理员补单
  const handleAdminComplete = async (tradeNo) => {
    try {
      const res = await API.post('/api/user/topup/complete', {
        trade_no: tradeNo,
      });
      const { success, message } = res.data;
      if (success) {
        Toast.success({ content: t('补单成功') });
        await loadTopups(page, pageSize);
      } else {
        Toast.error({ content: message || t('补单失败') });
      }
    } catch (e) {
      Toast.error({ content: t('补单失败') });
    }
  };

  const confirmAdminComplete = (tradeNo) => {
    Modal.confirm({
      title: t('确认补单'),
      content: t('是否将该订单标记为成功并为用户入账？'),
      onOk: () => handleAdminComplete(tradeNo),
    });
  };

  // 渲染状态徽章
  const renderStatusBadge = (status) => {
    const config = STATUS_CONFIG[status] || { type: 'primary', key: status };
    return (
      <span className='flex items-center gap-2'>
        <Badge dot type={config.type} />
        <span>{t(config.key)}</span>
      </span>
    );
  };

  // 渲染支付方式
  const renderPaymentMethod = (pm) => {
    const displayName = PAYMENT_METHOD_MAP[pm];
    return <Text>{displayName ? t(displayName) : pm || '-'}</Text>;
  };

  const isSubscriptionTopup = (record) => {
    const tradeNo = (record?.trade_no || '').toLowerCase();
    return Number(record?.amount || 0) === 0 && tradeNo.startsWith('sub');
  };

  const columns = useMemo(() => {
    const baseColumns = [
      {
        title: t('订单号'),
        dataIndex: 'trade_no',
        key: 'trade_no',
        render: (text) => <Text copyable>{text}</Text>,
      },
      // 管理员与普通用户均展示用户名列；普通用户仅能看到自己的订单，接口仍返回 username 便于统一展示
      {
        title: t('用户名'),
        dataIndex: 'username',
        key: 'username',
        render: (name) => <Text>{name || '-'}</Text>,
      },
      {
        title: t('支付方式'),
        dataIndex: 'payment_method',
        key: 'payment_method',
        render: renderPaymentMethod,
      },
      {
        title: t('充值额度'),
        dataIndex: 'amount',
        key: 'amount',
        render: (amount, record) => {
          if (isSubscriptionTopup(record)) {
            return (
              <Tag color='purple' shape='circle' size='small'>
                {t('订阅套餐')}
              </Tag>
            );
          }
          return (
            <span className='flex items-center gap-1'>
              <Coins size={16} />
              <Text>{amount}</Text>
            </span>
          );
        },
      },
      {
        title: t('支付金额'),
        dataIndex: 'money',
        key: 'money',
        render: (money) => <Text type='danger'>¥{money.toFixed(2)}</Text>,
      },
      {
        title: t('支付状态'),
        dataIndex: 'status',
        key: 'status',
        render: renderStatusBadge,
      },
    ];

    // 管理员才显示操作列
    if (userIsAdmin) {
      baseColumns.push({
        title: t('操作'),
        key: 'action',
        render: (_, record) => {
          const actions = [];
          if (record.status === 'pending') {
            actions.push(
              <Button
                key='complete'
                size='small'
                type='primary'
                theme='outline'
                onClick={() => confirmAdminComplete(record.trade_no)}
              >
                {t('补单')}
              </Button>,
            );
          }
          return actions.length > 0 ? <>{actions}</> : null;
        },
      });
    }

    baseColumns.push({
      title: t('创建时间'),
      dataIndex: 'create_time',
      key: 'create_time',
      render: (time) => timestamp2string(time),
    });

    return baseColumns;
  }, [t, userIsAdmin]);

  return (
    <Modal
      title={t('充值账单')}
      visible={visible}
      onCancel={onCancel}
      footer={null}
      size={isMobile ? 'full-width' : 'large'}
    >
      {/* 筛选条件单行排列；宽度不足时横向滚动，避免折成两行 */}
      <div className='mb-3 flex w-full flex-row flex-nowrap items-center gap-2 overflow-x-auto'>
        {userIsAdmin ? (
          <Input
            className='min-w-[104px] flex-1'
            prefix={<IconSearch />}
            placeholder={t('用户名')}
            value={usernameFilter}
            onChange={handleUsernameChange}
            showClear
          />
        ) : null}
        <Input
          className='min-w-[104px] flex-1'
          prefix={<IconSearch />}
          placeholder={t('订单号')}
          value={tradeNoFilter}
          onChange={handleTradeNoChange}
          showClear
        />
        <div className='shrink-0'>
          <Select
            value={statusFilter}
            onChange={handleStatusFilterChange}
            optionList={statusFilterOptions}
            placeholder={t('支付状态')}
            allowClear
            style={{ width: isMobile ? 132 : 158 }}
          />
        </div>
      </div>
      <Table
        columns={columns}
        dataSource={topups}
        loading={loading}
        rowKey='id'
        pagination={{
          currentPage: page,
          pageSize: pageSize,
          total: total,
          showSizeChanger: true,
          pageSizeOpts: [10, 20, 50, 100],
          onPageChange: handlePageChange,
          onPageSizeChange: handlePageSizeChange,
        }}
        size='small'
        empty={
          <Empty
            image={<IllustrationNoResult style={{ width: 150, height: 150 }} />}
            darkModeImage={
              <IllustrationNoResultDark style={{ width: 150, height: 150 }} />
            }
            description={t('暂无充值记录')}
            style={{ padding: 30 }}
          />
        }
      />
    </Modal>
  );
};

export default TopupHistoryModal;
