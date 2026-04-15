/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.
*/

import React from 'react';
import { Navigate, useParams } from 'react-router-dom';

/** 短链 /r/:aff → 带邀请码的注册页 */
export default function InviteRedirect() {
  const { aff } = useParams();
  const code = aff ? encodeURIComponent(aff) : '';
  return <Navigate to={`/register?aff=${code}`} replace />;
}
