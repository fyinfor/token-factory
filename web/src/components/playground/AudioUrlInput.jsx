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
import { Plus, X, Music, Upload as UploadIcon, Loader2 } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { PLAYGROUND_MEDIA_MAX_COUNT } from '../../constants/playground.constants';
import { showError, showSuccess } from '../../helpers';
import {
  appendUploadedMediaUrl,
  canAddMoreMediaUrls,
  countFilledMediaUrls,
  uploadPlaygroundMediaFile,
} from '../../helpers/playgroundMediaInputUtils';

/**
 * 操练场视频媒体：参考音频链接输入 + 本地音频上传。
 * 写入 inputs.audioUrls，生成请求时进入 metadata.audio_urls。
 */
const AudioUrlInput = ({
  audioUrls,
  audioEnabled,
  onAudioUrlsChange,
  onAudioEnabledChange,
  allowToggle = true,
  disabled = false,
  maxCount = PLAYGROUND_MEDIA_MAX_COUNT,
}) => {
  const { t } = useTranslation();
  const [uploading, setUploading] = useState(false);

  const enabled = audioEnabled !== false;
  const list = audioUrls || [];
  const filledCount = countFilledMediaUrls(list);
  const canAddMore = canAddMoreMediaUrls(list, maxCount);

  const handleAdd = () => {
    if (!canAddMore) {
      return;
    }
    onAudioUrlsChange([...list, '']);
  };

  const handleUpdate = (index, value) => {
    const next = [...list];
    next[index] = value;
    onAudioUrlsChange(next);
  };

  const handleRemove = (index) => {
    const next = list.filter((_, i) => i !== index);
    onAudioUrlsChange(next.length > 0 ? next : ['']);
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
        onAudioUrlsChange(urls);
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
    [list, maxCount, onAudioUrlsChange, t],
  );

  return (
    <div
      className={`playground-media-input playground-media-input-audio ${disabled ? 'opacity-50' : ''}`}
    >
      <div className='mb-2 flex flex-wrap items-center justify-between gap-2'>
        <div className='playground-media-action-bar'>
          <Music
            size={16}
            className={
              enabled && !disabled
                ? 'text-[var(--semi-color-primary)]'
                : 'text-[var(--semi-color-text-2)]'
            }
          />
          <Typography.Text strong className='text-sm'>
            {t('音频地址')}
          </Typography.Text>
          {disabled && (
            <Typography.Text className='text-xs text-[var(--semi-color-warning)]'>
              ({t('已在自定义模式中忽略')})
            </Typography.Text>
          )}
        </div>
        <div className='flex items-center gap-2'>
          {allowToggle && (
            <Switch
              checked={enabled}
              onChange={onAudioEnabledChange}
              checkedText={t('启用')}
              uncheckedText={t('停用')}
              size='small'
              className='flex-shrink-0'
              disabled={disabled}
            />
          )}
          <Upload
            action=''
            accept='audio/*,.mp3,.wav,.m4a,.aac,.ogg,.flac'
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
              className='playground-media-action !rounded-lg'
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
            className='playground-media-action-round'
            disabled={!enabled || disabled || !canAddMore}
          />
        </div>
      </div>

      {enabled && list.length > 0 && (
        <Typography.Text className='mb-2 block text-xs text-[var(--semi-color-text-2)]'>
          {t('已添加')} {filledCount}/{maxCount} {t('个音频')}
        </Typography.Text>
      )}

      <div
        className={`space-y-2 max-h-32 overflow-y-auto image-list-scroll ${!enabled || disabled ? 'opacity-50' : ''}`}
      >
        {list.map((url, index) => (
          <div key={index} className='flex items-center gap-2'>
            <div className='flex-1'>
              <Input
                placeholder={`https://example.com/audio${index + 1}.mp3`}
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
              className='playground-media-remove-btn flex-shrink-0'
              disabled={!enabled || disabled}
            />
          </div>
        ))}
      </div>
    </div>
  );
};

export default AudioUrlInput;
