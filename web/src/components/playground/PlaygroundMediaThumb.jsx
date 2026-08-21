/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import React, { useEffect, useState } from 'react';
import { Modal } from '@douyinfe/semi-ui';
import { Film, Image as ImageIcon, Play } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { useResolvedPlaygroundPreviewUrl } from '../../helpers/playgroundMediaPreview';

const PlaygroundMediaThumb = ({ value, kind = 'image', disabled = false }) => {
  const { t } = useTranslation();
  const src = useResolvedPlaygroundPreviewUrl(value);
  const [failed, setFailed] = useState(false);
  const [open, setOpen] = useState(false);
  const isVideo = kind === 'video';
  const canPreview = Boolean(src) && !failed;

  useEffect(() => {
    setFailed(false);
    setOpen(false);
  }, [src]);

  const placeholder = (
    <span className='playground-media-thumb-placeholder'>
      {isVideo ? <Film size={16} /> : <ImageIcon size={16} />}
    </span>
  );

  return (
    <>
      <button
        type='button'
        className={`playground-media-thumb ${canPreview ? 'is-ready' : ''}`}
        disabled={disabled || !canPreview}
        title={
          canPreview ? (isVideo ? t('视频预览') : t('图片预览')) : t('暂无预览')
        }
        onClick={() => {
          if (!canPreview || disabled) {
            return;
          }
          setOpen(true);
        }}
      >
        {!canPreview ? (
          placeholder
        ) : isVideo ? (
          <>
            <video
              src={src}
              muted
              playsInline
              preload='metadata'
              onLoadedMetadata={(event) => {
                try {
                  event.currentTarget.currentTime = 0.1;
                } catch {
                  // ignore seek failures on some remote videos
                }
              }}
              onError={() => setFailed(true)}
            />
            <span className='playground-media-thumb-play'>
              <Play size={12} />
            </span>
          </>
        ) : (
          <img src={src} alt='' onError={() => setFailed(true)} />
        )}
      </button>
      <Modal
        title={isVideo ? t('视频预览') : t('图片预览')}
        visible={open && canPreview}
        footer={null}
        onCancel={() => setOpen(false)}
        width={isVideo ? 560 : 720}
        centered
        bodyStyle={{ padding: 12 }}
      >
        {open && canPreview ? (
          isVideo ? (
            <video
              src={src}
              controls
              autoPlay
              playsInline
              className='playground-media-thumb-lightbox-media'
            />
          ) : (
            <img
              src={src}
              alt={t('图片预览')}
              className='playground-media-thumb-lightbox-media'
            />
          )
        ) : null}
      </Modal>
    </>
  );
};

export default PlaygroundMediaThumb;
