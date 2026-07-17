/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import { MESSAGE_ROLES } from '../constants/playground.constants';
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

/**
 * 合并侧栏图片/视频/音频地址与消息内图片。
 * sidebarAudioUrls：操练场视频媒体模块的音频链接列表。
 */
export function collectVideoMediaUrls(
  sidebarImageUrls,
  sidebarVideoUrls,
  messages,
  sidebarAudioUrls,
) {
  // 【需求5】展开多素材拼接条目（支持分隔符拆分）
  const expandedImageUrls = (sidebarImageUrls || []).flatMap((u) => {
    const s = String(u || '').trim();
    if (!s) return [];
    return splitAssetUris(s);
  });
  const fromImages = dedupeUrls([
    ...expandedImageUrls,
    ...extractImageUrlsFromMessages(messages),
  ]);
  const misclassifiedVideos = fromImages.filter((url) => isVideoURL(url));
  const images = fromImages.filter(
    (url) => !isVideoURL(url) && !isAudioURL(url),
  );
  const audios = dedupeUrls([
    ...(sidebarAudioUrls || [])
      .map((u) => String(u || '').trim())
      .filter(Boolean),
    ...fromImages.filter((url) => isAudioURL(url)),
  ]);
  const videos = dedupeUrls([
    ...(sidebarVideoUrls || [])
      .map((u) => String(u || '').trim())
      .filter(Boolean),
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
 * 图片 URL → media：1 张首帧；2 张首尾帧；3+ 张首帧+中间参考图+尾帧。
 */
export function buildImageMediaItems(imageUrls) {
  const images = (imageUrls || [])
    .map((u) => String(u || '').trim())
    .filter(Boolean);
  const media = [];
  if (images.length === 0) return media;

  media.push({ type: 'first_frame', url: images[0] });
  if (images.length === 1) return media;

  if (images.length === 2) {
    media.push({ type: 'last_frame', url: images[1] });
    return media;
  }

  for (let i = 1; i < images.length - 1; i++) {
    media.push({ type: 'reference_image', url: images[i] });
  }
  media.push({ type: 'last_frame', url: images[images.length - 1] });
  return media;
}

/** 写入 metadata 首尾帧字段 */
export function applyVideoFrameMetadata(metadata, imageUrls) {
  const images = (imageUrls || [])
    .map((u) => String(u || '').trim())
    .filter(Boolean);
  if (images.length === 0) return metadata;
  const next = { ...metadata, first_frame_url: images[0] };
  if (images.length >= 2) {
    next.last_frame_url = images[images.length - 1];
  }
  return next;
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
