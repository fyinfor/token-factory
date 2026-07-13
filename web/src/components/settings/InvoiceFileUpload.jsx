/*
Copyright (C) 2025 QuantumNous
*/

import React, { useState } from 'react';
import { Button, Progress, Typography, Upload } from '@douyinfe/semi-ui';
import { IconClose, IconFile } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../helpers';

const { Text } = Typography;

function isPdfFile(fileInstance) {
  const type = fileInstance?.type || '';
  const name = (fileInstance?.name || '').toLowerCase();
  return type === 'application/pdf' || name.endsWith('.pdf');
}

export default function InvoiceFileUpload({ url = '', onUrlChange, disabled = false }) {
  const { t } = useTranslation();
  const [uploadPct, setUploadPct] = useState(null);

  const customRequest = async ({ file, onSuccess, onError, onProgress }) => {
    const inst = file.fileInstance || file;
    if (!isPdfFile(inst)) {
      showError(t('仅支持 PDF 格式电子发票'));
      onError();
      return;
    }

    const fd = new FormData();
    fd.append('file', inst);
    setUploadPct(0);
    try {
      const res = await API.post('/api/user/invoice/admin/upload', fd, {
        skipErrorHandler: true,
        onUploadProgress: (ev) => {
          const total = ev.total || ev.loaded || 1;
          const pct = Math.min(99, Math.round((ev.loaded * 100) / total));
          setUploadPct(pct);
          if (typeof onProgress === 'function') {
            onProgress({ total, loaded: ev.loaded });
          }
        },
      });
      const { success, message, data } = res.data || {};
      if (!success || !data?.url) {
        onError(new Error(message || 'upload'));
        showError(message || t('上传失败'));
        return;
      }
      setUploadPct(100);
      onUrlChange?.(String(data.url).trim());
      onSuccess(data);
      showSuccess(t('电子发票已上传'));
    } catch (e) {
      onError(e);
      showError(e?.response?.data?.message || t('上传失败'));
    } finally {
      setUploadPct(null);
    }
  };

  const currentUrl = String(url || '').trim();

  return (
    <div className='space-y-2'>
      <Upload
        action=''
        accept='.pdf,application/pdf'
        showUploadList={false}
        customRequest={customRequest}
        disabled={disabled}
      >
        <Button disabled={disabled}>{t('上传电子发票 PDF')}</Button>
      </Upload>
      {uploadPct != null ? (
        <Progress percent={uploadPct} showInfo className='!mt-2' />
      ) : null}
      <Text type='tertiary' size='small' className='block'>
        {t('按运营设置中的文件上传配置保存至本地存储或 OSS；本地存储需挂载目录')}
      </Text>
      {currentUrl ? (
        <div className='flex items-center gap-2 rounded-lg border border-[var(--semi-color-border)] px-3 py-2'>
          <IconFile className='text-[var(--semi-color-primary)]' />
          <Text className='flex-1 truncate' title={currentUrl}>
            {currentUrl}
          </Text>
          <Button
            size='small'
            theme='borderless'
            type='tertiary'
            icon={<IconClose />}
            onClick={() => onUrlChange?.('')}
            disabled={disabled}
          />
        </div>
      ) : null}
    </div>
  );
}
