/*
Copyright (C) 2025 QuantumNous
*/

import React, { useCallback, useEffect, useState } from 'react';
import {
  Button,
  Input,
  Modal,
  Select,
  Table,
  Tag,
  Typography,
  Descriptions,
} from '@douyinfe/semi-ui';
import { Receipt } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import CardPro from '../../../components/common/ui/CardPro';
import InvoiceFileUpload from '../../../components/settings/InvoiceFileUpload';
import { API, showError, showSuccess, timestamp2string } from '../../../helpers';

const { Text } = Typography;

const statusOptions = [
  { value: '', label: '全部' },
  { value: 'pending', label: '待处理' },
  { value: 'processing', label: '处理中' },
  { value: 'issued', label: '已开具' },
  { value: 'rejected', label: '已驳回' },
  { value: 'cancelled', label: '已取消' },
];

const statusTag = (status) => {
  const map = {
    pending: { color: 'orange', label: '待处理' },
    processing: { color: 'blue', label: '处理中' },
    issued: { color: 'green', label: '已开具' },
    rejected: { color: 'red', label: '已驳回' },
    cancelled: { color: 'grey', label: '已取消' },
  };
  const item = map[status] || { color: 'grey', label: status };
  return <Tag color={item.color}>{item.label}</Tag>;
};

