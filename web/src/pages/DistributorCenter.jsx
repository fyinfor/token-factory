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
  useContext,
  useEffect,
  useState,
  useCallback,
  useMemo,
} from 'react';
import {
  Button,
  Card,
  Table,
  Typography,
  Banner,
  Space,
  Modal,
  Input,
  InputNumber,
  Upload,
  Popconfirm,
  Tag,
} from '@douyinfe/semi-ui';
import { QRCodeSVG } from 'qrcode.react';
import {
  TrendingUp,
  BarChart2,
  Users,
  Zap,
  Copy,
} from 'lucide-react';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';
import {
  API,
  showError,
  showSuccess,
  copy,
  formatCommissionRatioPercent,
  renderQuota,
  getQuotaPerUnit,
  quotaToDisplayInputAmount,
  displayInputAmountToQuota,
  userIsDistributorUser,
} from '../helpers';
import { StatusContext } from '../context/Status';
import { UserContext } from '../context/User';
import { useNavigate } from 'react-router-dom';
import AffInviteeCommissionDetailModal from '../components/distributor/AffInviteeCommissionDetailModal';
import TransferModal from '../components/topup/modals/TransferModal';
import { IconFile } from '@douyinfe/semi-icons';

function isPdfUrl(u) {
  return /\.pdf(\?|$)/i.test(u || '');
}

const { Text, Title } = Typography;

