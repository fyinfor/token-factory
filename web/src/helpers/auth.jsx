/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React from 'react';
import { Navigate, useSearchParams } from 'react-router-dom';
import { history } from './history';

/** 登录/注册页 ?redirect= 仅允许站内相对路径，防止开放重定向 */
export function safeInternalRedirectPath(raw) {
  if (raw == null || typeof raw !== 'string') return null;
  let path = raw.trim();
  if (!path) return null;
  try {
    path = decodeURIComponent(path);
  } catch {
    return null;
  }
  if (!path.startsWith('/') || path.startsWith('//')) return null;
  if (/[\s\r\n]/.test(path)) return null;
  const lower = path.toLowerCase();
  if (lower.startsWith('javascript:') || path.includes('://')) return null;
  return path;
}

export function authHeader() {
  // return authorization header with jwt token
  let user = JSON.parse(localStorage.getItem('user'));

  if (user && user.token) {
    return { Authorization: 'Bearer ' + user.token };
  } else {
    return {};
  }
}

export const AuthRedirect = ({ children }) => {
  const [searchParams] = useSearchParams();
  const user = localStorage.getItem('user');

  if (user) {
    const next = safeInternalRedirectPath(searchParams.get('redirect'));
    return <Navigate to={next || '/console'} replace />;
  }

  return children;
};

function PrivateRoute({ children }) {
  if (!localStorage.getItem('user')) {
    return <Navigate to='/login' state={{ from: history.location }} />;
  }
  return children;
}

export function AdminRoute({ children }) {
  const raw = localStorage.getItem('user');
  if (!raw) {
    return <Navigate to='/login' state={{ from: history.location }} />;
  }
  try {
    const user = JSON.parse(raw);
    if (user && typeof user.role === 'number' && user.role >= 10) {
      return children;
    }
  } catch (e) {
    // ignore
  }
  return <Navigate to='/forbidden' replace />;
}

export { PrivateRoute };