const SettingsInvoiceAdmin = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [items, setItems] = useState([]);
  const [statusFilter, setStatusFilter] = useState('pending');
  const [issueModal, setIssueModal] = useState({ visible: false, record: null });
  const [rejectModal, setRejectModal] = useState({ visible: false, record: null });
  const [detailModal, setDetailModal] = useState({ visible: false, data: null });
  const [issueForm, setIssueForm] = useState({ invoice_code: '', invoice_url: '', admin_note: '' });
  const [issueRecipientEmail, setIssueRecipientEmail] = useState('');
  const [rejectNote, setRejectNote] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const loadList = useCallback(async () => {
    setLoading(true);
    try {
      const res = await API.get('/api/user/invoice/admin/requests', {
        params: { p: 1, page_size: 100, status: statusFilter || undefined },
      });
      if (res.data.success) {
        setItems(res.data.data?.items || []);
      }
    } catch (e) {
      showError(e);
    } finally {
      setLoading(false);
    }
  }, [statusFilter]);

  useEffect(() => {
    loadList();
  }, [loadList]);

  const openDetail = async (record) => {
    try {
      const res = await API.get(`/api/user/invoice/admin/requests/${record.id}`);
      if (res.data.success) {
        setDetailModal({ visible: true, data: res.data.data });
      }
    } catch (e) {
      showError(e);
    }
  };

  const openIssueModal = async (record) => {
    setIssueModal({ visible: true, record });
    setIssueForm({ invoice_code: '', invoice_url: '', admin_note: '' });
    setIssueRecipientEmail('');
    try {
      const res = await API.get(`/api/user/invoice/admin/requests/${record.id}`);
      if (res.data.success) {
        const snapshot = res.data.data?.request?.profile_snapshot;
        if (snapshot) {
          try {
            const profile = JSON.parse(snapshot);
            setIssueRecipientEmail(profile?.email || res.data.data?.email || '');
          } catch {
            setIssueRecipientEmail(res.data.data?.email || '');
          }
        } else {
          setIssueRecipientEmail(res.data.data?.email || '');
        }
      }
    } catch {
      setIssueRecipientEmail(record.email || '');
    }
  };

  const submitIssue = async () => {
    if (!issueModal.record) return;
    if (!issueForm.invoice_url?.trim()) {
      showError(t('请先上传电子发票 PDF'));
      return;
    }
    setSubmitting(true);
    try {
      const res = await API.post(
        `/api/user/invoice/admin/requests/${issueModal.record.id}/issue`,
        issueForm,
      );
      if (res.data.success) {
        showSuccess(t('发票已开具'));
        setIssueModal({ visible: false, record: null });
        setIssueForm({ invoice_code: '', invoice_url: '', admin_note: '' });
        setIssueRecipientEmail('');
        loadList();
      }
    } catch (e) {
      showError(e);
    } finally {
      setSubmitting(false);
    }
  };

  const submitReject = async () => {
    if (!rejectModal.record) return;
    setSubmitting(true);
    try {
      const res = await API.post(
        `/api/user/invoice/admin/requests/${rejectModal.record.id}/reject`,
        { admin_note: rejectNote },
      );
      if (res.data.success) {
        showSuccess(t('已驳回'));
        setRejectModal({ visible: false, record: null });
        setRejectNote('');
        loadList();
      }
    } catch (e) {
      showError(e);
    } finally {
      setSubmitting(false);
    }
  };

  const columns = [
    { title: t('申请单号'), dataIndex: 'request_no' },
    { title: t('用户'), dataIndex: 'username' },
    {
      title: t('金额'),
      dataIndex: 'total_amount',
      render: (v) => `¥${Number(v || 0).toFixed(2)}`,
    },
    {
      title: t('状态'),
      dataIndex: 'status',
      render: (v) => statusTag(v),
    },
    {
      title: t('创建时间'),
      dataIndex: 'created_at',
      render: (v) => timestamp2string(v),
    },
    {
      title: t('操作'),
      render: (_, record) => (
        <div className='flex flex-wrap gap-2'>
          <Button size='small' onClick={() => openDetail(record)}>
            {t('详情')}
          </Button>
          {record.status === 'pending' || record.status === 'processing' ? (
            <>
              <Button
                size='small'
                type='primary'
                onClick={() => openIssueModal(record)}
              >
                {t('开具')}
              </Button>
              <Button
                size='small'
                type='danger'
                onClick={() => {
                  setRejectModal({ visible: true, record });
                  setRejectNote('');
                }}
              >
                {t('驳回')}
              </Button>
            </>
          ) : null}
          {record.invoice_url ? (
            <Button
              size='small'
              theme='borderless'
              onClick={() => window.open(record.invoice_url, '_blank')}
            >
              {t('下载')}
            </Button>
          ) : null}
        </div>
      ),
    },
  ];

  const profile = (() => {
    try {
      return detailModal.data?.request?.profile_snapshot
        ? JSON.parse(detailModal.data.request.profile_snapshot)
        : null;
    } catch {
      return null;
    }
  })();

  return (
    <>
      <CardPro
        type='type1'
        t={t}
        descriptionArea={
          <div className='flex items-center text-violet-500'>
            <Receipt size={16} className='mr-2' />
            <Text>{t('发票审批')}</Text>
          </div>
        }
        actionsArea={
          <div className='flex flex-wrap items-center justify-end gap-2 w-full'>
            <Select
              value={statusFilter}
              onChange={setStatusFilter}
              style={{ width: 140 }}
              optionList={statusOptions.map((o) => ({ value: o.value, label: t(o.label) }))}
            />
            <Button onClick={loadList}>{t('刷新')}</Button>
          </div>
        }
      >
        <Table rowKey='id' loading={loading} columns={columns} dataSource={items} pagination={false} />
      </CardPro>

      <Modal
        title={t('开具发票')}
        visible={issueModal.visible}
        onCancel={() => setIssueModal({ visible: false, record: null })}
        onOk={submitIssue}
        confirmLoading={submitting}
      >
        <div className='space-y-3'>
          <Text type='secondary'>
            {t('申请单号')}: {issueModal.record?.request_no}
          </Text>
          {issueRecipientEmail ? (
            <Text type='secondary'>
              {t('收票邮箱')}: {issueRecipientEmail}
            </Text>
          ) : null}
          <Input
            placeholder={t('发票号码')}
            value={issueForm.invoice_code}
            onChange={(v) => setIssueForm((f) => ({ ...f, invoice_code: v }))}
          />
          <InvoiceFileUpload
            url={issueForm.invoice_url}
            onUrlChange={(invoice_url) => setIssueForm((f) => ({ ...f, invoice_url }))}
            disabled={submitting}
          />
          <Input
            placeholder={t('备注')}
            value={issueForm.admin_note}
            onChange={(v) => setIssueForm((f) => ({ ...f, admin_note: v }))}
          />
        </div>
      </Modal>

      <Modal
        title={t('驳回申请')}
        visible={rejectModal.visible}
        onCancel={() => setRejectModal({ visible: false, record: null })}
        onOk={submitReject}
        confirmLoading={submitting}
      >
        <Input
          placeholder={t('驳回原因')}
          value={rejectNote}
          onChange={setRejectNote}
        />
      </Modal>

      <Modal
        title={t('申请详情')}
        visible={detailModal.visible}
        onCancel={() => setDetailModal({ visible: false, data: null })}
        footer={<Button onClick={() => setDetailModal({ visible: false, data: null })}>{t('关闭')}</Button>}
        width={720}
      >
        {detailModal.data ? (
          <div className='space-y-4'>
            <Descriptions align='left'>
              <Descriptions.Item itemKey={t('申请单号')}>
                {detailModal.data.request?.request_no}
              </Descriptions.Item>
              <Descriptions.Item itemKey={t('用户')}>
                {detailModal.data.username} ({detailModal.data.email})
              </Descriptions.Item>
              <Descriptions.Item itemKey={t('金额')}>
                ¥{Number(detailModal.data.request?.total_amount || 0).toFixed(2)}
              </Descriptions.Item>
              <Descriptions.Item itemKey={t('状态')}>
                {statusTag(detailModal.data.request?.status)}
              </Descriptions.Item>
            </Descriptions>
            {profile ? (
              <Descriptions align='left' title={t('开票信息')}>
                <Descriptions.Item itemKey={t('发票抬头')}>{profile.title}</Descriptions.Item>
                <Descriptions.Item itemKey={t('税号')}>{profile.tax_no || '-'}</Descriptions.Item>
                <Descriptions.Item itemKey={t('收票邮箱')}>{profile.email}</Descriptions.Item>
              </Descriptions>
            ) : null}
            <Table
              size='small'
              pagination={false}
              dataSource={detailModal.data.items || []}
              columns={[
                { title: t('订单号'), dataIndex: 'trade_no' },
                {
                  title: t('开票金额'),
                  dataIndex: 'invoice_amount',
                  render: (v) => `¥${Number(v || 0).toFixed(2)}`,
                },
              ]}
            />
          </div>
        ) : null}
      </Modal>
    </>
  );
};

export default SettingsInvoiceAdmin;