export default function DistributorCenter() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [statusState] = useContext(StatusContext);
  const [userState, userDispatch] = useContext(UserContext);
  const [loading, setLoading] = useState(true);
  const [center, setCenter] = useState(null);
  const [rows, setRows] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [withdrawOpen, setWithdrawOpen] = useState(false);
  const [withdrawLogOpen, setWithdrawLogOpen] = useState(false);
  const [wdRealName, setWdRealName] = useState('');
  const [wdBankName, setWdBankName] = useState('');
  const [wdBankAccount, setWdBankAccount] = useState('');
  /** TOKENS：内部点数字符串；法币展示模式：与划转弹窗一致，为展示数值 */
  const [wdQuotaInput, setWdQuotaInput] = useState('');
  const [wdFiatAmount, setWdFiatAmount] = useState(undefined);
  const [wdVoucherUrls, setWdVoucherUrls] = useState([]);
  const [wdUploading, setWdUploading] = useState(false);
  const [wdSubmitting, setWdSubmitting] = useState(false);
  const [wdLogLoading, setWdLogLoading] = useState(false);
  const [wdLogRows, setWdLogRows] = useState([]);
  const [wdLogTotal, setWdLogTotal] = useState(0);
  const [wdLogPage, setWdLogPage] = useState(1);
  const [wdLogPs, setWdLogPs] = useState(10);
  const [wdVoucherPreview, setWdVoucherPreview] = useState(null);
  const [detailOpen, setDetailOpen] = useState(false);
  const [detailInviteeId, setDetailInviteeId] = useState(null);
  const [detailInviteeLabel, setDetailInviteeLabel] = useState('');
  const [openTransfer, setOpenTransfer] = useState(false);
  const [transferAmount, setTransferAmount] = useState(() =>
    getQuotaPerUnit(),
  );

  const withdrawImg = (
    statusState?.status?.distributor_withdraw_cs_image_url || ''
  ).trim();

  const withdrawNotice = (
    statusState?.status?.distributor_withdraw_notice || ''
  ).trim();

  const minWithdrawQuota = (() => {
    const n = Number(statusState?.status?.distributor_min_withdraw_quota);
    if (Number.isFinite(n) && n > 0) return Math.floor(n);
    let q = getQuotaPerUnit();
    if (!Number.isFinite(q) || q <= 0) {
      q = Number(statusState?.status?.quota_per_unit);
    }
    if (!Number.isFinite(q) || q <= 0) {
      q = parseFloat(localStorage.getItem('quota_per_unit') || '') || 500000;
    }
    return q;
  })();

  const isQuotaTokensMode = useMemo(() => {
    if (typeof window === 'undefined') return false;
    return (localStorage.getItem('quota_display_type') || 'USD') === 'TOKENS';
  }, [withdrawOpen]);

  const affQuotaFloor = Math.floor(Number(center?.aff_quota) || 0);
  const minQInternal = minWithdrawQuota;
  /** 法币模式下与划转一致的展示上下界 */
  const wdFiatMin = useMemo(() => {
    if (isQuotaTokensMode || affQuotaFloor <= 0) return undefined;
    if (affQuotaFloor >= minQInternal) {
      return quotaToDisplayInputAmount(minQInternal);
    }
    return quotaToDisplayInputAmount(1);
  }, [isQuotaTokensMode, affQuotaFloor, minQInternal]);
  const wdFiatMax = useMemo(() => {
    if (isQuotaTokensMode || affQuotaFloor <= 0) return undefined;
    return quotaToDisplayInputAmount(affQuotaFloor);
  }, [isQuotaTokensMode, affQuotaFloor]);

  const isUserDistributor = useMemo(() => {
    const u = userState?.user;
    if (u) return userIsDistributorUser(u);
    try {
      const raw = localStorage.getItem('user');
      if (raw) return userIsDistributorUser(JSON.parse(raw));
    } catch {
      // ignore
    }
    return false;
  }, [userState?.user]);

  const loadCenter = async () => {
    const res = await API.get('/api/distributor/center');
    const { success, message, data } = res.data;
    if (!success) {
      showError(message);
      navigate('/console/topup');
      return;
    }
    setCenter(data);
  };

  const refreshUser = useCallback(async () => {
    const res = await API.get('/api/user/self');
    const { success, message, data } = res.data;
    if (success && data) {
      userDispatch({ type: 'login', payload: data });
      return data;
    }
    if (message) showError(message);
    return null;
  }, [userDispatch]);

  const loadInvitees = useCallback(
    async (p = 1, ps = pageSize) => {
      const res = await API.get(
        `/api/user/aff_invitees?p=${p}&page_size=${ps}`,
      );
      const { success, message, data } = res.data;
      if (!success) {
        showError(message);
        return;
      }
      setRows(data?.items || []);
      setTotal(data?.total ?? 0);
      setPage(p);
    },
    [pageSize],
  );

  useEffect(() => {
    if (!isUserDistributor) {
      navigate('/console/topup');
      return;
    }
    (async () => {
      setLoading(true);
      try {
        await loadCenter();
        await loadInvitees(1, pageSize);
        await refreshUser();
      } finally {
        setLoading(false);
      }
    })();
  }, [isUserDistributor]);

  useEffect(() => {
    setTransferAmount(getQuotaPerUnit());
  }, []);

  const handleWdVoucherUpload = async ({ file, onSuccess, onError }) => {
    const inst = file.fileInstance || file;
    if (!inst) {
      onError(new Error('no file'));
      return;
    }
    setWdUploading(true);
    const fd = new FormData();
    fd.append('file', inst);
    try {
      const res = await API.post('/api/oss/upload', fd, {
        skipErrorHandler: true,
      });
      const body = res.data || {};
      const { success, message, data } = body;
      const payload = data ?? body;
      const url =
        (payload && typeof payload === 'object'
          ? payload.url || payload.URL || payload.link
          : null) || (typeof payload === 'string' ? payload : '');
      if (!url || success === false) {
        showError(message || t('上传失败'));
        onError(new Error(message || 'upload'));
        return;
      }
      setWdVoucherUrls((prev) => [...prev, String(url).trim()]);
      onSuccess({ url: String(url).trim() });
      showSuccess(t('已上传'));
    } catch (e) {
      const msg =
        e?.response?.data?.message ||
        e?.message ||
        t('上传失败，请确认已启用 OSS 并完成配置');
      showError(msg);
      onError(e);
    } finally {
      setWdUploading(false);
    }
  };

  const submitWithdraw = async () => {
    if (!wdRealName.trim() || !wdBankName.trim() || !wdBankAccount.trim()) {
      showError(t('请填写真实姓名、开户行与银行卡号'));
      return;
    }
    const voucherList = wdVoucherUrls
      .map((u) => String(u || '').trim())
      .filter(Boolean);
    if (!voucherList.length) {
      showError(t('请上传票据'));
      return;
    }

    const maxQ = Math.floor(Number(center?.aff_quota) || 0);
    const minQ = minWithdrawQuota;
    let q;

    if (isQuotaTokensMode) {
      const trimmedAmt = String(wdQuotaInput).trim();
      if (trimmedAmt === '') {
        showError(t('请填写提现额度'));
        return;
      }
      if (!/^\d+$/.test(trimmedAmt)) {
        showError(t('提现额度须为正整数'));
        return;
      }
      q = parseInt(trimmedAmt, 10);
      if (q <= 0) {
        showError(t('提现额度须为正整数'));
        return;
      }
    } else {
      if (
        wdFiatAmount == null ||
        Number.isNaN(Number(wdFiatAmount)) ||
        Number(wdFiatAmount) <= 0
      ) {
        showError(t('请填写与上方待使用收益展示一致的提现金额'));
        return;
      }
      const fiatRounded =
        Math.round(Number(wdFiatAmount) * 100) / 100;
      q = displayInputAmountToQuota(fiatRounded);
      if (q <= 0) {
        showError(t('提现金额过小或无效'));
        return;
      }
    }

    if (maxQ >= minQ) {
      if (q < minQ) {
        showError(
          t('提现额度不能低于') + ' ' + renderQuota(minQ),
        );
        return;
      }
    } else {
      if (q < 1 || q > maxQ) {
        showError(t('提现额度须在 1 与当前待使用余额之间'));
        return;
      }
    }
    if (q > maxQ) {
      showError(
        t('提现额度不能大于当前可提现额度') +
          '（' +
          renderQuota(maxQ) +
          '）',
      );
      return;
    }
    setWdSubmitting(true);
    try {
      const res = await API.post('/api/distributor/withdrawal', {
        real_name: wdRealName.trim(),
        bank_name: wdBankName.trim(),
        bank_account: wdBankAccount.trim(),
        voucher_urls: voucherList,
        quota_amount: Math.round(Number(q)),
      });
      if (res.data.success) {
        showSuccess(t('已提交，请等待审核'));
        setWithdrawOpen(false);
        setWdVoucherUrls([]);
        setWdQuotaInput('');
        setWdFiatAmount(undefined);
        await loadCenter();
        await refreshUser();
        if (withdrawLogOpen) {
          await loadWithdrawLogs(wdLogPage, wdLogPs);
        }
      } else {
        showError(res.data.message);
      }
    } catch {
      showError(t('提交失败'));
    } finally {
      setWdSubmitting(false);
    }
  };

  const loadWithdrawLogs = async (p = 1, ps = wdLogPs) => {
    setWdLogLoading(true);
    try {
      const res = await API.get(
        `/api/distributor/withdrawals?p=${p}&page_size=${ps}`,
      );
      const { success, message, data } = res.data;
      if (!success) {
        showError(message);
        return;
      }
      setWdLogRows(data?.items || []);
      setWdLogTotal(data?.total ?? 0);
      setWdLogPage(p);
    } catch {
      showError(t('加载失败'));
    } finally {
      setWdLogLoading(false);
    }
  };

  const cancelWithdraw = async (id) => {
    try {
      const res = await API.post(`/api/distributor/withdrawals/${id}/cancel`);
      if (res.data.success) {
        showSuccess(t('已取消'));
        await loadCenter();
        await refreshUser();
        await loadWithdrawLogs(wdLogPage, wdLogPs);
      } else {
        showError(res.data.message);
      }
    } catch {
      showError(t('操作失败'));
    }
  };

  const wdStatusText = (s) => {
    if (s === 1) return t('提现中');
    if (s === 2) return t('提现成功');
    if (s === 3) return t('提现失败');
    if (s === 4) return t('已取消');
    return '—';
  };

  /** 提现记录表格：状态 Tag（成功/失败/中等） */
  const wdStatusTag = (s) => {
    const label = wdStatusText(s);
    if (s === 1) {
      return (
        <Tag color='blue' type='light' size='small'>
          {label}
        </Tag>
      );
    }
    if (s === 2) {
      return (
        <Tag color='green' type='light' size='small'>
          {label}
        </Tag>
      );
    }
    if (s === 3) {
      return (
        <Tag color='red' type='light' size='small'>
          {label}
        </Tag>
      );
    }
    if (s === 4) {
      return (
        <Tag color='grey' type='light' size='small'>
          {label}
        </Tag>
      );
    }
    return (
      <Tag size='small' type='light'>
        {label}
      </Tag>
    );
  };

  const openTransferModal = useCallback(async () => {
    await refreshUser();
    setTransferAmount(getQuotaPerUnit());
    setOpenTransfer(true);
  }, [refreshUser]);

  const handleTransfer = async () => {
    if (transferAmount < getQuotaPerUnit()) {
      showError(
        t('划转金额最低为') + ' ' + renderQuota(getQuotaPerUnit()),
      );
      return;
    }
    const res = await API.post('/api/user/aff_transfer', {
      quota: transferAmount,
    });
    const { success, message } = res.data;
    if (success) {
      showSuccess(message);
      setOpenTransfer(false);
      await refreshUser();
      await loadCenter();
    } else {
      showError(message);
    }
  };

  const handleTransferCancel = () => setOpenTransfer(false);

  const shortLink =
    center?.aff_code &&
    `${window.location.origin}/r/${center.aff_code}`;
  const registerLink =
    center?.aff_code &&
    `${window.location.origin}/register?aff=${center.aff_code}`;

  const openDetail = (r) => {
    setDetailInviteeId(r.invitee_id);
    setDetailInviteeLabel(
      String(r.display_name || r.username || `#${r.invitee_id}`).trim(),
    );
    setDetailOpen(true);
  };

  const columns = [
    {
      title: t('被邀请用户'),
      dataIndex: 'username',
      render: (_, r) => r.display_name || r.username || r.invitee_id,
    },
    {
      title: t('邀请时间'),
      dataIndex: 'created_at',
      width: 180,
      render: (ts) =>
        ts ? dayjs.unix(Number(ts)).format('YYYY-MM-DD HH:mm') : '—',
    },
    {
      title: t('累计分成额度'),
      dataIndex: 'commission_earned_quota',
      render: (q) => renderQuota(q || 0),
    },
    {
      title: t('操作'),
      width: 100,
      render: (_, r) => (
        <Button size='small' type='tertiary' onClick={() => openDetail(r)}>
          {t('详情')}
        </Button>
      ),
    },
  ];

  if (!isUserDistributor) {
    return null;
  }

  return (
    <div className='mt-14 px-4 pb-16 max-w-7xl mx-auto'>
      <Title heading={3} className='mb-8'>
        {t('分销中心')}
      </Title>

      <Banner
        type='info'
        className='mt-2 !rounded-xl'
        description={t(
          '邀请的用户充值后，您将获得对应比例的分销额度（待使用收益）。分成比例以您账号当前设置为准；可在「详情」中查看每笔充值的入账额度、当时比例与收益。',
        )}
      />

      <div className='mt-2 flex flex-col xl:flex-row gap-8 xl:gap-10 items-start'>
        {/* 窄屏：右侧栏在上，表格在下；宽屏：左表格、右栏 */}
        <aside className='flex w-full flex-col gap-[10px] xl:w-[520px] flex-shrink-0 order-1 xl:order-2'>
          <Card
            className='!rounded-xl w-full shadow-sm border-0'
            cover={
              <div
                className='relative min-h-[10.5rem]'
                style={{
                  '--palette-primary-darkerChannel': '0 75 80',
                  backgroundImage: `linear-gradient(0deg, rgba(var(--palette-primary-darkerChannel) / 80%), rgba(var(--palette-primary-darkerChannel) / 80%)), url('/cover-4.webp')`,
                  backgroundSize: 'cover',
                  backgroundPosition: 'center',
                  backgroundRepeat: 'no-repeat',
                }}
              >
                <div className='relative z-10 h-full flex flex-col justify-between p-4'>
                  <div className='flex justify-between items-start gap-2'>
                    <div>
                      <Text strong style={{ color: 'white', fontSize: '16px' }}>
                        {t('收益统计')}
                      </Text>
                      <Text
                        style={{
                          color: 'rgba(255,255,255,0.88)',
                          fontSize: '12px',
                          display: 'block',
                          marginTop: 6,
                        }}
                      >
                        {t('当前默认分销比例')}：
                        {formatCommissionRatioPercent(
                          center?.effective_commission_bps ?? 0,
                        )}
                      </Text>
                    </div>
                    <div className='flex flex-wrap items-center justify-end gap-2 flex-shrink-0'>
                      <Button
                        type='primary'
                        theme='solid'
                        size='small'
                        onClick={() => {
                          setWdRealName('');
                          setWdBankName('');
                          setWdBankAccount('');
                          setWdVoucherUrls([]);
                          setWdQuotaInput('');
                          setWdFiatAmount(undefined);
                          setWdVoucherPreview(null);
                          setWithdrawOpen(true);
                        }}
                        className='!rounded-lg !border-0 !bg-emerald-600 hover:!bg-emerald-700 active:!bg-emerald-800 !text-white'
                      >
                        {t('提现')}
                      </Button>
                      <Button
                        type='tertiary'
                        theme='solid'
                        size='small'
                        onClick={async () => {
                          setWithdrawLogOpen(true);
                          await loadWithdrawLogs(1, wdLogPs);
                        }}
                        className='!rounded-lg'
                      >
                        {t('提现记录')}
                      </Button>
                      <Button
                        type='primary'
                        theme='solid'
                        size='small'
                        disabled={
                          !center?.aff_quota || center.aff_quota <= 0
                        }
                        onClick={() => openTransferModal()}
                        className='!rounded-lg'
                      >
                        <Zap size={12} className='mr-1' />
                        {t('划转到余额')}
                      </Button>
                    </div>
                  </div>

                  <div className='grid grid-cols-3 gap-4 sm:gap-6 mt-4'>
                    <div className='text-center'>
                      <div
                        className='text-base sm:text-2xl font-bold mb-2'
                        style={{ color: 'white' }}
                      >
                        {renderQuota(center?.aff_quota || 0)}
                      </div>
                      <div className='flex items-center justify-center text-sm'>
                        <TrendingUp
                          size={14}
                          className='mr-1'
                          style={{ color: 'rgba(255,255,255,0.8)' }}
                        />
                        <Text
                          style={{
                            color: 'rgba(255,255,255,0.8)',
                            fontSize: '12px',
                          }}
                        >
                          {t('待使用收益')}
                        </Text>
                      </div>
                    </div>

                    <div className='text-center'>
                      <div
                        className='text-base sm:text-2xl font-bold mb-2'
                        style={{ color: 'white' }}
                      >
                        {renderQuota(center?.aff_history_quota || 0)}
                      </div>
                      <div className='flex items-center justify-center text-sm'>
                        <BarChart2
                          size={14}
                          className='mr-1'
                          style={{ color: 'rgba(255,255,255,0.8)' }}
                        />
                        <Text
                          style={{
                            color: 'rgba(255,255,255,0.8)',
                            fontSize: '12px',
                          }}
                        >
                          {t('总收益')}
                        </Text>
                      </div>
                    </div>

                    <div className='text-center'>
                      <div
                        className='text-base sm:text-2xl font-bold mb-2'
                        style={{ color: 'white' }}
                      >
                        {center?.aff_count ?? 0}
                      </div>
                      <div className='flex items-center justify-center text-sm'>
                        <Users
                          size={14}
                          className='mr-1'
                          style={{ color: 'rgba(255,255,255,0.8)' }}
                        />
                        <Text
                          style={{
                            color: 'rgba(255,255,255,0.8)',
                            fontSize: '12px',
                          }}
                        >
                          {t('邀请人数')}
                        </Text>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            }
          >
            <Space vertical style={{ width: '100%' }} className='!gap-3'>
              <Input
                value={registerLink || ''}
                readonly
                className='!rounded-lg'
                prefix={t('邀请链接')}
                suffix={
                  <Button
                    type='primary'
                    theme='solid'
                    onClick={async () => {
                      if (registerLink && (await copy(registerLink))) {
                        showSuccess(t('邀请链接已复制到剪切板'));
                      }
                    }}
                    icon={<Copy size={14} />}
                    className='!rounded-lg'
                  >
                    {t('复制')}
                  </Button>
                }
              />
              <Input
                value={shortLink || ''}
                readonly
                className='!rounded-lg'
                prefix={t('短链接')}
                suffix={
                  <Button
                    type='primary'
                    theme='solid'
                    onClick={async () => {
                      if (shortLink && (await copy(shortLink))) {
                        showSuccess(t('已复制'));
                      }
                    }}
                    icon={<Copy size={14} />}
                    className='!rounded-lg'
                  >
                    {t('复制')}
                  </Button>
                }
              />
            </Space>
          </Card>

          <Card
            className='!rounded-2xl'
            title={t('注册二维码')}
            bodyStyle={{ paddingTop: 20, paddingBottom: 24 }}
          >
            <div className='w-full flex flex-col items-center'>
              {registerLink ? (
                <>
                  <div className='inline-flex rounded-xl bg-[var(--semi-color-fill-0)] p-2 ring-1 ring-[var(--semi-color-border)]'>
                    <div
                      className='rounded-lg bg-semi-color-white p-2 shadow-[0_2px_8px_rgba(0,0,0,0.08)] dark:shadow-[0_4px_16px_rgba(0,0,0,0.55)] ring-1 ring-black/[0.06] dark:ring-white/[0.12]'
                      id='dist-qr-wrap'
                    >
                      <QRCodeSVG
                        value={registerLink}
                        size={168}
                        level='M'
                        bgColor='#ffffff'
                        fgColor='#000000'
                      />
                    </div>
                  </div>
                  <Button
                    className='mt-2'
                    size='small'
                    onClick={() => {
                      const svg = document
                        .getElementById('dist-qr-wrap')
                        ?.querySelector('svg');
                      if (!svg) return;
                      const blob = new Blob([svg.outerHTML], {
                        type: 'image/svg+xml;charset=utf-8',
                      });
                      const url = URL.createObjectURL(blob);
                      const a = document.createElement('a');
                      a.href = url;
                      a.download = `invite-${center?.aff_code}.svg`;
                      a.click();
                      URL.revokeObjectURL(url);
                    }}
                  >
                    {t('下载二维码')}
                  </Button>
                </>
              ) : (
                <Text type='tertiary'>—</Text>
              )}
            </div>
          </Card>

        </aside>

        <div className='flex-1 min-w-0 w-full order-2 xl:order-1 mt-2 sm:mt-4 xl:mt-0'>
          <Card
            className='!rounded-2xl'
            title={t('邀请用户列表')}
            loading={loading}
          >
            <Table
              rowKey='invitee_id'
              columns={columns}
              dataSource={rows}
              pagination={{
                currentPage: page,
                pageSize,
                total,
                onPageChange: (p) => loadInvitees(p, pageSize),
                onPageSizeChange: (ps) => {
                  setPageSize(ps);
                  loadInvitees(1, ps);
                },
              }}
            />
          </Card>
        </div>
      </div>

      <AffInviteeCommissionDetailModal
        visible={detailOpen}
        onCancel={() => {
          setDetailOpen(false);
          setDetailInviteeId(null);
          setDetailInviteeLabel('');
        }}
        inviteeId={detailInviteeId}
        inviteeLabel={detailInviteeLabel}
      />

      <TransferModal
        t={t}
        openTransfer={openTransfer}
        transfer={handleTransfer}
        handleTransferCancel={handleTransferCancel}
        userState={userState}
        renderQuota={renderQuota}
        getQuotaPerUnit={getQuotaPerUnit}
        transferAmount={transferAmount}
        setTransferAmount={setTransferAmount}
      />

      <Modal
        title={t('线下提现')}
        visible={withdrawOpen}
        onCancel={() => setWithdrawOpen(false)}
        footer={
          <Space>
            <Button onClick={() => setWithdrawOpen(false)}>{t('关闭')}</Button>
            <Button
              type='primary'
              theme='solid'
              loading={wdSubmitting}
              onClick={submitWithdraw}
            >
              {t('提交申请')}
            </Button>
          </Space>
        }
        width={Math.min(
          960,
          typeof window !== 'undefined' ? window.innerWidth - 48 : 960,
        )}
      >
        <Text type='tertiary' size='small' className='block mb-4'>
          {t(
            '提交后将暂扣对应待使用收益，审核通过即完成提现；驳回或取消将退回额度。',
          )}
        </Text>
        <div className='flex flex-col lg:flex-row gap-6 items-start'>
          <div className='flex-1 min-w-0 w-full space-y-3'>
            <Input
              value={wdRealName}
              onChange={(v) => setWdRealName(String(v ?? ''))}
              placeholder={t('真实姓名（必填）')}
            />
            <Input
              value={wdBankName}
              onChange={(v) => setWdBankName(String(v ?? ''))}
              placeholder={t('开户行（必填）')}
            />
            <Input
              value={wdBankAccount}
              onChange={(v) => setWdBankAccount(String(v ?? ''))}
              placeholder={t('银行卡号（必填）')}
            />
            <div>
              <Text size='small' className='block mb-1'>
                {t('提现余额（必填）')}
              </Text>
              {isQuotaTokensMode ? (
                <>
                  <Text type='tertiary' size='small' className='block mb-2'>
                    {t(
                      'TOKENS 模式：与上方「待使用收益」数字一致，填写系统内部点数（正整数）',
                    )}
                  </Text>
                  <Input
                    value={wdQuotaInput}
                    onChange={(v) =>
                      setWdQuotaInput(v == null ? '' : String(v))
                    }
                    placeholder={t('请输入内部点数')}
                  />
                </>
              ) : (
                <>
                  <Text type='tertiary' size='small' className='block mb-2'>
        
                  </Text>
                  <InputNumber
                    style={{ width: '100%' }}
                    value={wdFiatAmount}
                    onChange={(v) => setWdFiatAmount(v)}
                    min={wdFiatMin}
                    max={wdFiatMax}
                    precision={2}
                    step={0.01}
                    placeholder={t('填写收益金额')}
                  />
                </>
              )}
              <Text type='tertiary' size='small' className='block mt-1'>
                {affQuotaFloor >= minQInternal ? (
                  <>
                    {t('单笔最低（展示）')}: {renderQuota(minQInternal)} ·{' '}
                  </>
                ) : (
                  <>
                    {t('当前余额低于系统最低门槛时，可提范围（展示）')}:{' '}
                    {renderQuota(1)}～{renderQuota(affQuotaFloor)} ·{' '}
                  </>
                )}
                {t('当前待使用余额')}:{' '}
                {renderQuota(center?.aff_quota || 0)}
              </Text>
            </div>
            <div>
              <Text size='small' className='block mb-1'>
                {t('票据')}
              </Text>
              <Upload
                accept='image/*,.pdf'
                multiple
                showUploadList={false}
                customRequest={handleWdVoucherUpload}
              >
                <Button loading={wdUploading}>{t('上传票据')}</Button>
              </Upload>
              <Text type='tertiary' size='small' className='block mt-1'>
                {t('支持图片或 PDF；点击图片可大图预览')}
              </Text>
              {withdrawNotice ? (
                <div className='mt-3 rounded-lg border border-[var(--semi-color-border)] bg-[var(--semi-color-fill-0)] px-3 py-2'>
                  <Text strong size='small' className='block mb-1'>
                    {t('提现说明')}
                  </Text>
                  <Text
                    type='secondary'
                    size='small'
                    className='block whitespace-pre-wrap break-words'
                  >
                    {withdrawNotice}
                  </Text>
                </div>
              ) : null}
              {wdVoucherUrls.length > 0 ? (
                <div className='mt-3 flex flex-wrap gap-3'>
                  {wdVoucherUrls.map((u, idx) =>
                    isPdfUrl(u) ? (
                      <div
                        key={`wd-v-${u}-${idx}`}
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
                            setWdVoucherUrls((prev) =>
                              prev.filter((_, i) => i !== idx),
                            );
                          }}
                        >
                          ×
                        </Button>
                      </div>
                    ) : (
                      <div
                        key={`wd-v-${u}-${idx}`}
                        className='relative h-24 w-24 overflow-hidden rounded-lg border border-[var(--semi-color-border)] bg-[var(--semi-color-fill-0)]'
                      >
                        <button
                          type='button'
                          className='block h-full w-full cursor-zoom-in border-0 bg-transparent p-0'
                          onClick={() => setWdVoucherPreview(u)}
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
                          className='!absolute -right-1 -top-1 !min-w-0 z-10'
                          onClick={(e) => {
                            e.stopPropagation();
                            setWdVoucherUrls((prev) =>
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
              ) : null}
            </div>
          </div>
          <div className='w-full lg:w-[280px] flex-shrink-0'>
            <Text strong className='block mb-2'>
              {t('联系客服')}
            </Text>
            {withdrawImg ? (
              <div className='rounded-lg border border-[var(--semi-color-border)] bg-[var(--semi-color-fill-0)] p-2 overflow-auto max-h-[min(420px,50vh)]'>
                <img
                  src={withdrawImg}
                  alt=''
                  className='w-full object-contain'
                />
              </div>
            ) : (
              <Text type='tertiary' size='small'>
                {t('管理员可在运营设置中配置右侧客服图片')}
              </Text>
            )}
          </div>
        </div>
      </Modal>

      <Modal
        title={t('提现记录')}
        visible={withdrawLogOpen}
        onCancel={() => setWithdrawLogOpen(false)}
        footer={
          <Button onClick={() => setWithdrawLogOpen(false)}>
            {t('关闭')}
          </Button>
        }
        width={960}
      >
        <Table
          loading={wdLogLoading}
          size='small'
          rowKey='id'
          columns={[
            {
              title: t('额度'),
              dataIndex: 'quota_amount',
              width: 120,
              render: (q) => renderQuota(q || 0),
            },
            { title: t('提现月份'), dataIndex: 'withdraw_month', width: 100 },
            {
              title: t('收款信息'),
              render: (_, r) => (
                <div className='text-xs space-y-0.5'>
                  <div>{r.real_name}</div>
                  <div className='text-[var(--semi-color-text-2)]'>
                    {r.bank_name} {r.bank_account}
                  </div>
                </div>
              ),
            },
            {
              title: t('时间'),
              dataIndex: 'created_at',
              width: 158,
              render: (ts) =>
                ts
                  ? dayjs.unix(Number(ts)).format('YYYY-MM-DD HH:mm')
                  : '—',
            },
            {
              title: t('反馈原因'),
              width: 200,
              render: (_, r) => {
                const reason = String(r.reject_reason || '').trim();
                if (reason) {
                  return (
                    <Text size='small' className='break-all text-[var(--semi-color-text-0)]'>
                      {reason}
                    </Text>
                  );
                }
                return (
                  <Text type='tertiary' size='small'>
                    —
                  </Text>
                );
              },
            },
            {
              title: t('状态'),
              dataIndex: 'status',
              width: 112,
              render: (s) => wdStatusTag(s),
            },
            {
              title: t('操作'),
              width: 100,
              align: 'center',
              render: (_, r) =>
                r.status === 1 ? (
                  <Popconfirm
                    title={t('确认取消该笔提现？额度将退回')}
                    onConfirm={() => cancelWithdraw(r.id)}
                  >
                    <Tag color='red' type='light' size='small' className='cursor-pointer'>
                      {t('取消')}
                    </Tag>
                  </Popconfirm>
                ) : (
                  <Text type='tertiary' size='small'>
                    —
                  </Text>
                ),
            },
          ]}
          dataSource={wdLogRows}
          pagination={{
            currentPage: wdLogPage,
            pageSize: wdLogPs,
            total: wdLogTotal,
            onPageChange: (p) => loadWithdrawLogs(p, wdLogPs),
            onPageSizeChange: (ps) => {
              setWdLogPs(ps);
              loadWithdrawLogs(1, ps);
            },
          }}
        />
      </Modal>

      <Modal
        title={t('预览')}
        visible={Boolean(wdVoucherPreview)}
        onCancel={() => setWdVoucherPreview(null)}
        footer={null}
        width={Math.min(
          900,
          typeof window !== 'undefined' ? window.innerWidth - 48 : 900,
        )}
      >
        {wdVoucherPreview && !isPdfUrl(wdVoucherPreview) ? (
          <div className='flex max-h-[85vh] justify-center overflow-auto p-2'>
            <img
              src={wdVoucherPreview}
              alt=''
              className='max-h-[85vh] max-w-full object-contain'
            />
          </div>
        ) : null}
      </Modal>
    </div>
  );
}
