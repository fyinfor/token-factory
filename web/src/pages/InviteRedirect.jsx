/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import React, { useEffect } from 'react';
import { Navigate, useParams } from 'react-router-dom';
import { API } from '../helpers';

/** 邀请链接 /r/:aff → 带邀请码的注册页 */
export default function InviteRedirect() {
  const { aff } = useParams();
  useEffect(() => {
    const raw = aff ? String(aff).trim() : '';
    if (!raw) return;
    API.post(
      '/api/aff/track',
      { event: 'short_link_click', aff: raw },
      { skipErrorHandler: true },
    ).catch(() => {});
  }, [aff]);
  const code = aff ? encodeURIComponent(aff) : '';
  return <Navigate to={`/register?aff=${code}`} replace />;
}
