/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import { API } from './api';
import { PLAYGROUND_MEDIA_MAX_COUNT } from '../constants/playground.constants';

export function countFilledMediaUrls(urls) {
  return (urls || []).filter((url) => String(url || '').trim()).length;
}

export function canAddMoreMediaUrls(
  urls,
  max = PLAYGROUND_MEDIA_MAX_COUNT,
) {
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

  const emptyIndex = next.findIndex((url) => !String(url || '').trim());
  if (emptyIndex >= 0) {
    next[emptyIndex] = trimmedUrl;
    return { urls: next, ok: true };
  }

  if (next.length < max) {
    next.push(trimmedUrl);
    return { urls: next, ok: true };
  }

  return { urls: next, ok: false };
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
