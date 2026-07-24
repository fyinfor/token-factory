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

import React, { useState } from 'react';
import { Progress, Upload } from '@douyinfe/semi-ui';
import { FileCheck2, FileUp, Trash2 } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../helpers';
import { InvoiceSpinner } from '../invoice/InvoiceWorkspace';

function isPdfFile(file) {
  return (
    file?.type === 'application/pdf' ||
    file?.name?.toLowerCase().endsWith('.pdf')
  );
}

export default function InvoiceFileUpload({
  url = '',
  onUrlChange,
  disabled = false,
}) {
  const { t } = useTranslation();
  const [uploadPct, setUploadPct] = useState(null);

  const uploadFile = async ({ file, onSuccess, onError, onProgress }) => {
    const fileInstance = file?.fileInstance || file;
    if (!isPdfFile(fileInstance)) {
      showError(t('仅支持 PDF 格式电子发票'));
      onError?.(new Error('invalid pdf'));
      return;
    }
    const form = new FormData();
    form.append('file', fileInstance);
    setUploadPct(0);
    try {
      const res = await API.post('/api/user/invoice/admin/upload', form, {
        skipErrorHandler: true,
        onUploadProgress: (event) => {
          const total = event.total || event.loaded || 1;
          const percent = Math.min(
            99,
            Math.round((event.loaded * 100) / total),
          );
          setUploadPct(percent);
          onProgress?.({ total, loaded: event.loaded });
        },
      });
      const { success, message, data } = res.data || {};
      if (!success || !data?.url) {
        showError(message || t('上传失败'));
        onError?.(new Error(message || 'upload failed'));
        return;
      }
      setUploadPct(100);
      onUrlChange?.(String(data.url).trim());
      onSuccess?.(data);
      showSuccess(t('电子发票已上传'));
    } catch (error) {
      onError?.(error);
      showError(error?.response?.data?.message || t('上传失败'));
    } finally {
      setUploadPct(null);
    }
  };

  const currentUrl = String(url || '').trim();
  return (
    <div className='invoice-upload'>
      <Upload
        action=''
        className='invoice-semi-upload'
        accept='.pdf,application/pdf'
        customRequest={uploadFile}
        showUploadList={false}
        disabled={disabled || uploadPct != null}
      >
        <span className='invoice-button'>
          {uploadPct != null ? <InvoiceSpinner /> : <FileUp size={16} />}
          {t('上传电子发票 PDF')}
        </span>
      </Upload>
      {uploadPct != null ? (
        <Progress
          className='invoice-progress'
          percent={uploadPct}
          showInfo={false}
          size='small'
          aria-label={t('上传进度')}
        />
      ) : null}
      <small>
        {t('文件将按运营设置保存至本地存储或 OSS，开具时仅接受已上传的 PDF')}
      </small>
      {currentUrl ? (
        <div className='invoice-upload-file'>
          <FileCheck2 size={17} aria-hidden='true' />
          <span title={currentUrl}>{currentUrl}</span>
          <button
            type='button'
            className='invoice-icon-button'
            disabled={disabled}
            onClick={() => onUrlChange?.('')}
            title={t('移除文件')}
            aria-label={t('移除文件')}
          >
            <Trash2 size={15} />
          </button>
        </div>
      ) : null}
    </div>
  );
}
