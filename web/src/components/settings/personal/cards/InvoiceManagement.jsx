/*
Copyright (C) 2025 QuantumNous
*/

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Banner,
  Button,
  Card,
  Input,
  Modal,
  Radio,
  Table,
  Tabs,
  TabPane,
  Tag,
  Typography,
} from '@douyinfe/semi-ui';
import { API, showError, showSuccess, timestamp2string } from '../../../../helpers';

const { Text } = Typography;

const invoiceStatusTag = (status, t) => {
  const map = {
    pending: { color: 'orange', label: t('待处理') },
    processing: { color: 'blue', label: t('处理中') },
    issued: { color: 'green', label: t('已开具') },
    rejected: { color: 'red', label: t('已驳回') },
    cancelled: { color: 'grey', label: t('已取消') },
  };
  const item = map[status] || { color: 'grey', label: status };
  return <Tag color={item.color}>{item.label}</Tag>;
};

const InvoiceManagement = ({ t }) => {
  const [activeTab, setActiveTab] = useState('eligible');
  const [loading, setLoading] = useState(false);
  const [eligibleOrders, setEligibleOrders] = useState([]);
  const [records, setRecords] = useState([]);
  const [keyword, setKeyword] = useState('');
  const [selectedRowKeys, setSelectedRowKeys] = useState([]);
  const [profileModalVisible, setProfileModalVisible] = useState(false);
  const [profile, setProfile] = useState({
    title_type: 'personal',
    title: '',
    tax_no: '',
    email: '',
    phone: '',
  });

  const loadProfile = useCallback(async () => {
    try {
      const res = await API.get('/api/user/invoice/profile');
      if (res.data.success && res.data.data) {
        setProfile(res.data.data);
      }
    } catch (e) {
      // profile may not exist yet
    }
  }, []);

  const loadEligible = useCallback(async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/user/invoice/eligible-orders', {
        params: { keyword: keyword.trim() || undefined },
      });
      if (res.data.success) {
        setEligibleOrders(res.data.data || []);
      }
    } catch (e) {
      showError(e);
    } finally {
      setLoading(false);
    }
  }, [keyword]);

  const loadRecords = useCallback(async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/user/invoice/requests', { params: { p: 1, page_size: 50 } });
      if (res.data.success) {
        setRecords(res.data.data?.items || []);
      }
    } catch (e) {
      showError(e);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadProfile();
  }, [loadProfile]);

  useEffect(() => {
    if (activeTab === 'eligible') {
      loadEligible();
    } else {
      loadRecords();
    }
  }, [activeTab, loadEligible, loadRecords]);

  const selectedTotal = useMemo(() => {
    const map = new Map(eligibleOrders.map((o) => [o.topup_id, o]));
    return selectedRowKeys.reduce((sum, id) => {
      const row = map.get(id);
      return sum + (row?.invoiceable_amount || 0);
    }, 0);
  }, [eligibleOrders, selectedRowKeys]);

  const saveProfile = async () => {
    try {
      const res = await API.put('/api/user/invoice/profile', profile);
      if (res.data.success) {
        showSuccess(t('保存成功'));
        setProfileModalVisible(false);
        setProfile(res.data.data || profile);
      }
    } catch (e) {
      showError(e);
    }
  };

  const submitInvoice = async (rows) => {
    if (!profile?.title || !profile?.email) {
      showError(t('请先完善开票信息'));
      setProfileModalVisible(true);
      return;
    }
    const items = rows
      .filter((row) => row.invoiceable_amount > 0)
      .map((row) => ({
        topup_id: row.topup_id,
        invoice_amount: row.invoiceable_amount,
      }));
    if (items.length === 0) {
      showError(t('没有可开票金额'));
      return;
    }
    try {
      const res = await API.post('/api/user/invoice/request', { items });
      if (res.data.success) {
        showSuccess(t('开票申请已提交'));
        setSelectedRowKeys([]);
        loadEligible();
        setActiveTab('records');
      }
    } catch (e) {
      showError(e);
    }
  };

  const eligibleColumns = [
    { title: t('订单号'), dataIndex: 'trade_no' },
    {
      title: t('充值金额'),
      dataIndex: 'money',
      render: (v) => `¥${Number(v || 0).toFixed(2)}`,
    },
    {
      title: t('已消耗'),
      dataIndex: 'consumed_amount',
      render: (v) => `¥${Number(v || 0).toFixed(2)}`,
    },
    {
      title: t('可开票金额'),
      dataIndex: 'invoiceable_amount',
      render: (v) => `¥${Number(v || 0).toFixed(2)}`,
    },
    {
      title: t('已开票金额'),
      dataIndex: 'invoiced_amount',
      render: (v) => `¥${Number(v || 0).toFixed(2)}`,
    },
    {
      title: t('创建时间'),
      dataIndex: 'create_time',
      render: (v) => timestamp2string(v),
    },
    {
      title: t('操作'),
      render: (_, record) => (
        <Button
          size='small'
          disabled={!(record.invoiceable_amount > 0)}
          onClick={() => submitInvoice([record])}
        >
          {t('开票')}
        </Button>
      ),
    },
  ];

  const recordColumns = [
    { title: t('申请单号'), dataIndex: 'request_no' },
    {
      title: t('金额'),
      dataIndex: 'total_amount',
      render: (v) => `¥${Number(v || 0).toFixed(2)}`,
    },
    {
      title: t('状态'),
      dataIndex: 'status',
      render: (v) => invoiceStatusTag(v, t),
    },
    {
      title: t('创建时间'),
      dataIndex: 'created_at',
      render: (v) => timestamp2string(v),
    },
    {
      title: t('操作'),
      render: (_, record) =>
        record.invoice_url ? (
          <Button
            size='small'
            theme='borderless'
            onClick={() => window.open(record.invoice_url, '_blank')}
          >
            {t('下载')}
          </Button>
        ) : (
          '-'
        ),
    },
  ];

  return (
    <Card className='!rounded-2xl'>
      <div className='flex flex-wrap items-center justify-end gap-3 mb-4'>
        <Button onClick={() => setProfileModalVisible(true)}>{t('开票信息设置')}</Button>
      </div>

      <Banner
        fullMode={false}
        type='info'
        className='!mb-4'
        description={
          <ul className='list-disc pl-5 space-y-1 text-sm'>
            <li>{t('发票将在申请提交后的 3-5 个工作日内开具')}</li>
            <li>{t('电子发票将发送至您的注册邮箱，请注意查收')}</li>
            <li>{t('仅限「已支付」状态的订单可申请发票')}</li>
            <li>{t('按实际消耗金额开票，消耗多少即可申请开票多少')}</li>
            <li>{t('请确保开票信息的准确性，发票一经开出概不退换')}</li>
          </ul>
        }
      />

      <Tabs activeKey={activeTab} onChange={setActiveTab}>
        <TabPane tab={t('待开票')} itemKey='eligible'>
          <div className='flex flex-wrap items-center gap-2 mb-3'>
            <Input
              placeholder={t('搜索订单')}
              value={keyword}
              onChange={setKeyword}
              style={{ width: 220 }}
            />
            <Button onClick={loadEligible}>{t('查询')}</Button>
            <Button
              type='primary'
              disabled={selectedRowKeys.length === 0 || selectedTotal <= 0}
              onClick={() => {
                const rows = eligibleOrders.filter((o) =>
                  selectedRowKeys.includes(o.topup_id),
                );
                submitInvoice(rows);
              }}
            >
              {t('合并开票')}
              {selectedRowKeys.length > 0
                ? ` (${selectedRowKeys.length} / ¥${selectedTotal.toFixed(2)})`
                : ''}
            </Button>
          </div>
          <Table
            rowKey='topup_id'
            loading={loading}
            columns={eligibleColumns}
            dataSource={eligibleOrders}
            pagination={false}
            rowSelection={{
              selectedRowKeys,
              onChange: (keys) => setSelectedRowKeys(keys),
              getCheckboxProps: (record) => ({
                disabled: !(record.invoiceable_amount > 0),
              }),
            }}
          />
        </TabPane>
        <TabPane tab={t('开票记录')} itemKey='records'>
          <Table
            rowKey='id'
            loading={loading}
            columns={recordColumns}
            dataSource={records}
            pagination={false}
          />
        </TabPane>
      </Tabs>

      <Modal
        title={t('开票信息设置')}
        visible={profileModalVisible}
        onCancel={() => setProfileModalVisible(false)}
        onOk={saveProfile}
      >
        <div className='space-y-3'>
          <div>
            <Text className='block mb-1'>{t('抬头类型')}</Text>
            <Radio.Group
              value={profile.title_type}
              onChange={(e) =>
                setProfile((p) => ({ ...p, title_type: e.target.value }))
              }
            >
              <Radio value='personal'>{t('个人')}</Radio>
              <Radio value='company'>{t('企业')}</Radio>
            </Radio.Group>
          </div>
          <div>
            <Text className='block mb-1'>{t('发票抬头')}</Text>
            <Input
              value={profile.title}
              onChange={(v) => setProfile((p) => ({ ...p, title: v }))}
            />
          </div>
          <div>
            <Text className='block mb-1'>{t('税号')}</Text>
            <Input
              value={profile.tax_no}
              onChange={(v) => setProfile((p) => ({ ...p, tax_no: v }))}
            />
          </div>
          <div>
            <Text className='block mb-1'>{t('收票邮箱')}</Text>
            <Input
              value={profile.email}
              onChange={(v) => setProfile((p) => ({ ...p, email: v }))}
            />
          </div>
          <div>
            <Text className='block mb-1'>{t('联系电话')}</Text>
            <Input
              value={profile.phone}
              onChange={(v) => setProfile((p) => ({ ...p, phone: v }))}
            />
          </div>
        </div>
      </Modal>
    </Card>
  );
};

export default InvoiceManagement;
