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

import React, { useEffect, useState, useContext, useRef } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import {
  API,
  showError,
  showInfo,
  showSuccess,
  renderQuota,
  renderQuotaWithAmount,
  copy,
  getQuotaPerUnit,
} from '../../helpers';
import { Modal, Toast, Button } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { UserContext } from '../../context/User';
import { StatusContext } from '../../context/Status';
import { QRCodeSVG } from 'qrcode.react';
import { SiWechat } from 'react-icons/si';

import RechargeCard from './RechargeCard';
import TransferModal from './modals/TransferModal';
import PaymentConfirmModal from './modals/PaymentConfirmModal';
import PaymentMethodSelectModal from './modals/PaymentMethodSelectModal';
import TopupHistoryModal from './modals/TopupHistoryModal';
import UcoinPayResultModal from './modals/UcoinPayResultModal';

const DEFAULT_EPAY_MAX_TOPUP = 100000;

const isLimitedEpayMethod = (type) => {
  const lower = (type || '').toLowerCase();
  return (
    lower === 'alipay' ||
    lower === 'wxpay' ||
    lower === 'paypal' ||
    String(type || '')
      .toUpperCase()
      .startsWith('PP_')
  );
};

const normalizePayMethodMaxTopup = (method) => {
  const normalized = Number(method.max_topup);
  if (Number.isFinite(normalized) && normalized > 0) {
    method.max_topup = normalized;
    return method;
  }
  if (isLimitedEpayMethod(method.type)) {
    method.max_topup = DEFAULT_EPAY_MAX_TOPUP;
  } else {
    method.max_topup = 0;
  }
  return method;
};

const getPayMethodMaxTopup = (payMethods, type) => {
  const method = payMethods.find((m) => m.type === type);
  const configured = Number(method?.max_topup);
  if (Number.isFinite(configured) && configured > 0) return configured;
  if (isLimitedEpayMethod(type)) return DEFAULT_EPAY_MAX_TOPUP;
  return 0;
};

