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

import React, { useCallback, useEffect, useState } from 'react';
import {
  Banner,
  Button,
  Modal,
  Table,
  Tabs,
  Typography,
  Input,
  TextArea,
  Select,
  Empty,
  Upload,
} from '@douyinfe/semi-ui';
import {
  IllustrationNoResult,
  IllustrationNoResultDark,
} from '@douyinfe/semi-illustrations';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';
import {
  API,
  showError,
  showSuccess,
  formatCommissionRatioPercent,
  commissionBpsToPercentInputString,
  parseCommissionPercentStringToBps,
  renderQuota,
} from '../helpers';
import { createCardProPagination } from '../helpers/utils';
import { useIsMobile } from '../hooks/common/useIsMobile';
import CardPro from '../components/common/ui/CardPro';
import CardTable from '../components/common/ui/CardTable';
import { IconFile, IconSearch } from '@douyinfe/semi-icons';

const { Text } = Typography;

/** 分销商管理筛选：关键字最大长度（与后端查询参数长度协调） */
const ADMIN_KEYWORD_MAX_LEN = 120;

/** 资格证书上传数量上限（与申请页一致） */
const PROFILE_QUAL_MAX_FILES = 5;

/** 解析后端存库的资格证书 JSON 数组字符串 */
function parseQualificationUrls(raw) {
  if (raw == null) return [];
  if (Array.isArray(raw)) return raw.filter(Boolean);
  if (typeof raw === 'string') {
    const s = raw.trim();
    if (!s) return [];
    try {
      const j = JSON.parse(s);
      return Array.isArray(j) ? j.filter(Boolean) : [];
    } catch {
      return [];
    }
  }
  return [];
}

function isPdfUrl(u) {
  return /\.pdf(\?|$)/i.test(u || '');
}

/** 资格证书缩略图：点击图片回调大图预览；PDF 新窗口打开 */
function QualificationThumbnails({ urls, onImagePreview, compact }) {
  const list = urls || [];
  if (!list.length) {
    return (
      <Text type='tertiary' size='small'>
        —
      </Text>
    );
  }
  const max = compact ? 4 : 12;
  const shown = list.slice(0, max);
  const more = list.length - shown.length;
  const box = compact ? 'h-10 w-10' : 'h-24 w-24';

  return (
    <div className='flex flex-wrap items-center gap-2'>
      {shown.map((u, idx) =>
        isPdfUrl(u) ? (
          <button
            key={`pdf-${idx}-${u.slice(-20)}`}
            type='button'
            title='PDF'
            className={`flex ${box} flex-shrink-0 flex-col items-center justify-center rounded-lg border border-[var(--semi-color-border)] bg-[var(--semi-color-fill-0)] hover:bg-[var(--semi-color-fill-1)]`}
            onClick={() => window.open(u, '_blank', 'noopener,noreferrer')}
          >
            <IconFile size={compact ? 'default' : 'large'} />
            {!compact && (
              <span className='mt-0.5 text-[10px] text-[var(--semi-color-text-2)]'>
                PDF
              </span>
            )}
          </button>
        ) : (
          <button
            key={`img-${idx}-${u.slice(-24)}`}
            type='button'
            className={`relative flex-shrink-0 overflow-hidden rounded-lg border border-[var(--semi-color-border)] bg-[var(--semi-color-fill-0)] ${box} cursor-zoom-in p-0`}
            onClick={() => onImagePreview?.(u)}
          >
            <img src={u} alt='' className='h-full w-full object-cover' />
          </button>
        ),
      )}
      {more > 0 ? (
        <span className='text-xs text-[var(--semi-color-text-2)]'>+{more}</span>
      ) : null}
    </div>
  );
}

const appStatus = (s) => {
  if (s === 1) return '待审核';
  if (s === 2) return '已通过';
  if (s === 3) return '已驳回';
  return '—';
};

