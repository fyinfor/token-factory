/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import {
  MESSAGE_ROLES,
  PLAYGROUND_MEDIA_MAX_COUNT,
  PLAYGROUND_MEDIA_UNLIMITED_COUNT,
  PLAYGROUND_VIDEO_FRAME_MAX_COUNT,
  PLAYGROUND_VIDEO_IMAGE_TABS,
} from '../constants/playground.constants';
import { splitAssetUris } from './materialAssetUtils';

export function isVideoURL(url) {
  return /\.(mp4|mov|avi|mkv|webm)(\?.*)?$/i.test(String(url || '').trim());
}

export function isAudioURL(url) {
  return /\.(mp3|wav|m4a|aac|ogg|flac)(\?.*)?$/i.test(String(url || '').trim());
}

function dedupeUrls(urls) {
  const seen = new Set();
  const out = [];
  for (const u of urls) {
    const s = String(u || '').trim();
    if (!s || seen.has(s)) continue;
    seen.add(s);
    out.push(s);
  }
  return out;
}

/** 从最后一条用户消息的多模态 content 中提取 image_url */
export function extractImageUrlsFromMessages(messages) {
  if (!Array.isArray(messages)) return [];
  const urls = [];
  for (let i = messages.length - 1; i >= 0; i--) {
    const msg = messages[i];
    if (msg?.role !== MESSAGE_ROLES.USER) continue;
    const content = msg.content;
    if (!Array.isArray(content)) break;
    for (const item of content) {
      if (item?.type !== 'image_url') continue;
      const url =
        typeof item.image_url === 'string'
          ? item.image_url
          : item.image_url?.url;
      if (url && String(url).trim()) urls.push(String(url).trim());
    }
    break;
  }
  return urls;
}

export function expandFilledMediaUrls(urls) {
  return dedupeUrls(
    (urls || []).flatMap((url) => {
      const trimmed = String(url || '').trim();
      if (!trimmed) return [];
      return splitAssetUris(trimmed);
    }),
  );
}

export function collectFilledImageUrls(urls) {
  return expandFilledMediaUrls(urls).filter(
    (url) => !isVideoURL(url) && !isAudioURL(url),
  );
}

export function getVideoImageTab(inputs) {
  return inputs?.videoImageTab === PLAYGROUND_VIDEO_IMAGE_TABS.FRAMES
    ? PLAYGROUND_VIDEO_IMAGE_TABS.FRAMES
    : PLAYGROUND_VIDEO_IMAGE_TABS.REFERENCE;
}

export function isVideoFramesTab(inputs) {
  return getVideoImageTab(inputs) === PLAYGROUND_VIDEO_IMAGE_TABS.FRAMES;
}

export function getActiveVideoImageField(inputs) {
  return isVideoFramesTab(inputs) ? 'frameImageUrls' : 'imageUrls';
}

export function getActiveVideoImageUrls(inputs) {
  if (isVideoFramesTab(inputs)) {
    return inputs?.frameImageUrls || [''];
  }
  return inputs?.imageUrls || [''];
}

export function getActiveVideoImageMaxCount(inputs) {
  return isVideoFramesTab(inputs)
    ? PLAYGROUND_VIDEO_FRAME_MAX_COUNT
    : PLAYGROUND_MEDIA_UNLIMITED_COUNT;
}

export function getPlaygroundSidebarImageUrls(inputs) {
  const mode = inputs?.display_mode || 'text';
  if (mode === 'video') {
    return getActiveVideoImageUrls(inputs);
  }
  return inputs?.imageUrls || [''];
}

export function getPlaygroundSidebarImageMaxCount(inputs) {
  const mode = inputs?.display_mode || 'text';
  if (mode === 'video') {
    return getActiveVideoImageMaxCount(inputs);
  }
  return PLAYGROUND_MEDIA_MAX_COUNT;
}

export function getPlaygroundSidebarImageField(inputs) {
  const mode = inputs?.display_mode || 'text';
  if (mode === 'video') {
    return getActiveVideoImageField(inputs);
  }
  return 'imageUrls';
}

/**
 * 合并侧栏图片/视频/音频地址。
 * includeMessageImages 默认 false：视频请求以侧栏独立字段为准，避免对话历史图混入另一类参数。
 */
export function collectVideoMediaUrls(
  sidebarImageUrls,
  sidebarVideoUrls,
  messages,
  sidebarAudioUrls,
  options = {},
) {
  const includeMessageImages = options.includeMessageImages === true;
  const expandedImageUrls = expandFilledMediaUrls(sidebarImageUrls);
  const fromImages = dedupeUrls([
    ...expandedImageUrls,
    ...(includeMessageImages ? extractImageUrlsFromMessages(messages) : []),
  ]);
  const misclassifiedVideos = fromImages.filter((url) => isVideoURL(url));
  const images = fromImages.filter(
    (url) => !isVideoURL(url) && !isAudioURL(url),
  );
  const audios = dedupeUrls([
    ...expandFilledMediaUrls(sidebarAudioUrls),
    ...fromImages.filter((url) => isAudioURL(url)),
  ]);
  const videos = dedupeUrls([
    ...expandFilledMediaUrls(sidebarVideoUrls),
    ...misclassifiedVideos,
  ]);
  return {
    images,
    videos,
    audios,
    all: dedupeUrls([...images, ...videos, ...audios]),
  };
}

