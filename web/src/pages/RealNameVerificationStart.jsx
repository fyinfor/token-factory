import React, { useEffect, useRef, useState } from 'react';
import { Button, Input, Spin, Typography } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { API, showError } from '../helpers';

function getToken() {
  return new URLSearchParams(window.location.search).get('token') || '';
}

export default function RealNameVerificationStart() {
  const { t } = useTranslation();
  const [certName, setCertName] = useState('');
  const [certNo, setCertNo] = useState('');
  const [message, setMessage] = useState(
    t('请填写本人真实姓名和身份证号码后继续。'),
  );
  const [loading, setLoading] = useState(false);
  const startedRef = useRef(false);
  const token = getToken();

  const start = async () => {
    if (startedRef.current) return;
    if (!token) {
      setMessage(t('认证链接无效。'));
      return;
    }
    const normalizedName = certName.trim();
    const normalizedCertNo = certNo.trim().toUpperCase();
    if (!normalizedName || !normalizedCertNo) {
      setMessage(t('请填写真实姓名和身份证号码。'));
      return;
    }
    startedRef.current = true;
    setLoading(true);
    setMessage(t('正在准备实名认证…'));
    try {
      if (!window.getMetaInfo) {
        throw new Error(t('实名认证组件加载失败，请使用手机浏览器重试。'));
      }
      const response = await API.post(
        `/api/real-name/start?token=${encodeURIComponent(token)}`,
        {
          meta_info: JSON.stringify(window.getMetaInfo()),
          cert_name: normalizedName,
          cert_no: normalizedCertNo,
        },
      );
      if (!response.data.success) throw new Error(response.data.message);
      setCertName('');
      setCertNo('');
      window.location.replace(response.data.data.certify_url);
    } catch (error) {
      const nextMessage = error.message || t('发起实名认证失败，请稍后重试。');
      setMessage(nextMessage);
      showError(nextMessage);
      setLoading(false);
      startedRef.current = false;
    }
  };

  const handleSubmit = () => {
    if (loading || startedRef.current) return;
    const script = document.createElement('script');
    script.src = 'https://o.alicdn.com/yd-cloudauth/cloudauth-cdn/jsvm_all.js';
    script.async = true;
    script.onload = start;
    script.onerror = () => {
      startedRef.current = false;
      setLoading(false);
      setMessage(t('实名认证组件加载失败，请检查网络后重试。'));
    };
    document.head.appendChild(script);
  };

  useEffect(() => {
    document.title = t('实名认证');
  }, [t]);

  useEffect(() => {
    return () => {
      startedRef.current = true;
    };
  }, []);

  return (
    <div className='min-h-screen flex items-center justify-center p-6'>
      <style>{`.real-name-mobile-input .semi-input { font-size: 16px !important; }`}</style>
      <div className='w-full max-w-sm flex flex-col gap-4'>
        <div className='text-center flex flex-col gap-2'>
          {loading ? <Spin spinning size='large' /> : null}
          <Typography.Title heading={4}>{t('实名认证')}</Typography.Title>
          <Typography.Text type='tertiary'>{message}</Typography.Text>
        </div>
        <Input
          className='real-name-mobile-input [&_.semi-input]:!text-base'
          value={certName}
          disabled={loading}
          maxLength={64}
          placeholder={t('真实姓名')}
          onChange={setCertName}
        />
        <Input
          className='real-name-mobile-input [&_.semi-input]:!text-base'
          value={certNo}
          disabled={loading}
          maxLength={18}
          pattern='[0-9Xx]{18}'
          placeholder={t('居民身份证号码')}
          onChange={setCertNo}
        />
        <Typography.Text type='tertiary' size='small'>
          {t('证件信息仅用于本次提交至阿里云实名认证服务，不会保存到平台。')}
        </Typography.Text>
        <Button
          theme='solid'
          type='primary'
          loading={loading}
          onClick={handleSubmit}
        >
          {t('继续实名认证')}
        </Button>
      </div>
    </div>
  );
}
