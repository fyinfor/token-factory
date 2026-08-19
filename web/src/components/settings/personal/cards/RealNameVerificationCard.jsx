import React, { useEffect, useRef, useState } from 'react';
import { Button, Modal, Space, Tag, Typography } from '@douyinfe/semi-ui';
import { QRCodeSVG } from 'qrcode.react';
import {
  API,
  renderQuotaRounded,
  showError,
  showSuccess,
} from '../../../../helpers';
import { getQuotaPerUnit } from '../../../../helpers/quota';
import { useTranslation } from 'react-i18next';
import { BadgeCheck, Check, Gift, ScanFace, ShieldCheck } from 'lucide-react';

const statusConfig = {
  passed: {
    color: 'green',
    label: '\u5df2\u5b9e\u540d',
    title: '\u8eab\u4efd\u5df2\u5b89\u5168\u9a8c\u8bc1',
  },
  pending: {
    color: 'orange',
    label: '\u8ba4\u8bc1\u4e2d',
    title: '\u7b49\u5f85\u5b8c\u6210\u8ba4\u8bc1',
  },
  failed: {
    color: 'red',
    label: '\u8ba4\u8bc1\u5931\u8d25',
    title: '\u8bf7\u91cd\u65b0\u53d1\u8d77\u8ba4\u8bc1',
  },
  expired: {
    color: 'grey',
    label: '\u5df2\u8fc7\u671f',
    title: '\u8ba4\u8bc1\u94fe\u63a5\u5df2\u8fc7\u671f',
  },
  unverified: {
    color: 'grey',
    label: '\u672a\u8ba4\u8bc1',
    title: '\u5b8c\u6210\u91d1\u878d\u7ea7\u5b9e\u540d\u8ba4\u8bc1',
  },
};

