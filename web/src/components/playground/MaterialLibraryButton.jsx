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
import { getMaterialAssetUri } from '../../helpers/materialAssetUtils';
import MaterialSelectorModal from './MaterialSelectorModal';

const appendMaterialUris = (currentUrls, newUris) => {
  const existingFilled = (currentUrls || []).filter((u) =>
    String(u || '').trim(),
  );
  const merged = [...existingFilled, ...newUris];
  if (merged.length === 0) merged.push('');
  return merged;
};

/**
 * 视频媒体分栏：独立 SD 素材库入口，置于图片地址输入区外侧上方。
 */
const MaterialLibraryButton = ({
  disabled = false,
  imageUrls,
  onImageUrlsChange,
  videoUrls,
  onVideoUrlsChange,
  audioUrls,
  onAudioUrlsChange,
}) => {
  const { t } = useTranslation();
  const [visible, setVisible] = useState(false);

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
        onImageUrlsChange(appendMaterialUris(imageUrls, imageUris));
      }
      if (videoUris.length > 0 && onVideoUrlsChange) {
        onVideoUrlsChange(appendMaterialUris(videoUrls, videoUris));
      }
      if (audioUris.length > 0 && onAudioUrlsChange) {
        onAudioUrlsChange(appendMaterialUris(audioUrls, audioUris));
      }
    },
    [
      imageUrls,
      videoUrls,
      audioUrls,
      onImageUrlsChange,
      onVideoUrlsChange,
      onAudioUrlsChange,
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
