/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import { API } from './apiClient';
import { PLAYGROUND_MEDIA_MAX_COUNT } from '../constants/playground.constants';

export function isUnlimitedMediaCount(max) {
  return typeof max === 'number' && !Number.isFinite(max) && max > 0;
}

export function countFilledMediaUrls(urls) {
  return (urls || []).filter((url) => String(url || '').trim()).length;
}

export function canAddMediaUrlRow(
  urls,
  max = PLAYGROUND_MEDIA_MAX_COUNT,
) {
  if (isUnlimitedMediaCount(max)) {
    return true;
  }
  return (urls || []).length < max;
}

export function canAcceptMoreMediaUrls(
  urls,
  max = PLAYGROUND_MEDIA_MAX_COUNT,
) {
  if (isUnlimitedMediaCount(max)) {
    return true;
  }
  return countFilledMediaUrls(urls) < max;
}

export function canAddMoreMediaUrls(
  urls,
  max = PLAYGROUND_MEDIA_MAX_COUNT,
) {
  if (isUnlimitedMediaCount(max)) {
    return true;
  }
  if (!canAcceptMoreMediaUrls(urls, max)) {
    return false;
  }
  const list = urls || [];
  if (list.length < max) {
    return true;
  }
  return list.some((url) => !String(url || '').trim());
}

export function appendUploadedMediaUrl(
  urls,
  uploadedUrl,
  max = PLAYGROUND_MEDIA_MAX_COUNT,
) {
  const next = [...(urls || [])];
  const trimmedUrl = String(uploadedUrl || '').trim();
  if (!trimmedUrl) {
    return { urls: next, ok: false };
  }
  if (!canAcceptMoreMediaUrls(next, max)) {
    return { urls: next, ok: false };
  }

  const emptyIndex = next.findIndex((url) => !String(url || '').trim());
  if (emptyIndex >= 0) {
    next[emptyIndex] = trimmedUrl;
    return { urls: next, ok: true };
  }

  if (isUnlimitedMediaCount(max) || next.length < max) {
    next.push(trimmedUrl);
    return { urls: next, ok: true };
  }

  return { urls: next, ok: false };
}

export function appendMediaUrlsWithLimit(
  currentUrls,
  incomingUrls,
  max = PLAYGROUND_MEDIA_MAX_COUNT,
) {
  let next = [...(currentUrls || [])];
  let added = 0;
  let skipped = 0;
  for (const raw of incomingUrls || []) {
    const url = String(raw || '').trim();
    if (!url) {
      continue;
    }
    const result = appendUploadedMediaUrl(next, url, max);
    if (!result.ok) {
      skipped += 1;
      continue;
    }
    next = result.urls;
    added += 1;
  }
  if (next.length === 0) {
    next = [''];
  }
  return { urls: next, added, skipped };
}

export async function uploadPlaygroundMediaFile(file) {
  const fd = new FormData();
  fd.append('file', file);
  fd.append('purpose', 'playground');
  const res = await API.post('/api/oss/upload', fd, {
    skipErrorHandler: true,
  });
  const { success, message, data } = res.data || {};
  const url = String(data?.url || '').trim();
  if (!success || !url) {
    throw new Error(message || 'upload failed');
  }
  return url;
}