const TopUp = () => {
  const navigate = useNavigate();
  const { t } = useTranslation();
  const [searchParams, setSearchParams] = useSearchParams();
  const [userState, userDispatch] = useContext(UserContext);
  const [statusState] = useContext(StatusContext);

  const [redemptionCode, setRedemptionCode] = useState('');
  const [amount, setAmount] = useState(0.0);
  const [minTopUp, setMinTopUp] = useState(statusState?.status?.min_topup || 1);
  const [topUpCount, setTopUpCount] = useState(
    statusState?.status?.min_topup || 1,
  );
  const [topUpLink, setTopUpLink] = useState(
    statusState?.status?.top_up_link || '',
  );
  const [enableOnlineTopUp, setEnableOnlineTopUp] = useState(
    statusState?.status?.enable_online_topup || false,
  );
  const [priceRatio, setPriceRatio] = useState(statusState?.status?.price || 1);

  const [enableStripeTopUp, setEnableStripeTopUp] = useState(
    statusState?.status?.enable_stripe_topup || false,
  );
  const [statusLoading, setStatusLoading] = useState(true);
  const [rechargeDisplayCurrency, setRechargeDisplayCurrency] = useState('USD');

  // Creem 相关状态
  const [creemProducts, setCreemProducts] = useState([]);
  const [enableCreemTopUp, setEnableCreemTopUp] = useState(false);
  const [creemOpen, setCreemOpen] = useState(false);
  const [selectedCreemProduct, setSelectedCreemProduct] = useState(null);

  // Waffo 相关状态
  const [enableWaffoTopUp, setEnableWaffoTopUp] = useState(false);
  const [waffoPayMethods, setWaffoPayMethods] = useState([]);
  const [waffoMinTopUp, setWaffoMinTopUp] = useState(1);

  // U币（虚拟币）相关状态
  const [enableUcoinTopUp, setEnableUcoinTopUp] = useState(false);
  const [ucoinCoinPairs, setUcoinCoinPairs] = useState([]);
  const [ucoinMinTopUp, setUcoinMinTopUp] = useState(1);
  const [ucoinResultOpen, setUcoinResultOpen] = useState(false);
  const [ucoinResult, setUcoinResult] = useState(null);

  const [isSubmitting, setIsSubmitting] = useState(false);
  const [open, setOpen] = useState(false);
  const [paymentSelectOpen, setPaymentSelectOpen] = useState(false);
  const [payWay, setPayWay] = useState('');
  const [amountLoading, setAmountLoading] = useState(false);
  const [paymentLoading, setPaymentLoading] = useState(false);
  /** 当前正在发起支付的按钮标识，避免多种支付方式共用 loading */
  const [activePaymentKey, setActivePaymentKey] = useState('');
  const [confirmLoading, setConfirmLoading] = useState(false);
  const [payCurrency, setPayCurrency] = useState('USD');
  const [inputCurrency, setInputCurrency] = useState('USD');
  const [quoteQuotaToAdd, setQuoteQuotaToAdd] = useState(0);
  const [, setQuoteChargedUsd] = useState(0);
  const [payMethods, setPayMethods] = useState([]);
  const [wechatQrOpen, setWechatQrOpen] = useState(false);
  const [wechatQrValue, setWechatQrValue] = useState('');
  const [wechatQrBaseQuota, setWechatQrBaseQuota] = useState(null);

  const [openTransfer, setOpenTransfer] = useState(false);
  const [transferAmount, setTransferAmount] = useState(0);

  // 账单Modal状态
  const [openHistory, setOpenHistory] = useState(false);

  // 订阅相关
  const [subscriptionPlans, setSubscriptionPlans] = useState([]);
  const [subscriptionLoading, setSubscriptionLoading] = useState(true);
  const [billingPreference, setBillingPreference] =
    useState('subscription_first');
  const [activeSubscriptions, setActiveSubscriptions] = useState([]);
  const [allSubscriptions, setAllSubscriptions] = useState([]);

  // 预设充值额度选项
  const [presetAmounts, setPresetAmounts] = useState([]);
  const [selectedPreset, setSelectedPreset] = useState(null);

  // 充值配置信息
  const [topupInfo, setTopupInfo] = useState({
    amount_options: [],
    discount: {},
  });
  const [realNameVerificationRequired, setRealNameVerificationRequired] =
    useState(false);
  const [realNameVerificationChecked, setRealNameVerificationChecked] =
    useState(false);
  const realNameVerificationModalShownRef = useRef(false);

  const showRealNameVerificationModal = () => {
    Modal.confirm({
      title: t('\u8bf7\u5148\u5b8c\u6210\u5b9e\u540d\u8ba4\u8bc1'),
      content: t(
        '\u7cfb\u7edf\u5df2\u5f00\u542f\u5145\u503c\u524d\u5f3a\u5236\u5b9e\u540d\u8ba4\u8bc1\uff0c\u5b8c\u6210\u540e\u5373\u53ef\u7ee7\u7eed\u5145\u503c\u3002',
      ),
      okText: t('\u53bb\u5b9e\u540d\u8ba4\u8bc1'),
      cancelText: t('\u7a0d\u540e\u518d\u8bf4'),
      centered: true,
      onOk: () => navigate('/console/real-name-verification'),
    });
  };

  const syncRealNameVerificationGate = (
    data,
    { prompt = false, promptOnce = false } = {},
  ) => {
    const required = Boolean(
      data?.real_name_verification_required && !data?.real_name_verified,
    );
    setRealNameVerificationRequired(required);
    setRealNameVerificationChecked(true);
    if (required && prompt) {
      if (!promptOnce || !realNameVerificationModalShownRef.current) {
        if (promptOnce) realNameVerificationModalShownRef.current = true;
        showRealNameVerificationModal();
      }
    }
    return !required;
  };

  const ensureRealNameVerifiedForTopUp = async () => {
    if (realNameVerificationRequired) {
      showRealNameVerificationModal();
      return false;
    }
    if (realNameVerificationChecked) return true;
    try {
      const res = await API.get('/api/user/topup/info', {
        disableDuplicate: true,
      });
      const { message, data, success } = res.data;
      if (!success) {
        showError(message || t('获取实名认证状态失败，请刷新后重试'));
        return false;
      }
      return syncRealNameVerificationGate(data, { prompt: true });
    } catch (err) {
      showError(t('获取实名认证状态失败，请刷新后重试'));
      return false;
    }
  };

  const topUp = async () => {
    if (!(await ensureRealNameVerifiedForTopUp())) return;
    if (redemptionCode === '') {
      showInfo(t('请输入兑换码！'));
      return;
    }
    setIsSubmitting(true);
    try {
      const res = await API.post('/api/user/topup', {
        key: redemptionCode,
      });
      const { success, message, data } = res.data;
      if (success) {
        showSuccess(t('兑换成功！'));
        Modal.success({
          title: t('兑换成功！'),
          content: t('成功兑换额度：') + renderQuota(data),
          centered: true,
        });
        if (userState.user) {
          const updatedUser = {
            ...userState.user,
            quota: userState.user.quota + data,
          };
          userDispatch({ type: 'login', payload: updatedUser });
        }
        setRedemptionCode('');
      } else {
        showError(message);
      }
    } catch (err) {
      showError(t('请求失败'));
    } finally {
      setIsSubmitting(false);
    }
  };

  const openTopUpLink = async () => {
    if (!(await ensureRealNameVerifiedForTopUp())) return;
    if (!topUpLink) {
      showError(t('超级管理员未设置充值链接！'));
      return;
    }
    window.open(topUpLink, '_blank');
  };

  const getEffectiveQuotaDisplayType = () => {
    return (
      statusState?.status?.quota_display_type ||
      localStorage.getItem('quota_display_type') ||
      'USD'
    );
  };

  const getEffectiveRechargeDisplayCurrency = () => {
    const currency =
      statusState?.status?.recharge_display_currency ||
      localStorage.getItem('recharge_display_currency') ||
      rechargeDisplayCurrency ||
      'USD';
    return String(currency).toUpperCase() === 'CNY' ? 'CNY' : 'USD';
  };

  const resolveRechargeInputCurrency = () => {
    const displayType = getEffectiveQuotaDisplayType();
    if (displayType === 'CNY') return 'CNY';
    if (displayType === 'TOKENS') return getEffectiveRechargeDisplayCurrency();
    return 'USD';
  };

  const preTopUp = async (payment, count = topUpCount) => {
    if (!(await ensureRealNameVerifiedForTopUp())) return;
    if (payment === 'stripe') {
      if (!enableStripeTopUp) {
        showError(t('管理员未开启Stripe充值！'));
        return;
      }
    } else {
      if (!enableOnlineTopUp) {
        showError(t('管理员未开启在线充值！'));
        return;
      }
    }

    setPayWay(payment);
    setActivePaymentKey(payment);
    setPaymentLoading(true);
    try {
      if (payment === 'stripe') {
        await getStripeAmount(count);
      } else {
        await getAmount(count);
      }

      if (count < minTopUp) {
        showError(t('充值数量不能小于') + minTopUp);
        return;
      }
      if (payment !== 'stripe') {
        const maxTopUp = getPayMethodMaxTopup(payMethods, payment);
        if (maxTopUp > 0 && Number(count) > maxTopUp) {
          showError(t('充值数量不能大于') + maxTopUp);
          return;
        }
      }
      setOpen(true);
    } catch (error) {
      showError(t('获取金额失败'));
    } finally {
      setPaymentLoading(false);
      setActivePaymentKey('');
    }
  };

  const onlineTopUp = async () => {
    if (payWay === 'stripe') {
      // Stripe 支付处理
      if (amount === 0) {
        await getStripeAmount();
      }
    } else {
      // 普通支付处理
      if (amount === 0) {
        await getAmount();
      }
    }

    if (topUpCount < minTopUp) {
      showError('充值数量不能小于' + minTopUp);
      return;
    }
    if (payWay !== 'stripe') {
      const maxTopUp = getPayMethodMaxTopup(payMethods, payWay);
      if (maxTopUp > 0 && Number(topUpCount) > maxTopUp) {
        showError(t('充值数量不能大于') + maxTopUp);
        return;
      }
    }
    setConfirmLoading(true);
    try {
      let res;
      if (payWay === 'stripe') {
        // Stripe 支付请求
        res = await API.post('/api/user/stripe/pay', {
          amount: parseFloat(topUpCount),
          payment_method: 'stripe',
        });
      } else {
        // 普通支付请求
        res = await API.post('/api/user/pay', {
          amount: parseFloat(topUpCount),
          payment_method: payWay,
        });
      }

      if (res !== undefined) {
        const { message, data } = res.data;
        if (message === 'success') {
          if (payWay === 'stripe') {
            // Stripe 支付回调处理
            window.open(data.pay_link, '_blank');
          } else {
            // 普通支付表单提交
            let params = data;
            let url = res.data.url;
            /**
             * 桌面端遇到 weixin:// deep link 时，不直接唤起本地微信，
             * 改为展示二维码给用户扫码支付。
             */
            const isWeixinDeepLink =
              typeof url === 'string' &&
              url.toLowerCase().startsWith('weixin://');
            const isMobileDevice = /android|iphone|ipad|ipod|mobile/i.test(
              navigator.userAgent || '',
            );
            if (isWeixinDeepLink && !isMobileDevice) {
              setWechatQrBaseQuota(Number(userState?.user?.quota ?? 0));
              setWechatQrValue(url);
              setWechatQrOpen(true);
              return;
            }
            let form = document.createElement('form');
            form.action = url;
            form.method = 'POST';
            let isSafari =
              navigator.userAgent.indexOf('Safari') > -1 &&
              navigator.userAgent.indexOf('Chrome') < 1;
            if (!isSafari) {
              form.target = '_blank';
            }
            for (let key in params) {
              let input = document.createElement('input');
              input.type = 'hidden';
              input.name = key;
              input.value = params[key];
              form.appendChild(input);
            }
            document.body.appendChild(form);
            form.submit();
            document.body.removeChild(form);
          }
        } else {
          const errorMsg =
            typeof data === 'string' ? data : message || t('支付失败');
          showError(errorMsg);
        }
      } else {
        showError(res);
      }
    } catch (err) {
      showError(t('支付请求失败'));
    } finally {
      setOpen(false);
      setConfirmLoading(false);
    }
  };

  const creemPreTopUp = async (product) => {
    if (!(await ensureRealNameVerifiedForTopUp())) return;
    if (!enableCreemTopUp) {
      showError(t('管理员未开启 Creem 充值！'));
      return;
    }
    setSelectedCreemProduct(product);
    setCreemOpen(true);
  };

  const onlineCreemTopUp = async () => {
    if (!selectedCreemProduct) {
      showError(t('请选择产品'));
      return;
    }
    // Validate product has required fields
    if (!selectedCreemProduct.productId) {
      showError(t('产品配置错误，请联系管理员'));
      return;
    }
    setConfirmLoading(true);
    try {
      const res = await API.post('/api/user/creem/pay', {
        product_id: selectedCreemProduct.productId,
        payment_method: 'creem',
      });
      if (res !== undefined) {
        const { message, data } = res.data;
        if (message === 'success') {
          processCreemCallback(data);
        } else {
          const errorMsg =
            typeof data === 'string' ? data : message || t('支付失败');
          showError(errorMsg);
        }
      } else {
        showError(res);
      }
    } catch (err) {
      showError(t('支付请求失败'));
    } finally {
      setCreemOpen(false);
      setConfirmLoading(false);
    }
  };

  const waffoTopUp = async (payMethodIndex) => {
    if (!(await ensureRealNameVerifiedForTopUp())) return;
    try {
      if (topUpCount < waffoMinTopUp) {
        showError(t('充值数量不能小于') + waffoMinTopUp);
        return;
      }
      const waffoKey =
        payMethodIndex != null ? `waffo:${payMethodIndex}` : 'waffo';
      setActivePaymentKey(waffoKey);
      setPaymentLoading(true);
      const requestBody = {
        amount: parseFloat(topUpCount),
      };
      if (payMethodIndex != null) {
        requestBody.pay_method_index = payMethodIndex;
      }
      const res = await API.post('/api/user/waffo/pay', requestBody);
      if (res !== undefined) {
        const { message, data } = res.data;
        if (message === 'success' && data?.payment_url) {
          window.open(data.payment_url, '_blank');
        } else {
          showError(data || t('支付请求失败'));
        }
      } else {
        showError(res);
      }
    } catch (e) {
      showError(t('支付请求失败'));
    } finally {
      setPaymentLoading(false);
      setActivePaymentKey('');
    }
  };

  const ucoinTopUp = async (coinPairIndex) => {
    if (!(await ensureRealNameVerifiedForTopUp())) return;
    try {
      if (!enableUcoinTopUp) {
        showError(t('管理员未开启 U币支付！'));
        return;
      }
      if (topUpCount < ucoinMinTopUp) {
        showError(t('充值数量不能小于') + ucoinMinTopUp);
        return;
      }
      setActivePaymentKey(`ucoin:${coinPairIndex}`);
      setPaymentLoading(true);
      const res = await API.post('/api/user/ubcoin/pay', {
        amount: parseInt(topUpCount),
        coin_pair_index: coinPairIndex,
      });
      if (res !== undefined) {
        const { message, data } = res.data;
        if (message === 'success') {
          setUcoinResult(data);
          setUcoinResultOpen(true);
        } else {
          const errorMsg =
            typeof data === 'string' ? data : message || t('支付请求失败');
          showError(errorMsg);
        }
      } else {
        showError(res);
      }
    } catch (e) {
      showError(t('支付请求失败'));
    } finally {
      setPaymentLoading(false);
      setActivePaymentKey('');
    }
  };

  const processCreemCallback = (data) => {
    // 与 Stripe 保持一致的实现方式
    window.open(data.checkout_url, '_blank');
  };

  const getUserQuota = async () => {
    let res = await API.get(`/api/user/self`);
    const { success, message, data } = res.data;
    if (success) {
      userDispatch({ type: 'login', payload: data });
      return data;
    } else {
      showError(message);
      return null;
    }
  };

  /**
   * 静默拉取用户信息（用于二维码支付轮询，不弹错误提示，避免干扰用户）。
   * @returns {Promise<null|Record<string, unknown>>}
   */
  const getUserQuotaSilently = async () => {
    try {
      const res = await API.get(`/api/user/self`, { skipErrorHandler: true });
      const { success, data } = res.data || {};
      if (success && data) {
        userDispatch({ type: 'login', payload: data });
        return data;
      }
      return null;
    } catch {
      return null;
    }
  };

  const getSubscriptionPlans = async () => {
    setSubscriptionLoading(true);
    try {
      const res = await API.get('/api/subscription/plans');
      if (res.data?.success) {
        setSubscriptionPlans(res.data.data || []);
      }
    } catch (e) {
      setSubscriptionPlans([]);
    } finally {
      setSubscriptionLoading(false);
    }
  };

  const getSubscriptionSelf = async () => {
    try {
      const res = await API.get('/api/subscription/self');
      if (res.data?.success) {
        setBillingPreference(
          res.data.data?.billing_preference || 'subscription_first',
        );
        // Active subscriptions
        const activeSubs = res.data.data?.subscriptions || [];
        setActiveSubscriptions(activeSubs);
        // All subscriptions (including expired)
        const allSubs = res.data.data?.all_subscriptions || [];
        setAllSubscriptions(allSubs);
      }
    } catch (e) {
      // ignore
    }
  };

  const updateBillingPreference = async (pref) => {
    const previousPref = billingPreference;
    setBillingPreference(pref);
    try {
      const res = await API.put('/api/subscription/self/preference', {
        billing_preference: pref,
      });
      if (res.data?.success) {
        showSuccess(t('更新成功'));
        const normalizedPref =
          res.data?.data?.billing_preference || pref || previousPref;
        setBillingPreference(normalizedPref);
      } else {
        showError(res.data?.message || t('更新失败'));
        setBillingPreference(previousPref);
      }
    } catch (e) {
      showError(t('请求失败'));
      setBillingPreference(previousPref);
    }
  };

  // 获取充值配置信息
  const getTopupInfo = async () => {
    try {
      const res = await API.get('/api/user/topup/info');
      const { message, data, success } = res.data;
      if (success) {
        syncRealNameVerificationGate(data, { prompt: true, promptOnce: true });
        setTopupInfo({
          amount_options: data.amount_options || [],
          discount: data.discount || {},
        });

        // 处理支付方式
        let payMethods = data.pay_methods || [];
        try {
          if (typeof payMethods === 'string') {
            payMethods = JSON.parse(payMethods);
          }
          if (payMethods && payMethods.length > 0) {
            // 检查name和type是否为空
            payMethods = payMethods.filter((method) => {
              return method.name && method.type;
            });
            // 如果没有color，则设置默认颜色
            payMethods = payMethods.map((method) => {
              // 规范化最小充值数
              const normalizedMinTopup = Number(method.min_topup);
              method.min_topup = Number.isFinite(normalizedMinTopup)
                ? normalizedMinTopup
                : 0;
              normalizePayMethodMaxTopup(method);

              // Stripe 的最小充值从后端字段回填
              if (
                method.type === 'stripe' &&
                (!method.min_topup || method.min_topup <= 0)
              ) {
                const stripeMin = Number(data.stripe_min_topup);
                if (Number.isFinite(stripeMin)) {
                  method.min_topup = stripeMin;
                }
              }

              if (!method.color) {
                if (method.type === 'alipay') {
                  method.color = 'rgba(var(--semi-blue-5), 1)';
                } else if (method.type === 'wxpay') {
                  method.color = 'rgba(var(--semi-green-5), 1)';
                } else if (method.type === 'stripe') {
                  method.color = 'rgba(var(--semi-purple-5), 1)';
                } else {
                  method.color = 'rgba(var(--semi-primary-5), 1)';
                }
              }
              return method;
            });
          } else {
            payMethods = [];
          }

          // 如果启用了 Stripe 支付，添加到支付方法列表
          // 这个逻辑现在由后端处理，如果 Stripe 启用，后端会在 pay_methods 中包含它

          setPayMethods(payMethods);
          const enableStripeTopUp = data.enable_stripe_topup || false;
          const enableOnlineTopUp = data.enable_online_topup || false;
          const enableCreemTopUp = data.enable_creem_topup || false;
          const minTopUpValue = enableOnlineTopUp
            ? data.min_topup
            : enableStripeTopUp
              ? data.stripe_min_topup
              : data.enable_waffo_topup
                ? data.waffo_min_topup
                : data.enable_ubcoin_topup
                  ? data.ubcoin_min_topup
                  : 1;
          setEnableOnlineTopUp(enableOnlineTopUp);
          setEnableStripeTopUp(enableStripeTopUp);
          setEnableCreemTopUp(enableCreemTopUp);
          const enableWaffoTopUp = data.enable_waffo_topup || false;
          setEnableWaffoTopUp(enableWaffoTopUp);
          setWaffoPayMethods(data.waffo_pay_methods || []);
          setWaffoMinTopUp(data.waffo_min_topup || 1);
          setEnableUcoinTopUp(data.enable_ubcoin_topup || false);
          setUcoinCoinPairs(data.ubcoin_coin_pairs || []);
          setUcoinMinTopUp(data.ubcoin_min_topup || 1);
          setMinTopUp(minTopUpValue);
          setTopUpCount(minTopUpValue);

          // 设置 Creem 产品
          try {
            const products = JSON.parse(data.creem_products || '[]');
            setCreemProducts(products);
          } catch (e) {
            setCreemProducts([]);
          }

          // 如果没有自定义充值数量选项，根据最小充值金额生成预设充值额度选项
          if (topupInfo.amount_options.length === 0) {
            setPresetAmounts(generatePresetAmounts(minTopUpValue));
          }

          // 初始化显示实付金额
          getAmount(minTopUpValue);
        } catch (e) {
          setPayMethods([]);
        }

        // 如果有自定义充值数量选项，使用它们替换默认的预设选项
        if (data.amount_options && data.amount_options.length > 0) {
          const customPresets = data.amount_options.map((amount) => ({
            value: amount,
            discount: data.discount[amount] || 1.0,
          }));
          setPresetAmounts(customPresets);
        }
      } else {
        showError(data || t('获取充值配置失败'));
      }
    } catch (error) {
      showError(t('获取充值配置异常'));
    }
  };

  // 划转邀请额度
  const transfer = async () => {
    if (transferAmount < getQuotaPerUnit()) {
      showError(t('划转金额最低为') + ' ' + renderQuota(getQuotaPerUnit()));
      return;
    }
    const res = await API.post(`/api/user/aff_transfer`, {
      quota: transferAmount,
    });
    const { success, message } = res.data;
    if (success) {
      showSuccess(message);
      setOpenTransfer(false);
      getUserQuota().then();
    } else {
      showError(message);
    }
  };

  // URL 参数自动打开账单弹窗（支付回跳时触发）
  useEffect(() => {
    if (searchParams.get('show_history') === 'true') {
      setOpenHistory(true);
      searchParams.delete('show_history');
      setSearchParams(searchParams, { replace: true });
    }
  }, []);

  useEffect(() => {
    // 始终获取最新用户数据，确保余额等统计信息准确
    getUserQuota().then();
    setTransferAmount(getQuotaPerUnit());
  }, []);

  // 在 statusState 可用时获取充值信息
  useEffect(() => {
    getTopupInfo().then();
    getSubscriptionPlans().then();
    getSubscriptionSelf().then();
  }, []);

  useEffect(() => {
    if (statusState?.status) {
      // const minTopUpValue = statusState.status.min_topup || 1;
      // setMinTopUp(minTopUpValue);
      // setTopUpCount(minTopUpValue);
      setTopUpLink(statusState.status.top_up_link || '');
      setPriceRatio(statusState.status.price || 1);
      const nextRechargeCurrency = getEffectiveRechargeDisplayCurrency();
      setRechargeDisplayCurrency(nextRechargeCurrency);
      const displayType = getEffectiveQuotaDisplayType();
      setInputCurrency(
        displayType === 'CNY'
          ? 'CNY'
          : displayType === 'TOKENS'
            ? nextRechargeCurrency
            : 'USD',
      );
      setPayCurrency(nextRechargeCurrency);

      setStatusLoading(false);
    }
  }, [statusState?.status]);

  const getCurrencyMeta = (currency) => {
    const code = String(currency || 'USD').toUpperCase();
    if (code === 'CNY') {
      return { symbol: '¥', code: 'CNY' };
    }
    return { symbol: '$', code: 'USD' };
  };

  const formatCurrencyAmount = (value, currency) => {
    const numericAmount = Number(value);
    const formattedAmount = Number.isFinite(numericAmount)
      ? numericAmount.toFixed(2)
      : '0.00';
    const { symbol, code } = getCurrencyMeta(currency);
    if (code === 'USD') {
      return `${symbol}${formattedAmount} ${code}`;
    }
    return `${symbol}${formattedAmount}`;
  };

  const applyTopupQuote = (data, fallbackCurrency = 'CNY') => {
    if (data && typeof data === 'object') {
      const nextAmount = Number(data.pay_amount ?? 0);
      setAmount(Number.isFinite(nextAmount) ? nextAmount : 0);
      setPayCurrency(data.pay_currency || fallbackCurrency);
      setInputCurrency(data.input_currency || resolveRechargeInputCurrency());
      setQuoteQuotaToAdd(Number(data.quota_to_add || 0));
      setQuoteChargedUsd(Number(data.charged_usd || 0));
      return;
    }
    const nextAmount = Number(data);
    setAmount(Number.isFinite(nextAmount) ? nextAmount : 0);
    setPayCurrency(fallbackCurrency);
    setInputCurrency(resolveRechargeInputCurrency());
    setQuoteQuotaToAdd(0);
    setQuoteChargedUsd(0);
  };

  const renderAmount = () => {
    return formatCurrencyAmount(amount, payCurrency);
  };

  const renderTopUpCountDisplay = (value) => {
    return formatCurrencyAmount(
      value,
      inputCurrency || resolveRechargeInputCurrency(),
    );
  };

  const renderQuoteCreditAmount = () => {
    const displayType = getEffectiveQuotaDisplayType();
    if (displayType === 'TOKENS') {
      return quoteQuotaToAdd > 0 ? renderQuota(quoteQuotaToAdd) : '';
    }
    return '';
  };

  const getAmount = async (value) => {
    if (value === undefined) {
      value = topUpCount;
    }
    setAmountLoading(true);
    try {
      const res = await API.post('/api/user/amount', {
        amount: parseFloat(value),
      });
      if (res !== undefined) {
        const { message, data } = res.data;
        if (message === 'success') {
          applyTopupQuote(data, 'CNY');
        } else {
          setAmount(0);
          setQuoteQuotaToAdd(0);
          setQuoteChargedUsd(0);
          Toast.error({ content: '错误：' + data, id: 'getAmount' });
        }
      } else {
        showError(res);
      }
    } catch (err) {
      // amount fetch failed silently
    }
    setAmountLoading(false);
  };

  const getStripeAmount = async (value) => {
    if (value === undefined) {
      value = topUpCount;
    }
    setAmountLoading(true);
    try {
      const res = await API.post('/api/user/stripe/amount', {
        amount: parseFloat(value),
      });
      if (res !== undefined) {
        const { message, data } = res.data;
        if (message === 'success') {
          applyTopupQuote(data, 'USD');
        } else {
          setAmount(0);
          setQuoteQuotaToAdd(0);
          setQuoteChargedUsd(0);
          Toast.error({ content: '错误：' + data, id: 'getAmount' });
        }
      } else {
        showError(res);
      }
    } catch (err) {
      // amount fetch failed silently
    } finally {
      setAmountLoading(false);
    }
  };

  const handleCancel = () => {
    setOpen(false);
  };

  const handleTransferCancel = () => {
    setOpenTransfer(false);
  };

  const handleOpenHistory = () => {
    setOpenHistory(true);
  };

  const handleHistoryCancel = () => {
    setOpenHistory(false);
  };

  const handleCreemCancel = () => {
    setCreemOpen(false);
    setSelectedCreemProduct(null);
  };

  /**
   * 关闭微信扫码弹窗并清理二维码内容。
   */
  const handleWechatQrCancel = () => {
    setWechatQrOpen(false);
    setWechatQrValue('');
    setWechatQrBaseQuota(null);
  };

  /**
   * 微信扫码弹窗打开时轮询用户额度；检测到额度增长即认为支付到账并自动关闭弹窗。
   */
  useEffect(() => {
    if (!wechatQrOpen) {
      return;
    }
    let active = true;
    const startTs = Date.now();
    const timer = setInterval(async () => {
      if (!active) {
        return;
      }
      const user = await getUserQuotaSilently();
      if (!user) {
        return;
      }
      const currentQuota = Number(user.quota ?? 0);
      const baseQuota = Number(wechatQrBaseQuota ?? 0);
      if (currentQuota > baseQuota) {
        showSuccess(t('检测到支付已完成，二维码已自动关闭'));
        handleWechatQrCancel();
        // PC 扫码支付不会触发当前浏览器回跳，这里在到账后主动跳转到回调页面。
        window.location.href = '/console/log?show_history=true';
        return;
      }
      // 超过 10 分钟自动停止轮询，避免页面长时间空转。
      if (Date.now() - startTs > 10 * 60 * 1000) {
        handleWechatQrCancel();
      }
    }, 3000);
    return () => {
      active = false;
      clearInterval(timer);
    };
  }, [wechatQrOpen, wechatQrBaseQuota]);

  // 选择预设充值额度
  const selectPresetAmount = async (preset) => {
    setTopUpCount(preset.value);
    setSelectedPreset(preset.value);
    await getAmount(preset.value);
  };

  /** 获取当前额度下可用的在线支付方式（不含 waffo）。 */
  const getEnabledEpayMethods = (count = topUpCount) => {
    return payMethods.filter((method) => {
      if (method.type === 'waffo') {
        return false;
      }
      const minTopupVal = Number(method.min_topup) || 0;
      const maxTopupVal = Number(method.max_topup) || 0;
      const isStripe = method.type === 'stripe';
      const disabled =
        (!enableOnlineTopUp && !isStripe) ||
        (!enableStripeTopUp && isStripe) ||
        minTopupVal > Number(count || 0) ||
        (maxTopupVal > 0 && maxTopupVal < Number(count || 0));
      return !disabled;
    });
  };

  /** 点击额度卡片：选中额度后弹出支付方式，或仅一种方式时直接进入确认弹窗。 */
  const handlePresetCardClick = async (preset) => {
    if (!(await ensureRealNameVerifiedForTopUp())) return;
    await selectPresetAmount(preset);
    const enabledMethods = getEnabledEpayMethods(preset.value);
    if (enabledMethods.length === 0) {
      showError(t('当前额度暂无可用支付方式'));
      return;
    }
    if (enabledMethods.length === 1) {
      await preTopUp(enabledMethods[0].type, preset.value);
      return;
    }
    setPaymentSelectOpen(true);
  };

  const handlePaymentMethodSelect = async (payment) => {
    setPaymentSelectOpen(false);
    await preTopUp(payment);
  };

  const handlePaymentSelectCancel = () => {
    setPaymentSelectOpen(false);
  };

  // 格式化大数字显示
  const formatLargeNumber = (num) => {
    return num.toString();
  };

  // 根据最小充值金额生成预设充值额度选项
  const generatePresetAmounts = (minAmount) => {
    const multipliers = [1, 5, 10, 30, 50, 100, 300, 500];
    return multipliers.map((multiplier) => ({
      value: minAmount * multiplier,
    }));
  };

  return (
    <div className='w-full max-w-7xl mx-auto relative min-h-screen lg:min-h-0 mt-[60px] px-2'>
      {/* 划转模态框 */}
      <TransferModal
        t={t}
        openTransfer={openTransfer}
        transfer={transfer}
        handleTransferCancel={handleTransferCancel}
        userState={userState}
        renderQuota={renderQuota}
        getQuotaPerUnit={getQuotaPerUnit}
        transferAmount={transferAmount}
        setTransferAmount={setTransferAmount}
      />

      {/* 支付方式选择模态框（点击额度卡片后） */}
      <PaymentMethodSelectModal
        t={t}
        visible={paymentSelectOpen}
        onCancel={handlePaymentSelectCancel}
        topUpCount={topUpCount}
        renderTopUpCount={renderTopUpCountDisplay}
        payMethods={payMethods}
        enableOnlineTopUp={enableOnlineTopUp}
        enableStripeTopUp={enableStripeTopUp}
        onSelect={handlePaymentMethodSelect}
        activePaymentKey={activePaymentKey}
      />

      {/* 充值确认模态框 */}
      <PaymentConfirmModal
        t={t}
        open={open}
        onlineTopUp={onlineTopUp}
        handleCancel={handleCancel}
        confirmLoading={confirmLoading}
        topUpCount={topUpCount}
        renderTopUpCount={renderTopUpCountDisplay}
        amountLoading={amountLoading}
        renderAmount={renderAmount}
        payWay={payWay}
        payMethods={payMethods}
        amountNumber={amount}
        discountRate={1.0}
        creditDisplay={renderQuoteCreditAmount()}
        rechargeDisplayCurrency={payCurrency}
      />

      {/* 充值账单模态框 */}
      <TopupHistoryModal
        visible={openHistory}
        onCancel={handleHistoryCancel}
        t={t}
      />

      {/* 微信二维码支付弹窗（PC 遇到 weixin:// deep link 时展示） */}
      <Modal
        title={
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <SiWechat size={18} color='#07C160' />
            <span>{t('微信扫码支付')}</span>
          </div>
        }
        visible={wechatQrOpen}
        onCancel={handleWechatQrCancel}
        footer={null}
        maskClosable
        centered
        width={420}
      >
        <div
          style={{
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            gap: 12,
            paddingTop: 8,
            paddingBottom: 8,
          }}
        >
          <QRCodeSVG value={wechatQrValue || 'about:blank'} size={240} />
          <div style={{ color: 'var(--semi-color-text-1)' }}>
            {t('请使用微信扫一扫完成支付')}
          </div>
          <div
            style={{
              color: 'var(--semi-color-text-2)',
              fontSize: 12,
              wordBreak: 'break-all',
              textAlign: 'center',
            }}
          >
            {t('若扫码失败，可复制以下链接到其他设备打开：')}
            <br />
            {wechatQrValue}
          </div>
        </div>
      </Modal>

      {/* Creem 充值确认模态框 */}
      <Modal
        title={t('确定要充值 $')}
        visible={creemOpen}
        onOk={onlineCreemTopUp}
        onCancel={handleCreemCancel}
        maskClosable={false}
        size='small'
        centered
        confirmLoading={confirmLoading}
      >
        {selectedCreemProduct && (
          <>
            <p>
              {t('产品名称')}：{selectedCreemProduct.name}
            </p>
            <p>
              {t('价格')}：{selectedCreemProduct.currency === 'EUR' ? '€' : '$'}
              {selectedCreemProduct.price}
            </p>
            <p>
              {t('充值额度')}：{selectedCreemProduct.quota}
            </p>
            <p>{t('是否确认充值？')}</p>
          </>
        )}
      </Modal>

      {/* 主布局区域（邀请奖励已迁至代理分销） */}
      <div className='grid grid-cols-1 gap-6'>
        <RechargeCard
          t={t}
          enableOnlineTopUp={enableOnlineTopUp}
          enableStripeTopUp={enableStripeTopUp}
          enableCreemTopUp={enableCreemTopUp}
          creemProducts={creemProducts}
          creemPreTopUp={creemPreTopUp}
          enableWaffoTopUp={enableWaffoTopUp}
          waffoTopUp={waffoTopUp}
          waffoPayMethods={waffoPayMethods}
          enableUcoinTopUp={enableUcoinTopUp}
          ucoinTopUp={ucoinTopUp}
          ucoinCoinPairs={ucoinCoinPairs}
          presetAmounts={presetAmounts}
          selectedPreset={selectedPreset}
          selectPresetAmount={selectPresetAmount}
          onPresetCardClick={handlePresetCardClick}
          formatLargeNumber={formatLargeNumber}
          priceRatio={priceRatio}
          topUpCount={topUpCount}
          minTopUp={minTopUp}
          renderTopUpCount={renderTopUpCountDisplay}
          getAmount={getAmount}
          setTopUpCount={setTopUpCount}
          setSelectedPreset={setSelectedPreset}
          renderAmount={renderAmount}
          amountLoading={amountLoading}
          payMethods={payMethods}
          preTopUp={preTopUp}
          activePaymentKey={activePaymentKey}
          redemptionCode={redemptionCode}
          setRedemptionCode={setRedemptionCode}
          topUp={topUp}
          isSubmitting={isSubmitting}
          topUpLink={topUpLink}
          openTopUpLink={openTopUpLink}
          userState={userState}
          renderQuota={renderQuota}
          statusLoading={statusLoading}
          topupInfo={topupInfo}
          onOpenHistory={handleOpenHistory}
          subscriptionLoading={subscriptionLoading}
          subscriptionPlans={subscriptionPlans}
          billingPreference={billingPreference}
          onChangeBillingPreference={updateBillingPreference}
          activeSubscriptions={activeSubscriptions}
          allSubscriptions={allSubscriptions}
          reloadSubscriptionSelf={getSubscriptionSelf}
          ensureRealNameVerifiedForTopUp={ensureRealNameVerifiedForTopUp}
        />
      </div>

      {/* U币支付结果弹窗：展示生成的收款地址 */}
      <UcoinPayResultModal
        visible={ucoinResultOpen}
        onCancel={() => setUcoinResultOpen(false)}
        ucoinResult={ucoinResult}
      />
    </div>
  );
};

export default TopUp;
