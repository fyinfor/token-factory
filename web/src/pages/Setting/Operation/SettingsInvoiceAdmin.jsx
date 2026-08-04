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
import { Select } from '@douyinfe/semi-ui';
import {
  CheckCircle2,
  CirclePlay,
  Download,
  Eye,
  RefreshCw,
  Search,
  Send,
  XCircle,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import InvoiceFileUpload from '../../../components/settings/InvoiceFileUpload';
import {
  API,
  getPayMethodDisplayName,
  showError,
  showSuccess,
  timestamp2string,
} from '../../../helpers';
import {
  InvoiceDialog,
  InvoiceEmptyState,
  InvoicePagination,
  InvoiceSpinner,
  InvoiceStatusBadge,
  formatInvoiceMoney,
  parseInvoiceProfile,
} from '../../../components/invoice/InvoiceWorkspace';

const PAGE_SIZE = 12;
const statusOptions = [
  ['', '全部'],
  ['pending', '待处理'],
  ['processing', '处理中'],
  ['issued', '已开具'],
  ['rejected', '已驳回'],
  ['cancelled', '已取消'],
];

const SettingsInvoiceAdmin = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [items, setItems] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState('pending');
  const [searchDraft, setSearchDraft] = useState('');
  const [keyword, setKeyword] = useState('');
  const [counts, setCounts] = useState({
    pending: 0,
    processing: 0,
    issued: 0,
  });
  const [issueTarget, setIssueTarget] = useState(null);
  const [rejectTarget, setRejectTarget] = useState(null);
  const [detailOpen, setDetailOpen] = useState(false);
  const [detail, setDetail] = useState(null);
  const [issueDetail, setIssueDetail] = useState(null);
  const [issueForm, setIssueForm] = useState({
    invoice_code: '',
    invoice_url: '',
    admin_note: '',
  });
  const [rejectNote, setRejectNote] = useState('');

  const loadCounts = useCallback(async () => {
    try {
      const results = await Promise.all(
        ['pending', 'processing', 'issued'].map((status) =>
          API.get('/api/user/invoice/admin/requests', {
            params: { p: 1, page_size: 1, status },
          }),
        ),
      );
      setCounts(
        Object.fromEntries(
          results.map((res, index) => [
            ['pending', 'processing', 'issued'][index],
            Number(res.data?.data?.total || 0),
          ]),
        ),
      );
    } catch {
      // Counts are secondary; the primary queue remains usable.
    }
  }, []);

  const loadList = useCallback(
    async (targetPage = 1) => {
      setLoading(true);
      try {
        const res = await API.get('/api/user/invoice/admin/requests', {
          params: {
            p: targetPage,
            page_size: PAGE_SIZE,
            status: statusFilter || undefined,
            keyword: keyword || undefined,
          },
        });
        if (res.data.success) {
          setItems(res.data.data?.items || []);
          setTotal(Number(res.data.data?.total || 0));
          setPage(targetPage);
        }
      } catch (error) {
        showError(error);
      } finally {
        setLoading(false);
      }
    },
    [keyword, statusFilter],
  );

  useEffect(() => {
    loadList(1);
    loadCounts();
  }, [loadCounts, loadList]);

  const fetchDetail = async (record) => {
    const res = await API.get(`/api/user/invoice/admin/requests/${record.id}`);
    if (!res.data.success)
      throw new Error(res.data.message || 'invoice detail');
    return res.data.data || null;
  };

  const openDetail = async (record) => {
    setDetailOpen(true);
    setDetail(null);
    setDetailLoading(true);
    try {
      setDetail(await fetchDetail(record));
    } catch (error) {
      showError(error);
      setDetailOpen(false);
    } finally {
      setDetailLoading(false);
    }
  };

  const openIssue = async (record) => {
    setIssueTarget(record);
    setIssueDetail(null);
    setIssueForm({ invoice_code: '', invoice_url: '', admin_note: '' });
    setDetailLoading(true);
    try {
      setIssueDetail(await fetchDetail(record));
    } catch (error) {
      showError(error);
      setIssueTarget(null);
    } finally {
      setDetailLoading(false);
    }
  };

  const startProcessing = async (record) => {
    setSubmitting(true);
    try {
      const res = await API.post(
        `/api/user/invoice/admin/requests/${record.id}/process`,
      );
      if (res.data.success) {
        showSuccess(t('已开始处理'));
        await Promise.all([loadList(page), loadCounts()]);
      }
    } catch (error) {
      showError(error);
    } finally {
      setSubmitting(false);
    }
  };

  const submitIssue = async () => {
    if (!issueTarget) return;
    if (!issueForm.invoice_url?.trim()) {
      showError(t('请先上传电子发票 PDF'));
      return;
    }
    setSubmitting(true);
    try {
      const res = await API.post(
        `/api/user/invoice/admin/requests/${issueTarget.id}/issue`,
        issueForm,
      );
      if (res.data.success) {
        showSuccess(t('发票已开具'));
        setIssueTarget(null);
        await Promise.all([loadList(page), loadCounts()]);
      }
    } catch (error) {
      showError(error);
    } finally {
      setSubmitting(false);
    }
  };

  const submitReject = async () => {
    if (!rejectTarget) return;
    if (!rejectNote.trim()) {
      showError(t('请填写驳回原因'));
      return;
    }
    setSubmitting(true);
    try {
      const res = await API.post(
        `/api/user/invoice/admin/requests/${rejectTarget.id}/reject`,
        { admin_note: rejectNote.trim() },
      );
      if (res.data.success) {
        showSuccess(t('已驳回'));
        setRejectTarget(null);
        setRejectNote('');
        await Promise.all([loadList(page), loadCounts()]);
      }
    } catch (error) {
      showError(error);
    } finally {
      setSubmitting(false);
    }
  };

  const issueProfile = useMemo(
    () => parseInvoiceProfile(issueDetail?.request?.profile_snapshot),
    [issueDetail],
  );
  const detailProfile = useMemo(
    () => parseInvoiceProfile(detail?.request?.profile_snapshot),
    [detail],
  );

  return (
    <div className='invoice-workspace'>
      <section className='invoice-glass-panel invoice-notice-panel is-admin is-headerless'>
        <div className='invoice-notice-content'>
          <div className='invoice-notice-copy'>
            <h2>{t('审批须知')}</h2>
            <ul className='invoice-notice-list'>
              <li>{t('处理充值订单发票申请、文件开具与状态追踪')}</li>
              <li>{t('审核开票申请、上传电子发票并追踪处理状态')}</li>
              <li>{t('仅支持 PDF 格式电子发票')}</li>
              <li>{t('驳回后会立即释放该申请占用的可开票金额')}</li>
            </ul>
          </div>
          <div className='invoice-notice-metrics' aria-label={t('审批概览')}>
            <div>
              <span>{t('待处理')}</span>
              <strong>{counts.pending}</strong>
            </div>
            <div>
              <span>{t('处理中')}</span>
              <strong>{counts.processing}</strong>
            </div>
            <div className='is-highlight'>
              <span>{t('已开具')}</span>
              <strong>{counts.issued}</strong>
            </div>
          </div>
        </div>
      </section>

      <section className='invoice-glass-panel invoice-toolbar-panel'>
        <header className='invoice-toolbar is-admin'>
          <div className='invoice-toolbar-group'>
            <form
              className='invoice-search-form'
              onSubmit={(event) => {
                event.preventDefault();
                setKeyword(searchDraft.trim());
                setPage(1);
              }}
            >
              <div className='invoice-search'>
                <Search size={16} aria-hidden='true' />
                <input
                  className='invoice-input'
                  name='invoice-admin-search'
                  type='search'
                  value={searchDraft}
                  onChange={(event) => setSearchDraft(event.target.value)}
                  placeholder={`${t('搜索申请单号、用户或邮箱')}…`}
                  aria-label={t('搜索申请单号、用户或邮箱')}
                  autoComplete='off'
                  spellCheck={false}
                />
              </div>
              <button type='submit' className='invoice-button is-search'>
                {t('查询')}
              </button>
            </form>
          </div>
          <div className='invoice-toolbar-group is-end'>
            <Select
              className='invoice-semi-select'
              dropdownClassName='invoice-semi-select-dropdown'
              value={statusFilter}
              optionList={statusOptions.map(([value, label]) => ({
                value,
                label: t(label),
              }))}
              onChange={(value) => {
                setStatusFilter(String(value ?? ''));
                setPage(1);
              }}
              aria-label={t('状态筛选')}
            />
            <button
              type='button'
              className='invoice-icon-button'
              onClick={() => Promise.all([loadList(page), loadCounts()])}
              title={t('刷新')}
              aria-label={t('刷新')}
            >
              <RefreshCw size={17} />
            </button>
          </div>
        </header>

        <div className='invoice-table-wrap'>
          <table className='invoice-table'>
            <thead>
              <tr>
                <th>{t('申请单号')}</th>
                <th>{t('用户')}</th>
                <th>{t('金额')}</th>
                <th>{t('状态')}</th>
                <th>{t('创建时间')}</th>
                <th>{t('操作')}</th>
              </tr>
            </thead>
            <tbody>
              {loading ? (
                <tr>
                  <td className='invoice-loading-row' colSpan={6}>
                    <InvoiceSpinner label={t('正在加载...')} />
                  </td>
                </tr>
              ) : items.length ? (
                items.map((record) => (
                  <tr key={record.id}>
                    <td className='is-strong' data-label={t('申请单号')}>
                      {record.request_no}
                    </td>
                    <td data-label={t('用户')}>
                      <strong>{record.username || '-'}</strong>
                      <div className='invoice-inline-meta'>
                        {record.email || '-'}
                      </div>
                    </td>
                    <td className='is-numeric is-strong' data-label={t('金额')}>
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
                        {record.status === 'pending' ? (
                          <button
                            type='button'
                            className='invoice-button is-quiet'
                            disabled={submitting}
                            onClick={() => startProcessing(record)}
                          >
                            <CirclePlay size={15} />
                            {t('开始处理')}
                          </button>
                        ) : null}
                        {record.status === 'pending' ||
                        record.status === 'processing' ? (
                          <>
                            <button
                              type='button'
                              className='invoice-button is-quiet'
                              onClick={() => openIssue(record)}
                            >
                              <Send size={15} />
                              {t('开具')}
                            </button>
                            <button
                              type='button'
                              className='invoice-button is-quiet'
                              onClick={() => {
                                setRejectTarget(record);
                                setRejectNote('');
                              }}
                            >
                              <XCircle size={15} />
                              {t('驳回')}
                            </button>
                          </>
                        ) : null}
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
                      </div>
                    </td>
                  </tr>
                ))
              ) : (
                <tr>
                  <td className='invoice-empty-row' colSpan={6}>
                    <InvoiceEmptyState
                      title={t('暂无匹配的发票申请')}
                      description={t('调整筛选条件或等待用户提交新的开票申请')}
                    />
                  </td>
                </tr>
              )}
            </tbody>
          </table>
          <InvoicePagination
            page={page}
            total={total}
            pageSize={PAGE_SIZE}
            onChange={loadList}
            t={t}
          />
        </div>
      </section>

      <InvoiceDialog
        open={Boolean(issueTarget)}
        onClose={() => setIssueTarget(null)}
        eyebrow={t('电子发票')}
        title={t('开具发票')}
        size='large'
        footer={
          <>
            <button
              type='button'
              className='invoice-button'
              onClick={() => setIssueTarget(null)}
            >
              {t('取消')}
            </button>
            <button
              type='button'
              className='invoice-button is-primary'
              onClick={submitIssue}
              disabled={submitting || detailLoading}
            >
              {submitting ? <InvoiceSpinner /> : <CheckCircle2 size={16} />}
              {t('确认开具')}
            </button>
          </>
        }
      >
        {detailLoading ? (
          <InvoiceEmptyState
            title={t('正在加载...')}
            action={<InvoiceSpinner />}
          />
        ) : (
          <div className='invoice-form-grid'>
            <div className='invoice-detail-grid invoice-field is-full'>
              <div className='invoice-detail-item'>
                <span>{t('申请单号')}</span>
                <strong>{issueTarget?.request_no}</strong>
              </div>
              <div className='invoice-detail-item'>
                <span>{t('金额')}</span>
                <strong>{formatInvoiceMoney(issueTarget?.total_amount)}</strong>
              </div>
              <div className='invoice-detail-item'>
                <span>{t('发票类型')}</span>
                <strong>
                  {issueTarget?.invoice_type === 'electronic_special'
                    ? t('专票')
                    : t('普票')}
                </strong>
              </div>
              <div className='invoice-detail-item'>
                <span>{t('收票邮箱')}</span>
                <strong>
                  {issueProfile?.email || issueDetail?.email || '-'}
                </strong>
              </div>
            </div>
            <div className='invoice-field is-full'>
              <label htmlFor='invoice-code'>{t('发票号码')}</label>
              <input
                id='invoice-code'
                name='invoice-code'
                className='invoice-input'
                value={issueForm.invoice_code}
                autoComplete='off'
                spellCheck={false}
                onChange={(event) =>
                  setIssueForm((current) => ({
                    ...current,
                    invoice_code: event.target.value,
                  }))
                }
              />
            </div>
            <div className='invoice-field is-full'>
              <span>{t('电子发票文件')}</span>
              <InvoiceFileUpload
                url={issueForm.invoice_url}
                onUrlChange={(invoice_url) =>
                  setIssueForm((current) => ({ ...current, invoice_url }))
                }
                disabled={submitting}
              />
            </div>
            <div className='invoice-field is-full'>
              <label htmlFor='invoice-admin-note'>{t('审核备注')}</label>
              <textarea
                id='invoice-admin-note'
                name='invoice-admin-note'
                className='invoice-textarea'
                value={issueForm.admin_note}
                autoComplete='off'
                onChange={(event) =>
                  setIssueForm((current) => ({
                    ...current,
                    admin_note: event.target.value,
                  }))
                }
              />
            </div>
          </div>
        )}
      </InvoiceDialog>

      <InvoiceDialog
        open={Boolean(rejectTarget)}
        onClose={() => setRejectTarget(null)}
        eyebrow={t('驳回申请')}
        title={t('填写驳回原因')}
        size='small'
        footer={
          <>
            <button
              type='button'
              className='invoice-button'
              onClick={() => setRejectTarget(null)}
            >
              {t('取消')}
            </button>
            <button
              type='button'
              className='invoice-button is-danger'
              onClick={submitReject}
              disabled={submitting || !rejectNote.trim()}
            >
              {t('确认驳回')}
            </button>
          </>
        }
      >
        <div className='invoice-field'>
          <label htmlFor='invoice-reject-note'>{t('驳回原因')}</label>
          <textarea
            id='invoice-reject-note'
            name='invoice-reject-note'
            className='invoice-textarea'
            value={rejectNote}
            autoComplete='off'
            onChange={(event) => setRejectNote(event.target.value)}
            autoFocus
          />
          <small>{t('驳回后会立即释放该申请占用的可开票金额')}</small>
        </div>
      </InvoiceDialog>

      <InvoiceDialog
        open={detailOpen}
        onClose={() => setDetailOpen(false)}
        eyebrow={t('审批记录')}
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
                <span>{t('用户')}</span>
                <strong>
                  {detail.username || '-'} ({detail.email || '-'})
                </strong>
              </div>
              <div className='invoice-detail-item'>
                <span>{t('状态')}</span>
                <strong>
                  <InvoiceStatusBadge status={detail.request?.status} t={t} />
                </strong>
              </div>
              <div className='invoice-detail-item'>
                <span>{t('发票类型')}</span>
                <strong>
                  {detail.request?.invoice_type === 'electronic_special'
                    ? t('专票')
                    : t('普票')}
                </strong>
              </div>
              <div className='invoice-detail-item'>
                <span>{t('发票抬头')}</span>
                <strong>{detailProfile?.title || '-'}</strong>
              </div>
              <div className='invoice-detail-item'>
                <span>{t('税号')}</span>
                <strong>{detailProfile?.tax_no || '-'}</strong>
              </div>
              <div className='invoice-detail-item'>
                <span>{t('收票邮箱')}</span>
                <strong>{detailProfile?.email || detail.email || '-'}</strong>
              </div>
              <div className='invoice-detail-item'>
                <span>{t('金额')}</span>
                <strong>
                  {formatInvoiceMoney(detail.request?.total_amount)}
                </strong>
              </div>
              <div className='invoice-detail-item'>
                <span>{t('发票号码')}</span>
                <strong>{detail.request?.invoice_code || '-'}</strong>
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
                    <th>{t('支付方式')}</th>
                    <th>{t('开票金额')}</th>
                  </tr>
                </thead>
                <tbody>
                  {(detail.items || []).map((item) => (
                    <tr key={item.id}>
                      <td className='is-strong' data-label={t('订单号')}>
                        {item.trade_no}
                      </td>
                      <td data-label={t('支付方式')}>
                        {getPayMethodDisplayName(item.payment_method, t)}
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
    </div>
  );
};

export default SettingsInvoiceAdmin;