/**
 * 按当前 Tab 解析视频图片参数：参考图与首尾帧互斥，只返回其中一类。
 */
export function resolvePlaygroundVideoImages(inputs) {
  const mode = getVideoImageTab(inputs);
  const referenceImages = collectFilledImageUrls(inputs?.imageUrls);
  const frameImages = collectFilledImageUrls(inputs?.frameImageUrls).slice(
    0,
    PLAYGROUND_VIDEO_FRAME_MAX_COUNT,
  );

  if (mode === PLAYGROUND_VIDEO_IMAGE_TABS.FRAMES) {
    return {
      mode,
      referenceImages: [],
      frameImages,
      images: [],
    };
  }

  return {
    mode,
    referenceImages,
    frameImages: [],
    images: referenceImages,
  };
}

export function buildFrameMediaItems(imageUrls) {
  const images = collectFilledImageUrls(imageUrls).slice(
    0,
    PLAYGROUND_VIDEO_FRAME_MAX_COUNT,
  );
  const media = [];
  if (images[0]) {
    media.push({ type: 'first_frame', url: images[0] });
  }
  if (images[1]) {
    media.push({ type: 'last_frame', url: images[1] });
  }
  return media;
}

export function buildReferenceMediaItems(imageUrls) {
  return collectFilledImageUrls(imageUrls).map((url) => ({
    type: 'reference_image',
    url,
  }));
}

/**
 * 图片 URL → media。frames 仅首尾帧；默认全部为参考图。禁止混合两类。
 */
export function buildImageMediaItems(
  imageUrls,
  mode = PLAYGROUND_VIDEO_IMAGE_TABS.REFERENCE,
) {
  if (mode === PLAYGROUND_VIDEO_IMAGE_TABS.FRAMES) {
    return buildFrameMediaItems(imageUrls);
  }
  return buildReferenceMediaItems(imageUrls);
}

/** 仅写入首尾帧 metadata，不附带参考图字段 */
export function applyVideoFrameMetadata(metadata, frameImageUrls) {
  const images = collectFilledImageUrls(frameImageUrls).slice(
    0,
    PLAYGROUND_VIDEO_FRAME_MAX_COUNT,
  );
  if (images.length === 0) return metadata;
  const next = { ...metadata, first_frame_url: images[0] };
  if (images[1]) {
    next.last_frame_url = images[1];
  }
  return next;
}

export function payloadHasVideoReferenceImages(payload) {
  return (payload?.images || []).some((url) => String(url || '').trim());
}

export function payloadHasVideoFrameParams(payload) {
  const metadata = payload?.metadata || {};
  return Boolean(
    String(payload?.first_frame_url || '').trim() ||
      String(payload?.last_frame_url || '').trim() ||
      String(metadata.first_frame_url || '').trim() ||
      String(metadata.last_frame_url || '').trim(),
  );
}

export function assertExclusiveVideoImagePayload(payload) {
  if (
    payloadHasVideoReferenceImages(payload) &&
    payloadHasVideoFrameParams(payload)
  ) {
    return {
      ok: false,
      message: '首尾帧与参考图不能同时提交，请只选择其中一类',
    };
  }
  return { ok: true };
}

export function validatePlaygroundVideoImageParams(inputs) {
  const frameImages = collectFilledImageUrls(inputs?.frameImageUrls);
  if (frameImages.length > PLAYGROUND_VIDEO_FRAME_MAX_COUNT) {
    return {
      ok: false,
      message: '首尾帧最多只能填写 2 张图片',
    };
  }
  return { ok: true };
}

/**
 * 操练场 metadata.input：只传 prompt，media.type 由后端 alivideo adaptor 按模型规范化。
 */
export function buildVideoNativeInput(prompt) {
  return { prompt: String(prompt || '').trim() };
}

export function formatVideoTaskError(data) {
  if (!data || typeof data !== 'object') return '';
  const err = data.error;
  if (err && typeof err === 'object') {
    const msg = err.message || err.msg;
    const code = err.code || err.type;
    if (msg && code) return `${msg}（${code}）`;
    if (msg) return String(msg);
  }
  const taskPayload =
    data.data && typeof data.data === 'object' && !Array.isArray(data.data)
      ? data.data
      : data;
  if (taskPayload?.message) return String(taskPayload.message);
  if (taskPayload?.fail_reason) return String(taskPayload.fail_reason);
  if (taskPayload?.reason) return String(taskPayload.reason);
  if (
    taskPayload?.result_url &&
    !/^https?:\/\//i.test(String(taskPayload.result_url))
  ) {
    return String(taskPayload.result_url);
  }
  return '';
}
