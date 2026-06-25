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
import {
  Plus,
  X,
  Image,
  Upload as UploadIcon,
  Loader2,
  Library,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { PLAYGROUND_MEDIA_MAX_COUNT } from '../../constants/playground.constants';
import { CHANNEL_TYPE_SEEDANCE } from '../../constants/channel.constants';
import { getMaterialAssetUri } from '../../helpers/materialAssetUtils';
import MaterialSelectorModal from './MaterialSelectorModal';
import { showError, showSuccess } from '../../helpers';
import {
  appendUploadedMediaUrl,
  canAddMoreMediaUrls,
  countFilledMediaUrls,
  uploadPlaygroundMediaFile,
} from '../../helpers/playgroundMediaInputUtils';

const ImageUrlInput = ({
  imageUrls,
  imageEnabled,
  onImageUrlsChange,
  disabled = false,
  maxCount = PLAYGROUND_MEDIA_MAX_COUNT,
  channelType,
}) => {
  const { t } = useTranslation();
  const [uploading, setUploading] = useState(false);
  // 【需求3】素材库弹窗状态（仅渠道类型 === 65 时可用）
  const [materialModalVisible, setMaterialModalVisible] = useState(false);
  const isSeedanceChannel = Number(channelType) === CHANNEL_TYPE_SEEDANCE;

  const filledCount = countFilledMediaUrls(imageUrls);
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

  // 【需求4/5】素材库确认回调：将选中素材的 asset:// 地址填充到图片地址输入框
  const handleMaterialConfirm = useCallback(
    (selectedAssets) => {
      if (!Array.isArray(selectedAssets) || selectedAssets.length === 0) return;
      const currentUrls = imageUrls || [];
      const newUris = selectedAssets
        .filter((a) => a && (a.asset_uri || a.asset_id))
        .map((a) => getMaterialAssetUri(a));
      // 合并已有非空 URL + 新选素材 URI
      const existingFilled = currentUrls.filter((u) => String(u || '').trim());
      const merged = [...existingFilled, ...newUris];
      // 确保至少保留一个空槽位
      if (merged.length === 0) merged.push('');
      onImageUrlsChange(merged);
    },
    [imageUrls, onImageUrlsChange],
  );

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
              imageEnabled && !disabled ? 'text-blue-500' : 'text-gray-400'
            }
          />
          <Typography.Text strong className='text-sm'>
            {t('图片地址')}
          </Typography.Text>
          {disabled && (
            <Typography.Text className='text-xs text-orange-600'>
              ({t('已在自定义模式中忽略')})
            </Typography.Text>
          )}
        </div>
        <div className='flex items-center gap-2'>
          {/* 【需求3】渠道权限显隐控制：仅火山方舟-Seedance 2.0 视频渠道显示素材库按钮 */}
          {isSeedanceChannel && (
            <Button
              icon={<Library size={14} />}
              size='small'
              theme='light'
              className='playground-media-action !rounded-lg'
              onClick={() => setMaterialModalVisible(true)}
              disabled={!imageEnabled || disabled}
            >
              {t('素材库')}
            </Button>
          )}
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
              className='!rounded-full !w-6 !h-6 !p-0 !min-w-0 !text-red-500 hover:!bg-red-50 flex-shrink-0'
              disabled={!imageEnabled || disabled}
            />
          </div>
        ))}
      </div>
      {/* 【需求4】素材库选择弹窗 */}
      {isSeedanceChannel && (
        <MaterialSelectorModal
          visible={materialModalVisible}
          onClose={() => setMaterialModalVisible(false)}
          onConfirm={handleMaterialConfirm}
        />
      )}
    </div>
  );
};

export default ImageUrlInput;
