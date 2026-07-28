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
  Download,
  Eye,
  FileText,
  RefreshCw,
  Search,
  Send,
  Settings2,
  WalletCards,
  XCircle,
} from 'lucide-react';
import {
  API,
  showError,
  showSuccess,
  timestamp2string,
} from '../../../../helpers';
import {
  InvoiceDialog,
  InvoiceEmptyState,
  InvoicePagination,
  InvoiceSpinner,
  InvoiceStatusBadge,
  formatInvoiceMoney,
  parseInvoiceProfile,
} from '../../../invoice/InvoiceWorkspace';

const RECORD_PAGE_SIZE = 10;
const InvoiceManagement = ({ t }) => {
  const [activeTab, setActiveTab] = useState('eligible');
  const [eligibleLoading, setEligibleLoading] = useState(false);
  const [recordsLoading, setRecordsLoading] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [eligibleOrders, setEligibleOrders] = useState([]);
  const [records, setRecords] = useState([]);
  const [recordsTotal, setRecordsTotal] = useState(0);
  const [recordsPage, setRecordsPage] = useState(1);
  const [searchDraft, setSearchDraft] = useState('');
  const [searchKeyword, setSearchKeyword] = useState('');
  const [selectedRowKeys, setSelectedRowKeys] = useState([]);
  const [profileOpen, setProfileOpen] = useState(false);
  const [requestOpen, setRequestOpen] = useState(false);
  const [detailOpen, setDetailOpen] = useState(false);
  const [cancelTarget, setCancelTarget] = useState(null);
  const [requestRows, setRequestRows] = useState([]);
  const [requestAmounts, setRequestAmounts] = useState({});
  const [detail, setDetail] = useState(null);
  const [profile, setProfile] = useState({
    title_type: 'personal',
    title: '',
    tax_no: '',
    email: '',
    phone: '',
  });
  const [balanceSummary, setBalanceSummary] = useState(null);

  const loadBalanceSummary = useCallback(async () => {
    try {
      const res = await API.get('/api/user/invoice/balance-summary');
      if (res.data.success) setBalanceSummary(res.data.data || null);
    } catch {
      // The summary is helpful but should not block the order list.
    }
  }, []);

  const loadProfile = useCallback(async () => {
    try {
      const res = await API.get('/api/user/invoice/profile');
      if (res.data.success && res.data.data) setProfile(res.data.data);
    } catch {
      // New users do not have an invoice profile yet.
    }
  }, []);

  const loadEligible = useCallback(async () => {
    setEligibleLoading(true);
    try {
      const res = await API.get('/api/user/invoice/eligible-orders', {
        params: { keyword: searchKeyword || undefined },
      });
      if (res.data.success) setEligibleOrders(res.data.data || []);
    } catch (error) {
      showError(error);
    } finally {
      setEligibleLoading(false);
    }
  }, [searchKeyword]);

  const loadRecords = useCallback(async (page = 1) => {
    setRecordsLoading(true);
    try {
      const res = await API.get('/api/user/invoice/requests', {
        params: { p: page, page_size: RECORD_PAGE_SIZE },
      });
      if (res.data.success) {
        setRecords(res.data.data?.items || []);
        setRecordsTotal(Number(res.data.data?.total || 0));
        setRecordsPage(page);
      }
    } catch (error) {
      showError(error);
    } finally {
      setRecordsLoading(false);
    }
  }, []);

  useEffect(() => {
    loadProfile();
    loadBalanceSummary();
  }, [loadBalanceSummary, loadProfile]);

  useEffect(() => {
    if (activeTab === 'eligible') loadEligible();
    else loadRecords(recordsPage);
  }, [activeTab, loadEligible, loadRecords, recordsPage]);

  const selectedRows = useMemo(
    () =>
      eligibleOrders.filter((order) =>
        selectedRowKeys.includes(order.topup_id),
      ),
    [eligibleOrders, selectedRowKeys],
  );

  const selectedTotal = useMemo(
    () =>
      selectedRows.reduce(
        (sum, row) => sum + Number(row.invoiceable_amount || 0),
        0,
      ),
    [selectedRows],
  );

  const requestTotal = useMemo(
    () =>
      requestRows.reduce(
        (sum, row) =>
          sum + Math.max(0, Number(requestAmounts[row.topup_id] || 0)),
        0,
      ),
    [requestAmounts, requestRows],
  );

  const validateProfile = () => {
    if (!profile.title?.trim() || !profile.email?.trim()) {
      showError(t('请先完善开票信息'));
      return false;
    }
    if (profile.title_type === 'company' && !profile.tax_no?.trim()) {
      showError(t('企业开票请填写税号'));
      return false;
    }
    return true;
  };

  const saveProfile = async () => {
    if (!validateProfile()) return;
    setSubmitting(true);
    try {
      const res = await API.put('/api/user/invoice/profile', profile);
      if (res.data.success) {
        setProfile(res.data.data || profile);
        setProfileOpen(false);
        showSuccess(t('保存成功'));
      }
    } catch (error) {
      showError(error);
    } finally {
      setSubmitting(false);
    }
  };

  const openRequest = (rows) => {
    if (!validateProfile()) {
      setProfileOpen(true);
      return;
    }
    const validRows = rows.filter(
      (row) => Number(row.invoiceable_amount || 0) > 0,
    );
    if (!validRows.length) {
      showError(t('没有可开票金额'));
      return;
    }
    setRequestRows(validRows);
    setRequestAmounts(
      Object.fromEntries(
        validRows.map((row) => [
          row.topup_id,
          Number(row.invoiceable_amount).toFixed(2),
        ]),
      ),
    );
    setRequestOpen(true);
  };

  const submitInvoice = async () => {
    const items = requestRows.map((row) => ({
      topup_id: row.topup_id,
      invoice_amount: Number(requestAmounts[row.topup_id] || 0),
    }));
    const invalid = items.some((item, index) => {
      const maximum = Number(requestRows[index].invoiceable_amount || 0);
      return (
        !Number.isFinite(item.invoice_amount) ||
        item.invoice_amount <= 0 ||
        item.invoice_amount > maximum + 0.000001
      );
    });
    if (invalid) {
      showError(t('开票金额必须大于 0 且不能超过可开票金额'));
      return;
    }
    setSubmitting(true);
    try {
      const res = await API.post('/api/user/invoice/request', { items });
      if (res.data.success) {
        showSuccess(t('开票申请已提交'));
        setRequestOpen(false);
        setSelectedRowKeys([]);
        await Promise.all([loadEligible(), loadBalanceSummary()]);
        setRecordsPage(1);
        setActiveTab('records');
      }
    } catch (error) {
      showError(error);
    } finally {
      setSubmitting(false);
    }
  };

  const openDetail = async (record) => {
    setDetailOpen(true);
    setDetail(null);
    setDetailLoading(true);
    try {
      const res = await API.get(`/api/user/invoice/requests/${record.id}`);
      if (res.data.success) setDetail(res.data.data || null);
    } catch (error) {
      showError(error);
      setDetailOpen(false);
    } finally {
      setDetailLoading(false);
    }
  };

  const cancelRequest = async () => {
    if (!cancelTarget) return;
    setSubmitting(true);
    try {
      const res = await API.post(
        `/api/user/invoice/requests/${cancelTarget.id}/cancel`,
      );
      if (res.data.success) {
        showSuccess(t('申请已取消'));
        setCancelTarget(null);
        await Promise.all([
          loadRecords(recordsPage),
          loadEligible(),
          loadBalanceSummary(),
        ]);
      }
    } catch (error) {
      showError(error);
    } finally {
      setSubmitting(false);
    }
  };

  const toggleSelected = (topUpID, checked) => {
    setSelectedRowKeys((current) =>
      checked
        ? [...new Set([...current, topUpID])]
        : current.filter((id) => id !== topUpID),
    );
  };

  const profileSnapshot = parseInvoiceProfile(
    detail?.request?.profile_snapshot,
  );

  return (
    <div className='invoice-workspace'>
      <section className='invoice-glass-panel invoice-notice-panel is-user'>
        <div className='invoice-notice-content'>
          <div className='invoice-notice-copy'>
            <h2>{t('开票须知')}</h2>
            <ul className='invoice-notice-list'>
              <li>{t('发票将在申请提交后的 3-5 个工作日内开具')}</li>
              <li>{t('充值成功后即可申请开票，无需等待额度消耗')}</li>
              <li>{t('电子发票将发送至收票邮箱')}</li>
            </ul>
          </div>
          <div className='invoice-notice-metrics' aria-label={t('发票概览')}>
            <div>
              <span>{t('申请中金额')}</span>
              <strong>
                {formatInvoiceMoney(balanceSummary?.pending_invoice_amount)}
              </strong>
            </div>
            <div>
              <span>{t('已开票金额')}</span>
              <strong>
                {formatInvoiceMoney(balanceSummary?.invoiced_amount)}
              </strong>
            </div>
            <div className='is-highlight'>
              <span>{t('可开票金额')}</span>
              <strong>
                {formatInvoiceMoney(balanceSummary?.invoiceable_amount)}
              </strong>
            </div>
          </div>
        </div>
      </section>

      <section className='invoice-glass-panel invoice-toolbar-panel'>
        <header className='invoice-toolbar'>
          <div className='invoice-toolbar-group'>
            <div
              className='invoice-tabs'
              role='tablist'
              aria-label={t('发票视图')}
            >
              <button
                type='button'
                role='tab'
                aria-selected={activeTab === 'eligible'}
                className={activeTab === 'eligible' ? 'is-active' : ''}
                onClick={() => setActiveTab('eligible')}
              >
                {t('待开票')}
              </button>
              <button
                type='button'
                role='tab'
                aria-selected={activeTab === 'records'}
                className={activeTab === 'records' ? 'is-active' : ''}
                onClick={() => setActiveTab('records')}
              >
                {t('开票记录')}
              </button>
            </div>
          </div>
          <div className='invoice-toolbar-group is-end'>
            {activeTab === 'eligible' ? (
              <form
                className='invoice-search-form'
                onSubmit={(event) => {
                  event.preventDefault();
                  setSearchKeyword(searchDraft.trim());
                }}
              >
                <div className='invoice-search'>
                  <Search size={16} aria-hidden='true' />
                  <input
                    className='invoice-input'
                    name='invoice-order-search'
                    type='search'
                    value={searchDraft}
                    onChange={(event) => setSearchDraft(event.target.value)}
                    placeholder={`${t('搜索订单')}…`}
                    aria-label={t('搜索订单')}
                    autoComplete='off'
                    spellCheck={false}
                  />
                </div>
                <button type='submit' className='invoice-button is-search'>
                  {t('查询')}
                </button>
              </form>
            ) : null}
            <div className='invoice-command-group'>
              <button
                type='button'
                className='invoice-icon-button'
                onClick={() =>
                  activeTab === 'eligible'
                    ? loadEligible()
                    : loadRecords(recordsPage)
                }
                title={t('刷新')}
                aria-label={t('刷新')}
              >
                <RefreshCw size={17} />
              </button>
              <button
                type='button'
                className='invoice-button'
                onClick={() => setProfileOpen(true)}
              >
                <Settings2 size={16} />
                {t('开票信息')}
              </button>
              {activeTab === 'eligible' ? (
                <button
                  type='button'
                  className='invoice-button is-primary'
                  disabled={!selectedRows.length || selectedTotal <= 0}
                  onClick={() => openRequest(selectedRows)}
                >
                  <Send size={16} />
                  {t('合并开票')}{' '}
                  {selectedRows.length ? `(${selectedRows.length})` : ''}
                </button>
              ) : null}
            </div>
          </div>
        </header>

        {activeTab === 'eligible' ? (
          <div className='invoice-table-wrap' role='tabpanel'>
            <table className='invoice-table'>
              <thead>
                <tr>
                  <th aria-label={t('选择')} />
                  <th>{t('订单号')}</th>
                  <th>{t('充值金额')}</th>
                  <th>{t('已开票金额')}</th>
                  <th>{t('申请中金额')}</th>
                  <th>{t('可开票金额')}</th>
                  <th>{t('创建时间')}</th>
                  <th>{t('操作')}</th>
                </tr>
              </thead>
              <tbody>
                {eligibleLoading ? (
                  <tr>
                    <td className='invoice-loading-row' colSpan={8}>
                      <InvoiceSpinner label={t('正在加载...')} />
                    </td>
                  </tr>
                ) : eligibleOrders.length ? (
                  eligibleOrders.map((record) => {
                    const disabled = !(
                      Number(record.invoiceable_amount || 0) > 0
                    );
                    return (
                      <tr key={record.topup_id}>
                        <td className='is-select-cell' data-label={t('选择')}>
                          <input
                            className='invoice-checkbox'
                            type='checkbox'
                            checked={selectedRowKeys.includes(record.topup_id)}
                            disabled={disabled}
                            onChange={(event) =>
                              toggleSelected(
                                record.topup_id,
                                event.target.checked,
                              )
                            }
                            aria-label={`${t('选择')} ${record.trade_no}`}
                          />
                        </td>
                        <td className='is-strong' data-label={t('订单号')}>
                          {record.trade_no}
                        </td>
                        <td className='is-numeric' data-label={t('充值金额')}>
                          {formatInvoiceMoney(record.money)}
                        </td>
                        <td className='is-numeric' data-label={t('已开票金额')}>
                          {formatInvoiceMoney(record.invoiced_amount)}
                        </td>
                        <td className='is-numeric' data-label={t('申请中金额')}>
                          {formatInvoiceMoney(record.pending_amount)}
                        </td>
                        <td
                          className='is-numeric is-strong'
                          data-label={t('可开票金额')}
                        >
                          {formatInvoiceMoney(record.invoiceable_amount)}
                        </td>
                        <td data-label={t('创建时间')}>
                          {timestamp2string(record.create_time)}
                        </td>
                        <td className='is-action-cell' data-label={t('操作')}>
                          <button
                            type='button'
                            className='invoice-button is-quiet'
                            disabled={disabled}
                            onClick={() => openRequest([record])}
                          >
                            <FileText size={15} />
                            {t('开票')}
                          </button>
                        </td>
                      </tr>
                    );
                  })
                ) : (
                  <tr>
                    <td className='invoice-empty-row' colSpan={8}>
                      <InvoiceEmptyState
                        title={t('暂无可开票订单')}
                        description={t('充值成功后，可开票金额会显示在这里')}
                      />
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        ) : (
          <div className='invoice-table-wrap' role='tabpanel'>
            <table className='invoice-table'>
              <thead>
                <tr>
                  <th>{t('申请单号')}</th>
                  <th>{t('金额')}</th>
                  <th>{t('状态')}</th>
                  <th>{t('创建时间')}</th>
                  <th>{t('操作')}</th>
                </tr>
              </thead>
              <tbody>
                {recordsLoading ? (
                  <tr>
                    <td className='invoice-loading-row' colSpan={5}>
                      <InvoiceSpinner label={t('正在加载...')} />
                    </td>
                  </tr>
                ) : records.length ? (
                  records.map((record) => (
                    <tr key={record.id}>
                      <td className='is-strong' data-label={t('申请单号')}>
                        {record.request_no}
                      </td>
                      <td
                        className='is-numeric is-strong'
                        data-label={t('金额')}
                      >
                        {formatInvoiceMoney(record.total_amount)}
                      </td>
                      <td data-label={t('状态')}>
                        <InvoiceStatusBadge status={record.status} t={t} />
                      </td>
                      <td data-label={t('创建时间')}>
                        {timestamp2string(record.created_at)}
                      </td>
                      <td className='is-action-cell' data-label={t('操作')}>
                        <div className='invoice-action-row'>
                          <button
                            type='button'
                            className='invoice-button is-quiet'
                            onClick={() => openDetail(record)}
                          >
                            <Eye size={15} />
                            {t('详情')}
                          </button>
                          {record.invoice_url ? (
                            <button
                              type='button'
                              className='invoice-button is-quiet'
                              onClick={() =>
                                window.open(
                                  record.invoice_url,
                                  '_blank',
                                  'noopener,noreferrer',
                                )
                              }
                            >
                              <Download size={15} />
                              {t('下载')}
                            </button>
                          ) : null}
                          {record.status === 'pending' ? (
                            <button
                              type='button'
                              className='invoice-button is-quiet'
                              onClick={() => setCancelTarget(record)}
                            >
                              <XCircle size={15} />
                              {t('取消申请')}
                            </button>
                          ) : null}
                        </div>
                      </td>
                    </tr>
                  ))
                ) : (
                  <tr>
                    <td className='invoice-empty-row' colSpan={5}>
                      <InvoiceEmptyState
                        title={t('暂无开票记录')}
                        description={t('提交开票申请后，可在这里查看处理进度')}
                      />
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
            <InvoicePagination
              page={recordsPage}
              total={recordsTotal}
              pageSize={RECORD_PAGE_SIZE}
              onChange={loadRecords}
              t={t}
            />
          </div>
        )}
      </section>

      <InvoiceDialog
        open={profileOpen}
        onClose={() => setProfileOpen(false)}
        eyebrow={t('发票资料')}
        title={t('开票信息设置')}
        footer={
          <>
            <button
              type='button'
              className='invoice-button'
              onClick={() => setProfileOpen(false)}
            >
              {t('取消')}
            </button>
            <button
              type='button'
              className='invoice-button is-primary'
              onClick={saveProfile}
              disabled={submitting}
            >
              {submitting ? <InvoiceSpinner /> : null}
              {t('保存')}
            </button>
          </>
        }
      >
        <div className='invoice-form-grid'>
          <div className='invoice-field is-full'>
            <span>{t('抬头类型')}</span>
            <div className='invoice-radio-group'>
              {[
                ['personal', '个人'],
                ['company', '企业'],
              ].map(([value, label]) => (
                <label className='invoice-radio-option' key={value}>
                  <input
                    type='radio'
                    name='invoice-title-type'
                    value={value}
                    checked={profile.title_type === value}
                    onChange={(event) =>
                      setProfile((current) => ({
                        ...current,
                        title_type: event.target.value,
                      }))
                    }
                  />
                  {t(label)}
                </label>
              ))}
            </div>
          </div>
          <div className='invoice-field is-full'>
            <label htmlFor='invoice-profile-title'>{t('发票抬头')}</label>
            <input
              id='invoice-profile-title'
              name='invoice-profile-title'
              className='invoice-input'
              value={profile.title || ''}
              autoComplete='organization'
              onChange={(event) =>
                setProfile((current) => ({
                  ...current,
                  title: event.target.value,
                }))
              }
            />
          </div>
          <div className='invoice-field'>
            <label htmlFor='invoice-profile-tax'>{t('税号')}</label>
            <input
              id='invoice-profile-tax'
              name='invoice-profile-tax'
              className='invoice-input'
              value={profile.tax_no || ''}
              autoComplete='off'
              spellCheck={false}
              onChange={(event) =>
                setProfile((current) => ({
                  ...current,
                  tax_no: event.target.value,
                }))
              }
            />
            {profile.title_type === 'company' ? (
              <small>{t('企业抬头必填')}</small>
            ) : null}
          </div>
          <div className='invoice-field'>
            <label htmlFor='invoice-profile-phone'>{t('联系电话')}</label>
            <input
              id='invoice-profile-phone'
              name='invoice-profile-phone'
              type='tel'
              className='invoice-input'
              value={profile.phone || ''}
              autoComplete='tel'
              onChange={(event) =>
                setProfile((current) => ({
                  ...current,
                  phone: event.target.value,
                }))
              }
            />
          </div>
          <div className='invoice-field is-full'>
            <label htmlFor='invoice-profile-email'>{t('收票邮箱')}</label>
            <input
              id='invoice-profile-email'
              name='invoice-profile-email'
              type='email'
              className='invoice-input'
              value={profile.email || ''}
              autoComplete='email'
              spellCheck={false}
              onChange={(event) =>
                setProfile((current) => ({
                  ...current,
                  email: event.target.value,
                }))
              }
            />
          </div>
        </div>
      </InvoiceDialog>

      <InvoiceDialog
        open={requestOpen}
        onClose={() => setRequestOpen(false)}
        eyebrow={requestRows.length > 1 ? t('合并开票') : t('部分开票')}
        title={t('确认开票金额')}
        size='large'
        footer={
          <>
            <span className='invoice-inline-meta'>
              {t('本次申请合计')}{' '}
              <strong>{formatInvoiceMoney(requestTotal)}</strong>
            </span>
            <button
              type='button'
              className='invoice-button'
              onClick={() => setRequestOpen(false)}
            >
              {t('取消')}
            </button>
            <button
              type='button'
              className='invoice-button is-primary'
              onClick={submitInvoice}
              disabled={submitting || requestTotal <= 0}
            >
              {submitting ? <InvoiceSpinner /> : <Send size={16} />}
              {t('确认提交')}
            </button>
          </>
        }
      >
        <div className='invoice-amount-editor'>
          {requestRows.map((row) => (
            <div className='invoice-amount-row' key={row.topup_id}>
              <div>
                <strong>{row.trade_no}</strong>
                <span>
                  {t('最多可开')} {formatInvoiceMoney(row.invoiceable_amount)}
                </span>
              </div>
              <input
                className='invoice-input'
                name={`invoice-amount-${row.topup_id}`}
                type='number'
                inputMode='decimal'
                min='0.01'
                max={Number(row.invoiceable_amount || 0)}
                step='0.01'
                value={requestAmounts[row.topup_id] ?? ''}
                aria-label={`${t('开票金额')} ${row.trade_no}`}
                onChange={(event) =>
                  setRequestAmounts((current) => ({
                    ...current,
                    [row.topup_id]: event.target.value,
                  }))
                }
              />
            </div>
          ))}
        </div>
      </InvoiceDialog>

      <InvoiceDialog
        open={detailOpen}
        onClose={() => setDetailOpen(false)}
        eyebrow={t('开票记录')}
        title={t('申请详情')}
        size='large'
      >
        {detailLoading ? (
          <InvoiceEmptyState
            title={t('正在加载...')}
            action={<InvoiceSpinner />}
          />
        ) : detail ? (
          <>
            <div className='invoice-detail-grid'>
              <div className='invoice-detail-item'>
                <span>{t('申请单号')}</span>
                <strong>{detail.request?.request_no}</strong>
              </div>
              <div className='invoice-detail-item'>
                <span>{t('金额')}</span>
                <strong>
                  {formatInvoiceMoney(detail.request?.total_amount)}
                </strong>
              </div>
              <div className='invoice-detail-item'>
                <span>{t('状态')}</span>
                <strong>
                  <InvoiceStatusBadge status={detail.request?.status} t={t} />
                </strong>
              </div>
              <div className='invoice-detail-item'>
                <span>{t('发票抬头')}</span>
                <strong>{profileSnapshot?.title || '-'}</strong>
              </div>
              <div className='invoice-detail-item'>
                <span>{t('税号')}</span>
                <strong>{profileSnapshot?.tax_no || '-'}</strong>
              </div>
              <div className='invoice-detail-item'>
                <span>{t('收票邮箱')}</span>
                <strong>{profileSnapshot?.email || '-'}</strong>
              </div>
              <div className='invoice-detail-item'>
                <span>{t('发票号码')}</span>
                <strong>{detail.request?.invoice_code || '-'}</strong>
              </div>
              <div className='invoice-detail-item'>
                <span>{t('开具时间')}</span>
                <strong>
                  {detail.request?.issued_at
                    ? timestamp2string(detail.request.issued_at)
                    : '-'}
                </strong>
              </div>
              <div className='invoice-detail-item'>
                <span>{t('审核备注')}</span>
                <strong>{detail.request?.admin_note || '-'}</strong>
              </div>
            </div>
            <div className='invoice-section-title'>
              <h3>{t('订单明细')}</h3>
            </div>
            <div className='invoice-table-wrap'>
              <table className='invoice-table'>
                <thead>
                  <tr>
                    <th>{t('订单号')}</th>
                    <th>{t('开票金额')}</th>
                  </tr>
                </thead>
                <tbody>
                  {(detail.items || []).map((item) => (
                    <tr key={item.id}>
                      <td className='is-strong' data-label={t('订单号')}>
                        {item.trade_no}
                      </td>
                      <td className='is-numeric' data-label={t('开票金额')}>
                        {formatInvoiceMoney(item.invoice_amount)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </>
        ) : null}
      </InvoiceDialog>

      <InvoiceDialog
        open={Boolean(cancelTarget)}
        onClose={() => setCancelTarget(null)}
        eyebrow={t('取消申请')}
        title={t('确认取消开票申请')}
        size='small'
        footer={
          <>
            <button
              type='button'
              className='invoice-button'
              onClick={() => setCancelTarget(null)}
            >
              {t('返回')}
            </button>
            <button
              type='button'
              className='invoice-button is-danger'
              onClick={cancelRequest}
              disabled={submitting}
            >
              {t('确认取消')}
            </button>
          </>
        }
      >
        <p className='invoice-inline-meta'>
          {t('取消后，暂扣的可开票金额会立即释放，可重新提交申请。')}
        </p>
      </InvoiceDialog>
    </div>
  );
};

export default InvoiceManagement;
