import React, { useEffect, useState } from 'react';
import { Spin, Typography } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { API } from '../helpers';

function getToken() {
  return new URLSearchParams(window.location.search).get('token') || '';
}

const statusContent = {
  passed: [
    '\u5b9e\u540d\u8ba4\u8bc1\u5b8c\u6210',
    '\u60a8\u7684\u5b9e\u540d\u8ba4\u8bc1\u5df2\u901a\u8fc7\uff0c\u73b0\u5728\u53ef\u4ee5\u9000\u51fa\u5f53\u524d\u9875\u9762\u3002',
  ],
  failed: [
    '\u5b9e\u540d\u8ba4\u8bc1\u672a\u901a\u8fc7',
    '\u8ba4\u8bc1\u672a\u901a\u8fc7\uff0c\u8bf7\u8fd4\u56de\u63a7\u5236\u53f0\u91cd\u65b0\u53d1\u8d77\u8ba4\u8bc1\u3002',
  ],
  expired: [
    '\u8ba4\u8bc1\u5df2\u8fc7\u671f',
    '\u8bf7\u8fd4\u56de\u63a7\u5236\u53f0\u91cd\u65b0\u751f\u6210\u4e8c\u7ef4\u7801\u3002',
  ],
  invalid: [
    '\u8ba4\u8bc1\u94fe\u63a5\u65e0\u6548',
    '\u8bf7\u8fd4\u56de\u63a7\u5236\u53f0\u91cd\u65b0\u53d1\u8d77\u8ba4\u8bc1\u3002',
  ],
  pending: [
    '\u6b63\u5728\u786e\u8ba4\u8ba4\u8bc1\u7ed3\u679c\u2026',
    '\u8bf7\u7a0d\u5019\uff0c\u8ba4\u8bc1\u7ed3\u679c\u786e\u8ba4\u540e\u4f1a\u81ea\u52a8\u66f4\u65b0\u3002',
  ],
};

export default function RealNameVerificationResult() {
  const { t } = useTranslation();
  const [status, setStatus] = useState('pending');
  const token = getToken();

  useEffect(() => {
    const setPageTitle = () => {
      document.title = t('\u5b9e\u540d\u8ba4\u8bc1');
    };
    setPageTitle();
    const titleTimer = window.setTimeout(setPageTitle, 0);
    return () => window.clearTimeout(titleTimer);
  }, [t]);

  useEffect(() => {
    if (!token) {
      setStatus('invalid');
      return undefined;
    }
    let cancelled = false;
    const load = async () => {
      try {
        const response = await API.get(
          `/api/real-name/status?token=${encodeURIComponent(token)}`,
        );
        if (cancelled) return;
        setStatus(
          response.data.success ? response.data.data.status : 'invalid',
        );
      } catch {
        if (!cancelled) setStatus('invalid');
      }
    };
    load();
    const timer = window.setInterval(load, 2500);
    return () => {
      cancelled = true;
      window.clearInterval(timer);
    };
  }, [token]);

  const content = statusContent[status] || statusContent.pending;

  return (
    <div className='min-h-screen flex items-center justify-center p-6 bg-gradient-to-b from-blue-50/70 to-white'>
      <style>{`
        @keyframes real-name-success-pop {
          0% { opacity: 0; transform: scale(0.55); }
          65% { opacity: 1; transform: scale(1.08); }
          100% { opacity: 1; transform: scale(1); }
        }
        @keyframes real-name-success-draw {
          0% { stroke-dashoffset: 48; }
          100% { stroke-dashoffset: 0; }
        }
        @keyframes real-name-success-pulse {
          0%, 100% { opacity: 0.22; transform: scale(0.92); }
          50% { opacity: 0.08; transform: scale(1.16); }
        }
        .real-name-success-icon { animation: real-name-success-pop 520ms cubic-bezier(.2,.9,.3,1.2) both; }
        .real-name-success-icon::before {
          content: '';
          position: absolute;
          inset: -12px;
          border-radius: 9999px;
          background: #16a34a;
          animation: real-name-success-pulse 2.2s ease-in-out infinite;
        }
        .real-name-success-check {
          stroke-dasharray: 48;
          stroke-dashoffset: 48;
          animation: real-name-success-draw 480ms ease-out 320ms forwards;
        }
      `}</style>
      <div className='w-full max-w-sm text-center flex flex-col items-center gap-4'>
        {status === 'pending' ? <Spin spinning size='large' /> : null}
        {status === 'passed' ? (
          <div className='real-name-success-icon relative flex h-20 w-20 items-center justify-center rounded-full bg-green-500 shadow-lg shadow-green-500/25'>
            <svg
              aria-hidden='true'
              className='relative z-10 h-11 w-11'
              viewBox='0 0 48 48'
              fill='none'
            >
              <path
                className='real-name-success-check'
                d='M12 25.5L20.5 34L37 16'
                stroke='white'
                strokeWidth='5'
                strokeLinecap='round'
                strokeLinejoin='round'
              />
            </svg>
          </div>
        ) : null}
        <Typography.Title heading={4}>{t(content[0])}</Typography.Title>
        <Typography.Text type='tertiary'>{t(content[1])}</Typography.Text>
      </div>
    </div>
  );
}
