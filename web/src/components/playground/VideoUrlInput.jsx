/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import React, { useCallback, useState } from 'react';
import { Input, Typography, Button, Switch, Upload } from '@douyinfe/semi-ui';
import { IconFile } from '@douyinfe/semi-icons';
import { Plus, X, Film, Upload as UploadIcon, Loader2 } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { PLAYGROUND_MEDIA_MAX_COUNT } from '../../constants/playground.constants';
import { showError, showSuccess } from '../../helpers';
import {
  appendUploadedMediaUrl,
  canAddMoreMediaUrls,
  countFilledMediaUrls,
  uploadPlaygroundMediaFile,
} from '../../helpers/playgroundMediaInputUtils';

const VideoUrlInput = ({
  videoUrls,
  videoEnabled,
  onVideoUrlsChange,
  onVideoEnabledChange,
  allowToggle = true,
  disabled = false,
  maxCount = PLAYGROUND_MEDIA_MAX_COUNT,
}) => {
  const { t } = useTranslation();
  const [uploading, setUploading] = useState(false);

  const enabled = videoEnabled !== false;
  const list = videoUrls || [];
  const filledCount = countFilledMediaUrls(list);
  const canAddMore = canAddMoreMediaUrls(list, maxCount);

  const handleAdd = () => {
    if (!canAddMore) {
      return;
    }
    onVideoUrlsChange([...list, '']);
  };

  const handleUpdate = (index, value) => {
    const next = [...list];
    next[index] = value;
    onVideoUrlsChange(next);
  };

  const handleRemove = (index) => {
    const next = list.filter((_, i) => i !== index);
    onVideoUrlsChange(next.length > 0 ? next : ['']);
  };

  const handleUpload = useCallback(
    async ({ file, onSuccess, onError }) => {
      const inst = file?.fileInstance || file;
      if (!inst) {
        onError?.(new Error('no file'));
        return;
      }

      if (!canAddMoreMediaUrls(list, maxCount)) {
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
          list,
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
        onVideoUrlsChange(urls);
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
    [list, maxCount, onVideoUrlsChange, t],
  );

  return (
    <div
      className={`playground-media-input playground-media-input-video ${disabled ? 'opacity-50' : ''}`}
    >
      <div className='flex items-center justify-between mb-2'>
        <div className='flex items-center gap-2'>
          <Film
            size={16}
            className={
              enabled && !disabled ? 'text-violet-500' : 'text-gray-400'
            }
          />
          <Typography.Text strong className='text-sm'>
            {t('视频地址')}
          </Typography.Text>
          {disabled && (
            <Typography.Text className='text-xs text-orange-600'>
              ({t('已在自定义模式中忽略')})
            </Typography.Text>
          )}
        </div>
        <div className='flex items-center gap-2'>
          {allowToggle && (
            <Switch
              checked={enabled}
              onChange={onVideoEnabledChange}
              checkedText={t('启用')}
              uncheckedText={t('停用')}
              size='small'
              className='flex-shrink-0'
              disabled={disabled}
            />
          )}
          <Upload
            action=''
            accept='video/*,.mp4,.mov,.webm,.avi,.mkv'
            showUploadList={false}
            customRequest={handleUpload}
            disabled={!enabled || disabled || uploading || !canAddMore}
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
              className='!rounded-lg'
              disabled={!enabled || disabled || uploading || !canAddMore}
            >
              {uploading ? t('上传中') : t('上传')}
            </Button>
          </Upload>
          <Button
            icon={<Plus size={14} />}
            size='small'
            theme='solid'
            type='primary'
            onClick={handleAdd}
            className='!rounded-full !w-4 !h-4 !p-0 !min-w-0'
            disabled={!enabled || disabled || !canAddMore}
          />
        </div>
      </div>

      {!enabled ? (
        <Typography.Text className='text-xs text-gray-500 mb-2 block'>
          {t(
            '操练场视频地址停用提示',
            '启用后可添加视频 URL（视频生视频、视频编辑等）',
          )}
        </Typography.Text>
      ) : list.length === 0 ? (
        <Typography.Text className='text-xs text-gray-500 mb-2 block'>
          {t(
            '操练场视频地址空列表提示',
            '点击上传或 + 添加 .mp4 / .mov 等可访问的视频链接',
          )}
        </Typography.Text>
      ) : (
        <Typography.Text className='text-xs text-gray-500 mb-2 block'>
          {t('已添加')} {filledCount}/{maxCount} {t('个视频')}
        </Typography.Text>
      )}

      <div
        className={`space-y-2 max-h-32 overflow-y-auto image-list-scroll ${!enabled || disabled ? 'opacity-50' : ''}`}
      >
        {list.map((url, index) => (
          <div key={index} className='flex items-center gap-2'>
            <div className='flex-1'>
              <Input
                placeholder={`https://example.com/video${index + 1}.mp4`}
                value={url}
                onChange={(value) => handleUpdate(index, value)}
                className='!rounded-lg'
                size='small'
                prefix={<IconFile size='small' />}
                disabled={!enabled || disabled}
              />
            </div>
            <Button
              icon={<X size={12} />}
              size='small'
              theme='borderless'
              type='danger'
              onClick={() => handleRemove(index)}
              className='!rounded-full !w-6 !h-6 !p-0 !min-w-0 !text-red-500 hover:!bg-red-50 flex-shrink-0'
              disabled={!enabled || disabled}
            />
          </div>
        ))}
      </div>
    </div>
  );
};

export default VideoUrlInput;
