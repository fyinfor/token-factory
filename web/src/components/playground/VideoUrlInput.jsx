/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import React from 'react';
import { Input, Typography, Button, Switch } from '@douyinfe/semi-ui';
import { IconFile } from '@douyinfe/semi-icons';
import { Plus, X, Film } from 'lucide-react';
import { useTranslation } from 'react-i18next';

const VideoUrlInput = ({
  videoUrls,
  videoEnabled,
  onVideoUrlsChange,
  onVideoEnabledChange,
  allowToggle = true,
  disabled = false,
}) => {
  const { t } = useTranslation();

  const handleAdd = () => {
    onVideoUrlsChange([...(videoUrls || []), '']);
  };

  const handleUpdate = (index, value) => {
    const next = [...(videoUrls || [])];
    next[index] = value;
    onVideoUrlsChange(next);
  };

  const handleRemove = (index) => {
    onVideoUrlsChange((videoUrls || []).filter((_, i) => i !== index));
  };

  const enabled = videoEnabled !== false;
  const list = videoUrls || [];

  return (
    <div className={`mt-4 ${disabled ? 'opacity-50' : ''}`}>
      <div className='flex items-center justify-between mb-2'>
        <div className='flex items-center gap-2'>
          <Film
            size={16}
            className={enabled && !disabled ? 'text-violet-500' : 'text-gray-400'}
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
          <Button
            icon={<Plus size={14} />}
            size='small'
            theme='solid'
            type='primary'
            onClick={handleAdd}
            className='!rounded-full !w-4 !h-4 !p-0 !min-w-0'
            disabled={!enabled || disabled}
          />
        </div>
      </div>

      {!enabled ? (
        <Typography.Text className='text-xs text-gray-500 mb-2 block'>
          {t('操练场视频地址停用提示', '启用后可添加视频 URL（视频生视频、视频编辑等）')}
        </Typography.Text>
      ) : list.length === 0 ? (
        <Typography.Text className='text-xs text-gray-500 mb-2 block'>
          {t('操练场视频地址空列表提示', '点击 + 添加 .mp4 / .mov 等可访问的视频链接')}
        </Typography.Text>
      ) : (
        <Typography.Text className='text-xs text-gray-500 mb-2 block'>
          {t('已添加')} {list.length} {t('个视频')}
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
