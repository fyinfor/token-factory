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
  useRef,
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
  Popconfirm,
  Tag,
  Spin,
} from '@douyinfe/semi-ui';
import { QRCodeSVG } from 'qrcode.react';
import {
  TrendingUp,
  BarChart2,
  Users,
  Zap,
  Copy,
  Download,
  Wallet,
  History,
  UserPlus,
  BadgeCheck,
  Building2,
  Clock3,
  UserRound,
  XCircle,
  ReceiptText,
  SlidersHorizontal,
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
  isValidPhoneNumber,
} from '../helpers';
import { StatusContext } from '../context/Status';
import { UserContext } from '../context/User';
import { useNavigate } from 'react-router-dom';
import AffInviteeCommissionDetailModal from '../components/distributor/AffInviteeCommissionDetailModal';
import InviteeModelDiscountModal from '../components/distributor/InviteeModelDiscountModal';
import InviteeTopupHistoryModal from '../components/distributor/InviteeTopupHistoryModal';
import DistributorWithdrawFormFields from '../components/distributor/DistributorWithdrawFormFields';
import DistributorApplyFileUpload from '../components/distributor/DistributorApplyFileUpload';
import {
  WD_ACCOUNT_PERSONAL,
  WD_ACCOUNT_ENTERPRISE,
  EMPTY_WD_FORM,
  validateWithdrawForm,
  buildWithdrawSubmitBody,
  wdAccountTypeLabel,
  parseWithdrawRow,
  parseVoucherUrls,
} from '../components/distributor/withdrawProfileUtils';
import TransferModal from '../components/topup/modals/TransferModal';

function isPdfUrl(u) {
  return /\.pdf(\?|$)/i.test(u || '');
}

const { Text, Title } = Typography;

const QR_CODE_SIZE = 168;
const QR_DOWNLOAD_PADDING = 20;
const QR_DOWNLOAD_BORDER_WIDTH = 2;
const QR_DOWNLOAD_BORDER_RADIUS = 14;
const IDENTITY_APP_STATUS_PENDING = 1;
const IDENTITY_APP_STATUS_APPROVED = 2;
const IDENTITY_APP_STATUS_REJECTED = 3;

function distributorIdentityLabel(t, type) {
  return Number(type) === WD_ACCOUNT_ENTERPRISE ? t('企业') : t('个人');
}

function oppositeDistributorIdentity(type) {
  return Number(type) === WD_ACCOUNT_ENTERPRISE
    ? WD_ACCOUNT_PERSONAL
    : WD_ACCOUNT_ENTERPRISE;
}

function drawRoundedRect(ctx, x, y, width, height, radius) {
  const r = Math.min(radius, width / 2, height / 2);
  ctx.beginPath();
  ctx.moveTo(x + r, y);
  ctx.lineTo(x + width - r, y);
  ctx.quadraticCurveTo(x + width, y, x + width, y + r);
  ctx.lineTo(x + width, y + height - r);
  ctx.quadraticCurveTo(x + width, y + height, x + width - r, y + height);
  ctx.lineTo(x + r, y + height);
  ctx.quadraticCurveTo(x, y + height, x, y + height - r);
  ctx.lineTo(x, y + r);
  ctx.quadraticCurveTo(x, y, x + r, y);
  ctx.closePath();
}

function downloadBlob(blob, filename) {
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}

