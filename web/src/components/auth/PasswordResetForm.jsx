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

import React, { useEffect, useState } from 'react';
import {
  API,
  getLogo,
  showError,
  showInfo,
  showSuccess,
  getSystemName,
} from '../../helpers';
import Turnstile from 'react-turnstile';
import { Button, Card, Form, Typography } from '@douyinfe/semi-ui';
import { IconLock } from '@douyinfe/semi-icons';
import { Link, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';

const { Text, Title } = Typography;

const PasswordResetForm = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [inputs, setInputs] = useState({
    phone: '',
    sms_verification_code: '',
    new_password: '',
    confirm_password: '',
  });
  const { phone, sms_verification_code, new_password, confirm_password } =
    inputs;

  const [loading, setLoading] = useState(false);
  const [smsLoading, setSmsLoading] = useState(false);
  const [turnstileEnabled, setTurnstileEnabled] = useState(false);
  const [turnstileSiteKey, setTurnstileSiteKey] = useState('');
  const [turnstileToken, setTurnstileToken] = useState('');
  const [disableSMSButton, setDisableSMSButton] = useState(false);
  const [smsCountdown, setSMSCountdown] = useState(60);

  const logo = getLogo();
  const systemName = getSystemName();

  useEffect(() => {
    let status = localStorage.getItem('status');
    if (status) {
      status = JSON.parse(status);
      if (status.turnstile_check) {
        setTurnstileEnabled(true);
        setTurnstileSiteKey(status.turnstile_site_key);
      }
    }
  }, []);

  useEffect(() => {
    let countdownInterval = null;
    if (disableSMSButton && smsCountdown > 0) {
      countdownInterval = setInterval(() => {
        setSMSCountdown(smsCountdown - 1);
      }, 1000);
    } else if (smsCountdown === 0) {
      setDisableSMSButton(false);
      setSMSCountdown(60);
    }
    return () => clearInterval(countdownInterval);
  }, [disableSMSButton, smsCountdown]);

  /** handleChange 统一处理表单字段修改。 */
  function handleChange(name, value) {
    setInputs((prev) => ({ ...prev, [name]: value }));
  }

  /** sendSMSCode 发送找回密码短信验证码（5分钟有效）。 */
  async function sendSMSCode() {
    if (!/^1[3-9]\d{9}$/.test((phone || '').trim())) {
      showInfo(t('请输入有效的 11 位手机号'));
      return;
    }
    if (turnstileEnabled && turnstileToken === '') {
      showInfo(t('请稍后几秒重试，Turnstile 正在检查用户环境！'));
      return;
    }
    setSmsLoading(true);
    try {
      const res = await API.get(
        `/api/reset_password_sms?phone=${encodeURIComponent(phone.trim())}&turnstile=${turnstileToken}`,
      );
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('短信验证码发送成功，请注意查收（5分钟内有效）'));
        setDisableSMSButton(true);
      } else {
        showError(message);
      }
    } catch (error) {
      showError(
        error?.response?.data?.message || t('发送短信验证码失败，请重试'),
      );
    } finally {
      setSmsLoading(false);
    }
  }

  /** handleSubmit 通过手机号+短信验证码重置密码。 */
  async function handleSubmit(e) {
    if (!/^1[3-9]\d{9}$/.test((phone || '').trim())) {
      showError(t('请输入有效的 11 位手机号'));
      return;
    }
    if (!/^\d{6}$/.test((sms_verification_code || '').trim())) {
      showError(t('请输入 6 位短信验证码'));
      return;
    }
    if ((new_password || '').length < 8 || (new_password || '').length > 20) {
      showError(t('密码长度需为 8-20 位'));
      return;
    }
    if (new_password !== confirm_password) {
      showError(t('两次输入的密码不一致'));
      return;
    }
    if (turnstileEnabled && turnstileToken === '') {
      showInfo(t('请稍后几秒重试，Turnstile 正在检查用户环境！'));
      return;
    }
    setLoading(true);
    try {
      const res = await API.post(`/api/user/reset_by_phone`, {
        phone: phone.trim(),
        sms_verification_code: sms_verification_code.trim(),
        new_password,
        confirm_password,
      });
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('密码重置成功，请使用新密码登录'));
        setInputs({
          phone: '',
          sms_verification_code: '',
          new_password: '',
          confirm_password: '',
        });
        navigate('/login');
      } else {
        showError(message);
      }
    } catch (error) {
      showError(error?.response?.data?.message || t('密码重置失败，请重试'));
    }
    setLoading(false);
  }

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
                  {t('密码重置')}
                </Title>
              </div>
              <div className='px-2 py-8'>
                <Form className='space-y-3'>
                  <Form.Input
                    field='phone'
                    label={t('手机号')}
                    placeholder={t('输入 11 位手机号')}
                    name='phone'
                    value={phone}
                    onChange={(value) => handleChange('phone', value)}
                  />
                  <Form.Input
                    field='sms_verification_code'
                    label={t('短信验证码')}
                    placeholder={t('输入短信验证码')}
                    name='sms_verification_code'
                    value={sms_verification_code}
                    onChange={(value) =>
                      handleChange('sms_verification_code', value)
                    }
                    suffix={
                      <Button
                        size='small'
                        loading={smsLoading}
                        disabled={disableSMSButton || smsLoading}
                        onClick={sendSMSCode}
                      >
                        {disableSMSButton
                          ? `${t('重新发送')} (${smsCountdown})`
                          : t('获取验证码')}
                      </Button>
                    }
                  />
                  <Form.Input
                    field='new_password'
                    label={t('新密码')}
                    placeholder={t('输入新密码，最短 8 位，最长 20 位')}
                    name='new_password'
                    value={new_password}
                    mode='password'
                    onChange={(value) => handleChange('new_password', value)}
                    prefix={<IconLock />}
                  />
                  <Form.Input
                    field='confirm_password'
                    label={t('确认新密码')}
                    placeholder={t('再次输入新密码')}
                    name='confirm_password'
                    value={confirm_password}
                    mode='password'
                    onChange={(value) =>
                      handleChange('confirm_password', value)
                    }
                    prefix={<IconLock />}
                  />

                  <div className='space-y-2 pt-2'>
                    <Button
                      theme='solid'
                      className='w-full !rounded-full'
                      type='primary'
                      htmlType='submit'
                      onClick={handleSubmit}
                      loading={loading}
                    >
                      {t('确认重置密码')}
                    </Button>
                  </div>
                </Form>

                <div className='mt-6 text-center text-sm'>
                  <Text>
                    {t('想起来了？')}{' '}
                    <Link
                      to='/login'
                      className='text-blue-600 hover:text-blue-800 font-medium'
                    >
                      {t('登录')}
                    </Link>
                  </Text>
                </div>
              </div>
            </Card>

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
      </div>
    </div>
  );
};

export default PasswordResetForm;