export default function DistributorAdmin() {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const [tab, setTab] = useState('app');

  const [appLoading, setAppLoading] = useState(false);
  const [appRows, setAppRows] = useState([]);
  const [appTotal, setAppTotal] = useState(0);
  const [appPage, setAppPage] = useState(1);
  const [appPageSize, setAppPageSize] = useState(10);
  const [appKeyword, setAppKeyword] = useState('');
  const [appStatusFilter, setAppStatusFilter] = useState(0);

  const [detailOpen, setDetailOpen] = useState(false);
  const [detail, setDetail] = useState(null);
  const [qualImagePreview, setQualImagePreview] = useState(null);

  const [rejectOpen, setRejectOpen] = useState(false);
  const [rejectId, setRejectId] = useState(null);
  const [rejectReason, setRejectReason] = useState('');

  const [distLoading, setDistLoading] = useState(false);
  const [distRows, setDistRows] = useState([]);
  const [distTotal, setDistTotal] = useState(0);
  const [distPage, setDistPage] = useState(1);
  const [distPageSize, setDistPageSize] = useState(10);
  const [distKeyword, setDistKeyword] = useState('');

  const [bpsOpen, setBpsOpen] = useState(false);
  const [bpsUser, setBpsUser] = useState(null);
  /** 弹窗内输入：0～100 的百分比字符串，提交时转为万分之一 bps */
  const [bpsPercentInput, setBpsPercentInput] = useState('0');

  const [invOpen, setInvOpen] = useState(false);
  const [invRows, setInvRows] = useState([]);
  const [invTotal, setInvTotal] = useState(0);
  const [invPage, setInvPage] = useState(1);
  const [invPs, setInvPs] = useState(10);
  const [invDistributorId, setInvDistributorId] = useState(null);

  const [profileOpen, setProfileOpen] = useState(false);
  const [profileLoading, setProfileLoading] = useState(false);
  const [profileSaving, setProfileSaving] = useState(false);
  const [profileUserId, setProfileUserId] = useState(null);
  const [profileUsername, setProfileUsername] = useState('');
  const [profileNeedsManual, setProfileNeedsManual] = useState(false);
  const [profileAppStatus, setProfileAppStatus] = useState(null);
  const [profileRealName, setProfileRealName] = useState('');
  const [profileIdCard, setProfileIdCard] = useState('');
  const [profileContact, setProfileContact] = useState('');
  /** 资格证书文件 URL 列表（上传 OSS 或历史数据解析） */
  const [profileQualUrls, setProfileQualUrls] = useState([]);

  const [wdLoading, setWdLoading] = useState(false);
  const [wdRows, setWdRows] = useState([]);
  const [wdTotal, setWdTotal] = useState(0);
  const [wdPage, setWdPage] = useState(1);
  const [wdPs, setWdPs] = useState(10);
  const [wdKeyword, setWdKeyword] = useState('');
  const [wdStatusFilter, setWdStatusFilter] = useState(0);
  const [wdRejectOpen, setWdRejectOpen] = useState(false);
  const [wdRejectId, setWdRejectId] = useState(null);
  const [wdRejectReason, setWdRejectReason] = useState('');

  const loadApps = useCallback(async () => {
    setAppLoading(true);
    try {
      const q = new URLSearchParams({
        p: String(appPage),
        page_size: String(appPageSize),
      });
      if (appKeyword.trim()) q.set('keyword', appKeyword.trim());
      if (appStatusFilter) q.set('status', String(appStatusFilter));
      const res = await API.get(`/api/distributor/admin/applications?${q}`);
      const { success, message, data } = res.data;
      if (!success) {
        showError(message);
        return;
      }
      setAppRows(data?.items || []);
      setAppTotal(data?.total ?? 0);
    } catch {
      showError(t('加载失败'));
    } finally {
      setAppLoading(false);
    }
  }, [appPage, appPageSize, appKeyword, appStatusFilter, t]);

  const loadDists = useCallback(async () => {
    setDistLoading(true);
    try {
      const q = new URLSearchParams({
        p: String(distPage),
        page_size: String(distPageSize),
      });
      if (distKeyword.trim()) q.set('keyword', distKeyword.trim());
      const res = await API.get(`/api/distributor/admin/distributors?${q}`);
      const { success, message, data } = res.data;
      if (!success) {
        showError(message);
        return;
      }
      setDistRows(data?.items || []);
      setDistTotal(data?.total ?? 0);
    } catch {
      showError(t('加载失败'));
    } finally {
      setDistLoading(false);
    }
  }, [distPage, distPageSize, distKeyword, t]);

  useEffect(() => {
    if (tab === 'app') loadApps();
  }, [tab, loadApps]);

  useEffect(() => {
    if (tab === 'dist') loadDists();
  }, [tab, loadDists]);

  const loadWdWithdrawals = useCallback(async () => {
    setWdLoading(true);
    try {
      const q = new URLSearchParams({
        p: String(wdPage),
        page_size: String(wdPs),
      });
      if (wdKeyword.trim()) q.set('keyword', wdKeyword.trim());
      if (wdStatusFilter) q.set('status', String(wdStatusFilter));
      const res = await API.get(`/api/distributor/admin/withdrawals?${q}`);
      const { success, message, data } = res.data;
      if (!success) {
        showError(message);
        return;
      }
      setWdRows(data?.items || []);
      setWdTotal(data?.total ?? 0);
    } catch {
      showError(t('加载失败'));
    } finally {
      setWdLoading(false);
    }
  }, [wdPage, wdPs, wdKeyword, wdStatusFilter, t]);

  useEffect(() => {
    if (tab === 'wd') loadWdWithdrawals();
  }, [tab, loadWdWithdrawals]);

  const openDistributorProfile = async (row) => {
    setProfileUserId(row.id);
    setProfileUsername(row.username || '');
    setProfileOpen(true);
    setProfileLoading(true);
    setProfileNeedsManual(false);
    setProfileAppStatus(null);
    setProfileRealName('');
    setProfileIdCard('');
    setProfileContact('');
    setProfileQualUrls([]);
    try {
      const res = await API.get(
        `/api/distributor/admin/distributors/${row.id}/application`,
      );
      const { success, message, data } = res.data;
      if (!success) {
        showError(message);
        return;
      }
      setProfileNeedsManual(Boolean(data?.needs_manual_entry));
      const app = data?.application;
      if (app) {
        setProfileAppStatus(app.status ?? null);
        setProfileRealName(app.real_name || '');
        setProfileIdCard(app.id_card_no || '');
        setProfileContact(app.contact || '');
        setProfileQualUrls(parseQualificationUrls(app.qualification_urls));
      }
    } catch {
      showError(t('加载失败'));
    } finally {
      setProfileLoading(false);
    }
  };

  const saveDistributorProfile = async () => {
    if (!profileUserId) return;
    const urls = profileQualUrls.filter(Boolean);
    if (!profileRealName.trim()) {
      showError(t('请填写真实姓名'));
      return;
    }
    if (!profileIdCard.trim()) {
      showError(t('请填写身份证号码'));
      return;
    }
    if (!profileContact.trim()) {
      showError(t('请填写联系方式'));
      return;
    }
    if (urls.length === 0) {
      showError(t('请上传资格证书'));
      return;
    }
    setProfileSaving(true);
    try {
      const res = await API.put(
        `/api/distributor/admin/distributors/${profileUserId}/application`,
        {
          real_name: profileRealName.trim(),
          id_card_no: profileIdCard.trim(),
          contact: profileContact.trim(),
          qualification_urls: urls,
        },
      );
      if (res.data.success) {
        showSuccess(t('已保存'));
        try {
          const r2 = await API.get(
            `/api/distributor/admin/distributors/${profileUserId}/application`,
          );
          const d = r2.data?.data;
          if (r2.data.success && d) {
            setProfileNeedsManual(Boolean(d.needs_manual_entry));
            const app = d.application;
            if (app) {
              setProfileAppStatus(app.status ?? null);
              setProfileRealName(app.real_name || '');
              setProfileIdCard(app.id_card_no || '');
              setProfileContact(app.contact || '');
              setProfileQualUrls(parseQualificationUrls(app.qualification_urls));
            }
          }
        } catch {
          /* ignore */
        }
      } else {
        showError(res.data.message);
      }
    } catch {
      showError(t('保存失败'));
    } finally {
      setProfileSaving(false);
    }
  };

  const openDetail = async (id) => {
    try {
      const res = await API.get(`/api/distributor/admin/applications/${id}`);
      const { success, message, data } = res.data;
      if (!success) {
        showError(message);
        return;
      }
      setDetail(data);
      setDetailOpen(true);
    } catch {
      showError(t('加载失败'));
    }
  };

  const approve = async (id) => {
    try {
      const res = await API.post(
        `/api/distributor/admin/applications/${id}/approve`,
      );
      if (res.data.success) {
        showSuccess(t('已通过'));
        loadApps();
      } else {
        showError(res.data.message);
      }
    } catch {
      showError(t('操作失败'));
    }
  };

  const submitReject = async () => {
    if (!rejectReason.trim()) {
      showError(t('请填写驳回原因'));
      return;
    }
    try {
      const res = await API.post(
        `/api/distributor/admin/applications/${rejectId}/reject`,
        { reason: rejectReason },
      );
      if (res.data.success) {
        showSuccess(t('已驳回'));
        setRejectOpen(false);
        setRejectReason('');
        loadApps();
      } else {
        showError(res.data.message);
      }
    } catch {
      showError(t('操作失败'));
    }
  };

  const openInvitees = async (userId) => {
    setInvDistributorId(userId);
    setInvPage(1);
    setInvOpen(true);
    await fetchInvitees(userId, 1, invPs);
  };

  const fetchInvitees = async (userId, p, ps) => {
    const res = await API.get(
      `/api/distributor/admin/distributors/${userId}/invitees?p=${p}&page_size=${ps}`,
    );
    const { success, message, data } = res.data;
    if (!success) {
      showError(message);
      return;
    }
    setInvRows(data?.items || []);
    setInvTotal(data?.total ?? 0);
    setInvPage(p);
    setInvPs(ps);
  };

  const saveBps = async () => {
    if (!bpsUser) return;
    const bps = parseCommissionPercentStringToBps(bpsPercentInput);
    if (Number.isNaN(bps)) {
      showError(t('请输入 0～100 之间的百分比'));
      return;
    }
    try {
      const res = await API.put(
        `/api/distributor/admin/distributors/${bpsUser.id}/commission`,
        { distributor_commission_bps: bps },
      );
      if (res.data.success) {
        showSuccess(t('已保存'));
        setBpsOpen(false);
        loadDists();
      } else {
        showError(res.data.message);
      }
    } catch {
      showError(t('保存失败'));
    }
  };

  const wdApprove = async (id) => {
    try {
      const res = await API.post(
        `/api/distributor/admin/withdrawals/${id}/approve`,
      );
      if (res.data.success) {
        showSuccess(t('已通过'));
        loadWdWithdrawals();
      } else {
        showError(res.data.message);
      }
    } catch {
      showError(t('操作失败'));
    }
  };

  const wdSubmitReject = async () => {
    if (!wdRejectReason.trim()) {
      showError(t('请填写驳回原因'));
      return;
    }
    try {
      const res = await API.post(
        `/api/distributor/admin/withdrawals/${wdRejectId}/reject`,
        { reason: wdRejectReason },
      );
      if (res.data.success) {
        showSuccess(t('已驳回'));
        setWdRejectOpen(false);
        setWdRejectReason('');
        loadWdWithdrawals();
      } else {
        showError(res.data.message);
      }
    } catch {
      showError(t('操作失败'));
    }
  };

  const wdStatusLabel = (s) => {
    if (s === 1) return t('提现中');
    if (s === 2) return t('提现成功');
    if (s === 3) return t('提现失败');
    if (s === 4) return t('已取消');
    return '—';
  };

  const settle = async (userId) => {
    Modal.confirm({
      title: t('确认结账'),
      content: t('将清空该分销商的待使用收益额度，是否继续？'),
      onOk: async () => {
        try {
          const res = await API.post(
            `/api/distributor/admin/distributors/${userId}/settle`,
          );
          if (res.data.success) {
            showSuccess(t('已结账'));
            loadDists();
          } else {
            showError(res.data.message);
          }
        } catch {
          showError(t('操作失败'));
        }
      },
    });
  };

  const appColumns = [
    { title: 'ID', dataIndex: 'id', width: 70 },
    { title: t('用户名'), dataIndex: 'username' },
    { title: t('姓名'), dataIndex: 'real_name' },
    { title: t('联系方式'), dataIndex: 'contact' },
    {
      title: t('资格证书'),
      dataIndex: 'qualification_urls',
      width: 220,
      render: (_, r) => (
        <QualificationThumbnails
          urls={parseQualificationUrls(r.qualification_urls)}
          compact
          onImagePreview={(u) => setQualImagePreview(u)}
        />
      ),
    },
    {
      title: t('状态'),
      dataIndex: 'status',
      render: (s) => appStatus(s),
    },
    {
      title: t('提交时间'),
      dataIndex: 'created_at',
      render: (ts) =>
        ts ? dayjs.unix(Number(ts)).format('YYYY-MM-DD HH:mm') : '—',
    },
    {
      title: t('操作'),
      render: (_, r) => (
        <div className='flex flex-wrap gap-1'>
          <Button size='small' onClick={() => openDetail(r.id)}>
            {t('详情')}
          </Button>
          {r.status === 1 && (
            <>
              <Button
                size='small'
                type='primary'
                onClick={() => approve(r.id)}
              >
                {t('通过')}
              </Button>
              <Button
                size='small'
                type='danger'
                onClick={() => {
                  setRejectId(r.id);
                  setRejectReason('');
                  setRejectOpen(true);
                }}
              >
                {t('驳回')}
              </Button>
            </>
          )}
        </div>
      ),
    },
  ];

  const distColumns = [
    { title: 'ID', dataIndex: 'id', width: 80 },
    { title: t('用户名'), dataIndex: 'username' },
    {
      title: t('申请真实姓名'),
      dataIndex: 'application_real_name',
      render: (name) => {
        const s = name != null ? String(name).trim() : '';
        return s || '—';
      },
    },
    { title: 'aff', dataIndex: 'aff_code', width: 90 },
    {
      title: t('默认分成'),
      dataIndex: 'effective_commission_bps',
      render: (bps) => formatCommissionRatioPercent(bps),
    },
    {
      title: t('待结算'),
      dataIndex: 'aff_quota',
      render: (q) => renderQuota(q || 0),
    },
    {
      title: t('操作'),
      render: (_, r) => (
        <div className='flex flex-wrap gap-1'>
          <Button
            size='small'
            type={r.needs_supplement ? 'warning' : 'tertiary'}
            onClick={() => openDistributorProfile(r)}
          >
            {r.needs_supplement ? t('补充资料') : t('查看资料')}
          </Button>
          <Button size='small' onClick={() => openInvitees(r.id)}>
            {t('邀请明细')}
          </Button>
          <Button
            size='small'
            onClick={() => {
              setBpsUser(r);
              setBpsPercentInput(
                commissionBpsToPercentInputString(
                  r.distributor_commission_bps || 0,
                ),
              );
              setBpsOpen(true);
            }}
          >
            {t('分成比例')}
          </Button>
          <Button size='small' type='warning' onClick={() => settle(r.id)}>
            {t('结账')}
          </Button>
        </div>
      ),
    },
  ];

  const wdColumns = [
    { title: 'ID', dataIndex: 'id', width: 70 },
    { title: t('用户名'), dataIndex: 'username', width: 120 },
    { title: t('姓名'), dataIndex: 'real_name', width: 90 },
    {
      title: t('收款账户'),
      render: (_, r) => (
        <div className='text-xs space-y-0.5 max-w-[200px]'>
          <div>{r.bank_name}</div>
          <div className='text-[var(--semi-color-text-2)]'>{r.bank_account}</div>
        </div>
      ),
    },
    {
      title: t('票据'),
      dataIndex: 'voucher_urls',
      width: 200,
      render: (raw) => (
        <QualificationThumbnails
          urls={parseQualificationUrls(raw)}
          compact
          onImagePreview={(u) => setQualImagePreview(u)}
        />
      ),
    },
    {
      title: t('提现月份'),
      dataIndex: 'withdraw_month',
      width: 100,
    },
    {
      title: t('额度'),
      dataIndex: 'quota_amount',
      render: (q) => renderQuota(q || 0),
    },
    {
      title: t('状态'),
      dataIndex: 'status',
      width: 90,
      render: (s) => wdStatusLabel(s),
    },
    {
      title: t('申请时间'),
      dataIndex: 'created_at',
      width: 160,
      render: (ts) =>
        ts ? dayjs.unix(Number(ts)).format('YYYY-MM-DD HH:mm') : '—',
    },
    {
      title: t('操作'),
      width: 160,
      render: (_, r) => (
        <div className='flex flex-wrap gap-1'>
          {r.status === 1 && (
            <>
              <Button size='small' type='primary' onClick={() => wdApprove(r.id)}>
                {t('通过')}
              </Button>
              <Button
                size='small'
                type='danger'
                onClick={() => {
                  setWdRejectId(r.id);
                  setWdRejectReason('');
                  setWdRejectOpen(true);
                }}
              >
                {t('驳回')}
              </Button>
            </>
          )}
        </div>
      ),
    },
  ];

  const invColumns = [
    { title: t('用户'), dataIndex: 'username' },
    {
      title: t('分成'),
      dataIndex: 'commission_ratio_bps',
      render: (b) => formatCommissionRatioPercent(b),
    },
    {
      title: t('累计分成额度'),
      dataIndex: 'commission_earned_quota',
      render: (q) => renderQuota(q || 0),
    },
  ];

  const tableEmpty = (
    <Empty
      image={<IllustrationNoResult style={{ width: 150, height: 150 }} />}
      darkModeImage={
        <IllustrationNoResultDark style={{ width: 150, height: 150 }} />
      }
      description={t('搜索无结果')}
      style={{ padding: 30 }}
    />
  );

  return (
    <div className='mt-[60px] px-2 pb-16'>
      <Typography.Title heading={3}>{t('分销商管理中心')}</Typography.Title>

      <Tabs
        activeKey={tab}
        onChange={setTab}
        type='line'
        className='mt-4 w-full distributor-admin-tabs'
      >
        <Tabs.TabPane tab={t('申请审核')} itemKey='app'>
          <CardPro
            className='w-full'
            type='type1'
            actionsArea={
              <div className='flex flex-col md:flex-row justify-end items-center gap-2 w-full'>
                <div className='flex flex-col md:flex-row items-center gap-2 w-full md:w-auto'>
                  <div className='relative w-full md:w-64'>
                    <Input
                      value={appKeyword}
                      maxLength={ADMIN_KEYWORD_MAX_LEN}
                      prefix={<IconSearch />}
                      showClear
                      pure
                      size='small'
                      placeholder={t('搜索姓名、用户名、联系方式')}
                      onChange={(v) =>
                        setAppKeyword(
                          String(v ?? '').slice(0, ADMIN_KEYWORD_MAX_LEN),
                        )
                      }
                    />
                  </div>
                  <div className='w-full md:w-48'>
                    <Select
                      value={appStatusFilter}
                      className='w-full'
                      size='small'
                      onChange={(v) => setAppStatusFilter(Number(v))}
                    >
                      <Select.Option value={0}>{t('全部状态')}</Select.Option>
                      <Select.Option value={1}>{t('待审核')}</Select.Option>
                      <Select.Option value={2}>{t('已通过')}</Select.Option>
                      <Select.Option value={3}>{t('已驳回')}</Select.Option>
                    </Select>
                  </div>
                  <div className='flex gap-2 w-full md:w-auto'>
                    <Button
                      type='tertiary'
                      size='small'
                      className='flex-1 md:flex-initial'
                      loading={appLoading}
                      onClick={() => loadApps()}
                    >
                      {t('查询')}
                    </Button>
                    <Button
                      type='tertiary'
                      size='small'
                      className='flex-1 md:flex-initial'
                      onClick={() => {
                        setAppKeyword('');
                        setAppStatusFilter(0);
                        setAppPage(1);
                      }}
                    >
                      {t('重置')}
                    </Button>
                  </div>
                </div>
              </div>
            }
            paginationArea={createCardProPagination({
              currentPage: appPage,
              pageSize: appPageSize,
              total: appTotal,
              onPageChange: (p) => setAppPage(p),
              onPageSizeChange: (ps) => {
                setAppPageSize(ps);
                setAppPage(1);
              },
              isMobile,
              t,
            })}
            t={t}
          >
            <CardTable
              rowKey='id'
              loading={appLoading}
              columns={appColumns}
              dataSource={appRows}
              hidePagination
              empty={tableEmpty}
              className='w-full min-w-0 overflow-hidden'
              style={{ width: '100%' }}
              size='middle'
            />
          </CardPro>
        </Tabs.TabPane>
        <Tabs.TabPane tab={t('提现审核')} itemKey='wd'>
          <CardPro
            className='w-full'
            type='type1'
            actionsArea={
              <div className='flex flex-col md:flex-row justify-end items-center gap-2 w-full'>
                <div className='flex flex-col md:flex-row items-center gap-2 w-full md:w-auto'>
                  <div className='relative w-full md:w-64'>
                    <Input
                      value={wdKeyword}
                      maxLength={ADMIN_KEYWORD_MAX_LEN}
                      prefix={<IconSearch />}
                      showClear
                      pure
                      size='small'
                      placeholder={t('搜索姓名、卡号、用户名')}
                      onChange={(v) =>
                        setWdKeyword(
                          String(v ?? '').slice(0, ADMIN_KEYWORD_MAX_LEN),
                        )
                      }
                    />
                  </div>
                  <div className='w-full md:w-48'>
                    <Select
                      value={wdStatusFilter}
                      className='w-full'
                      size='small'
                      onChange={(v) => setWdStatusFilter(Number(v))}
                    >
                      <Select.Option value={0}>{t('全部状态')}</Select.Option>
                      <Select.Option value={1}>{t('提现中')}</Select.Option>
                      <Select.Option value={2}>{t('提现成功')}</Select.Option>
                      <Select.Option value={3}>{t('提现失败')}</Select.Option>
                      <Select.Option value={4}>{t('已取消')}</Select.Option>
                    </Select>
                  </div>
                  <div className='flex gap-2 w-full md:w-auto'>
                    <Button
                      type='tertiary'
                      size='small'
                      className='flex-1 md:flex-initial'
                      loading={wdLoading}
                      onClick={() => loadWdWithdrawals()}
                    >
                      {t('查询')}
                    </Button>
                    <Button
                      type='tertiary'
                      size='small'
                      className='flex-1 md:flex-initial'
                      onClick={() => {
                        setWdKeyword('');
                        setWdStatusFilter(0);
                        setWdPage(1);
                      }}
                    >
                      {t('重置')}
                    </Button>
                  </div>
                </div>
              </div>
            }
            paginationArea={createCardProPagination({
              currentPage: wdPage,
              pageSize: wdPs,
              total: wdTotal,
              onPageChange: (p) => setWdPage(p),
              onPageSizeChange: (ps) => {
                setWdPs(ps);
                setWdPage(1);
              },
              isMobile,
              t,
            })}
            t={t}
          >
            <CardTable
              rowKey='id'
              loading={wdLoading}
              columns={wdColumns}
              dataSource={wdRows}
              hidePagination
              empty={tableEmpty}
              className='w-full min-w-0 overflow-hidden'
              style={{ width: '100%' }}
              size='middle'
            />
          </CardPro>
        </Tabs.TabPane>
        <Tabs.TabPane tab={t('分销商人员')} itemKey='dist'>
          <CardPro
            className='w-full'
            type='type1'
            actionsArea={
              <div className='flex flex-col md:flex-row justify-end items-center gap-2 w-full'>
                <div className='flex flex-col md:flex-row items-center gap-2 w-full md:w-auto'>
                  <div className='relative w-full md:w-64'>
                    <Input
                      value={distKeyword}
                      maxLength={ADMIN_KEYWORD_MAX_LEN}
                      prefix={<IconSearch />}
                      showClear
                      pure
                      size='small'
                      placeholder={t(
                        '搜索用户名、显示名、申请真实姓名、联系方式、身份证',
                      )}
                      onChange={(v) =>
                        setDistKeyword(
                          String(v ?? '').slice(0, ADMIN_KEYWORD_MAX_LEN),
                        )
                      }
                    />
                  </div>
                  <div className='flex gap-2 w-full md:w-auto'>
                    <Button
                      type='tertiary'
                      size='small'
                      className='flex-1 md:flex-initial'
                      loading={distLoading}
                      onClick={() => loadDists()}
                    >
                      {t('查询')}
                    </Button>
                    <Button
                      type='tertiary'
                      size='small'
                      className='flex-1 md:flex-initial'
                      onClick={() => {
                        setDistKeyword('');
                        setDistPage(1);
                      }}
                    >
                      {t('重置')}
                    </Button>
                  </div>
                </div>
              </div>
            }
            paginationArea={createCardProPagination({
              currentPage: distPage,
              pageSize: distPageSize,
              total: distTotal,
              onPageChange: (p) => setDistPage(p),
              onPageSizeChange: (ps) => {
                setDistPageSize(ps);
                setDistPage(1);
              },
              isMobile,
              t,
            })}
            t={t}
          >
            <CardTable
              rowKey='id'
              loading={distLoading}
              columns={distColumns}
              dataSource={distRows}
              hidePagination
              empty={tableEmpty}
              className='w-full min-w-0 overflow-hidden'
              style={{ width: '100%' }}
              size='middle'
            />
          </CardPro>
        </Tabs.TabPane>
      </Tabs>

      <Modal
        title={t('申请详情')}
        visible={detailOpen}
        onCancel={() => setDetailOpen(false)}
        footer={<Button onClick={() => setDetailOpen(false)}>{t('关闭')}</Button>}
        width={800}
      >
        {detail && (
          <div className='space-y-3 text-sm'>
            <div>
              <Text strong>ID</Text>：{detail.id}
            </div>
            <div>
              <Text strong>{t('用户名')}</Text>：{detail.username}
            </div>
            <div>
              <Text strong>{t('姓名')}</Text>：{detail.real_name}
            </div>
            <div>
              <Text strong>{t('身份证')}</Text>：{detail.id_card_no}
            </div>
            <div>
              <Text strong>{t('联系方式')}</Text>：{detail.contact}
            </div>
            <div>
              <Text strong className='block mb-2'>{t('资格证书')}</Text>
              <QualificationThumbnails
                urls={parseQualificationUrls(detail.qualification_urls)}
                onImagePreview={(u) => setQualImagePreview(u)}
              />
            </div>
          </div>
        )}
      </Modal>

      <Modal
        title={t('分销商申请资料')}
        visible={profileOpen}
        onCancel={() => setProfileOpen(false)}
        width={720}
        footer={
          <div className='flex justify-end gap-2'>
            <Button onClick={() => setProfileOpen(false)}>{t('关闭')}</Button>
            <Button
              type='primary'
              loading={profileSaving}
              onClick={saveDistributorProfile}
            >
              {t('保存资料')}
            </Button>
          </div>
        }
      >
        {profileLoading ? (
          <div className='py-8 text-center text-[var(--semi-color-text-2)]'>
            {t('加载中')}…
          </div>
        ) : (
          <div className='space-y-3 text-sm'>
            <div>
              <Text strong>ID</Text>：{profileUserId}
            </div>
            <div>
              <Text strong>{t('用户名')}</Text>：{profileUsername}
            </div>
            {profileAppStatus != null ? (
              <div>
                <Text strong>{t('申请状态')}</Text>：{appStatus(profileAppStatus)}
              </div>
            ) : null}
            {profileNeedsManual ? (
              <Banner
                type='warning'
                fullMode={false}
                bordered
                description={t(
                  '该用户为管理员手动开通分销商，暂无完整申请资料，请代为补全并保存。',
                )}
              />
            ) : null}
            <div>
              <div className='mb-1'>
                <Text strong>{t('姓名')}</Text>
              </div>
              <Input
                value={profileRealName}
                onChange={(v) => setProfileRealName(String(v ?? ''))}
                placeholder={t('真实姓名')}
              />
            </div>
            <div>
              <div className='mb-1'>
                <Text strong>{t('身份证号码')}</Text>
              </div>
              <Input
                value={profileIdCard}
                onChange={(v) => setProfileIdCard(String(v ?? ''))}
                placeholder={t('身份证号码')}
              />
            </div>
            <div>
              <div className='mb-1'>
                <Text strong>{t('联系方式')}</Text>
              </div>
              <Input
                value={profileContact}
                onChange={(v) => setProfileContact(String(v ?? ''))}
                placeholder={t('手机或邮箱等')}
              />
            </div>
            <div>
              <Text strong className='block mb-2'>
                {t('资格证书')}
              </Text>
              <Upload
                action=''
                accept='image/*,.pdf'
                showUploadList={false}
                customRequest={async ({ file, onSuccess, onError }) => {
                  const fd = new FormData();
                  const inst = file.fileInstance || file;
                  fd.append('file', inst);
                  try {
                    const res = await API.post('/api/oss/upload', fd, {
                      skipErrorHandler: true,
                    });
                    const { success, message, data } = res.data || {};
                    if (!success || !data?.url) {
                      onError(new Error(message || 'upload'));
                      showError(message || t('上传失败'));
                      return;
                    }
                    setProfileQualUrls((prev) =>
                      prev.length >= PROFILE_QUAL_MAX_FILES
                        ? prev
                        : [...prev, data.url],
                    );
                    onSuccess(data);
                    showSuccess(t('已上传'));
                  } catch (e) {
                    onError(e);
                    showError(e?.response?.data?.message || t('上传失败'));
                  }
                }}
                limit={PROFILE_QUAL_MAX_FILES}
                multiple
                disabled={profileQualUrls.length >= PROFILE_QUAL_MAX_FILES}
              >
                <Button
                  disabled={profileQualUrls.length >= PROFILE_QUAL_MAX_FILES}
                >
                  {t('上传文件')}
                </Button>
              </Upload>
              <Text type='tertiary' size='small' className='block mt-1'>
                {t('支持图片或 PDF，最多 5 个；点击图片可大图预览')}
              </Text>
              {profileQualUrls.length > 0 && (
                <div className='mt-3 flex flex-wrap gap-3'>
                  {profileQualUrls.map((u, idx) =>
                    isPdfUrl(u) ? (
                      <div
                        key={`prof-pdf-${u}-${idx}`}
                        className='relative flex h-24 w-24 flex-col items-center justify-center rounded-lg border border-[var(--semi-color-border)] bg-[var(--semi-color-fill-0)]'
                      >
                        <IconFile size='large' />
                        <span className='mt-1 text-xs text-[var(--semi-color-text-2)]'>
                          PDF
                        </span>
                        <button
                          type='button'
                          className='absolute inset-0 rounded-lg focus:outline-none focus-visible:ring-2 focus-visible:ring-primary'
                          title={t('在新窗口打开')}
                          onClick={() =>
                            window.open(u, '_blank', 'noopener,noreferrer')
                          }
                        />
                        <Button
                          size='small'
                          type='danger'
                          theme='borderless'
                          className='!absolute -right-1 -top-1 !min-w-0 z-10'
                          onClick={(e) => {
                            e.stopPropagation();
                            setProfileQualUrls((prev) =>
                              prev.filter((_, i) => i !== idx),
                            );
                          }}
                        >
                          ×
                        </Button>
                      </div>
                    ) : (
                      <div
                        key={`prof-img-${u}-${idx}`}
                        className='relative h-24 w-24 overflow-hidden rounded-lg border border-[var(--semi-color-border)] bg-[var(--semi-color-fill-0)]'
                      >
                        <button
                          type='button'
                          className='block h-full w-full cursor-zoom-in border-0 bg-transparent p-0'
                          onClick={() => setQualImagePreview(u)}
                        >
                          <img
                            src={u}
                            alt=''
                            className='h-full w-full object-cover'
                          />
                        </button>
                        <Button
                          size='small'
                          type='danger'
                          theme='borderless'
                          className='!absolute -right-1 -top-1 !min-w-0'
                          onClick={(e) => {
                            e.stopPropagation();
                            setProfileQualUrls((prev) =>
                              prev.filter((_, i) => i !== idx),
                            );
                          }}
                        >
                          ×
                        </Button>
                      </div>
                    ),
                  )}
                </div>
              )}
            </div>
          </div>
        )}
      </Modal>

      <Modal
        title={t('预览')}
        visible={Boolean(qualImagePreview)}
        onCancel={() => setQualImagePreview(null)}
        footer={null}
        width={Math.min(
          960,
          typeof window !== 'undefined' ? window.innerWidth - 48 : 960,
        )}
      >
        {qualImagePreview ? (
          <div className='flex max-h-[85vh] justify-center overflow-auto p-2'>
            <img
              src={qualImagePreview}
              alt=''
              className='max-h-[85vh] max-w-full object-contain'
            />
          </div>
        ) : null}
      </Modal>

      <Modal
        title={t('驳回原因')}
        visible={rejectOpen}
        onOk={submitReject}
        onCancel={() => setRejectOpen(false)}
      >
        <TextArea
          value={rejectReason}
          onChange={setRejectReason}
          placeholder={t('请填写驳回原因')}
          rows={3}
        />
      </Modal>

      <Modal
        title={t('驳回提现')}
        visible={wdRejectOpen}
        onOk={wdSubmitReject}
        onCancel={() => setWdRejectOpen(false)}
      >
        <Text type='tertiary' size='small' className='block mb-2'>
          {t('驳回后将退回该笔暂扣的分成收益')}
        </Text>
        <TextArea
          value={wdRejectReason}
          onChange={setWdRejectReason}
          placeholder={t('请填写驳回原因')}
          rows={3}
        />
      </Modal>

      <Modal
        title={t('设置默认分销比例')}
        visible={bpsOpen}
        onOk={saveBps}
        onCancel={() => setBpsOpen(false)}
      >
        <Text type='tertiary' size='small'>
          {t('填写 0～100 之间的百分比，例如 10 表示 10%，10.5 表示 10.5%。填 0 表示跟随系统默认。')}
        </Text>
        <Input
          className='mt-2'
          value={bpsPercentInput}
          onChange={(v) => setBpsPercentInput(String(v ?? ''))}
          suffix='%'
          placeholder={t('如 10 或 10.5')}
        />
      </Modal>

      <Modal
        title={t('邀请用户明细')}
        visible={invOpen}
        onCancel={() => setInvOpen(false)}
        footer={null}
        width={800}
      >
        <Table
          columns={invColumns}
          dataSource={invRows}
          pagination={{
            currentPage: invPage,
            pageSize: invPs,
            total: invTotal,
            onPageChange: (p) =>
              invDistributorId && fetchInvitees(invDistributorId, p, invPs),
            onPageSizeChange: (ps) =>
              invDistributorId && fetchInvitees(invDistributorId, 1, ps),
          }}
        />
      </Modal>
    </div>
  );
}