function downloadQrSvgAsPng(svg, filename) {
  return new Promise((resolve, reject) => {
    const svgText = new XMLSerializer().serializeToString(svg);
    const svgBlob = new Blob([svgText], {
      type: 'image/svg+xml;charset=utf-8',
    });
    const svgUrl = URL.createObjectURL(svgBlob);
    const image = new Image();

    image.onload = () => {
      try {
        const canvas = document.createElement('canvas');
        const totalSize = QR_CODE_SIZE + QR_DOWNLOAD_PADDING * 2;
        const dpr = Math.max(window.devicePixelRatio || 1, 1);
        canvas.width = totalSize * dpr;
        canvas.height = totalSize * dpr;

        const ctx = canvas.getContext('2d');
        if (!ctx) {
          reject(new Error('canvas context unavailable'));
          return;
        }

        ctx.scale(dpr, dpr);
        ctx.fillStyle = '#ffffff';
        drawRoundedRect(
          ctx,
          0,
          0,
          totalSize,
          totalSize,
          QR_DOWNLOAD_BORDER_RADIUS,
        );
        ctx.fill();

        ctx.strokeStyle = '#d9d9d9';
        ctx.lineWidth = QR_DOWNLOAD_BORDER_WIDTH;
        drawRoundedRect(
          ctx,
          QR_DOWNLOAD_BORDER_WIDTH / 2,
          QR_DOWNLOAD_BORDER_WIDTH / 2,
          totalSize - QR_DOWNLOAD_BORDER_WIDTH,
          totalSize - QR_DOWNLOAD_BORDER_WIDTH,
          QR_DOWNLOAD_BORDER_RADIUS,
        );
        ctx.stroke();

        ctx.drawImage(
          image,
          QR_DOWNLOAD_PADDING,
          QR_DOWNLOAD_PADDING,
          QR_CODE_SIZE,
          QR_CODE_SIZE,
        );

        canvas.toBlob((blob) => {
          if (!blob) {
            reject(new Error('png export failed'));
            return;
          }
          downloadBlob(blob, filename);
          resolve();
        }, 'image/png');
      } catch (e) {
        reject(e);
      } finally {
        URL.revokeObjectURL(svgUrl);
      }
    };

    image.onerror = () => {
      URL.revokeObjectURL(svgUrl);
      reject(new Error('qr image load failed'));
    };
    image.src = svgUrl;
  });
}

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
  const [withdrawNoticeLoading, setWithdrawNoticeLoading] = useState(false);
  const [withdrawNoticeContent, setWithdrawNoticeContent] = useState('');
  const [withdrawLogOpen, setWithdrawLogOpen] = useState(false);
  const [wdAccountType, setWdAccountType] = useState(WD_ACCOUNT_PERSONAL);
  const [wdForm, setWdForm] = useState({ ...EMPTY_WD_FORM });
  /** TOKENS：内部点数字符串；法币展示模式：与划转弹窗一致，为展示数值 */
  const [wdQuotaInput, setWdQuotaInput] = useState('');
  const [wdFiatAmount, setWdFiatAmount] = useState(undefined);
  const [wdSubmitting, setWdSubmitting] = useState(false);
  const wdFiatCommitRef = useRef(null);
  const [wdLogLoading, setWdLogLoading] = useState(false);
  const [wdLogRows, setWdLogRows] = useState([]);
  const [wdLogTotal, setWdLogTotal] = useState(0);
  const [wdLogPage, setWdLogPage] = useState(1);
  const [wdLogPs, setWdLogPs] = useState(10);
  const [wdVoucherPreview, setWdVoucherPreview] = useState(null);
  const [detailOpen, setDetailOpen] = useState(false);
  const [detailInviteeId, setDetailInviteeId] = useState(null);
  const [detailInviteeLabel, setDetailInviteeLabel] = useState('');
  const [topupHistoryOpen, setTopupHistoryOpen] = useState(false);
  const [topupHistoryInviteeId, setTopupHistoryInviteeId] = useState(null);
  const [topupHistoryInviteeLabel, setTopupHistoryInviteeLabel] = useState('');
  const [discountModalOpen, setDiscountModalOpen] = useState(false);
  const [discountModalInviteeId, setDiscountModalInviteeId] = useState(null);
  const [discountModalInviteeLabel, setDiscountModalInviteeLabel] =
    useState('');
  const [openTransfer, setOpenTransfer] = useState(false);
  const [transferAmount, setTransferAmount] = useState(() => getQuotaPerUnit());
  const [bindModalOpen, setBindModalOpen] = useState(false);
  const [bindKeyword, setBindKeyword] = useState('');
  const [bindSearchLoading, setBindSearchLoading] = useState(false);
  const [bindSubmitting, setBindSubmitting] = useState(false);
  const [bindUser, setBindUser] = useState(null);
  const [identityApp, setIdentityApp] = useState(null);
  const [identityModalOpen, setIdentityModalOpen] = useState(false);
  const [identitySubmitting, setIdentitySubmitting] = useState(false);
  const [identityTargetType, setIdentityTargetType] = useState(
    WD_ACCOUNT_ENTERPRISE,
  );
  const [identitySupplementMode, setIdentitySupplementMode] = useState(false);
  const [supplementBlink, setSupplementBlink] = useState(false);
  const [identityForm, setIdentityForm] = useState({
    real_name: '',
    id_card_no: '',
    contact: '',
  });
  const [identityUrls, setIdentityUrls] = useState([]);

  const withdrawImg = (
    statusState?.status?.distributor_withdraw_cs_image_url || ''
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
  const isProfitShareMode = useMemo(
    () =>
      String(statusState?.status?.distributor_commission_mode || 'topup') ===
      'profit_share',
    [statusState?.status?.distributor_commission_mode],
  );

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

  useEffect(() => {
    if (!withdrawOpen) return;
    let cancelled = false;
    (async () => {
      setWithdrawNoticeLoading(true);
      setWithdrawNoticeContent('');
      try {
        const res = await API.get('/api/status');
        const { success, data, message } = res.data || {};
        if (!success) {
          if (!cancelled) showError(message || t('加载失败'));
          return;
        }
        const txt = String(data?.distributor_withdraw_notice ?? '').trim();
        if (!cancelled) setWithdrawNoticeContent(txt);
      } catch {
        if (!cancelled) showError(t('加载失败'));
      } finally {
        if (!cancelled) setWithdrawNoticeLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [withdrawOpen, t]);

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

  const loadIdentityApplication = async () => {
    try {
      const res = await API.get('/api/distributor/identity_application');
      const { success, message, data } = res.data;
      if (!success) {
        showError(message);
        return;
      }
      setIdentityApp(data || null);
    } catch {
      showError(t('加载失败'));
    }
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

  const searchBindableUser = useCallback(async () => {
    const keyword = bindKeyword.trim();
    if (!keyword) {
      showError(t('请输入用户名或手机号'));
      return;
    }
    setBindSearchLoading(true);
    setBindUser(null);
    try {
      const res = await API.get('/api/distributor/bindable_user', {
        params: { keyword },
        disableDuplicate: true,
      });
      const { success, message, data } = res.data || {};
      if (!success) {
        showError(message || t('搜索失败'));
        return;
      }
      if (!data) {
        showError(t('未找到匹配用户'));
        return;
      }
      setBindUser(data);
    } catch {
      showError(t('搜索失败'));
    } finally {
      setBindSearchLoading(false);
    }
  }, [bindKeyword, t]);

  const submitBindRequest = useCallback(async () => {
    if (!bindUser?.user_id || bindSubmitting) {
      return;
    }
    setBindSubmitting(true);
    try {
      const res = await API.post('/api/distributor/bind_requests', {
        user_id: bindUser.user_id,
      });
      const { success, message, data } = res.data || {};
      if (!success) {
        showError(message || t('发送绑定请求失败'));
        return;
      }
      showSuccess(message || t('绑定请求已发送，请等待对方确认'));
      setBindUser((prev) =>
        prev
          ? {
              ...prev,
              status: 'pending',
              request_id: data?.request_id || prev.request_id,
            }
          : prev,
      );
    } catch {
      showError(t('发送绑定请求失败'));
    } finally {
      setBindSubmitting(false);
    }
  }, [bindSubmitting, bindUser, t]);

  const bindButtonText = useMemo(() => {
    if (!bindUser) return t('绑定');
    if (bindUser.status === 'bound') return t('已绑定');
    if (bindUser.status === 'distributor') return t('不允许多级代理');
    if (bindUser.status === 'pending') return t('已发送绑定请求');
    if (bindUser.status === 'not_allowed') return t('不可绑定');
    return t('绑定');
  }, [bindUser, t]);

  const bindButtonDisabled = !bindUser || bindUser.status !== 'bindable';

  useEffect(() => {
    if (!isUserDistributor) {
      navigate('/console/topup');
      return;
    }
    (async () => {
      setLoading(true);
      try {
        await loadCenter();
        await loadIdentityApplication();
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

  const setWdField = (key, val) => {
    setWdForm((prev) => ({ ...prev, [key]: val }));
  };

  const distributorApplyType = useMemo(() => {
    return Number(center?.apply_type) === WD_ACCOUNT_ENTERPRISE
      ? WD_ACCOUNT_ENTERPRISE
      : WD_ACCOUNT_PERSONAL;
  }, [center?.apply_type]);

  const identityNeedsSupplement = Boolean(center?.identity_needs_supplement);

  const resetWdForm = () => {
    const applicationName = String(center?.application_real_name || '').trim();
    const applicationIdCardNo = String(
      center?.application_id_card_no || '',
    ).trim();
    const applicationContact = String(
      center?.application_contact || '',
    ).trim();
    const applicationUrls = parseVoucherUrls(
      center?.application_qualification_urls,
    );
    const firstApplicationUrl = applicationUrls[0] || '';
    const applicationPrefill =
      distributorApplyType === WD_ACCOUNT_ENTERPRISE
        ? {
            real_name: applicationName,
            credit_code: applicationIdCardNo,
            legal_person_phone: applicationContact,
            business_license_url: firstApplicationUrl,
          }
        : {
            real_name: applicationName,
            id_card_no: applicationIdCardNo,
            mobile: applicationContact,
          };
    setWdAccountType(distributorApplyType);
    setWdForm({
      ...EMPTY_WD_FORM,
      ...applicationPrefill,
    });
  };

  const identityApplicationPending =
    Number(identityApp?.status) === IDENTITY_APP_STATUS_PENDING;
  const identityApplicationPendingSupplement =
    identityApplicationPending && Boolean(identityApp?.is_supplement);
  const identityApplicationRejected =
    Number(identityApp?.status) === IDENTITY_APP_STATUS_REJECTED;

  const openIdentityApplicationModal = (supplement = false) => {
    const target = supplement
      ? distributorApplyType
      : oppositeDistributorIdentity(distributorApplyType);
    setIdentitySupplementMode(Boolean(supplement));
    setIdentityTargetType(target);
    setIdentityForm({
      real_name: '',
      id_card_no: '',
      contact: '',
    });
    setIdentityUrls([]);
    setIdentityModalOpen(true);
  };

  const blinkSupplementButton = () => {
    setSupplementBlink(true);
    window.setTimeout(() => setSupplementBlink(false), 2000);
  };

  const setIdentityField = (key, val) => {
    setIdentityForm((prev) => ({ ...prev, [key]: val }));
  };

  const submitIdentityApplication = async () => {
    const realName = String(identityForm.real_name || '').trim();
    const idCardNo = String(identityForm.id_card_no || '').trim();
    const contact = String(identityForm.contact || '').trim();
    const urls = identityUrls.filter(Boolean);
    if (!realName || !idCardNo || !contact) {
      showError(t('请填写完整资料'));
      return;
    }
    if (!isValidPhoneNumber(contact)) {
      showError(t('请输入有效的手机号'));
      return;
    }
    if (identityTargetType === WD_ACCOUNT_ENTERPRISE && !urls.length) {
      showError(t('请上传营业执照'));
      return;
    }
    setIdentitySubmitting(true);
    try {
      const res = await API.post('/api/distributor/identity_application', {
        apply_type: identityTargetType,
        real_name: realName,
        id_card_no: idCardNo,
        contact,
        qualification_urls:
          identityTargetType === WD_ACCOUNT_ENTERPRISE ? urls : [],
      });
      const { success, message } = res.data || {};
      if (!success) {
        showError(message);
        return;
      }
      showSuccess(t('申请已提交，请等待审核'));
      setIdentityModalOpen(false);
      await loadIdentityApplication();
    } catch {
      showError(t('提交失败'));
    } finally {
      setIdentitySubmitting(false);
    }
  };

  const submitWithdraw = async () => {
    const formErr = validateWithdrawForm(t, distributorApplyType, wdForm);
    if (formErr) {
      showError(formErr);
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
      const committedFiat = wdFiatCommitRef.current?.commit?.();
      const fiatVal =
        committedFiat != null && !Number.isNaN(Number(committedFiat))
          ? committedFiat
          : wdFiatAmount;
      if (
        fiatVal == null ||
        Number.isNaN(Number(fiatVal)) ||
        Number(fiatVal) <= 0
      ) {
        showError(t('请填写与上方待使用收益一致的提现金额'));
        return;
      }
      const fiatRounded = Math.round(Number(fiatVal) * 100) / 100;
      q = displayInputAmountToQuota(fiatRounded);
      if (q <= 0) {
        showError(t('提现金额过小或无效'));
        return;
      }
    }

    if (maxQ >= minQ) {
      if (q < minQ) {
        showError(t('提现额度不能低于') + ' ' + renderQuota(minQ));
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
        t('提现额度不能大于当前可提现额度') + '（' + renderQuota(maxQ) + '）',
      );
      return;
    }
    setWdSubmitting(true);
    try {
      const body = buildWithdrawSubmitBody(distributorApplyType, wdForm, q);
      const res = await API.post('/api/distributor/withdrawal', body);
      if (res.data.success) {
        showSuccess(t('已提交，请等待审核'));
        setWithdrawOpen(false);
        resetWdForm();
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
      showError(t('划转金额最低为') + ' ' + renderQuota(getQuotaPerUnit()));
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
    center?.aff_code && `${window.location.origin}/r/${center.aff_code}`;

  const downloadRegistrationQrCode = useCallback(async () => {
    const svg = document.getElementById('dist-qr-wrap')?.querySelector('svg');
    if (!svg) return;

    try {
      await downloadQrSvgAsPng(
        svg,
        `invite-${center?.aff_code || 'qrcode'}.png`,
      );
    } catch {
      showError(t('下载失败'));
    }
  }, [center?.aff_code, t]);

  const openDetail = (r) => {
    setDetailInviteeId(r.invitee_id);
    setDetailInviteeLabel(
      String(r.display_name || r.username || `#${r.invitee_id}`).trim(),
    );
    setDetailOpen(true);
  };

  const openTopupHistoryModal = (r) => {
    setTopupHistoryInviteeId(r.invitee_id);
    setTopupHistoryInviteeLabel(
      String(r.display_name || r.username || `#${r.invitee_id}`).trim(),
    );
    setTopupHistoryOpen(true);
  };

  const openDiscountModal = (r) => {
    setDiscountModalInviteeId(r.invitee_id);
    setDiscountModalInviteeLabel(
      String(r.display_name || r.username || `#${r.invitee_id}`).trim(),
    );
    setDiscountModalOpen(true);
  };

  const columns = useMemo(
    () => [
      {
        title: t('用户'),
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
        title: isProfitShareMode ? t('累计分润') : t('累计分成额度'),
        dataIndex: isProfitShareMode
          ? 'profit_share_earned_quota'
          : 'commission_earned_quota',
        render: (q) => renderQuota(q || 0),
      },
      {
        title: t('操作'),
        width: 280,
        render: (_, r) => (
          <div className='flex flex-wrap gap-2'>
            <Button
              size='small'
              type='primary'
              theme='light'
              icon={<BarChart2 size={14} aria-hidden={true} />}
              onClick={() => openDetail(r)}
            >
              {t('详情')}
            </Button>
            <Button
              size='small'
              theme='light'
              icon={<ReceiptText size={14} aria-hidden={true} />}
              style={{
                color: 'var(--semi-color-success)',
                backgroundColor: 'var(--semi-color-success-light-default)',
              }}
              onClick={() => openTopupHistoryModal(r)}
            >
              {t('充值记录')}
            </Button>
            {isProfitShareMode ? (
              <Button
                size='small'
                type='warning'
                theme='light'
                icon={<SlidersHorizontal size={14} aria-hidden={true} />}
                onClick={() => openDiscountModal(r)}
              >
                {t('模型折扣率')}
              </Button>
            ) : null}
          </div>
        ),
      },
    ],
    [isProfitShareMode, t],
  );

  if (!isUserDistributor) {
    return null;
  }

  return (
    <div className='distributor-center-root relative mt-14 mx-auto max-w-[1600px] overflow-hidden px-4 pb-16'>
      <Title heading={3} className='mb-8'>
        {t('代理分销')}
      </Title>

      {!isProfitShareMode && (
        <Banner
          type='info'
          className='mt-2 !rounded-xl'
          description={t(
            '邀请的用户充值后，您将获得对应比例的分销额度（待使用收益）。分成比例以您账号当前设置为准；可在「详情」中查看每笔充值的入账额度、当时比例与收益。',
          )}
        />
      )}

      <div className='relative z-[1] mt-2 flex flex-col xl:flex-row gap-8 xl:gap-4 items-start'>
        {/* 窄屏：右侧栏在上，表格在下；宽屏：左表格、右栏 */}
        <aside className='flex w-full flex-col gap-[10px] xl:w-[400px] 2xl:w-[420px] flex-shrink-0 order-1 xl:order-2'>
          <Card
            className='distributor-center-stats-card !rounded-xl w-full shadow-sm border-0'
            cover={
              <div
                className='distributor-center-stats-cover relative min-h-[8.75rem]'
              >
                <div className='distributor-center-sea-scene' aria-hidden='true'>
                  <span className='distributor-center-celestial' />
                  <span className='distributor-center-cloud distributor-center-cloud--a' />
                  <span className='distributor-center-cloud distributor-center-cloud--b' />
                  <span className='distributor-center-boat'>
                    <span className='distributor-center-boat-sail' />
                    <span className='distributor-center-boat-hull' />
                  </span>
                  <span className='distributor-center-wave distributor-center-wave--a' />
                  <span className='distributor-center-wave distributor-center-wave--b' />
                  <span className='distributor-center-wave distributor-center-wave--c' />
                </div>
                <div className='relative z-10 h-full flex flex-col justify-between p-4'>
                  <div className='flex flex-wrap items-center gap-x-3 gap-y-1'>
                    <div className='flex min-w-0 items-center gap-2'>
                      <Text strong style={{ color: 'white', fontSize: '16px' }}>
                        {t('收益统计')}
                      </Text>
                      <Text
                        className='rounded-full px-2 py-0.5'
                        style={{
                          color: 'rgba(255,255,255,0.88)',
                          backgroundColor: 'rgba(255,255,255,0.14)',
                          fontSize: '12px',
                          lineHeight: 1.6,
                        }}
                      >
                        {t('当前分销比例')}：
                        {formatCommissionRatioPercent(
                          center?.effective_commission_bps ?? 0,
                        )}
                      </Text>
                    </div>
                  </div>

                  <div className='grid grid-cols-3 gap-3 sm:gap-4 mt-4'>
                    <div className='text-center'>
                      <div
                        className='text-base sm:text-xl font-bold mb-1.5'
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
                        className='text-base sm:text-xl font-bold mb-1.5'
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
                        className='text-base sm:text-xl font-bold mb-1.5'
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
            <div className='grid grid-cols-1 gap-2 sm:grid-cols-3'>
              <Button
                type='primary'
                theme='solid'
                size='small'
                onClick={() => {
                  if (identityNeedsSupplement) {
                    showError(t('请先补充代理身份信息后再提现'));
                    blinkSupplementButton();
                    return;
                  }
                  resetWdForm();
                  setWdQuotaInput('');
                  setWdFiatAmount(undefined);
                  setWdVoucherPreview(null);
                  setWithdrawOpen(true);
                }}
                className='!rounded-lg !border-0 !bg-emerald-600 hover:!bg-emerald-700 active:!bg-emerald-800 !text-white'
              >
                <Wallet size={12} className='mr-1' />
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
                <History size={12} className='mr-1' />
                {t('提现记录')}
              </Button>
              <Button
                type='primary'
                theme='solid'
                size='small'
                disabled={!center?.aff_quota || center.aff_quota <= 0}
                onClick={() => openTransferModal()}
                className='!rounded-lg'
              >
                <Zap size={12} className='mr-1' />
                {t('划转到余额')}
              </Button>
            </div>
          </Card>

          <Card
            className='!rounded-xl w-full shadow-sm border-0'
            bodyStyle={{ padding: 14 }}
          >
            <Space vertical style={{ width: '100%' }} className='!gap-2'>
              <Input
                value={shortLink || ''}
                readonly
                className='!rounded-lg'
                prefix={t('邀请链接')}
                suffix={
                  <Button
                    type='primary'
                    theme='solid'
                    onClick={async () => {
                      if (shortLink && (await copy(shortLink))) {
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
            </Space>
          </Card>

          <Card
            className='!rounded-2xl shadow-sm border-0'
            bodyStyle={{ padding: 18 }}
          >
            <div className='space-y-4'>
              <div className='flex items-start justify-between gap-3'>
                <div className='flex min-w-0 items-center gap-3'>
                  <div
                    className={[
                      'flex h-11 w-11 shrink-0 items-center justify-center rounded-xl',
                      'border border-solid border-[var(--semi-color-border)] bg-[var(--semi-color-fill-0)]',
                    ].join(' ')}
                  >
                    {distributorApplyType === WD_ACCOUNT_ENTERPRISE ? (
                      <Building2
                        size={21}
                        className='text-[var(--semi-color-warning)]'
                      />
                    ) : (
                      <UserRound
                        size={21}
                        className='text-[var(--semi-color-primary)]'
                      />
                    )}
                  </div>
                  <div className='min-w-0'>
                    <Text
                      type='tertiary'
                      size='small'
                      className='block !leading-tight'
                    >
                      {t('当前代理身份')}
                    </Text>
                    <div className='mt-1 flex min-w-0 items-center gap-2'>
                      <Text strong className='!text-base !leading-tight'>
                        {distributorIdentityLabel(t, distributorApplyType)}
                      </Text>
                      <Tag
                        color={
                          distributorApplyType === WD_ACCOUNT_ENTERPRISE
                            ? 'orange'
                            : 'blue'
                        }
                        type='light'
                        size='small'
                      >
                        {t('已生效')}
                      </Tag>
                    </div>
                  </div>
                </div>
                {identityApplicationPending ? (
                  <Tag color='light-blue' type='light' size='small'>
                    {t('审核中')}
                  </Tag>
                ) : identityNeedsSupplement ? (
                  <Tag color='orange' type='light' size='small'>
                    {t('待补充')}
                  </Tag>
                ) : identityApplicationRejected ? (
                  <Tag color='red' type='light' size='small'>
                    {t('已驳回')}
                  </Tag>
                ) : (
                  <Tag color='green' type='light' size='small'>
                    {t('正常')}
                  </Tag>
                )}
              </div>

              <div className='grid grid-cols-1 gap-2 sm:grid-cols-2'>
                <div className='rounded-lg border border-solid border-[var(--semi-color-border)] bg-[var(--semi-color-fill-0)] px-3 py-2.5'>
                  <Text type='tertiary' size='small' className='block'>
                    {t('当前主体')}
                  </Text>
                  <Text strong className='mt-1 block break-all !leading-snug'>
                    {String(center?.application_real_name || '').trim() || '—'}
                  </Text>
                </div>
                <div className='rounded-lg border border-solid border-[var(--semi-color-border)] bg-[var(--semi-color-fill-0)] px-3 py-2.5'>
                  <Text type='tertiary' size='small' className='block'>
                    {identityApplicationPending
                      ? identityApplicationPendingSupplement
                        ? t('补充身份')
                        : t('申请身份')
                      : t('可申请身份')}
                  </Text>
                  <Text strong className='mt-1 block !leading-snug'>
                    {distributorIdentityLabel(
                      t,
                      identityApplicationPending
                        ? identityApp?.target_apply_type
                        : oppositeDistributorIdentity(distributorApplyType),
                    )}
                  </Text>
                </div>
              </div>

              {identityApplicationPending ? (
                <div className='flex gap-2 rounded-lg border border-solid border-[var(--semi-color-info-light-hover)] bg-[var(--semi-color-info-light-default)] p-3'>
                  <Clock3
                    size={17}
                    className='mt-0.5 shrink-0 text-[var(--semi-color-primary)]'
                  />
                  <Text size='small' className='!leading-relaxed'>
                    {identityApplicationPendingSupplement
                      ? t('信息补充审核中，审核通过后即可按补全后的资料提现')
                      : t('审核中，提现仍按当前已生效身份处理，审核通过后生效')}
                  </Text>
                </div>
              ) : identityNeedsSupplement ? (
                <div className='flex gap-2 rounded-lg border border-solid border-[var(--semi-color-warning-light-hover)] bg-[var(--semi-color-warning-light-default)] p-3'>
                  <XCircle
                    size={17}
                    className='mt-0.5 shrink-0 text-[var(--semi-color-warning)]'
                  />
                  <Text size='small' className='!leading-relaxed'>
                    {t('当前代理身份信息待补充，请先补充并等待审核通过后再提现。')}
                  </Text>
                </div>
              ) : identityApplicationRejected ? (
                <div className='flex gap-2 rounded-lg border border-solid border-[var(--semi-color-danger-light-hover)] bg-[var(--semi-color-danger-light-default)] p-3'>
                  <XCircle
                    size={17}
                    className='mt-0.5 shrink-0 text-[var(--semi-color-danger)]'
                  />
                  <div className='min-w-0'>
                    <Text size='small' className='block !leading-relaxed'>
                      {t('身份变更申请已驳回')}：
                      {identityApp?.reject_reason || t('未填写原因')}
                    </Text>
                    <Text type='tertiary' size='small' className='mt-1 block'>
                      {t('可修改资料后重新提交')}
                    </Text>
                  </div>
                </div>
              ) : (
                <div className='flex gap-2 rounded-lg border border-solid border-[var(--semi-color-success-light-hover)] bg-[var(--semi-color-success-light-default)] p-3'>
                  <BadgeCheck
                    size={17}
                    className='mt-0.5 shrink-0 text-[var(--semi-color-success)]'
                  />
                  <Text size='small' className='!leading-relaxed'>
                    {t('身份资料已生效，可按需提交身份变更申请')}
                  </Text>
                </div>
              )}

              <Button
                type='primary'
                theme={identityApplicationPending ? 'light' : 'solid'}
                className={[
                  'w-full !rounded-lg',
                  identityNeedsSupplement ? 'distributor-supplement-btn' : '',
                  supplementBlink ? 'distributor-supplement-btn--blink' : '',
                ]
                  .filter(Boolean)
                  .join(' ')}
                disabled={identityApplicationPending}
                icon={
                  identityApplicationPending ? (
                    <Clock3 size={14} />
                  ) : oppositeDistributorIdentity(distributorApplyType) ===
                    WD_ACCOUNT_ENTERPRISE ? (
                    <Building2 size={14} />
                  ) : (
                    <UserRound size={14} />
                  )
                }
                onClick={() =>
                  openIdentityApplicationModal(identityNeedsSupplement)
                }
              >
                {identityApplicationPending
                  ? identityApplicationPendingSupplement
                    ? t('信息补充审核中')
                    : t('身份变更审核中')
                  : identityNeedsSupplement
                    ? t('信息补充')
                  : t('申请') +
                    distributorIdentityLabel(
                      t,
                      oppositeDistributorIdentity(distributorApplyType),
                    ) +
                    t('身份')}
              </Button>
            </div>
          </Card>

          <Card
            className='!rounded-2xl'
            title={t('注册二维码')}
            headerExtraContent={
              <Button
                size='small'
                type='primary'
                theme='light'
                icon={<Download size={14} />}
                disabled={!shortLink}
                onClick={downloadRegistrationQrCode}
              >
                {t('下载二维码')}
              </Button>
            }
            bodyStyle={{ paddingTop: 20, paddingBottom: 24 }}
          >
            <div className='w-full flex flex-col items-center'>
              {shortLink ? (
                <div className='inline-flex rounded-xl bg-[var(--semi-color-fill-0)] p-2 ring-1 ring-[var(--semi-color-border)]'>
                  <div
                    className='rounded-lg bg-semi-color-white p-2 shadow-[0_2px_8px_rgba(0,0,0,0.08)] dark:shadow-[0_4px_16px_rgba(0,0,0,0.55)] ring-1 ring-black/[0.06] dark:ring-white/[0.12]'
                    id='dist-qr-wrap'
                  >
                    <QRCodeSVG
                      value={shortLink}
                      size={QR_CODE_SIZE}
                      level='M'
                      bgColor='#ffffff'
                      fgColor='#000000'
                    />
                  </div>
                </div>
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
            headerExtraContent={
              <Button
                size='small'
                theme='light'
                type='primary'
                icon={<UserPlus size={14} />}
                onClick={() => {
                  setBindModalOpen(true);
                  setBindKeyword('');
                  setBindUser(null);
                }}
              >
                {t('用户绑定')}
              </Button>
            }
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
        commissionMode={isProfitShareMode ? 'profit_share' : 'topup'}
      />

      <InviteeTopupHistoryModal
        visible={topupHistoryOpen}
        onCancel={() => {
          setTopupHistoryOpen(false);
          setTopupHistoryInviteeId(null);
          setTopupHistoryInviteeLabel('');
        }}
        inviteeId={topupHistoryInviteeId}
        inviteeLabel={topupHistoryInviteeLabel}
        t={t}
      />

      <InviteeModelDiscountModal
        visible={discountModalOpen}
        onCancel={() => {
          setDiscountModalOpen(false);
          setDiscountModalInviteeId(null);
          setDiscountModalInviteeLabel('');
        }}
        inviteeId={discountModalInviteeId}
        inviteeLabel={discountModalInviteeLabel}
        t={t}
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
        title={t('用户绑定')}
        visible={bindModalOpen}
        onCancel={() => setBindModalOpen(false)}
        footer={
          <Space>
            <Button onClick={() => setBindModalOpen(false)}>{t('关闭')}</Button>
            <Button
              type='primary'
              theme='solid'
              disabled={bindButtonDisabled}
              loading={bindSubmitting}
              onClick={submitBindRequest}
            >
              {bindButtonText}
            </Button>
          </Space>
        }
      >
        <Banner
          type='info'
          className='mb-3 !rounded-lg'
          closeIcon={null}
          description={t(
            '请输入普通用户的用户名或手机号搜索并发送绑定请求；对方会在站内消息中选择接受或拒绝，处理后您会收到结果通知。对方接受后，该用户将出现在您的邀请用户列表中。',
          )}
        />
        <Space vertical align='start' style={{ width: '100%' }}>
          <Input
            value={bindKeyword}
            onChange={setBindKeyword}
            onEnterPress={searchBindableUser}
            placeholder={t('输入普通用户的用户名或手机号，按回车搜索')}
            suffix={
              <Button
                size='small'
                type='primary'
                theme='light'
                loading={bindSearchLoading}
                onClick={searchBindableUser}
              >
                {t('搜索')}
              </Button>
            }
          />
          {bindUser ? (
            <div className='w-full rounded-lg border border-semi-color-border bg-semi-color-fill-0 p-3'>
              <div className='flex flex-col gap-1'>
                <Text strong>{bindUser.display_name || bindUser.username}</Text>
                {/* <Text type='tertiary' size='small'>
                  {t('用户名')}：{bindUser.username || '—'}
                </Text>
                <Text type='tertiary' size='small'>
                  {t('用户ID')}：{bindUser.user_id}
                </Text> */}
                {bindUser.phone ? (
                  <Text type='tertiary' size='small'>
                    {t('手机号')}：{bindUser.phone}
                  </Text>
                ) : null}
              </div>
              {/* <div className='mt-3'>
                <Button
                  size='small'
                  type='primary'
                  theme='solid'
                  disabled={bindButtonDisabled}
                  loading={bindSubmitting}
                  onClick={submitBindRequest}
                >
                  {bindButtonText}
                </Button>
              </div> */}
            </div>
          ) : null}
        </Space>
      </Modal>

      <Modal
        title={
          identitySupplementMode
            ? t('信息补充')
            : t('申请') +
              distributorIdentityLabel(t, identityTargetType) +
              t('身份')
        }
        visible={identityModalOpen}
        onCancel={() => setIdentityModalOpen(false)}
        footer={
          <Space>
            <Button onClick={() => setIdentityModalOpen(false)}>
              {t('关闭')}
            </Button>
            <Button
              type='primary'
              theme='solid'
              loading={identitySubmitting}
              onClick={submitIdentityApplication}
            >
              {t('提交申请')}
            </Button>
          </Space>
        }
        width={720}
      >
        <Banner
          type='info'
          className='mb-4 !rounded-lg'
          closeIcon={null}
          description={t(
            '提交后由管理员审批；审核通过前，提现仍按当前已生效身份处理。',
          )}
        />
        <div className='space-y-4'>
          <div className='flex items-center justify-between gap-3 rounded-lg border border-[var(--semi-color-border)] bg-[var(--semi-color-fill-0)] px-3 py-2'>
            <Text type='secondary'>
              {identitySupplementMode ? t('补充身份') : t('申请身份')}
            </Text>
            <Tag
              color={
                identityTargetType === WD_ACCOUNT_ENTERPRISE ? 'orange' : 'blue'
              }
              type='light'
            >
              {distributorIdentityLabel(t, identityTargetType)}
            </Tag>
          </div>
          <div>
            <Text strong className='mb-1 block'>
              {identityTargetType === WD_ACCOUNT_ENTERPRISE
                ? t('企业名称')
                : t('姓名')}
            </Text>
            <Input
              value={identityForm.real_name}
              onChange={(v) => setIdentityField('real_name', String(v ?? ''))}
              placeholder={
                identityTargetType === WD_ACCOUNT_ENTERPRISE
                  ? t('企业名称')
                  : t('真实姓名')
              }
            />
          </div>
          <div>
            <Text strong className='mb-1 block'>
              {identityTargetType === WD_ACCOUNT_ENTERPRISE
                ? t('统一社会信用代码')
                : t('身份证号码')}
            </Text>
            <Input
              value={identityForm.id_card_no}
              onChange={(v) => setIdentityField('id_card_no', String(v ?? ''))}
              placeholder={
                identityTargetType === WD_ACCOUNT_ENTERPRISE
                  ? t('统一社会信用代码')
                  : t('身份证号码')
              }
            />
          </div>
          <div>
            <Text strong className='mb-1 block'>
              {identityTargetType === WD_ACCOUNT_ENTERPRISE
                ? t('法人手机号码')
                : t('手机号码')}
            </Text>
            <Input
              value={identityForm.contact}
              onChange={(v) => setIdentityField('contact', String(v ?? ''))}
              placeholder={
                identityTargetType === WD_ACCOUNT_ENTERPRISE
                  ? t('法人手机号码')
                  : t('手机号码')
              }
            />
          </div>
          {identityTargetType === WD_ACCOUNT_ENTERPRISE ? (
            <DistributorApplyFileUpload
              label={t('营业执照')}
              labelExtra={t('需加盖公章')}
              required
              urls={identityUrls}
              onUrlsChange={setIdentityUrls}
              maxCount={5}
              multiple
              onPreview={setWdVoucherPreview}
              hint={t('支持图片或 PDF，最多 5 个；点击图片可大图预览')}
            />
          ) : null}
        </div>
      </Modal>

      <Modal
        title={t('提现')}
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
        <div className='mb-4'>
          <Text type='tertiary' size='small' className='block'>
            {t(
              '提交后将暂扣对应待使用收益，审核通过即完成提现；驳回或取消将退回额度。',
            )}
          </Text>
        </div>
        <div className='flex flex-col lg:flex-row gap-6 items-start'>
          <div className='flex-1 min-w-0 w-full space-y-5'>
            <DistributorWithdrawFormFields
              accountType={wdAccountType}
              form={wdForm}
              onFieldChange={setWdField}
              onPreview={setWdVoucherPreview}
              fiatCommitRef={wdFiatCommitRef}
              isQuotaTokensMode={isQuotaTokensMode}
              quotaInput={wdQuotaInput}
              onQuotaInputChange={setWdQuotaInput}
              fiatAmount={wdFiatAmount}
              onFiatAmountChange={setWdFiatAmount}
              fiatMin={wdFiatMin}
              fiatMax={wdFiatMax}
              affQuotaFloor={affQuotaFloor}
              minQInternal={minQInternal}
              affQuota={center?.aff_quota}
              renderQuota={renderQuota}
            />
            {(withdrawNoticeLoading || withdrawNoticeContent) && (
              <div className='mt-4'>
                <Text size='small' strong className='block mb-2'>
                  {t('提现说明')}
                </Text>
                <Spin spinning={withdrawNoticeLoading}>
                  {withdrawNoticeContent ? (
                    <Text
                      type='secondary'
                      size='small'
                      className='block whitespace-pre-wrap break-words'
                    >
                      {withdrawNoticeContent}
                    </Text>
                  ) : null}
                </Spin>
              </div>
            )}
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
          <Button onClick={() => setWithdrawLogOpen(false)}>{t('关闭')}</Button>
        }
        width={960}
      >
        <Table
          loading={wdLogLoading}
          size='small'
          rowKey='id'
          scroll={{ x: 1100 }}
          columns={[
            {
              title: t('额度'),
              dataIndex: 'quota_amount',
              width: 120,
              render: (q) => renderQuota(q || 0),
            },
            {
              title: t('状态'),
              dataIndex: 'status',
              width: 112,
              render: (s) => wdStatusTag(s),
            },
            { title: t('提现月份'), dataIndex: 'withdraw_month', width: 100 },
            {
              title: t('类型'),
              dataIndex: 'account_type',
              width: 72,
              render: (v) => wdAccountTypeLabel(t, v),
            },
            {
              title: t('收款信息'),
              render: (_, r) => {
                const f = parseWithdrawRow(r);
                return (
                  <div className='text-xs space-y-0.5'>
                    <div>{f.real_name}</div>
                    <div className='text-[var(--semi-color-text-2)]'>
                      {f.bank_name} {f.bank_account}
                    </div>
                  </div>
                );
              },
            },
            {
              title: t('时间'),
              dataIndex: 'created_at',
              width: 158,
              render: (ts) =>
                ts ? dayjs.unix(Number(ts)).format('YYYY-MM-DD HH:mm') : '—',
            },
            {
              title: t('反馈原因'),
              width: 200,
              render: (_, r) => {
                const reason = String(r.reject_reason || '').trim();
                if (reason) {
                  return (
                    <Text
                      size='small'
                      className='break-all text-[var(--semi-color-text-0)]'
                    >
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
              title: t('操作'),
              width: 100,
              fixed: 'right',
              align: 'center',
              render: (_, r) =>
                r.status === 1 ? (
                  <Popconfirm
                    title={t('确认取消该笔提现？额度将退回')}
                    onConfirm={() => cancelWithdraw(r.id)}
                  >
                    <Tag
                      color='red'
                      type='light'
                      size='small'
                      className='cursor-pointer'
                    >
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
        {wdVoucherPreview && isPdfUrl(wdVoucherPreview) ? (
          <div className='py-6 text-center'>
            <Button
              type='primary'
              onClick={() =>
                window.open(wdVoucherPreview, '_blank', 'noopener,noreferrer')
              }
            >
              {t('在新窗口打开 PDF')}
            </Button>
          </div>
        ) : wdVoucherPreview ? (
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
