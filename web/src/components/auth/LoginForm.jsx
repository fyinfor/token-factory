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

import React, { useContext, useEffect, useMemo, useRef, useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { UserContext } from '../../context/User';
import { StatusContext } from '../../context/Status';
import {
  API,
  getLogo,
  showError,
  showInfo,
  showSuccess,
  updateAPI,
  getSystemName,
  getOAuthProviderIcon,
  setUserData,
  onGitHubOAuthClicked,
  onDiscordOAuthClicked,
  onOIDCClicked,
  onLinuxDOOAuthClicked,
  onCustomOAuthClicked,
  prepareCredentialRequestOptions,
  buildAssertionResult,
  isPasskeySupported,
  safeInternalRedirectPath,
  redirectApplyIntentToAdminIfNeeded,
} from '../../helpers';
import Turnstile from 'react-turnstile';
import {
  Button,
  Card,
  Checkbox,
  Divider,
  Form,
  Icon,
  Modal,
  Tabs,
  InputNumber,
  Select,
  Input,
} from '@douyinfe/semi-ui';
import Title from '@douyinfe/semi-ui/lib/es/typography/title';
import Text from '@douyinfe/semi-ui/lib/es/typography/text';
import TelegramLoginButton from 'react-telegram-login';

import {
  IconGithubLogo,
  IconMail,
  IconLock,
  IconKey,
  IconPhone,
} from '@douyinfe/semi-icons';
import OIDCIcon from '../common/logo/OIDCIcon';
import WeChatIcon from '../common/logo/WeChatIcon';
import LinuxDoIcon from '../common/logo/LinuxDoIcon';
import TwoFAVerification from './TwoFAVerification';
import { useTranslation } from 'react-i18next';
import { SiDiscord } from 'react-icons/si';
import {
  COUNTRY_DIAL_CODES,
  findCountryByCode,
  toE164,
} from '../../constants/countryDialCodes';

const LoginForm = () => {
  let navigate = useNavigate();
  const { t } = useTranslation();
  const githubButtonTextKeyByState = {
    idle: '使用 GitHub 继续',
    redirecting: '正在跳转 GitHub...',
    timeout: '请求超时，请刷新页面后重新发起 GitHub 登录',
  };
  const [inputs, setInputs] = useState({
    username: '',
    password: '',
    wechat_verification_code: '',
  });
  const [smsInputs, setSmsInputs] = useState({
    countryCode: 'CN',
    localNumber: '',
    code: '',
  });
  const [smsCountdown, setSmsCountdown] = useState(0);
  const [smsSending, setSmsSending] = useState(false);
  const [smsLoginLoading, setSmsLoginLoading] = useState(false);
  const [activeLoginTab, setActiveLoginTab] = useState('password');

  // 拼装当前真实发送/校验的手机号：国内走 11 位，国际走 +{dial}{local}。
  const buildE164Phone = (state) => {
    const country = findCountryByCode(state.countryCode);
    if (!country) return '';
    if (country.code === 'CN') {
      return String(state.localNumber || '').replace(/[^\d]/g, '');
    }
    return toE164(country, state.localNumber);
  };
  const { username, password } = inputs;
  const [searchParams, setSearchParams] = useSearchParams();
  const postLoginRedirect = useMemo(
    () => safeInternalRedirectPath(searchParams.get('redirect')),
    [searchParams],
  );
  const navigateAfterLogin = (fallbackPath, loggedInUser) => {
    const base = postLoginRedirect || fallbackPath;
    const target = loggedInUser
      ? redirectApplyIntentToAdminIfNeeded(base, loggedInUser)
      : base;
    navigate(target);
  };
  const [submitted, setSubmitted] = useState(false);
  const [userState, userDispatch] = useContext(UserContext);
  const [statusState] = useContext(StatusContext);
  const [turnstileEnabled, setTurnstileEnabled] = useState(false);
  const [turnstileSiteKey, setTurnstileSiteKey] = useState('');
  const [turnstileToken, setTurnstileToken] = useState('');
  const [showWeChatLoginModal, setShowWeChatLoginModal] = useState(false);
  const [showEmailLogin, setShowEmailLogin] = useState(false);
  const [wechatLoading, setWechatLoading] = useState(false);
  const [githubLoading, setGithubLoading] = useState(false);
  const [discordLoading, setDiscordLoading] = useState(false);
  const [oidcLoading, setOidcLoading] = useState(false);
  const [linuxdoLoading, setLinuxdoLoading] = useState(false);
  const [emailLoginLoading, setEmailLoginLoading] = useState(false);
  const [loginLoading, setLoginLoading] = useState(false);
  const [resetPasswordLoading, setResetPasswordLoading] = useState(false);
  const [otherLoginOptionsLoading, setOtherLoginOptionsLoading] =
    useState(false);
  const [wechatCodeSubmitLoading, setWechatCodeSubmitLoading] = useState(false);
  const [showTwoFA, setShowTwoFA] = useState(false);
  const [passkeySupported, setPasskeySupported] = useState(false);
  const [passkeyLoading, setPasskeyLoading] = useState(false);
  const [agreedToTerms, setAgreedToTerms] = useState(false);
  const [hasUserAgreement, setHasUserAgreement] = useState(false);
  const [hasPrivacyPolicy, setHasPrivacyPolicy] = useState(false);
  const [githubButtonState, setGithubButtonState] = useState('idle');
  const [githubButtonDisabled, setGithubButtonDisabled] = useState(false);
  const githubTimeoutRef = useRef(null);
  const githubButtonText = t(githubButtonTextKeyByState[githubButtonState]);
  const [customOAuthLoading, setCustomOAuthLoading] = useState({});

  const logo = getLogo();
  const systemName = getSystemName();

  let affCode = new URLSearchParams(window.location.search).get('aff');
  if (affCode) {
    localStorage.setItem('aff', affCode);
  }

  const status = useMemo(() => {
    if (statusState?.status) return statusState.status;
    const savedStatus = localStorage.getItem('status');
    if (!savedStatus) return {};
    try {
      return JSON.parse(savedStatus) || {};
    } catch (err) {
      return {};
    }
  }, [statusState?.status]);
  const hasCustomOAuthProviders =
    (status.custom_oauth_providers || []).length > 0;
  const hasOAuthLoginOptions = Boolean(
    status.github_oauth ||
    status.discord_oauth ||
    status.oidc_enabled ||
    status.wechat_login ||
    status.linuxdo_oauth ||
    status.telegram_oauth ||
    hasCustomOAuthProviders,
  );

  useEffect(() => {
    if (status?.turnstile_check) {
      setTurnstileEnabled(true);
      setTurnstileSiteKey(status.turnstile_site_key);
    }

    // 从 status 获取用户协议和隐私政策的启用状态
    setHasUserAgreement(status?.user_agreement_enabled || false);
    setHasPrivacyPolicy(status?.privacy_policy_enabled || false);
  }, [status]);

  useEffect(() => {
    isPasskeySupported()
      .then(setPasskeySupported)
      .catch(() => setPasskeySupported(false));

    return () => {
      if (githubTimeoutRef.current) {
        clearTimeout(githubTimeoutRef.current);
      }
    };
  }, []);

  useEffect(() => {
    if (searchParams.get('expired')) {
      showError(t('未登录或登录已过期，请重新登录'));
    }
  }, []);

  const onWeChatLoginClicked = () => {
    if ((hasUserAgreement || hasPrivacyPolicy) && !agreedToTerms) {
      showInfo(t('请先阅读并同意用户协议和隐私政策'));
      return;
    }
    setWechatLoading(true);
    setShowWeChatLoginModal(true);
    setWechatLoading(false);
  };

  const onSubmitWeChatVerificationCode = async () => {
    if (turnstileEnabled && turnstileToken === '') {
      showInfo(t('请稍后几秒重试，Turnstile 正在检查用户环境！'));
      return;
    }
    setWechatCodeSubmitLoading(true);
    try {
      const res = await API.get(
        `/api/oauth/wechat?code=${inputs.wechat_verification_code}`,
      );
      const { success, message, data } = res.data;
      if (success) {
        userDispatch({ type: 'login', payload: data });
        localStorage.setItem('user', JSON.stringify(data));
        setUserData(data);
        updateAPI();
        navigateAfterLogin('/', data);
        showSuccess(t('登录成功！'));
        setShowWeChatLoginModal(false);
      } else {
        showError(t(message));
      }
    } catch (error) {
      showError(t('登录失败，请重试'));
    } finally {
      setWechatCodeSubmitLoading(false);
    }
  };

  function handleChange(name, value) {
    setInputs((inputs) => ({ ...inputs, [name]: value }));
  }

  async function handleSubmit(e) {
    if ((hasUserAgreement || hasPrivacyPolicy) && !agreedToTerms) {
      showInfo(t('请先阅读并同意用户协议和隐私政策'));
      return;
    }
    if (turnstileEnabled && turnstileToken === '') {
      showInfo(t('请稍后几秒重试，Turnstile 正在检查用户环境！'));
      return;
    }
    setSubmitted(true);
    setLoginLoading(true);
    try {
      if (username && password) {
        const res = await API.post(
          `/api/user/login?turnstile=${turnstileToken}`,
          {
            username,
            password,
          },
        );
        const { success, message, data } = res.data;
        if (success) {
          // 检查是否需要2FA验证
          if (data && data.require_2fa) {
            setShowTwoFA(true);
            setLoginLoading(false);
            return;
          }

          userDispatch({ type: 'login', payload: data });
          setUserData(data);
          updateAPI();
          showSuccess(t('登录成功！'));
          if (username === 'root' && password === '123456') {
            Modal.error({
              title: t('您正在使用默认密码！'),
              content: t('请立刻修改默认密码！'),
              centered: true,
            });
          }
          navigateAfterLogin('/console', data);
        } else {
          showError(t(message));
        }
      } else {
        showError(t('请输入用户名和密码！'));
      }
    } catch (error) {
      showError(t('登录失败，请重试'));
    } finally {
      setLoginLoading(false);
    }
  }

  // 添加Telegram登录处理函数
  const onTelegramLoginClicked = async (response) => {
    if ((hasUserAgreement || hasPrivacyPolicy) && !agreedToTerms) {
      showInfo(t('请先阅读并同意用户协议和隐私政策'));
      return;
    }
    const fields = [
      'id',
      'first_name',
      'last_name',
      'username',
      'photo_url',
      'auth_date',
      'hash',
      'lang',
    ];
    const params = {};
    fields.forEach((field) => {
      if (response[field]) {
        params[field] = response[field];
      }
    });
    try {
      const res = await API.get(`/api/oauth/telegram/login`, { params });
      const { success, message, data } = res.data;
      if (success) {
        userDispatch({ type: 'login', payload: data });
        localStorage.setItem('user', JSON.stringify(data));
        showSuccess(t('登录成功！'));
        setUserData(data);
        updateAPI();
        navigateAfterLogin('/', data);
      } else {
        showError(t(message));
      }
    } catch (error) {
      showError(t('登录失败，请重试'));
    }
  };

  // 包装的GitHub登录点击处理
  const handleGitHubClick = () => {
    if ((hasUserAgreement || hasPrivacyPolicy) && !agreedToTerms) {
      showInfo(t('请先阅读并同意用户协议和隐私政策'));
      return;
    }
    if (githubButtonDisabled) {
      return;
    }
    setGithubLoading(true);
    setGithubButtonDisabled(true);
    setGithubButtonState('redirecting');
    if (githubTimeoutRef.current) {
      clearTimeout(githubTimeoutRef.current);
    }
    githubTimeoutRef.current = setTimeout(() => {
      setGithubLoading(false);
      setGithubButtonState('timeout');
      setGithubButtonDisabled(true);
    }, 20000);
    try {
      onGitHubOAuthClicked(status.github_client_id, { shouldLogout: true });
    } finally {
      // 由于重定向，这里不会执行到，但为了完整性添加
      setTimeout(() => setGithubLoading(false), 3000);
    }
  };

  // 包装的Discord登录点击处理
  const handleDiscordClick = () => {
    if ((hasUserAgreement || hasPrivacyPolicy) && !agreedToTerms) {
      showInfo(t('请先阅读并同意用户协议和隐私政策'));
      return;
    }
    setDiscordLoading(true);
    try {
      onDiscordOAuthClicked(status.discord_client_id, { shouldLogout: true });
    } finally {
      // 由于重定向，这里不会执行到，但为了完整性添加
      setTimeout(() => setDiscordLoading(false), 3000);
    }
  };

  // 包装的OIDC登录点击处理
  const handleOIDCClick = () => {
    if ((hasUserAgreement || hasPrivacyPolicy) && !agreedToTerms) {
      showInfo(t('请先阅读并同意用户协议和隐私政策'));
      return;
    }
    setOidcLoading(true);
    try {
      onOIDCClicked(
        status.oidc_authorization_endpoint,
        status.oidc_client_id,
        false,
        { shouldLogout: true },
      );
    } finally {
      // 由于重定向，这里不会执行到，但为了完整性添加
      setTimeout(() => setOidcLoading(false), 3000);
    }
  };

  // 包装的LinuxDO登录点击处理
  const handleLinuxDOClick = () => {
    if ((hasUserAgreement || hasPrivacyPolicy) && !agreedToTerms) {
      showInfo(t('请先阅读并同意用户协议和隐私政策'));
      return;
    }
    setLinuxdoLoading(true);
    try {
      onLinuxDOOAuthClicked(status.linuxdo_client_id, { shouldLogout: true });
    } finally {
      // 由于重定向，这里不会执行到，但为了完整性添加
      setTimeout(() => setLinuxdoLoading(false), 3000);
    }
  };

  // 包装的自定义OAuth登录点击处理
  const handleCustomOAuthClick = (provider) => {
    if ((hasUserAgreement || hasPrivacyPolicy) && !agreedToTerms) {
      showInfo(t('请先阅读并同意用户协议和隐私政策'));
      return;
    }
    setCustomOAuthLoading((prev) => ({ ...prev, [provider.slug]: true }));
    try {
      onCustomOAuthClicked(provider, { shouldLogout: true });
    } finally {
      // 由于重定向，这里不会执行到，但为了完整性添加
      setTimeout(() => {
        setCustomOAuthLoading((prev) => ({ ...prev, [provider.slug]: false }));
      }, 3000);
    }
  };

  // 包装的邮箱登录选项点击处理
  const handleEmailLoginClick = () => {
    setEmailLoginLoading(true);
    setShowEmailLogin(true);
    setEmailLoginLoading(false);
  };

  // 短信验证码登录
  const handleSMSLoginSendCode = async () => {
    if ((hasUserAgreement || hasPrivacyPolicy) && !agreedToTerms) {
      showInfo(t('请先阅读并同意用户协议和隐私政策'));
      return;
    }
    const phone = buildE164Phone(smsInputs);
    if (!phone) {
      showError(t('请输入手机号'));
      return;
    }
    setSmsSending(true);
    try {
      const res = await API.post('/api/user/login/sms/send', { phone });
      const { success, message } = res.data || {};
      if (success) {
        showSuccess(t('验证码已发送，请注意查收'));
        setSmsCountdown(60);
        const timer = setInterval(() => {
          setSmsCountdown((prev) => {
            if (prev <= 1) {
              clearInterval(timer);
              return 0;
            }
            return prev - 1;
          });
        }, 1000);
      } else {
        showError(t(message || '发送失败'));
      }
    } catch (err) {
      showError(t(err?.response?.data?.message || '发送失败，请稍后重试'));
    } finally {
      setSmsSending(false);
    }
  };

  const handleSMSLoginSubmit = async () => {
    if ((hasUserAgreement || hasPrivacyPolicy) && !agreedToTerms) {
      showInfo(t('请先阅读并同意用户协议和隐私政策'));
      return;
    }
    const phone = buildE164Phone(smsInputs);
    const code = (smsInputs.code || '').trim();
    if (!phone) {
      showError(t('请输入手机号'));
      return;
    }
    if (!code) {
      showError(t('请输入验证码'));
      return;
    }
    setSmsLoginLoading(true);
    try {
      const res = await API.post('/api/user/login/sms', { phone, code });
      const { success, message, data } = res.data || {};
      if (success) {
        if (data && data.require_2fa) {
          setShowTwoFA(true);
          setSmsLoginLoading(false);
          return;
        }
        userDispatch({ type: 'login', payload: data });
        setUserData(data);
        updateAPI();
        showSuccess(t('登录成功！'));
        navigateAfterLogin('/console', data);
      } else {
        showError(t(message || '登录失败'));
      }
    } catch (err) {
      showError(t(err?.response?.data?.message || '登录失败，请重试'));
    } finally {
      setSmsLoginLoading(false);
    }
  };

  const handlePasskeyLogin = async () => {
    if ((hasUserAgreement || hasPrivacyPolicy) && !agreedToTerms) {
      showInfo(t('请先阅读并同意用户协议和隐私政策'));
      return;
    }
    if (!passkeySupported) {
      showInfo(t('当前环境无法使用 Passkey 登录'));
      return;
    }
    if (!window.PublicKeyCredential) {
      showInfo(t('当前浏览器不支持 Passkey'));
      return;
    }

    setPasskeyLoading(true);
    try {
      const beginRes = await API.post('/api/user/passkey/login/begin');
      const { success, message, data } = beginRes.data;
      if (!success) {
        showError(t(message || '无法发起 Passkey 登录'));
        return;
      }

      const publicKeyOptions = prepareCredentialRequestOptions(
        data?.options || data?.publicKey || data,
      );
      const assertion = await navigator.credentials.get({
        publicKey: publicKeyOptions,
      });
      const payload = buildAssertionResult(assertion);
      if (!payload) {
        showError(t('Passkey 验证失败，请重试'));
        return;
      }

      const finishRes = await API.post(
        '/api/user/passkey/login/finish',
        payload,
      );
      const finish = finishRes.data;
      if (finish.success) {
        userDispatch({ type: 'login', payload: finish.data });
        setUserData(finish.data);
        updateAPI();
        showSuccess(t('登录成功！'));
        navigateAfterLogin('/console', finish.data);
      } else {
        showError(t(finish.message || 'Passkey 登录失败，请重试'));
      }
    } catch (error) {
      if (error?.name === 'AbortError') {
        showInfo(t('已取消 Passkey 登录'));
      } else {
        showError(t('Passkey 登录失败，请重试'));
      }
    } finally {
      setPasskeyLoading(false);
    }
  };

  // 包装的重置密码点击处理
  const handleResetPasswordClick = () => {
    setResetPasswordLoading(true);
    navigate('/reset');
    setResetPasswordLoading(false);
  };

  // 包装的其他登录选项点击处理
  const handleOtherLoginOptionsClick = () => {
    setOtherLoginOptionsLoading(true);
    setShowEmailLogin(false);
    setOtherLoginOptionsLoading(false);
  };

  // 2FA验证成功处理
  const handle2FASuccess = (data) => {
    userDispatch({ type: 'login', payload: data });
    setUserData(data);
    updateAPI();
    showSuccess(t('登录成功！'));
    navigateAfterLogin('/console', data);
  };

  // 返回登录页面
  const handleBackToLogin = () => {
    setShowTwoFA(false);
    setInputs({ username: '', password: '', wechat_verification_code: '' });
  };

  const renderOAuthOptions = () => {
    return (
      <div className='flex flex-col items-center'>
        <div className='w-full max-w-md'>
          <div className='flex items-center justify-center mb-6 gap-2'>
            <img src={logo} alt='Logo' className='h-10 rounded-full' />
            <Title heading={3} className='!text-gray-800'>
              {systemName}
            </Title>
          </div>

          <Card className='border-0 !rounded-2xl overflow-hidden'>
            <div className='flex justify-center pt-6 pb-2'>
              <Title heading={3} className='text-gray-800 dark:text-gray-200'>
                {t('登 录')}
              </Title>
            </div>
            <div className='px-2 py-8'>
              <div className='space-y-3'>
                {status.wechat_login && (
                  <Button
                    theme='outline'
                    className='w-full h-12 flex items-center justify-center !rounded-full border border-gray-200 hover:bg-gray-50 transition-colors'
                    type='tertiary'
                    icon={
                      <Icon svg={<WeChatIcon />} style={{ color: '#07C160' }} />
                    }
                    onClick={onWeChatLoginClicked}
                    loading={wechatLoading}
                  >
                    <span className='ml-3'>{t('使用 微信 继续')}</span>
                  </Button>
                )}

                {status.github_oauth && (
                  <Button
                    theme='outline'
                    className='w-full h-12 flex items-center justify-center !rounded-full border border-gray-200 hover:bg-gray-50 transition-colors'
                    type='tertiary'
                    icon={<IconGithubLogo size='large' />}
                    onClick={handleGitHubClick}
                    loading={githubLoading}
                    disabled={githubButtonDisabled}
                  >
                    <span className='ml-3'>{githubButtonText}</span>
                  </Button>
                )}

                {status.discord_oauth && (
                  <Button
                    theme='outline'
                    className='w-full h-12 flex items-center justify-center !rounded-full border border-gray-200 hover:bg-gray-50 transition-colors'
                    type='tertiary'
                    icon={
                      <SiDiscord
                        style={{
                          color: '#5865F2',
                          width: '20px',
                          height: '20px',
                        }}
                      />
                    }
                    onClick={handleDiscordClick}
                    loading={discordLoading}
                  >
                    <span className='ml-3'>{t('使用 Discord 继续')}</span>
                  </Button>
                )}

                {status.oidc_enabled && (
                  <Button
                    theme='outline'
                    className='w-full h-12 flex items-center justify-center !rounded-full border border-gray-200 hover:bg-gray-50 transition-colors'
                    type='tertiary'
                    icon={<OIDCIcon style={{ color: '#1877F2' }} />}
                    onClick={handleOIDCClick}
                    loading={oidcLoading}
                  >
                    <span className='ml-3'>{t('使用 OIDC 继续')}</span>
                  </Button>
                )}

                {status.linuxdo_oauth && (
                  <Button
                    theme='outline'
                    className='w-full h-12 flex items-center justify-center !rounded-full border border-gray-200 hover:bg-gray-50 transition-colors'
                    type='tertiary'
                    icon={
                      <LinuxDoIcon
                        style={{
                          color: '#E95420',
                          width: '20px',
                          height: '20px',
                        }}
                      />
                    }
                    onClick={handleLinuxDOClick}
                    loading={linuxdoLoading}
                  >
                    <span className='ml-3'>{t('使用 LinuxDO 继续')}</span>
                  </Button>
                )}

                {status.custom_oauth_providers &&
                  status.custom_oauth_providers.map((provider) => (
                    <Button
                      key={provider.slug}
                      theme='outline'
                      className='w-full h-12 flex items-center justify-center !rounded-full border border-gray-200 hover:bg-gray-50 transition-colors'
                      type='tertiary'
                      icon={getOAuthProviderIcon(provider.icon || '', 20)}
                      onClick={() => handleCustomOAuthClick(provider)}
                      loading={customOAuthLoading[provider.slug]}
                    >
                      <span className='ml-3'>
                        {t('使用 {{name}} 继续', { name: provider.name })}
                      </span>
                    </Button>
                  ))}

                {status.telegram_oauth && (
                  <div className='flex justify-center my-2'>
                    <TelegramLoginButton
                      dataOnauth={onTelegramLoginClicked}
                      botName={status.telegram_bot_name}
                    />
                  </div>
                )}

                {status.passkey_login && passkeySupported && (
                  <Button
                    theme='outline'
                    className='w-full h-12 flex items-center justify-center !rounded-full border border-gray-200 hover:bg-gray-50 transition-colors'
                    type='tertiary'
                    icon={<IconKey size='large' />}
                    onClick={handlePasskeyLogin}
                    loading={passkeyLoading}
                  >
                    <span className='ml-3'>{t('使用 Passkey 登录')}</span>
                  </Button>
                )}

                <Divider margin='12px' align='center'>
                  {t('或')}
                </Divider>

                <Button
                  theme='solid'
                  type='primary'
                  className='w-full h-12 flex items-center justify-center bg-black text-white !rounded-full hover:bg-gray-800 transition-colors'
                  icon={<IconMail size='large' />}
                  onClick={handleEmailLoginClick}
                  loading={emailLoginLoading}
                >
                  <span className='ml-3'>{t('使用 邮箱或用户名 登录')}</span>
                </Button>
              </div>

              {(hasUserAgreement || hasPrivacyPolicy) && (
                <div className='mt-6'>
                  <Checkbox
                    checked={agreedToTerms}
                    onChange={(e) => setAgreedToTerms(e.target.checked)}
                  >
                    <Text size='small' className='text-gray-600'>
                      {t('我已阅读并同意')}
                      {hasUserAgreement && (
                        <>
                          <a
                            href='/user-agreement'
                            target='_blank'
                            rel='noopener noreferrer'
                            className='text-blue-600 hover:text-blue-800 mx-1'
                          >
                            {t('用户协议')}
                          </a>
                        </>
                      )}
                      {hasUserAgreement && hasPrivacyPolicy && t('和')}
                      {hasPrivacyPolicy && (
                        <>
                          <a
                            href='/privacy-policy'
                            target='_blank'
                            rel='noopener noreferrer'
                            className='text-blue-600 hover:text-blue-800 mx-1'
                          >
                            {t('隐私政策')}
                          </a>
                        </>
                      )}
                    </Text>
                  </Checkbox>
                </div>
              )}

              {!status.self_use_mode_enabled && status.register_enabled && (
                <div className='mt-6 text-center text-sm'>
                  <Text>
                    {t('没有账户？')}{' '}
                    <Link
                      to='/register'
                      className='text-blue-600 hover:text-blue-800 font-medium'
                    >
                      {t('注册')}
                    </Link>
                  </Text>
                </div>
              )}
            </div>
          </Card>
        </div>
      </div>
    );
  };

  const renderEmailLoginForm = () => {
    const smsLoginEnabled = Boolean(status.sms_login_enabled);
    const smsLoginIntlEnabled = Boolean(status.sms_login_international_enabled);
    const showSMSTab = smsLoginEnabled;
    const showPasswordTab = status.password_login_enabled !== false;
    const renderPasswordForm = () => (
      <Form className='space-y-3'>
        <Form.Input
          field='username'
          label={t('用户名或邮箱')}
          placeholder={t('请输入您的用户名或邮箱地址')}
          name='username'
          onChange={(value) => handleChange('username', value)}
          prefix={<IconMail />}
        />

        <Form.Input
          field='password'
          label={t('密码')}
          placeholder={t('请输入您的密码')}
          name='password'
          mode='password'
          onChange={(value) => handleChange('password', value)}
          prefix={<IconLock />}
        />

        {(hasUserAgreement || hasPrivacyPolicy) && (
          <div className='pt-4'>
            <Checkbox
              checked={agreedToTerms}
              onChange={(e) => setAgreedToTerms(e.target.checked)}
            >
              <Text size='small' className='text-gray-600'>
                {t('我已阅读并同意')}
                {hasUserAgreement && (
                  <>
                    <a
                      href='/user-agreement'
                      target='_blank'
                      rel='noopener noreferrer'
                      className='text-blue-600 hover:text-blue-800 mx-1'
                    >
                      {t('用户协议')}
                    </a>
                  </>
                )}
                {hasUserAgreement && hasPrivacyPolicy && t('和')}
                {hasPrivacyPolicy && (
                  <>
                    <a
                      href='/privacy-policy'
                      target='_blank'
                      rel='noopener noreferrer'
                      className='text-blue-600 hover:text-blue-800 mx-1'
                    >
                      {t('隐私政策')}
                    </a>
                  </>
                )}
              </Text>
            </Checkbox>
          </div>
        )}

        <div className='space-y-2 pt-2'>
          <Button
            theme='solid'
            className='w-full !rounded-full'
            type='primary'
            htmlType='submit'
            onClick={handleSubmit}
            loading={loginLoading}
            disabled={(hasUserAgreement || hasPrivacyPolicy) && !agreedToTerms}
          >
            {t('继续')}
          </Button>

          <Button
            theme='borderless'
            type='tertiary'
            className='w-full !rounded-full'
            onClick={handleResetPasswordClick}
            loading={resetPasswordLoading}
          >
            {t('忘记密码？')}
          </Button>
        </div>
      </Form>
    );

    const renderSMSForm = () => {
      const renderCountryOption = (c) => (
        <span
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            gap: 6,
            minWidth: 0,
          }}
        >
          <span
            style={{
              fontFamily:
                '"Segoe UI Emoji","Apple Color Emoji","Noto Color Emoji","Twemoji Mozilla",sans-serif',
              fontSize: 16,
              lineHeight: 1,
            }}
          >
            {c.flag}
          </span>
          <span style={{ flexShrink: 0, fontSize: 12 }}>+{c.dial}</span>
        </span>
      );
      const renderSelectedCountry = (c) => (
        <span
          style={{
            display: 'inline-flex',
            alignItems: 'center',
            gap: 4,
            lineHeight: 1,
          }}
        >
          <span
            style={{
              fontFamily:
                '"Segoe UI Emoji","Apple Color Emoji","Noto Color Emoji","Twemoji Mozilla",sans-serif',
              fontSize: 16,
              lineHeight: 1,
            }}
          >
            {c.flag}
          </span>
          <span style={{ fontSize: 12 }}>+{c.dial}</span>
        </span>
      );
      const fieldLabelStyle = {
        marginBottom: 4,
        color: 'var(--semi-color-text-1)',
        fontSize: 14,
      };
      // 国内分支：可选项里只放 CN（其它地区走 SendMessageToGlobe，必须走国际通道）
      const domesticCountryList = COUNTRY_DIAL_CODES.filter(
        (c) => c.code === 'CN',
      );
      const showCountryPicker = smsLoginIntlEnabled;
      return (
        <div className='space-y-3'>
          <Form className='space-y-3'>
            <div>
              <div style={fieldLabelStyle}>{t('手机号')}</div>
              <div
                style={{
                  display: 'flex',
                  gap: 8,
                  alignItems: 'stretch',
                  width: '100%',
                }}
              >
                {showCountryPicker && (
                  <div style={{ width: 100, flexShrink: 0 }}>
                    <Select
                      value={smsInputs.countryCode}
                      onChange={(val) =>
                        setSmsInputs((prev) => ({
                          ...prev,
                          countryCode: val,
                        }))
                      }
                      filter
                      searchPosition='dropdown'
                      renderSelectedItem={(option) => {
                        const c =
                          findCountryByCode(option.value) ||
                          COUNTRY_DIAL_CODES[0];
                        return renderSelectedCountry(c);
                      }}
                      optionList={COUNTRY_DIAL_CODES.map((c) => ({
                        value: c.code,
                        label: renderCountryOption(c),
                        text: `${c.name} +${c.dial} ${c.code}`,
                      }))}
                      style={{ width: '100%' }}
                    />
                  </div>
                )}
                {!showCountryPicker && (
                  <div style={{ width: 100, flexShrink: 0 }}>
                    <Select
                      value='CN'
                      disabled
                      renderSelectedItem={() => renderSelectedCountry(COUNTRY_DIAL_CODES[0])}
                      optionList={domesticCountryList.map((c) => ({
                        value: c.code,
                        label: renderCountryOption(c),
                        text: `${c.name} +${c.dial} ${c.code}`,
                      }))}
                      style={{ width: '100%' }}
                    />
                  </div>
                )}
                <div style={{ flex: 1, minWidth: 0 }}>
                  <Input
                    placeholder={t('13800000000')}
                    value={smsInputs.localNumber}
                    onChange={(value) =>
                      setSmsInputs((prev) => ({
                        ...prev,
                        localNumber: value,
                      }))
                    }
                    prefix={<IconPhone />}
                  />
                </div>
              </div>
            </div>
            <div>
              <div style={fieldLabelStyle}>{t('验证码')}</div>
              <div
                style={{
                  display: 'flex',
                  gap: 8,
                  alignItems: 'stretch',
                  width: '100%',
                }}
              >
                <div style={{ flex: 1, minWidth: 0 }}>
                  <Input
                    placeholder={t('请输入短信验证码')}
                    value={smsInputs.code}
                    onChange={(value) =>
                      setSmsInputs((prev) => ({ ...prev, code: value }))
                    }
                  />
                </div>
                <Button
                  type='tertiary'
                  onClick={handleSMSLoginSendCode}
                  disabled={smsCountdown > 0 || smsSending}
                  loading={smsSending}
                  style={{ height: 32, flexShrink: 0 }}
                >
                  {smsCountdown > 0
                    ? t('{{s}} 秒后重试', { s: smsCountdown })
                    : t('发送验证码')}
                </Button>
              </div>
            </div>
            {(hasUserAgreement || hasPrivacyPolicy) && (
              <div className='pt-2'>
                <Checkbox
                  checked={agreedToTerms}
                  onChange={(e) => setAgreedToTerms(e.target.checked)}
                >
                  <Text size='small' className='text-gray-600'>
                    {t('我已阅读并同意')}
                    {hasUserAgreement && (
                      <a
                        href='/user-agreement'
                        target='_blank'
                        rel='noopener noreferrer'
                        className='text-blue-600 hover:text-blue-800 mx-1'
                      >
                        {t('用户协议')}
                      </a>
                    )}
                    {hasUserAgreement && hasPrivacyPolicy && t('和')}
                    {hasPrivacyPolicy && (
                      <a
                        href='/privacy-policy'
                        target='_blank'
                        rel='noopener noreferrer'
                        className='text-blue-600 hover:text-blue-800 mx-1'
                      >
                        {t('隐私政策')}
                      </a>
                    )}
                  </Text>
                </Checkbox>
              </div>
            )}
            <Button
              theme='solid'
              type='primary'
              className='w-full !rounded-full'
              onClick={handleSMSLoginSubmit}
              loading={smsLoginLoading}
              disabled={
                (hasUserAgreement || hasPrivacyPolicy) && !agreedToTerms
              }
            >
              {t('登录')}
            </Button>
          </Form>
        </div>
      );
    };

    const renderLoginTabs = () => {
      if (!showSMSTab) {
        return renderPasswordForm();
      }
      if (!showPasswordTab) {
        return renderSMSForm();
      }
      return (
        <Tabs
          activeKey={activeLoginTab}
          onChange={(key) => setActiveLoginTab(key)}
          type='line'
        >
          <Tabs.TabPane
            tab={
              <span className='flex items-center gap-1'>
                <IconLock /> {t('密码登录')}
              </span>
            }
            itemKey='password'
          >
            {renderPasswordForm()}
          </Tabs.TabPane>
          <Tabs.TabPane
            tab={
              <span className='flex items-center gap-1'>
                <IconPhone /> {t('手机验证码登录')}
              </span>
            }
            itemKey='sms'
          >
            {renderSMSForm()}
          </Tabs.TabPane>
        </Tabs>
      );
    };

    return (
      <div className='flex flex-col items-center'>
        <div className='w-full max-w-md'>
          <div className='flex items-center justify-center mb-6 gap-2'>
            <img src={logo} alt='Logo' className='h-10 rounded-full' />
            <Title heading={3}>{systemName}</Title>
          </div>

          <Card className='border-0 !rounded-2xl overflow-hidden'>
            <div className='flex justify-center pt-6 pb-2'>
              <Title heading={3} className='text-gray-800 dark:text-gray-200'>
                {t('登 录')}
              </Title>
            </div>
            <div className='px-2 py-8'>
              {status.passkey_login && passkeySupported && (
                <Button
                  theme='outline'
                  type='tertiary'
                  className='w-full h-12 flex items-center justify-center !rounded-full border border-gray-200 hover:bg-gray-50 transition-colors mb-4'
                  icon={<IconKey size='large' />}
                  onClick={handlePasskeyLogin}
                  loading={passkeyLoading}
                >
                  <span className='ml-3'>{t('使用 Passkey 登录')}</span>
                </Button>
              )}
              {renderLoginTabs()}

              {hasOAuthLoginOptions && (
                <>
                  <Divider margin='12px' align='center'>
                    {t('或')}
                  </Divider>

                  <div className='mt-4 text-center'>
                    <Button
                      theme='outline'
                      type='tertiary'
                      className='w-full !rounded-full'
                      onClick={handleOtherLoginOptionsClick}
                      loading={otherLoginOptionsLoading}
                    >
                      {t('其他登录选项')}
                    </Button>
                  </div>
                </>
              )}

              {!status.self_use_mode_enabled && status.register_enabled && (
                <div className='mt-6 text-center text-sm'>
                  <Text>
                    {t('没有账户？')}{' '}
                    <Link
                      to='/register'
                      className='text-blue-600 hover:text-blue-800 font-medium'
                    >
                      {t('注册')}
                    </Link>
                  </Text>
                </div>
              )}
            </div>
          </Card>
        </div>
      </div>
    );
  };

  // 微信登录模态框
  const renderWeChatLoginModal = () => {
    return (
      <Modal
        title={t('微信扫码登录')}
        visible={showWeChatLoginModal}
        maskClosable={true}
        onOk={onSubmitWeChatVerificationCode}
        onCancel={() => setShowWeChatLoginModal(false)}
        okText={t('登录')}
        centered={true}
        okButtonProps={{
          loading: wechatCodeSubmitLoading,
        }}
      >
        <div className='flex flex-col items-center'>
          <img src={status.wechat_qrcode} alt='微信二维码' className='mb-4' />
        </div>

        <div className='text-center mb-4'>
          <p>
            {t('微信扫码关注公众号，输入「验证码」获取验证码（三分钟内有效）')}
          </p>
        </div>

        <Form>
          <Form.Input
            field='wechat_verification_code'
            placeholder={t('验证码')}
            label={t('验证码')}
            value={inputs.wechat_verification_code}
            onChange={(value) =>
              handleChange('wechat_verification_code', value)
            }
          />
        </Form>
      </Modal>
    );
  };

  // 2FA验证弹窗
  const render2FAModal = () => {
    return (
      <Modal
        title={
          <div className='flex items-center'>
            <div className='w-8 h-8 rounded-full bg-green-100 dark:bg-green-900 flex items-center justify-center mr-3'>
              <svg
                className='w-4 h-4 text-green-600 dark:text-green-400'
                fill='currentColor'
                viewBox='0 0 20 20'
              >
                <path
                  fillRule='evenodd'
                  d='M6 8a2 2 0 11-4 0 2 2 0 014 0zM8 7a1 1 0 100 2h8a1 1 0 100-2H8zM6 14a2 2 0 11-4 0 2 2 0 014 0zM8 13a1 1 0 100 2h8a1 1 0 100-2H8z'
                  clipRule='evenodd'
                />
              </svg>
            </div>
            两步验证
          </div>
        }
        visible={showTwoFA}
        onCancel={handleBackToLogin}
        footer={null}
        width={450}
        centered
      >
        <TwoFAVerification
          onSuccess={handle2FASuccess}
          onBack={handleBackToLogin}
          isModal={true}
        />
      </Modal>
    );
  };

  return (
    <div className='relative overflow-hidden bg-gray-100 flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8'>
      {/* 背景模糊晕染球 */}
      <div
        className='blur-ball blur-ball-indigo'
        style={{ top: '-80px', right: '-80px', transform: 'none' }}
      />
      <div
        className='blur-ball blur-ball-teal'
        style={{ top: '50%', left: '-120px' }}
      />
      <div className='w-full max-w-sm mt-[60px]'>
        {showEmailLogin || !hasOAuthLoginOptions
          ? renderEmailLoginForm()
          : renderOAuthOptions()}
        {renderWeChatLoginModal()}
        {render2FAModal()}

        {turnstileEnabled && (
          <div className='flex justify-center mt-6'>
            <Turnstile
              sitekey={turnstileSiteKey}
              onVerify={(token) => {
                setTurnstileToken(token);
              }}
            />
          </div>
        )}
      </div>
    </div>
  );
};

export default LoginForm;
