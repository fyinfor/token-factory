/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import { useEffect, useState } from 'react';
import {
  buildAssetMap,
  isAssetUri,
  resolveAssetUriToUrl,
  splitAssetUris,
} from './materialAssetUtils';

const PREVIEW_DEBOUNCE_MS = 250;
const ASSET_MAP_TTL_MS = 60 * 1000;

let assetMapCache = null;
let assetMapCachedAt = 0;
let assetMapInflight = null;

export function getFirstMediaPreviewCandidate(rawValue) {
  const parts = splitAssetUris(String(rawValue || ''));
  return parts[0] || '';
}

export function isPlaygroundPreviewableUrl(url) {
  const value = String(url || '').trim();
  if (!value) {
    return false;
  }
  if (
    value.startsWith('data:') ||
    value.startsWith('blob:') ||
    isAssetUri(value)
  ) {
    return true;
  }
  return /^https?:\/\//i.test(value);
}

export async function loadPlaygroundAssetPreviewMap() {
  const now = Date.now();
  if (assetMapCache && now - assetMapCachedAt < ASSET_MAP_TTL_MS) {
    return assetMapCache;
  }
  if (assetMapInflight) {
    return assetMapInflight;
  }
  assetMapInflight = import('./materialApi')
    .then(({ listMaterialAssets }) =>
      listMaterialAssets({ page: 1, pageSize: 100 }),
    )
    .then((res) => {
      const map =
        res?.success && Array.isArray(res.data?.items)
          ? buildAssetMap(res.data.items)
          : {};
      assetMapCache = map;
      assetMapCachedAt = Date.now();
      return map;
    })
    .catch(() => assetMapCache || {})
    .finally(() => {
      assetMapInflight = null;
    });
  return assetMapInflight;
}

export function useResolvedPlaygroundPreviewUrl(rawValue) {
  const candidate = getFirstMediaPreviewCandidate(rawValue);
  const [src, setSrc] = useState('');

  useEffect(() => {
    if (!isPlaygroundPreviewableUrl(candidate)) {
      setSrc('');
      return undefined;
    }

    let cancelled = false;
    const timer = setTimeout(() => {
      if (!isAssetUri(candidate)) {
        if (!cancelled) {
          setSrc(candidate);
        }
        return;
      }
      loadPlaygroundAssetPreviewMap().then((map) => {
        if (cancelled) {
          return;
        }
        setSrc(resolveAssetUriToUrl(candidate, map) || '');
      });
    }, PREVIEW_DEBOUNCE_MS);

    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [candidate]);

  return src;
}