export default function RealNameVerificationCard() {
  const { t } = useTranslation();
  const [verification, setVerification] = useState({ status: 'unverified' });
  const [mobileURL, setMobileURL] = useState('');
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const pollingRef = useRef();
  const statusRef = useRef('unverified');
  const shouldNotifyPassedRef = useRef(false);
  const loadStatus = async () => {
    const response = await API.get('/api/user/self/real-name-verification');
    if (!response.data.success) return;
    const nextVerification = response.data.data;
    const previousStatus = statusRef.current;
    statusRef.current = nextVerification.status;
    setVerification(nextVerification);
    if (
      shouldNotifyPassedRef.current &&
      previousStatus !== 'passed' &&
      nextVerification.status === 'passed'
    ) {
      shouldNotifyPassedRef.current = false;
      showSuccess(t('\u5b9e\u540d\u8ba4\u8bc1\u5b8c\u6210'));
    }
    if (['passed', 'failed', 'expired'].includes(nextVerification.status)) {
      shouldNotifyPassedRef.current = false;
      setModalVisible(false);
      setMobileURL('');
    }
  };

  useEffect(() => {
    loadStatus().catch(() => {});
    return () => window.clearInterval(pollingRef.current);
  }, []);
  useEffect(() => {
    window.clearInterval(pollingRef.current);
    if (!modalVisible) return undefined;
    loadStatus().catch(() => {});
    pollingRef.current = window.setInterval(
      () => loadStatus().catch(() => {}),
      2500,
    );
    return () => window.clearInterval(pollingRef.current);
  }, [modalVisible]);

  const createVerification = async () => {
    setLoading(true);
    try {
      const response = await API.post('/api/user/self/real-name-verification');
      if (!response.data.success) {
        showError(response.data.message);
        return;
      }
      statusRef.current = 'pending';
      shouldNotifyPassedRef.current = true;
      setVerification((current) => ({ ...current, status: 'pending' }));
      setMobileURL(response.data.data.mobile_url);
      setModalVisible(true);
    } catch {
      showError(t('\u53d1\u8d77\u5b9e\u540d\u8ba4\u8bc1\u5931\u8d25'));
    } finally {
      setLoading(false);
    }
  };

  const closeVerificationModal = () => {
    setModalVisible(false);
    setMobileURL('');
    if (verification.status === 'pending') {
      statusRef.current = 'unverified';
      setVerification((current) => ({ ...current, status: 'unverified' }));
      loadStatus().catch(() => {});
    }
  };

  const config = statusConfig[verification.status] || statusConfig.unverified;
  const rewardQuota = Math.round(
    Number(verification.reward_amount || 0) * getQuotaPerUnit(),
  );
  return (
    <div className='real-name-card'>
      <div className='flex items-center justify-between gap-4'>
        <div className='flex min-w-0 items-center gap-3'>
          <div className='real-name-icon real-name-icon--primary'>
            <ScanFace size={20} />
          </div>
          <div className='min-w-0'>
            <div className='real-name-card__title'>
              {t('\u5b9e\u540d\u8ba4\u8bc1')}
            </div>
            <Typography.Text className='real-name-card__subtitle'>
              {t(
                '\u7531\u652f\u4ed8\u5b9d\u4e0e\u963f\u91cc\u4e91\u5b8c\u6210\u5b89\u5168\u8ba4\u8bc1',
              )}
            </Typography.Text>
          </div>
        </div>
        <Tag
          color={config.color}
          className='!w-auto !shrink-0 !rounded-full !px-3 !py-1'
        >
          {t(config.label)}
        </Tag>
      </div>

      <div className='mt-8 flex gap-4'>
        <div className='real-name-icon real-name-icon--success'>
          {verification.status === 'passed' ? (
            <BadgeCheck size={21} />
          ) : (
            <ShieldCheck size={21} />
          )}
        </div>
        <div className='min-w-0 flex-1 pt-0.5'>
          <div className='real-name-card__section-title'>{t(config.title)}</div>
          <Typography.Paragraph className='real-name-card__paragraph'>
            {t(
              '\u4ec5\u4fdd\u5b58\u8ba4\u8bc1\u72b6\u6001\u4e0e\u963f\u91cc\u4e91\u6d41\u6c34\u53f7\uff0c\u4e0d\u4fdd\u5b58\u8eab\u4efd\u8bc1\u3001\u4eba\u8138\u7b49\u8ba4\u8bc1\u6750\u6599\u3002',
            )}
          </Typography.Paragraph>
        </div>
      </div>

      {verification.reward_enabled && verification.reward_amount > 0 ? (
        <div className='real-name-reward'>
          <div className='real-name-icon real-name-icon--reward'>
            <Gift size={20} />
          </div>
          <div className='min-w-0 flex-1'>
            <div className='flex flex-wrap items-baseline gap-x-2'>
              <span className='real-name-reward__label'>
                {t('\u5b9e\u540d\u5956\u52b1')}
              </span>
              <span className='real-name-reward__amount'>
                {renderQuotaRounded(rewardQuota)}
              </span>
            </div>
            <div className='real-name-reward__desc'>
              {verification.reward_granted
                ? t('\u5df2\u53d1\u653e\u81f3\u8d60\u9001\u4f59\u989d')
                : t(
                    '\u8ba4\u8bc1\u901a\u8fc7\u540e\u81ea\u52a8\u53d1\u653e\u4e00\u6b21',
                  )}
            </div>
          </div>
        </div>
      ) : null}

      <Button
        size='large'
        block
        type='primary'
        loading={loading}
        disabled={verification.status === 'passed'}
        onClick={createVerification}
        className='!mt-6 !h-12 !rounded-2xl'
      >
        {verification.status === 'passed'
          ? t('\u5b9e\u540d\u8ba4\u8bc1\u5df2\u5b8c\u6210')
          : t('\u5f00\u59cb\u5b9e\u540d\u8ba4\u8bc1')}
      </Button>

      <div className='real-name-footnotes'>
        <span className='flex items-center gap-1.5'>
          <Check size={13} />
          {t('\u652f\u4ed8\u5b9d\u5b89\u5168\u8ba4\u8bc1')}
        </span>
        <span className='flex items-center gap-1.5'>
          <Check size={13} />
          {t('\u4e0d\u4fdd\u5b58\u8bc1\u4ef6\u6750\u6599')}
        </span>
        <span className='flex items-center gap-1.5'>
          <Check size={13} />
          {t('\u8ba4\u8bc1\u7ed3\u679c\u7ed1\u5b9a\u8d26\u6237')}
        </span>
      </div>

      <Modal
        title={t('\u626b\u7801\u5f00\u59cb\u5b9e\u540d\u8ba4\u8bc1')}
        visible={modalVisible}
        footer={null}
        onCancel={closeVerificationModal}
        centered
      >
        <div className='flex flex-col items-center gap-4 py-4 text-center'>
          <div className='real-name-qr-frame'>
            <QRCodeSVG
              value={mobileURL || 'about:blank'}
              size={220}
              includeMargin
            />
          </div>
          <Typography.Text>
            {t(
              '\u8bf7\u4f7f\u7528\u624b\u673a\u626b\u7801\uff0c\u5728\u652f\u4ed8\u5b9d\u9875\u9762\u5b8c\u6210\u5b9e\u540d\u8ba4\u8bc1\u3002',
            )}
          </Typography.Text>
          <Typography.Text type='tertiary' size='small'>
            {t(
              '\u4e8c\u7ef4\u7801\u6709\u6548\u671f\u4e3a 15 \u5206\u949f\uff0c\u8ba4\u8bc1\u5b8c\u6210\u540e\u672c\u9875\u4f1a\u81ea\u52a8\u66f4\u65b0\u3002',
            )}
          </Typography.Text>
          <Space>
            <Button onClick={() => loadStatus().catch(() => {})}>
              {t('\u5237\u65b0\u72b6\u6001')}
            </Button>
            <Button type='primary' onClick={closeVerificationModal}>
              {t('\u5173\u95ed')}
            </Button>
          </Space>
        </div>
      </Modal>
    </div>
  );
}
