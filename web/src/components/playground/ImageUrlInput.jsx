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

import React, { useCallback, useState } from 'react';
import { Input, Typography, Button, Upload } from '@douyinfe/semi-ui';
import { IconFile } from '@douyinfe/semi-icons';
import { Plus, X, Image, Upload as UploadIcon, Loader2 } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { PLAYGROUND_MEDIA_MAX_COUNT } from '../../constants/playground.constants';
import { showError, showSuccess } from '../../helpers';
import {
  appendUploadedMediaUrl,
  canAddMoreMediaUrls,
  uploadPlaygroundMediaFile,
} from '../../helpers/playgroundMediaInputUtils';

const ImageUrlInput = ({
  imageUrls,
  imageEnabled,
  onImageUrlsChange,
  disabled = false,
  maxCount = PLAYGROUND_MEDIA_MAX_COUNT,
}) => {
  const { t } = useTranslation();
  const [uploading, setUploading] = useState(false);

  const canAddMore = canAddMoreMediaUrls(imageUrls, maxCount);

  const handleAddImageUrl = () => {
    if (!canAddMore) {
      return;
    }
    onImageUrlsChange([...(imageUrls || []), '']);
  };

  const handleUpdateImageUrl = (index, value) => {
    const newUrls = [...imageUrls];
    newUrls[index] = value;
    onImageUrlsChange(newUrls);
  };

  const handleRemoveImageUrl = (index) => {
    const newUrls = imageUrls.filter((_, i) => i !== index);
    onImageUrlsChange(newUrls.length > 0 ? newUrls : ['']);
  };

  const handleUpload = useCallback(
    async ({ file, onSuccess, onError }) => {
      const inst = file?.fileInstance || file;
      if (!inst) {
        onError?.(new Error('no file'));
        return;
      }

      if (!canAddMoreMediaUrls(imageUrls, maxCount)) {
        showError(
          t('操练场素材已达上限', '最多添加 {{count}} 个', {
            count: maxCount,
          }),
        );
        onError?.(new Error('limit'));
        return;
      }

      setUploading(true);
      try {
        const uploadedUrl = await uploadPlaygroundMediaFile(inst);
        const { urls, ok } = appendUploadedMediaUrl(
          imageUrls,
          uploadedUrl,
          maxCount,
        );
        if (!ok) {
          showError(
            t('操练场素材已达上限', '最多添加 {{count}} 个', {
              count: maxCount,
            }),
          );
          onError?.(new Error('limit'));
          return;
        }
        onImageUrlsChange(urls);
        onSuccess?.({ url: uploadedUrl });
        showSuccess(t('上传成功'));
      } catch (error) {
        const message =
          error?.response?.data?.message ||
          error?.message ||
          t('上传失败，请确认已启用文件上传并完成配置');
        showError(message);
        onError?.(error);
      } finally {
        setUploading(false);
      }
    },
    [imageUrls, maxCount, onImageUrlsChange, t],
  );

  return (
    <div
      className={`playground-media-input playground-media-input-image ${disabled ? 'opacity-50' : ''}`}
    >
      <div className='mb-2 flex flex-wrap items-center justify-between gap-2'>
        <div className='playground-media-action-bar'>
          <Image
            size={16}
            className={
              imageEnabled && !disabled
                ? 'text-[var(--semi-color-primary)]'
                : 'text-[var(--semi-color-text-2)]'
            }
          />
          <Typography.Text strong className='text-sm'>
            {t('图片地址')}
          </Typography.Text>
          {disabled && (
            <Typography.Text className='text-xs text-[var(--semi-color-warning)]'>
              ({t('已在自定义模式中忽略')})
            </Typography.Text>
          )}
        </div>
        <div className='flex items-center gap-2'>
          <Upload
            action=''
            accept='image/*,.jpg,.jpeg,.png,.gif,.webp'
            showUploadList={false}
            customRequest={handleUpload}
            disabled={!imageEnabled || disabled || uploading || !canAddMore}
          >
            <Button
              icon={
                uploading ? (
                  <Loader2 size={14} className='animate-spin' />
                ) : (
                  <UploadIcon size={14} />
                )
              }
              size='small'
              theme='light'
              className='playground-media-action !rounded-lg'
              disabled={!imageEnabled || disabled || uploading || !canAddMore}
            >
              {uploading ? t('上传中') : t('上传')}
            </Button>
          </Upload>
          <Button
            icon={<Plus size={14} />}
            size='small'
            theme='solid'
            type='primary'
            onClick={handleAddImageUrl}
            className='playground-media-action-round'
            disabled={!imageEnabled || disabled || !canAddMore}
          />
        </div>
      </div>

      <div
        className={`space-y-2 max-h-32 overflow-y-auto image-list-scroll ${!imageEnabled || disabled ? 'opacity-50' : ''}`}
      >
        {imageUrls.map((url, index) => (
          <div key={index} className='flex items-center gap-2'>
            <div className='flex-1'>
              <Input
                placeholder={`https://example.com/image${index + 1}.jpg`}
                value={url}
                onChange={(value) => handleUpdateImageUrl(index, value)}
                className='!rounded-lg'
                size='small'
                prefix={<IconFile size='small' />}
                disabled={!imageEnabled || disabled}
              />
            </div>
            <Button
              icon={<X size={12} />}
              size='small'
              theme='borderless'
              type='danger'
              onClick={() => handleRemoveImageUrl(index)}
              className='playground-media-remove-btn flex-shrink-0'
              disabled={!imageEnabled || disabled}
            />
          </div>
        ))}
      </div>
    </div>
  );
};

export default ImageUrlInput;
