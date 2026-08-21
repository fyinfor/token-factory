/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import React from 'react';
import { useTranslation } from 'react-i18next';
import {
  PLAYGROUND_MEDIA_UNLIMITED_COUNT,
  PLAYGROUND_VIDEO_FRAME_MAX_COUNT,
  PLAYGROUND_VIDEO_IMAGE_TABS,
} from '../../constants/playground.constants';
import { getVideoImageTab } from '../../helpers/playgroundVideoUtils';
import ImageUrlInput from './ImageUrlInput';

const VideoImageTabButton = ({ active, label, onClick, disabled }) => (
  <button
    type='button'
    role='tab'
    aria-selected={active}
    disabled={disabled}
    onClick={onClick}
    className={`playground-media-sub-tab ${active ? 'is-active' : ''}`}
  >
    {label}
  </button>
);

/**
 * 视频媒体：参考图 / 首尾帧分 Tab 独立填写，提交时互斥只带一类参数。
 */
const VideoImageUrlInput = ({
  imageUrls,
  frameImageUrls,
  videoImageTab,
  onImageUrlsChange,
  onFrameImageUrlsChange,
  onVideoImageTabChange,
  disabled = false,
}) => {
  const { t } = useTranslation();
  const tab = getVideoImageTab({ videoImageTab });
  const isFramesTab = tab === PLAYGROUND_VIDEO_IMAGE_TABS.FRAMES;

  const extraHeader = (
    <div className='playground-media-sub-tabs' role='tablist'>
      <VideoImageTabButton
        active={!isFramesTab}
        disabled={disabled}
        label={t('参考图')}
        onClick={() =>
          onVideoImageTabChange(PLAYGROUND_VIDEO_IMAGE_TABS.REFERENCE)
        }
      />
      <VideoImageTabButton
        active={isFramesTab}
        disabled={disabled}
        label={t('首尾帧')}
        onClick={() =>
          onVideoImageTabChange(PLAYGROUND_VIDEO_IMAGE_TABS.FRAMES)
        }
      />
    </div>
  );

  if (isFramesTab) {
    return (
      <ImageUrlInput
        imageUrls={frameImageUrls || ['']}
        imageEnabled={true}
        onImageUrlsChange={onFrameImageUrlsChange}
        disabled={disabled}
        maxCount={PLAYGROUND_VIDEO_FRAME_MAX_COUNT}
        extraHeader={extraHeader}
        hint={t('最多 2 张，分别作为首帧和尾帧；与参考图互斥，不能同时提交')}
        showCount={true}
        placeholder={(index) =>
          index === 0
            ? t('首帧图片地址，例如 https://example.com/first.jpg')
            : t('尾帧图片地址，例如 https://example.com/last.jpg')
        }
      />
    );
  }

  return (
    <ImageUrlInput
      imageUrls={imageUrls || ['']}
      imageEnabled={true}
      onImageUrlsChange={onImageUrlsChange}
      disabled={disabled}
      maxCount={PLAYGROUND_MEDIA_UNLIMITED_COUNT}
      extraHeader={extraHeader}
      hint={t('不限制数量；与首尾帧互斥，请勿在同一请求中同时提交')}
      showCount={true}
    />
  );
};

export default VideoImageUrlInput;
