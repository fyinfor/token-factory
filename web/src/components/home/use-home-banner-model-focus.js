/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

For commercial licensing, please contact support@quantumnous.com
*/

import { useCallback, useEffect } from 'react';
import { useSearchParams } from 'react-router-dom';

/**
 * 首页模型列表：响应 ?model= 与轮播 CTA 的 home-banner-focus-model 事件，仅填入搜索筛选。
 */
export function useHomeBannerModelFocus({ loading, setSearchValue }) {
  const [searchParams, setSearchParams] = useSearchParams();

  const applyFilter = useCallback(
    (modelName) => {
      const name = (modelName || '').trim();
      if (!name) return;
      setSearchValue(name);
    },
    [setSearchValue],
  );

  useEffect(() => {
    const fromUrl = searchParams.get('model')?.trim();
    if (!fromUrl || loading) return;

    applyFilter(fromUrl);
    const next = new URLSearchParams(searchParams);
    next.delete('model');
    setSearchParams(next, { replace: true });
  }, [searchParams, loading, applyFilter, setSearchParams]);

  useEffect(() => {
    const onFocus = (e) => {
      applyFilter(e.detail?.model);
    };
    window.addEventListener('home-banner-focus-model', onFocus);
    return () => window.removeEventListener('home-banner-focus-model', onFocus);
  }, [applyFilter]);
}
