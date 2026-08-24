/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import React, { useCallback, useState } from 'react';
import { Button } from '@douyinfe/semi-ui';
import { Library } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { MaterialAssetType } from '../../constants';
import {
  PLAYGROUND_MEDIA_UNLIMITED_COUNT,
  PLAYGROUND_VIDEO_FRAME_MAX_COUNT,
} from '../../constants/playground.constants';
import { showError } from '../../helpers';
import { getMaterialAssetUri } from '../../helpers/materialAssetUtils';
import { appendMediaUrlsWithLimit } from '../../helpers/playgroundMediaInputUtils';
import { isVideoFramesTab } from '../../helpers/playgroundVideoUtils';
import MaterialSelectorModal from './MaterialSelectorModal';

/**
 * 视频媒体分栏：独立 SD 素材库入口，置于图片地址输入区外侧上方。
 * 图片按当前 Tab 写入参考图或首尾帧，互不混入同一列表。
 */
const MaterialLibraryButton = ({
  disabled = false,
  videoImageTab,
  imageUrls,
  onImageUrlsChange,
  frameImageUrls,
  onFrameImageUrlsChange,
  videoUrls,
  onVideoUrlsChange,
  audioUrls,
  onAudioUrlsChange,
}) => {
  const { t } = useTranslation();
  const [visible, setVisible] = useState(false);
  const framesTab = isVideoFramesTab({ videoImageTab });

  const handleConfirm = useCallback(
    (selectedAssets) => {
      if (!Array.isArray(selectedAssets) || selectedAssets.length === 0) return;

      const imageUris = [];
      const videoUris = [];
      const audioUris = [];
      for (const asset of selectedAssets) {
        if (!asset || !(asset.asset_uri || asset.asset_id)) continue;
        const uri = getMaterialAssetUri(asset);
        if (asset.asset_type === MaterialAssetType.VIDEO) {
          videoUris.push(uri);
        } else if (asset.asset_type === MaterialAssetType.AUDIO) {
          audioUris.push(uri);
        } else {
          imageUris.push(uri);
        }
      }

      if (imageUris.length > 0) {
        const imageMax = framesTab
          ? PLAYGROUND_VIDEO_FRAME_MAX_COUNT
          : PLAYGROUND_MEDIA_UNLIMITED_COUNT;
        const currentImages = framesTab ? frameImageUrls : imageUrls;
        const { urls, added, skipped } = appendMediaUrlsWithLimit(
          currentImages,
          imageUris,
          imageMax,
        );
        if (added > 0) {
          if (framesTab) {
            onFrameImageUrlsChange?.(urls);
          } else {
            onImageUrlsChange(urls);
          }
        }
        if (skipped > 0) {
          showError(
            framesTab
              ? t('首尾帧最多只能填写 2 张图片')
              : t('部分素材未添加，已达数量上限'),
          );
        }
      }
      if (videoUris.length > 0 && onVideoUrlsChange) {
        const { urls, added, skipped } = appendMediaUrlsWithLimit(
          videoUrls,
          videoUris,
          PLAYGROUND_MEDIA_UNLIMITED_COUNT,
        );
        if (added > 0) {
          onVideoUrlsChange(urls);
        }
        if (skipped > 0) {
          showError(t('部分素材未添加，已达数量上限'));
        }
      }
      if (audioUris.length > 0 && onAudioUrlsChange) {
        const { urls, added, skipped } = appendMediaUrlsWithLimit(
          audioUrls,
          audioUris,
          PLAYGROUND_MEDIA_UNLIMITED_COUNT,
        );
        if (added > 0) {
          onAudioUrlsChange(urls);
        }
        if (skipped > 0) {
          showError(t('部分素材未添加，已达数量上限'));
        }
      }
    },
    [
      audioUrls,
      frameImageUrls,
      framesTab,
      imageUrls,
      onAudioUrlsChange,
      onFrameImageUrlsChange,
      onImageUrlsChange,
      onVideoUrlsChange,
      t,
      videoUrls,
    ],
  );

  return (
    <>
      <div className='playground-media-library-row'>
        <Button
          icon={<Library size={14} />}
          size='small'
          theme='light'
          className='playground-media-action !rounded-lg'
          onClick={() => setVisible(true)}
          disabled={disabled}
        >
          {t('SD 素材库')}
        </Button>
      </div>
      <MaterialSelectorModal
        visible={visible}
        onClose={() => setVisible(false)}
        onConfirm={handleConfirm}
      />
    </>
  );
};

export default MaterialLibraryButton;
